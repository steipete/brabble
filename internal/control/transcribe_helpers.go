package control

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-audio/wav"
)

func removeWakeWordLocal(text, word string) string {
	lw := strings.ToLower(word)
	fields := strings.Fields(text)
	out := make([]string, 0, len(fields))
	skipped := false
	for _, f := range fields {
		if !skipped && strings.EqualFold(strings.Trim(f, " ,.!?;:\"'"), lw) {
			skipped = true
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
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
