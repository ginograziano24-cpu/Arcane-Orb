package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"orb/levelgen"
)

// ---------------------------------------------------------------- archetypes

type Archetype struct {
	ID, Name, Desc string
	Passive        string
	Cost           int // shards to unlock (0 = free)
	Apply          func(s *Stats)
	Ability        string // active ability on right-click / E ("" = none)
	AbilityDesc    string
	Cooldown       float64 // seconds
}

var archetypes = []Archetype{
	{ID: "guardian", Name: "Guardian", Passive: "Parry",
		Desc: "Catch the orb on the centre of your paddle to loose a holy wave up that column. Wide paddle, 120 health, a barrier every chamber, slightly weaker hits.",
		Apply: func(s *Stats) {
			s.PaddleWidth += 0.2
			s.MaxHP += 20
			s.BarrierPerLevel++
			s.DmgMult *= 0.9
			s.ClassParry = true
		}},
	{ID: "berserker", Name: "Berserker", Passive: "Rampage",
		Desc: "Every brick you break adds Rage: +5% damage each, up to +100%. Rage fades after 5 seconds without a break, or when you drop the orb. Narrow paddle, +0.2 damage, +10% crit.",
		Apply: func(s *Stats) {
			s.PaddleWidth -= 0.1
			s.Damage += 0.2
			s.CritChance += 0.1
			s.CritMult += 0.5
			s.ClassRage = true
		}},
	{ID: "arcane", Name: "Arcanist", Passive: "Attunement",
		Desc: "Your orb cycles Fire, Frost, Storm and Void every 8 seconds; each hit has a 50% chance to unleash the current element. +0.15 damage.",
		Apply: func(s *Stats) {
			s.ClassAttune = true
			s.Damage += 0.15
		}},
	{ID: "engineer", Name: "Engineer", Passive: "Overclock", Cost: 60,
		Desc: "Paddle lasers fire every 2 seconds. Hold the paddle still for 1.5 seconds to overclock: the next volley is a piercing beam for double damage.",
		Apply: func(s *Stats) {
			s.LaserInterval = 2.0
			s.ClassOverclock = true
		}},
	{ID: "lucky", Name: "Gambler", Passive: "High Roller", Cost: 90,
		Desc: "Gold is power: +1% damage for every 6 gold you carry, up to +60%. Starts with 25 extra gold. Spend or hoard? +40% gold, +30% power-ups, +8% crit and a free shop reroll.",
		Apply: func(s *Stats) {
			s.GoldMult += 0.4
			s.Luck += 0.3
			s.CritChance += 0.08
			s.FreeRerolls++
			s.ClassHighRoller = true
			s.StartGold += 25
		}},
	{ID: "pyro", Name: "Pyromancer", Passive: "Stoke", Cost: 120,
		Desc: "Every 5th hit ignites. Hitting a burning brick stokes it: +50% damage and the fire is renewed. Fire burns 50% hotter.",
		Apply: func(s *Stats) {
			s.ClassIgnite = 5
			s.BurnMult += 0.5
			s.ClassStoke = true
		}, Ability: "Firestorm", AbilityDesc: "Set the two lowest rows ablaze.", Cooldown: 32},
	{ID: "storm", Name: "Stormcaller", Passive: "Static Charge", Cost: 120,
		Desc: "Sweep the paddle back and forth to build static. When full, your next bounce supercharges the orb: for 3 seconds every hit arcs lightning. Lightning +10%.",
		Apply: func(s *Stats) {
			s.ChainMult += 0.1
			s.ClassStatic = true
		}, Ability: "Thunderstrike", AbilityDesc: "Smite the toughest brick; it chains to 3 more.", Cooldown: 28},
}

// ---------------------------------------------------------------- map

type NodeType int

const (
	NCombat NodeType = iota
	NElite
	NShop
	NTreasure
	NEvent
	NRest
	NBoss
)

var nodeNames = []string{"Combat", "Elite", "Shop", "Treasure", "Event", "Rest", "Boss"}
var nodeGlyph = []string{"⚔", "☠", "$", "◆", "?", "+", "♛"}

type MapNode struct {
	Row, Lane int
	Type      NodeType
	Next      []int
	Visited   bool
	Seed      int64
	Omen      string // optional chamber omen (omens.go)
}

type ActMap struct {
	Nodes []*MapNode
	Rows  int
}

const mapRows = 7

