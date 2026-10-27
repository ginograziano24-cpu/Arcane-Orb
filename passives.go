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
		if c.attuneT >= 8 {
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
			if c.stillT >= 1.5 {
				c.overclock = true
				L.g.snd.PlayVol("abilready", 0.5)
				L.text(V3{L.PaddleX, 1.4, PaddleZ + 1.2}, "OVERCLOCKED", color.RGBA{255, 170, 70, 255}, false)
			}
		}
	}
	if s.ClassStatic && c.static < 100 {
		c.static = math.Min(100, c.static+math.Abs(L.PaddleX-L.lastPadX)*100/(6*HalfW))
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
	if s.ClassRage {
		dmg *= 1 + 0.04*float64(L.cls.rage)
	}
	if s.ClassHighRoller {
		dmg *= 1 + highRollerBonus(L.g.run.Gold)
	}
	if s.ClassStoke && b.Burn > 0 {
		dmg *= 1.5
		b.Burn = 3.5
		L.sparks(V3{b.X(), 0.8, b.Z()}, 4, color.RGBA{255, 150, 50, 255}, 4)
	}
	return dmg
}

func highRollerBonus(gold int) float64 { return math.Min(0.6, float64(gold)/8/100) }

// passiveOnHit fires after a ball hits a brick.
func (L *Level) passiveOnHit(ball *Ball, b *Brick, dmg float64) {
	s := &L.g.run.Stats
	bx, bz := b.X(), b.Z()
	if ball.Storm > 0 {
		L.lightning(bx, bz, 2, math.Max(1, dmg*0.45), b, false)
	}
	if s.ClassAttune && L.rng.Float64() < 0.35 && b.Alive {
		switch L.cls.attuneEl {
		case 0:
			b.Burn = 3.5
		case 1:
			b.Frozen = 3 + s.FreezeDur
		case 2:
			L.lightning(bx, bz, 2, math.Max(1, dmg*0.5), b, false)
		case 3:
			L.damage(b, Hit{Dmg: dmg * 0.6, Void: true, Silent: true})
		}
		L.sparks(V3{bx, 0.8, bz}, 6, attuneCols[L.cls.attuneEl], 5)
	}
}

// passiveOnBreak fires when any brick is destroyed.
func (L *Level) passiveOnBreak() {
	if L.g.run.Stats.ClassRage {
		c := &L.cls
		if c.rage < 20 {
			c.rage++
			if c.rage == 20 {
				L.text(V3{L.PaddleX, 1.6, PaddleZ + 2.5}, "RAMPAGE!", color.RGBA{255, 70, 50, 255}, true)
				L.g.snd.PlayVol("crit", 0.8)
			}
		}
		c.rageT = 4
	}
}

// passiveOnPaddle fires when a ball bounces off the paddle; t is the hit
// offset from the paddle centre (-1..1).
func (L *Level) passiveOnPaddle(b *Ball, t float64) {
	s := &L.g.run.Stats
	c := &L.cls
	if s.ClassParry && math.Abs(t) < 0.22 && c.parryCD <= 0 {
		c.parryCD = 0.8
		L.holyWave(b.X)
	}
	if s.ClassStatic && c.static >= 100 && b.Phantom <= 0 {
		c.static = 0
		b.Storm = 4
		L.text(V3{b.X, 1.4, b.Z + 1}, "SUPERCHARGED", color.RGBA{150, 210, 255, 255}, true)
		L.g.snd.PlayVol("thunder", 0.55)
		L.Rings = append(L.Rings, &Ring{X: b.X, Z: b.Z, MaxR: 2.6, Life: 0.35, Max: 0.35, Col: color.RGBA{150, 210, 255, 255}})
	}
}

// holyWave strikes the lowest bricks in the paddle's column band.
func (L *Level) holyWave(x float64) {
	s := &L.g.run.Stats
	dmg := math.Max(1, s.Damage*s.DmgMult)
	hits := 0
	for r := levelgen.MaxRows - 1; r >= 0 && hits < 4; r-- {
		for c := 0; c < levelgen.Cols; c++ {
			b := L.Grid[r][c]
			if b == nil || !b.Alive || math.Abs(b.X()-x) > CellW*1.1 {
				continue
			}
			L.damage(b, Hit{Dmg: dmg, Elemental: true, Silent: true})
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
	L.cls.rage = 0
	L.cls.static = 0
}
