// Example: run the dropdown component in isolation.
//
//	go run ./examples/dropdown
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/isfopo/rig-components/dropdown"
	"github.com/isfopo/rig-components/demo"
)

type model struct {
	drop dropdown.Model
}

func newModel() model {
	return model{drop: dropdown.New([]string{"alpha", "beta", "gamma", "delta"}, 0)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "o":
			m.drop = m.drop.Open()
			return m, nil
		}
	}
	// delegate to the component while it is focused
	if m.drop.Focus {
		nm, cmd := m.drop.Update(msg)
		m.drop = nm.(dropdown.Model)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	s := "dropdown demo — press 'o' to open, ↑/↓ move, ⏎ pick, esc cancel, q quit\n\n"
	s += m.drop.View()
	if !m.drop.Focus {
		s += fmt.Sprintf("\ncurrent: %s\n", m.drop.Options[m.drop.Current])
	}
	return s
}

func main() {
	demo.Fatal(newModel())
}
