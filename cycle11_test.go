package main

import (
	"os"
	"testing"
)

// TestMain keeps every test away from the player's real save file
// (unlock, keepsake purchases and settings all call writeSave).
func TestMain(m *testing.M) {
	noPersist = true
	os.Exit(m.Run())
}

func TestFxLevelDial(t *testing.T) {
	g := &Game{settings: &Settings{}}
	for lv, want := range []float64{1, 0.5, 0} {
		g.settings.FxLevel = lv
		if g.fxAmount() != want {
			t.Fatalf("level %d: amount %v", lv, g.fxAmount())
		}
	}
	g.settings.FxLevel = fxMedium
	if !g.ambientOff() || g.flourishOff() {
		t.Fatal("Medium should cut ambient decoration only")
	}
	g.settings.FxLevel = 7
	if g.fxLevel() != fxLow {
		t.Fatal("out-of-range level not clamped")
	}
}

func TestSystemAchievements(t *testing.T) {
	g := &Game{save: defaultSave(), settings: &Settings{}}
	r := NewRun("TEST", &archetypes[0], &g.save, false)
	r.Coop = true
	r.Trials = []string{"choices", "fragile", "guardians"} // 6 points
	g.onRunEnd(r, true)
	for _, id := range []string{"ACH_COOP_WIN", "ACH_TRIAL_WIN", "ACH_NO_BANISH"} {
		if !g.save.Ach[id] {
			t.Errorf("%s not unlocked", id)
		}
	}
	g2 := &Game{save: defaultSave(), settings: &Settings{}}
	r2 := NewRun("TEST2", &archetypes[0], &g2.save, false)
	r2.Trials, r2.BanishUsed = []string{"choices", "fragile"}, 1
	g2.onRunEnd(r2, true)
	if g2.save.Ach["ACH_NO_BANISH"] || g2.save.Ach["ACH_TRIAL_WIN"] || g2.save.Ach["ACH_COOP_WIN"] {
		t.Error("unlocked without meeting the goal")
	}
}
