package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// SystemdPath returns the user unit path under the platform config directory.
func SystemdPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "systemd", "user", "brabble.service"), nil
}

func systemdQuote(value string) string {
	return strconv.Quote(strings.ReplaceAll(value, "%", "%%"))
}

// WriteSystemdUnit installs the unit without starting microphone capture.
func WriteSystemdUnit(params LaunchdParams) (string, error) {
	path, err := SystemdPath()
	if err != nil {
		return "", err
	}
	var content strings.Builder
	content.WriteString("[Unit]\nDescription=Brabble local voice assistant\n\n[Service]\nType=simple\n")
	fmt.Fprintf(&content, "ExecStart=%s start --foreground --config %s\n", systemdQuote(strings.ReplaceAll(params.Binary, "$", "$$")), systemdQuote(strings.ReplaceAll(params.Config, "$", "$$")))
	keys := make([]string, 0, len(params.Env))
	for k := range params.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(k) {
			return "", fmt.Errorf("invalid environment name %q", k)
		}
		fmt.Fprintf(&content, "Environment=%s\n", systemdQuote(k+"="+params.Env[k]))
	}
	content.WriteString("Restart=on-failure\nRestartSec=3\n\n[Install]\nWantedBy=default.target\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content.String()), 0600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0600); err != nil {
		return "", err
	}
	return path, nil
}
