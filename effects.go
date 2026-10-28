package main

import (
	"fmt"
	"image/color"
	"math"

	"orb/levelgen"
)

// Effect helpers used by upgrades, relics and fusions.

// comboCap is the maximum combo damage bonus (1 = +100%).
func (L *Level) comboCap() float64 {
	r := L.g.run
	switch {
	case r.Stats.InfiniteCombo:
		return 1e9
	case r.HasFusion("perpetual"):
		return 2
	}
	return 1
}

// neighbours returns the live bricks orthogonally adjacent to a cell.
func (L *Level) neighbours(row, col int) []*Brick {
	var out []*Brick
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		rr, cc := row+d[0], col+d[1]
		if rr >= 0 && rr < levelgen.MaxRows && cc >= 0 && cc < levelgen.Cols {
			if n := L.Grid[rr][cc]; n != nil && n.Alive && n.Kind != levelgen.Steel {
				out = append(out, n)
			}
		}
	}
	return out
}

// zapNearest strikes the closest brick with a bolt from (x, z).
func (L *Level) zapNearest(x, z, dmg float64) {
	var best *Brick
	bd := 1e9
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && b.Kind != levelgen.Steel && !b.BossPart {
			if d := math.Hypot(b.X()-x, b.Z()-z); d < bd {
				best, bd = b, d
			}
		}
	}
	if best == nil {
		return
	}
	L.Arcs = append(L.Arcs, &Arc{Pts: jag(V3{x, 0.8, z}, V3{best.X(), 0.8, best.Z()}, L.rng), Life: 0.25, Col: color.RGBA{150, 210, 255, 255}})
	L.damage(best, Hit{Src: "Lightning", Dmg: dmg, Elemental: true, Silent: true})
	L.g.snd.PlayVol("zap", 0.6)
}

// skyStrike drops a bolt from the sky onto a point.
func (L *Level) skyStrike(x, z, dmg float64, col color.RGBA) {
	L.Arcs = append(L.Arcs, &Arc{Pts: jag(V3{x + (L.rng.Float64()-0.5)*2, 14, z}, V3{x, 0.8, z}, L.rng), Life: 0.35, Col: col})
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 2.4, Life: 0.3, Max: 0.3, Col: col})
	L.Flash = math.Max(L.Flash, 0.15)
	for _, b := range L.near(x, z, 1.4) {
		L.damage(b, Hit{Src: "Lightning", Dmg: dmg, Elemental: true, Silent: true})
	}
	L.g.snd.Play("zap")
}

// meteor (Cataclysm) slams into a random brick.
func (L *Level) meteor() {
	t := L.top(L.randomBrick())
	if t == nil {
		return
	}
	x, z := t.X(), t.Z()
	s := &L.g.run.Stats
	L.Arcs = append(L.Arcs, &Arc{Pts: []V3{{x - 6, 18, z + 8}, {x - 3, 9, z + 4}, {x, 0.8, z}}, Life: 0.4, Col: color.RGBA{255, 150, 50, 255}})
	L.text(V3{x, 2.6, z}, "CATACLYSM", color.RGBA{255, 140, 50, 255}, true)
	L.explode(x, z, s.ExplodeRadius+1.5, 4*s.Damage*s.DmgMult, true)
}

// blackHole (Event Horizon) devours bricks around a point.
func (L *Level) blackHole(x, z float64) {
	s := &L.g.run.Stats
	rad := 3.4 * CellW
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: rad, Life: 0.8, Max: 0.8, Col: color.RGBA{150, 80, 255, 255}},
		&Ring{X: x, Z: z, MaxR: rad * 0.6, Life: 0.6, Max: 0.6, Col: color.RGBA{40, 10, 70, 255}})
	L.sparks(V3{x, 1, z}, 60, color.RGBA{170, 110, 255, 255}, 12)
	L.text(V3{x, 2.8, z}, "EVENT HORIZON", color.RGBA{190, 140, 255, 255}, true)
	L.Shake, L.Flash = 0.9, 0.4
	L.g.snd.Play("warp")
	L.g.snd.Play("boom")
	for _, b := range L.near(x, z, rad) {
		if b.Layer <= 1 {
			L.damage(b, Hit{Src: "Event Horizon", Dmg: 5 * s.Damage * s.DmgMult, Void: true, Blast: true, Silent: true})
		}
	}
}

