package main

import (
	"math"
	"testing"
)

// A stalled orb looping up the arena is turned home within a handful of
// wall bounces, keeps its speed, and a fresh orb is never touched.
func TestStallNudgeTurnsHome(t *testing.T) {
	L := &Level{}
	fresh := &Ball{VX: 6, VZ: 8}
	L.stallNudge(fresh)
	if fresh.VX != 6 || fresh.VZ != 8 {
		t.Fatal("an orb making progress was nudged")
	}
	b := &Ball{VX: 6, VZ: 8, Idle: stallAfter + 1}
	sp := math.Hypot(b.VX, b.VZ)
	n := 0
	for b.VZ > 0 && n < 20 {
		L.stallNudge(b)
		b.VX = -b.VX // the next wall flips it
		n++
	}
	if b.VZ >= 0 {
		t.Fatalf("still heading away after %d bounces", n)
	}
	if n > 8 {
		t.Fatalf("took %d bounces to turn home", n)
	}
	if got := math.Hypot(b.VX, b.VZ); math.Abs(got-sp) > 1e-9 {
		t.Fatalf("speed changed %.3f -> %.3f", sp, got)
	}
}
