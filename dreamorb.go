package main

import (
	"image/color"
	"math"
)

// The dream world's light orbs as real objects instead of flat glowing
// discs: a faceted crystal heart in a gilded chalice, circled by two
// slowly turning armillary rings, like a little orrery on every post and
// spire.

// faceOut appends a triangle wound so its normal points along out.
func faceOut(m *Mesh, p0, p1, p2, out V3, c color.RGBA, emi float32, fixed bool) {
	if p1.Sub(p0).Cross(p2.Sub(p0)).Dot(out) < 0 {
		p1, p2 = p2, p1
	}
	m.Tris = append(m.Tris, Tri{P: [3]V3{p0, p1, p2}, Col: c, Emi: emi, Fixed: fixed})
}

// orbCrystal is the glowing heart: a faceted gem, brighter at the top.
func orbCrystal() *Mesh {
	return cached("dreamOrbCrystal", func() *Mesh {
		src := Icosphere(1, 0.12, 77, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 210, 245, 255})
		m := &Mesh{}
		for i, t := range src.Tris {
			c := t.P[0].Add(t.P[1]).Add(t.P[2]).Mul(1.0 / 3)
			// stretch into a gem: taller than wide, pointed ends
			var p [3]V3
			for k := 0; k < 3; k++ {
				q := t.P[k]
				p[k] = V3{q.X * 0.8, q.Y * 1.25, q.Z * 0.8}
			}
			shade := uint8(225 + (i*37)%30)
			faceOut(m, p[0], p[1], p[2], c, color.RGBA{shade, shade, shade, 255}, 1.4, false)
		}
		return m
	})
}

// orbRing is one armillary band: a thin gold torus with raised beads.
func orbRing(key string, R float64) *Mesh {
	return cached("dreamOrbRing"+key, func() *Mesh {
		m := &Mesh{}
		const seg, sides = 28, 5
		r := 0.035
		gold := color.RGBA{236, 196, 104, 255}
		dark := color.RGBA{176, 128, 58, 255}
		pt := func(u, v float64) (V3, V3) {
			c := V3{math.Cos(u) * R, 0, math.Sin(u) * R}
			rad := V3{math.Cos(u), 0, math.Sin(u)}
			return c.Add(rad.Mul(math.Cos(v) * r)).Add(V3{0, math.Sin(v) * r, 0}), c
		}
		for i := 0; i < seg; i++ {
			u0 := float64(i) / seg * 2 * math.Pi
			u1 := float64(i+1) / seg * 2 * math.Pi
			col := gold
			if i%4 == 0 {
				col = dark
			}
			for j := 0; j < sides; j++ {
				v0 := float64(j) / sides * 2 * math.Pi
				v1 := float64(j+1) / sides * 2 * math.Pi
				a, ca := pt(u0, v0)
				b, _ := pt(u1, v0)
				c, _ := pt(u1, v1)
				d, _ := pt(u0, v1)
				mid := a.Add(b).Add(c).Add(d).Mul(0.25)
				out := mid.Sub(ca)
				faceOut(m, a, b, c, out, col, 0.15, true)
				faceOut(m, a, c, d, out, col, 0.15, true)
			}
		}
		// four beads (little planets) riding the band
		for k := 0; k < 4; k++ {
			u := float64(k)*math.Pi/2 + 0.4
			ctr := V3{math.Cos(u) * R, 0, math.Sin(u) * R}
			b := Icosphere(0, 0, int64(k), color.RGBA{255, 230, 160, 255}, color.RGBA{220, 170, 90, 255})
			for _, t := range b.Tris {
				var p [3]V3
				for q := 0; q < 3; q++ {
					p[q] = ctr.Add(t.P[q].Mul(0.075))
				}
				faceOut(m, p[0], p[1], p[2], p[0].Add(p[1]).Add(p[2]).Mul(1.0/3).Sub(ctr), t.Col, 0.25, true)
			}
		}
		return m
	})
}

// orbChalice is the gilded cup the crystal floats in.
func orbChalice() *Mesh {
	return cached("dreamOrbChalice", func() *Mesh {
		prof := [][2]float64{{0.2, 0}, {0.21, 0.04}, {0.09, 0.1}, {0.06, 0.24}, {0.1, 0.3}, {0.22, 0.36}, {0.32, 0.46}, {0.34, 0.52}}
		src := Lathe(prof, 12, []color.RGBA{{226, 184, 96, 255}, {200, 150, 70, 255}, {236, 198, 110, 255}}, false)
		m := &Mesh{}
		for _, t := range src.Tris {
			c := t.P[0].Add(t.P[1]).Add(t.P[2]).Mul(1.0 / 3)
			faceOut(m, t.P[0], t.P[1], t.P[2], V3{c.X, 0, c.Z}, t.Col, 0.1, true)
		}
		return m
	})
}