// steamBurst (Steam Engine): a brick both burning and frozen explodes.
func (L *Level) steamBurst(b *Brick) {
	s := &L.g.run.Stats
	b.Burn, b.Frozen = 0, 0
	x, z := b.X(), b.Z()
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 3.2, Life: 0.45, Max: 0.45, Col: color.RGBA{235, 240, 250, 255}})
	if L.g.settings.Particles > 0 {
		for i := 0; i < 10; i++ {
			L.Parts = append(L.Parts, &Particle{P: V3{x + (L.rng.Float64()-0.5)*2.4, 0.8, z + (L.rng.Float64()-0.5)*1.4},
				V: V3{0, 2 + L.rng.Float64()*2, 0}, Life: 1.2, Max: 1.2, Size: 1.4 + L.rng.Float64(), Col: color.RGBA{230, 235, 245, 255}, Kind: pSmoke})
		}
	}
	L.text(V3{x, 1.8, z}, "STEAM!", color.RGBA{230, 240, 255, 255}, false)
	L.g.snd.PlayVol("shatter", 0.7)
	for _, n := range L.near(x, z, 2.8) {
		L.damage(n, Hit{Src: "Steam Engine", Dmg: 1.5 * s.Damage * s.DmgMult, Elemental: true, Silent: true})
	}
}

// barrierSaved runs relic and fusion effects when a barrier saves an orb.
func (L *Level) barrierSaved(x float64) {
	r := L.g.run
	if r.Stats.Sanctuary {
		r.HP = math.Min(r.Stats.MaxHP, r.HP+5)
		L.Rings = append(L.Rings, &Ring{X: x, Z: BarrierZ, MaxR: 3, Life: 0.4, Max: 0.4, Col: color.RGBA{140, 255, 180, 255}})
		L.text(V3{x, 1.2, BarrierZ + 1}, "+5", color.RGBA{140, 255, 180, 255}, false)
	}
	if r.Stats.ThornWard {
		// strike the lowest brick in every column
		for c := 0; c < levelgen.Cols; c++ {
			for row := levelgen.MaxRows - 1; row >= 0; row-- {
				if b := L.Grid[row][c]; b != nil && b.Alive && b.Kind != levelgen.Steel {
					L.Rings = append(L.Rings, &Ring{X: b.X(), Z: b.Z(), MaxR: 1.6, Life: 0.3, Max: 0.3, Col: color.RGBA{190, 120, 255, 255}})
					L.damage(b, Hit{Src: "Crown of Thorns", Dmg: 2 * r.Stats.Damage * r.Stats.DmgMult, Blast: true, Silent: true})
					break
				}
			}
		}
		L.text(V3{0, 1.5, 6}, "THORNS", color.RGBA{190, 120, 255, 255}, true)
		L.g.snd.Play("boom")
	}
}

// dmgNumber floats a damage number that grows and warms with the combo
// (Crescendo shows these).
func (L *Level) dmgNumber(x, z, amt float64, crit bool) {
	if amt < 0.05 || len(L.Floats) > 40 {
		return
	}
	k := math.Min(1, float64(L.Combo)/30)
	c := mixc(color.RGBA{235, 235, 245, 255}, color.RGBA{255, 214, 90, 255}, math.Min(1, k*2))
	if k > 0.5 {
		c = mixc(c, color.RGBA{255, 110, 190, 255}, (k-0.5)*2)
	}
	s := fmt.Sprintf("%.1f", amt)
	if amt >= 10 {
		s = fmt.Sprintf("%.0f", amt)
	}
	size := 18 + 22*k
	if crit {
		s += "!"
		size += 6
		c = mixc(c, color.RGBA{255, 90, 90, 255}, 0.6)
	}
	j := (L.rng.Float64() - 0.5) * 0.8
	L.Floats = append(L.Floats, &FloatText{P: V3{x + j, 1.8, z}, S: s, Col: c, Life: 0.95, Scale: size})
}

const finaleDur = 1.3

// startFinale is the last brick of a chamber: slow motion, the camera
// leaning in, a shockwave from where it broke and a swell into the
// clear fanfare.
func (L *Level) startFinale() {
	g := L.g
	x, z := L.lastBrkX, L.lastBrkZ
	L.finX, L.finZ = x, z
	g.snd.PlayKeyed("finale")
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 9, Life: 0.9, Max: 0.9, Col: lighten(L.Biome.Trim, 0.4)},
		&Ring{X: x, Z: z, MaxR: 4, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 240, 200, 255}})
	L.sparks(V3{x, 1, z}, 50, lighten(L.Biome.Trim, 0.5), 13)
	L.Flash = math.Max(L.Flash, 0.35)
	L.Shake = math.Max(L.Shake, 0.5)
	if g.settings.FastAnim {
		return
	}
	L.finaleT = finaleDur
	L.hadFinale = true
}

// clearShown is how long the "chamber cleared" card has been up; it waits
// for the last brick's slow motion to play out.
func (L *Level) clearShown() float64 {
	if L.hadFinale {
		return L.clearT - 0.45
	}
	return L.clearT
}
