//go:build capture && js

package main

// The trailer's soundtrack: an original cue in D minor at 120 BPM built
// from the game's own synth (sfx.go), with every sound the game made
// during capture mixed underneath.

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
)

const beat = 60 / trailerBPM // 0.5 s
const bar = 4 * beat         // 2 s

// pitch helpers: D minor
func hz(midi float64) float64 { return 440 * math.Pow(2, (midi-69)/12) }

const (
	mD2 = 38.0
	mD3 = 50.0
	mD4 = 62.0
	mD5 = 74.0
)

// chord tones as semitones above D (the root may sit below)
type chord struct{ root, third, fifth float64 }

var (
	chDm = chord{0, 3, 7}
	chBb = chord{-4, 0, 3}  // Bb D F
	chF  = chord{3, 7, 10}  // F A C
	chC  = chord{-2, 2, 5}  // C E G
	chGm = chord{5, 8, 12}  // G Bb D
	chA  = chord{7, 11, 14} // A C# E
)

// progA runs under the first half, progB under the climax.
var progA = []chord{chDm, chBb, chF, chC}
var progB = []chord{chDm, chBb, chGm, chA}

func chordAt(t float64) chord {
	b := int(t / bar)
	if t >= 38 {
		return progB[(b-19)%4]
	}
	return progA[(b-4+400)%4]
}

// ------------------------------------------------------------ instruments

func (b *sb) kick(t, g float64) {
	b.tone(t, 0.42, sweep(140, 42, 0.09), expEnv(0.001, 9), wSin, g)
	b.noise(t, 0.02, expEnv(0, 120), fHP, cst(2500), 0.7, g*0.35)
}

func (b *sb) snare(t, g float64) {
	b.noise(t, 0.28, expEnv(0.001, 16), fBP, cst(1900), 0.9, g)
	b.noise(t, 0.18, expEnv(0.001, 24), fHP, cst(5000), 0.7, g*0.5)
	b.tone(t, 0.12, sweep(220, 160, 0.08), expEnv(0.001, 28), wTri, g*0.55)
}

func (b *sb) hat(t, g float64) {
	b.noise(t, 0.06, expEnv(0, 70), fHP, cst(8500), 0.7, g)
}

func (b *sb) tom(t, f, g float64) {
	b.tone(t, 0.45, sweep(f*1.6, f, 0.06), expEnv(0.001, 8), wSin, g)
	b.noise(t, 0.1, expEnv(0.001, 30), fLP, cst(1200), 0.7, g*0.25)
}

// warDrum is a huge low taiko.
func (b *sb) warDrum(t, g float64) {
	b.tone(t, 1.2, sweep(96, 38, 0.18), expEnv(0.002, 3.6), wSin, g)
	b.noise(t, 0.5, expEnv(0.002, 9), fLP, sweep(900, 200, 0.3), 0.8, g*0.7)
	b.noise(t, 0.04, expEnv(0, 60), fBP, cst(2400), 1.2, g*0.3)
}

// bassNote is a round sub with a little bite.
func (b *sb) bassNote(t, dur, f, g float64) {
	b.gen(t, dur, func(x float64) float64 {
		env := math.Min(1, x/0.004) * math.Exp(-x*3.5) * math.Min(1, (dur-x)/0.03)
		return (math.Sin(2*math.Pi*f*x) + 0.35*wTri(2*f*x) + 0.12*wSaw(f*x)) * env * g
	})
}

// string is a bowed-string ostinato note: two detuned saws, a filter
// that opens on the attack, and a short release.
func (b *sb) strNote(t, dur, f, bright, g float64) {
	tmp := newSB(dur+0.15, b.r.Int63())
	ph := tmp.r.Float64()
	tmp.filtered(0, dur+0.15, func(x float64) float64 {
		return wSaw(f*x+ph) + 0.8*wSaw(f*1.006*x) + 0.3*wSq(f*0.5*x)
	}, fLP, func(x float64) float64 { return 500 + bright*(1-math.Exp(-x*40))*math.Exp(-x*6) }, 1.1, 1)
	for i := range tmp.x {
		x := float64(i) / sampleRate
		tmp.x[i] *= math.Min(1, x/0.006) * math.Min(1, math.Max(0, (dur+0.12-x)/0.12))
	}
	b.mix(tmp, t, g)
}

