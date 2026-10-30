package main

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// Rendered UI icons (tools/artkit/scripts/icons.py): the map's room icons.
//
//go:embed assets/icons/*.png
var uiIconFS embed.FS

var uiIconCache = map[string]*ebiten.Image{}

func uiIcon(name string) *ebiten.Image {
	if img, ok := uiIconCache[name]; ok {
		return img
	}
	var img *ebiten.Image
	if b, err := uiIconFS.ReadFile("assets/icons/" + name + ".png"); err == nil {
		if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			img = ebiten.NewImageFromImage(im)
		}
	}
	uiIconCache[name] = img
	return img
}

var nodeIconFile = map[NodeType]string{
	NCombat: "node_combat", NElite: "node_elite", NShop: "node_shop", NTreasure: "node_treasure",
	NEvent: "node_event", NRest: "node_rest", NBoss: "node_boss",
}

// drawUIIcon draws a rendered icon centred on (x, y) in design units.
func (g *Game) drawUIIcon(s *ebiten.Image, img *ebiten.Image, x, y, size, alpha float64) {
	u := g.ui
	b := img.Bounds()
	sc := size * u.S / float64(b.Dx())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(u.X(x), u.Y(y))
	op.ColorScale.ScaleAlpha(float32(alpha))
	s.DrawImage(img, op)
}
