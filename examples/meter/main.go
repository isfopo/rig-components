// Example: run the meter component in isolation.
//
//	go run ./examples/meter
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/isfopo/rig-components/demo"
	"github.com/isfopo/rig-components/meter"
)

type model struct {
	peak   int16
	frames int64
	drops  uint64
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.peak < 32767 {
				m.peak += 1000
			}
		case "down", "j":
			if m.peak > 0 {
				m.peak -= 1000
			}
		case "d":
			m.drops++
		case "r":
			m.drops = 0
		}
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("meter demo — ↑/↓ peak, d drop, r reset drops, q quit\n\n  %s\n\npeak=%d\n", meter.Bar(m.peak, 200, m.drops), m.peak)
}

func main() {
	demo.Fatal(model{peak: 8000, frames: 200})
}
