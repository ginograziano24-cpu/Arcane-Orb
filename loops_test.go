package main

import "testing"

// Each endless loop adds one named twist; they stack and cycle (#31).
func TestLoopTwists(t *testing.T) {
	r := &Run{Act: numActs}
	if len(r.activeTwists()) != 0 || r.newestTwist() != nil || r.loopCritterPace() != 1 {
		t.Fatal("no twists before Endless")
	}
	for l := 1; l <= 2*len(loopTwists)+1; l++ {
		r.Act = numActs + l
		tw := r.activeTwists()
		if len(tw) != l {
			t.Fatalf("loop %d: %d twists", l, len(tw))
		}
		if r.newestTwist() != tw[l-1] {
			t.Fatalf("loop %d: newest twist is not the last one", l)
		}
		if tw[l-1].Name == "" || tw[l-1].Desc == "" || tw[l-1].Apply == nil {
			t.Fatalf("loop %d: incomplete twist %+v", l, tw[l-1])
		}
	}
	r.Act = numActs + len(loopTwists) + 4 // Critter Season twice
	if p := r.loopCritterPace(); p != 4 {
		t.Fatalf("two Critter Seasons should quadruple the pace, got %v", p)
	}
}
