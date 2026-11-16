package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Keepsakes: one charm carried into a run. They live in their own menu
// (keepsaketree.go) as an upgrade tree bought with shards: every keepsake
// has a few ranks and each rank is a small step, so a fully upgraded
// keepsake is still only about one Common card's worth. The balance target
// is "a small head start", never a strategy on its own. Daily, weekly and
// challenge runs go without, so everyone plays those on the same footing.

type keepsake struct {
	ID, Name, Glyph string
	Parent          string // needs keepsakeParentRanks ranks of this one first ("" = open)
	Ranks           int
	Base            int // shard cost of rank 1; rank n costs Base*n
	Col, Row        int // place in the tree
	Desc            func(rank int) string
	Apply           func(st *Stats, rank int)
	legacy          func(s *SaveData) bool // the old unlock, for saves from before the tree
}

const keepsakeParentRanks = 2

var keepsakes = []keepsake{
	{ID: "coin", Name: "Lucky Coin", Glyph: "coin", Ranks: 5, Base: 12, Col: 0, Row: 0,
		Desc:   func(n int) string { return fmt.Sprintf(tr("+%d gold at the start of every act."), 6*n) },
		legacy: func(s *SaveData) bool { return true }},
	{ID: "feather", Name: "Feather Charm", Glyph: "feather", Ranks: 6, Base: 12, Col: 1, Row: 0,
		Desc:   func(n int) string { return fmt.Sprintf(tr("+%d%% paddle width."), n) },
		Apply:  func(st *Stats, n int) { st.PaddleWidth += 0.01 * float64(n) },
		legacy: func(s *SaveData) bool { return s.Runs >= 3 }},
	{ID: "whetstone", Name: "Whetstone", Glyph: "sword", Ranks: 5, Base: 14, Col: 2, Row: 0,
		Desc:   func(n int) string { return fmt.Sprintf(tr("+%.2f orb damage."), 0.04*float64(n)) },
		Apply:  func(st *Stats, n int) { st.Damage += 0.04 * float64(n) },
		legacy: func(s *SaveData) bool { return s.BestAct >= 2 }},
	{ID: "token", Name: "Trader's Token", Glyph: "tag", Parent: "coin", Ranks: 4, Base: 22, Col: 0, Row: 1,
		Desc:   func(n int) string { return fmt.Sprintf(tr("Shop prices −%d%%."), 2*n) },
		Apply:  func(st *Stats, n int) { st.ShopDiscount += 0.02 * float64(n) },
		legacy: func(s *SaveData) bool { return s.TotalGold >= 1500 }},
	{ID: "ward", Name: "Spare Ward", Glyph: "shield", Parent: "feather", Ranks: 3, Base: 25, Col: 1, Row: 1,
		Desc: func(n int) string {
			switch n {
			case 1:
				return tr("The first chamber of act 1 starts with a barrier.")
			case 2:
				return tr("The first chamber of acts 1 and 2 starts with a barrier.")
			}
			return tr("The first chamber of every act starts with a barrier.")
		},
		legacy: func(s *SaveData) bool { return true }},
	{ID: "ring", Name: "Banisher's Ring", Glyph: "ring", Parent: "whetstone", Ranks: 2, Base: 30, Col: 2, Row: 1,
		Desc:   func(n int) string { return fmt.Sprintf(tr("Banishes per run: +%d."), n) },
		Apply:  func(st *Stats, n int) { st.ExtraBanishes += n },
		legacy: func(s *SaveData) bool { return s.Runs >= 8 }},
	{ID: "clover", Name: "Clover Pin", Glyph: "clover", Parent: "token", Ranks: 5, Base: 28, Col: 0, Row: 2,
		Desc:   func(n int) string { return fmt.Sprintf(tr("+%d%% power-up drop chance and slightly better card odds."), 3*n) },
		Apply:  func(st *Stats, n int) { st.Luck += 0.03 * float64(n) },
		legacy: func(s *SaveData) bool { return s.Wins >= 1 }},
	{ID: "ember", Name: "Ember Locket", Glyph: "flame", Parent: "ring", Ranks: 4, Base: 30, Col: 2, Row: 2,
		Desc:   func(n int) string { return fmt.Sprintf(tr("+%d%% chance to ignite bricks."), n) },
		Apply:  func(st *Stats, n int) { st.BurnChance += 0.01 * float64(n) },
		legacy: func(s *SaveData) bool { return s.Bosses >= 2 }},
	{ID: "egg", Name: "Wisp Egg", Glyph: "orb", Parent: "clover", Ranks: 3, Base: 45, Col: 0, Row: 3,
		Desc: func(n int) string {
			return fmt.Sprintf(tr("Start with a young Arcane Wisp at %d%% strength (until you find a real one)."), eggStrength(n))
		},
		Apply: func(st *Stats, n int) {
			if st.Wisps == 0 {
				st.Wisps = 1
				st.FamPower -= 1 - float64(eggStrength(n))/100
			}
		},
		legacy: func(s *SaveData) bool { return s.Bosses >= 3 }},
	{ID: "knot", Name: "Guardian's Knot", Glyph: "halo", Parent: "ward", Ranks: 3, Base: 60, Col: 1, Row: 3,
		Desc: func(n int) string {
			return fmt.Sprintf(tr("Once per run, the drop that would end it leaves you on one life instead; you keep %d%% of your gold."), knotKeep(n))
		},
		legacy: func(s *SaveData) bool { return s.Wins >= 3 }},
}

