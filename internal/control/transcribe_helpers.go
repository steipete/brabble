package control

import (
	"strings"
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
