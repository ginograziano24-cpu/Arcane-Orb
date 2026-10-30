package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

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
	id   uint8
	w, h float32
	avg  color.RGBA // mean colour, for distant ground drawn untextured
}

var bakedTexCache = map[string]*bakedTex{}

// bakedTexture loads (once) an embedded texture; nil if it doesn't exist.
func bakedTexture(name string) *bakedTex {
	if t, ok := bakedTexCache[name]; ok {
		return t
	}
	var t *bakedTex
	if data, err := floorFS.ReadFile("assets/floor/" + name + ".jpg"); err == nil {
		if src, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			img := ebiten.NewImageFromImage(src)
			b := src.Bounds()
			t = &bakedTex{id: registerTex(img), w: float32(b.Dx()), h: float32(b.Dy()), avg: meanColour(src)}
		}
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

// drawBakedFloor draws the arena floor from its baked texture in small
// quads (texture mapping is affine, so small pieces keep it straight),
// clipped to the arena. Neon floors are greyscale, tinted and pulsed.
func drawBakedFloor(B *Batch, L *Level, t *bakedTex, gp float64) {
	bio := L.Biome
	const step = 1.0
	white := B.lit(color.RGBA{255, 255, 255, 255}, V3{0, 1, 0})
	for z := floorZ0; z < FarZ+1; z += step {
		for x := floorX0; x < floorX1; x += step {
			// clip by the 2x2 tile the piece belongs to, as the old floor did
			tx := floorX0 + math.Floor((x-floorX0)/2)*2
			tz := floorZ0 + math.Floor((z-floorZ0)/2)*2
			if !levelgen.Inside(L.Walls, levelgen.Pt{X: tx + 1, Z: math.Min(tz+1, FarZ-0.2)}) {
				continue
			}
			pts := [4]V3{{x, 0, z + step}, {x + step, 0, z + step}, {x + step, 0, z}, {x, 0, z}}
			uv := [4][2]float32{t.floorUV(x, z+step), t.floorUV(x+step, z+step), t.floorUV(x+step, z), t.floorUV(x, z)}
			c := white
			if bio.Neon {
				c = bio.Floor
				if int(math.Floor((z-floorZ0)/2))%4 == 0 {
					c = mixc(bio.Floor, bio.FloorAlt, 0.6)
				}
				c = B.lit(c, V3{0, 1, 0})
			}
			B.QuadUV(pts, t.id, uv, c, [4]float32{1, 1, 1, 1}, false)
			B.quads[len(B.quads)-1].depth += 2000
			if bio.Neon {
				B.QuadUV(pts, t.id, uv, alpha(c, 0.3*gp), [4]float32{1, 1, 1, 1}, true)
				B.glow[len(B.glow)-1].depth += 2000
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
	for i := 0; i+1 < len(xs); i++ {
		for j := 0; j+1 < len(zs); j++ {
			x0, x1, z0, z1 := xs[i], xs[i+1], zs[j], zs[j+1]
			pts := [4]V3{{x0, y, z1}, {x1, y, z1}, {x1, y, z0}, {x0, y, z0}}
			if math.Max(math.Abs(x0), math.Abs(x1)) <= 64 && z1 <= 96 {
				B.QuadUV(pts, t.id, [4][2]float32{uv(x0, z1), uv(x1, z1), uv(x1, z0), uv(x0, z0)}, tint, [4]float32{1, 1, 1, 1}, false)
			} else {
				B.Quad(pts, sprWhite, far, [4]float32{1, 1, 1, 1}, false)
			}
			B.quads[len(B.quads)-1].depth = 1e9
		}
	}
}

func mulc(a, b color.RGBA) color.RGBA {
	return color.RGBA{uint8(int(a.R) * int(b.R) / 255), uint8(int(a.G) * int(b.G) / 255), uint8(int(a.B) * int(b.B) / 255), 255}
}
