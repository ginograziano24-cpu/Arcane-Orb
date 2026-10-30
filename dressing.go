package main

import (
	"image/color"
	"math"
	"math/rand"
)

// Set dressing: every world is built in three bands.
//   1. The play volume stays clean (nothing tall between camera and orb).
//   2. A reactive ring just outside the walls: props that answer the orb
//      bouncing off that wall (crows scatter, scarecrows spin, banners
//      snap, urns wobble, steam vents puff, crystals chime).
//   3. A background with one landmark and layered silhouettes, plus
//      ambient life (birds, butterflies, motes, sand, fireflies).
// The scenery also follows the chamber's progress: braziers and lanterns
// light, the windmill and gears speed up, the temple door grinds open
// and the sky changes on the final wave.

// envState is the scenery's memory of what the play has done.
type envState struct {
	bump  [3]float64 // seconds since the orb last hit the left, right and far wall
	bumpZ [3]float64
	mill  float64 // windmill sail angle
	gearA float64
	thump float64 // speaker pulse
	prog  float64 // smoothed chamber progress 0..1
	lit   [6]float64
	door  float64
	ready bool
}

// progress is how far through the chamber this level is.
func (L *Level) progress() float64 {
	if L.Boss != nil {
		if L.Boss.MaxHP <= 0 {
			return 0
		}
		return clamp01(1 - L.Boss.HP/L.Boss.MaxHP)
	}
	if L.Cleared {
		return 1
	}
	waves := max(1, L.waves)
	f := 0.0
	if L.startLeft > 0 {
		f = clamp01(float64(L.startLeft-L.Left) / float64(L.startLeft))
	}
	return clamp01((float64(max(1, L.wave)-1) + f) / float64(waves))
}

func (e *envState) onBump(side int, z float64) {
	if side < 0 || side > 2 {
		return
	}
	if e.bump[side] > 0.25 {
		e.bump[side] = 0
		e.bumpZ[side] = z
	}
	e.thump = 1
}

func (L *Level) updateEnv(dt float64) {
	e := &L.env
	if !e.ready {
		e.ready = true
		e.bump = [3]float64{99, 99, 99}
	}
	for i := range e.bump {
		e.bump[i] += dt
	}
	e.prog += (L.progress() - e.prog) * math.Min(1, dt*1.5)
	e.mill += dt * (0.5 + 2.4*e.prog)
	e.gearA += dt * (0.25 + 1.1*e.prog)
	e.thump = math.Max(0, e.thump-dt*3)
	if L.Combo > 0 && L.Combo%5 == 0 && L.comboT > 2.9 {
		e.thump = 1
	}
	for k := range e.lit {
		on := 0.0
		if e.prog >= float64(k)/float64(len(e.lit)) {
			on = 1
		}
		e.lit[k] += (on - e.lit[k]) * math.Min(1, dt*2)
	}
	open := 0.0
	if L.wave >= L.waves && (L.waves > 1 || e.prog > 0.5) || L.Boss != nil && e.prog > 0.5 {
		open = 1
	}
	e.door += (open - e.door) * math.Min(1, dt*0.8)
}

// sideBump is how fresh the last bounce on a side of the arena is (1 = now).
func (e *envState) sideBump(x float64, window float64) float64 {
	s := 0
	if x > 0 {
		s = 1
	}
	return clamp01(1 - e.bump[s]/window)
}

// ---------------------------------------------------------------- shared props

func flapBird(B *Batch, p V3, heading, t float64, col color.RGBA, s float64) {
	flap := math.Sin(t*14) * 0.7
	dir := V3{math.Cos(heading), 0, math.Sin(heading)}
	perp := V3{-dir.Z, 0, dir.X}
	B.OBox(p, dir.Mul(0.28*s), V3{0, 0.09 * s, 0}, perp.Mul(0.09*s), col, 0)
	for _, sg := range []float64{-1, 1} {
		w := perp.Mul(sg * 0.38 * s).Add(V3{0, flap * 0.3 * s, 0})
		B.OBox(p.Add(w), w, V3{0, 0.02 * s, 0}, dir.Mul(0.16*s), col, 0)
	}
}

func hillRange(B *Batch, seed int64, z0, z1, xspan float64, n int, col color.RGBA, hmin, hmax float64) {
	r := rand.New(rand.NewSource(seed))
	for i := 0; i < n; i++ {
		x := (r.Float64()*2 - 1) * xspan
		z := z0 + r.Float64()*(z1-z0)
		if math.Abs(x) < 30 && z < 70 {
			x = math.Copysign(30+r.Float64()*20, x)
		}
		h := hmin + r.Float64()*(hmax-hmin)
		w := h * (2.2 + r.Float64()*1.5)
		B.Draw(cached("hill"+colKey(col), func() *Mesh {
			return Icosphere(2, 0.04, seed, col, darken(col, 0.82))
		}), V3{x, -h * 0.35, z}, V3{w, h, w * 0.7}, rotXYZ(0, r.Float64()*3, 0), color.RGBA{}, 0)
	}
}

// ---------------------------------------------------------------- per world

func (g *Game) drawDressing(B *Batch, L *Level, t, side float64) {
	bio := L.Biome
	e := &L.env
	low := g.lowFX()
	r := rand.New(rand.NewSource(L.propSeed + 7))
	switch bio.ID {
	case WMeadow:
		g.dressMeadow(B, L, e, t, side, r, low)
	case WRuins:
		g.dressRuins(B, L, e, t, side, r, low)
	case WClouds:
		g.dressClouds(B, L, e, t, side, r, low)
	case WNeon:
		g.dressNeon(B, L, e, t, side, r, low)
	case WCaverns:
		g.dressCaverns(B, L, e, t, side, r, low)
	case WMachine:
		g.dressMachine(B, L, e, t, side, r, low)
	}
}

// ---------------------------------------------------------------- Sunset Meadow

