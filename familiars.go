package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Familiars: little companions that circle above the paddle and help. Each
// one adds a steady trickle (about 5-8% of a typical build's damage) or a
// rare save, so they're worth taking without carrying a run; Familiar Bond
// and Spirit Banner reward building around them.
//
//   Arcane Wisp   every 2.6 s, a bolt at the nearest brick (55% of an orb hit); up to two
//   Ember Sprite  every 3.2 s, sets the nearest unlit brick burning
//   Frost Moth    every 4 s, freezes the nearest unfrozen brick
//   Warden Owl    catches an orb that would be lost, then rests for 40 s

var familiarMods = []*Mod{
	{ID: "fam_wisp", Icon: "orb", Name: "Wisp Lantern", Desc: "An Arcane Wisp circles your paddle and fires a bolt at the nearest brick every few seconds (55% of an orb hit).", Rarity: Uncommon, Tags: []string{"void"}, Max: 2,
		Apply: func(s *Stats) { s.Wisps++ }},
	{ID: "fam_ember", Icon: "flame", Name: "Ember Sprite", Desc: "A fire sprite sets the nearest unlit brick burning every few seconds.", Rarity: Uncommon, Tags: []string{"fire"}, Max: 1,
		Apply: func(s *Stats) { s.Embers++ }},
	{ID: "fam_moth", Icon: "snow", Name: "Frost Moth", Desc: "A frost moth freezes the nearest brick every few seconds.", Rarity: Uncommon, Tags: []string{"frost"}, Max: 1,
		Apply: func(s *Stats) { s.Moths++ }},
	{ID: "fam_owl", Icon: "feather", Name: "Warden Owl", Desc: "An owl catches an orb that would be lost, then rests for 40 seconds.", Rarity: Rare, Relic: true, Tags: []string{"guard"}, Max: 1,
		Apply: func(s *Stats) { s.Owl = true }},
	{ID: "fam_bond", Icon: "heart", Name: "Familiar Bond", Desc: "Your familiars act 35% faster and hit 50% harder.", Rarity: Epic, Tags: []string{"power"}, Max: 1,
		Apply: func(s *Stats) { s.FamHaste += 0.35; s.FamPower += 0.5 }},
	{ID: "fam_banner", Icon: "star", Name: "Spirit Banner", Desc: "+5% damage for each kind of familiar you have (up to +20%).", Rarity: Rare, Tags: []string{"power"}, Max: 1,
		Apply: func(s *Stats) { s.FamBanner = true }},
	{ID: "fam_lure", Icon: "eye", Name: "Spirit Lure", Desc: "Familiars hit 25% harder and the owl rests 10 seconds less.", Rarity: Common, Tags: []string{"power"}, Max: 1,
		Apply: func(s *Stats) { s.FamPower += 0.25; s.OwlRest -= 10 }},
}

func init() {
	for _, m := range familiarMods {
		modList = append(modList, m)
		modByID[m.ID] = m
	}
}

// familiarKinds counts the kinds of familiar in a build.
func (s *Stats) familiarKinds() int {
	n := 0
	for _, k := range []bool{s.Wisps > 0, s.Embers > 0, s.Moths > 0, s.Owl} {
		if k {
			n++
		}
	}
	return n
}

// familiarOnly: the items that work through familiars are only offered to
// a build that has one.
func (r *Run) familiarOnly(m *Mod) bool {
	switch m.ID {
	case "fam_bond", "fam_banner", "fam_lure":
		return r.Stats.familiarKinds() > 0
	}
	return true
}

// familiarDmg is a familiar bolt's damage at the given share of an orb hit.
func (L *Level) familiarDmg(share float64) float64 {
	s := &L.g.run.Stats
	return share * s.Damage * s.DmgMult * math.Max(0.25, 1+s.FamPower)
}

// nearestBrick finds the nearest breakable brick to (x, z) that ok accepts.
func (L *Level) nearestBrick(x, z float64, ok func(b *Brick) bool) *Brick {
	var best *Brick
	bd := 1e9
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && b.Kind != levelgen.Steel && (ok == nil || ok(b)) {
			if d := math.Hypot(b.X()-x, b.Z()-z); d < bd {
				best, bd = b, d
			}
		}
	}
	return best
}

// familiarPos is where familiar i of n circles, above the paddle.
func (L *Level) familiarPos(i, n int) V3 {
	a := L.famAng + float64(i)*2*math.Pi/float64(max(1, n))
	return V3{L.PaddleX + math.Cos(a)*1.6, 1.7 + 0.25*math.Sin(L.T*2.3+float64(i)), PaddleZ + 1.4 + math.Sin(a)*0.7}
}

type famSlot struct {
	kind int // 0 wisp, 1 ember, 2 moth, 3 owl
	col  color.RGBA
}

var famCols = [...]color.RGBA{{190, 150, 255, 255}, {255, 150, 60, 255}, {160, 230, 255, 255}, {240, 220, 170, 255}}

// familiars lists the build's familiars in drawing order.
func (L *Level) familiars() []famSlot {
	s := &L.g.run.Stats
	var out []famSlot
	for i := 0; i < s.Wisps; i++ {
		out = append(out, famSlot{0, famCols[0]})
	}
	if s.Embers > 0 {
		out = append(out, famSlot{1, famCols[1]})
	}
	if s.Moths > 0 {
		out = append(out, famSlot{2, famCols[2]})
	}
	if s.Owl {
		out = append(out, famSlot{3, famCols[3]})
	}
	return out
}

