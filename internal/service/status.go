package service

import (
	"os"
	"path/filepath"
)

// Status returns whether the plist exists.
func Status(label string) (string, bool) {
	plist := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", label+".plist")
	if _, err := os.Stat(plist); err == nil {
		return plist, true
	}
	return plist, false
}
