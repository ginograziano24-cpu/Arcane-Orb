package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Weekly mutator run (roadmap #108). Every week (ISO weeks, Monday to
// Sunday) the same seed and the same two mutators for every player: a story
// run under rules nobody has planned a build for. It starts from the
// Challenges screen. The best act reached each week is kept, and the first
// win of a week pays 100 shards.

type mutator struct {
	ID, Name, Rules, Glyph string
	Stats                  func(s *Stats)
	Map                    func(m *ActMap, rng *rand.Rand)
}

var mutators = []mutator{
	{ID: "glasscannon", Name: "Glass Cannon", Glyph: "glass", Rules: "Paddle 25% narrower; critical chance +20%.",
		Stats: func(s *Stats) { s.PaddleWidth -= 0.25; s.CritChance += 0.2 }},
	{ID: "goldrush", Name: "Gold Rush", Glyph: "coinstack", Rules: "Gold +75%, but bricks have 25% more health.",
		Stats: func(s *Stats) { s.GoldMult += 0.75; s.BrickHPMult += 0.25 }},
	{ID: "overdrive", Name: "Overdrive", Glyph: "bolt", Rules: "Orbs fly 20% faster and hit 25% harder.",
		Stats: func(s *Stats) { s.SpeedMult += 0.2; s.DmgMult += 0.25 }},
	{ID: "giants", Name: "Giant Orbs", Glyph: "orb", Rules: "Orbs are 35% larger; orb damage -15%.",
		Stats: func(s *Stats) { s.SizeMult += 0.35; s.DmgMult -= 0.15 }},
	{ID: "omenstorm", Name: "Omen Storm", Glyph: "eye", Rules: "Every fight carries an omen.",
		Map: func(m *ActMap, rng *rand.Rand) {
			for _, n := range m.Nodes {
				if (n.Type == NCombat && n.Row > 0 || n.Type == NElite) && n.Omen == "" {
					n.Omen = randomOmen(rng)
				}
			}
		}},
	{ID: "shopkeep", Name: "Friends in Trade", Glyph: "tag", Rules: "Shops 25% cheaper, with a free reroll each visit.",
		Stats: func(s *Stats) { s.ShopDiscount += 0.25; s.ShopRerolls++ }},
	{ID: "elements", Name: "Elemental Surge", Glyph: "storm", Rules: "Burn, freeze and chain chances +8% each.",
		Stats: func(s *Stats) { s.BurnChance += 0.08; s.FreezeChance += 0.08; s.ChainChance += 0.08 }},
	{ID: "luck", Name: "Lucky Week", Glyph: "clover", Rules: "Luck +25% and one more card to choose from.",
		Stats: func(s *Stats) { s.Luck += 0.25; s.ExtraChoices++ }},
}

// mutatorByID is filled in init: as a plain variable it keeps the stat
// code (Recompute) out of the omen tables' initialisation order.
var mutatorByID = map[string]*mutator{}

func init() {
	for i := range mutators {
		mutatorByID[mutators[i].ID] = &mutators[i]
	}
}

// weekKey is this week's key, "2026-W46".
func weekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf(weekFmt, y, w)
}

// weeklyCode is the week's shared seed code.
func weeklyCode(t time.Time) string { return weeklyPrefix + weekKey(t) }

// weekFmt is the ISO week key's format ("2026-W46").
var weekFmt = strings.Join([]string{"%d", "W%02d"}, "-")

// weeklyPrefix starts every weekly seed code.
var weeklyPrefix = strings.ToUpper("weekly-")

// weeklyMutators picks the week's two mutators from its key, the same for
// everyone.
func weeklyMutators(t time.Time) []string {
	rng := rand.New(rand.NewSource(int64(hashStr(weekKey(t)))))
	p := rng.Perm(len(mutators))
	return []string{mutators[p[0]].ID, mutators[p[1]].ID}
}

// applyWeeklyStats adds the run's mutators to its stats (Recompute).
func (r *Run) applyWeeklyStats(s *Stats) {
	for _, id := range r.Weekly {
		if m := mutatorByID[id]; m != nil && m.Stats != nil {
			m.Stats(s)
		}
	}
}

// applyWeeklyMap reshapes a fresh act map for the run's mutators.
func (r *Run) applyWeeklyMap() {
	for i, id := range r.Weekly {
		if m := mutatorByID[id]; m != nil && m.Map != nil {
			m.Map(r.Map, rand.New(rand.NewSource(r.SeedN^int64(r.Act*977+i*31))))
		}
	}
}

