package main

import (
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Feel and visual polish, 2026-11-11 session 7:
//   - the paddle squashes and stretches on a hard return, with a very slight
//     arcane crackle where the orb struck
//   - bricks show cracks at two-thirds and one-third health
//   - the combo number heats from white to gold to fire, with embers
//   - a slain Guardian's armour shatters outward from its core, piece by piece
//   - the map wears the season: snow, petals, fireflies or falling leaves

// ---------------------------------------------------------------- paddle

// paddleStrike records a paddle return: how hard (0..1) and where.
func (L *Level) paddleStrike(b *Ball, sp float64) {
	base := math.Max(1, L.speed())
	k := clamp01((sp/base - 0.85) / 0.6) // a normal return is ~0.25, a fast orb 1
	k = math.Max(0.25, k)
	L.padSquash = math.Max(L.padSquash, k)
	if k < 0.45 || L.g.settings.Particles == 0 {
		return // the crackle is for the hard ones only, and stays slight
	}
	rng := L.fxr()
	c0 := color.RGBA{190, 140, 255, 255}
	c1 := color.RGBA{120, 230, 255, 255}
	n := 2 + int(k*3)
	for i := 0; i < n; i++ {
		a := rng.Float64() * math.Pi // up off the paddle's face
		s := 2 + rng.Float64()*2.5
		c := mixc(c0, c1, rng.Float64())
		L.Parts = append(L.Parts, Particle{P: V3{b.X + (rng.Float64()-0.5)*0.5, PaddleH + 0.1, PaddleZ + PaddleD/2},
			V: V3{math.Cos(a) * s, 1 + rng.Float64()*1.5, math.Sin(a) * s * 0.5},
			Life: 0.12 + rng.Float64()*0.1, Max: 0.22, Size: 0.12 + rng.Float64()*0.1, Col: c, Kind: pSpark, Grav: 2})
	}
	L.crackT = 0.16
	L.crackX = b.X
}

// paddleScale is the squash: flatter and a touch wider just after a hard
// return, springing back with a small overshoot.
func (L *Level) paddleScale() (sx, sy float64) {
	k := L.padSquash
	if k <= 0 {
		return 1, 1
	}
	// a damped spring: squash, then a little stretch, then rest
	ph := (1 - k) * math.Pi * 1.6
	s := k * math.Cos(ph)
	return 1 + 0.07*s, 1 - 0.22*s
}

func (L *Level) updatePaddleFeel(dt float64) {
	L.padSquash = math.Max(0, L.padSquash-dt*4.5)
	L.crackT = math.Max(0, L.crackT-dt)
}

// drawCrackle is the faint arc of light along the paddle's face after a
// hard return: two short jagged threads, gone in a sixth of a second.
func (L *Level) drawCrackle(B *Batch) {
	if L.crackT <= 0 {
		return
	}
	a := L.crackT / 0.16
	rng := L.fxr()
	y := PaddleH + 0.08
	z := PaddleZ + PaddleD/2 + 0.02
	for t := 0; t < 2; t++ {
		x := L.crackX
		dir := float64(t*2 - 1)
		for i := 0; i < 4; i++ {
			nx := x + dir*(0.25+rng.Float64()*0.25)
			dz := (rng.Float64() - 0.5) * 0.18
			p := [4]V3{{x, y, z}, {nx, y, z + dz}, {nx, y + 0.035, z + dz}, {x, y + 0.035, z}}
			B.Quad(p, sprWhite, alpha(color.RGBA{200, 170, 255, 255}, 0.55*a), [4]float32{1, 1, 1, 1}, true)
			x = nx
		}
	}
}

// ---------------------------------------------------------------- cracks

// drawCracks scores the top of a worn brick: a couple of cracks at two
// thirds of its health, a web of them at one third. The pattern is fixed per
// brick so it doesn't flicker.
func drawCracks(B *Batch, b *Brick, x, z, hw, hd, top, frac float64) {
	if frac > 0.66 || b.MaxHP <= 1 || b.BossPart || b.Up != nil {
		return
	}
	n := 2
	if frac <= 0.33 {
		n = 5
	}
	seed := uint32(b.R*131+b.C*977) | 1
	rnd := func() float64 {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return float64(seed%10000) / 10000
	}
	y := top + 0.008
	ink := color.RGBA{18, 12, 20, 170}
	for i := 0; i < n; i++ {
		// each crack starts near an edge and wanders inward in 3 steps
		px := x + (rnd()-0.5)*hw*1.6
		pz := z + (rnd()-0.5)*hd*1.6
		a := rnd() * 2 * math.Pi
		for s := 0; s < 3; s++ {
			a += (rnd() - 0.5) * 1.4
			l := 0.18 + rnd()*0.22
			nx := math.Max(x-hw*0.95, math.Min(x+hw*0.95, px+math.Cos(a)*l))
			nz := math.Max(z-hd*0.95, math.Min(z+hd*0.95, pz+math.Sin(a)*l))
			w := 0.035 * (1 - float64(s)*0.25)
			ox, oz := -math.Sin(a)*w, math.Cos(a)*w
			p := [4]V3{{px - ox, y, pz - oz}, {nx - ox*0.6, y, nz - oz*0.6}, {nx + ox*0.6, y, nz + oz*0.6}, {px + ox, y, pz + oz}}
			B.Quad(p, sprWhite, ink, [4]float32{1, 1, 1, 1}, false)
			B.lastQuad().depth -= 0.02 // in front of the brick's own top
			px, pz = nx, nz
		}
	}
}

// ---------------------------------------------------------------- combo heat

// comboHeat is the combo number's colour: white, warming to gold by 10,
// orange by 25 and fire by 40.
func comboHeat(n int) (color.RGBA, float64) {
	f := float64(n)
	white, gold := rgba(250, 248, 240, 255), rgba(255, 214, 90, 255)
	orange, fire := rgba(255, 140, 50, 255), rgba(255, 64, 70, 255)
	switch {
	case f < 10:
		return mixc(white, gold, clamp01((f-3)/7)), 0
	case f < 25:
		return mixc(gold, orange, (f-10)/15), (f - 10) / 30
	default:
		return mixc(orange, fire, clamp01((f-25)/15)), math.Min(1, 0.5+(f-25)/30)
	}
}

// drawComboEmbers lifts a few embers off a hot combo number.
func (g *Game) drawComboEmbers(s *ebiten.Image, cx, cy, heat, a float64) {
	if heat <= 0 {
		return
	}
	u := g.ui
	c, _ := comboHeat(40)
	drawSprite(s, sprSoft, u.X(cx-70), u.Y(cy-50), 140*u.S, 100*u.S, ca(c, 0.22*heat*a), true)
	n := int(3 + 7*heat)
	for i := 0; i < n; i++ {
		ph := math.Mod(u.time*0.9+float64(i)*0.37, 1)
		ex := cx + math.Sin(float64(i)*2.3+u.time*2)*30*(0.4+ph)
		ey := cy + 20 - ph*70
		sz := (5 + 4*math.Sin(float64(i))) * (1 - ph)
		drawSprite(s, sprSoft, u.X(ex-sz), u.Y(ey-sz), 2*sz*u.S, 2*sz*u.S, ca(lighten(c, 0.3), (1-ph)*heat*a), true)
	}
}

// ---------------------------------------------------------------- Guardian

// startShatter lines the Guardian's remaining armour up from the core
// outward, for the death to break it in that order.
func (L *Level) startShatter() {
	cx, cz := 0.0, 0.0
	var parts []*Brick
	for _, br := range L.Bricks {
		if br.Alive && br.BossPart {
			parts = append(parts, br)
			if br.Core {
				cx, cz = br.X(), br.Z()
			}
		}
	}
	sort.SliceStable(parts, func(i, j int) bool {
		di := math.Hypot(parts[i].X()-cx, parts[i].Z()-cz)
		dj := math.Hypot(parts[j].X()-cx, parts[j].Z()-cz)
		return di < dj
	})
	L.shatter, L.shatterN = parts, 0
}

// updateShatter breaks the armour on a quickening beat over the death's
// first 1.8 s: the core first, then ring after ring outward.
func (L *Level) updateShatter(deathT float64) {
	if len(L.shatter) == 0 {
		return
	}
	k := clamp01(deathT / 1.8)
	want := int(math.Ceil(float64(len(L.shatter)) * k * k))
	rng := L.fxr()
	for L.shatterN < want && L.shatterN < len(L.shatter) {
		br := L.shatter[L.shatterN]
		L.shatterN++
		if !br.Alive {
			continue
		}
		br.Alive = false
		if L.Grid[br.R][br.C] == br {
			L.Grid[br.R][br.C] = nil
		}
		x, z := br.X(), br.Z()
		c := lighten(br.Col, 0.2)
		for i := 0; i < 3; i++ {
			a := rng.Float64() * 2 * math.Pi
			s := 4 + rng.Float64()*5
			L.Parts = append(L.Parts, Particle{P: V3{x, 0.8 + rng.Float64(), z}, V: V3{math.Cos(a) * s, 5 + rng.Float64()*5, math.Sin(a) * s},
				Life: 0.9, Max: 0.9, Size: 0.35 + rng.Float64()*0.3, Col: c, Kind: pDebris, Grav: 22, Rot: rng.Float64() * 6, Spin: (rng.Float64() - 0.5) * 16})
		}
		L.Parts = append(L.Parts, Particle{P: V3{x, 1, z}, Life: 0.14, Max: 0.14, Size: 2.2, Col: lighten(L.Biome.Trim, 0.5), Kind: pSpark})
		if L.shatterN%4 == 1 {
			L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 1.6, Life: 0.3, Max: 0.3, Col: lighten(L.Biome.Trim, 0.4)})
		}
		L.Shake = math.Max(L.Shake, 0.18)
	}
}

