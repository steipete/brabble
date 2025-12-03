//go:build whisper

package control

import (
	"fmt"
	"runtime"

	"github.com/gordonklaus/portaudio"
	"github.com/spf13/cobra"
)

// NewListMicsCmd lists available input devices (whisper build only).
func NewListMicsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-mics",
		Short: "List available microphones",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := portaudio.Initialize(); err != nil {
				return fmt.Errorf("portaudio init: %w", err)
			}
			defer portaudio.Terminate()

			devs, err := portaudio.Devices()
			if err != nil {
				return err
			}
			for i, d := range devs {
				if d.MaxInputChannels < 1 {
					continue
				}
				fmt.Printf("[%d] %s (in %d ch, default latency %.2fms)\n", i, d.Name, d.MaxInputChannels, d.DefaultLowInputLatency.Seconds()*1000)
			}
			if def := portaudio.DefaultInputDevice(); def != nil {
				fmt.Printf("default input: %s\n", def.Name)
			}
			if runtime.GOOS == "darwin" {
				fmt.Println("tip: if no devices appear, install PortAudio: brew install portaudio")
			}
			return nil
		},
	}
}
