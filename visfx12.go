package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 12 (ROADMAP section T): click feedback on the new
// panels and toggles, familiar shot flavour, co-op catches and a couple of
// HUD and shop cues. Screen bursts use uiBurst (visfx11.go).

// keepsakeHover lifts the keepsake card under the cursor with a soft glow.
func (g *Game) keepsakeHover(s *ebiten.Image, x, y, w, h float64, have bool) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	if !u.hover(x, y, w, h) {
		return
	}
	c := cGold
	if !have {
		c = rgba(150, 140, 170, 255)
	}
	drawSprite(s, sprSoft, u.X(x-20), u.Y(y-16), (w+40)*u.S, (h+32)*u.S, ca(c, 0.16), true)
}

// keepsakeButtonCharm puts the chosen keepsake's emblem on the origin
// screen's Keepsake button.
func (g *Game) keepsakeButtonCharm(s *ebiten.Image, x, y float64) {
	if g.ambientOff() {
		return
	}
	k := keepsakeByID[g.save.Keepsake]
	if k == nil || g.save.keepRank(k.ID) <= 0 {
		return
	}
	u := g.ui
	p := 0.8 + 0.2*math.Sin(g.stateT*2.5)
	drawSprite(s, sprSoft, u.X(x-18), u.Y(y-18), 36*u.S, 36*u.S, ca(cGold, 0.35*p), true)
	u.drawEmblem(s, glyphImage(k.Glyph, cGold), x, y, 24, 1)
}

// healNudge pulses the shop's "Regain a life" button on the last life.
func (g *Game) healNudge(s *ebiten.Image, x, y, w, h float64, can bool) {
	if g.flourishOff() {
		return
	}
	r := g.run
	if !can || r.Lives != 1 || r.maxLives() <= 1 {
		return
	}
	u := g.ui
	p := 0.5 + 0.5*math.Sin(g.stateT*4)
	drawSprite(s, sprSoft, u.X(x-24), u.Y(y-20), (w+48)*u.S, (h+40)*u.S, ca(cGreen, 0.12+0.18*p), true)
}

// familiarDots: one dot per familiar in its colour, after the HUD charms.
func (g *Game) familiarDots(dst *ebiten.Image, L *Level, x, y float64) float64 {
	if L == nil {
		return x
	}
	u := g.ui
	fams := L.familiars()
	for i, f := range fams {
		px := x + float64(i)*14
		a := 1.0
		if f.kind == 3 && L.owlCD > 0 {
			a = 0.4
		}
		drawSprite(dst, sprSoft, u.X(px-9), u.Y(y-9), 18*u.S, 18*u.S, ca(f.col, 0.5*a), true)
		fillPath(dst, circlePath(u.X(px), u.Y(y), 4*u.S), ca(lighten(f.col, 0.4), a), ca(f.col, a), vgrad(u.Y(y-4), u.Y(y+4)), false)
		if f.kind == 3 {
			g.owlRestArc(dst, L, px, y)
		}
	}
	if len(fams) > 0 && u.hover(x-8, y-10, float64(len(fams))*14+8, 20) {
		g.tooltip = tr("Familiars")
	}
	return x + float64(len(fams))*14
}

// ---------------------------------------------------------------- world

// familiarShotFX gives each familiar's shot its own touch: a flash at the
// familiar, embers rising off a burning brick, a frost ring on a frozen one.
func (L *Level) familiarShotFX(kind int, from, at V3, col color.RGBA) {
	L.Parts = append(L.Parts, Particle{P: from, Life: 0.15, Max: 0.15, Size: 0.5, Col: lighten(col, 0.3), Kind: pSpark})
	switch kind {
	case 1:
		n := 3
		if !L.fullParts() {
			n = 1
		}
		for i := 0; i < n; i++ {
			L.Parts = append(L.Parts, Particle{P: at.Add(V3{(L.rng.Float64() - 0.5) * 0.6, 0, (L.rng.Float64() - 0.5) * 0.4}),
				V: V3{0, 1.6 + L.rng.Float64(), 0}, Life: 0.8, Max: 0.8, Size: 0.18, Col: color.RGBA{255, 170, 70, 255}, Kind: pSpark, Grav: -0.4})
		}
	case 2:
		L.Rings = append(L.Rings, &Ring{X: at.X, Z: at.Z, MaxR: 1.2, Life: 0.35, Max: 0.35, Col: color.RGBA{180, 235, 255, 255}})
	}
}

// owlWakes: a soft ring round the paddle when the owl is ready again.
func (L *Level) owlWakes() {
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 2.6, Life: 0.5, Max: 0.5, Col: famCols[3]})
}

// partnerCatchFX: a teal sparkle where player two catches a coin.
func (L *Level) partnerCatchFX(x float64) {
	L.sparks(V3{x, 0.8, PaddleZ}, 4, cP2, 3)
}
