package mus

import "math"

// Instruments for the music engine. Every sound is synthesised: drums from
// pitched sines, metallic square clusters and filtered noise; basses, leads
// and pads from band-limited oscillators through state-variable filters;
// the electric piano and bells by FM.

type instKind int

const (
	iKick instKind = iota
	iSnare
	iClap
	iHat
	iRide
	iCrash
	iShaker
	iTom
	iCowbell
	iRim
	iScratch
	iAcid
	iFunkBass
	iReese
	iSub
	iSuper
	iLead
	iPluck
	iEPiano
	iBrass
	iBell
	iRiser
	iDown
	iImpact
)

// inst holds an instrument's settings; each kind reads the fields it uses.
type inst struct {
	kind            instKind
	f0, f1, pd      float64 // drum pitch: start, end, sweep time
	decay, click    float64
	drive, tone     float64
	cut, res, env   float64 // filter: base cutoff, resonance, envelope (octaves)
	fdec            float64 // filter envelope decay
	a, d, s, r      float64 // amplitude envelope
	detune          float64 // supersaw spread (semitones)
	voices          int
	glide           float64 // portamento time (s)
	vib             float64 // vibrato depth (semitones)
	pw, sub, bright float64
	level           float64
}

// voice is one sounding note.
type voice struct {
	in        *inst
	delay     int // samples before it starts in the current block
	age       int
	gate      int // gate length in samples
	vel       float64
	key       float64 // MIDI note
	freq, tgt float64
	gl        float64 // glide coefficient
	o         [8]osc
	pan       [8]float64
	ratio     [8]float64
	f         [4]svf
	e, fe     adsr
	n, n2     noise
	ph        float64
	fade      float64 // 1 normally; decays when choked
	choked    bool
	retrig    int // age at the last legato retrigger
	x         [4]float64
	dead      bool
}

var voiceSeed uint32 = 12345

func newVoice(in *inst, key, vel float64, gate int) *voice {
	v := &voice{in: in, key: key, vel: vel, gate: gate, fade: 1}
	v.freq = midiHz(key)
	v.tgt = v.freq
	voiceSeed = voiceSeed*1664525 + 1013904223
	v.n.s = voiceSeed | 1
	v.n2.s = (voiceSeed >> 7) ^ 0x5bd1e995
	v.e = adsr{a: in.a, d: in.d, s: in.s, r: in.r}
	v.fe = adsr{a: 0.0005, d: in.fdec, s: 0, r: in.fdec}
	switch in.kind {
	case iSuper:
		nv := max(1, min(8, in.voices))
		for i := 0; i < nv; i++ {
			u := 0.0
			if nv > 1 {
				u = float64(i)/float64(nv-1)*2 - 1
			}
			v.o[i].ph = v.n.next()*0.5 + 0.5
			v.pan[i] = u
			v.ratio[i] = fpow2(in.detune * u / 12)
		}
	case iBrass:
		for i := range v.o[:4] {
			v.o[i].ph = v.n.next()*0.5 + 0.5
		}
	}
	return v
}

// legato retunes a mono voice to a new note without restarting it.
func (v *voice) legato(key, vel float64, gate int, slide bool) {
	v.key = key
	v.tgt = midiHz(key)
	if !slide {
		v.freq = v.tgt
		v.fe = adsr{a: 0.0005, d: v.in.fdec, s: 0, r: v.in.fdec}
		v.retrig = v.age
	}
	v.vel = vel
	v.gate = v.age + gate
	if v.e.stage >= 2 {
		v.e.stage = 1
	}
}

func (v *voice) choke() { v.choked = true }

// tailCut is where a released note stops ringing (-40 dB): below the mix,
// and it keeps long pad and stab tails from piling up voices.
const tailCut = 0.01

