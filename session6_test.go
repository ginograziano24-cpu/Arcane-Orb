package main

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

// The 165 and 240 Hz caps are offered and survive a save round trip.
func TestFrameCapsHighRefresh(t *testing.T) {
	for _, c := range []int{30, 60, 120, 144, 165, 240, frameCapDisplay} {
		if frameCaps[frameCapIndex(c)] != c {
			t.Errorf("cap %d not offered", c)
		}
	}
	if len(frameCapLabels) != len(frameCaps) {
		t.Fatal("labels and caps differ in length")
	}
}

// The startup trace names its slowest step.
func TestBootReportMarksSlowest(t *testing.T) {
	old := bootSteps
	defer func() { bootSteps = old }()
	bootSteps = []bootStep{{"a", 5 * time.Millisecond}, {"b", 40 * time.Millisecond}, {"c", 45 * time.Millisecond}}
	r := bootReport()
	for _, l := range strings.Split(r, "\n") {
		if strings.Contains(l, "slowest") && !strings.Contains(l, " b ") {
			t.Fatalf("wrong slowest step:\n%s", r)
		}
	}
	if !strings.Contains(r, "slowest") {
		t.Fatal("no slowest step marked")
	}
}

// The menu benchmark's summary: average, 1% low and worst frame.
func TestMenuBenchSummary(t *testing.T) {
	ms := make([]float64, 200)
	for i := range ms {
		ms[i] = 5
	}
	ms[10], ms[20] = 20, 25
	r := mbSummarize("x", ms)
	if r.worstMS != 25 {
		t.Errorf("worst %.1f", r.worstMS)
	}
	if r.low1 > 50 || r.low1 < 40 { // the slowest 1% (two frames: 25, 20) = 44 fps
		t.Errorf("1%% low %.1f", r.low1)
	}
	if r.avgFPS > 200 || r.avgFPS < 180 {
		t.Errorf("average %.1f", r.avgFPS)
	}
	if z := mbSummarize("x", nil); z.avgFPS != 0 {
		t.Error("an empty scene has a frame rate")
	}
}

// Day to night: neutral in the morning, cooler and darker by the Guardian;
// the styles move contrast the way they say; Low without a style or a
// time of day stays a plain copy.
func TestDayNightAndStyles(t *testing.T) {
	g := &Game{settings: &Settings{}, save: defaultSave()}
	g.run = NewRun("STYLE", &archetypes[0], &g.save, false)
	L := &Level{Biome: &worlds[0], Node: NCombat}
	g.run.ActDepth = 1
	morning := g.worldGrade(L)
	if morning != worldGrades[worlds[0].ID] {
		t.Error("the first chamber is not the world's own grade")
	}
	L.Node = NBoss
	night := g.worldGrade(L)
	if night.Tint[2] <= night.Tint[0] {
		t.Error("the Guardian's chamber isn't cooler (night)")
	}
	g.settings.NoDayNight = true
	if g.worldGrade(L) != morning {
		t.Error("turning the day off doesn't")
	}
	g.settings.VisualStyle = StyleContrast
	if g.worldGrade(L).Contrast <= morning.Contrast {
		t.Error("high contrast doesn't raise contrast")
	}
	g.settings.VisualStyle = StyleCalm
	if c := g.worldGrade(L); c.Saturation >= morning.Saturation || g.shakeScale() >= 1 {
		t.Error("calm isn't calmer")
	}
	g.settings.VisualStyle, g.settings.NoBloom = StyleClassic, true
	L.Node = NCombat
	if !gradeIsPlain(g.worldGrade(L)) {
		t.Error("Low with the classic style isn't a plain copy")
	}
}

