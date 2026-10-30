package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Class passives: each origin has one mechanic that changes how a chamber
// is played. State lives on the Level (it resets every chamber).

type classState struct {
	parryCD float64

	rage  int
	rageT float64

	attuneT  float64
	attuneEl int

	stillT    float64
	stillX    float64
	overclock bool

	static float64
}

var attuneNames = []string{"FIRE", "FROST", "STORM", "VOID"}
var attuneCols = []color.RGBA{{255, 120, 50, 255}, {140, 210, 255, 255}, {150, 190, 255, 255}, {190, 110, 255, 255}}

// updatePassives runs once per frame.
func (L *Level) updatePassives(dt float64) {
	s := &L.g.run.Stats
	c := &L.cls
	c.parryCD -= dt
	if s.ClassRage && c.rage > 0 {
		c.rageT -= dt
		if c.rageT <= 0 {
			c.rage = 0
		}
	}
	if s.ClassAttune {
		c.attuneT += dt
		if c.attuneT >= s.attunePeriod() {
			c.attuneT = 0
			c.attuneEl = (c.attuneEl + 1) % 4
			L.text(V3{L.PaddleX, 1.6, PaddleZ + 2.5}, "ATTUNED: "+attuneNames[c.attuneEl], attuneCols[c.attuneEl], false)
			L.g.snd.PlayVol("abilready", 0.45)
		}
	}
	if s.ClassOverclock {
		if math.Abs(L.PaddleX-c.stillX) > 0.5 {
			c.stillX = L.PaddleX
			c.stillT = 0
		} else if !c.overclock {
			c.stillT += dt
			if c.stillT >= s.overclockTime() {
				c.overclock = true
				L.g.snd.PlayVol("abilready", 0.5)
				L.text(V3{L.PaddleX, 1.4, PaddleZ + 1.2}, "OVERCLOCKED", color.RGBA{255, 170, 70, 255}, false)
			}
		}
	}
	if s.ClassStatic && c.static < 100 {
		c.static = math.Min(100, c.static+math.Abs(L.PaddleX-L.lastPadX)*100/(10*HalfW)*s.staticRate())
		if c.static >= 100 {
			L.g.snd.PlayVol("abilready", 0.5)
		}
	}
	L.lastPadX = L.PaddleX
	// supercharged orbs crackle
	for _, b := range L.Balls {
		if b.Storm > 0 {
			b.Storm -= dt
			if L.rng.Float64() < dt*20 {
				L.sparks(V3{b.X, 0.6, b.Z}, 1, color.RGBA{170, 215, 255, 255}, 3)
			}
		}
	}
}

// passiveDamage scales a ball hit's damage.
func (L *Level) passiveDamage(b *Brick, dmg float64) float64 {
	s := &L.g.run.Stats
	if L.hasOmen("frenzy") {
		dmg *= 1.3
	}
	if s.CurseDmg > 0 {
		dmg *= 1 + s.CurseDmg*float64(s.CurseCount)
	}
	if s.ClassRage {
		dmg *= 1 + s.ragePer()*float64(L.cls.rage)
	}
	if s.ClassHighRoller {
		dmg *= 1 + s.highRoller(L.g.run.Gold)
	}
	if s.ClassStoke && b.Burn > 0 {
		dmg *= s.stokeMult()
		b.Burn = 3.5
		L.sparks(V3{b.X(), 0.8, b.Z()}, 4, color.RGBA{255, 150, 50, 255}, 4)
	}
	return dmg
}

// passiveOnHit fires after a ball hits a brick.
func (L *Level) passiveOnHit(ball *Ball, b *Brick, dmg float64) {
	s := &L.g.run.Stats
	bx, bz := b.X(), b.Z()
	if ball.Storm > 0 {
		n := 2
		if s.SigConduit {
			n = 3
		}
		L.lightning(bx, bz, n, math.Max(0.5, dmg*0.4), b, false)
	}
	if s.ClassAttune && L.rng.Float64() < s.attuneChance() && b.Alive {
		pw := 1.0
		if s.SigLens {
			pw = 1.5
		}
		switch L.cls.attuneEl {
		case 0:
			b.Burn = 3.5 * pw
		case 1:
			b.Frozen = (3 + s.FreezeDur) * pw
		case 2:
			L.lightning(bx, bz, 2, math.Max(1, dmg*0.6*pw), b, false)
		case 3:
			L.damage(b, Hit{Src: "Attunement", Dmg: dmg * 0.8 * pw, Void: true, Silent: true})
		}
		L.sparks(V3{bx, 0.8, bz}, 6, attuneCols[L.cls.attuneEl], 5)
	}
}

// passiveOnBreak fires when any brick is destroyed.
func (L *Level) passiveOnBreak() {
	if s := &L.g.run.Stats; s.ClassRage {
		c := &L.cls
		if c.rage < s.rageMax() {
			c.rage++
			if c.rage == s.rageMax() {
				L.text(V3{L.PaddleX, 1.6, PaddleZ + 2.5}, "RAMPAGE!", color.RGBA{255, 70, 50, 255}, true)
				L.g.snd.PlayVol("crit", 0.8)
				if L == L.g.level {
					L.g.unlock("ACH_RAMPAGE")
				}
			}
		}
		c.rageT = s.rageTime()
	}
}

