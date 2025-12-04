package run

import (
	"testing"

	"brabble/internal/config"
)

func TestSelectHookConfigMatchesWakeTokens(t *testing.T) {
	cfg, _ := config.Default()
	cfg.Hooks = []config.HookConfig{
		{Wake: []string{"alpha"}, Command: "/bin/echo"},
		{Wake: []string{"clawd", "claude", "cloud"}, Command: "/bin/echo"},
	}

	if hk := selectHookConfig(cfg, "Claude can you hear me"); hk == nil {
		t.Fatalf("expected hook match for Claude")
	}
	if hk := selectHookConfig(cfg, "hello alpha"); hk == nil {
		t.Fatalf("expected hook match for alpha")
	}
	if hk := selectHookConfig(cfg, "no wake here"); hk != &cfg.Hooks[0] {
		t.Fatalf("expected fallback to first hook")
	}
}

func TestWakeMatchesAliases(t *testing.T) {
	if !wakeMatches("hi Claude", "clawd", []string{"claude", "cloud"}) {
		t.Fatalf("expected alias match")
	}
	if wakeMatches("hi there", "clawd", []string{"claude"}) {
		t.Fatalf("expected no match")
	}
}
