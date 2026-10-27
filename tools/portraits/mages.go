package main

import "math"

// coat is a long coat body: torso plus a skirt flaring to the knees.
func coatBody(q V, broad, flare float64) float64 {
	qz := V{q.X, q.Y, q.Z * 1.25}
	sk := sdCone(qz, V{0, 1.3, 0}, V{0, 0.35, 0.02}, 0.19*broad, flare)
	return smin(sk, torso(q, broad), 0.06)
}

// ---------------------------------------------------------------- Engineer

func engineer() *Scene {
	glow := hexc(0xffa040)
	s := base("engineer", glow)
	s.RimCol = V{1.5, 0.9, 0.4}
	s.Rim2Col = V{0.45, 0.6, 0.9}
	yaw := -0.3
	s.Mats = map[int]Mat{
		1: {Alb: hexc(0x4a2e1c), Rough: 0.6, Grime: 0.5, Sheen: 0.3, Bump: 0.2, BumpF: 50}, // coat leather
		2: {Alb: hexc(0xb0843a), Rough: 0.3, Metal: 1, Grime: 0.5},                         // brass
		3: {Alb: hexc(0x4a4e54), Rough: 0.45, Metal: 1, Grime: 0.7},                        // steel
		4: {Alb: V{0.002, 0.002, 0.002}, Rough: 1},
		5: {Alb: V{}, Emit: V{7, 3.2, 0.8}},                                                   // goggles
		6: {Alb: V{}, Emit: V{9, 2.2, 0.6}},                                                   // cannon core
		7: {Alb: hexc(0x2b201a), Rough: 0.7, Cloth: true, Grime: 0.3, Bump: 0.15, BumpF: 110}, // hood/scarf
		8: {Alb: hexc(0x251812), Rough: 0.55, Grime: 0.3},                                     // straps
	}
	R := Arm{S: V{0.21, 1.5, 0}, E: V{0.32, 1.26, 0.08}, W: V{0.3, 1.14, 0.3}, Hand: V{0.29, 1.13, 0.35}, Fist: true}
	L := Arm{S: V{-0.21, 1.5, 0}, E: V{-0.31, 1.25, 0.0}, W: V{-0.3, 1.03, 0.12}, Hand: V{-0.29, 0.99, 0.15}, Fist: true}
	headC := V{0, 1.76, 0.02}
	coreP := V{0.3, 1.12, 0.45}
	pz := vary(Pose{Yaw: yaw, Roll: 0.11, Nod: 0.05, Turn: -0.12, Neck: V{0, 1.64, 0.01}})
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		cb := coatBody(q, 1.08, 0.3) + folds(q, 9, 0.01, 1.0, 0.4)
		// open front showing a dark waistcoat
		wedge := math.Abs(q.X) - (0.02 + 0.09*smooth(1.35, 0.4, q.Y))
		h.add(smax(cb, -smax(wedge, -q.Z, 0.01), 0.01), 1)
		h.add(coatBody(q, 1.0, 0.27)+0.012, 7)
		// high collar
		col := layer(sdCone(V{q.X, q.Y, q.Z * 1.2}, V{0, 1.7, -0.01}, V{0, 1.52, 0}, 0.11, 0.17), 0, 0.012)
		col = smax(col, -smax(math.Abs(q.X)-0.05, -q.Z, 0.01), 0.01)
		h.add(col, 1)
		// belts and pouches
		h.add(bandY(layer(torso(q, 1.08), 0.03, 0.012), q, 1.06, 1.11), 8)
		h.add(bandY(layer(torso(q, 1.08), 0.03, 0.01), q, 0.98, 1.02), 8)
		for _, x := range []float64{-0.15, 0.13} {
			h.add(sdBox(q.Sub(V{x, 1.0, 0.13}), V{0.04, 0.045, 0.025}, 0.012), 8)
			h.add(sdBox(q.Sub(V{x, 1.03, 0.155}), V{0.008, 0.008, 0.004}, 0.002), 2)
		}
		bandolier := math.Abs(q.Sub(V{0, 1.3, 0}).Dot(V{0.55, 1, 0}.Norm())) - 0.022
		h.add(math.Max(layer(torso(q, 1.08), 0.03, 0.01), bandolier), 8)
		// brass backpack with tank and pipes
		tank := sdCyl(q.Sub(V{-0.07, 1.36, -0.22}), 0.075, 0.17, 0.05)
		tank2 := sdCyl(q.Sub(V{0.08, 1.33, -0.22}), 0.06, 0.13, 0.04)
		h.add(math.Min(tank, tank2), 2)
		for i := 0; i < 3; i++ {
			y := 1.22 + float64(i)*0.1
			h.add(sdTorus(q.Sub(V{-0.07, y, -0.22}), 0.077, 0.006), 3)
		}
		pipe := chain(q, []V{{0.08, 1.48, -0.22}, {0.18, 1.56, -0.15}, {0.27, 1.52, -0.04}, {0.34, 1.36, 0.06}, {0.33, 1.2, 0.18}}, 0.016, 0.014)
		h.add(pipe, 2)
		// arms
		for _, a := range []Arm{R, L} {
			h.add(limb(q, a, 0.062, 0.052), 1)
			h.add(hand(q, a), 8)
		}
		// gauntlet cannon on the right forearm
		cd := R.W.Sub(R.E).Norm()
		cs := R.E.Lerp(R.W, 0.2)
		can := sdCone(q, cs, coreP, 0.07, 0.06)
		bore := sdCone(q, coreP.Sub(cd.Mul(0.06)), coreP.Add(cd.Mul(0.02)), 0.035, 0.035)
		can = smax(can, -bore, 0.005)
		h.add(can, 3)
		for i := 0; i < 3; i++ {
			c := cs.Lerp(coreP, 0.25+0.25*float64(i))
			ring := sdTorus(rotX(rotZ(q.Sub(c), -0.05), math.Pi/2-math.Atan2(cd.Y, cd.Z)), 0.072, 0.007)
			h.add(ring, 3)
		}
		h.add(sdSphere(q.Sub(coreP.Sub(cd.Mul(0.035))), 0.03), 6)
		// head: leather hood, brass goggles, respirator
		o := q.Sub(headC)
		sh, void, _ := cowl(q, headC, 0.0)
		h.add(sh, 7)
		h.add(void, 4)
		face := sdEllipsoid(o.Sub(V{0, -0.01, 0.03}), V{0.07, 0.09, 0.07})
		h.add(face, 7)
		for _, sx := range []float64{-0.034, 0.034} {
			g := sdCyl(rotX(o.Sub(V{sx, 0.02, 0.085}), math.Pi/2), 0.028, 0.018, 0.006)
			h.add(g, 2)
			h.add(sdCyl(rotX(o.Sub(V{sx, 0.02, 0.1}), math.Pi/2), 0.021, 0.006, 0.003), 5)
		}
		h.add(sdBox(o.Sub(V{0, 0.02, 0.09}), V{0.012, 0.006, 0.01}, 0.004), 2)
		strap := bandY(layer(sdEllipsoid(o.Sub(V{0, 0.02, 0.0}), V{0.12, 0.13, 0.13}), 0, 0.006), q, headC.Y+0.008, headC.Y+0.032)
		h.add(strap, 8)
		resp := sdCone(o, V{0, -0.05, 0.06}, V{0, -0.07, 0.12}, 0.045, 0.03)
		h.add(resp, 3)
		for _, sx := range []float64{-1, 1} {
			h.add(sdCyl(rotZ(o.Sub(V{sx * 0.055, -0.07, 0.08}), sx*1.2), 0.022, 0.025, 0.006), 2)
		}
		return h.d, h.m
	}
	s.Lights = []Light{
		{P: rotY(coreP.Add(V{0, -0.02, 0.1}), yaw), Col: V{1.0, 0.3, 0.08}, R: 0.12},
		{P: rotY(headC.Add(V{0, 0.02, 0.13}), yaw), Col: V{0.9, 0.45, 0.1}, R: 0.07},
	}
	for i := 0; i < 16; i++ {
		f := float64(i)
		c := rotY(coreP.Add(V{0.02 * math.Sin(f*3), -0.04 - f*0.03, 0.05 + 0.02*math.Cos(f*2)}), yaw)
		s.Sparks = append(s.Sparks, Spark{P: c, Col: V{2.5, 1.0, 0.3}, Size: 1.4 + math.Mod(f*0.7, 1.5)})
	}
	s.Embers = V{1.4, 0.8, 0.3}
	s.Anchors = []Anchor{
		{Kind: "core", P: rotY(coreP.Add(V{0, -0.01, 0.03}), yaw), Col: V{1, 0.4, 0.1}, Size: 0.8},
		{Kind: "eye", P: pz.head(headC.Add(V{-0.034, 0.02, 0.105})), Col: V{1, 0.6, 0.2}, Size: 0.55},
		{Kind: "eye", P: pz.head(headC.Add(V{0.034, 0.02, 0.105})), Col: V{1, 0.6, 0.2}, Size: 0.55},
		{Kind: "steam", P: pz.body(V{0.08, 1.48, -0.22}), Col: V{0.7, 0.65, 0.6}, Size: 0.8},
	}
	return s
}