// render adds the voice into l and r for n samples; false when finished.
func (v *voice) render(l, r []float64, n int) bool {
	if v.dead {
		return false
	}
	s := 0
	if v.delay > 0 {
		if v.delay >= n {
			v.delay -= n
			return true
		}
		s = v.delay
		v.delay = 0
	}
	switch v.in.kind {
	case iKick, iTom:
		v.kick(l, r, s, n)
	case iSnare:
		v.snare(l, r, s, n)
	case iClap:
		v.clap(l, r, s, n)
	case iHat, iRide, iCrash, iCowbell:
		v.metal(l, r, s, n)
	case iShaker, iRim:
		v.click(l, r, s, n)
	case iScratch:
		v.scratch(l, r, s, n)
	case iAcid, iFunkBass, iReese, iSub, iLead:
		v.mono(l, r, s, n)
	case iSuper:
		v.super(l, r, s, n)
	case iPluck, iBrass:
		v.synth(l, r, s, n)
	case iEPiano, iBell:
		v.fm(l, r, s, n)
	case iRiser, iDown, iImpact:
		v.fx(l, r, s, n)
	}
	if v.age > int(12*msr) {
		v.dead = true
	}
	return !v.dead
}

// out writes one stereo sample, applying velocity, level and choke fade.
func (v *voice) out(l, r []float64, i int, a, b float64) {
	g := v.vel * v.in.level * v.fade
	if v.choked {
		v.fade *= 0.992
		if v.fade < 1e-3 {
			v.dead = true
		}
	}
	l[i] += a * g
	r[i] += b * g
}

func (v *voice) t() float64 { return float64(v.age) / msr }

// ---------------------------------------------------------------- drums

func (v *voice) kick(l, r []float64, s, n int) {
	in := v.in
	f0, f1 := in.f0, in.f1
	if in.kind == iTom {
		f1 = v.freq
		f0 = v.freq * 1.7
	}
	dk := decayCoef(in.decay)
	pk := decayCoef(in.pd)
	if v.age == 0 {
		v.x[3] = 1
	}
	for i := s; i < n; i++ {
		t := v.t()
		f := f1 + (f0-f1)*v.x[3]
		v.x[3] *= pk
		v.ph += f / msr
		if v.ph > 1 {
			v.ph--
		}
		if v.age == 0 {
			v.e.v = 1
		}
		v.e.v *= dk
		att := math.Min(1, t/0.0006)
		y := fsin(v.ph) * v.e.v * att
		if t < 0.005 {
			if v.age == 0 {
				v.f[0].set(1500, 0.7)
			}
			y += v.f[0].hp(v.n.next()) * in.click * (1 - t/0.005)
		}
		if in.drive > 0 {
			y = ftanh(y*in.drive) / ftanh(in.drive)
		}
		v.out(l, r, i, y, y)
		v.age++
		if v.e.v < 2e-4 {
			v.dead = true
			return
		}
	}
}

func (v *voice) snare(l, r []float64, s, n int) {
	in := v.in
	if v.age == 0 {
		v.f[0].set(in.cut, 0.8)
		v.f[1].set(in.tone, 1.2)
	}
	for i := s; i < n; i++ {
		t := v.t()
		bend := 1 + 0.25*fexp(-t/0.01)
		v.o[0].setHz(in.f0 * bend)
		v.o[1].setHz(in.f0 * 1.72 * bend)
		tone := (v.o[0].sin() + 0.5*v.o[1].sin()) * fexp(-t/0.055)
		nz := v.n.next()
		body := v.f[0].hp(nz)
		snap := v.f[1].bp(nz)
		ne := fexp(-t / in.decay)
		y := 0.55*tone + (0.7*body+0.5*snap)*ne
		y *= math.Min(1, t/0.0008)
		y = ftanh(y*1.4) / 1.2
		v.out(l, r, i, y, y)
		v.age++
		if t > in.decay*8 {
			v.dead = true
			return
		}
	}
}

