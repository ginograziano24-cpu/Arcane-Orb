package main

// A small purpose-built 3D renderer on top of Ebitengine's DrawTriangles:
// perspective camera, flat-shaded boxes with per-vertex lighting, billboards,
// floor decals and painter's-algorithm depth sorting. Everything is drawn in
// two batched calls per frame (opaque-ish + additive glow) from one atlas.

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type V3 struct{ X, Y, Z float64 }

func (a V3) Add(b V3) V3           { return V3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a V3) Sub(b V3) V3           { return V3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a V3) Mul(s float64) V3      { return V3{a.X * s, a.Y * s, a.Z * s} }
func (a V3) Dot(b V3) float64      { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a V3) Cross(b V3) V3         { return V3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X} }
func (a V3) Len() float64          { return math.Sqrt(a.Dot(a)) }
func (a V3) Norm() V3              { l := a.Len(); return V3{a.X / l, a.Y / l, a.Z / l} }
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

type Camera struct {
	Eye, Target V3
	FOV         float64 // vertical, radians
	Roll        float64 // radians about the view axis (screen shake)
	W, H        float64
	right, up   V3
	fwd         V3
	f           float64
}

func (c *Camera) Setup(w, h float64) {
	c.W, c.H = w, h
	c.fwd = c.Target.Sub(c.Eye).Norm()
	c.right = V3{0, 1, 0}.Cross(c.fwd).Norm()
	c.up = c.fwd.Cross(c.right)
	if c.Roll != 0 {
		cr, sr := math.Cos(c.Roll), math.Sin(c.Roll)
		c.right, c.up = c.right.Mul(cr).Add(c.up.Mul(sr)), c.up.Mul(cr).Sub(c.right.Mul(sr))
	}
	c.f = (h / 2) / math.Tan(c.FOV/2)
}

// Project returns screen coordinates and view depth.
func (c *Camera) Project(p V3) (float64, float64, float64) {
	d := p.Sub(c.Eye)
	z := d.Dot(c.fwd)
	if z < 0.05 {
		z = 0.05
	}
	return c.W/2 + d.Dot(c.right)*c.f/z, c.H/2 - d.Dot(c.up)*c.f/z, z
}

// ScreenToFloorX maps a screen x position to world x on the plane z=zLine.
func (c *Camera) ScreenToFloorX(sx, zLine float64) float64 {
	x0, _, _ := c.Project(V3{-10, 0, zLine})
	x1, _, _ := c.Project(V3{10, 0, zLine})
	return -10 + (sx-x0)/(x1-x0)*20
}

// ---------------------------------------------------------------- atlas

const atlasSize = 1024

type Sprite struct{ X0, Y0, X1, Y1 float32 }

var (
	atlas        *ebiten.Image
	sprWhite     Sprite
	sprSphere    Sprite
	sprSoft      Sprite
	sprShadow    Sprite
	sprRing      Sprite
	sprSpark     Sprite
	sprIcons     = map[string]Sprite{}
	sprCoin      Sprite
	sprHexFloor  Sprite
	sprGrid      Sprite
	floorSprites []Sprite
	matTop       [6]Sprite
	matSide      [6]Sprite
	matFloor     [6]Sprite
)

