package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"brabble/internal/config"
)

func TestWaitForShutdownSucceedsWhenPidFileRemoved(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Default()
	cfg.Paths.ConfigPath = dir + "/config.toml"
	cfg.Paths.PidPath = dir + "/brabble.pid"
	if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	if err := os.WriteFile(cfg.Paths.PidPath, []byte("12345"), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.Remove(cfg.Paths.PidPath)
	}()
	if err := waitForShutdown(cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForShutdownTimesOutOnAlivePid(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := config.Default()
	cfg.Paths.ConfigPath = dir + "/config.toml"
	cfg.Paths.PidPath = dir + "/brabble.pid"
	if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	selfPid := os.Getpid()
	if err := os.WriteFile(cfg.Paths.PidPath, []byte(fmt.Sprintf("%d", selfPid)), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	cfg.Daemon.StopTimeoutSec = 0.3
	if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
		t.Fatalf("save cfg: %v", err)
	}
	if err := waitForShutdown(cfg.Paths.ConfigPath); err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestWaitForDaemonReadyErrorsWhenPidFileMissing(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() { stopCmd(cmd) })

	err := waitForDaemonReady(t.TempDir()+"/missing.pid", cmd.Process, 2, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "pid file") {
		t.Fatalf("expected missing pid file error, got %v", err)
	}
}

func TestWaitForDaemonReadyErrorsWhenChildExited(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() { stopCmd(cmd) })
	waitUntilChildGone(t, cmd.Process.Pid)

	err := waitForDaemonReady(t.TempDir()+"/missing.pid", cmd.Process, 2, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("expected child exit status 7, got %v", err)
	}
}

func TestWaitForDaemonReadyErrorsWhenChildExitedAfterPidFile(t *testing.T) {
	dir := t.TempDir()
	pidPath := dir + "/brabble.pid"
	if err := os.WriteFile(pidPath, []byte("1\n"), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() { stopCmd(cmd) })
	waitUntilChildGone(t, cmd.Process.Pid)

	err := waitForDaemonReady(pidPath, cmd.Process, 2, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("expected child exit status 7, got %v", err)
	}
}

func TestWaitForDaemonReadySucceedsWhenRunningChildWritesPid(t *testing.T) {
	dir := t.TempDir()
	pidPath := dir + "/brabble.pid"
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() { stopCmd(cmd) })
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}

	if err := waitForDaemonReady(pidPath, cmd.Process, 2, time.Millisecond); err != nil {
		t.Fatalf("expected ready daemon, got %v", err)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("ready wait reaped a live child: %v", err)
	}
}

func TestStartReadyWaitIsTwoSeconds(t *testing.T) {
	if startReadyAttempts*startReadyInterval != 2*time.Second {
		t.Fatalf("start wait = %s, want 2s", time.Duration(startReadyAttempts)*startReadyInterval)
	}
}

func stopCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func waitUntilChildGone(t *testing.T, pid int) {
	t.Helper()
	for i := 0; i < 200; i++ {
		out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
		stat := strings.TrimSpace(string(out))
		if err != nil || stat == "" || strings.Contains(stat, "Z") {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("child %d did not exit", pid)
}
