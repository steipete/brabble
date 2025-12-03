package run

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"brabble/internal/asr"
	"brabble/internal/config"
	"brabble/internal/control"
	"brabble/internal/hook"

	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg       *config.Config
	logger    *logrus.Logger
	hook      *hook.Runner
	startedAt time.Time
	lastHeard atomic.Int64

	transcriptsMu sync.Mutex
	transcripts   []control.Transcript

	metrics metrics
	hookCh  chan hook.Job

	wg sync.WaitGroup
}

// Serve runs the daemon until interrupted.
func Serve(cfg *config.Config, logger *logrus.Logger) error {
	if err := config.MustStatePaths(cfg); err != nil {
		return err
	}
	// Write pid file.
	if err := os.WriteFile(cfg.Paths.PidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
		return err
	}
	defer os.Remove(cfg.Paths.PidPath)
	// Ensure socket removed
	_ = os.Remove(cfg.Paths.SocketPath)

	srv := &Server{
		cfg:         cfg,
		logger:      logger,
		hook:        hook.NewRunner(cfg, logger),
		startedAt:   time.Now(),
		transcripts: make([]control.Transcript, 0, cfg.UI.StatusTail),
		hookCh:      make(chan hook.Job, max(1, cfg.Hook.QueueSize)),
	}
	srv.metrics.reset()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Control socket
	go srv.controlLoop(ctx)

	// Hook worker
	go srv.hookWorker(ctx)

	// Metrics server
	if cfg.Metrics.Enabled {
		go srv.metricsServe(ctx.Done(), cfg.Metrics.Addr, logger)
	}

	// Watchdog
	go srv.watchdog(ctx.Done())

	// Audio/ASR loop
	go srv.asrLoop(ctx)

	// Handle signals
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	select {
	case s := <-sigCh:
		logger.Infof("received signal %s, shutting down", s)
		cancel()
	case <-ctx.Done():
	}
	// Wait for hook worker to drain
	srv.wg.Wait()
	return nil
}

func (s *Server) asrLoop(ctx context.Context) {
	rec, err := asr.NewRecognizer(s.cfg, s.logger)
	if err != nil {
		s.logger.Errorf("asr init: %v", err)
		return
	}
	segCh := make(chan asr.Segment, 8)
	go func() {
		if err := rec.Run(ctx, segCh); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Errorf("asr run: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case seg := <-segCh:
			s.handleSegment(ctx, seg)
		}
	}
}

func (s *Server) handleSegment(ctx context.Context, seg asr.Segment) {
	text := strings.TrimSpace(seg.Text)
	if text == "" {
		return
	}
	s.lastHeard.Store(time.Now().UnixNano())
	s.metrics.incHeard()
	s.logger.Infof("heard: %q", text)
	s.recordTranscript(text)
	if s.cfg.Wake.Enabled && !strings.Contains(strings.ToLower(text), strings.ToLower(s.cfg.Wake.Word)) {
		return
	}
	// strip wake word
	if s.cfg.Wake.Enabled {
		text = removeWakeWord(text, s.cfg.Wake.Word)
	}
	if len(text) < s.cfg.Hook.MinChars {
		return
	}
	if !s.hook.ShouldRun() {
		s.logger.Debug("hook skipped (cooldown)")
		s.metrics.incSkipped()
		return
	}
	job := hook.Job{
		Text:      text,
		Timestamp: time.Now(),
	}
	select {
	case s.hookCh <- job:
	default:
		s.metrics.incDropped()
		s.logger.Warn("hook queue full, dropping job")
	}
}

func removeWakeWord(text, word string) string {
	lw := strings.ToLower(word)
	fields := strings.Fields(text)
	out := make([]string, 0, len(fields))
	skipped := false
	for _, f := range fields {
		if !skipped && strings.EqualFold(stripPunct(f), lw) {
			skipped = true
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

func stripPunct(s string) string {
	return strings.Trim(s, " ,.!?;:\"'")
}

func (s *Server) recordTranscript(text string) {
	if !s.cfg.Transcripts.Enabled {
		return
	}
	entry := control.Transcript{
		Text:      text,
		Timestamp: time.Now(),
	}
	s.transcriptsMu.Lock()
	defer s.transcriptsMu.Unlock()
	s.transcripts = append(s.transcripts, entry)
	if len(s.transcripts) > s.cfg.UI.StatusTail {
		s.transcripts = s.transcripts[len(s.transcripts)-s.cfg.UI.StatusTail:]
	}
	// append to file
	f, err := os.OpenFile(s.cfg.Paths.TranscriptPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		fmt.Fprintf(f, "%s\t%s\n", entry.Timestamp.Format(time.RFC3339), entry.Text)
		_ = f.Close()
	}
}

func (s *Server) controlLoop(ctx context.Context) {
	ln, err := net.Listen("unix", s.cfg.Paths.SocketPath)
	if err != nil {
		s.logger.Errorf("control listen: %v", err)
		return
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Errorf("control accept: %v", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	if !sc.Scan() {
		return
	}
	var req control.Request
	if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
		return
	}
	switch req.Op {
	case "status":
		resp := control.Status{
			Running:     true,
			UptimeSec:   time.Since(s.startedAt).Seconds(),
			Transcripts: s.copyTranscripts(),
		}
		_ = json.NewEncoder(conn).Encode(resp)
	case "health":
		_ = json.NewEncoder(conn).Encode(control.SimpleResponse{OK: true, Message: "ok"})
	default:
		// ignore unknown
	}
}
func (s *Server) copyTranscripts() []control.Transcript {
	s.transcriptsMu.Lock()
	defer s.transcriptsMu.Unlock()
	out := make([]control.Transcript, len(s.transcripts))
	copy(out, s.transcripts)
	return out
}
