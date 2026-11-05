package main

import (
	"embed"
	_ "image/jpeg"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Set pieces painted behind the between-chamber screens: a campfire glade,
// the merchant's tent, a rune shrine and a treasure vault, rendered in
// Blender (tools/artkit/scripts/scenes_bl.py).
//
//go:embed assets/scenes
var sceneFS embed.FS

var sceneCache = map[string]*ebiten.Image{}

func sceneImage(name string) *ebiten.Image {
	if img, ok := sceneCache[name]; ok {
		return img
	}
	img := loadImage("scene/" + name).img
	sceneCache[name] = img
	return img
}

// drawScene fills the screen with a set piece (cover-fit, with a slow
// drift so it feels alive) and a veil so the interface stays readable.
func (g *Game) drawScene(s *ebiten.Image, name string, veil float64) bool {
	img := sceneImage(name)
	if img == nil {
		return false
	}
	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	b := img.Bounds()
	iw, ih := float64(b.Dx()), float64(b.Dy())
	k := 1.04 + 0.01*math.Sin(g.stateT*0.15)
	sc := sw / iw
	if ih*sc < sh {
		sc = sh / ih
	}
	sc *= k
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-iw/2, -ih/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(sw/2+math.Sin(g.stateT*0.11)*sw*0.006, sh/2)
	s.DrawImage(img, op)
	if veil > 0 {
		fillRectA(s, 0, 0, sw, sh, rgba(4, 2, 10, 255), veil)
	}
	vignette(s, sw, sh, 170)
	return true
}
