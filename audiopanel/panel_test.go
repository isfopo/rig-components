package audiopanel

import (
	"errors"
	"fmt"
	"testing"

	dropdown "github.com/isfopo/rig-components/dropdown"

	"github.com/isfopo/rig-core/audio"
)

func newTestPanel() *Panel {
	p := &Panel{Buf: make(chan []int16, 4)}
	p.InDrop = dropdown.New([]string{"system default"}, 0)
	p.OutDrop = dropdown.New([]string{"system default"}, 0)
	p.ChanDrop = dropdown.New([]string{"2 channels", "1 channels"}, 0)
	return p
}

func TestPanelDropdownNavigation(t *testing.T) {
	p := newTestPanel()
	p.Ins = []audio.Device{{Name: "A"}, {Name: "B"}}
	p.Outs = []audio.Device{{Name: "Out"}}

	inOpts := make([]string, len(p.InOptions()))
	for i, d := range p.InOptions() {
		if i == 0 {
			inOpts[i] = "system default"
		} else {
			inOpts[i] = d.Name
		}
	}
	p.InDrop = p.InDrop.SetOptions(inOpts)

	outOpts := make([]string, len(p.OutOptions()))
	for i, d := range p.OutOptions() {
		if i == 0 {
			outOpts[i] = "system default"
		} else {
			outOpts[i] = d.Name
		}
	}
	p.OutDrop = p.OutDrop.SetOptions(outOpts)

	opts := p.InOptions()
	if len(opts) != 3 || opts[0].Name != "system default" {
		t.Fatalf("inOptions = %+v, want default + 2 entries", opts)
	}
	if deviceIDFor(opts, 0) != nil {
		t.Fatal("default selection must map to nil DeviceID")
	}

	p.ToggleDropdown(DropIn)
	if p.OpenWhich() != DropIn || p.InDrop.Cursor != 0 || p.DropdownCount() != 3 {
		t.Fatalf("open state: which=%d cursor=%d count=%d", p.OpenWhich(), p.InDrop.Cursor, p.DropdownCount())
	}

	p.MoveCursor(-1)
	if p.InDrop.Cursor != 2 {
		t.Fatalf("cursor %d, want wrapped to 2", p.InDrop.Cursor)
	}
	p.MoveCursor(2)
	if p.InDrop.Cursor != 1 {
		t.Fatalf("cursor %d, want 1", p.InDrop.Cursor)
	}

	if !p.Pick() {
		t.Fatal("pick should report a changed selection")
	}
	if p.InDrop.Current != 1 || p.OpenWhich() != DropNone {
		t.Fatalf("after pick: inDrop.Current=%d which=%d", p.InDrop.Current, p.OpenWhich())
	}

	p.ToggleDropdown(DropIn)
	if p.InDrop.Cursor != 1 {
		t.Fatalf("cursor %d, want restored selection 1", p.InDrop.Cursor)
	}
	if p.Pick() {
		t.Fatal("re-picking the same entry should report no change")
	}

	chanOpts := make([]string, len(ChannelCounts))
	for i, c := range ChannelCounts {
		chanOpts[i] = fmt.Sprintf("%d channels", c)
	}
	p.ChanDrop = p.ChanDrop.SetOptions(chanOpts)
	p.ToggleDropdown(DropChan)
	p.MoveCursor(1)
	if !p.Pick() || p.ChanDrop.Current != ChanMono {
		t.Fatalf("channels pick: chans=%d", p.ChanDrop.Current)
	}
	if p.Channels() != 1 {
		t.Fatalf("channels() = %d, want 1", p.Channels())
	}

	p.ToggleDropdown(DropOut)
	if p.DropdownCount() != 2 {
		t.Fatalf("out dropdown count = %d, want 2", p.DropdownCount())
	}
	if p.Pick() {
		t.Fatal("re-picking default out should report no change")
	}
}