// passiveOnPaddle fires when a ball bounces off the paddle; t is the hit
// offset from the paddle centre (-1..1).
func (L *Level) passiveOnPaddle(b *Ball, t float64) {
	s := &L.g.run.Stats
	c := &L.cls
	if s.ClassParry && math.Abs(t) < s.parryWindow() && c.parryCD <= 0 {
		c.parryCD = s.parryCooldown()
		L.holyWave(b.X)
		if s.SigHalo {
			r := L.g.run
			r.HP = math.Min(s.MaxHP, r.HP+1)
		}
	}
	if s.ClassStatic && c.static >= 100 && b.Phantom <= 0 {
		c.static = 0
		b.Storm = s.stormTime()
		L.text(V3{b.X, 1.4, b.Z + 1}, "SUPERCHARGED", color.RGBA{150, 210, 255, 255}, true)
		L.g.snd.PlayVol("thunder", 0.55)
		if L == L.g.level {
			L.g.unlock("ACH_SUPERCHARGE")
		}
		L.Rings = append(L.Rings, &Ring{X: b.X, Z: b.Z, MaxR: 2.6, Life: 0.35, Max: 0.35, Col: color.RGBA{150, 210, 255, 255}})
	}
}

// holyWave strikes the lowest bricks in the paddle's column band.
func (L *Level) holyWave(x float64) {
	s := &L.g.run.Stats
	dmg := math.Max(1, s.Damage*s.DmgMult)
	hits := 0
	for r := levelgen.MaxRows - 1; r >= 0 && hits < s.parryHits(); r-- {
		for c := 0; c < levelgen.Cols; c++ {
			b := L.Grid[r][c]
			if b == nil || !b.Alive || math.Abs(b.X()-x) > effCell*0.6 {
				continue
			}
			L.damage(b, Hit{Src: "Parry", Dmg: dmg, Elemental: true, Silent: true})
			L.sparks(V3{b.X(), 0.8, b.Z()}, 5, color.RGBA{255, 225, 150, 255}, 5)
			hits++
		}
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: PaddleZ + 1, MaxR: 3, Life: 0.35, Max: 0.35, Col: color.RGBA{255, 220, 140, 255}})
	L.text(V3{x, 1.2, PaddleZ + 1.6}, "PARRY", color.RGBA{255, 225, 150, 255}, false)
	L.g.snd.PlayVol("parry", 0.75)
}

// passiveOnLoss resets streak-based passives.
func (L *Level) passiveOnLoss() {
	if !L.g.run.Stats.SigBloodPrice {
		L.cls.rage = 0
	}
	L.cls.static = 0
}

// ---------------------------------------------------------------- arena sigil

// floorLine lays a thin glowing line on the arena floor.
func floorLine(B *Batch, x0, z0, x1, z1, w float64, col color.RGBA) {
	dx, dz := x1-x0, z1-z0
	l := math.Hypot(dx, dz)
	if l == 0 {
		return
	}
	nx, nz := -dz/l*w/2, dx/l*w/2
	y := 0.014
	B.Quad([4]V3{{x0 + nx, y, z0 + nz}, {x1 + nx, y, z1 + nz}, {x1 - nx, y, z1 - nz}, {x0 - nx, y, z0 - nz}}, sprWhite, col, [4]float32{1, 1, 1, 1}, true)
}

// drawArenaSigil paints a turning magic circle on the arena floor that
// brightens with the combo, flares on paddle hits and blooms when the
// chamber is cleared.
func (g *Game) drawArenaSigil(B *Batch, L *Level) {
	cx, cz := 0.0, PaddleZ+6.5
	R := 4.6
	k := 0.1 + 0.22*math.Min(1, float64(L.Combo)/20) + 0.25*L.paddleBounce
	if L.Cleared {
		e := math.Min(1, L.clearT*1.4)
		R *= 1 + 0.35*e
		k = math.Max(k, 0.9*(1-math.Max(0, L.clearT-0.8)/1.6))
	}
	if k <= 0.01 {
		return
	}
	col := L.Biome.Trim
	rot := L.T * 0.15
	ring := func(r, w, a float64) {
		const n = 56
		for i := 0; i < n; i++ {
			a0 := float64(i) / n * 2 * math.Pi
			a1 := float64(i+1) / n * 2 * math.Pi
			floorLine(B, cx+math.Cos(a0)*r, cz+math.Sin(a0)*r*0.92, cx+math.Cos(a1)*r, cz+math.Sin(a1)*r*0.92, w, alpha(col, a*k))
		}
	}
	ring(R, 0.07, 0.9)
	ring(R*0.9, 0.04, 0.7)
	ring(R*0.55, 0.05, 0.7)
	for i := 0; i < 36; i++ {
		an := rot + float64(i)/36*2*math.Pi
		l0 := R * 0.91
		if i%3 == 0 {
			l0 = R * 0.84
		}
		floorLine(B, cx+math.Cos(an)*l0, cz+math.Sin(an)*l0*0.92, cx+math.Cos(an)*R*0.99, cz+math.Sin(an)*R*0.99*0.92, 0.05, alpha(col, 0.8*k))
	}
	for tri := 0; tri < 2; tri++ {
		for j := 0; j < 3; j++ {
			a0 := -rot*1.5 + float64(tri)*math.Pi/3 + float64(j)*2*math.Pi/3
			a1 := a0 + 2*math.Pi/3
			r := R * 0.55
			floorLine(B, cx+math.Cos(a0)*r, cz+math.Sin(a0)*r*0.92, cx+math.Cos(a1)*r, cz+math.Sin(a1)*r*0.92, 0.05, alpha(col, 0.7*k))
		}
	}
	B.Decal(cx, cz, 0.012, R*1.6, R*1.5, sprSoft, alpha(col, 0.18*k), true)
}