func (v *voice) clap(l, r []float64, s, n int) {
	in := v.in
	if v.age == 0 {
		v.f[0].set(in.tone, 1.3)
		v.f[1].set(in.tone, 1.3)
	}
	for i := s; i < n; i++ {
		t := v.t()
		e := 0.0
		for k := 0; k < 3; k++ {
			if d := t - float64(k)*0.0095; d >= 0 {
				e += fexp(-d/0.0035) * (1 - 0.15*float64(k))
			}
		}
		if d := t - 0.028; d >= 0 {
			e += 0.75 * fexp(-d/in.decay)
		}
		a := v.f[0].bp(v.n.next()) * e * 1.6
		b := v.f[1].bp(v.n2.next()) * e * 1.6
		v.out(l, r, i, a, b)
		v.age++
		if t > 0.03+in.decay*7 {
			v.dead = true
			return
		}
	}
}

// metal covers hats, ride, crash and cowbell: square clusters through a
// band-pass, with noise for the cymbals.
var hatRatios = [6]float64{1, 1.342, 1.2312, 1.6532, 1.9523, 2.1523}

func (v *voice) metal(l, r []float64, s, n int) {
	in := v.in
	k := 6
	if in.kind == iCowbell {
		k = 2
	}
	if v.age == 0 {
		for j := 0; j < k; j++ {
			ratio := hatRatios[j]
			if in.kind == iCowbell {
				ratio = []float64{1, 1.48}[j]
			}
			v.o[j].inc = in.f0 * ratio / msr
			v.o[j].ph = v.n.next()*0.5 + 0.5
		}
		v.f[0].set(in.tone, 1.1)
		v.f[1].set(in.cut, 0.7)
		v.f[2].set(in.cut, 0.7)
		v.x[0], v.x[1] = 1, 1
	}
	dk := decayCoef(in.decay)
	k05, k35 := decayCoef(0.05), decayCoef(0.35)
	mix := in.bright // noise share
	for i := s; i < n; i++ {
		var m float64
		for j := 0; j < k; j++ {
			o := &v.o[j]
			if o.ph < 0.5 {
				m++
			} else {
				m--
			}
			o.ph += o.inc
			if o.ph >= 1 {
				o.ph--
			}
		}
		m /= float64(k)
		if v.age == 0 {
			v.e.v = 1
		}
		v.e.v *= dk
		e := v.e.v
		switch in.kind {
		case iCowbell:
			v.x[0] *= k05
			y := v.f[0].bp(m) * (0.6*v.x[0] + 0.4*e) * 2.5
			v.out(l, r, i, y, y)
		case iCrash:
			e *= math.Min(1, float64(v.age)/(0.004*msr))
			body := v.f[0].bp(m * (1 - mix))
			a := v.f[1].hp(body + v.n.next()*mix)
			b := v.f[2].hp(body + v.n2.next()*mix)
			v.out(l, r, i, a*e*3, b*e*3)
		default:
			y := v.f[1].hp(v.f[0].bp(m*(1-mix) + v.n.next()*mix))
			if in.kind == iRide {
				v.x[1] *= k35
				y += fsin(in.f0*7.3*float64(v.age)/msr) * v.x[1] * 0.12
			}
			y *= e * 3
			v.out(l, r, i, y, y)
		}
		v.age++
		if v.e.v < 1.5e-3 {
			v.dead = true
			return
		}
	}
}

// click covers the shaker and the rim shot.
func (v *voice) click(l, r []float64, s, n int) {
	in := v.in
	if v.age == 0 {
		v.f[0].set(in.tone, 1.6)
	}
	for i := s; i < n; i++ {
		t := v.t()
		var y float64
		if in.kind == iShaker {
			e := math.Min(1, t/0.006) * fexp(-math.Max(0, t-0.006)/in.decay)
			y = v.f[0].bp(v.n.next()) * e * 2
		} else {
			y = fsin(in.f0*t)*fexp(-t/0.011) + v.f[0].bp(v.n.next())*fexp(-t/0.005)*1.5
		}
		v.out(l, r, i, y, y)
		v.age++
		if t > in.decay*8+0.01 {
			v.dead = true
			return
		}
	}
}

