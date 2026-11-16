package main

import (
	"image/color"
	"math"
	"math/rand"
)

// Board polish from ROADMAP-50 section I. Everything here is cosmetic and
// draws its randomness from L.fx, never from L.rng, so seeded chambers and
// the simulations play out exactly as before.
//
//   - #83 impact marks: blasts, crits and Guardian hits leave a scorch,
//     frost or spark mark on the floor that fades over a few seconds
//   - #85 element trails: a burning orb sheds embers, a frozen-heart orb
//     sparkles, a storm orb crackles, a void orb trails dark smoke
//   - #86 paddle charge glow: the paddle glows as the origin's passive
//     builds up (Overclock, Rage, Parry, Attunement) and pulses when ready
//   - #89 victory confetti in the act's colours when a Guardian falls
//   - #88 weather: some chambers get rain, petals or snow
//   - #82 orb-lit bricks: bricks near an orb catch a little of its light

// fxr is the cosmetic random source.
func (L *Level) fxr() *rand.Rand {
	if L.fx == nil {
		L.fx = rand.New(rand.NewSource(int64(L.propSeed) ^ 0x5f3759df))
	}
	return L.fx
}

// ---------------------------------------------------------------- #83 marks

type floorMark struct {
	X, Z, R   float64
	Life, Max float64
	Col       color.RGBA
	Glow      bool
}

const maxMarks = 28

// addMark leaves a mark on the floor at (x, z).
func (L *Level) addMark(x, z, r float64, col color.RGBA, glow bool) {
	if L.g.settings.Particles == 0 && len(L.Marks) > maxMarks/2 {
		return
	}
	life := 3.5 + L.fxr().Float64()
	if len(L.Marks) >= maxMarks {
		copy(L.Marks, L.Marks[1:])
		L.Marks = L.Marks[:maxMarks-1]
	}
	L.Marks = append(L.Marks, floorMark{x, z, r, life, life, col, glow})
}

func (L *Level) updateMarks(dt float64) {
	keep := L.Marks[:0]
	for _, m := range L.Marks {
		m.Life -= dt
		if m.Life > 0 {
			keep = append(keep, m)
		}
	}
	L.Marks = keep
}

func (L *Level) drawMarks(B *Batch) {
	for _, m := range L.Marks {
		f := m.Life / m.Max
		a := math.Min(1, f*1.5)
		// a dark scorch with the element's colour glowing at its heart
		B.Decal(m.X, m.Z, 0.012, m.R*2.2, m.R*2.2, sprSoft, color.RGBA{0, 0, 0, uint8(120 * a)}, false)
		if m.Glow {
			B.Decal(m.X, m.Z, 0.014, m.R*1.3, m.R*1.3, sprSoft, alpha(m.Col, 0.35*a*a), true)
		}
	}
}

// ---------------------------------------------------------------- #85 trails

// orbElement is the element an orb shows in its trail (0 none, 1 fire,
// 2 frost, 3 storm, 4 void).
func (L *Level) orbElement(b *Ball) int {
	r := L.g.run
	switch {
	case L.Timers[PFire] > 0 || r.Owned("inferno"):
		return 1
	case b.Storm > 0 || r.Owned("staticfield"):
		return 3
	case r.Owned("frozenheart"):
		return 2
	case r.Owned("voidbounce"):
		return 4
	}
	return 0
}

