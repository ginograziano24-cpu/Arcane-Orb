package main

// Procedural sound design.
//
// Every sound is synthesised offline into a stereo clip from a small DSP
// toolkit: modal synthesis (marimba, glass, metal, handpan), FM bells and
// clanks, Karplus-Strong plucks (harp), filtered noise (wood, stone,
// puffs, whooshes), pitch-swept subs for impacts, and a stereo Freeverb
// for space. Clips are rendered lazily, cached, and come in several
// variants so repeated hits never sound identical.
//
// Brick breaks are musical: each world has its own instrument, and every
// consecutive hit climbs a pentatonic scale in the key of the act's music,
// so combos play melodies that sit on top of the soundtrack.

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------- builder

type sb struct {
	x []float64 // mono dry signal
	r *rand.Rand
}

func newSB(dur float64, seed int64) *sb {
	return &sb{x: make([]float64, int(dur*sampleRate)), r: rand.New(rand.NewSource(seed))}
}

func (b *sb) dur() float64 { return float64(len(b.x)) / sampleRate }

// gen adds samples from f(t) (t local to start) for dur seconds.
func (b *sb) gen(start, dur float64, f func(t float64) float64) {
	i0 := int(start * sampleRate)
	n := int(dur * sampleRate)
	for i := 0; i < n; i++ {
		j := i0 + i
		if j < 0 {
			continue
		}
		if j >= len(b.x) {
			break
		}
		b.x[j] += f(float64(i) / sampleRate)
	}
}

func wSin(ph float64) float64 { return math.Sin(2 * math.Pi * ph) }
func wTri(ph float64) float64 { p := ph - math.Floor(ph); return 4*math.Abs(p-0.5) - 1 }
func wSaw(ph float64) float64 { p := ph - math.Floor(ph); return 2*p - 1 }
func wSq(ph float64) float64 {
	if ph-math.Floor(ph) < 0.5 {
		return 1
	}
	return -1
}

// tone adds an oscillator with a frequency curve and amplitude envelope.
func (b *sb) tone(start, dur float64, freq, env func(t float64) float64, wave func(float64) float64, gain float64) {
	ph := 0.0
	b.gen(start, dur, func(t float64) float64 {
		ph += freq(t) / sampleRate
		return wave(ph) * env(t) * gain
	})
}

func expEnv(attack, decay float64) func(float64) float64 {
	return func(t float64) float64 {
		a := 1.0
		if attack > 0 && t < attack {
			a = t / attack
		}
		return a * math.Exp(-t*decay)
	}
}

func cst(f float64) func(float64) float64 { return func(float64) float64 { return f } }

// sweep goes exponentially from f0 to f1 over dur.
func sweep(f0, f1, dur float64) func(float64) float64 {
	return func(t float64) float64 { return f0 * math.Pow(f1/f0, math.Min(1, t/dur)) }
}

// modes adds a struck resonator: partials at ratios with their own decays.
func (b *sb) modes(start, f0 float64, ratios, amps, decays []float64, gain, attack float64) {
	for k, rt := range ratios {
		f := f0 * rt
		if f > 18000 {
			continue
		}
		d := decays[k]
		life := math.Min(b.dur()-start, 7/d)
		ph := b.r.Float64()
		b.gen(start, life, func(t float64) float64 {
			a := 1.0
			if t < attack {
				a = t / attack
			}
			return math.Sin(2*math.Pi*(f*t+ph)) * amps[k] * math.Exp(-t*d) * a * gain
		})
	}
}

// fm adds a two-operator FM voice (bells, clanks, growls).
func (b *sb) fm(start, dur, fc, ratio, index, idxDecay, ampAttack, ampDecay, gain float64) {
	b.gen(start, dur, func(t float64) float64 {
		m := index * math.Exp(-t*idxDecay) * math.Sin(2*math.Pi*fc*ratio*t)
		a := 1.0
		if ampAttack > 0 && t < ampAttack {
			a = t / ampAttack
		}
		return math.Sin(2*math.Pi*fc*t+m) * a * math.Exp(-t*ampDecay) * gain
	})
}

// ---------------------------------------------------------------- filters

type biquad struct{ b0, b1, b2, a1, a2, x1, x2, y1, y2 float64 }

const (
	fLP = iota
	fHP
	fBP
)

func (q *biquad) set(kind int, f, res float64) {
	f = math.Max(20, math.Min(f, sampleRate*0.45))
	w := 2 * math.Pi * f / sampleRate
	cs, sn := math.Cos(w), math.Sin(w)
	al := sn / (2 * res)
	var b0, b1, b2 float64
	switch kind {
	case fLP:
		b0, b1, b2 = (1-cs)/2, 1-cs, (1-cs)/2
	case fHP:
		b0, b1, b2 = (1+cs)/2, -(1 + cs), (1+cs)/2
	default:
		b0, b1, b2 = al, 0, -al
	}
	a0 := 1 + al
	q.b0, q.b1, q.b2, q.a1, q.a2 = b0/a0, b1/a0, b2/a0, -2*cs/a0, (1-al)/a0
}

func (q *biquad) proc(x float64) float64 {
	y := q.b0*x + q.b1*q.x1 + q.b2*q.x2 - q.a1*q.y1 - q.a2*q.y2
	q.x2, q.x1, q.y2, q.y1 = q.x1, x, q.y1, y
	return y
}

// filtered adds src(t) passed through a filter whose cutoff follows cut(t).
func (b *sb) filtered(start, dur float64, src func(t float64) float64, kind int, cut func(t float64) float64, res, gain float64) {
	var q biquad
	k := 0
	b.gen(start, dur, func(t float64) float64 {
		if k%16 == 0 {
			q.set(kind, cut(t), res)
		}
		k++
		return q.proc(src(t)) * gain
	})
}

// noise adds a filtered noise burst with envelope env.
func (b *sb) noise(start, dur float64, env func(float64) float64, kind int, cut func(float64) float64, res, gain float64) {
	r := b.r
	b.filtered(start, dur, func(t float64) float64 { return (r.Float64()*2 - 1) * env(t) }, kind, cut, res, gain)
}

// pluck is a Karplus-Strong string (harp, kalimba-like).
func (b *sb) pluck(start, f, sustain, bright, gain float64) {
	n := int(sampleRate / f)
	if n < 2 {
		return
	}
	buf := make([]float64, n)
	lp := 0.0
	for i := range buf {
		lp += (b.r.Float64()*2 - 1 - lp) * bright
		buf[i] = lp
	}
	idx := 0
	b.gen(start, b.dur()-start, func(t float64) float64 {
		nx := (idx + 1) % n
		v := buf[idx]
		buf[idx] = (buf[idx] + buf[nx]) * 0.5 * sustain
		idx = nx
		return v * gain
	})
}

