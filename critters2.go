package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Mischief-makers: critters with a job, in the spirit of Magic Ball's board
// creatures. A Pilferer runs down falling power-ups and makes off with them,
// a Mason lays fresh bricks in front of the formation, and a bomber flies
// over the bricks with a bomb that drops wherever the orb strikes it. Each
// is worth hitting for a different reason.

const (
	critWander = iota // bursts into coins
	critThief
	critMason
	critBomber
)

// spawnMischief tries a mischief-maker when there is something for it to
// do; it reports whether one appeared.
func (L *Level) spawnMischief() bool {
	x := L.rng.Float64()
	switch {
	case len(L.Drops) > 0 && x < 0.45 && !L.hasCritter(critThief):
		return L.spawnThief(L.Drops[0])
	case x < 0.2:
		return L.spawnBomber()
	case x < 0.36 && L.g.run.Act >= 2 && !L.hasCritter(critMason):
		return L.spawnMason()
	}
	return false
}

func (L *Level) hasCritter(kind int) bool {
	for _, c := range L.Critters {
		if c.Kind == kind && c.gone <= 0 {
			return true
		}
	}
	return false
}

func critterKindCol(world, kind int) color.RGBA {
	// per world, in world-ID order: Neon, Meadow, Caverns, Ruins, Clouds, Machine
	w := ((world % NumWorlds) + NumWorlds) % NumWorlds
	switch kind {
	case critThief:
		return [NumWorlds]color.RGBA{WNeon: {255, 80, 200, 255}, WMeadow: {150, 110, 80, 255}, WCaverns: {120, 100, 170, 255}, WRuins: {200, 150, 90, 255}, WClouds: {200, 200, 230, 255}, WMachine: {150, 150, 160, 255}}[w]
	case critMason:
		return [NumWorlds]color.RGBA{WNeon: {110, 90, 170, 255}, WMeadow: {150, 140, 120, 255}, WCaverns: {90, 110, 150, 255}, WRuins: {210, 180, 130, 255}, WClouds: {225, 230, 245, 255}, WMachine: {140, 120, 100, 255}}[w]
	case critBomber:
		return [NumWorlds]color.RGBA{WNeon: {80, 240, 255, 255}, WMeadow: {40, 36, 44, 255}, WCaverns: {110, 70, 150, 255}, WRuins: {120, 80, 50, 255}, WClouds: {245, 245, 250, 255}, WMachine: {200, 160, 80, 255}}[w]
	}
	return critterCol(world, false)
}

// spawnThief sends a Pilferer after a falling power-up, from the wall
// nearest to it and a little ahead of where it falls.
func (L *Level) spawnThief(target *Drop) bool {
	z0, z1, ok := L.critterBand()
	if !ok {
		return false
	}
	for try := 0; try < 10; try++ {
		side := float64(L.rng.Intn(2)*2 - 1)
		if target != nil && math.Abs(target.X) > 1.5 && try < 5 {
			side = math.Copysign(1, target.X)
		}
		x := side * (HalfW - 2 - float64(try%3)*1.5)
		z := z0 + L.rng.Float64()*(z1-z0)
		if target != nil && try < 5 {
			z = math.Max(z0, math.Min(z1, target.Z-6))
		}
		if !L.critterInside(x, z) {
			continue
		}
		L.Critters = append(L.Critters, &Critter{Kind: critThief, X: x, Z: z, H: math.Atan2(-side, 0), Speed: 6.2, Life: 16, enter: 0.6})
		L.sparks(V3{x, 0.5, z}, 8, critterKindCol(L.Biome.ID, critThief), 3)
		L.g.coach("thief")
		return true
	}
	return false
}

// spawnMason brings a builder who lays fresh bricks in front of the formation.
func (L *Level) spawnMason() bool {
	z0, z1, ok := L.critterBand()
	if !ok {
		return false
	}
	c := &Critter{Kind: critMason, Speed: 1.7, Life: 30}
	if !L.masonTarget(c) {
		return false
	}
	side := float64(L.rng.Intn(2)*2 - 1)
	c.Z = math.Min(z1, math.Max(z0, cellZ(c.tr)-CellD))
	placed := false
	for k := 0; k < 6 && !placed; k++ {
		c.X = side * (HalfW - 2 - float64(k)*1.5)
		placed = L.critterInside(c.X, c.Z)
	}
	if !placed {
		return false
	}
	L.Critters = append(L.Critters, c)
	L.sparks(V3{c.X, 0.5, c.Z}, 8, critterKindCol(L.Biome.ID, critMason), 3)
	L.g.coach("mason")
	return true
}