// drawDreamOrb draws a crystal orrery at p, scale s, tinted col.
func drawDreamOrb(B *Batch, p V3, s float64, col color.RGBA, t, ph float64, cup, both bool) {
	if cup {
		B.Draw(orbChalice(), p.Add(V3{0, -1.13 * s, 0}), v3(s), rotXYZ(0, ph, 0), color.RGBA{}, 0)
	}
	bob := math.Sin(t*1.3+ph) * 0.05 * s
	c := p.Add(V3{0, bob, 0})
	B.DrawLive(orbCrystal(), c, v3(0.3*s), rotXYZ(0, t*0.6+ph, 0), lighten(col, 0.45), 0.6)
	B.DrawLive(orbRing("a", 0.55), c, v3(s), rotXYZ(1.05, t*0.5+ph, 0.2), color.RGBA{}, 0)
	if both {
		B.DrawLive(orbRing("b", 0.47), c, v3(s), rotXYZ(-0.5, -t*0.8+ph*1.7, 1.25), color.RGBA{}, 0)
	}
}

// dreamLanternMesh is a ribbed paper lantern: a glowing body with gold
// caps top and bottom, a hanging loop and a tassel. The body is white so
// the lantern's colour comes from its tint; the gold keeps its own colour.
func dreamLanternMesh() *Mesh {
	return cached("dreamLantern2", func() *Mesh {
		m := &Mesh{}
		const sides, rings = 16, 10
		body := func(i, j int) V3 {
			y := -0.3 + 0.6*float64(i)/rings
			r := 0.3 * math.Sqrt(math.Max(0, 1-math.Pow(y/0.34, 2)))
			if j%2 == 1 {
				r *= 0.93 // the ribs of the paper
			}
			a := float64(j) / sides * 2 * math.Pi
			return V3{math.Cos(a) * r, y, math.Sin(a) * r}
		}
		for i := 0; i < rings; i++ {
			for j := 0; j < sides; j++ {
				a, b := body(i, j), body(i, j+1)
				c, d := body(i+1, j+1), body(i+1, j)
				mid := a.Add(c).Mul(0.5)
				out := V3{mid.X, 0, mid.Z}
				shade := uint8(255)
				if j%2 == 1 {
					shade = 232
				}
				col := color.RGBA{shade, shade, shade, 255}
				faceOut(m, a, b, c, out, col, 0.3, false)
				faceOut(m, a, c, d, out, col, 0.3, false)
			}
		}
		gold := []color.RGBA{{214, 168, 84, 255}, {170, 120, 52, 255}, {236, 196, 110, 255}}
		addLathe := func(prof [][2]float64, dy float64) {
			src := Lathe(prof, 12, gold, true)
			for _, t := range src.Tris {
				var p [3]V3
				for k := 0; k < 3; k++ {
					p[k] = t.P[k].Add(V3{0, dy, 0})
				}
				c := p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3)
				out := V3{c.X, 0, c.Z}
				if math.Abs(p[0].Y-p[1].Y) < 1e-6 && math.Abs(p[1].Y-p[2].Y) < 1e-6 {
					out = V3{0, 1, 0} // a cap
				}
				faceOut(m, p[0], p[1], p[2], out, t.Col, 0.1, true)
			}
		}
		addLathe([][2]float64{{0.17, 0}, {0.19, 0.02}, {0.19, 0.06}, {0.13, 0.08}}, 0.27)  // top cap
		addLathe([][2]float64{{0.13, 0}, {0.19, 0.02}, {0.19, 0.06}, {0.16, 0.08}}, -0.35) // bottom cap
		addLathe([][2]float64{{0.02, 0}, {0.03, 0.02}, {0.02, 0.1}}, 0.35)                 // the hanging loop's stem
		// tassel: a few hanging threads below the bottom cap
		red := color.RGBA{230, 70, 110, 255}
		for k := 0; k < 6; k++ {
			a := float64(k) / 6 * 2 * math.Pi
			x, z := math.Cos(a)*0.035, math.Sin(a)*0.035
			top := V3{x, -0.35, z}
			bot := V3{x * 2.2, -0.62, z * 2.2}
			side := V3{-math.Sin(a) * 0.015, 0, math.Cos(a) * 0.015}
			out := V3{x, 0, z}
			faceOut(m, top.Sub(side), top.Add(side), bot.Add(side), out, red, 0.2, true)
			faceOut(m, top.Sub(side), bot.Add(side), bot.Sub(side), out, red, 0.2, true)
		}
		return m
	})
}
