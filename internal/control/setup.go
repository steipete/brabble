package control

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"brabble/internal/config"
	"brabble/internal/doctor"

	"github.com/spf13/cobra"
)

// NewSetupCmd downloads the default model if missing.
func NewSetupCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Download default whisper model if missing and set config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			name := "ggml-large-v3-turbo-q8_0.bin"
			url := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q8_0.bin"
			modelPath := os.ExpandEnv(filepath.Join(cfg.Paths.StateDir, "models", name))
			if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
				return err
			}
			if _, err := os.Stat(modelPath); err == nil {
				fmt.Println("model already present at", modelPath)
			} else {
				fmt.Printf("downloading model to %s\n", modelPath)
				resp, err := http.Get(url)
				if err != nil {
					return err
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode != 200 {
					return fmt.Errorf("download failed: %s", resp.Status)
				}
				tmp := modelPath + ".part"
				out, err := os.Create(tmp)
				if err != nil {
					return err
				}
				defer func() { _ = out.Close() }()
				if _, err := io.Copy(out, resp.Body); err != nil {
					_ = os.Remove(tmp)
					return err
				}
				if err := out.Close(); err != nil {
					_ = os.Remove(tmp)
					return err
				}
				if err := os.Rename(tmp, modelPath); err != nil {
					_ = os.Remove(tmp)
					return err
				}
				fmt.Println("model download complete")
			}
			cfg.ASR.ModelPath = modelPath
			if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
				return err
			}
			fmt.Println("config updated with model_path =", modelPath)
			results := doctor.Run(cfg)
			for _, r := range results {
				status := "ok"
				if !r.Pass {
					status = "fail"
				}
				fmt.Printf("%-12s %-4s %s\n", r.Name, status, r.Detail)
			}
			return nil
		},
	}
}
