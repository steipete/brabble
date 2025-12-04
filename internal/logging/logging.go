package logging

import (
	"io"
	"os"
	"strings"

	"brabble/internal/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Configure sets up logrus with rotation.
func Configure(cfg *config.Config) (*logrus.Logger, error) {
	if err := config.MustStatePaths(cfg); err != nil {
		return nil, err
	}
	logger := logrus.New()
	switch strings.ToLower(cfg.Logging.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	default:
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
	if lvl, err := logrus.ParseLevel(strings.ToLower(cfg.Logging.Level)); err == nil {
		logger.SetLevel(lvl)
	}
	rotator := &lumberjack.Logger{
		Filename:   cfg.Paths.LogPath,
		MaxSize:    20, // megabytes
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   false,
	}
	if cfg.Logging.Stdout {
		logger.SetOutput(io.MultiWriter(os.Stdout, rotator))
	} else {
		logger.SetOutput(rotator)
	}
	return logger, nil
}
