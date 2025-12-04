package daemon

import (
	"fmt"
	"os"
	"testing"
	"time"

	"brabble/internal/config"
)

func TestWaitForShutdownSucceedsWhenPidFileRemoved(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Default()
	cfg.Paths.ConfigPath = dir + "/config.toml"
	cfg.Paths.PidPath = dir + "/brabble.pid"
	if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	if err := os.WriteFile(cfg.Paths.PidPath, []byte("12345"), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.Remove(cfg.Paths.PidPath)
	}()
	if err := waitForShutdown(cfg.Paths.ConfigPath, 2*time.Second); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForShutdownTimesOutOnAlivePid(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Default()
	cfg.Paths.ConfigPath = dir + "/config.toml"
	cfg.Paths.PidPath = dir + "/brabble.pid"
	if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	selfPid := os.Getpid()
	if err := os.WriteFile(cfg.Paths.PidPath, []byte(fmt.Sprintf("%d", selfPid)), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if err := waitForShutdown(cfg.Paths.ConfigPath, 300*time.Millisecond); err == nil {
		t.Fatalf("expected timeout error")
	}
}