// A vault opens beside the next row, leads on to the row after, and only
// once per act; random omens never pick it.
func TestSecretVault(t *testing.T) {
	m := genMap(rand.New(rand.NewSource(3)), 1)
	cur := -1
	for i, n := range m.Nodes {
		if n.Row == 1 {
			cur = i
			break
		}
	}
	opened := false
	for s := int64(0); s < 40 && !opened; s++ {
		opened = m.openVault(cur, rand.New(rand.NewSource(s)))
	}
	if !opened {
		t.Fatal("no vault ever opened")
	}
	v := m.Nodes[len(m.Nodes)-1]
	if v.Omen != "vault" || v.Row != 2 || (v.Lane >= 0 && v.Lane <= 3) || len(v.Next) == 0 {
		t.Fatalf("bad vault node %+v", v)
	}
	for _, j := range v.Next {
		if m.Nodes[j].Row != 3 && m.Nodes[j].Type != NBoss {
			t.Errorf("the vault leads to row %d", m.Nodes[j].Row)
		}
	}
	found := false
	for _, j := range m.Nodes[cur].Next {
		found = found || j == len(m.Nodes)-1
	}
	if !found {
		t.Error("the vault can't be reached")
	}
	for s := int64(0); s < 40; s++ {
		if m.openVault(cur, rand.New(rand.NewSource(s))) {
			t.Fatal("a second vault in one act")
		}
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		if randomOmen(rng) == "vault" {
			t.Fatal("a random omen picked the secret vault")
		}
	}
	if omenByID["vault"] == nil || !omenByID["vault"].Secret {
		t.Fatal("vault omen missing")
	}
}

// The weekly run: the same two different mutators all week, a new pair
// most weeks, and their rules really change the stats.
func TestWeeklyMutators(t *testing.T) {
	mon := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)
	sun := time.Date(2026, 11, 15, 22, 0, 0, 0, time.UTC)
	if weekKey(mon) != "2026-W46" || weekKey(sun) != weekKey(mon) {
		t.Fatalf("week keys %s %s", weekKey(mon), weekKey(sun))
	}
	a, b := weeklyMutators(mon), weeklyMutators(sun)
	if a[0] != b[0] || a[1] != b[1] || a[0] == a[1] {
		t.Fatalf("mutators %v %v", a, b)
	}
	diff := 0
	for w := 0; w < 10; w++ {
		x := weeklyMutators(mon.AddDate(0, 0, 7*(w+1)))
		if x[0] != a[0] || x[1] != a[1] {
			diff++
		}
	}
	if diff < 7 {
		t.Errorf("only %d of 10 weeks changed the mutators", diff)
	}
	if !strings.HasPrefix(weeklyCode(mon), "WEEKLY-") {
		t.Error(weeklyCode(mon))
	}
	sv := defaultSave()
	r := NewRun("W", &archetypes[0], &sv, false)
	r.Recompute(&sv)
	base := r.Stats
	r.Weekly = []string{"overdrive", "glasscannon"}
	r.Recompute(&sv)
	if r.Stats.SpeedMult <= base.SpeedMult || r.Stats.CritChance <= base.CritChance || r.Stats.PaddleWidth >= base.PaddleWidth {
		t.Error("the mutators didn't change the stats")
	}
	if r.storyOn() {
		t.Error("a weekly run plays the story")
	}
	for _, m := range mutators {
		if m.Stats == nil && m.Map == nil {
			t.Errorf("mutator %s does nothing", m.ID)
		}
	}
}

// Skins are worn only once earned.
func TestOrbSkins(t *testing.T) {
	g := &Game{save: defaultSave()}
	g.save.OrbSkin = "glitter"
	if g.activeSkin().ID != "" {
		t.Error("an unearned skin is worn")
	}
	g.save.Weekly = map[string]int{"2026-W46": numActs + 1}
	if g.activeSkin().ID != "glitter" {
		t.Error("an earned skin isn't worn")
	}
	g.save.OrbSkin = "cookie"
	if g.activeSkin().ID != "" {
		t.Error("cookie worn without a sealed challenge")
	}
	g.save.Challenges = map[string]int{challenges[0].ID: numActs + 1}
	if g.activeSkin().ID != "cookie" {
		t.Error("cookie not worn after a seal")
	}
}
