package main

// Quiet sounds for the session-9 systems (keepsakes, banish, shop locks,
// familiars, co-op, trials). All short and soft; played through PlayVol so
// they follow the SFX volume and sit under the gameplay mix.
//
//   - ktrank.N: a keepsake rank bought: a small glass chime, one note
//     higher per rank, over the existing "buy" coin sound
//   - kteq / ktuneq: a charm clasped on (latch click + bright clink) / off
//   - banish: a card burned away: a falling breath and a low dark tone
//   - lock / unlock: a small latch shutting / lifting
//   - owlsave: two soft "hoo"s as the owl turns the orb back
//   - coopgo: two plucks a fifth apart when player two launches
//   - knot: a rope pulled taut and a warm bell
//   - trialon / trialoff: an ember catching / a quiet tick

import (
	"math"
	"math/rand"
)

func init() {
	for name, rc := range systemSounds {
		recipes[name] = rc
	}
}

var systemSounds = map[string]recipe{
	"ktrank": {1, func(p []int, v int, r *rand.Rand) []byte {
		n := min(max(arg(p, 0, 1), 1), 6)
		f := semis(1046.5, []int{0, 2, 4, 7, 9, 12}[n-1])
		return one(1.0, r.Int63(), verbMid.with(0.32), func(b *sb) {
			b.glass(0.04, f, 0.5)
			b.glass(0.1, f*1.5, 0.22)
		})
	}},
	"kteq": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.5, r.Int63(), verbSmall.with(0.32), func(b *sb) {
			b.catchClick(0, 900, 0.5)
			b.clink(0.05, 2400, 0.32)
		})
	}},
	"ktuneq": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.45, r.Int63(), verbSmall.with(0.28), func(b *sb) {
			b.clink(0, 1700, 0.28)
			b.catchClick(0.04, 700, 0.4)
		})
	}},
	"banish": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.1, r.Int63(), verbMid.with(0.36), func(b *sb) {
			b.whoosh(0, 0.55, 3800, 380, 0.45)
			b.tone(0.04, 0.6, sweep(196, 147, 0.6), expEnv(0.03, 4), wTri, 0.16)
			b.noise(0.05, 0.35, expEnv(0.02, 9), fBP, cst(900+float64(v)*120), 1.4, 0.12)
		})
	}},
	"lock": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.4, r.Int63(), verbSmall.with(0.3), func(b *sb) {
			b.tick(0, 1500, 0.5)
			b.clink(0.035, 1900, 0.3)
		})
	}},
	"unlock": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.4, r.Int63(), verbSmall.with(0.26), func(b *sb) {
			b.clink(0, 2300, 0.22)
			b.tick(0.04, 1200, 0.4)
		})
	}},
	"owlsave": {2, func(p []int, v int, r *rand.Rand) []byte {
		f := 420.0 + float64(v)*18
		hoo := func(t float64) float64 {
			return math.Min(1, t/0.04) * math.Exp(-t*5)
		}
		return one(1.0, r.Int63(), verbMid.with(0.3), func(b *sb) {
			b.tone(0, 0.24, sweep(f*1.06, f, 0.24), hoo, wSin, 0.32)
			b.tone(0.28, 0.34, sweep(f*1.04, f*0.94, 0.34), hoo, wSin, 0.26)
		})
	}},
	"coopgo": {2, func(p []int, v int, r *rand.Rand) []byte {
		f := 392.0 * (1 + float64(v)*0.02)
		return one(0.7, r.Int63(), verbSmall.with(0.3), func(b *sb) {
			b.pluck(0, f, 0.4, 0.6, 0.4)
			b.pluck(0.07, f*1.5, 0.4, 0.6, 0.32)
		})
	}},
	"knot": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbMid.with(0.36), func(b *sb) {
			b.filtered(0, 0.32, func(t float64) float64 {
				return wSaw(t*(70+t*90)) * math.Min(1, t/0.05) * math.Max(0, 1-t/0.32)
			}, fBP, sweep(500, 1300, 0.3), 3, 0.26)
			b.glass(0.3, 659.3, 0.42)
			b.glass(0.34, 988, 0.2)
		})
	}},
	"trialon": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.6, r.Int63(), verbSmall.with(0.3), func(b *sb) {
			b.noise(0, 0.18, expEnv(0.005, 16), fBP, cst(2600+float64(v)*300), 1.2, 0.3)
			b.tone(0.01, 0.3, sweep(150, 190, 0.3), expEnv(0.01, 8), wTri, 0.2)
		})
	}},
	"trialoff": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.3, r.Int63(), verbSmall.with(0.24), func(b *sb) {
			b.tick(0, 1000, 0.45)
		})
	}},
}