// ---------------------------------------------------------------- seasons

// season is the time of year on the player's clock: 0 winter, 1 spring,
// 2 summer, 3 autumn (northern months; the dressing is light either way).
func season(t time.Time) int {
	switch t.Month() {
	case time.December, time.January, time.February:
		return 0
	case time.March, time.April, time.May:
		return 1
	case time.June, time.July, time.August:
		return 2
	}
	return 3
}

// drawSeason dresses the map: drifting snow, petals, fireflies or leaves.
func (g *Game) drawSeason(s *ebiten.Image) {
	u := g.ui
	n := 46
	if g.lowFX() {
		n = 18
	}
	t := u.time
	se := season(time.Now())
	for i := 0; i < n; i++ {
		fi := float64(i)
		sp := 0.5 + math.Mod(fi*0.618, 1)*0.7 // each its own pace
		x := math.Mod(fi*137.5+math.Sin(t*0.4+fi)*40, 1920)
		y := math.Mod(fi*263.1+t*60*sp, 1140) - 30
		var c color.RGBA
		size := 5.0
		switch se {
		case 0: // snow
			c, size = rgba(235, 245, 255, 255), 3+3*sp
			x += math.Sin(t*1.3+fi) * 14
		case 1: // petals
			c, size = rgba(255, 180, 215, 255), 4+2*sp
			x += math.Sin(t*0.9+fi*1.7) * 30
		case 2: // fireflies: they wander instead of falling
			c, size = rgba(220, 255, 140, 255), 3+2*sp
			y = math.Mod(fi*263.1, 1080) + math.Sin(t*0.7+fi)*40
			x = math.Mod(fi*137.5, 1920) + math.Cos(t*0.5+fi*1.3)*60
			fl := 0.5 + 0.5*math.Sin(t*3+fi*2.1)
			drawSprite(s, sprSoft, u.X(x-size*3), u.Y(y-size*3), size*6*u.S, size*6*u.S, ca(c, 0.25*fl), true)
			drawSprite(s, sprSoft, u.X(x-size), u.Y(y-size), size*2*u.S, size*2*u.S, ca(c, 0.8*fl), true)
			continue
		default: // falling leaves
			c = []color.RGBA{rgba(230, 140, 50, 255), rgba(200, 80, 40, 255), rgba(240, 190, 70, 255)}[i%3]
			size = 5 + 3*sp
			x += math.Sin(t*1.1+fi*2) * 40
		}
		drawSprite(s, sprSoft, u.X(x-size), u.Y(y-size), size*2*u.S, size*1.6*u.S, ca(c, 0.55), false)
	}
}
