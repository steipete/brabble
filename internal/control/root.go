package control

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"brabble/internal/config"
	"brabble/internal/doctor"
	"brabble/internal/hook"
	"brabble/internal/logging"

	"github.com/spf13/cobra"
)

// NewStatusCmd queries daemon status.
func NewStatusCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			conn, err := net.Dial("unix", cfg.Paths.SocketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer conn.Close()
			req := Request{Op: "status"}
			if err := json.NewEncoder(conn).Encode(req); err != nil {
				return err
			}
			var status Status
			if err := json.NewDecoder(conn).Decode(&status); err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			fmt.Printf("running: %v\nuptime: %.1fs\n", status.Running, status.UptimeSec)
			for _, t := range status.Transcripts {
				fmt.Printf("%s  %s\n", t.Timestamp.Format("15:04:05"), t.Text)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON")
	return cmd
}

// NewTailLogCmd tails the main log file (simple last N lines).
func NewTailLogCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tail-log",
		Short: "Show last 50 log lines",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			return tailFile(cfg.Paths.LogPath, 50)
		},
	}
}

func tailFile(path string, n int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			fmt.Println(l)
		}
	}
	return nil
}

// NewTestHookCmd triggers hook manually.
func NewTestHookCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "test-hook \"some text\"",
		Short: "Send sample text through hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			logger, err := logging.Configure(cfg)
			if err != nil {
				return err
			}
			r := hook.NewRunner(cfg, logger)
			job := hook.Job{Text: args[0], Timestamp: time.Now()}
			return r.Run(cmd.Context(), job)
		},
	}
}

// NewDoctorCmd runs environment checks.
func NewDoctorCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check dependencies and config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			results := doctor.Run(cfg)
			exitCode := 0
			for _, r := range results {
				status := "ok"
				if !r.Pass {
					status = "fail"
					exitCode = 1
				}
				fmt.Printf("%-12s %-4s %s\n", r.Name, status, r.Detail)
			}
			if exitCode != 0 {
				return fmt.Errorf("doctor found issues")
			}
			return nil
		},
	}
}
