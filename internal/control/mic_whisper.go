//go:build whisper

package control

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/gordonklaus/portaudio"
	"github.com/spf13/cobra"
)

// NewMicCmd groups mic subcommands (whisper build).
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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available microphones",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOut, _ := cmd.Flags().GetBool("json")
			if err := portaudio.Initialize(); err != nil {
				return fmt.Errorf("portaudio init: %w", err)
			}
			defer portaudio.Terminate()

			devs, err := portaudio.Devices()
			if err != nil {
				return err
			}
			type mic struct {
				Index     int     `json:"index"`
				Name      string  `json:"name"`
				Channels  int     `json:"channels"`
				LatencyMs float64 `json:"latency_ms"`
				Default   bool    `json:"default"`
			}
			def := portaudio.DefaultInputDevice()
			out := []mic{}
			for i, d := range devs {
				if d.MaxInputChannels < 1 {
					continue
				}
				out = append(out, mic{
					Index:     i,
					Name:      d.Name,
					Channels:  d.MaxInputChannels,
					LatencyMs: d.DefaultLowInputLatency.Seconds() * 1000,
					Default:   def != nil && d.Name == def.Name,
				})
			}
			if jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			}
			for _, m := range out {
				defMark := ""
				if m.Default {
					defMark = " (default)"
				}
				fmt.Printf("[%d] %s%s (in %d ch, latency %.2fms)\n", m.Index, m.Name, defMark, m.Channels, m.LatencyMs)
			}
			if runtime.GOOS == "darwin" {
				fmt.Println("tip: if no devices appear, install PortAudio: brew install portaudio")
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON")
	return cmd
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
			fmt.Printf("mic set to %q in %s\n", args[0], cfg.Paths.ConfigPath)
			return nil
		},
	}
}