// ---------------------------------------------------------------- building blocks

func (b *sb) sub(start, f0, f1, dur, gain float64) {
	b.tone(start, dur, sweep(f0, f1, dur*0.7), expEnv(0.002, 4/dur), wSin, gain)
}

func (b *sb) impact(start, gain float64) {
	b.sub(start, 150, 38, 0.6, gain)
	b.noise(start, 0.4, expEnv(0.001, 14), fLP, sweep(5000, 300, 0.3), 0.7, gain*0.9)
	b.noise(start, 0.05, expEnv(0, 80), fHP, cst(3000), 0.7, gain*0.5)
}

func (b *sb) whoosh(start, dur, f0, f1, gain float64) {
	env := func(t float64) float64 {
		x := t / dur
		return math.Sin(math.Pi*math.Min(1, x)) * math.Sin(math.Pi*math.Min(1, x))
	}
	b.noise(start, dur, env, fBP, sweep(f0, f1, dur), 1.4, gain)
}

func (b *sb) riser(start, dur, gain float64) {
	env := func(t float64) float64 { return math.Pow(t/dur, 2) }
	b.noise(start, dur, env, fBP, sweep(300, 7000, dur), 2.2, gain)
	b.tone(start, dur, sweep(110, 880, dur), env, wSaw, gain*0.12)
}

func (b *sb) sparkle(start, dur float64, n int, fmin, fmax, gain float64) {
	for i := 0; i < n; i++ {
		t := start + b.r.Float64()*dur
		f := fmin * math.Pow(fmax/fmin, b.r.Float64())
		b.modes(t, f, []float64{1, 2.76}, []float64{1, 0.3}, []float64{22, 40}, gain*(0.4+0.6*b.r.Float64()), 0.001)
	}
}

// glass is a celesta/glass bell note.
func (b *sb) glass(start, f, gain float64) {
	b.modes(start, f, []float64{1, 2.32, 4.25, 6.63}, []float64{1, 0.45, 0.22, 0.1}, []float64{3, 6, 11, 18}, gain, 0.001)
	b.fm(start, 1.2, f*2, 3.5, 0.8, 9, 0, 5, gain*0.15)
}

func (b *sb) brass(start, dur, f, gain float64) {
	b.filtered(start, dur, func(t float64) float64 {
		vib := 1 + 0.004*math.Sin(2*math.Pi*5.5*t)*math.Min(1, t*3)
		return wSaw(f*vib*t) + 0.6*wSaw(f*1.003*vib*t) + 0.3*wSq(f*0.5*t)
	}, fLP, func(t float64) float64 { return 400 + 2600*math.Min(1, t*12)*math.Exp(-t*1.5) }, 0.9,
		gain)
}

// brassNote is brass with a proper envelope.
func (b *sb) brassNote(start, dur, f, gain float64) {
	tmp := newSB(dur+0.05, b.r.Int63())
	tmp.brass(0, dur, f, 1)
	for i := range tmp.x {
		t := float64(i) / sampleRate
		a := math.Min(1, t/0.03) * math.Min(1, math.Max(0, (dur-t)/0.25))
		tmp.x[i] *= a
	}
	b.mix(tmp, start, gain)
}

func (b *sb) pad(start, dur, f, gain float64) {
	tmp := newSB(dur, b.r.Int63())
	for k, det := range []float64{0.996, 1, 1.004, 2.002} {
		ph := tmp.r.Float64()
		g := 1.0
		if k == 3 {
			g = 0.35
		}
		tmp.filtered(0, dur, func(t float64) float64 { return wSaw(f*det*t+ph) * g }, fLP, cst(1400), 0.7, 1)
	}
	for i := range tmp.x {
		t := float64(i) / sampleRate
		tmp.x[i] *= math.Min(1, t/0.35) * math.Min(1, (dur-t)/0.6)
	}
	b.mix(tmp, start, gain)
}

func (b *sb) mix(o *sb, start, gain float64) {
	i0 := int(start * sampleRate)
	for i, v := range o.x {
		if j := i0 + i; j >= 0 && j < len(b.x) {
			b.x[j] += v * gain
		}
	}
}

// ---------------------------------------------------------------- mastering

type master struct {
	wet, room, damp float64
	peak            float64
}

var (
	verbSmall = master{wet: 0.14, room: 0.55, damp: 0.4, peak: 0.6}
	verbMid   = master{wet: 0.24, room: 0.72, damp: 0.35, peak: 0.6}
	verbBig   = master{wet: 0.36, room: 0.86, damp: 0.25, peak: 0.6}
	verbDry   = master{wet: 0.05, room: 0.4, damp: 0.5, peak: 0.6}
)

func (m master) with(peak float64) master { m.peak = peak; return m }

type comb struct {
	buf      []float64
	i        int
	fb, damp float64
	store    float64
}

func (c *comb) proc(x float64) float64 {
	y := c.buf[c.i]
	c.store = y*(1-c.damp) + c.store*c.damp
	c.buf[c.i] = x + c.store*c.fb
	c.i = (c.i + 1) % len(c.buf)
	return y
}

type allpass struct {
	buf []float64
	i   int
}

func (a *allpass) proc(x float64) float64 {
	y := a.buf[a.i]
	o := -x + y
	a.buf[a.i] = x + y*0.5
	a.i = (a.i + 1) % len(a.buf)
	return o
}

// finish applies reverb, normalises and returns interleaved 16-bit stereo.
func (b *sb) finish(m master) []byte {
	n := len(b.x)
	l := make([]float64, n)
	r := make([]float64, n)
	if m.wet > 0 {
		var cl, cr []*comb
		for _, d := range []int{1116, 1188, 1277, 1356, 1422, 1491} {
			cl = append(cl, &comb{buf: make([]float64, d), fb: m.room, damp: m.damp})
			cr = append(cr, &comb{buf: make([]float64, d+23), fb: m.room, damp: m.damp})
		}
		var al, ar []*allpass
		for _, d := range []int{556, 441, 341, 225} {
			al = append(al, &allpass{buf: make([]float64, d)})
			ar = append(ar, &allpass{buf: make([]float64, d+23)})
		}
		for i, x := range b.x {
			in := x * 0.02
			var sl, sr float64
			for k := range cl {
				sl += cl[k].proc(in)
				sr += cr[k].proc(in)
			}
			for k := range al {
				sl = al[k].proc(sl)
				sr = ar[k].proc(sr)
			}
			l[i] = x + sl*m.wet*8
			r[i] = x + sr*m.wet*8
		}
	} else {
		copy(l, b.x)
		copy(r, b.x)
	}
	pk := 1e-9
	for i := range l {
		pk = math.Max(pk, math.Max(math.Abs(l[i]), math.Abs(r[i])))
	}
	g := m.peak / pk
	// trim the silent tail and fade the last few ms
	end := n
	for end > 0 && math.Abs(l[end-1]*g) < 2e-4 && math.Abs(r[end-1]*g) < 2e-4 {
		end--
	}
	end = min(n, end+64)
	out := make([]byte, end*4)
	for i := 0; i < end; i++ {
		f := 1.0
		if rem := end - i; rem < 256 {
			f = float64(rem) / 256
		}
		if i < 32 {
			f *= float64(i) / 32
		}
		lv := softClip(l[i] * g * f)
		rv := softClip(r[i] * g * f)
		putS16(out[i*4:], lv)
		putS16(out[i*4+2:], rv)
	}
	return out
}

