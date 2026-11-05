// Package list provides a reusable keyboard-navigable list with a cursor and
// an optional "active/open" marker per item. It is a tea.Model.
package list

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectedMsg is emitted when the user presses enter/space on an item.
type SelectedMsg struct {
	Index int
}

// Model holds list state.
type Model struct {
	Items  []string
	Cursor int
	Active int // index marked as "open/active"; -1 for none

	// Render hooks
	SelectedStyle lipgloss.Style
	ActiveStyle   lipgloss.Style
	CursorGlyph   string // defaults to "> "
	ActiveGlyph   string // defaults to "•"
}

// New creates a list over items with nothing active.
func New(items []string) Model {
	return Model{
		Items:         items,
		Active:        -1,
		CursorGlyph:   "> ",
		ActiveGlyph:   "•",
		SelectedStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		ActiveStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	}
}

// SetItems replaces items, clamping cursor.
func (m Model) SetItems(items []string) Model {
	m.Items = items
	if len(items) == 0 {
		m.Cursor = 0
		return m
	}
	if m.Cursor >= len(items) {
		m.Cursor = 0
	}
	return m
}

// SetActive marks an item as active (e.g. open port).
func (m Model) SetActive(i int) Model {
	m.Active = i
	return m
}

// Move shifts cursor with wrap-around.
func (m Model) Move(delta int) Model {
	n := len(m.Items)
	if n == 0 {
		return m
	}
	m.Cursor = (m.Cursor + delta + n) % n
	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m = m.Move(-1)
		case "down", "j":
			m = m.Move(1)
		case "enter", "space":
			if len(m.Items) > 0 {
				return m, func() tea.Msg { return SelectedMsg{Index: m.Cursor} }
			}
		}
	}
	return m, nil
}

// View renders the list. Each row: cursor glyph, name, active marker.
func (m Model) View() string {
	if len(m.Items) == 0 {
		return ""
	}
	var b string
	for i, it := range m.Items {
		cursor := "  "
		name := it
		if i == m.Cursor {
			cursor = m.CursorGlyph
			name = m.SelectedStyle.Render(it)
		}
		state := "  "
		if i == m.Active {
			state = m.ActiveStyle.Render(m.ActiveGlyph)
		}
		b += "  " + cursor + name + " " + state + "\n"
	}
	return b
}
