package main

import "testing"

// Every track has a name, a place and exactly one row in the soundtrack tab (#43).
func TestSoundtrackListsEveryTrack(t *testing.T) {
	seen := map[int]bool{}
	for _, ti := range soundtrackOrder {
		if seen[ti] {
			t.Fatalf("track %d listed twice", ti)
		}
		seen[ti] = true
		if trackTitle(ti) == "?" || trackTitle(ti) == "" {
			t.Errorf("track %d (%s) has no title", ti, trackSongs[ti])
		}
		if trackWhere(ti) == "" {
			t.Errorf("track %d has no place", ti)
		}
	}
	if len(seen) != len(trackSongs) {
		t.Fatalf("soundtrack lists %d of %d tracks", len(seen), len(trackSongs))
	}
	// the siblings belong to the act whose key they share
	for i := range trackSongs {
		if a := trackAct(i); a >= 0 && a < 3 && trackKey[i] != a {
			t.Errorf("track %d: act %d but key %d", i, a, trackKey[i])
		}
	}
}
