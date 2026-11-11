package main

// Sculpted one-off sounds.
//
// The first sound pass gave every power-up the same rising glass triad with
// a little extra on top, and borrowed a handful of clips ("warp",
// "abilready", "coinbig", "heal") for a dozen unrelated moments. This file
// gives each of those moments its own voice, built from the same toolkit in
// sfx.go so it still sits in the game's palette:
//
//   - every pickup has a physical story (a bubble splitting into three, a
//     rubber band stretching, a clock running down, a match striking...)
//   - the "good catch" family shares only a tiny wooden catch-click, tuned
//     per item, so the player hears "caught it" first and "what it is" right
//     after
//   - bad pickups never use the reward bells, so a glance away from the
//     screen still tells you something went wrong
//
// Recipes registered here replace any recipe of the same name in sfx.go.

import (
	"math"
	"math/rand"
)

func init() {
	for name, rc := range sculpted {
		recipes[name] = rc
	}
}

// ---------------------------------------------------------------- extra building blocks

// catchClick is the paddle closing on a capsule: a short, dry wooden knock.
func (b *sb) catchClick(start, f, gain float64) {
	b.modes(start, f, []float64{1, 2.71, 4.9}, []float64{1, 0.32, 0.1}, []float64{55, 85, 130}, gain, 0.0006)
	b.noise(start, 0.012, expEnv(0, 320), fBP, cst(f*3.2), 1.1, gain*0.45)
}

// boing is a spring: a sine whose pitch wobbles and settles.
func (b *sb) boing(start, dur, f, depth, rate, gain float64) {
	ph := 0.0
	b.gen(start, dur, func(t float64) float64 {
		w := 1 + depth*math.Exp(-t*5)*math.Sin(2*math.Pi*rate*t)
		ph += f * w / sampleRate
		return math.Sin(2*math.Pi*ph) * math.Min(1, t*200) * math.Exp(-t*4/dur) * gain
	})
}

// bubble is a single blip whose pitch leaps up as it pops (the classic
// water-drop shape: fast rising sine, very short).
func (b *sb) bubble(start, f, gain float64) {
	ph := 0.0
	b.gen(start, 0.09, func(t float64) float64 {
		ph += f * (1 + 2.2*math.Min(1, t/0.05)) / sampleRate
		return math.Sin(2*math.Pi*ph) * math.Min(1, t*900) * math.Exp(-t*42) * gain
	})
}

// vowel runs a buzzy source through a resonant band-pass that moves like a
// mouth opening (lo -> hi) or closing (hi -> lo).
func (b *sb) vowel(start, dur float64, freq func(float64) float64, lo, hi, gain float64) {
	ph := 0.0
	b.filtered(start, dur, func(t float64) float64 {
		ph += freq(t) / sampleRate
		a := math.Min(1, t/0.02) * math.Min(1, (dur-t)/0.06)
		return (wSaw(ph) + 0.5*wSaw(ph*1.004)) * math.Max(0, a)
	}, fBP, sweep(lo, hi, dur*0.8), 4.5, gain)
}

// tick is a clock tick: a tiny bright wooden click.
func (b *sb) tick(start, f, gain float64) {
	b.modes(start, f, []float64{1, 3.1}, []float64{1, 0.3}, []float64{160, 260}, gain, 0.0003)
}

// clink is a small metal hit (coins, latches, filings).
func (b *sb) clink(start, f, gain float64) {
	b.modes(start, f, []float64{1, 2.76, 5.40, 8.93}, []float64{1, 0.5, 0.25, 0.12}, []float64{26, 38, 55, 75}, gain, 0.0004)
}

// heartThump is one "lub-dub".
func (b *sb) heartThump(start, gain float64) {
	b.sub(start, 92, 48, 0.16, gain)
	b.sub(start+0.17, 84, 44, 0.18, gain*0.72)
}

// ---------------------------------------------------------------- recipes

