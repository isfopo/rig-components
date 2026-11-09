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

// scaleBreaks define the perceptual (non-linear) mapping from dBFS to bar
// position. More resolution is given to the upper range, matching how humans
// perceive level changes (and standard PPM/VU meters).
var scaleBreaks = []struct {
	db  float64
	pos float64
}{
	{-60, 0.00},
	{-40, 0.12},
	{-30, 0.25},
	{-20, 0.40},
	{-12, 0.55},
	{-6, 0.72},
	{-3, 0.84},
	{0, 1.00},
}

// dbToPos maps a dBFS value to a 0..1 bar position using the perceptual scale.
func dbToPos(db float64) float64 {
	if db < scaleBreaks[0].db {
		return 0
	}
	if db >= scaleBreaks[len(scaleBreaks)-1].db {
		return 1
	}
	for i := 0; i < len(scaleBreaks)-1; i++ {
		a, b := scaleBreaks[i], scaleBreaks[i+1]
		if db >= a.db && db <= b.db {
			t := (db - a.db) / (b.db - a.db)
			return a.pos + t*(b.pos-a.pos)
		}
	}
	return 0
}

// scaleTicks are the dB values labeled on the ruler.
var scaleTicks = []float64{-60, -40, -20, -12, -6, -3, 0}

// renderBar builds the filled bar string for a 0..1 ratio at 1/8-cell precision.
func renderBar(ratio float64, width int) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	total := ratio * float64(width) * 8
	fullCells := int(total / 8)
	rem := int(total - float64(fullCells*8))
	if rem < 0 {
		rem = 0
	}
	if rem > 8 {
		rem = 8
	}
	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < fullCells {
			b.WriteRune('█')
		} else if i == fullCells {
			if rem > 0 {
				b.WriteRune(blocks[rem])
			} else {
				b.WriteRune(' ')
			}
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// ruler builds a two-line dB scale for the bar. Labels are placed greedily so
// they never overlap.
func ruler(width int) string {
	// tick row
	ticks := make([]rune, width)
	for i := range ticks {
		ticks[i] = ' '
	}
	type tick struct {
		pos int
		lbl string
	}
	var ts []tick
	for _, db := range scaleTicks {
		pos := int(math.Round(dbToPos(db) * float64(width-1)))
		if pos < 0 {
			pos = 0
		}
		if pos >= width {
			pos = width - 1
		}
		lbl := fmt.Sprintf("%.0f", db)
		if db == 0 {
			lbl = "0"
		}
		ts = append(ts, tick{pos: pos, lbl: lbl})
		ticks[pos] = '┬'
	}
	// label row, greedy non-overlapping placement
	row := make([]rune, width)
	for i := range row {
		row[i] = ' '
	}
	cursor := 0
	for _, t := range ts {
		start := t.pos - len(t.lbl)/2
		if start < cursor {
			start = cursor
		}
		if start+len(t.lbl) > width {
			// skip if it won't fit
			continue
		}
		for j, r := range t.lbl {
			row[start+j] = r
		}
		cursor = start + len(t.lbl) + 1
	}
	return string(ticks) + "\n" + string(row)
}

// Bar renders a capture level meter from a raw int16 peak.
func Bar(peak int16, frames int64, drops uint64) string {
	db := toDbFS(peak)
	return BarDB(db, db, frames, drops)
}

// BarDB renders a meter from a dBFS value (e.g. an RMS level). peakDb, when
// greater than db, is drawn as a peak-hold marker (│) so transients remain
// visible. A perceptual (non-linear) scale is used and a dB ruler is printed
// beneath the bar.
func BarDB(db, peakDb float64, frames int64, drops uint64) string {
	const width = 20
	if db < -60 {
		db = -60
	}
	if db > 0 {
		db = 0
	}
	if peakDb < -60 {
		peakDb = -60
	}
	if peakDb > 0 {
		peakDb = 0
	}

	ratio := dbToPos(db)
	barStr := renderBar(ratio, width)

	// overlay peak marker
	runes := []rune(barStr)
	if peakDb > db {
		peakPos := int(math.Round(dbToPos(peakDb) * float64(width-1)))
		if peakPos < 0 {
			peakPos = 0
		}
		if peakPos >= width {
			peakPos = width - 1
		}
		if runes[peakPos] == ' ' {
			runes[peakPos] = '│'
		} else {
			runes[peakPos] = '┃'
		}
	}
	barStr = string(runes)

	color := "42"
	if db > -3 {
		color = "196"
	} else if db > -12 {
		color = "214"
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
	top := okStyle.Render(fmt.Sprintf("[%s] %6.1fdBFS%s%s", barStyled, db, rate, dropTxt))
	return top + "\n" + okStyle.Render(ruler(width))
}

func toDbFS(peak int16) float64 {
	if peak == 0 {
		return -100
	}
	return 20 * math.Log10(float64(peak)/32768)
}