func buildAtlas() {
	img := image.NewRGBA(image.Rect(0, 0, atlasSize, atlasSize))
	// white block
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	sprWhite = Sprite{3, 3, 5, 5}
	slot := func(i int) (int, int) { return 64 * (i % 8), 64 + 64*(i/8) }
	paint := func(i int, f func(dx, dy float64) color.RGBA) Sprite {
		ox, oy := slot(i)
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				dx, dy := (float64(x)+0.5)/32-1, (float64(y)+0.5)/32-1
				img.SetRGBA(ox+x, oy+y, f(dx, dy))
			}
		}
		return Sprite{float32(ox) + 0.5, float32(oy) + 0.5, float32(ox) + 63.5, float32(oy) + 63.5}
	}
	pm := func(v, a float64) color.RGBA { // premultiplied grey
		a = math.Max(0, math.Min(1, a))
		v = math.Max(0, math.Min(1, v))
		return color.RGBA{uint8(255 * v * a), uint8(255 * v * a), uint8(255 * v * a), uint8(255 * a)}
	}
	sprSphere = paint(0, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		if d > 1 {
			return color.RGBA{}
		}
		nz := math.Sqrt(1 - d*d)
		l := math.Max(0, -0.45*dx-0.55*dy+0.7*nz)
		spec := math.Pow(math.Max(0, l), 24)
		v := 0.35 + 0.65*l + spec*0.8
		aa := math.Min(1, (1-d)*32)
		return pm(math.Min(1, v), aa)
	})
	sprSoft = paint(1, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		a := math.Max(0, 1-d)
		return pm(1, a*a)
	})
	sprShadow = paint(2, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		a := math.Max(0, 1-d)
		a = a * a * 0.75
		return color.RGBA{0, 0, 0, uint8(255 * a)}
	})
	sprRing = paint(3, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		a := math.Max(0, 1-math.Abs(d-0.82)*9)
		return pm(1, a)
	})
	sprSpark = paint(4, func(dx, dy float64) color.RGBA {
		a := math.Max(0, 1-math.Abs(dx)*6)*math.Max(0, 1-math.Abs(dy)) +
			math.Max(0, 1-math.Abs(dy)*6)*math.Max(0, 1-math.Abs(dx))
		return pm(1, math.Min(1, a))
	})
	sprCoin = paint(5, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		if d > 0.95 {
			return color.RGBA{}
		}
		c := color.RGBA{255, 200, 60, 255}
		if d > 0.78 {
			c = color.RGBA{190, 130, 30, 255}
		}
		if math.Abs(dx+0.15) < 0.12 && math.Abs(dy) < 0.5 {
			c = color.RGBA{255, 240, 170, 255}
		}
		return c
	})
	sprHexFloor = paint(6, func(dx, dy float64) color.RGBA {
		// dark synthwave tile with bright neon edges (tinted by vertex colour)
		e := math.Max(math.Abs(dx), math.Abs(dy))
		v := 0.1
		if e > 0.94 {
			v = 1.0
		} else if e > 0.86 {
			v = 0.1 + 0.5*(e-0.86)/0.08
		}
		return pm(v, 1)
	})
	sprGrid = paint(7, func(dx, dy float64) color.RGBA {
		// additive glow of the tile edges
		e := math.Max(math.Abs(dx), math.Abs(dy))
		a := math.Max(0, (e-0.7)/0.3)
		return pm(1, a*a)
	})
	edge := func(e float64) float64 { // neon edge falloff
		if e > 0.94 {
			return 1
		} else if e > 0.86 {
			return (e - 0.86) / 0.08 * 0.5
		}
		return 0
	}
	floorDots := paint(24, func(dx, dy float64) color.RGBA {
		v := 0.08 + 0.35*edge(math.Max(math.Abs(dx), math.Abs(dy)))
		for _, c := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			if d := math.Hypot(dx-c[0], dy-c[1]); d < 0.22 {
				v = math.Max(v, 1-d/0.22*0.4)
			}
		}
		return pm(v, 1)
	})
	floorDiamond := paint(25, func(dx, dy float64) color.RGBA {
		e := math.Abs(dx) + math.Abs(dy)
		v := 0.08 + edge(1-math.Abs(e-1)*0.5)*0.9
		return pm(math.Min(1, v), 1)
	})
	floorRings := paint(26, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx, dy)
		v := 0.08 + math.Max(0, 1-math.Abs(d-0.62)*14)*0.9 + 0.5*edge(math.Max(math.Abs(dx), math.Abs(dy)))
		return pm(math.Min(1, v), 1)
	})
	floorSprites = []Sprite{sprHexFloor, floorDots, floorDiamond, floorRings}
	// ---- world materials (white-based, tinted per brick) ----
	hash := func(x, y float64) float64 {
		v := math.Sin(x*127.1+y*311.7) * 43758.5453
		return v - math.Floor(v)
	}
	noise := func(x, y float64) float64 {
		xi, yi := math.Floor(x), math.Floor(y)
		fx, fy := x-xi, y-yi
		a, b2 := hash(xi, yi), hash(xi+1, yi)
		c, d := hash(xi, yi+1), hash(xi+1, yi+1)
		ux, uy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
		return a + (b2-a)*ux + (c-a)*uy + (a-b2-c+d)*ux*uy
	}
	bev := func(dx, dy float64) float64 { // bevel lighting across a tile
		e := math.Max(math.Abs(dx), math.Abs(dy))
		if e > 0.88 {
			if dx-dy < 0 {
				return 1.15
			}
			return 0.7
		}
		return 1
	}
	// neon: holographic frame
	matTop[0] = paint(27, func(dx, dy float64) color.RGBA {
		e := math.Max(math.Abs(dx), math.Abs(dy))
		v := 0.55
		if e > 0.86 {
			v = 1.25
		} else if math.Abs(e-0.55) < 0.04 {
			v = 0.95
		}
		return pm(math.Min(1, v), 1)
	})
	matSide[0] = paint(28, func(dx, dy float64) color.RGBA {
		v := 0.6
		if math.Abs(dy) > 0.82 || math.Abs(dx) > 0.92 {
			v = 1
		}
		if math.Mod((dy+1)*6, 1) < 0.08 {
			v *= 0.85
		}
		return pm(v, 1)
	})
	// meadow: wooden crate planks
	matTop[1] = paint(29, func(dx, dy float64) color.RGBA {
		row := math.Floor((dy + 1) * 1.5)
		f := math.Mod((dy+1)*1.5, 1)
		v := 0.82 + 0.12*math.Sin((dx*9+row*3)+noise(dx*3, row)*4) + 0.06*noise(dx*20, dy*3+row)
		if f < 0.08 || f > 0.94 {
			v *= 0.6
		}
		if math.Abs(dx) > 0.9 || math.Abs(dy) > 0.9 {
			v *= 0.75
		}
		for _, cxy := range [][2]float64{{-0.78, -0.78}, {0.78, -0.78}, {0.78, 0.78}, {-0.78, 0.78}} {
			if math.Hypot(dx-cxy[0], dy-cxy[1]) < 0.07 {
				v = 0.45
			}
		}
		return pm(math.Min(1, v), 1)
	})
	matSide[1] = paint(30, func(dx, dy float64) color.RGBA {
		v := 0.8 + 0.12*math.Sin(dx*7+noise(dx*2, dy*8)*5)
		if math.Abs(dx) > 0.82 || math.Abs(dy) > 0.78 {
			v = 0.62 + 0.08*noise(dx*10, dy*10)
		}
		if math.Abs(dx-dy) < 0.1 && math.Abs(dx) < 0.82 && math.Abs(dy) < 0.78 {
			v = 0.66
		}
		return pm(v, 1)
	})
	// caverns: cut gemstone facets
	matTop[2] = paint(31, func(dx, dy float64) color.RGBA {
		var v float64
		switch {
		case dy < -math.Abs(dx):
			v = 1.0
		case dx > math.Abs(dy):
			v = 0.8
		case dy > math.Abs(dx):
			v = 0.62
		default:
			v = 0.9
		}
		if math.Abs(math.Abs(dx)-math.Abs(dy)) < 0.04 {
			v = 1
		}
		if math.Hypot(dx+0.35, dy+0.45) < 0.12 {
			v = 1
		}
		return pm(v, 1)
	})
	matSide[2] = paint(32, func(dx, dy float64) color.RGBA {
		v := 0.7 + 0.25*math.Max(0, 1-math.Abs(dx+dy*0.3+0.3)*2.5)
		return pm(v, 1)
	})
	// ruins: carved stone with an engraved glyph
	matTop[3] = paint(33, func(dx, dy float64) color.RGBA {
		v := (0.82 + 0.14*noise(dx*6, dy*6) + 0.05*noise(dx*25, dy*25)) * bev(dx, dy)
		e := math.Max(math.Abs(dx), math.Abs(dy))
		if math.Abs(e-0.72) < 0.035 {
			v *= 0.7
		}
		d := math.Hypot(dx, dy)
		if (math.Abs(d-0.35) < 0.05 && dy < 0.15) || (math.Abs(dx) < 0.05 && dy > 0 && dy < 0.5) || math.Hypot(dx, dy+0.05) < 0.08 {
			v *= 0.62
		}
		return pm(math.Min(1, v), 1)
	})
	matSide[3] = paint(34, func(dx, dy float64) color.RGBA {
		v := 0.8 + 0.15*noise(dx*5, dy*5) + 0.05*noise(dx*30, dy*30)
		if math.Mod((dx+1)*2+math.Floor((dy+1)*1.5)*0.5, 1) < 0.04 || math.Mod((dy+1)*1.5, 1) < 0.05 {
			v *= 0.65
		}
		return pm(v, 1)
	})
	// clouds: painted castle masonry / soft puffy top
	matTop[4] = paint(35, func(dx, dy float64) color.RGBA {
		d := math.Hypot(dx*0.9, dy*0.9)
		v := 0.85 + 0.15*math.Max(0, 1-d)
		if math.Abs(math.Max(math.Abs(dx), math.Abs(dy))-0.82) < 0.05 {
			v = 0.72
		}
		for _, c := range [][2]float64{{-0.3, -0.3}, {0.35, -0.2}, {0, 0.3}} {
			if math.Hypot(dx-c[0], dy-c[1]) < 0.16 {
				v = 1
			}
		}
		return pm(math.Min(1, v), 1)
	})
	matSide[4] = paint(36, func(dx, dy float64) color.RGBA {
		row := math.Floor((dy + 1) * 2)
		off := math.Mod(row, 2) * 0.5
		v := 0.92
		if math.Mod((dy+1)*2, 1) < 0.08 || math.Mod((dx+1)*1.5+off, 1) < 0.05 {
			v = 0.7
		}
		return pm(v, 1)
	})
	// machine: riveted metal plate with hazard stripe
	matTop[5] = paint(37, func(dx, dy float64) color.RGBA {
		v := (0.78 + 0.08*math.Sin(dx*40+noise(dx*5, dy*40)*3)) * bev(dx, dy)
		for _, c := range [][2]float64{{-0.72, -0.72}, {0.72, -0.72}, {0.72, 0.72}, {-0.72, 0.72}} {
			d := math.Hypot(dx-c[0], dy-c[1])
			if d < 0.1 {
				v = 1 - d*3
			}
		}
		if math.Abs(dy) < 0.16 && math.Abs(dx) < 0.55 {
			if math.Mod((dx+dy)*5+10, 1) < 0.5 {
				v = 1
			} else {
				v = 0.35
			}
		}
		return pm(math.Min(1, v), 1)
	})
	matSide[5] = paint(38, func(dx, dy float64) color.RGBA {
		v := 0.72 + 0.06*math.Sin(dy*30)
		if math.Abs(dy) > 0.8 {
			v = 0.95
		}
		if math.Abs(dx) > 0.86 {
			v = 0.55
		}
		return pm(v, 1)
	})
	// floors
	matFloor[0] = sprHexFloor
	matFloor[1] = paint(39, func(dx, dy float64) color.RGBA { // terracotta tiles
		fx, fy := math.Mod(dx+1, 1), math.Mod(dy+1, 1)
		v := 0.86 + 0.1*noise(dx*4, dy*4)
		if fx < 0.05 || fy < 0.05 {
			v = 0.6
		}
		return pm(v, 1)
	})
	matFloor[2] = paint(40, func(dx, dy float64) color.RGBA { // cave slabs
		v := 0.7 + 0.25*noise(dx*3, dy*3) + 0.08*noise(dx*15, dy*15)
		if math.Abs(noise(dx*2+3, dy*2)-0.5) < 0.02 {
			v *= 0.5
		}
		if math.Max(math.Abs(dx), math.Abs(dy)) > 0.95 {
			v *= 0.6
		}
		return pm(v, 1)
	})
	matFloor[3] = paint(41, func(dx, dy float64) color.RGBA { // sandstone
		v := (0.84 + 0.12*noise(dx*3, dy*3) + 0.05*noise(dx*20, dy*20)) * bev(dx, dy)
		if math.Abs(math.Max(math.Abs(dx), math.Abs(dy))-0.6) < 0.025 {
			v *= 0.78
		}
		return pm(math.Min(1, v), 1)
	})
	matFloor[4] = paint(42, func(dx, dy float64) color.RGBA { // marble
		vein := math.Abs(math.Sin(dx*3 + dy*2 + noise(dx*3, dy*3)*5))
		v := 0.92 + 0.06*vein
		if vein < 0.06 {
			v = 0.78
		}
		if math.Max(math.Abs(dx), math.Abs(dy)) > 0.94 {
			v = 1
		}
		return pm(v, 1)
	})
	matFloor[5] = paint(43, func(dx, dy float64) color.RGBA { // grating
		fx, fy := math.Mod((dx+1)*3, 1), math.Mod((dy+1)*3, 1)
		v := 0.85
		if fx > 0.2 && fx < 0.8 && fy > 0.2 && fy < 0.8 {
			v = 0.25
		}
		if math.Max(math.Abs(dx), math.Abs(dy)) > 0.92 {
			v = 1
		}
		return pm(v, 1)
	})
	// icons (white glyphs, tinted per use)
	icon := func(i int, name string, f func(dx, dy float64) float64) {
		sprIcons[name] = paint(8+i, func(dx, dy float64) color.RGBA { return pm(1, f(dx, dy)) })
	}
	soft := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	icon(0, "bomb", func(dx, dy float64) float64 {
		d := math.Hypot(dx, dy+0.1)
		return soft((0.55-d)*20) + soft((0.08-math.Abs(dx-dy-0.75))*20)*soft((dx-0.3)*20)*soft((0.75-dx)*20)
	})
	icon(1, "bolt", func(dx, dy float64) float64 {
		// zig-zag lightning
		y := dy
		xc := 0.0
		if y < 0 {
			xc = 0.25 + y*0.5
		} else {
			xc = -0.25 + y*0.5
		}
		return soft((0.16-math.Abs(dx-xc))*14) * soft((0.85-math.Abs(y))*14)
	})
	icon(2, "split", func(dx, dy float64) float64 {
		return soft((0.12-math.Abs(dx))*16)*soft((0.8-math.Abs(dy))*12) + soft((0.12-math.Abs(dy))*16)*soft((0.8-math.Abs(dx))*12)
	})
	icon(3, "plus", func(dx, dy float64) float64 {
		d := math.Hypot(dx, dy)
		return soft((0.1-math.Abs(d-0.55))*14) + soft((0.1-math.Abs(dx))*14)*soft((0.35-math.Abs(dy))*14) + soft((0.1-math.Abs(dy))*14)*soft((0.35-math.Abs(dx))*14)
	})
	icon(4, "arrows", func(dx, dy float64) float64 {
		ax := math.Abs(dx)
		return soft((0.1-math.Abs(dy))*14)*soft((0.8-ax)*14) + soft((0.12-math.Abs(math.Abs(dy)-(0.8-ax)))*10)*soft((ax-0.45)*14)*soft((0.85-ax)*14)
	})
	icon(5, "shield", func(dx, dy float64) float64 {
		w := 0.65 - math.Max(0, dy)*0.65
		in := soft((w-math.Abs(dx))*14) * soft((0.75-math.Abs(dy+0.05))*14)
		inner := soft((w-0.2-math.Abs(dx))*14) * soft((0.55-math.Abs(dy+0.05))*14)
		return in - inner*0.6
	})
	icon(6, "warp", func(dx, dy float64) float64 {
		d := math.Hypot(dx, dy)
		a := math.Atan2(dy, dx)
		return soft((0.12-math.Abs(math.Mod(d*3+a/math.Pi+4, 1)-0.5)*0.6)*10) * soft((0.85-d)*10)
	})
	icon(7, "seed", func(dx, dy float64) float64 {
		d := math.Hypot(dx, dy)
		return soft((0.3-d)*14) + soft((0.08-math.Abs(d-0.65))*14)
	})
	icon(8, "x2", func(dx, dy float64) float64 {
		a := soft((0.11-math.Abs(dx-dy))*12)*soft((0.7-math.Abs(dx))*12) + soft((0.11-math.Abs(dx+dy))*12)*soft((0.7-math.Abs(dx))*12)
		return a
	})
	icon(9, "gem", func(dx, dy float64) float64 {
		return soft((0.8 - math.Abs(dx) - math.Abs(dy)*0.8) * 8)
	})
	icon(10, "q", func(dx, dy float64) float64 {
		d := math.Hypot(dx, dy+0.25)
		arc := soft((0.1-math.Abs(d-0.38))*14) * soft((dy+0.25+0.15+dx*0.6)*8)
		stem := soft((0.1-math.Abs(dx))*14) * soft((0.12-math.Abs(dy-0.25))*8)
		dot := soft((0.13 - math.Hypot(dx, dy-0.65)) * 14)
		return math.Min(1, arc+stem+dot)
	})
	icon(11, "crown", func(dx, dy float64) float64 {
		base := soft((0.7-math.Abs(dx))*14) * soft((0.12-math.Abs(dy-0.45))*14)
		teeth := soft((0.7-math.Abs(dx))*14) * soft((dy-(-0.5+math.Abs(math.Mod(dx+1.05, 0.7)-0.35)*2.2))*10) * soft((0.45-dy)*14)
		return math.Min(1, base+teeth)
	})
	icon(12, "rock", func(dx, dy float64) float64 {
		return soft((0.7 - math.Max(math.Abs(dx+dy*0.3), math.Abs(dy-dx*0.2))) * 10)
	})
	atlas = ebiten.NewImageFromImage(img)
}

