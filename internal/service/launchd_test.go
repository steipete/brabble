package service

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePlistRestrictsPermissionsAndEscapesValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".config", "brabble", "config.toml")
	path, err := WritePlist(LaunchdParams{
		Label:  "com.brabble.agent",
		Binary: "/tmp/brabble&test",
		Config: configPath,
		Log:    filepath.Join(home, "brabble.log"),
		Env:    map[string]string{"TOKEN": "private&value"},
	})
	if err != nil {
		t.Fatalf("write plist: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat plist: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("plist mode = %o, want 600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plist: %v", err)
	}
	var document struct {
		XMLName xml.Name `xml:"plist"`
	}
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatalf("plist is not valid XML: %v", err)
	}
	if document.XMLName.Local != "plist" {
		t.Fatalf("plist root = %q, want plist", document.XMLName.Local)
	}
	if bytes.Contains(data, []byte("private&value")) || !bytes.Contains(data, []byte("private&amp;value")) {
		t.Fatalf("plist value was not XML-escaped: %s", data)
	}
	if bytes.Contains(data, []byte("--foreground")) {
		t.Fatalf("plist still passes --foreground: %s", data)
	}
	if !bytes.Contains(data, []byte("<string>serve</string>")) {
		t.Fatalf("plist = %s; want serve", data)
	}
}

func TestWritePlistRunsServe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, "brabble.toml")
	path, err := WritePlist(LaunchdParams{
		Label:  "com.example.brabble",
		Binary: "/tmp/brabble",
		Config: configPath,
		Log:    filepath.Join(home, "brabble.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "--foreground") {
		t.Fatalf("plist still passes --foreground:\n%s", text)
	}
	if !strings.Contains(text, "<string>serve</string>") {
		t.Fatalf("plist = %s; want serve", text)
	}
}
