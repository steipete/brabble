package asr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"brabble/internal/config"

	"github.com/sirupsen/logrus"
)

// Segment is a recognized piece of text.
type Segment struct {
	Text      string
	Start     time.Time
	End       time.Time
	Confidence float64
	Partial    bool
}

// Recognizer converts audio into segments.
type Recognizer interface {
	Run(ctx context.Context, out chan<- Segment) error
}

// NewRecognizer returns the default recognizer (stdin stub unless built with whisper tags).
func NewRecognizer(cfg *config.Config, logger *logrus.Logger) (Recognizer, error) {
	if isWhisperEnabled() {
		if r, err := newWhisperRecognizer(cfg, logger); err == nil && r != nil {
			return r, nil
		} else if err != nil {
			logger.Warnf("whisper recognizer unavailable: %v; falling back to stdin", err)
		}
	}
	logger.Warn("using stdin recognizer (build with -tags whisper for real ASR)")
	return &stdinRecognizer{logger: logger}, nil
}

type stdinRecognizer struct {
	logger *logrus.Logger
}

func (s *stdinRecognizer) Run(ctx context.Context, out chan<- Segment) error {
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		text := strings.TrimSpace(reader.Text())
		out <- Segment{Text: text, Start: time.Now(), End: time.Now(), Confidence: 1.0, Partial: false}
	}
	if err := reader.Err(); err != nil {
		return fmt.Errorf("stdin read: %w", err)
	}
	// keep the goroutine alive so the daemon doesn't exit immediately when stdin closes
	<-ctx.Done()
	return ctx.Err()
}

// isWhisperEnabled is replaced in whisper build.
func isWhisperEnabled() bool {
	return false
}
