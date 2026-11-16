package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 17 (ROADMAP section AE): power-up pickups, elites
// and scrolls in the chamber, and the settings, codex, seed and challenges
// screens.

// ---------------------------------------------------------------- world

// pickupFX: catching a power-up rings the paddle in its colour, with a
// column of light for a good one.
func (L *Level) pickupFX(p Power, x float64) {
	d := powerDefs[p]
	L.Rings = append(L.Rings, &Ring{X: x, Z: PaddleZ, MaxR: 2.2, Life: 0.35, Max: 0.35, Col: d.Col})
	if !d.Good {
		return
	}
	n := 6
	if !L.fullParts() {
		n = 2
	}
	for i := 0; i < n; i++ {
		L.Parts = append(L.Parts, Particle{P: V3{x + (L.rng.Float64()-0.5)*1.2, 0.4, PaddleZ}, V: V3{0, 4 + L.rng.Float64()*2, 0},
			Life: 0.5, Max: 0.5, Size: 0.26, Col: lighten(d.Col, 0.3), Kind: pSpark})
	}
}

// eliteFall: a toppled elite leaves a violet shock ring and a burst.
func (L *Level) eliteFall(x, z float64) {
	if L.g.flourishOff() {
		return
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 4, Life: 0.5, Max: 0.5, Col: rgba(220, 160, 255, 255)})
	L.sparks(V3{x, 1, z}, 18, rgba(230, 180, 255, 255), 7)
}

// scrollCast: reading a scroll unfurls a ring of its colour round the
// paddle.
func (L *Level) scrollCast(sd *scrollDef) {
	if L.g.flourishOff() {
		return
	}
	L.Rings = append(L.Rings,
		&Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 3.2, Life: 0.5, Max: 0.5, Col: sd.Col},
		&Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 1.6, Life: 0.35, Max: 0.35, Col: lighten(sd.Col, 0.4)})
	L.sparks(V3{L.PaddleX, 0.8, PaddleZ + 0.4}, 12, sd.Col, 5)
}

// ---------------------------------------------------------------- screens

// tabSlide: the glow under the chosen tab slides to it instead of jumping.
func (g *Game) tabSlide(s *ebiten.Image, key string, target, y, w float64) {
	u := g.ui
	cur, ok := g.tabGlowX[key]
	if !ok {
		cur = target
	}
	cur += (target - cur) * math.Min(1, u.DT*14)
	if g.tabGlowX == nil {
		g.tabGlowX = map[string]float64{}
	}
	g.tabGlowX[key] = cur
	u.glowLine(s, cur, cur+w, y, 3, cGold, 1)
	drawSprite(s, sprSoft, u.X(cur-10), u.Y(y-14), (w+20)*u.S, 28*u.S, ca(cGold, 0.25), true)
}

// seedTyping: the seed box glows a little more with each character, and a
// fresh character pops.
func (g *Game) seedTyping(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	n := len(g.seedText)
	if n != g.seedLen {
		g.seedLen, g.seedPopT = n, 0
	}
	g.seedPopT += u.DT
	k := math.Min(1, float64(n)/10)
	drawSprite(s, sprSoft, u.X(660-40), u.Y(440-30), 680*u.S, 160*u.S, ca(cGold, 0.08+0.18*k), true)
	if n > 0 && g.seedPopT < 0.25 {
		a := 1 - g.seedPopT/0.25
		w := u.measure(g.seedText, fTitle, 56)
		x := 960 + w/2 - 12
		drawSprite(s, sprSoft, u.X(x-24), u.Y(484-24), 48*u.S, 48*u.S, ca(cWhite, 0.6*a), true)
	}
}

// sealShine: a sealed challenge's medallion has a slow gold sheen.
func (g *Game) sealShine(s *ebiten.Image, cx, cy float64, i int) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	e := math.Mod(g.stateT*0.3+float64(i)*0.21, 1.4)
	if e > 1 {
		return
	}
	a := e*2*math.Pi - math.Pi/2
	px, py := cx+math.Cos(a)*56, cy+math.Sin(a)*56
	drawSprite(s, sprSoft, u.X(px-16), u.Y(py-16), 32*u.S, 32*u.S, ca(cGold, 0.8*math.Sin(e/1*math.Pi)), true)
}

