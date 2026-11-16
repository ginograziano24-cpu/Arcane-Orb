package main

import "testing"

// Reward cards preview what they change, without touching the run (#33).
func TestStatChangesPreview(t *testing.T) {
	meta := defaultSave()
	r := &Run{Arch: &archetypes[0], Mods: map[string]int{}, Lives: baseLives}
	r.Recompute(&meta)
	// own two "power" items and a spare Active slot, to catch slice aliasing
	r.Add("heavy", &meta)
	r.Active = append(make([]string, 0, 8), r.Active...)
	before := r.Stats
	order := append([]string(nil), r.Order...)
	rows := r.statChanges(modByID["heavy"], &meta, 2)
	if len(rows) == 0 || rows[0].Label != "Orb damage" || !rows[0].Up {
		t.Fatalf("Heavy Core should raise orb damage first, got %+v", rows)
	}
	if r.Mods["heavy"] != 1 || len(r.Order) != len(order) || r.Stats != before {
		t.Fatalf("the preview changed the run: mods %v order %v", r.Mods, r.Order)
	}
	if got := r.statChanges(modByID["wide"], &meta, 2); len(got) == 0 || got[0].Label != "Paddle width" {
		t.Fatalf("Broad Paddle: %+v", got)
	}
	for _, c := range []struct {
		v    float64
		f    statFmt
		want string
	}{{1.5, sfNum, "1.5"}, {2, sfNum, "2"}, {0.14, sfPct, "14%"}, {1.1, sfBase, "110%"}, {0.3, sfBonus, "+30%"}, {2.75, sfTimes, "×2.75"}, {3, sfInt, "3"}} {
		if got := fmtStat(c.v, c.f); got != c.want {
			t.Errorf("fmtStat(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

// The recap names the best combo and where it happened (#35).
func TestBestMoments(t *testing.T) {
	r := &Run{Arch: &archetypes[0], Mods: map[string]int{}}
	if got := bestMoments(r); len(got) != 0 {
		t.Fatalf("an empty run has no highlights: %v", got)
	}
	r.MaxCombo = 64
	rc := r.rc()
	rc.ComboAct, rc.ComboWhere = 2, "Crystal Caverns"
	rc.Dmg["Lightning"], rc.Dmg["Orb"] = 300, 100
	got := bestMoments(r)
	if len(got) != 2 || got[0] != "Best combo ×64 in Crystal Caverns (Act II)" || got[1] != "Star of the run: Lightning, 75% of all damage" {
		t.Fatalf("highlights: %q", got)
	}
}

// The Build panel groups items by rarity, rarest first, curses last (#32).
func TestBuildGroups(t *testing.T) {
	meta := defaultSave()
	r := &Run{Arch: &archetypes[0], Mods: map[string]int{}}
	r.Recompute(&meta)
	var mythic, common string
	curse := curseList[0].ID // curses live in their own list (curses.go)
	for _, m := range modList {
		switch {
		case !m.Curse && m.Rarity == Mythic && mythic == "":
			mythic = m.ID
		case !m.Curse && m.Rarity == Common && common == "":
			common = m.ID
		}
	}
	for _, id := range []string{common, curse, mythic, common} {
		r.Add(id, &meta)
	}
	gs := buildGroups(r)
	if len(gs) != 3 || gs[0].Title != "Mythic" || gs[1].Title != "Common" || gs[2].Title != "Curses" {
		t.Fatalf("groups: %+v", gs)
	}
	if len(gs[1].Mods) != 1 || r.Mods[common] != 2 {
		t.Fatalf("a stacked item is one line: %+v", gs[1].Mods)
	}
}