// ---------------------------------------------------------------- Gambler

func gambler() *Scene {
	glow := hexc(0x3fd69a)
	s := base("lucky", glow)
	s.Rim2Col = V{1.5, 1.2, 0.6}
	yaw := -0.3
	gold := hexc(0xd6ad4c)
	s.Mats = map[int]Mat{
		1: {Alb: hexc(0x14261f), Rough: 0.75, Cloth: true, Grime: 0.3, Bump: 0.15, BumpF: 110}, // coat
		2: {Alb: gold, Rough: 0.25, Metal: 1, Grime: 0.15},
		3: {Alb: hexc(0x0f0d0c), Rough: 0.65, Sheen: 0.4}, // hat felt
		4: {Alb: V{0.002, 0.002, 0.002}, Rough: 1},
		5: {Alb: V{}, Emit: V{6, 4.5, 1.2}}, // eyes
		6: {Alb: V{}, Emit: V{6, 4.2, 1.2}}, // coin
		7: {Alb: hexc(0xe8e0cc), Rough: 0.5, Pattern: func(p V, a V) V { // cards
			if math.Mod(math.Abs(p.Y*60), 1) < 0.15 {
				return hexc(0x8c1c24)
			}
			return a
		}},
		8:  {Alb: hexc(0x0d1a15), Rough: 0.85, Cloth: true, Bump: 0.2, BumpF: 110}, // scarf
		9:  {Alb: hexc(0x1d1712), Rough: 0.5, Sheen: 0.4},
		10: {Alb: hexc(0x7a1622), Rough: 0.6, Sheen: 0.5}, // cravat
	}
	R := Arm{S: V{0.2, 1.5, 0}, E: V{0.34, 1.33, 0.12}, W: V{0.33, 1.52, 0.27}, Hand: V{0.32, 1.57, 0.3}, Curl: 0.8}
	L := Arm{S: V{-0.2, 1.5, 0}, E: V{-0.3, 1.24, 0.06}, W: V{-0.2, 1.14, 0.26}, Hand: V{-0.17, 1.15, 0.3}, Curl: 0.5}
	headC := V{0, 1.76, 0.02}
	coinP := V{0.33, 1.72, 0.32}
	pz := vary(Pose{Yaw: yaw, Roll: -0.13, Nod: 0.06, Turn: 0.1, Neck: V{0, 1.64, 0.01}})
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		cb := coatBody(q, 1.02, 0.31) + folds(q, 10, 0.012, 1.0, 0.35)
		h.add(cb, 1)
		// lapels and gold buttons
		for _, sx := range []float64{-1, 1} {
			lap := bandY(layer(cb, 0.008, 0.006), q, 1.2, 1.55)
			lap = smax(lap, math.Abs(q.X-sx*(0.05+0.25*(q.Y-1.2)))-0.035, 0.005)
			h.add(smax(lap, -q.Z, 0.01), 1)
		}
		for i := 0; i < 4; i++ {
			y := 1.12 + float64(i)*0.08
			h.add(sdSphere(q.Sub(V{0.06, y, 0.135 - 0.01*float64(i)}), 0.012), 2)
		}
		// high collar and scarf hiding the lower face
		scarf := sdCone(V{q.X, q.Y, q.Z * 1.1}, V{0, 1.67, 0.01}, V{0, 1.55, 0.0}, 0.08, 0.14) + folds(q, 12, 0.005, 1.7, 1.55)
		h.add(scarf, 8)
		tail := sdBox(rotZ(q.Sub(V{0.0, 1.47, 0.14}), 0.05), V{0.022, 0.07, 0.008}, 0.006)
		h.add(tail, 10)
		// arms
		for _, a := range []Arm{R, L} {
			h.add(limb(q, a, 0.06, 0.055), 1)
			h.add(bandY(layer(limb(q, a, 0.06, 0.055), 0.008, 0.006), q, a.W.Y-0.03, a.W.Y+0.01), 2)
			h.add(hand(q, a), 9)
		}
		// fan of cards in the left hand
		for i := 0; i < 4; i++ {
			c := rotZ(rotY(q.Sub(L.Hand.Add(V{0, 0.03, 0.02})), 0.3), 0.5-float64(i)*0.32)
			h.add(sdBox(c.Sub(V{0, 0.09, 0.003 * float64(i)}), V{0.042, 0.064, 0.0015}, 0.005), 7)
		}
		// head and wide-brimmed hat
		o := q.Sub(headC)
		h.add(sdEllipsoid(o.Sub(V{0, -0.01, 0.01}), V{0.08, 0.1, 0.09}), 4)
		for _, sx := range []float64{-0.03, 0.03} {
			h.add(sdEllipsoid(o.Sub(V{sx, 0.0, 0.085}), V{0.013, 0.005, 0.005}), 5)
		}
		ho := rotX(rotZ(o, -0.08), -0.12)
		brim := sdCyl(ho.Sub(V{0, 0.075, 0}), 0.25, 0.006, 0.005)
		brim += 0.02*(ho.X*ho.X+ho.Z*ho.Z)*4 - 0.012 // brim curls up at the edge
		crown := sdCyl(ho.Sub(V{0, 0.14, 0}), 0.11, 0.065, 0.04)
		crown = smax(crown, -sdEllipsoid(ho.Sub(V{0, 0.23, 0}), V{0.03, 0.04, 0.12}), 0.02) // pinch
		h.add(smin(brim, crown, 0.02), 3)
		h.add(bandY(layer(crown, 0.002, 0.005), ho, 0.085, 0.11), 2)
		// a long feather tucked into the hat band
		fpts := []V{{0.09, 0.1, -0.03}, {0.15, 0.16, -0.07}, {0.19, 0.25, -0.12}, {0.18, 0.33, -0.17}}
		fe := chain(ho, fpts, 0.004, 0.002)
		vane := chain(ho, fpts, 0.022, 0.006)
		vane = math.Max(vane, math.Abs(ho.Sub(V{0.15, 0.2, -0.1}).Dot(V{0.3, -0.2, 0.93}.Norm()))-0.003)
		vane += 0.002 * math.Sin(ho.Y*400)
		h.add(math.Min(fe, vane), 10)
		// the coin
		h.add(sdCyl(rotX(rotZ(p.Sub(rotY(coinP, yaw)), 0.4), 1.2), 0.03, 0.004, 0.002), 6)
		return h.d, h.m
	}
	cp := rotY(coinP, yaw)
	s.Lights = []Light{{P: cp, Col: V{1.2, 0.9, 0.25}, R: 0.15}}
	// orbiting coins
	for i := 0; i < 22; i++ {
		f := float64(i)
		a := f * 0.9
		r := 0.42 + 0.12*math.Sin(f*1.7)
		pp := V{math.Cos(a) * r, 1.1 + f*0.045, math.Sin(a)*r*0.6 - 0.05}
		s.Sparks = append(s.Sparks, Spark{P: rotY(pp, yaw), Col: V{2.2, 1.6, 0.4}, Size: 2 + math.Mod(f*0.37, 1)*2})
	}
	s.Sparks = append(s.Sparks, Spark{P: cp, Col: V{3, 2.2, 0.6}, Size: 9})
	s.Embers = V{1.4, 1.1, 0.4}
	s.Anchors = []Anchor{
		{Kind: "coin", P: cp, Col: V{1, 0.8, 0.3}, Size: 1},
		{Kind: "eye", P: pz.head(headC.Add(V{-0.03, 0.0, 0.09})), Col: V{1, 0.8, 0.3}, Size: 0.35},
		{Kind: "eye", P: pz.head(headC.Add(V{0.03, 0.0, 0.09})), Col: V{1, 0.8, 0.3}, Size: 0.35},
		{Kind: "glint", P: pz.body(V{0.06, 1.2, 0.14}), Col: V{1, 0.9, 0.6}, Size: 0.4},
	}
	return s
}

