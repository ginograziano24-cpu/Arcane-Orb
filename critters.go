package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Critters: little creatures that wander the open floor between the bricks
// and the paddle, one kind per world, in the spirit of the animals that
// roam Magic Ball's boards. The orb bounces off them and they burst into
// coins (and sometimes a power-up); a rare golden one pays far more. They
// never hurt you; they're a moving target worth aiming for. Duels and
// Guardian fights have none.

type Critter struct {
	Kind    int     // critWander, critThief, critMason, critBomber (critters2.go)
	X, Z, H float64 // position and heading (radians, 0 = +Z)
	Speed   float64
	T, Life float64
	turnT   float64
	hop     float64 // animation phase
	Golden  bool
	gone    float64 // >0 while it pops or leaves
	popped  bool
	enter   float64 // 0..1 fade/scale in
	scareT  float64 // brief dash after a near miss

	carry   []Power // thief: what it has stolen
	fleeing bool    // thief/mason: heading for the edge
	tr, tc  int     // mason: the cell it is building on
	buildT  float64 // mason: progress on the current brick
	built   int
	vx      float64 // bomber: flight speed along x
	z0      float64 // bomber: flight line
}

const critterR = 0.95

// critterSize scales the models up from their unit design.
const critterSize = 1.8

// critterBand is the strip of open floor critters may walk: clear of the
// paddle and in front of the formation. ok is false when it's too narrow.
func (L *Level) critterBand() (z0, z1 float64, ok bool) {
	z0 = PaddleZ + 4.5
	z1 = math.Min(L.frontZ()-CellD*0.9, FarZ-3)
	return z0, z1, z1-z0 >= 2.5
}

func (L *Level) critterInside(x, z float64) bool {
	m := critterR + 0.4
	for _, d := range [][2]float64{{m, 0}, {-m, 0}, {0, m}, {0, -m}} {
		if !levelgen.Inside(L.Walls, levelgen.Pt{X: x + d[0], Z: z + d[1]}) {
			return false
		}
	}
	return true
}

func (L *Level) updateCritters(dt float64) {
	r := L.g.run
	if r.Duel || L.Boss != nil || L.Node == NBoss {
		return
	}
	if L.critT == 0 {
		L.critT = 8 + L.rng.Float64()*6
	}
	alive := 0
	for _, c := range L.Critters {
		if c.gone <= 0 {
			alive++
		}
	}
	if !L.Cleared && L.waveT <= 0 && !L.anyStuck() {
		L.critT -= dt
	}
	if L.critT <= 0 {
		L.critT = 15 + L.rng.Float64()*10
		if alive < 2 {
			L.spawnCritter()
		}
	}
	// a fresh capsule sometimes brings a Pilferer running
	if n := len(L.Drops); n > L.dropsSeen && !L.Cleared && !L.hasCritter(critThief) && L.rng.Float64() < 0.2 {
		L.spawnThief(L.Drops[n-1])
	}
	L.dropsSeen = len(L.Drops)
	z0, z1, bandOK := L.critterBand()
	for i := 0; i < len(L.Critters); i++ {
		c := L.Critters[i]
		c.T += dt
		c.enter = math.Min(1, c.enter+dt*2.5)
		if c.gone > 0 {
			c.gone -= dt
			if c.gone <= 0 {
				L.Critters = append(L.Critters[:i], L.Critters[i+1:]...)
				i--
			}
			continue
		}
		if c.Kind != critWander && L.updateSpecial(c, dt, z0, z1, bandOK) {
			continue
		}
		sp := c.Speed
		if c.scareT > 0 {
			c.scareT -= dt
			sp *= 2.2
		}
		c.hop += dt * sp * 3.2
		// wander: new heading now and then, turn back from walls and bricks
		c.turnT -= dt
		if c.turnT <= 0 {
			c.turnT = 1.2 + L.rng.Float64()*2.2
			c.H += (L.rng.Float64() - 0.5) * 2.4
		}
		nx, nz := c.X+math.Sin(c.H)*sp*dt, c.Z+math.Cos(c.H)*sp*dt
		outBand := bandOK && (nz < z0 || nz > z1)
		if !L.critterInside(nx, nz) || outBand {
			// steer toward the middle of the open floor instead
			tz := (z0 + z1) / 2
			if !bandOK {
				tz = c.Z
			}
			c.H = math.Atan2(-c.X*0.6+(L.rng.Float64()-0.5)*4, tz-c.Z+0.01)
			c.turnT = 0.8
		} else {
			c.X, c.Z = nx, nz
		}
		// time's up, or the formation has crowded it out: it scurries off
		if c.T > c.Life || L.Cleared || (!bandOK && c.T > 1.5) {
			c.gone = 0.45
			L.sparks(V3{c.X, 0.5, c.Z}, 10, critterCol(L.Biome.ID, c.Golden), 4)
		}
	}
}

