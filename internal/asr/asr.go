package asr

import (
	"context"
	"time"

	"brabble/internal/config"
	"brabble/internal/logging"
)

// Segment is a recognized piece of text.
type Segment struct {
	Text       string
	Start      time.Time
	End        time.Time
	Confidence float64
	Partial    bool
}

// Recognizer converts audio into segments.
type Recognizer interface {
	Run(ctx context.Context, out chan<- Segment) error
}

// NewRecognizer returns the whisper recognizer.
func NewRecognizer(cfg *config.Config, logger *logging.Logger) (Recognizer, error) {
	return newWhisperRecognizer(cfg, logger)
}
