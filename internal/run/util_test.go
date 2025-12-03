package run

import "testing"

func TestMax(t *testing.T) {
	if max(1, 2) != 2 {
		t.Fatalf("max failed")
	}
	if max(5, -1) != 5 {
		t.Fatalf("max failed negative")
	}
}
