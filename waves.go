package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Chambers come in waves, the formation creeps toward the paddle, and
// heat (how strong the build has grown this world) hardens the bricks.
// Rune caches drop from bricks mid-fight so a build grows during a
// chamber, not just after it.

// gridShift is how far the brick grid has advanced toward the paddle in
// the level being updated or drawn (world units).
var gridShift float64

const (
	dangerZ      = PaddleZ + 4.6 // bricks that reach this line smash into the paddle's side of the arena
	advanceSlide = 0.45          // seconds an advance takes
	advanceStep  = 2 * CellD     // how far the formation creeps each advance
)

// ---------------------------------------------------------------- heat

// heat counts the power a build has gathered this world: one per item
// copy, two per fusion. Each point makes bricks a little tougher and the
// formation a little quicker to advance.
func (r *Run) heat() int {
	n := 0
	for id, k := range r.Mods {
		if m := modByID[id]; m != nil && !m.Curse && !strings.HasPrefix(id, "__") {
			n += k
		}
	}
	return n + 2*len(r.Fused)
}

// hpMul is the brick health multiplier for this chamber.
func (L *Level) hpMul() float64 {
	r := L.g.run
	act := r.Act
	m := (1.0 + float64(act-1)*0.15) * (1 + 0.025*float64(r.ActDepth)) * ascHP(r.Asc, L.Node == NElite, false) * math.Pow(1.25, float64(r.Loop()))
	m *= 1 + heatHP*float64(L.heat)
	if r.Cursed && L.Node != NBoss {
		m *= 1.5
	}
	return m * (1 + r.Stats.BrickHPMult)
}

const heatHP = 0.07 // extra brick health per point of heat

// heatArmour turns some plain bricks into armoured ones; the share grows
// with heat and with each world.
func (L *Level) heatArmour() {
	if L.Node == NBoss {
		return
	}
	r := L.g.run
	p := math.Min(0.3, 0.012*float64(L.heat)+0.03*float64(r.Act-1))
	if p <= 0 {
		return
	}
	for _, b := range L.Bricks {
		if b.Alive && b.Kind == levelgen.Basic && !b.BossPart && L.rng.Float64() < p {
			b.Kind = levelgen.Armored
			b.Col = kindColor(levelgen.Armored, b.Col)
		}
	}
}

// ---------------------------------------------------------------- waves

func (L *Level) setupWaves() {
	r := L.g.run
	L.heat = r.heat()
	// a chamber is one big, packed board; elites and duel rounds still
	// come in two waves
	L.wave, L.waves = 1, 1
	switch {
	case L.Node == NBoss:
		L.waves = 1
	case L.Node == NElite:
		L.waves = 2
	case r.Duel:
		L.waves = 2 // every duel round is two waves
	}
	L.advT = L.advInterval()
	L.cacheAt = -1
}

// waveBudget is each wave's share of a whole chamber's work.
func waveBudget(waves int) float64 {
	switch waves {
	case 1:
		return 1
	case 2:
		return 0.5
	}
	return 0.42
}

func (L *Level) waveParams(seed int64) levelgen.Params {
	r := L.g.run
	return levelgen.Params{Seed: seed, Depth: r.ActDepth + (r.Act-1)*2, Act: r.Act, Elite: L.Node == NElite}
}

// waveBreak is the share of an early wave still standing when it breaks.
const waveBreak = 0.15

// finalBreak is the share of a packed last wave left when it crumbles.
const finalBreak = 0.07

// crumbleRest breaks every brick still standing, counting the required
// ones as broken.
func (L *Level) crumbleRest() {
	g := L.g
	for _, b := range L.Bricks {
		if b.Alive && b.Kind.Required() {
			b.Alive = false
			g.run.Bricks++
			L.breakPop(b, b.X(), b.Z())
			L.debris(V3{b.X(), 0.5, b.Z()}, 6, b.Col)
			if b.R >= 0 && b.R < levelgen.MaxRows && b.C >= 0 && b.C < levelgen.Cols && L.Grid[b.R][b.C] == b {
				L.Grid[b.R][b.C] = nil
			}
		}
	}
	L.Left = 0
	L.Shake = math.Max(L.Shake, 0.3)
}

