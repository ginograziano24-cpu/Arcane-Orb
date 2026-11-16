package main

import (
	"fmt"
	"testing"
)

// Both writers send letters, each moment gets its own, and unread letters
// come first (#65).
func TestPickLetter(t *testing.T) {
	from := map[string]int{}
	for seed := 0; seed < 60; seed++ {
		r := &Run{SeedN: int64(seed * 7919), Act: 2}
		l := pickLetter(r, nil)
		if l == nil {
			t.Fatal("no letter for act 2")
		}
		if l.Tier != 1 {
			t.Fatalf("after the first Guardian, got a tier-%d letter (%s)", l.Tier, l.ID)
		}
		from[l.From]++
	}
	if from["fumbleton"] < 15 || from["mirabel"] < 15 {
		t.Fatalf("both writers should write often: %v", from)
	}
	// once a letter is read, the writer's other letters come first
	r := &Run{SeedN: 1, Act: 2}
	first := pickLetter(r, nil)
	second := pickLetter(r, map[string]bool{first.ID: true})
	if second == nil || second.ID == first.ID || second.From != first.From {
		t.Fatalf("expected another letter from %s, got %+v", first.From, second)
	}
	// every letter is complete and every greeting takes the hero's name
	for _, l := range letterDefs {
		if len(l.Body) == 0 || l.Sign == "" || fmt.Sprintf(l.Hello, "X") == l.Hello {
			t.Errorf("letter %s is incomplete", l.ID)
		}
	}
	// endless loops get letters too
	if l := pickLetter(&Run{SeedN: 5, Act: 6}, nil); l == nil {
		t.Fatal("no letter in endless")
	}
}

// The P.S. remembers something the player did.
func TestLetterPS(t *testing.T) {
	r := &Run{MaxCombo: 64}
	if got := letterPS(r); got != "P.S. A 64-hit combo! I had to sit down." {
		t.Fatalf("P.S.: %q", got)
	}
	r = &Run{}
	if letterPS(r) == "" {
		t.Fatal("empty P.S.")
	}
}

// The Build panel lists fusions one item away (#58).
func TestNearFusions(t *testing.T) {
	meta := defaultSave()
	r := &Run{Arch: &archetypes[0], Mods: map[string]int{}}
	f := fusionByID["lancer"]
	r.Add(f.A, &meta)
	found := false
	for _, nf := range nearFusions(r) {
		if nf.F.ID == "lancer" && nf.Need == f.B {
			found = true
		}
	}
	if !found {
		t.Fatalf("Lancer Formation should be one item away: %+v", nearFusions(r))
	}
	r.Add(f.B, &meta)
	for _, nf := range nearFusions(r) {
		if nf.F.ID == "lancer" {
			t.Fatal("a forged fusion is not 'one item away'")
		}
	}
}
