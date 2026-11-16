package main

import (
	"bytes"
	"math"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// World ambience: a quiet, seamless 24-second stereo bed per world, under
// the music while a chamber is being played. Each bed is synthesised the
// first time its world is entered (on a worker goroutine) and cached:
//
//	Glimmer Grove  night crickets, a slow glassy shimmer, a distant owl
//	Meadow         breeze through grass, leaf rustles, scattered birdsong
//	Caverns        low earth rumble, echoing water drips, a far draught
//	Ruins          desert wind in gusts, sand hiss, a circling hawk's cry
//	Clouds         high thin wind, wind-chimes, a far gull
//	Machine Works  mains hum, a slow clank-and-hiss cycle, steam
//	Dreamscape     a soft major shimmer, music-box twinkles, lake lapping
//
// Volume follows the SFX slider and fades with the game state.

const ambLen = 24.0 // seconds; every periodic element divides this

type ambPlayer struct {
	mu      sync.Mutex
	world   int
	p       *audio.Player
	vol     float64
	cache   map[int][]byte
	pending map[int]bool
}

var amb = ambPlayer{world: -1, cache: map[int][]byte{}, pending: map[int]bool{}}

// SetAmbience fades the bed for world in (on) or out. Call every frame.
func (s *Sounds) SetAmbience(world int, on bool, dt float64) {
	if s == nil || s.ctx == nil {
		return
	}
	amb.mu.Lock()
	defer amb.mu.Unlock()
	if world != amb.world && amb.p != nil {
		// a new world: let the old bed go (the change happens between
		// chambers, where the bed is already faded down)
		_ = amb.p.Close()
		amb.p = nil
		amb.vol = 0
	}
	amb.world = world
	if amb.p == nil && on && world >= 0 && world < NumWorlds {
		if data, ok := amb.cache[world]; ok {
			p, err := s.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data))))
			if err == nil {
				amb.p = p
				p.SetVolume(0)
				p.Play()
			}
		} else if !amb.pending[world] {
			amb.pending[world] = true
			go func(w int) {
				data := renderAmbience(w)
				amb.mu.Lock()
				amb.cache[w] = data
				delete(amb.pending, w)
				amb.mu.Unlock()
			}(world)
		}
	}
	if amb.p == nil {
		return
	}
	target := 0.0
	if on {
		target = 1
	}
	amb.vol += (target - amb.vol) * math.Min(1, dt*1.4)
	v := s.SFX * 0.5 * amb.vol * ambGain[world]
	amb.p.SetVolume(v)
	if !on && amb.vol < 0.005 && amb.p.IsPlaying() {
		amb.p.Pause()
	} else if on && !amb.p.IsPlaying() {
		amb.p.Play()
	}
}

var ambGain = [NumWorlds]float64{WNeon: 0.9, WMeadow: 1, WCaverns: 1, WRuins: 0.9, WClouds: 0.85, WMachine: 0.5, WDream: 0.85}

// ---------------------------------------------------------------- synthesis

type ambBuf struct {
	l, r []float64
	rng  *rand.Rand
}

func (a *ambBuf) n() int { return len(a.l) }

// add mixes mono f(t) into the bed at pan (-1 left .. 1 right), wrapping
// around the loop point so every event loops seamlessly.
func (a *ambBuf) add(start, dur, pan float64, f func(t float64) float64) {
	n := a.n()
	i0 := int(start * sampleRate)
	cnt := int(dur * sampleRate)
	gl := math.Sqrt(0.5 * (1 - pan))
	gr := math.Sqrt(0.5 * (1 + pan))
	for i := 0; i < cnt; i++ {
		j := ((i0+i)%n + n) % n
		v := f(float64(i) / sampleRate)
		a.l[j] += v * gl
		a.r[j] += v * gr
	}
}

// bed adds a continuous filtered noise layer whose level breathes with
// slow sines that complete whole cycles within the loop.
func (a *ambBuf) bed(kind int, cut, res, gain float64, cycles []float64, depth float64, cutSwing float64) {
	var ql, qr biquad
	n := a.n()
	// prime the filters with a second of audio so the loop seam is clean
	for i := -sampleRate; i < n; i++ {
		t := float64((i+n)%n) / sampleRate
		m := 1.0
		c := cut
		for k, cy := range cycles {
			ph := 2 * math.Pi * cy * t / ambLen
			m += depth * math.Sin(ph+float64(k)*1.7) / float64(len(cycles))
			c *= 1 + cutSwing*math.Sin(ph*1.3+float64(k))/float64(len(cycles))
		}
		if i%32 == 0 {
			ql.set(kind, c, res)
			qr.set(kind, c*1.03, res)
		}
		xl := ql.proc(a.rng.Float64()*2-1) * gain * math.Max(0, m)
		xr := qr.proc(a.rng.Float64()*2-1) * gain * math.Max(0, m)
		if i >= 0 {
			a.l[i] += xl
			a.r[i] += xr
		}
	}
}