// nextWave sweeps the floor and drops the next formation in from above.
func (L *Level) nextWave() {
	g := L.g
	L.wave++
	for _, b := range L.Bricks {
		if b.Alive {
			b.Alive = false
			if b.Kind.Required() {
				g.run.Bricks++ // the stragglers count as broken
				L.breakPop(b, b.X(), b.Z())
			}
			L.debris(V3{b.X(), 0.5, b.Z()}, 6, b.Col)
		}
	}
	L.Bricks = nil
	L.Grid = [levelgen.MaxRows][levelgen.Cols]*Brick{}
	L.Left, L.Crystals = 0, 0
	L.shift, L.shiftTarget, gridShift = 0, 0, 0

	shape := L.Data.Shape
	d := pacedLevelScaled(L.waveParams(L.Data.Seed+int64(L.wave)*1013), waveBudget(L.waves))
	// keep the arena the chamber started in
	poly := levelgen.Arena(shape)
	for rr := 0; rr < levelgen.MaxRows; rr++ {
		for c := 0; c < levelgen.Cols; c++ {
			out := !levelgen.CellInside(poly, rr, c)
			d.Outside[rr][c] = out
			if out {
				d.Cells[rr][c] = levelgen.Cell{}
			}
		}
	}
	d.Shape = shape
	L.Data = d
	L.Designed = strings.HasPrefix(d.Formation, "design")
	L.heat = g.run.heat()
	hpMul := L.hpMul()
	for row := 0; row < levelgen.MaxRows; row++ {
		for c := 0; c < levelgen.Cols; c++ {
			cell := d.Cells[row][c]
			if cell.Kind == levelgen.Empty {
				continue
			}
			hp := float64(cell.HP)
			if cell.Kind != levelgen.Steel {
				hp = math.Ceil(hp * hpMul)
			}
			L.addColumn(row, c, cell, hp, hpMul)
		}
	}
	// the new formation falls in from above, front rows last
	for _, b := range L.Bricks {
		b.Appear = 0
		b.FallY = 6 + float64(b.R)*0.35
	}
	if o := L.omen; o != nil {
		if o.ID == "fortified" {
			for _, b := range L.Bricks {
				if b.Kind != levelgen.Steel {
					b.MaxHP = math.Ceil(b.MaxHP * 1.4)
					b.HP = b.MaxHP
				}
			}
		}
		if o.Setup != nil && (o.ID == "volatile" || o.ID == "seam") {
			o.Setup(L)
		}
	}
	L.heatArmour()
	L.startLeft = L.Left
	L.planCache()
	L.advT = L.advInterval()
	L.advWarned = false
	// the orbs come home and the player serves again
	L.Balls = nil // capsules still falling are kept
	L.spawnServe()
	L.waveT = 1.3
	L.text(V3{0, 3, 16}, fmt.Sprintf("WAVE %d OF %d", L.wave, L.waves), color.RGBA{255, 222, 128, 255}, true)
	L.Shake = math.Max(L.Shake, 0.4)
	L.Flash = math.Max(L.Flash, 0.2)
	g.snd.Play("warp")
	g.snd.PlayVol("abilready", 0.7)
	g.react(reactCombo)
}

// ---------------------------------------------------------------- advancing rows

// advInterval is how long the formation holds before it creeps forward.
func (L *Level) advInterval() float64 {
	r := L.g.run
	t := 20 - 0.4*float64(L.heat) - 1.5*float64(r.Act-1)
	if L.Node == NElite {
		t *= 0.85
	}
	return math.Max(10, t)
}

func (L *Level) updateAdvance(dt float64) {
	if L.waveT > 0 {
		L.waveT -= dt
	}
	if L.Boss != nil || L.Cleared {
		return
	}
	if L.shift < L.shiftTarget {
		L.shift = math.Min(L.shiftTarget, L.shift+dt*advanceStep/advanceSlide)
		gridShift = L.shift
		L.crushCheck()
	}
	inPlay := false
	for _, b := range L.Balls {
		if !b.Stuck {
			inPlay = true
			break
		}
	}
	if !inPlay || L.waveT > 0 || L.paddleGone > 0 {
		return
	}
	L.advT -= dt
	if L.advT <= 2 && !L.advWarned {
		L.advWarned = true
		L.g.snd.PlayVol("thump", 0.8)
	}
	if L.advT <= 0 {
		L.shiftTarget += advanceStep
		L.advT = L.advInterval()
		L.advWarned = false
		L.Shake = math.Max(L.Shake, 0.35)
		L.g.snd.Play("boom")
		L.text(V3{0, 2.6, L.frontZ() - 1.5}, "THE WALL ADVANCES", color.RGBA{255, 170, 120, 255}, false)
	}
}

// frontZ is the nearest brick's z (the formation's front line).
func (L *Level) frontZ() float64 {
	z := FarZ
	for _, b := range L.Bricks {
		if b.Alive && !b.BossPart {
			z = math.Min(z, b.Z())
		}
	}
	return z
}

