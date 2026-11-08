package readout

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New("BPM")
	if m.Label != "BPM" {
		t.Fatalf("label: %q", m.Label)
	}
	if m.View() == "" {
		t.Fatal("empty view")
	}
}

func TestSetValueMsg(t *testing.T) {
	m := New("BPM")
	m2, _ := m.Update(SetValueMsg{Value: "120"})
	if m2.Value != "120" {
		t.Fatalf("value: %q", m2.Value)
	}
	if m2.View() == m.View() {
		t.Fatal("view should change after value update")
	}
}

func TestStale(t *testing.T) {
	m := New("BPM")
	m2, _ := m.Update(SetValueMsg{Value: "120", Stale: true})
	if !m2.Stale {
		t.Fatal("expected stale")
	}
}

func TestSet(t *testing.T) {
	m := New("BPM").Set("90")
	if m.Value != "90" {
		t.Fatalf("value: %q", m.Value)
	}
}

func TestWidthPadding(t *testing.T) {
	m := New("BPM")
	m.Value = "120"
	m.Width = 20
	out := m.View()
	if len(out) < 20 {
		t.Fatalf("expected padding, got len=%d: %q", len(out), out)
	}
}

func TestTickCmd(t *testing.T) {
	cmd := Tick(0, func() string { return "x" })
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	msg := cmd()
	sm, ok := msg.(SetValueMsg)
	if !ok || sm.Value != "x" {
		t.Fatalf("unexpected msg: %#v", msg)
	}
	_ = tea.Batch
}
