package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// drawWorld renders the 3D arena for the current level.
func (g *Game) drawWorld(screen *ebiten.Image, L *Level) {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	// camera with shake and gentle sway that follows the paddle
	sh2 := L.Shake
	if !g.settings.Shake {
		sh2 = 0
	}
	cam := &g.cam
	cam.Eye = V3{L.PaddleX * 0.12, 15, -10.5}
	cam.Target = V3{L.PaddleX * 0.05, 3, 17}
	if L.Designed {
		// a steeper view so the drawing reads clearly
		cam.Eye = V3{L.PaddleX * 0.1, 21, -6.5}
		cam.Target = V3{L.PaddleX * 0.04, 0, 17.5}
	}
	if L.introT > 0 {
		// boss intro: start low beside the Guardian and pull back to the
		// playing view as it finishes assembling
		k := 1 - L.introT/L.introMax
		e := smootherstep(clamp01((k - 0.18) / 0.78))
		orbit := 1 - k
		eye0 := V3{-7 + 4*k, 6.5 + 2*k, 1 + 3*k}
		tgt0 := V3{0, 3.8 + orbit*0.6, 24}
		cam.Eye = eye0.Add(cam.Eye.Sub(eye0).Mul(e))
		cam.Target = tgt0.Add(cam.Target.Sub(tgt0).Mul(e))
	}
	if L.finaleT > 0 {
		// push in on the last brick and ease back out
		p := 1 - L.finaleT/finaleDur
		e := math.Sin(math.Pi*p) * 0.4
		eye := V3{L.finX * 0.55, 9, L.finZ - 13}
		tgt := V3{L.finX, 0.8, L.finZ}
		cam.Eye = cam.Eye.Add(eye.Sub(cam.Eye).Mul(e))
		cam.Target = cam.Target.Add(tgt.Sub(cam.Target).Mul(e * 1.3))
	}
	if camHook != nil {
		camHook(cam, L)
	}
	if sh2 > 0 {
		cam.Eye = cam.Eye.Add(V3{(g.rng.Float64() - 0.5) * sh2, (g.rng.Float64() - 0.5) * sh2, 0})
	}
	cam.FOV = 60 * math.Pi / 180
	aspect := sw / sh
	if aspect < 1.7 { // narrower screens: widen to keep the arena in view
		cam.FOV = 2 * math.Atan(math.Tan(cam.FOV/2)*1.7/aspect)
	}
	cam.Setup(sw, sh)
	// sky, aligned so its horizon matches the 3D horizon
	classic := g.settings.Classic
	if classic && g.sky != nil {
		g.drawClassicBackdrop(screen, L)
	} else if g.sky != nil {
		_, hy, _ := cam.Project(V3{cam.Eye.X, 0, cam.Eye.Z + 4000})
		bw, bh := float64(g.sky.Bounds().Dx()), float64(g.sky.Bounds().Dy())
		sc := sw / bw
		fillRect(screen, 0, 0, sw, sh, prem(L.Biome.SkyTop, 1))
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(0, hy-bh*skyHorizon*sc)
		screen.DrawImage(g.sky, op)
		if !g.lowFX() {
			g.drawSkyFX(screen, L.Biome.ID, hy, L.T)
		}
	}
	B := g.batch
	B.Reset()
	bio := L.Biome
	B.fogColor = [3]float32{float32(bio.Fog.R) / 255, float32(bio.Fog.G) / 255, float32(bio.Fog.B) / 255}
	B.fogNear, B.fogFar = bio.FogNear, bio.FogFar
	B.SetLight(bio.LightDir, bio.LightCol, bio.AmbCol)

	if !classic {
		g.drawScenery(B, L)
	}

	// platform under the arena
	n0 := len(B.quads)
	B.Box(-HalfW-1.6, -0.6, -4, HalfW+1.6, -0.02, FarZ+1.2, darken(bio.Wall, 0.6), sprWhite, 0)
	for i := n0; i < len(B.quads); i++ {
		B.quads[i].depth += 3000
	}
	// floor tiles, clipped to the arena shape
	gp := 0.55 + 0.15*math.Sin(L.T*1.3)
	fspr := matFloor[bio.FloorMat]
	if bio.Neon {
		fspr = floorSprites[L.Vis.FloorPattern%len(floorSprites)]
	}
	for z := -3.0; z < FarZ+1; z += 2 {
		for x := -HalfW; x < HalfW; x += 2 {
			if !levelgen.Inside(L.Walls, levelgen.Pt{X: x + 1, Z: math.Min(z+1, FarZ-0.2)}) {
				continue
			}
			c := bio.Floor
			if bio.Neon {
				if int(math.Floor(z/2))%4 == 0 {
					c = mixc(bio.Floor, bio.FloorAlt, 0.6)
				}
			} else if (int(math.Floor(z/2))+int(math.Floor(x/2)))%2 == 0 {
				c = bio.FloorAlt
			}
			if bio.ID == WMachine && z < 1 {
				c = color.RGBA{230, 170, 40, 255} // hazard strip near the paddle
			}
			pts := [4]V3{{x, 0, z + 2}, {x + 2, 0, z + 2}, {x + 2, 0, z}, {x, 0, z}}
			B.Quad(pts, fspr, B.lit(c, V3{0, 1, 0}), [4]float32{1, 1, 1, 1}, false)
			B.quads[len(B.quads)-1].depth += 2000
			if bio.Neon {
				B.Quad(pts, fspr, alpha(c, 0.3*gp), [4]float32{1, 1, 1, 1}, true)
			}
		}
	}
	// glowing lane line near the paddle
	B.Decal(0, PaddleZ-1.3, 0.01, HalfW*2, 0.12, sprWhite, alpha(bio.Trim, 0.35), true)
	g.drawArenaSigil(B, L)
	// walls follow the arena outline
	wallH := 1.4
	wallTop, wallSide := matTop[bio.BrickMat], matSide[bio.BrickMat]
	for i := 0; i+1 < len(L.Walls); i++ {
		a, c := L.Walls[i], L.Walls[i+1]
		ex, ez := c.X-a.X, c.Z-a.Z
		ln := math.Hypot(ex, ez)
		ux, uz := ex/ln, ez/ln
		nx, nz := -uz, ux // inward normal
		if nx*(0-a.X)+nz*(10-a.Z) < 0 {
			nx, nz = -nx, -nz
		}
		steps := int(math.Ceil(ln / 2.2))
		for k := 0; k < steps; k++ {
			t0, t1 := float64(k)/float64(steps)*ln, float64(k+1)/float64(steps)*ln
			p0 := [2]float64{a.X + ux*t0, a.Z + uz*t0}
			p1 := [2]float64{a.X + ux*t1, a.Z + uz*t1}
			fp := [4][2]float64{p0, p1, {p1[0] - nx*1.4, p1[1] - nz*1.4}, {p0[0] - nx*1.4, p0[1] - nz*1.4}}
			if math.Abs(ux) < 0.01 || math.Abs(uz) < 0.01 {
				// axis-aligned segments get the textured material
				x0, x1 := math.Min(fp[0][0], fp[2][0]), math.Max(fp[0][0], fp[2][0])
				z0, z1 := math.Min(fp[0][1], fp[2][1]), math.Max(fp[0][1], fp[2][1])
				B.BoxTex(x0, 0, z0, x1, wallH, z1, bio.Wall, wallTop, wallSide, 0)
			} else {
				B.Prism(fp, 0, wallH, 1, bio.Wall, 0)
			}
			trimCol := bio.Trim
			B.Quad([4]V3{{p0[0], wallH + 0.01, p0[1]}, {p1[0], wallH + 0.01, p1[1]},
				{p1[0] - nx*0.2, wallH + 0.01, p1[1] - nz*0.2}, {p0[0] - nx*0.2, wallH + 0.01, p0[1] - nz*0.2}},
				sprWhite, trimCol, [4]float32{1, 1, 1, 1}, bio.Neon)
		}
	}
	// corner posts
	posts := []levelgen.Pt{}
	for i, v := range L.Walls {
		if i > 0 && i < len(L.Walls)-1 {
			posts = append(posts, v)
		}
	}
	for z := 0.0; z <= 12; z += 6.5 {
		posts = append(posts, levelgen.Pt{X: -HalfW - 0.7, Z: z}, levelgen.Pt{X: HalfW + 0.7, Z: z})
	}
	for _, v := range posts {
		g.drawPost(B, L, v.X, v.Z, wallH)
	}
	// barrier
	if L.Barrier > 0 {
		a := 0.45 + 0.2*math.Sin(L.T*6)
		B.Decal(0, BarrierZ, 0.05, HalfW*2, 0.35, sprSoft, alpha(color.RGBA{170, 140, 255, 255}, a), true)
		B.Decal(0, BarrierZ, 0.06, HalfW*2, 0.08, sprWhite, alpha(color.RGBA{220, 200, 255, 255}, a), true)
	}

	// bricks
	t := L.T + g.stateT
	for _, b := range L.Bricks {
		if !b.Alive {
			continue
		}
		g.drawBrick(B, L, b, t)
	}
	// boss eyes
	if L.Boss != nil && !L.Boss.Dead {
		for _, br := range L.Bricks {
			if br.Alive && br.Core {
				x, z := br.X(), br.Z()
				B.Billboard(V3{x, 2.6, z}, 4.5, sprSoft, alpha(bio.Trim, 0.4+0.3*math.Sin(t*5)), true)
			}
		}
		// eye sockets (row 5, cols 5 & 8 are empty in the shape); during
		// the intro they ignite once the face has formed
		lit := 1.0
		if L.introT > 0 {
			lit = smootherstep(clamp01((L.introMax - L.introT - 3.0) / 0.7))
		}
		if lit > 0 {
			eye := mixc(color.RGBA{255, 60, 40, 255}, lighten(bio.Trim, 0.3), 0.25)
			for _, c := range []int{5, 8} {
				ey := bossHeight(5, c) + 0.4
				p := V3{cellX(c), ey, cellZ(5)}
				B.Billboard(p, 6*lit, sprSoft, alpha(eye, (0.5+0.3*L.Boss.Hurt+0.1*math.Sin(t*4))*lit), true)
				B.Billboard(p, 2.2*lit, sprSoft, alpha(lighten(eye, 0.6), 0.95*lit), true)
				B.Billboard(p, 0.9*lit, sprSphere, lighten(eye, 0.5), false)
			}
		}
	}
	// rocks
	for _, rk := range L.Rocks {
		if rk.Warn > 0 {
			a := 0.4 + 0.4*math.Sin(rk.Warn*20)
			B.Decal(rk.TX, PaddleZ, 0.03, 2.2, 2.2, sprRing, alpha(color.RGBA{255, 60, 60, 255}, a), true)
			continue
		}
		s := 0.6
		B.DecalBelow(rk.X, rk.Z-0.2, 0.02, 1.8, 1.4, sprShadow, color.RGBA{0, 0, 0, 160})
		B.Box(rk.X-s, 0, rk.Z-s, rk.X+s, 1.1, rk.Z+s, color.RGBA{110, 100, 92, 255}, sprWhite, 0)
		B.Decal(rk.TX, PaddleZ, 0.03, 2, 2, sprRing, alpha(color.RGBA{255, 60, 60, 255}, 0.4), true)
	}
	// drops
	for _, d := range L.Drops {
		def := powerDefs[d.P]
		bob := 0.6 + 0.15*math.Sin(d.T*6)
		B.DecalBelow(d.X, d.Z-0.2, 0.02, 1.4, 0.8, sprShadow, color.RGBA{0, 0, 0, 140})
		B.Box(d.X-0.6, bob-0.25, d.Z-0.3, d.X+0.6, bob+0.25, d.Z+0.3, def.Col, sprWhite, 0.25)
		B.Billboard(V3{d.X, bob, d.Z}, 2.2, sprSoft, alpha(def.Col, 0.35), true)
	}
	// coins
	for _, c := range L.Coins {
		w := 0.55 * math.Abs(math.Cos(c.T*7))
		r := B.cam.right.Mul(math.Max(0.06, w) / 2)
		u := B.cam.up.Mul(0.55 / 2)
		p := V3{c.X, c.Y, c.Z}
		B.Quad([4]V3{p.Sub(r).Add(u), p.Add(r).Add(u), p.Add(r).Sub(u), p.Sub(r).Sub(u)}, sprCoin, color.RGBA{255, 255, 255, 255}, [4]float32{1, 1, 1, 1}, false)
		B.Billboard(p, 1, sprSoft, alpha(color.RGBA{255, 200, 60, 255}, 0.25), true)
	}
	// lasers
	for _, bl := range L.Bolts {
		if bl.Mult > 1 { // overclocked beam
			B.Box(bl.X-0.12, 0.48, bl.Z-1.2, bl.X+0.12, 0.66, bl.Z+0.6, color.RGBA{255, 230, 170, 255}, sprWhite, 0.9)
			B.Billboard(V3{bl.X, 0.55, bl.Z}, 2.4, sprSoft, alpha(color.RGBA{255, 150, 50, 255}, 0.8), true)
			continue
		}
		B.Box(bl.X-0.07, 0.5, bl.Z-0.5, bl.X+0.07, 0.62, bl.Z+0.5, color.RGBA{255, 220, 220, 255}, sprWhite, 0.8)
		B.Billboard(V3{bl.X, 0.55, bl.Z}, 1.2, sprSoft, alpha(color.RGBA{255, 60, 60, 255}, 0.6), true)
	}
	g.drawPaddle(B, L)
	// balls
	fire := L.Timers[PFire] > 0
	for _, b := range L.Balls {
		glow := color.RGBA{160, 130, 255, 255}
		core := color.RGBA{240, 235, 255, 255}
		if fire {
			glow, core = color.RGBA{255, 130, 40, 255}, color.RGBA{255, 230, 170, 255}
		} else if g.run.Stats.MeteorHeart {
			glow, core = color.RGBA{255, 90, 60, 255}, color.RGBA{255, 200, 170, 255}
		}
		if b.Storm > 0 {
			glow, core = color.RGBA{120, 190, 255, 255}, color.RGBA{225, 240, 255, 255}
		} else if g.run.Stats.ClassAttune && !fire {
			glow = attuneCols[L.cls.attuneEl]
		}
		a := 1.0
		if b.Phantom > 0 {
			a = 0.5
			glow = color.RGBA{180, 255, 240, 255}
		}
		B.DecalBelow(b.X+0.2, b.Z-0.2, 0.02, b.R*3, b.R*2.4, sprShadow, color.RGBA{0, 0, 0, 200})
		for i, p := range b.Trail {
			f := float64(i+1) / float64(len(b.Trail)+1)
			B.Billboard(V3{p[0], b.R, p[1]}, b.R*2*f, sprSoft, alpha(glow, 0.35*f*a), true)
		}
		B.Billboard(V3{b.X, b.R, b.Z}, b.R*6, sprSoft, alpha(glow, 0.35*a), true)
		B.Billboard(V3{b.X, b.R, b.Z}, b.R*2.05, sprSphere, alpha(core, a), false)
		if b.Charged {
			B.Billboard(V3{b.X, b.R, b.Z}, b.R*3.4, sprRing, alpha(color.RGBA{255, 90, 90, 255}, 0.8), true)
		}
	}
	// particles
	for _, p := range L.Parts {
		f := p.Life / p.Max
		switch p.Kind {
		case pDebris:
			// tumbling shards: a spinning prism, slightly flattened
			s := p.Size * (0.5 + 0.5*f)
			fp := rotQuad(p.P.X, p.P.Z, s*2, s*1.4, p.Rot)
			hh := s * (0.6 + 0.4*math.Abs(math.Sin(p.Rot*0.7)))
			B.Prism(fp, p.P.Y-hh, p.P.Y+hh, 0.8, p.Col, 0.05)
		case pSmoke:
			B.Billboard(p.P, p.Size*(1.5-f*0.5), sprSoft, alpha(p.Col, 0.35*f), false)
		case pAmbient:
			a := math.Min(1, math.Min(f*3, (1-f)*3))
			B.Billboard(p.P, p.Size, sprSoft, alpha(p.Col, 0.8*a), true)
		default:
			B.Billboard(p.P, p.Size*(0.4+0.6*f), sprSoft, alpha(p.Col, f), true)
		}
	}
	for _, rg := range L.Rings {
		f := 1 - rg.Life/rg.Max
		r := rg.MaxR * (0.3 + 0.7*f) * 2
		B.Decal(rg.X, rg.Z, 0.1, r, r, sprRing, alpha(rg.Col, 1-f), true)
		B.Decal(rg.X, rg.Z, 0.1, r*0.8, r*0.8, sprSoft, alpha(rg.Col, (1-f)*0.6), true)
	}
	B.Flush(screen)

	if !g.deferOverlay {
		g.drawWorldOverlay(screen, L, 1)
	}
}

