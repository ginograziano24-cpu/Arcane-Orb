package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// drawLastBrickMarkers points out the last few required bricks
// (ROADMAP-50 #13). When only a handful are left, a small bobbing chevron
// hangs over each one so the end of a chamber is never a hunt for a brick
// hiding behind steel. Shape, not just colour, so it reads for every
// colour filter. The paddle-bounce lean in lastBrickAim does the rest.
func (L *Level) drawLastBrickMarkers(screen *ebiten.Image, proj func(V3) (float64, float64)) {
	if L.Boss != nil || L.Cleared || L.Left <= 0 || L.Left > max(3, L.startLeft/12) {
		return
	}
	t := L.activeT
	size := max(5, 7*L.g.ui.S)
	ink := color.RGBA{255, 236, 140, 255}
	for _, b := range L.Bricks {
		if !b.Alive || b.Appear < 1 || !b.Kind.Required() {
			continue
		}
		top := float64(b.Layer+1)*LayerH + 0.5
		x, y := proj(V3{b.X() + b.OffX, top, b.Z() + b.OffZ})
		bob := math.Sin(t*5+b.X()) * size * 0.35
		a := 0.65 + 0.35*math.Sin(t*6)
		y += bob - size*1.6
		c := alpha(ink, a)
		strokeLine(screen, x-size, y-size*0.6, x, y+size*0.4, size*0.36, c)
		strokeLine(screen, x, y+size*0.4, x+size, y-size*0.6, size*0.36, c)
	}
}