// ---------------------------------------------------------------- batch

type quad struct {
	v     [4]ebiten.Vertex
	depth float64
	tex   uint8 // 0 = the sprite atlas, else texImages[tex]
}

// texImages holds the stand-alone textures quads can sample (index 0 is
// the atlas, filled in lazily).
var texImages = []*ebiten.Image{nil}

// registerTex adds a texture for QuadUV and returns its index.
func registerTex(img *ebiten.Image) uint8 {
	texImages = append(texImages, img)
	return uint8(len(texImages) - 1)
}

type Batch struct {
	cam      *Camera
	quads    []quad
	glow     []quad
	verts    []ebiten.Vertex
	order    []int32
	keys     [2][]uint64 // radix sort scratch
	idx2     []int32
	proj     []projected // mesh corner scratch (drawShared)
	sides    []*Batch    // side batches built on other goroutines (side)
	runs     []quadRun   // flush: draw-call runs (planRuns)
	qp       []*quad     // flush: every quad to draw, in submission order
	extQ     [][]quad    // side batches' quads drawn before b's own (drawFirst)
	extG     [][]quad
	extB     []*Batch     // the side batches behind extQ/extG, for their presorted orders
	pre      [2]presorted // a side batch's own far-first orders (presort)
	wantMeta [2]bool      // presort: work out run metadata (the flush is likely to plan)
	mergeK   []uint64     // flush scratch: the own quads' keys when merging
	gp       *Batch       // the glow list's planning scratch (glowPlanner)
	mk       [2][]uint64  // mergeSorted scratch: keys and indices, ping-pong
	mi       [2][]int32
	runIdx   []int32
	cut      cutWindow // drawing one piece of a job (beginPiece)
	skipped  bool      // the last primitive fell outside the piece's window
	scratch  quad      // lastQuad/lastGlow after a skipped primitive
	qtex     []uint8   // planList: each quad's texture, by index (beside qp)
	runCell  [runCols * runRows]int16
	meta     []uint64 // per quad (submission order): cells and run key
	switches [2]int   // texture switches in last frame's opaque and glow lists
	planned  [2]bool
	cellW    float32
	cellH    float32
	runBigB  [runBX * runBY]int16
	runFineB [runBX * runBY]int16
	joinQ    []quad // scratch for prepend
	fogColor [3]float32
	fogNear  float64
	fogFar   float64
	light    V3
	lightCol [3]float32
	ambCol   [3]float32
	pts      []ptLight
}

// ptLight is a coloured point light (an orb, a blast, a glowing brick)
// that brightens the brick faces near it.
type ptLight struct {
	p    V3
	c    [3]float32
	r, s float64
}

const maxPtLights = 48

// AddLight adds a point light for this frame: radius r, strength s.
func (b *Batch) AddLight(p V3, c color.RGBA, r, s float64) {
	if len(b.pts) >= maxPtLights || s <= 0.01 {
		return
	}
	b.pts = append(b.pts, ptLight{p, [3]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255}, r, s})
}

// ptGlow sums the point lights reaching p on a face facing n.
func (b *Batch) ptGlow(p, n V3) (g [3]float32) {
	for i := range b.pts {
		b.addGlow(&g, &b.pts[i], p, n)
	}
	return
}

// addGlow adds one point light's contribution at p (facing n) to g.
func (b *Batch) addGlow(g *[3]float32, l *ptLight, p, n V3) {
	d := l.p.Sub(p)
	d2 := d.Dot(d)
	if d2 >= l.r*l.r {
		return
	}
	dist := math.Sqrt(d2)
	k := 1 - dist/l.r
	k = k * k * l.s
	ndl := 0.3
	if dist > 1e-4 {
		ndl += 0.7 * math.Max(0, n.Dot(d)/dist)
	}
	w := float32(k * ndl)
	g[0] += l.c[0] * w
	g[1] += l.c[1] * w
	g[2] += l.c[2] * w
}

// litPt applies the point lights to a face colour: they brighten what is
// there and add a little of their own colour so dark bricks catch it too.
func (b *Batch) litPt(col color.RGBA, p, n V3) color.RGBA {
	if len(b.pts) == 0 {
		return col
	}
	return b.applyGlow(col, b.ptGlow(p, n))
}

// litPtSome is litPt over a subset of the lights (indices into b.pts, in
// order), for callers that have already ruled the rest out.
func (b *Batch) litPtSome(col color.RGBA, p, n V3, some []int8) color.RGBA {
	if len(b.pts) == 0 {
		return col
	}
	var g [3]float32
	for _, i := range some {
		b.addGlow(&g, &b.pts[i], p, n)
	}
	return b.applyGlow(col, g)
}

func (b *Batch) applyGlow(col color.RGBA, g [3]float32) color.RGBA {
	f := func(v uint8, k float32) uint8 {
		if k > 1.6 {
			k = 1.6
		}
		x := float32(v)*(1+k) + 70*k
		if x > 255 {
			x = 255
		}
		return uint8(x)
	}
	return color.RGBA{f(col.R, g[0]), f(col.G, g[1]), f(col.B, g[2]), col.A}
}

func newBatch(cam *Camera) *Batch {
	return &Batch{cam: cam, light: V3{-0.35, 0.85, -0.4}.Norm(), fogNear: 30, fogFar: 75,
		lightCol: [3]float32{0.75, 0.7, 0.85}, ambCol: [3]float32{0.45, 0.4, 0.6}}
}

// side returns side batch i, emptied and set up like b (camera, fog, key
// light, point lights), for drawing on another goroutine. Only b's owner
// may call it, and only between b's Reset and Flush.
func (b *Batch) side(i int) *Batch {
	for len(b.sides) <= i {
		b.sides = append(b.sides, &Batch{})
	}
	s := b.sides[i]
	s.cam = b.cam
	s.fogColor, s.fogNear, s.fogFar = b.fogColor, b.fogNear, b.fogFar
	s.light, s.lightCol, s.ambCol = b.light, b.lightCol, b.ambCol
	s.pts = append(s.pts[:0], b.pts...)
	s.quads = s.quads[:0]
	s.glow = s.glow[:0]
	for l := range s.wantMeta {
		// metadata only matters if the flush plans runs, which follows last
		// frame's texture switches (see flush)
		s.wantMeta[l] = planRunsOn && b.switches[l] > runPlanOff
	}
	return s
}

// A piece of a job: one job (the scenery, say) run on several goroutines,
// each drawing a share of it. Every piece runs the job's code in full
// (layout, random numbers, everything that decides what is drawn) but only
// emits the primitives whose running cost falls in its window, so the
// pieces' lists, laid end to end, are exactly what the job alone draws.
// The windows split last frame's total cost evenly.
type cutWindow struct {
	on     bool
	lo, hi float64 // emit primitives starting in [lo, hi)
	at     float64 // the running cost so far
}

// keep advances the running cost by c and reports whether the primitive
// starting there is this piece's to emit. A skipped primitive leaves the
// lists untouched, so code that edits the quad it just added checks
// skipped first.
func (b *Batch) keep(c float64) bool {
	at := b.cut.at
	b.cut.at += c
	b.skipped = at < b.cut.lo || at >= b.cut.hi
	return !b.skipped
}

// beginPiece sets b up to draw piece k of n of a job whose whole cost was
// total last frame (0: unknown, so piece 0 draws it all).
func (b *Batch) beginPiece(k, n int, total float64) {
	b.skipped = false
	if n <= 1 {
		b.cut = cutWindow{}
		return
	}
	lo, hi := math.Inf(-1), math.Inf(1)
	switch {
	case total <= 0:
		if k > 0 {
			lo = hi // nothing
		}
	default:
		if k > 0 {
			lo = total * float64(k) / float64(n)
		}
		if k < n-1 {
			hi = total * float64(k+1) / float64(n)
		}
	}
	b.cut = cutWindow{on: true, lo: lo, hi: hi}
}

// endPiece turns the window off and returns the job's whole running cost.
func (b *Batch) endPiece() float64 {
	t := b.cut.at
	b.cut = cutWindow{}
	b.skipped = false
	return t
}

// add appends a finished quad to the opaque or glow list, unless it lies
// wholly off the screen. The test runs on the very corners that will be
// drawn (after the camera's near clamp), so a dropped quad would not have
// covered a single pixel. A dropped quad counts as skipped (see lastQuad).
func (b *Batch) add(q *quad, glow bool) {
	if offScreen(&q.v, float32(b.cam.W), float32(b.cam.H)) {
		b.skipped = true
		return
	}
	b.skipped = false
	if glow {
		b.glow = append(b.glow, *q)
	} else {
		b.quads = append(b.quads, *q)
	}
}

// offScreen reports whether all four corners lie beyond one edge of a w x h
// screen.
func offScreen(v *[4]ebiten.Vertex, w, h float32) bool {
	return (v[0].DstX < 0 && v[1].DstX < 0 && v[2].DstX < 0 && v[3].DstX < 0) ||
		(v[0].DstX > w && v[1].DstX > w && v[2].DstX > w && v[3].DstX > w) ||
		(v[0].DstY < 0 && v[1].DstY < 0 && v[2].DstY < 0 && v[3].DstY < 0) ||
		(v[0].DstY > h && v[1].DstY > h && v[2].DstY > h && v[3].DstY > h)
}

