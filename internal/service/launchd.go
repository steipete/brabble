// Package service provides launchd plist generation for macOS.
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const launchdTemplate = `<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>{{.Label}}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.Binary}}</string>
    <string>start</string>
    <string>--config</string>
    <string>{{.Config}}</string>
    <string>--foreground</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>StandardOutPath</key><string>{{.Log}}</string>
  <key>StandardErrorPath</key><string>{{.Log}}</string>
  {{- if .Env }}
  <key>EnvironmentVariables</key>
  <dict>
    {{- range $k, $v := .Env }}
    <key>{{$k}}</key><string>{{$v}}</string>
    {{- end }}
  </dict>
  {{- end }}
</dict>
</plist>`

type LaunchdParams struct {
	Label  string
	Binary string
	Config string
	Log    string
	Env    map[string]string
}

// LaunchdPath returns the plist path for a label.
func LaunchdPath(label string) string {
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", fmt.Sprintf("%s.plist", label))
}

// WritePlist writes a user-level launchd plist.
func WritePlist(params LaunchdParams) (string, error) {
	if err := os.MkdirAll(filepath.Dir(params.Config), 0o755); err != nil {
		return "", err
	}
	plistDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(plistDir, fmt.Sprintf("%s.plist", params.Label))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	tpl := template.Must(template.New("launchd").Parse(launchdTemplate))
	if err := tpl.Execute(f, params); err != nil {
		return "", err
	}
	return path, nil
}