// startWeekly begins this week's run with origin arch.
func (g *Game) startWeekly(arch *Archetype) {
	now := time.Now()
	g.seedText = weeklyCode(now)
	g.daily, g.endlessMode, g.challengeSel, g.weeklySel = false, false, "", false
	asc := g.save.AscSel
	g.save.AscSel = 0 // the week's rules are its difficulty
	g.pendingWeekly = weeklyMutators(now)
	g.startRunWith(arch, "")
	g.pendingWeekly = nil
	g.save.AscSel = asc
	g.seedText = ""
}

// setupWeekly is called by startRunWith once the run exists.
func (g *Game) setupWeekly() {
	if len(g.pendingWeekly) == 0 {
		return
	}
	r := g.run
	r.Weekly = append([]string(nil), g.pendingWeekly...)
	r.Recompute(&g.save)
	r.Lives = r.maxLives()
	r.applyWeeklyMap()
}

// onWeeklyEnd records the week's best and pays the first win.
func (g *Game) onWeeklyEnd(r *Run, win bool) {
	if len(r.Weekly) == 0 || !strings.HasPrefix(r.SeedCode, weeklyPrefix) {
		return
	}
	key := strings.TrimPrefix(r.SeedCode, weeklyPrefix)
	if g.save.Weekly == nil {
		g.save.Weekly = map[string]int{}
	}
	reached := r.Act
	if win {
		reached = numActs + 1
	}
	best := g.save.Weekly[key]
	if reached > best {
		g.save.Weekly[key] = reached
	}
	if win && best <= numActs {
		g.save.Shards += 100
		g.shardsWon += 100
	}
}

// drawWeeklyCard is this week's run on the Challenges screen.
func (g *Game) drawWeeklyCard(s *ebiten.Image, x, y, w, h float64) {
	u := g.ui
	now := time.Now()
	ids := weeklyMutators(now)
	col := rgba(120, 220, 255, 255)
	hv, clk := u.hit("weekly", x, y, w, h)
	k := u.anim("weekly", hv, 10)
	if k > 0.01 {
		drawSprite(s, sprSoft, u.X(x-40), u.Y(y-24), (w+80)*u.S, (h+48)*u.S, ca(col, 0.2*k), true)
	}
	u.frame(s, x, y, w, h, 0.95)
	if hv && u.nav.mode {
		u.focusRing(s, x, y, w, h, 12)
	}
	u.drawEmblem(s, glyphImage("hourglass", col), x+44, y+h/2, 46, 1)
	u.text(s, fmt.Sprintf(tr("This week's mutators  ·  %s"), weekKey(now)), fBold, 18, x+84, y+12, lighten(col, 0.3), alignLeft)
	var names []string
	for _, id := range ids {
		names = append(names, tr(mutatorByID[id].Name))
	}
	u.rawText = true
	u.text(s, strings.Join(names, "  +  "), fSerif, 24, x+84, y+38, cWhite, alignLeft)
	u.rawText = false
	best := g.save.Weekly[weekKey(now)]
	st := tr("Not attempted this week")
	switch {
	case best > numActs:
		st = tr("Won this week!")
	case best > 0:
		st = fmt.Sprintf(tr("Best this week: Act %d"), best)
	}
	u.rawText = true
	u.text(s, st, fBold, 16, x+w-24, y+h/2-10, cAcc, alignRight)
	u.rawText = false
	if hv {
		var rules []string
		for _, id := range ids {
			m := mutatorByID[id]
			rules = append(rules, tr(m.Name)+" — "+tr(m.Rules))
		}
		g.tooltip = strings.Join(rules, "\n")
	}
	if clk {
		g.weeklySel = true
		g.challengeSel = ""
		g.daily, g.endlessMode, g.seedText = false, false, ""
		g.snd.Play("click")
		g.setState(StArch)
	}
}

// weeklyBanner marks the origin screen when the weekly run is chosen.
func (g *Game) weeklyBanner(s *ebiten.Image) {
	u := g.ui
	var names []string
	for _, id := range weeklyMutators(time.Now()) {
		names = append(names, tr(mutatorByID[id].Name))
	}
	t := tr("WEEKLY") + "  ·  " + strings.ToUpper(strings.Join(names, " + "))
	u.rawText = true
	w := u.measure(t, fBold, 20) + 90
	x := 960 - w/2
	u.frame(s, x, 1000, w, 48, 0.95)
	u.drawEmblem(s, glyphImage("hourglass", rgba(120, 220, 255, 255)), x+32, 1024, 30, 1)
	u.text(s, t, fBold, 20, x+56, 1012, rgba(170, 235, 255, 255), alignLeft)
	u.rawText = false
}
