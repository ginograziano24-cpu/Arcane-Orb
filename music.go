package main

import (
	"math"
	"math/rand"
)

// Procedural soundtrack. The shape follows what made the old tracker
// soundtracks of the genre work (studied from structure, not copied):
// a drone-anchored modal minor, a pumping octave bass, a short mallet motif
// that repeats with variation, and long loops so the music does not wear
// thin inside a single chamber. Everything here is synthesised.

type trackStyle struct {
	bpm    float64
	bars   int
	swing  float64 // 0 = straight eighths
	pumpOn bool    // pad ducks on the kick
	octave bool    // root/octave eighth-note bass (tracker techno)
	motif  []int   // pentatonic degrees per eighth, -1 = rest (seeded if nil)
}

var trackStyles = []trackStyle{
	{bpm: 100, bars: 16, swing: 0.0, pumpOn: true, octave: true},   // act 1
	{bpm: 112, bars: 16, swing: 0.0, pumpOn: true, octave: true},   // act 2
	{bpm: 124, bars: 16, swing: 0.0, pumpOn: true, octave: true},   // act 3
	{bpm: 92, bars: 16, swing: 0.12, pumpOn: false, octave: false}, // menu
}

// minor and major pentatonic, as semitones from the key root
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

// mallet is a marimba-ish tone: fundamental plus the bright 4x bar partial.
func mallet(f, amp float64) func(float64) float64 {
	return func(t float64) float64 {
		a := math.Min(1, t*400)
		return amp * a * (sine(f, t)*math.Exp(-t*5) + 0.35*sine(f*3.98, t)*math.Exp(-t*22) + 0.12*sine(f*9.1, t)*math.Exp(-t*40))
	}
}

// bell is a glassy chime with inharmonic partials.
func bell(f, amp float64) func(float64) float64 {
	return func(t float64) float64 {
		a := math.Min(1, t*600)
		return amp * a * (sine(f, t)*math.Exp(-t*2.2) + 0.5*sine(f*2.76, t)*math.Exp(-t*4) + 0.25*sine(f*5.4, t)*math.Exp(-t*7))
	}
}

// bassVoice is a soft-saw bass with a filter-ish decay on its harmonics.
func bassVoice(f, amp, dec float64) func(float64) float64 {
	return func(t float64) float64 {
		a := math.Min(1, t*300)
		br := math.Exp(-t * dec * 3)
		v := sine(f, t) + br*(0.45*sine(2*f, t)+0.22*sine(3*f, t)+0.12*sine(4*f, t))
		return amp * a * v * math.Exp(-t*dec)
	}
}

func kickVoice(amp float64) func(float64) float64 {
	return func(t float64) float64 {
		ph := 2 * math.Pi * (45*t + (110/18.0)*(1-math.Exp(-t*18)))
		return amp * math.Sin(ph) * math.Exp(-t*8)
	}
}

func noiseVoice(rng *rand.Rand, amp, dec float64, hp bool) func(float64) float64 {
	prev := 0.0
	return func(t float64) float64 {
		w := rng.Float64()*2 - 1
		v := w
		if hp {
			v = w - prev
			prev = w
		}
		return amp * v * math.Exp(-t*dec)
	}
}