func (g *Game) dressMeadow(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	// background: rolling hills and a red barn on the far ridge
	hillRange(B, 41, 80, 170, 160, 9, color.RGBA{112, 150, 84, 255}, 8, 18)
	bx := -side * 52
	if !drawModel(B, "barn", V3{bx, -0.6, 76}, v3(1), rotXYZ(0, side*0.25, 0), color.RGBA{}, 0) {
		B.Box(bx-5, -0.6, 72, bx+5, 5, 80, color.RGBA{170, 60, 48, 255}, sprWhite, 0)
		B.OBox(V3{bx, 6.4, 76}, V3{5.4, 0, 0}, V3{0, 1.8, 0}, V3{0, 0, 4.2}, color.RGBA{96, 70, 60, 255}, 0)
		B.Box(bx-1.4, -0.6, 71.9, bx+1.4, 3.2, 72, color.RGBA{240, 230, 210, 255}, sprWhite, 0)
	}
	for _, sx := range []float64{-1, 1} { // hedgerows
		if hm := model("hedgerow"); hm != nil {
			for k := 0; k < 5; k++ {
				B.Draw(hm, V3{sx * (36 + float64(k)*12), -0.6, 59}, V3{2, 1, 1}, identity, color.RGBA{}, 0)
			}
			continue
		}
		B.Box(sx*30, -0.6, 58, sx*90, 1.4, 60, color.RGBA{70, 112, 60, 255}, sprWhite, 0)
	}
	// sunflowers turning to the low sun, swaying
	for i := 0; i < 7; i++ {
		x := -side * (HalfW + 9 + float64(i%3)*1.6)
		z := 2 + float64(i)*2.2
		sway := math.Sin(t*1.3+float64(i)) * 0.08
		B.Box(x-0.06, -0.6, z-0.06, x+0.06, 2.6, z+0.06, color.RGBA{90, 140, 60, 255}, sprWhite, 0)
		if drawModel(B, "sunflower_head", V3{x + sway, 2.75, z}, v3(1), rotXYZ(-0.3, side*0.6, sway), color.RGBA{}, 0.05) {
			continue
		}
		B.Draw(cached("sunflower", func() *Mesh {
			return Lathe([][2]float64{{0, 0.06}, {0.25, 0.08}, {0.32, 0.02}, {0.62, 0}, {0.55, -0.04}, {0, -0.02}}, 12,
				[]color.RGBA{{96, 60, 30, 255}, {110, 70, 36, 255}, {250, 200, 50, 255}, {240, 180, 40, 255}, {200, 150, 40, 255}}, true)
		}), V3{x + sway, 2.7, z}, v3(1), rotXYZ(math.Pi/2-0.5, side*0.6, sway), color.RGBA{}, 0.05)
	}
	// pumpkin patch
	for i := 0; i < 5; i++ {
		x := side * (HalfW + 7 + r.Float64()*5)
		z := 22 + r.Float64()*8
		s := 0.6 + r.Float64()*0.5
		if drawModel(B, "pumpkin", V3{x, -0.6, z}, v3(s), rotXYZ(0, x, 0), color.RGBA{}, 0) {
			continue
		}
		B.Draw(cached("pumpkin", func() *Mesh {
			return Icosphere(1, 0.06, 13, color.RGBA{250, 140, 40, 255}, color.RGBA{200, 90, 20, 255})
		}), V3{x, -0.6 + 0.5*s, z}, V3{s, s * 0.7, s}, identity, color.RGBA{}, 0)
		B.Box(x-0.05, -0.6+0.85*s, z-0.05, x+0.05, -0.6+1.1*s, z+0.05, color.RGBA{80, 110, 50, 255}, sprWhite, 0)
	}
	// scarecrows that spin when the orb rattles their side
	for k, p := range [][2]float64{{-1, 13}, {1, 25}} {
		x := p[0] * (HalfW + 6)
		z := p[1]
		bump := e.sideBump(x, 1.6)
		spin := 0.0
		if bump > 0 {
			age := 1.6 * (1 - bump)
			spin = 4 * math.Pi * (1 - math.Exp(-age*2.5))
		}
		a := spin + float64(k)
		if drawModel(B, "scarecrow", V3{x, -0.6, z}, v3(1), rotXYZ(0, a, 0), color.RGBA{}, 0) {
			continue
		}
		dir := V3{math.Cos(a), 0, math.Sin(a)}
		B.Box(x-0.08, -0.6, z-0.08, x+0.08, 3.2, z+0.08, color.RGBA{130, 96, 62, 255}, sprWhite, 0)
		B.OBox(V3{x, 2.4, z}, dir.Mul(1.4), V3{0, 0.07, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.07), color.RGBA{130, 96, 62, 255}, 0)
		B.OBox(V3{x, 2.1, z}, dir.Mul(0.55), V3{0, 0.6, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.25), color.RGBA{170, 70, 60, 255}, 0)
		B.Draw(cached("strawHead", func() *Mesh {
			return Icosphere(1, 0.05, 3, color.RGBA{236, 210, 150, 255}, color.RGBA{200, 170, 110, 255})
		}), V3{x, 3.1, z}, v3(0.42), identity, color.RGBA{}, 0)
		B.Draw(cached("strawHat", func() *Mesh {
			return Lathe([][2]float64{{0.9, 0}, {0.9, 0.05}, {0.45, 0.1}, {0.38, 0.55}, {0, 0.6}}, 10,
				[]color.RGBA{{200, 160, 80, 255}, {210, 170, 90, 255}, {190, 150, 70, 255}, {200, 160, 80, 255}}, true)
		}), V3{x, 3.38, z}, v3(1), rotXYZ(0.1, a, 0), color.RGBA{}, 0)
	}
	// crows perched on the fence rail; a bounce on their side scatters them
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 3.2)
		z := 1.5 + float64(i)*5
		perch := V3{x, 1.05, z}
		age := e.bump[0]
		if sx > 0 {
			age = e.bump[1]
		}
		heading := sx*math.Pi/2 + float64(i)
		switch {
		case age < 2.2:
			// burst up and away
			f := age / 2.2
			p := perch.Add(V3{sx * f * 10, f*7 + math.Sin(age*6)*0.2, f * (3 + float64(i))})
			flapBird(B, p, math.Atan2(3+float64(i), sx*10), t+float64(i), color.RGBA{30, 28, 36, 255}, 1)
		case age < 4.2:
			// glide back to the rail
			f := 1 - (age-2.2)/2
			p := perch.Add(V3{sx * f * 10, f * 7, f * (3 + float64(i))})
			flapBird(B, p, math.Atan2(-(3+float64(i)), -sx*10), t*0.6+float64(i), color.RGBA{30, 28, 36, 255}, 1)
		default:
			bob := math.Max(0, math.Sin(t*2+float64(i)*2)) * 0.05
			if drawModel(B, "crow", perch.Add(V3{0, bob, 0}), v3(1.3), rotXYZ(0, math.Atan2(math.Cos(heading), math.Sin(heading)), 0), color.RGBA{}, 0) {
				continue
			}
			dir := V3{math.Cos(heading), 0, math.Sin(heading)}
			B.OBox(perch.Add(V3{0, 0.2 + bob, 0}), dir.Mul(0.26), V3{0, 0.14, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.12), color.RGBA{30, 28, 36, 255}, 0)
			B.OBox(perch.Add(dir.Mul(0.26)).Add(V3{0, 0.34 + bob, 0}), dir.Mul(0.1), V3{0, 0.1, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.09), color.RGBA{30, 28, 36, 255}, 0)
			B.OBox(perch.Add(dir.Mul(0.42)).Add(V3{0, 0.32 + bob, 0}), dir.Mul(0.07), V3{0, 0.025, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.025), color.RGBA{230, 170, 60, 255}, 0)
		}
	}
	// lanterns on every other fence post light up as the chamber is won
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 3.2)
		z := -2 + float64(i/2)*12 + 6
		lit := e.lit[i]
		if !drawModel(B, "lantern", V3{x, 1.85, z}, v3(1), identity, color.RGBA{}, float32(0.8*lit)) {
			B.Box(x-0.18, 1.6, z-0.18, x+0.18, 2.1, z+0.18, color.RGBA{60, 50, 40, 255}, sprWhite, 0)
		}
		B.Billboard(V3{x, 1.85, z}, 0.32, sprSphere, mixc(color.RGBA{120, 100, 80, 255}, color.RGBA{255, 220, 140, 255}, lit), false)
		if lit > 0.05 {
			B.Billboard(V3{x, 1.85, z}, 2.4, sprSoft, alpha(color.RGBA{255, 190, 100, 255}, 0.6*lit*(0.85+0.15*math.Sin(t*7+float64(i)))), true)
		}
	}
	if low {
		return
	}
	// butterflies and drifting dandelion seeds
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		ph := float64(i) * 1.7
		p := V3{sx * (HalfW + 5 + 3*math.Sin(t*0.3+ph)), 1.2 + 0.6*math.Sin(t*0.9+ph), 6 + float64(i)*4 + 3*math.Cos(t*0.25+ph)}
		col := []color.RGBA{{255, 210, 80, 255}, {255, 255, 255, 255}, {170, 200, 255, 255}}[i%3]
		f := math.Abs(math.Sin(t*16 + ph))
		B.OBox(p.Add(V3{-0.12 * f, 0, 0}), V3{0.12 * f, 0.1, 0}, V3{0, 0.02, 0}, V3{0, 0, 0.12}, col, 0.2)
		B.OBox(p.Add(V3{0.12 * f, 0, 0}), V3{0.12 * f, -0.1, 0}, V3{0, 0.02, 0}, V3{0, 0, 0.12}, col, 0.2)
	}
	for i := 0; i < 18; i++ {
		ph := float64(i) * 2.3
		u := math.Mod(t*0.04+float64(i)/18, 1)
		p := V3{-60 + u*120, 2 + 3*math.Sin(t*0.5+ph) + float64(i%4), 4 + float64(i)*3}
		if math.Abs(p.X) < HalfW+2 && p.Z < FarZ {
			continue
		}
		B.Billboard(p, 0.35, sprSoft, color.RGBA{255, 250, 240, 160}, true)
	}
}

