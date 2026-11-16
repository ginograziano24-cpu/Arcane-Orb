package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 9 (ROADMAP section M): small touches for the new
// systems (familiars, keepsakes, trials, banish/lock, co-op) and a few
// readability cues. Everything here is cheap: a handful of particles,
// rings or sprites, and it all thins out with "Full particles" off.

var (
	cGoldRing = color.RGBA{255, 220, 140, 255}
	cBanish   = color.RGBA{200, 90, 255, 255}
	cTrial    = color.RGBA{255, 140, 120, 255}
)

// fullParts reports the "Full particles" setting.
func (L *Level) fullParts() bool { return L.g.settings.Particles == 1 }

// familiarMotes: each familiar sheds a few drifting motes, and the first
// frame they're out they arrive with a little burst (updateFamiliars).
func (L *Level) familiarMotes(dt float64) {
	if L.g.ambientOff() {
		return
	}
	fams := L.familiars()
	if len(fams) == 0 {
		return
	}
	if !L.famIntro {
		L.famIntro = true
		for i, f := range fams {
			L.sparks(L.familiarPos(i, len(fams)), 10, f.col, 4)
		}
	}
	if !L.fullParts() {
		return
	}
	for i, f := range fams {
		if L.rng.Float64() > dt*5 {
			continue
		}
		p := L.familiarPos(i, len(fams))
		L.Parts = append(L.Parts, Particle{P: p.Add(V3{(L.rng.Float64() - 0.5) * 0.4, 0, (L.rng.Float64() - 0.5) * 0.4}),
			V: V3{0, 0.6, 0}, Life: 0.7, Max: 0.7, Size: 0.14, Col: lighten(f.col, 0.2), Kind: pSpark, Grav: -0.5})
	}
}

// familiarImpact marks where a familiar's shot lands.
func (L *Level) familiarImpact(p V3, c color.RGBA) {
	L.sparks(p, 5, c, 4)
}

// owlFlourish: feathers and a soft ring where the owl catches an orb.
func (L *Level) owlFlourish(x float64) {
	if L.g.flourishOff() {
		return
	}
	n := 8
	if !L.fullParts() {
		n = 3
	}
	for i := 0; i < n; i++ {
		a := L.rng.Float64() * 2 * math.Pi
		L.Parts = append(L.Parts, Particle{P: V3{x, 0.6, BarrierZ}, V: V3{math.Cos(a) * 2.5, 3 + L.rng.Float64()*2, math.Sin(a) * 1.5},
			Life: 1.1, Max: 1.1, Size: 0.22, Col: famCols[3], Kind: pDebris, Grav: 3, Rot: a, Spin: (L.rng.Float64() - 0.5) * 6})
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: BarrierZ, MaxR: 2.4, Life: 0.45, Max: 0.45, Col: famCols[3]})
}

// knotFlourish: the Guardian's Knot catching the run sends a gold ring
// across the arena.
func (L *Level) knotFlourish() {
	if L.g.flourishOff() {
		return
	}
	L.Rings = append(L.Rings,
		&Ring{X: L.PaddleX, Z: PaddleZ, MaxR: HalfW * 1.6, Life: 0.8, Max: 0.8, Col: cGoldRing},
		&Ring{X: L.PaddleX, Z: PaddleZ, MaxR: HalfW * 0.8, Life: 0.5, Max: 0.5, Col: cWhite})
	L.sparks(V3{L.PaddleX, 0.8, PaddleZ}, 24, cGoldRing, 8)
	if L.g.settings.Shake {
		L.Shake = math.Max(L.Shake, 0.3)
	}
}

// barrierRipple: a ring where the orb bounces off the barrier.
func (L *Level) barrierRipple(x float64) {
	L.Rings = append(L.Rings, &Ring{X: x, Z: BarrierZ, MaxR: 1.8, Life: 0.3, Max: 0.3, Col: color.RGBA{170, 140, 255, 255}})
}