// tone is a pure partial with an envelope.
func (a *ambBuf) tone(start, dur, pan, f, gain float64, env func(t float64) float64) {
	a.add(start, dur, pan, func(t float64) float64 { return math.Sin(2*math.Pi*f*t) * gain * env(t) })
}

func envAD(att, dec float64) func(float64) float64 {
	return func(t float64) float64 {
		if t < att {
			return t / att
		}
		return math.Exp(-(t - att) / dec)
	}
}

// chirp is one bird note: a fast pitch glide with vibrato.
func (a *ambBuf) chirp(start, pan, f0, f1, dur, gain float64) {
	ph := 0.0
	a.add(start, dur+0.02, pan, func(t float64) float64 {
		u := math.Min(1, t/dur)
		f := f0 + (f1-f0)*u + 120*math.Sin(2*math.Pi*38*t)
		ph += f / sampleRate
		e := math.Sin(math.Pi * u)
		return math.Sin(2*math.Pi*ph) * gain * e * e
	})
}

// song is a short phrase of chirps, one bird.
func (a *ambBuf) song(start, pan, base, gain float64) {
	t := start
	notes := 3 + a.rng.Intn(5)
	for k := 0; k < notes; k++ {
		f0 := base * (0.85 + 0.4*a.rng.Float64())
		f1 := f0 * (0.7 + 0.7*a.rng.Float64())
		d := 0.05 + 0.09*a.rng.Float64()
		a.chirp(t, pan, f0, f1, d, gain)
		t += d + 0.03 + 0.08*a.rng.Float64()
	}
}

// drip is a water drop: a plunk whose pitch rises, with echoes.
func (a *ambBuf) drip(start, pan, f, gain float64) {
	for e := 0; e < 4; e++ {
		g := gain * math.Pow(0.42, float64(e))
		ph := 0.0
		a.add(start+float64(e)*0.23, 0.25, pan*(1-0.3*float64(e)), func(t float64) float64 {
			ph += f * (1 + 1.6*math.Min(1, t/0.05)) / sampleRate
			return math.Sin(2*math.Pi*ph) * g * math.Exp(-t/0.045)
		})
	}
}

// chime is a struck bar: inharmonic partials ringing down.
func (a *ambBuf) chime(start, pan, f, gain float64) {
	for k, r := range []float64{1, 2.76, 5.4} {
		g := gain / float64(k+1)
		a.tone(start, 3, pan, f*r, g, envAD(0.002, 1.4/float64(k+1)))
	}
}

func (a *ambBuf) crickets(gain, f float64, rate float64, pan float64) {
	// pulse trains: chirp groups of 3 pulses, repeating
	period := ambLen / math.Round(ambLen*rate)
	for t0 := a.rng.Float64() * period; t0 < ambLen; t0 += period {
		for k := 0; k < 3; k++ {
			s := t0 + float64(k)*0.045
			a.tone(s, 0.035, pan, f, gain, func(t float64) float64 { return math.Sin(math.Pi * t / 0.035) })
		}
	}
}

