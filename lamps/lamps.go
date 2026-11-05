// Package lamps provides reusable MIDI-style activity indicators: a set of
// named "lamps" that light up briefly after traffic and darken again after a
// configurable blink window. Stamps are stored in atomics so producer
// goroutines (e.g. a demo sender) can record activity while the tea.Model
// value-copy discipline is preserved.
//
// Each Model is a tea.Model: route LampOffMsg to it so the dousing repaint
// clears the pending flag.
package lamps

import (
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spec describes one lamp: a name (used as key and label) and a lipgloss
// color string used when lit.
type Spec struct {
	Name  string
	Color string
}

// Model is a tea.Model holding a set of activity lamps.
type Model struct {
	window time.Duration

	names  []string
	colors map[string]string
	stamps map[string]*atomic.Int64

	blinking *atomic.Bool
}

// New creates a Model with the given lamps. window is how long a lamp stays
// lit after a Light call. A zero window defaults to 250ms.
func New(specs []Spec, window time.Duration) Model {
	if window <= 0 {
		window = 250 * time.Millisecond
	}
	m := Model{
		window:   window,
		colors:   make(map[string]string, len(specs)),
		stamps:   make(map[string]*atomic.Int64, len(specs)),
		blinking: new(atomic.Bool),
	}
	for _, s := range specs {
		m.names = append(m.names, s.Name)
		m.colors[s.Name] = s.Color
		m.stamps[s.Name] = new(atomic.Int64)
	}
	return m
}

// LampOffMsg is emitted (via WithLampOff) to schedule the dousing repaint.
type LampOffMsg struct{}

// Light records activity for the named lamp at the current time. Unknown
// names are ignored. Safe to call from any goroutine.
func (m Model) Light(name string) {
	m.LightAt(name, time.Now())
}

// LightAt records activity for the named lamp at the given time. Unknown
// names are ignored. Safe to call from any goroutine.
func (m Model) LightAt(name string, t time.Time) {
	if st, ok := m.stamps[name]; ok {
		st.Store(t.UnixNano())
	}
}

// Lit reports whether the named lamp is inside its blink window.
func (m Model) Lit(name string) bool {
	st, ok := m.stamps[name]
	if !ok {
		return false
	}
	t := st.Load()
	return t > 0 && time.Since(time.Unix(0, t)) < m.window
}

// Window returns the configured blink window.
func (m Model) Window() time.Duration { return m.window }

// Names returns the lamp names in display order.
func (m Model) Names() []string { return m.names }

// IsBlinking reports whether a lamp-off repaint is currently scheduled.
func (m Model) IsBlinking() bool {
	if m.blinking == nil {
		return false
	}
	return m.blinking.Load()
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model. It handles LampOffMsg by clearing the pending
// flag; other messages pass through.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case LampOffMsg:
		if m.blinking != nil {
			m.blinking.Store(false)
		}
	}
	return m, nil
}

// WithLampOff schedules a LampOffMsg repaint if none is pending. cmd is
// otherwise returned as-is (batched with the lamp-off tick when scheduled).
// Safe to call from Update on a value copy: the blinking atomic is shared.
func (m Model) WithLampOff(cmd tea.Cmd) tea.Cmd {
	if m.blinking == nil {
		return cmd
	}
	if m.blinking.Swap(true) {
		return cmd // a repaint is already scheduled
	}
	tick := func() tea.Msg { return LampOffMsg{} }
	if cmd == nil {
		return tea.Tick(m.window, func(time.Time) tea.Msg { return tick() })
	}
	return tea.Batch(cmd, tea.Tick(m.window, func(time.Time) tea.Msg { return tick() }))
}

// View renders the lamps in order as a single line: dark hollow circle when
// idle, bold filled dot in the lamp's color when lit.
func (m Model) View() string {
	dark := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("○")
	var b string
	for i, name := range m.names {
		if i > 0 {
			b += " "
		}
		if m.Lit(name) {
			b += lipgloss.NewStyle().Foreground(lipgloss.Color(m.colors[name])).Bold(true).Render("●")
		} else {
			b += dark
		}
	}
	return b
}
