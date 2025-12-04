package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"brabble/internal/config"
	"brabble/internal/hook"
	"brabble/internal/logging"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewTranscribeCmd transcribes a WAV file using whisper and optionally fires the hook.
func NewTranscribeCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe <wavfile>",
		Short: "Transcribe a WAV file",
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
			file := args[0]
			wantHook, _ := cmd.Flags().GetBool("hook")
			noWake, _ := cmd.Flags().GetBool("no-wake")

			samples, err := readWAV16kMono(file)
			if err != nil {
				return err
			}

			txt, err := runWhisperOnce(cfg, logger, samples)
			if err != nil {
				return err
			}
			txt = strings.TrimSpace(txt)
			rawTxt := txt
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), txt)

			if !wantHook {
				return nil
			}

			// Apply wake/min_chars gating like daemon.
			if cfg.Wake.Enabled && !noWake && !strings.Contains(strings.ToLower(txt), strings.ToLower(cfg.Wake.Word)) {
				return fmt.Errorf("wake word %q not found; use --no-wake to override", cfg.Wake.Word)
			}
			if cfg.Wake.Enabled && !noWake {
				txt = removeWakeWordLocal(txt, cfg.Wake.Word)
			}
			hk := hook.SelectHookConfig(cfg, rawTxt)
			if hk == nil {
				return fmt.Errorf("no hook configured; add [[hooks]] entries")
			}
			if hk.MinChars > 0 && len(txt) < hk.MinChars {
				return fmt.Errorf("skipped: len(text)=%d < min_chars=%d", len(txt), hk.MinChars)
			}

			r := hook.NewRunner(cfg, logger)
			r.SelectHook(hk)
			if !r.ShouldRun() {
				return fmt.Errorf("hook on cooldown")
			}
			return r.Run(context.Background(), hook.Job{Text: txt, Timestamp: time.Now()})
		},
	}
	cmd.Flags().Bool("hook", false, "also send through configured hook")
	cmd.Flags().Bool("no-wake", false, "ignore wake word requirement for this file")
	return cmd
}

func runWhisperOnce(cfg *config.Config, logger *logrus.Logger, samples []float32) (string, error) {
	model, err := whisper.New(cfg.ASR.ModelPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = model.Close() }()
	ctx, err := model.NewContext()
	if err != nil {
		return "", err
	}
	if lang := strings.TrimSpace(cfg.ASR.Language); lang != "" {
		_ = ctx.SetLanguage(lang)
	}
	if err := ctx.Process(samples, nil, nil, nil); err != nil {
		return "", err
	}
	var b strings.Builder
	for {
		seg, err := ctx.NextSegment()
		if err != nil {
			break
		}
		b.WriteString(seg.Text)
		if !strings.HasSuffix(seg.Text, " ") {
			b.WriteByte(' ')
		}
	}
	return b.String(), nil
}