// ---------------------------------------------------------------- Ancient Ruins

func (g *Game) dressRuins(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	// background: dunes and a row of stepped pyramids
	hillRange(B, 77, 70, 150, 170, 10, color.RGBA{226, 188, 128, 255}, 5, 12)
	for i, p := range [][3]float64{{-side * 60, 140, 26}, {side * 30, 165, 34}, {-side * 10, 190, 20}} {
		B.Draw(cached("pyramid", func() *Mesh {
			prof := [][2]float64{}
			cols := []color.RGBA{}
			for k := 0; k <= 6; k++ {
				w := 1 - float64(k)/6
				prof = append(prof, [2]float64{w, float64(k) / 6 * 0.9}, [2]float64{w - 0.04, float64(k)/6*0.9 + 0.15/6})
				cols = append(cols, color.RGBA{214, 176, 118, 255}, color.RGBA{196, 160, 104, 255})
			}
			return Lathe(prof, 4, cols[:len(prof)-1], true)
		}), V3{p[0], -0.6, p[1]}, V3{p[2], p[2], p[2]}, rotXYZ(0, math.Pi/4+float64(i)*0.2, 0), color.RGBA{}, 0)
	}
	// the temple gate opens on the final wave
	gx := -side * 21
	gz := 44.0
	stone := lighten(L.Biome.Wall, 0.05)
	open := e.door
	if gm, dm := model("temple_gate"), model("temple_door"); gm != nil && dm != nil {
		B.Draw(gm, V3{gx, -0.6, gz}, v3(1), identity, color.RGBA{}, 0)
		B.Draw(dm, V3{gx, -0.6 + open*8.5, gz - 0.4}, v3(1), identity, color.RGBA{}, 0)
	} else {
		for _, dx := range []float64{-4.4, 4.4} {
			B.BoxTex(gx+dx-1.1, -0.6, gz-1.2, gx+dx+1.1, 9, gz+1.2, stone, matTop[WRuins], matSide[WRuins], 0)
		}
		B.BoxTex(gx-5.8, 9, gz-1.4, gx+5.8, 10.6, gz+1.4, darken(stone, 0.9), matTop[WRuins], matSide[WRuins], 0)
		B.Box(gx-3.3, -0.6, gz-0.2, gx+3.3, 9, gz+0.2, color.RGBA{30, 22, 18, 255}, sprWhite, 0) // the dark inside
		dy := open * 8.5
		B.BoxTex(gx-3.3, -0.6+dy, gz-0.5, gx+3.3, 9+dy*0.02, gz-0.25, darken(stone, 0.85), matTop[WRuins], matSide[WRuins], 0)
	}
	if open > 0.05 {
		B.Billboard(V3{gx, 4, gz - 0.6}, 9*open, sprSoft, alpha(color.RGBA{255, 210, 120, 255}, 0.55*open), true)
		for k := 0; k < 4; k++ { // light shafts spilling out
			x := gx - 2.4 + float64(k)*1.6
			B.Quad([4]V3{{x - 0.3, 8 * open, gz - 0.6}, {x + 0.3, 8 * open, gz - 0.6}, {x + 1.2, -0.55, gz - 8}, {x - 0.6, -0.55, gz - 8}}, sprWhite,
				alpha(color.RGBA{255, 220, 140, 255}, 0.18*open), [4]float32{1, 1, 1, 1}, true)
		}
	}
	// braziers light one by one as the chamber falls
	for k, p := range [][2]float64{{-1, 6}, {1, 6}, {-1, 18}, {1, 18}, {-1, 30}, {1, 30}} {
		x := p[0] * (HalfW + 5)
		z := p[1]
		if !drawModel(B, "brazier", V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0) {
			B.Draw(cached("brazierStand", func() *Mesh {
				return Lathe([][2]float64{{0.6, 0}, {0.5, 0.2}, {0.18, 0.4}, {0.15, 2}, {0.25, 2.1}, {0.8, 2.3}, {0.9, 2.7}, {0.75, 2.75}}, 10,
					[]color.RGBA{{120, 96, 70, 255}, {130, 104, 76, 255}, {120, 96, 70, 255}, {200, 150, 60, 255}, {210, 160, 70, 255}, {220, 170, 80, 255}, {170, 120, 50, 255}}, true)
			}), V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0)
		}
		lit := e.lit[k]
		if lit > 0.03 {
			fl := 0.8 + 0.2*math.Sin(t*11+float64(k)*3)
			B.Billboard(V3{x, 2.6 + 0.5*lit, z}, 1.6*lit*fl, sprSoft, alpha(color.RGBA{255, 150, 40, 255}, 0.9), true)
			B.Billboard(V3{x, 2.4 + 0.3*lit, z}, 0.7*lit, sprSphere, color.RGBA{255, 230, 160, 255}, false)
			B.Billboard(V3{x, 2.4, z}, 5*lit, sprSoft, alpha(color.RGBA{255, 160, 60, 255}, 0.25*lit), true)
			if !low {
				for m := 0; m < 4; m++ { // embers
					ph := math.Mod(t*0.8+float64(m)*0.25+float64(k)*0.13, 1)
					B.Billboard(V3{x + math.Sin(ph*9+float64(m))*0.4, 2.8 + ph*3, z}, 0.18, sprSoft, alpha(color.RGBA{255, 190, 90, 255}, (1-ph)*lit), true)
				}
			}
		}
	}
	// urns along the wall wobble when the orb strikes that side
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 3.4)
		z := 3 + float64(i)*5
		w := e.sideBump(x, 1.2)
		tilt := 0.0
		if w > 0 {
			age := 1.2 * (1 - w)
			tilt = math.Sin(age*16) * math.Exp(-age*3) * 0.35
		}
		if drawModel(B, "urn", V3{x, -0.6, z}, v3(0.9), rotXYZ(tilt, float64(i), tilt*0.5), color.RGBA{}, 0) {
			continue
		}
		B.Draw(cached("urn", func() *Mesh {
			return Lathe([][2]float64{{0.3, 0}, {0.55, 0.3}, {0.6, 0.7}, {0.4, 1.1}, {0.25, 1.25}, {0.35, 1.4}, {0, 1.42}}, 10,
				[]color.RGBA{{170, 96, 60, 255}, {190, 110, 70, 255}, {60, 150, 150, 255}, {190, 110, 70, 255}, {170, 96, 60, 255}, {160, 90, 56, 255}}, true)
		}), V3{x, -0.6, z}, v3(0.9), rotXYZ(tilt, 0, tilt*0.5), color.RGBA{}, 0)
	}
	// glyph slabs
	for _, sx := range []float64{-1, 1} {
		x := sx * (HalfW + 11)
		if drawModel(B, "glyph_slab", V3{x, -0.6, 25}, v3(1), rotXYZ(0, sx*math.Pi/2, 0), color.RGBA{}, 0) {
			continue
		}
		B.BoxTex(x-0.4, -0.6, 22, x+0.4, 6, 28, stone, matTop[WRuins], matSide[WRuins], 0)
		gr := rand.New(rand.NewSource(int64(sx * 9)))
		for k := 0; k < 12; k++ {
			gy := 0.6 + float64(k/3)*1.3
			gzz := 22.8 + float64(k%3)*1.7
			B.Box(x-sx*0.42-0.02, gy, gzz, x-sx*0.42+0.02, gy+0.3+gr.Float64()*0.6, gzz+0.4+gr.Float64()*0.5, color.RGBA{70, 140, 140, 255}, sprWhite, 0.25)
		}
	}
	if low {
		return
	}
	// a vulture circling, a dust devil, sand blowing off the dunes
	va := t * 0.25
	flapBird(B, V3{side*12 + math.Cos(va)*20, 24 + math.Sin(t*0.5), 64 + math.Sin(va)*14}, va+math.Pi/2, t*0.3, color.RGBA{50, 40, 36, 255}, 3)
	dx, dz := -side*30, 26.0
	for k := 0; k < 14; k++ {
		h := float64(k) * 0.8
		a := t*4 + float64(k)*0.9
		rad := 0.4 + h*0.18
		B.Billboard(V3{dx + math.Cos(a)*rad + math.Sin(t*0.4)*3, h, dz + math.Sin(a)*rad}, 1.2+h*0.3, sprSoft, color.RGBA{230, 200, 150, uint8(70 - k*3)}, false)
	}
	for i := 0; i < 10; i++ {
		u := math.Mod(t*0.15+float64(i)/10, 1)
		x := -70 + u*140
		if math.Abs(x) < HalfW+3 {
			continue
		}
		B.Decal(x, 8+float64(i)*4, -0.5, 6, 0.4, sprSoft, color.RGBA{240, 214, 170, 90}, true)
	}
}

