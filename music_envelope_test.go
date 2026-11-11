package main

import (
	"math"
	"testing"
	"time"
)

func TestMusicEnvelopeAcrossFrameRates(t *testing.T) {
	for _, fps := range []int{30, 60, 120, 144, 360} {
		s := Sounds{duck: 0.9, duckT: nowFn().Add(-time.Second)}
		for range fps / 3 {
			s.SetIntensity(1, 1/float64(fps))
		}
		seconds := float64(fps/3) / float64(fps)
		wantIntensity := 1 - math.Pow(0.97, seconds*60)
		wantDuck := 0.9 - 1.2*seconds
		if math.Abs(s.intensity-wantIntensity) > 1e-12 || math.Abs(s.duck-wantDuck) > 1e-12 {
			t.Fatalf("%d FPS: intensity %.8f (want %.8f), duck %.8f (want %.8f)", fps, s.intensity, wantIntensity, s.duck, wantDuck)
		}
	}
}
