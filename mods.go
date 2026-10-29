package main

import (
	"image/color"
	"sort"
)

// Rarity tiers.
const (
	Common = iota
	Uncommon
	Rare
	Epic
	Legendary
	Mythic
)

var rarityNames = []string{"Common", "Uncommon", "Rare", "Epic", "Legendary", "Mythic"}
var rarityColors = []color.RGBA{
	{190, 196, 210, 255}, {90, 220, 120, 255}, {80, 160, 255, 255},
	{190, 110, 255, 255}, {255, 180, 50, 255}, {255, 80, 110, 255},
}
var rarityWeights = []float64{50, 30, 14, 5, 1.2, 0.25}

// Stats are the combined effect of the archetype, every upgrade and relic,
// and active synergies. They're rebuilt from scratch whenever mods change.
type Stats struct {
	MaxHP                                                            float64
	Damage, DmgMult                                                  float64
	CritChance, CritMult                                             float64
	Pierce                                                           int
	SpeedMult, SizeMult                                              float64
	ExtraBalls                                                       int
	SplitChance                                                      float64
	BurnChance, FreezeChance, ChainChance, ExplodeChance, VoidChance float64
	ChainCount                                                       int
	ExplodeRadius                                                    float64
	GoldMult, Interest, Luck                                         float64
	PaddleWidth                                                      float64
	BarrierPerLevel                                                  int
	LaserInterval                                                    float64
	ComboDmg                                                         float64
	Crescendo                                                        float64 // bonus note per combo step
	BallLossDmg                                                      float64 // extra health lost when the orb is dropped
	DropBarrier                                                      int     // barrier charges gained when a life is lost
	LifeHeal                                                         float64 // health restored when a life is lost
	Lifesteal                                                        int
	Magnet                                                           bool
	MoltenEvery, EchoEvery, StaticEvery                              int
	ThunderCrown, BloodMoon, Phoenix, Momentum, HeavyPaddle          bool
	GoldenPaddle, PyroAsh, StormCaller, Philosopher, Wildfire        bool
	Shatter, EagleEye, MeteorHeart, FrozenBonus, PrismOrb, Hydra     bool
	BarrierRegen, CritPierce, VoidDouble, GoldPerBrick, ChainRecast  bool
	StartElement                                                     string
	BrickHPMult                                                      float64 // curses: extra brick health
	ScrollSlots, CurseCount                                          int
	ScrollKeep, ScribeChance, CurseDmg, ChallengeGold                float64
	CurseWard                                                        bool
	StartGold                                                        float64 // extra gold at the start of a run
	FreeRerolls                                                      int

	// added in 0.5
	Ricochet, BurnMult, ChainMult, BlastMult, FreezeDur, BossDmg  float64
	ComboTime, ShopDiscount, CoinChance, LaserDmg, KillSplash     float64
	HiveDmg, LoneDmg, PowerDur, EchoChance, HealOnClear, CritHeal float64
	CritCoin, ExecuteBelow, Avarice, OpeningCrit, BattleHymn      float64
	ExtraChoices, ShopRerolls, SunderMult                         int
	StaticPaddle, FreezeSpread, Blessed, ThornWard, OpeningPower  bool
	BlackHole, InfiniteCombo, DragonHeart                         bool
	ClassIgnite                                                   int // ignite every Nth hit (Pyromancer)
	ClassParry, ClassRage, ClassAttune, ClassOverclock            bool
	ClassHighRoller, ClassStoke, ClassStatic                      bool
	// fusions
	PlasmaStorm, SteamEngine, CryoCharge, GoldRush, Thunderclap    bool
	Berserker, TeslaCoil, VoidPrism, Sanctuary, Cataclysm, Volcano bool
	Hawkstrike, AbsoluteZero, Haymaker                             bool
	// origin mastery
	Aspect                                                                     string
	SigHalo, SigBloodPrice, SigLens, SigCoolant, SigDice, SigEmber, SigConduit bool
}

type Mod struct {
	ID     string
	Name   string
	Desc   string
	Rarity int
	Relic  bool
	Tags   []string
	Max    int // max stacks (0 = unlimited)
	Apply  func(s *Stats)
	Icon   string // glyph drawn on cards, badges and the Codex (icons.go)
	Unlock int    // player rank required before it can appear
	Needs  string // only offered once this mod is owned
	Curse  bool   // a curse: never offered, removed by cleansing
	Origin string // signature item: only this origin, from Mastery 3
}

