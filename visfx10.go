package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 10 (ROADMAP section P): familiar character, co-op
// motion, shop and origin-screen polish. All cheap: a few sprites or
// billboards each, no new per-brick work.

var cP2 = color.RGBA{120, 220, 255, 255}

// famLook adjusts a familiar's drawn position and size: the ember sprite
// flickers, the frost moth flutters.
func (L *Level) famLook(kind int, i int, p V3) (V3, float64) {
	sz := 1.0
	switch kind {
	case 1:
		sz = 0.9 + 0.18*math.Abs(math.Sin(L.T*13+float64(i)*2.1))*math.Abs(math.Sin(L.T*7.3))
	case 2:
		p.Y += 0.14 * math.Sin(L.T*11+float64(i))
		p.X += 0.06 * math.Sin(L.T*5.5)
	}
	return p, sz
}

// wispTrail draws two fading afterimages behind an Arcane Wisp.
func (L *Level) wispTrail(B *Batch, i, n int, col color.RGBA) {
	ang := L.famAng
	for k := 1; k <= 2; k++ {
		L.famAng = ang - 0.12*float64(k)
		p := L.familiarPos(i, n)
		B.Billboard(p, 0.42-0.1*float64(k), sprSoft, alpha(col, 0.28/float64(k)), true)
	}
	L.famAng = ang
}

// owlReadyAlpha: the resting owl brightens over its last 3 seconds of rest
// and pulses once it's ready again.
func (L *Level) owlReadyAlpha() float64 {
	if L.owlCD > 0 {
		return 0.35 + 0.65*math.Max(0, 1-L.owlCD/3)
	}
	return 0.85 + 0.15*math.Sin(L.T*4)
}

// trackPartner keeps a short history of player two's paddle for its trail.
func (L *Level) trackPartner() {
	copy(L.p2Hist[1:], L.p2Hist[:len(L.p2Hist)-1])
	L.p2Hist[0] = L.p2X
}

// drawPartnerTrail streaks player two's paddle when it moves fast.
func (L *Level) drawPartnerTrail(B *Batch) {
	if L.g.ambientOff() {
		return
	}
	if !L.p2On {
		return
	}
	for k := 1; k < len(L.p2Hist); k++ {
		x := L.p2Hist[k]
		if math.Abs(x-L.p2X) < 0.4 {
			continue
		}
		B.Billboard(V3{x, 0.5, PaddleZ}, L.PaddleW*0.35, sprSoft, alpha(cP2, 0.16*(1-float64(k)/float64(len(L.p2Hist)))), true)
	}
}

// ---------------------------------------------------------------- screens

// keepsakeRunes: little lights circling the chosen keepsake's emblem.
func (g *Game) keepsakeRunes(s *ebiten.Image, x, y float64) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for k := 0; k < 5; k++ {
		a := g.stateT*1.4 + float64(k)*2*math.Pi/5
		px, py := x+math.Cos(a)*36, y+math.Sin(a)*36
		drawSprite(s, sprSoft, u.X(px-7), u.Y(py-7), 14*u.S, 14*u.S, ca(cGold, 0.7), true)
	}
}

// lockedCharm puts a small chain over a keepsake that isn't unlocked yet.
func (g *Game) lockedCharm(s *ebiten.Image, x, y float64) {
	g.ui.drawEmblem(s, glyphImage("chain", rgba(150, 140, 170, 255)), x, y, 22, 0.9)
}

// trialRowGlow: an ember line flickers along a trial that's switched on.
func (g *Game) trialRowGlow(s *ebiten.Image, x, y, w, h float64, i int) {
	u := g.ui
	fl := 0.7 + 0.3*math.Sin(g.stateT*7+float64(i)*1.3)
	u.glowLine(s, x+12, x+w-12, y+h-2, 2.5, cTrial, 0.8*fl)
}

