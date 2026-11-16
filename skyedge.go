package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// skyFoot is the colour along the bottom edge of a sky painting. The space
// below the painting (visible wherever no 3D ground covers it, e.g. under
// Cloud Kingdom's arena) is filled with it, so the picture never ends in a
// hard band of flat sky colour.
var skyFootCache = map[*ebiten.Image]color.RGBA{}

func skyFoot(img *ebiten.Image) color.RGBA {
	if c, ok := skyFootCache[img]; ok {
		return c
	}
	b := img.Bounds()
	rows := 8
	sub := img.SubImage(image.Rect(b.Min.X, b.Max.Y-rows, b.Max.X, b.Max.Y)).(*ebiten.Image)
	pix := make([]byte, 4*b.Dx()*rows)
	sub.ReadPixels(pix)
	var r, g, bl, n int
	for i := 0; i+3 < len(pix); i += 16 {
		r, g, bl, n = r+int(pix[i]), g+int(pix[i+1]), bl+int(pix[i+2]), n+1
	}
	c := color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}
	skyFootCache[img] = c
	return c
}
