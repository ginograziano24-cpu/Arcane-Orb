package main

import "math"

var classes = []func() *Scene{guardian, berserker, arcanist, engineer, gambler, pyromancer, stormcaller}

// base fills in shared camera and lighting defaults.
func base(name string, glow V) *Scene {
	return &Scene{
		Name:    name,
		Mats:    map[int]Mat{},
		Key:     V{-0.75, 0.55, 0.42}.Norm(),
		KeyCol:  V{1.0, 0.8, 0.6},
		Fill:    V{0.8, 0.1, 0.6}.Norm(),
		FillCol: V{0.06, 0.07, 0.1},
		Rim:     V{0.65, 0.35, -0.8}.Norm(),
		RimCol:  glow.Mul(1.5),
		Rim2:    V{-0.75, 0.3, -0.65}.Norm(),
		Rim2Col: V{1.5, 1.05, 0.6},
		Amb:     V{0.035, 0.035, 0.045},
		Glow:    glow,
		GlowPos: [2]float64{0.12, 0.25},
		Mist:    glow.Mul(0.025).Add(V{0.016, 0.015, 0.017}),
		Bound:   1.25,
		CamPos:  V{0.3, 1.5, 3.6},
		CamTgt:  V{0, 1.51, 0},
		FOV:     24,
	}
}

// ---------------------------------------------------------------- Arcanist