// scratch moves a virtual record back and forth under a vocal "ah": the
// playback rate scales the pitch and formants, the fader chops it. The
// note's key picks the move: 0 baby scratch, 1 chirp, 2 transformer.
func (v *voice) scratch(l, r []float64, s, n int) {
	in := v.in
	D := float64(max(1, v.gate)) / msr
	move := int(v.key) % 3
	for i := s; i < n; i++ {
		t := v.t()
		u := t / D
		if u >= 1 {
			v.dead = true
			return
		}
		rate := 2.2 * fsin(u)
		fader := 1.0
		switch move {
		case 1:
			if rate < 0 {
				fader = 0
			}
		case 2:
			if int(u*12)%2 == 1 {
				fader = 0.05
			}
		}
		ar := math.Abs(rate)
		v.o[0].setHz(in.f0 * ar)
		src := v.o[0].saw() + 0.3*v.n.next()
		v.f[0].set(750*ar+60, 4)
		v.f[1].set(1250*ar+90, 5)
		y := (v.f[0].bp(src) + 0.7*v.f[1].bp(src)) * math.Min(1, ar*1.3) * fader
		y *= math.Min(1, t/0.002) * math.Min(1, (D-t)/0.004)
		v.out(l, r, i, y, y)
		v.age++
	}
}

// ---------------------------------------------------------------- melodic

func (v *voice) vibrato(t float64) float64 {
	if v.in.vib <= 0 {
		return 1
	}
	ramp := math.Min(1, math.Max(0, (t-0.18)/0.3))
	return 1 + 0.05776*v.in.vib*ramp*fsin(5.6*t)
}

// mono covers the monophonic basses and the lead, with glide.
func (v *voice) mono(l, r []float64, s, n int) {
	in := v.in
	gl := 1.0
	if in.glide > 0 {
		gl = 1 - fexp(-1/(in.glide*msr))
	}
	for i := s; i < n; i++ {
		t := v.t()
		v.freq += (v.tgt - v.freq) * gl
		gate := v.age < v.gate
		amp := v.e.next(gate)
		if v.e.done() {
			v.dead = true
			return
		}
		if v.e.stage == 2 && amp < tailCut && !v.choked {
			v.choke() // released to -40 dB: fade the rest out quickly
		}
		fe := v.fe.next(gate && v.age-v.retrig < 88)
		acc := 1.0
		if v.vel > 0.92 {
			acc = 1.6
		}
		f := v.freq * v.vibrato(t)
		var x float64
		switch in.kind {
		case iAcid:
			v.o[0].setHz(f)
			if in.pw > 0 {
				x = v.o[0].pulse(in.pw)
			} else {
				x = v.o[0].saw()
			}
			if v.age&3 == 0 {
				c := in.cut * fpow2(in.env*fe*acc)
				v.f[0].set(c, 0.7)
				v.f[1].set(c, in.res)
			}
			x = v.f[1].lp(v.f[0].lp(x))
			x = ftanh(x*in.drive) * 0.8
		case iFunkBass:
			v.o[0].setHz(f)
			v.o[1].setHz(f * 1.004)
			v.o[2].setHz(f)
			x = 0.55*v.o[0].pulse(0.38) + 0.4*v.o[1].saw()
			if v.age&3 == 0 {
				c := in.cut * fpow2(in.env*fe*(0.5+0.5*v.vel*acc))
				v.f[0].set(c, in.res)
			}
			x = v.f[0].lp(x) + in.sub*v.o[2].sin()
			if tt := float64(v.age-v.retrig) / msr; tt < 0.006 {
				v.f[1].set(2600, 1.5)
				x += v.f[1].bp(v.n.next()) * (1 - tt/0.006) * 0.6 * v.vel
			}
			x = ftanh(x*1.6) * 0.75
		case iReese:
			v.o[0].setHz(f * 0.9965)
			v.o[1].setHz(f * 1.0035)
			v.o[2].setHz(f)
			x = 0.5*(v.o[0].saw()+v.o[1].saw()) + in.sub*v.o[2].sin()
			if v.age&3 == 0 {
				c := in.cut * (1 + 0.5*fsin(0.25*t)) * fpow2(in.env*fe)
				v.f[0].set(c, in.res)
			}
			x = ftanh(v.f[0].lp(x)*in.drive) * 0.7
		case iSub:
			v.o[0].setHz(f)
			x = ftanh(v.o[0].sin()*in.drive) / ftanh(in.drive)
		case iLead:
			v.o[0].setHz(f)
			v.o[1].setHz(f * 2.002)
			pw := in.pw + 0.1*fsin(0.8*t)
			x = v.o[0].pulse(pw) + in.bright*v.o[1].saw()
			if v.age&3 == 0 {
				c := in.cut * fpow2(in.env*fe)
				v.f[0].set(c, in.res)
			}
			x = v.f[0].lp(x) * 0.6
		}
		y := x * amp
		v.out(l, r, i, y, y)
		v.age++
	}
}

