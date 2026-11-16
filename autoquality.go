package main

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Automatic quality on first launch (2026-11-11 #20 list, item 19). The
// first time the main menu is up, the game draws a busy chamber off screen
// at High quality and the screen's full size, 24 times, waits for the GPU to
// finish, and picks the preset from the time per frame:
//   up to 6 ms   High    (room for 144 Hz)
//   up to 12 ms  Medium
//   slower       Low
// It runs once (a brief pause on the very first menu), never on a save that
// has played a run, and never on the Steam Deck, which has its own preset.
// A toast says what it chose; Settings > Display changes it any time.

const (
	aqFrames   = 24
	aqHighMS   = 6.0
	aqMediumMS = 12.0
)

func aqPick(ms float64) int {
	switch {
	case ms <= aqHighMS:
		return 2
	case ms <= aqMediumMS:
		return 1
	}
	return 0
}

// wantAutoQuality reports whether this launch should measure.
func (g *Game) wantAutoQuality() bool {
	return !g.save.QualityAuto && g.save.Runs == 0 && bench == nil && !soakOn && !benchOn &&
		g.mbench == nil && !steamOnDeck() && g.state == StMenu && g.stateT > 1.5
}

// autoQuality measures and applies; it is called from Draw, before the
// frame's own drawing, so the off-screen work never shows.
func (g *Game) autoQuality(screen *ebiten.Image) {
	g.save.QualityAuto = true
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w < 64 || h < 36 {
		return
	}
	savedRun, savedLevel, savedQ, savedWorld := g.run, g.level, g.settings.Quality, g.snd.World
	savedRes, savedBloom, savedParts, savedAA := g.settings.RenderRes, g.settings.NoBloom, g.settings.Particles, g.settings.NoAA
	defer func() {
		g.run, g.level = savedRun, savedLevel
		g.snd.World = savedWorld // the test chamber set its world's sounds
	}()
	g.applyQuality(2)
	g.run = NewRun("BENCH", &archetypes[0], &g.save, false)
	forceWorld = 3
	L := g.newLevel(NCombat, 4245)
	forceWorld = -1
	L.introT = 0
	g.ensureSky(L.Biome, L.propSeed)
	out := ebiten.NewImage(w, h)
	defer out.Deallocate()
	px := make([]byte, 4)
	sync := func() { out.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image).ReadPixels(px) }
	frame := func() {
		tgt := g.post.Target(w, h)
		g.deferOverlay = true
		g.drawWorld(tgt, L)
		g.deferOverlay = false
		g.post.Apply(out, g.worldGrade(L), true)
	}
	for i := 0; i < 3; i++ { // first draws build shaders and textures
		frame()
	}
	sync()
	t0 := time.Now()
	for i := 0; i < aqFrames; i++ {
		L.Update(1.0/60, Input{UseKeys: true})
		frame()
	}
	sync()
	ms := float64(time.Since(t0).Microseconds()) / 1000 / aqFrames
	q := aqPick(ms)
	g.settings.RenderRes, g.settings.NoBloom, g.settings.Particles, g.settings.NoAA = savedRes, savedBloom, savedParts, savedAA
	g.settings.Quality = savedQ
	g.applyQuality(q)
	writeSave(&g.save)
	g.toast(trJoin("Graphics set to ", qualityNames[q]), "Picked for this PC. Change it any time in Settings > Display.", cAcc, icStar)
}
