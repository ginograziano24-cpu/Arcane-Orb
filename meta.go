package main

import "math"

// Long-term progression:
//   * XP from every run raises your Arcane Rank; each rank unlocks new
//     items into the pool (Mod.Unlock), so runs keep getting richer.
//   * Shards buy permanent Sanctum upgrades with several ranks each.
//   * Winning a run unlocks the next Ascension: stacking difficulty
//     modifiers with bigger XP and shard rewards.
//   * The Codex records every item and fusion you have discovered.

// ---------------------------------------------------------------- rank

// xpForRank is the total XP needed to reach rank n (rank 0 needs 0).
func xpForRank(n int) int {
	t := 0.0
	for k := 1; k <= n; k++ {
		t += 220 * math.Pow(float64(k), 1.25)
	}
	return int(t)
}

func rankOf(xp int) int {
	r := 0
	for r < 60 && xp >= xpForRank(r+1) {
		r++
	}
	return r
}

// runXP is the experience a finished run awards.
func runXP(r *Run, win bool) int {
	xp := float64(r.Depth*15 + r.Elites*30 + r.Bosses*120 + r.Bricks/30 + r.MaxCombo)
	if win {
		xp += 300
	}
	return int(xp * (1 + 0.1*float64(r.Asc)))
}

// unlocksAt lists the items that become available at a rank.
func unlocksAt(rank int) []*Mod {
	var out []*Mod
	for _, m := range modList {
		if m.Unlock == rank {
			out = append(out, m)
		}
	}
	return out
}

// ---------------------------------------------------------------- sanctum

type MetaUpgrade struct {
	ID, Name, Desc, Icon string
	Base, Max            int
}

var metaUpgrades = []MetaUpgrade{
	{"vitality", "Vitality", "+10 starting max health per rank.", "heart", 30, 5},
	{"power", "Arcane Might", "+0.15 orb damage per rank.", "fist", 60, 5},
	{"keen", "Keen Eye", "+2% critical chance per rank.", "sword", 40, 5},
	{"steady", "Steady Hands", "+5% paddle width per rank.", "paddle", 40, 4},
	{"padding", "Cushioned Fall", "Losing a life restores 10 health per rank.", "shield", 40, 4},
	{"purse", "Heavy Purse", "+20 starting gold per rank.", "coin", 25, 5},
	{"merchant", "Trade Routes", "Shop prices −5% per rank.", "tag", 45, 4},
	{"fortune", "Fortune", "Rarer upgrades and relics appear more often.", "star", 60, 5},
	{"reroll", "Fortune's Favour", "+1 free reroll each run per rank.", "clover", 60, 3},
	{"recover", "Recovery", "Heal 4 health after each cleared chamber per rank.", "feather", 50, 3},
	{"combo", "Rhythm", "Your combo lasts 0.5s longer per rank.", "hourglass", 45, 3},
	{"armory", "Armory", "Start each run with a random upgrade per rank.", "anvil", 90, 3},
	{"collector", "Collector", "Start each run with a random relic per rank.", "gem", 150, 2},
	{"ward", "First Ward", "Start every chamber with one barrier charge.", "halo", 300, 1},
	{"scholar", "Scholar", "Every reward offers one extra choice.", "book", 450, 1},
}

func metaCost(m MetaUpgrade, rank int) int {
	return int(math.Round(float64(m.Base) * math.Pow(1.6, float64(rank))))
}

// applyMeta adds Sanctum ranks and Ascension modifiers to a run's stats.
func applyMeta(s *Stats, meta *SaveData, asc int) {
	u := meta.Upg
	s.MaxHP += float64(10 * u["vitality"])
	s.Damage += 0.15 * float64(u["power"])
	s.CritChance += 0.02 * float64(u["keen"])
	s.PaddleWidth += 0.05 * float64(u["steady"])
	s.LifeHeal += 10 * float64(u["padding"])
	s.ShopDiscount += 0.05 * float64(u["merchant"])
	s.FreeRerolls += u["reroll"]
	s.HealOnClear += 4 * float64(u["recover"])
	s.ComboTime += 0.5 * float64(u["combo"])
	s.BarrierPerLevel += u["ward"]
	s.ExtraChoices += u["scholar"]
	if asc >= 2 {
		s.MaxHP -= 10
	}
	if asc >= 5 {
		s.BallLossDmg += 10
	}
	if asc >= 8 {
		s.GoldMult -= 0.15
	}
}

// ---------------------------------------------------------------- ascension

var ascensions = []struct{ Name, Desc string }{
	{"Hardened", "Elite bricks have 25% more health."},
	{"Frail", "Start with 10 less max health."},
	{"Inflation", "Shop prices +15%."},
	{"Fortified", "All bricks have 15% more health."},
	{"Brittle Orb", "Dropping the orb also costs 10 health."},
	{"Weary", "Rest sites heal 25% instead of 35%."},
	{"Titans", "Bosses have 30% more health."},
	{"Poverty", "Earn 15% less gold."},
	{"Bulwark", "All bricks have another 15% more health."},
	{"Final Form", "Bosses have another 30% more health and attack faster."},
}

// ascHP is the brick health multiplier from ascension.
func ascHP(asc int, elite, boss bool) float64 {
	m := 1.0
	if asc >= 4 {
		m *= 1.15
	}
	if asc >= 9 {
		m *= 1.15
	}
	if elite && asc >= 1 {
		m *= 1.25
	}
	if boss {
		if asc >= 7 {
			m *= 1.3
		}
		if asc >= 10 {
			m *= 1.3
		}
	}
	return m
}

func ascShardMult(asc int) float64 { return 1 + 0.1*float64(asc) }
