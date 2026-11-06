package audiopanel

import (
	"fmt"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	dropdown "github.com/isfopo/rig-components/dropdown"

	"github.com/isfopo/rig-core/audio"
)

// dropdown subjects
const (
	DropNone = iota
	DropIn
	DropOut
	DropChan
)

// channel entries (indices into the channels dropdown)
const (
	ChanStereo = 0
	ChanMono   = 1
)

// ChannelCounts are the Options.Channels values for the channels dropdown.
var ChannelCounts = []uint32{2, 1}

// audioPanel owns everything audio for one Model instance.
type Panel struct {
	Eng     *audio.Engine // nil when the audio backend failed to initialize
	InitErr error

	Ins, Outs []audio.Device // last enumeration (may be nil until listed)

	// dropdown components (one per subject). At most one has Focus==true.
	InDrop   dropdown.Model
	OutDrop  dropdown.Model
	ChanDrop dropdown.Model

	Seq atomic.Int64 // restart token: only the newest async open is adopted

	CapStream *audio.Stream
	Capturing bool

	// meter (windows read+reset in Update; only atomics touched on the
	// audio thread)
	WinPeak    atomic.Int64 // max magnitude since last read
	WinFrames  atomic.Int64 // frames since last read
	DispPeak   int16        // snapshot for View
	DispFrames int64        // snapshot for View
	DispDrops  uint64       // snapshots for View

	// monitor path: captured blocks are pushed to the selected output
	Monitor    bool
	MonOn      atomic.Bool // gates deliver; true only while playStream runs
	Buf        chan []int16
	Drops      atomic.Int64
	PlayStream *audio.Stream
}

// newAudioPanel initializes the engine eagerly (like the MIDI monitor is
// created in New, never in a toggle) and starts with the system default
// devices selected but nothing opened until the user picks a device.
func New() *Panel {
	p := &Panel{Buf: make(chan []int16, 400)}
	p.Eng, p.InitErr = audio.NewEngine()
	if p.InitErr != nil {
		p.Eng = nil
	}
	// channel dropdown options are static
	opts := make([]string, len(ChannelCounts))
	for i, c := range ChannelCounts {
		opts[i] = fmt.Sprintf("%d channels", c)
	}
	p.ChanDrop = dropdown.New(opts, 0)
	// device dropdowns start with just the system-default entry; they are
	// rebuilt when enumeration arrives (see audioListMsg handling in model.go)
	p.InDrop = dropdown.New([]string{"system default"}, 0)
	p.OutDrop = dropdown.New([]string{"system default"}, 0)
	return p
}

// ---- options ----------------------------------------------------------

// inOptions returns the capture-device dropdown: "system default" always
// first, then every enumerated capture device.
func (p *Panel) InOptions() []audio.Device {
	o := []audio.Device{{Name: "system default", IsDefault: true}}
	return append(o, p.Ins...)
}

// outOptions returns the playback-device dropdown entries.
func (p *Panel) OutOptions() []audio.Device {
	o := []audio.Device{{Name: "system default", IsDefault: true}}
	return append(o, p.Outs...)
}

// deviceID resolves a dropdown selection to an Options.DeviceID (nil keeps
// the system default: do not use the first enumerated device blindly).
func deviceIDFor(opts []audio.Device, sel int) []byte {
	if sel <= 0 || sel >= len(opts) {
		return nil
	}
	return opts[sel].ID()
}

func (p *Panel) InName() string {
	sel := p.InDrop.Current
	if sel == 0 {
		return "system default"
	}
	if sel <= len(p.Ins) {
		return p.Ins[sel-1].Name
	}
	return "?"
}

func (p *Panel) OutName() string {
	sel := p.OutDrop.Current
	if sel == 0 {
		return "system default"
	}
	if sel <= len(p.Outs) {
		return p.Outs[sel-1].Name
	}
	return "?"
}

func (p *Panel) Channels() uint32 {
	return ChannelCounts[p.ChanDrop.Current]
}

// ---- messages ---------------------------------------------------------

type ListMsg struct {
	Ins  []audio.Device
	Outs []audio.Device
}

type ErrMsg struct{ Err error }

type CaptureStartedMsg struct {
	Stream *audio.Stream
}

type PlaybackStartedMsg struct {
	Stream *audio.Stream
}

type MeterMsg struct{}

// ---- commands ---------------------------------------------------------

