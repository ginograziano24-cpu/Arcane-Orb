package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
)

// Every recipe renders a non-silent clip without clipping past the master
// ceiling, for every variant and a typical argument set.
func TestEveryRecipeRenders(t *testing.T) {
	for name, rc := range recipes {
		vars := max(1, rc.vars)
		for v := 0; v < vars; v++ {
			pcm := rc.build([]int{1, 0, 3}, v, rand.New(rand.NewSource(int64(v)+7)))
			if len(pcm) < 4*256 {
				t.Fatalf("%s#%d: clip too short (%d bytes)", name, v, len(pcm))
			}
			peak := 0.0
			for i := 0; i+1 < len(pcm); i += 2 {
				peak = math.Max(peak, math.Abs(float64(int16(binary.LittleEndian.Uint16(pcm[i:])))/32000))
			}
			if peak < 0.05 || peak > 1.0001 {
				t.Fatalf("%s#%d: peak %.3f out of range", name, v, peak)
			}
		}
	}
}

// The good power-ups must not collapse back into one shared sound: compare
// coarse log spectra of the first 0.8 s pairwise.
func TestPowerUpSoundsAreDistinct(t *testing.T) {
	var specs [][]float64
	for p := PMulti; p <= PMagnet; p++ {
		if !powerDefs[p].Good {
			continue
		}
		pcm := recipes["pw"].build([]int{int(p), 0}, 0, rand.New(rand.NewSource(1)))
		specs = append(specs, coarseSpectrum(pcm))
	}
	for i := range specs {
		for j := i + 1; j < len(specs); j++ {
			if s := cosine(specs[i], specs[j]); s > 0.85 {
				t.Errorf("power-up sounds %d and %d too similar (%.2f)", i, j, s)
			}
		}
	}
}

// coarseSpectrum: energy in 12 log-spaced bands for 16 time slices, logged
// and mean-removed.
func coarseSpectrum(pcm []byte) []float64 {
	const n = 35280 // 0.8 s
	x := make([]float64, n)
	for i := 0; i < n && i*4+1 < len(pcm); i++ {
		x[i] = float64(int16(binary.LittleEndian.Uint16(pcm[i*4:]))) / 32768
	}
	var out []float64
	slice := n / 16
	for s := 0; s < 16; s++ {
		seg := x[s*slice : (s+1)*slice]
		for band := 0; band < 12; band++ {
			f := 120 * math.Pow(2, float64(band)*0.55)
			// Goertzel at the band centre
			w := 2 * math.Pi * f / sampleRate
			c := 2 * math.Cos(w)
			var s1, s2 float64
			for _, v := range seg {
				s0 := v + c*s1 - s2
				s2, s1 = s1, s0
			}
			out = append(out, math.Log(s1*s1+s2*s2-c*s1*s2+1e-6))
		}
	}
	m := 0.0
	for _, v := range out {
		m += v
	}
	m /= float64(len(out))
	for i := range out {
		out[i] -= m
	}
	return out
}

func cosine(a, b []float64) float64 {
	var ab, aa, bb float64
	for i := range a {
		ab += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	return ab / math.Sqrt(aa*bb)
}

// BenchmarkWorldSoundWarm renders roughly what the warming workers do on
// entering a world: the brick ladder, debris, tough hits and the pickups.
// (2026-11-11: block-wise reverb and phasor resonators took this from about
// 490 ms to 350 ms per op on the 2-core workspace.)
func BenchmarkWorldSoundWarm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for st := 0; st < 15; st++ {
			recipes["brk"].build([]int{2, 0, st}, 0, rand.New(rand.NewSource(1)))
		}
		recipes["deb"].build([]int{2}, 0, rand.New(rand.NewSource(1)))
		recipes["tuf"].build([]int{2, 0}, 0, rand.New(rand.NewSource(1)))
		for p := 0; p < 10; p++ {
			recipes["pw"].build([]int{p, 0}, 0, rand.New(rand.NewSource(1)))
		}
		recipes["pick"].build([]int{3, 0}, 0, rand.New(rand.NewSource(1)))
	}
}
