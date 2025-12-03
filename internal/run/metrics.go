package run

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type metrics struct {
	heard   atomic.Int64
	sent    atomic.Int64
	skipped atomic.Int64
	dropped atomic.Int64
	started int64
}

func (m *metrics) reset() {
	m.heard.Store(0)
	m.sent.Store(0)
	m.skipped.Store(0)
	m.dropped.Store(0)
}

func (m *metrics) incHeard()   { m.heard.Add(1) }
func (m *metrics) incSent()    { m.sent.Add(1) }
func (m *metrics) incSkipped() { m.skipped.Add(1) }
func (m *metrics) incDropped() { m.dropped.Add(1) }

func (s *Server) metricsServe(ctxDone <-chan struct{}, addr string, logger interface {
	Infof(string, ...any)
	Warnf(string, ...any)
}) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "brabble_heard_total %d\n", s.metrics.heard.Load())
		fmt.Fprintf(w, "brabble_hooks_sent_total %d\n", s.metrics.sent.Load())
		fmt.Fprintf(w, "brabble_hooks_skipped_total %d\n", s.metrics.skipped.Load())
		fmt.Fprintf(w, "brabble_hooks_dropped_total %d\n", s.metrics.dropped.Load())
	})
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		<-ctxDone
		_ = server.Close()
	}()
	logger.Infof("metrics listening on http://%s/metrics", addr)
	if err := server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
		logger.Warnf("metrics server: %v", err)
	}
}
