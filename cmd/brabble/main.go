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
		Short: "Brabble — always‑on voice hook daemon",
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

	// Hidden internal serve command used by start.
	root.AddCommand(daemon.NewServeCmd(cfgPath))

	if err := root.Execute(); err != nil {
		return err
	}
	return nil
}
