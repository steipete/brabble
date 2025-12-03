package control

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"brabble/internal/config"

	"github.com/spf13/cobra"
)

// NewSetupCmd downloads the default model if missing.
func NewSetupCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Download default whisper model if missing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			modelPath := os.ExpandEnv(cfg.ASR.ModelPath)
			if _, err := os.Stat(modelPath); err == nil {
				fmt.Println("model already present at", modelPath)
				return nil
			}
			url := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_1.bin"
			if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
				return err
			}
			fmt.Printf("downloading model to %s\n", modelPath)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return fmt.Errorf("download failed: %s", resp.Status)
			}
			out, err := os.Create(modelPath)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, resp.Body); err != nil {
				return err
			}
			fmt.Println("model download complete")
			return nil
		},
	}
}
