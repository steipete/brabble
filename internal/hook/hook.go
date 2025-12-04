package hook

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"brabble/internal/config"
	"brabble/internal/logging"

	"github.com/google/shlex"
)

// Job represents a hook invocation request.
type Job struct {
	Text      string
	Timestamp time.Time
}

// Runner executes hooks with cooldown and prefix handling.
type Runner struct {
	cfg      *config.Config
	logger   *logging.Logger
	lastRun  time.Time
	mu       sync.Mutex
	hostname string

	activeHook *config.HookConfig
}

// NewRunner constructs a hook runner with hostname cached.
func NewRunner(cfg *config.Config, logger *logging.Logger) *Runner {
	host, _ := os.Hostname()
	return &Runner{
		cfg:      cfg,
		logger:   logger,
		hostname: host,
	}
}

// ShouldRun returns whether cooldown allows a new hook.
func (r *Runner) ShouldRun() bool {
	hk, last := r.state()
	if hk == nil {
		return false
	}
	if hk.CooldownSec <= 0 {
		return true
	}
	return time.Since(last).Seconds() >= hk.CooldownSec
}

// Run executes the configured command with text payload.
func (r *Runner) Run(ctx context.Context, job Job) error {
	hk := r.selectedHook()
	if hk == nil {
		return fmt.Errorf("no hook selected")
	}
	cmdStr := hk.Command
	if cmdStr == "" {
		return fmt.Errorf("hook command not configured")
	}
	args := append([]string{}, hk.Args...)

	prefix := strings.ReplaceAll(hk.Prefix, "${hostname}", r.hostname)
	text := job.Text
	if hk.RedactPII {
		text = redactPII(text)
	}
	payload := strings.TrimSpace(prefix + text)
	args = append(args, payload)

	runCtx := ctx
	var cancel context.CancelFunc
	if hk.TimeoutSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(float64(time.Second)*hk.TimeoutSec))
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, cmdStr, args...)
	cmd.Env = os.Environ()
	for k, v := range hk.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("BRABBLE_TEXT=%s", text))
	cmd.Env = append(cmd.Env, fmt.Sprintf("BRABBLE_PREFIX=%s", prefix))

	r.logger.Info("hook exec",
		"cmd", cmdStr,
		"args", args,
		"timeout_sec", hk.TimeoutSec,
		"env_extra", hk.Env,
		"redact_pii", hk.RedactPII,
	)

	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		r.logger.Infof("hook output: %s", strings.TrimSpace(string(out)))
	}
	if err != nil {
		return fmt.Errorf("hook failed: %w", err)
	}

	r.mu.Lock()
	r.lastRun = time.Now()
	r.mu.Unlock()
	return nil
}

// SelectHook sets the active hook (used by server dispatcher).
func (r *Runner) SelectHook(hk *config.HookConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.activeHook = hk
}

// ParseArgs allows Hook.Args to be configured as a single string.
func ParseArgs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	return shlex.Split(raw)
}

func (r *Runner) selectedHook() *config.HookConfig {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeHook != nil {
		return r.activeHook
	}
	return nil
}

func (r *Runner) state() (*config.HookConfig, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeHook, r.lastRun
}

var (
	emailRE = regexp.MustCompile(`[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}`)
	phoneRE = regexp.MustCompile(`\+?\d[\d\s\-\(\)]{6,}\d`)
)

func redactPII(s string) string {
	s = emailRE.ReplaceAllString(s, "[redacted-email]")
	s = phoneRE.ReplaceAllString(s, "[redacted-phone]")
	return s
}