func eggStrength(rank int) int { return [...]int{0, 25, 38, 50}[min(rank, 3)] }
func knotKeep(rank int) int    { return [...]int{0, 20, 35, 50}[min(rank, 3)] }

var keepsakeByID = func() map[string]*keepsake {
	m := map[string]*keepsake{}
	for i := range keepsakes {
		m[keepsakes[i].ID] = &keepsakes[i]
	}
	return m
}()

// keepRank is the rank the player owns of a keepsake (0 = not yet).
func (s *SaveData) keepRank(id string) int { return s.KeepRanks[id] }

// keepCost is the shard price of a keepsake's next rank.
func (k *keepsake) cost(have int) int { return k.Base * (have + 1) }

// keepOpen reports whether a keepsake can be upgraded at all yet.
func (s *SaveData) keepOpen(k *keepsake) bool {
	return k.Parent == "" || s.keepRank(k.Parent) >= keepsakeParentRanks
}

// migrateKeepsakes gives saves from before the tree rank 1 of every
// keepsake they had unlocked the old way (load time).
func (s *SaveData) migrateKeepsakes() {
	if s.KeepRanks != nil {
		return
	}
	s.KeepRanks = map[string]int{}
	if s.Runs == 0 {
		return // a new player starts the tree from scratch
	}
	for _, k := range keepsakes {
		if k.legacy != nil && k.legacy(s) {
			s.KeepRanks[k.ID] = 1
		}
	}
}

// keepsakeAllowed: only runs with personal rules take a keepsake (and
// trials): not dailies, weeklies, challenges, duels or the scripted modes.
func (r *Run) personalRules() bool {
	if r.Daily || r.Duel || r.Challenge != "" || len(r.Weekly) > 0 {
		return false
	}
	for _, code := range noShowRuns {
		if r.SeedCode == code {
			return false
		}
	}
	return true
}

func (r *Run) applyKeepsake(st *Stats) {
	if k := keepsakeByID[r.Keepsake]; k != nil && k.Apply != nil && r.KeepRank > 0 {
		k.Apply(st, r.KeepRank)
	}
}

// keepsakeActStart is the act-start part of a keepsake (a run's first act
// and each new one).
func (r *Run) keepsakeActStart() {
	if r.Keepsake == "coin" {
		r.Gold += 6 * r.KeepRank
	}
}

