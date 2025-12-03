package main

import (
	"fmt"
	"os"

	"brabble/internal/control"
	"brabble/internal/daemon"

	"github.com/spf13/cobra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := &cobra.Command{
		Use:   "brabble",
		Short: "Brabble — local wake-word voice hook daemon",
		Long: `Brabble listens on your mic, waits for a wake word ("clawd"), transcribes locally with whisper.cpp,
and fires a configurable hook (default: ../warelay send "Voice brabble from ${hostname}: <text>").

Common tasks:
  - start/stop/restart the daemon
  - list/set microphones (whisper build)
  - download/set whisper models (models list|download|set)
  - setup (download default model), doctor (check deps), install-service (launchd)
  - reload hook/wake config, health check, tail logs/status`,
	}

	cfgPath := root.PersistentFlags().StringP("config", "c", "", "Path to config file (TOML). Defaults to ~/.config/brabble/config.toml")

	root.AddCommand(daemon.NewStartCmd(cfgPath))
	root.AddCommand(daemon.NewStopCmd(cfgPath))
	root.AddCommand(daemon.NewRestartCmd(cfgPath))
	root.AddCommand(control.NewStatusCmd(cfgPath))
	root.AddCommand(control.NewTailLogCmd(cfgPath))
	root.AddCommand(control.NewListMicsCmd())
	root.AddCommand(control.NewSetMicCmd(cfgPath))
	root.AddCommand(control.NewTestHookCmd(cfgPath))
	root.AddCommand(control.NewDoctorCmd(cfgPath))
	root.AddCommand(control.NewServiceCmd(cfgPath))
	root.AddCommand(control.NewUninstallServiceCmd())
	root.AddCommand(control.NewSetupCmd(cfgPath))
	root.AddCommand(control.NewReloadCmd(cfgPath))
	root.AddCommand(control.NewHealthCmd(cfgPath))
	root.AddCommand(control.NewModelsCmd(cfgPath))

	// Hidden internal serve command used by start.
	root.AddCommand(daemon.NewServeCmd(cfgPath))

	if err := root.Execute(); err != nil {
		return err
	}
	return nil
}
