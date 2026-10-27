package main

import "math"

// ---------------------------------------------------------------- Guardian

func guardian() *Scene {
	glow := hexc(0xffd27a)
	s := base("guardian", glow)
	s.RimCol = V{1.6, 1.3, 0.8}
	s.Rim2Col = V{0.5, 0.65, 1.1}
	s.KeyCol = V{1.0, 0.82, 0.62}
	yaw := 0.28
	steel := hexc(0x8d939c)
	s.Mats = map[int]Mat{
		1: {Alb: steel, Rough: 0.3, Metal: 1, Grime: 0.45},
		2: {Alb: hexc(0x1d2d5c), Rough: 0.85, Cloth: true, Grime: 0.35, Bump: 0.15, BumpF: 110},
		3: {Alb: hexc(0xc9a24a), Rough: 0.28, Metal: 1, Grime: 0.2},
		4: {Alb: V{0.002, 0.002, 0.003}, Rough: 1},
		5: {Alb: V{}, Emit: V{5, 3.6, 1.6}},
		6: {Alb: hexc(0x3a281c), Rough: 0.6, Grime: 0.3},
		7: {Alb: hexc(0x5a5f68), Rough: 0.45, Metal: 1, Grime: 0.6, Bump: 0.6, BumpF: 320}, // mail
		8: {Alb: hexc(0x7a1a1e), Rough: 0.8, Cloth: true},
	}
	R := Arm{S: V{0.22, 1.5, 0}, E: V{0.3, 1.24, -0.02}, W: V{0.33, 1.02, 0.08}, Hand: V{0.34, 0.98, 0.11}, Fist: true}
	L := Arm{S: V{-0.22, 1.5, 0}, E: V{-0.3, 1.27, 0.12}, W: V{-0.17, 1.2, 0.3}, Hand: V{-0.13, 1.21, 0.33}, Fist: true}
	headC := V{0, 1.76, 0.02}
	pz := Pose{Yaw: yaw, Roll: 0.05, Nod: 0.07, Turn: -0.18, Neck: V{0, 1.64, 0.01}}
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		tor := torso(q, 1.18)
		// mail beneath, plate over it
		h.add(tor+0.006, 7)
		plate := layer(tor, 0.02, 0.012)
		plate = bandY(plate, q, 1.12, 1.56)
		plate = smax(plate, -q.Z-0.02, 0.01)
		ridge := sdCone(q, V{0, 1.52, 0.135}, V{0, 1.16, 0.12}, 0.012, 0.008)
		h.add(smin(plate, ridge, 0.01), 1)
		// faulds (skirt plates)
		for i := 0; i < 3; i++ {
			y := 1.1 - float64(i)*0.07
			f := layer(sdCone(V{q.X, q.Y, q.Z * 1.2}, V{0, 1.15, 0}, V{0, 0.7, 0}, 0.17, 0.25), 0.01+0.008*float64(i), 0.009)
			h.add(bandY(f, q, y-0.065, y), 1)
		}
		h.add(legs(q, 0.02), 7)
		// tabard
		tab := sdBox(q.Sub(V{0, 0.78, 0.17}), V{0.1, 0.32, 0.006}, 0.004)
		tab += 0.004 * math.Sin(q.X*40)
		h.add(tab, 2)
		belt := bandY(layer(tor, 0.03, 0.01), q, 1.1, 1.15)
		h.add(belt, 6)
		h.add(sdBox(q.Sub(V{0, 1.125, 0.185}), V{0.03, 0.026, 0.01}, 0.004), 3)
		// cape
		h.add(cloak(q, V{0, 1.55, -0.04}, V{0, 0.2, -0.22}, 0.2, 0.44, -0.05, 9), 2)
		// pauldrons and arms
		h.add(pauldron(q, R.S.Add(V{0.02, 0.03, 0}), 1, 1.1), 1)
		h.add(pauldron(q, L.S.Add(V{-0.02, 0.03, 0}), -1, 1.1), 1)
		for _, a := range []Arm{R, L} {
			h.add(limb(q, a, 0.06, 0.05), 7)
			v := bandY(layer(limb(q, a, 0.06, 0.05), 0.012, 0.008), q, math.Min(a.E.Y, a.W.Y)-0.01, math.Max(a.E.Y, a.W.Y)+0.0)
			h.add(v, 1)
			h.add(hand(q, a)-0.006, 1)
		}
		// great helm with crest
		o := q.Sub(headC)
		helm := sdCyl(o.Sub(V{0, 0.0, 0.005}), 0.098, 0.13, 0.05)
		helm = smin(helm, sdEllipsoid(o.Sub(V{0, 0.11, 0.0}), V{0.1, 0.06, 0.1}), 0.03)
		slit := sdBox(o.Sub(V{0, 0.025, 0.1}), V{0.07, 0.007, 0.04}, 0.002)
		vslit := sdBox(o.Sub(V{0, -0.03, 0.1}), V{0.006, 0.05, 0.04}, 0.002)
		helm = smax(helm, -math.Min(slit, vslit), 0.003)
		h.add(helm, 1)
		h.add(sdEllipsoid(o.Sub(V{0, 0.02, 0.0}), V{0.085, 0.1, 0.085}), 4)
		cross := sdBox(o.Sub(V{0, 0.0, 0.1}), V{0.012, 0.11, 0.008}, 0.004)
		cross = math.Min(cross, sdBox(o.Sub(V{0, 0.046, 0.1}), V{0.09, 0.009, 0.008}, 0.004))
		h.add(smax(cross, -math.Min(slit, vslit)+0.002, 0.002), 3)
		crest := chain(o, []V{{0, 0.16, 0.07}, {0, 0.22, 0.0}, {0, 0.22, -0.09}, {0, 0.15, -0.19}, {0, 0.04, -0.24}, {0, -0.06, -0.25}}, 0.032, 0.01)
		crest += 0.006 * math.Sin(o.X*300) // brushed plume strands
		h.add(crest, 8)
		// sword point-down in the right hand
		sw := q.Sub(V{0.35, 0, 0.12})
		blade := sdBox(rotZ(sw.Sub(V{0, 0.55, 0}), 0.03), V{0.022, 0.42, 0.005}, 0.002)
		h.add(blade, 1)
		guard := sdBox(sw.Sub(V{0.0, 0.98, 0}), V{0.085, 0.012, 0.014}, 0.006)
		h.add(guard, 3)
		h.add(sdCone(sw, V{0, 0.99, 0}, V{0, 1.08, 0}, 0.014, 0.014), 6)
		// heater shield on the left arm, angled toward the viewer
		sh := rotY(q.Sub(V{-0.2, 1.15, 0.36}), 0.55)
		// heater shape in the shield plane, extruded with a slight bulge
		box2 := sdBox(V{sh.X, sh.Y, 0}, V{0.21, 0.27, 1}, 0.01)
		heater := math.Max(math.Hypot(sh.X-0.16, sh.Y-0.22)-0.5, math.Hypot(sh.X+0.16, sh.Y-0.22)-0.5)
		s2 := smax(box2, heater, 0.01)
		bulge := sh.X*sh.X*0.5 + sh.Y*sh.Y*0.15
		zz := sh.Z + bulge
		shield := smax(s2, math.Abs(zz)-0.015, 0.006)
		h.add(shield, 2)
		rim := smax(math.Abs(s2+0.014)-0.014, math.Abs(zz)-0.022, 0.004)
		h.add(rim, 3)
		// sun emblem
		e := sh.Sub(V{0, 0.03, 0.03 - bulge})
		sun := sdCyl(rotX(e, math.Pi/2), 0.07, 0.008, 0.004)
		a := math.Atan2(e.Y, e.X)
		ray := math.Hypot(e.X, e.Y) - (0.09 + 0.04*math.Pow(math.Abs(math.Cos(a*6)), 6))
		ray = math.Max(ray, math.Abs(e.Z)-0.006)
		h.add(math.Min(sun, ray), 3)
		h.add(sdSphere(e.Sub(V{0, 0, 0.01}), 0.03), 5)
		return h.d, h.m
	}
	sunP := rotY(V{-0.2, 1.18, 0.42}, yaw)
	s.Lights = []Light{{P: sunP, Col: V{1.2, 0.85, 0.4}, R: 0.15}}
	s.Embers = V{1.2, 0.9, 0.45}
	s.Anchors = []Anchor{
		{Kind: "core", P: rotY(V{-0.2, 1.18, 0.4}, yaw), Col: V{1, 0.8, 0.45}, Size: 1},
		{Kind: "glint", P: pz.head(headC.Add(V{0.05, 0.08, 0.09})), Col: V{1, 0.95, 0.8}, Size: 0.6},
		{Kind: "glint", P: pz.body(V{0.35, 0.9, 0.13}), Col: V{1, 0.95, 0.85}, Size: 0.5},
		{Kind: "eye", P: pz.head(headC.Add(V{-0.03, 0.025, 0.08})), Col: V{1, 0.75, 0.35}, Size: 0.35},
		{Kind: "eye", P: pz.head(headC.Add(V{0.03, 0.025, 0.08})), Col: V{1, 0.75, 0.35}, Size: 0.35},
	}
	return s
}