// masonTarget picks an empty cell right in front of a column of the
// formation, inside the walls and well clear of the danger line.
func (L *Level) masonTarget(c *Critter) bool {
	type cell struct{ r, c int }
	var opts []cell
	for col := 0; col < levelgen.Cols; col++ {
		for r := levelgen.MaxRows - 1; r >= 1; r-- {
			if b := L.Grid[r][col]; b != nil && (b.Alive || b.Up != nil) {
				break // this column reaches further forward than any gap
			}
			a := L.Grid[r-1][col]
			if a == nil || !a.Alive || a.BossPart {
				continue
			}
			z := cellZ(r)
			if z-CellD/2 >= dangerZ+CellD*2.5 && levelgen.Inside(L.Walls, levelgen.Pt{X: cellX(col), Z: z}) {
				opts = append(opts, cell{r, col})
			}
			break
		}
	}
	if len(opts) == 0 {
		return false
	}
	o := opts[L.rng.Intn(len(opts))]
	c.tr, c.tc, c.buildT = o.r, o.c, 0
	return true
}

// spawnBomber sends a bird across the thick of the formation with a bomb.
func (L *Level) spawnBomber() bool {
	front := L.frontZ()
	if front >= FarZ-1 {
		return false
	}
	best, bestN, bestS := front, -1, -1e9
	for z := front; z < FarZ-1; z += CellD {
		r := rowAt(z)
		if r < 0 || r >= levelgen.MaxRows {
			continue
		}
		n := 0
		for col := 0; col < levelgen.Cols; col++ {
			if b := L.Grid[r][col]; b != nil && b.Alive {
				n++
			}
		}
		// the thick of the bricks, leaning toward the front where it reads best
		if sc := float64(n) - 0.6*(z-front)/CellD; n > 1 && sc > bestS {
			best, bestN, bestS = z, n, sc
		}
	}
	if bestN <= 1 {
		return false
	}
	side := float64(L.rng.Intn(2)*2 - 1)
	c := &Critter{Kind: critBomber, X: side * (HalfW + 1.5), Z: best, z0: best, vx: -side * (2.6 + L.rng.Float64()*0.8), Life: 20, enter: 1}
	L.Critters = append(L.Critters, c)
	L.g.coach("bomber")
	L.g.snd.PlayVol("warp", 0.35)
	return true
}

