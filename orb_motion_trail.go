package main

const (
	orbTrailLife = 0.09
	orbTrailStep = 0.015
)

type orbTrailPoint struct {
	x, z, at float64
}

type orbMotionTrail struct {
	samples [6]orbTrailPoint
	count   int
	elapsed float64
	next    float64
}

func (t *orbMotionTrail) advance(x, z, dt float64) {
	t.elapsed += dt
	if t.elapsed < t.next {
		return
	}
	// Store real positions at a bounded cadence. Skip missed samples during
	// a slow frame so the trail never draws a straight line through a bounce.
	if t.count == len(t.samples) {
		copy(t.samples[:], t.samples[1:])
		t.count--
	}
	t.samples[t.count] = orbTrailPoint{x, z, t.elapsed}
	t.count++
	for t.next <= t.elapsed {
		t.next += orbTrailStep
	}
}
