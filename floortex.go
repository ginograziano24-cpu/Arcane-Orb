package main

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/bits"

	"orb/levelgen"
)

// Baked arena floors and ground tiles (tools/textures/floors.py and
// grounds.py). A floor covers x -14..14 and z -3..33; a ground tile
// repeats every groundTile world units.
//
//go:embed assets/floor/*.jpg
var floorFS embed.FS

const (
	floorX0, floorX1 = -14.0, 14.0
	floorZ0, floorZ1 = -3.0, 33.0
	groundTile       = 8.0
)

type bakedTex struct {
	id      uint8
	w, h    float32
	avg     color.RGBA // mean colour, for distant ground drawn untextured
	specOff float32    // brick sheet: the specular layer sits this far below the diffuse
}

var bakedTexCache = map[string]*bakedTex{}

// bakedTexture loads (once) an embedded texture; nil if it doesn't exist.
func bakedTexture(name string) *bakedTex {
	assetMu.RLock()
	t, ok := bakedTexCache[name]
	assetMu.RUnlock()
	if ok {
		return t
	}
	e := loadImage("floor/" + name) // (outside the lock: it may wait on a prefetch)
	assetMu.Lock()
	defer assetMu.Unlock()
	if t, ok := bakedTexCache[name]; ok {
		return t
	}
	t = nil
	if e.img != nil {
		t = &bakedTex{id: registerTex(e.img), w: float32(e.w), h: float32(e.h), avg: e.avg}
	}
	bakedTexCache[name] = t
	return t
}

var floorTexNames = map[int]string{WMeadow: "meadow", WRuins: "ruins", WCaverns: "caverns", WClouds: "clouds", WMachine: "machine"}

// floorTexture returns the baked floor for this level's world, or nil.
func floorTexture(L *Level) *bakedTex {
	if L.Biome.Neon {
		return bakedTexture("neon_" + itoa(L.Vis.FloorPattern%4))
	}
	if n, ok := floorTexNames[L.Biome.ID]; ok {
		return bakedTexture(n)
	}
	return nil
}

// groundTexture returns the baked ground tile around the arena, or nil.
func groundTexture(world int) *bakedTex {
	if n, ok := floorTexNames[world]; ok {
		return bakedTexture("ground_" + n)
	}
	return nil
}

// floorUV maps a world point on the floor into the baked floor texture.
func (t *bakedTex) floorUV(x, z float64) [2]float32 {
	u := (x - floorX0) / (floorX1 - floorX0) * float64(t.w)
	v := (floorZ1 - z) / (floorZ1 - floorZ0) * float64(t.h)
	return [2]float32{float32(u), float32(v)}
}

// The 1x1 textured quads inherit the clipping of their 2x2 arena tile.
// Compute their row masks once from the chamber's fixed wall polygon.
type floorClip [int(floorZ1 - floorZ0)]uint32

func makeFloorClip(walls []levelgen.Pt) (clip floorClip) {
	for row, z := 0, floorZ0; z < FarZ+1; row, z = row+1, z+1 {
		tz := floorZ0 + float64(row/2)*2
		for col, x := 0, floorX0; x < floorX1; col, x = col+2, x+2 {
			if levelgen.Inside(walls, levelgen.Pt{X: x + 1, Z: math.Min(tz+1, FarZ-0.2)}) {
				clip[row] |= 3 << col
			}
		}
	}
	return
}

// drawBakedFloor projects the chamber's textured quads in row-major order.
func drawBakedFloor(B *Batch, L *Level, t *bakedTex, gp float64) {
	bio := L.Biome
	const step = 1.0
	up := V3{0, 1, 0}
	white := B.lit(color.RGBA{255, 255, 255, 255}, up)
	for row, mask := range L.floorClip {
		z := floorZ0 + float64(row)
		c := white
		if bio.Neon {
			c = bio.Floor
			if row/2%4 == 0 {
				c = mixc(bio.Floor, bio.FloorAlt, 0.6)
			}
			c = B.lit(c, up)
		}
		for ; mask != 0; mask &= mask - 1 {
			x := floorX0 + float64(bits.TrailingZeros32(mask))
			pts := [4]V3{{x, 0, z + step}, {x + step, 0, z + step}, {x + step, 0, z}, {x, 0, z}}
			uv := [4][2]float32{t.floorUV(x, z+step), t.floorUV(x+step, z+step), t.floorUV(x+step, z), t.floorUV(x, z)}
			B.QuadUV(pts, t.id, uv, c, [4]float32{1, 1, 1, 1}, false)
			B.lastQuad().depth += 2000
			if bio.Neon {
				B.QuadUV(pts, t.id, uv, alpha(c, 0.3*gp), [4]float32{1, 1, 1, 1}, true)
				B.lastGlow().depth += 2000
			}
		}
	}
}

