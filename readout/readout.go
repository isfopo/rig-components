// Package readout provides a reusable compact live-value display: a label
// followed by a value that updates via messages. It is intentionally generic
// so it can render BPM, peak dBFS, counters, status text, or any other small
// piece of live data in a terminal UI.
//
// Each Model is a tea.Model: route SetValueMsg to it to update the value.
package readout

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Default styles.
var (
	defaultLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	defaultValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

// Model is a tea.Model holding a compact live value.
type Model struct {
	Label string
	Value string

	LabelStyle lipgloss.Style
	ValueStyle lipgloss.Style

	// Sep is rendered between label and value. Defaults to ": ".
	Sep string

	// Width, when > 0, pads the rendered output to that width.
	Width int

	// Stale marks the value as out of date (rendered dimmed).
	Stale bool
}

// New creates a Model with the given label and an empty value.
func New(label string) Model {
	return Model{
		Label:      label,
		LabelStyle: defaultLabelStyle,
		ValueStyle: defaultValueStyle,
		Sep:        ": ",
	}
}

// SetValueMsg carries a new value for a readout.
type SetValueMsg struct {
	Value string
	Stale bool
}

// Update handles SetValueMsg and ignores other messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if v, ok := msg.(SetValueMsg); ok {
		m.Value = v.Value
		m.Stale = v.Stale
	}
	return m, nil
}

// Set returns a copy with the value updated (convenience for callers that
// do not route through Update).
func (m Model) Set(value string) Model {
	m.Value = value
	return m
}

// View renders the compact "label: value" string.
func (m Model) View() string {
	sep := m.Sep
	if sep == "" {
		sep = ": "
	}
	label := m.LabelStyle.Render(m.Label)
	valueStyle := m.ValueStyle
	if m.Stale {
		valueStyle = valueStyle.Foreground(lipgloss.Color("240"))
	}
	value := valueStyle.Render(m.Value)
	out := label + sep + value
	if m.Width > 0 {
		if pad := m.Width - len(out); pad > 0 {
			out += strings.Repeat(" ", pad)
		}
	}
	return out
}

// Tick is a helper that periodically calls fetch and emits a SetValueMsg.
// The returned command drives a self-updating readout when composed into a
// tea.Batch. fetch must be safe to call from the tea runtime.
func Tick(interval time.Duration, fetch func() string) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return SetValueMsg{Value: fetch()}
	})
}
