package lamps

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLampBlinkLogic(t *testing.T) {
	l := New([]Spec{{Name: "in", Color: "42"}, {Name: "out", Color: "205"}}, 0)

	if l.Lit("in") {
		t.Fatal("in lamp lit without traffic")
	}
	l.Light("in")
	if !l.Lit("in") {
		t.Fatal("in lamp did not light on a fresh stamp")
	}
	// expire
	st := l.stamps["in"]
	st.Store(time.Now().Add(-2 * l.Window()).UnixNano())
	if l.Lit("in") {
		t.Fatal("in lamp did not expire after the blink window")
	}

	// LampOffMsg clears pending flag
	if first := l.WithLampOff(nil); first == nil {
		t.Fatal("first call must schedule the lamp-off repaint")
	}
	if again := l.WithLampOff(nil); again != nil {
		t.Fatal("second call within the window must not schedule another repaint")
	}
	upd, _ := l.Update(LampOffMsg{})
	lm := upd.(Model)
	if lm.blinking.Load() {
		t.Fatal("LampOffMsg did not clear the pending flag")
	}
	if fresh := lm.WithLampOff(nil); fresh == nil {
		t.Fatal("after dousing a new lamp-off repaint should be schedulable")
	}
	_ = tea.Msg(nil)
}