// choir is a sustained "aah": detuned saws through vowel formants.
func (b *sb) choir(t, dur, f, g float64) {
	tmp := newSB(dur, b.r.Int63())
	var f1, f2, f3 biquad
	f1.set(fBP, 760, 5)
	f2.set(fBP, 1180, 6)
	f3.set(fBP, 2650, 7)
	phs := []float64{tmp.r.Float64(), tmp.r.Float64(), tmp.r.Float64()}
	for i := range tmp.x {
		x := float64(i) / sampleRate
		vib := 1 + 0.006*math.Sin(2*math.Pi*5.2*x+phs[0]*6)
		src := wSaw(f*vib*x+phs[0]) + wSaw(f*1.004*vib*x+phs[1]) + wSaw(f*0.996*vib*x+phs[2])
		v := f1.proc(src)*1.0 + f2.proc(src)*0.7 + f3.proc(src)*0.3
		env := math.Min(1, x/0.7) * math.Min(1, (dur-x)/0.9)
		tmp.x[i] = v * env
	}
	b.mix(tmp, t, g)
}

func (b *sb) reverseSwell(t, dur, g float64) {
	b.noise(t, dur, func(x float64) float64 { return math.Pow(x/dur, 3) }, fHP, sweep(1800, 7000, dur), 0.8, g)
}

func (b *sb) timpaniRoll(t, dur, f, g float64) {
	n := int(dur / 0.055)
	for i := 0; i < n; i++ {
		k := float64(i) / float64(n)
		b.tone(t+float64(i)*0.055, 0.35, cst(f*(1+0.01*b.r.Float64())), expEnv(0.002, 7), wSin, g*(0.25+0.75*k*k))
		b.noise(t+float64(i)*0.055, 0.05, expEnv(0, 50), fLP, cst(700), 0.7, g*0.2*k)
	}
}

// ------------------------------------------------------------ the cue

type stem struct {
	b         *sb
	m         master
	gain, pan float64
}