var modList = []*Mod{
	// ---------------- upgrades ----------------
	{ID: "heavy", Icon: "anvil", Name: "Heavy Core", Desc: "+0.5 orb damage.", Rarity: Common, Tags: []string{"power"}, Apply: func(s *Stats) { s.Damage += 0.5 }},
	{ID: "sharp", Icon: "sword", Name: "Keen Edge", Desc: "+6% critical chance.", Rarity: Common, Tags: []string{"crit"}, Apply: func(s *Stats) { s.CritChance += 0.06 }},
	{ID: "wide", Icon: "paddle", Name: "Broad Paddle", Desc: "+15% paddle width.", Rarity: Common, Tags: []string{"guard"}, Max: 4, Apply: func(s *Stats) { s.PaddleWidth += 0.15 }},
	{ID: "temper", Icon: "comet", Name: "Tempered Orb", Desc: "+10% orb speed and +0.3 damage.", Rarity: Common, Tags: []string{"power"}, Apply: func(s *Stats) { s.SpeedMult += 0.1; s.Damage += 0.3 }},
	{ID: "goldleaf", Icon: "leaf", Name: "Gold Leaf", Desc: "+20% gold from every source.", Rarity: Common, Tags: []string{"wealth"}, Apply: func(s *Stats) { s.GoldMult += 0.2 }},
	{ID: "kindling", Icon: "flame", Name: "Kindling", Desc: "+12% chance to ignite bricks. Burning bricks take damage over time.", Rarity: Common, Tags: []string{"fire"}, Apply: func(s *Stats) { s.BurnChance += 0.12 }},
	{ID: "static", Icon: "bolt", Name: "Static Charge", Desc: "+12% chance for lightning to arc to nearby bricks.", Rarity: Common, Tags: []string{"storm"}, Apply: func(s *Stats) { s.ChainChance += 0.12 }},
	{ID: "frost", Icon: "snow", Name: "Frost Touch", Desc: "+12% chance to freeze bricks. Frozen bricks take +50% damage.", Rarity: Common, Tags: []string{"frost"}, Apply: func(s *Stats) { s.FreezeChance += 0.12 }},
	{ID: "fort", Icon: "heart", Name: "Fortitude", Desc: "+25 max health.", Rarity: Common, Tags: []string{"guard"}, Apply: func(s *Stats) { s.MaxHP += 25 }},
	{ID: "bigorb", Icon: "orb", Name: "Swollen Orb", Desc: "+20% orb size and +0.2 damage.", Rarity: Common, Tags: []string{"power"}, Max: 4, Apply: func(s *Stats) { s.SizeMult += 0.2; s.Damage += 0.2 }},
	{ID: "barrier", Icon: "shield", Name: "Warding Line", Desc: "Start each level with +1 barrier charge.", Rarity: Uncommon, Tags: []string{"guard"}, Apply: func(s *Stats) { s.BarrierPerLevel++ }},
	{ID: "split", Icon: "split", Name: "Split Shot", Desc: "15% chance a paddle bounce splits off an extra orb.", Rarity: Uncommon, Tags: []string{"swarm"}, Apply: func(s *Stats) { s.SplitChance += 0.15 }},
	{ID: "twin", Icon: "twinorb", Name: "Twin Orbs", Desc: "Launch one extra orb.", Rarity: Rare, Tags: []string{"swarm"}, Max: 3, Apply: func(s *Stats) { s.ExtraBalls++ }},
	{ID: "brutal", Icon: "fist", Name: "Brutal Strikes", Desc: "Critical hits deal +75% damage.", Rarity: Uncommon, Tags: []string{"crit"}, Apply: func(s *Stats) { s.CritMult += 0.75 }},
	{ID: "blast", Icon: "bomb", Name: "Blast Core", Desc: "+10% chance to explode on impact.", Rarity: Uncommon, Tags: []string{"blast"}, Apply: func(s *Stats) { s.ExplodeChance += 0.1 }},
	{ID: "pierce", Icon: "arrow", Name: "Piercing Orb", Desc: "The orb passes through +1 brick before bouncing.", Rarity: Uncommon, Tags: []string{"power"}, Max: 3, Apply: func(s *Stats) { s.Pierce++ }},
	{ID: "voidtouch", Icon: "swirl", Name: "Void Touch", Desc: "+15% chance to ignore armour and shields.", Rarity: Uncommon, Tags: []string{"void"}, Apply: func(s *Stats) { s.VoidChance += 0.15 }},
	{ID: "conductor", Icon: "chain", Name: "Conductor", Desc: "Lightning arcs strike +2 more bricks.", Rarity: Uncommon, Tags: []string{"storm"}, Apply: func(s *Stats) { s.ChainCount += 2 }},
	{ID: "wildfire", Icon: "blaze", Name: "Wildfire", Desc: "Burning bricks spread fire to their neighbours when destroyed.", Rarity: Rare, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.Wildfire = true }},
	{ID: "shatter", Icon: "shatter", Name: "Shatterpoint", Desc: "Frozen bricks shatter, damaging everything around them.", Rarity: Rare, Tags: []string{"frost"}, Max: 1, Apply: func(s *Stats) { s.Shatter = true }},
	{ID: "comboeng", Icon: "gear", Name: "Combo Engine", Desc: "+2% damage per combo step (up to +100%).", Rarity: Uncommon, Tags: []string{"combo"}, Apply: func(s *Stats) { s.ComboDmg += 0.02 }},
	{ID: "crescendo", Icon: "note", Name: "Crescendo", Desc: "Orb hits ring a second note: +2% of the hit per combo step, up to +40% (beyond other combo caps). Damage numbers swell with your combo.", Rarity: Uncommon, Tags: []string{"combo"}, Max: 2, Apply: func(s *Stats) { s.Crescendo += 0.02 }},
	{ID: "luckyc", Icon: "clover", Name: "Lucky Charm", Desc: "+50% power-up drop chance.", Rarity: Uncommon, Tags: []string{"wealth"}, Apply: func(s *Stats) { s.Luck += 0.5 }},
	{ID: "interest", Icon: "coinstack", Name: "Compound Interest", Desc: "Earn 10% interest on your gold after each level (max 30).", Rarity: Rare, Tags: []string{"wealth"}, Max: 2, Apply: func(s *Stats) { s.Interest += 0.1 }},
	{ID: "fuse", Icon: "fuse", Name: "Long Fuse", Desc: "Explosions are 50% larger.", Rarity: Uncommon, Tags: []string{"blast"}, Apply: func(s *Stats) { s.ExplodeRadius += 0.5 }},
	{ID: "laser", Icon: "laser", Name: "Laser Mount", Desc: "Your paddle fires twin lasers automatically.", Rarity: Rare, Tags: []string{"tech"}, Max: 3, Apply: func(s *Stats) {
		if s.LaserInterval == 0 {
			s.LaserInterval = 2.4
		} else {
			s.LaserInterval *= 0.65
		}
	}},
	{ID: "eagle", Icon: "eye", Name: "Eagle Eye", Desc: "Critical hits splash damage onto adjacent bricks.", Rarity: Rare, Tags: []string{"crit"}, Max: 1, Apply: func(s *Stats) { s.EagleEye = true }},
	{ID: "momentum", Icon: "comet", Name: "Momentum", Desc: "Each brick hit without touching the paddle adds +4% speed and damage.", Rarity: Rare, Tags: []string{"combo"}, Max: 1, Apply: func(s *Stats) { s.Momentum = true }},
	{ID: "padding", Icon: "feather", Name: "Shock Padding", Desc: "Losing a life leaves a barrier charge behind.", Rarity: Common, Tags: []string{"guard"}, Max: 3, Apply: func(s *Stats) { s.DropBarrier++ }},

	// ---------------- relics ----------------
	{ID: "molten", Icon: "volcano", Name: "Molten Core", Desc: "Every 50 orb impacts unleash a huge explosion.", Rarity: Uncommon, Relic: true, Tags: []string{"blast"}, Max: 1, Apply: func(s *Stats) { s.MoltenEvery = 50 }},
	{ID: "glass", Icon: "glass", Name: "Glass Cannon", Desc: "Damage ×1.6, but your paddle is 25% narrower.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.DmgMult *= 1.6; s.PaddleWidth -= 0.25 }},
	{ID: "magnet", Icon: "magnet", Name: "Magnet Core", Desc: "Gold and power-ups drift toward your paddle.", Rarity: Common, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.Magnet = true }},
	{ID: "echo", Icon: "wave", Name: "Echo Chamber", Desc: "Every 5th impact strikes twice.", Rarity: Rare, Relic: true, Tags: []string{"combo"}, Max: 1, Apply: func(s *Stats) { s.EchoEvery = 5 }},
	{ID: "thunder", Icon: "crown", Name: "Thunder Crown", Desc: "Critical hits call down chain lightning.", Rarity: Epic, Relic: true, Tags: []string{"crit", "storm"}, Max: 1, Apply: func(s *Stats) { s.ThunderCrown = true }},
	{ID: "bloodmoon", Icon: "moon", Name: "Blood Moon", Desc: "Below half health, damage ×1.75.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.BloodMoon = true }},
	{ID: "inferno", Icon: "flame", Name: "Inferno Core", Desc: "+20% chance to ignite bricks.", Rarity: Uncommon, Relic: true, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.BurnChance += 0.2 }},
	{ID: "staticfield", Icon: "storm", Name: "Static Field", Desc: "Every 15 impacts, lightning strikes a random brick.", Rarity: Uncommon, Relic: true, Tags: []string{"storm"}, Max: 1, Apply: func(s *Stats) { s.StaticEvery = 15 }},
	{ID: "greed", Icon: "coin", Name: "Greed", Desc: "+25% gold.", Rarity: Common, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldMult += 0.25 }},
	{ID: "voidbounce", Icon: "swirl", Name: "Void Bounce", Desc: "+20% chance to ignore armour and shields.", Rarity: Uncommon, Relic: true, Tags: []string{"void"}, Max: 1, Apply: func(s *Stats) { s.VoidChance += 0.2 }},
	{ID: "phoenix", Icon: "wing", Name: "Phoenix Feather", Desc: "Once per run, survive a fatal blow with 50 health.", Rarity: Legendary, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.Phoenix = true }},
	{ID: "prism", Icon: "gem", Name: "Prism Orb", Desc: "Every launch also releases a phantom orb.", Rarity: Epic, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.PrismOrb = true }},
	{ID: "meteor", Icon: "comet", Name: "Meteor Heart", Desc: "Your orb explodes on every brick it hits.", Rarity: Epic, Relic: true, Tags: []string{"blast"}, Max: 1, Apply: func(s *Stats) { s.MeteorHeart = true }},
	{ID: "frozenheart", Icon: "iceheart", Name: "Frozen Heart", Desc: "+25% freeze chance. Frozen bricks take double damage.", Rarity: Uncommon, Relic: true, Tags: []string{"frost"}, Max: 1, Apply: func(s *Stats) { s.FreezeChance += 0.25; s.FrozenBonus = true }},
	{ID: "angel", Icon: "halo", Name: "Guardian Halo", Desc: "+2 barrier charges every level.", Rarity: Uncommon, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.BarrierPerLevel += 2 }},
	{ID: "goldpaddle", Icon: "paddle", Name: "Golden Paddle", Desc: "Each paddle bounce earns 1 gold.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldenPaddle = true }},
	{ID: "hydra", Icon: "hydra", Name: "Hydra Seed", Desc: "Every 30 bricks destroyed spawns an extra orb.", Rarity: Epic, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.Hydra = true }},
	{ID: "leech", Icon: "fang", Name: "Leech Orb", Desc: "Heal 1 health for every 6 bricks destroyed.", Rarity: Rare, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.Lifesteal = 6 }},
	{ID: "clover", Icon: "clover", Name: "Four-Leaf Clover", Desc: "+5% crit chance and +30% power-up drops.", Rarity: Uncommon, Relic: true, Tags: []string{"wealth", "crit"}, Max: 1, Apply: func(s *Stats) { s.CritChance += 0.05; s.Luck += 0.3 }},
	{ID: "philo", Icon: "gem", Name: "Philosopher's Stone", Desc: "Crystals drop triple gold, and every brick may drop gold.", Rarity: Legendary, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.Philosopher = true }},
	{ID: "storm", Icon: "storm", Name: "Storm Caller", Desc: "Lightning arcs can jump a second time.", Rarity: Legendary, Relic: true, Tags: []string{"storm"}, Max: 1, Apply: func(s *Stats) { s.StormCaller = true }},
	{ID: "pyro", Icon: "urn", Name: "Pyromancer's Ash", Desc: "Burning bricks take double damage from everything.", Rarity: Epic, Relic: true, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.PyroAsh = true }},
	{ID: "overclock", Icon: "gear", Name: "Overclock", Desc: "Orb speed +20%, damage +40%.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.SpeedMult += 0.2; s.DmgMult *= 1.4 }},
	{ID: "heavypad", Icon: "battery", Name: "Charged Paddle", Desc: "Paddle bounces charge the orb: its next hit is a guaranteed critical.", Rarity: Rare, Relic: true, Tags: []string{"crit"}, Max: 1, Apply: func(s *Stats) { s.HeavyPaddle = true }},
	{ID: "idol", Icon: "mask", Name: "Cursed Idol", Desc: "Double gold, but −30 max health.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldMult += 1; s.MaxHP -= 30 }},
	{ID: "mirror", Icon: "mirror", Name: "Mirror Shard", Desc: "+10% chance a paddle bounce splits off an extra orb.", Rarity: Uncommon, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.SplitChance += 0.1 }},
	{ID: "singularity", Icon: "blackhole", Name: "Singularity", Desc: "Void hits deal triple damage and +10% void chance.", Rarity: Mythic, Relic: true, Tags: []string{"void"}, Max: 1, Apply: func(s *Stats) { s.VoidChance += 0.1; s.VoidDouble = true }},
	// ---------------- upgrades (0.5) ----------------
	{ID: "afterburn", Icon: "comet", Name: "Afterburner", Desc: "+12% orb speed and +12% damage.", Rarity: Common, Tags: []string{"power"}, Max: 4, Unlock: 1, Apply: func(s *Stats) { s.SpeedMult += 0.12; s.DmgMult *= 1.12 }},
	{ID: "ricochet", Icon: "ricochet", Name: "Ricochet", Desc: "Each wall bounce adds +8% damage until the orb returns to your paddle.", Rarity: Common, Tags: []string{"combo"}, Max: 3, Unlock: 1, Apply: func(s *Stats) { s.Ricochet += 0.08 }},
	{ID: "accel", Icon: "drop", Name: "Accelerant", Desc: "Burning deals +50% damage.", Rarity: Common, Tags: []string{"fire"}, Max: 4, Apply: func(s *Stats) { s.BurnMult += 0.5 }},
	{ID: "overcharge", Icon: "battery", Name: "Overcharge", Desc: "Lightning deals +40% damage.", Rarity: Common, Tags: []string{"storm"}, Max: 4, Apply: func(s *Stats) { s.ChainMult += 0.4 }},
	{ID: "nitro", Icon: "flask", Name: "Nitro", Desc: "Explosions deal +40% damage.", Rarity: Common, Tags: []string{"blast"}, Max: 4, Apply: func(s *Stats) { s.BlastMult += 0.4 }},
	{ID: "deepfreeze", Icon: "snow", Name: "Deep Freeze", Desc: "+8% freeze chance; bricks stay frozen 2s longer.", Rarity: Common, Tags: []string{"frost"}, Max: 3, Unlock: 2, Apply: func(s *Stats) { s.FreezeChance += 0.08; s.FreezeDur += 2 }},
	{ID: "sunder", Icon: "axe", Name: "Sunder", Desc: "Armoured bricks crack twice as fast, and +5% void chance.", Rarity: Uncommon, Tags: []string{"void"}, Max: 2, Unlock: 2, Apply: func(s *Stats) { s.SunderMult++; s.VoidChance += 0.05 }},
	{ID: "slayer", Icon: "sword", Name: "Giant Slayer", Desc: "+30% damage to elites and bosses.", Rarity: Uncommon, Tags: []string{"power"}, Max: 3, Unlock: 3, Apply: func(s *Stats) { s.BossDmg += 0.3 }},
	{ID: "patience", Icon: "hourglass", Name: "Patience", Desc: "Your combo lasts 1 second longer before fading.", Rarity: Common, Tags: []string{"combo"}, Max: 3, Apply: func(s *Stats) { s.ComboTime += 1 }},
	{ID: "secondwind", Icon: "feather", Name: "Second Wind", Desc: "Heal 6 health after every cleared chamber.", Rarity: Common, Tags: []string{"guard"}, Max: 4, Apply: func(s *Stats) { s.HealOnClear += 6 }},
	{ID: "bargain", Icon: "tag", Name: "Bargain Hunter", Desc: "Shop prices −10%.", Rarity: Common, Tags: []string{"wealth"}, Max: 3, Unlock: 1, Apply: func(s *Stats) { s.ShopDiscount += 0.1 }},
	{ID: "scavenger", Icon: "lens", Name: "Scavenger", Desc: "Bricks are 8% more likely to drop gold.", Rarity: Common, Tags: []string{"wealth"}, Max: 3, Apply: func(s *Stats) { s.CoinChance += 0.08 }},
	{ID: "laserfocus", Icon: "target", Name: "Focusing Lens", Desc: "Lasers deal +60% damage.", Rarity: Uncommon, Tags: []string{"tech"}, Max: 3, Needs: "laser", Apply: func(s *Stats) { s.LaserDmg += 0.6 }},
	{ID: "rapidfire", Icon: "laser", Name: "Rapid Fire", Desc: "Lasers fire 25% faster.", Rarity: Uncommon, Tags: []string{"tech"}, Max: 3, Needs: "laser", Apply: func(s *Stats) {
		if s.LaserInterval > 0 {
			s.LaserInterval *= 0.75
		}
	}},
	{ID: "shrapnel", Icon: "burst", Name: "Shrapnel", Desc: "Destroyed bricks hurl shards that deal 40% damage to their neighbours.", Rarity: Uncommon, Tags: []string{"blast"}, Max: 2, Unlock: 2, Apply: func(s *Stats) { s.KillSplash += 0.4 }},
	{ID: "discharge", Icon: "bolt", Name: "Static Discharge", Desc: "Every paddle bounce zaps the nearest brick with lightning.", Rarity: Uncommon, Tags: []string{"storm"}, Max: 1, Unlock: 3, Apply: func(s *Stats) { s.StaticPaddle = true }},
	{ID: "frostbite", Icon: "fang", Name: "Frostbite", Desc: "Hitting a frozen brick spreads the freeze to its neighbours.", Rarity: Uncommon, Tags: []string{"frost"}, Max: 1, Unlock: 4, Apply: func(s *Stats) { s.FreezeSpread = true }},
	{ID: "hive", Icon: "hive", Name: "Hive Mind", Desc: "+8% damage for every extra orb in play (up to +64%).", Rarity: Uncommon, Tags: []string{"swarm"}, Max: 2, Unlock: 3, Apply: func(s *Stats) { s.HiveDmg += 0.08 }},
	{ID: "precision", Icon: "target", Name: "Precision", Desc: "+4% critical chance and +25% critical damage.", Rarity: Common, Tags: []string{"crit"}, Apply: func(s *Stats) { s.CritChance += 0.04; s.CritMult += 0.25 }},
	{ID: "extended", Icon: "hourglass", Name: "Extended Play", Desc: "Timed power-ups last 30% longer.", Rarity: Common, Tags: []string{"tech"}, Max: 3, Apply: func(s *Stats) { s.PowerDur += 0.3 }},
	{ID: "echochip", Icon: "wave", Name: "Echo Chip", Desc: "8% chance for any hit to strike a second time.", Rarity: Uncommon, Tags: []string{"combo"}, Max: 3, Unlock: 4, Apply: func(s *Stats) { s.EchoChance += 0.08 }},

	// ---------------- relics (0.5) ----------------
	{ID: "scholar", Icon: "book", Name: "Scholar's Lens", Desc: "Every reward offers one extra choice.", Rarity: Epic, Relic: true, Tags: []string{"wealth"}, Max: 1, Unlock: 5, Apply: func(s *Stats) { s.ExtraChoices++ }},
	{ID: "merchant", Icon: "tag", Name: "Merchant's Seal", Desc: "Shop prices −25% and one free reroll in every shop.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Unlock: 2, Apply: func(s *Stats) { s.ShopDiscount += 0.25; s.ShopRerolls++ }},
	{ID: "sandglass", Icon: "hourglass", Name: "Eternal Sandglass", Desc: "Your combo lasts twice as long.", Rarity: Uncommon, Relic: true, Tags: []string{"combo"}, Max: 1, Unlock: 1, Apply: func(s *Stats) { s.ComboTime += 3 }},
	{ID: "thorns", Icon: "crown", Name: "Crown of Thorns", Desc: "Whenever a barrier saves your orb, it explodes along the bottom row.", Rarity: Rare, Relic: true, Tags: []string{"guard", "blast"}, Max: 1, Unlock: 3, Apply: func(s *Stats) { s.ThornWard = true }},
	{ID: "bane", Icon: "axe", Name: "Giant's Bane", Desc: "+60% damage to elites and bosses.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Unlock: 2, Apply: func(s *Stats) { s.BossDmg += 0.6 }},
	{ID: "vampfang", Icon: "fang", Name: "Vampire Fang", Desc: "Critical hits heal 1 health.", Rarity: Rare, Relic: true, Tags: []string{"crit", "guard"}, Max: 1, Unlock: 4, Apply: func(s *Stats) { s.CritHeal += 1 }},
	{ID: "lodestone", Icon: "magnet", Name: "Lodestone", Desc: "+15% damage for every extra orb in play (up to +90%).", Rarity: Epic, Relic: true, Tags: []string{"swarm"}, Max: 1, Unlock: 6, Apply: func(s *Stats) { s.HiveDmg += 0.15 }},
	{ID: "lonestar", Icon: "star", Name: "Lone Star", Desc: "While only one orb is in play, it deals +50% damage.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Unlock: 2, Apply: func(s *Stats) { s.LoneDmg += 0.5 }},
	{ID: "headsman", Icon: "axe", Name: "Headsman's Axe", Desc: "Bricks below 20% health are destroyed instantly.", Rarity: Epic, Relic: true, Tags: []string{"crit"}, Max: 1, Unlock: 7, Apply: func(s *Stats) { s.ExecuteBelow = 0.2 }},
	{ID: "shrapshell", Icon: "burst", Name: "Shrapnel Shell", Desc: "Destroyed bricks deal 60% damage to their neighbours.", Rarity: Uncommon, Relic: true, Tags: []string{"blast"}, Max: 1, Unlock: 3, Apply: func(s *Stats) { s.KillSplash += 0.6 }},
	{ID: "midasglove", Icon: "fist", Name: "Midas Glove", Desc: "Critical hits have a 40% chance to drop a coin.", Rarity: Uncommon, Relic: true, Tags: []string{"wealth", "crit"}, Max: 1, Unlock: 1, Apply: func(s *Stats) { s.CritCoin += 0.4 }},
	{ID: "seraph", Icon: "wing", Name: "Seraph Wing", Desc: "Heal 12 health after every cleared chamber.", Rarity: Uncommon, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.HealOnClear += 12 }},
	{ID: "tesla", Icon: "battery", Name: "Overcharger", Desc: "Lightning deals +60% damage and arcs to +1 brick.", Rarity: Uncommon, Relic: true, Tags: []string{"storm"}, Max: 1, Unlock: 2, Apply: func(s *Stats) { s.ChainMult += 0.6; s.ChainCount++ }},
	{ID: "nitroflask", Icon: "flask", Name: "Nitro Flask", Desc: "Explosions deal +60% damage and are 20% larger.", Rarity: Uncommon, Relic: true, Tags: []string{"blast"}, Max: 1, Unlock: 1, Apply: func(s *Stats) { s.BlastMult += 0.6; s.ExplodeRadius += 0.3 }},
	{ID: "kinetic", Icon: "ricochet", Name: "Kinetic Battery", Desc: "Each wall bounce adds +15% damage until the orb returns to your paddle.", Rarity: Rare, Relic: true, Tags: []string{"combo"}, Max: 1, Unlock: 4, Apply: func(s *Stats) { s.Ricochet += 0.15 }},
	{ID: "gambit", Icon: "target", Name: "Opening Gambit", Desc: "The first hit after every launch is a guaranteed critical.", Rarity: Uncommon, Relic: true, Tags: []string{"crit"}, Max: 1, Unlock: 1, Apply: func(s *Stats) { s.OpeningCrit = 1 }},
	{ID: "blessed", Icon: "halo", Name: "Blessed Charm", Desc: "Harmful power-ups never drop.", Rarity: Uncommon, Relic: true, Tags: []string{"guard"}, Max: 1, Unlock: 2, Apply: func(s *Stats) { s.Blessed = true }},
	{ID: "timecrystal", Icon: "gem", Name: "Time Crystal", Desc: "Timed power-ups last 60% longer.", Rarity: Common, Relic: true, Tags: []string{"tech"}, Max: 1, Apply: func(s *Stats) { s.PowerDur += 0.6 }},
	{ID: "hymn", Icon: "drum", Name: "Battle Hymn", Desc: "For the first 8 seconds of every chamber, deal double damage.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Unlock: 5, Apply: func(s *Stats) { s.BattleHymn = 8 }},
	{ID: "avarice", Icon: "ring", Name: "Ring of Avarice", Desc: "+1% damage for every 10 gold you carry (up to +60%).", Rarity: Epic, Relic: true, Tags: []string{"wealth", "power"}, Max: 1, Unlock: 6, Apply: func(s *Stats) { s.Avarice = 0.6 }},
	{ID: "openact", Icon: "star", Name: "Overture", Desc: "Begin every chamber with a random helpful power-up.", Rarity: Rare, Relic: true, Tags: []string{"tech"}, Max: 1, Unlock: 3, Apply: func(s *Stats) { s.OpeningPower = true }},
	{ID: "blackhole", Icon: "blackhole", Name: "Event Horizon", Desc: "Every 40 bricks destroyed, a black hole devours everything nearby.", Rarity: Mythic, Relic: true, Tags: []string{"void", "blast"}, Max: 1, Unlock: 9, Apply: func(s *Stats) { s.BlackHole = true }},
	{ID: "infinity", Icon: "infinity", Name: "Infinity Loop", Desc: "Combo damage bonuses have no cap.", Rarity: Mythic, Relic: true, Tags: []string{"combo"}, Max: 1, Unlock: 10, Apply: func(s *Stats) { s.InfiniteCombo = true; s.ComboDmg += 0.01 }},
	{ID: "dragonheart", Icon: "dragonheart", Name: "Dragon Heart", Desc: "Fire burns twice as fast, and every hit on a burning brick spreads the flames.", Rarity: Legendary, Relic: true, Tags: []string{"fire"}, Max: 1, Unlock: 8, Apply: func(s *Stats) { s.DragonHeart = true; s.BurnMult += 1 }},
	{ID: "excalibur", Icon: "sword", Name: "Excalibur", Desc: "+10% critical chance and critical hits deal +150% damage.", Rarity: Legendary, Relic: true, Tags: []string{"crit"}, Max: 1, Unlock: 7, Apply: func(s *Stats) { s.CritChance += 0.1; s.CritMult += 1.5 }},
	{ID: "phylactery", Icon: "urn", Name: "Phylactery", Desc: "+40 max health, but dropping the orb also costs 10 health.", Rarity: Rare, Relic: true, Tags: []string{"guard"}, Max: 1, Unlock: 5, Apply: func(s *Stats) { s.MaxHP += 40; s.BallLossDmg += 10 }},
}

var modByID = func() map[string]*Mod {
	m := map[string]*Mod{}
	for _, md := range modList {
		m[md.ID] = md
	}
	for _, md := range curseList {
		m[md.ID] = md
	}
	return m
}()

// Synergies activate when you own 3+ mods sharing a tag.
type Synergy struct {
	Tag, Name, Desc string
	Apply           func(s *Stats)
}

var synergies = []Synergy{
	{"fire", "Inferno", "Burning bricks explode when destroyed.", func(s *Stats) { s.Wildfire = true; s.ExplodeChance += 0.05 }},
	{"storm", "Tempest", "+2 arc targets, +15% arc chance, arcs re-strike.", func(s *Stats) { s.ChainCount += 2; s.ChainChance += 0.15; s.ChainRecast = true }},
	{"wealth", "Midas", "+1 gold per brick and doubled interest.", func(s *Stats) { s.GoldPerBrick = true; s.Interest *= 2 }},
	{"crit", "Executioner", "Critical hits pierce and deal +50% damage.", func(s *Stats) { s.CritPierce = true; s.CritMult += 0.5 }},
	{"swarm", "Swarm", "+1 orb at launch.", func(s *Stats) { s.ExtraBalls++ }},
	{"blast", "Demolition", "Explosions are much larger.", func(s *Stats) { s.ExplodeRadius += 0.8 }},
	{"frost", "Permafrost", "Frozen bricks shatter and take double damage.", func(s *Stats) { s.Shatter = true; s.FrozenBonus = true }},
	{"guard", "Bastion", "Barrier recharges every 20 seconds.", func(s *Stats) { s.BarrierRegen = true }},
	{"void", "Abyssal", "Void hits deal triple damage.", func(s *Stats) { s.VoidDouble = true }},
	{"power", "Juggernaut", "+1 damage and +15% orb size.", func(s *Stats) { s.Damage += 1; s.SizeMult += 0.15 }},
	{"combo", "Overdrive", "Combo never decays while an orb is in play.", func(s *Stats) { s.ComboDmg += 0.01 }},
	{"tech", "Arsenal", "Lasers fire 40% faster.", func(s *Stats) {
		if s.LaserInterval > 0 {
			s.LaserInterval *= 0.6
		}
	}},
}

var tagColors = map[string]color.RGBA{
	"fire": {255, 120, 40, 255}, "storm": {120, 200, 255, 255}, "wealth": {255, 210, 70, 255},
	"crit": {255, 80, 80, 255}, "swarm": {255, 160, 230, 255}, "blast": {255, 170, 60, 255},
	"frost": {150, 230, 255, 255}, "guard": {120, 230, 140, 255}, "void": {170, 110, 255, 255},
	"power": {240, 240, 240, 255}, "combo": {255, 230, 120, 255}, "tech": {200, 200, 255, 255},
	"curse": {200, 60, 120, 255},
}

func sortedTags(m map[string]int) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