// emberRise sends embers up from the Trials button once 4+ points are on.
func (g *Game) emberRise(s *ebiten.Image, x, y, w float64) {
	if g.ambientOff() {
		return
	}
	n := trialPoints(g.selectedTrials())
	if n < 4 {
		return
	}
	u := g.ui
	for k := 0; k < min(n, 8); k++ {
		ph := math.Mod(g.stateT*0.6+float64(k)*0.137, 1)
		px := x + w*math.Mod(float64(k)*0.618, 1) + 6*math.Sin(g.stateT*3+float64(k))
		py := y - ph*46
		drawSprite(s, sprSoft, u.X(px-5), u.Y(py-5), 10*u.S, 10*u.S, ca(cTrial, 0.8*(1-ph)), true)
	}
}

// coopGlow lights the co-op button in player two's colour when it's on.
func (g *Game) coopGlow(s *ebiten.Image, x, y, w, h float64) {
	if !g.coopSel {
		return
	}
	u := g.ui
	p := 0.8 + 0.2*math.Sin(g.stateT*3)
	drawSprite(s, sprSoft, u.X(x-26), u.Y(y-22), (w+52)*u.S, (h+44)*u.S, ca(cP2, 0.3*p), true)
}

// banishHover outlines the card the cursor is on in banish mode.
func (g *Game) banishHover(s *ebiten.Image, x, y, w, h float64) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	if !u.hover(x, y, w, h) {
		return
	}
	fl := 0.7 + 0.3*math.Sin(g.stateT*18)
	strokePath(s, boxPath(u.X(x-4), u.Y(y-4), (w+8)*u.S, (h+8)*u.S, 14*u.S, false), 3*u.S, ca(cBanish, fl), false)
}

// shopShine sweeps a band of light across the shop's items for a moment
// after they're stocked (on opening and after a reroll).
func (g *Game) shopShine(s *ebiten.Image, x, y, w, h float64) {
	if g.ambientOff() {
		return
	}
	k := g.shopShineT / 0.5
	if k >= 1 {
		return
	}
	u := g.ui
	sx := x - 80 + (w+160)*k
	drawSprite(s, sprSoft, u.X(sx-40), u.Y(y), 80*u.S, h*u.S, ca(cWhite, 0.35*math.Sin(k*math.Pi)), true)
}

// shopGoldFlash glows the gold counter when the amount changes.
func (g *Game) shopGoldFlash(s *ebiten.Image, gold int) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	if gold != g.shopGoldShown {
		if g.shopGoldShown >= 0 {
			g.goldFlash = 1
		}
		g.shopGoldShown = gold
	}
	if g.goldFlash <= 0 {
		return
	}
	g.goldFlash = math.Max(0, g.goldFlash-u.DT*3)
	drawSprite(s, sprSoft, u.X(960-120), u.Y(98), 240*u.S, 60*u.S, ca(cGold, 0.45*g.goldFlash), true)
}

// charmFlare: the keepsake on the HUD flares when it does something.
func (g *Game) charmFlare(s *ebiten.Image, x, y float64) {
	if g.flourishOff() {
		return
	}
	if g.charmFlash <= 0 {
		return
	}
	u := g.ui
	k := g.charmFlash
	g.charmFlash = math.Max(0, k-u.DT*1.5)
	r := 18 + 30*(1-k)
	drawSprite(s, sprSoft, u.X(x-r), u.Y(y-r), 2*r*u.S, 2*r*u.S, ca(cGold, 0.7*k), true)
	strokePath(s, circlePath(u.X(x), u.Y(y), r*u.S), 2*u.S, ca(cGold, k), false)
}

// lastHeartRing: on the last life the remaining heart sends out a slow
// ring with each beat.
func (g *Game) lastHeartRing(dst *ebiten.Image, x, y float64) {
	u := g.ui
	ph := math.Mod(u.time/0.75, 1)
	r := 14 + 16*ph
	strokePath(dst, circlePath(u.X(x), u.Y(y), r*u.S), 2*u.S, ca(rgba(255, 90, 130, 255), 0.6*(1-ph)), false)
}
