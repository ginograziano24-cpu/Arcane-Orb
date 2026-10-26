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
	BallLossDmg                                                      float64
	Lifesteal                                                        int
	Magnet                                                           bool
	MoltenEvery, EchoEvery, StaticEvery                              int
	ThunderCrown, BloodMoon, Phoenix, Momentum, HeavyPaddle          bool
	GoldenPaddle, PyroAsh, StormCaller, Philosopher, Wildfire        bool
	Shatter, EagleEye, MeteorHeart, FrozenBonus, PrismOrb, Hydra     bool
	BarrierRegen, CritPierce, VoidDouble, GoldPerBrick, ChainRecast  bool
	StartElement                                                     string
	FreeRerolls                                                      int
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
}

var modList = []*Mod{
	// ---------------- upgrades ----------------
	{ID: "heavy", Name: "Heavy Core", Desc: "+0.5 orb damage.", Rarity: Common, Tags: []string{"power"}, Apply: func(s *Stats) { s.Damage += 0.5 }},
	{ID: "sharp", Name: "Keen Edge", Desc: "+6% critical chance.", Rarity: Common, Tags: []string{"crit"}, Apply: func(s *Stats) { s.CritChance += 0.06 }},
	{ID: "wide", Name: "Broad Paddle", Desc: "+15% paddle width.", Rarity: Common, Tags: []string{"guard"}, Max: 4, Apply: func(s *Stats) { s.PaddleWidth += 0.15 }},
	{ID: "temper", Name: "Tempered Orb", Desc: "+10% orb speed and +0.3 damage.", Rarity: Common, Tags: []string{"power"}, Apply: func(s *Stats) { s.SpeedMult += 0.1; s.Damage += 0.3 }},
	{ID: "goldleaf", Name: "Gold Leaf", Desc: "+20% gold from every source.", Rarity: Common, Tags: []string{"wealth"}, Apply: func(s *Stats) { s.GoldMult += 0.2 }},
	{ID: "kindling", Name: "Kindling", Desc: "+12% chance to ignite bricks. Burning bricks take damage over time.", Rarity: Common, Tags: []string{"fire"}, Apply: func(s *Stats) { s.BurnChance += 0.12 }},
	{ID: "static", Name: "Static Charge", Desc: "+12% chance for lightning to arc to nearby bricks.", Rarity: Common, Tags: []string{"storm"}, Apply: func(s *Stats) { s.ChainChance += 0.12 }},
	{ID: "frost", Name: "Frost Touch", Desc: "+12% chance to freeze bricks. Frozen bricks take +50% damage.", Rarity: Common, Tags: []string{"frost"}, Apply: func(s *Stats) { s.FreezeChance += 0.12 }},
	{ID: "fort", Name: "Fortitude", Desc: "+25 max health.", Rarity: Common, Tags: []string{"guard"}, Apply: func(s *Stats) { s.MaxHP += 25 }},
	{ID: "bigorb", Name: "Swollen Orb", Desc: "+20% orb size and +0.2 damage.", Rarity: Common, Tags: []string{"power"}, Max: 4, Apply: func(s *Stats) { s.SizeMult += 0.2; s.Damage += 0.2 }},
	{ID: "barrier", Name: "Warding Line", Desc: "Start each level with +1 barrier charge.", Rarity: Uncommon, Tags: []string{"guard"}, Apply: func(s *Stats) { s.BarrierPerLevel++ }},
	{ID: "split", Name: "Split Shot", Desc: "15% chance a paddle bounce splits off an extra orb.", Rarity: Uncommon, Tags: []string{"swarm"}, Apply: func(s *Stats) { s.SplitChance += 0.15 }},
	{ID: "twin", Name: "Twin Orbs", Desc: "Launch one extra orb.", Rarity: Rare, Tags: []string{"swarm"}, Max: 3, Apply: func(s *Stats) { s.ExtraBalls++ }},
	{ID: "brutal", Name: "Brutal Strikes", Desc: "Critical hits deal +75% damage.", Rarity: Uncommon, Tags: []string{"crit"}, Apply: func(s *Stats) { s.CritMult += 0.75 }},
	{ID: "blast", Name: "Blast Core", Desc: "+10% chance to explode on impact.", Rarity: Uncommon, Tags: []string{"blast"}, Apply: func(s *Stats) { s.ExplodeChance += 0.1 }},
	{ID: "pierce", Name: "Piercing Orb", Desc: "The orb passes through +1 brick before bouncing.", Rarity: Uncommon, Tags: []string{"power"}, Max: 3, Apply: func(s *Stats) { s.Pierce++ }},
	{ID: "voidtouch", Name: "Void Touch", Desc: "+15% chance to ignore armour and shields.", Rarity: Uncommon, Tags: []string{"void"}, Apply: func(s *Stats) { s.VoidChance += 0.15 }},
	{ID: "conductor", Name: "Conductor", Desc: "Lightning arcs strike +2 more bricks.", Rarity: Uncommon, Tags: []string{"storm"}, Apply: func(s *Stats) { s.ChainCount += 2 }},
	{ID: "wildfire", Name: "Wildfire", Desc: "Burning bricks spread fire to their neighbours when destroyed.", Rarity: Rare, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.Wildfire = true }},
	{ID: "shatter", Name: "Shatterpoint", Desc: "Frozen bricks shatter, damaging everything around them.", Rarity: Rare, Tags: []string{"frost"}, Max: 1, Apply: func(s *Stats) { s.Shatter = true }},
	{ID: "comboeng", Name: "Combo Engine", Desc: "+2% damage per combo step (up to +100%).", Rarity: Uncommon, Tags: []string{"combo"}, Apply: func(s *Stats) { s.ComboDmg += 0.02 }},
	{ID: "luckyc", Name: "Lucky Charm", Desc: "+50% power-up drop chance.", Rarity: Uncommon, Tags: []string{"wealth"}, Apply: func(s *Stats) { s.Luck += 0.5 }},
	{ID: "interest", Name: "Compound Interest", Desc: "Earn 10% interest on your gold after each level (max 30).", Rarity: Rare, Tags: []string{"wealth"}, Max: 2, Apply: func(s *Stats) { s.Interest += 0.1 }},
	{ID: "fuse", Name: "Long Fuse", Desc: "Explosions are 50% larger.", Rarity: Uncommon, Tags: []string{"blast"}, Apply: func(s *Stats) { s.ExplodeRadius += 0.5 }},
	{ID: "laser", Name: "Laser Mount", Desc: "Your paddle fires twin lasers automatically.", Rarity: Rare, Tags: []string{"tech"}, Max: 3, Apply: func(s *Stats) {
		if s.LaserInterval == 0 {
			s.LaserInterval = 2.4
		} else {
			s.LaserInterval *= 0.65
		}
	}},
	{ID: "eagle", Name: "Eagle Eye", Desc: "Critical hits splash damage onto adjacent bricks.", Rarity: Rare, Tags: []string{"crit"}, Max: 1, Apply: func(s *Stats) { s.EagleEye = true }},
	{ID: "momentum", Name: "Momentum", Desc: "Each brick hit without touching the paddle adds +4% speed and damage.", Rarity: Rare, Tags: []string{"combo"}, Max: 1, Apply: func(s *Stats) { s.Momentum = true }},
	{ID: "padding", Name: "Shock Padding", Desc: "Losing your orb costs 6 less health.", Rarity: Common, Tags: []string{"guard"}, Max: 3, Apply: func(s *Stats) { s.BallLossDmg -= 6 }},

	// ---------------- relics ----------------
	{ID: "molten", Name: "Molten Core", Desc: "Every 50 orb impacts unleash a huge explosion.", Rarity: Uncommon, Relic: true, Tags: []string{"blast"}, Max: 1, Apply: func(s *Stats) { s.MoltenEvery = 50 }},
	{ID: "glass", Name: "Glass Cannon", Desc: "Damage ×1.6, but your paddle is 25% narrower.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.DmgMult *= 1.6; s.PaddleWidth -= 0.25 }},
	{ID: "magnet", Name: "Magnet Core", Desc: "Gold and power-ups drift toward your paddle.", Rarity: Common, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.Magnet = true }},
	{ID: "echo", Name: "Echo Chamber", Desc: "Every 5th impact strikes twice.", Rarity: Rare, Relic: true, Tags: []string{"combo"}, Max: 1, Apply: func(s *Stats) { s.EchoEvery = 5 }},
	{ID: "thunder", Name: "Thunder Crown", Desc: "Critical hits call down chain lightning.", Rarity: Epic, Relic: true, Tags: []string{"crit", "storm"}, Max: 1, Apply: func(s *Stats) { s.ThunderCrown = true }},
	{ID: "bloodmoon", Name: "Blood Moon", Desc: "Below half health, damage ×1.75.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.BloodMoon = true }},
	{ID: "inferno", Name: "Inferno Core", Desc: "+20% chance to ignite bricks.", Rarity: Uncommon, Relic: true, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.BurnChance += 0.2 }},
	{ID: "staticfield", Name: "Static Field", Desc: "Every 15 impacts, lightning strikes a random brick.", Rarity: Uncommon, Relic: true, Tags: []string{"storm"}, Max: 1, Apply: func(s *Stats) { s.StaticEvery = 15 }},
	{ID: "greed", Name: "Greed", Desc: "+25% gold.", Rarity: Common, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldMult += 0.25 }},
	{ID: "voidbounce", Name: "Void Bounce", Desc: "+20% chance to ignore armour and shields.", Rarity: Uncommon, Relic: true, Tags: []string{"void"}, Max: 1, Apply: func(s *Stats) { s.VoidChance += 0.2 }},
	{ID: "phoenix", Name: "Phoenix Feather", Desc: "Once per run, survive a fatal blow with 50 health.", Rarity: Legendary, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.Phoenix = true }},
	{ID: "prism", Name: "Prism Orb", Desc: "Every launch also releases a phantom orb.", Rarity: Epic, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.PrismOrb = true }},
	{ID: "meteor", Name: "Meteor Heart", Desc: "Your orb explodes on every brick it hits.", Rarity: Epic, Relic: true, Tags: []string{"blast"}, Max: 1, Apply: func(s *Stats) { s.MeteorHeart = true }},
	{ID: "frozenheart", Name: "Frozen Heart", Desc: "+25% freeze chance. Frozen bricks take double damage.", Rarity: Uncommon, Relic: true, Tags: []string{"frost"}, Max: 1, Apply: func(s *Stats) { s.FreezeChance += 0.25; s.FrozenBonus = true }},
	{ID: "angel", Name: "Guardian Halo", Desc: "+2 barrier charges every level.", Rarity: Uncommon, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.BarrierPerLevel += 2 }},
	{ID: "goldpaddle", Name: "Golden Paddle", Desc: "Each paddle bounce earns 1 gold.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldenPaddle = true }},
	{ID: "hydra", Name: "Hydra Seed", Desc: "Every 30 bricks destroyed spawns an extra orb.", Rarity: Epic, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.Hydra = true }},
	{ID: "leech", Name: "Leech Orb", Desc: "Heal 1 health for every 6 bricks destroyed.", Rarity: Rare, Relic: true, Tags: []string{"guard"}, Max: 1, Apply: func(s *Stats) { s.Lifesteal = 6 }},
	{ID: "clover", Name: "Four-Leaf Clover", Desc: "+5% crit chance and +30% power-up drops.", Rarity: Uncommon, Relic: true, Tags: []string{"wealth", "crit"}, Max: 1, Apply: func(s *Stats) { s.CritChance += 0.05; s.Luck += 0.3 }},
	{ID: "philo", Name: "Philosopher's Stone", Desc: "Crystals drop triple gold, and every brick may drop gold.", Rarity: Legendary, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.Philosopher = true }},
	{ID: "storm", Name: "Storm Caller", Desc: "Lightning arcs can jump a second time.", Rarity: Legendary, Relic: true, Tags: []string{"storm"}, Max: 1, Apply: func(s *Stats) { s.StormCaller = true }},
	{ID: "pyro", Name: "Pyromancer's Ash", Desc: "Burning bricks take double damage from everything.", Rarity: Epic, Relic: true, Tags: []string{"fire"}, Max: 1, Apply: func(s *Stats) { s.PyroAsh = true }},
	{ID: "overclock", Name: "Overclock", Desc: "Orb speed +20%, damage +40%.", Rarity: Rare, Relic: true, Tags: []string{"power"}, Max: 1, Apply: func(s *Stats) { s.SpeedMult += 0.2; s.DmgMult *= 1.4 }},
	{ID: "heavypad", Name: "Charged Paddle", Desc: "Paddle bounces charge the orb: its next hit is a guaranteed critical.", Rarity: Rare, Relic: true, Tags: []string{"crit"}, Max: 1, Apply: func(s *Stats) { s.HeavyPaddle = true }},
	{ID: "idol", Name: "Cursed Idol", Desc: "Double gold, but −30 max health.", Rarity: Rare, Relic: true, Tags: []string{"wealth"}, Max: 1, Apply: func(s *Stats) { s.GoldMult += 1; s.MaxHP -= 30 }},
	{ID: "mirror", Name: "Mirror Shard", Desc: "+10% chance a paddle bounce splits off an extra orb.", Rarity: Uncommon, Relic: true, Tags: []string{"swarm"}, Max: 1, Apply: func(s *Stats) { s.SplitChance += 0.1 }},
	{ID: "singularity", Name: "Singularity", Desc: "Void hits deal triple damage and +10% void chance.", Rarity: Mythic, Relic: true, Tags: []string{"void"}, Max: 1, Apply: func(s *Stats) { s.VoidChance += 0.1; s.VoidDouble = true }},
}

var modByID = func() map[string]*Mod {
	m := map[string]*Mod{}
	for _, md := range modList {
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
}

func sortedTags(m map[string]int) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