var sculpted = map[string]recipe{
	// pw.power.key : catching a good power-up
	"pw": {2, func(p []int, v int, r *rand.Rand) []byte {
		id, k := Power(arg(p, 0, 0)), arg(p, 1, 0)
		root := relMajor(k) * 4 // ~ 500-700 Hz, follows the music's key
		nudge := 1 + (float64(v)-0.5)*0.012
		root *= nudge
		dur := 1.5
		return one(dur, r.Int63(), verbMid.with(0.6), func(b *sb) {
			switch id {
			case PMulti:
				// one orb wobbles, then three bubbles pop out of it, each a
				// step higher, and rattle off like marbles
				b.catchClick(0, 820, 0.5)
				b.boing(0, 0.18, root, 0.08, 22, 0.35)
				for i, s := range []int{0, 4, 7} {
					t := 0.07 + float64(i)*0.055
					b.bubble(t, semis(root, s), 0.7)
					b.glass(t+0.03, semis(root*2, s+12), 0.12)
				}
				for i := 0; i < 7; i++ {
					b.tick(0.26+float64(i)*(0.035+0.012*float64(i))+r.Float64()*0.01, 2400+r.Float64()*1800, 0.22-0.02*float64(i))
				}
			case PExpand:
				// a rubber band pulled long: the vowel opens "ooo-aaah" while
				// the pitch climbs a fifth, then the paddle springs into place
				b.catchClick(0, 640, 0.5)
				b.vowel(0.02, 0.36, sweep(root/4, root*1.5/4, 0.32), 380, 2300, 0.5)
				b.boing(0.34, 0.55, root/2, 0.07, 9, 0.45)
				b.sub(0.34, 170, 70, 0.14, 0.4)
				b.modes(0.36, root*1.5, []float64{1, 2.02}, []float64{1, 0.3}, []float64{9, 15}, 0.25, 0.002)
			case PSlow:
				// the clock runs down: a bell chord sags like a tape stopping,
				// and the ticks spread further and further apart
				b.catchClick(0, 900, 0.45)
				for i, s := range []int{0, 4, 7} {
					f := semis(root*2, s)
					b.tone(0.02, 1.2, func(t float64) float64 { return f * math.Pow(2, -4.5*math.Min(1, t/1.0)/12) },
						expEnv(0.004, 2.6), wSin, 0.28-0.05*float64(i))
				}
				t := 0.06
				for i, gap := range []float64{0.09, 0.12, 0.16, 0.21, 0.28, 0.36} {
					f := 3100.0
					if i%2 == 1 {
						f = 2300 // tick ... tock
					}
					b.tick(t, f*(1-0.03*float64(i)), 0.42-0.05*float64(i))
					t += gap
				}
				b.pad(0.1, 1.3, root/2, 0.08)
			case PFire:
				// a match: scratch, the "fwoomp" of the flame catching, then
				// it settles into a crackle
				b.noise(0, 0.07, expEnv(0.004, 30), fBP, sweep(1800, 5200, 0.07), 1.6, 0.8)
				b.noise(0.06, 0.55, expEnv(0.02, 5.5), fLP, sweep(180, 2600, 0.22), 1.1, 1.2)
				b.sub(0.06, 110, 46, 0.35, 0.55)
				b.tone(0.07, 0.4, sweep(root/2, root, 0.2), expEnv(0.03, 6), wTri, 0.18)
				for i := 0; i < 16; i++ {
					b.noise(0.12+r.Float64()*0.9, 0.009, expEnv(0, 260), fHP, cst(2400+r.Float64()*4500), 1, 0.25+0.2*r.Float64())
				}
			case PLaser:
				// a capacitor charging, two arming blips, then the latch clacks
				b.catchClick(0, 760, 0.4)
				b.gen(0.02, 0.46, func(t float64) float64 {
					f := 520 * math.Pow(8, t/0.46)
					return math.Sin(2*math.Pi*f*t+0.6*math.Sin(2*math.Pi*f*0.5*t)) * math.Pow(t/0.46, 1.5) * 0.32
				})
				for i := 0; i < 2; i++ {
					t := 0.5 + float64(i)*0.09
					b.tone(t, 0.07, sweep(semis(root*4, 7-i*3), semis(root*2, 7-i*3), 0.06), expEnv(0.001, 30), wSq, 0.09)
				}
				b.clink(0.7, 1350, 0.35)
				b.noise(0.7, 0.02, expEnv(0, 240), fHP, cst(3500), 0.8, 0.3)
			case PBarrier:
				// stone grinds up out of the floor, crystal grows on it, and it
				// locks into place with a solid "chunk" that keeps ringing
				b.noise(0, 0.38, func(t float64) float64 { return math.Min(1, t*6) }, fLP, sweep(90, 620, 0.38), 1.3, 0.8)
				for i, ratio := range []float64{1, 1.5, 2} {
					f := root * ratio / 2
					b.gen(0.05, 1.4, func(t float64) float64 {
						swell := math.Min(1, t/0.33)
						beat := 1 + 0.25*math.Sin(2*math.Pi*(5+float64(i))*t)
						return math.Sin(2*math.Pi*f*t) * swell * swell * beat * math.Exp(-math.Max(0, t-0.33)*3) * 0.2
					})
				}
				b.sub(0.38, 165, 58, 0.24, 0.8)
				b.modes(0.38, 340, []float64{1, 1.71, 2.6}, []float64{1, 0.4, 0.2}, []float64{30, 45, 60}, 0.45, 0.0005)
				b.glass(0.39, root*2, 0.22)
			case PGlue:
				// sticky: two fat bubbles, a resonant squelch and the "pop" of
				// something coming unstuck
				b.gen(0, 0.12, func(t float64) float64 {
					return math.Sin(2*math.Pi*(360*math.Exp(-t*14)+120)*t) * math.Exp(-t*22) * 0.8
				})
				b.gen(0.1, 0.12, func(t float64) float64 {
					return math.Sin(2*math.Pi*(300*math.Exp(-t*14)+100)*t) * math.Exp(-t*22) * 0.7
				})
				b.filtered(0.04, 0.42, func(t float64) float64 { return wSaw(104*t) * math.Exp(-t*4) }, fLP,
					func(t float64) float64 { return 260 + 1500*math.Exp(-t*7)*(0.6+0.4*math.Sin(2*math.Pi*11*t)) }, 7, 0.45)
				b.noise(0.18, 0.28, expEnv(0.04, 8), fBP, func(t float64) float64 { return 420 + 260*math.Sin(2*math.Pi*13*t) }, 3, 0.35)
				b.bubble(0.48, root, 0.55)
			case PMagnet:
				// the field switches on with a hum, filings skitter in faster
				// and faster, and the last one lands with a bright ding
				b.filtered(0, 0.75, func(t float64) float64 {
					return wSq(98*t+0.4*math.Sin(2*math.Pi*(4+14*t)*t)) * math.Min(1, t*5) * math.Exp(-t*2.2)
				}, fLP, cst(900), 1.4, 0.32)
				t := 0.08
				gap := 0.11
				for i := 0; i < 8; i++ {
					b.clink(t, 1900+float64(i)*170+r.Float64()*90, 0.16+0.02*float64(i))
					t += gap
					gap *= 0.74
				}
				b.glass(t+0.02, root*4, 0.42)
				b.sparkle(t, 0.3, 6, 3500, 8000, 0.15)
			default:
				b.catchClick(0, 800, 0.5)
				b.glass(0.02, root*2, 0.4)
			}
		})
	}},
	// pwbad.power : catching a bad capsule. No bells; it should feel like a
	// small "oops", not a punishment fanfare.
	"pwbad": {2, func(p []int, v int, r *rand.Rand) []byte {
		id := Power(arg(p, 0, 0))
		return one(1.2, r.Int63(), verbSmall.with(0.52), func(b *sb) {
			b.catchClick(0, 480, 0.45)
			switch id {
			case PShrink:
				// a balloon letting go: a squeaky vowel that closes as it falls
				b.vowel(0.02, 0.55, func(t float64) float64 {
					return 620 * math.Pow(0.42, t/0.55) * (1 + 0.025*math.Sin(2*math.Pi*17*t))
				}, 2600, 450, 0.55)
				b.boing(0.5, 0.3, 140, 0.12, 7, 0.4)
			case PHaste:
				// a nervous little alarm: ticks crowd together and climb
				t := 0.03
				gap := 0.12
				for i := 0; i < 10; i++ {
					b.tick(t, 1800*math.Pow(1.06, float64(i)), 0.32)
					t += gap
					gap *= 0.84
				}
				b.gen(0.05, 0.6, func(t float64) float64 {
					return wTri(880*t) * math.Max(0, math.Sin(2*math.Pi*14*t)) * math.Min(1, t*6) * math.Exp(-t*3) * 0.14
				})
				b.whoosh(0.35, 0.3, 900, 4200, 0.35)
			default:
				b.filtered(0.02, 0.5, func(t float64) float64 { return wSaw(200 * math.Pow(0.5, t/0.5) * t) }, fLP, cst(1200), 2, 0.35)
			}
		})
	}},
	// pick.rarity.key : taking a card. Each rarity has its own instrument on
	// top of the escalating weight, so you can hear what you got without
	// looking: wood for common, a kalimba for uncommon, glass bells for rare,
	// a choir swell for epic, brass for legendary and a dark gong for mythic.
	"pick": {1, func(p []int, v int, r *rand.Rand) []byte {
		rar, k := arg(p, 0, 0), arg(p, 1, 0)
		root := relMajor(k) * 4
		dur := 1.2 + 0.5*float64(rar)
		return one(dur+1.5, r.Int63(), verbBig.with(0.5+0.08*float64(rar)), func(b *sb) {
			t0 := 0.0
			if rar >= 4 {
				b.riser(0, 0.4, 0.4)
				t0 = 0.4
			}
			notes := 2 + min(rar, 3)
			for i := 0; i < notes; i++ {
				t := t0 + float64(i)*0.06
				f := ladder(3, i*2, root)
				switch rar {
				case 0: // marimba: warm wooden bars
					b.modes(t, f/2, []float64{1, 3.99, 10.6}, []float64{1, 0.35, 0.08}, []float64{7, 18, 40}, 0.55, 0.001)
				case 1: // kalimba: plucked tines
					b.pluck(t, f, 0.996, 0.55, 0.6)
					b.modes(t, f*2, []float64{1, 2.7}, []float64{1, 0.2}, []float64{12, 30}, 0.15, 0.001)
				default:
					b.glass(t, f, 0.5)
				}
			}
			if rar >= 1 {
				b.sparkle(t0, 0.3+0.2*float64(rar), 6*rar, 3000, 9000, 0.3)
			}
			if rar >= 2 {
				b.sub(t0, 160, 50, 0.3, 0.5)
			}
			if rar == 3 { // epic: a choir-like swell under the bells
				for _, st := range []int{0, 4, 7, 11} {
					b.vowel(t0+0.05, dur, cst(semis(root/4, st)), 650, 900, 0.12)
				}
			}
			if rar >= 3 {
				b.impact(t0, 0.7)
				chord := []int{0, 4, 7, 11}
				if rar == 5 {
					chord = []int{0, 3, 7, 10}
				}
				for _, st := range chord {
					b.pad(t0, dur, semis(root/4, st), 0.14)
				}
			}
			if rar == 4 {
				for _, st := range []int{0, 7, 12, 16} {
					b.brassNote(t0+0.02, dur*0.8, semis(root/4, st), 0.18)
				}
			}
			if rar >= 5 { // mythic: a dark gong and low brass instead of the fanfare
				b.fm(t0, dur+1, root/8, 1.4, 3.5, 0.7, 0.002, 1.0, 0.4)
				for _, st := range []int{0, 7, 10} {
					b.brassNote(t0+0.05, dur*0.9, semis(root/8, st), 0.16)
				}
			}
		})
	}},
	// omenmet.key : a chamber challenge is met. A wax seal stamped onto a
	// scroll, then a bright little two-bell flourish.
	"omenmet": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) * 4
		return one(1.8, r.Int63(), verbMid.with(0.52), func(b *sb) {
			b.sub(0, 150, 70, 0.12, 0.6)
			b.noise(0, 0.05, expEnv(0.002, 60), fLP, cst(1400), 1, 0.6) // the stamp
			b.noise(0.01, 0.12, expEnv(0.01, 18), fBP, sweep(900, 2600, 0.1), 1.2, 0.25)
			b.glass(0.12, semis(root, 7), 0.4)
			b.glass(0.2, semis(root, 12), 0.45)
			b.sparkle(0.2, 0.4, 8, 3500, 8500, 0.18)
		})
	}},
	// omenfail : a challenge slips away. A soft, good-natured "aww".
	"omenfail": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.0, r.Int63(), verbSmall.with(0.32), func(b *sb) {
			b.vowel(0, 0.5, sweep(330, 247, 0.45), 900, 600, 0.45)
			b.modes(0.05, 392, []float64{1, 2.01}, []float64{1, 0.2}, []float64{9, 16}, 0.2, 0.002)
		})
	}},
	// life.key : catching a falling heart. A heartbeat that blooms into a
	// warm major chord.
	"life": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) * 2
		return one(2.4, r.Int63(), verbBig.with(0.55), func(b *sb) {
			b.catchClick(0, 700, 0.35)
			b.heartThump(0.02, 0.9)
			for i, s := range []int{0, 4, 7, 12} {
				b.pluck(0.36+float64(i)*0.045, semis(root, s), 0.998, 0.45, 0.55)
			}
			for _, s := range []int{0, 4, 7} {
				b.pad(0.34, 1.7, semis(root/2, s), 0.1)
			}
			b.glass(0.55, semis(root*4, 4), 0.18)
		})
	}},
	// heartfall.key : a heart has been knocked loose and is falling.
	// A soft descending harp run, then one heartbeat to say "catch me".
	"heartfall": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 4
		return one(1.5, r.Int63(), verbMid.with(0.42), func(b *sb) {
			for i, s := range []int{12, 7, 4, 0} {
				b.pluck(float64(i)*0.06, semis(root, s), 0.997, 0.5, 0.5)
			}
			b.heartThump(0.3, 0.7)
		})
	}},
	// cachefall : a rune cache is falling. Low, curious and a little
	// mysterious: a hum under a slowly turning chime.
	"cachefall": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.6, r.Int63(), verbBig.with(0.42), func(b *sb) {
			b.fm(0, 1.2, 98, 2, 1.5, 1.5, 0.08, 2.4, 0.3)
			for i, f := range []float64{1174.7, 1568, 1396.9, 1760} {
				b.modes(float64(i)*0.09, f, []float64{1, 2.4, 5.1}, []float64{1, 0.3, 0.1}, []float64{6, 11, 18}, 0.28, 0.002)
			}
		})
	}},
	// cache.key : the paddle catches a rune cache: a latch clacks open and a
	// chord of possibilities spills out.
	"cache": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) * 2
		return one(2.2, r.Int63(), verbBig.with(0.55), func(b *sb) {
			b.catchClick(0, 560, 0.45)
			b.clink(0.04, 980, 0.4)
			b.clink(0.09, 1310, 0.3)
			b.noise(0.1, 0.28, expEnv(0.02, 9), fBP, sweep(600, 2600, 0.25), 1.3, 0.35) // lid creak
			for i, s := range []int{0, 7, 12, 16, 19} {
				b.glass(0.22+float64(i)*0.05, semis(root*2, s), 0.3)
			}
			b.pad(0.22, 1.4, root/2, 0.12)
			b.pad(0.22, 1.4, semis(root/2, 7), 0.09)
		})
	}},
	// wave.key : the next wave rises. A two-note horn call and a drum, so it
	// reads as "here they come", not as another magic effect.
	"wave": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := keyRoots[arg(p, 0, 0)]
		return one(2.4, r.Int63(), verbBig.with(0.7), func(b *sb) {
			b.sub(0, 90, 40, 0.5, 0.9)
			b.noise(0, 0.3, expEnv(0.002, 9), fLP, cst(700), 0.8, 0.5)
			b.brassNote(0.05, 0.28, semis(root, 7), 0.32)
			b.brassNote(0.36, 0.95, semis(root, 12), 0.36)
			b.brassNote(0.36, 0.95, semis(root, 7), 0.2)
			b.sub(0.36, 80, 36, 0.6, 0.7)
			b.whoosh(0.3, 0.5, 300, 2200, 0.3)
		})
	}},
	// attune : the Arcanist's element changes. A soft glass turn.
	"attune": {3, func(p []int, v int, r *rand.Rand) []byte {
		f := []float64{1046.5, 1174.7, 1318.5}[v%3]
		return one(0.9, r.Int63(), verbMid.with(0.32), func(b *sb) {
			b.glass(0, f, 0.4)
			b.glass(0.07, f*1.5, 0.3)
			b.whoosh(0, 0.2, 2000, 6000, 0.15)
		})
	}},
	// overclock : the Engineer's paddle has stood still long enough. A motor
	// spinning up to a satisfied click.
	"overclock": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbSmall.with(0.4), func(b *sb) {
			b.filtered(0, 0.42, func(t float64) float64 {
				return wSaw(60*math.Pow(6, t/0.42)*t) * math.Min(1, t*10)
			}, fLP, sweep(500, 3500, 0.42), 1.6, 0.35)
			b.clink(0.42, 1600, 0.4)
			b.tick(0.46, 3000, 0.3)
		})
	}},
	// staticfull : the Stormcaller's static charge is full. A crackling
	// build that snaps.
	"staticfull": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbMid.with(0.42), func(b *sb) {
			for i := 0; i < 22; i++ {
				t := 0.4 * math.Sqrt(r.Float64())
				b.noise(t, 0.006, expEnv(0, 400), fHP, cst(3000+r.Float64()*5000), 1, 0.2+0.4*t)
			}
			b.noise(0.4, 0.05, expEnv(0, 70), fHP, cst(1800), 0.7, 0.8)
			b.fm(0.4, 0.4, 2400, 1.41, 3, 8, 0, 9, 0.22)
		})
	}},
	// jackpot : the Gambler's ability pays out. Reels spin, then the bell.
	"jackpot": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.8, r.Int63(), verbMid.with(0.55), func(b *sb) {
			t := 0.0
			gap := 0.04
			for i := 0; i < 12; i++ {
				b.tick(t, 1500+float64(i%3)*260, 0.3)
				t += gap
				gap *= 1.09
			}
			b.fm(t, 1.2, 1568, 3.5, 2.5, 4, 0, 2.6, 0.45)
			for i := 0; i < 8; i++ {
				b.clink(t+0.04+float64(i)*0.05+r.Float64()*0.02, 2200+r.Float64()*1400, 0.2)
			}
		})
	}},
	// nova : the Arcanist's Prismatic Nova. Light splitting into colours:
	// one note fans out into a shimmering spread of detuned partials.
	"nova": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(2.0, r.Int63(), verbBig.with(0.75), func(b *sb) {
			b.impact(0, 0.5)
			for i, s := range []int{0, 2, 4, 7, 9, 12, 14} {
				f := semis(523.25, s)
				b.tone(0, 1.5, func(t float64) float64 { return 523.25 + (f-523.25)*math.Min(1, t/0.25) },
					expEnv(0.005, 2.4), wTri, 0.13)
				b.glass(0.2+float64(i)*0.03, f*2, 0.12)
			}
			b.whoosh(0, 0.5, 1200, 8000, 0.35)
		})
	}},
	// horizon : a black hole opens. Everything bends towards it.
	"horizon": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(2.6, r.Int63(), verbBig.with(0.85), func(b *sb) {
			b.noise(0, 1.1, func(t float64) float64 { return math.Pow(math.Min(1, t/1.0), 2) }, fBP, sweep(6000, 180, 1.0), 2.5, 0.8)
			b.tone(0, 1.1, sweep(880, 55, 1.0), func(t float64) float64 { return math.Min(1, t*3) }, wSaw, 0.08)
			b.sub(1.0, 70, 26, 1.4, 1.3)
			b.noise(1.0, 0.06, expEnv(0, 70), fHP, cst(2000), 0.7, 0.6)
			b.pad(1.0, 1.5, 55, 0.18)
		})
	}},
	// reform : the paddle pulls itself back together after a duel drop.
	"reform": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbMid.with(0.4), func(b *sb) {
			for i := 0; i < 6; i++ {
				b.glass(0.03*float64(5-i), 1200+float64(i)*260, 0.12)
			}
			b.whoosh(0, 0.2, 5000, 900, 0.4)
			b.catchClick(0.2, 620, 0.5)
		})
	}},
	// purify : a curse lifts. A dark chord resolves upward into light.
	"purify": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) * 2
		return one(2.6, r.Int63(), verbBig.with(0.5), func(b *sb) {
			for _, s := range []int{0, 3, 6} {
				b.pad(0, 0.7, semis(root/2, s), 0.1)
			}
			for _, s := range []int{0, 4, 7, 12} {
				b.pad(0.55, 1.8, semis(root/2, s), 0.1)
			}
			b.whoosh(0.4, 0.5, 400, 5000, 0.3)
			b.sparkle(0.6, 1.0, 16, 3000, 9000, 0.2)
		})
	}},
	// lucky : a golden critter pops. Coins and a little "ding-ding!".
	"lucky": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.3, r.Int63(), verbMid.with(0.52), func(b *sb) {
			b.bubble(0, 700, 0.5)
			for i := 0; i < 7; i++ {
				b.clink(0.04+float64(i)*0.035+r.Float64()*0.015, 2600+r.Float64()*1500, 0.22)
			}
			b.glass(0.12, 2093, 0.32)
			b.glass(0.22, 2637, 0.32)
		})
	}},
	// shoo : a critter is chased off. A startled squeak and scampering feet.
	"shoo": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.8, r.Int63(), verbSmall.with(0.42), func(b *sb) {
			f := 1400 + float64(v)*180
			b.vowel(0, 0.12, sweep(f, f*1.6, 0.1), 1800, 3800, 0.4)
			for i := 0; i < 6; i++ {
				b.noise(0.12+float64(i)*0.05, 0.02, expEnv(0.001, 120), fBP, cst(900+r.Float64()*500), 1.4, 0.35)
			}
			b.clink(0.08, 2600, 0.18)
		})
	}},
	// thiefstop : a thief is caught and drops what it stole.
	"thiefstop": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.noise(0, 0.05, expEnv(0, 70), fBP, cst(1200), 1, 0.6)
			b.sub(0, 140, 60, 0.15, 0.6)
			for i := 0; i < 9; i++ {
				b.clink(0.08+float64(i)*0.045+r.Float64()*0.02, 2100+r.Float64()*1800, 0.2)
			}
		})
	}},
	// bomber : a bomber flies in. A fuse fizzing over a buzzing propeller.
	"bomber": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbSmall.with(0.36), func(b *sb) {
			b.filtered(0, 1.0, func(t float64) float64 {
				return wSq(38*t) * (0.5 + 0.5*math.Sin(2*math.Pi*24*t)) * math.Min(1, t*4) * math.Min(1, (1-t)*4)
			}, fLP, cst(700), 1.2, 0.4)
			b.noise(0, 1.0, func(t float64) float64 {
				return (0.6 + 0.4*math.Sin(2*math.Pi*31*t)) * math.Min(1, t*8) * math.Min(1, (1-t)*5)
			}, fHP, cst(4800), 0.8, 0.25)
		})
	}},
	// shift : a brick teleports to a new cell. A quick pinched blip.
	"shift": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.6, r.Int63(), verbMid.with(0.32), func(b *sb) {
			f := 900 + float64(v)*120
			b.tone(0, 0.08, sweep(f, f*3, 0.08), expEnv(0.002, 30), wTri, 0.4)
			b.tone(0.08, 0.1, sweep(f*3, f*1.5, 0.1), expEnv(0.002, 25), wTri, 0.3)
		})
	}},
	// elite : a core brick awakens. A low warning pulse with a metallic edge.
	"elite": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbBig.with(0.55), func(b *sb) {
			for i := 0; i < 2; i++ {
				t := float64(i) * 0.22
				b.sub(t, 110, 55, 0.25, 0.7)
				b.fm(t, 0.4, 220, 1.41, 3, 6, 0.003, 7, 0.22)
			}
			b.modes(0.44, 660, []float64{1, 2.76, 5.4}, []float64{1, 0.4, 0.2}, []float64{4, 8, 13}, 0.3, 0.001)
		})
	}},
	// duelgo : the duel countdown ends. A bright starting bell.
	"duelgo": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.fm(0, 1.0, 1318.5, 2, 2.2, 6, 0, 3.4, 0.45)
			b.glass(0.01, 2637, 0.25)
			b.noise(0, 0.04, expEnv(0, 90), fHP, cst(3000), 0.7, 0.4)
		})
	}},
}