func genMap(r *rand.Rand, act int) *ActMap {
	m := &ActMap{Rows: mapRows + 1}
	rows := make([][]int, mapRows+1)
	for row := 0; row < mapRows; row++ {
		lanes := r.Perm(4)
		n := 2 + r.Intn(3)
		if row == 0 {
			n = 3
		}
		lanes = lanes[:n]
		sort.Ints(lanes)
		for _, ln := range lanes {
			t := NCombat
			x := r.Float64()
			switch {
			case row == 0:
				t = NCombat
			case row == mapRows-1:
				if x < 0.5 {
					t = NRest
				} else {
					t = NShop
				}
			case row >= 2 && x < 0.16:
				t = NElite
			case x < 0.3:
				t = NEvent
			case x < 0.4:
				t = NShop
			case x < 0.47:
				t = NTreasure
			case x < 0.52 && row >= 3:
				t = NRest
			}
			m.Nodes = append(m.Nodes, &MapNode{Row: row, Lane: ln, Type: t, Seed: r.Int63()})
			rows[row] = append(rows[row], len(m.Nodes)-1)
		}
	}
	// guarantee one treasure mid-act
	mid := rows[3]
	m.Nodes[mid[r.Intn(len(mid))]].Type = NTreasure
	placeOmens(m, r, act)
	m.Nodes = append(m.Nodes, &MapNode{Row: mapRows, Lane: 1, Type: NBoss, Seed: r.Int63()})
	rows[mapRows] = []int{len(m.Nodes) - 1}
	// edges
	for row := 0; row < mapRows; row++ {
		for _, i := range rows[row] {
			a := m.Nodes[i]
			for _, j := range rows[row+1] {
				b := m.Nodes[j]
				if b.Type == NBoss || abs(a.Lane-b.Lane) <= 1 {
					a.Next = append(a.Next, j)
				}
			}
			if len(a.Next) == 0 { // connect to nearest
				best, bd := -1, 99
				for _, j := range rows[row+1] {
					if d := abs(m.Nodes[j].Lane - a.Lane); d < bd {
						best, bd = j, d
					}
				}
				a.Next = append(a.Next, best)
			}
		}
		// every next-row node needs an incoming edge
		for _, j := range rows[row+1] {
			has := false
			for _, i := range rows[row] {
				for _, n := range m.Nodes[i].Next {
					if n == j {
						has = true
					}
				}
			}
			if !has {
				best, bd := -1, 99
				for _, i := range rows[row] {
					if d := abs(m.Nodes[i].Lane - m.Nodes[j].Lane); d < bd {
						best, bd = i, d
					}
				}
				m.Nodes[best].Next = append(m.Nodes[best].Next, j)
			}
		}
	}
	return m
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---------------------------------------------------------------- run

const numActs = 3

type Run struct {
	SeedCode     string
	SeedN        int64
	Daily        bool
	Rng          *rand.Rand
	Arch         *Archetype
	AbilT        float64 // class ability cooldown remaining
	Act          int
	Map          *ActMap
	Cur          int // current node index (-1 = start of act)
	Depth        int // levels entered
	HP           float64
	Gold         int
	Mods         map[string]int
	Order        []string // acquisition order for display
	Stats        Stats
	Active       []string // active synergy names
	PhoenixUsed  bool
	Lives        int // lives left (lives.go)
	LifeFinds    int // hearts caught this run
	Cursed       bool
	LastWorld    int
	Element      string
	Rerolls      int
	Fused        []string // fusions formed this run
	Scrolls      []string // carried scrolls (scrolls.go)
	NewFusions   []string // formed since last announced
	Rank, Asc    int      // player rank and ascension level at run start
	Endless      bool     // continued past the final boss
	EndlessStart bool     // started from the Endless menu (no story)
	FinalScore   int
	xpPaid       int // XP already awarded (victory before endless)
	shardsPaid   int
	statsPaid    [3]int // bricks, bosses, elites already added to the save
	metaLuck     int

	// statistics
	Bricks, Bosses, Elites, MaxCombo, Score int
	Start                                   time.Time
}

func NewRun(code string, arch *Archetype, meta *SaveData, daily bool) *Run {
	code = strings.ToUpper(strings.TrimSpace(code))
	n := levelgen.Mix(int64(len(code)), hashStr(code))
	r := &Run{SeedCode: code, SeedN: n, Daily: daily, Rng: rand.New(rand.NewSource(n)), Arch: arch, LastWorld: -1,
		Mods: map[string]int{}, Cur: -1, Start: time.Now()}
	r.Act = 1
	r.Lives = baseLives
	r.Rank = rankOf(meta.XP)
	r.Asc = meta.AscSel
	r.metaLuck = meta.Upg["fortune"]
	r.Map = genMap(rand.New(rand.NewSource(levelgen.Mix(n, 1))), 1)
	r.Recompute(meta)
	if r.Stats.StartElement == "random" {
		els := []string{"inferno", "staticfield", "frozenheart", "voidbounce"}
		r.Add(els[r.Rng.Intn(len(els))], meta)
	}
	r.HP = r.Stats.MaxHP
	r.Gold = 20 + 20*meta.Upg["purse"] + int(r.Stats.StartGold)
	// Sanctum: start with relics / upgrades
	rng := rand.New(rand.NewSource(levelgen.Mix(n, 9)))
	for i := 0; i < meta.Upg["collector"]; i++ {
		if ms := r.Offer(1, true, Common, rng); len(ms) > 0 {
			r.Add(ms[0].ID, meta)
		}
	}
	for i := 0; i < meta.Upg["armory"]; i++ {
		if ms := r.Offer(1, false, Uncommon, rng); len(ms) > 0 {
			r.Add(ms[0].ID, meta)
		}
	}
	r.NewFusions = nil
	r.HP = r.Stats.MaxHP
	return r
}

func hashStr(s string) int64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return int64(h >> 1)
}