// super is the supersaw: detuned saws spread across the stereo field.
func (v *voice) super(l, r []float64, s, n int) {
	in := v.in
	nv := max(1, min(8, in.voices))
	norm := 1 / math.Sqrt(float64(nv))
	for i := s; i < n; i++ {
		t := v.t()
		gate := v.age < v.gate
		amp := v.e.next(gate)
		if v.e.done() {
			v.dead = true
			return
		}
		if v.e.stage == 2 && amp < tailCut && !v.choked {
			v.choke() // released to -40 dB: fade the rest out quickly
		}
		fe := v.fe.next(gate && v.age < 88)
		f := v.freq * v.vibrato(t)
		if v.age&7 == 0 {
			for j := 0; j < nv; j++ {
				v.o[j].setHz(f * v.ratio[j])
			}
			c := in.cut * fpow2(in.env*fe)
			v.f[0].set(c, in.res)
			v.f[1].set(c, in.res)
		}
		var a, b float64
		for j := 0; j < nv; j++ {
			x := v.o[j].saw()
			p := v.pan[j]
			a += x * (1 - 0.6*p)
			b += x * (1 + 0.6*p)
		}
		a, b = a*norm, b*norm
		if in.sub > 0 {
			v.o[7].setHz(f * 0.5)
			sub := v.o[7].sin() * in.sub
			a += sub
			b += sub
		}
		a = v.f[0].lp(a) * amp * 0.5
		b = v.f[1].lp(b) * amp * 0.5
		v.out(l, r, i, a, b)
		v.age++
	}
}

// synth covers plucks and brass.
func (v *voice) synth(l, r []float64, s, n int) {
	in := v.in
	for i := s; i < n; i++ {
		t := v.t()
		gate := v.age < v.gate
		amp := v.e.next(gate)
		if v.e.done() {
			v.dead = true
			return
		}
		if v.e.stage == 2 && amp < tailCut && !v.choked {
			v.choke() // released to -40 dB: fade the rest out quickly
		}
		var x float64
		switch in.kind {
		case iPluck:
			if v.age&7 == 0 {
				v.o[0].setHz(v.freq)
				v.o[1].setHz(v.freq * 1.006)
				c := in.cut * fpow2(in.env*fexp(-t/in.fdec))
				v.f[0].set(c, in.res)
			}
			x = v.f[0].lp(0.6*v.o[0].saw() + 0.5*v.o[1].pulse(0.5))
		case iBrass:
			scoop := 1 - 0.025*fexp(-t/0.035)
			if v.age&7 == 0 {
				for j := 0; j < 3; j++ {
					v.o[j].setHz(v.freq * scoop * fpow2(float64(j-1)*0.11/12))
				}
				v.o[3].setHz(v.freq * scoop * 0.5)
				fe := math.Min(1, t/0.035) * (0.4 + 0.6*fexp(-t/in.fdec))
				c := in.cut * fpow2(in.env*fe*(0.6+0.4*v.vel))
				v.f[0].set(c, in.res)
			}
			x = v.f[0].lp((v.o[0].saw()+v.o[1].saw()+v.o[2].saw())*0.4 + 0.25*v.o[3].pulse(0.5))
		}
		y := x * amp * 0.7
		v.out(l, r, i, y, y)
		v.age++
	}
}

