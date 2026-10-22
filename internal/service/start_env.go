package service

import "os"

// StartChildEnv returns nil when no override is set, so the child inherits
// the parent environment. A non-nil Env replaces the environment entirely.
func StartChildEnv(noWake bool, metricsAddr string) []string {
	if !noWake && metricsAddr == "" {
		return nil
	}
	env := os.Environ()
	if noWake {
		env = append(env, "BRABBLE_WAKE_ENABLED=0")
	}
	if metricsAddr != "" {
		env = append(env, "BRABBLE_METRICS_ADDR="+metricsAddr)
	}
	return env
}
