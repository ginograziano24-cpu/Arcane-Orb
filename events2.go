package main

import (
	"fmt"
	"math"
	"math/rand"
)

// More events: bargains, gambles, trials and kindnesses, so the "?" on the
// map is always worth a look. Each returns nil when it can't happen now.

var moreEventGlyphs = map[string]string{
	"The Bargaining Imp": "fang", "The Scriptorium": "book", "Wishing Well": "coin",
	"A Warden's Grave": "urn", "Mirror of Echoes": "mirror", "The Alchemist's Cart": "flask",
	"Training Grounds": "sword", "Shrine of Purity": "halo", "The Toll Bridge": "coinstack",
	"Lost Caravan": "leaf", "Your Shadow": "mask", "The Celestial Orrery": "moon",
	"A Goblin Thief": "fist", "The Ancient Library": "lens",
}

func pickRelic(g *Game, rng *rand.Rand, min int) *Mod {
	if rl := g.run.Offer(1, true, min, rng); len(rl) > 0 {
		return rl[0]
	}
	return nil
}

func moreEvents(g *Game, rng *rand.Rand) []func() *Event {
	r := g.run
	heal := func(n float64) { r.HP = math.Min(r.Stats.MaxHP, r.HP+n) }
	hurt := func(n float64) { r.HP = math.Max(1, r.HP-n) }
	return []func() *Event{
		func() *Event {
			rel := pickRelic(g, rng, Rare)
			if rel == nil {
				return nil
			}
			return &Event{"The Bargaining Imp", "A grinning imp juggles a glowing trinket. \"Power, warden! Only a teensy little curse attached.\"",
				[]EventChoice{
					{"Take " + rel.Name + " (and a curse)", func(g *Game) string {
						g.take(rel)
						c := g.addCurse(rng)
						if c == nil {
							return "The imp shrugs. You're already as cursed as can be."
						}
						return "The imp cackles as " + c.Name + " settles on you."
					}, nil},
					{"Take 70 gold (and a curse)", func(g *Game) string {
						g.run.Gold += 70
						g.addCurse(rng)
						g.snd.Play("coinbig")
						return "Coins rain into your purse. Something cold follows them."
					}, nil},
					{"Refuse", func(g *Game) string { return "\"Your loss!\" The imp vanishes in a puff of soot." }, nil},
				}}
		},
		func() *Event {
			id := randomScroll(rng)
			sd := scrollByID[id]
			return &Event{"The Scriptorium", "Quills scratch by themselves over half-finished scrolls. A sign reads: \"Copies made. Inquire within.\"",
				[]EventChoice{
					{fmt.Sprintf("Buy %s (25 gold)", sd.Name), func(g *Game) string {
						g.run.Gold -= 25
						g.addScroll(id)
						return "The ink is still wet."
					}, func(g *Game) bool { return g.run.Gold >= 25 && len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Copy a scroll you carry (15 gold)", func(g *Game) string {
						g.run.Gold -= 15
						src := g.run.Scrolls[rng.Intn(len(g.run.Scrolls))]
						g.addScroll(src)
						return "A perfect copy of " + scrollByID[src].Name + "."
					}, func(g *Game) bool {
						return g.run.Gold >= 15 && len(g.run.Scrolls) > 0 && len(g.run.Scrolls) < g.run.scrollCap()
					}},
					{"Leave", func(g *Game) string { return "The quills keep writing as you go." }, nil},
				}}
		},
		func() *Event {
			return &Event{"Wishing Well", "Coins glitter at the bottom of a deep, humming well.",
				[]EventChoice{
					{"Toss in 20 gold", func(g *Game) string {
						g.run.Gold -= 20
						if rng.Float64() < 0.4 {
							if rel := pickRelic(g, rng, Common); rel != nil {
								g.take(rel)
								return "The water glows. " + rel.Name + " floats up to you!"
							}
						}
						return "A soft plink. Nothing else."
					}, func(g *Game) bool { return g.run.Gold >= 20 }},
					{"Toss in everything (60+ gold)", func(g *Game) string {
						g.run.Gold = 0
						if rel := pickRelic(g, rng, Uncommon); rel != nil {
							g.take(rel)
							return "The well roars with light and returns " + rel.Name + "."
						}
						return "The well swallows it all."
					}, func(g *Game) bool { return g.run.Gold >= 60 }},
					{"Walk on", func(g *Game) string { return "You keep your coins." }, nil},
				}}
		},
		func() *Event {
			return &Event{"A Warden's Grave", "A cairn marks the resting place of a warden who came before you. Their paddle lies broken beside it.",
				[]EventChoice{
					{"Dig (lose 12 health): take what they carried", func(g *Game) string {
						hurt(12)
						if rel := pickRelic(g, rng, Uncommon); rel != nil {
							g.take(rel)
							return "Among the bones: " + rel.Name + "."
						}
						return "Only dust."
					}, func(g *Game) bool { return g.run.HP > 15 }},
					{"Pay your respects", func(g *Game) string {
						if c := g.cleanse(); c != nil {
							return "Peace settles on you. " + c.Name + " is lifted."
						}
						heal(18)
						return "You feel steadier. (+18 health)"
					}, nil},
				}}
		},
		func() *Event {
			return &Event{"Mirror of Echoes", "A tall mirror shows you a little stronger than you are. It wants something in return.",
				[]EventChoice{
					{"Gaze into it (lose 15 health): empower a random upgrade", func(g *Game) string {
						emp := g.run.empowerable()
						hurt(15)
						m := emp[rng.Intn(len(emp))]
						g.take(m)
						return m.Name + " doubles in strength."
					}, func(g *Game) bool { return len(g.run.empowerable()) > 0 && g.run.HP > 18 }},
					{"Shatter it: two scrolls, and a curse", func(g *Game) string {
						g.addScroll(randomScroll(rng))
						g.addScroll(randomScroll(rng))
						g.addCurse(rng)
						return "Shards scatter. Some are covered in writing."
					}, func(g *Game) bool { return len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Look away", func(g *Game) string { return "The reflection frowns." }, nil},
				}}
		},
		func() *Event {
			return &Event{"The Alchemist's Cart", "Bottles clink on a cart pulled by a very patient goat.",
				[]EventChoice{
					{"Draught of Mending (20 gold)", func(g *Game) string {
						g.run.Gold -= 20
						g.addScroll("heal")
						return "It smells of mint and thunder."
					}, func(g *Game) bool { return g.run.Gold >= 20 && len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Mystery bottle (30 gold)", func(g *Game) string {
						g.run.Gold -= 30
						id := randomScroll(rng)
						g.addScroll(id)
						return "It turns out to be " + scrollByID[id].Name + "."
					}, func(g *Game) bool { return g.run.Gold >= 30 && len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Sell a scroll (+45 gold)", func(g *Game) string {
						g.run.Scrolls = g.run.Scrolls[:len(g.run.Scrolls)-1]
						g.run.Gold += 45
						g.snd.Play("coinbig")
						return "The goat eats the receipt."
					}, func(g *Game) bool { return len(g.run.Scrolls) > 0 }},
				}}
		},
		func() *Event {
			return &Event{"Training Grounds", "Straw dummies and a rack of practice orbs. An old instructor nods at you.",
				[]EventChoice{
					{"Spar (lose 12 health): choose an Uncommon+ upgrade", func(g *Game) string {
						hurt(12)
						g.rewards = []rewardSet{{Title: "Training — choose an upgrade", Mods: g.run.Offer(g.run.choices(), false, Uncommon, rng), Skip: 10}}
						g.setState(StReward)
						return ""
					}, func(g *Game) bool { return g.run.HP > 15 }},
					{"Meditate (+20 health)", func(g *Game) string {
						heal(20)
						return "Your breathing slows. (+20 health)"
					}, nil},
				}}
		},
		func() *Event {
			if len(r.curses()) == 0 {
				return nil
			}
			return &Event{"Shrine of Purity", "Clear water runs over white stone. Curses hiss when they come near it.",
				[]EventChoice{
					{"Wash away a curse", func(g *Game) string {
						c := g.cleanse()
						return c.Name + " washes away."
					}, nil},
					{"Wash away every curse (40 gold each)", func(g *Game) string {
						n := 0
						for len(g.run.curses()) > 0 && g.run.Gold >= 40 {
							g.run.Gold -= 40
							g.cleanse()
							n++
						}
						return fmt.Sprintf("%d curses dissolve into the water.", n)
					}, func(g *Game) bool { return g.run.Gold >= 40 && len(g.run.curses()) > 1 }},
				}}
		},
		func() *Event {
			return &Event{"The Toll Bridge", "A troll in a very small hat blocks the bridge. \"Toll's thirty-five. Or there's the long way.\"",
				[]EventChoice{
					{"Pay 35 gold", func(g *Game) string {
						g.run.Gold -= 35
						heal(10)
						return "The troll tips his hat. You cross in comfort. (+10 health)"
					}, func(g *Game) bool { return g.run.Gold >= 35 }},
					{"Take the long way (next chamber: tougher bricks, double gold)", func(g *Game) string {
						g.run.Cursed = true
						return "The long way is long, and full of very solid bricks."
					}, nil},
				}}
		},
		func() *Event {
			up := g.run.Offer(1, false, Common, rng)
			return &Event{"Lost Caravan", "An overturned wagon. Its merchants are nowhere to be seen, but their goods are.",
				[]EventChoice{
					{"Take the goods (+45 gold and an upgrade)", func(g *Game) string {
						g.run.Gold += 45
						if len(up) > 0 {
							g.take(up[0])
							return "Gold, and " + up[0].Name + ". You try not to feel bad."
						}
						return "Gold, at least."
					}, nil},
					{"Find the merchants (+15 health, a scroll)", func(g *Game) string {
						heal(15)
						g.addScroll(randomScroll(rng))
						return "Grateful, they press a scroll into your hands."
					}, func(g *Game) bool { return len(g.run.Scrolls) < g.run.scrollCap() }},
				}}
		},
		func() *Event {
			rel := pickRelic(g, rng, Rare)
			if rel == nil {
				return nil
			}
			return &Event{"Your Shadow", "Your shadow peels itself off the floor and raises a paddle of its own.",
				[]EventChoice{
					{"Duel it (lose 22 health): win " + rel.Name, func(g *Game) string {
						hurt(22)
						g.take(rel)
						return "It fades with a bow, leaving " + rel.Name + "."
					}, func(g *Game) bool { return g.run.HP > 25 }},
					{"Let it in: empower two upgrades, gain a curse", func(g *Game) string {
						emp := g.run.empowerable()
						rng.Shuffle(len(emp), func(i, j int) { emp[i], emp[j] = emp[j], emp[i] })
						for i := 0; i < 2 && i < len(emp); i++ {
							g.take(emp[i])
						}
						g.addCurse(rng)
						return "You feel stronger, and darker."
					}, func(g *Game) bool { return len(g.run.empowerable()) > 0 }},
					{"Step into the light", func(g *Game) string { return "It slinks back to the floor." }, nil},
				}}
		},
		func() *Event {
			var pool []*Mod
			for _, id := range []string{"inferno", "frozenheart", "staticfield", "voidbounce"} {
				if m := modByID[id]; m != nil && !g.run.Owned(id) {
					pool = append(pool, m)
				}
			}
			if len(pool) < 2 {
				return nil
			}
			return &Event{"The Celestial Orrery", "Brass planets turn above a glowing table. Each one hums with a different element.",
				[]EventChoice{
					{"Align the planets (choose an element relic)", func(g *Game) string {
						g.rewards = []rewardSet{{Title: "The Orrery — choose your element", Mods: pool[:min(3, len(pool))], Skip: 20}}
						g.setState(StReward)
						return ""
					}, nil},
					{"Pocket a brass moon (+35 gold)", func(g *Game) string {
						g.run.Gold += 35
						return "It's surprisingly heavy."
					}, nil},
				}}
		},
		func() *Event {
			if r.Gold < 30 {
				return nil
			}
			stolen := r.Gold / 4
			return &Event{"A Goblin Thief", fmt.Sprintf("A goblin snatches %d gold and bolts for a hole in the wall!", stolen),
				[]EventChoice{
					{"Chase it (lose 8 health)", func(g *Game) string {
						hurt(8)
						g.run.Gold += 25
						return fmt.Sprintf("You catch it, and it pays you 25 gold to let go. (You never lost the %d.)", stolen)
					}, func(g *Game) bool { return g.run.HP > 10 }},
					{"Let it go", func(g *Game) string {
						g.run.Gold -= stolen
						return "It blows a raspberry from the hole."
					}, nil},
				}}
		},
		func() *Event {
			return &Event{"The Ancient Library", "Shelves vanish into the dark. One book is open, its pages turning on their own.",
				[]EventChoice{
					{"Study (half your gold): choose a Rare upgrade", func(g *Game) string {
						g.run.Gold /= 2
						g.rewards = []rewardSet{{Title: "The Library — choose an upgrade", Mods: g.run.Offer(g.run.choices(), false, Rare, rng), Skip: 0}}
						g.setState(StReward)
						return ""
					}, func(g *Game) bool { return g.run.Gold >= 20 }},
					{"Borrow a scroll", func(g *Game) string {
						g.addScroll(randomScroll(rng))
						return "Nobody stops you. Nobody is there."
					}, func(g *Game) bool { return len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Leave quietly", func(g *Game) string { return "The pages stop turning as you go." }, nil},
				}}
		},
	}
}

func init() {
	for k, v := range moreEventGlyphs {
		eventGlyphs[k] = v
	}
}