// critRing: a quick bright ring on a critical hit (thinned when many land
// in one frame, like the particle bursts).
func (L *Level) critRing(x, z float64) {
	if L.bursts > burstsFull {
		return
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 1.3, Life: 0.2, Max: 0.2, Col: color.RGBA{255, 200, 200, 255}})
}

// coinGlint: falling coins leave a few sparkles.
func (L *Level) coinGlint(c *Coin, dt float64) {
	if L.g.ambientOff() {
		return
	}
	if !L.fullParts() || L.rng.Float64() > dt*7 {
		return
	}
	L.Parts = append(L.Parts, Particle{P: V3{c.X, c.Y, c.Z}, Life: 0.3, Max: 0.3, Size: 0.16, Col: color.RGBA{255, 236, 150, 255}, Kind: pSpark})
}

// ---------------------------------------------------------------- screens

// lastLifeEdge tints the screen edges red on the last life, pulsing with
// the heartbeat and stronger while an orb is dropping.
func (g *Game) lastLifeEdge(dst *ebiten.Image, L *Level) {
	r := g.run
	if r == nil || L == nil || g.duel != nil || r.Lives != 1 || r.maxLives() <= 1 || L.Cleared {
		return
	}
	k := 0.35
	if L.orbFalling() {
		k = 0.75
	}
	if !g.settings.Flashes {
		k *= 0.5
	}
	beat := math.Pow(0.5+0.5*math.Sin(g.ui.time*8.4), 6)
	a := uint8(math.Min(150, (38+60*beat)*k))
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	red := color.RGBA{a, 0, uint8(float64(a) * 0.15), a}
	clr := color.RGBA{}
	e := sw * 0.12
	fillPath(dst, boxPath(0, 0, e, sh, 0, false), red, clr, hgrad(0, e), false)
	fillPath(dst, boxPath(sw-e, 0, e, sh, 0, false), clr, red, hgrad(sw-e, sw), false)
	ev := sh * 0.14
	fillPath(dst, boxPath(0, sh-ev, sw, ev, 0, false), clr, red, vgrad(sh-ev, sh), false)
}

// drawRunCharms shows the run's keepsake and trial points under the lives.
func (g *Game) drawRunCharms(dst *ebiten.Image, x, y float64) {
	u := g.ui
	r := g.run
	if r == nil || g.duel != nil {
		return
	}
	cx := x
	if k := keepsakeByID[r.Keepsake]; k != nil {
		a := 1.0
		if r.Keepsake == "knot" && r.KeepUsed {
			a = 0.3 // spent
		}
		g.charmFlare(dst, cx, y)
		g.charmReminder(dst, cx, y)
		drawSprite(dst, sprSoft, u.X(cx-22), u.Y(y-22), 44*u.S, 44*u.S, ca(cGold, 0.25*a), true)
		u.drawEmblem(dst, glyphImage(k.Glyph, cGold), cx, y, 26, a)
		if u.hover(cx-16, y-16, 32, 32) {
			g.tooltip = tr(k.Name) + " " + roman(r.KeepRank) + " — " + k.Desc(r.KeepRank)
		}
		cx += 36
	}
	if g.level != nil && g.state != StMap {
		cx = g.familiarDots(dst, g.level, cx, y) + 10
	}
	if n := trialPoints(r.Trials); n > 0 {
		for i := 0; i < min(n, 6); i++ {
			px := cx + float64(i)*12
			rate := 6.0
			if g.level != nil && g.level.Boss != nil {
				rate = 14 // Relentless or not, a Guardian stokes them
			}
			fl := 0.8 + 0.2*math.Sin(u.time*rate+float64(i)*1.7)
			fillPath(dst, circlePath(u.X(px), u.Y(y), 4.2*fl*u.S), lighten(cTrial, 0.3), cTrial, vgrad(u.Y(y-4), u.Y(y+4)), false)
		}
		g.trialFlare(dst, cx, y, float64(min(n, 6))*12)
		if u.hover(cx-8, y-10, float64(min(n, 6))*12+8, 20) {
			g.tooltip = trJoin(tr("Trials"), ": ", trialNames(r.Trials))
		}
	}
}

// ---------------------------------------------------------------- banish

