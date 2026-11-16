package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Brick emphasis (Settings > Effects; ROADMAP section AG): the bricks are
// the focal point.
//   - A bright rim on each brick's top edges facing the camera, so every
//     brick reads on its own against busy scenery.
//   - The scenery beyond the arena walls steps back: a soft, cool, darker
//     veil outside the play area, feathered at the walls, so the board is
//     the most vivid thing on screen.
// Off / subtle (default) / strong.

const (
	emphSubtle = 0 // the zero value, so old settings files get it
	emphStrong = 1
	emphOff    = 2
)

var emphNames = [...]string{"Subtle", "Strong", "Off"}

func (g *Game) brickEmph() int {
	if g.settings == nil {
		return emphSubtle
	}
	return g.settings.BrickEmph
}

// brickRim draws the bright edge along a brick's top: the front edge
// always (it faces the camera), the side edges when the camera sees them.
func (g *Game) brickRim(B *Batch, x0, y1, z0, x1, z1 float64, col color.RGBA) {
	e := g.brickEmph()
	if e == emphOff {
		return
	}
	a := 0.32
	if e == emphStrong {
		a = 0.55
	}
	c := alpha(lighten(col, 0.65), a)
	inset := math.Min(0.07, math.Min(x1-x0, z1-z0)*0.2)
	w := 0.045
	y := y1 + 0.004
	ix0, ix1, iz0, iz1 := x0+inset, x1-inset, z0+inset, z1-inset
	one := [4]float32{1, 1, 1, 1}
	// front edge
	B.Quad([4]V3{{ix0, y, iz0 + w}, {ix1, y, iz0 + w}, {ix1, y, iz0}, {ix0, y, iz0}}, sprWhite, c, one, true)
	if e != emphStrong {
		return // Subtle: front edge only (bench: side strips cost ~2-4% on dense chambers)
	}
	eye := B.cam.Eye
	side := alpha(lighten(col, 0.65), a*0.7)
	if eye.X < x1 {
		B.Quad([4]V3{{ix0, y, iz1}, {ix0 + w, y, iz1}, {ix0 + w, y, iz0}, {ix0, y, iz0}}, sprWhite, side, one, true)
	}
	if eye.X > x0 {
		B.Quad([4]V3{{ix1 - w, y, iz1}, {ix1, y, iz1}, {ix1, y, iz0}, {ix1 - w, y, iz0}}, sprWhite, side, one, true)
	}
}

var veilImg *ebiten.Image

// sceneryVeil darkens and cools everything outside the arena, feathered at
// the walls (drawn over the finished world, under the HUD).
func (g *Game) sceneryVeil(screen *ebiten.Image, L *Level, k float64) {
	e := g.brickEmph()
	if e == emphOff || L == nil || len(L.Walls) == 0 || g.photo != nil {
		return
	}
	a := float32(0.26)
	if e == emphStrong {
		a = 0.42
	}
	minX, maxX, maxZ := math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, w := range L.Walls {
		minX, maxX, maxZ = math.Min(minX, w.X), math.Max(maxX, w.X), math.Max(maxZ, w.Z)
	}
	cam := &g.cam
	pt := func(p V3) (float32, float32) {
		x, y, _ := cam.Project(p)
		return float32(x * k), float32(y * k)
	}
	nearZ := PaddleZ - 3
	// the play area: near-left, far-left, far-right, near-right
	var in [4][2]float32
	in[0][0], in[0][1] = pt(V3{minX, arenaWallH, nearZ})
	in[1][0], in[1][1] = pt(V3{minX, arenaWallH, maxZ})
	in[2][0], in[2][1] = pt(V3{maxX, arenaWallH, maxZ})
	in[3][0], in[3][1] = pt(V3{maxX, arenaWallH, nearZ})
	// The veil maps the arena's corners onto the screen's corners, which
	// only holds for the play camera looking down the arena from the
	// paddle's end. From any other angle (boss intro, finale, a camera
	// moved elsewhere) it would fold into huge dark bands, so skip it.
	if cam.Eye.Z >= nearZ-4 || math.Abs(cam.Eye.X) > HalfW || cam.fwd.Z < 0.5 || cam.depthOf(V3{minX, arenaWallH, nearZ}) <= 1 || cam.depthOf(V3{maxX, arenaWallH, nearZ}) <= 1 ||
		!(in[0][0] < in[3][0] && in[1][0] < in[2][0] && in[1][1] < in[0][1] && in[2][1] < in[3][1]) {
		return
	}
	sw, sh := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	cx := (in[0][0] + in[1][0] + in[2][0] + in[3][0]) / 4
	cy := (in[0][1] + in[1][1] + in[2][1] + in[3][1]) / 4
	feather := float32(70 * g.ui.S)
	var mid [4][2]float32
	for i, p := range in {
		dx, dy := p[0]-cx, p[1]-cy
		l := float32(math.Hypot(float64(dx), float64(dy)))
		if l < 1 {
			l = 1
		}
		mid[i] = [2]float32{p[0] + dx/l*feather, p[1] + dy/l*feather}
	}
	out := [4][2]float32{{-2, sh + 2}, {-2, -2}, {sw + 2, -2}, {sw + 2, sh + 2}}
	vc := func(p [2]float32, al float32) ebiten.Vertex {
		return ebiten.Vertex{DstX: p[0], DstY: p[1], SrcX: sprWhite.X0 + 0.5, SrcY: sprWhite.Y0 + 0.5,
			ColorR: 0.05 * al, ColorG: 0.05 * al, ColorB: 0.10 * al, ColorA: al}
	}
	var vs []ebiten.Vertex
	for _, p := range in {
		vs = append(vs, vc(p, 0))
	}
	for _, p := range mid {
		vs = append(vs, vc(p, a))
	}
	for _, p := range out {
		vs = append(vs, vc(p, a))
	}
	var is []uint16
	for i := uint16(0); i < 4; i++ {
		j := (i + 1) % 4
		// inner band: in -> mid (fades in), outer band: mid -> screen edge
		is = append(is, i, j, 4+j, i, 4+j, 4+i)
		is = append(is, 4+i, 4+j, 8+j, 4+i, 8+j, 8+i)
	}
	screen.DrawTriangles(vs, is, atlas, &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha})
}
