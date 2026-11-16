package main

// Sounds for the card and fusion animations (pickfx.go, fusionfx.go).
//
//   - cardfly: the chosen card whooshes up into your build and lands with a
//     chime that grows richer with the card's rarity
//   - statup: one rising marimba step per stat line as a number counts up,
//     climbing the major chord so a long list sounds like a little fanfare
//   - fuse: two tones spiral toward each other (they beat, then lock into
//     unison), a rush of air, an anvil strike at the moment of collision,
//     then a warm chord blooms with sparkles; timed to the forge animation
//   - letter: an envelope slid across a table and a sheet unfolding (two
//     papery rustles and a soft music-box note)
//   - scribble: a quick pen stroke as each line of a letter appears
//   - kiss: Mirabel's lipstick pressed on: a soft "mm-wah" and a little pop

import (
	"math"
	"math/rand"
)

func init() {
	for name, rc := range fxSounds {
		recipes[name] = rc
	}
}

var fxSounds = map[string]recipe{
	// cardfly.rarity.key
	"cardfly": {2, func(p []int, v int, r *rand.Rand) []byte {
		rar, k := arg(p, 0, 0), arg(p, 1, 0)
		root := relMajor(k) * 4 * (1 + (float64(v)-0.5)*0.01)
		return one(1.3, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.whoosh(0, 0.36, 500, 5200, 0.55)
			b.tone(0, 0.36, sweep(root/4, root/2, 0.36), func(t float64) float64 { return math.Pow(t/0.36, 2) * 0.6 }, wTri, 0.18)
			// the landing chime: a major triad, one more note per rarity
			notes := []int{0, 4, 7, 12, 16, 19}
			for i := 0; i <= min(2+rar/2, len(notes)-1); i++ {
				b.glass(0.36+float64(i)*0.025, semis(root, notes[i]), 0.34-0.03*float64(i))
			}
			if rar >= Epic {
				b.sparkle(0.38, 0.6, 10+4*rar, 3000, 9000, 0.14)
			}
		})
	}},
	// statup.step.key
	"statup": {2, func(p []int, v int, r *rand.Rand) []byte {
		i, k := arg(p, 0, 0), arg(p, 1, 0)
		steps := []int{0, 4, 7, 12, 16, 19}
		f := semis(relMajor(k)*2, steps[i%len(steps)])
		return one(0.6, r.Int63(), verbSmall.with(0.42), func(b *sb) {
			// marimba: a warm wooden bar
			b.modes(0, f, []float64{1, 3.99, 10.6}, []float64{1, 0.28, 0.06}, []float64{9, 22, 40}, 0.6, 0.0008)
			b.tick(0, f*4, 0.12)
		})
	}},
	// letter
	"letter": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbSmall.with(0.45), func(b *sb) {
			// paper slides: band-passed noise with a slow, grainy envelope
			slide := func(start, dur, f0, f1, gain float64) {
				b.noise(start, dur, func(t float64) float64 {
					x := t / dur
					return math.Sin(math.Pi*math.Min(1, x)) * (0.7 + 0.3*math.Sin(t*170+float64(v)))
				}, fBP, sweep(f0, f1, dur), 0.9, gain)
			}
			slide(0, 0.32, 1800, 3200, 0.5)
			slide(0.3, 0.22, 3400, 2200, 0.38)
			b.noise(0.52, 0.03, expEnv(0, 120), fHP, cst(2600), 0.8, 0.25) // the flap
			b.glass(0.6, 1568*(1+(float64(v)-0.5)*0.02), 0.16)
		})
	}},
	// scribble: one pen stroke
	"scribble": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.4, r.Int63(), verbDry.with(0.35), func(b *sb) {
			n := 3 + v
			for i := 0; i < n; i++ {
				st := float64(i) * (0.05 + 0.01*float64(v))
				b.noise(st, 0.045, func(t float64) float64 { return math.Sin(math.Pi * math.Min(1, t/0.045)) }, fBP, cst(4200+r.Float64()*1800), 2.2, 0.5)
			}
		})
	}},
	// kiss: a soft "mm-wah" and a pop
	"kiss": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbSmall.with(0.48), func(b *sb) {
			b.vowel(0, 0.16, cst(240), 300, 420, 0.25)                  // "mm"
			b.vowel(0.14, 0.18, sweep(260, 330, 0.18), 420, 1300, 0.32) // "wah"
			b.bubble(0.13, 900, 0.5)                                    // the pop of the lips
			b.glass(0.22, 2093, 0.12)                                   // a sparkle of charm
			b.glass(0.3, 2637, 0.09)
		})
	}},
	// flawless.key: a rubber stamp slams down, then a bright rising run
	"flawless": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) * 2
		return one(1.4, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.sub(0, 120, 55, 0.18, 0.7)
			b.noise(0, 0.06, expEnv(0, 60), fLP, cst(1800), 0.8, 0.6) // the stamp's thud
			for i, st := range []int{0, 4, 7, 12, 16} {
				b.glass(0.12+float64(i)*0.05, semis(root*2, st), 0.26)
			}
			b.sparkle(0.35, 0.6, 14, 4000, 9500, 0.12)
		})
	}},
	// lastlife: a soft double heartbeat under a low swell, for dropping to
	// the last life (quiet: a warning, not an alarm)
	"lastlife": {1, func(p []int, v int, r *rand.Rand) []byte {
		root := relMajor(arg(p, 0, 0)) / 2
		return one(1.6, r.Int63(), verbMid.with(0.35), func(b *sb) {
			b.sub(0, 70, 44, 0.16, 0.6)
			b.sub(0.24, 64, 42, 0.18, 0.45)
			b.tone(0, 1.2, cst(root), func(t float64) float64 { return math.Sin(math.Pi*math.Min(1, t/1.2)) * 0.5 }, wTri, 0.08)
		})
	}},
	// fuse.key
	"fuse": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 2
		const hit = 1.15
		return one(3.6, r.Int63(), verbBig.with(0.62), func(b *sb) {
			// two voices spiral in: one from an octave below, one from an
			// octave above, gliding to the same note; the beating slows to
			// nothing just as they meet
			env := func(t float64) float64 { return math.Pow(math.Min(1, t/hit), 1.6) }
			lo := func(t float64) float64 { return root * math.Pow(2, -1+math.Min(1, t/hit)) }
			hi := func(t float64) float64 { return root * math.Pow(2, 1-math.Min(1, t/hit)) }
			b.tone(0, hit, lo, env, wTri, 0.22)
			b.tone(0, hit, hi, env, wSin, 0.16)
			b.whoosh(0.45, hit-0.42, 300, 4500, 0.5)
			// the strike: a deep thump and an anvil ring
			b.impact(hit, 0.95)
			b.modes(hit, root, []float64{1, 2.76, 5.40, 8.93}, []float64{1, 0.55, 0.3, 0.15}, []float64{2.6, 4, 7, 11}, 0.55, 0.0005)
			b.noise(hit, 0.08, expEnv(0, 50), fHP, cst(4200), 0.8, 0.35)
			// the bloom: a major chord swelling under sparkles
			for i, s := range []int{0, 4, 7, 12} {
				b.pad(hit+0.05, 2.2, semis(root/2, s), 0.07)
				b.glass(hit+0.08+float64(i)*0.06, semis(root*2, s), 0.22)
			}
			b.brassNote(hit+0.1, 1.4, root, 0.12)
			b.brassNote(hit+0.1, 1.4, semis(root, 7), 0.09)
			b.sparkle(hit+0.05, 1.8, 36, 2500, 9500, 0.18)
		})
	}},
}
