// Package dropdown provides a reusable, keyboard-driven option list used
// inside a larger TUI: an open/closed focus state, a cursor with wrap-around
// navigation, and a selection that reports whether it actually changed.
//
// It is a tea.Model so a parent can delegate input while the dropdown is
// focused, and it also exposes methods for parents that prefer to drive keys
// themselves.
package dropdown

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PickedMsg is emitted when the user confirms a selection.
type PickedMsg struct {
	Index   int
	Changed bool
}

// ClosedMsg is emitted when the user cancels (esc).
type ClosedMsg struct{}

// Model holds dropdown state.
type Model struct {
	Options []string
	Cursor  int
	Current int // selected index (the "current" marker)
	Focus   bool

	// styling hooks (optional; defaults applied in View)
	CursorStyle   lipgloss.Style
	CurrentMarker string // defaults to "[current] "
}

// New creates a closed dropdown over options with current as the selection.
func New(options []string, current int) Model {
	if current < 0 || current >= len(options) {
		current = 0
	}
	return Model{
		Options:       options,
		Current:       current,
		Cursor:        current,
		CurrentMarker: "[current] ",
		CursorStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
	}
}

// Count returns the number of options.
func (m Model) Count() int { return len(m.Options) }

// SetOptions replaces the option list, clamping Current/Cursor.
func (m Model) SetOptions(options []string) Model {
	m.Options = options
	if m.Current >= len(options) {
		m.Current = 0
	}
	if m.Cursor >= len(options) {
		m.Cursor = m.Current
	}
	return m
}

// Open focuses the dropdown, seeding the cursor on the current selection.
func (m Model) Open() Model {
	m.Focus = true
	m.Cursor = m.Current
	return m
}

// Close unfocuses without changing the selection.
func (m Model) Close() Model {
	m.Focus = false
	return m
}

// Toggle flips focus, seeding the cursor when opening.
func (m Model) Toggle() Model {
	if m.Focus {
		return m.Close()
	}
	return m.Open()
}

// Move shifts the cursor with wrap-around.
func (m Model) Move(delta int) Model {
	n := len(m.Options)
	if n == 0 {
		return m
	}
	m.Cursor = (m.Cursor + delta + n) % n
	return m
}

// Pick confirms the cursor as the new selection. Returns the model and
// whether the selection actually changed.
func (m Model) Pick() (Model, bool) {
	if len(m.Options) == 0 {
		m.Focus = false
		return m, false
	}
	changed := m.Cursor != m.Current
	m.Current = m.Cursor
	m.Focus = false
	return m, changed
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model. Handles up/down/enter/esc while focused.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.Focus {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m = m.Move(-1)
		case "down", "j":
			m = m.Move(1)
		case "enter", "space":
			nm, changed := m.Pick()
			m = nm
			return m, func() tea.Msg { return PickedMsg{Index: m.Current, Changed: changed} }
		case "esc":
			m = m.Close()
			return m, func() tea.Msg { return ClosedMsg{} }
		}
	}
	return m, nil
}

// View renders the option list with cursor and current marker.
func (m Model) View() string {
	var b string
	for i, o := range m.Options {
		line := o
		if i == m.Current {
			line = m.CurrentMarker + line
		}
		if i == m.Cursor && m.Focus {
			b += "    > " + m.CursorStyle.Render(line) + "\n"
		} else {
			b += "      " + line + "\n"
		}
	}
	return b
}

// String returns a compact debug representation.
func (m Model) String() string {
	return fmt.Sprintf("dropdown{focus=%v cursor=%d current=%d n=%d}", m.Focus, m.Cursor, m.Current, len(m.Options))
}