func meanColour(src image.Image) color.RGBA {
	b := src.Bounds()
	var r, g, bl, n uint64
	for y := b.Min.Y; y < b.Max.Y; y += 16 {
		for x := b.Min.X; x < b.Max.X; x += 16 {
			cr, cg, cb, _ := src.At(x, y).RGBA()
			r, g, bl, n = r+uint64(cr>>8), g+uint64(cg>>8), bl+uint64(cb>>8), n+1
		}
	}
	if n == 0 {
		return color.RGBA{128, 128, 128, 255}
	}
	return color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}
}

// drawBakedGround lays the repeating ground tile around the arena, finely
// tessellated; beyond the textured band the ground is its mean colour
// (no mipmaps, so a minified texture would only shimmer there).
func drawBakedGround(B *Batch, t *bakedTex, tint color.RGBA, y float64) {
	xs := []float64{-300, -200, -140, -100, -80, -64}
	for x := -56.0; x <= 56; x += 8 {
		xs = append(xs, x)
	}
	xs = append(xs, 64, 80, 100, 140, 200, 300)
	zs := []float64{-20, -12, -6}
	for z := 0.0; z <= 96; z += 8 {
		zs = append(zs, z)
	}
	zs = append(zs, 120, 150, 200, 280, 400)
	uv := func(x, z float64) [2]float32 {
		return [2]float32{float32(x / groundTile * float64(t.w)), float32(-z / groundTile * float64(t.h))}
	}
	far := mulc(t.avg, tint)
	// the plain outer ring first, then the textured band: two runs of one
	// texture each instead of a switch at every tile along the band's edge
	for pass := 0; pass < 2; pass++ {
		for i := 0; i+1 < len(xs); i++ {
			for j := 0; j+1 < len(zs); j++ {
				x0, x1, z0, z1 := xs[i], xs[i+1], zs[j], zs[j+1]
				pts := [4]V3{{x0, y, z1}, {x1, y, z1}, {x1, y, z0}, {x0, y, z0}}
				inner := math.Max(math.Abs(x0), math.Abs(x1)) <= 64 && z1 <= 96
				switch {
				case inner && pass == 1:
					B.QuadUV(pts, t.id, [4][2]float32{uv(x0, z1), uv(x1, z1), uv(x1, z0), uv(x0, z0)}, tint, [4]float32{1, 1, 1, 1}, false)
				case !inner && pass == 0:
					B.Quad(pts, sprWhite, far, [4]float32{1, 1, 1, 1}, false)
				default:
					continue
				}
				B.lastQuad().depth = 1e9
			}
		}
	}
}

func mulc(a, b color.RGBA) color.RGBA {
	return color.RGBA{uint8(int(a.R) * int(b.R) / 255), uint8(int(a.G) * int(b.G) / 255), uint8(int(a.B) * int(b.B) / 255), 255}
}

// ---------------------------------------------------------------- bricks

// The brick sheet (tools/textures/bricks.py): 8 x 6 cells of 256 x 128.
var brickSheetCol = map[int]int{WNeon: 0, WMeadow: 1, WCaverns: 2, WRuins: 3, WClouds: 4, WMachine: 5}

var brickSpecialCell = map[levelgen.Kind]int{
	levelgen.Armored: 0, levelgen.Explosive: 1, levelgen.Chain: 2, levelgen.Split: 3, levelgen.Regen: 4,
	levelgen.Moving: 5, levelgen.Shield: 6, levelgen.Teleport: 7, levelgen.Spawner: 8,
	levelgen.Multiplier: 9, levelgen.Crystal: 10, levelgen.Mystery: 11,
}

type sheetRect struct{ u0, v0, u1, v1 float32 }

var brickCellW, brickCellH float32 = 256, 128

func brickCell(i, j int) sheetRect {
	w, h, in := brickCellW, brickCellH, float32(2)
	return sheetRect{float32(i)*w + in, float32(j)*h + in, float32(i+1)*w - in, float32(j+1)*h - in}
}

