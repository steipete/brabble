package config

import (
	"os"
	"testing"
)

func TestDefaultDeviceIndexIsUnset(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Audio.DeviceIndex != -1 {
		t.Fatalf("DeviceIndex = %d, want -1", cfg.Audio.DeviceIndex)
	}
}

func TestEnvOverrides(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	cfg.Paths.ConfigPath = "/tmp/config" // avoid creation

	t.Setenv("BRABBLE_WAKE_ENABLED", "0")
	t.Setenv("BRABBLE_METRICS_ADDR", "1.2.3.4:9999")
	t.Setenv("BRABBLE_LOG_LEVEL", "debug")
	t.Setenv("BRABBLE_LOG_FORMAT", "json")

	applyEnvOverrides(cfg)

	if cfg.Wake.Enabled {
		t.Fatalf("wake should be disabled via env")
	}
	if !cfg.Metrics.Enabled || cfg.Metrics.Addr != "1.2.3.4:9999" {
		t.Fatalf("metrics override failed: %+v", cfg.Metrics)
	}
	if cfg.Logging.Level != "debug" || cfg.Logging.Format != "json" {
		t.Fatalf("logging overrides failed: %+v", cfg.Logging)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.toml"

	cfg, err := Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	cfg.Paths.ConfigPath = path
	cfg.Hook.Command = "/bin/echo"

	if err := Save(cfg, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Hook.Command != "/bin/echo" {
		t.Fatalf("expected hook command to persist")
	}

	// cleanup to avoid residue
	_ = os.Remove(path)
}

func TestEffectiveHooksFallsBackToSingleHook(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	cfg.Hook.Command = "/bin/echo"
	cfg.Hook.Args = []string{"--flag"}
	cfg.Hook.QueueSize = 32

	hooks := cfg.EffectiveHooks()
	if len(hooks) != 1 {
		t.Fatalf("effective hooks = %d, want 1", len(hooks))
	}
	if hooks[0].Command != cfg.Hook.Command || hooks[0].QueueSize != cfg.Hook.QueueSize {
		t.Fatalf("legacy hook was not preserved: %+v", hooks[0])
	}
}

func TestRedactPIIOverrideAppliesToAllHooks(t *testing.T) {
	cfg, err := Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	cfg.Hooks = []HookConfig{{}, {}}
	t.Setenv("BRABBLE_REDACT_PII", "1")

	applyEnvOverrides(cfg)

	if !cfg.Hook.RedactPII || !cfg.Hooks[0].RedactPII || !cfg.Hooks[1].RedactPII {
		t.Fatal("redaction override did not apply to every hook")
	}
}
