// Package meter provides a reusable terminal capture-level meter: a fixed-width
// bar plus dBFS readout and frame-rate proof. Rendering is a pure function
// so callers keep ownership of the underlying atomic state.
package meter

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bar renders a capture level meter.
//
//	peak:    max magnitude in the window (int16)
//	frames:  frames observed in the window (used for the rate readout)
//	drops:   dropped blocks (rendered in red when > 0)
func Bar(peak int16, frames int64, drops uint64) string {
	const width = 20
	db := toDbFS(peak)
	filled := 0
	if db > -60 {
		filled = int((db + 60) / 60 * width)
		if filled > width {
			filled = width
		}
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	rate := ""
	if frames > 0 {
		rate = fmt.Sprintf(" %4.1fkf/s", float64(frames)/1000/0.2)
	}
	dropTxt := ""
	if drops > 0 {
		dropTxt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf(" drops=%d", drops))
	}
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	return okStyle.Render(fmt.Sprintf("[%s] %6.1fdBFS%s%s", bar, db, rate, dropTxt))
}

func toDbFS(peak int16) float64 {
	if peak == 0 {
		return -100
	}
	return 20 * math.Log10(float64(peak) / 32768)
}
