// Package meter provides a reusable terminal capture-level meter: a fixed-width
// bar plus dBFS readout and frame-rate proof. Rendering is a pure function
// so callers keep ownership of the underlying atomic state.
//
// The meter is highly configurable: scale breakpoints, ticks, colors, width,
// ruler visibility and the frame-rate window can all be adjusted via Config
// or functional options. Sensible defaults are provided by DefaultConfig.
package meter

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// blocks are the 9 rendering levels for a single cell (empty .. full).
var blocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// ScaleBreak maps a dBFS value to a 0..1 bar position.
type ScaleBreak struct {
	DB  float64
	Pos float64
}

// Config holds all tunable parameters for a Meter.
type Config struct {
	// Width is the number of character cells in the bar.
	Width int

	// Scale defines the (possibly non-linear) mapping from dBFS to bar
	// position. Sorted by DB ascending. Defaults to PerceptualScale().
	Scale []ScaleBreak

	// Ticks are the dB values labeled on the ruler.
	Ticks []float64

	// ShowRuler controls whether the dB ruler is printed beneath the bar.
	ShowRuler bool

	// Floor is the lowest dBFS value represented (below it reads as silence).
	Floor float64

	// Ceiling is the highest dBFS value (usually 0).
	Ceiling float64

	// Color thresholds (dBFS): below GreenBelow is green, above RedAbove is red.
	GreenBelow float64
	RedAbove   float64

	// WindowSecs is the capture window length in seconds used for the
	// frame-rate readout (frames / 1000 / WindowSecs). Default 0.2.
	WindowSecs float64

	// PeakMarker is the rune drawn for a peak that lands on a filled cell.
	PeakMarker rune
	// PeakMarkerEmpty is the rune drawn for a peak on an empty cell.
	PeakMarkerEmpty rune
}

// DefaultConfig returns a ready-to-use config with a perceptual scale.
func DefaultConfig() Config {
	return Config{
		Width:           20,
		Scale:           PerceptualScale(),
		Ticks:           []float64{-60, -40, -20, -12, -6, -3, 0},
		ShowRuler:       true,
		Floor:           -60,
		Ceiling:         0,
		GreenBelow:      -12,
		RedAbove:        -3,
		WindowSecs:      0.2,
		PeakMarker:      '┃',
		PeakMarkerEmpty: '│',
	}
}

// PerceptualScale returns a non-linear scale giving more resolution to the
// upper range, matching human loudness perception and standard PPM meters.
func PerceptualScale() []ScaleBreak {
	return []ScaleBreak{
		{-60, 0.00},
		{-40, 0.12},
		{-30, 0.25},
		{-20, 0.40},
		{-12, 0.55},
		{-6, 0.72},
		{-3, 0.84},
		{0, 1.00},
	}
}

// LinearScale returns a plain linear-in-dB scale.
func LinearScale() []ScaleBreak {
	return []ScaleBreak{
		{-60, 0.00},
		{0, 1.00},
	}
}

// Option configures a Config.
type Option func(*Config)

func WithWidth(w int) Option {
	return func(c *Config) {
		if w > 0 {
			c.Width = w
		}
	}
}

func WithScale(breaks []ScaleBreak) Option {
	return func(c *Config) {
		if len(breaks) >= 2 {
			c.Scale = breaks
		}
	}
}

func WithTicks(ticks []float64) Option {
	return func(c *Config) {
		c.Ticks = ticks
	}
}

func WithRuler(show bool) Option {
	return func(c *Config) { c.ShowRuler = show }
}

func WithFloor(floor float64) Option {
	return func(c *Config) { c.Floor = floor }
}

func WithCeiling(ceiling float64) Option {
	return func(c *Config) { c.Ceiling = ceiling }
}

func WithColors(greenBelow, redAbove float64) Option {
	return func(c *Config) {
		c.GreenBelow = greenBelow
		c.RedAbove = redAbove
	}
}

func WithWindowSecs(s float64) Option {
	return func(c *Config) {
		if s > 0 {
			c.WindowSecs = s
		}
	}
}

// Meter is a configured meter instance. It is safe to reuse across renders.
type Meter struct {
	cfg Config
}

// New creates a Meter from options applied to DefaultConfig.
func New(opts ...Option) *Meter {
	cfg := DefaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Width <= 0 {
		cfg.Width = 20
	}
	if len(cfg.Scale) < 2 {
		cfg.Scale = PerceptualScale()
	}
	if cfg.WindowSecs <= 0 {
		cfg.WindowSecs = 0.2
	}
	return &Meter{cfg: cfg}
}