// updateSpecial moves a mischief-maker; it reports true when it handled
// the critter this frame.
func (L *Level) updateSpecial(c *Critter, dt, z0, z1 float64, bandOK bool) bool {
	leave := func() {
		c.gone = 0.45
		L.sparks(V3{c.X, 0.5, c.Z}, 10, critterKindCol(L.Biome.ID, c.Kind), 4)
	}
	walk := func(tx, tz, sp float64) bool {
		dx, dz := tx-c.X, tz-c.Z
		d := math.Hypot(dx, dz)
		if d < 0.15 {
			return true
		}
		c.H = math.Atan2(dx, dz)
		step := math.Min(d, sp*dt)
		nx, nz := c.X+dx/d*step, c.Z+dz/d*step
		switch {
		case L.critterInside(nx, nz):
			c.X, c.Z = nx, nz
		case L.critterInside(nx, c.Z):
			c.X = nx
		case L.critterInside(c.X, nz):
			c.Z = nz
		}
		c.hop += dt * sp * 3.2
		return false
	}
	flee := func(sp float64) {
		side := 1.0
		if c.X < 0 {
			side = -1
		}
		walk(side*(HalfW+2), c.Z, sp)
		if math.Abs(c.X) > HalfW-1.6 || !L.critterInside(c.X+side*0.3, c.Z) {
			leave()
		}
	}
	if c.Kind != critBomber && (L.Cleared || c.T > c.Life+6) {
		leave()
		return true
	}
	switch c.Kind {
	case critThief:
		if c.fleeing || c.T > c.Life {
			flee(c.Speed * 0.85)
			return true
		}
		// chase the nearest capsule it can reach
		var best *Drop
		bd := 1e9
		for _, d := range L.Drops {
			if d.Z < z0-0.5 || (bandOK && d.Z > z1+3) {
				continue
			}
			if dd := math.Hypot(d.X-c.X, d.Z-c.Z); dd < bd {
				best, bd = d, dd
			}
		}
		if best == nil {
			if len(c.carry) > 0 || c.T > 4 {
				c.fleeing = true
			}
			walk(c.X+math.Sin(c.T)*2, (z0+z1)/2, c.Speed*0.5)
			return true
		}
		walk(best.X, math.Max(z0, math.Min(z1, best.Z)), c.Speed)
		if bd < 1.2 {
			for i, d := range L.Drops {
				if d == best {
					L.Drops = append(L.Drops[:i], L.Drops[i+1:]...)
					break
				}
			}
			L.dropsSeen = len(L.Drops)
			c.carry = append(c.carry, best.P)
			L.text(V3{c.X, 1.5, c.Z}, "SNATCHED!", color.RGBA{255, 150, 120, 255}, false)
			L.g.snd.PlayVol("pwbad", 0.6)
			c.fleeing = true
		}
		return true
	case critMason:
		if c.fleeing || c.T > c.Life {
			flee(c.Speed * 1.4)
			return true
		}
		// the target may have been filled, or pushed too close by an advance
		b, a := L.Grid[c.tr][c.tc], L.Grid[c.tr-1][c.tc]
		if (b != nil && (b.Alive || b.Up != nil)) || a == nil || !a.Alive || cellZ(c.tr)-CellD/2 < dangerZ+CellD*2 {
			if !L.masonTarget(c) {
				c.fleeing = true
			}
			return true
		}
		tx, tz := cellX(c.tc), cellZ(c.tr)-CellD*0.95
		if bandOK {
			tz = math.Max(z0, math.Min(z1, tz))
		}
		if !walk(tx, tz, c.Speed) && math.Hypot(tx-c.X, tz-c.Z) > 0.6 {
			c.buildT = 0
			return true
		}
		c.H = 0 // face the formation while building
		c.buildT += dt
		if L.rng.Float64() < dt*8 {
			L.sparks(V3{cellX(c.tc), 0.6, cellZ(c.tr)}, 1, critterKindCol(L.Biome.ID, critMason), 2)
		}
		if c.buildT >= 2.6 {
			bio := L.Biome
			nb := L.addBrick(c.tr, c.tc, levelgen.Basic, math.Ceil(L.hpMul()), bio.Hues[L.rng.Intn(6)])
			nb.Appear = 0
			L.sparks(V3{cellX(c.tc), 0.8, cellZ(c.tr)}, 12, lighten(nb.Col, 0.3), 5)
			L.g.snd.BrickHit()
			c.built++
			c.buildT = 0
			if c.built >= 3 || !L.masonTarget(c) {
				c.fleeing = true
			}
		}
		return true
	case critBomber:
		c.X += c.vx * dt
		c.Z = c.z0 + math.Sin(c.T*1.3)*0.8
		c.H = math.Atan2(c.vx, 0)
		if (math.Abs(c.X) > HalfW+3 && c.T > 1) || c.T > c.Life {
			c.gone = 0.01
		}
		return true
	}
	return false
}

// popMischief is what each mischief-maker gives up when the orb hits it.
func (L *Level) popMischief(c *Critter) {
	g := L.g
	col := critterKindCol(L.Biome.ID, c.Kind)
	L.sparks(V3{c.X, 0.7, c.Z}, 18, col, 6)
	L.Rings = append(L.Rings, &Ring{X: c.X, Z: c.Z, MaxR: 1.8, Life: 0.3, Max: 0.3, Col: col})
	L.Shake = math.Max(L.Shake, 0.15)
	switch c.Kind {
	case critThief:
		// everything it stole comes tumbling back, with a tip for your trouble
		for i, p := range c.carry {
			L.Drops = append(L.Drops, &Drop{X: c.X + float64(i)*0.6 - float64(len(c.carry)-1)*0.3, Z: c.Z, P: p})
		}
		L.dropsSeen = len(L.Drops)
		L.gainGold(4, c.X, c.Z, true)
		if len(c.carry) > 0 {
			L.text(V3{c.X, 1.6, c.Z}, "STOP, THIEF!", color.RGBA{255, 220, 120, 255}, true)
		}
		g.snd.Play("coinbig")
	case critMason:
		L.gainGold(5, c.X, c.Z, true)
		if L.rng.Float64() < 0.3 {
			L.dropPower(c.X, c.Z, true)
		}
		L.text(V3{c.X, 1.6, c.Z}, "SHOO!", color.RGBA{230, 210, 170, 255}, false)
		g.snd.Play("coinbig")
	case critBomber:
		// the bomb drops right where it was struck
		L.explode(c.X, c.Z, 1.8, math.Max(3, 3*L.hpMul()), true)
		L.gainGold(2, c.X, c.Z, true)
		L.text(V3{c.X, 1.8, c.Z}, "BOMBS AWAY!", color.RGBA{255, 170, 80, 255}, true)
		g.snd.Play("boom")
	}
}

