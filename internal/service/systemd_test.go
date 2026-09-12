package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSystemdUnit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := WriteSystemdUnit(LaunchdParams{Binary: "/tmp/voice app", Config: "/tmp/a%b.toml", Env: map[string]string{"TOKEN": "a%b"}})
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "systemd/user/brabble.service") {
		t.Fatal(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`ExecStart="/tmp/voice app" start --foreground --config "/tmp/a%%b.toml"`, `Environment="TOKEN=a%%b"`, `WantedBy=default.target`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s in %s", want, data)
		}
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("unit permissions", info.Mode())
	}
	if _, err := WriteSystemdUnit(LaunchdParams{Env: map[string]string{"BAD=NAME": "x"}}); err == nil {
		t.Fatal("bad environment name accepted")
	}
}
