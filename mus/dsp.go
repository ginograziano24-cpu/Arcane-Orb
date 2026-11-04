package mus

import "math"

// DSP building blocks for the streaming music engine (mus_*.go): band-
// limited oscillators, a state-variable filter, envelopes, noise, a
// stereo reverb, a ping-pong delay and the master bus.

// SampleRate is the engine's output rate.
const SampleRate = 44100

const msr = float64(SampleRate)

func midiHz(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

const sinN = 4096

var sinTab = func() []float64 {
	t := make([]float64, sinN+1)
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / sinN)
	}
	return t
}()

// fexp is a fast e^x (relative error ~1e-7), for envelopes.
func fexp(x float64) float64 {
	if x < -700 {
		return 0
	}
	return fpow2(x * 1.4426950408889634)
}

// fpow2 is a fast 2^y.
func fpow2(y float64) float64 {
	i := math.Floor(y)
	f := y - i
	p := 1 + f*(0.6931471805599453+f*(0.24022650695910071+f*(0.0555041086648216+f*(0.009618129107628477+f*0.0013333558146428443))))
	return math.Float64frombits(math.Float64bits(p) + uint64(int64(i))<<52)
}

// ftanh is a fast rational tanh (exact at 0, saturating past ±3).
func ftanh(x float64) float64 {
	if x > 3 {
		return 1
	}
	if x < -3 {
		return -1
	}
	x2 := x * x
	return x * (27 + x2) / (27 + 9*x2)
}

// fsin is a table sine of a phase in cycles (any real value).
func fsin(ph float64) float64 {
	ph -= math.Floor(ph)
	x := ph * sinN
	i := int(x)
	f := x - float64(i)
	return sinTab[i] + (sinTab[i+1]-sinTab[i])*f
}

// polyBLEP residual for a discontinuity at phase 0 (t, dt in cycles).
func blep(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

// osc is a phase accumulator with band-limited waveforms.
type osc struct{ ph, inc float64 }

func (o *osc) setHz(f float64) {
	o.inc = f / msr
	if o.inc > 0.45 {
		o.inc = 0.45
	}
}

func (o *osc) step() {
	o.ph += o.inc
	if o.ph >= 1 {
		o.ph -= 1
	}
}

func (o *osc) saw() float64 {
	v := 2*o.ph - 1 - blep(o.ph, o.inc)
	o.step()
	return v
}

func (o *osc) pulse(pw float64) float64 {
	v := -1.0
	if o.ph < pw {
		v = 1
	}
	v += blep(o.ph, o.inc)
	t := o.ph - pw
	if t < 0 {
		t++
	}
	v -= blep(t, o.inc)
	o.step()
	return v
}

func (o *osc) sin() float64 {
	v := fsin(o.ph)
	o.step()
	return v
}

func (o *osc) tri() float64 {
	v := 4*math.Abs(o.ph-0.5) - 1
	o.step()
	return v
}

// noise is a fast deterministic white noise source.
type noise struct{ s uint32 }

func (n *noise) next() float64 {
	if n.s == 0 {
		n.s = 0x9E3779B9
	}
	n.s ^= n.s << 13
	n.s ^= n.s >> 17
	n.s ^= n.s << 5
	return float64(int32(n.s)) / 2147483648
}

// svf is a topology-preserving state-variable filter (Zavalishin).
type svf struct {
	ic1, ic2      float64
	a1, a2, a3, k float64
	lastF, lastQ  float64
}

func (f *svf) set(fc, q float64) {
	if q == f.lastQ && math.Abs(fc-f.lastF) < 0.003*f.lastF {
		return // inaudible change: skip the tan
	}
	f.lastF, f.lastQ = fc, q
	fc = math.Max(20, math.Min(fc, msr*0.45))
	g := math.Tan(math.Pi * fc / msr)
	f.k = 1 / math.Max(0.3, q)
	f.a1 = 1 / (1 + g*(g+f.k))
	f.a2 = g * f.a1
	f.a3 = g * f.a2
}

// proc returns low-, band- and high-pass outputs.
func (f *svf) proc(v0 float64) (lp, bp, hp float64) {
	v3 := v0 - f.ic2
	v1 := f.a1*f.ic1 + f.a2*v3
	v2 := f.ic2 + f.a2*f.ic1 + f.a3*v3
	f.ic1 = 2*v1 - f.ic1
	f.ic2 = 2*v2 - f.ic2
	return v2, v1, v0 - f.k*v1 - v2
}

func (f *svf) lp(v float64) float64 { l, _, _ := f.proc(v); return l }
func (f *svf) bp(v float64) float64 { _, b, _ := f.proc(v); return b }
func (f *svf) hp(v float64) float64 { _, _, h := f.proc(v); return h }

// onepole is a gentle one-pole lowpass (for smoothing and damping).
type onepole struct{ y, a float64 }

func (o *onepole) setHz(f float64) { o.a = 1 - math.Exp(-2*math.Pi*f/msr) }
func (o *onepole) proc(x float64) float64 {
	o.y += o.a * (x - o.y)
	return o.y
}

// adsr is a gated envelope: linear attack, exponential decay and release.
type adsr struct {
	a, d, s, r float64 // seconds, seconds, level, seconds
	v          float64
	stage      int // 0 attack, 1 decay/sustain, 2 release, 3 done
	dk, rk     float64
	ready      bool
}

func (e *adsr) next(gate bool) float64 {
	const dt = 1 / msr
	if !e.ready {
		e.ready = true
		if e.d > 0 {
			e.dk = math.Exp(-dt / e.d)
		}
		if e.r > 0 {
			e.rk = math.Exp(-dt / e.r)
		}
	}
	if !gate && e.stage < 2 {
		e.stage = 2
	}
	switch e.stage {
	case 0:
		if e.a <= 0 {
			e.v = 1
		} else {
			e.v += dt / e.a
		}
		if e.v >= 1 {
			e.v, e.stage = 1, 1
		}
	case 1:
		e.v = e.s + (e.v-e.s)*e.dk
	case 2:
		e.v *= e.rk
		if e.v < 1e-4 {
			e.v, e.stage = 0, 3
		}
	}
	return e.v
}

func (e *adsr) done() bool { return e.stage == 3 }

// decayCoef is the per-sample multiplier for an exponential decay with
// time constant tau seconds.
func decayCoef(tau float64) float64 { return math.Exp(-1 / (tau * msr)) }

// ---------------------------------------------------------------- effects

type fvComb struct {
	buf      []float64
	i        int
	fb, d, s float64
}

func (c *fvComb) proc(x float64) float64 {
	y := c.buf[c.i]
	c.s = y*(1-c.d) + c.s*c.d
	c.buf[c.i] = x + c.s*c.fb
	c.i++
	if c.i == len(c.buf) {
		c.i = 0
	}
	return y
}

type fvAll struct {
	buf []float64
	i   int
}

func (a *fvAll) proc(x float64) float64 {
	b := a.buf[a.i]
	a.buf[a.i] = x + b*0.5
	a.i++
	if a.i == len(a.buf) {
		a.i = 0
	}
	return b - x
}

// reverb is a stereo Freeverb.
type reverb struct {
	cl, cr [8]fvComb
	al, ar [4]fvAll
	hp     [2]svf
}

func newReverb(room, damp float64) *reverb {
	combs := []int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617}
	alls := []int{556, 441, 341, 225}
	r := &reverb{}
	fb := 0.7 + 0.28*room
	for i, n := range combs {
		r.cl[i] = fvComb{buf: make([]float64, n), fb: fb, d: damp}
		r.cr[i] = fvComb{buf: make([]float64, n+23), fb: fb, d: damp}
	}
	for i, n := range alls {
		r.al[i] = fvAll{buf: make([]float64, n)}
		r.ar[i] = fvAll{buf: make([]float64, n+23)}
	}
	r.hp[0].set(180, 0.7)
	r.hp[1].set(180, 0.7)
	return r
}