// updateTrails sheds element particles behind the first few orbs.
func (L *Level) updateTrails(dt float64) {
	if L.g.settings.Particles == 0 {
		return
	}
	rng := L.fxr()
	for i, b := range L.Balls {
		if i >= 3 || b.Stuck || b.Phantom > 0 {
			continue
		}
		el := L.orbElement(b)
		if el == 0 {
			continue
		}
		rate := 26.0
		if i > 0 {
			rate = 10 // the extra orbs shed less (multiball clutter, #9)
		}
		if rng.Float64() < dt*rate {
			jx, jz := (rng.Float64()-0.5)*b.R, (rng.Float64()-0.5)*b.R
			p := Particle{P: V3{b.X + jx, b.R, b.Z + jz}}
			switch el {
			case 1: // embers rise off the orb
				p.V = V3{(rng.Float64() - 0.5) * 1.2, 2 + rng.Float64()*1.5, (rng.Float64() - 0.5) * 1.2}
				p.Life, p.Max, p.Size, p.Kind, p.Grav = 0.5, 0.5, 0.22, pSpark, -1.5
				p.Col = color.RGBA{255, uint8(120 + rng.Intn(100)), 40, 255}
			case 2: // frost glitter that hangs in the air
				p.V = V3{(rng.Float64() - 0.5) * 0.4, 0.3, (rng.Float64() - 0.5) * 0.4}
				p.Life, p.Max, p.Size, p.Kind = 0.8, 0.8, 0.18, pAmbient
				p.Col = color.RGBA{210, 245, 255, 255}
			case 3: // a crackle of static
				p.V = V3{(rng.Float64() - 0.5) * 6, rng.Float64() * 2, (rng.Float64() - 0.5) * 6}
				p.Life, p.Max, p.Size, p.Kind, p.Grav = 0.18, 0.18, 0.16, pSpark, 0
				p.Col = color.RGBA{150, 210, 255, 255}
			case 4: // dark smoke
				p.V = V3{0, 0.5, 0}
				p.Life, p.Max, p.Size, p.Kind = 0.7, 0.7, 0.55, pSmoke
				p.Col = color.RGBA{70, 30, 110, 255}
			}
			L.Parts = append(L.Parts, p)
		}
	}
}

// ---------------------------------------------------------------- #86 paddle glow

// passiveCharge is how full the origin's passive is (0..1) and whether it
// is ready, with its colour; ok is false for origins with no meter.
func (L *Level) passiveCharge() (frac float64, ready bool, col color.RGBA, ok bool) {
	st := &L.g.run.Stats
	c := &L.cls
	switch {
	case st.ClassOverclock:
		return clamp01(c.stillT / st.overclockTime()), c.overclock, rgba(255, 160, 64, 255), true
	case st.ClassRage:
		return float64(c.rage) / float64(max(1, st.rageMax())), c.rage == st.rageMax(), rgba(255, 70, 50, 255), true
	case st.ClassParry:
		return 1 - clamp01(c.parryCD/st.parryCooldown()), c.parryCD <= 0, rgba(255, 215, 130, 255), true
	case st.ClassAttune:
		return 1 - c.attuneT/st.attunePeriod(), false, attuneCols[c.attuneEl], true
	}
	return 0, false, color.RGBA{}, false
}

func (L *Level) drawPaddleCharge(B *Batch, w float64) {
	frac, ready, col, ok := L.passiveCharge()
	if !ok || frac <= 0.02 {
		return
	}
	a := 0.18 + 0.32*frac
	if ready {
		a = 0.45 + 0.2*math.Sin(L.T*7)
	}
	// a pool of light under the paddle that widens as it fills
	B.Decal(L.PaddleX, PaddleZ, 0.02, w*(0.4+0.8*frac), 2.2, sprSoft, alpha(col, a), true)
	if ready {
		B.Billboard(V3{L.PaddleX, 0.9, PaddleZ}, 1.2+0.2*math.Sin(L.T*7), sprSoft, alpha(lighten(col, 0.3), 0.35), true)
	}
}

// ---------------------------------------------------------------- #89 confetti

