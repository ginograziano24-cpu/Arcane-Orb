package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 13 (ROADMAP section W): mostly the Keepsakes tree
// screen (purchase feedback, branches opening, maxed and equipped
// keepsakes, a starry backdrop), plus a reminder of the carried charm as a
// chamber starts.

type ktFX struct {
	popID    string  // the keepsake just upgraded
	popT     float64 // time since
	popRank  int     // the rank it reached (its gem flashes)
	linkTo   string  // a keepsake whose branch just opened
	linkT    float64
	shards   int // shards shown last frame (for the counter flash)
	shardsOK bool
	flash    float64
	equipT   float64 // time since Equip was pressed
}

// ktStars: a faint field of twinkling stars behind the tree.
func (g *Game) ktStars(s *ebiten.Image) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for i := 0; i < 70; i++ {
		x := 40 + math.Mod(float64(i)*397.13, 1840)
		y := 160 + math.Mod(float64(i)*211.71, 820)
		tw := 0.5 + 0.5*math.Sin(g.stateT*(0.8+math.Mod(float64(i)*0.37, 1.2))+float64(i))
		sz := 3 + 4*math.Mod(float64(i)*0.53, 1)
		drawSprite(s, sprSoft, u.X(x-sz), u.Y(y-sz), 2*sz*u.S, 2*sz*u.S, ca(rgba(200, 190, 255, 255), 0.12+0.2*tw), true)
	}
}

// ktUpgraded starts the purchase effects for keepsake k reaching rank.
func (g *Game) ktUpgraded(k *keepsake, rank int) {
	f := &g.kt
	f.popID, f.popT, f.popRank = k.ID, 0, rank
	if rank == keepsakeParentRanks {
		for i := range keepsakes {
			if keepsakes[i].Parent == k.ID {
				f.linkTo, f.linkT = keepsakes[i].ID, 0 // a branch opens
			}
		}
	}
}

// ktNodePop: the upgraded node swells with light for a moment.
func (g *Game) ktNodePop(s *ebiten.Image, k *keepsake, x, y float64) {
	if g.flourishOff() {
		return
	}
	f := &g.kt
	if f.popID != k.ID || f.popT > 0.6 {
		return
	}
	u := g.ui
	e := f.popT / 0.6
	a := 1 - e
	grow := 20 + 50*e
	drawSprite(s, sprSoft, u.X(x-grow), u.Y(y-grow), (ktNodeW+2*grow)*u.S, (ktNodeH+2*grow)*u.S, ca(cGold, 0.45*a), true)
}

// ktGemFlash: the gem for the rank just bought flashes white.
func (g *Game) ktGemFlash(s *ebiten.Image, k *keepsake, x, y float64) {
	if g.flourishOff() {
		return
	}
	f := &g.kt
	if f.popID != k.ID || f.popT > 0.5 || f.popRank < 1 {
		return
	}
	u := g.ui
	px := x + 108 + float64(f.popRank-1)*24
	r := 10 + 18*f.popT/0.5
	drawSprite(s, sprSoft, u.X(px-r), u.Y(y+78-r), 2*r*u.S, 2*r*u.S, ca(cWhite, 0.9*(1-f.popT/0.5)), true)
}

// ktLinkPulse: a light runs down a link the moment its branch opens.
func (g *Game) ktLinkPulse(s *ebiten.Image, k *keepsake, x0, y0, x1, y1 float64) {
	if g.flourishOff() {
		return
	}
	f := &g.kt
	if f.linkTo != k.ID || f.linkT > 0.9 {
		return
	}
	u := g.ui
	e := f.linkT / 0.9
	px, py := x0+(x1-x0)*e, y0+(y1-y0)*e
	drawSprite(s, sprSoft, u.X(px-26), u.Y(py-26), 52*u.S, 52*u.S, ca(cGold, 0.9*(1-e*0.5)), true)
}

// ktLinkFlow: small lights drift down the links that are open.
func (g *Game) ktLinkFlow(s *ebiten.Image, x0, y0, x1, y1 float64) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	for i := 0; i < 2; i++ {
		e := math.Mod(g.stateT*0.5+float64(i)*0.5, 1)
		px, py := x0+(x1-x0)*e, y0+(y1-y0)*e
		drawSprite(s, sprSoft, u.X(px-6), u.Y(py-6), 12*u.S, 12*u.S, ca(cGold, 0.5*math.Sin(e*math.Pi)), true)
	}
}

