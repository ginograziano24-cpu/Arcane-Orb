package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 11 (ROADMAP section R): feedback on shop and
// reward buttons, familiar build cues and co-op teamwork. A small screen
// spark system covers the menu bursts; everything else is a few sprites.

type uiSpark struct {
	x, y, vx, vy, life, max, size float64
	col                           color.RGBA
}

// uiBurst throws n sparks from (x, y) in virtual screen coordinates; up
// tilts them upward (coins, hearts) rather than all round.
func (g *Game) uiBurst(x, y float64, n int, col color.RGBA, up bool) {
	if g.flourishOff() {
		return
	}
	if g.settings.Particles == 0 {
		n = max(2, n/3)
	}
	for i := 0; i < n; i++ {
		a := g.rng.Float64() * 2 * math.Pi
		sp := 120 + g.rng.Float64()*220
		vx, vy := math.Cos(a)*sp, math.Sin(a)*sp
		if up {
			vx, vy = (g.rng.Float64()-0.5)*180, -160-g.rng.Float64()*220
		}
		l := 0.5 + g.rng.Float64()*0.4
		g.uiSparks = append(g.uiSparks, &uiSpark{x: x, y: y, vx: vx, vy: vy, life: l, max: l, size: 10 + g.rng.Float64()*10, col: col})
	}
}

// drawUISparks moves and draws the screen sparks (shop and reward screens).
func (g *Game) drawUISparks(s *ebiten.Image) {
	u := g.ui
	out := g.uiSparks[:0]
	for _, p := range g.uiSparks {
		p.life -= u.DT
		if p.life <= 0 {
			continue
		}
		p.vy += 420 * u.DT
		p.x += p.vx * u.DT
		p.y += p.vy * u.DT
		k := p.life / p.max
		sz := p.size * (0.5 + 0.5*k)
		drawSprite(s, sprSoft, u.X(p.x-sz/2), u.Y(p.y-sz/2), sz*u.S, sz*u.S, ca(p.col, k), true)
		out = append(out, p)
	}
	g.uiSparks = out
}

// banishPips: one purple dot per banish left, under the Banish button.
func (g *Game) banishPips(s *ebiten.Image, x, y float64) {
	u := g.ui
	n := g.run.banishesLeft()
	for i := 0; i < min(n, 6); i++ {
		px := x + float64(i)*16 - float64(min(n, 6)-1)*8
		fillPath(s, circlePath(u.X(px), u.Y(y), 4.5*u.S), lighten(cBanish, 0.3), cBanish, vgrad(u.Y(y-4), u.Y(y+4)), false)
	}
}

// trialHeat: the trial total turns from amber to deep red as it climbs.
func trialHeat(n int) color.RGBA {
	return mixc(rgba(255, 210, 150, 255), rgba(255, 70, 60, 255), math.Min(1, float64(n)/10))
}

// epicHalo: a faint glow behind an Epic card (Legendary and Mythic get the
// full light column).
func (g *Game) epicHalo(s *ebiten.Image, m *Mod, x, y, w, h float64) {
	if g.flourishOff() {
		return
	}
	if m.Rarity != Epic {
		return
	}
	u := g.ui
	p := 0.75 + 0.25*math.Sin(g.stateT*2)
	drawSprite(s, sprSoft, u.X(x-40), u.Y(y-40), (w+80)*u.S, (h+80)*u.S, ca(rarityColors[Epic], 0.14*p), true)
}

// banishArrival: the card dealt in after a banish glows in.
func (g *Game) banishArrival(s *ebiten.Image, i int, x, y, w, h float64) {
	if g.flourishOff() {
		return
	}
	if i != g.banishSlot || g.banishArriveT >= 0.6 || !g.banishArriveOn {
		return
	}
	u := g.ui
	k := 1 - g.banishArriveT/0.6
	drawSprite(s, sprSoft, u.X(x-30), u.Y(y-30), (w+60)*u.S, (h+60)*u.S, ca(cBanish, 0.4*k), true)
}

// ---------------------------------------------------------------- world

// familiarPulse is how fast a familiar's glow breathes: quicker with
// Familiar Bond's haste.
func (L *Level) familiarPulse(i int) float64 {
	rate := 2.0 * (1 + 2*L.g.run.Stats.FamHaste)
	return 0.85 + 0.15*math.Sin(L.T*rate+float64(i))
}

// bannerAura: with the Spirit Banner, a faint ring of light under the
// paddle, wider for each kind of familiar.
func (L *Level) bannerAura(B *Batch) {
	if L.g.ambientOff() {
		return
	}
	s := &L.g.run.Stats
	if !s.FamBanner {
		return
	}
	k := float64(min(4, s.familiarKinds()))
	B.Billboard(V3{L.PaddleX, 0.25, PaddleZ}, L.PaddleW*(0.5+0.15*k), sprSoft, alpha(rgba(255, 230, 160, 255), 0.08+0.03*k), true)
}

// owlGlint: the ready owl's eyes catch the light every few seconds.
func (L *Level) owlGlint(B *Batch, p V3) {
	if L.g.flourishOff() {
		return
	}
	if L.owlCD > 0 || math.Mod(L.T, 3.2) > 0.18 {
		return
	}
	B.Billboard(p.Add(V3{-0.09, 0.05, -0.2}), 0.16, sprSpark, alpha(cWhite, 0.9), true)
	B.Billboard(p.Add(V3{0.09, 0.05, -0.2}), 0.16, sprSpark, alpha(cWhite, 0.9), true)
}

// partnerLaunch: player two's launch flashes their paddle.
func (L *Level) partnerLaunch() {
	L.sparks(V3{L.p2X, 0.6, PaddleZ}, 10, cP2, 5)
}

// partnerLink: a soft bridge of light when the two paddles are side by
// side.
func (L *Level) partnerLink(B *Batch) {
	if !L.p2On {
		return
	}
	gap := math.Abs(L.p2X - L.PaddleX)
	if gap > L.PaddleW*1.3 {
		return
	}
	a := 0.25 * (1 - gap/(L.PaddleW*1.3))
	mid := (L.p2X + L.PaddleX) / 2
	B.Billboard(V3{mid, 0.7, PaddleZ}, 0.9, sprSoft, alpha(mixc(cP2, rgba(255, 200, 120, 255), 0.5), a), true)
}
