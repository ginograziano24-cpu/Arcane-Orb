package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 15 (ROADMAP section AA): the run's keepsake,
// trials, familiars and co-op shown on the menus between chambers (map,
// pause, act intro, main menu, origin screen), plus keepsake flavour on
// the shop and reward screens.

// keepsakeBadge draws the run's keepsake emblem with a soft glow (no
// keepsake: nothing). Returns true when it drew.
func (g *Game) keepsakeBadge(s *ebiten.Image, x, y, size float64) bool {
	r := g.run
	if r == nil {
		return false
	}
	k := keepsakeByID[r.Keepsake]
	if k == nil || r.KeepRank <= 0 {
		return false
	}
	u := g.ui
	a := 1.0
	if r.Keepsake == "knot" && r.KeepUsed {
		a = 0.35
	}
	p := 0.8 + 0.2*math.Sin(g.stateT*2)
	drawSprite(s, sprSoft, u.X(x-size), u.Y(y-size), 2*size*u.S, 2*size*u.S, ca(cGold, 0.3*a*p), true)
	u.drawEmblem(s, glyphImage(k.Glyph, cGold), x, y, size, a)
	if u.hover(x-size/2, y-size/2, size, size) {
		g.tooltip = tr(k.Name) + " " + roman(r.KeepRank) + " — " + k.Desc(r.KeepRank)
	}
	return true
}

// trialEmbersAt draws one ember per trial point in a row from x.
func (g *Game) trialEmbersAt(s *ebiten.Image, x, y float64) {
	if g.ambientOff() {
		return
	}
	r := g.run
	if r == nil {
		return
	}
	u := g.ui
	n := trialPoints(r.Trials)
	for i := 0; i < min(n, 8); i++ {
		px := x + float64(i)*13
		fl := 0.8 + 0.2*math.Sin(g.stateT*6+float64(i)*1.7)
		drawSprite(s, sprSoft, u.X(px-8), u.Y(y-8), 16*u.S, 16*u.S, ca(cTrial, 0.5*fl), true)
		fillPath(s, circlePath(u.X(px), u.Y(y), 4*fl*u.S), lighten(cTrial, 0.3), cTrial, vgrad(u.Y(y-4), u.Y(y+4)), false)
	}
	if n > 0 && u.hover(x-8, y-10, float64(min(n, 8))*13+8, 20) {
		g.tooltip = trJoin(tr("Trials"), ": ", trialNames(r.Trials))
	}
}

// familiarOrbit: the build's familiars circle the hero portrait on the
// map's run panel.
func (g *Game) familiarOrbit(s *ebiten.Image, cx, cy float64) {
	r := g.run
	if r == nil {
		return
	}
	u := g.ui
	cols := familiarColours(&r.Stats)
	for i, c := range cols {
		a := g.stateT*1.1 + float64(i)*2*math.Pi/float64(len(cols))
		px, py := cx+math.Cos(a)*62, cy+math.Sin(a)*30
		drawSprite(s, sprSoft, u.X(px-10), u.Y(py-10), 20*u.S, 20*u.S, ca(c, 0.7), true)
		fillPath(s, circlePath(u.X(px), u.Y(py), 4.5*u.S), lighten(c, 0.4), c, vgrad(u.Y(py-4), u.Y(py+4)), false)
	}
}

// familiarColours lists the build's familiars by colour (no level needed).
func familiarColours(s *Stats) []color.RGBA {
	var out []color.RGBA
	for i := 0; i < s.Wisps; i++ {
		out = append(out, famCols[0])
	}
	if s.Embers > 0 {
		out = append(out, famCols[1])
	}
	if s.Moths > 0 {
		out = append(out, famCols[2])
	}
	if s.Owl {
		out = append(out, famCols[3])
	}
	return out
}

// coopMark: a teal second glow under the hero on the map panel and the
// origin screen when co-op is on.
func (g *Game) coopMark(s *ebiten.Image, cx, cy, w float64) {
	u := g.ui
	p := 0.75 + 0.25*math.Sin(g.stateT*2.5)
	drawSprite(s, sprSoft, u.X(cx-w/2), u.Y(cy-24), w*u.S, 48*u.S, ca(cP2, 0.35*p), true)
}

// pauseCharms: the run's keepsake, trial embers and banishes left, under
// the pause screen's seed line.
func (g *Game) pauseCharms(s *ebiten.Image) {
	r := g.run
	if r == nil || g.duel != nil {
		return
	}
	x := 960.0 - 90
	if g.keepsakeBadge(s, x, 352, 30) {
		x += 40
	}
	if trialPoints(r.Trials) > 0 {
		g.trialEmbersAt(s, x, 352)
		x += float64(min(trialPoints(r.Trials), 8))*13 + 16
	}
	g.banishPips(s, x+float64(min(r.banishesLeft(), 6)-1)*8+8, 352)
}

// actIntroCharm: the carried keepsake rises into view under the act's name.
func (g *Game) actIntroCharm(s *ebiten.Image) {
	t := clamp01((g.stateT - 0.8) / 0.8)
	if t <= 0 {
		return
	}
	u := g.ui
	y := 760 - 20*ease(t)
	r := g.run
	if k := keepsakeByID[r.Keepsake]; k != nil && r.KeepRank > 0 {
		drawSprite(s, sprSoft, u.X(960-40), u.Y(y-40), 80*u.S, 80*u.S, ca(cGold, 0.35*t), true)
		u.drawEmblem(s, glyphImage(k.Glyph, cGold), 960, y, 40, t)
	}
}

