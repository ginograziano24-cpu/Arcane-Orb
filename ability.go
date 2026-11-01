package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Class active abilities (right mouse / E).

func (r *Run) hasAbility() bool { return r.Arch != nil && r.Arch.Ability != "" }

// abilityReady reports the cooldown fraction remaining (0 = ready).
func (r *Run) abilityFrac() float64 {
	if !r.hasAbility() || r.Arch.Cooldown <= 0 {
		return 0
	}
	return clamp01(r.AbilT / r.Arch.Cooldown)
}

func (L *Level) updateAbility(dt float64, in Input) {
	r := L.g.run
	if !r.hasAbility() {
		return
	}
	if r.AbilT > 0 {
		r.AbilT = math.Max(0, r.AbilT-dt)
		if r.AbilT == 0 {
			L.g.snd.PlayVol("abilready", 0.7)
			L.abilReadyT = 1
		}
	}
	L.abilReadyT = math.Max(0, L.abilReadyT-dt*1.5)
	if !in.Ability || L.Cleared {
		return
	}
	if r.AbilT > 0 {
		L.g.snd.PlayVol("deny", 0.5)
		return
	}
	used := false
	switch r.Arch.ID {
	case "pyro":
		used = L.firestorm()
	case "storm":
		used = L.thunderstrike()
	case "guardian":
		used = L.aegis()
	case "berserker":
		used = L.bloodlust()
	case "arcane":
		used = L.prismNova()
	case "engineer":
		used = L.overload()
	case "lucky":
		used = L.jackpot()
	}
	if used {
		r.AbilT = r.Arch.Cooldown
		L.abilFlash = 1
	}
}

func (L *Level) abilityDmg() float64 {
	s := &L.g.run.Stats
	return math.Max(2, s.Damage*s.DmgMult)
}

// firestorm sets the two rows nearest the paddle ablaze.
func (L *Level) firestorm() bool {
	rows := map[int]bool{}
	maxR := -1
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && !b.BossPart && b.R > maxR {
			maxR = b.R
		}
	}
	if maxR < 0 {
		return false
	}
	second := -1
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && !b.BossPart && b.R < maxR && b.R > second {
			second = b.R
		}
	}
	rows[maxR] = true
	if second >= 0 {
		rows[second] = true
	}
	dmg := L.abilityDmg() * 1.5 * (1 + 0.5*L.g.run.Stats.BurnMult)
	var targets []*Brick
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && rows[b.R] {
			targets = append(targets, b)
		}
	}
	// a wall of fire sweeps across the rows
	for r := range rows {
		z := cellZ(r)
		L.Rings = append(L.Rings, &Ring{X: 0, Z: z, MaxR: HalfW, Life: 0.45, Max: 0.45, Col: color.RGBA{255, 120, 40, 255}})
		for x := -HalfW + 1; x < HalfW; x += 1.6 {
			L.sparks(V3{x, 0.6, z}, 4, color.RGBA{255, uint8(110 + L.rng.Intn(110)), 40, 255}, 7)
		}
	}
	if L.Boss != nil {
		for _, b := range L.Bricks {
			if b.Alive && b.Core {
				targets = append(targets, b)
			}
		}
	}
	for _, b := range targets {
		if b.Kind != levelgen.Steel {
			b.Burn = 3.5
		}
		L.damage(b, Hit{Src: "Firestorm", Dmg: dmg, Fire: true, Elemental: true, Silent: true})
	}
	L.text(V3{0, 2.4, cellZ(maxR)}, "FIRESTORM", color.RGBA{255, 150, 60, 255}, true)
	L.Shake = math.Max(L.Shake, 0.45)
	L.Flash = math.Max(L.Flash, 0.25)
	L.g.snd.Play("firestorm")
	L.g.snd.Duck(0.5, 0.8)
	return true
}

// thunderstrike smites the toughest brick, then chains.
func (L *Level) thunderstrike() bool {
	var best *Brick
	for _, b := range L.Bricks {
		if !b.Alive || b.Kind == levelgen.Steel || (b.BossPart && !b.Core) || (b.Layer != 0 && !b.Core) {
			continue
		}
		if best == nil || b.Core || (!best.Core && b.HP > best.HP) {
			best = b
		}
	}
	if best == nil {
		return false
	}
	s := &L.g.run.Stats
	dmg := L.abilityDmg() * 4 * (1 + s.ChainMult)
	if best.Core {
		dmg *= 0.35 // a Guardian's heart resists a single bolt
	}
	x, z := best.X(), best.Z()
	for i := 0; i < 3; i++ {
		L.Arcs = append(L.Arcs, &Arc{Pts: jag(V3{x + (L.rng.Float64()-0.5)*4, 16, z + (L.rng.Float64()-0.5)*2}, V3{x, 0.8, z}, L.rng), Life: 0.5, Col: color.RGBA{190, 225, 255, 255}})
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 3.6, Life: 0.4, Max: 0.4, Col: color.RGBA{150, 210, 255, 255}})
	L.sparks(V3{x, 0.8, z}, 30, color.RGBA{200, 230, 255, 255}, 12)
	L.damage(best, Hit{Src: "Thunderstrike", Dmg: dmg, Elemental: true})
	L.lightningRaw(x, z, 3+s.ChainCount/2, dmg*0.35, best, s.StormCaller || s.ChainRecast)
	L.text(V3{x, 2.6, z}, "THUNDERSTRIKE", color.RGBA{180, 220, 255, 255}, true)
	L.Shake = math.Max(L.Shake, 0.55)
	L.Flash = math.Max(L.Flash, 0.4)
	L.g.snd.Play("thunder")
	L.g.snd.Duck(0.55, 0.9)
	return true
}

