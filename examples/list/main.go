// Example: run the list component in isolation.
//
//	go run ./examples/list
package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/isfopo/rig-components/demo"
	"github.com/isfopo/rig-components/list"
)

type model struct {
	l list.Model
}

func newModel() model {
	m := list.New([]string{"one", "two", "three", "four"})
	m = m.SetActive(1)
	return model{l: m}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	nm, cmd := m.l.Update(msg)
	m.l = nm.(list.Model)
	return m, cmd
}

func (m model) View() string {
	return "list demo — ↑/↓ move, ⏎ select, q quit\n\n" + m.l.View()
}

func main() {
	demo.Fatal(newModel())
}