// listCmd enumerates capture/playback devices (async: WASAPI/ALSA
// enumeration can block for a noticeable while).
func (p *Panel) ListCmd() tea.Cmd {
	return func() tea.Msg {
		if p.Eng == nil {
			return ErrMsg{Err: p.InitErr}
		}
		ins, err := p.Eng.Inputs()
		if err != nil {
			return ErrMsg{Err: err}
		}
		outs, err := p.Eng.Outputs()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ListMsg{Ins: ins, Outs: outs}
	}
}

// restartCmd re-opens capture (and playback when the monitor is on) with the
// current dropdown state. Old streams are stopped by prepareRestart before
// this is called. One seq token covers both opens so a newer restart
// invalidates both in-flight opens at once.
func (p *Panel) RestartCmd() tea.Cmd {
	if p.Eng == nil {
		return nil
	}
	// snapshot all selection state on the Update thread
	seq := p.Seq.Add(1)
	opts := audio.Options{
		DeviceID: deviceIDFor(p.InOptions(), p.InDrop.Current),
		Channels: p.Channels(),
	}
	outID := deviceIDFor(p.OutOptions(), p.OutDrop.Current)
	mon := p.Monitor

	captureCmd := func() tea.Msg {
		stream, err := p.Eng.OpenCapture(opts, p.deliver)
		if err != nil {
			return ErrMsg{Err: err}
		}
		if p.Seq.Load() != seq { // a newer restart won: abort
			stream.Stop()
			return nil
		}
		return CaptureStartedMsg{Stream: stream}
	}

	if !mon {
		return captureCmd
	}

	playbackCmd := func() tea.Msg {
		stream, err := p.Eng.OpenPlayback(audio.Options{
			DeviceID: outID,
			Channels: opts.Channels,
		}, p.pull)
		if err != nil {
			return ErrMsg{Err: err}
		}
		if p.Seq.Load() != seq {
			stream.Stop()
			return nil
		}
		return PlaybackStartedMsg{Stream: stream}
	}
	return tea.Batch(captureCmd, playbackCmd)
}

func (p *Panel) StopSimplex() {
	if p.CapStream != nil {
		p.CapStream.Stop()
		p.CapStream = nil
	}
	if p.PlayStream != nil {
		p.PlayStream.Stop()
		p.PlayStream = nil
	}
	p.MonOn.Store(false)
	p.Capturing = false
	p.DrainBuf()
}

// metering reports whether a live meter exists (simplex capture or looper).
func (p *Panel) Metering() bool {
	return p.Capturing
}

// ---- audio thread side -------------------------------------------------

// deliver is the capture callback: meters the block, and forwards a copy to
// the monitor buffer when the monitor path is live.
func (p *Panel) deliver(in []int16) {
	n := len(in) / 2
	p.WinFrames.Add(int64(n))
	peak := audio.Peak(in)
	for {
		old := p.WinPeak.Load()
		if int64(peak) <= old || p.WinPeak.CompareAndSwap(old, int64(peak)) {
			break
		}
	}

	if !p.MonOn.Load() {
		return
	}
	block := make([]int16, len(in)) // fresh copy: capture buffer is reused
	copy(block, in)
	select {
	case p.Buf <- block:
	default:
		p.Drops.Add(1)
	}
}

// pull is the playback callback (drains the monitor buffer).
func (p *Panel) pull(n int) []int16 {
	select {
	case b := <-p.Buf:
		return b
	default:
		return nil // capture not warmed up (or too slow): silence
	}
}

// shutdown stops streams and closes the engine (Model.Shutdown calls this).
func (p *Panel) Shutdown() {
	if p.CapStream != nil {
		p.CapStream.Stop()
		p.CapStream = nil
	}
	if p.PlayStream != nil {
		p.PlayStream.Stop()
		p.PlayStream = nil
	}
	p.MonOn.Store(false)
	if p.Eng != nil {
		p.Eng.Close()
	}
}