func softClip(v float64) float64 {
	if math.Abs(v) < 0.8 {
		return v
	}
	return math.Copysign(0.8+0.2*math.Tanh((math.Abs(v)-0.8)/0.2), v)
}

func putS16(b []byte, v float64) {
	s := int16(math.Max(-1, math.Min(1, v)) * 32000)
	b[0] = byte(uint16(s))
	b[1] = byte(uint16(s) >> 8)
}

// ---------------------------------------------------------------- music theory

var keyRoots = []float64{110, 98, 123.47, 130.81} // act 1-3 (minor), menu (major)

var minorPenta = []int{0, 3, 5, 7, 10}
var majorPenta = []int{0, 2, 4, 7, 9}

// ladder returns the n-th note of the key's pentatonic scale above base.
func ladder(key, n int, base float64) float64 {
	sc := minorPenta
	if key == 3 {
		sc = majorPenta
	}
	semi := sc[n%5] + 12*(n/5)
	return base * math.Pow(2, float64(semi)/12)
}

func semis(f float64, s int) float64 { return f * math.Pow(2, float64(s)/12) }

// relMajor is the root of the relative major (for triumphant stingers).
func relMajor(key int) float64 {
	if key == 3 {
		return keyRoots[3]
	}
	return semis(keyRoots[key], 3)
}

// ---------------------------------------------------------------- instruments

// worldNote plays one note on the world's signature instrument.
func (b *sb) worldNote(world int, start, f, gain, bright float64) {
	switch world {
	case WNeon: // analogue synth pluck with filter snap
		ph1, ph2 := b.r.Float64(), b.r.Float64()
		b.filtered(start, 0.6, func(t float64) float64 {
			e := math.Exp(-t * 6)
			return (wSaw(f*0.996*t+ph1) + wSaw(f*1.004*t+ph2) + 0.5*wSq(f*0.5*t)) * e
		}, fLP, func(t float64) float64 { return 250 + (3500+3000*bright)*math.Exp(-t*16) }, 1.6, gain*0.55)
		b.fm(start, 0.25, f*2, 2, 1.5, 20, 0, 18, gain*0.25)
	case WMeadow: // marimba
		b.modes(start, f, []float64{1, 3.93, 9.2}, []float64{1, 0.32 + 0.2*bright, 0.1}, []float64{6, 20, 45}, gain, 0.0015)
		b.noise(start, 0.012, expEnv(0, 300), fBP, cst(2400), 1.5, gain*0.4)
	case WCaverns: // crystal glass
		b.glass(start, f*2, gain*0.9)
	case WRuins: // handpan
		b.modes(start, f, []float64{1, 2.005, 2.99}, []float64{1, 0.5 + 0.2*bright, 0.28}, []float64{3.2, 5.5, 9}, gain, 0.004)
		b.noise(start, 0.08, expEnv(0.002, 50), fLP, cst(380), 0.8, gain*0.6)
	case WClouds: // harp
		b.pluck(start, f, 0.9985, 0.35+0.4*bright, gain*1.2)
		b.tone(start, 1.2, cst(f*2), expEnv(0.004, 4), wSin, gain*0.12)
	case WMachine: // tuned steel pipe
		b.modes(start, f, []float64{1, 2.76, 5.40, 8.93}, []float64{1, 0.55, 0.3, 0.16}, []float64{4, 8, 14, 22}, gain, 0.001)
		b.fm(start, 0.12, f*3, 1.414, 5, 40, 0, 40, gain*0.25)
	}
}

func worldMaster(world int) master {
	switch world {
	case WNeon:
		return master{wet: 0.22, room: 0.7, damp: 0.3, peak: 0.42}
	case WMeadow:
		return master{wet: 0.12, room: 0.5, damp: 0.5, peak: 0.45}
	case WCaverns:
		return master{wet: 0.42, room: 0.88, damp: 0.2, peak: 0.4}
	case WRuins:
		return master{wet: 0.3, room: 0.8, damp: 0.35, peak: 0.45}
	case WClouds:
		return master{wet: 0.32, room: 0.82, damp: 0.25, peak: 0.42}
	}
	return master{wet: 0.2, room: 0.66, damp: 0.2, peak: 0.42}
}

// debris adds the world's material breaking.
func (b *sb) debris(world int, start, gain float64) {
	r := b.r
	switch world {
	case WNeon: // digital crunch
		hold, cnt := 0.0, 0
		b.gen(start, 0.09, func(t float64) float64 {
			if cnt%7 == 0 {
				hold = r.Float64()*2 - 1
			}
			cnt++
			return hold * math.Exp(-t*40) * gain * 0.5
		})
		b.tone(start, 0.06, sweep(1400, 180, 0.06), expEnv(0, 40), wSq, gain*0.12)
	case WMeadow: // wood crack + knock
		b.noise(start, 0.05, expEnv(0, 70), fBP, cst(1400+r.Float64()*600), 2.5, gain*0.9)
		b.modes(start, 190+r.Float64()*40, []float64{1, 2.6}, []float64{1, 0.4}, []float64{30, 55}, gain*0.5, 0.001)
	case WCaverns: // crystal crumble
		for i := 0; i < 9; i++ {
			b.modes(start+r.Float64()*0.12, 3000+r.Float64()*4500, []float64{1}, []float64{1}, []float64{70}, gain*0.22, 0.0005)
		}
		b.noise(start, 0.1, expEnv(0, 35), fHP, cst(5000), 0.7, gain*0.25)
	case WRuins: // stone crumble
		for i := 0; i < 6; i++ {
			b.noise(start+r.Float64()*0.1, 0.05, expEnv(0.001, 60), fLP, cst(700+r.Float64()*900), 0.8, gain*0.6)
		}
		b.sub(start, 120, 60, 0.15, gain*0.5)
	case WClouds: // soft puff
		b.noise(start, 0.22, expEnv(0.008, 14), fLP, sweep(2400, 400, 0.2), 0.7, gain*0.8)
		b.noise(start, 0.12, expEnv(0.004, 25), fHP, cst(4500), 0.7, gain*0.2)
	case WMachine: // metal clank
		b.fm(start, 0.2, 480+r.Float64()*120, 1.414, 6, 25, 0, 24, gain*0.5)
		b.noise(start, 0.04, expEnv(0, 90), fBP, cst(3200), 1.2, gain*0.6)
	}
}

