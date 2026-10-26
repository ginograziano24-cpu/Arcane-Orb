package main

import (
	"image/color"
	"math"
	"math/rand"
)

// A tiny triangle-mesh toolkit for stylised low-poly scenery: faceted
// spheres for foliage and boulders, lathed shapes for towers, trunks,
// crystals, balloons and pipes. Meshes are built once and cached.

type Tri struct {
	P   [3]V3
	Col color.RGBA
	Emi float32
}

type Mesh struct{ Tris []Tri }

type Mat3 [3]V3 // rows

func rotXYZ(rx, ry, rz float64) Mat3 {
	cx, sx := math.Cos(rx), math.Sin(rx)
	cy, sy := math.Cos(ry), math.Sin(ry)
	cz, sz := math.Cos(rz), math.Sin(rz)
	// R = Ry * Rx * Rz
	return Mat3{
		{cy*cz + sy*sx*sz, -cy*sz + sy*sx*cz, sy * cx},
		{cx * sz, cx * cz, -sx},
		{-sy*cz + cy*sx*sz, sy*sz + cy*sx*cz, cy * cx},
	}
}

func (m Mat3) Mul(v V3) V3 {
	return V3{m[0].Dot(v), m[1].Dot(v), m[2].Dot(v)}
}

// Draw renders the mesh with scale, rotation and translation.
func (b *Batch) Draw(m *Mesh, pos V3, scale V3, rot Mat3, tint color.RGBA, emissive float32) {
	eye := b.cam.Eye
	useTint := tint.A > 0
	for _, t := range m.Tris {
		var p [3]V3
		for i := 0; i < 3; i++ {
			q := V3{t.P[i].X * scale.X, t.P[i].Y * scale.Y, t.P[i].Z * scale.Z}
			p[i] = rot.Mul(q).Add(pos)
		}
		n := p[1].Sub(p[0]).Cross(p[2].Sub(p[0]))
		if n.Len() < 1e-9 {
			continue
		}
		n = n.Norm()
		c := p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3)
		if n.Dot(eye.Sub(c)) <= 0 {
			continue
		}
		col := t.Col
		if useTint {
			col = color.RGBA{uint8(int(col.R) * int(tint.R) / 255), uint8(int(col.G) * int(tint.G) / 255), uint8(int(col.B) * int(tint.B) / 255), col.A}
		}
		e := 1 + t.Emi + emissive
		b.Quad([4]V3{p[0], p[1], p[2], p[2]}, sprWhite, b.lit(col, n), [4]float32{e, e, e, e}, false)
	}
}

// ---------------------------------------------------------------- builders

var meshCache = map[string]*Mesh{}

func cached(key string, build func() *Mesh) *Mesh {
	if m, ok := meshCache[key]; ok {
		return m
	}
	m := build()
	meshCache[key] = m
	return m
}

// Icosphere builds a faceted sphere; jitter roughens it for foliage/rocks.
// col2 is used on lower faces for a darker underside.
func Icosphere(sub int, jitter float64, seed int64, col1, col2 color.RGBA) *Mesh {
	t := (1 + math.Sqrt(5)) / 2
	verts := []V3{{-1, t, 0}, {1, t, 0}, {-1, -t, 0}, {1, -t, 0}, {0, -1, t}, {0, 1, t}, {0, -1, -t}, {0, 1, -t}, {t, 0, -1}, {t, 0, 1}, {-t, 0, -1}, {-t, 0, 1}}
	for i := range verts {
		verts[i] = verts[i].Norm()
	}
	faces := [][3]int{{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11}, {1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9}, {4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1}}
	for s := 0; s < sub; s++ {
		mid := map[[2]int]int{}
		get := func(a, b int) int {
			k := [2]int{min(a, b), max(a, b)}
			if i, ok := mid[k]; ok {
				return i
			}
			verts = append(verts, verts[a].Add(verts[b]).Mul(0.5).Norm())
			mid[k] = len(verts) - 1
			return len(verts) - 1
		}
		var nf [][3]int
		for _, f := range faces {
			a, b, c := get(f[0], f[1]), get(f[1], f[2]), get(f[2], f[0])
			nf = append(nf, [3]int{f[0], a, c}, [3]int{f[1], b, a}, [3]int{f[2], c, b}, [3]int{a, b, c})
		}
		faces = nf
	}
	r := rand.New(rand.NewSource(seed))
	for i := range verts {
		verts[i] = verts[i].Mul(1 + (r.Float64()-0.5)*2*jitter)
	}
	m := &Mesh{}
	for _, f := range faces {
		c := verts[f[0]].Add(verts[f[1]]).Add(verts[f[2]]).Mul(1.0 / 3)
		col := mixc(col2, col1, math.Max(0, math.Min(1, c.Y*0.8+0.55)))
		col = mixc(col, lighten(col, 0.12), r.Float64()*0.6)
		m.Tris = append(m.Tris, Tri{P: [3]V3{verts[f[0]], verts[f[2]], verts[f[1]]}, Col: col})
	}
	return m
}