// ---------------------------------------------------------------- Pyromancer

func pyromancer() *Scene {
	glow := hexc(0xff6a1a)
	s := base("pyro", glow)
	s.Rim2Col = V{1.6, 0.7, 0.25}
	yaw := -0.3
	s.Mats = map[int]Mat{
		1: {Alb: hexc(0x2a201c), Rough: 0.9, Cloth: true, Grime: 0.5, Bump: 0.2, BumpF: 110},
		2: {Alb: hexc(0x5a1a10), Rough: 0.85, Cloth: true, Grime: 0.4},
		3: {Alb: hexc(0xe6dccb), Rough: 0.35, Grime: 0.35}, // porcelain mask
		4: {Alb: V{0.002, 0.002, 0.002}, Rough: 1},
		5: {Alb: V{}, Emit: V{10, 3.5, 0.6}},
		6: {Alb: V{}, Emit: V{14, 5, 0.9}},
		7: {Alb: hexc(0x2a1c14), Rough: 0.6, Sheen: 0.3}, // wraps
		8: {Alb: hexc(0x8a6a3a), Rough: 0.35, Metal: 1, Grime: 0.5},
	}
	R := Arm{S: V{0.2, 1.5, 0}, E: V{0.36, 1.32, 0.04}, W: V{0.45, 1.4, 0.2}, Hand: V{0.48, 1.42, 0.24}, Flip: true, Curl: 0.9}
	L := Arm{S: V{-0.2, 1.5, 0}, E: V{-0.32, 1.25, 0.02}, W: V{-0.36, 1.05, 0.14}, Hand: V{-0.37, 1.0, 0.17}, Curl: 0.7}
	headC := V{0, 1.76, 0.02}
	fireP := V{0.46, 1.56, 0.26}
	pz := vary(Pose{Yaw: yaw, Roll: 0.06, Nod: 0.06, Turn: 0.3, Neck: V{0, 1.64, 0.01}})
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		qz := V{q.X, q.Y, q.Z * 1.25}
		body := smin(sdCone(qz, V{0, 1.48, 0}, V{0, 0.05, 0.02}, 0.17, 0.3), torso(q, 1.02), 0.06)
		robe := body + folds(q, 10, 0.016, 1.25, 0.2)
		// tattered hem: ragged teeth
		a := math.Atan2(q.X, q.Z)
		robe = smax(robe, 0.55+0.08*math.Abs(math.Sin(a*7+1))*fbm(V{a * 3, 0, 0}, 2)-q.Y, 0.01)
		wedge := math.Abs(q.X) - (0.02 + 0.1*smooth(1.35, 0.3, q.Y))
		h.add(smax(robe, -smax(wedge, -q.Z, 0.01), 0.012), 1)
		h.add(body+0.018, 2)
		// wrapped sash with a brass clasp
		h.add(bandY(layer(body, 0.012, 0.016), q, 1.02, 1.13), 7)
		h.add(sdCyl(rotX(q.Sub(V{0, 1.075, 0.155}), math.Pi/2), 0.03, 0.008, 0.004), 8)
		mant := capelet(q, 1.66, 1.34, 0.28, 1.3, 12)
		mant = smax(mant, 1.36+0.06*math.Abs(math.Sin(a*6))-q.Y, 0.01)
		sh, void, _ := cowlW(q, headC, 0.1, 1.25)
		h.add(smin(mant, sh, 0.03), 1)
		h.add(void, 4)
		for _, sx := range []float64{-0.03, 0.03} {
			h.add(sdEllipsoid(q.Sub(headC.Add(V{sx, 0.0, 0.1})), V{0.014, 0.006, 0.006}), 5)
		}
		for _, ar := range []Arm{R, L} {
			sl := limb(q, ar, 0.06, 0.07) + folds(q.Sub(ar.S), 8, 0.006, -0.4, 0)
			cuff := sdSphere(q.Sub(ar.W.Lerp(ar.Hand, 0.2)), 0.055)
			sl = smax(sl, -cuff, 0.01)
			h.add(sl, 1)
			h.add(limb(q, Arm{S: ar.E.Lerp(ar.W, 0.6), E: ar.W, W: ar.W}, 0.04, 0.035), 7)
			h.add(hand(q, ar), 7)
		}
		// fireball: turbulent emissive core
		fp := p.Sub(rotY(fireP, yaw))
		fb := fp.Len() - 0.075 - 0.025*fbm(fp.Mul(18).Add(V{0, -2, 0}), 3)
		h.add(fb*0.7, 6)
		return h.d, h.m
	}
	fw := rotY(fireP, yaw)
	s.Lights = []Light{
		{P: fw, Col: V{4, 1.4, 0.3}, R: 0.2},
		{P: rotY(L.Hand, yaw), Col: V{0.8, 0.25, 0.05}, R: 0.08},
	}
	// flames licking upward from the fireball and embers from the left hand
	for i := 0; i < 160; i++ {
		f := float64(i)
		t := math.Mod(f*0.618, 1)
		ang := f * 2.4
		sway := 0.03 * math.Sin(t*9+f*0.1)
		r := 0.075 * (1 - t*t) * (0.4 + 0.6*math.Mod(f*0.377, 1))
		pp := fw.Add(V{math.Cos(ang)*r + sway, t*0.36 + 0.01, math.Sin(ang) * r})
		col := V{3.6, 1.6, 0.35}.Lerp(V{1.8, 0.3, 0.05}, t)
		s.Sparks = append(s.Sparks, Spark{P: pp, Col: col.Mul(0.55 * (1 - 0.7*t)), Size: 9 * (1 - 0.75*t)})
	}
	lh := rotY(L.Hand, yaw)
	for i := 0; i < 14; i++ {
		f := float64(i)
		s.Sparks = append(s.Sparks, Spark{P: lh.Add(V{0.03 * math.Sin(f*2), f * 0.03, 0.03 * math.Cos(f*3)}), Col: V{2.4, 0.8, 0.15}, Size: 1.5 + math.Mod(f*0.6, 1.4)})
	}
	s.Embers = V{2.0, 0.7, 0.15}
	s.Anchors = []Anchor{
		{Kind: "fire", P: fw, Col: V{1, 0.5, 0.12}, Size: 1},
		{Kind: "ember", P: lh, Col: V{1, 0.4, 0.1}, Size: 0.6},
		{Kind: "eye", P: pz.head(headC.Add(V{-0.03, 0.0, 0.08})), Col: V{1, 0.45, 0.1}, Size: 0.4},
		{Kind: "eye", P: pz.head(headC.Add(V{0.03, 0.0, 0.08})), Col: V{1, 0.45, 0.1}, Size: 0.4},
	}
	return s
}