type banishFX struct {
	x, y, w, h, t float64
	col           color.RGBA
}

// addBanishFX starts the burn-away over a banished card's slot.
func (g *Game) addBanishFX(x, y, w, h float64, col color.RGBA) {
	if g.flourishOff() {
		return
	}
	g.banishFXs = append(g.banishFXs, &banishFX{x: x, y: y, w: w, h: h, col: col})
}

// drawBanishFX: purple flame sweeps up the card and embers rise off it.
func (g *Game) drawBanishFX(s *ebiten.Image) {
	u := g.ui
	S := u.S
	out := g.banishFXs[:0]
	for _, f := range g.banishFXs {
		f.t += u.DT
		k := f.t / 0.7
		if k >= 1 {
			continue
		}
		out = append(out, f)
		edge := f.y + f.h*(1-k)
		fillPath(s, boxPath(u.X(f.x), u.Y(edge), f.w*S, (f.y+f.h-edge)*S, 12*S, false), ca(cBanish, 0.0), ca(cBanish, 0.55*(1-k)), vgrad(u.Y(edge), u.Y(f.y+f.h)), true)
		drawSprite(s, sprSoft, u.X(f.x-30), u.Y(edge-40), (f.w+60)*S, 80*S, ca(cBanish, 0.7*(1-k)), true)
		for i := 0; i < 14; i++ {
			// a fixed scatter per ember, drifting up as the burn goes on
			fx := f.x + f.w*math.Mod(float64(i)*0.6180339, 1)
			fy := edge - k*160*math.Mod(float64(i)*0.371, 1) - 10
			sz := 14 + 10*math.Mod(float64(i)*0.29, 1)
			drawSprite(s, sprSoft, u.X(fx-sz/2), u.Y(fy-sz/2), sz*S, sz*S, ca(lighten(cBanish, 0.4), 0.8*(1-k)), true)
		}
	}
	g.banishFXs = out
}

// rarityPillar: a soft column of light behind a Legendary or Mythic card.
func (g *Game) rarityPillar(s *ebiten.Image, m *Mod, x, y, w, h float64) {
	if m.Rarity < Legendary {
		return
	}
	u := g.ui
	p := 0.75 + 0.25*math.Sin(g.stateT*2.4)
	c := rarityColors[m.Rarity]
	drawSprite(s, sprSoft, u.X(x+w*0.15), u.Y(y-200), w*0.7*u.S, (h+380)*u.S, ca(c, 0.2*p), true)
	drawSprite(s, sprSoft, u.X(x+w*0.35), u.Y(y-260), w*0.3*u.S, (h+300)*u.S, ca(lighten(c, 0.4), 0.16*p), true)
}

// lockBadge marks a locked shop item.
func (g *Game) lockBadge(s *ebiten.Image, x, y float64) {
	u := g.ui
	y += 3 * math.Sin(u.time*2.2) // the chain sways a little
	drawSprite(s, sprSoft, u.X(x-30), u.Y(y-30), 60*u.S, 60*u.S, ca(cGold, 0.4), true)
	fillPath(s, circlePath(u.X(x), u.Y(y), 20*u.S), rgba(60, 40, 20, 240), rgba(30, 18, 8, 240), vgrad(u.Y(y-20), u.Y(y+20)), false)
	strokePath(s, circlePath(u.X(x), u.Y(y), 20*u.S), 2*u.S, cGold, false)
	u.drawEmblem(s, glyphImage("chain", cGold), x, y, 26, 1)
}

// trialsGlow warms the Trials button as more points are on.
func (g *Game) trialsGlow(s *ebiten.Image, x, y, w, h float64) {
	n := trialPoints(g.selectedTrials())
	if n == 0 {
		return
	}
	u := g.ui
	k := math.Min(1, float64(n)/8)
	p := 0.8 + 0.2*math.Sin(g.stateT*3)
	drawSprite(s, sprSoft, u.X(x-30), u.Y(y-24), (w+60)*u.S, (h+48)*u.S, ca(cTrial, (0.12+0.3*k)*p), true)
}
