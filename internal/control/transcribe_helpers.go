package control

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-audio/wav"
)

func removeWakeWordLocal(text, word string, aliases []string) string {
	variants := wakeWordVariants(word, aliases)
	fields := strings.Fields(text)
	out := make([]string, 0, len(fields))
	skipped := false
	for _, f := range fields {
		if !skipped && fieldIsWake(f, variants) {
			skipped = true
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

func gateWakeText(txt, word string, aliases []string, enabled, noWake bool) (string, error) {
	if enabled && !noWake && !textHasWake(txt, word, aliases) {
		return "", fmt.Errorf("wake word %q not found; use --no-wake to override", word)
	}
	if enabled && !noWake {
		txt = removeWakeWordLocal(txt, word, aliases)
	}
	return txt, nil
}

func textHasWake(text, word string, aliases []string) bool {
	lower := strings.ToLower(text)
	for _, variant := range wakeWordVariants(word, aliases) {
		if strings.Contains(lower, variant) {
			return true
		}
	}
	return false
}

func wakeWordVariants(word string, aliases []string) []string {
	variants := []string{strings.ToLower(word)}
	for _, alias := range aliases {
		alias = strings.ToLower(strings.TrimSpace(alias))
		if alias == "" {
			continue
		}
		variants = append(variants, alias)
	}
	return variants
}

func fieldIsWake(field string, variants []string) bool {
	trimmed := strings.Trim(field, " ,.!?;:\"'")
	for _, variant := range variants {
		if strings.EqualFold(trimmed, variant) {
			return true
		}
	}
	return false
}

func resampleLinear(in []float32, srcSR, dstSR int) []float32 {
	if srcSR == dstSR || len(in) == 0 {
		out := make([]float32, len(in))
		copy(out, in)
		return out
	}
	ratio := float64(dstSR) / float64(srcSR)
	outLen := int(float64(len(in))*ratio + 0.9999)
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

func readWAV16kMono(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
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
	bitDepth := buf.SourceBitDepth
	scale, err := wavFullScale(bitDepth)
	if err != nil {
		return nil, err
	}
	frames := len(buf.Data) / ch
	mono := make([]float32, frames)
	for i := 0; i < frames; i++ {
		var sum int64
		for c := 0; c < ch; c++ {
			sample := int64(buf.Data[i*ch+c])
			// 8-bit WAV PCM is unsigned, with 128 at silence.
			if bitDepth == 8 {
				sample -= 128
			}
			sum += sample
		}
		mono[i] = float32(sum) / float32(ch) / scale
	}
	const targetSR = 16000
	if srcSR == targetSR {
		return mono, nil
	}
	return resampleLinear(mono, srcSR, targetSR), nil
}

func wavFullScale(bitDepth int) (float32, error) {
	switch bitDepth {
	case 8, 16, 24, 32:
		return float32(uint64(1) << uint(bitDepth-1)), nil
	default:
		return 0, fmt.Errorf("unsupported wav bit depth %d", bitDepth)
	}
}
