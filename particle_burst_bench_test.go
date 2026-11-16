package main

import (
	"image/color"
	"math/rand"
	"testing"
)

func BenchmarkParticleBurst(b *testing.B) {
	g := &Game{settings: &Settings{}}
	g.settings.Particles = 2
	level := Level{g: g, rng: rand.New(rand.NewSource(2026))}
	ink := color.RGBA{255, 192, 90, 255}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		level.Parts = level.Parts[:0]
		level.bursts = 0 // one burst per frame: measure the full burst
		level.sparks(V3{0, 0.7, 10}, 192, ink, 6)
	}
}

// Many bursts in one frame are thinned; the first ones stay full (#9).
func TestBurstBudget(t *testing.T) {
	g := &Game{settings: &Settings{}}
	g.settings.Particles = 2
	L := Level{g: g, rng: rand.New(rand.NewSource(1))}
	for i := 0; i < burstsFull; i++ {
		if n := L.burstBudget(30); n != 30 {
			t.Fatalf("burst %d thinned to %d", i+1, n)
		}
	}
	if n := L.burstBudget(30); n != 10 {
		t.Fatalf("thinned burst = %d, want 10", n)
	}
	if n := L.burstBudget(1); n != 1 {
		t.Fatalf("a thinned burst keeps at least one particle, got %d", n)
	}
	for L.bursts < burstsThinned {
		L.burstBudget(30)
	}
	if n := L.burstBudget(30); n != 5 {
		t.Fatalf("late burst = %d, want 5", n)
	}
	before := len(L.Parts)
	L.bursts = 0
	L.sparks(V3{}, 12, color.RGBA{255, 255, 255, 255}, 4)
	if len(L.Parts)-before != 12 {
		t.Fatalf("first burst of a frame spawned %d sparks, want 12", len(L.Parts)-before)
	}
}