// confetti throws a burst of act-coloured confetti over the board.
func (L *Level) confetti() {
	if L.Biome == nil {
		return
	}
	rng := L.fxr()
	n := 220
	if L.g.settings.Particles == 0 {
		n = 70
	}
	for i := 0; i < n; i++ {
		col := L.Biome.Hues[rng.Intn(len(L.Biome.Hues))]
		if i%5 == 0 {
			col = color.RGBA{255, 225, 120, 255} // gold flecks
		}
		x := (rng.Float64() - 0.5) * 2 * HalfW
		L.Parts = append(L.Parts, Particle{
			P:    V3{x, 9 + rng.Float64()*5, 10 + rng.Float64()*26},
			V:    V3{(rng.Float64() - 0.5) * 4, -1 - rng.Float64()*2, (rng.Float64() - 0.5) * 3},
			Life: 2.6 + rng.Float64()*1.4, Max: 4, Size: 0.16 + rng.Float64()*0.12,
			Col: lighten(col, 0.15), Kind: pDebris, Grav: 2.2,
			Rot: rng.Float64() * 6.28, Spin: (rng.Float64() - 0.5) * 16,
		})
	}
}

// ---------------------------------------------------------------- #88 weather

const (
	weatherNone = iota
	weatherRain
	weatherPetals
	weatherSnow
)

// pickWeather gives about one chamber in six a weather that suits its world.
func (L *Level) pickWeather() {
	L.weatherSet = true
	if L.Boss != nil || L.Biome == nil || L.fxr().Float64() > 0.17 {
		return
	}
	switch L.Biome.ID {
	case WMeadow, WRuins, WDream:
		L.weather = []int{weatherRain, weatherPetals}[L.fxr().Intn(2)]
	case WCaverns, WClouds:
		L.weather = weatherSnow
	case WNeon:
		L.weather = weatherPetals
	default:
		L.weather = weatherRain
	}
}

func (L *Level) updateWeather(dt float64) {
	if !L.weatherSet {
		L.pickWeather()
	}
	if L.weather == weatherNone || L.g.settings.Particles == 0 {
		return
	}
	rng := L.fxr()
	L.weatherT += dt
	step := map[int]float64{weatherRain: 0.012, weatherPetals: 0.09, weatherSnow: 0.035}[L.weather]
	for L.weatherT > step {
		L.weatherT -= step
		x, z := (rng.Float64()-0.5)*40, rng.Float64()*46
		var p Particle
		switch L.weather {
		case weatherRain:
			p = Particle{P: V3{x, 10, z}, V: V3{-1, -22, 0}, Life: 0.5, Max: 0.5, Size: 0.08, Col: color.RGBA{180, 210, 255, 255}, Kind: pSpark}
		case weatherPetals:
			p = Particle{P: V3{x, 8, z}, V: V3{1.2 + rng.Float64(), -0.8, (rng.Float64() - 0.5) * 0.6}, Life: 7, Max: 7, Size: 0.3,
				Col: color.RGBA{255, uint8(170 + rng.Intn(50)), 200, 255}, Kind: pDebris, Grav: 0, Rot: rng.Float64() * 6, Spin: (rng.Float64() - 0.5) * 4}
		case weatherSnow:
			p = Particle{P: V3{x, 9, z}, V: V3{(rng.Float64() - 0.5) * 0.6, -1.1 - rng.Float64()*0.4, 0}, Life: 8, Max: 8, Size: 0.24, Col: color.RGBA{245, 250, 255, 255}, Kind: pAmbient}
		}
		L.Parts = append(L.Parts, p)
	}
}

// ---------------------------------------------------------------- #82 orb light

// orbLight is how much orb light a brick at (x, z) catches (0..1) and its
// colour: the nearest orb within four units.
func (L *Level) orbLight(x, z float64) (float64, color.RGBA) {
	best, bc := 0.0, color.RGBA{}
	for i, b := range L.Balls {
		if i >= 4 {
			break
		}
		dx, dz := b.X-x, b.Z-z
		d2 := dx*dx + dz*dz
		if d2 > 16 {
			continue
		}
		k := 1 - math.Sqrt(d2)/4
		if k > best {
			best, bc = k, L.g.orbGlow(L, b)
		}
	}
	return best * best, bc
}
