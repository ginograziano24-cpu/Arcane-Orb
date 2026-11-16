package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestPadFamilies(t *testing.T) {
	for name, want := range map[string]int{
		"Xbox Wireless Controller": padXbox, "DualSense Wireless Controller": padPlayStation,
		"Wireless Controller": padPlayStation, "PS4 Controller": padPlayStation,
		"Nintendo Switch Pro Controller": padNintendo, "Steam Deck": padDeck, "Generic USB Joystick": padXbox,
	} {
		if got := padFamilyOf(name); got != want {
			t.Errorf("%q is family %d, want %d", name, got, want)
		}
	}
	old := padFamily
	defer func() { padFamily = old }()
	padFamily = padPlayStation
	if padLabel(ebiten.StandardGamepadButtonRightBottom) != "Cross" || padLabel(ebiten.StandardGamepadButtonFrontTopLeft) != "L1" {
		t.Error("PlayStation names not used")
	}
	padFamily = padNintendo
	if padLabel(ebiten.StandardGamepadButtonRightBottom) != "B" {
		t.Error("the bottom face button on a Nintendo pad is B")
	}
	padFamily = padXbox
	if padLabel(ebiten.StandardGamepadButtonRightBottom) != "A" {
		t.Error("Xbox default lost")
	}
}

func TestChroniclePages(t *testing.T) {
	g := &Game{save: defaultSave()}
	open := 0
	for i := range chronicle {
		if g.chronicleOpen(&chronicle[i]) {
			open++
		}
	}
	if open != 1 || !g.chronicleOpen(&chronicle[0]) {
		t.Fatalf("a new save reads %d pages, want only the foreword", open)
	}
	g.save.BestAct = 2 // reached act 2: act 1 beaten; acts 1 and 2's worlds played
	for i := range chronicle {
		p := &chronicle[i]
		want := p.Key == "intro" || p.Act == 1
		if p.World >= 0 {
			want = false
			for _, a := range acts[:2] {
				for _, w := range a.Pool {
					want = want || w == p.World
				}
			}
		}
		if got := g.chronicleOpen(p); got != want {
			t.Errorf("page %s open=%v, want %v", p.Key, got, want)
		}
	}
	g.run = NewRun("CH", &archetypes[0], &g.save, false)
	g.noteChronicle(WCaverns, true)
	if !g.save.Chronicle["w"+itoa(WCaverns)] || !g.save.Chronicle["g1"] {
		t.Error("entering a chamber didn't open its pages")
	}
	g.run.SeedCode = "BENCH"
	g.noteChronicle(WMachine, false)
	if g.save.Chronicle["w"+itoa(WMachine)] {
		t.Error("a benchmark opened a chronicle page")
	}
	for _, p := range chronicle {
		if len(p.Paras) == 0 || p.Title == "" {
			t.Errorf("page %s is empty", p.Key)
		}
	}
}

// The packed opening stems decode to the same length, so the three stay in
// sync (the narration is mono and must widen to stereo).
func TestOpeningStemsPacked(t *testing.T) {
	var n []int
	for _, name := range []string{"music", "effects", "narration"} {
		b, err := openingStem(name)
		if err != nil {
			t.Fatal(name, err)
		}
		s, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if err != nil {
			t.Fatal(name, err)
		}
		pcm, err := io.ReadAll(s)
		if err != nil {
			t.Fatal(name, err)
		}
		n = append(n, len(pcm))
	}
	if n[0] != n[1] || n[1] != n[2] || n[0] == 0 {
		t.Fatalf("stem lengths differ: %v", n)
	}
}

// Photo mode keeps the camera above the floor and within its zoom range.
func TestPhotoCameraLimits(t *testing.T) {
	g := &Game{state: StPause, photo: &photoMode{yaw: 9, pitch: -9, zoom: 99}}
	g.photo.yaw, g.photo.pitch, g.photo.zoom = 1.4, -0.55, 1.8
	cam := &Camera{Eye: V3{0, 15, -10.5}, Target: V3{0, 3, 17}}
	g.photoCamera(cam)
	if cam.Eye.Y <= cam.Target.Y {
		t.Fatalf("the photo camera went under the board: %+v", cam.Eye)
	}
}

func TestDeckTextSize(t *testing.T) {
	g := &Game{settings: &Settings{}, ui: &UI{SW: 1280, SH: 800}}
	if g.textScale() <= 1 {
		t.Error("text doesn't start larger at 1280x800")
	}
	g.ui.SW, g.ui.SH = 2560, 1440
	if g.textScale() != 1 {
		t.Error("text isn't normal on a large screen")
	}
	g.settings.TextScale = 0.9
	g.ui.SW, g.ui.SH = 1280, 800
	if g.textScale() != 0.9 {
		t.Error("a chosen text size is overridden")
	}
}