func arcanist() *Scene {
	glow := hexc(0x8a5cff)
	s := base("arcane", glow)
	yaw := -0.32
	gold := hexc(0xc9a24a)
	s.Mats = map[int]Mat{
		1:  {Alb: hexc(0x302b5e), Rough: 0.85, Cloth: true, Grime: 0.4, Bump: 0.15, BumpF: 110}, // coat
		2:  {Alb: hexc(0x252148), Rough: 0.9, Cloth: true, Grime: 0.3, Bump: 0.15, BumpF: 110},  // hood/mantle
		3:  {Alb: gold, Rough: 0.28, Metal: 1, Grime: 0.3},
		4:  {Alb: V{0.002, 0.002, 0.003}, Rough: 1},
		5:  {Alb: V{}, Emit: V{6, 3.5, 16}},
		6:  {Alb: V{}, Emit: V{3.5, 2.4, 8}},
		7:  {Alb: V{}, Emit: V{1.6, 0.9, 4.2}},
		8:  {Alb: hexc(0x3a2a1e), Rough: 0.7, Grime: 0.5},
		9:  {Alb: hexc(0x2a2018), Rough: 0.5, Sheen: 0.4},
		11: {Alb: hexc(0x2d2b40), Rough: 0.8, Cloth: true, Grime: 0.45}, // inner tunic
		12: {Alb: hexc(0x4a2a16), Rough: 0.55, Grime: 0.3},              // belt leather
		13: {Alb: hexc(0x4a1420), Rough: 0.75, Cloth: true, Pattern: func(p V, a V) V { // embroidered stole
			f := math.Mod(p.Y*7+100, 1)
			if math.Abs(f-0.5) < 0.035 {
				return hexc(0x9a7a34)
			}
			return a
		}},
	}
	R := Arm{S: V{0.19, 1.52, 0.0}, E: V{0.33, 1.27, 0.08}, W: V{0.42, 1.2, 0.27}, Hand: V{0.44, 1.21, 0.32}}
	L := Arm{S: V{-0.19, 1.52, 0.0}, E: V{-0.3, 1.24, 0.05}, W: V{-0.31, 1.06, 0.16}, Hand: V{-0.31, 1.05, 0.2}, Fist: true}
	orbC := rotY(V{0.46, 1.36, 0.34}, yaw)
	headC := V{0, 1.77, 0.02}
	s.SDF = func(p V) (float64, int) {
		q := rotY(p, -yaw)
		h := newH()
		qz := V{q.X, q.Y, q.Z * 1.25}
		body := smin(sdCone(qz, V{0, 1.48, 0}, V{0, 0.05, 0.02}, 0.17, 0.33), torso(q, 1.05), 0.06)
		fold := folds(q, 11, 0.016, 1.2, 0.2)
		// outer coat, open down the front
		coat := body + fold
		wedge := math.Abs(q.X) - (0.03 + 0.11*smooth(1.35, 0.2, q.Y))
		coatC := smax(coat, -smax(wedge, -q.Z, 0.01), 0.012)
		h.add(coatC, 1)
		// inner tunic behind the opening
		tun := body + 0.02 + fold*0.4
		h.add(tun, 11)
		// embroidered stole hanging from the neck
		for _, sx := range []float64{-1, 1} {
			st := math.Max(math.Abs(coat-0.006)-0.006, math.Abs(q.X-sx*(0.07+0.1*smooth(1.4, 0.3, q.Y)))-0.034)
			st = smax(st, -q.Z, 0.01)
			st = smax(st, q.Y-1.5, 0.01)
			st = smax(st, 0.42-q.Y, 0.01)
			h.add(st, 13)
			edge := math.Max(math.Abs(coat-0.008)-0.0075, math.Abs(math.Abs(q.X-sx*(0.07+0.1*smooth(1.4, 0.3, q.Y)))-0.034)-0.005)
			edge = smax(edge, -q.Z, 0.01)
			edge = smax(edge, q.Y-1.5, 0.01)
			edge = smax(edge, 0.42-q.Y, 0.01)
			h.add(edge, 3)
		}
		// belt with buckle and a hanging tome
		belt := math.Max(math.Abs(body-0.012)-0.012, math.Abs(q.Y-1.06)-0.032)
		h.add(belt, 12)
		h.add(sdBox(q.Sub(V{0, 1.06, 0.15}), V{0.035, 0.03, 0.012}, 0.006), 3)
		h.add(sdBox(rotZ(q.Sub(V{-0.16, 0.92, 0.08}), 0.2), V{0.05, 0.065, 0.025}, 0.008), 12)
		h.add(sdBox(rotZ(q.Sub(V{-0.16, 0.92, 0.106}), 0.2), V{0.03, 0.04, 0.003}, 0.002), 3)
		// layered mantle with a scalloped edge
		mant := capelet(q, 1.66, 1.33, 0.29, 1.3, 14)
		sh, void, _ := cowl(q, headC, 0.08)
		h.add(smin(mant, sh, 0.03), 2)
		h.add(void, 4)
		mk, eyes := mask(q, headC)
		h.add(mk, 3)
		h.add(eyes, 5)
		// sleeves
		for _, a := range []Arm{R, L} {
			sl := limb(q, a, 0.062, 0.085)
			sl += folds(q.Sub(a.S), 8, 0.006, -0.4, 0.0)
			cuff := sdSphere(q.Sub(a.W.Lerp(a.Hand, 0.3)), 0.066)
			sl = smax(sl, -cuff, 0.01)
			h.blend(sl, 1, 0.03)
			cb := math.Max(math.Abs(sl-0.003)-0.004, math.Abs(q.Sub(a.W).Dot(a.W.Sub(a.E).Norm())+0.035)-0.012)
			h.add(cb, 3)
			h.add(hand(q, a), 9)
		}
		// gnarled staff
		sp := q
		sp.X += 0.012 * math.Sin(q.Y*9)
		sp.Z += 0.01 * math.Cos(q.Y*7)
		h.add(sdCone(sp, V{-0.33, 0.1, 0.2}, V{-0.3, 2.06, 0.15}, 0.024, 0.017), 8)
		top := V{-0.3, 2.1, 0.15}
		cres := sdTorus(rotX(q.Sub(top.Add(V{0, 0.07, 0})), math.Pi/2), 0.075, 0.011)
		cres = smax(cres, -(q.Y - top.Y - 0.03), 0.01)
		h.add(cres, 3)
		h.add(sdSphere(q.Sub(top.Add(V{0, 0.075, 0})), 0.026), 7)
		// floating orb and rune rings
		h.add(sdSphere(p.Sub(orbC), 0.07), 6)
		h.add(ring(p, orbC, 0.12, 0.0035, 1.1, 0.3), 7)
		h.add(ring(p, orbC, 0.16, 0.0025, 0.4, -0.7), 7)
		return h.d, h.m
	}
	s.Lights = []Light{
		{P: orbC, Col: V{1.3, 0.8, 3.2}, R: 0.2},
		{P: rotY(V{-0.3, 2.17, 0.15}, yaw), Col: V{0.5, 0.3, 1.2}, R: 0.12},
	}
	for i := 0; i < 28; i++ {
		a := float64(i) * 2.399
		r := 0.12 + 0.12*math.Mod(float64(i)*0.37, 1)
		pp := orbC.Add(V{math.Cos(a) * r, (math.Mod(float64(i)*0.61, 1) - 0.4) * 0.35, math.Sin(a) * r})
		s.Sparks = append(s.Sparks, Spark{P: pp, Col: V{1.2, 0.8, 3}, Size: 1.6 + 2*math.Mod(float64(i)*0.73, 1)})
	}
	s.Embers = V{0.6, 0.45, 1.4}
	return s
}
