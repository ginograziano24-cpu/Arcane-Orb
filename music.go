package main

import (
	"math"
	"math/rand"
)

// Procedural soundtrack, built around a driving dance groove: a sixteenth-
// note bass sequencer, syncopated chord stabs, a short lead motif, and a
// punchy drum kit. The shape follows a structural study of the classic
// tracker soundtracks of the genre (tempo, groove placement, layering) —
// nothing is copied, everything is synthesised here.
//
// The base layer always plays (bass, stabs, kick and hats, so it is never
// flat); the intensity layer adds claps, percussion, an arpeggio and the
// lead as the action heats up.

type trackStyle struct {
	bpm   float64
	bars  int
	major bool
	drive float64 // 0..1, how hard the base groove hits
}

var trackStyles = []trackStyle{
	{bpm: 124, bars: 16, drive: 0.8},              // act 1
	{bpm: 128, bars: 16, drive: 0.9},              // act 2
	{bpm: 132, bars: 16, drive: 1.0},              // act 3
	{bpm: 118, bars: 16, major: true, drive: 0.6}, // menu
}

var pentMinor = []int{0, 3, 5, 7, 10}
var pentMajor = []int{0, 2, 4, 7, 9}

func pentNote(scale []int, deg int) int {
	o := int(math.Floor(float64(deg) / float64(len(scale))))
	d := deg - o*len(scale)
	return scale[d] + 12*o
}

// voice helpers ------------------------------------------------------------

func addVoice(buf []float64, start float64, dur float64, fn func(t float64) float64) {
	i0 := int(start * sampleRate)
	i1 := int((start + dur) * sampleRate)
	if i0 < 0 {
		i0 = 0
	}
	n := len(buf)
	for i := i0; i < i1; i++ {
		t := float64(i-i0) / sampleRate
		buf[i%n] += fn(t) // wraps, so tails ring into the loop start
	}
}

// sawOsc is a PolyBLEP sawtooth through a one-pole low-pass; voices call
// it once per sample in order, so it keeps its own phase and filter state.
type sawOsc struct{ ph, lp, lp2 float64 }

func (o *sawOsc) next(f, cutoff float64) float64 {
	dt := f / sampleRate
	o.ph += dt
	if o.ph >= 1 {
		o.ph -= 1
	}
	v := 2*o.ph - 1
	// PolyBLEP smooths the wrap so the saw doesn't alias
	if o.ph < dt {
		x := o.ph / dt
		v -= x + x - x*x - 1
	} else if o.ph > 1-dt {
		x := (o.ph - 1) / dt
		v -= x*x + x + x + 1
	}
	a := 1 - math.Exp(-2*math.Pi*cutoff/sampleRate)
	o.lp += (v - o.lp) * a
	o.lp2 += (o.lp - o.lp2) * a
	return o.lp2
}

func bassVoice(f, amp float64) func(float64) float64 {
	var o sawOsc
	return func(t float64) float64 {
		a := math.Min(1, t*500)
		cut := 180 + 2600*math.Exp(-t*20) // the filter snaps shut
		return amp * a * (o.next(f, cut)*1.1 + 0.6*math.Sin(2*math.Pi*f*0.5*t)) * math.Exp(-t*7)
	}
}

func stabVoice(fs []float64, amp float64) func(float64) float64 {
	osc := make([]sawOsc, len(fs)*2)
	return func(t float64) float64 {
		a := math.Min(1, t*700)
		env := math.Exp(-t * 11)
		cut := 600 + 5000*math.Exp(-t*14)
		v := 0.0
		for i, f := range fs {
			v += osc[2*i].next(f*0.996, cut) + osc[2*i+1].next(f*1.004, cut)
		}
		return amp * a * env * v / float64(len(fs))
	}
}

func pluck(f, amp, dec float64) func(float64) float64 {
	return func(t float64) float64 {
		a := math.Min(1, t*900)
		return amp * a * (math.Sin(2*math.Pi*f*t)*math.Exp(-t*dec) + 0.4*math.Sin(2*math.Pi*f*2*t)*math.Exp(-t*dec*2) +
			0.15*math.Sin(2*math.Pi*f*3.98*t)*math.Exp(-t*dec*4))
	}
}

func bell(f, amp float64) func(float64) float64 {
	return func(t float64) float64 {
		a := math.Min(1, t*600)
		return amp * a * (sine(f, t)*math.Exp(-t*2.2) + 0.5*sine(f*2.76, t)*math.Exp(-t*4) + 0.25*sine(f*5.4, t)*math.Exp(-t*7))
	}
}

func kickVoice(amp float64) func(float64) float64 {
	return func(t float64) float64 {
		ph := 2 * math.Pi * (48*t + (140/22.0)*(1-math.Exp(-t*22)))
		click := math.Exp(-t*300) * 0.5
		return amp * (math.Tanh(1.6*math.Sin(ph))*math.Exp(-t*6.5) + click)
	}
}

func noiseVoice(rng *rand.Rand, amp, dec float64, hp bool) func(float64) float64 {
	prev, prev2 := 0.0, 0.0
	return func(t float64) float64 {
		w := rng.Float64()*2 - 1
		v := w
		if hp {
			v = w - 2*prev + prev2 // second-order high-pass for crisp hats
			v *= 0.5
		}
		prev2, prev = prev, w
		return amp * v * math.Exp(-t*dec)
	}
}

func clapVoice(rng *rand.Rand, amp float64) func(float64) float64 {
	n := noiseVoice(rng, 1, 0, false)
	return func(t float64) float64 {
		// three quick bursts then a tail, like hands
		e := 0.0
		for _, o := range []float64{0, 0.011, 0.022} {
			if t >= o {
				e = math.Max(e, math.Exp(-(t-o)*160))
			}
		}
		e = math.Max(e, 0.55*math.Exp(-t*18))
		return amp * n(t) * e * (0.8 + 0.2*math.Sin(2*math.Pi*1400*t))
	}
}

