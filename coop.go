package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Couch co-op: a second player steers a second paddle with a controller.
// Both paddles return orbs and catch drops; lives, gold and the build are
// shared, and player one keeps the origin's ability and the scrolls.
//
// Balance: two paddles cover far more of the floor, so in co-op both are
// 20% narrower and bricks have 15% more health. Rewards are unchanged.
//
// Player two uses the last connected controller; while a co-op chamber is
// on, that pad is taken out of player one's input entirely. Player one
// plays with the mouse/keyboard or another controller.

const (
	coopWidth = 0.80 // each paddle's width in co-op
	coopHP    = 0.15 // extra brick health in co-op
)

// coopPad is player two's controller while a co-op chamber runs (-1: none).
var coopPad ebiten.GamepadID = -1

type coopInput struct {
	Move float64 // -1..1
	Fire bool
}

func (r *Run) applyCoop(s *Stats) {
	if r.Coop {
		s.PaddleWidth *= coopWidth
		s.BrickHPMult += coopHP
	}
}

// coopPadAvailable reports a controller for player two.
func coopPadAvailable() bool {
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			return true
		}
	}
	return false
}

// updateCoop picks player two's controller and reads it (Game.Update,
// before anything reads player one's pads).
func (g *Game) updateCoop() {
	coopPad = -1
	g.p2in = coopInput{}
	if g.run == nil || !g.run.Coop || g.level == nil || (g.state != StLevel && g.state != StPause) {
		return
	}
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && (coopPad < 0 || id > coopPad) {
			coopPad = id
		}
	}
	if coopPad < 0 || g.state != StLevel {
		return
	}
	id := coopPad
	m := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	if math.Abs(m) < 0.15 {
		m = 0
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
		m = -1
	}
	if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
		m = 1
	}
	g.p2in.Move = m
	for _, b := range []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightBottom, ebiten.StandardGamepadButtonFrontBottomRight} {
		if ebiten.IsStandardGamepadButtonPressed(id, b) {
			g.p2in.Fire = true
		}
	}
}

// startCoopLevel places player two's paddle in a new chamber (newLevel).
func (L *Level) startCoopLevel() {
	if !L.g.run.Coop {
		return
	}
	L.p2On = true
	L.PaddleX = -3.5
	L.p2X = 3.5
	for i := range L.p2Hist {
		L.p2Hist[i] = L.p2X
	}
	L.partnerJoin()
}

// updatePartner moves player two's paddle and lets them launch.
func (L *Level) updatePartner(dt float64, in coopInput) {
	if !L.p2On {
		return
	}
	half := L.PaddleW / 2
	if L.stun <= 0 {
		L.p2X += in.Move * 34 * dt
	}
	L.p2X = math.Max(-HalfW+half, math.Min(HalfW-half, L.p2X))
	L.trackPartner()
	L.partnerContact()
	for _, b := range L.Balls {
		if b.Stuck && b.OnP2 {
			h := L.PaddleW/2 - 0.2
			b.Off = math.Max(-h, math.Min(h, b.Off))
			b.X, b.Z = L.p2X+b.Off, PaddleZ+PaddleD/2+b.R+0.02
		}
	}
	if in.Fire && !L.Cleared {
		hadP2 := false
		for _, b := range L.Balls {
			hadP2 = hadP2 || (b.Stuck && b.OnP2)
		}
		if L.release() && hadP2 {
			L.partnerLaunch()
			L.g.snd.PlayVol("coopgo", 0.5)
		}
	}
}

// asPartner runs f with player two's paddle standing in for player one's,
// so every paddle effect (bounce angle, passives, glue) works for both.
func (L *Level) asPartner(f func()) {
	x := L.PaddleX
	L.PaddleX = L.p2X
	f()
	L.p2X, L.PaddleX = L.PaddleX, x
}

// collidePartner bounces an orb off player two's paddle.
func (L *Level) collidePartner(b *Ball) {
	if !L.p2On || b.Stuck {
		return
	}
	L.asPartner(func() {
		L.collidePaddle(b)
		if b.Stuck {
			b.OnP2 = true // glued: it rides player two's paddle
		}
	})
}

// partnerCatches reports whether player two's paddle is under (x, z).
func (L *Level) partnerCatches(x, z, slack float64) bool {
	return L.p2On && math.Abs(z-PaddleZ) <= PaddleD/2+0.4 && math.Abs(x-L.p2X) <= L.PaddleW/2+slack
}

// drawPartner draws player two's paddle (world.go) with a marker ring.
func (g *Game) drawPartner(B *Batch, L *Level) {
	if !L.p2On {
		return
	}
	L.drawPartnerTrail(B)
	L.partnerLink(B)
	L.drawingP2 = true
	L.asPartner(func() { g.drawPaddle(B, L) })
	L.drawingP2 = false
	col := color.RGBA{120, 220, 255, 255}
	B.Billboard(V3{L.p2X, 1.1, PaddleZ - 0.9}, 0.5, sprSoft, alpha(col, 0.55), true)
	B.Billboard(V3{L.PaddleX, 1.1, PaddleZ - 0.9}, 0.5, sprSoft, alpha(color.RGBA{255, 200, 120, 255}, 0.55), true)
}
