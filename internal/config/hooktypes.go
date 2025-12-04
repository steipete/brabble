package config

// HookConfig defines a per-wake hook invocation entry.
type HookConfig struct {
	Wake        []string          `toml:"wake"`    // tokens to match (case-insensitive)
	Aliases     []string          `toml:"aliases"` // optional extra tokens
	Command     string            `toml:"command"`
	Args        []string          `toml:"args"`
	Prefix      string            `toml:"prefix"`
	CooldownSec float64           `toml:"cooldown_sec"`
	MinChars    int               `toml:"min_chars"`
	MaxLatency  int               `toml:"max_latency_ms"`
	QueueSize   int               `toml:"queue_size"`
	TimeoutSec  float64           `toml:"timeout_sec"`
	Env         map[string]string `toml:"env"`
	RedactPII   bool              `toml:"redact_pii"`
}
