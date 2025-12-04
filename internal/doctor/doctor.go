package doctor

import (
	"os"
	"os/exec"
	"strings"

	"brabble/internal/config"
)

// Result represents a diagnostic check.
type Result struct {
	Name   string
	Pass   bool
	Detail string
}

// Run executes doctor checks.
func Run(cfg *config.Config) []Result {
	results := []Result{
		checkFile("config path", cfg.Paths.ConfigPath),
		checkFile("model file", cfg.ASR.ModelPath),
		checkHookExecutable(cfg.Hook.Command),
		checkPortAudioPkgConfig(),
	}
	results = append(results, checkPortAudio(false))
	return results
}

func checkFile(label, path string) Result {
	if path == "" {
		return Result{Name: label, Pass: false, Detail: "not set"}
	}
	if _, err := os.Stat(os.ExpandEnv(path)); err != nil {
		return Result{Name: label, Pass: false, Detail: err.Error()}
	}
	return Result{Name: label, Pass: true, Detail: path}
}

func checkHookExecutable(cmd string) Result {
	label := "hook.command"
	if cmd == "" {
		return Result{Name: label, Pass: false, Detail: "not set"}
	}
	path := os.ExpandEnv(cmd)
	// If contains a path separator, treat as explicit path.
	if strings.Contains(path, "/") || strings.Contains(path, "\\") {
		info, err := os.Stat(path)
		if err != nil {
			return Result{Name: label, Pass: false, Detail: err.Error()}
		}
		if info.IsDir() {
			return Result{Name: label, Pass: false, Detail: "is a directory; set hook.command to an executable file"}
		}
		if info.Mode().Perm()&0o111 == 0 {
			return Result{Name: label, Pass: false, Detail: "not executable; chmod +x or choose another command"}
		}
		return Result{Name: label, Pass: true, Detail: path}
	}
	// Else search PATH.
	resolved, err := exec.LookPath(path)
	if err != nil {
		return Result{Name: label, Pass: false, Detail: err.Error()}
	}
	return Result{Name: label, Pass: true, Detail: resolved}
}

func checkPortAudioPkgConfig() Result {
	pkg, err := exec.LookPath("pkg-config")
	if err != nil {
		return Result{Name: "pkg-config", Pass: false, Detail: "pkg-config not found (brew install pkg-config)"}
	}
	cmd := exec.Command(pkg, "--exists", "portaudio-2.0")
	if err := cmd.Run(); err != nil {
		return Result{Name: "portaudio", Pass: false, Detail: "portaudio-2.0 not found (brew install portaudio)"}
	}
	// Optional display version
	versionCmd := exec.Command(pkg, "--modversion", "portaudio-2.0")
	if out, err := versionCmd.Output(); err == nil {
		return Result{Name: "portaudio", Pass: true, Detail: strings.TrimSpace(string(out))}
	}
	return Result{Name: "portaudio", Pass: true, Detail: "found via pkg-config"}
}
