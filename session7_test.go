package main

import (
	"math"
	"testing"
	"time"
)

// The paddle squash springs back to rest and never inverts the paddle.
func TestPaddleSquashSettles(t *testing.T) {
	L := &Level{}
	if sx, sy := L.paddleScale(); sx != 1 || sy != 1 {
		t.Fatal("a paddle at rest is scaled")
	}
	L.padSquash = 1
	sx, sy := L.paddleScale()
	if sy >= 1 || sx <= 1 {
		t.Errorf("a hard return doesn't squash: %.2f %.2f", sx, sy)
	}
	for i := 0; i < 60; i++ {
		L.updatePaddleFeel(1.0 / 60)
		sx, sy := L.paddleScale()
		if sy < 0.7 || sy > 1.3 || sx < 0.9 || sx > 1.1 {
			t.Fatalf("squash out of range: %.2f %.2f", sx, sy)
		}
	}
	if sx, sy := L.paddleScale(); sx != 1 || sy != 1 {
		t.Error("the squash never settles")
	}
}

// The combo colour warms monotonically: more red, less blue as it climbs.
func TestComboHeatWarms(t *testing.T) {
	prevB, prevG := 999, 999
	for _, n := range []int{3, 10, 25, 40, 80} {
		c, heat := comboHeat(n)
		// warming = the green falls (white -> gold -> orange -> red); fire
		// keeps a touch of pink, so blue isn't checked past orange
		if int(c.G) > prevG || (n <= 25 && int(c.B) > prevB) {
			t.Errorf("combo %d is cooler than the one before: %v", n, c)
		}
		if heat < 0 || heat > 1 {
			t.Errorf("heat %.2f", heat)
		}
		prevB, prevG = int(c.B), int(c.G)
	}
	if _, h := comboHeat(5); h != 0 {
		t.Error("a small combo has embers")
	}
}

func TestSeasons(t *testing.T) {
	for m, want := range map[time.Month]int{time.January: 0, time.April: 1, time.July: 2, time.November: 3, time.December: 0} {
		if s := season(time.Date(2026, m, 5, 0, 0, 0, 0, time.UTC)); s != want {
			t.Errorf("%v is season %d, want %d", m, s, want)
		}
	}
}

// Auto quality: fast PCs get High, slow ones Low.
func TestAutoQualityPick(t *testing.T) {
	if aqPick(2.5) != 2 || aqPick(9) != 1 || aqPick(30) != 0 {
		t.Fatal("wrong presets for the measured times")
	}
	g := &Game{save: defaultSave(), settings: &Settings{}}
	g.save.Runs = 3
	g.stateT = 5
	if g.wantAutoQuality() {
		t.Error("auto quality would run on a save that has played")
	}
}

// Big numbers in the history graph read short.
func TestFmtBig(t *testing.T) {
	if s := fmtBig(1_500_000); s != "1.5M" {
		t.Error(s)
	}
	if s := fmtBig(950); s == "" {
		t.Error("empty")
	}
}

// The reachable chambers' worlds are listed once each.
func TestNextWorlds(t *testing.T) {
	sv := defaultSave()
	r := NewRun("NW", &archetypes[0], &sv, false)
	ws := r.nextWorlds()
	if len(ws) == 0 {
		t.Fatal("no next worlds at the start of a run")
	}
	seen := map[int]bool{}
	for _, w := range ws {
		if seen[w] || w < 0 || w >= len(worlds) {
			t.Fatalf("bad world list %v", ws)
		}
		seen[w] = true
	}
}

// Build hints skip what the run owns and the card itself.
func TestBuildHints(t *testing.T) {
	g := &Game{save: defaultSave(), settings: &Settings{}}
	g.run = NewRun("BH", &archetypes[0], &g.save, false)
	found := false
	for _, f := range fusions {
		m := modByID[f.A]
		if m == nil || m.Curse || m.Origin != "" {
			continue
		}
		hs := g.buildHints(m)
		for _, h := range hs {
			if h.ID == m.ID || g.run.Mods[h.ID] > 0 {
				t.Fatalf("hint %s for %s", h.ID, m.ID)
			}
		}
		if len(hs) > 3 {
			t.Fatal("more than three hints")
		}
		found = found || len(hs) > 0
	}
	if !found {
		t.Error("no fusion item gets any hint")
	}
}

// A damage number's colour depends on its source.
func TestDamageColours(t *testing.T) {
	if dmgColor("Orb") == dmgColor("Burning") {
		t.Error("orb and fire damage look the same")
	}
	_ = math.Pi
}