// ---------------------------------------------------------------- Cloud Kingdom

func (g *Game) dressClouds(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	// a rainbow that strengthens as the chamber is won
	ra := 0.04 + 0.6*e.prog*e.prog
	cols := []color.RGBA{{255, 90, 90, 255}, {255, 170, 60, 255}, {255, 240, 90, 255}, {110, 230, 120, 255}, {90, 170, 255, 255}, {170, 110, 255, 255}}
	for bi, c := range cols {
		R := 38.0 - float64(bi)*1.5
		for k := 0; k < 24; k++ {
			a0, a1 := math.Pi*float64(k)/24, math.Pi*float64(k+1)/24
			p0 := V3{side*20 + math.Cos(a0)*R, math.Sin(a0)*R - 7, 72}
			p1 := V3{side*20 + math.Cos(a1)*R, math.Sin(a1)*R - 7, 72}
			q0 := V3{side*20 + math.Cos(a0)*(R-1.5), math.Sin(a0)*(R-1.5) - 7, 72}
			q1 := V3{side*20 + math.Cos(a1)*(R-1.5), math.Sin(a1)*(R-1.5) - 7, 72}
			B.Quad([4]V3{p0, p1, q1, q0}, sprWhite, alpha(c, ra), [4]float32{1, 1, 1, 1}, false) // alpha, not additive: the sky is too bright to add onto
		}
	}
	// banners on poles snap when the orb strikes their side
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 4)
		z := 4 + float64(i/2)*12
		if !drawModel(B, "banner_pole", V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0) {
			B.Box(x-0.07, -0.6, z-0.07, x+0.07, 5.4, z+0.07, color.RGBA{230, 220, 240, 255}, sprWhite, 0)
			B.Billboard(V3{x, 5.5, z}, 0.36, sprSphere, color.RGBA{255, 210, 100, 255}, false)
		}
		snap := 0.15 + 0.6*e.sideBump(x, 1.5)
		col := []color.RGBA{{120, 150, 240, 255}, {240, 130, 170, 255}, {255, 210, 100, 255}}[i%3]
		for k := 0; k < 4; k++ {
			f0, f1 := float64(k)/4, float64(k+1)/4
			w0 := math.Sin(t*6-f0*5+float64(i)) * snap * f0
			w1 := math.Sin(t*6-f1*5+float64(i)) * snap * f1
			B.Quad([4]V3{{x + w0, 5.2, z + f0*2.2}, {x + w1, 5.2, z + f1*2.2}, {x + w1, 3.6 + f1*0.2, z + f1*2.2}, {x + w0, 3.6 + f0*0.2, z + f0*2.2}},
				sprWhite, col, [4]float32{1, 1, 0.8, 0.8}, false)
		}
	}
	// balloon buoys bobbing on their tethers
	for i := 0; i < 4; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 9 + float64(i))
		z := 8 + float64(i)*7
		y := 4 + math.Sin(t*0.8+float64(i))*0.6
		B.Box(x-0.02, -0.6, z-0.02, x+0.02, y-0.9, z+0.02, color.RGBA{200, 200, 220, 255}, sprWhite, 0)
		col := []color.RGBA{{255, 140, 160, 255}, {120, 200, 255, 255}, {255, 220, 110, 255}, {170, 240, 150, 255}}[i]
		if !drawModel(B, "buoy", V3{x, y, z}, v3(1), rotXYZ(0, t*0.3+float64(i), 0), col, 0.05) {
			B.Draw(cached("buoy"+colKey(col), func() *Mesh { return Icosphere(1, 0.02, 9, lighten(col, 0.2), darken(col, 0.8)) }),
				V3{x, y, z}, V3{0.9, 1.1, 0.9}, identity, color.RGBA{}, 0.05)
		}
	}
	// on the final wave a thunderhead gathers in the distance
	if L.wave >= L.waves && L.waves > 1 || L.Boss != nil && e.prog > 0.5 {
		f := math.Mod(t*0.7, 4)
		if f < 0.15 {
			fl := 1 - f/0.15
			lx := side * -40
			B.Billboard(V3{lx, 26, 100}, 50, sprSoft, alpha(color.RGBA{220, 230, 255, 255}, 0.5*fl), true)
			p := V3{lx, 34, 100}
			for k := 0; k < 6; k++ {
				q := p.Add(V3{(math.Sin(float64(k)*3.1+math.Floor(t*0.7)) * 3), -7, 0})
				B.OBox(p.Add(q).Mul(0.5), q.Sub(p).Mul(0.5), V3{0.25, 0, 0}, V3{0, 0, 0.25}, color.RGBA{230, 240, 255, 255}, 1)
				p = q
			}
		}
	}
	if low {
		return
	}
	// a flock crossing the sky, and petals on the wind
	fx := math.Mod(t*3, 220) - 110
	for k := 0; k < 7; k++ {
		off := float64(k-3) * 2.2
		flapBird(B, V3{fx - math.Abs(off)*1.5, 26 + math.Abs(off)*0.4, 90 + off*2}, 0, t+float64(k), color.RGBA{250, 250, 255, 255}, 1.4)
	}
	for i := 0; i < 16; i++ {
		u := math.Mod(t*0.06+float64(i)/16, 1)
		p := V3{-50 + u*100, 9 - u*8 + math.Sin(t+float64(i))*0.5, 3 + float64(i)*2.5}
		if math.Abs(p.X) < HalfW+1 && p.Z < FarZ {
			continue
		}
		B.Billboard(p, 0.3, sprSoft, color.RGBA{255, 190, 220, 200}, true)
	}
}

