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
		L.damage(b, Hit{Dmg: dmg, Fire: true, Elemental: true, Silent: true})
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
	L.damage(best, Hit{Dmg: dmg, Elemental: true})
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