// challengeMotes: motes in each challenge's colour drift up its card.
func (g *Game) challengeMotes(s *ebiten.Image, x, y, w, h float64, col color.RGBA, i int) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for k := 0; k < 4; k++ {
		ph := math.Mod(g.stateT*0.25+float64(k)*0.25+float64(i)*0.13, 1)
		px := x + 30 + (w-60)*math.Mod(float64(k)*0.618+float64(i)*0.31, 1)
		py := y + h - 20 - ph*(h-40)
		drawSprite(s, sprSoft, u.X(px-6), u.Y(py-6), 12*u.S, 12*u.S, ca(col, 0.5*math.Sin(ph*math.Pi)), true)
	}
}

// settingsToggleFlash: a switch flashes when it changes.
func (g *Game) settingsPulse(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	if g.setPulseT > 0.4 {
		return
	}
	u := g.ui
	g.setPulseT += u.DT
	k := 1 - g.setPulseT/0.4
	drawSprite(s, sprSoft, u.X(460), u.Y(80), 1000*u.S, 900*u.S, ca(cGold, 0.06*k), true)
}

// codexPageGlow: a soft light sweeps down the Codex as a tab opens.
func (g *Game) codexPageGlow(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	if g.codexTab != g.codexShown {
		g.codexShown, g.codexOpenT = g.codexTab, 0
	}
	u := g.ui
	g.codexOpenT += u.DT
	if g.codexOpenT > 0.5 {
		return
	}
	e := g.codexOpenT / 0.5
	y := 210 + 800*e
	drawSprite(s, sprSoft, u.X(160), u.Y(y-40), 1600*u.S, 80*u.S, ca(cGold, 0.18*math.Sin(e*math.Pi)), true)
}

// badPickupSmoke: a bad power-up (Shrink, Haste) leaves dark puffs.
func (L *Level) badPickupSmoke(x float64) {
	n := 4
	if !L.fullParts() {
		n = 2
	}
	for i := 0; i < n; i++ {
		L.Parts = append(L.Parts, Particle{P: V3{x + (L.rng.Float64()-0.5)*1.4, 0.5, PaddleZ}, V: V3{(L.rng.Float64() - 0.5) * 1.5, 1.5, 0},
			Life: 0.7, Max: 0.7, Size: 0.6, Col: rgba(60, 40, 70, 255), Kind: pSmoke})
	}
}

// dropMarkers: a small glowing tick on the paddle line where each falling
// power-up will arrive.
func (L *Level) dropMarkers(B *Batch) {
	for _, d := range L.Drops {
		if d.Z > 16 {
			continue
		}
		c := powerDefs[d.P].Col
		a := 0.25 + 0.35*(1-(d.Z-PaddleZ)/16)
		B.Decal(d.X, PaddleZ, 0.04, 0.9, 0.25, sprSoft, alpha(c, a), true)
	}
}

// sealOrbit: a light circles each reachable chamber on the map.
func (g *Game) sealOrbit(s *ebiten.Image, x, y, rad float64, i int) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	a := g.stateT*1.6 + float64(i)*1.1
	px, py := x+math.Cos(a)*(rad+8), y+math.Sin(a)*(rad+8)
	drawSprite(s, sprSoft, u.X(px-7), u.Y(py-7), 14*u.S, 14*u.S, ca(cGold, 0.75), true)
}

// seedSparkle: stars drift slowly round the Play a Seed heading.
func (g *Game) seedStars(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for i := 0; i < 12; i++ {
		a := g.stateT*0.2 + float64(i)*2*math.Pi/12
		px, py := 960+math.Cos(a)*380, 240+math.Sin(a)*70
		tw := 0.5 + 0.5*math.Sin(g.stateT*3+float64(i))
		drawSprite(s, sprSoft, u.X(px-6), u.Y(py-6), 12*u.S, 12*u.S, ca(rgba(220, 210, 255, 255), 0.5*tw), true)
	}
}