// drawMischief draws a mischief-maker at scale s.
func (g *Game) drawMischief(B *Batch, L *Level, c *Critter, s float64, rot Mat3, at func(x, y, z float64) V3) {
	col := critterKindCol(L.Biome.ID, c.Kind)
	ball := func(key string, p V3, sc V3, c1, c2 color.RGBA, e float32) {
		B.Draw(cached("crit"+key+colKey(c1), func() *Mesh { return Icosphere(1, 0.03, 5, c1, c2) }), p, sc.Mul(s), rot, color.RGBA{}, e)
	}
	eye := func(p V3, sz float64) {
		B.Billboard(p, sz*s, sprSphere, color.RGBA{20, 16, 24, 255}, false)
		B.Billboard(p.Add(V3{0.03, 0.04, -0.05}), sz*0.4*s, sprSphere, color.RGBA{255, 255, 255, 255}, false)
	}
	switch c.Kind {
	case critThief: // a masked pilferer with a swag bag
		bob := math.Abs(math.Sin(c.hop)) * 0.18
		if m := model("critter_thief"); m != nil {
			B.Draw(m, V3{c.X, bob * s, c.Z}, v3(s/critterSize), rot, col, 0)
		} else {
			ball("thf", at(0, 0.45+bob, 0), V3{0.34, 0.36, 0.38}, col, darken(col, 0.7), 0)
			ball("thf", at(0, 0.88+bob, 0.12), V3{0.26, 0.24, 0.26}, col, darken(col, 0.7), 0)
			B.OBox(at(0, 0.92+bob, 0.33), rot.Mul(V3{0.24, 0, 0}).Mul(s), V3{0, 0.05 * s, 0}, rot.Mul(V3{0, 0, 0.04}).Mul(s), color.RGBA{20, 18, 24, 255}, 0)
			for _, sx := range []float64{-1, 1} {
				eye(at(sx*0.1, 0.93+bob, 0.37), 0.08)
				B.OBox(at(sx*0.16, 1.12+bob, 0.1), rot.Mul(V3{0.05, 0, 0}).Mul(s), V3{0, 0.08 * s, 0}, rot.Mul(V3{0, 0, 0.04}).Mul(s), darken(col, 0.8), 0)
			}
		}
		// the bag swells with every theft and glows with what is inside
		bag := 0.18 + 0.08*float64(len(c.carry))
		bc := color.RGBA{170, 140, 90, 255}
		ball("bag", at(0, 0.6+bob, -0.42), V3{bag, bag * 1.1, bag}, bc, darken(bc, 0.7), 0)
		if len(c.carry) > 0 {
			B.Billboard(at(0, 0.65+bob, -0.42), 1.4*s, sprSoft, alpha(color.RGBA{255, 230, 140, 255}, 0.4+0.2*math.Sin(c.T*10)), true)
		}
	case critMason: // a little stone golem in a hard hat
		bob := math.Abs(math.Sin(c.hop)) * 0.1
		if m := model("critter_mason"); m != nil {
			B.Draw(m, V3{c.X, bob * s, c.Z}, v3(s/critterSize), rot, col, 0)
		} else {
			ball("msn", at(0, 0.5+bob, 0), V3{0.4, 0.42, 0.34}, col, darken(col, 0.65), 0)
			ball("msnh", at(0, 1.0+bob, 0.05), V3{0.24, 0.22, 0.22}, lighten(col, 0.1), darken(col, 0.6), 0)
			eye(at(-0.09, 1.02+bob, 0.25), 0.07)
			eye(at(0.09, 1.02+bob, 0.25), 0.07)
			hat := color.RGBA{255, 200, 60, 255}
			ball("hat", at(0, 1.2+bob, 0.03), V3{0.25, 0.12, 0.25}, hat, darken(hat, 0.75), 0.1)
		}
		if c.buildT > 0 {
			// hammering: the arm swings and a ring fills over the cell
			sw := math.Sin(c.T*16) * 0.25
			B.OBox(at(0.36, 0.8+sw, 0.25), rot.Mul(V3{0.06, 0, 0}).Mul(s), V3{0, 0.22 * s, 0}, rot.Mul(V3{0, 0, 0.06}).Mul(s), darken(col, 0.8), 0)
			k := clamp01(c.buildT / 2.6)
			B.Decal(cellX(c.tc), cellZ(c.tr), 0.05, CellW*0.9*k, CellD*0.8*k, sprRing, alpha(color.RGBA{255, 220, 140, 255}, 0.7), true)
		} else {
			// carrying the next brick on its head
			bc := L.Biome.Hues[(c.built+2)%6]
			p := at(0, 1.45+bob, 0)
			B.Box(p.X-0.32*s, p.Y-0.1*s, p.Z-0.18*s, p.X+0.32*s, p.Y+0.1*s, p.Z+0.18*s, bc, sprWhite, 0.05)
		}
	case critBomber: // a bird with a fizzing bomb in its claws
		k := s / critterSize
		p := V3{c.X, 2.0 + math.Sin(c.T*5)*0.12, c.Z}
		B.Billboard(p, 3.6*k, sprSoft, alpha(lighten(col, 0.3), 0.35), true)
		if bb, bw := model("bird_body"), model("bird_wing"); bb != nil && bw != nil {
			ry := math.Atan2(c.vx, 0)
			fl := math.Sin(c.T*9) * 0.7
			ks := 2 * k
			e := float32(0.25)
			if L.Biome.Neon || L.Biome.ID == WCaverns {
				e = 0.8
			}
			B.Draw(bb, p, v3(ks), rotXYZ(0, ry, 0), col, e)
			B.Draw(bw, p.Add(V3{0, 0.1, 0}), v3(ks), rotXYZ(0, ry, fl), col, e)
			B.Draw(bw, p.Add(V3{0, 0.1, 0}), v3(ks), rotXYZ(0, ry, math.Pi-fl), col, e)
			goto bomb
		}
		{
			// the bird: a body and two beating wings, lit so it reads over any floor
			dir := V3{math.Copysign(1, c.vx), 0, 0}
			perp := V3{0, 0, 1}
			bs := 3.4 * k
			em := float32(0.35)
			if L.Biome.Neon || L.Biome.ID == WCaverns {
				em = 0.85 // dark worlds: let it glow like everything else there
			}
			flap := math.Sin(c.T*9) * 0.8
			B.OBox(p, dir.Mul(0.32*bs), V3{0, 0.11 * bs, 0}, perp.Mul(0.12*bs), col, em)
			B.OBox(p.Add(dir.Mul(0.36*bs)).Add(V3{0, 0.05 * bs, 0}), dir.Mul(0.1*bs), V3{0, 0.09 * bs, 0}, perp.Mul(0.09*bs), lighten(col, 0.15), em)
			B.OBox(p.Add(dir.Mul(0.5*bs)), dir.Mul(0.08*bs), V3{0, 0.025 * bs, 0}, perp.Mul(0.03*bs), color.RGBA{255, 190, 60, 255}, 0.4)
			for _, sg := range []float64{-1, 1} {
				w := perp.Mul(sg * 0.42 * bs).Add(V3{0, flap * 0.3 * bs, 0})
				B.OBox(p.Add(w), w, V3{0, 0.025 * bs, 0}, dir.Mul(0.18*bs), darken(col, 0.85), em)
			}
		}
	bomb:
		bp := p.Add(V3{0, -0.75, 0})
		B.Billboard(bp, 1.1*k, sprSphere, color.RGBA{40, 36, 44, 255}, false)
		if math.Mod(c.T*6, 1) < 0.5 {
			B.Billboard(bp.Add(V3{0.1, 0.4, 0}), 0.5, sprSoft, color.RGBA{255, 220, 90, 255}, true)
		}
		B.DecalBelow(c.X, c.Z, 0.02, 1.6, 1.0, sprShadow, color.RGBA{0, 0, 0, 110})
		// a target ring hints where the bomb would land
		rr := 1.8 * CellW * 1.9
		B.Decal(c.X, c.Z, 0.04, rr, rr, sprRing, alpha(color.RGBA{255, 170, 80, 255}, 0.35+0.15*math.Sin(c.T*6)), true)
	}
}
