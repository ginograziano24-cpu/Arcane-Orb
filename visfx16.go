package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 16 (ROADMAP section AC): the run-end tally,
// events, the campfire, toasts and a few more charm cues.

// summaryCharms: the run's keepsake and trial embers under the run-end
// title; with a win and trials on, the embers rise in triumph.
func (g *Game) summaryCharms(s *ebiten.Image) {
	r := g.run
	if r == nil {
		return
	}
	x := 790.0 - 60
	if g.keepsakeBadge(s, x, 160, 30) {
		x += 40
	}
	if n := trialPoints(r.Trials); n > 0 {
		g.trialEmbersAt(s, x, 160)
		if g.victory {
			u := g.ui
			for i := 0; i < min(n*2, 14); i++ {
				ph := math.Mod(g.stateT*0.5+float64(i)*0.17, 1)
				px := x + float64(i%min(n, 8))*13 + 6*math.Sin(g.stateT*3+float64(i))
				py := 160 - ph*90
				drawSprite(s, sprSoft, u.X(px-5), u.Y(py-5), 10*u.S, 10*u.S, ca(cTrial, 0.7*(1-ph)), true)
			}
		}
	}
}

// summaryCompanions: in a win, the familiars circle the hero's portrait,
// and a co-op run shows player two's glow beside it.
func (g *Game) summaryCompanions(s *ebiten.Image) {
	r := g.run
	if r == nil {
		return
	}
	if r.Coop {
		g.coopMark(s, 260, 1030, 300)
	}
	if !g.victory {
		return
	}
	u := g.ui
	cols := familiarColours(&r.Stats)
	for i, c := range cols {
		a := g.stateT*0.8 + float64(i)*2*math.Pi/float64(len(cols))
		px, py := 190+math.Cos(a)*170, 560+math.Sin(a)*60
		drawSprite(s, sprSoft, u.X(px-16), u.Y(py-16), 32*u.S, 32*u.S, ca(c, 0.7), true)
		fillPath(s, circlePath(u.X(px), u.Y(py), 7*u.S), lighten(c, 0.4), c, vgrad(u.Y(py-7), u.Y(py+7)), false)
	}
}

// summaryAsh: in defeat, grey ash drifts down over the tally.
func (g *Game) summaryAsh(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	if g.victory || (g.run != nil && g.run.Endless) {
		return
	}
	u := g.ui
	n := 30
	if g.settings.Particles == 0 {
		n = 12
	}
	for i := 0; i < n; i++ {
		ph := math.Mod(g.stateT*0.08+float64(i)*0.0371, 1)
		x := 420 + 1450*math.Mod(float64(i)*0.618, 1) + 18*math.Sin(g.stateT*0.7+float64(i))
		y := 40 + ph*1000
		drawSprite(s, sprSoft, u.X(x-4), u.Y(y-4), 8*u.S, 8*u.S, ca(rgba(170, 160, 175, 255), 0.35*math.Sin(ph*math.Pi)), true)
	}
}

// eventOutcome: a warm glow behind a good outcome, a red pulse behind a bad
// one.
func (g *Game) eventOutcome(s *ebiten.Image, y float64) {
	if g.eventMsg == "" {
		return
	}
	u := g.ui
	if g.eventGoodT < 0.9 {
		g.eventGoodT += u.DT
	}
	k := 1 - g.eventGoodT/0.9
	c := cGold
	if g.eventBad {
		c = cRed
	}
	if k > 0 {
		drawSprite(s, sprSoft, u.X(960-460), u.Y(y-60), 920*u.S, 200*u.S, ca(c, 0.4*k), true)
	}
}

// eventHover: the event's sigil quickens while a choice is under the
// cursor.
func (g *Game) eventHover(s *ebiten.Image, hovering bool) {
	if g.flourishOff() {
		return
	}
	if !hovering {
		return
	}
	u := g.ui
	col := u.theme().Accent
	p := 0.6 + 0.4*math.Sin(g.stateT*6)
	drawSprite(s, sprSoft, u.X(960-140), u.Y(462-140), 280*u.S, 280*u.S, ca(col, 0.2*p), true)
}

// fireflies drift over the campfire.
func (g *Game) fireflies(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for i := 0; i < 16; i++ {
		t := g.stateT*0.3 + float64(i)*0.9
		x := 960 + 600*math.Sin(t*0.7+float64(i)) + 80*math.Sin(t*2.3)
		y := 700 + 180*math.Sin(t*0.5+float64(i)*1.3)
		bl := math.Max(0, math.Sin(t*3+float64(i)*2))
		drawSprite(s, sprSoft, u.X(x-8), u.Y(y-8), 16*u.S, 16*u.S, ca(rgba(255, 230, 120, 255), 0.7*bl), true)
	}
}

// restHoverFire: hovering Rest stokes the fire's glow.
func (g *Game) restHoverFire(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	if !u.hover(380, 380, 560, 76) {
		return
	}
	p := 0.7 + 0.3*math.Sin(g.stateT*9)
	drawSprite(s, sprSoft, u.X(960-320), u.Y(760-240), 640*u.S, 480*u.S, ca(rgba(255, 140, 50, 255), 0.18*p), true)
}

// toastSweep: a light sweeps across a toast as it arrives, and its colour
// bar glows.
func (g *Game) toastSweep(dst *ebiten.Image, x, y, w, h, t float64, k float64, col color.RGBA) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	drawSprite(dst, sprSoft, u.X(x+2), u.Y(y+8), 24*u.S, (h-16)*u.S, ca(col, 0.45*k), true)
	if t > 0.7 {
		return
	}
	e := t / 0.7
	sx := x + (w+80)*e - 40
	drawSprite(dst, sprSoft, u.X(sx-30), u.Y(y), 60*u.S, h*u.S, ca(cWhite, 0.25*math.Sin(e*math.Pi)*k), true)
}

// actIntroFamiliars: the familiars circle the act's sigil.
func (g *Game) actIntroFamiliars(s *ebiten.Image) {
	r := g.run
	cols := familiarColours(&r.Stats)
	if len(cols) == 0 {
		return
	}
	u := g.ui
	k := ease(clamp01(g.stateT - 0.5))
	for i, c := range cols {
		a := g.stateT*0.6 + float64(i)*2*math.Pi/float64(len(cols))
		px, py := 960+math.Cos(a)*330, 450+math.Sin(a)*330
		drawSprite(s, sprSoft, u.X(px-14), u.Y(py-14), 28*u.S, 28*u.S, ca(c, 0.7*k), true)
	}
}

// mapBanishPips: banishes left under the map's run panel.
func (g *Game) mapBanishPips(s *ebiten.Image) {
	r := g.run
	if r == nil || r.banishesLeft() <= 0 {
		return
	}
	g.banishPips(s, 360-float64(min(r.banishesLeft(), 6)-1)*8, 340)
}

// pauseFamiliars: the familiar dots beside the pause screen's charms.
func (g *Game) pauseFamiliars(s *ebiten.Image) {
	if g.level == nil || g.duel != nil {
		return
	}
	g.familiarDots(s, g.level, 960-6*float64(len(g.level.familiars())), 386)
}

// ktNextArrow: a small arrow pulses between Now and Next rank in the
// Keepsakes detail panel.
func (g *Game) ktNextArrow(s *ebiten.Image, x, y float64) {
	u := g.ui
	d := 4 * math.Sin(g.stateT*4)
	c := ca(cGold, 0.8)
	fillPath(s, ngonPath(u.X(x), u.Y(y+d), 8*u.S, 3, math.Pi/2), c, c, grad{}, false)
}
