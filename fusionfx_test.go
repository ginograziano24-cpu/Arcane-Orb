package main

import "testing"

// The forge shows what a fusion adds on its own, without touching the run.
func TestFusionGains(t *testing.T) {
	meta := defaultSave()
	r := &Run{Arch: &archetypes[0], Mods: map[string]int{}, SeedCode: "TEST"}
	r.Recompute(&meta)
	lancer := fusionByID["lancer"]
	r.Add(lancer.A, &meta)
	r.Add(lancer.B, &meta)
	if !r.HasFusion("lancer") {
		t.Fatal("the test build didn't forge Lancer Formation")
	}
	before := r.Stats
	gains := r.fusionGains(lancer, &meta, 4)
	if len(gains) == 0 || gains[0].Label != "Pierce" || !gains[0].Up {
		t.Fatalf("Lancer Formation should show its pierce, got %+v", gains)
	}
	if r.Stats != before || r.skipFusion != "" {
		t.Fatal("fusionGains changed the run")
	}
	// a fusion that only changes how things work has no numbers to show
	cryo := fusionByID["cryo"]
	r2 := &Run{Arch: &archetypes[0], Mods: map[string]int{}, SeedCode: "TEST"}
	r2.Add(cryo.A, &meta)
	r2.Add(cryo.B, &meta)
	if g := r2.fusionGains(cryo, &meta, 4); len(g) != 0 {
		t.Fatalf("Cryo Charge is a passive, got numbers %+v", g)
	}
}

// Numbers count up in their own format and land exactly on the new value.
func TestCountUp(t *testing.T) {
	for _, c := range []struct{ from, to, mid string }{
		{"1.5", "2", "1.8"}, {"14%", "20%", "17%"}, {"+30%", "+60%", "+45%"}, {"×2", "×2.75", "×2.38"},
	} {
		if got := countUp(c.from, c.to, 0); got != c.from {
			t.Errorf("%s→%s starts at %q", c.from, c.to, got)
		}
		if got := countUp(c.from, c.to, 0.5); got != c.mid {
			t.Errorf("%s→%s halfway is %q, want %q", c.from, c.to, got, c.mid)
		}
		if got := countUp(c.from, c.to, 1); got != c.to {
			t.Errorf("%s→%s ends at %q", c.from, c.to, got)
		}
	}
}

// Presentation shows never hold up simulations, the menu demo or duels.
func TestCinematicOnlyInRealRuns(t *testing.T) {
	g := &Game{save: defaultSave()}
	for _, code := range []string{"MENU", "SIM", "CAMP", "TRAILER", "BENCH"} {
		g.run = &Run{SeedCode: code}
		if g.cinematic() {
			t.Errorf("%s run plays cinematics", code)
		}
	}
	g.run = &Run{SeedCode: "MYSEED"}
	if !g.cinematic() {
		t.Error("a real run should play the forge")
	}
	g.run.Duel = true
	if g.cinematic() {
		t.Error("duels must not stop for the forge")
	}
}

// Cards are dealt one after another and all land.
func TestDealIn(t *testing.T) {
	if dealIn(0, 0) != 0 || dealIn(0.2, 2) >= dealIn(0.2, 0) || dealIn(1, 4) != 1 {
		t.Fatal("deal-in order")
	}
}
