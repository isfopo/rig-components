// Example: run the lamps component in isolation.
//
//	go run ./examples/lamps
package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/isfopo/rig-components/demo"
	"github.com/isfopo/rig-components/lamps"
)

type model struct {
	lamps lamps.Model
}

func newModel() model {
	return model{lamps: lamps.New([]lamps.Spec{
		{Name: "in", Color: "42"},
		{Name: "out", Color: "205"},
	}, 0)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "i":
			m.lamps.Light("in")
		case "o":
			m.lamps.Light("out")
		}
	case lamps.LampOffMsg:
		nm, cmd := m.lamps.Update(msg)
		m.lamps = nm.(lamps.Model)
		return m, cmd
	}
	return m, m.lamps.WithLampOff(nil)
}

func (m model) View() string {
	return "lamps demo — press 'i'/'o' to light in/out, q quit\n\n  " + m.lamps.View() + "\n"
}

func main() {
	// keep the process alive long enough to see blinks in the example
	_ = time.Second
	demo.Fatal(newModel())
}