// updateFamiliars runs the familiars each tick (Level.Update).
func (L *Level) updateFamiliars(dt float64) {
	s := &L.g.run.Stats
	if s.familiarKinds() == 0 {
		return
	}
	L.famAng += dt * 1.3
	if L.owlCD > 0 && L.owlCD-dt <= 0 && s.Owl {
		L.owlWakes()
	}
	L.owlCD = math.Max(0, L.owlCD-dt)
	if L.Cleared || L.introT > 0 || L.paddleGone > 0 || !L.ballsInPlay() {
		return
	}
	L.familiarMotes(dt)
	haste := 1 + s.FamHaste
	fams := L.familiars()
	for i, f := range fams {
		if f.kind == 3 {
			continue
		}
		L.famT[i] += dt * haste
		every := [...]float64{2.6, 3.2, 4}[f.kind]
		if L.famT[i] < every {
			continue
		}
		L.famT[i] = 0
		p := L.familiarPos(i, len(fams))
		var b *Brick
		switch f.kind {
		case 0:
			b = L.nearestBrick(p.X, p.Z, nil)
		case 1:
			b = L.nearestBrick(p.X, p.Z, func(b *Brick) bool { return b.Burn <= 0 })
		case 2:
			b = L.nearestBrick(p.X, p.Z, func(b *Brick) bool { return b.Frozen <= 0 })
		}
		if b == nil {
			continue
		}
		L.Arcs = append(L.Arcs, &Arc{Pts: jag(p, V3{b.X(), 0.8, b.Z()}, L.rng), Life: 0.22, Col: f.col})
		L.familiarImpact(V3{b.X(), 0.8, b.Z()}, f.col)
		L.familiarShotFX(f.kind, p, V3{b.X(), 0.8, b.Z()}, f.col)
		switch f.kind {
		case 0:
			L.damage(b, Hit{Src: "Familiars", Dmg: L.familiarDmg(0.55), Silent: true})
			L.g.snd.PlayVol("zap", 0.35)
		case 1:
			if b.Burn <= 0 {
				L.text(V3{b.X(), 1.2, b.Z()}, "BURN", color.RGBA{255, 140, 40, 255}, false)
			}
			b.Burn = 3.5
			L.damage(b, Hit{Src: "Familiars", Dmg: L.familiarDmg(0.2), Fire: true, Silent: true})
		case 2:
			b.Frozen = 4 + s.FreezeDur
			L.damage(b, Hit{Src: "Familiars", Dmg: L.familiarDmg(0.2), Silent: true})
			L.g.snd.PlayVol("freeze", 0.5)
		}
	}
}

// owlCatch is the Warden Owl saving an orb at the barrier line (the ball
// loop, before the barriers spend a charge).
func (L *Level) owlCatch(b *Ball) bool {
	s := &L.g.run.Stats
	if !s.Owl || L.owlCD > 0 || b.VZ >= 0 || b.Z-b.R > BarrierZ || b.Z < BarrierZ-1 || b.Phantom > 0 {
		return false
	}
	b.Z = BarrierZ + b.R
	b.VZ = math.Abs(b.VZ)
	L.owlCD = math.Max(15, 40+s.OwlRest)
	L.sparks(V3{b.X, 0.4, BarrierZ}, 14, famCols[3], 6)
	L.text(V3{b.X, 1.6, BarrierZ + 1}, "SAVED", famCols[3], false)
	L.owlFlourish(b.X)
	L.g.snd.Play("shield")
	L.g.snd.PlayVol("owlsave", 0.5)
	return true
}

// drawFamiliars draws the familiars circling the paddle.
func (L *Level) drawFamiliars(B *Batch) {
	fams := L.familiars()
	if len(fams) > 0 {
		L.bannerAura(B)
		L.bondLinks(B, fams)
	}
	rise, fade := L.familiarsLeave()
	for i, f := range fams {
		p, sz := L.famLook(f.kind, i, L.familiarPos(i, len(fams)))
		p.Y += rise
		if f.kind == 0 && L.youngWisp() {
			sz *= 0.7 // the Wisp Egg's young wisp
			f.col = mixc(f.col, cWhite, 0.4)
		}
		a := fade // familiars fade as they leave a cleared chamber
		if f.kind == 3 {
			a *= L.owlReadyAlpha() // dim while resting, brightening as it wakes
		}
		if f.kind == 0 {
			L.wispTrail(B, i, len(fams), f.col)
		}
		if f.kind == 3 {
			L.owlGlint(B, p)
		}
		B.Billboard(p, 0.9*sz, sprSoft, alpha(f.col, 0.35*a*L.familiarPulse(i)), true)
		B.Billboard(p, 0.36*sz, sprSphere, alpha(lighten(f.col, 0.3), a), false)
		// a charge glint as the next shot comes due
		if f.kind != 3 {
			every := [...]float64{2.6, 3.2, 4}[f.kind]
			if k := L.famT[i] / every; k > 0.8 {
				B.Billboard(p, 0.6*(k-0.8)*5, sprSpark, alpha(color.RGBA{255, 255, 255, 255}, 0.8), true)
			}
		}
	}
}
