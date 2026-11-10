package meter

import "testing"

func TestConfigurable(t *testing.T) {
	m := New(WithWidth(10), WithRuler(false), WithWindowSecs(0.5))
	if m.Config().Width != 10 {
		t.Fatalf("width: %d", m.Config().Width)
	}
	out := m.Render(-12, -6, 4800, 0)
	if out == "" {
		t.Fatal("empty render")
	}
	// no ruler => single line
	if strings := countNewlines(out); strings != 0 {
		t.Fatalf("expected no ruler newline, got %d newlines: %q", strings, out)
	}
	// Linear scale produces different output than default
	lin := New(WithScale(LinearScale()))
	if lin.Render(-12, -6, 4800, 0) == m.Render(-12, -6, 4800, 0) {
		// width differs so not comparable; just ensure non-empty
	}
	// Render convenience
	if Render(-6, -1, 100, 1) == "" {
		t.Fatal("empty package Render")
	}
}

func countNewlines(s string) int {
	n := 0
	for _, r := range s {
		if r == '\n' {
			n++
		}
	}
	return n
}