// ---------------------------------------------------------------- Neon Grove

func (g *Game) dressNeon(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	bio := L.Biome
	// wireframe mountains on the horizon
	for i := 0; i < 9; i++ {
		x0 := -90 + float64(i)*20
		h := 9 + 7*math.Abs(math.Sin(float64(i)*1.7))
		pk := V3{x0 + 10, h, 74}
		for _, base := range []V3{{x0, -0.6, 74}, {x0 + 20, -0.6, 74}} {
			mid := pk.Add(base).Mul(0.5)
			d := pk.Sub(base).Mul(0.5)
			B.OBox(mid, d, V3{0.25, 0, 0}, V3{0, 0, 0.25}, bio.Floor, 0.9)
		}
	}
	// equalizer bars along the horizon, pulsing with the play
	for i := 0; i < 32; i++ {
		x := -62 + float64(i)*4
		if math.Abs(x) < 20 {
			continue
		}
		h := 0.5 + 3*math.Abs(math.Sin(t*3+float64(i)*0.7))*(0.4+0.6*e.thump) + 1.6*e.prog
		col := mixc(bio.Trim, bio.Floor, float64(i%4)/4)
		B.Box(x-0.9, -0.6, 58, x+0.9, -0.6+h, 58.6, col, sprWhite, 0.9)
	}
	// speaker stacks that thump with bounces and combos
	for _, sx := range []float64{-1, 1} {
		for k, z := range []float64{9, 23} {
			x := sx * (HalfW + 6)
			ry := 0.0
			if sx < 0 {
				ry = math.Pi
			}
			if drawModel(B, "speaker", V3{x, -0.6, z}, V3{1, 1 + 0.03*e.thump, 1}, rotXYZ(0, ry, 0), color.RGBA{}, float32(0.6*e.thump)) {
				B.Billboard(V3{x - sx*1.6, 2.2, z}, 3+2*e.thump, sprSoft, alpha(bio.Trim, 0.25+0.4*e.thump), true)
				_ = k
				continue
			}
			B.Box(x-1.4, -0.6, z-1.2, x+1.4, 4.4, z+1.2, color.RGBA{24, 18, 40, 255}, sprWhite, 0)
			for j := 0; j < 2; j++ {
				cy := 0.8 + float64(j)*2.2
				pulse := 1 + 0.18*e.thump*float64(1-j)
				fx := x - sx*1.42
				B.Draw(cached("cone", func() *Mesh {
					return Lathe([][2]float64{{0, 0}, {0.3, 0.05}, {0.8, 0.2}, {0.95, 0.25}, {0.95, 0.3}}, 14,
						[]color.RGBA{{40, 30, 60, 255}, {60, 46, 90, 255}, {80, 60, 120, 255}, {255, 90, 220, 255}}, true)
				}), V3{fx, cy + 0.8, z}, v3(pulse*(1-0.3*float64(j))), rotXYZ(0, 0, -sx*math.Pi/2), color.RGBA{}, float32(0.2+0.6*e.thump))
			}
			B.Billboard(V3{x - sx*1.6, 2.2, z}, 3+2*e.thump, sprSoft, alpha(bio.Trim, 0.25+0.4*e.thump), true)
			_ = k
		}
	}
	// holographic billboards with scrolling bars
	for _, sx := range []float64{-1, 1} {
		x := sx * (HalfW + 15)
		z := 18.0
		if !drawModel(B, "holo_frame", V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0) {
			B.Box(x-0.15, -0.6, z-0.15, x+0.15, 7, z+0.15, color.RGBA{60, 50, 90, 255}, sprWhite, 0)
		}
		B.Quad([4]V3{{x, 11, z - 4}, {x, 11, z + 4}, {x, 6.5, z + 4}, {x, 6.5, z - 4}}, sprWhite, alpha(color.RGBA{30, 200, 255, 255}, 0.18), [4]float32{1, 1, 1, 1}, true)
		for k := 0; k < 5; k++ {
			y := 6.8 + math.Mod(float64(k)*0.9+t*1.2, 4.2)
			w := 1.5 + 2*math.Abs(math.Sin(float64(k)*2+t*0.5))
			B.Quad([4]V3{{x - sx*0.05, y + 0.18, z - w}, {x - sx*0.05, y + 0.18, z + w}, {x - sx*0.05, y, z + w}, {x - sx*0.05, y, z - w}}, sprWhite,
				alpha(mixc(bio.Trim, color.RGBA{60, 230, 255, 255}, float64(k%2)), 0.7), [4]float32{1, 1, 1, 1}, true)
		}
	}
	// neon palms
	for i := 0; i < 4; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 10 + float64(i/2)*8)
		z := 6 + float64(i)*8
		if drawModel(B, "neon_palm", V3{x, -0.6, z}, v3(1), rotXYZ(0, float64(i)+math.Sin(t*0.8+float64(i))*0.05, 0), color.RGBA{}, 0) {
			continue
		}
		B.Box(x-0.18, -0.6, z-0.18, x+0.18, 6, z+0.18, color.RGBA{40, 30, 70, 255}, sprWhite, 0.1)
		for k := 0; k < 6; k++ {
			a := float64(k)/6*2*math.Pi + math.Sin(t*0.8+float64(i))*0.08
			dir := V3{math.Cos(a), -0.3, math.Sin(a)}.Norm()
			B.OBox(V3{x, 6, z}.Add(dir.Mul(1.6)), dir.Mul(1.7), V3{0, 0.04, 0}, V3{-math.Sin(a), 0, math.Cos(a)}.Mul(0.18), bio.Hues[(k+i)%6], 0.9)
		}
	}
	// the final wave makes the horizon glitch
	if L.wave >= L.waves && L.waves > 1 || L.Boss != nil && e.prog > 0.5 {
		if math.Mod(t*1.3, 2) < 0.25 {
			for k := 0; k < 4; k++ {
				y := 6 + float64(k)*5 + math.Sin(t*30+float64(k))*2
				B.Quad([4]V3{{-200, y + 0.6, 160}, {200, y + 0.6, 160}, {200, y, 160}, {-200, y, 160}}, sprWhite, alpha(color.RGBA{60, 255, 230, 255}, 0.35), [4]float32{1, 1, 1, 1}, true)
			}
		}
	}
	if low {
		return
	}
	for i := 0; i < 20; i++ { // data motes
		ph := math.Mod(t*0.2+float64(i)*0.137, 1)
		x := float64(i%2*2-1) * (HalfW + 3 + float64(i%5)*4)
		B.Billboard(V3{x, ph * 10, 2 + float64(i)*1.6}, 0.25, sprSoft, alpha(bio.Hues[i%6], 1-ph), true)
	}
}