// lastQuad is the opaque quad just added, or a scratch one when it was
// skipped: outside this piece's window (keep) or off the screen (add).
func (b *Batch) lastQuad() *quad {
	if b.skipped || len(b.quads) == 0 {
		return &b.scratch
	}
	return &b.quads[len(b.quads)-1]
}

// lastGlow is lastQuad for the glow list.
func (b *Batch) lastGlow() *quad {
	if b.skipped || len(b.glow) == 0 {
		return &b.scratch
	}
	return &b.glow[len(b.glow)-1]
}

// insertSides puts the side batches' quads into b's lists at positions
// nq (opaque) and ng (glow), in argument order: exactly where they would
// have landed had they been drawn into b at that point.
func (b *Batch) insertSides(nq, ng int, sides ...*Batch) {
	join := func(own []quad, at int, get func(*Batch) []quad) []quad {
		q := append(b.joinQ[:0], own[:at]...)
		for _, s := range sides {
			q = append(q, get(s)...)
		}
		q = append(q, own[at:]...)
		b.joinQ = own[:0] // the old buffer becomes next frame's scratch
		return q
	}
	b.quads = join(b.quads, nq, func(s *Batch) []quad { return s.quads })
	b.glow = join(b.glow, ng, func(s *Batch) []quad { return s.glow })
}

func (b *Batch) Reset() {
	b.extQ, b.extG, b.extB = b.extQ[:0], b.extG[:0], b.extB[:0]
	b.quads = b.quads[:0]
	b.glow = b.glow[:0]
	b.pts = b.pts[:0]
}

func cf(c color.RGBA) [4]float32 {
	return [4]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
}

func (b *Batch) vtx(p V3, u, v float32, col [4]float32, shade float32) (ebiten.Vertex, float64) {
	sx, sy, d := b.cam.Project(p)
	fog := float32(math.Max(0, math.Min(1, (d-b.fogNear)/(b.fogFar-b.fogNear))))
	r := col[0]*shade*(1-fog) + b.fogColor[0]*fog
	g := col[1]*shade*(1-fog) + b.fogColor[1]*fog
	bb := col[2]*shade*(1-fog) + b.fogColor[2]*fog
	a := col[3]
	return ebiten.Vertex{DstX: float32(sx), DstY: float32(sy), SrcX: u, SrcY: v,
		ColorR: r * a, ColorG: g * a, ColorB: bb * a, ColorA: a}, d
}

// Quad adds a 4-point polygon (p0..p3 in order) with uniform colour and
// per-corner shade multipliers.
func (b *Batch) Quad(p [4]V3, spr Sprite, col color.RGBA, shades [4]float32, glow bool) {
	if b.cut.on && !b.keep(1) {
		return
	}
	var q quad
	uv := [4][2]float32{{spr.X0, spr.Y0}, {spr.X1, spr.Y0}, {spr.X1, spr.Y1}, {spr.X0, spr.Y1}}
	c := cf(col)
	sum := 0.0
	for i := 0; i < 4; i++ {
		var d float64
		q.v[i], d = b.vtx(p[i], uv[i][0], uv[i][1], c, shades[i])
		sum += d
	}
	q.depth = sum / 4
	b.add(&q, glow)
}

// QuadUV is Quad with explicit texture coordinates (in pixels) into a
// registered texture, sampled with wrap-around.
func (b *Batch) QuadUV(p [4]V3, tex uint8, uv [4][2]float32, col color.RGBA, shades [4]float32, glow bool) {
	if b.cut.on && !b.keep(1) {
		return
	}
	q := quad{tex: tex}
	c := cf(col)
	sum := 0.0
	for i := 0; i < 4; i++ {
		var d float64
		q.v[i], d = b.vtx(p[i], uv[i][0], uv[i][1], c, shades[i])
		sum += d
	}
	q.depth = sum / 4
	b.add(&q, glow)
}

// SetLight configures the world's key light direction and colours.
func (b *Batch) SetLight(dir V3, light, amb [3]float32) {
	b.light = dir.Norm()
	b.lightCol, b.ambCol = light, amb
}

// lit tints a face colour by the key light (warm/cool) and ambient fill.
func (b *Batch) lit(col color.RGBA, n V3) color.RGBA {
	d := float32(math.Max(0, n.Dot(b.light)))
	f := func(v uint8, i int) uint8 {
		t := b.ambCol[i] + b.lightCol[i]*d*0.85
		if t > 1.25 {
			t = 1.25
		}
		x := float32(v) * t
		if x > 255 {
			x = 255
		}
		return uint8(x)
	}
	return color.RGBA{f(col.R, 0), f(col.G, 1), f(col.B, 2), col.A}
}

func (b *Batch) shade(n V3) float32 {
	return float32(0.42 + 0.58*math.Max(0, n.Dot(b.light)))
}

// Box draws an axis-aligned box from (x0,y0,z0) to (x1,y1,z1).
func (b *Batch) Box(x0, y0, z0, x1, y1, z1 float64, col color.RGBA, topSpr Sprite, emissive float32) {
	b.BoxTex(x0, y0, z0, x1, y1, z1, col, topSpr, sprWhite, emissive)
}

// BoxTex is Box with a texture on the side faces too.
func (b *Batch) BoxTex(x0, y0, z0, x1, y1, z1 float64, col color.RGBA, topSpr, sideSpr Sprite, emissive float32) {
	eye := b.cam.Eye
	type face struct {
		n   V3
		pts [4]V3
		ao  [4]float32
	}
	faces := [5]face{
		{V3{0, 1, 0}, [4]V3{{x0, y1, z1}, {x1, y1, z1}, {x1, y1, z0}, {x0, y1, z0}}, [4]float32{1.12, 1.12, 1, 1}},
		{V3{0, 0, -1}, [4]V3{{x0, y1, z0}, {x1, y1, z0}, {x1, y0, z0}, {x0, y0, z0}}, [4]float32{1, 1, 0.6, 0.6}},
		{V3{0, 0, 1}, [4]V3{{x1, y1, z1}, {x0, y1, z1}, {x0, y0, z1}, {x1, y0, z1}}, [4]float32{1, 1, 0.6, 0.6}},
		{V3{-1, 0, 0}, [4]V3{{x0, y1, z1}, {x0, y1, z0}, {x0, y0, z0}, {x0, y0, z1}}, [4]float32{1, 1, 0.6, 0.6}},
		{V3{1, 0, 0}, [4]V3{{x1, y1, z0}, {x1, y1, z1}, {x1, y0, z1}, {x1, y0, z0}}, [4]float32{1, 1, 0.6, 0.6}},
	}
	cx, cy, cz := (x0+x1)/2, (y0+y1)/2, (z0+z1)/2
	for i, f := range faces {
		fc := V3{cx + f.n.X*(x1-x0)/2, cy + f.n.Y*(y1-y0)/2, cz + f.n.Z*(z1-z0)/2}
		if f.n.Dot(eye.Sub(fc)) <= 0 {
			continue
		}
		var sh [4]float32
		for k := 0; k < 4; k++ {
			sh[k] = f.ao[k] + emissive
		}
		spr := sideSpr
		if i == 0 {
			spr = topSpr
		}
		b.Quad(f.pts, spr, b.lit(col, f.n), sh, false)
	}
}

// Prism draws a vertical prism over a convex quad footprint (x,z pairs),
// optionally tapering its top by topScale (1 = straight, ~0 = pyramid).
func (b *Batch) Prism(fp [4][2]float64, y0, y1, topScale float64, col color.RGBA, emissive float32) {
	cx := (fp[0][0] + fp[1][0] + fp[2][0] + fp[3][0]) / 4
	cz := (fp[0][1] + fp[1][1] + fp[2][1] + fp[3][1]) / 4
	var bot, top [4]V3
	for i, p := range fp {
		bot[i] = V3{p[0], y0, p[1]}
		top[i] = V3{cx + (p[0]-cx)*topScale, y1, cz + (p[1]-cz)*topScale}
	}
	eye := b.cam.Eye
	face := func(pts [4]V3, ao [4]float32) {
		e1 := pts[1].Sub(pts[0])
		e2 := pts[3].Sub(pts[0])
		n := e1.Cross(e2)
		if n.Len() < 1e-9 {
			n = pts[2].Sub(pts[1]).Cross(pts[0].Sub(pts[1]))
		}
		n = n.Norm()
		fc := pts[0].Add(pts[1]).Add(pts[2]).Add(pts[3]).Mul(0.25)
		mid := V3{cx, (y0 + y1) / 2, cz}
		if n.Dot(fc.Sub(mid)) < 0 {
			n = n.Mul(-1)
		}
		if n.Dot(eye.Sub(fc)) <= 0 {
			return
		}
		b.Quad(pts, sprWhite, b.lit(col, n), [4]float32{ao[0] + emissive, ao[1] + emissive, ao[2] + emissive, ao[3] + emissive}, false)
	}
	if topScale > 0.05 {
		face([4]V3{top[0], top[1], top[2], top[3]}, [4]float32{1.1, 1.1, 1.1, 1.1})
	}
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		face([4]V3{top[i], top[j], bot[j], bot[i]}, [4]float32{1, 1, 0.6, 0.6})
	}
}

// OBox draws a box with arbitrary orientation: centre c and half-axis vectors.
func (b *Batch) OBox(c, ax, ay, az V3, col color.RGBA, emissive float32) {
	eye := b.cam.Eye
	corner := func(sx, sy, sz float64) V3 { return c.Add(ax.Mul(sx)).Add(ay.Mul(sy)).Add(az.Mul(sz)) }
	type fdef struct {
		n   V3
		pts [4]V3
	}
	faces := []fdef{
		{ay, [4]V3{corner(-1, 1, 1), corner(1, 1, 1), corner(1, 1, -1), corner(-1, 1, -1)}},
		{ay.Mul(-1), [4]V3{corner(-1, -1, -1), corner(1, -1, -1), corner(1, -1, 1), corner(-1, -1, 1)}},
		{ax, [4]V3{corner(1, 1, -1), corner(1, 1, 1), corner(1, -1, 1), corner(1, -1, -1)}},
		{ax.Mul(-1), [4]V3{corner(-1, 1, 1), corner(-1, 1, -1), corner(-1, -1, -1), corner(-1, -1, 1)}},
		{az, [4]V3{corner(1, 1, 1), corner(-1, 1, 1), corner(-1, -1, 1), corner(1, -1, 1)}},
		{az.Mul(-1), [4]V3{corner(-1, 1, -1), corner(1, 1, -1), corner(1, -1, -1), corner(-1, -1, -1)}},
	}
	for _, f := range faces {
		n := f.n.Norm()
		fc := f.pts[0].Add(f.pts[2]).Mul(0.5)
		if n.Dot(eye.Sub(fc)) <= 0 {
			continue
		}
		e := 1 + emissive
		b.Quad(f.pts, sprWhite, b.lit(col, n), [4]float32{e, e, e, e}, false)
	}
}