func renderScore() (l, r []float64) {
	dur := trailerLength() + 1
	drums := newSB(dur, 11)
	low := newSB(dur, 12)  // bass + ostinato
	high := newSB(dur, 13) // second ostinato voice
	pads := newSB(dur, 14) // pads + choir
	brass := newSB(dur, 15)
	fx := newSB(dur, 16) // bells, risers, impacts

	note := func(semi float64, base float64) float64 { return hz(base + semi) }

	// ---- 0-6: cold open --------------------------------------------
	pads.pad(0, 6.2, hz(mD2), 0.5)
	pads.pad(0.4, 5.8, hz(mD2+7), 0.3)
	pads.choir(1.2, 5.0, hz(mD4), 0.28)
	pads.choir(2.4, 3.8, hz(mD4+3), 0.2)
	pads.choir(3.1, 3.1, hz(mD4+7), 0.18)
	for _, t := range []float64{0, 2, 4} {
		drums.warDrum(t, 0.55+0.15*t/4)
	}
	for i, n := range []float64{mD5, mD5 + 3, mD5 + 7, mD5 + 5} {
		fx.glass(0.8+float64(i)*0.55, hz(n), 0.32)
	}
	fx.glass(3.1, hz(mD5+12), 0.4)
	fx.glass(3.1, hz(mD5), 0.3)
	fx.sparkle(3.1, 1.5, 14, 2500, 8000, 0.15)
	fx.riser(4.0, 2.0, 0.9)
	fx.reverseSwell(4.6, 1.4, 0.9)
	drums.timpaniRoll(4.9, 1.1, hz(mD2), 0.55)

	// ---- 6: title slam ---------------------------------------------
	fx.impact(6, 1.3)
	drums.warDrum(6, 1.0)
	for _, s := range []float64{0, 3, 7, 12} {
		brass.brassNote(6, 1.8, note(s, mD3), 0.32)
	}
	brass.brassNote(6, 1.8, hz(mD2), 0.3)
	pads.choir(6, 2.3, hz(mD4+7), 0.3)
	pads.choir(6, 2.3, hz(mD4+3), 0.3)
	drums.timpaniRoll(7.0, 1.0, hz(mD2), 0.5)
	fx.reverseSwell(7.2, 0.8, 0.7)

	// ---- 8-32: main theme ------------------------------------------
	melody := []struct{ t, d, n float64 }{ // from bar 8 (16 s), semis above D4
		{0, 1, 0}, {1, 0.5, 7}, {1.5, 0.5, 5}, {2, 1, 3}, {3, 1, 5},
		{4, 1.5, 7}, {5.5, 0.5, 10}, {6, 2, 9},
		{8, 1, 7}, {9, 0.5, 12}, {9.5, 0.5, 10}, {10, 1, 9}, {11, 1, 7},
		{12, 1.5, 5}, {13.5, 0.5, 3}, {14, 2, 2},
	}
	for t := 8.0; t < 32-1e-9; t += beat / 2 {
		ch := chordAt(t)
		i := int(math.Round((t - 8) / (beat / 2)))
		pos := i % 16
		inGap := t >= 24 && t < 26 // the fusion moment breathes
		// drums
		if !inGap {
			if pos == 0 || pos == 8 || pos == 11 {
				drums.kick(t, 0.85)
			}
			if pos == 4 || pos == 12 {
				drums.snare(t, 0.45+0.25*clamp01((t-16)/8))
			}
			if t >= 16 {
				drums.hat(t, 0.13+0.06*float64(pos%2))
			}
			if t >= 20 && pos%4 == 2 {
				drums.hat(t+beat/4, 0.08)
			}
		}
		// bass on eighths (root, octave bounce)
		bn := note(ch.root, mD2)
		if pos%4 == 3 {
			bn *= 2
		}
		if !inGap {
			low.bassNote(t, beat/2*0.9, bn, 0.5)
		}
		// ostinato: root-fifth-octave-fifth figure, brighter as it builds
		fig := []float64{ch.root, ch.fifth, ch.root + 12, ch.fifth, ch.third + 12, ch.fifth, ch.root + 12, ch.fifth}
		bright := 1400 + 1800*clamp01((t-8)/20)
		if !inGap {
			low.strNote(t, beat/2*0.85, note(fig[pos%8], mD3), bright, 0.22)
			if t >= 12 {
				high.strNote(t+0.004, beat/2*0.7, note(fig[(pos+2)%8]+12, mD3), bright*1.2, 0.12)
			}
		}
	}
	for t := 8.0; t < 32-1e-9; t += bar {
		ch := chordAt(t)
		for k, s := range []float64{ch.root, ch.third, ch.fifth} {
			pads.pad(t, bar+0.3, note(s, mD3), 0.16-0.02*float64(k))
		}
		if t >= 16 {
			pads.choir(t, bar+0.4, note(ch.third, mD4), 0.11)
			pads.choir(t, bar+0.4, note(ch.fifth, mD4), 0.09)
		}
	}
	for _, m := range melody {
		t := 16 + m.t*beat
		if t >= 24 && t < 26 {
			continue
		}
		brass.brassNote(t, m.d*beat*0.95, note(m.n, mD4), 0.22)
		brass.brassNote(t, m.d*beat*0.95, note(m.n-12, mD4), 0.12)
	}
	// 24-26: the fusion flourish
	for i := 0; i < 8; i++ {
		fx.glass(24.1+float64(i)*0.09, note([]float64{0, 3, 7, 12, 15, 19, 24, 27}[i], mD5-12), 0.24)
	}
	fx.sparkle(24.9, 1.2, 26, 3000, 9000, 0.2)
	fx.impact(25.0, 0.6)
	pads.choir(24, 2.2, hz(mD4+7), 0.18)
	drums.timpaniRoll(25.2, 0.8, hz(mD2), 0.4)
	fx.reverseSwell(25.3, 0.7, 0.6)
	// 26-32 build, then the fill
	for i := 0; i < 8; i++ {
		drums.tom(30+float64(i)*beat/2, 150-float64(i)*10, 0.45+0.05*float64(i))
	}
	for i := 0; i < 16; i++ {
		drums.snare(31+float64(i)*beat/8, 0.12+0.35*float64(i)/16)
	}
	fx.riser(29.5, 2.5, 1.0)
	fx.reverseSwell(30.5, 1.5, 1.0)

	// ---- 32-38: the Guardian rises (the game's own sting leads) -----
	fx.impact(32, 1.0)
	pads.pad(32, 6.2, hz(mD2-12), 0.45)
	pads.pad(32, 6.2, hz(mD2), 0.3)
	pads.choir(32.6, 5.2, hz(mD4-12+3), 0.2)
	pads.choir(34.6, 3.4, hz(mD4-12+10), 0.16) // Bb-ish tension
	for _, t := range []float64{34.0, 35.0, 36.2, 36.7} {
		drums.warDrum(t, 0.5)
	}
	drums.timpaniRoll(37.0, 1.0, hz(mD2), 0.7)
	fx.riser(36.6, 1.4, 0.8)
	fx.reverseSwell(37.1, 0.9, 1.0)

	// ---- 38-46: climax ---------------------------------------------
	climaxMel := []struct{ t, d, n float64 }{
		{0, 1.5, 12}, {1.5, 0.5, 10}, {2, 1, 9}, {3, 1, 7},
		{4, 1.5, 5}, {5.5, 0.5, 7}, {6, 2, 10},
		{8, 1.5, 7}, {9.5, 0.5, 10}, {10, 1, 12}, {11, 1, 15},
		{12, 4, 13},
	}
	for t := 38.0; t < 45.5-1e-9; t += beat / 4 {
		ch := chordAt(t)
		i := int(math.Round((t - 38) / (beat / 4)))
		pos := i % 16
		if pos%4 == 0 {
			drums.kick(t, 0.95)
		}
		if pos == 4 || pos == 12 {
			drums.snare(t, 0.7)
		}
		if pos%2 == 0 {
			drums.hat(t, 0.12)
		}
		fig := []float64{ch.root, ch.fifth, ch.root + 12, ch.fifth + 12}
		low.strNote(t, beat/4*0.8, note(fig[pos%4], mD3), 3600, 0.18)
		if pos%2 == 0 {
			low.bassNote(t, beat/4*1.8, note(ch.root, mD2), 0.5)
		}
	}
	for _, t := range []float64{42, 43, 44, 45} { // a hit on every cut
		drums.warDrum(t, 0.7)
		drums.snare(t, 0.5)
	}
	for t := 38.0; t < 45.5-1e-9; t += bar {
		ch := chordAt(t)
		d := math.Min(bar+0.3, 45.6-t)
		for k, s := range []float64{ch.root, ch.third, ch.fifth} {
			pads.pad(t, d, note(s, mD3), 0.16-0.02*float64(k))
			pads.choir(t, d, note(s, mD4), 0.1)
		}
	}
	for _, m := range climaxMel {
		t := 38 + m.t*beat
		d := math.Min(m.d*beat*0.95, 45.5-t)
		brass.brassNote(t, d, note(m.n, mD4), 0.26)
		brass.brassNote(t, d, note(m.n-12, mD4), 0.16)
	}
	fx.riser(43.5, 2.0, 1.1)
	fx.reverseSwell(44.4, 1.15, 1.1)
	// 45.5-46 is silence, then everything at once

	// ---- 46: the end card ------------------------------------------
	fx.impact(46, 1.4)
	drums.warDrum(46, 1.1)
	drums.warDrum(46.02, 0.6)
	for _, s := range []float64{0, 7, 12, 15, 19} {
		brass.brassNote(46, 3.2, note(s, mD3), 0.22)
	}
	brass.brassNote(46, 3.2, hz(mD2), 0.3)
	pads.pad(46, 9, hz(mD2), 0.4)
	pads.pad(46, 9, hz(mD3+7), 0.22)
	pads.choir(46, 8.5, hz(mD4), 0.2)
	pads.choir(46, 8.5, hz(mD4+3), 0.16)
	pads.choir(46, 8.5, hz(mD4+7), 0.14)
	for i, n := range []float64{mD5, mD5 + 7, mD5 + 3, mD5 + 12, mD5 + 10, mD5 + 7} {
		fx.glass(47.9+float64(i)*0.5, hz(n), 0.3-0.02*float64(i))
	}
	fx.sparkle(47.8, 3, 24, 2500, 9000, 0.14)
	drums.warDrum(50, 0.35)

	stems := []stem{
		{drums, verbSmall.with(0.9), 0.95, 0},
		{low, verbSmall.with(0.9), 0.62, -0.15},
		{high, verbMid.with(0.9), 0.32, 0.35},
		{pads, verbBig.with(0.9), 0.5, 0},
		{brass, verbMid.with(0.9), 0.6, 0.1},
		{fx, verbBig.with(0.9), 0.7, 0},
	}
	n := int(dur * sampleRate)
	l, r = make([]float64, n), make([]float64, n)
	for _, s := range stems {
		addStereo(l, r, s.b.finish(s.m), 0, s.gain, s.pan)
	}
	return l, r
}