// renderTrack builds the base and intensity layers for track idx.
func (s *Sounds) renderTrack(idx int, root float64) [2][]byte {
	st := trackStyles[idx%len(trackStyles)]
	beat := 60 / st.bpm
	six := beat / 4
	n := int(float64(st.bars*4) * beat * sampleRate)
	base := make([]float64, n)
	hot := make([]float64, n)
	rng := rand.New(rand.NewSource(int64(idx)*7 + 11))
	nrng := rand.New(rand.NewSource(int64(idx) + 99))
	chords := biomeChords[idx%len(biomeChords)]
	scale := pentMinor
	if st.major {
		scale = pentMajor
	}
	chordAt := func(bar int) [3]int { return chords[(bar/2)%len(chords)] }
	at := func(bar, step int) float64 { return float64(bar*16+step) * six }
	drive := st.drive

	for bar := 0; bar < st.bars; bar++ {
		ch := chordAt(bar)
		rootSemi := ch[0]
		if rootSemi > 2 {
			rootSemi -= 12
		}
		intro := bar < 2 // the opening two bars: drums and bass only, then everything lands
		// ---- bass sequencer: sixteenths, root-octave bounce with a fifth
		seq := []int{0, 12, 0, 12, 0, 12, 7, 12, 0, 12, 0, 12, 3, 12, 7, 10}
		for k := 0; k < 16; k++ {
			if k%4 == 0 && k > 0 && rng.Float64() < 0.06 {
				continue
			}
			semi := rootSemi + seq[k]
			acc := 0.13
			if k%4 == 0 {
				acc = 0.17
			}
			addVoice(base, at(bar, k), six*0.95, bassVoice(note(root/2, semi), acc*(0.75+0.25*drive)))
		}
		// ---- kick on every beat, hats on the offbeats
		for b := 0; b < 4; b++ {
			addVoice(base, at(bar, b*4), beat, kickVoice(0.34*(0.8+0.2*drive)))
			addVoice(base, at(bar, b*4+2), six*1.6, noiseVoice(nrng, 0.08, 22, true))
		}
		for k := 1; k < 16; k += 2 {
			addVoice(base, at(bar, k), six*0.5, noiseVoice(nrng, 0.03, 70, true))
		}
		if intro {
			continue
		}
		// ---- syncopated chord stabs on 1, 1e, 2&, 3, 3e, 4& (a 3-3-2 push)
		var fs []float64
		for _, semi := range ch {
			fs = append(fs, note(root, semi))
		}
		fs = append(fs, note(root, ch[0]+12))
		for _, k := range []int{0, 1, 5, 8, 9, 13} {
			amp := 0.075
			if k == 1 || k == 9 {
				amp = 0.05
			}
			addVoice(base, at(bar, k), six*2.5, stabVoice(fs, amp))
		}
		// ---- intensity: claps, ride, shaker, arpeggio
		for b := 1; b < 4; b += 2 {
			addVoice(hot, at(bar, b*4), beat*0.6, clapVoice(nrng, 0.22))
		}
		for k := 0; k < 16; k++ {
			a := 0.035
			if k%4 == 2 {
				a = 0.06
			}
			addVoice(hot, at(bar, k), six*0.8, noiseVoice(nrng, a, 40, true))
		}
		if bar%4 == 3 { // a snare roll into each phrase
			for k := 8; k < 16; k++ {
				addVoice(hot, at(bar, k), six, clapVoice(nrng, 0.05+0.015*float64(k-8)))
			}
		}
		order := []int{0, 1, 2, 3, 2, 1, 2, 3}
		for k := 0; k < 16; k++ {
			step := order[k%8]
			semi := ch[step%3] + 12*(step/3)
			addVoice(hot, at(bar, k), six*1.4, pluck(note(root*2, semi), 0.05, 16))
		}
	}

	// ---- lead motif: two bars, call and answer, from bar 4 on
	motif := make([]int, 16)
	deg := 6 + rng.Intn(2)
	for i := range motif {
		if i%4 != 0 && rng.Float64() < 0.45 {
			motif[i] = -1
			continue
		}
		deg += []int{-2, -1, -1, 0, 1, 1, 2}[rng.Intn(7)]
		deg = max(3, min(10, deg))
		motif[i] = deg
	}
	motif[15] = -1
	for bar := 4; bar < st.bars; bar++ {
		phrase := (bar / 2) % 4
		for k := 0; k < 8; k++ {
			d := motif[(bar%2)*8+k]
			if d < 0 {
				continue
			}
			if phrase == 3 && k >= 4 {
				d += 1
			}
			f := note(root*2, pentNote(scale, d))
			addVoice(hot, at(bar, k*2), beat*1.2, pluck(f, 0.07, 6))
			if bar >= st.bars/2 {
				addVoice(base, at(bar, k*2), beat*2, bell(f, 0.03))
			}
		}
	}

	// ---- a little room: a short feedback echo on each layer
	echo := func(b []float64, d, g float64) {
		k := int(d * sampleRate)
		for i := range b {
			b[i] += b[(i-k+len(b))%len(b)] * g
		}
	}
	echo(base, six*3, 0.18)
	echo(hot, six*3, 0.22)
	for i := range base {
		base[i] = math.Tanh(base[i]*1.2) / 1.2
		hot[i] = math.Tanh(hot[i]*1.2) / 1.2
	}
	return [2][]byte{pcm(base), pcm(hot)}
}
