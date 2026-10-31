package main

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The rendered UI kit (tools/artkit/scripts/ui_bl.py): nine-slice panel
// frames per theme, the bar frame and HUD icons, lit in Cycles.
//
//go:embed assets/ui
var uiKitFS embed.FS

var uiKitCache = map[string]*ebiten.Image{}

func uiKit(name string) *ebiten.Image {
	if img, ok := uiKitCache[name]; ok {
		return img
	}
	var img *ebiten.Image
	if b, err := uiKitFS.ReadFile("assets/ui/" + name + ".png"); err == nil {
		if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			img = ebiten.NewImageFromImage(im)
		}
	}
	uiKitCache[name] = img
	return img
}

// frameKit names the rendered frame for the current theme.
func (u *UI) frameKit() *ebiten.Image {
	th := u.theme()
	if th == &themes[themeArcane] {
		return uiKit("frame_arcane")
	}
	switch th.Style {
	case styleHolo:
		return uiKit("frame_holo")
	case styleBrass:
		return uiKit("frame_brass")
	case styleCrystal:
		return uiKit("frame_crystal")
	case styleStone:
		return uiKit("frame_stone")
	case styleCloud:
		return uiKit("frame_cloud")
	case styleMetal:
		return uiKit("frame_metal")
	}
	return uiKit("frame_arcane")
}

// nineSlice draws img stretched over the screen rect (sx, sy, sw, sh) with
// its corners kept square: src is the slice inset in image pixels, dst the
// corner size on screen.
func nineSlice(dst, img *ebiten.Image, sx, sy, sw, sh, src, dc, a float64) {
	b := img.Bounds()
	iw, ih := float64(b.Dx()), float64(b.Dy())
	dc = math.Min(dc, math.Min(sw, sh)/2)
	xs := [4]float64{sx, sx + dc, sx + sw - dc, sx + sw}
	ys := [4]float64{sy, sy + dc, sy + sh - dc, sy + sh}
	us := [4]float64{0, src, iw - src, iw}
	vs := [4]float64{0, src, ih - src, ih}
	verts := make([]ebiten.Vertex, 0, 16)
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			verts = append(verts, ebiten.Vertex{DstX: float32(xs[i]), DstY: float32(ys[j]), SrcX: float32(us[i]), SrcY: float32(vs[j]),
				ColorR: float32(a), ColorG: float32(a), ColorB: float32(a), ColorA: float32(a)})
		}
	}
	idx := make([]uint16, 0, 54)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			k := uint16(j*4 + i)
			idx = append(idx, k, k+1, k+5, k, k+5, k+4)
		}
	}
	dst.DrawTriangles(verts, idx, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

// kitFrame draws the rendered frame border for the current theme over a
// panel; false when the kit has no frame.
func (u *UI) kitFrame(dst *ebiten.Image, x, y, w, h, a float64) bool {
	img := u.frameKit()
	if img == nil {
		return false
	}
	S := u.S
	pad := 7.0
	corner := math.Min(46, math.Min(w, h)*0.5+pad)
	nineSlice(dst, img, u.X(x-pad), u.Y(y-pad), (w+2*pad)*S, (h+2*pad)*S, 104, corner*S, a)
	return true
}

// kitBar frames a bar well with the rendered capsule frame.
func (u *UI) kitBar(dst *ebiten.Image, x, y, w, h float64) bool {
	img := uiKit("bar")
	if img == nil {
		return false
	}
	S := u.S
	pad := h * 0.42
	bh := h + 2*pad
	b := img.Bounds()
	// the caps are square: as wide as the frame is tall
	cap := float64(b.Dy())
	sx, sy, sw, sh := u.X(x-pad-bh*0.18), u.Y(y-pad), (w+2*pad+bh*0.36)*S, bh*S
	dc := sh
	iw, ih := float64(b.Dx()), float64(b.Dy())
	xs := [4]float64{sx, sx + dc, sx + sw - dc, sx + sw}
	us := [4]float64{0, cap, iw - cap, iw}
	verts := make([]ebiten.Vertex, 0, 8)
	for j, yy := range [2]float64{sy, sy + sh} {
		for i := 0; i < 4; i++ {
			verts = append(verts, ebiten.Vertex{DstX: float32(xs[i]), DstY: float32(yy), SrcX: float32(us[i]), SrcY: float32(float64(j) * ih),
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
		}
	}
	idx := []uint16{0, 1, 5, 0, 5, 4, 1, 2, 6, 1, 6, 5, 2, 3, 7, 2, 7, 6}
	dst.DrawTriangles(verts, idx, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
	return true
}

// kitIcon returns the rendered version of a HUD icon, if there is one.
func kitIcon(k iconKind) *ebiten.Image {
	switch k {
	case icCoin:
		return uiKit("ic_coin")
	case icHeart:
		return uiKit("ic_heart")
	case icShield:
		return uiKit("ic_shield")
	case icShard:
		return uiKit("ic_gem")
	}
	return nil
}
