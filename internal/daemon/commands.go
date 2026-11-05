package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"brabble/internal/config"
	"brabble/internal/logging"
	"brabble/internal/run"
	"brabble/internal/service"

	"github.com/spf13/cobra"
)

// NewStartCmd starts the daemon (background).
func NewStartCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start brabble daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			if err := ensureNotRunning(cfg); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(cfg.Paths.PidPath), 0o755); err != nil {
				return err
			}
			self, err := os.Executable()
			if err != nil {
				return err
			}
			child := exec.Command(self, "serve", "--config", cfg.Paths.ConfigPath)
			child.Env = service.StartChildEnv(cmd.Flag("no-wake").Changed, cmd.Flag("metrics-addr").Value.String())
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			if err := child.Start(); err != nil {
				return err
			}
			// Wait a moment and confirm the child stayed up and wrote its pid file.
			if err := waitForDaemonReady(cfg.Paths.PidPath, child.Process, startReadyAttempts, startReadyInterval); err != nil {
				return err
			}
			fmt.Printf("brabble started (pid %d)\n", child.Process.Pid)
			return nil
		},
	}
	cmd.Flags().Bool("no-wake", false, "disable wake word requirement for this run")
	cmd.Flags().String("metrics-addr", "", "enable metrics at address (e.g., 127.0.0.1:9317) for this run")
	return cmd
}

// NewServeCmd runs the daemon foreground (internal).
func NewServeCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "serve",
		Short:  "Run brabble daemon (internal)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag("no-wake").Changed {
				if err := os.Setenv("BRABBLE_WAKE_ENABLED", "0"); err != nil {
					return fmt.Errorf("set BRABBLE_WAKE_ENABLED: %w", err)
				}
			}
			if addr := cmd.Flag("metrics-addr").Value.String(); addr != "" {
				if err := os.Setenv("BRABBLE_METRICS_ADDR", addr); err != nil {
					return fmt.Errorf("set BRABBLE_METRICS_ADDR: %w", err)
				}
			}
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			logger, err := logging.Configure(cfg)
			if err != nil {
				return err
			}
			return run.Serve(cfg, logger)
		},
	}
	cmd.Flags().Bool("no-wake", false, "disable wake word requirement")
	cmd.Flags().String("metrics-addr", "", "enable metrics at address (e.g., 127.0.0.1:9317)")
	return cmd
}

// NewStopCmd stops the daemon.
func NewStopCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop brabble daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			pid, err := readPID(cfg.Paths.PidPath)
			if err != nil {
				return err
			}
			proc, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return err
			}
			fmt.Println("stop signal sent")
			return nil
		},
	}
}

// NewRestartCmd stops then starts.
func NewRestartCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart brabble daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopCmd := NewStopCmd(cfgPath)
			_ = stopCmd.RunE(stopCmd, args) // ignore error if not running

			if err := waitForShutdown(*cfgPath); err != nil {
				return err
			}

			startCmd := NewStartCmd(cfgPath)
			return startCmd.RunE(startCmd, args)
		},
	}
}

const (
	startReadyAttempts = 20
	startReadyInterval = 100 * time.Millisecond
)

// waitForDaemonReady reports an error when proc has exited or pidPath is still
// missing after attempts of interval. A live child is not reaped.
func waitForDaemonReady(pidPath string, proc *os.Process, attempts int, interval time.Duration) error {
	if proc == nil {
		return fmt.Errorf("brabble process missing")
	}
	if attempts < 1 {
		attempts = 1
	}
	for waited := 0; waited < attempts; waited++ {
		if err := readyOrWaiting(pidPath, proc); err == nil || !errors.Is(err, errDaemonNotReady) {
			return err
		}
		time.Sleep(interval)
	}
	if err := readyOrWaiting(pidPath, proc); err == nil || !errors.Is(err, errDaemonNotReady) {
		return err
	}
	return fmt.Errorf("brabble did not write pid file %s (pid %d still running)", pidPath, proc.Pid)
}

var errDaemonNotReady = errors.New("brabble pid file not written yet")

func readyOrWaiting(pidPath string, proc *os.Process) error {
	if err := exitedChild(proc); err != nil {
		return err
	}
	if _, err := os.Stat(pidPath); err != nil {
		return errDaemonNotReady
	}
	return exitedChild(proc)
}

func exitedChild(proc *os.Process) error {
	var ws syscall.WaitStatus
	var wpid int
	var err error
	for {
		wpid, err = syscall.Wait4(proc.Pid, &ws, syscall.WNOHANG, nil)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if errors.Is(err, syscall.ECHILD) {
		return fmt.Errorf("brabble exited before it became ready")
	}
	if err != nil {
		return fmt.Errorf("check brabble process: %w", err)
	}
	if wpid == 0 {
		return nil
	}
	if ws.Signaled() {
		return fmt.Errorf("brabble exited before it became ready (signal %s)", ws.Signal())
	}
	return fmt.Errorf("brabble exited before it became ready (exit status %d)", ws.ExitStatus())
}

func ensureNotRunning(cfg *config.Config) error {
	pid, err := readPID(cfg.Paths.PidPath)
	if err != nil {
		return nil
	}
	// Check if process alive.
	proc, err := os.FindProcess(pid)
	if err == nil {
		if err := proc.Signal(syscall.Signal(0)); err == nil {
			return fmt.Errorf("already running with pid %d", pid)
		}
	}
	return nil
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, err
	}
	return pid, nil
}

func waitForShutdown(cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	timeout := time.Duration(cfg.Daemon.StopTimeoutSec * float64(time.Second))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pid, err := readPID(cfg.Paths.PidPath)
		if err != nil {
			return nil // pid file gone
		}
		proc, _ := os.FindProcess(pid)
		if proc != nil {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				_ = os.Remove(cfg.Paths.PidPath)
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("restart: daemon did not stop within %s", timeout)
}
