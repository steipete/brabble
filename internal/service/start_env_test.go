package service

import "testing"

func TestStartChildEnvKeepsParentHome(t *testing.T) {
	t.Setenv("HOME", "/tmp/brabble-home")
	env := StartChildEnv(true, "127.0.0.1:9317")
	if !hasEnv(env, "HOME=/tmp/brabble-home") {
		t.Fatalf("env = %v; want parent HOME", env)
	}
	if !hasEnv(env, "BRABBLE_WAKE_ENABLED=0") {
		t.Fatalf("env = %v; want wake override", env)
	}
	if !hasEnv(env, "BRABBLE_METRICS_ADDR=127.0.0.1:9317") {
		t.Fatalf("env = %v; want metrics override", env)
	}
	if got := StartChildEnv(false, ""); got != nil {
		t.Fatalf("env = %v; want nil so the child inherits", got)
	}
}

func hasEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