// addStereo mixes interleaved 16-bit stereo into l/r at time t.
func addStereo(l, r []float64, pcm []byte, t, gain, pan float64) {
	i0 := int(t * sampleRate)
	gl := gain * math.Min(1, 1-pan)
	gr := gain * math.Min(1, 1+pan)
	for i := 0; i+3 < len(pcm); i += 4 {
		j := i0 + i/4
		if j < 0 {
			continue
		}
		if j >= len(l) {
			break
		}
		l[j] += float64(int16(binary.LittleEndian.Uint16(pcm[i:]))) / 32768 * gl
		r[j] += float64(int16(binary.LittleEndian.Uint16(pcm[i+2:]))) / 32768 * gr
	}
}

// ------------------------------------------------------------ the mix

// sfxLevel shapes a few sounds against the score.
func sfxLevel(name string) float64 {
	base, _ := parseName(name)
	switch base {
	case "bossintro":
		return 1.25
	case "boom", "roar", "bossdie":
		return 1.1
	case "brk", "deb", "tuf", "wall", "coin", "thump":
		return 0.7
	case "hover", "cardhover":
		return 0.5
	}
	return 0.9
}

func renderSFX(g *Game, ev []sfxEvent, n int) (l, r []float64) {
	l, r = make([]float64, n), make([]float64, n)
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].T < ev[j].T })
	var ends []float64
	for _, e := range ev {
		c := g.snd.clip(e.Name, e.V)
		if c == nil {
			continue
		}
		live := ends[:0]
		for _, x := range ends {
			if x > e.T {
				live = append(live, x)
			}
		}
		ends = live
		base, _ := parseName(e.Name)
		important := base != "brk" && base != "deb" && base != "wall" && base != "coin" && base != "tuf" && base != "thump"
		if len(ends) >= 22 && !important {
			continue
		}
		ends = append(ends, e.T+float64(len(c)/4)/sampleRate)
		addStereo(l, r, c, e.T, e.Vol*sfxLevel(e.Name), 0)
	}
	return
}