// Cylinder draws an n-sided vertical prism (towers, boilers, trunks),
// tapering to topScale.
func (b *Batch) Cylinder(x, z, r, y0, y1 float64, sides int, topScale float64, col color.RGBA, emissive float32) {
	eye := b.cam.Eye
	for i := 0; i < sides; i++ {
		a0 := float64(i) / float64(sides) * 2 * math.Pi
		a1 := float64(i+1) / float64(sides) * 2 * math.Pi
		p0 := V3{x + math.Cos(a0)*r, y0, z + math.Sin(a0)*r}
		p1 := V3{x + math.Cos(a1)*r, y0, z + math.Sin(a1)*r}
		t0 := V3{x + math.Cos(a0)*r*topScale, y1, z + math.Sin(a0)*r*topScale}
		t1 := V3{x + math.Cos(a1)*r*topScale, y1, z + math.Sin(a1)*r*topScale}
		am := (a0 + a1) / 2
		n := V3{math.Cos(am), r * (1 - topScale) / math.Max(0.01, y1-y0), math.Sin(am)}.Norm()
		fc := p0.Add(t1).Mul(0.5)
		if n.Dot(eye.Sub(fc)) <= 0 {
			continue
		}
		e := float32(emissive)
		b.Quad([4]V3{t0, t1, p1, p0}, sprWhite, b.lit(col, n), [4]float32{1 + e, 1 + e, 0.65 + e, 0.65 + e}, false)
	}
	if topScale > 0.05 && eye.Y > y1 {
		// cap as a fan of quads
		lc := b.lit(col, V3{0, 1, 0})
		for i := 0; i < sides; i += 2 {
			pts := [4]V3{}
			for k := 0; k < 3; k++ {
				a := float64(i+k) / float64(sides) * 2 * math.Pi
				pts[k] = V3{x + math.Cos(a)*r*topScale, y1, z + math.Sin(a)*r*topScale}
			}
			pts[3] = V3{x, y1, z}
			b.Quad(pts, sprWhite, lc, [4]float32{1.05 + float32(emissive), 1.05, 1.05, 1.05}, false)
		}
	}
}

// rotQuad returns a w×d rectangle footprint centred at (x,z) rotated by a.
func rotQuad(x, z, w, d, a float64) [4][2]float64 {
	ca, sa := math.Cos(a), math.Sin(a)
	var out [4][2]float64
	for i, p := range [][2]float64{{-w / 2, -d / 2}, {w / 2, -d / 2}, {w / 2, d / 2}, {-w / 2, d / 2}} {
		out[i] = [2]float64{x + p[0]*ca - p[1]*sa, z + p[0]*sa + p[1]*ca}
	}
	return out
}

// Billboard draws a camera-facing sprite centred at p with world size s.
func (b *Batch) Billboard(p V3, s float64, spr Sprite, col color.RGBA, glow bool) {
	r := b.cam.right.Mul(s / 2)
	u := b.cam.up.Mul(s / 2)
	pts := [4]V3{p.Sub(r).Add(u), p.Add(r).Add(u), p.Add(r).Sub(u), p.Sub(r).Sub(u)}
	b.Quad(pts, spr, col, [4]float32{1, 1, 1, 1}, glow)
}

// Decal draws a flat sprite lying on the plane y=h.
func (b *Batch) Decal(x, z, h, w, d float64, spr Sprite, col color.RGBA, glow bool) {
	pts := [4]V3{{x - w/2, h, z + d/2}, {x + w/2, h, z + d/2}, {x + w/2, h, z - d/2}, {x - w/2, h, z - d/2}}
	b.Quad(pts, spr, col, [4]float32{1, 1, 1, 1}, glow)
}

// DecalBelow is a decal forced to sort behind everything else at its depth
// (for shadows and floor marks).
func (b *Batch) DecalBelow(x, z, h, w, d float64, spr Sprite, col color.RGBA) {
	b.Decal(x, z, h, w, d, spr, col, false)
	b.lastQuad().depth += 1000
}

// planList orders a list's quads far first and groups them into draw-call
// runs, in b's scratch. It makes no Ebitengine calls; prepMeta must have run.
func (b *Batch) planList(segs [][]quad, list int) ([]*quad, []quadRun) {
	// the quads of every segment, in order, by pointer (no copying)
	qs := b.qp[:0]
	for _, s := range segs {
		for i := range s {
			qs = append(qs, &s[i])
		}
	}
	b.qp = qs
	if cap(b.qtex) < len(qs) {
		b.qtex = make([]uint8, len(qs), len(qs)+len(qs)/4)
	}
	b.qtex = b.qtex[:len(qs)]
	if len(qs) == 0 {
		return nil, nil
	}
	// plan runs only where texture switches are many enough to repay it
	// (judged by the last frame, with a gap so the choice doesn't flicker)
	sw := b.switches[list]
	plan := planRunsOn && (sw > runPlanOn || (b.planned[list] && sw > runPlanOff))
	b.planned[list] = plan
	// sort indices rather than the (large) quads themselves: far first
	var order []int32
	tA := time.Now()
	if presortOn && len(b.extB) == len(segs)-1 && len(b.extB) > 0 && len(b.extB) < maxMerge {
		order = b.mergeSorted(segs, list, plan)
	} else {
		order = b.sortFarFirst(qs, plan)
	}
	tB := time.Now()
	runs, switches := b.planRuns(qs, order, plan)
	planT[0] += tB.Sub(tA)
	planT[1] += time.Since(tB)
	b.switches[list] = switches
	return qs, runs
}

// drawRuns submits planned runs: one DrawTriangles per run, at most maxQ
// quads each.
func (b *Batch) drawRuns(dst *ebiten.Image, qs []*quad, runs []quadRun, blend ebiten.Blend, scale float32) {
	if flushTrace != nil {
		for i := 1; i < len(runs); i++ {
			flushTrace(runs[i-1].tex, runs[i].tex, float64(runs[i].last-runs[i].first))
		}
	}
	b.fillVerts(qs, scale)
	for ri := range runs {
		rn := &runs[ri]
		op := &ebiten.DrawTrianglesOptions{Blend: blend, AntiAlias: worldAA && !aaOwn}
		src := atlas
		if rn.tex != 0 {
			src = texImages[rn.tex]
			op.Address = ebiten.AddressRepeat
			op.Filter = ebiten.FilterLinear
			op.DisableMipmaps = rn.noMip
			op.UniformMipmapLevel = rn.oneMip
		}
		vs := b.verts[rn.first*4 : rn.last*4]
		for len(vs) > 0 {
			n := min(len(vs)/4, maxQ)
			dst.DrawTriangles32(vs[:n*4], quadIndices32[:n*6], src, op)
			perf.cur.world++
			vs = vs[n*4:]
		}
	}
}

// fillVerts gathers the planned quads' vertices, in run order, into
// b.verts (scaled to the target). The quads are scattered through memory
// and this copy is cache-miss bound, so a big list is split across
// goroutines; Ebitengine copies the vertices it is given, so the buffer is
// free again as soon as each draw call returns.
func (b *Batch) fillVerts(qs []*quad, scale float32) {
	ids := b.runIdx
	n := len(ids) * 4
	if cap(b.verts) < n {
		b.verts = make([]ebiten.Vertex, n, n+n/4)
	}
	b.verts = b.verts[:n]
	fill := func(lo, hi int) {
		vs := b.verts[lo*4 : hi*4]
		for i, oi := range ids[lo:hi] {
			copy(vs[i*4:i*4+4], qs[oi].v[:])
		}
		if scale != 1 {
			for k := range vs {
				vs[k].DstX *= scale
				vs[k].DstY *= scale
			}
		}
	}
	parts := min(len(ids)/fillChunk, runtime.GOMAXPROCS(0), 8)
	if parts < 2 {
		fill(0, len(ids))
		return
	}
	var wg sync.WaitGroup
	step := (len(ids) + parts - 1) / parts
	for lo := step; lo < len(ids); lo += step {
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			fill(lo, hi)
		}(lo, min(lo+step, len(ids)))
	}
	fill(0, min(step, len(ids)))
	wg.Wait()
}

// flushBenchHook, when set (capture builds), runs after a flush with the
// lists it drew, so the planner can be timed on real frames.
var flushBenchHook func(b *Batch, segQ, segG [][]quad)

// fillChunk is the fewest quads worth handing to another goroutine.
const fillChunk = 3000

// Runs: painter's order only matters where quads overlap on screen. A quad
// may therefore join the most recent run of its own texture (and mip
// level), ahead of the runs that came after it, as long as it overlaps none
// of their quads; disjoint quads blend the same in either order, so the
// picture is unchanged while texture switches (each its own draw call)
// collapse. Overlap is tested conservatively on a coarse screen grid that
// remembers, per cell, the latest run that touched it, with a pixel of
// margin for anti-aliased edges.

const (
	runCols  = 128
	runRows  = 72
	runBlock = 8 // fine cells per block side; blocks hold the big quads
	runBX    = runCols / runBlock
	runBY    = runRows / runBlock
	runBig   = 64 // a quad covering more fine cells than this is big
)

type quadRun struct {
	tex         uint8
	noMip       bool
	oneMip      bool // every quad needs the same mip level (planned runs are keyed by it)
	first, last int  // the run's quad indices are runIdx[first:last] once planned
	ids         []int32
}

// noMip marks a run drawn with mipmaps off: its quads tile their texture,
// and Ebitengine's mip levels don't wrap correctly (the wrap region stays at
// the full-size image), so these always sample the full-size texture.
const noMip = 7

// texSize caches each registered texture's size in pixels.
var texSize [256][2]float32

// quadWraps reports whether a quad samples outside its texture (and so
// relies on wrap-around addressing).
func quadWraps(q *quad, sz [2]float32) bool {
	for k := range q.v {
		if q.v[k].SrcX < 0 || q.v[k].SrcY < 0 || q.v[k].SrcX > sz[0] || q.v[k].SrcY > sz[1] {
			return true
		}
	}
	return false
}

