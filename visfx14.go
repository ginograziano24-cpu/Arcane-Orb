package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Visual upgrades, round 14 (ROADMAP section Y): the new systems shown in
// the world (trials, keepsakes, familiars, co-op) and a few shop, reward
// and origin-screen cues.

// ---------------------------------------------------------------- world

// guardianTrialAura: with the Relentless Guardians trial, the Guardian's
// core sits in a slow red glow.
func (L *Level) guardianTrialAura(B *Batch) {
	if L.g.ambientOff() {
		return
	}
	if L.Boss == nil || L.Boss.Dead || L.g.run.Stats.BossHPMult <= 0 {
		return
	}
	for _, b := range L.Bricks {
		if b.Core && b.Alive {
			p := 0.7 + 0.3*math.Sin(L.T*2.4)
			B.Billboard(V3{b.X(), 1.2, b.Z()}, 4.2, sprSoft, alpha(cTrial, 0.18*p), true)
			return
		}
	}
}

// wardBarrier: a barrier from the Spare Ward shines gold for the first
// moments of the chamber, so you know where it came from.
func (L *Level) wardBarrier(B *Batch) {
	r := L.g.run
	if r.Keepsake != "ward" || r.ActDepth != 1 || L.Barrier <= 0 || L.T > 2.5 {
		return
	}
	a := 0.6 * (1 - L.T/2.5)
	B.Decal(0, BarrierZ, 0.07, HalfW*2, 0.45, sprSoft, alpha(cGoldRing, a), true)
}

// bondLinks: with Familiar Bond, faint threads of light join the familiars.
func (L *Level) bondLinks(B *Batch, fams []famSlot) {
	if L.g.run.Stats.FamHaste <= 0 || len(fams) < 2 {
		return
	}
	n := len(fams)
	for i := 0; i < n; i++ {
		a, b := L.familiarPos(i, n), L.familiarPos((i+1)%n, n)
		for k := 1; k < 4; k++ {
			t := float64(k) / 4
			p := V3{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t}
			B.Billboard(p, 0.16, sprSoft, alpha(rgba(255, 230, 200, 255), 0.35), true)
		}
	}
}

// youngWisp reports whether the build's only wisp is the Wisp Egg's young
// one: it's drawn smaller and paler until a real one joins.
func (L *Level) youngWisp() bool {
	r := L.g.run
	return r.Keepsake == "egg" && r.Mods["fam_wisp"] == 0
}

// familiarsLeave: once the chamber is cleared the familiars drift up and
// fade (a scale and fade for drawFamiliars).
func (L *Level) familiarsLeave() (rise, fade float64) {
	if !L.Cleared {
		return 0, 1
	}
	t := math.Min(1, L.clearShown()/1.5)
	return 1.6 * t, 1 - 0.7*t
}

// partnerContact: sparks when the two co-op paddles bump.
func (L *Level) partnerContact() {
	if !L.p2On {
		return
	}
	touching := math.Abs(L.p2X-L.PaddleX) < L.PaddleW*1.02
	if touching && !L.p2Touch {
		L.sparks(V3{(L.p2X + L.PaddleX) / 2, 0.6, PaddleZ}, 8, mixc(cP2, rgba(255, 200, 120, 255), 0.5), 4)
	}
	L.p2Touch = touching
}

// partnerJoin: a teal ring at player two's paddle as a chamber starts.
func (L *Level) partnerJoin() {
	L.Rings = append(L.Rings, &Ring{X: L.p2X, Z: PaddleZ, MaxR: 2.4, Life: 0.6, Max: 0.6, Col: cP2})
}

// ---------------------------------------------------------------- screens

// knotFlash: the Guardian's Knot catching the run washes the screen gold.
func (g *Game) knotFlash(dst *ebiten.Image) {
	if g.knotFlashT <= 0 {
		return
	}
	k := g.knotFlashT
	g.knotFlashT = math.Max(0, k-g.ui.DT*2)
	if !g.settings.Flashes {
		k *= 0.4
	}
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	fillRect(dst, 0, 0, sw, sh, color.RGBA{uint8(80 * k), uint8(64 * k), uint8(20 * k), uint8(80 * k)})
}

