package main

import (
	"math/rand"
	"testing"
)

// The cosmetic effects never touch the chamber's own random source, so
// seeded chambers and sims play out the same with or without them.
func TestVisualFXLeaveGameplayRandomAlone(t *testing.T) {
	g := &Game{settings: &Settings{Particles: 1}, save: defaultSave()}
	g.run = &Run{Arch: &archetypes[0], Mods: map[string]int{"inferno": 1}}
	b := worlds[WMeadow]
	L := &Level{g: g, Biome: &b, rng: rand.New(rand.NewSource(42))}
	L.Balls = []*Ball{{X: 1, Z: 6, R: 0.5}}
	want := rand.New(rand.NewSource(42)).Int63()
	L.confetti()
	for i := 0; i < 120; i++ {
		L.updateTrails(1.0 / 60)
		L.updateWeather(1.0 / 60)
		L.addMark(0, 5, 1, cGold, true)
		L.updateMarks(1.0 / 60)
	}
	if got := L.rng.Int63(); got != want {
		t.Fatal("a cosmetic effect used the chamber's random source")
	}
	if len(L.Parts) < 100 {
		t.Fatalf("confetti and embers: only %d particles", len(L.Parts))
	}
	if len(L.Marks) > maxMarks {
		t.Fatalf("%d floor marks, cap is %d", len(L.Marks), maxMarks)
	}
	if k, _ := L.orbLight(1.5, 6); k <= 0 {
		t.Fatal("a brick beside the orb should catch its light")
	}
	if k, _ := L.orbLight(12, 30); k != 0 {
		t.Fatal("a far brick should not")
	}
}