func quadMipLevel(q *quad) int {
	v := &q.v
	k := float32(1) // the scale the quad is drawn at (see aaBegin)
	if worldAA {
		k = aaScale
	}
	edge := func(a, b int) int {
		return mipLevelFromDistance(k*v[a].DstX, k*v[a].DstY, k*v[b].DstX, k*v[b].DstY, v[a].SrcX, v[a].SrcY, v[b].SrcX, v[b].SrcY)
	}
	// the two triangles (0 1 2) and (0 2 3), every edge of each
	l := edge(0, 1)
	for _, e := range [5][2]int{{1, 2}, {2, 0}, {0, 2}, {2, 3}, {3, 0}} {
		l = min(l, edge(e[0], e[1]))
	}
	return l
}

// mipLevelFromDistance is Ebitengine's (internal/mipmap) level choice for
// one edge.
func mipLevelFromDistance(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1 float32) int {
	const maxLevel = 6
	d := (dx1-dx0)*(dx1-dx0) + (dy1-dy0)*(dy1-dy0)
	s := (sx1-sx0)*(sx1-sx0) + (sy1-sy0)*(sy1-sy0)
	if s == 0 {
		return 0
	}
	scale := d / s
	if math.IsInf(float64(scale), 0) || scale == 0 {
		return 0
	}
	level := 0
	for scale < 0.25 {
		level++
		scale *= 4
	}
	if level > 0 {
		w, h := int(sx1-sx0), int(sy1-sy0)
		for level >= 0 {
			s := 1 << uint(level)
			if (w > 0 && w/s == 0) || (h > 0 && h/s == 0) {
				level--
				continue
			}
			break
		}
		if level < 0 {
			return 0
		}
	}
	return min(level, maxLevel)
}

// worldAA draws the 3D world anti-aliased: it is rendered at twice the
// resolution and filtered down, four times the pixels. The Low preset turns
// it off (settings: Anti-aliasing).
//
// The supersampled target is kept here rather than left to Ebitengine's
// AntiAlias option: that one measures every draw call's vertices to find
// the region it touches, scales them in a second pass, and resolves and
// re-copies the whole region each time the blend mode changes. Here the
// region is the destination, the scale is folded into the vertex copy
// drawRuns already makes, and the opaque and additive lists share a single
// copy up and a single resolve.
var worldAA = true

const aaScale = 2

// aaOwn selects the supersampled target kept here over Ebitengine's
// AntiAlias option (bench: -ebitenaa).
var aaOwn = true

var aaImg *ebiten.Image

// aaBegin returns the supersampled target for dst, holding dst's current
// picture (so blending onto it is unchanged).
func aaBegin(dst *ebiten.Image) *ebiten.Image {
	r := dst.Bounds()
	w, h := r.Dx()*aaScale, r.Dy()*aaScale
	if aaImg == nil || aaImg.Bounds().Dx() != w || aaImg.Bounds().Dy() != h {
		if aaImg != nil {
			aaImg.Deallocate()
		}
		aaImg = ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
	}
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
	op.GeoM.Translate(-float64(r.Min.X), -float64(r.Min.Y))
	op.GeoM.Scale(aaScale, aaScale)
	aaImg.DrawImage(dst, op)
	return aaImg
}

// aaEnd resolves the supersampled target into dst: each pixel the average
// of its 2x2 block (a bilinear sample at the block's centre).
func aaEnd(dst *ebiten.Image) {
	r := dst.Bounds()
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy, Filter: ebiten.FilterLinear, DisableMipmaps: true}
	op.GeoM.Scale(1.0/aaScale, 1.0/aaScale)
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	dst.DrawImage(aaImg, op)
}

// planRunsOn enables run planning (-noruns turns it off, for benchmarks).
var planRunsOn = true

// A list is planned when its last frame had more texture switches than
// runPlanOn, and stays planned until they fall below runPlanOff.
const (
	runPlanOn  = 60
	runPlanOff = 30
)

// prepMeta readies quadMeta for this flush (grid cell size, texture sizes).
func (b *Batch) prepMeta() {
	for i := 1; i < len(texImages) && i < len(texSize); i++ {
		if texSize[i] == ([2]float32{}) && texImages[i] != nil {
			bd := texImages[i].Bounds()
			texSize[i] = [2]float32{float32(bd.Dx()), float32(bd.Dy())}
		}
	}
	b.cellW = float32(b.cam.W) / runCols
	b.cellH = float32(b.cam.H) / runRows
}

// quadMeta packs what planRuns needs about a quad: its grid cells (with a
// pixel of margin) and its run key (texture and mip level). It is computed
// in the sort's sequential pass, so planning, which visits quads in depth
// order, reads a small table instead of jumping around the quads.
func (b *Batch) quadMeta(q *quad) uint64 { return quadMetaAt(q, b.cellW, b.cellH) }

// quadMetaAt is quadMeta for a grid of cw x ch pixel cells.
func quadMetaAt(q *quad, cw, ch float32) uint64 {
	x0, y0 := q.v[0].DstX, q.v[0].DstY
	x1, y1 := x0, y0
	for k := 1; k < 4; k++ {
		x0, x1 = min(x0, q.v[k].DstX), max(x1, q.v[k].DstX)
		y0, y1 = min(y0, q.v[k].DstY), max(y1, q.v[k].DstY)
	}
	c0 := uint64(max(0, min(runCols-1, (x0-1)/cw)))
	c1 := uint64(max(0, min(runCols-1, (x1+1)/cw)))
	r0 := uint64(max(0, min(runRows-1, (y0-1)/ch)))
	r1 := uint64(max(0, min(runRows-1, (y1+1)/ch)))
	key := uint64(q.tex) * 8
	if q.tex != 0 {
		if quadWraps(q, texSize[q.tex]) {
			key += noMip
		} else {
			key += uint64(quadMipLevel(q))
		}
	}
	return c0 | c1<<8 | r0<<16 | r1<<24 | key<<32
}

func (b *Batch) planRuns(qs []*quad, order []int32, plan bool) ([]quadRun, int) {
	for i := range b.runs {
		b.runs[i].ids = b.runs[i].ids[:0]
	}
	runs := b.runs[:0]
	var lastOf [256 * 8]int16 // latest run per texture and mip level
	for i := range lastOf {
		lastOf[i] = -1
	}
	cell := &b.runCell // latest run touching each fine cell (small quads)
	for i := range cell {
		cell[i] = -1
	}
	// per block: the latest run of a big quad covering it, and the latest
	// run in any of its fine cells. Every overlapping pair shares a fine
	// cell and so a block, which keeps the test conservative both ways.
	big, fineMax := &b.runBigB, &b.runFineB
	for i := range big {
		big[i], fineMax[i] = -1, -1
	}
	meta := b.meta
	switches := 0
	prevTex := uint8(255)
	if !plan {
		// plain runs: a new draw call at every texture switch, the
		// indices straight from the order
		b.runIdx = append(b.runIdx[:0], order...)
		tx := b.qtex[:len(qs)]
		for i, oi := range order {
			tex := tx[oi]
			if len(runs) > 0 && runs[len(runs)-1].tex == tex {
				continue
			}
			if len(runs) > 0 {
				switches++
				runs[len(runs)-1].last = i
			}
			if len(runs) < cap(runs) {
				runs = runs[:len(runs)+1] // (keeping the slot's ids storage)
			} else {
				runs = append(runs, quadRun{})
			}
			rn := &runs[len(runs)-1]
			rn.tex, rn.first, rn.noMip, rn.oneMip = tex, i, false, false
		}
		if len(runs) > 0 {
			runs[len(runs)-1].last = len(order)
		}
		b.runs = runs
		return runs, switches
	}
	for _, oi := range order {
		m := meta[oi]
		c0, c1, r0, r1 := int(m&0xff), int(m>>8&0xff), int(m>>16&0xff), int(m>>24&0xff)
		key := int(m >> 32 & 0xffff)
		tex := uint8(key / 8)
		if tex != prevTex {
			switches++
			prevTex = tex
		}
		// it can join its key's latest run if no later run touched its cells
		isBig := (c1-c0+1)*(r1-r0+1) > runBig
		bx0, bx1, by0, by1 := c0/runBlock, c1/runBlock, r0/runBlock, r1/runBlock
		at := int(lastOf[key])
		if at == len(runs)-1 {
			// joining the newest run: no later run can be in the way
		} else if at >= 0 && isBig {
		scanB:
			for y := by0; y <= by1; y++ {
				for x := bx0; x <= bx1; x++ {
					if i := y*runBX + x; int(big[i]) > at || int(fineMax[i]) > at {
						at = -1
						break scanB
					}
				}
			}
		} else if at >= 0 {
		scan:
			for y := by0; y <= by1; y++ {
				for x := bx0; x <= bx1; x++ {
					if int(big[y*runBX+x]) > at {
						at = -1
						break scan
					}
				}
			}
			if at >= 0 {
			scanC:
				for y := r0; y <= r1; y++ {
					for _, c := range cell[y*runCols+c0 : y*runCols+c1+1] {
						if int(c) > at {
							at = -1
							break scanC
						}
					}
				}
			}
		}
		if at < 0 {
			if len(runs) < cap(runs) {
				runs = runs[:len(runs)+1]
				rn := &runs[len(runs)-1]
				rn.tex, rn.ids = tex, rn.ids[:0]
			} else {
				runs = append(runs, quadRun{tex: tex})
			}
			at = len(runs) - 1
			runs[at].noMip = key%8 == noMip
			runs[at].oneMip = true // keyed by mip level: every quad shares it
			lastOf[key] = int16(at)
		}
		// The scan left nothing later than at under the quad (and a new run
		// is the latest of all), so its own cells or blocks are simply set;
		// only a block's latest-of-its-fine-cells needs a max.
		a := int16(at)
		if isBig {
			for y := by0; y <= by1; y++ {
				for x := bx0; x <= bx1; x++ {
					big[y*runBX+x] = a
				}
			}
		} else {
			for y := r0; y <= r1; y++ {
				row := cell[y*runCols+c0 : y*runCols+c1+1]
				for x := range row {
					row[x] = a
				}
			}
			for y := by0; y <= by1; y++ {
				for x := bx0; x <= bx1; x++ {
					if i := y*runBX + x; fineMax[i] < a {
						fineMax[i] = a
					}
				}
			}
		}
		runs[at].ids = append(runs[at].ids, oi)
	}
	// flatten the runs' indices into one list
	b.runIdx = b.runIdx[:0]
	for i := range runs {
		runs[i].first = len(b.runIdx)
		b.runIdx = append(b.runIdx, runs[i].ids...)
		runs[i].last = len(b.runIdx)
	}
	b.runs = runs
	return runs, switches
}

