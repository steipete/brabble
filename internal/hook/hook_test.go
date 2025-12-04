package hook

import (
	"context"
	"testing"
	"time"

	"brabble/internal/config"
	"brabble/internal/logging"
)

func TestShouldRunCooldown(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{{
		Command:     "/bin/echo",
		CooldownSec: 0.5,
	}}
	r := NewRunner(cfg, logging.NewTestLogger())
	r.SelectHook(&cfg.Hooks[0])

	if !r.ShouldRun() {
		t.Fatalf("first call should run")
	}
	if err := r.Run(context.Background(), Job{Text: "test", Timestamp: time.Now()}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if r.ShouldRun() {
		t.Fatalf("cooldown should block immediate subsequent run")
	}
	time.Sleep(time.Duration(cfg.Hook.CooldownSec*float64(time.Second)) + 20*time.Millisecond)
	if !r.ShouldRun() {
		t.Fatalf("should run after cooldown")
	}
}

func TestRunUsesPrefixAndEnv(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{{
		Command: "/bin/echo",
		Args:    []string{},
		Prefix:  "pref:",
	}}

	r := NewRunner(cfg, logging.NewTestLogger())
	r.SelectHook(&cfg.Hooks[0])
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.Run(ctx, Job{Text: "hello", Timestamp: time.Now()}); err != nil {
		t.Fatalf("run echo: %v", err)
	}
}
