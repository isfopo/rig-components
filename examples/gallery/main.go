// Component gallery: browse and interact with each component in isolation.
//
//	go run ./examples/gallery
//
// Tab toggles focus between the story list (left) and the component (right).
// In the list: ↑/↓ select a story. q quits.
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/isfopo/rig-components/dropdown"
	"github.com/isfopo/rig-components/demo"
	"github.com/isfopo/rig-components/lamps"
	"github.com/isfopo/rig-components/list"
	"github.com/isfopo/rig-components/meter"
)

type focus int

const (
	focusList focus = iota
	focusComp
)

type story struct {
	name    string
	factory func() tea.Model
}

type gallery struct {
	stories []story
	cursor  int
	focus   focus
	active  tea.Model
	width   int
	height  int
}

func newGallery() gallery {
	g := gallery{focus: focusList}
	g.stories = []story{
		{"dropdown / closed", func() tea.Model { return dropdown.New([]string{"alpha", "beta", "gamma"}, 1) }},
		{"dropdown / open", func() tea.Model {
			m := dropdown.New([]string{"alpha", "beta", "gamma"}, 1)
			m.Focus = true
			m.Cursor = 0
			m.CursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
			m.CurrentMarker = "[current] "
			return m
		}},
		{"list / basic", func() tea.Model { return list.New([]string{"one", "two", "three", "four"}) }},
		{"list / active", func() tea.Model {
			m := list.New([]string{"one", "two", "three", "four"})
			m = m.SetActive(2)
			m.SelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
			m.ActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
			return m
		}},
		{"lamps / idle", func() tea.Model {
			return lampDemo{m: lamps.New([]lamps.Spec{{Name: "in", Color: "42"}, {Name: "out", Color: "205"}}, 0)}
		}},
		{"lamps / lit (in)", func() tea.Model {
			lm := lamps.New([]lamps.Spec{{Name: "in", Color: "42"}, {Name: "out", Color: "205"}}, 0)
			lm.Light("in")
			return lampDemo{m: lm}
		}},
		{"meter / low", func() tea.Model { return meterDemo{peak: 4000, frames: 200} }},
		{"meter / high", func() tea.Model { return meterDemo{peak: 28000, frames: 200} }},
		{"meter / with drops", func() tea.Model { return meterDemo{peak: 16000, frames: 200, drops: 3} }},
	}
	g.active = g.stories[0].factory()
	return g
}

// lampDemo wraps lamps.Model so it satisfies tea.Model with quit handling.
type lampDemo struct{ m lamps.Model }

func (d lampDemo) Init() tea.Cmd { return nil }
func (d lampDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "i" {
			d.m.Light("in")
		}
		if msg.String() == "o" {
			d.m.Light("out")
		}
	case lamps.LampOffMsg:
		nm, cmd := d.m.Update(msg)
		d.m = nm.(lamps.Model)
		return d, cmd
	}
	return d, d.m.WithLampOff(nil)
}
func (d lampDemo) View() string { return "  " + d.m.View() + "\n\n(press i/o to light)" }

// meterDemo wraps the meter render function.
type meterDemo struct {
	peak   int16
	frames int64
	drops  uint64
}

func (d meterDemo) Init() tea.Cmd { return nil }
func (d meterDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.peak < 32767 {
				d.peak += 1000
			}
		case "down", "j":
			if d.peak > 0 {
				d.peak -= 1000
			}
		case "d":
			d.drops++
		case "r":
			d.drops = 0
		}
	}
	return d, nil
}
func (d meterDemo) View() string {
	return fmt.Sprintf("  %s\n\npeak=%d  (↑/↓ peak, d drop, r reset)", meter.Bar(d.peak, d.frames, d.drops), d.peak)
}

func (g gallery) Init() tea.Cmd { return nil }

func (g gallery) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width, g.height = msg.Width, msg.Height
		return g, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return g, tea.Quit
		case "tab":
			if g.focus == focusList {
				g.focus = focusComp
			} else {
				g.focus = focusList
			}
			return g, nil
		}
		if g.focus == focusList {
			switch msg.String() {
			case "up", "k":
				if g.cursor > 0 {
					g.cursor--
					g.active = g.stories[g.cursor].factory()
				}
			case "down", "j":
				if g.cursor < len(g.stories)-1 {
					g.cursor++
					g.active = g.stories[g.cursor].factory()
				}
			}
			return g, nil
		}
		// forward to active component
		nm, cmd := g.active.Update(msg)
		g.active = nm
		return g, cmd
	}
	// forward non-key messages to active component too
	nm, cmd := g.active.Update(msg)
	g.active = nm
	return g, cmd
}

func (g gallery) View() string {
	listStyle := lipgloss.NewStyle().Width(26).Padding(1, 1)
	compStyle := lipgloss.NewStyle().Padding(1, 2)

	var lb string
	for i, s := range g.stories {
		line := "  " + s.name
		if i == g.cursor {
			line = "> " + s.name
			if g.focus == focusList {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render(line)
			}
		}
		lb += line + "\n"
	}
	lb += "\n"
	if g.focus == focusList {
		lb += "↑/↓ select • tab focus component • q quit"
	} else {
		lb += "tab back to list • q quit"
	}

	left := listStyle.Render(lb)
	right := compStyle.Render(g.active.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func main() {
	demo.Fatal(newGallery())
}
