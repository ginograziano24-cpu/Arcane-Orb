package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

func makeIcon(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			d := math.Hypot(float64(x)+0.5-c, float64(y)+0.5-c) / c
			if d > 1 {
				continue
			}
			hl := math.Hypot(float64(x)+0.5-c*0.7, float64(y)+0.5-c*0.7) / c
			v := 1 - d*0.6
			col := color.RGBA{uint8(255 * v), uint8(70 * v), uint8(210 * v), 255}
			if d > 0.86 {
				col = color.RGBA{40, 230, 255, 255}
			}
			if hl < 0.3 {
				col = lighten(col, (0.3-hl)/0.3*0.9)
			}
			img.SetRGBA(x, y, col)
		}
	}
	return img
}

func main() {
	buildAtlas()
	ebiten.SetWindowSize(1600, 900)
	ebiten.SetWindowTitle("Arcane Orb")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowIcon([]image.Image{makeIcon(16), makeIcon(32), makeIcon(48), makeIcon(256)})
	ebiten.SetTPS(ebiten.SyncWithFPS) // one update per rendered frame: true high-refresh play
	ebiten.SetScreenClearedEveryFrame(true)
	g := NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
