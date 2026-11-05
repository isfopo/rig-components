package meter

import "testing"

func TestBarSmoke(t *testing.T) {
	out := Bar(0, 0, 0)
	if out == "" {
		t.Fatal("empty")
	}
	out2 := Bar(32767, 100, 3)
	if out2 == out {
		t.Fatal("expected different output for different input")
	}
}