// ktInvite: a node that's open but not bought yet breathes softly.
func (g *Game) ktInvite(s *ebiten.Image, x, y float64) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	p := 0.5 + 0.5*math.Sin(g.stateT*2.2)
	drawSprite(s, sprSoft, u.X(x-14), u.Y(y-12), (ktNodeW+28)*u.S, (ktNodeH+24)*u.S, ca(rgba(190, 170, 255, 255), 0.06+0.08*p), true)
}

// ktAffordable: a small shard glint on nodes whose next rank you can buy.
func (g *Game) ktAffordable(s *ebiten.Image, x, y float64) {
	u := g.ui
	p := 0.6 + 0.4*math.Sin(g.stateT*4)
	cx, cy := x+ktNodeW-24, y+22
	drawSprite(s, sprSoft, u.X(cx-12), u.Y(cy-12), 24*u.S, 24*u.S, ca(rgba(200, 180, 255, 255), 0.7*p), true)
	u.gem(s, cx, cy, 5, 7, rgba(210, 190, 255, 255), 1)
}

// ktMaxedSheen: a slow gold sheen crosses a fully upgraded node.
func (g *Game) ktMaxedSheen(s *ebiten.Image, x, y float64, i int) {
	if g.ambientOff() {
		return
	}
	u := g.ui
	e := math.Mod(g.stateT*0.35+float64(i)*0.23, 1)
	sx := x + (ktNodeW+60)*e - 30
	drawSprite(s, sprSoft, u.X(sx-24), u.Y(y), 48*u.S, ktNodeH*u.S, ca(cGold, 0.22*math.Sin(e*math.Pi)), true)
}

// ktParentHint: hovering a locked keepsake outlines the one it needs.
func (g *Game) ktParentHint(s *ebiten.Image, k *keepsake) {
	if k.Parent == "" {
		return
	}
	u := g.ui
	px, py := ktNodePos(keepsakeByID[k.Parent])
	p := 0.6 + 0.4*math.Sin(g.stateT*6)
	strokePath(s, boxPath(u.X(px-5), u.Y(py-5), (ktNodeW+10)*u.S, (ktNodeH+10)*u.S, 12*u.S, false), 2.5*u.S, ca(rgba(255, 150, 140, 255), p), false)
}

// ktShardFlash glows the shard counter when shards are spent.
func (g *Game) ktShardFlash(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	f := &g.kt
	if f.shardsOK && g.save.Shards != f.shards {
		f.flash = 1
	}
	f.shards, f.shardsOK = g.save.Shards, true
	if f.flash <= 0 {
		return
	}
	f.flash = math.Max(0, f.flash-u.DT*2.5)
	drawSprite(s, sprSoft, u.X(960-140), u.Y(104), 280*u.S, 50*u.S, ca(rgba(200, 180, 255, 255), 0.5*f.flash), true)
}

// ktDetailGlow: a slow glow behind the detail panel's emblem, and a ring
// when the keepsake is equipped.
func (g *Game) ktDetailGlow(s *ebiten.Image, x, y float64, maxed bool) {
	if g.flourishOff() {
		return
	}
	u := g.ui
	p := 0.75 + 0.25*math.Sin(g.stateT*1.8)
	c := rgba(190, 170, 255, 255)
	if maxed {
		c = cGold
	}
	drawSprite(s, sprSoft, u.X(x-80), u.Y(y-80), 160*u.S, 160*u.S, ca(c, 0.3*p), true)
	if f := &g.kt; f.equipT < 0.6 {
		e := f.equipT / 0.6
		strokePath(s, circlePath(u.X(x), u.Y(y), (50+60*e)*u.S), 3*u.S, ca(cGold, 1-e), false)
	}
}

// ktTick advances the screen's timers (once per frame).
func (g *Game) ktTick() {
	dt := g.ui.DT
	g.kt.popT += dt
	g.kt.linkT += dt
	g.kt.equipT += dt
}

// charmReminder: as a chamber starts, the HUD keepsake pulses once so you
// remember what you're carrying.
func (g *Game) charmReminder(dst *ebiten.Image, x, y float64) {
	L := g.level
	if L == nil || L.T > 1.6 || g.state != StLevel {
		return
	}
	u := g.ui
	e := L.T / 1.6
	r := 16 + 22*e
	strokePath(dst, circlePath(u.X(x), u.Y(y), r*u.S), 2*u.S, ca(cGold, 0.8*(1-e)), false)
}