// actIntroEmbers: with trials on, embers rise past the act title.
func (g *Game) actIntroEmbers(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	n := trialPoints(g.run.Trials)
	if n == 0 {
		return
	}
	u := g.ui
	for i := 0; i < min(n*3, 18); i++ {
		ph := math.Mod(g.stateT*0.35+float64(i)*0.173, 1)
		x := 560 + 800*math.Mod(float64(i)*0.618, 1) + 10*math.Sin(g.stateT*2+float64(i))
		y := 700 - ph*420
		drawSprite(s, sprSoft, u.X(x-6), u.Y(y-6), 12*u.S, 12*u.S, ca(cTrial, 0.6*math.Sin(ph*math.Pi)), true)
	}
}

// menuKeepBadge: a shard gem beside the main menu's Keepsakes entry when an
// upgrade is affordable.
func (g *Game) menuKeepBadge(s *ebiten.Image, y float64) {
	if !g.keepAffordable() {
		return
	}
	u := g.ui
	p := 0.6 + 0.4*math.Sin(g.stateT*4)
	drawSprite(s, sprSoft, u.X(92-14), u.Y(y+22-14), 28*u.S, 28*u.S, ca(rgba(200, 180, 255, 255), 0.8*p), true)
	u.gem(s, 92, y+22, 6, 8, rgba(215, 195, 255, 255), 1)
}

// tokenShimmer: with the Trader's Token, a gold sheen crosses the shop's
// heading now and then.
func (g *Game) tokenShimmer(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	if g.run == nil || g.run.Keepsake != "token" {
		return
	}
	u := g.ui
	e := math.Mod(g.stateT*0.4, 1.6)
	if e > 1 {
		return
	}
	x := 620 + 680*e
	drawSprite(s, sprSoft, u.X(x-40), u.Y(36), 80*u.S, 60*u.S, ca(cGold, 0.4*math.Sin(e*math.Pi)), true)
}

// cloverFall: with the Clover Pin, a few clover-green motes drift over the
// reward cards as they're dealt.
func (g *Game) cloverFall(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	if g.run == nil || g.run.Keepsake != "clover" || g.stateT > 1.6 {
		return
	}
	u := g.ui
	for i := 0; i < 10; i++ {
		t := g.stateT - float64(i)*0.08
		if t < 0 {
			continue
		}
		x := 420 + 1080*math.Mod(float64(i)*0.618, 1) + 20*math.Sin(t*3+float64(i))
		y := 220 + t*260
		a := math.Max(0, 1-t/1.4)
		drawSprite(s, sprSoft, u.X(x-7), u.Y(y-7), 14*u.S, 14*u.S, ca(rgba(120, 230, 120, 255), 0.7*a), true)
	}
}

// treeComplete: a crown of light over the Keepsakes title once every rank
// is bought.
func (g *Game) treeComplete(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	for i := range keepsakes {
		if g.save.keepRank(keepsakes[i].ID) < keepsakes[i].Ranks {
			return
		}
	}
	u := g.ui
	p := 0.7 + 0.3*math.Sin(g.stateT*2)
	drawSprite(s, sprSoft, u.X(960-320), u.Y(10), 640*u.S, 120*u.S, ca(cGold, 0.35*p), true)
}

// trialsHeatBack: the Trials panel's backdrop warms with the points on.
func (g *Game) trialsHeatBack(s *ebiten.Image, x, y, w, h float64) {
	if g.ambientOff() {
		return
	}
	n := trialPoints(g.selectedTrials())
	if n == 0 {
		return
	}
	u := g.ui
	k := math.Min(1, float64(n)/10)
	drawSprite(s, sprSoft, u.X(x-60), u.Y(y+h*0.4), (w+120)*u.S, h*0.8*u.S, ca(cTrial, 0.12+0.18*k), true)
}

// ringOnBanish: with the Banisher's Ring, its emblem sits by the Banish
// button.
func (g *Game) ringOnBanish(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	if g.run == nil || g.run.Keepsake != "ring" || g.run.banishesLeft() <= 0 {
		return
	}
	u := g.ui
	p := 0.7 + 0.3*math.Sin(g.stateT*3)
	drawSprite(s, sprSoft, u.X(456-18), u.Y(930-18), 36*u.S, 36*u.S, ca(cBanish, 0.4*p), true)
	u.drawEmblem(s, glyphImage("ring", lighten(cBanish, 0.3)), 456, 930, 26, 1)
}

// featherGleam: with the Feather Charm, the paddle's ends gleam as a
// chamber starts.
func (L *Level) featherGleam(B *Batch) {
	if L.g.ambientOff() {
		return
	}
	if L.g.run.Keepsake != "feather" || L.T > 2 {
		return
	}
	a := 0.6 * (1 - L.T/2)
	for _, sx := range []float64{-1, 1} {
		B.Billboard(V3{L.PaddleX + sx*L.PaddleW/2, 0.6, PaddleZ}, 0.9, sprSoft, alpha(cWhite, a), true)
	}
}
