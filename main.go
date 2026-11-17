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
	bootMark("package init")
	benchParse()
	if steamInit() {
		return // Steam is relaunching the game through the client
	}
	bootMark("Steam init")
	loadFeelOverrides()
	buildAtlas()
	bootMark("UI atlas")
	ebiten.SetWindowSize(1600, 900)
	ebiten.SetWindowTitle("Arcane Orb")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	bootMark("window and icons")
	ebiten.SetTPS(ebiten.SyncWithFPS) // one update per rendered frame: true high-refresh play
	ebiten.SetScreenClearedEveryFrame(true)
	g := NewGame()
	bootMark(bootSetUp)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// gameVersion is shown on the title screen.
const gameVersion = "v1.0 RC"

var windowIconSet bool

// syncWindowIcon sets the window icon the first time the game is windowed.
// Ebitengine can't set an icon in fullscreen, so it keeps the request
// pending and re-converts every icon image pixel by pixel each frame until
// it can (about 70,000 pixels a frame, around 18% of a fullscreen
// benchmark's CPU time). Setting it only while windowed avoids that; in
// fullscreen the taskbar shows the exe's own icon.
func syncWindowIcon(fullscreen bool) {
	if windowIconSet || fullscreen {
		return
	}
	windowIconSet = true
	ebiten.SetWindowIcon([]image.Image{makeIcon(16), makeIcon(32), makeIcon(48), makeIcon(256)})
}
