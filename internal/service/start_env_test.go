package service

import (
	"os"
	"os/exec"
	"testing"
)

func TestStartChildEnv(t *testing.T) {
	t.Setenv("HOME", "/synthetic/brabble-home")
	t.Setenv("PATH", "/synthetic/brabble-bin")
	t.Setenv("BRABBLE_WAKE_ENABLED", "1")
	t.Setenv("BRABBLE_METRICS_ADDR", "127.0.0.1:9000")
	for _, tc := range []struct {
		name    string
		noWake  bool
		metrics string
	}{
		{name: "inherit"},
		{name: "wake", noWake: true},
		{name: "metrics", metrics: "127.0.0.1:9317"},
		{name: "both", noWake: true, metrics: "127.0.0.1:9317"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := exec.Command(os.Args[0], "-test.run=^TestStartChildEnvProcess$")
			child.Env = StartChildEnv(tc.noWake, tc.metrics)
			if !tc.noWake && tc.metrics == "" && child.Env != nil {
				t.Fatal("no overrides should inherit the environment")
			}
			if child.Env == nil {
				child.Env = os.Environ()
			}
			wake := "1"
			if tc.noWake {
				wake = "0"
			}
			metrics := "127.0.0.1:9000"
			if tc.metrics != "" {
				metrics = tc.metrics
			}
			child.Env = append(child.Env, "BRABBLE_TEST_CHILD=1", "BRABBLE_TEST_WAKE="+wake, "BRABBLE_TEST_METRICS="+metrics)
			if output, err := child.CombinedOutput(); err != nil {
				t.Fatalf("child environment check failed: %v\n%s", err, output)
			}
		})
	}
}

func TestStartChildEnvProcess(t *testing.T) {
	if os.Getenv("BRABBLE_TEST_CHILD") != "1" {
		return
	}
	for key, want := range map[string]string{
		"HOME":                 "/synthetic/brabble-home",
		"PATH":                 "/synthetic/brabble-bin",
		"BRABBLE_WAKE_ENABLED": os.Getenv("BRABBLE_TEST_WAKE"),
		"BRABBLE_METRICS_ADDR": os.Getenv("BRABBLE_TEST_METRICS"),
	} {
		if os.Getenv(key) != want {
			t.Errorf("%s was not preserved or overridden correctly", key)
		}
	}
}