// crushCheck: bricks pushed past the danger line smash and hurt; bricks
// pushed outside the walls crumble harmlessly.
func (L *Level) crushCheck() {
	poly := L.Walls
	smashed := 0
	for r := levelgen.MaxRows - 1; r >= 0; r-- {
		for c := 0; c < levelgen.Cols; c++ {
			b := L.Grid[r][c]
			if b == nil || !b.Alive {
				continue
			}
			x, z := b.X(), b.Z()
			switch {
			case z-CellD/2 < dangerZ:
				smashed++
				L.crumbleColumn(r, c)
				L.sparks(V3{x, 0.6, z}, 14, color.RGBA{255, 120, 80, 255}, 8)
			case len(poly) > 0 && !levelgen.Inside(poly, levelgen.Pt{X: x, Z: z}):
				L.crumbleColumn(r, c)
			}
		}
	}
	if smashed > 0 {
		// no health outside a Guardian fight: the smashed bricks are lost
		// loot, the paddle reels and a little gold spills
		L.crushes++
		r := L.g.run
		lost := min(r.Gold, 2+smashed/2+r.Act)
		r.Gold -= lost
		L.stun = math.Max(L.stun, 0.5)
		L.Shake = math.Max(L.Shake, 0.6)
		L.hurtFlash = 0.6
		msg := "CRUSHED"
		if lost > 0 {
			msg = fmt.Sprintf("CRUSHED  −%d gold", lost)
		}
		L.text(V3{L.PaddleX, 1.5, PaddleZ + 2}, msg, color.RGBA{255, 120, 100, 255}, true)
		L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: dangerZ, MaxR: 6, Life: 0.4, Max: 0.4, Col: color.RGBA{255, 90, 70, 255}})
	}
}

// crumbleColumn removes a whole stack without rewards.
func (L *Level) crumbleColumn(r, c int) {
	for b := L.Grid[r][c]; b != nil; b = b.Up {
		if !b.Alive {
			continue
		}
		b.Alive = false
		if b.Kind.Required() && !b.BossPart {
			L.Left--
		}
		if b.Kind == levelgen.Crystal {
			L.Crystals--
		}
		L.debris(V3{b.X(), 0.5 + float64(b.Layer)*LayerH, b.Z()}, 6, b.Col)
	}
	L.Grid[r][c] = nil
}

// ---------------------------------------------------------------- rune caches

// planCache decides whether this wave hides a rune cache and how far
// into the wave it breaks loose.
func (L *Level) planCache() {
	L.cacheAt = -1
	if L.Node == NBoss || L.g.run.SeedCode == "MENU" || L.g.run.Duel {
		return // duels stay equal: no rune caches
	}
	p := 0.72
	if L.Node == NElite {
		p = 0.85
	}
	if L.waves == 1 {
		p = 0.9 // a short chamber always pays something
	}
	L.cacheTwice = L.waves == 1 && L.Node != NElite
	if L.cacheTwice {
		// one big board stands in for two waves: up to two caches,
		// the first in its opening half
		if L.rng.Float64() < p {
			L.cacheAt = 0.2 + 0.3*L.rng.Float64()
		}
		return
	}
	if L.rng.Float64() < p {
		L.cacheAt = 0.25 + 0.6*L.rng.Float64()
	}
}

func (L *Level) maybeDropCache(x, z float64) {
	if L.cacheAt < 0 || L.startLeft <= 0 {
		return
	}
	if float64(L.startLeft-L.Left)/float64(L.startLeft) >= L.cacheAt {
		L.cacheAt = -1
		if L.cacheTwice {
			L.cacheTwice = false
			if L.rng.Float64() < 0.6 {
				L.cacheAt = 0.6 + 0.3*L.rng.Float64()
			}
		}
		L.caches++
		L.Drops = append(L.Drops, &Drop{X: x, Z: z, P: PCache})
		L.text(V3{x, 2.2, z}, "A RUNE CACHE FALLS!", color.RGBA{255, 222, 110, 255}, true)
		L.g.snd.PlayVol("abilready", 0.8)
	}
}

// cacheCaught is the paddle catching a cache: the game stops to offer a
// choice of items.
func (L *Level) cacheCaught() {
	L.pendingPick++
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2}, "RUNE CACHE", color.RGBA{255, 222, 110, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 4, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 214, 90, 255}})
	L.sparks(V3{L.PaddleX, 1, PaddleZ}, 30, color.RGBA{255, 220, 120, 255}, 8)
	L.g.snd.Play("pick.2." + itoa(L.g.snd.Key))
}

// ---------------------------------------------------------------- the cache choice

// openCache stops play to offer the caught cache's items.
func (g *Game) openCache() {
	L := g.level
	r := g.run
	rng := rand.New(rand.NewSource(L.rng.Int63()))
	offers := r.Offer(3, false, Common, rng)
	if len(offers) == 0 || inputHook != nil {
		// nothing to offer, or a scripted run: take the first and play on
		if len(offers) > 0 {
			g.take(offers[0])
		}
		L.pendingPick--
		return
	}
	g.cacheOffers = offers
	g.setState(StCache)
	g.coach("cache")
}