// ---------------------------------------------------------------- Crystal Caverns

func (g *Game) dressCaverns(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	bio := L.Biome
	// cavern walls: dark rock ridges far out
	hillRange(B, 91, 42, 72, 120, 12, color.RGBA{50, 54, 80, 255}, 14, 30)
	// chime crystals by the walls flash when the orb strikes their side
	for i := 0; i < 8; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 3.6)
		z := 1 + float64(i)*4
		col := bio.Hues[i%6]
		f := e.sideBump(x, 0.9)
		drawCrystal(B, col, V3{x, -0.4, z}, V3{0.45, 1.4 + float64(i%3)*0.5, 0.45},
			rotXYZ(0.2*sx, float64(i), 0.1), color.RGBA{}, float32(0.25+0.75*f))
		B.Billboard(V3{x, 0.8, z}, 1.8+2*f, sprSoft, alpha(col, 0.25+0.6*f), true)
	}
	// glow-worm curtains hanging from rock arches, swaying on bounces
	for _, sx := range []float64{-1, 1} {
		ax := sx * (HalfW + 9)
		if !drawModel(B, "cave_arch", V3{ax, 7, 16}, v3(1), identity, color.RGBA{}, 0) {
			B.Draw(cached("arch", func() *Mesh {
				return Icosphere(1, 0.25, 33, color.RGBA{64, 70, 100, 255}, color.RGBA{30, 34, 52, 255})
			}), V3{ax, 7, 16}, V3{3, 1.6, 9}, identity, color.RGBA{}, 0)
		}
		sway := 0.1 + 0.6*e.sideBump(ax, 2)
		for k := 0; k < 12; k++ {
			z := 9 + float64(k)*1.2
			ln := 2 + float64((k*7)%5)*0.6
			dx := math.Sin(t*2+float64(k)) * sway * 0.4
			B.Box(ax+dx-0.015, 6-ln, z-0.015, ax+dx+0.015, 6, z+0.015, color.RGBA{150, 220, 255, 255}, sprWhite, 0.6)
			B.Billboard(V3{ax + dx, 6 - ln, z}, 0.5, sprSoft, alpha(color.RGBA{120, 230, 255, 255}, 0.7*(0.6+0.4*e.prog)), true)
		}
	}
	// a ring of crystals that brightens as the chamber is won
	g0 := 0.2 + 0.8*e.prog
	B.Billboard(V3{side * 14, 8, 48}, 26, sprSoft, alpha(bio.Trim, 0.12*g0), true)
	if low {
		return
	}
	for i := 0; i < 26; i++ { // fireflies
		ph := float64(i) * 1.31
		x := float64(i%2*2-1) * (HalfW + 3 + 10*math.Abs(math.Sin(ph)))
		p := V3{x + math.Sin(t*0.7+ph)*1.5, 1 + 3*math.Abs(math.Sin(t*0.4+ph)), 2 + float64(i)*1.6 + math.Cos(t*0.5+ph)}
		B.Billboard(p, 0.28, sprSoft, alpha(color.RGBA{190, 255, 170, 255}, 0.5+0.5*math.Sin(t*5+ph)), true)
	}
	for i := 0; i < 4; i++ { // drips into the pool
		ph := math.Mod(t*0.6+float64(i)*0.27, 1)
		x, z := -side*24+float64(i-2)*2.5, 26+float64(i%2)*2
		if ph < 0.85 {
			B.Billboard(V3{x, 14 * (1 - ph/0.85), z}, 0.22, sprSphere, color.RGBA{170, 230, 255, 255}, false)
		} else {
			k := (ph - 0.85) / 0.15
			B.Decal(x, z, -0.52, 0.5+2*k, 0.5+2*k, sprRing, alpha(color.RGBA{170, 230, 255, 255}, 1-k), true)
		}
	}
}