// drawWorldOverlay draws the 2D layer over the world (lightning, capsule
// letters, floating text and flashes). k maps the world camera's pixels to
// dst's, so the overlay stays sharp when the 3D scene renders below
// native resolution.
func (g *Game) drawWorldOverlay(screen *ebiten.Image, L *Level, k float64) {
	cam := &g.cam
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	proj := func(p V3) (float64, float64) {
		x, y, _ := cam.Project(p)
		return x * k, y * k
	}
	// lightning arcs (2D, over everything)
	for _, a := range L.Arcs {
		al := a.Life / 0.3
		var prev [2]float64
		for i, p := range a.Pts {
			x, y := proj(p)
			if i > 0 {
				strokeLine(screen, prev[0], prev[1], x, y, 7*g.ui.S, prem(a.Col, 0.25*al))
				strokeLine(screen, prev[0], prev[1], x, y, 2.5*g.ui.S, prem(color.RGBA{240, 250, 255, 255}, al))
			}
			prev = [2]float64{x, y}
		}
	}
	// capsule letters
	for _, d := range L.Drops {
		x, y := proj(V3{d.X, 0.95 + 0.15*math.Sin(d.T*6), d.Z})
		g.ui.textAt(screen, powerDefs[d.P].Letter, fBold, 20, x, y-13*g.ui.S, color.RGBA{20, 20, 30, 255}, alignCenter)
	}
	// floating texts
	for _, f := range L.Floats {
		x, y := proj(f.P)
		size := 22.0
		if f.Big {
			size = 34
		}
		if f.Scale > 0 {
			// damage numbers pop in large and settle
			pop := 1 + 0.35*clamp01((f.Life-0.75)/0.2)
			size = f.Scale * pop
		}
		g.ui.textOutlinedAt(screen, f.S, fBold, size, x, y, alpha(f.Col, math.Min(1, f.Life*2)), alignCenter)
	}
	if L.Flash > 0 && g.settings.Flashes {
		fillRect(screen, 0, 0, sw, sh, prem(color.RGBA{255, 240, 220, 255}, L.Flash*0.5))
	}
	if L.hurtFlash > 0 && g.settings.Flashes {
		fillRect(screen, 0, 0, sw, sh, prem(color.RGBA{255, 0, 0, 255}, L.hurtFlash*0.22))
	}
}

