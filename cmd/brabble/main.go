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

Key commands:
  start|stop|restart        Daemon lifecycle
  status --json             Uptime + last transcripts
  list-mics / set-mic       Select input device (whisper build)
  doctor                    Check deps/model/hook/portaudio
  setup                     Download default whisper model
  models list|download|set  Manage whisper.cpp models
  install-service           Write launchd plist (macOS)
  reload                    Reload hook/wake config live
  health                    Control-socket liveness ping

Notable flags/env:
  --metrics-addr <addr>     Enable /metrics (Prometheus text)
  --no-wake                 Disable wake word requirement
  Env overrides: BRABBLE_WAKE_ENABLED, BRABBLE_METRICS_ADDR,
                 BRABBLE_LOG_LEVEL/FORMAT, BRABBLE_TRANSCRIPTS_ENABLED,
                 BRABBLE_REDACT_PII`,
		Example: `  brabble start --metrics-addr 127.0.0.1:9317
  brabble list-mics
  brabble models download ggml-medium-q5_1.bin
  brabble models set ggml-medium-q5_1.bin
  brabble install-service --env BRABBLE_METRICS_ADDR=127.0.0.1:9317
  brabble reload
  brabble health`,
		DisableFlagsInUseLine: true,
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
	root.AddCommand(control.NewSetupCmd(cfgPath))
	root.AddCommand(control.NewHealthCmd(cfgPath))
	root.AddCommand(control.NewModelsCmd(cfgPath))

	// Hidden internal serve command used by start.
	root.AddCommand(daemon.NewServeCmd(cfgPath))

	applyColorHelp(root)

	if err := root.Execute(); err != nil {
		return err
	}
	return nil
}

func applyColorHelp(root *cobra.Command) {
	const (
		boldBlue = "\033[1;34m"
		bold     = "\033[1m"
		dim      = "\033[2m"
		reset    = "\033[0m"
	)
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "%sBrabble%s — local wake-word voice hook daemon\n", boldBlue, reset)
		fmt.Fprintf(out, "%sBuilds (if needed), listens on mic, transcribes locally, and runs your hook.%s\n\n", dim, reset)

		fmt.Fprintf(out, "%sUsage%s\n", bold, reset)
		fmt.Fprintf(out, "  brabble [command] [flags]\n\n")

		fmt.Fprintf(out, "%sKey commands%s\n", bold, reset)
		fmt.Fprintln(out, "  start|stop|restart          daemon lifecycle")
		fmt.Fprintln(out, "  status [--json]             uptime + last transcripts")
		fmt.Fprintln(out, "  list-mics / set-mic         select input device (whisper build)")
		fmt.Fprintln(out, "  doctor                      check deps/model/hook/portaudio")
		fmt.Fprintln(out, "  setup                       download default whisper model")
		fmt.Fprintln(out, "  models list|download|set    manage whisper.cpp models")
		fmt.Fprintln(out, "  service install|uninstall   manage launchd plist (macOS)")
		fmt.Fprintln(out, "  health                      control-socket liveness ping")
		fmt.Fprintln(out, "  tail-log                    show last log lines")
		fmt.Fprintln(out, "  test-hook \"text\"           invoke hook manually")
		fmt.Fprintln(out, "")

		fmt.Fprintf(out, "%sNotable flags & env%s\n", bold, reset)
		fmt.Fprintln(out, "  --metrics-addr <addr>   enable /metrics (Prometheus)")
		fmt.Fprintln(out, "  --no-wake               disable wake word requirement")
		fmt.Fprintln(out, "  -c, --config <path>     config file (default ~/.config/brabble/config.toml)")
		fmt.Fprintln(out, "  Env: BRABBLE_WAKE_ENABLED=0, BRABBLE_METRICS_ADDR=host:port,")
		fmt.Fprintln(out, "       BRABBLE_LOG_LEVEL=debug, BRABBLE_LOG_FORMAT=json,")
		fmt.Fprintln(out, "       BRABBLE_TRANSCRIPTS_ENABLED=0, BRABBLE_REDACT_PII=1")
		fmt.Fprintln(out, "")

		fmt.Fprintf(out, "%sExamples%s\n", bold, reset)
		fmt.Fprintln(out, "  brabble start --metrics-addr 127.0.0.1:9317")
		fmt.Fprintln(out, "  brabble list-mics")
		fmt.Fprintln(out, "  brabble models download ggml-medium-q5_1.bin")
		fmt.Fprintln(out, "  brabble models set ggml-medium-q5_1.bin")
		fmt.Fprintln(out, "  brabble install-service --env BRABBLE_METRICS_ADDR=127.0.0.1:9317")
		fmt.Fprintln(out, "  brabble reload")
		fmt.Fprintln(out, "  brabble health")
		fmt.Fprintln(out, "")

		fmt.Fprintf(out, "%sCommands%s\n", bold, reset)
		for _, c := range cmd.Commands() {
			if c.Hidden {
				continue
			}
			fmt.Fprintf(out, "  %-15s %s\n", c.Name(), c.Short)
		}
	})
}
