package control

import (
	"path/filepath"
	"testing"

	"brabble/internal/config"
)

func TestModelDirUsesConfiguredStateDir(t *testing.T) {
	cfg := &config.Config{}
	cfg.Paths.StateDir = filepath.Join(t.TempDir(), "state")

	got := modelDir(cfg)
	want := filepath.Join(cfg.Paths.StateDir, "models")
	if got != want {
		t.Fatalf("modelDir=%q want %q", got, want)
	}
}