// classIgnite is the Pyromancer passive: every Nth ball hit ignites.
func (L *Level) classIgnite() bool {
	n := L.g.run.Stats.ClassIgnite
	if n <= 0 {
		return false
	}
	L.igniteN++
	return L.igniteN%n == 0
}

// ---------------------------------------------------------------- the other five

// aegis raises a wall of holy light along the paddle line: orbs that would
// slip past rebound off it, and every rebound looses a holy wave.
func (L *Level) aegis() bool {
	L.cls.aegisT = aegisDur
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: BarrierZ, MaxR: HalfW, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 220, 140, 255}})
	for x := -HalfW + 0.5; x < HalfW; x += 1.2 {
		L.sparks(V3{x, 0.4, BarrierZ}, 2, color.RGBA{255, 230, 160, 255}, 5)
	}
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2.5}, "AEGIS", color.RGBA{255, 225, 150, 255}, true)
	L.Flash = math.Max(L.Flash, 0.2)
	L.g.snd.Play("shield")
	L.g.snd.PlayVol("parry", 0.9)
	return true
}

const aegisDur = 6.0

// aegisCatch bounces a falling orb off the Aegis wall.
func (L *Level) aegisCatch(b *Ball) bool {
	if L.cls.aegisT <= 0 || b.VZ >= 0 || b.Z-b.R > BarrierZ || b.Z < BarrierZ-1 {
		return false
	}
	b.Z = BarrierZ + b.R
	b.VZ = math.Abs(b.VZ)
	L.sparks(V3{b.X, 0.3, BarrierZ}, 14, color.RGBA{255, 225, 150, 255}, 6)
	L.holyWave(b.X)
	return true
}

// bloodlust fills Rage at once; for a few seconds it cannot fade and every
// hit lands half again as hard.
func (L *Level) bloodlust() bool {
	s := &L.g.run.Stats
	c := &L.cls
	c.rage = s.rageMax()
	c.rageT = s.rageTime() + bloodDur
	c.bloodT = bloodDur
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2.5}, "BLOODLUST", color.RGBA{255, 60, 50, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ + 1, MaxR: 5, Life: 0.45, Max: 0.45, Col: color.RGBA{255, 60, 40, 255}})
	L.Shake = math.Max(L.Shake, 0.35)
	L.Flash = math.Max(L.Flash, 0.2)
	L.g.snd.Play("roar")
	L.g.snd.Duck(0.5, 0.6)
	return true
}

const bloodDur = 5.0

// prismNova bursts every element at once around each orb: bricks nearby
// burn, freeze, take void damage and a bolt chains out from the centre.
func (L *Level) prismNova() bool {
	if len(L.Balls) == 0 {
		return false
	}
	dmg := L.abilityDmg() * 1.6
	const rad = 4.2
	for _, ball := range L.Balls {
		hit := 0
		for _, b := range L.Bricks {
			if !b.Alive || (b.BossPart && !b.Core) {
				continue
			}
			if math.Hypot(b.X()-ball.X, b.Z()-ball.Z) > rad {
				continue
			}
			if b.Kind != levelgen.Steel {
				b.Burn = 3.5
				b.Frozen = 3
			}
			L.damage(b, Hit{Src: "Prismatic Nova", Dmg: dmg, Void: true, Elemental: true, Silent: true})
			hit++
		}
		L.lightning(ball.X, ball.Z, 4, dmg*0.6, nil, false)
		for i, col := range attuneCols {
			L.Rings = append(L.Rings, &Ring{X: ball.X, Z: ball.Z, MaxR: rad * (0.6 + 0.15*float64(i)), Life: 0.35 + 0.08*float64(i), Max: 0.35 + 0.08*float64(i), Col: col})
			L.sparks(V3{ball.X, 0.8, ball.Z}, 8, col, 9)
		}
	}
	L.text(V3{L.Balls[0].X, 2.4, L.Balls[0].Z}, "PRISMATIC NOVA", color.RGBA{210, 170, 255, 255}, true)
	L.Shake = math.Max(L.Shake, 0.4)
	L.Flash = math.Max(L.Flash, 0.3)
	L.g.snd.Play("warp")
	L.g.snd.PlayVol("zap", 0.7)
	return true
}

// overload dumps the reactor: three piercing beam volleys in quick
// succession.
func (L *Level) overload() bool {
	L.cls.overloadN = 3
	L.cls.overloadT = 0
	L.text(V3{L.PaddleX, 2.0, PaddleZ + 2.2}, "OVERLOAD", color.RGBA{255, 170, 70, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 3.5, Life: 0.35, Max: 0.35, Col: color.RGBA{255, 170, 70, 255}})
	L.g.snd.PlayVol("zap", 0.9)
	return true
}

// jackpot stakes a little gold on fortune: three good power-ups rain down
// and a few coins come back with them.
func (L *Level) jackpot() bool {
	r := L.g.run
	if r.Gold < jackpotCost {
		L.text(V3{L.PaddleX, 1.6, PaddleZ + 1.6}, "NEED 15 GOLD", color.RGBA{255, 200, 90, 255}, false)
		return false
	}
	r.Gold -= jackpotCost
	for i := 0; i < 3; i++ {
		x := (L.rng.Float64()*2 - 1) * (HalfW - 2)
		L.dropPower(x, 9+L.rng.Float64()*5, true)
	}
	L.gainGold(6, L.PaddleX, PaddleZ+4, true)
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2.5}, "JACKPOT!", color.RGBA{255, 214, 90, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ + 1, MaxR: 4, Life: 0.45, Max: 0.45, Col: color.RGBA{255, 214, 90, 255}})
	L.g.snd.Play("coinbig")
	L.g.snd.PlayKeyed("good")
	return true
}

const jackpotCost = 15