// Config returns the meter's configuration.
func (m *Meter) Config() Config { return m.cfg }

// Render renders the meter for the given levels.
func (m *Meter) Render(db, peakDb float64, frames int64, drops uint64) string {
	return render(m.cfg, db, peakDb, frames, drops)
}

// Render is a package-level convenience using DefaultConfig.
func Render(db, peakDb float64, frames int64, drops uint64) string {
	return render(DefaultConfig(), db, peakDb, frames, drops)
}

// --- internal ---

func dbToPos(cfg Config, db float64) float64 {
	if db < cfg.Floor {
		return 0
	}
	if db >= cfg.Ceiling {
		return 1
	}
	scale := cfg.Scale
	if len(scale) < 2 {
		scale = PerceptualScale()
	}
	for i := 0; i < len(scale)-1; i++ {
		a, b := scale[i], scale[i+1]
		if db >= a.DB && db <= b.DB {
			if b.DB == a.DB {
				return a.Pos
			}
			t := (db - a.DB) / (b.DB - a.DB)
			return a.Pos + t*(b.Pos-a.Pos)
		}
	}
	if db < scale[0].DB {
		return scale[0].Pos
	}
	return scale[len(scale)-1].Pos
}

func render(cfg Config, db, peakDb float64, frames int64, drops uint64) string {
	if db < cfg.Floor {
		db = cfg.Floor
	}
	if db > cfg.Ceiling {
		db = cfg.Ceiling
	}
	if peakDb < cfg.Floor {
		peakDb = cfg.Floor
	}
	if peakDb > cfg.Ceiling {
		peakDb = cfg.Ceiling
	}

	width := cfg.Width
	if width <= 0 {
		width = 20
	}

	ratio := dbToPos(cfg, db)
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

	// overlay peak marker
	runes := []rune(barStr)
	if peakDb > db {
		peakPos := int(math.Round(dbToPos(cfg, peakDb) * float64(width-1)))
		if peakPos < 0 {
			peakPos = 0
		}
		if peakPos >= width {
			peakPos = width - 1
		}
		if runes[peakPos] == ' ' {
			runes[peakPos] = cfg.PeakMarkerEmpty
		} else {
			runes[peakPos] = cfg.PeakMarker
		}
	}
	barStr = string(runes)

	color := "42"
	if db > cfg.RedAbove {
		color = "196"
	} else if db > cfg.GreenBelow {
		color = "214"
	}
	barStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(barStr)

	rate := ""
	if frames > 0 && cfg.WindowSecs > 0 {
		rate = fmt.Sprintf(" %4.1fkf/s", float64(frames)/1000/cfg.WindowSecs)
	}
	dropTxt := ""
	if drops > 0 {
		dropTxt = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf(" drops=%d", drops))
	}
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	top := okStyle.Render(fmt.Sprintf("[%s] %6.1fdBFS%s%s", barStyled, db, rate, dropTxt))

	if !cfg.ShowRuler {
		return top
	}
	return top + "\n" + okStyle.Render(ruler(cfg, width))
}

func ruler(cfg Config, width int) string {
	ticks := make([]rune, width)
	for i := range ticks {
		ticks[i] = ' '
	}
	type tick struct {
		pos int
		lbl string
	}
	var ts []tick
	for _, db := range cfg.Ticks {
		pos := int(math.Round(dbToPos(cfg, db) * float64(width-1)))
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
			continue
		}
		for j, r := range t.lbl {
			row[start+j] = r
		}
		cursor = start + len(t.lbl) + 1
	}
	return string(ticks) + "\n" + string(row)
}

// --- backward-compatible API ---

// Bar renders a meter from a raw int16 peak using DefaultConfig.
func Bar(peak int16, frames int64, drops uint64) string {
	db := toDbFS(peak)
	return Render(db, db, frames, drops)
}

// BarDB renders a meter from a dBFS value using DefaultConfig.
func BarDB(db, peakDb float64, frames int64, drops uint64) string {
	return Render(db, peakDb, frames, drops)
}

func toDbFS(peak int16) float64 {
	if peak == 0 {
		return -100
	}
	return 20 * math.Log10(float64(peak)/32768)
}