// ---------------------------------------------------------------- the Machine

func (g *Game) dressMachine(B *Batch, L *Level, e *envState, t, side float64, r *rand.Rand, low bool) {
	// smokestacks on the skyline
	for i, p := range [][2]float64{{-side * 45, 100}, {side * 25, 120}, {-side * 5, 140}, {side * 60, 95}} {
		h := 26 + float64(i%2)*10
		if m := model("smokestack"); m != nil {
			B.Draw(m, V3{p[0], -0.6, p[1]}, V3{1, h / 22, 1}, identity, color.RGBA{}, 0)
		} else {
			B.Draw(cached("stack", func() *Mesh {
				return Lathe([][2]float64{{3.2, 0}, {2.6, 2}, {2.2, 20}, {2.6, 21}, {2.6, 22}, {2.1, 22.2}}, 10,
					[]color.RGBA{{70, 66, 70, 255}, {110, 70, 56, 255}, {150, 90, 60, 255}, {60, 58, 62, 255}, {40, 40, 44, 255}}, true)
			}), V3{p[0], -0.6, p[1]}, V3{1, h / 22, 1}, identity, color.RGBA{}, 0)
		}
		if !low {
			for k := 0; k < 6; k++ {
				ph := math.Mod(t*0.12+float64(k)/6+float64(i)*0.3, 1)
				B.Billboard(V3{p[0] + ph*12, h + ph*18, p[1]}, 5+ph*14, sprSoft, color.RGBA{90, 84, 84, uint8(140 * (1 - ph))}, false)
			}
		}
	}
	// steam vents in the floor beside the walls puff when the orb strikes that side
	for i := 0; i < 6; i++ {
		sx := float64(i%2*2 - 1)
		x := sx * (HalfW + 2.7)
		z := 4 + float64(i/2)*11
		if !drawModel(B, "vent", V3{x, -0.6, z}, v3(1), rotXYZ(0, math.Pi/2, 0), color.RGBA{}, 0) {
			B.Box(x-0.6, -0.58, z-0.6, x+0.6, -0.5, z+0.6, color.RGBA{50, 52, 58, 255}, sprWhite, 0)
			for k := -1; k <= 1; k++ {
				B.Box(x-0.5, -0.5, z+float64(k)*0.3-0.04, x+0.5, -0.49, z+float64(k)*0.3+0.04, color.RGBA{25, 25, 30, 255}, sprWhite, 0)
			}
		}
		f := e.sideBump(x, 1.4)
		if f > 0 {
			age := 1.4 * (1 - f)
			for k := 0; k < 5; k++ {
				ph := clamp01(age/1.4 + float64(k)*0.08)
				B.Billboard(V3{x + math.Sin(float64(k)*2)*0.4, ph * 6, z}, 1+ph*3, sprSoft, color.RGBA{230, 230, 235, uint8(160 * (1 - ph))}, false)
			}
		}
	}
	// pressure gauges whose needles climb with the chamber's progress
	for _, sx := range []float64{-1, 1} {
		x := sx * (HalfW + 5.5)
		z := 33.0
		if drawModel(B, "gauge", V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0.1) {
			goto needle
		}
		B.Box(x-0.15, -0.6, z-0.15, x+0.15, 3, z+0.15, color.RGBA{80, 82, 90, 255}, sprWhite, 0)
		B.Draw(cached("gaugeFace", func() *Mesh {
			return Lathe([][2]float64{{0, 0.12}, {1.1, 0.12}, {1.2, 0.05}, {1.2, -0.05}, {0, -0.06}}, 18,
				[]color.RGBA{{236, 228, 210, 255}, {200, 150, 60, 255}, {150, 110, 50, 255}, {80, 82, 90, 255}}, true)
		}), V3{x, 4.2, z}, v3(1), rotXYZ(math.Pi/2, 0, 0), color.RGBA{}, 0.1)
	needle:
		a := math.Pi*1.25 - e.prog*math.Pi*1.5 + math.Sin(t*9)*0.02
		dir := V3{math.Cos(a), math.Sin(a), 0}
		B.OBox(V3{x, 4.2, z - 0.16}.Add(dir.Mul(0.45)), dir.Mul(0.5), V3{-dir.Y, dir.X, 0}.Mul(0.04), V3{0, 0, 0.02}, color.RGBA{200, 40, 40, 255}, 0.4)
		if e.prog > 0.8 {
			B.Billboard(V3{x, 4.2, z - 0.3}, 3, sprSoft, alpha(color.RGBA{255, 80, 50, 255}, 0.3+0.3*math.Sin(t*10)), true)
		}
	}
	// the final wave sets the alarm lights sweeping
	if L.wave >= L.waves && L.waves > 1 || L.Boss != nil && e.prog > 0.5 {
		for _, sx := range []float64{-1, 1} {
			x := sx * (HalfW + 3)
			a := t * 5 * sx
			B.Billboard(V3{x, 4.8, FarZ}, 1.2, sprSphere, color.RGBA{255, 70, 50, 255}, false)
			B.Quad([4]V3{{x, 4.8, FarZ}, {x, 4.8, FarZ}, {x + math.Cos(a)*14, 4.8 + math.Sin(a*0.5)*2, FarZ - 8 + math.Sin(a)*8}, {x + math.Cos(a+0.25)*14, 4.8, FarZ - 8 + math.Sin(a+0.25)*8}},
				sprWhite, alpha(color.RGBA{255, 60, 40, 255}, 0.2), [4]float32{1, 1, 1, 1}, true)
		}
	}
	if low {
		return
	}
	// sparks spitting from the consoles now and then
	for i := 0; i < 4; i++ {
		ph := math.Mod(t*0.5+float64(i)*0.31, 1)
		if ph > 0.3 {
			continue
		}
		sx := float64(i%2*2 - 1)
		x, z := sx*(HalfW+7.4), 6+float64(i/2)*13
		for k := 0; k < 6; k++ {
			a := float64(k) * 1.1
			d := ph / 0.3
			B.Billboard(V3{x + math.Cos(a)*d*1.5, 2.5 + d*1.5 - d*d*3, z + math.Sin(a)*d*1.5}, 0.15, sprSoft, alpha(color.RGBA{255, 200, 90, 255}, 1-d), true)
		}
	}
}