// flushTrace, when set (debug builds), sees every texture switch in a flush.
var flushTrace func(from, to uint8, depth float64)

// maxQ is the most quads one DrawTriangles call carries (uint16 indices).
const maxQ = 16000

// quadIndices is the index list for maxQ consecutive quads (0,1,2 0,2,3).
var quadIndices = func() []uint16 {
	ix := make([]uint16, 0, maxQ*6)
	for i := 0; i < maxQ; i++ {
		b := uint16(i * 4)
		ix = append(ix, b, b+1, b+2, b, b+2, b+3)
	}
	return ix
}()

// quadIndices32 is quadIndices as 32-bit indices, which DrawTriangles32
// passes on without converting them every call.
var quadIndices32 = func() []uint32 {
	ix := make([]uint32, len(quadIndices))
	for i, v := range quadIndices {
		ix[i] = uint32(v)
	}
	return ix
}()

// sortFarFirst returns the quad indices ordered far to near, keeping the
// submission order among equal depths: exactly what a stable sort on
// descending depth gives, but as an LSD radix sort on the depth's bits
// (11 bits a pass, skipping passes where every key shares the digit), which
// is several times faster for tens of thousands of quads.
func (b *Batch) sortFarFirst(qs []*quad, withMeta bool) []int32 {
	n := len(qs)
	b.growSort(n)
	if cap(b.meta) < n {
		b.meta = make([]uint64, n)
	}
	meta := b.meta[:n]
	k0 := b.keys[0][:n]
	tx := b.qtex[:n]
	for i := range qs {
		if withMeta {
			meta[i] = b.quadMeta(qs[i])
		}
		k0[i] = depthKey(qs[i].depth)
		tx[i] = qs[i].tex
	}
	return b.radixOrder(n)
}

// depthKey maps a depth to a key whose ascending order is far to near.
func depthKey(d float64) uint64 {
	u := math.Float64bits(d)
	if u>>63 != 0 {
		u = ^u // negative: flip every bit
	} else {
		u |= 1 << 63 // positive: set the sign bit
	}
	return ^u // ascending key = descending depth
}

func (b *Batch) growSort(n int) {
	if cap(b.keys[0]) < n {
		b.keys[0] = make([]uint64, n)
		b.keys[1] = make([]uint64, n)
		b.order = make([]int32, n)
		b.idx2 = make([]int32, n)
	}
}

// radixOrder stable-sorts the n keys in b.keys[0] (which it uses as
// scratch) and returns the order, in b's scratch.
func (b *Batch) radixOrder(n int) []int32 {
	k0, k1 := b.keys[0][:n], b.keys[1][:n]
	o0, o1 := b.order[:n], b.idx2[:n]
	const bits, passes = 11, 6
	var hist [passes][1 << bits]int32
	for i, u := range k0 {
		o0[i] = int32(i)
		for p := 0; p < passes; p++ {
			hist[p][(u>>(p*bits))&(1<<bits-1)]++
		}
	}
	for p := 0; p < passes; p++ {
		h := &hist[p]
		shift := uint(p * bits)
		if h[(k0[0]>>shift)&(1<<bits-1)] == int32(n) {
			continue // every key has this digit: the pass would change nothing
		}
		var sum int32
		for d := range h {
			c := h[d]
			h[d] = sum
			sum += c
		}
		for i := 0; i < n; i++ {
			d := (k0[i] >> shift) & (1<<bits - 1)
			j := h[d]
			h[d]++
			k1[j] = k0[i]
			o1[j] = o0[i]
		}
		k0, k1 = k1, k0
		o0, o1 = o1, o0
	}
	return o0
}

// ---------------------------------------------------------------- presorting
//
// The scenery and the arena are most of a frame's quads and barely move
// against each other from one frame to the next, so their side batches sort
// themselves on their own goroutines (presort), starting from last frame's
// order: an insertion pass over a nearly sorted list costs about one sweep,
// against six for the radix sort. The flush then only sorts the frame's own
// quads (bricks, orbs, effects) and merges the three lists. Ties keep
// submission order throughout, so the result is exactly the order one
// stable sort of everything would give.

// presorted is one of a side batch's lists, sorted far first.
type presorted struct {
	keys   []uint64 // per quad, in submission order
	order  []int32
	meta   []uint64 // per quad: what planRuns needs (quadMeta)
	tex    []uint8  // per quad, in submission order
	skeys  []uint64 // keys in sorted order (skeys[j] = keys[order[j]])
	metaOK bool     // false: some texture's size was unknown; recompute
	valid  bool
}

// presort sorts the batch's own opaque and glow lists and works out their
// run metadata. Call on the goroutine that built the batch, after drawing.
func (b *Batch) presort() {
	cw, ch := float32(b.cam.W)/runCols, float32(b.cam.H)/runRows
	b.presortList(&b.pre[0], b.quads, cw, ch, b.wantMeta[0])
	b.presortList(&b.pre[1], b.glow, cw, ch, b.wantMeta[1])
}

func (b *Batch) presortList(p *presorted, qs []quad, cw, ch float32, wantMeta bool) {
	n := len(qs)
	if cap(p.keys) < n {
		p.keys = make([]uint64, n, n+n/4)
		p.meta = make([]uint64, n, n+n/4)
	}
	if cap(p.tex) < n {
		p.tex = make([]uint8, n, n+n/4)
	}
	keys, meta, tex := p.keys[:n], p.meta[:n], p.tex[:n]
	p.keys, p.meta, p.tex = keys, meta, tex
	p.metaOK = wantMeta
	for i := range qs {
		q := &qs[i]
		keys[i] = depthKey(q.depth)
		tex[i] = q.tex
		if !wantMeta {
			continue
		}
		if q.tex != 0 && texSize[q.tex] == ([2]float32{}) {
			p.metaOK = false // registered since the last flush: the flush works it out
			continue
		}
		meta[i] = quadMetaAt(q, cw, ch)
	}
	p.valid = true
	if n == 0 {
		p.order = p.order[:0]
		p.skeys = p.skeys[:0]
		return
	}
	if len(p.order) != n || !insertionFix(p.order, keys, 8*n+1024) {
		// first frame, a change in what is drawn, or too much movement:
		// sort afresh
		b.growSort(n)
		copy(b.keys[0][:n], keys)
		o := b.radixOrder(n)
		p.order = append(p.order[:0], o...)
	}
	// the keys in order, gathered here on the batch's own goroutine so the
	// flush's merge reads them straight through
	if cap(p.skeys) < n {
		p.skeys = make([]uint64, n, n+n/4)
	}
	p.skeys = p.skeys[:n]
	for j, oi := range p.order {
		p.skeys[j] = keys[oi]
	}
}

// insertionFix sorts order by (key, index) in place, giving up (false,
// leaving order a permutation) once it has moved more than budget entries.
func insertionFix(order []int32, keys []uint64, budget int) bool {
	moves := 0
	for i := 1; i < len(order); i++ {
		x := order[i]
		kx := keys[x]
		j := i - 1
		for j >= 0 {
			y := order[j]
			if ky := keys[y]; ky < kx || (ky == kx && y < x) {
				break
			}
			order[j+1] = y
			j--
		}
		order[j+1] = x
		if moves += i - 1 - j; moves > budget {
			return false
		}
	}
	return true
}

// presortOn uses the side batches' presorted orders (bench: -nopresort).
var presortOn = true

// maxMerge is the most lists mergeSorted merges (side batches plus own).
const maxMerge = 20

// mergeSorted orders the flush's segments far first: the side batches'
// presorted lists merged with the batch's own quads, sorted here. Ties go
// to the earlier segment, then to submission order. It fills b.meta for
// planRuns when plan is set.
func (b *Batch) mergeSorted(segs [][]quad, list int, plan bool) []int32 {
	own := segs[len(segs)-1]
	n := 0
	for _, s := range segs {
		n += len(s)
	}
	b.growSort(n)
	if cap(b.meta) < n {
		b.meta = make([]uint64, n)
	}
	meta := b.meta[:n]
	// the own quads: keys and metadata, then their order
	no := len(own)
	if cap(b.mergeK) < no {
		b.mergeK = make([]uint64, no, no+no/4)
	}
	ownK := b.mergeK[:no]
	base := n - no
	k0 := b.keys[0][:no]
	tx := b.qtex[:n]
	for i := range own {
		ownK[i] = depthKey(own[i].depth)
		k0[i] = ownK[i]
		tx[base+i] = own[i].tex
		if plan {
			meta[base+i] = b.quadMeta(&own[i])
		}
	}
	var ownOrd []int32
	if no > 0 {
		ownOrd = b.radixOrder(no)
		// keep it out of the scratch the merge writes into
		ownOrd = append(b.idx2[n-no : n][:0], ownOrd...)
	}
	// the side segments' metadata, copied into place, and every list laid
	// out back to back as (key, global index) pairs in sorted order
	if cap(b.mk[0]) < n {
		for i := range b.mk {
			b.mk[i] = make([]uint64, n, n+n/4)
			b.mi[i] = make([]int32, n, n+n/4)
		}
	}
	mk, mi := b.mk[0][:n], b.mi[0][:n]
	var bounds [maxMerge + 1]int
	nl := 0
	off := 0
	for si, s := range segs[:len(segs)-1] {
		p := &b.extB[si].pre[list]
		if !p.valid || len(p.order) != len(s) {
			// not presorted after all: fall back to one sort of everything
			qs := b.qp
			return b.sortFarFirst(qs, plan)
		}
		copy(tx[off:off+len(s)], p.tex[:len(s)])
		if plan {
			if p.metaOK {
				copy(meta[off:off+len(s)], p.meta[:len(s)])
			} else {
				for i := range s {
					meta[off+i] = b.quadMeta(&s[i])
				}
			}
		}
		bounds[nl] = off
		nl++
		copy(mk[off:off+len(s)], p.skeys)
		o32 := int32(off)
		for j, oi := range p.order {
			mi[off+j] = o32 + oi
		}
		off += len(s)
	}
	bounds[nl] = off
	nl++
	for j, oi := range ownOrd {
		mk[off+j] = ownK[oi]
		mi[off+j] = int32(base) + oi
	}
	bounds[nl] = n
	// merge neighbouring lists pairwise until one is left; ties go to the
	// earlier list, so the result is the order one stable sort would give.
	// A big pass is shared out: each merge cut into slices of the output
	// (mergeSplit), merged on their own goroutines.
	par := parMergeOn && n >= parMergeMin && (runtime.GOMAXPROCS(0) >= 4 || forcePieces)
	src := 0
	for nl > 1 {
		dk, di := b.mk[1-src][:n], b.mi[1-src][:n]
		sk, si := b.mk[src][:n], b.mi[src][:n]
		m := 0
		var wg sync.WaitGroup
		for l := 0; l < nl; l += 2 {
			lo, mid := bounds[l], bounds[l+1]
			if l+1 >= nl {
				copy(dk[lo:mid], sk[lo:mid])
				copy(di[lo:mid], si[lo:mid])
				bounds[m] = lo
				m++
				continue
			}
			hi := bounds[l+2]
			bounds[m] = lo
			m++
			parts := 1
			if par {
				parts = min(8, (hi-lo+parMergeSlice-1)/parMergeSlice)
			}
			if parts <= 1 {
				if par {
					wg.Add(1)
					go func() {
						defer wg.Done()
						mergeRuns(dk, di, sk, si, lo, mid, mid, hi, lo)
					}()
				} else {
					mergeRuns(dk, di, sk, si, lo, mid, mid, hi, lo)
				}
				continue
			}
			for c := 0; c < parts; c++ {
				k0, k1 := (hi-lo)*c/parts, (hi-lo)*(c+1)/parts
				i0 := mergeSplit(sk, lo, mid, mid, hi, k0)
				i1 := mergeSplit(sk, lo, mid, mid, hi, k1)
				a0, a1 := lo+i0, lo+i1
				b0, b1 := mid+k0-i0, mid+k1-i1
				wg.Add(1)
				go func() {
					defer wg.Done()
					mergeRuns(dk, di, sk, si, a0, a1, b0, b1, lo+k0)
				}()
			}
		}
		wg.Wait()
		bounds[m] = n
		nl = m
		src = 1 - src
	}
	out := append(b.order[:0], b.mi[src][:n]...)
	b.order = out[:cap(out)]
	return out
}