// brickSheet loads the brick textures: the diffuse sheet and, when there is
// one, the specular sheet stacked below it in the same image (highlights
// as premultiplied white over alpha), so a brick's shine draws right after
// its colour without switching textures.
func brickSheet() *bakedTex {
	assetMu.RLock()
	t, ok := bakedTexCache["#bricks"]
	assetMu.RUnlock()
	if ok {
		return t
	}
	e := loadImage("bricks/sheet")
	assetMu.Lock()
	defer assetMu.Unlock()
	if t, ok := bakedTexCache["#bricks"]; ok {
		return t
	}
	t = nil
	if e.img != nil {
		h := e.aux // the diffuse layer's height
		t = &bakedTex{id: registerTex(e.img), w: float32(e.w), h: float32(e.h)}
		if e.h > h {
			t.specOff = float32(h)
		}
		brickCellW, brickCellH = float32(e.w)/8, float32(h)/6
	}
	bakedTexCache["#bricks"] = t
	return t
}

// composeBrickSheet builds the brick sheet image (see brickSheet) and
// returns it with the diffuse layer's height.
func composeBrickSheet() (image.Image, int) {
	dimg := decodeEmbedded(floorFS.ReadFile("assets/floor/bricks.jpg"))
	if dimg == nil {
		return nil, 0
	}
	b := dimg.Bounds()
	w, h := b.Dx(), b.Dy()
	simg := decodeEmbedded(floorFS.ReadFile("assets/floor/bricks_spec.jpg"))
	rows := h
	if simg != nil {
		rows = 2 * h
	}
	out := image.NewRGBA(image.Rect(0, 0, w, rows))
	draw.Draw(out, image.Rect(0, 0, w, h), dimg, b.Min, draw.Src)
	if simg != nil {
		sb := simg.Bounds()
		jpeg, isYCbCr := simg.(*image.YCbCr)
		for y := 0; y < h && y < sb.Dy(); y++ {
			for x := 0; x < w && x < sb.Dx(); x++ {
				var r, g, bl uint32
				if isYCbCr {
					r, g, bl, _ = jpeg.YCbCrAt(sb.Min.X+x, sb.Min.Y+y).RGBA()
				} else {
					r, g, bl, _ = simg.At(sb.Min.X+x, sb.Min.Y+y).RGBA()
				}
				l := (r*30 + g*59 + bl*11) / 100 >> 8
				i := out.PixOffset(x, h+y)
				out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = uint8(r>>8), uint8(g>>8), uint8(bl>>8), uint8(min(255, l))
			}
		}
	}
	return out, h
}

// brickShine scales the baked highlights.
const brickShine = 1.0

// brickRects picks the top and side cells for a brick, and whether its
// colours are its own (specials) rather than tinted.
func brickRects(world int, k levelgen.Kind) (top, side sheetRect, own bool) {
	switch k {
	case levelgen.Steel:
		return brickCell(6, 0), brickCell(6, 1), false
	case levelgen.Elite:
		return brickCell(7, 0), brickCell(7, 1), true
	}
	if i, ok := brickSpecialCell[k]; ok {
		return brickCell(i%8, 2+i/8), brickCell(i%8, 4+i/8), true
	}
	c := brickSheetCol[world]
	return brickCell(c, 0), brickCell(c, 1), false
}

