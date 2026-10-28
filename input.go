package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Screens drawn in Draw read clicks and key presses there. When a frame
// takes longer than a tick, several Updates run per Draw and a one-tick
// "just pressed" would be lost, so presses are latched until drawn.

func (g *Game) latchInput() {
	if g.keyLatch == nil {
		g.keyLatch = map[ebiten.Key]bool{}
	}
	if g.drawnSince {
		g.clickLatch = false
		clear(g.keyLatch)
		g.navLatch = navInput{}
		g.drawnSince = false
	}
	if anyPadInput() {
		g.padActive = true
	} else if len(inpututil.AppendJustPressedKeys(nil)) > 0 || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.padActive = false
	} else if mx, my := ebiten.CursorPosition(); mx != g.lastCurX || my != g.lastCurY {
		g.padActive = false
		g.lastCurX, g.lastCurY = mx, my
	}
	n := g.pollNav(1.0 / 60)
	l := &g.navLatch
	if n.dx != 0 {
		l.dx = n.dx
	}
	if n.dy != 0 {
		l.dy = n.dy
	}
	l.confirm = l.confirm || n.confirm
	l.back = l.back || n.back
	l.pageL = l.pageL || n.pageL
	l.pageR = l.pageR || n.pageR
	if padJust(ebiten.StandardGamepadButtonRightBottom) {
		g.keyLatch[ebiten.KeyEnter] = true // A advances story and intro cards
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.clickLatch = true
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		g.keyLatch[k] = true
	}
}

// keyHit reports a key press for code that runs during Draw.
func (g *Game) keyHit(k ebiten.Key) bool { return g.keyLatch[k] }

// consumeInput drops the latched presses so one press acts only once.
func (g *Game) consumeInput() {
	g.clickLatch = false
	clear(g.keyLatch)
	g.navLatch = navInput{}
	g.ui.Click = false
	g.ui.nav.confirm = false
}

// backPressed is Escape or the pad's B button (for Update-time handlers).
func backPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEscape) || padJust(ebiten.StandardGamepadButtonRightRight)
}

// pausePressed is Escape, P or the pad's Start button.
func (g *Game) pausePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEscape) || g.actJust("pause")
}