// ---------------------------------------------------------------- recipes

type recipe struct {
	vars  int
	build func(p []int, v int, r *rand.Rand) []byte
}

// parseName splits "name.1.2.3" into ("name", [1 2 3]).
func parseName(name string) (string, []int) {
	parts := strings.Split(name, ".")
	var p []int
	for _, s := range parts[1:] {
		n, _ := strconv.Atoi(s)
		p = append(p, n)
	}
	return parts[0], p
}

func arg(p []int, i, def int) int {
	if i < len(p) {
		return p[i]
	}
	return def
}

func one(dur float64, seed int64, m master, f func(b *sb)) []byte {
	b := newSB(dur, seed)
	f(b)
	return b.finish(m)
}

var recipes = map[string]recipe{
	// ---- bricks ----------------------------------------------------
	// brk.world.key.step : musical break note
	"brk": {2, func(p []int, v int, r *rand.Rand) []byte {
		w, k, st := arg(p, 0, 0), arg(p, 1, 0), arg(p, 2, 0)
		f := ladder(k, st, keyRoots[k]*2)
		return one(1.8, r.Int63(), worldMaster(w), func(b *sb) {
			b.worldNote(w, 0, f*(1+(r.Float64()-0.5)*0.004), 1, math.Min(1, float64(st)/14))
		})
	}},
	"deb": {4, func(p []int, v int, r *rand.Rand) []byte {
		w := arg(p, 0, 0)
		m := worldMaster(w)
		m.peak = 0.4
		m.wet *= 0.5
		return one(0.6, r.Int63(), m, func(b *sb) { b.debris(w, 0, 1) })
	}},
	"tuf": {3, func(p []int, v int, r *rand.Rand) []byte {
		w, k := arg(p, 0, 0), arg(p, 1, 0)
		m := worldMaster(w)
		m.peak = 0.36
		return one(0.8, r.Int63(), m, func(b *sb) {
			f := keyRoots[k] * 2 * []float64{1, 1.5, 1.335}[v%3]
			tmp := newSB(0.8, r.Int63())
			tmp.worldNote(w, 0, f, 1, 0)
			for i := range tmp.x {
				tmp.x[i] *= math.Exp(-float64(i) / sampleRate * 14)
			}
			b.mix(tmp, 0, 0.8)
			b.debris(w, 0, 0.5)
		})
	}},
	"thump": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.5, r.Int63(), verbSmall.with(0.5), func(b *sb) {
			b.sub(0, 160+r.Float64()*30, 45, 0.3, 1)
			b.noise(0, 0.08, expEnv(0, 40), fLP, cst(900), 0.7, 0.5)
		})
	}},
	"crit": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbMid.with(0.62), func(b *sb) {
			b.noise(0, 0.03, expEnv(0, 140), fHP, cst(2500), 0.7, 1.2)
			b.sub(0, 220, 55, 0.25, 0.9)
			b.fm(0, 0.6, 1760+r.Float64()*200, 2.01, 3, 12, 0, 7, 0.4)
			b.tone(0.01, 0.18, sweep(2600, 900, 0.18), expEnv(0, 18), wSaw, 0.08)
		})
	}},
	"steel": {4, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbMid.with(0.4), func(b *sb) {
			b.modes(0, 1100+r.Float64()*500, []float64{1, 2.76, 5.40, 8.93}, []float64{1, 0.6, 0.35, 0.2}, []float64{9, 14, 22, 30}, 1, 0.0005)
			b.noise(0, 0.02, expEnv(0, 200), fHP, cst(4000), 0.7, 0.5)
		})
	}},
	"shieldhit": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.7, r.Int63(), verbMid.with(0.42), func(b *sb) {
			f := 640 + r.Float64()*160
			b.gen(0, 0.5, func(t float64) float64 {
				vib := 1 + 0.03*math.Sin(2*math.Pi*28*t)
				return math.Sin(2*math.Pi*f*vib*t+2*math.Exp(-t*10)*math.Sin(2*math.Pi*f*2*t)) * math.Exp(-t*9)
			})
		})
	}},
	// ---- ball / paddle ----------------------------------------------
	"pad": {3, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		return one(0.5, r.Int63(), verbSmall.with(0.5), func(b *sb) {
			b.sub(0, 210, 110, 0.12, 0.9)
			b.noise(0, 0.015, expEnv(0, 200), fBP, cst(3000), 1, 0.5)
			f := semis(keyRoots[k]*2, []int{0, 7, 12}[v%3])
			b.modes(0, f, []float64{1, 2, 3}, []float64{1, 0.4, 0.2}, []float64{14, 24, 34}, 0.5, 0.002)
		})
	}},
	"wall": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.25, r.Int63(), verbSmall.with(0.22), func(b *sb) {
			b.modes(0, 340+r.Float64()*80, []float64{1, 2.3}, []float64{1, 0.3}, []float64{40, 70}, 1, 0.001)
			b.noise(0, 0.01, expEnv(0, 300), fBP, cst(2500), 1, 0.4)
		})
	}},
	"launch": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.8, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.whoosh(0, 0.3, 400, 3500, 0.8)
			b.tone(0.02, 0.35, sweep(300, 1200, 0.3), expEnv(0.01, 7), wTri, 0.4)
			b.glass(0.18, 1568, 0.35)
		})
	}},
	"laser": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.4, r.Int63(), verbSmall.with(0.32), func(b *sb) {
			f0 := 2200 + r.Float64()*400
			b.gen(0, 0.16, func(t float64) float64 {
				f := f0 * math.Pow(0.25, t/0.16)
				return math.Sin(2*math.Pi*f*t+1.5*math.Sin(2*math.Pi*f*1.5*t)) * math.Exp(-t*12)
			})
		})
	}},
	"split": {2, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		return one(0.8, r.Int63(), verbMid.with(0.4), func(b *sb) {
			b.pluck(0, semis(keyRoots[k]*4, 0), 0.997, 0.6, 1)
			b.pluck(0.06, semis(keyRoots[k]*4, 7), 0.997, 0.6, 1)
		})
	}},
	// ---- area effects -----------------------------------------------
	"boom": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(2.2, r.Int63(), verbBig.with(0.95), func(b *sb) {
			b.sub(0, 120+r.Float64()*30, 30, 1.1, 1.2)
			b.noise(0, 1.2, expEnv(0.002, 3.5), fLP, sweep(6000, 180, 0.9), 0.7, 1.4)
			b.noise(0, 0.06, expEnv(0, 60), fHP, cst(2000), 0.7, 0.8)
			for i := 0; i < 14; i++ {
				b.noise(0.1+r.Float64()*0.7, 0.04, expEnv(0.001, 70), fBP, cst(800+r.Float64()*2500), 1.5, 0.25)
			}
		})
	}},
	"zap": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.7, r.Int63(), verbMid.with(0.5), func(b *sb) {
			b.gen(0, 0.35, func(t float64) float64 {
				crack := 0.0
				if r.Float64() < 0.02 {
					crack = (r.Float64()*2 - 1) * 2
				}
				buzz := wSq(110*t+0.3*math.Sin(2*math.Pi*37*t)) * 0.3
				return (crack + buzz*(0.5+0.5*math.Sin(2*math.Pi*60*t))) * math.Exp(-t*8)
			})
			b.noise(0, 0.3, expEnv(0, 10), fBP, cst(3000), 1.2, 0.8)
		})
	}},
	"freeze": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbBig.with(0.42), func(b *sb) {
			b.fm(0, 0.9, 2600, 1.5, 2, 3, 0.01, 4, 0.6)
			b.noise(0, 0.5, expEnv(0.05, 6), fHP, sweep(9000, 3000, 0.5), 0.8, 0.3)
			b.sparkle(0, 0.4, 8, 3000, 8000, 0.4)
		})
	}},
	"shatter": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbMid.with(0.55), func(b *sb) {
			b.noise(0, 0.25, expEnv(0, 16), fHP, cst(3500), 0.7, 0.9)
			b.sparkle(0, 0.3, 22, 2500, 9000, 0.6)
			b.sub(0, 200, 80, 0.15, 0.4)
		})
	}},
	"warp": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.0, r.Int63(), verbBig.with(0.45), func(b *sb) {
			for _, d := range []float64{1, 1.012} {
				b.tone(0, 0.45, func(t float64) float64 { return 220 * d * math.Pow(8, t/0.45) }, expEnv(0.01, 5), wTri, 0.4)
			}
			b.whoosh(0, 0.45, 300, 5000, 0.4)
		})
	}},
	// ---- gold -------------------------------------------------------
	"coin": {2, func(p []int, v int, r *rand.Rand) []byte {
		st := arg(p, 0, 0)
		f := ladder(3, st+5, 1046.5)
		return one(0.6, r.Int63(), verbSmall.with(0.32), func(b *sb) {
			b.gen(0, 0.4, func(t float64) float64 {
				ff := f
				if t > 0.045 {
					ff = f * 1.335
				}
				return (wSq(ff*t)*0.35 + math.Sin(2*math.Pi*ff*t)) * math.Exp(-t*11)
			})
		})
	}},
	"coinbig": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbMid.with(0.5), func(b *sb) {
			for i := 0; i < 6; i++ {
				f := ladder(3, i+3, 1046.5)
				b.modes(float64(i)*0.045, f, []float64{1, 2, 3.01}, []float64{1, 0.3, 0.15}, []float64{9, 16, 24}, 0.7, 0.001)
			}
			b.sparkle(0.2, 0.5, 10, 3000, 8000, 0.3)
		})
	}},
	"crystal": {3, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		return one(2.2, r.Int63(), verbBig.with(0.5), func(b *sb) {
			for i := 0; i < 4; i++ {
				b.glass(float64(i)*0.06, ladder(k, i*2+v, keyRoots[k]*8), 0.6)
			}
			b.sparkle(0.1, 0.8, 18, 3500, 9000, 0.35)
		})
	}},
	// ---- combo & frenzy ---------------------------------------------
	"cmb": {2, func(p []int, v int, r *rand.Rand) []byte {
		tier, k := arg(p, 0, 1), arg(p, 1, 0)
		root := relMajor(k) * 4
		return one(2.6, r.Int63(), verbBig.with(0.55+0.08*float64(tier)), func(b *sb) {
			notes := 2 + tier
			for i := 0; i < notes; i++ {
				b.glass(float64(i)*0.055, ladder(3, i*2, root), 0.55)
			}
			b.whoosh(0, 0.3, 600, 6000, 0.35)
			if tier >= 2 {
				b.sparkle(0.1, 0.6, 10*tier, 3000, 9000, 0.3)
			}
			if tier >= 3 {
				b.impact(0, 0.7)
				b.pad(0.05, 1.6, root/4, 0.25)
				b.pad(0.05, 1.6, semis(root/4, 4), 0.2)
				b.pad(0.05, 1.6, semis(root/4, 7), 0.2)
			}
			if tier >= 4 {
				for _, s := range []int{0, 4, 7, 12} {
					b.brassNote(0.05, 1.3, semis(root/4, s), 0.22)
				}
			}
		})
	}},
	"frenzy": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 4
		return one(2.0, r.Int63(), verbBig.with(0.7), func(b *sb) {
			b.riser(0, 0.35, 0.5)
			b.impact(0.35, 0.6)
			for i := 0; i < 8; i++ {
				b.glass(0.35+float64(i)*0.04, ladder(3, i, root), 0.45)
			}
			b.sparkle(0.35, 1.0, 30, 3000, 9500, 0.3)
		})
	}},
	// ---- power-ups --------------------------------------------------
	"pw": {1, func(p []int, v int, r *rand.Rand) []byte {
		id, k := arg(p, 0, 0), arg(p, 1, 0)
		root := relMajor(k) * 4
		return one(1.6, r.Int63(), verbMid.with(0.6), func(b *sb) {
			// shared "you got something good" signature: rising triad
			for i, s := range []int{0, 4, 7, 12} {
				b.glass(float64(i)*0.045, semis(root, s), 0.4)
			}
			switch Power(id) {
			case PMulti:
				for i := 0; i < 3; i++ {
					b.tone(float64(i)*0.06, 0.15, sweep(900+float64(i)*300, 2400, 0.12), expEnv(0, 14), wTri, 0.35)
				}
			case PExpand:
				b.tone(0, 0.5, sweep(250, 900, 0.45), expEnv(0.02, 5), wSaw, 0.12)
				b.tone(0, 0.5, sweep(252, 905, 0.45), expEnv(0.02, 5), wSaw, 0.12)
			case PSlow:
				b.gen(0, 0.7, func(t float64) float64 {
					f := 900 * math.Pow(0.35, t/0.7)
					return math.Sin(2*math.Pi*f*t) * (0.6 + 0.4*math.Sin(2*math.Pi*9*t)) * math.Exp(-t*3) * 0.4
				})
			case PFire:
				b.noise(0, 0.6, expEnv(0.05, 5), fLP, sweep(300, 3000, 0.4), 1.2, 0.9)
				for i := 0; i < 10; i++ {
					b.noise(0.1+r.Float64()*0.4, 0.02, expEnv(0, 120), fHP, cst(3000), 0.7, 0.3)
				}
			case PLaser:
				b.fm(0, 0.6, 440, 2, 4, 2, 0.2, 4, 0.35)
				b.tone(0, 0.5, sweep(200, 2000, 0.45), expEnv(0.05, 6), wSq, 0.06)
			case PBarrier:
				b.gen(0, 0.9, func(t float64) float64 {
					vib := 1 + 0.01*math.Sin(2*math.Pi*6*t)
					return (math.Sin(2*math.Pi*220*vib*t) + 0.6*math.Sin(2*math.Pi*330*vib*t)) * math.Min(1, t*6) * math.Exp(-t*2.5) * 0.35
				})
			case PGlue:
				b.filtered(0, 0.4, func(t float64) float64 { return wSaw(140*t) * math.Exp(-t*6) }, fLP, sweep(2400, 200, 0.3), 6, 0.4)
			case PMagnet:
				b.gen(0, 0.7, func(t float64) float64 {
					return math.Sin(2*math.Pi*300*t+3*math.Sin(2*math.Pi*(6+20*t)*t)) * math.Min(1, t*8) * math.Exp(-t*3) * 0.3
				})
			}
			b.sparkle(0.1, 0.4, 8, 3000, 8000, 0.25)
		})
	}},
	"pwbad": {1, func(p []int, v int, r *rand.Rand) []byte {
		id := arg(p, 0, 0)
		return one(1.2, r.Int63(), verbMid.with(0.55), func(b *sb) {
			b.filtered(0, 0.7, func(t float64) float64 {
				f := 300 * math.Pow(0.5, t/0.7)
				return wSaw(f*t) + wSaw(f*1.06*t)
			}, fLP, func(t float64) float64 { return 300 + 1500*(0.5+0.5*math.Sin(2*math.Pi*5*t)) }, 3, 0.35)
			if Power(id) == PHaste {
				b.gen(0, 0.5, func(t float64) float64 {
					return wSq(880*t) * math.Max(0, math.Sin(2*math.Pi*12*t)) * math.Exp(-t*4) * 0.12
				})
			}
			b.sub(0, 120, 50, 0.4, 0.4)
		})
	}},
	// ---- health -----------------------------------------------------
	"lose": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.8, r.Int63(), verbBig.with(0.75), func(b *sb) {
			b.filtered(0, 0.9, func(t float64) float64 {
				f := 440 * math.Pow(0.2, t/0.9)
				return wSaw(f*t) + wSaw(f*0.99*t)
			}, fLP, sweep(3000, 200, 0.9), 1.5, 0.35)
			b.sub(0.05, 110, 35, 0.8, 0.9)
			b.noise(0.05, 0.4, expEnv(0.001, 8), fLP, cst(600), 0.7, 0.4)
		})
	}},
	"heal": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 2
		return one(2.2, r.Int63(), verbBig.with(0.55), func(b *sb) {
			for i, s := range []int{0, 4, 7, 12, 16, 19} {
				b.pluck(float64(i)*0.05, semis(root, s), 0.998, 0.5, 0.7)
			}
			for _, s := range []int{0, 4, 7} {
				b.pad(0.05, 1.6, semis(root/2, s), 0.12)
			}
		})
	}},
	"shield": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.9, r.Int63(), verbMid.with(0.55), func(b *sb) {
			b.gen(0, 0.6, func(t float64) float64 {
				return (math.Sin(2*math.Pi*196*t+1.5*math.Exp(-t*8)*math.Sin(2*math.Pi*392*t)) + 0.5*math.Sin(2*math.Pi*294*t)) * math.Exp(-t*6)
			})
			b.noise(0, 0.15, expEnv(0, 25), fBP, cst(1800), 2, 0.4)
		})
	}},
	"heartbeat": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.7, r.Int63(), verbSmall.with(0.5), func(b *sb) {
			b.sub(0, 80, 45, 0.18, 1)
			b.sub(0.2, 75, 42, 0.2, 0.7)
		})
	}},
	// ---- level flow -------------------------------------------------
	"clear": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 2
		return one(4.0, r.Int63(), verbBig.with(0.85), func(b *sb) {
			// da-da-da DAAA fanfare, then a sustained major chord with shimmer
			for i, s := range []int{7, 7, 7} {
				b.brassNote(float64(i)*0.11, 0.1, semis(root, s), 0.3)
			}
			for _, s := range []int{0, 4, 7, 12} {
				b.brassNote(0.36, 1.6, semis(root, s), 0.22)
			}
			b.impact(0.36, 0.7)
			for i := 0; i < 8; i++ {
				b.glass(0.36+float64(i)*0.05, ladder(3, i+5, root*2), 0.35)
			}
			b.sparkle(0.4, 1.4, 30, 3000, 9000, 0.25)
		})
	}},
	"actstart": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k]
		return one(4.5, r.Int63(), verbBig.with(0.85), func(b *sb) {
			b.impact(0, 1)
			b.fm(0, 3.5, root, 1.4, 3, 0.8, 0.002, 1.1, 0.45) // gong
			for _, s := range []int{0, 7, 12, 15} {
				b.pad(0.1, 3.2, semis(root, s), 0.18)
			}
			b.riser(1.6, 1.0, 0.15)
		})
	}},
	"over": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k] * 2
		return one(4.0, r.Int63(), verbBig.with(0.7), func(b *sb) {
			for i, s := range []int{12, 10, 7, 3, 0} {
				b.glass(float64(i)*0.32, semis(root, s), 0.5)
			}
			for _, s := range []int{0, 3, 7} {
				b.pad(0.2, 3.2, semis(root/2, s), 0.15)
			}
			b.sub(1.4, 90, 35, 1.2, 0.6)
		})
	}},
	"mapenter": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbMid.with(0.55), func(b *sb) {
			b.whoosh(0, 0.45, 300, 2500, 0.8)
			b.sub(0.38, 140, 45, 0.4, 0.8)
		})
	}},
	"deal": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.0, r.Int63(), verbSmall.with(0.4), func(b *sb) {
			for i := 0; i < 3; i++ {
				b.whoosh(float64(i)*0.09, 0.14, 2500, 7000, 0.6)
				b.noise(float64(i)*0.09+0.12, 0.02, expEnv(0, 150), fBP, cst(1800), 1.2, 0.4)
			}
		})
	}},
	// pick.rarity.key : choosing an upgrade or relic
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
			notes := 2 + rar
			for i := 0; i < notes; i++ {
				b.glass(t0+float64(i)*0.06, ladder(3, i*2, root), 0.5)
			}
			if rar >= 1 {
				b.sparkle(t0, 0.3+0.2*float64(rar), 6*rar, 3000, 9000, 0.3)
			}
			if rar >= 2 {
				b.sub(t0, 160, 50, 0.3, 0.5)
			}
			if rar >= 3 {
				b.impact(t0, 0.7)
				chord := []int{0, 4, 7, 11}
				if rar == 5 {
					chord = []int{0, 3, 7, 10} // mythic: darker
				}
				for _, s := range chord {
					b.pad(t0, dur, semis(root/4, s), 0.14)
				}
			}
			if rar >= 4 {
				for _, s := range []int{0, 7, 12, 16} {
					b.brassNote(t0+0.02, dur*0.8, semis(root/4, s), 0.18)
				}
			}
		})
	}},
	"synergy": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k)
		return one(5.0, r.Int63(), verbBig.with(0.95), func(b *sb) {
			b.riser(0, 0.6, 0.5)
			b.impact(0.6, 1)
			for _, s := range []int{0, 7, 11, 16, 19} {
				b.brassNote(0.6, 2.0, semis(root*2, s), 0.16)
				b.pad(0.6, 2.8, semis(root, s), 0.08)
			}
			for i := 0; i < 12; i++ {
				b.glass(0.6+float64(i)*0.04, ladder(3, i, root*8), 0.32)
			}
			b.sparkle(0.6, 2.0, 50, 2500, 9500, 0.25)
		})
	}},
	"buy": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbMid.with(0.55), func(b *sb) {
			for i := 0; i < 5; i++ {
				f := ladder(3, i+5, 1046.5)
				b.modes(float64(i)*0.035, f, []float64{1, 2.01, 3.0}, []float64{1, 0.3, 0.12}, []float64{14, 22, 30}, 0.5, 0.001)
			}
			b.fm(0.2, 1.0, 1568, 3.5, 2.5, 4, 0, 3, 0.4) // register bell
			b.noise(0.18, 0.05, expEnv(0, 60), fBP, cst(2500), 1.5, 0.4)
		})
	}},
	// ---- class abilities ------------------------------------------
	"firestorm": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(2.6, r.Int63(), verbBig.with(0.95), func(b *sb) {
			b.sub(0, 90, 38, 1.0, 1.1)
			b.whoosh(0, 0.55, 200, 4200, 1.0)
			// roaring flame body: low-passed noise that opens up then settles
			b.noise(0.05, 1.9, expEnv(0.08, 1.6), fLP, sweep(900, 3800, 0.5), 0.9, 1.3)
			b.noise(0.05, 1.6, expEnv(0.1, 2.2), fBP, sweep(400, 1400, 1.2), 2.0, 0.6)
			// crackles
			for i := 0; i < 40; i++ {
				b.noise(0.1+r.Float64()*1.8, 0.012, expEnv(0, 180), fHP, cst(2500+r.Float64()*4000), 1, 0.35)
			}
			b.brass(0.04, 0.9, 110, 0.35)
		})
	}},
	"thunder": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(3.4, r.Int63(), verbBig.with(1.0), func(b *sb) {
			// the crack
			b.noise(0, 0.08, expEnv(0, 40), fHP, cst(1800), 0.7, 1.4)
			b.gen(0, 0.25, func(t float64) float64 {
				c := 0.0
				if r.Float64() < 0.05 {
					c = (r.Float64()*2 - 1) * 2.5
				}
				return c * math.Exp(-t*12)
			})
			// the rolling boom
			b.sub(0.02, 70, 30, 1.6, 1.3)
			b.noise(0.06, 2.8, expEnv(0.12, 1.3), fLP, sweep(1400, 160, 2.6), 0.8, 1.4)
			for i := 0; i < 6; i++ {
				b.noise(0.3+r.Float64()*1.8, 0.4, expEnv(0.05, 5), fLP, cst(200+r.Float64()*300), 1.2, 0.5)
			}
			b.fm(0, 0.6, 3200, 1.41, 4, 6, 0, 6, 0.25)
		})
	}},
	"story": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k]
		return one(5.0, r.Int63(), verbBig.with(0.55), func(b *sb) {
			b.sub(0, 70, 50, 2.5, 0.5)
			for _, st := range []int{0, 7, 12, 16} {
				b.pad(0.05, 4.2, semis(root, st), 0.13)
			}
			b.fm(0.2, 3.5, semis(root*4, 7), 3.5, 2.2, 1.5, 0, 1.2, 0.22)
			b.sparkle(0.3, 2.5, 18, 2500, 7000, 0.12)
		})
	}},
	"storyline": {3, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k] * 4
		return one(2.6, r.Int63(), verbBig.with(0.32), func(b *sb) {
			n := []int{0, 4, 7}[v%3]
			b.glass(0, semis(root, n), 0.5)
			b.pluck(0.02, semis(root/2, n), 0.998, 0.35, 0.35)
		})
	}},
	"parry": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.6, r.Int63(), verbBig.with(0.6), func(b *sb) {
			b.impact(0, 0.5)
			b.modes(0, 523.25*float64(1+v%3)/2+523.25, []float64{1, 2.76, 5.4}, []float64{1, 0.45, 0.2}, []float64{3, 6, 10}, 0.5, 0.001)
			b.glass(0.02, 2093, 0.3)
			b.whoosh(0, 0.25, 600, 5000, 0.35)
		})
	}},
	"bossintro": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(5.5, r.Int63(), verbBig.with(0.95), func(b *sb) {
			// three slow war-drum hits, a rising brass swell and a dark choir pad
			for i, tm := range []float64{0.0, 0.9, 1.8} {
				b.sub(tm, 70-float64(i)*6, 34, 1.2, 1.0)
				b.noise(tm, 0.4, expEnv(0.002, 7), fLP, cst(600), 0.8, 0.6)
			}
			for _, st := range []int{0, 7, 12} {
				b.brass(1.9, 2.8, semis(55, st), 0.28)
			}
			for _, st := range []int{0, 3, 7, 10} {
				b.pad(0.4, 4.6, semis(110, st), 0.11)
			}
			b.riser(2.2, 1.6, 0.5)
			b.impact(3.85, 0.9)
			b.sub(3.85, 90, 30, 1.4, 1.2)
		})
	}},
	"abilready": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.2, r.Int63(), verbMid.with(0.4), func(b *sb) {
			b.glass(0, 1318.5, 0.5)
			b.glass(0.08, 1975.5, 0.45)
			b.sparkle(0.05, 0.4, 10, 3000, 8000, 0.25)
		})
	}},
	"achievement": {1, func(p []int, v int, r *rand.Rand) []byte {
		// a bright bell arpeggio over a warm major pad: unmistakably "you did it"
		return one(3.2, r.Int63(), verbBig.with(0.5), func(b *sb) {
			for i, st := range []int{0, 4, 7, 12, 16} {
				b.glass(float64(i)*0.07, semis(523.25, st), 0.42-float64(i)*0.03)
			}
			for _, st := range []int{0, 4, 7, 11} {
				b.pad(0.05, 2.4, semis(261.63, st), 0.09)
			}
			b.brass(0.3, 1.4, 261.63, 0.12)
			b.sparkle(0.25, 1.4, 22, 3500, 9000, 0.22)
		})
	}},
	"deny": {1, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.5, r.Int63(), verbSmall.with(0.35), func(b *sb) {
			for i := 0; i < 2; i++ {
				b.filtered(float64(i)*0.11, 0.09, func(t float64) float64 { return wSq(140 * t) }, fLP, cst(900), 0.7, 0.6)
			}
		})
	}},
	"event": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k] * 2
		return one(3.5, r.Int63(), verbBig.with(0.6), func(b *sb) {
			for i := 0; i < 9; i++ {
				b.pluck(float64(i)*0.04, ladder(k, i, root), 0.998, 0.45, 0.5)
			}
			for _, s := range []int{0, 2, 7} {
				b.pad(0.3, 2.6, semis(root/2, s), 0.12)
			}
		})
	}},
	"good": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 4
		return one(1.8, r.Int63(), verbBig.with(0.55), func(b *sb) {
			for i, s := range []int{0, 4, 7, 12} {
				b.glass(float64(i)*0.06, semis(root, s), 0.5)
			}
			b.sparkle(0.1, 0.5, 12, 3000, 9000, 0.3)
		})
	}},
	"bad": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := keyRoots[k] * 2
		return one(1.8, r.Int63(), verbBig.with(0.55), func(b *sb) {
			for i, s := range []int{6, 3, 0} {
				b.glass(float64(i)*0.1, semis(root, s), 0.45)
			}
			b.sub(0.2, 90, 40, 0.6, 0.5)
		})
	}},
	"rest": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k)
		return one(3.0, r.Int63(), verbBig.with(0.5), func(b *sb) {
			for _, s := range []int{0, 4, 7, 11, 14} {
				b.pad(0, 2.6, semis(root, s), 0.1)
			}
			for i := 0; i < 25; i++ {
				b.noise(r.Float64()*2.4, 0.015, expEnv(0, 200), fBP, cst(1500+r.Float64()*2000), 1.5, 0.25)
			}
		})
	}},
	// ---- boss -------------------------------------------------------
	"roar": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(2.6, r.Int63(), verbBig.with(0.9), func(b *sb) {
			b.gen(0, 1.6, func(t float64) float64 {
				f := 70 * (1 + 0.2*math.Sin(2*math.Pi*1.3*t))
				return math.Sin(2*math.Pi*f*t+3*math.Sin(2*math.Pi*f*0.5*t)+(r.Float64()-0.5)*1.5) * math.Min(1, t*5) * math.Exp(-t*1.4)
			})
			b.noise(0, 1.6, func(t float64) float64 { return math.Min(1, t*4) * math.Exp(-t*1.6) }, fBP, cst(450), 2, 1.2)
			b.noise(0, 1.6, func(t float64) float64 { return math.Min(1, t*4) * math.Exp(-t*1.6) }, fBP, cst(900), 2.5, 0.7)
			b.sub(0, 60, 35, 1.4, 0.8)
		})
	}},
	"bosshit": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(1.4, r.Int63(), verbBig.with(0.75), func(b *sb) {
			b.modes(0, 170+r.Float64()*40, []float64{1, 2.76, 5.40, 8.93}, []float64{1, 0.6, 0.35, 0.2}, []float64{5, 9, 14, 20}, 0.8, 0.0008)
			b.sub(0, 140, 40, 0.4, 1)
			b.noise(0, 0.05, expEnv(0, 90), fHP, cst(2500), 0.7, 0.7)
		})
	}},
	"bossdie": {1, func(p []int, v int, r *rand.Rand) []byte {
		k := arg(p, 0, 0)
		root := relMajor(k) * 2
		return one(6.0, r.Int63(), verbBig.with(0.98), func(b *sb) {
			for i := 0; i < 6; i++ {
				t := float64(i)*0.22 + r.Float64()*0.06
				b.sub(t, 130-float64(i)*10, 28, 1.0, 0.9)
				b.noise(t, 0.9, expEnv(0.002, 4), fLP, sweep(5000, 200, 0.8), 0.7, 1.0)
			}
			b.noise(0, 3.0, func(t float64) float64 { return math.Min(1, t*2) * math.Exp(-t*1.2) }, fLP, cst(250), 0.7, 0.9)
			for _, s := range []int{0, 4, 7, 12, 16} {
				b.brassNote(1.5, 2.4, semis(root, s), 0.13)
				b.pad(1.5, 3.4, semis(root/2, s), 0.08)
			}
			b.sparkle(1.5, 2.5, 50, 2500, 9500, 0.25)
		})
	}},
	// ---- interface --------------------------------------------------
	"hover": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.12, r.Int63(), verbDry.with(0.12), func(b *sb) {
			b.modes(0, 2600+float64(v)*180, []float64{1}, []float64{1}, []float64{70}, 1, 0.0008)
		})
	}},
	"click": {2, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.2, r.Int63(), verbDry.with(0.3), func(b *sb) {
			b.modes(0, 900+float64(v)*60, []float64{1, 2.1}, []float64{1, 0.4}, []float64{45, 80}, 1, 0.0008)
			b.noise(0, 0.008, expEnv(0, 400), fHP, cst(3000), 0.7, 0.5)
		})
	}},
	"cardhover": {3, func(p []int, v int, r *rand.Rand) []byte {
		return one(0.5, r.Int63(), verbMid.with(0.2), func(b *sb) {
			b.glass(0, []float64{2093, 2349, 2637}[v%3], 0.6)
		})
	}},
}

// prewarm lists clips worth rendering before they are first needed.
func prewarmNames() []string {
	var out []string
	for w := 0; w < NumWorlds; w++ {
		out = append(out, "deb."+itoa(w), "tuf."+itoa(w)+".0")
		for st := 0; st < 15; st++ {
			out = append(out, "brk."+itoa(w)+".0."+itoa(st))
		}
	}
	for _, n := range []string{"crit", "steel", "shieldhit", "pad.0", "wall", "launch", "laser", "boom", "zap",
		"thump", "coinbig", "hover", "click", "cardhover", "deal", "mapenter", "lose.0", "clear.0", "pick.0.0", "pick.1.0",
		"pick.2.0", "pick.3.0", "pick.4.0", "synergy.0", "actstart.0", "heartbeat", "firestorm", "thunder", "abilready", "bossintro", "parry", "achievement"} {
		out = append(out, n)
	}
	for i := 0; i < 12; i++ {
		out = append(out, "coin."+itoa(i))
	}
	return out
}
