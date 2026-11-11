package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"math"
)

var chamberSigilCache = map[NodeType]*ebiten.Image{}

// Drawn once and cached: the combat path is an orb spell, rather than repeated crossed swords.
func chamberSigil(kind NodeType) *ebiten.Image {
	if kind != NCombat && kind != NElite && kind != NBoss {
		return nil
	}
	if im := chamberSigilCache[kind]; im != nil {
		return im
	}
	im := ebiten.NewImage(192, 192)
	c := nodeColors[kind]
	strokePath(im, circlePath(96, 96, 72), 2, ca(c, .8), true)
	for i := 0; i < 16; i++ {
		a := float64(i) * math.Pi / 8
		r0, r1 := 61.0, 65.0
		if i%4 == 0 {
			r0, r1 = 58, 69
		}
		strokeLine(im, 96+math.Cos(a)*r0, 96+math.Sin(a)*r0, 96+math.Cos(a)*r1, 96+math.Sin(a)*r1, 2, ca(lighten(c, .4), .9))
	}
	if kind == NCombat {
		for k := 0; k < 3; k++ {
			rot := float64(k) * math.Pi / 3
			for i := 0; i < 24; i++ {
				a, b := float64(i)*math.Pi/12, float64(i+1)*math.Pi/12
				xx, yy := math.Cos(a)*43, math.Sin(a)*17
				xx2, yy2 := math.Cos(b)*43, math.Sin(b)*17
				strokeLine(im, 96+xx*math.Cos(rot)-yy*math.Sin(rot), 96+xx*math.Sin(rot)+yy*math.Cos(rot), 96+xx2*math.Cos(rot)-yy2*math.Sin(rot), 96+xx2*math.Sin(rot)+yy2*math.Cos(rot), 2, lighten(c, .45))
			}
		}
	} else {
		tips := 6
		if kind == NBoss {
			tips = 8
		}
		for i := 0; i < tips; i++ {
			a := float64(i)*2*math.Pi/float64(tips) - math.Pi/2
			b := a + math.Pi*2/float64(tips)
			strokeLine(im, 96+math.Cos(a)*48, 96+math.Sin(a)*48, 96+math.Cos(b)*48, 96+math.Sin(b)*48, 3, lighten(c, .35))
			strokeLine(im, 96+math.Cos(a)*48, 96+math.Sin(a)*48, 96+math.Cos(a+math.Pi)*26, 96+math.Sin(a+math.Pi)*26, 1.5, ca(c, .7))
		}
	}
	fillPath(im, circlePath(96, 96, 15), lighten(c, .8), c, vgrad(81, 111), true)
	fillPath(im, circlePath(92, 91, 4), cWhite, cWhite, grad{}, true)
	chamberSigilCache[kind] = im
	return im
}
