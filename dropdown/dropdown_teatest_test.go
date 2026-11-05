package dropdown

import (
	"io"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestDropdownGolden renders the component in a known state and compares the
// captured terminal output against testdata/TestDropdownGolden.golden.
//
// Regenerate with: go test ./... -run TestDropdownGolden -update
func TestDropdownGolden(t *testing.T) {
	m := New([]string{"alpha", "beta", "gamma"}, 1)
	m.Focus = true
	m.Cursor = 0
	m.CursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	m.CurrentMarker = "[current] "

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(40, 10))
	time.Sleep(200 * time.Millisecond)
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	r := tm.FinalOutput(t)
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	golden.RequireEqual(t, b)
}
