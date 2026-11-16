package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Achievement audit (Steam release readiness): every achievement can be
// earned by some code path, the Steam table lists exactly the game's
// achievements, and benchmark runs earn nothing.
func TestAchievementAudit(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	var src strings.Builder
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, _ := os.ReadFile(f)
		src.Write(b)
	}
	code := src.String()
	dynamic := map[string]bool{}
	for _, a := range archetypes {
		dynamic["ACH_WIN_"+strings.ToUpper(a.ID)] = true
	}
	for act := 1; act <= numActs; act++ {
		dynamic["ACH_GUARDIAN_"+itoa(act)] = true
	}
	ids := map[string]bool{}
	for _, a := range achievements {
		if ids[a.ID] {
			t.Errorf("%s defined twice", a.ID)
		}
		ids[a.ID] = true
		if a.Prog != nil || dynamic[a.ID] {
			continue
		}
		if strings.Count(code, `"`+a.ID+`"`) < 2 {
			t.Errorf("%s is defined but nothing unlocks it", a.ID)
		}
	}
	for id := range dynamic {
		if !ids[id] {
			t.Errorf("the game can unlock %s, which isn't defined", id)
		}
	}
	f, err := os.Open(filepath.Join("store", "achievements.csv"))
	if err != nil {
		t.Skip("store/achievements.csv not present")
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	inCSV := map[string]bool{}
	for _, r := range rows[1:] {
		inCSV[r[0]] = true
		if !ids[r[0]] {
			t.Errorf("store/achievements.csv lists %s, which the game doesn't have", r[0])
		}
		for _, icon := range r[4:6] {
			if _, err := os.Stat(filepath.Join("store", icon)); err != nil {
				t.Errorf("missing achievement icon %s", icon)
			}
		}
	}
	for id := range ids {
		if !inCSV[id] {
			t.Errorf("%s is missing from store/achievements.csv", id)
		}
	}
}

func TestBenchRunsEarnNothing(t *testing.T) {
	g := &Game{save: defaultSave(), settings: &Settings{}}
	g.run = NewRun("BENCH", &archetypes[0], &g.save, false)
	g.unlock("ACH_FIRST_CHAMBER")
	if g.save.Ach["ACH_FIRST_CHAMBER"] {
		t.Fatal("a benchmark run unlocked an achievement")
	}
}
