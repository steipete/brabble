package control

import (
	"brabble/internal/config"

	"github.com/spf13/cobra"
)

// NewMicCmd groups mic subcommands.
func NewMicCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mic",
		Short: "Microphone management",
	}
	cmd.AddCommand(newMicListCmd())
	cmd.AddCommand(newMicSetCmd(cfgPath))
	return cmd
}

func newMicListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available microphones",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("build with '-tags whisper' to enable microphone listing (PortAudio required)")
			return nil
		},
	}
}

func newMicSetCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set microphone device name in config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			cfg.Audio.DeviceName = args[0]
			if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
				return err
			}
			cmd.Printf("mic set to %q in %s\n", args[0], cfg.Paths.ConfigPath)
			return nil
		},
	}
}