// parMergeOn enables sharing out the merge (bench: -noparmerge).
var parMergeOn = true

// parMergeMin is the fewest quads whose merge passes are shared out across
// goroutines, and parMergeSlice the output each one takes at least.
const (
	parMergeMin   = 6000
	parMergeSlice = 3000
)

// mergeRuns merges the sorted runs sk/si[a0:a1] and [b0:b1] into dk/di from
// position w, ties to the first run.
func mergeRuns(dk []uint64, di []int32, sk []uint64, si []int32, a0, a1, b0, b1, w int) {
	i, j := a0, b0
	for i < a1 && j < b1 {
		if sk[j] < sk[i] {
			dk[w], di[w] = sk[j], si[j]
			j++
		} else {
			dk[w], di[w] = sk[i], si[i]
			i++
		}
		w++
	}
	n := a1 - i
	copy(dk[w:w+n], sk[i:a1])
	copy(di[w:w+n], si[i:a1])
	w += n
	n = b1 - j
	copy(dk[w:w+n], sk[j:b1])
	copy(di[w:w+n], si[j:b1])
}

// mergeSplit returns how many of the first run's elements (sk[a0:a1]) the
// first k outputs of merging it with sk[b0:b1] take (ties to the first run),
// so the merge can be cut at output k and its slices merged independently.
func mergeSplit(sk []uint64, a0, a1, b0, b1, k int) int {
	lo, hi := max(0, k-(b1-b0)), min(k, a1-a0)
	for lo < hi {
		mid := (lo + hi) / 2
		// a[mid] comes before b[k-mid-1] if it is not greater
		if sk[a0+mid] <= sk[b0+k-mid-1] {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// lastQuads is how many quads the last world flush drew (for profiling).
var lastQuads int

func (b *Batch) Flush(dst *ebiten.Image) {
	perfBegin(secFlush)
	defer perfEnd(secFlush)
	lastQuads = len(b.quads) + len(b.glow)
	nGlow := len(b.glow)
	for i := range b.extQ {
		lastQuads += len(b.extQ[i]) + len(b.extG[i])
		nGlow += len(b.extG[i])
	}
	b.prepMeta()
	tgt, sc := dst, float32(1)
	own := worldAA && aaOwn
	if own {
		tgt, sc = aaBegin(dst), aaScale
	}
	segQ := append(b.extQ, b.quads)
	segG := append(b.extG, b.glow)
	if nGlow >= glowPlanAside && glowAsideOn {
		// the glow list is sorted and planned on another goroutine while
		// this one submits the opaque list (planning makes no Ebitengine
		// calls; drawing stays here)
		gp := b.glowPlanner()
		var gqs []*quad
		var gruns []quadRun
		var crash any
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { crash = recover() }()
			gqs, gruns = gp.planList(segG, 1)
		}()
		qs, runs := b.planList(segQ, 0)
		b.drawRuns(tgt, qs, runs, ebiten.BlendSourceOver, sc)
		<-done
		if crash != nil {
			panic(crash)
		}
		b.switches[1], b.planned[1] = gp.switches[1], gp.planned[1]
		gp.drawRuns(tgt, gqs, gruns, ebiten.BlendLighter, sc)
	} else {
		qs, runs := b.planList(segQ, 0)
		b.drawRuns(tgt, qs, runs, ebiten.BlendSourceOver, sc)
		qs, runs = b.planList(segG, 1)
		b.drawRuns(tgt, qs, runs, ebiten.BlendLighter, sc)
	}
	if own {
		aaEnd(dst)
	}
	if flushBenchHook != nil {
		flushBenchHook(b, segQ, segG)
	}
	b.extQ, b.extG, b.extB = b.extQ[:0], b.extG[:0], b.extB[:0]
}

// glowPlanAside is the glow-list size from which Flush plans it on a
// goroutine of its own.
const glowPlanAside = 1500

// glowAsideOn enables planning the glow list on its own goroutine (bench:
// -glowaside). Off: measured on a 32-thread PC it saved nothing, the
// goroutine hand-off costing what the overlap won.
var glowAsideOn = false

// glowPlanner returns the scratch batch the glow list is planned in, set up
// to read the same side batches and grid as b.
func (b *Batch) glowPlanner() *Batch {
	if b.gp == nil {
		b.gp = &Batch{}
	}
	g := b.gp
	g.cam, g.extB, g.cellW, g.cellH = b.cam, b.extB, b.cellW, b.cellH
	g.switches[1], g.planned[1] = b.switches[1], b.planned[1]
	return g
}

// drawFirst queues the side batches' quads to be drawn ahead of b's own,
// in argument order, as if they had been drawn into b before anything
// else (b must not have quads of its own from before they were split off).
func (b *Batch) drawFirst(sides ...*Batch) {
	for _, s := range sides {
		b.extQ = append(b.extQ, s.quads)
		b.extG = append(b.extG, s.glow)
		b.extB = append(b.extB, s)
	}
}

// ---------------------------------------------------------------- 2D helpers
//
// These take premultiplied colours (as produced by prem). Shapes with
// curved or diagonal edges go through the SDF renderer (sdf.go), which
// anti-aliases analytically; axis-aligned rects need no anti-aliasing.

// unprem converts a premultiplied colour back to straight alpha.
func unprem(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return color.RGBA{}
	}
	return color.RGBA{uint8(r * 255 / a), uint8(g * 255 / a), uint8(b * 255 / a), uint8(a >> 8)}
}

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	perf.cur.rect++
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func fillCircle(dst *ebiten.Image, x, y, r float64, c color.Color) {
	s := unprem(c)
	fillPath(dst, circlePath(x, y, r), s, s, grad{}, false)
}

func strokeLine(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.Color) {
	strokePath(dst, linePath(x0, y0, x1, y1), w, unprem(c), false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, sw float64, c color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(sw), c, false)
}

func roundRect(dst *ebiten.Image, x, y, w, h, r float64, c color.Color) {
	s := unprem(c)
	fillPath(dst, boxPath(x, y, w, h, r, false), s, s, grad{}, false)
}

func strokeRoundRect(dst *ebiten.Image, x, y, w, h, r, sw float64, c color.Color) {
	strokePath(dst, boxPath(x, y, w, h, r, false), sw, unprem(c), false)
}

// drawSprite draws an atlas sprite in 2D.
func drawSprite(dst *ebiten.Image, spr Sprite, x, y, w, h float64, c color.RGBA, additive bool) {
	perf.cur.sprite++
	cc := cf(c)
	// package scratch: 2D drawing happens on the main goroutine only, and
	// DrawTriangles copies the vertices before it returns
	vs := spriteVerts[:]
	vs[0] = ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: spr.X0, SrcY: spr.Y0}
	vs[1] = ebiten.Vertex{DstX: float32(x + w), DstY: float32(y), SrcX: spr.X1, SrcY: spr.Y0}
	vs[2] = ebiten.Vertex{DstX: float32(x + w), DstY: float32(y + h), SrcX: spr.X1, SrcY: spr.Y1}
	vs[3] = ebiten.Vertex{DstX: float32(x), DstY: float32(y + h), SrcX: spr.X0, SrcY: spr.Y1}
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = cc[0]*cc[3], cc[1]*cc[3], cc[2]*cc[3], cc[3]
	}
	spriteOp = ebiten.DrawTrianglesOptions{}
	if additive {
		spriteOp.Blend = ebiten.BlendLighter
	}
	dst.DrawTriangles(vs, quadIndices[:6], atlas, &spriteOp)
}

var (
	spriteVerts [4]ebiten.Vertex
	spriteOp    ebiten.DrawTrianglesOptions
)

// fillRectA fills with colour c at opacity a (straight alpha in, so the
// premultiplication vector drawing expects is done here).
func fillRectA(dst *ebiten.Image, x, y, w, h float64, c color.RGBA, a float64) {
	perf.cur.rect++
	a = clamp01(a) * float64(c.A) / 255
	pm := color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), pm, false)
}

// planT adds up planList's time sorting and planning runs (the capture
// build's planner benchmark reads it).
var planT [2]time.Duration
