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
		g.drawnSince = false
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
	g.ui.Click = false
}