func renderAmbience(w int) []byte {
	n := int(ambLen * sampleRate)
	a := &ambBuf{l: make([]float64, n), r: make([]float64, n), rng: rand.New(rand.NewSource(int64(4100 + w)))}
	R := a.rng.Float64
	switch w {
	case WMeadow:
		a.bed(fLP, 700, 0.7, 0.10, []float64{1, 3, 5}, 0.9, 0.5) // breeze
		a.bed(fBP, 3200, 1.2, 0.025, []float64{2, 7}, 1.2, 0.3)  // grass hiss
		for k := 0; k < 9; k++ {                                 // leaf rustles
			s := R() * ambLen
			a.add(s, 1.2, R()*2-1, func() func(float64) float64 {
				var q biquad
				q.set(fBP, 2400+1800*R(), 1.5)
				return func(t float64) float64 { return q.proc(R()*2-1) * 0.06 * math.Sin(math.Pi*t/1.2) }
			}())
		}
		birds := []float64{3200, 4100, 2600, 5200}
		for k := 0; k < 14; k++ {
			a.song(R()*ambLen, R()*1.6-0.8, birds[k%len(birds)], 0.05+0.04*R())
		}
	case WNeon:
		a.bed(fLP, 400, 0.7, 0.05, []float64{1, 2}, 0.6, 0.3)
		a.crickets(0.022, 4600, 1.6, -0.6)
		a.crickets(0.018, 5100, 2.1, 0.5)
		a.crickets(0.012, 4300, 1.2, 0.1)
		// a slow glassy shimmer: a fifth that swells twice per loop
		for _, f := range []float64{440, 660, 880, 1320} {
			a.add(0, ambLen, 0, func(f float64) func(float64) float64 {
				return func(t float64) float64 {
					return math.Sin(2*math.Pi*f*t) * 0.012 * (0.5 + 0.5*math.Sin(2*math.Pi*2*t/ambLen+f))
				}
			}(f))
		}
		for _, s := range []float64{5, 17} { // an owl
			for k := 0; k < 2; k++ {
				a.tone(s+float64(k)*0.55, 0.5, -0.4, 380, 0.05, func(t float64) float64 { return math.Sin(math.Pi*t/0.5) * math.Sin(math.Pi*t/0.5) })
			}
		}
	case WCaverns:
		a.bed(fLP, 90, 0.8, 0.35, []float64{1, 2}, 0.5, 0.2) // rumble
		a.bed(fBP, 500, 3, 0.03, []float64{1, 3}, 1.0, 0.6)  // far draught
		for k := 0; k < 18; k++ {
			a.drip(R()*ambLen, R()*1.8-0.9, 900+1400*R(), 0.08+0.06*R())
		}
	case WRuins:
		a.bed(fBP, 900, 0.9, 0.12, []float64{1, 2, 4}, 1.3, 0.8) // gusts
		a.bed(fHP, 6000, 0.7, 0.012, []float64{3}, 0.8, 0.2)     // sand hiss
		for _, s := range []float64{7, 19} {                     // a hawk
			ph := 0.0
			a.add(s, 1.4, 0.6, func(t float64) float64 {
				f := 2600 - 900*t/1.4 + 60*math.Sin(2*math.Pi*25*t)
				ph += f / sampleRate
				return math.Sin(2*math.Pi*ph) * 0.035 * math.Sin(math.Pi*t/1.4)
			})
		}
	case WClouds:
		a.bed(fBP, 1400, 0.8, 0.07, []float64{1, 2, 3}, 1.1, 0.7) // high wind
		a.bed(fLP, 250, 0.7, 0.05, []float64{1}, 0.5, 0.2)
		notes := []float64{1046, 1175, 1318, 1568, 1760}
		for k := 0; k < 16; k++ {
			a.chime(R()*ambLen, R()*1.6-0.8, notes[a.rng.Intn(len(notes))], 0.035)
		}
		for _, s := range []float64{9} { // a far gull
			for k := 0; k < 3; k++ {
				a.chirp(s+float64(k)*0.3, -0.7, 1800, 1300, 0.22, 0.03)
			}
		}
	case WMachine:
		for _, h := range [][2]float64{{50, 0.05}, {100, 0.035}, {150, 0.02}, {200, 0.012}} {
			a.add(0, ambLen, 0, func(f, g float64) func(float64) float64 {
				return func(t float64) float64 { return math.Sin(2*math.Pi*f*t) * g }
			}(h[0], h[1]))
		}
		a.bed(fLP, 160, 0.8, 0.12, []float64{6}, 0.3, 0.1)
		// the cycle: clank, clank, hiss every 4 s
		for c := 0.0; c < ambLen; c += 4 {
			for _, o := range []float64{0, 0.6} {
				s, pan := c+o, -0.5+o
				a.add(s, 0.5, pan, func() func(float64) float64 {
					var q biquad
					q.set(fBP, 1300, 4)
					return func(t float64) float64 {
						return (q.proc(R()*2-1)*0.4 + math.Sin(2*math.Pi*310*t)*0.3) * 0.12 * math.Exp(-t/0.07)
					}
				}())
			}
			a.add(c+2.2, 1.1, 0.4, func() func(float64) float64 {
				var q biquad
				q.set(fHP, 3500, 0.7)
				return func(t float64) float64 { return q.proc(R()*2-1) * 0.05 * math.Sin(math.Pi*t/1.1) }
			}())
		}
	case WDream:
		// a soft major shimmer, slowly breathing
		for i, f := range []float64{261.6, 329.6, 392, 523.3, 659.3} {
			a.add(0, ambLen, float64(i%3-1)*0.4, func(f float64, i int) func(float64) float64 {
				return func(t float64) float64 {
					return (math.Sin(2*math.Pi*f*t) + 0.4*math.Sin(2*math.Pi*f*1.003*t)) * 0.010 * (0.6 + 0.4*math.Sin(2*math.Pi*float64(1+i%2)*t/ambLen+float64(i)))
				}
			}(f, i))
		}
		box := []float64{1046.5, 1318.5, 1568, 2093, 1760}
		for k := 0; k < 18; k++ {
			s := R() * ambLen
			f := box[a.rng.Intn(len(box))]
			a.tone(s, 1.6, R()*1.4-0.7, f, 0.03, envAD(0.003, 0.5))
			a.tone(s, 1.6, R()*1.4-0.7, f*3, 0.008, envAD(0.003, 0.25))
		}
		a.bed(fLP, 380, 0.8, 0.08, []float64{4, 6}, 1.0, 0.4) // lake lapping
		a.crickets(0.008, 5200, 1.0, 0.6)
	}
	// normalise to a gentle level and pack 16-bit stereo
	pk := 1e-9
	for i := range a.l {
		pk = math.Max(pk, math.Max(math.Abs(a.l[i]), math.Abs(a.r[i])))
	}
	g := 0.55 / pk
	out := make([]byte, n*4)
	for i := 0; i < n; i++ {
		putS16(out[i*4:], softClip(a.l[i]*g))
		putS16(out[i*4+2:], softClip(a.r[i]*g))
	}
	return out
}
