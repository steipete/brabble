package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"brabble/internal/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is a thin wrapper over slog.Logger that offers *f-style helpers to ease migration.
type Logger struct {
	*slog.Logger
}

// Configure sets up slog with rotation (lumberjack) and optional stdout tee.
func Configure(cfg *config.Config) (*Logger, error) {
	if err := config.MustStatePaths(cfg); err != nil {
		return nil, err
	}

	level := parseLevel(cfg.Logging.Level)

	rotator := &lumberjack.Logger{
		Filename:   cfg.Paths.LogPath,
		MaxSize:    20, // MB
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   false,
	}

	var writer io.Writer = rotator
	if cfg.Logging.Stdout {
		writer = io.MultiWriter(os.Stdout, rotator)
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch strings.ToLower(cfg.Logging.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{slog.New(handler)}, nil
}

// NewTestLogger returns a discard logger for tests.
func NewTestLogger() *Logger {
	return &Logger{slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))}
}

// f-style helpers ------------------------------------------------------------

// Infof formats and logs at info level (legacy fmt-style helper).
func (l *Logger) Infof(format string, args ...any) { l.Logger.Info(fmt.Sprintf(format, args...)) }

// Warnf formats and logs at warn level (legacy fmt-style helper).
func (l *Logger) Warnf(format string, args ...any) { l.Logger.Warn(fmt.Sprintf(format, args...)) }

// Errorf formats and logs at error level (legacy fmt-style helper).
func (l *Logger) Errorf(format string, args ...any) { l.Logger.Error(fmt.Sprintf(format, args...)) }

// Debugf formats and logs at debug level (legacy fmt-style helper).
func (l *Logger) Debugf(format string, args ...any) { l.Logger.Debug(fmt.Sprintf(format, args...)) }

// parseLevel maps string level to slog.Leveler.
func parseLevel(level string) slog.Leveler {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
