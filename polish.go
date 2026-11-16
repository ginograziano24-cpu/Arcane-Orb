package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Small presentation touches from the 2026-11-11 polish list.

// drawFlawless stamps FLAWLESS on the reward screen after a chamber cleared
// without dropping an orb (ROADMAP-50 #56): it slams down a beat after the
// cards are dealt, with its own stinger.
func (g *Game) drawFlawless(s *ebiten.Image) {
	if !g.flawless || g.pick.on {
		return
	}
	u := g.ui
	t := g.stateT - 0.85 // after the cards have turned over
	if t < 0 {
		return
	}
	if g.flawlessT == 0 {
		g.flawlessT = 1
		g.snd.PlayKeyed("flawless")
	}
	k := clamp01(t / 0.18)
	sc := 1 + 1.6*(1-ease(k)) // slams down from big
	a := math.Min(1, k*2)
	cx, cy := 1610.0, 110.0
	c := rgba(120, 230, 255, 255)
	if k >= 1 && t < 0.6 {
		// the dust ring as it lands
		e := (t - 0.18) / 0.42
		rr := 70 + 140*ease(e)
		drawSprite(s, sprRing, u.X(cx-rr), u.Y(cy-rr), 2*rr*u.S, 2*rr*u.S, ca(c, 0.7*(1-e)), true)
	}
	u.withScale(sc, cx, cy, 0, func() {
		fillPath(s, circlePath(u.X(cx), u.Y(cy), 78*u.S), ca(rgba(10, 30, 44, 255), 0.85*a), ca(rgba(6, 18, 28, 255), 0.85*a), vgrad(u.Y(cy-78), u.Y(cy+78)), false)
		strokePath(s, circlePath(u.X(cx), u.Y(cy), 78*u.S), 4*u.S, ca(c, a), false)
		strokePath(s, circlePath(u.X(cx), u.Y(cy), 66*u.S), 1.5*u.S, ca(c, 0.6*a), false)
		u.drawEmblem(s, glyphImage("mirror", c), cx, cy-26, 40, a)
		u.textOutlined(s, "FLAWLESS", fRune, 22, cx, cy+4, ca(lighten(c, 0.3), a), alignCenter)
		u.text(s, "no orb dropped", fReg, 13, cx, cy+36, ca(cDim, a), alignCenter)
	})
}

// drawSaleTag is the bouncing "40% off" ribbon on a shop card (#61).
func (g *Game) drawSaleTag(s *ebiten.Image, x, y float64) {
	u := g.ui
	bob := math.Sin(u.time*4) * 4
	w, h := 118.0, 38.0
	c := rgba(232, 52, 74, 255)
	tx, ty := x-w/2, y+bob
	drawSprite(s, sprSoft, u.X(tx-20), u.Y(ty-20), (w+40)*u.S, (h+40)*u.S, ca(c, 0.35), true)
	fillPath(s, boxPath(u.X(tx), u.Y(ty), w*u.S, h*u.S, 8*u.S, true), c, darken(c, 0.3), vgrad(u.Y(ty), u.Y(ty+h)), false)
	strokePath(s, boxPath(u.X(tx), u.Y(ty), w*u.S, h*u.S, 8*u.S, true), 2*u.S, rgba(255, 220, 140, 255), false)
	u.text(s, "SALE −40%", fBold, 18, x, ty+8, cWhite, alignCenter)
}