// drawTexBrick draws a brick as a bevelled, textured block: an inset top,
// chamfered edges that catch the light, and the faces toward the camera.
func drawTexBrick(B *Batch, t *bakedTex, x0, y0, z0, x1, y1, z1 float64, col color.RGBA, top, side sheetRect, emis float32) {
	eye := B.cam.Eye
	c := math.Min(0.07, math.Min((x1-x0), (z1-z0))*0.2)
	if y1-y0 < c*2.5 {
		c = (y1 - y0) * 0.35
	}
	ix0, ix1, iz0, iz1, iy := x0+c, x1-c, z0+c, z1-c, y1-c
	// the baked tiles already carry their lighting: the world's light only
	// nudges the colour
	cx, cy, cz := (x0+x1)/2, (y0+y1)/2, (z0+z1)/2
	hx, hy, hz := (x1-x0)/2, (y1-y0)/2, (z1-z0)/2
	// only the point lights that can reach this brick at all: one whose
	// reach ends short of the brick's bounding sphere can reach none of its
	// faces, so leaving it out changes nothing but the time spent
	var near [maxPtLights]int8
	nn := 0
	rb := math.Sqrt(hx*hx+hy*hy+hz*hz) + 1e-6
	for i := range B.pts {
		l := &B.pts[i]
		dx, dy, dz := l.p.X-cx, l.p.Y-cy, l.p.Z-cz
		if reach := l.r + rb; dx*dx+dy*dy+dz*dz < reach*reach {
			near[nn] = int8(i)
			nn++
		}
	}
	lc := func(n V3) color.RGBA {
		p := V3{cx + n.X*hx, cy + n.Y*hy, cz + n.Z*hz}
		return B.litPtSome(mixc(col, B.lit(col, n), 0.3), p, n, near[:nn])
	}
	sh := func(a, b, cc, d float32) [4]float32 { return [4]float32{a + emis, b + emis, cc + emis, d + emis} }
	// every face is drawn twice when the sheet has a specular layer: its
	// colour, then its highlights screened on top
	spec := color.RGBA{255, 255, 255, uint8(255 * clamp01(brickShine))}
	quad := func(p [4]V3, uv [4][2]float32, c color.RGBA, shades [4]float32) {
		B.QuadUV(p, t.id, uv, c, shades, false)
		if t.specOff > 0 {
			for i := range uv {
				uv[i][1] += t.specOff
			}
			B.QuadUV(p, t.id, uv, spec, [4]float32{1, 1, 1, 1}, false)
		}
	}
	// texture coordinates: the top cell covers the inset top; bevels use the
	// cell's rim; sides use the side cell
	tu := func(x float64) float32 { return top.u0 + (top.u1-top.u0)*float32((x-x0)/(x1-x0)) }
	tv := func(z float64) float32 { return top.v0 + (top.v1-top.v0)*float32((z1-z)/(z1-z0)) }
	topUV := func(p [4]V3) [4][2]float32 {
		var o [4][2]float32
		for i := range p {
			o[i] = [2]float32{tu(p[i].X), tv(p[i].Z)}
		}
		return o
	}
	if eye.Y > y1 {
		p := [4]V3{{ix0, y1, iz1}, {ix1, y1, iz1}, {ix1, y1, iz0}, {ix0, y1, iz0}}
		quad(p, topUV(p), lc(V3{0, 1, 0}), sh(1, 1, 1, 1))
		// front bevel (toward the camera)
		p = [4]V3{{ix0, y1, iz0}, {ix1, y1, iz0}, {x1, iy, z0}, {x0, iy, z0}}
		quad(p, topUV(p), lc(V3{0, 0.7, -0.7}), sh(1.25, 1.25, 1.05, 1.05))
		// side bevels
		if eye.X < x1 {
			p = [4]V3{{ix0, y1, iz1}, {ix0, y1, iz0}, {x0, iy, z0}, {x0, iy, z1}}
			quad(p, topUV(p), lc(V3{-0.7, 0.7, 0}), sh(1.15, 1.15, 0.95, 0.95))
		}
		if eye.X > x0 {
			p = [4]V3{{ix1, y1, iz0}, {ix1, y1, iz1}, {x1, iy, z1}, {x1, iy, z0}}
			quad(p, topUV(p), lc(V3{0.7, 0.7, 0}), sh(1.15, 1.15, 0.95, 0.95))
		}
		// back bevel shows when the camera is high above
		if eye.Y-y1 > (z1-eye.Z)*0.9 {
			p = [4]V3{{x0, iy, z1}, {x1, iy, z1}, {ix1, y1, iz1}, {ix0, y1, iz1}}
			quad(p, topUV(p), lc(V3{0, 0.7, 0.7}), sh(0.85, 0.85, 1, 1))
		}
	}
	uv := func(r sheetRect, x0f, x1f float32) [4][2]float32 {
		w := r.u1 - r.u0
		return [4][2]float32{{r.u0 + w*x0f, r.v0}, {r.u0 + w*x1f, r.v0}, {r.u0 + w*x1f, r.v1}, {r.u0 + w*x0f, r.v1}}
	}
	if eye.Z < z0 {
		quad([4]V3{{x0, iy, z0}, {x1, iy, z0}, {x1, y0, z0}, {x0, y0, z0}}, uv(side, 0, 1), lc(V3{0, 0, -1}), sh(0.95, 0.95, 0.78, 0.78))
	}
	ef := float32((z1 - z0) / (x1 - x0))
	if eye.X < x0 {
		quad([4]V3{{x0, iy, z1}, {x0, iy, z0}, {x0, y0, z0}, {x0, y0, z1}}, uv(side, 0, ef), lc(V3{-1, 0, 0}), sh(0.82, 0.82, 0.66, 0.66))
	} else if eye.X > x1 {
		quad([4]V3{{x1, iy, z0}, {x1, iy, z1}, {x1, y0, z1}, {x1, y0, z0}}, uv(side, 1-ef, 1), lc(V3{1, 0, 0}), sh(0.82, 0.82, 0.66, 0.66))
	}
}