// coinRain: with the Lucky Coin, a few coins tumble past the HUD gold as an
// act's first chamber starts.
func (g *Game) coinRain(dst *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	L := g.level
	r := g.run
	if L == nil || r == nil || r.Keepsake != "coin" || r.ActDepth != 1 || L.T > 1.4 || g.state != StLevel {
		return
	}
	u := g.ui
	for i := 0; i < 6; i++ {
		t := L.T - float64(i)*0.12
		if t < 0 || t > 0.9 {
			continue
		}
		x := 250 + float64(i%3)*22
		y := 10 + t*80
		a := 1 - t/0.9
		drawSprite(dst, sprSoft, u.X(x-9), u.Y(y-9), 18*u.S, 18*u.S, ca(cGold, 0.8*a), true)
		u.icon(dst, icCoin, x, y, 16, a)
	}
}

// soldStamp: a SOLD card stamps down when the item is bought.
func (g *Game) soldStamp(s *ebiten.Image, i int, x, y, w, h float64) {
	if g.flourishOff() {
		return
	}
	t, ok := g.soldT[i]
	if !ok {
		return
	}
	t += g.ui.DT
	g.soldT[i] = t
	if t > 0.5 {
		return
	}
	u := g.ui
	k := t / 0.5
	sc := 1.6 - 0.6*ease(k)
	u.withScale(sc, x+w/2, y+h/2-10, 0, func() {
		u.text(s, "SOLD", fRune, 40, x+w/2, y+h/2-24, ca(cGold, 1-k), alignCenter)
	})
	drawSprite(s, sprSoft, u.X(x), u.Y(y+h/2-60), w*u.S, 100*u.S, ca(cGold, 0.3*(1-k)), true)
}

// trialFlare: the HUD trial embers flare as a Guardian chamber begins.
func (g *Game) trialFlare(dst *ebiten.Image, x, y, w float64) {
	if g.flourishOff() {
		return
	}
	L := g.level
	if L == nil || L.Boss == nil || L.T > 1.5 {
		return
	}
	u := g.ui
	a := 1 - L.T/1.5
	drawSprite(dst, sprSoft, u.X(x-14), u.Y(y-18), (w+28)*u.S, 36*u.S, ca(cTrial, 0.6*a), true)
}

// owlRestArc: the HUD owl dot shows its rest as a closing ring.
func (g *Game) owlRestArc(dst *ebiten.Image, L *Level, x, y float64) {
	if L == nil || L.owlCD <= 0 {
		return
	}
	u := g.ui
	full := math.Max(15, 40+g.run.Stats.OwlRest)
	k := L.owlCD / full
	strokePath(dst, arcPath(u.X(x), u.Y(y), 8*u.S, -math.Pi/2, -math.Pi/2+2*math.Pi*k), 1.6*u.S, ca(famCols[3], 0.8), false)
}

// keepAffordable reports whether any keepsake rank can be bought now.
func (g *Game) keepAffordable() bool {
	sv := &g.save
	for i := range keepsakes {
		k := &keepsakes[i]
		r := sv.keepRank(k.ID)
		if r < k.Ranks && sv.keepOpen(k) && sv.Shards >= k.cost(r) {
			return true
		}
	}
	return false
}

// keepUpgradeBadge: a shard gem on the origin screen's Keepsake button when
// an upgrade is affordable.
func (g *Game) keepUpgradeBadge(s *ebiten.Image, x, y float64) {
	if !g.keepAffordable() {
		return
	}
	u := g.ui
	p := 0.6 + 0.4*math.Sin(g.stateT*4)
	drawSprite(s, sprSoft, u.X(x-14), u.Y(y-14), 28*u.S, 28*u.S, ca(rgba(200, 180, 255, 255), 0.8*p), true)
	u.gem(s, x, y, 6, 8, rgba(215, 195, 255, 255), 1)
}

// banishPipPop: the spent banish pip pops away under the Banish button.
func (g *Game) banishPipPop(s *ebiten.Image) {
	if g.flourishOff() {
		return
	}
	if !g.banishArriveOn || g.banishArriveT > 0.4 {
		return
	}
	u := g.ui
	n := g.run.banishesLeft() + 1
	px := 610 + float64(n-1)*16 - float64(min(n, 6)-1)*8
	k := g.banishArriveT / 0.4
	r := 5 + 12*k
	drawSprite(s, sprSoft, u.X(px-r), u.Y(974-r), 2*r*u.S, 2*r*u.S, ca(cBanish, 0.9*(1-k)), true)
}
