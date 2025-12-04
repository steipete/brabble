package control

import "testing"

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
		if got := removeWakeWordLocal(c.text, c.word); got != c.expect {
			t.Fatalf("removeWakeWordLocal(%q)=%q want %q", c.text, got, c.expect)
		}
	}
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