// fm covers the electric piano (with tine bark and stereo tremolo) and bells.
func (v *voice) fm(l, r []float64, s, n int) {
	in := v.in
	inc := v.freq / msr
	const tau = 1 / (2 * math.Pi)
	if v.age == 0 {
		v.x[0], v.x[1] = 1, 1 // index and bark envelopes
	}
	k0, k1 := decayCoef(0.4), decayCoef(0.06)
	if in.kind == iBell {
		k0 = decayCoef(0.12)
	}
	for i := s; i < n; i++ {
		gate := v.age < v.gate
		amp := v.e.next(gate)
		if v.e.done() {
			v.dead = true
			return
		}
		if v.e.stage == 2 && amp < tailCut && !v.choked {
			v.choke() // released to -40 dB: fade the rest out quickly
		}
		ph := v.ph
		v.ph += inc
		if v.ph > 1e6 {
			v.ph -= 1e6
		}
		v.x[0] *= k0
		if in.kind == iEPiano {
			v.x[1] *= k1
			m := fsin(ph)
			idx := (0.8+1.6*v.vel)*v.x[0] + 0.3
			y := fsin(ph + idx*m*tau)
			if v.x[1] > 1e-3 {
				y += v.vel * v.vel * 0.35 * v.x[1] * fsin(7*ph+1.2*m*tau)
			}
			y *= amp * 0.5
			v.x[2] += 4.2 / msr
			trem := 0.22 * fsin(v.x[2])
			v.out(l, r, i, y*(1+trem), y*(1-trem))
		} else {
			idx := 3*v.x[0] + 0.4
			y := fsin(ph+idx*fsin(3.5*ph)*tau) * amp * 0.5
			v.out(l, r, i, y, y)
		}
		v.age++
	}
}

func (v *voice) fx(l, r []float64, s, n int) {
	in := v.in
	D := float64(max(1, v.gate)) / msr
	for i := s; i < n; i++ {
		t := v.t()
		var a, b float64
		switch in.kind {
		case iRiser, iDown:
			u := t / D
			if u >= 1 {
				v.dead = true
				return
			}
			if in.kind == iDown {
				u = 1 - u
			}
			if v.age&3 == 0 {
				fc := 250 * fpow2(5.2*u)
				v.f[0].set(fc, 2.2)
				v.f[1].set(fc*1.07, 2.2)
				v.o[0].setHz(v.freq * fpow2(2*u))
			}
			e := u * u
			if in.kind == iDown {
				e = math.Pow(u, 1.5)
			}
			saw := v.o[0].saw() * 0.25 * in.tone
			a = (v.f[0].bp(v.n.next()) + saw) * e
			b = (v.f[1].bp(v.n2.next()) + saw) * e
			fadeIn := math.Min(1, t/0.02)
			a *= fadeIn
			b *= fadeIn
		case iImpact:
			if t > 2.5 {
				v.dead = true
				return
			}
			v.ph += (32 + 40*fexp(-t/0.12)) / msr
			sub := fsin(v.ph) * fexp(-t/0.9)
			if v.age&3 == 0 {
				v.f[0].set(150+7000*fexp(-t/0.08), 0.8)
				v.f[1].set(150+7000*fexp(-t/0.08), 0.8)
			}
			ne := fexp(-t / 0.35)
			a = ftanh(sub*1.6) + v.f[0].lp(v.n.next())*ne*0.8
			b = ftanh(sub*1.6) + v.f[1].lp(v.n2.next())*ne*0.8
		}
		v.out(l, r, i, a, b)
		v.age++
	}
}