func baseStats() Stats {
	return Stats{MaxHP: 100, Damage: 1, DmgMult: 1, CritChance: 0.08, CritMult: 2, SpeedMult: 1, SizeMult: 1,
		ChainCount: 2, ExplodeRadius: 1.4, GoldMult: 1, Luck: 1, PaddleWidth: 1, BallLossDmg: 0}
}

func (r *Run) Recompute(meta *SaveData) {
	s := baseStats()
	applyMeta(&s, meta, r.Asc)
	if l := r.Loop(); l > 0 {
		// endless escalation: faster orbs, harsher losses
		s.SpeedMult += 0.06 * float64(l)
		s.BallLossDmg += 3 * float64(l)
	}
	s.MaxHP -= 20 * float64(r.Mods["__hp"])
	r.Arch.Apply(&s)
	tags := map[string]int{}
	for _, id := range r.Order {
		m := modByID[id]
		for k := 0; k < r.Mods[id]; k++ {
			m.Apply(&s)
		}
		for _, t := range m.Tags {
			tags[t] += r.Mods[id]
		}
	}
	for _, id := range r.Order {
		if m := modByID[id]; m != nil && m.Curse && r.Mods[id] > 0 {
			s.CurseCount++
		}
	}
	if s.CurseWard && r.Mods["c_doubt"] > 0 {
		s.MaxHP += 15 // the talisman shields your health from curses
	}
	r.Active = r.Active[:0]
	for _, sy := range synergies {
		if tags[sy.Tag] >= 3 {
			sy.Apply(&s)
			r.Active = append(r.Active, sy.Name)
		}
	}
	for i := range fusions {
		f := &fusions[i]
		if r.Owned(f.A) && r.Owned(f.B) {
			f.Apply(&s)
			if !r.HasFusion(f.ID) {
				r.Fused = append(r.Fused, f.ID)
				r.NewFusions = append(r.NewFusions, f.ID)
			}
		}
	}
	s.PaddleWidth = math.Max(0.5, s.PaddleWidth)
	s.BallLossDmg = math.Max(0, s.BallLossDmg)
	s.MaxHP = math.Max(20, s.MaxHP)
	r.Stats = s
	if r.HP > s.MaxHP {
		r.HP = s.MaxHP
	}
}

// Add grants a mod and returns any synergy that just activated.
func (r *Run) Add(id string, meta *SaveData) string {
	before := map[string]bool{}
	for _, a := range r.Active {
		before[a] = true
	}
	if r.Mods[id] == 0 {
		r.Order = append(r.Order, id)
	}
	r.Mods[id]++
	oldMax := r.Stats.MaxHP
	r.Recompute(meta)
	if r.Stats.MaxHP > oldMax {
		r.HP += r.Stats.MaxHP - oldMax
	}
	for _, a := range r.Active {
		if !before[a] {
			return a
		}
	}
	return ""
}

func (r *Run) Owned(id string) bool { return r.Mods[id] > 0 }

// AI is the act index used for worlds, bosses and music; endless loops
// cycle through the three acts again.
func (r *Run) AI() int { return (r.Act - 1) % numActs }

// Loop is the endless loop number (0 before endless).
func (r *Run) Loop() int { return max(0, r.Act-numActs) }

// choices is how many cards a reward offers.
func (r *Run) choices() int { return 3 + r.Stats.ExtraChoices }