func (g *Game) drawBrick(B *Batch, L *Level, b *Brick, t float64) {
	x, z := b.X()+b.OffX, b.Z()+b.OffZ
	ap := clamp01(b.Appear)
	if ap <= 0 {
		return
	}
	ap = 1 - (1-ap)*(1-ap)
	hw, hd := CellW/2*0.94, CellD/2*0.9
	frac := b.HP / b.MaxHP
	y0 := float64(b.Layer)*LayerH + b.FallY
	h := LayerH * 0.94
	if b.Kind == levelgen.Elite {
		h = 1.7
	}
	if b.Kind != levelgen.Steel && b.Layer == 0 && b.Up == nil {
		h *= 0.8 + 0.2*frac
	}
	if b.BossPart {
		// the Guardian is sculpted: a domed head and heavy shoulders
		h = bossHeight(b.R, b.C) * (0.85 + 0.15*frac)
		if b.Core {
			h += 0.5
		}
	}
	h *= ap
	col := b.Col
	if b.Kind == levelgen.Basic && b.MaxHP > 1 {
		col = darken(col, 0.78+0.22*frac)
	}
	emis := float32(0)
	if b.Frozen > 0 {
		col = mixc(col, color.RGBA{200, 240, 255, 255}, 0.6)
		emis = 0.1
	}
	if b.Burn > 0 {
		col = mixc(col, color.RGBA{255, 120, 30, 255}, 0.35+0.15*math.Sin(t*20))
		emis = 0.2
	}
	switch b.Kind {
	case levelgen.Elite:
		emis = 0.15 + 0.1*float32(math.Sin(t*3))
	case levelgen.Crystal:
		emis = 0.2
	}
	if b.Core {
		emis = 0.4 + 0.3*float32(math.Sin(t*6))
	}
	if b.Flash > 0 {
		col = mixc(col, color.RGBA{255, 255, 255, 255}, b.Flash*0.6)
	}
	if b.Kind == levelgen.Regen && b.HP < b.MaxHP && b.SinceHit > 3 {
		emis += 0.15
	}
	if b.Layer == 0 {
		B.DecalBelow(x+0.5, z-0.15, 0.015, CellW*1.15, CellD*1.4, sprShadow, color.RGBA{0, 0, 0, 120})
	}
	bio := L.Biome
	if b.BossPart && !b.Core {
		// carved armour with glowing rune seams in the world's trim colour
		pulse := 0.55 + 0.45*math.Sin(t*2.2-float64(b.R)*0.7)
		B.BoxTex(x-hw, y0, z-hd, x+hw, y0+h, z+hd, col, matTop[bio.BrickMat], matSide[bio.BrickMat], emis+0.04)
		seam := lighten(bio.Trim, 0.2)
		B.Box(x-hw-0.03, y0+h*0.62, z-hd-0.03, x+hw+0.03, y0+h*0.68, z+hd+0.03, seam, sprWhite, float32(0.5+0.4*pulse))
		B.Decal(x, z, y0+h+0.01, CellW*0.9, CellD*0.9, sprSoft, alpha(seam, 0.22*pulse*ap), true)
		if b.R-2 <= 1 && ap > 0.5 {
			// a crown of crystal spikes along the brow
			sh := (1.4 + 1.6*(1-math.Abs(float64(b.C)-6.5)/6.5)) * ap
			q := 0.42
			B.Prism([4][2]float64{{x - q, z - q*0.7}, {x + q, z - q*0.7}, {x + q, z + q*0.7}, {x - q, z + q*0.7}}, y0+h, y0+h+sh, 0.08, lighten(bio.Trim, 0.1), float32(0.35+0.25*pulse))
		}
		g.drawBadge(B, b, x, y0+h, z, t)
		return
	}
	mat := bio.BrickMat
	special := b.Kind != levelgen.Basic && b.Kind != levelgen.Steel
	switch {
	case b.Kind == levelgen.Steel:
		B.BoxTex(x-hw, y0, z-hd, x+hw, y0+h, z+hd, col, matTop[WMachine], matSide[WMachine], emis)
	case bio.Taper < 1 && !special:
		// faceted gemstone bricks
		B.Prism([4][2]float64{{x - hw, z - hd}, {x + hw, z - hd}, {x + hw, z + hd}, {x - hw, z + hd}}, y0, y0+h*0.75, 1, col, emis+0.12)
		B.Prism([4][2]float64{{x - hw, z - hd}, {x + hw, z - hd}, {x + hw, z + hd}, {x - hw, z + hd}}, y0+h*0.75, y0+h, bio.Taper, lighten(col, 0.15), emis+0.2)
	default:
		e := emis
		if bio.Neon {
			e += 0.12
		}
		B.BoxTex(x-hw, y0, z-hd, x+hw, y0+h, z+hd, col, matTop[mat], matSide[mat], e)
	}
	if bio.Neon && b.Layer == 0 {
		B.Decal(x, z, 0.02, CellW*1.1, CellD*1.3, sprSoft, alpha(col, 0.25), true)
	}
	// HP trim: silver for 2, gold for 3, crimson for 4+
	if b.MaxHP >= 2 && b.Kind == levelgen.Basic {
		trim := []color.RGBA{{220, 225, 240, 255}, {255, 210, 90, 255}, {255, 80, 90, 255}}[min(2, int(b.MaxHP)-2)]
		B.Box(x-hw-0.02, y0+h*0.42, z-hd-0.02, x+hw+0.02, y0+h*0.58, z+hd+0.02, trim, sprWhite, 0.25)
	}
	h = y0 + h
	g.drawBadge(B, b, x, h, z, t)
	if ic, ok := kindIcon[b.Kind]; ok && false {
		icCol := color.RGBA{255, 255, 255, 230}
		switch b.Kind {
		case levelgen.Explosive:
			icCol = color.RGBA{30, 20, 20, 255}
		case levelgen.Mystery:
			icCol = color.RGBA{80, 40, 10, 255}
		case levelgen.Elite:
			icCol = color.RGBA{255, 210, 90, 255}
		}
		B.Decal(x, z, h+0.004, 1.0, 0.9, sprIcons[ic], icCol, false)
		B.quads[len(B.quads)-1].depth -= 0.02
	}
	if b.Kind == levelgen.Crystal {
		bob := h + 0.7 + 0.15*math.Sin(t*3+x)
		B.Billboard(V3{x, bob, z}, 2.2, sprSoft, alpha(crystalGlow, 0.35), true)
	}
	if b.Kind == levelgen.Explosive && math.Mod(t*3+x, 1) < 0.5 {
		B.Billboard(V3{x + 0.45, h + 0.35, z + 0.2}, 0.35, sprSoft, color.RGBA{255, 220, 90, 255}, true)
	}
	if b.Kind == levelgen.Shield {
		B.Billboard(V3{x, h, z}, 4.4, sprRing, alpha(color.RGBA{120, 220, 255, 255}, 0.35+0.2*math.Sin(t*3)), true)
	} else if b.shieldFlash > 0 {
		B.Billboard(V3{x, h / 2, z}, 2.4, sprRing, alpha(color.RGBA{120, 220, 255, 255}, b.shieldFlash), true)
	}
	if b.Frozen > 0 {
		B.Billboard(V3{x, h, z}, 1.6, sprSpark, alpha(color.RGBA{220, 245, 255, 255}, 0.5), true)
	}
	if b.Kind == levelgen.Elite || b.Core {
		B.Billboard(V3{x, h + 0.2, z}, 3.2, sprSoft, alpha(color.RGBA{200, 120, 255, 255}, 0.25), true)
	}
}