// chip renders the compact audio state for the status bar.
func (p *Panel) Chip() string {
	live := p.Capturing
	state, color := "audio:off", "250"
	switch {
	case p.Monitor && p.PlayStream != nil:
		state, color = "audio:monitor", "213"
	case p.Capturing:
		state, color = "audio:live", "42"
	case p.Eng == nil:
		state = "audio:err"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(live).
		Render("[" + state + "]")
}

// ---- dropdown state helpers -------------------------------------------

// prepareRestart stops running streams and flushes the monitor buffer so
// restartCmd comes after a clean slate. The looper's stream is stopped with
// everything else; restartAfterChange re-opens it attached (loop content
// survives: only the *audio.Stream* churns).
func (p *Panel) PrepareRestart() {
	if p.CapStream != nil {
		p.CapStream.Stop()
		p.CapStream = nil
	}
	if p.PlayStream != nil {
		p.PlayStream.Stop()
		p.PlayStream = nil
	}
	p.MonOn.Store(false)
	p.DrainBuf()
}

func (p *Panel) DrainBuf() {
	for {
		select {
		case <-p.Buf:
		default:
			return
		}
	}
}

// openWhich returns DropIn/DropOut/DropChan for the currently open dropdown,
// or DropNone when none is open.
func (p *Panel) OpenWhich() int {
	switch {
	case p.InDrop.Focus:
		return DropIn
	case p.OutDrop.Focus:
		return DropOut
	case p.ChanDrop.Focus:
		return DropChan
	}
	return DropNone
}

// anyOpen reports whether any dropdown is currently open.
func (p *Panel) AnyOpen() bool { return p.OpenWhich() != DropNone }

// closeAll closes every dropdown.
func (p *Panel) CloseAll() {
	p.InDrop = p.InDrop.Close()
	p.OutDrop = p.OutDrop.Close()
	p.ChanDrop = p.ChanDrop.Close()
}

// toggleDropdown opens the focused dropdown (or closes it if already open).
func (p *Panel) ToggleDropdown(which int) {
	if p.OpenWhich() == which {
		p.CloseAll()
		return
	}
	p.CloseAll()
	switch which {
	case DropIn:
		opts := make([]string, len(p.InOptions()))
		for i, d := range p.InOptions() {
			if i == 0 {
				opts[i] = "system default"
			} else {
				opts[i] = d.Name
			}
		}
		p.InDrop = p.InDrop.SetOptions(opts)
		p.InDrop = p.InDrop.Open()
	case DropOut:
		opts := make([]string, len(p.OutOptions()))
		for i, d := range p.OutOptions() {
			if i == 0 {
				opts[i] = "system default"
			} else {
				opts[i] = d.Name
			}
		}
		p.OutDrop = p.OutDrop.SetOptions(opts)
		p.OutDrop = p.OutDrop.Open()
	case DropChan:
		p.ChanDrop = p.ChanDrop.Open()
	}
}

// moveCursor moves the cursor inside the open dropdown with wrap-around.
func (p *Panel) MoveCursor(delta int) {
	switch p.OpenWhich() {
	case DropIn:
		p.InDrop = p.InDrop.Move(delta)
	case DropOut:
		p.OutDrop = p.OutDrop.Move(delta)
	case DropChan:
		p.ChanDrop = p.ChanDrop.Move(delta)
	}
}

// dropdownCount is the number of entries of the currently open dropdown.
func (p *Panel) DropdownCount() int {
	switch p.OpenWhich() {
	case DropIn:
		return p.InDrop.Count()
	case DropOut:
		return p.OutDrop.Count()
	case DropChan:
		return p.ChanDrop.Count()
	}
	return 0
}

// pick selects the entry under the cursor; the caller restarts the streams.
// Returns true when the selection actually moved (restart only then).
func (p *Panel) Pick() bool {
	which := p.OpenWhich()
	if which == DropNone {
		return false
	}
	var changed bool
	switch which {
	case DropIn:
		nm, ch := p.InDrop.Pick()
		p.InDrop = nm
		changed = ch
	case DropOut:
		nm, ch := p.OutDrop.Pick()
		p.OutDrop = nm
		changed = ch
	case DropChan:
		nm, ch := p.ChanDrop.Pick()
		p.ChanDrop = nm
		changed = ch
	}
	return changed
}

// takeMeter snapshots the metering window into the plain display fields that
// View reads (Update runs between renders, so this stays race-free). While
// the looper owns capture, its pass-through is the source instead.
func (p *Panel) TakeMeter() {
	p.DispPeak = int16(p.WinPeak.Swap(0))
	p.DispFrames = p.WinFrames.Swap(0)
	p.DispDrops = uint64(p.Drops.Load())
}

// meterTickCmd keeps the meter subplot refreshing while capture runs.
func MeterTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return MeterMsg{}
	})
}
