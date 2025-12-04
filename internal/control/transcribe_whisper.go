//go:build whisper

package control

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"brabble/internal/config"
	"brabble/internal/hook"
	"brabble/internal/logging"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewTranscribeCmd transcribes a WAV file using whisper and optionally fires the hook.
func NewTranscribeCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe <wavfile>",
		Short: "Transcribe a WAV file (whisper build)",
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
			fmt.Fprintln(cmd.OutOrStdout(), txt)

			if !wantHook {
				return nil
			}

			// Apply wake/min_chars gating like daemon.
			if cfg.Wake.Enabled && !noWake && !strings.Contains(strings.ToLower(txt), strings.ToLower(cfg.Wake.Word)) {
				return fmt.Errorf("wake word %q not found; use --no-wake to override", cfg.Wake.Word)
			}
			if cfg.Wake.Enabled && !noWake {
				txt = removeWakeWord(txt, cfg.Wake.Word)
			}
			if len(txt) < cfg.Hook.MinChars {
				return fmt.Errorf("skipped: len(text)=%d < min_chars=%d", len(txt), cfg.Hook.MinChars)
			}

			r := hook.NewRunner(cfg, logger)
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

func readWAV16kMono(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := wav.NewDecoder(f)
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV: %s", path)
	}
	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, err
	}
	if buf == nil || len(buf.Data) == 0 {
		return nil, fmt.Errorf("empty audio")
	}
	srcSR := buf.Format.SampleRate
	ch := buf.Format.NumChannels
	if ch < 1 {
		return nil, fmt.Errorf("no channels in wav")
	}
	// To mono: average channels.
	frames := len(buf.Data) / ch
	mono := make([]float32, frames)
	for i := 0; i < frames; i++ {
		var sum int
		for c := 0; c < ch; c++ {
			sum += int(buf.Data[i*ch+c])
		}
		mono[i] = float32(sum) / float32(ch) / float32(1<<15)
	}

	const targetSR = 16000
	if srcSR == targetSR {
		return mono, nil
	}
	return resampleLinear(mono, srcSR, targetSR), nil
}

func resampleLinear(in []float32, srcSR, dstSR int) []float32 {
	if srcSR == dstSR || len(in) == 0 {
		out := make([]float32, len(in))
		copy(out, in)
		return out
	}
	ratio := float64(dstSR) / float64(srcSR)
	outLen := int(math.Ceil(float64(len(in)) * ratio))
	out := make([]float32, outLen)
	for i := 0; i < outLen; i++ {
		pos := float64(i) / ratio
		idx := int(pos)
		if idx >= len(in)-1 {
			out[i] = in[len(in)-1]
			continue
		}
		frac := float32(pos - float64(idx))
		out[i] = in[idx]*(1-frac) + in[idx+1]*frac
	}
	return out
}

func runWhisperOnce(cfg *config.Config, logger *logrus.Logger, samples []float32) (string, error) {
	model, err := whisper.New(cfg.ASR.ModelPath)
	if err != nil {
		return "", err
	}
	defer model.Close()
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