// Lathe revolves a profile of (radius, height) points around the Y axis.
// cols gives a colour per profile segment (last colour repeats).
func Lathe(profile [][2]float64, sides int, cols []color.RGBA, capTop bool) *Mesh {
	m := &Mesh{}
	ring := func(i int, a float64) V3 {
		return V3{math.Cos(a) * profile[i][0], profile[i][1], math.Sin(a) * profile[i][0]}
	}
	for i := 0; i+1 < len(profile); i++ {
		col := cols[min(i, len(cols)-1)]
		for s := 0; s < sides; s++ {
			a0 := float64(s) / float64(sides) * 2 * math.Pi
			a1 := float64(s+1) / float64(sides) * 2 * math.Pi
			p00, p01 := ring(i, a0), ring(i, a1)
			p10, p11 := ring(i+1, a0), ring(i+1, a1)
			if profile[i][0] > 1e-6 {
				m.Tris = append(m.Tris, Tri{P: [3]V3{p00, p10, p01}, Col: col})
			}
			if profile[i+1][0] > 1e-6 {
				m.Tris = append(m.Tris, Tri{P: [3]V3{p01, p10, p11}, Col: col})
			}
		}
	}
	if capTop {
		last := len(profile) - 1
		c := V3{0, profile[last][1], 0}
		col := cols[min(last, len(cols)-1)]
		for s := 0; s < sides; s++ {
			a0 := float64(s) / float64(sides) * 2 * math.Pi
			a1 := float64(s+1) / float64(sides) * 2 * math.Pi
			m.Tris = append(m.Tris, Tri{P: [3]V3{c, ring(last, a1), ring(last, a0)}, Col: col})
		}
	}
	return m
}

// Crystal is a hexagonal prism with a pointed tip.
func Crystal(col color.RGBA) *Mesh {
	m := Lathe([][2]float64{{0.0, 0}, {0.55, 0}, {0.6, 0.15}, {0.5, 0.75}, {0, 1}}, 6,
		[]color.RGBA{darken(col, 0.6), darken(col, 0.75), col, lighten(col, 0.25)}, false)
	for i := range m.Tris {
		m.Tris[i].Emi = 0.25
	}
	return m
}

// Merge appends transformed copies of meshes into one.
func Merge(parts ...struct {
	M   *Mesh
	Pos V3
	S   V3
	R   Mat3
}) *Mesh {
	out := &Mesh{}
	for _, p := range parts {
		for _, t := range p.M.Tris {
			var q [3]V3
			for i := 0; i < 3; i++ {
				v := V3{t.P[i].X * p.S.X, t.P[i].Y * p.S.Y, t.P[i].Z * p.S.Z}
				q[i] = p.R.Mul(v).Add(p.Pos)
			}
			out.Tris = append(out.Tris, Tri{P: q, Col: t.Col, Emi: t.Emi})
		}
	}
	return out
}

var identity = rotXYZ(0, 0, 0)

func v3(s float64) V3 { return V3{s, s, s} }