// ---------------------------------------------------------------- Berserker

func berserker() *Scene {
	glow := hexc(0xff3a24)
	s := base("berserker", glow)
	s.Rim2Col = V{1.4, 0.8, 0.45}
	yaw := -0.25
	s.Mats = map[int]Mat{
		1: {Alb: hexc(0x4a4440), Rough: 0.4, Metal: 1, Grime: 0.7},
		2: {Alb: hexc(0x2e2117), Rough: 0.95, Bump: 0.4, BumpF: 120, Sheen: 0.8}, // fur
		3: {Alb: hexc(0xd8ccb0), Rough: 0.55, Grime: 0.5},                        // horn/bone
		4: {Alb: V{0.002, 0.002, 0.002}, Rough: 1},
		5: {Alb: V{}, Emit: V{9, 1.4, 0.6}},
		6: {Alb: hexc(0x3f2618), Rough: 0.55, Grime: 0.5, Sheen: 0.3, Bump: 0.25, BumpF: 50}, // leather
		7: {Alb: hexc(0x6b2016), Rough: 0.85, Cloth: true, Grime: 0.3},
		8: {Alb: hexc(0x2b1d14), Rough: 0.7, Grime: 0.4},
		9: {Alb: hexc(0x9aa0a8), Rough: 0.25, Metal: 1, Grime: 0.5},
	}
	R := Arm{S: V{0.24, 1.5, 0}, E: V{0.38, 1.3, 0.12}, W: V{0.3, 1.42, 0.3}, Hand: V{0.27, 1.47, 0.32}, Fist: true}
	L := Arm{S: V{-0.24, 1.5, 0}, E: V{-0.36, 1.25, 0.02}, W: V{-0.38, 1.02, 0.1}, Hand: V{-0.38, 0.97, 0.12}, Fist: true}
	headC := V{0, 1.75, 0.03}
	pz := Pose{Yaw: yaw, Roll: -0.09, Nod: -0.07, Turn: 0.14, Neck: V{0, 1.63, 0.02}}
	axeA := V{0.27, 1.47, 0.32}
	axeB := V{0.05, 2.15, -0.25}
	s.SDF = func(p V) (float64, int) {
		q := pz.local(p)
		h := newH()
		tor := torso(q, 1.3)
		h.add(tor, 6)
		// crossed straps and a red sash
		for _, sx := range []float64{-1, 1} {
			n := V{sx * 0.6, 1, 0}.Norm()
			st := math.Abs(q.Sub(V{0, 1.3, 0}).Dot(n)) - 0.025
			h.add(math.Max(layer(tor, 0.008, 0.008), st), 8)
		}
		sash := bandY(layer(tor, 0.012, 0.016), q, 1.0, 1.1)
		h.add(sash, 7)
		h.add(sdCyl(rotX(q.Sub(V{0, 1.24, 0.15}), math.Pi/2), 0.035, 0.01, 0.004), 1)
		h.add(legs(q, 0.04), 6)
		kilt := sdCone(V{q.X, q.Y, q.Z * 1.2}, V{0, 1.08, 0}, V{0, 0.6, 0.0}, 0.2, 0.28) + folds(q, 10, 0.012, 1.0, 0.6)
		kilt = smax(kilt, 0.62-q.Y+0.03*math.Abs(math.Sin(math.Atan2(q.X, q.Z)*5)), 0.01)
		kilt -= 0.02 * fbm(V{q.X * 28, q.Y * 5, q.Z * 28}, 3)
		h.add(kilt*0.8, 2)
		// shaggy fur mantle
		fur := capelet(q, 1.68, 1.34, 0.31, 1.15, 18)
		fur -= 0.028*fbm(V{q.X * 28, q.Y * 5, q.Z * 28}, 3) + 0.01*fbm(V{q.X * 90, q.Y * 14, q.Z * 90}, 2)
		h.add(fur*0.8, 2)
		// arms: bare-ish leather wraps with metal bracers
		for _, a := range []Arm{R, L} {
			h.add(limb(q, a, 0.07, 0.058), 6)
			br := bandY(layer(limb(q, a, 0.07, 0.058), 0.012, 0.01), q, math.Min(a.E.Y, a.W.Y)+0.02, math.Max(a.E.Y, a.W.Y)-0.02)
			h.add(br, 1)
			h.add(hand(q, a), 6)
		}
		// horned helm
		o := q.Sub(headC)
		dome := sdEllipsoid(o.Sub(V{0, 0.03, -0.01}), V{0.105, 0.11, 0.115})
		face := sdBox(o.Sub(V{0, -0.03, 0.06}), V{0.075, 0.07, 0.05}, 0.03)
		helm := smin(dome, face, 0.03)
		for _, sx := range []float64{-0.033, 0.033} {
			eye := sdEllipsoid(rotZ(o.Sub(V{sx, 0.01, 0.1}), sx*10), V{0.026, 0.009, 0.04})
			helm = smax(helm, -eye, 0.004)
			h.add(sdEllipsoid(o.Sub(V{sx, 0.01, 0.075}), V{0.012, 0.006, 0.006}), 5)
		}
		nose := sdBox(o.Sub(V{0, -0.02, 0.112}), V{0.01, 0.05, 0.01}, 0.006)
		helm = smin(helm, nose, 0.01)
		h.add(helm, 1)
		h.add(sdEllipsoid(o.Sub(V{0, 0.0, 0.02}), V{0.08, 0.09, 0.09}), 4)
		// braided beard hanging below the face guard, with iron beads
		for _, sx := range []float64{-0.028, 0.028} {
			pts := []V{{sx, -0.08, 0.085}, {sx * 1.2, -0.14, 0.1}, {sx * 1.1, -0.21, 0.105}, {sx * 0.9, -0.27, 0.1}}
			br := chain(o, pts, 0.022, 0.01)
			br += 0.004 * math.Sin(o.Y*120+sx*50)
			h.add(br, 2)
			h.add(sdTorus(o.Sub(V{sx * 1.0, -0.245, 0.103}), 0.015, 0.006), 1)
		}
		h.add(sdEllipsoid(o.Sub(V{0, -0.08, 0.075}), V{0.06, 0.03, 0.04})-0.01*fbm(o.Mul(60), 2), 2)
		for _, sx := range []float64{-1, 1} {
			pts := []V{{sx * 0.09, 0.06, 0}, {sx * 0.16, 0.08, 0.02}, {sx * 0.22, 0.13, 0.06}, {sx * 0.24, 0.2, 0.11}, {sx * 0.22, 0.25, 0.16}}
			hd := chain(o, pts, 0.034, 0.005)
			hd += 0.003 * math.Sin(o.Dot(V{sx, 0.4, 0})*160) // growth ridges
			h.add(hd, 3)
		}
		// great axe over the shoulder (unaffected by the head tilt)
		q = rotY(p, -yaw)
		h.add(sdCone(q, axeA.Sub(axeB.Sub(axeA).Norm().Mul(0.12)), axeB, 0.02, 0.018), 8)
		ax := q.Sub(axeB.Lerp(axeA, 0.14))
		dir := axeB.Sub(axeA).Norm()
		side := dir.Cross(V{0, 0, 1}).Norm()
		u, w := ax.Dot(side), ax.Dot(dir)
		z := ax.Dot(side.Cross(dir))
		// bearded crescent blade on one side, a spike on the other
		bl := math.Hypot(u+0.02, w) - 0.22
		bl = smax(bl, 0.035-u, 0.01)
		bl = smax(bl, -(math.Hypot(u-0.1, w-0.25) - 0.17), 0.01)
		bl = smax(bl, -(math.Hypot(u-0.1, w+0.25) - 0.17), 0.01)
		bl = math.Max(bl, math.Abs(z)-0.004-0.022*smooth(0.2, 0.04, u))
		h.add(bl, 9)
		sock := sdBox(V{u, w, z}, V{0.04, 0.06, 0.03}, 0.01)
		spike := sdCone(V{u, w, z}, V{0, 0, 0}, V{-0.14, 0, 0}, 0.03, 0.004)
		h.add(math.Min(sock, spike), 1)
		return h.d, h.m
	}
	s.Lights = []Light{{P: pz.head(headC.Add(V{0, 0.01, 0.12})), Col: V{0.6, 0.08, 0.03}, R: 0.08}}
	s.Embers = V{1.6, 0.4, 0.15}
	s.Anchors = []Anchor{
		{Kind: "eye", P: pz.head(headC.Add(V{-0.033, 0.01, 0.09})), Col: V{1, 0.2, 0.08}, Size: 0.5},
		{Kind: "eye", P: pz.head(headC.Add(V{0.033, 0.01, 0.09})), Col: V{1, 0.2, 0.08}, Size: 0.5},
		{Kind: "glint", P: pz.body(axeB.Lerp(axeA, 0.14).Add(V{0.15, 0.05, 0.02})), Col: V{1, 0.9, 0.85}, Size: 0.7},
		{Kind: "ember", P: pz.body(V{0, 1.24, 0.17}), Col: V{1, 0.35, 0.1}, Size: 0.5},
	}
	return s
}