// freeRerolls left this run.
func (r *Run) freeRerolls() int { return r.Stats.FreeRerolls - r.Rerolls }

// price applies shop discounts and ascension inflation.
func (r *Run) price(p int) int {
	m := math.Max(0.4, 1-r.Stats.ShopDiscount)
	if r.Asc >= 3 {
		m *= 1.15
	}
	return max(1, int(math.Round(float64(p)*m)))
}

func (r *Run) canTake(m *Mod) bool {
	return m.Max == 0 || r.Mods[m.ID] < m.Max
}

// Offer draws n distinct mods (relics or upgrades) weighted by rarity.
func (r *Run) Offer(n int, relic bool, minRarity int, rng *rand.Rand) []*Mod {
	var pool []*Mod
	for _, m := range modList {
		if m.Relic == relic && m.Rarity >= minRarity && r.canTake(m) && m.Unlock <= r.Rank && (m.Needs == "" || r.Owned(m.Needs)) {
			pool = append(pool, m)
		}
	}
	var out []*Mod
	luck := 1 + (r.Stats.Luck-1)*0.3 + float64(min(r.Act-1, 4))*0.25 + 0.04*float64(r.metaLuck)
	weight := func(m *Mod) float64 {
		w := rarityWeights[m.Rarity] * math.Pow(luck, float64(m.Rarity))
		if r.fusionWith(m.ID) != nil {
			w *= 4 // completing a fusion is exciting: offer it more
		}
		return w
	}
	for len(out) < n && len(pool) > 0 {
		total := 0.0
		for _, m := range pool {
			total += weight(m)
		}
		x := rng.Float64() * total
		for i, m := range pool {
			w := weight(m)
			if x < w {
				out = append(out, m)
				pool = append(pool[:i], pool[i+1:]...)
				break
			}
			x -= w
		}
	}
	return out
}

func (r *Run) NodeSeed(i int) int64 {
	return levelgen.Mix(r.SeedN, int64(r.Act), int64(i), 77)
}

func (r *Run) NextAct(meta *SaveData) bool {
	if r.Act >= numActs && !r.Endless {
		return false
	}
	r.Act++
	r.Map = genMap(rand.New(rand.NewSource(levelgen.Mix(r.SeedN, int64(r.Act)))), r.Act)
	r.Cur = -1
	r.HP = math.Min(r.Stats.MaxHP, r.HP+r.Stats.MaxHP*0.3)
	r.Lives = max(r.Lives, baseLives) // every act starts with at least two lives
	return true
}

func (r *Run) Reachable() []int {
	if r.Cur < 0 {
		var out []int
		for i, n := range r.Map.Nodes {
			if n.Row == 0 {
				out = append(out, i)
			}
		}
		return out
	}
	return r.Map.Nodes[r.Cur].Next
}

func modPrice(m *Mod, rng *rand.Rand) int {
	base := []int{40, 65, 100, 150, 220, 320}[m.Rarity]
	if m.Relic {
		base = base * 13 / 10
	}
	return base + rng.Intn(12) - 6
}

// ---------------------------------------------------------------- events

type EventChoice struct {
	Label string
	Do    func(g *Game) string // returns result text
	Can   func(g *Game) bool
}

type Event struct {
	Title, Text string
	Choices     []EventChoice
}

