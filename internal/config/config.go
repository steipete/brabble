package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultWakeWord      = "clawd"
	defaultSilenceMS     = 1000
	defaultMinChars      = 24
	defaultCooldown      = 1.0
	defaultStatusTail    = 10
	defaultStateDirLinux = ".local/state/brabble"
	defaultConfigDir     = ".config/brabble"
)

// Config holds user configuration loaded from TOML.
type Config struct {
	Audio struct {
		DeviceName  string `toml:"device_name"`
		DeviceIndex int    `toml:"device_index"`
		SampleRate  int    `toml:"sample_rate"`
		Channels    int    `toml:"channels"`
		FrameMS     int    `toml:"frame_ms"`
	} `toml:"audio"`

	VAD struct {
		Enabled        bool    `toml:"enabled"`
		SilenceMS      int     `toml:"silence_ms"`
		Aggressiveness int     `toml:"aggressiveness"`
		EnergyThresh   float64 `toml:"energy_threshold"`
		MinSpeechMS    int     `toml:"min_speech_ms"`
		MaxSegmentMS   int     `toml:"max_segment_ms"`
		PartialFlushMS int     `toml:"partial_flush_ms"`
	} `toml:"vad"`

	ASR struct {
		ModelPath   string `toml:"model_path"`
		Language    string `toml:"language"`
		ComputeType string `toml:"compute_type"` // q5_1, q8_0, float16
		Device      string `toml:"device"`       // auto, cpu, metal
	} `toml:"asr"`

	Wake struct {
		Enabled     bool    `toml:"enabled"`
		Word        string  `toml:"word"`
		Sensitivity float64 `toml:"sensitivity"`
	} `toml:"wake"`

	Hook struct {
		Command      string            `toml:"command"`
		Args         []string          `toml:"args"`
		Prefix       string            `toml:"prefix"`
		CooldownSec  float64           `toml:"cooldown_sec"`
		MinChars     int               `toml:"min_chars"`
		MaxLatencyMS int               `toml:"max_latency_ms"`
		QueueSize    int               `toml:"queue_size"`
		TimeoutSec   float64           `toml:"timeout_sec"`
		Env          map[string]string `toml:"env"`
		RedactPII    bool              `toml:"redact_pii"`
	} `toml:"hook"`

	Logging struct {
		Level  string `toml:"level"`  // debug, info, warn, error
		Format string `toml:"format"` // text, json
	} `toml:"logging"`

	Paths struct {
		StateDir       string `toml:"state_dir"`
		LogPath        string `toml:"log_path"`
		TranscriptPath string `toml:"transcript_path"`
		SocketPath     string `toml:"socket_path"`
		PidPath        string `toml:"pid_path"`
		ConfigPath     string `toml:"-"`
	} `toml:"paths"`

	UI struct {
		StatusTail int `toml:"status_tail"`
	} `toml:"ui"`

	Metrics struct {
		Enabled bool   `toml:"enabled"`
		Addr    string `toml:"addr"`
	} `toml:"metrics"`

	Transcripts struct {
		Enabled bool `toml:"enabled"`
	} `toml:"transcripts"`
}

// Default returns Config populated with defaults.
func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(home, defaultStateDirLinux)
	// macOS prefers ~/Library/Application Support/brabble for state/logs
	if isMac() {
		stateDir = filepath.Join(home, "Library", "Application Support", "brabble")
	}

	cfg := &Config{}

	cfg.Audio.SampleRate = 16000
	cfg.Audio.Channels = 1
	cfg.Audio.FrameMS = 20

	cfg.VAD.Enabled = true
	cfg.VAD.SilenceMS = defaultSilenceMS
	cfg.VAD.Aggressiveness = 2
	cfg.VAD.MinSpeechMS = 300
	cfg.VAD.MaxSegmentMS = 10000
	cfg.VAD.EnergyThresh = 0.0
	cfg.VAD.PartialFlushMS = 4000

	cfg.ASR.ModelPath = filepath.Join(stateDir, "models", "ggml-medium-q5_1.bin")
	cfg.ASR.Language = "auto"
	cfg.ASR.ComputeType = "q5_1"
	cfg.ASR.Device = "auto"

	cfg.Wake.Enabled = true
	cfg.Wake.Word = DefaultWakeWord
	cfg.Wake.Sensitivity = 0.6

	cfg.Hook.Command = "../warelay"
	cfg.Hook.Args = []string{"send"}
	cfg.Hook.Prefix = "Voice brabble from ${hostname}: "
	cfg.Hook.CooldownSec = defaultCooldown
	cfg.Hook.MinChars = defaultMinChars
	cfg.Hook.MaxLatencyMS = 5000
	cfg.Hook.QueueSize = 16
	cfg.Hook.TimeoutSec = 5
	cfg.Hook.Env = map[string]string{}
	cfg.Hook.RedactPII = false

	cfg.Logging.Level = "info"
	cfg.Logging.Format = "text"

	cfg.Paths.StateDir = stateDir
	cfg.Paths.LogPath = filepath.Join(stateDir, "brabble.log")
	cfg.Paths.TranscriptPath = filepath.Join(stateDir, "transcripts.log")
	cfg.Paths.SocketPath = filepath.Join(stateDir, "brabble.sock")
	cfg.Paths.PidPath = filepath.Join(stateDir, "brabble.pid")

	cfg.UI.StatusTail = defaultStatusTail

	cfg.Metrics.Enabled = false
	cfg.Metrics.Addr = "127.0.0.1:9317"

	cfg.Transcripts.Enabled = true

	return cfg, nil
}

// Load loads config from file, applying defaults.
func Load(path string) (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, err
	}

	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, defaultConfigDir, "config.toml")
	}

	// Read if exists; otherwise write template.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// ensure dir
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := Save(cfg, path); err != nil {
				return nil, err
			}
			cfg.Paths.ConfigPath = path
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.Paths.ConfigPath = path
	applyEnvOverrides(cfg)
	return cfg, nil
}

// Save writes cfg to path.
func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

func isMac() bool {
	return runtime.GOOS == "darwin"
}

// MustStatePaths ensures state dirs exist.
func MustStatePaths(cfg *Config) error {
	for _, p := range []string{cfg.Paths.StateDir, filepath.Dir(cfg.Paths.LogPath), filepath.Dir(cfg.Paths.TranscriptPath)} {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("BRABBLE_WAKE_ENABLED"); v != "" {
		cfg.Wake.Enabled = v != "0" && strings.ToLower(v) != "false"
	}
	if v := os.Getenv("BRABBLE_METRICS_ADDR"); v != "" {
		cfg.Metrics.Addr = v
		cfg.Metrics.Enabled = true
	}
	if v := os.Getenv("BRABBLE_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("BRABBLE_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("BRABBLE_TRANSCRIPTS_ENABLED"); v != "" {
		cfg.Transcripts.Enabled = v != "0" && strings.ToLower(v) != "false"
	}
	if v := os.Getenv("BRABBLE_REDACT_PII"); v != "" {
		cfg.Hook.RedactPII = v != "0" && strings.ToLower(v) != "false"
	}
}

// NowUnixMilli returns milliseconds since epoch.
func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}