func (r *reverb) proc(x float64) (float64, float64) {
	x *= 0.015
	var l, rr float64
	for i := range r.cl {
		l += r.cl[i].proc(x)
		rr += r.cr[i].proc(x)
	}
	for i := range r.al {
		l = r.al[i].proc(l)
		rr = r.ar[i].proc(rr)
	}
	return r.hp[0].hp(l), r.hp[1].hp(rr)
}

// pingpong is a tempo-synced stereo delay with a darkening feedback path.
type pingpong struct {
	l, r []float64
	i    int
	fb   float64
	damp [2]onepole
	hp   [2]svf
}

func newPingPong(secs, fb float64) *pingpong {
	n := max(1, int(secs*msr))
	p := &pingpong{l: make([]float64, n), r: make([]float64, n), fb: fb}
	p.damp[0].setHz(3500)
	p.damp[1].setHz(3500)
	p.hp[0].set(250, 0.7)
	p.hp[1].set(250, 0.7)
	return p
}

func (p *pingpong) proc(x float64) (float64, float64) {
	yl, yr := p.l[p.i], p.r[p.i]
	p.l[p.i] = p.hp[0].hp(p.damp[0].proc(x + yr*p.fb))
	p.r[p.i] = p.hp[1].hp(p.damp[1].proc(yl * p.fb))
	p.i++
	if p.i == len(p.l) {
		p.i = 0
	}
	return yl, yr
}

// busComp is a gentle feed-forward stereo-linked compressor followed by a
// soft clipper; it glues the mix and keeps peaks under full scale.
type busComp struct {
	env, thr, ratio, att, rel float64
	g                         float64
	k                         int
}

func newBusComp() *busComp {
	return &busComp{thr: 0.35, ratio: 2.5, att: decayCoef(0.004), rel: decayCoef(0.15)}
}

func (c *busComp) proc(l, r float64) (float64, float64) {
	p := math.Max(math.Abs(l), math.Abs(r))
	if p > c.env {
		c.env = p + (c.env-p)*c.att
	} else {
		c.env = p + (c.env-p)*c.rel
	}
	if c.k&7 == 0 {
		c.g = 1
		if c.env > c.thr {
			c.g = math.Pow(c.env/c.thr, 1/c.ratio-1)
		}
	}
	c.k++
	return softSat(l * c.g), softSat(r * c.g)
}

// softSat is linear up to 0.6 and rounds smoothly into ±0.97.
func softSat(x float64) float64 {
	a := math.Abs(x)
	if a <= 0.6 {
		return x
	}
	y := 0.6 + 0.37*ftanh((a-0.6)/0.37)
	if x < 0 {
		return -y
	}
	return y
}