// ---------------------------------------------------------------- Stormcaller

func stormcaller() *Scene {
	glow := hexc(0x5ec4ff)
	s := base("storm", glow)
	s.Rim2Col = V{0.7, 0.8, 1.2}
	yaw := -0.28
	s.Mats = map[int]Mat{
		1: {Alb: hexc(0x1d2630), Rough: 0.85, Cloth: true, Grime: 0.4, Bump: 0.2, BumpF: 110},
		2: {Alb: hexc(0x2e4458), Rough: 0.8, Cloth: true, Grime: 0.3},
		3: {Alb: hexc(0x8c96a4), Rough: 0.3, Metal: 1, Grime: 0.5},
		4: {Alb: V{0.002, 0.002, 0.003}, Rough: 1},
		5: {Alb: V{}, Emit: V{1.6, 4.5, 10}},
		6: {Alb: V{}, Emit: V{2.5, 6, 12}},
		7: {Alb: hexc(0x2a2018), Rough: 0.65, Grime: 0.5},
		8: {Alb: hexc(0x3a3f48), Rough: 0.6, Cloth: true},
	}
	R := Arm{S: V{0.2, 1.5, 0}, E: V{0.34, 1.32, 0.06}, W: V{0.34, 1.5, 0.22}, Hand: V{0.34, 1.54, 0.25}, Fist: true}
	L := Arm{S: V{-0.2, 1.5, 0}, E: V{-0.33, 1.3, 0.1}, W: V{-0.4, 1.24, 0.28}, Hand: V{-0.43, 1.25, 0.33}, Curl: 0.2}
	headC := V{0, 1.76, 0.02}
	staffBot := V{0.36, 0.2, 0.3}
	staffTop := V{0.33, 2.06, 0.2}
	pz := vary(Pose{Yaw: yaw, Roll: -0.05, Nod: -0.1, Turn: 0.22, Neck: V{0, 1.64, 0.01}})
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		qz := V{q.X, q.Y, q.Z * 1.25}
		body := smin(sdCone(qz, V{0, 1.48, 0}, V{0, 0.05, 0.02}, 0.17, 0.3), torso(q, 1.04), 0.06)
		h.add(body+folds(q, 9, 0.012, 1.2, 0.2), 1)
		h.add(bandY(layer(body, 0.012, 0.012), q, 1.04, 1.1), 7)
		// wind-swept cloak streaming to the left
		cq := q
		wind := smooth(1.5, 0.3, q.Y)
		cq.X -= wind * 0.12 * (1 + 0.2*math.Sin(q.Y*6))
		cq.Z += wind * 0.03
		cl := cloak(cq, V{0, 1.58, -0.03}, V{0, 0.2, -0.2}, 0.21, 0.5, 0.02, 7)
		h.add(cl, 2)
		mant := capelet(q, 1.66, 1.36, 0.27, 1.3, 10)
		sh, void, _ := cowl(q, headC, 0.12)
		h.add(smin(mant, sh, 0.03), 2)
		h.add(void, 4)
		// cloth face wrap, glowing eyes above it
		wrap := sdEllipsoid(q.Sub(headC.Add(V{0, -0.05, 0.035})), V{0.065, 0.05, 0.062})
		h.add(wrap+0.003*math.Sin(q.Y*300), 8)
		for _, sx := range []float64{-0.03, 0.03} {
			h.add(sdEllipsoid(q.Sub(headC.Add(V{sx, 0.012, 0.1})), V{0.014, 0.005, 0.006}), 5)
		}
		for _, a := range []Arm{R, L} {
			h.add(limb(q, a, 0.06, 0.06)+folds(q.Sub(a.S), 8, 0.005, -0.4, 0), 1)
			h.add(bandY(layer(limb(q, a, 0.06, 0.06), 0.006, 0.006), q, a.W.Y-0.04, a.W.Y+0.02), 3)
			h.add(hand(q, a), 7)
		}
		// staff with a forked iron crown
		h.add(sdCone(q, staffBot, staffTop, 0.022, 0.018), 7)
		for _, sx := range []float64{-1, 1} {
			pts := []V{staffTop, staffTop.Add(V{sx * 0.06, 0.05, 0}), staffTop.Add(V{sx * 0.08, 0.14, 0}), staffTop.Add(V{sx * 0.05, 0.22, 0})}
			h.add(chain(q, pts, 0.013, 0.004), 3)
		}
		h.add(sdSphere(q.Sub(staffTop.Add(V{0, 0.1, 0})), 0.03), 6)
		for i := 0; i < 3; i++ {
			h.add(sdTorus(q.Sub(staffTop.Add(V{0, -0.05 - 0.04*float64(i), 0})), 0.024, 0.005), 3)
		}
		return h.d, h.m
	}
	top := rotY(staffTop.Add(V{0, 0.1, 0}), yaw)
	lh := rotY(L.Hand.Add(V{0.0, 0, 0.04}), yaw)
	s.Lights = []Light{
		{P: top, Col: V{0.6, 1.6, 3.6}, R: 0.25},
		{P: lh, Col: V{0.4, 1.0, 2.4}, R: 0.12},
	}
	c := V{0.6, 1.3, 2.6}
	s.Arcs = []Arc{
		{A: top, B: top.Add(V{-0.35, 0.2, -0.1}), Col: c, Width: 1.6, Seed: 3, Kids: 3},
		{A: top, B: top.Add(V{0.25, 0.25, 0.0}), Col: c, Width: 1.4, Seed: 7, Kids: 2},
		{A: top, B: lh, Col: c, Width: 1.5, Seed: 11, Kids: 3},
		{A: lh, B: lh.Add(V{-0.2, -0.35, 0.1}), Col: c, Width: 1.2, Seed: 19, Kids: 2},
	}
	s.Sparks = append(s.Sparks, Spark{P: top, Col: V{1.5, 3, 6}, Size: 10}, Spark{P: lh, Col: V{1, 2, 4}, Size: 7})
	s.Embers = V{0.5, 0.9, 1.6}
	s.Anchors = []Anchor{
		{Kind: "bolt", P: top, Col: V{0.55, 0.85, 1}, Size: 1},
		{Kind: "spark", P: lh, Col: V{0.55, 0.85, 1}, Size: 0.7},
		{Kind: "eye", P: pz.head(headC.Add(V{-0.03, 0.008, 0.08})), Col: V{0.5, 0.85, 1}, Size: 0.4},
		{Kind: "eye", P: pz.head(headC.Add(V{0.03, 0.008, 0.08})), Col: V{0.5, 0.85, 1}, Size: 0.4},
	}
	s.GlowPos = [2]float64{-0.1, 0.4}
	return s
}