func TestPanelOptionsAndNames(t *testing.T) {
	p := newTestPanel()
	p.Ins = []audio.Device{{Name: "Mic"}}
	p.Outs = []audio.Device{{Name: "Speakers"}}

	p.InDrop = p.InDrop.SetOptions([]string{"system default", "Mic"})
	p.InDrop.Current = 1
	p.OutDrop = p.OutDrop.SetOptions([]string{"system default", "Speakers"})
	p.OutDrop.Current = 1

	if len(p.InOptions()) != 2 || p.InOptions()[1].Name != "Mic" {
		t.Fatalf("InOptions = %+v", p.InOptions())
	}
	if len(p.OutOptions()) != 2 || p.OutOptions()[1].Name != "Speakers" {
		t.Fatalf("OutOptions = %+v", p.OutOptions())
	}
	if p.InName() != "Mic" {
		t.Fatalf("InName = %q", p.InName())
	}
	if p.OutName() != "Speakers" {
		t.Fatalf("OutName = %q", p.OutName())
	}
	if p.InName() == "" || p.OutName() == "" {
		t.Fatal("names must not be empty")
	}
}

func TestPanelTakeMeter(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1)}
	p.WinPeak.Store(7)
	p.WinFrames.Store(11)
	p.Drops.Store(3)

	p.TakeMeter()

	if p.DispPeak != 7 || p.DispFrames != 11 || p.DispDrops != 3 {
		t.Fatalf("meter = %d/%d/%d", p.DispPeak, p.DispFrames, p.DispDrops)
	}
	// atomics were consumed
	if p.WinPeak.Load() != 0 || p.WinFrames.Load() != 0 {
		t.Fatalf("atomics not reset: %d/%d", p.WinPeak.Load(), p.WinFrames.Load())
	}
}

func TestPanelChipStates(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1)}
	if got := p.Chip(); !contains(got, "audio:err") {
		t.Fatalf("nil engine chip = %q", got)
	}

	p.Eng = &audio.Engine{}
	p.Capturing = true
	if got := p.Chip(); !contains(got, "audio:live") {
		t.Fatalf("capturing chip = %q", got)
	}

	p.Capturing = false
	p.Monitor = true
	p.PlayStream = &audio.Stream{}
	if got := p.Chip(); !contains(got, "audio:monitor") {
		t.Fatalf("monitor chip = %q", got)
	}
}

func TestPanelMetering(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1)}
	if p.Metering() {
		t.Fatal("idle panel should not be metering")
	}
	p.Capturing = true
	if !p.Metering() {
		t.Fatal("capturing panel should be metering")
	}
}

func TestPanelBufferFlow(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1)}
	p.MonOn.Store(true)

	p.deliver([]int16{1, 2, 3, 4})
	if p.WinFrames.Load() != 2 {
		t.Fatalf("frames = %d, want 2", p.WinFrames.Load())
	}
	if len(p.Buf) != 1 {
		t.Fatalf("buffer len = %d, want 1", len(p.Buf))
	}

	block := p.pull(4)
	if len(block) != 4 {
		t.Fatalf("pull len = %d, want 4", len(block))
	}
	if got := p.pull(4); got != nil {
		t.Fatalf("second pull = %v, want nil", got)
	}

	p.WinPeak.Store(9)
	p.WinFrames.Store(4)
	p.TakeMeter()
	p.DrainBuf()
	if len(p.Buf) != 0 {
		t.Fatal("DrainBuf did not empty the buffer")
	}
}

func TestPanelCommandsWhenEngineNil(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1), InitErr: errors.New("no audio")}

	cmd := p.ListCmd()
	if cmd == nil {
		t.Fatal("ListCmd should not be nil")
	}
	msg := cmd()
	errMsg, ok := msg.(ErrMsg)
	if !ok {
		t.Fatalf("ListCmd msg type = %T", msg)
	}
	if errMsg.Err == nil || errMsg.Err.Error() != "no audio" {
		t.Fatalf("ListCmd err = %v", errMsg.Err)
	}

	if p.RestartCmd() != nil {
		t.Fatal("RestartCmd should be nil without an engine")
	}
}

func TestPanelLifecycleNoPanic(t *testing.T) {
	p := &Panel{Buf: make(chan []int16, 1)}
	p.StopSimplex()
	p.PrepareRestart()
	p.Shutdown()
	p.CloseAll()
	if p.AnyOpen() {
		t.Fatal("CloseAll should leave no dropdown open")
	}
}

func TestPanelOpenWhichNoneByDefault(t *testing.T) {
	p := newTestPanel()
	if p.OpenWhich() != DropNone {
		t.Fatalf("OpenWhich = %d, want DropNone", p.OpenWhich())
	}
	if p.AnyOpen() {
		t.Fatal("no dropdown should be open by default")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
