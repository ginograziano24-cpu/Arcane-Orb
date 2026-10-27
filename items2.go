package main

// The 1.0 item expansion: new relics built around scrolls, curses and
// omens, plus upgrades that fill gaps between the elemental builds.

var moreMods = []*Mod{
	// scrolls
	{ID: "scrollcase", Icon: "book", Name: "Scroll Case", Desc: "Carry one more scroll.", Rarity: Uncommon, Relic: true, Tags: []string{"tech"}, Max: 2,
		Apply: func(s *Stats) { s.ScrollSlots++ }},
	{ID: "quill", Icon: "feather", Name: "Scribe's Quill", Desc: "Each cleared chamber has a 35% chance to leave a scroll.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Unlock: 2,
		Apply: func(s *Stats) { s.ScribeChance += 0.35 }},
	{ID: "echoscript", Icon: "ring", Name: "Echoing Script", Desc: "Reading a scroll has a 30% chance not to use it up.", Rarity: Rare, Relic: true, Tags: []string{"tech"}, Max: 1, Unlock: 3,
		Apply: func(s *Stats) { s.ScrollKeep += 0.3 }},
	// curses
	{ID: "hexfeeder", Icon: "fang", Name: "Hex Feeder", Desc: "+12% damage for every curse you carry.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Unlock: 3,
		Apply: func(s *Stats) { s.CurseDmg += 0.12 }},
	{ID: "talisman", Icon: "halo", Name: "Warding Talisman", Desc: "Curses can't lower your max health, and +15 max health.", Rarity: Uncommon, Relic: true, Tags: []string{"guard"}, Max: 1,
		Apply: func(s *Stats) { s.MaxHP += 15; s.CurseWard = true }},
	// omens
	{ID: "omenreader", Icon: "eye", Name: "Omen Reader", Desc: "Challenges you complete also pay 35 gold.", Rarity: Uncommon, Relic: true, Tags: []string{"wealth"}, Max: 1,
		Apply: func(s *Stats) { s.ChallengeGold += 35 }},
	// upgrades
	{ID: "keen2", Icon: "sword", Name: "Honed Edges", Desc: "+0.3 orb damage and +4% critical chance.", Rarity: Common, Tags: []string{"crit"},
		Apply: func(s *Stats) { s.Damage += 0.3; s.CritChance += 0.04 }},
	{ID: "feather", Icon: "feather", Name: "Featherweight", Desc: "+12% paddle width and orbs fly 5% slower.", Rarity: Common, Tags: []string{"guard"}, Max: 3,
		Apply: func(s *Stats) { s.PaddleWidth += 0.12; s.SpeedMult -= 0.05 }},
	{ID: "ironwill", Icon: "shield", Name: "Iron Will", Desc: "+20 max health; losing a life restores 15 health.", Rarity: Uncommon, Tags: []string{"guard"}, Max: 3,
		Apply: func(s *Stats) { s.MaxHP += 20; s.LifeHeal += 15 }},
	{ID: "emberheart", Icon: "blaze", Name: "Ember Heart", Desc: "+8% ignite chance; burning deals +25% damage.", Rarity: Uncommon, Tags: []string{"fire"}, Max: 3,
		Apply: func(s *Stats) { s.BurnChance += 0.08; s.BurnMult += 0.25 }},
	{ID: "glacier", Icon: "iceheart", Name: "Glacier Core", Desc: "+8% freeze chance; frozen bricks stay frozen 1s longer.", Rarity: Uncommon, Tags: []string{"frost"}, Max: 3,
		Apply: func(s *Stats) { s.FreezeChance += 0.08; s.FreezeDur += 1 }},
	{ID: "arclens", Icon: "lens", Name: "Arc Lens", Desc: "+6% lightning chance, +1 arc and +15% lightning damage.", Rarity: Uncommon, Tags: []string{"storm"}, Max: 3,
		Apply: func(s *Stats) { s.ChainChance += 0.06; s.ChainCount++; s.ChainMult += 0.15 }},
	{ID: "volatilemix", Icon: "bomb", Name: "Volatile Mix", Desc: "+8% explosion chance; explosions deal +20% damage.", Rarity: Uncommon, Tags: []string{"blast"}, Max: 3,
		Apply: func(s *Stats) { s.ExplodeChance += 0.08; s.BlastMult += 0.2 }},
	{ID: "huntmark", Icon: "target", Name: "Hunter's Mark", Desc: "+40% damage to elites and Guardians, +5% critical chance.", Rarity: Rare, Tags: []string{"crit"}, Max: 2, Unlock: 2,
		Apply: func(s *Stats) { s.BossDmg += 0.4; s.CritChance += 0.05 }},
	{ID: "glassorb", Icon: "glass", Name: "Glass Orb", Desc: "Damage ×1.35, but dropping the orb also costs 15 health.", Rarity: Rare, Tags: []string{"power"}, Max: 1, Unlock: 3,
		Apply: func(s *Stats) { s.DmgMult *= 1.35; s.BallLossDmg += 15 }},
	{ID: "prismshard", Icon: "gem", Name: "Prismatic Shard", Desc: "+6% chance to ignite, freeze, arc lightning and explode.", Rarity: Epic, Relic: true, Tags: []string{"void"}, Max: 1, Unlock: 4,
		Apply: func(s *Stats) {
			s.BurnChance += 0.06
			s.FreezeChance += 0.06
			s.ChainChance += 0.06
			s.ExplodeChance += 0.06
		}},
	{ID: "wandplenty", Icon: "star", Name: "Wand of Plenty", Desc: "Rewards offer one more choice, and +15% gold.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Unlock: 4,
		Apply: func(s *Stats) { s.ExtraChoices++; s.GoldMult += 0.15 }},
	{ID: "wardstone", Icon: "urn", Name: "Ward Stone", Desc: "Start every chamber with a barrier charge; barriers recharge.", Rarity: Epic, Relic: true, Tags: []string{"guard"}, Max: 1, Unlock: 5,
		Apply: func(s *Stats) { s.BarrierPerLevel++; s.BarrierRegen = true }},
	{ID: "comet", Icon: "comet", Name: "Comet Tail", Desc: "+15% orb speed and +0.4 damage.", Rarity: Uncommon, Tags: []string{"power"}, Max: 2,
		Apply: func(s *Stats) { s.SpeedMult += 0.15; s.Damage += 0.4 }},
	{ID: "pickpocket", Icon: "fist", Name: "Pickpocket", Desc: "Bricks are 6% more likely to drop gold, and +10% gold.", Rarity: Common, Tags: []string{"wealth"}, Max: 3,
		Apply: func(s *Stats) { s.CoinChance += 0.06; s.GoldMult += 0.1 }},
}

func init() {
	for _, m := range moreMods {
		modList = append(modList, m)
		modByID[m.ID] = m
	}
}

func (r *Run) scrollCap() int { return 3 + r.Stats.ScrollSlots }