func (L *Level) spawnCritter() {
	if L.spawnMischief() {
		return
	}
	L.spawnWanderer()
}

func (L *Level) spawnWanderer() {
	z0, z1, ok := L.critterBand()
	if !ok {
		return
	}
	for try := 0; try < 12; try++ {
		side := float64(L.rng.Intn(2)*2 - 1)
		x := side * (HalfW - 2 - L.rng.Float64()*3)
		z := z0 + L.rng.Float64()*(z1-z0)
		if !L.critterInside(x, z) {
			continue
		}
		c := &Critter{X: x, Z: z, H: math.Atan2(-side, (L.rng.Float64()-0.5)*1.2), Speed: 1.6 + L.rng.Float64()*0.9,
			Life: 14 + L.rng.Float64()*6, turnT: 1.5}
		c.Golden = L.rng.Float64() < 0.12
		if c.Golden {
			c.Speed *= 1.35 // the lucky ones are quick
		}
		L.Critters = append(L.Critters, c)
		L.sparks(V3{x, 0.5, z}, 8, critterCol(L.Biome.ID, c.Golden), 3)
		L.g.coach("critter")
		return
	}
}

// collideCritters bounces an orb off a critter, which bursts into loot.
func (L *Level) collideCritters(b *Ball) {
	for _, c := range L.Critters {
		if c.gone > 0 || c.enter < 0.5 {
			continue
		}
		dx, dz := b.X-c.X, b.Z-c.Z
		d := math.Hypot(dx, dz)
		if d < b.R+critterR && d > 0 {
			nx, nz := dx/d, dz/d
			if dot := b.VX*nx + b.VZ*nz; dot < 0 {
				b.VX -= 2 * dot * nx
				b.VZ -= 2 * dot * nz
			}
			b.X, b.Z = c.X+nx*(b.R+critterR+0.01), c.Z+nz*(b.R+critterR+0.01)
			L.popCritter(c)
			return
		}
		if d < b.R+critterR+1.2 && c.scareT <= 0 && (c.Kind == critWander || c.Kind == critThief && len(c.carry) == 0) {
			// a near miss startles it into a dash away from the orb
			c.scareT = 0.6
			c.H = math.Atan2(-dx, -dz)
		}
	}
}

func (L *Level) popCritter(c *Critter) {
	g := L.g
	c.gone, c.popped = 0.35, true
	if c.Kind != critWander {
		L.critters++
		L.popMischief(c)
		return
	}
	col := critterCol(L.Biome.ID, c.Golden)
	L.sparks(V3{c.X, 0.7, c.Z}, 18, col, 6)
	L.Rings = append(L.Rings, &Ring{X: c.X, Z: c.Z, MaxR: 1.8, Life: 0.3, Max: 0.3, Col: col})
	L.Shake = math.Max(L.Shake, 0.15)
	if c.Golden {
		L.gainGold(8, c.X, c.Z, true)
		L.dropPower(c.X, c.Z, true)
		L.text(V3{c.X, 1.6, c.Z}, "LUCKY!", color.RGBA{255, 220, 90, 255}, true)
		g.snd.Play("coinbig")
		g.snd.PlayVol("pw", 0.7)
	} else {
		L.gainGold(3, c.X, c.Z, true)
		if L.rng.Float64() < 0.3 {
			L.dropPower(c.X, c.Z, true)
		}
		g.snd.Play("coinbig")
	}
	L.critters++
}

func critterCol(world int, golden bool) color.RGBA {
	if golden {
		return color.RGBA{255, 214, 90, 255}
	}
	switch world {
	case WMeadow:
		return color.RGBA{240, 230, 215, 255}
	case WRuins:
		return color.RGBA{60, 190, 170, 255}
	case WClouds:
		return color.RGBA{255, 255, 255, 255}
	case WNeon:
		return color.RGBA{80, 240, 255, 255}
	case WCaverns:
		return color.RGBA{255, 140, 190, 255}
	default:
		return color.RGBA{220, 170, 90, 255}
	}
}

// ---------------------------------------------------------------- drawing

