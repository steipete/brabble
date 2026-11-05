package control

import (
	"os"
	"strings"
	"testing"

	"brabble/internal/config"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

func TestRemoveWakeWordLocal(t *testing.T) {
	cases := []struct {
		text   string
		word   string
		expect string
	}{
		{"clawd make it so", "clawd", "make it so"},
		{"hey ClAwD computer", "clawd", "hey computer"},
		{"clawd, launch torpedo", "clawd", "launch torpedo"},
		{"we said clawd twice clawd", "clawd", "we said twice clawd"},
		{"no wake here", "clawd", "no wake here"},
	}
	for _, c := range cases {
		if got := removeWakeWordLocal(c.text, c.word, nil); got != c.expect {
			t.Fatalf("removeWakeWordLocal(%q)=%q want %q", c.text, got, c.expect)
		}
	}
}

func TestGateWakeTextUsesAliases(t *testing.T) {
	cfg, err := config.Default()
	if err != nil {
		t.Fatalf("default config: %v", err)
	}
	if len(cfg.Wake.Aliases) == 0 || !strings.EqualFold(cfg.Wake.Aliases[0], "claude") {
		t.Fatalf("default aliases = %v, want claude", cfg.Wake.Aliases)
	}

	t.Run("alias only", func(t *testing.T) {
		got, err := gateWakeText("claude open the door", cfg.Wake.Word, cfg.Wake.Aliases, cfg.Wake.Enabled, false)
		if err != nil {
			t.Fatalf("alias-only transcript rejected: %v", err)
		}
		if got != "open the door" {
			t.Fatalf("alias-only hook text = %q", got)
		}
	})

	t.Run("alias and word", func(t *testing.T) {
		got, err := gateWakeText("claude, clawd open the door", cfg.Wake.Word, cfg.Wake.Aliases, cfg.Wake.Enabled, false)
		if err != nil {
			t.Fatalf("alias plus word rejected: %v", err)
		}
		if got != "clawd open the door" {
			t.Fatalf("hook text kept the alias: %q", got)
		}
	})

	t.Run("primary word", func(t *testing.T) {
		got, err := gateWakeText("clawd open the door", cfg.Wake.Word, cfg.Wake.Aliases, true, false)
		if err != nil || got != "open the door" {
			t.Fatalf("primary word: text=%q err=%v", got, err)
		}
	})

	t.Run("missing wake", func(t *testing.T) {
		_, err := gateWakeText("open the door", cfg.Wake.Word, cfg.Wake.Aliases, true, false)
		if err == nil {
			t.Fatal("expected missing wake word to fail")
		}
	})

	t.Run("no-wake", func(t *testing.T) {
		got, err := gateWakeText("open the door", cfg.Wake.Word, cfg.Wake.Aliases, true, true)
		if err != nil || got != "open the door" {
			t.Fatalf("no-wake: text=%q err=%v", got, err)
		}
	})

	t.Run("wake disabled", func(t *testing.T) {
		got, err := gateWakeText("claude open the door", "clawd", []string{"claude"}, false, false)
		if err != nil || got != "claude open the door" {
			t.Fatalf("disabled wake: text=%q err=%v", got, err)
		}
	})

	t.Run("blank alias", func(t *testing.T) {
		_, err := gateWakeText("open the door", "clawd", []string{" ", ""}, true, false)
		if err == nil {
			t.Fatal("blank alias matched every transcript")
		}
	})
}

func TestResampleLinearLength(t *testing.T) {
	in := []float32{0, 1, 2, 3}
	out := resampleLinear(in, 16000, 8000)
	if len(out) != 2 {
		t.Fatalf("downsample length got %d", len(out))
	}
	out = resampleLinear(in, 8000, 16000)
	if len(out) != 8 {
		t.Fatalf("upsample length got %d", len(out))
	}
}

func TestResampleLinearEnds(t *testing.T) {
	in := []float32{0, 10}
	out := resampleLinear(in, 1000, 2000)
	if out[0] != 0 || out[len(out)-1] != 10 {
		t.Fatalf("endpoints not preserved: %v", out)
	}
}

func TestReadWAV16kMonoResamples(t *testing.T) {
	tmp := t.TempDir() + "/test.wav"
	sr := 8000
	ch := 2
	// 4 frames stereo: L,R pairs (non-cancelling)
	data := []int{1000, 900, 2000, 1900, 3000, 2900, 4000, 3900}
	buf := &audio.IntBuffer{
		Data:           data,
		Format:         &audio.Format{SampleRate: sr, NumChannels: ch},
		SourceBitDepth: 16,
	}
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	enc := wav.NewEncoder(f, sr, 16, ch, 1)
	if err := enc.Write(buf); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := readWAV16kMono(tmp)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(out) != len(data)/ch*2 { // upsample 8k -> 16k doubles frames
		t.Fatalf("expected %d samples got %d", len(data)/ch*2, len(out))
	}
	// Ensure non-zero energy somewhere
	max := float32(0)
	for _, v := range out {
		if v < 0 {
			v = -v
		}
		if v > max {
			max = v
		}
	}
	if max == 0 {
		t.Fatalf("expected some energy after resample, got all zeros")
	}
}
