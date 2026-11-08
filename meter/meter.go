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

// blocks are the 9 rendering levels for a single cell (empty .. full).
var blocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// Bar renders a capture level meter.
//
//	peak:    max magnitude in the window (int16)
//	frames:  frames observed in the window (used for the rate readout)
//	drops:   dropped blocks (rendered in red when > 0)
//
// The amplitude is converted to dBFS (a logarithmic scale) and then rendered
// with 1/8-cell resolution so small level changes are visible.
func Bar(peak int16, frames int64, drops uint64) string {
	const width = 20
	db := toDbFS(peak)

	// ratio in [0,1] over the -60..0 dBFS display range (log scale of amplitude).
	ratio := (db + 60) / 60
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	// total eighths across the whole bar
	total := ratio * float64(width) * 8
	fullCells := int(total / 8)
	rem := int(total - float64(fullCells*8))
	if rem < 0 {
		rem = 0
	}
	if rem > 8 {
		rem = 8
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < fullCells {
			bar.WriteRune('█')
		} else if i == fullCells {
			if rem > 0 {
				bar.WriteRune(blocks[rem])
			} else {
				bar.WriteRune(' ')
			}
		} else {
			bar.WriteRune(' ')
		}
	}
	barStr := bar.String()

	// color: green under -12 dB, yellow -12..-3, red above -3
	color := "42" // green
	if db > -3 {
		color = "196" // red
	} else if db > -12 {
		color = "214" // orange/yellow
	}
	barStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(barStr)

	rate := ""
	if frames > 0 {
		rate = fmt.Sprintf(" %4.1fkf/s", float64(frames)/1000/0.2)
	}
	dropTxt := ""
	if drops > 0 {
		dropTxt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf(" drops=%d", drops))
	}
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	return okStyle.Render(fmt.Sprintf("[%s] %6.1fdBFS%s%s", barStyled, db, rate, dropTxt))
}

func toDbFS(peak int16) float64 {
	if peak == 0 {
		return -100
	}
	return 20 * math.Log10(float64(peak)/32768)
}
