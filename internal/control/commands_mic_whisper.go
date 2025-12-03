//go:build whisper

package control

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/gordonklaus/portaudio"
	"github.com/spf13/cobra"
)

// NewListMicsCmd lists available input devices (whisper build only).
func NewListMicsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-mics",
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