func rms(x []float64, t0, t1 float64) float64 {
	i0, i1 := int(t0*sampleRate), min(len(x), int(t1*sampleRate))
	s := 0.0
	for i := i0; i < i1; i++ {
		s += x[i] * x[i]
	}
	return math.Sqrt(s / float64(max(1, i1-i0)))
}

// limit is a look-ahead peak limiter followed by a gentle soft clip.
func limit(l, r []float64, ceiling float64) {
	n := len(l)
	const look = 220 // 5 ms
	gain := make([]float64, n)
	for i := range gain {
		p := math.Max(math.Abs(l[i]), math.Abs(r[i]))
		gain[i] = 1
		if p > ceiling {
			gain[i] = ceiling / p
		}
	}
	// spread each reduction back over the look-ahead and release slowly
	env := make([]float64, n)
	m := 1.0
	for i := n - 1; i >= 0; i-- {
		m = math.Min(gain[i], m+(1-m)*(1.0/look))
		env[i] = m
	}
	g := 1.0
	rel := math.Exp(-1 / (0.08 * sampleRate))
	for i := 0; i < n; i++ {
		if env[i] < g {
			g = env[i]
		} else {
			g = env[i] + (g-env[i])*rel
		}
		l[i] = softClip(l[i] * g)
		r[i] = softClip(r[i] * g)
	}
}