func (g *Game) drawCritters(B *Batch, L *Level) {
	for _, c := range L.Critters {
		s := 1.0
		switch {
		case c.gone > 0 && c.popped:
			s = 1 + (0.35-c.gone)*2 // swells as it bursts
			if c.gone < 0.2 {
				continue
			}
		case c.gone > 0:
			s = c.gone / 0.45
		default:
			s = ease(c.enter)
		}
		if s <= 0.01 {
			continue
		}
		s *= critterSize
		if c.Kind != critBomber {
			B.DecalBelow(c.X+0.15, c.Z-0.15, 0.02, 1.4*s, 1.1*s, sprShadow, color.RGBA{0, 0, 0, 150})
		}
		rot := rotXYZ(0, c.H, 0)
		at := func(x, y, z float64) V3 { return V3{c.X, 0, c.Z}.Add(rot.Mul(V3{x * s, y * s, z * s})) }
		if c.Kind != critWander {
			g.drawMischief(B, L, c, s, rot, at)
			continue
		}
		col := critterCol(L.Biome.ID, c.Golden)
		emis := float32(0)
		if c.Golden {
			emis = 0.35
			B.Billboard(at(0, 0.6, 0), 2.6*s, sprSoft, alpha(color.RGBA{255, 220, 100, 255}, 0.35+0.15*math.Sin(c.T*8)), true)
		}
		ball := func(key string, p V3, sc V3, c1, c2 color.RGBA, e float32) {
			B.Draw(cached("crit"+key+colKey(c1), func() *Mesh { return Icosphere(1, 0.03, 5, c1, c2) }), p, sc.Mul(s), rot, color.RGBA{}, e)
		}
		eyes := func(y, z, sep, sz float64) {
			for _, sx := range []float64{-1, 1} {
				p := at(sx*sep, y, z)
				B.Billboard(p, sz*s, sprSphere, color.RGBA{20, 16, 24, 255}, false)
				B.Billboard(p.Add(V3{0.03, 0.04, -0.05}), sz*0.4*s, sprSphere, color.RGBA{255, 255, 255, 255}, false)
			}
		}
		switch L.Biome.ID {
		case WMeadow: // a hopping rabbit
			hop := math.Abs(math.Sin(c.hop)) * 0.45
			ball("rab", at(0, 0.45+hop, 0), V3{0.42, 0.36, 0.5}, col, darken(col, 0.8), emis)
			ball("rab", at(0, 0.8+hop, 0.38), V3{0.28, 0.27, 0.28}, col, darken(col, 0.8), emis)
			for _, sx := range []float64{-1, 1} {
				ear := at(sx*0.11, 1.18+hop, 0.32)
				B.OBox(ear, rot.Mul(V3{0.05, 0, 0}).Mul(s), V3{0, 0.24 * s, 0}, rot.Mul(V3{0, 0, 0.05}).Mul(s), lighten(col, 0.1), emis)
			}
			ball("tail", at(0, 0.55+hop, -0.5), V3{0.12, 0.12, 0.12}, color.RGBA{255, 255, 255, 255}, color.RGBA{230, 230, 230, 255}, emis)
			eyes(0.86+hop, 0.62, 0.12, 0.09)
		case WRuins: // a scarab with scuttling legs
			ball("scb", at(0, 0.32, 0), V3{0.45, 0.25, 0.55}, col, darken(col, 0.6), emis+0.15)
			ball("scbh", at(0, 0.28, 0.5), V3{0.22, 0.16, 0.18}, darken(col, 0.5), darken(col, 0.3), emis)
			B.Box(c.X-0.02*s, 0.5*s, c.Z-0.02*s, c.X+0.02*s, 0.58*s, c.Z+0.02*s, color.RGBA{255, 210, 90, 255}, sprWhite, 0.4)
			for k := 0; k < 3; k++ {
				for _, sx := range []float64{-1, 1} {
					w := math.Sin(c.hop*2+float64(k)*2+sx) * 0.12
					p := at(sx*0.45, 0.14, -0.25+float64(k)*0.25+w)
					B.OBox(p, rot.Mul(V3{0.18, 0, 0}).Mul(s), V3{0, 0.03 * s, 0}, rot.Mul(V3{0, 0, 0.03}).Mul(s), color.RGBA{30, 40, 40, 255}, 0)
				}
			}
			eyes(0.36, 0.64, 0.1, 0.07)
		case WClouds: // a puff of cloud with little wings
			bob := math.Sin(c.T*3) * 0.12
			ball("puf", at(0, 0.75+bob, 0), V3{0.42, 0.34, 0.4}, col, color.RGBA{215, 225, 245, 255}, emis+0.1)
			ball("puf", at(-0.3, 0.68+bob, -0.05), V3{0.26, 0.24, 0.26}, col, color.RGBA{215, 225, 245, 255}, emis+0.1)
			ball("puf", at(0.3, 0.7+bob, -0.02), V3{0.28, 0.24, 0.28}, col, color.RGBA{215, 225, 245, 255}, emis+0.1)
			fl := math.Sin(c.T*16) * 0.5
			for _, sx := range []float64{-1, 1} {
				d := V3{sx * math.Cos(fl), math.Sin(fl) * 0.6, 0}
				B.OBox(at(sx*0.55, 0.85+bob, 0), rot.Mul(d.Mul(0.22)).Mul(s), V3{0, 0.02 * s, 0}, rot.Mul(V3{0, 0, 0.14}).Mul(s), color.RGBA{255, 240, 200, 255}, 0.3)
			}
			eyes(0.8+bob, 0.38, 0.12, 0.09)
		case WNeon: // a hovering drone
			bob := 0.9 + math.Sin(c.T*4)*0.1
			B.OBox(at(0, bob, 0), rot.Mul(V3{0.38, 0, 0}).Mul(s), V3{0, 0.14 * s, 0}, rot.Mul(V3{0, 0, 0.32}).Mul(s), color.RGBA{90, 70, 150, 255}, 0.3)
			B.OBox(at(0, bob, 0.33), rot.Mul(V3{0.24, 0, 0}).Mul(s), V3{0, 0.06 * s, 0}, rot.Mul(V3{0, 0, 0.02}).Mul(s), col, 0.9)
			for _, sx := range []float64{-1, 1} {
				for _, sz := range []float64{-1, 1} {
					p := at(sx*0.45, bob+0.18, sz*0.38)
					B.Billboard(p, 0.55*s, sprRing, alpha(col, 0.55+0.3*math.Sin(c.T*30+sx+sz)), true)
				}
			}
			B.Billboard(at(0, bob, 0), 1.6*s, sprSoft, alpha(col, 0.25), true)
		case WCaverns: // a crystal crab, scuttling sideways
			ball("crb", at(0, 0.32, 0), V3{0.5, 0.22, 0.38}, col, darken(col, 0.6), emis+0.2)
			for _, sx := range []float64{-1, 1} {
				pinch := 0.08 * math.Sin(c.T*6+sx)
				ball("claw", at(sx*0.55, 0.38, 0.32), V3{0.16, 0.12 + pinch, 0.2}, lighten(col, 0.2), col, emis+0.25)
				for k := 0; k < 3; k++ {
					w := math.Sin(c.hop*2.5+float64(k)+sx) * 0.08
					B.OBox(at(sx*0.48, 0.16+w, -0.2+float64(k)*0.18), rot.Mul(V3{0.16, 0, 0}).Mul(s), V3{0, 0.03 * s, 0}, rot.Mul(V3{0, 0, 0.03}).Mul(s), darken(col, 0.5), 0.1)
				}
			}
			for _, sx := range []float64{-1, 1} {
				st := at(sx*0.14, 0.62, 0.26)
				B.OBox(at(sx*0.14, 0.5, 0.26), V3{0.02 * s, 0, 0}, V3{0, 0.12 * s, 0}, V3{0, 0, 0.02 * s}, darken(col, 0.4), 0)
				B.Billboard(st, 0.12*s, sprSphere, color.RGBA{20, 16, 24, 255}, false)
			}
			B.Billboard(at(0, 0.4, 0), 1.8*s, sprSoft, alpha(col, 0.22), true)
		default: // Machine Works: a clockwork beetle with a turning key
			ball("cog", at(0, 0.38, 0), V3{0.42, 0.3, 0.5}, col, darken(col, 0.55), emis+0.1)
			B.Box(c.X-0.03*s, 0.62*s, c.Z-0.03*s, c.X+0.03*s, 0.85*s, c.Z+0.03*s, color.RGBA{120, 100, 70, 255}, sprWhite, 0)
			ka := c.T * 5
			kp := at(0, 0.92, -0.05)
			B.OBox(kp, rot.Mul(V3{math.Cos(ka) * 0.22, 0, math.Sin(ka) * 0.03}).Mul(s), V3{0, 0.08 * s, 0}, rot.Mul(V3{-math.Sin(ka) * 0.03, 0, math.Cos(ka) * 0.03}).Mul(s), color.RGBA{230, 190, 90, 255}, 0.2)
			for k := 0; k < 3; k++ {
				for _, sx := range []float64{-1, 1} {
					w := math.Sin(c.hop*2+float64(k)*2+sx) * 0.1
					B.OBox(at(sx*0.42, 0.14, -0.22+float64(k)*0.22+w), rot.Mul(V3{0.16, 0, 0}).Mul(s), V3{0, 0.03 * s, 0}, rot.Mul(V3{0, 0, 0.03}).Mul(s), color.RGBA{70, 60, 50, 255}, 0)
				}
			}
			for _, sx := range []float64{-1, 1} {
				B.Billboard(at(sx*0.12, 0.45, 0.52), 0.13*s, sprSphere, color.RGBA{255, 90, 60, 255}, false)
				B.Billboard(at(sx*0.12, 0.45, 0.52), 0.4*s, sprSoft, color.RGBA{255, 90, 60, 160}, true)
			}
		}
	}
}
