package main

import (
	"image/color"
	"math"
	"sync"
)

// Terrain: the ground around the arena is no longer one flat sheet. Near
// the arena it stays level (props, decals and critters sit at y = -0.6),
// but it is "distressed" with patches of light and shade, and from a ring
// well clear of the props it rolls up into hills, dunes or rock walls that
// close the scene in. The hills are built once per world and lit per
// corner, so they read as real landforms from any camera angle.

type terrainStyle struct {
	amp      float64 // hill height at full rise
	start    float64 // distance from the arena where the rise begins
	span     float64 // distance over which it rises to full height
	scale    float64 // horizontal size of the landforms
	ridged   bool    // sharp crests (rock walls, dunes) instead of round hills
	distress float64 // strength of the light/shade patches on the flat
}

var terrainStyles = map[int]terrainStyle{
	WNeon:    {amp: 22, start: 34, span: 50, scale: 46, distress: 0.22},
	WMeadow:  {amp: 26, start: 30, span: 50, scale: 52, distress: 0.2},
	WCaverns: {amp: 34, start: 30, span: 45, scale: 30, ridged: true, distress: 0.25},
	WRuins:   {amp: 18, start: 34, span: 55, scale: 40, ridged: true, distress: 0.18},
	WMachine: {amp: 12, start: 34, span: 45, scale: 26, distress: 0.25},
	WDream:   {amp: 16, start: 64, span: 40, scale: 44, distress: 0.2}, // flat past the lake shore
}

// activeTerrain is the world the next ground draw belongs to (set by the
// scenery pass before it draws the ground).
var activeTerrain = -1

func hash2(x, z int, seed uint32) float64 {
	h := uint32(x)*374761393 + uint32(z)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0xffffff)
}

func valueNoise(x, z float64, seed uint32) float64 {
	x0, z0 := math.Floor(x), math.Floor(z)
	fx, fz := x-x0, z-z0
	ux, uz := fx*fx*(3-2*fx), fz*fz*(3-2*fz)
	ix, iz := int(x0), int(z0)
	a := hash2(ix, iz, seed)
	b := hash2(ix+1, iz, seed)
	c := hash2(ix, iz+1, seed)
	d := hash2(ix+1, iz+1, seed)
	return a + (b-a)*ux + (c-a)*uz + (a-b-c+d)*ux*uz
}

func fbm2(x, z float64, oct int, seed uint32) float64 {
	sum, amp, tot := 0.0, 1.0, 0.0
	for o := 0; o < oct; o++ {
		sum += valueNoise(x, z, seed+uint32(o)*101) * amp
		tot += amp
		amp *= 0.5
		x, z = x*2.03+17.1, z*2.03-9.4
	}
	return sum / tot
}

// arenaDist is how far (x, z) lies outside the arena and its lane.
func arenaDist(x, z float64) float64 {
	dx := math.Max(0, math.Abs(x)-(HalfW+4))
	dz := math.Max(0, math.Max(-14-z, z-(FarZ+8)))
	return math.Hypot(dx, dz)
}

// terrainY is the ground height at (x, z) for world w.
func terrainY(w int, x, z float64) float64 {
	st, ok := terrainStyles[w]
	if !ok {
		return -0.6
	}
	d := arenaDist(x, z)
	if d <= st.start {
		return -0.6
	}
	k := smoothstep(st.start, st.start+st.span, d)
	seed := uint32(w*7919 + 13)
	n := fbm2(x/st.scale, z/st.scale, 4, seed)
	if st.ridged {
		n = 1 - math.Abs(n*2-1)
		n = n * n
	}
	// small crumple on top so slopes are never glassy
	bump := (fbm2(x/5.5, z/5.5, 2, seed+7) - 0.5) * 0.9
	return -0.6 + k*(st.amp*(0.3+0.7*n)+bump)
}

// groundPatch is the light/shade of the flat ground at (x, z): broad
// worn patches plus a finer mottle, for a lived-in, uneven look.
func groundPatch(w int, x, z float64) float32 {
	st := terrainStyles[w]
	if st.distress == 0 {
		return 1
	}
	seed := uint32(w*31 + 5)
	p := fbm2(x/9, z/9, 3, seed)*0.7 + valueNoise(x/2.2, z/2.2, seed+3)*0.3
	return float32(1 + (p-0.5)*2*st.distress)
}

type terrainCell struct {
	p     [4]V3
	n     [4]V3
	patch [4]float32
	flat  bool
}

var (
	terrainMu    sync.Mutex
	terrainCache = map[int][]terrainCell{}
)

func terrainCells(w int, xs, zs []float64) []terrainCell {
	terrainMu.Lock()
	defer terrainMu.Unlock()
	if c, ok := terrainCache[w]; ok {
		return c
	}
	h := func(x, z float64) float64 { return terrainY(w, x, z) }
	norm := func(x, z float64) V3 {
		e := 1.5
		return V3{h(x-e, z) - h(x+e, z), 2 * e, h(x, z-e) - h(x, z+e)}.Norm()
	}
	var out []terrainCell
	for i := 0; i+1 < len(xs); i++ {
		for j := 0; j+1 < len(zs); j++ {
			x0, x1, z0, z1 := xs[i], xs[i+1], zs[j], zs[j+1]
			c := terrainCell{}
			corners := [4][2]float64{{x0, z1}, {x1, z1}, {x1, z0}, {x0, z0}}
			c.flat = true
			for k, q := range corners {
				y := h(q[0], q[1])
				c.p[k] = V3{q[0], y, q[1]}
				c.n[k] = norm(q[0], q[1])
				c.patch[k] = groundPatch(w, q[0], q[1])
				if y != -0.6 {
					c.flat = false
				}
			}
			out = append(out, c)
		}
	}
	terrainCache[w] = out
	return out
}

// terrainGrid is the cell layout: fine near the arena, coarse far away.
func terrainGrid() (xs, zs []float64) {
	xs = []float64{-320, -230, -170, -130, -100, -80, -66}
	for x := -58.0; x <= 58; x += 6 {
		xs = append(xs, x)
	}
	xs = append(xs, 66, 80, 100, 130, 170, 230, 320)
	zs = []float64{-300, -210, -150, -110, -80, -58, -42, -30, -20, -12, -6}
	for z := 0.0; z <= 96; z += 6 {
		zs = append(zs, z)
	}
	zs = append(zs, 106, 120, 140, 170, 210, 260, 330, 420)
	return
}

// drawTerrain draws world w's ground with texture t (tint already lit for
// a level surface).
func drawTerrain(B *Batch, w int, t *bakedTex, tint color.RGBA) {
	xs, zs := terrainGrid()
	cells := terrainCells(w, xs, zs)
	up := B.shade(V3{0, 1, 0})
	uv := func(p V3) [2]float32 {
		return [2]float32{float32(p.X / groundTile * float64(t.w)), float32(-p.Z / groundTile * float64(t.h))}
	}
	for i := range cells {
		c := &cells[i]
		var sh [4]float32
		for k := 0; k < 4; k++ {
			s := c.patch[k]
			if !c.flat {
				s *= B.shade(c.n[k]) / up
			}
			sh[k] = s
		}
		B.QuadUV(c.p, t.id, [4][2]float32{uv(c.p[0]), uv(c.p[1]), uv(c.p[2]), uv(c.p[3])}, tint, sh, false)
		if c.flat {
			B.lastQuad().depth = 1e9
		}
	}
}

func smoothstep(a, b, x float64) float64 {
	return smoothstep01(clamp01((x - a) / (b - a)))
}