// keepsakeChamber is the chamber-start part (newLevel).
func (L *Level) keepsakeChamber() {
	r := L.g.run
	if r.Keepsake == "ward" && r.ActDepth == 1 && L.Barrier < 4 && (r.Act <= r.KeepRank || r.KeepRank >= 3) {
		L.Barrier++
		L.g.charmFlash = 1
	}
}

// keepsakeSave is the Guardian's Knot: true when it caught the run's last
// life (loseLife).
func (r *Run) keepsakeSave() bool {
	if r.Keepsake != "knot" || r.KeepRank <= 0 || r.KeepUsed || r.Duel {
		return false
	}
	r.KeepUsed = true
	r.Lives = 1
	r.Gold = r.Gold * knotKeep(r.KeepRank) / 100
	return true
}

// keepsakeLabel names the chosen keepsake for the origin screen's button.
func (g *Game) keepsakeLabel() string {
	if k := keepsakeByID[g.save.Keepsake]; k != nil && g.save.keepRank(k.ID) > 0 {
		return trJoin("Keepsake: ", tr(k.Name), " ", roman(g.save.keepRank(k.ID)))
	}
	return tr("Keepsake: none")
}

// setupPersonal gives a new run the player's keepsake, trials and co-op
// choice, where the run allows personal rules (startRunWith).
func (g *Game) setupPersonal() {
	r := g.run
	if !r.personalRules() {
		return
	}
	if k := keepsakeByID[g.save.Keepsake]; k != nil && g.save.keepRank(k.ID) > 0 {
		r.Keepsake, r.KeepRank = k.ID, g.save.keepRank(k.ID)
	}
	r.Trials = g.selectedTrials()
	r.Coop = g.coopSel && coopPadAvailable()
	r.Recompute(&g.save)
	r.Lives = r.maxLives()
	r.keepsakeActStart()
}

// drawPersonalRow is the origin screen's row of personal run options:
// keepsake, trials and couch co-op. Daily, weekly and challenge runs go
// without them, so the row hides there.
func (g *Game) drawPersonalRow(s *ebiten.Image) {
	u := g.ui
	if g.challengeSel != "" || g.weeklySel {
		return
	}
	u.rawText = true
	if u.button(s, "keeppick", g.keepsakeLabel(), 520, 990, 300, 56, cGold, true) {
		g.openKeepsakes(StArch)
	}
	g.keepsakeButtonCharm(s, 500, 1018)
	g.keepUpgradeBadge(s, 812, 996)
	g.trialsGlow(s, 840, 990, 220, 56)
	g.emberRise(s, 840, 990, 220)
	g.coopGlow(s, 1080, 990, 220, 56)
	if g.coopSel {
		g.coopMark(s, 960, 960, 520) // a second glow under the hero: two players
	}
	if u.button(s, "trialpick", g.trialsLabel(), 840, 990, 220, 56, rgba(255, 140, 120, 255), true) {
		g.archPanel = 2
	}
	u.rawText = false
	pad := coopPadAvailable()
	if !pad {
		g.coopSel = false
	}
	label := "Co-op: off"
	if g.coopSel {
		label = "Co-op: on"
	}
	if u.button(s, "coop", label, 1080, 990, 220, 56, cAcc, pad) {
		g.coopSel = !g.coopSel
		if g.coopSel {
			g.uiBurst(1190, 1018, 12, cP2, false)
		}
	}
	if g.archPanel == 0 {
		g.drawUISparks(s) // the panels draw them when open
	}
	if u.hover(1080, 990, 220, 56) {
		if pad {
			g.tooltip = tr("Couch co-op: a friend on a controller plays a second paddle. Both paddles are a little narrower, and bricks are a little tougher.")
		} else {
			g.tooltip = tr("Couch co-op needs a controller for the second player.")
		}
	}
}
