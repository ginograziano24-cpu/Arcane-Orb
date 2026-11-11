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
		level.sparks(V3{0, 0.7, 10}, 192, ink, 6)
	}
}
