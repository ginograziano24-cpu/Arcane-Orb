package main

// A small purpose-built 3D renderer on top of Ebitengine's DrawTriangles:
// perspective camera, flat-shaded boxes with per-vertex lighting, billboards,
// floor decals and painter's-algorithm depth sorting. Everything is drawn in
// two batched calls per frame (opaque-ish + additive glow) from one atlas.

import (
	"image"
	"image/color"
	"math"
	"sort"

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
}

type Batch struct {
	cam      *Camera
	quads    []quad
	glow     []quad
	verts    []ebiten.Vertex
	idx      []uint16
	fogColor [3]float32
	fogNear  float64
	fogFar   float64
	light    V3
	lightCol [3]float32
	ambCol   [3]float32
}

func newBatch(cam *Camera) *Batch {
	return &Batch{cam: cam, light: V3{-0.35, 0.85, -0.4}.Norm(), fogNear: 30, fogFar: 75,
		lightCol: [3]float32{0.75, 0.7, 0.85}, ambCol: [3]float32{0.45, 0.4, 0.6}}
}

func (b *Batch) Reset() {
	b.quads = b.quads[:0]
	b.glow = b.glow[:0]
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
	if glow {
		b.glow = append(b.glow, q)
	} else {
		b.quads = append(b.quads, q)
	}
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
	b.quads[len(b.quads)-1].depth += 1000
}

func (b *Batch) flush(dst *ebiten.Image, qs []quad, blend ebiten.Blend) {
	if len(qs) == 0 {
		return
	}
	sort.SliceStable(qs, func(i, j int) bool { return qs[i].depth > qs[j].depth })
	const maxQ = 16000
	for start := 0; start < len(qs); start += maxQ {
		end := min(len(qs), start+maxQ)
		b.verts = b.verts[:0]
		b.idx = b.idx[:0]
		for _, q := range qs[start:end] {
			base := uint16(len(b.verts))
			b.verts = append(b.verts, q.v[:]...)
			b.idx = append(b.idx, base, base+1, base+2, base, base+2, base+3)
		}
		op := &ebiten.DrawTrianglesOptions{Blend: blend, AntiAlias: true}
		dst.DrawTriangles(b.verts, b.idx, atlas, op)
	}
}

func (b *Batch) Flush(dst *ebiten.Image) {
	b.flush(dst, b.quads, ebiten.BlendSourceOver)
	b.flush(dst, b.glow, ebiten.BlendLighter)
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
	cc := cf(c)
	vs := []ebiten.Vertex{
		{DstX: float32(x), DstY: float32(y), SrcX: spr.X0, SrcY: spr.Y0},
		{DstX: float32(x + w), DstY: float32(y), SrcX: spr.X1, SrcY: spr.Y0},
		{DstX: float32(x + w), DstY: float32(y + h), SrcX: spr.X1, SrcY: spr.Y1},
		{DstX: float32(x), DstY: float32(y + h), SrcX: spr.X0, SrcY: spr.Y1},
	}
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = cc[0]*cc[3], cc[1]*cc[3], cc[2]*cc[3], cc[3]
	}
	op := &ebiten.DrawTrianglesOptions{}
	if additive {
		op.Blend = ebiten.BlendLighter
	}
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, atlas, op)
}

// fillRectA fills with colour c at opacity a (straight alpha in, so the
// premultiplication vector drawing expects is done here).
func fillRectA(dst *ebiten.Image, x, y, w, h float64, c color.RGBA, a float64) {
	a = clamp01(a) * float64(c.A) / 255
	pm := color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), pm, false)
}
