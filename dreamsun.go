package main

import (
	"image/color"
	"math"
)

// The Sunking: a low-poly 3D sun with a face, built from faceted primitives
// and flat-shaded like a late-90s CGI render. It hangs in the sky in its
// own fog-free pass (drawn before the world), turns to watch the camera,
// turns its flame rays, bobs, and puffs a little smoke. Very relaxed.

var white = color.RGBA{255, 255, 255, 255}

// sunFaceMesh is the Sunking's head: a thick disc of a ball whose surface
// is pitted and mottled like a molten, embossed rind (fine noise pushed in
// and out, speckled in deep red, ember orange and gold).
func sunFaceMesh() *Mesh {
	return cached("sunFace4", func() *Mesh {
		base := Icosphere(3, 0, 31, white, white)
		r := newFloraB(77).r
		disp := map[V3]V3{}
		shape := func(v V3) V3 {
			if p, ok := disp[v]; ok {
				return p
			}
			d := v.Norm()
			k := 1 + 0.035*(fbm3(d.Mul(9), 2, 5)-0.5) + 0.02*(fbm3(d.Mul(22), 1, 9)-0.5)
			p := d.Mul(k)
			disp[v] = p
			return p
		}
		m := &Mesh{}
		for _, t := range base.Tris {
			q := [3]V3{shape(t.P[0]), shape(t.P[2]), shape(t.P[1])} // wound outward
			c := q[0].Add(q[1]).Add(q[2]).Mul(1.0 / 3)
			n := fbm3(c.Mul(14), 2, 21)
			col := mixc(color.RGBA{150, 24, 10, 255}, color.RGBA{232, 70, 22, 255}, n*1.3)
			if fl := fbm3(c.Mul(26), 1, 33); fl > 0.68 {
				col = mixc(col, color.RGBA{255, 170, 50, 255}, (fl-0.68)*3) // gold flecks
			}
			col = mixc(col, darken(col, 0.8), r.Float64()*0.4)
			m.Tris = append(m.Tris, Tri{P: q, Col: col, Fixed: true})
		}
		return m
	})
}

// sunBall is a smooth white ball (wound outward) for the sculpted features.
func sunBall() *Mesh {
	return cached("sunBall2", func() *Mesh {
		m := &Mesh{}
		for _, t := range Icosphere(2, 0, 5, white, white).Tris {
			m.Tris = append(m.Tris, Tri{P: [3]V3{t.P[0], t.P[2], t.P[1]}, Col: t.Col})
		}
		return m
	})
}

// sunFlame is one straight ray: a flattened four-sided spike from a base
// of width 1 to a point at y=1, banded along its length in streaks of
// gold, flame orange and ember red, like a fire texture.
func sunFlame() *Mesh {
	return cached("sunFlame", func() *Mesh {
		fb := newFloraB(9)
		bands := 7
		corner := func(y float64, k int) V3 {
			w := 1 - y
			return [4]V3{{w, y, 0}, {0, y, w * 0.35}, {-w, y, 0}, {0, y, -w * 0.35}}[k]
		}
		pal := []color.RGBA{{255, 226, 80, 255}, {255, 160, 30, 255}, {240, 90, 20, 255}, {255, 200, 60, 255}}
		for bnd := 0; bnd < bands; bnd++ {
			y0 := float64(bnd) / float64(bands)
			y1 := float64(bnd+1) / float64(bands)
			for k := 0; k < 4; k++ {
				k1 := (k + 1) % 4
				a, b2 := corner(y0, k), corner(y0, k1)
				c, d := corner(y1, k1), corner(y1, k)
				out := a.Add(b2).Mul(0.5)
				out.Y = 0
				if out.Len() < 1e-6 {
					continue
				}
				out = out.Norm()
				col := pal[fb.r.Intn(len(pal))]
				if bnd == bands-1 {
					col = color.RGBA{255, 240, 140, 255}
				}
				fb.face(a, b2, c, out, col)
				fb.face(a, c, d, out, mixc(col, pal[(bnd+k)%len(pal)], 0.5))
			}
		}
		return fb.m
	})
}

// sunBasis turns local sun space (front = -z) to face from p toward the eye,
// then spins it by roll about its own axis.
func sunBasis(cam *Camera, p V3, roll float64) Mat3 {
	z := p.Sub(cam.Eye).Norm()
	x := cam.up.Cross(z).Norm()
	y := z.Cross(x)
	cr, sr := math.Cos(roll), math.Sin(roll)
	// columns of the result: x', y' rolled, then z
	xr := x.Mul(cr).Add(y.Mul(sr))
	yr := y.Mul(cr).Sub(x.Mul(sr))
	return Mat3{{xr.X, yr.X, z.X}, {xr.Y, yr.Y, z.Y}, {xr.Z, yr.Z, z.Z}}
}