func makeEvent(g *Game, rng *rand.Rand) *Event {
	r := g.run
	extra := moreEvents(g, rng)
	for try := 0; try < 4; try++ {
		k := rng.Intn(6 + len(extra))
		if k < 6 {
			break
		}
		if ev := extra[k-6](); ev != nil {
			return ev
		}
	}
	switch rng.Intn(6) {
	case 0:
		relic := r.Offer(1, true, Rare, rng)
		if len(relic) == 0 {
			break
		}
		m := relic[0]
		return &Event{"Mysterious Merchant", "A hooded figure offers you " + m.Name + " — \"" + m.Desc + "\" — in exchange for a little of your life force.",
			[]EventChoice{
				{"Pay 20 max health", func(g *Game) string {
					g.run.Mods["__hp"]++
					g.run.Add(m.ID, &g.save)
					return "You feel weaker, but " + m.Name + " hums in your hand."
				}, func(g *Game) bool { return g.run.Stats.MaxHP > 40 }},
				{"Walk away", func(g *Game) string { return "The merchant fades into the dark." }, nil},
			}}
	case 1:
		return &Event{"Gambling Den", "Spectral dice clatter across a stone table. \"Double or nothing?\"",
			[]EventChoice{
				{"Bet half your gold", func(g *Game) string {
					bet := g.run.Gold / 2
					if rng.Float64() < 0.5 {
						g.run.Gold += bet
						g.snd.Play("coinbig")
						return fmt.Sprintf("Fortune smiles! You win %d gold.", bet)
					}
					g.run.Gold -= bet
					return fmt.Sprintf("The dice turn cold. You lose %d gold.", bet)
				}, func(g *Game) bool { return g.run.Gold >= 10 }},
				{"Roll for a relic (60 gold)", func(g *Game) string {
					g.run.Gold -= 60
					if rng.Float64() < 0.55 {
						rl := g.run.Offer(1, true, Common, rng)
						if len(rl) > 0 {
							g.run.Add(rl[0].ID, &g.save)
							return "The dice spell a name: " + rl[0].Name + "!"
						}
					}
					return "Snake eyes. The house keeps your gold."
				}, func(g *Game) bool { return g.run.Gold >= 60 }},
				{"Leave", func(g *Game) string { return "You keep your coins close." }, nil},
			}}
	case 2:
		return &Event{"Upgrade Shrine", "An altar of humming crystal. Offerings placed here are reforged into something greater.",
			[]EventChoice{
				{"Sacrifice a random upgrade for a rare relic", func(g *Game) string {
					var ups []string
					for _, id := range g.run.Order {
						if m := modByID[id]; m != nil && !m.Relic && g.run.Mods[id] > 0 {
							ups = append(ups, id)
						}
					}
					lost := ups[rng.Intn(len(ups))]
					g.run.Mods[lost]--
					rl := g.run.Offer(1, true, Rare, rng)
					msg := "You offer " + modByID[lost].Name + "."
					if len(rl) > 0 {
						g.run.Add(rl[0].ID, &g.save)
						msg += " The shrine returns " + rl[0].Name + "!"
					}
					g.run.Recompute(&g.save)
					return msg
				}, func(g *Game) bool {
					for _, id := range g.run.Order {
						if m := modByID[id]; m != nil && !m.Relic && g.run.Mods[id] > 0 {
							return true
						}
					}
					return false
				}},
				{"Pray (heal 25%)", func(g *Game) string {
					g.run.HP = math.Min(g.run.Stats.MaxHP, g.run.HP+g.run.Stats.MaxHP*0.25)
					return "Warm light washes over you."
				}, nil},
			}}
	case 3:
		return &Event{"Cursed Room", "Black runes crawl across the next chamber's walls. The bricks within are stronger — and richer.",
			[]EventChoice{
				{"Accept the curse (next fight: +50% brick health, double gold)", func(g *Game) string {
					g.run.Cursed = true
					return "The runes flare. Your next battle will be brutal, and lucrative."
				}, nil},
				{"Seal the door", func(g *Game) string { return "You bar the door and move on." }, nil},
			}}
	case 4:
		return &Event{"Abandoned Forge", "An anvil still glows with heat. You could temper something here.",
			[]EventChoice{
				{"Temper your orb (+0.5 damage)", func(g *Game) string {
					g.run.Add("heavy", &g.save)
					return "The orb rings like a bell."
				}, nil},
				{"Salvage scrap (+40 gold)", func(g *Game) string {
					g.run.Gold += 40
					return "You pocket a handful of gilded scrap."
				}, nil},
			}}
	}
	return &Event{"Wandering Spirit", "A friendly wisp circles your paddle and leaves a small gift.",
		[]EventChoice{
			{"Accept (+30 gold)", func(g *Game) string { g.run.Gold += 30; return "The wisp giggles and vanishes." }, nil},
			{"Ask for healing (+20 health)", func(g *Game) string {
				g.run.HP = math.Min(g.run.Stats.MaxHP, g.run.HP+20)
				return "You feel refreshed."
			}, nil},
		}}
}

// baseMaxPenalty applies permanent max-HP penalties stored as pseudo-mod "__hp".
func baseMaxPenalty(r *Run) {
	r.Stats.MaxHP = math.Max(20, r.Stats.MaxHP)
	if r.HP > r.Stats.MaxHP {
		r.HP = r.Stats.MaxHP
	}
}

// pickWorld chooses this level's world from the act pool, never repeating
// the previous one, so every chamber feels like a new place.
func (r *Run) pickWorld(seed int64) int {
	pool := acts[r.AI()].Pool
	rng := rand.New(rand.NewSource(seed ^ 0x3a7))
	for i := 0; i < 8; i++ {
		w := pool[rng.Intn(len(pool))]
		if w != r.LastWorld {
			return w
		}
	}
	return pool[0]
}
