package main

import (
	"encoding/json"
	"testing"
)

// A resumed run keeps the origin's mastery from when it began: it decides
// which origin items are offered and the paddle livery (roadmap #6).
func TestSavedRunKeepsMastery(t *testing.T) {
	in := savedRun{Version: runSaveVersion, Phase: "map", Node: -1, Arch: archetypes[0].ID,
		Map: []savedNode{{Row: 0, Lane: 0}}, Mods: map[string]int{}, Lives: 2, Gold: 77,
		MasterySet: true, Mastery: 4, Livery: true, TrailCol: true}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	s := parseSavedRun(b)
	if s == nil {
		t.Fatal("a valid save did not parse")
	}
	if !s.MasterySet || s.Mastery != 4 || !s.Livery || !s.TrailCol || s.Gold != 77 || s.Lives != 2 {
		t.Fatalf("round trip lost fields: %+v", s)
	}
	// an older save without the mastery fields still loads
	old := []byte(`{"Version":2,"Phase":"map","Node":-1,"Arch":"` + archetypes[0].ID + `","Map":[{"Row":0}]}`)
	if s := parseSavedRun(old); s == nil || s.MasterySet {
		t.Fatalf("older save: %+v", s)
	}
}