func matMul(a, b Mat3) Mat3 {
	var r Mat3
	col := func(m Mat3, j int) V3 {
		return V3{[3]float64{m[0].X, m[0].Y, m[0].Z}[j], [3]float64{m[1].X, m[1].Y, m[1].Z}[j], [3]float64{m[2].X, m[2].Y, m[2].Z}[j]}
	}
	for i := 0; i < 3; i++ {
		r[i] = V3{a[i].Dot(col(b, 0)), a[i].Dot(col(b, 1)), a[i].Dot(col(b, 2))}
	}
	return r
}

// drawSunking adds the 3D sun to the world batch B, at screen point (sx, sy)
// with radius px pixels. It is drawn without fog and in its own light, and
// its polygons are sorted behind everything else in the scene.
func (g *Game) drawSunking(B *Batch, L *Level, sx, sy, px, t float64) {
	cam := B.cam
	fogN, fogF := B.fogNear, B.fogFar
	light, lc, ac := B.light, B.lightCol, B.ambCol
	first := len(B.quads)
	B.fogNear, B.fogFar = 1e6, 1e6+1
	B.SetLight(V3{-0.55, 0.6, -0.6}, [3]float32{1.15, 1.0, 0.9}, [3]float32{0.5, 0.36, 0.4})
	defer func() {
		// behind every prop and hill but in front of the level far ground
		// (which sorts at 1e9), keeping the sun's own order among its parts:
		// hills can rise in front of it, the flat ground never cuts it off
		for i := first; i < len(B.quads); i++ {
			B.quads[i].depth += 9e8
		}
		B.fogNear, B.fogFar = fogN, fogF
		B.light, B.lightCol, B.ambCol = light, lc, ac
	}()

	const D = 220.0
	p := cam.Eye.Add(cam.fwd.Mul(D)).Add(cam.right.Mul((sx - cam.W/2) * D / cam.f)).Add(cam.up.Mul((cam.H/2 - sy) * D / cam.f))
	R := px * D / cam.f / 1.85 // face radius (rays reach ~1.85 R)
	kick := math.Min(1, L.Shake*0.8)
	base := sunBasis(cam, p, math.Sin(t*0.5)*0.08)
	at := func(local V3) V3 { return base.Mul(local.Mul(R)).Add(p) }
	// moving parts skip the static-mesh bake cache (Draw would remember
	// every new placement): draw them straight through drawShared
	part := func(m *Mesh, pos, scale V3, rot Mat3, col color.RGBA, emi float32) {
		B.drawShared(m, at(pos), scale.Mul(R), matMul(base, rot), col, emi)
	}

	// long straight flame rays, alternating long and short, turning
	// lazily behind the head
	const nRays = 14
	spin := t * 0.06
	for i := 0; i < nRays; i++ {
		a := spin + float64(i)*2*math.Pi/nRays
		long := 0.95 + 0.35*float64(i%2) + 0.03*math.Sin(t*1.1+float64(i))
		c, sn := math.Cos(a), math.Sin(a)
		part(sunFlame(), V3{c * 0.72, sn * 0.72, 0.18}, V3{0.24, long, 0.24}, rotXYZ(0, 0, a-math.Pi/2), color.RGBA{}, 1.0)
	}
	// the head: a thick, pitted, molten disc
	part(sunFaceMesh(), V3{}, V3{1, 1, 0.62}, identity, color.RGBA{}, 0.35)
	surf := func(x, y float64) float64 { return -0.62*math.Sqrt(math.Max(0.02, 1-x*x-y*y)) - 0.01 }
	skin := color.RGBA{214, 58, 24, 255}
	shade := color.RGBA{120, 22, 10, 255}
	ball := func(x, y, dz, sx, sy, sz float64, rot Mat3, c color.RGBA, e float32) {
		part(sunBall(), V3{x, y, surf(x, y) - dz}, V3{sx, sy, sz}, rot, c, e)
	}

	// heavy features: puffy cheeks, a long broad nose, a soft chin
	for _, sx := range []float64{-1, 1} {
		ball(sx*0.42, -0.2, 0.0, 0.25, 0.22, 0.14, identity, skin, 0.12)
	}
	ball(0, 0.0, 0.06, 0.065, 0.28, 0.1, identity, skin, 0.12) // bridge
	ball(0, -0.19, 0.1, 0.14, 0.1, 0.11, identity, skin, 0.14) // tip
	for _, sx := range []float64{-1, 1} {                      // nostrils
		ball(sx*0.075, -0.225, 0.15, 0.05, 0.03, 0.03, identity, shade, 0)
	}
	ball(0, -0.68, 0.0, 0.2, 0.1, 0.1, identity, skin, 0.1) // chin

	// stoned eyes: wide, pinkish whites half buried under heavy drooping
	// lids, irises sunk low and drifting, puffy bags beneath
	drift := 0.35 * math.Sin(t*0.23)
	if len(L.Balls) > 0 {
		drift = 0.5*drift + 0.5*math.Max(-1, math.Min(1, L.Balls[0].X/HalfW))
	}
	droop := 0.03*math.Sin(t*0.4) - 0.03*kick // the lids sag and lift; a big hit perks him up a bit
	for _, side := range []float64{-1, 1} {
		ex, ey := side*0.31, 0.25
		tilt := rotXYZ(0, 0, -side*0.12)                                                  // outer corners drooping
		ball(ex, ey, 0.0, 0.26, 0.16, 0.08, tilt, color.RGBA{196, 56, 24, 255}, 0.08)     // socket
		ball(ex, ey, 0.025, 0.25, 0.15, 0.04, tilt, color.RGBA{246, 226, 214, 255}, 0.22) // white, kept flat under the lid
		ix, iy := ex+0.08*drift, ey-0.045
		ball(ix, iy, 0.07, 0.095, 0.095, 0.02, identity, color.RGBA{96, 46, 22, 255}, 0.1)                       // iris
		ball(ix, iy, 0.088, 0.05, 0.05, 0.012, identity, color.RGBA{8, 4, 4, 255}, 0)                            // pupil
		ball(ix-0.025, iy+0.02, 0.1, 0.016, 0.016, 0.008, identity, white, 1)                                    // glint
		ball(ex, ey+0.12-droop, 0.14, 0.27, 0.11, 0.1, tilt, skin, 0.16)                                         // upper lid, wholly in front of the eye
		part(unitBox(), V3{ex, ey + 0.012 - droop, surf(ex, ey) - 0.16}, V3{0.24, 0.012, 0.012}, tilt, shade, 0) // lid edge
		ball(ex, ey+0.23, 0.04, 0.25, 0.05, 0.08, rotXYZ(0, 0, -side*0.2), color.RGBA{190, 46, 20, 255}, 0.1)    // brow, relaxed
		ball(ex, ey-0.185, 0.1, 0.22, 0.05, 0.07, tilt, color.RGBA{200, 50, 22, 255}, 0.14)                      // bag, in front of the eye's lower rim
	}

	// full red lips in a lazy, lopsided smirk
	smirk := rotXYZ(0, 0, 0.12+0.03*math.Sin(t*0.3))
	ball(0.02, -0.4, 0.08, 0.3, 0.06, 0.08, smirk, color.RGBA{196, 20, 28, 255}, 0.25)
	ball(0.0, -0.48, 0.08, 0.26, 0.08, 0.09, smirk, color.RGBA{214, 30, 38, 255}, 0.3)
	part(unitBox(), V3{0.01, -0.44, surf(0, -0.44) - 0.16}, V3{0.27, 0.009, 0.02}, smirk, color.RGBA{60, 6, 8, 255}, 0)

	// slow lilac smoke curling out of the corner of his mouth
	for k := 0; k < 4; k++ {
		ph := math.Mod(t*0.22+float64(k)*0.25, 1)
		if ph < 0.05 {
			continue
		}
		mp := V3{0.32 + 0.25*ph + 0.06*math.Sin(t*0.9+float64(k)), -0.38 + 0.9*ph, surf(0.26, -0.38) - 0.25}
		a := 0.45 * math.Sin(ph*math.Pi)
		B.Billboard(at(mp), R*(0.16+0.4*ph), sprSoft, alpha(color.RGBA{220, 190, 255, 255}, a), false)
	}
	// he shines: layered additive bloom around the head and rays, breathing
	glowK := 0.85 + 0.15*math.Sin(t*1.1)
	B.Billboard(p, R*2.3, sprSoft, alpha(color.RGBA{255, 150, 50, 255}, 0.5*glowK), true)
	B.Billboard(p, R*4.4, sprSoft, alpha(color.RGBA{255, 110, 40, 255}, 0.28*glowK), true)
	B.Billboard(p, R*7.5, sprSoft, alpha(color.RGBA{255, 90, 60, 255}, 0.12*glowK), true)
}