// scoreShape is the score's level over the trailer in dB: a hushed cold
// open, the slam, the build, the held breath before the Guardian, the
// climax, and a beat of silence before the logo.
func scoreShape(t float64) float64 {
	pts := [][2]float64{{0, -10}, {4.0, -8}, {5.9, -4}, {6.0, 0}, {7.9, -1}, {8, -3.5}, {16, -2.5}, {24, -4.5}, {26, -2}, {30, -1}, {32, -1.5},
		{33, -4}, {37, -3}, {38, 0}, {46, 0}, {60, 0}}
	db := 0.0
	for i := 1; i < len(pts); i++ {
		if t <= pts[i][0] {
			a, b := pts[i-1], pts[i]
			k := (t - a[0]) / (b[0] - a[0])
			db = a[1] + (b[1]-a[1])*k
			break
		}
	}
	return math.Pow(10, db/20)
}

// breath is the held breath: total silence just before the end card.
func breath(t float64) float64 {
	g := 1.0
	if t > 45.55 && t < 45.99 {
		switch {
		case t < 45.57:
			g *= (45.57 - t) / 0.02
		case t > 45.97:
			g *= (t - 45.97) / 0.02
		default:
			g = 0
		}
	}
	return g
}

// mixTrailer renders score + sound effects into a 16-bit stereo WAV.
// stems: 0 full mix, 1 score only, 2 effects only.
func mixTrailer(g *Game, ev []sfxEvent, stems int) []byte {
	sl, sr := renderScore()
	n := len(sl)
	fl, fr := renderSFX(g, ev, n)
	// effects sit about 6 dB under the music through the gameplay
	ms := (rms(sl, 8, 32) + rms(sr, 8, 32)) / 2
	fs := (rms(fl, 8, 32) + rms(fr, 8, 32)) / 2
	fg := 1.0
	if fs > 1e-6 {
		fg = ms / fs * math.Pow(10, -6.0/20)
	}
	// a gentle sidechain: the score dips under loud effects
	env := 0.0
	att, rel := math.Exp(-1/(0.005*sampleRate)), math.Exp(-1/(0.25*sampleRate))
	l, r := make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		a := math.Max(math.Abs(fl[i]), math.Abs(fr[i])) * fg
		if a > env {
			env = a + (env-a)*att
		} else {
			env = a + (env-a)*rel
		}
		duck := (1 - 0.3*clamp01(env/0.5)) * scoreShape(float64(i)/sampleRate)
		var ml, mr, el, er float64
		if stems != 2 {
			ml, mr = sl[i]*duck, sr[i]*duck
		}
		if stems != 1 {
			el, er = fl[i]*fg, fr[i]*fg
		}
		bg := breath(float64(i) / sampleRate)
		l[i], r[i] = (ml+el)*bg, (mr+er)*bg
	}
	// normalise to the ceiling, limit, and fade the tail
	pk := 1e-9
	for i := range l {
		pk = math.Max(pk, math.Max(math.Abs(l[i]), math.Abs(r[i])))
	}
	pre := 1.6 / pk // push into the limiter for density
	for i := range l {
		l[i] *= pre
		r[i] *= pre
	}
	limit(l, r, 0.89)
	fade := int(1.5 * sampleRate)
	for i := 0; i < fade && i < n; i++ {
		k := float64(i) / float64(fade)
		l[n-1-i] *= k
		r[n-1-i] *= k
	}
	return wavBytes(l, r)
}

func wavBytes(l, r []float64) []byte {
	var buf bytes.Buffer
	n := len(l)
	w := func(v any) { _ = binary.Write(&buf, binary.LittleEndian, v) }
	buf.WriteString("RIFF")
	w(uint32(36 + n*4))
	buf.WriteString("WAVEfmt ")
	w(uint32(16))
	w(uint16(1))
	w(uint16(2))
	w(uint32(sampleRate))
	w(uint32(sampleRate * 4))
	w(uint16(4))
	w(uint16(16))
	buf.WriteString("data")
	w(uint32(n * 4))
	out := make([]byte, n*4)
	for i := 0; i < n; i++ {
		putS16(out[i*4:], l[i])
		putS16(out[i*4+2:], r[i])
	}
	buf.Write(out)
	return buf.Bytes()
}
