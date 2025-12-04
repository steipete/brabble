//go:build !whisper

package control

import "github.com/spf13/cobra"

func NewTranscribeCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "transcribe",
		Short: "Transcribe a WAV file (build with -tags whisper)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("build with '-tags whisper' to use transcribe")
			return nil
		},
	}
}
