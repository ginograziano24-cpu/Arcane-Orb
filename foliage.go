package main

import (
	"image/color"
	"math"
	"sync"
)

// Grass and flowers for the open ground around the arena. The whole field
// is baked once per chamber into one world-space mesh (one draw, shared
// corners): blades face the camera (the playing camera always looks down
// +z), each a dark lower quad and a lighter tip, grown in clumps that
// thin out with distance. Flower heads are billboards. Low quality and
// the Classic backdrop skip it.

type grassStyle struct {
	base, tip  color.RGBA
	bloom      []color.RGBA // flower heads (nil = none)
	glow       float32      // emissive tips (Glimmer Grove)
	n          int          // tufts per chamber
	height     float64
	bloomEvery int
}

var grassStyles = map[int]grassStyle{
	WMeadow: {base: color.RGBA{58, 104, 40, 255}, tip: color.RGBA{168, 214, 96, 255},
		n: 120, height: 1.5}, // blades only: the coloured dots read as litter
	WNeon: {base: color.RGBA{24, 50, 70, 255}, tip: color.RGBA{90, 220, 200, 255},
		bloom: []color.RGBA{{120, 240, 255, 255}, {200, 140, 255, 255}}, glow: 0.35, n: 100, height: 1.4, bloomEvery: 5},
	WRuins: {base: color.RGBA{96, 104, 52, 255}, tip: color.RGBA{178, 168, 92, 255}, n: 60, height: 1.3},
	WDream: {base: color.RGBA{60, 140, 100, 255}, tip: color.RGBA{190, 255, 200, 255},
		bloom: []color.RGBA{{255, 150, 220, 255}, {200, 170, 255, 255}, {255, 230, 150, 255}}, glow: 0.25, n: 120, height: 1.2, bloomEvery: 4},
}

type grassBloom struct {
	p V3
	c color.RGBA
	s float64
}

type grassField struct {
	seed  int64
	world int
	n     int
	mesh  *Mesh
	bloom []grassBloom
}

var (
	grassMu   sync.Mutex
	grassLast *grassField
)

// bakeGrass builds (or returns the cached) field for this chamber.
func bakeGrass(L *Level, st grassStyle, n int) *grassField {
	grassMu.Lock()
	defer grassMu.Unlock()
	if f := grassLast; f != nil && f.seed == L.propSeed && f.world == L.Biome.ID && f.n == n {
		return f
	}
	r := seededRand(L.propSeed + 303)
	defer releaseRand(r)
	m := &Mesh{}
	f := &grassField{seed: L.propSeed, world: L.Biome.ID, n: n}
	var cx, cz float64
	for i := 0; i < n; i++ {
		if i%5 == 0 { // a new clump
			sx := float64(r.Intn(2)*2 - 1)
			d := r.Float64()
			cx = sx * (HalfW + 3.2 + d*d*44)
			cz = -6 + r.Float64()*58
			if cz > FarZ+2 && math.Abs(cx) < HalfW+8 {
				cz += 8 // clear of the far wall
			}
		}
		x := cx + (r.Float64()-0.5)*3.2
		z := cz + (r.Float64()-0.5)*3.2
		if math.Abs(x) < HalfW+1.6 {
			x = math.Copysign(HalfW+1.6, x)
		}
		s := 0.9 + r.Float64()*0.8
		blades := 7 + r.Intn(4)
		for b := 0; b < blades; b++ {
			h := st.height * s * (0.55 + r.Float64()*0.65)
			w := (0.09 + r.Float64()*0.05) * s
			ox, oz := x+(r.Float64()-0.5)*0.7*s, z+(r.Float64()-0.5)*0.6*s
			lean := (r.Float64() - 0.5) * 0.9 * h // sideways, in the camera's plane
			// a real blade: three segments tapering to a point and curving
			// over as it rises, dark at the root and light at the tip
			y0 := terrainY(L.Biome.ID, ox, oz) - 0.05
			tc := mixc(st.tip, st.base, 0.05+r.Float64()*0.4)
			pt := func(f float64) (V3, float64) {
				bend := lean * f * f
				return V3{ox + bend, y0 + h*(f-0.12*f*f*math.Abs(lean)/h), oz + 0.05*f}, w * (1 - f*0.92)
			}
			var prevL, prevR V3
			for sg := 0; sg <= 3; sg++ {
				f := float64(sg) / 3
				c, hw := pt(f)
				L, R := V3{c.X - hw, c.Y, c.Z}, V3{c.X + hw, c.Y, c.Z}
				if sg > 0 {
					col := mixc(st.base, tc, f)
					if sg == 3 {
						m.Tris = append(m.Tris, Tri{P: [3]V3{prevL, c, prevR}, Col: col, Emi: st.glow, Fixed: true})
					} else {
						m.Tris = append(m.Tris, Tri{P: [3]V3{prevL, L, prevR}, Col: darken(col, 0.94), Emi: st.glow * float32(f), Fixed: true},
							Tri{P: [3]V3{prevR, L, R}, Col: col, Emi: st.glow * float32(f), Fixed: true})
					}
				}
				prevL, prevR = L, R
			}
		}
		if st.bloom != nil && st.bloomEvery > 0 && i%st.bloomEvery == 0 {
			c := st.bloom[(i/st.bloomEvery)%len(st.bloom)]
			f.bloom = append(f.bloom, grassBloom{V3{x, terrainY(L.Biome.ID, x, z) + st.height*0.95*s, z}, c, s})
		}
	}
	// make sure every blade faces the camera whatever the winding above
	for i := range m.Tris {
		t := &m.Tris[i]
		nrm := t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0]))
		if nrm.Z > 0 {
			t.P[1], t.P[2] = t.P[2], t.P[1]
		}
	}
	m.topology()
	f.mesh = m
	grassLast = f
	return f
}

func (g *Game) drawGrass(B *Batch, L *Level, t float64) {
	st, ok := grassStyles[L.Biome.ID]
	if !ok || g.lowFX() || g.settings.Classic {
		return
	}
	n := st.n
	if g.settings.Quality == 1 {
		n = n * 2 / 3
	}
	f := bakeGrass(L, st, n)
	B.Draw(f.mesh, V3{}, v3(1), identity, color.RGBA{}, 0)
	for i, bl := range f.bloom {
		sway := math.Sin(t*1.3+float64(i)*1.7) * 0.06
		p := bl.p.Add(V3{sway, 0, 0})
		B.Billboard(p, 0.42*bl.s, sprSphere, bl.c, false)
		if st.glow > 0 {
			B.Billboard(p, 1.2*bl.s, sprSoft, alpha(bl.c, 0.35), true)
		}
	}
}
