package control

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"brabble/internal/config"

	"github.com/spf13/cobra"
)

// simple registry of known ggml models.
var modelRegistry = map[string]string{
	"ggml-small-q5_1.bin":          "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small-q5_1.bin",
	"ggml-medium-q5_1.bin":         "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_1.bin",
	"ggml-large-v3-q5_0.bin":       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-q5_0.bin",
	"ggml-large-v3-turbo-q8_0.bin": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q8_0.bin",
	"ggml-large-v3-turbo.bin":      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin",
}

// NewModelsCmd wires up the models subcommands (list/download/set).
func NewModelsCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List/download/set whisper models",
	}
	cmd.AddCommand(newModelsListCmd())
	cmd.AddCommand(newModelsDownloadCmd(cfgPath))
	cmd.AddCommand(newModelsSetCmd(cfgPath))
	return cmd
}

func newModelsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known models and those present locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			modelDir := filepath.Join(home, "Library", "Application Support", "brabble", "models")
			local := map[string]bool{}
			entries, _ := os.ReadDir(modelDir)
			for _, e := range entries {
				if !e.IsDir() {
					local[e.Name()] = true
				}
			}
			names := make([]string, 0, len(modelRegistry))
			for n := range modelRegistry {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				avail := ""
				if local[n] {
					avail = "(downloaded)"
				}
				fmt.Printf("- %s %s\n", n, avail)
			}
			return nil
		},
	}
}

func newModelsDownloadCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "download <model>",
		Short: "Download a model from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			url, ok := modelRegistry[name]
			if !ok {
				return fmt.Errorf("unknown model %q; run models list", name)
			}
			home, _ := os.UserHomeDir()
			dest := filepath.Join(home, "Library", "Application Support", "brabble", "models", name)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			fmt.Printf("downloading %s -> %s\n", name, dest)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != 200 {
				return fmt.Errorf("download failed: %s", resp.Status)
			}
			tmp := dest + ".part"
			out, err := os.Create(tmp)
			if err != nil {
				return err
			}
			defer func() { _ = out.Close() }()
			if _, err := io.Copy(out, resp.Body); err != nil {
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
			return os.Rename(tmp, dest)
		},
	}
}

func newModelsSetCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set <model-name-or-path>",
		Short: "Set asr.model_path in config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			val := args[0]
			home, _ := os.UserHomeDir()
			modelDir := filepath.Join(home, "Library", "Application Support", "brabble", "models")
			// if short name, resolve in modelDir
			if !strings.Contains(val, "/") {
				val = filepath.Join(modelDir, val)
			}
			cfg.ASR.ModelPath = val
			if err := config.Save(cfg, cfg.Paths.ConfigPath); err != nil {
				return err
			}
			fmt.Printf("model set to %s\n", val)
			return nil
		},
	}
}
