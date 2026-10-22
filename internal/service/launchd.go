// Package service provides launchd plist generation for macOS.
package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const launchdTemplate = `<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>{{xml .Label}}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{xml .Binary}}</string>
    <string>serve</string>
    <string>--config</string>
    <string>{{xml .Config}}</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>StandardOutPath</key><string>{{xml .Log}}</string>
  <key>StandardErrorPath</key><string>{{xml .Log}}</string>
  {{- if .Env }}
  <key>EnvironmentVariables</key>
  <dict>
    {{- range $k, $v := .Env }}
    <key>{{xml $k}}</key><string>{{xml $v}}</string>
    {{- end }}
  </dict>
  {{- end }}
</dict>
</plist>`

// LaunchdParams contains values used to render a launchd plist.
type LaunchdParams struct {
	Label  string
	Binary string
	Config string
	Log    string
	Env    map[string]string
}

func escapeXML(value string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

// LaunchdPath returns the plist path for a label.
func LaunchdPath(label string) string {
	return filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", fmt.Sprintf("%s.plist", label))
}

// WritePlist writes a user-level launchd plist.
func WritePlist(params LaunchdParams) (path string, err error) {
	if err := os.MkdirAll(filepath.Dir(params.Config), 0o755); err != nil {
		return "", err
	}
	plistDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		return "", err
	}
	path = filepath.Join(plistDir, fmt.Sprintf("%s.plist", params.Label))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	tpl := template.Must(template.New("launchd").Funcs(template.FuncMap{"xml": escapeXML}).Parse(launchdTemplate))
	if err := tpl.Execute(f, params); err != nil {
		return "", err
	}
	return path, err
}