// renderTrack builds the base and intensity layers for track idx.
func (s *Sounds) renderTrack(idx int, root float64) [2][]byte {
	st := trackStyles[idx%len(trackStyles)]
	beat := 60 / st.bpm
	n := int(float64(st.bars*4) * beat * sampleRate)
	base := make([]float64, n)
	hot := make([]float64, n)
	rng := rand.New(rand.NewSource(int64(idx)*7 + 11))
	chords := biomeChords[idx%len(biomeChords)]
	major := idx == 3
	scale := pentMinor
	if major {
		scale = pentMajor
	}
	eighth := func(bar, e int) float64 {
		t := (float64(bar*8+e) * 0.5) * beat
		if e%2 == 1 {
			t += st.swing * beat * 0.5
		}
		return t
	}
	chordAt := func(bar int) [3]int { return chords[(bar/2)%len(chords)] }

	// pad: three detuned chord tones per two-bar chord, slow swell
	for bar := 0; bar < st.bars; bar += 2 {
		ch := chordAt(bar)
		t0 := float64(bar*4) * beat
		dur := 8 * beat
		for k, semi := range ch {
			f := note(root, semi)
			det := 1 + 0.003*float64(k-1)
			addVoice(base, t0, dur, func(t float64) float64 {
				env := math.Min(1, t/(beat*1.5)) * math.Min(1, (dur-t)/(beat*0.8))
				return 0.045 * env * (sine(f*det, t) + 0.4*sine(f*2*det, t) + 0.15*tri(f*0.5, t))
			})
		}
	}
	if st.pumpOn {
		for i := range base {
			bt := float64(i) / sampleRate / beat
			ph := bt - math.Floor(bt)
			base[i] *= 0.62 + 0.38*math.Min(1, ph*3.2)
		}
	}

	// bass
	for bar := 0; bar < st.bars; bar++ {
		ch := chordAt(bar)
		rootSemi := ch[0]
		if rootSemi > 2 {
			rootSemi -= 12
		}
		for e := 0; e < 8; e++ {
			if !st.octave && e%2 == 1 {
				continue
			}
			semi := rootSemi
			if st.octave && e%2 == 1 {
				semi += 12
			}
			if e == 7 && bar%2 == 1 {
				semi = ch[2] - 12 // walk up into the next chord
			}
			f := note(root/2, semi)
			amp := 0.17
			if e%2 == 1 {
				amp = 0.11
			}
			addVoice(base, eighth(bar, e), beat*0.5, bassVoice(f, amp, 6))
		}
	}

	// mallet motif: a two-bar figure, played A A' A B over each 8 bars
	motif := st.motif
	if motif == nil {
		motif = make([]int, 16)
		deg := 5 + rng.Intn(3)
		for i := range motif {
			if i%4 != 0 && rng.Float64() < 0.35 {
				motif[i] = -1
				continue
			}
			deg += []int{-2, -1, -1, 0, 1, 1, 2}[rng.Intn(7)]
			if deg < 3 {
				deg = 4
			}
			if deg > 10 {
				deg = 9
			}
			motif[i] = deg
		}
		motif[15] = -1
	}
	answer := make([]int, 16)
	for i, d := range motif {
		answer[i] = d
		if d >= 0 && i >= 8 {
			answer[i] = d + []int{-1, 1, 2}[rng.Intn(3)]
		}
	}
	for bar := 0; bar < st.bars; bar++ {
		phrase := (bar / 2) % 4
		pat := motif
		if phrase == 3 {
			pat = answer
		}
		for e := 0; e < 8; e++ {
			d := pat[(bar%2)*8+e]
			if d < 0 {
				continue
			}
			if phrase == 1 && e >= 6 && bar%2 == 1 {
				continue // A' leaves a breath at its end
			}
			semi := pentNote(scale, d)
			f := note(root, semi)
			addVoice(base, eighth(bar, e), beat*1.6, mallet(f, 0.075))
			if bar >= st.bars/2 {
				// second half: a soft bell doubles the motif an octave up
				addVoice(base, eighth(bar, e), beat*2, bell(f*2, 0.022))
			}
		}
	}

	// intensity layer -----------------------------------------------------
	nrng := rand.New(rand.NewSource(int64(idx) + 99))
	for bar := 0; bar < st.bars; bar++ {
		for b := 0; b < 4; b++ {
			t := float64(bar*4+b) * beat
			addVoice(hot, t, beat*0.9, kickVoice(0.36))
			// offbeat open hat
			addVoice(hot, t+beat*0.5+st.swing*beat*0.25, beat*0.4, noiseVoice(nrng, 0.07, 18, true))
			if b == 1 || b == 3 {
				addVoice(hot, t, beat*0.5, noiseVoice(nrng, 0.13, 13, false))
				addVoice(hot, t, beat*0.3, func(t float64) float64 { return 0.05 * sine(196, t) * math.Exp(-t*30) })
			}
			// closed 16th ticks
			for q := 1; q < 4; q += 2 {
				addVoice(hot, t+float64(q)*beat*0.25, beat*0.1, noiseVoice(nrng, 0.025, 60, true))
			}
		}
		// 16th arpeggio over the chord, rising then folding back
		ch := chordAt(bar)
		order := []int{0, 1, 2, 3, 2, 1, 2, 3}
		for q := 0; q < 16; q++ {
			step := order[q%8]
			semi := ch[step%3] + 12*(step/3)
			f := note(root*2, semi)
			tq := float64(bar*16+q) * beat * 0.25
			addVoice(hot, tq, beat*0.25, func(t float64) float64 {
				return 0.06 * (sq(f, t)*0.3 + sine(f, t)) * math.Exp(-t*14)
			})
		}
		// fills: snare roll leading into every 4th bar
		if bar%4 == 3 {
			for q := 12; q < 16; q++ {
				tq := float64(bar*16+q) * beat * 0.25
				addVoice(hot, tq, beat*0.2, noiseVoice(nrng, 0.05+0.02*float64(q-12), 20, false))
			}
		}
	}
	// soft limiter so stacked voices never clip harshly
	for i := range base {
		base[i] = math.Tanh(base[i]*1.1) / 1.1
		hot[i] = math.Tanh(hot[i]*1.1) / 1.1
	}
	return [2][]byte{pcm(base), pcm(hot)}
}