// drawCache is the in-chamber item choice over the frozen arena.
func (g *Game) drawCache(s *ebiten.Image) {
	u := g.ui
	L := g.level
	fillRect(s, 0, 0, u.SW, u.SH, color.RGBA{6, 4, 14, 175})
	u.heading(s, "RUNE CACHE", 56, 90, rgba(255, 222, 128, 255), 1)
	u.text(s, "Choose one. The fight resumes the moment you do.", fReg, 22, 960, 172, cDim, alignCenter)
	n := len(g.cacheOffers)
	w, h := 400.0, 560.0
	x0 := 960 - (float64(n)*w+float64(n-1)*40)/2
	done := func() {
		L.pendingPick--
		if L.pendingPick > 0 {
			g.openCache()
			g.stateT = 0
			return
		}
		g.cacheOffers = nil
		g.setState(StLevel)
	}
	for i, m := range g.cacheOffers {
		if g.drawCard(s, m, x0+float64(i)*(w+40), 250, w, h, 0, true) && g.stateT > 0.3 {
			g.take(m)
			g.snd.Pick(m.Rarity)
			done()
			return
		}
	}
	if u.button(s, "cacheskip", "Take 10 gold instead", 760, 900, 400, 60, cAcc, true) && g.stateT > 0.3 {
		g.run.Gold += 10
		done()
	}
}

// drawWaveStrip shows the wave, the heat and the time to the next advance
// under the location plaque; it returns the y below it.
func (g *Game) drawWaveStrip(s *ebiten.Image, L *Level, y float64) float64 {
	u := g.ui
	S := u.S
	const w, h = 460.0, 44.0
	x := 960 - w/2
	u.frame(s, x, y, w, h, 0.85)
	gold := rgba(255, 222, 128, 255)
	u.text(s, fmt.Sprintf("WAVE %d / %d", L.wave, L.waves), fBold, 20, x+20, y+10, gold, alignLeft)
	hc := mixc(rgba(255, 190, 120, 255), rgba(255, 80, 60, 255), math.Min(1, float64(L.heat)/20))
	u.text(s, fmt.Sprintf("HEAT %d", L.heat), fBold, 20, x+160, y+10, hc, alignLeft)
	// advance gauge: fills as the formation gets ready to move
	bx, bw := x+272, 168.0
	k := 1 - clamp01(L.advT/L.advInterval())
	fillPath(s, boxPath(u.X(bx), u.Y(y+24), bw*S, 10*S, 5*S, false), rgba(0, 0, 0, 160), rgba(20, 14, 30, 160), grad{}, false)
	c := mixc(rgba(255, 200, 140, 255), rgba(255, 70, 60, 255), k)
	if L.advT < 2 {
		c = mixc(c, cWhite, 0.4+0.4*math.Sin(L.T*14))
	}
	if k*bw > 2 {
		fillPath(s, boxPath(u.X(bx), u.Y(y+24), k*bw*S, 10*S, 5*S, false), lighten(c, 0.2), darken(c, 0.3), grad{}, false)
	}
	u.text(s, "ADVANCE", fBold, 13, bx, y+6, ca(cDim, 0.95), alignLeft)
	if L.advT < 5 {
		u.text(s, fmt.Sprintf("%.0fs", math.Ceil(L.advT)), fBold, 13, bx+bw, y+6, c, alignRight)
	}
	if u.hover(x, y, w, h) {
		g.tooltip = fmt.Sprintf("Heat %d: your build's strength this world. Each point adds %.0f%% brick health, more armoured bricks and a quicker advance. The formation creeps one row closer when the gauge fills; bricks that reach the red line smash for damage.", L.heat, heatHP*100)
	}
	return y + h + 8
}

// drawDangerLine marks where advancing bricks smash, brighter as the
// front line nears it.
func (g *Game) drawDangerLine(B *Batch, L *Level) {
	if L.Boss != nil || L.Cleared {
		return
	}
	gap := L.frontZ() - CellD/2 - dangerZ
	a := clamp01(1 - gap/(CellD*4))
	if a <= 0.02 {
		return
	}
	pulse := 0.7 + 0.3*math.Sin(L.T*6)
	B.Decal(0, dangerZ, 0.03, 2*HalfW, 0.16, sprWhite, alpha(color.RGBA{255, 70, 60, 255}, 0.75*a*pulse), true)
	B.Decal(0, dangerZ, 0.02, 2*HalfW, 0.9, sprSoft, alpha(color.RGBA{255, 60, 40, 255}, 0.3*a*pulse), true)
}