func (g *Game) drawPaddle(B *Batch, L *Level) {
	shown := L.paddleShown()
	if shown <= 0 {
		return
	}
	w := L.PaddleW * (0.4 + 0.6*shown)
	x0, x1 := L.PaddleX-w/2, L.PaddleX+w/2
	z0, z1 := PaddleZ-PaddleD/2, PaddleZ+PaddleD/2
	body := L.Biome.PaddleBody
	trim := L.Biome.Trim
	if L.Timers[PGlue] > 0 {
		body = color.RGBA{60, 120, 60, 255}
	}
	if L.stun > 0 {
		body = color.RGBA{120, 60, 60, 255}
	}
	lift := 0.08 * L.paddleBounce
	B.DecalBelow(L.PaddleX+0.3, PaddleZ-0.25, 0.02, w+1.6, 1.8, sprShadow, color.RGBA{0, 0, 0, 200})
	B.Box(x0+0.5, lift, z0, x1-0.5, PaddleH+lift, z1, body, sprWhite, 0.05)
	cap := L.Biome.PaddleCap
	B.Box(x0, lift, z0-0.05, x0+0.6, PaddleH+0.12+lift, z1+0.05, cap, sprWhite, 0.05)
	B.Box(x1-0.6, lift, z0-0.05, x1, PaddleH+0.12+lift, z1+0.05, cap, sprWhite, 0.05)
	B.Decal(L.PaddleX, z1-0.02, PaddleH+lift+0.01, w-1.3, 0.16, sprWhite, alpha(lighten(trim, 0.4), 0.9), true)
	B.Billboard(V3{L.PaddleX, PaddleH + 0.2, PaddleZ}, w*0.9, sprSoft, alpha(trim, 0.18+0.25*L.paddleBounce), true)
	if L.Timers[PLaser] > 0 || g.run.Stats.LaserInterval > 0 {
		for _, cx := range []float64{x0 + 0.3, x1 - 0.3} {
			B.Box(cx-0.15, PaddleH, PaddleZ-0.15, cx+0.15, PaddleH+0.5, PaddleZ+0.25, color.RGBA{90, 90, 100, 255}, sprWhite, 0)
			B.Billboard(V3{cx, PaddleH + 0.55, PaddleZ + 0.25}, 0.5, sprSoft, color.RGBA{255, 80, 80, 255}, true)
		}
	}
}

// bossHeight gives each cell of the Guardian its sculpted height.
func bossHeight(r, c int) float64 {
	gr := float64(r - 2) // row within the shape, 0 = crown
	if gr >= 6 {
		return 1.25 // legs
	}
	dx := math.Abs(float64(c)-6.5) / 6.5
	dome := 1 - dx*dx
	return 1.0 + 3.0*dome*(0.5+0.5*(1-gr/6))
}

// drawClassicBackdrop is the "classic" look: the world's painting fills
// the screen as a still picture behind the arena, like the arcade
// brick-breakers of the early 2000s.
func (g *Game) drawClassicBackdrop(screen *ebiten.Image, L *Level) {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	bw, bh := float64(g.sky.Bounds().Dx()), float64(g.sky.Bounds().Dy())
	sc := math.Max(sw/bw, sh/bh) * 1.04
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate((sw-bw*sc)/2, (sh-bh*sc)*0.35)
	op.ColorScale.Scale(0.86, 0.86, 0.9, 1) // keep the play field readable
	screen.DrawImage(g.sky, op)
	vignette(screen, sw, sh, 210)
}
