package main

import (
	"fmt"
	"math/rand"
)

// Even more events (ROADMAP-50 #27), in the same voice as the troll in the
// small hat and the goat that eats receipts: warm, a bit silly, and always
// a real choice. Each returns nil when it can't happen now.

var events3Glyphs = map[string]string{
	"The Snack Dragon": "dragonheart", "A Bard With Opinions": "note", "The Duckling Parade": "feather",
	"Fumbleton's Hat": "moon", "The Slime Spa": "drop", "A Very Ambitious Rock": "anvil",
	"The Fortune Cookie": "clover",
}

func evenMoreEvents(g *Game, rng *rand.Rand) []func() *Event {
	r := g.run
	ward := func(n int) { r.NextBarrier += n }
	return []func() *Event{
		func() *Event {
			return &Event{"The Snack Dragon", "A very small dragon sits on a very large pile of cookies. \"Hoard tax,\" it squeaks. \"Or tickles. Tickles are also accepted.\"",
				[]EventChoice{
					{fmt.Sprintf("Pay the hoard tax (%d gold): choose an upgrade", r.price(30)), func(g *Game) string {
						g.run.Gold -= g.run.price(30)
						g.rewards = []rewardSet{{Title: "The Snack Dragon — choose an upgrade", Mods: g.run.Offer(g.run.choices(), false, Uncommon, rng), Skip: 10}}
						g.setState(StReward)
						return ""
					}, func(g *Game) bool { return g.run.Gold >= g.run.price(30) }},
					{"Tickle it (it might sneeze gold, or fire)", func(g *Game) string {
						if rng.Float64() < 0.55 {
							g.run.Gold += 45
							g.snd.Play("coinbig")
							return "Achoo! A shower of coins. The dragon looks embarrassed. (+45 gold)"
						}
						g.run.Cursed = true
						return "Achoo! A tiny fireball. The next chamber is singed: tougher bricks, double gold."
					}, nil},
					{"Leave the cookies alone", func(g *Game) string { return "It waves goodbye with a cookie. You were never getting one." }, nil},
				}}
		},
		func() *Event {
			return &Event{"A Bard With Opinions", "A bard blocks the corridor, lute raised. \"I'm writing a ballad about you. Pay me, and it's a flattering one.\"",
				[]EventChoice{
					{"Commission the ballad (25 gold): one free reroll", func(g *Game) string {
						g.run.Gold -= 25
						g.run.Rerolls-- // one more free reroll than the build gives
						return "\"…and the orb flew TRUE!\" You feel legendary. (one free reroll)"
					}, func(g *Game) bool { return g.run.Gold >= 25 }},
					{"Request a song about the Guardians (a scroll)", func(g *Game) string {
						g.addScroll(randomScroll(rng))
						return "The song is terrible, but the lyrics sheet is a real scroll."
					}, func(g *Game) bool { return len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Heckle", func(g *Game) string {
						g.run.Gold += 15
						return "He pays you 15 gold to stop. Fair."
					}, nil},
				}}
		},
		func() *Event {
			return &Event{"The Duckling Parade", "Five ducklings have decided your orb is their mother. They follow it everywhere, in a perfect line.",
				[]EventChoice{
					{"Lead them home (two barriers in the next chamber)", func(g *Game) string {
						ward(2)
						return "Mother Duck honks her thanks and lends you her best ducklings as guards. (two barriers in the next chamber)"
					}, nil},
					{"Let them keep following (+30 gold)", func(g *Game) string {
						g.run.Gold += 30
						return "Passing merchants pay to see the parade. The ducklings take a cut in bread."
					}, nil},
				}}
		},
		func() *Event {
			rel := pickRelic(g, rng, Uncommon)
			if rel == nil {
				return nil
			}
			return &Event{"Fumbleton's Hat", "A pointy hat lies in the dust, embroidered \"PROPERTY OF ARCHMAGE FUMBLETON. IF FOUND, PLEASE DON'T WEAR.\"",
				[]EventChoice{
					{fmt.Sprintf("Return it to Fumbleton: take %s", rel.Name), func(g *Game) string {
						g.take(rel)
						return fmt.Sprintf("Fumbleton pops out of a puff of smoke, hugs you, and presses %s into your hands. \"I was looking EVERYWHERE.\"", rel.Name)
					}, nil},
					{"Wear it (two scrolls, and a curse)", func(g *Game) string {
						g.addScroll(randomScroll(rng))
						g.addScroll(randomScroll(rng))
						g.addCurse(rng)
						return "The hat whispers spells into your ear. Most of them are wrong."
					}, func(g *Game) bool { return len(g.run.Scrolls)+2 <= g.run.scrollCap() }},
					{"Leave it", func(g *Game) string { return "Somewhere far away, a wizard sneezes." }, nil},
				}}
		},
		func() *Event {
			return &Event{"The Slime Spa", "A cheerful slime runs a spa in a puddle. \"Full-body treatment! Very relaxing! Only slightly sticky!\"",
				[]EventChoice{
					{"Book the treatment (20 gold)", func(g *Game) string {
						g.run.Gold -= 20
						if c := g.cleanse(); c != nil {
							return fmt.Sprintf("Sticky, but %s slides right off.", c.Name)
						}
						ward(2)
						return "You've never felt so relaxed, or so sticky. (two barriers in the next chamber)"
					}, func(g *Game) bool { return g.run.Gold >= 20 }},
					{"Buy a jar of slime (10 gold): a scroll", func(g *Game) string {
						g.run.Gold -= 10
						g.addScroll(randomScroll(rng))
						return "The jar wobbles. There's a scroll stuck to the bottom."
					}, func(g *Game) bool { return g.run.Gold >= 10 && len(g.run.Scrolls) < g.run.scrollCap() }},
					{"Politely decline", func(g *Game) string { return "\"Your loss!\" it blorps." }, nil},
				}}
		},
		func() *Event {
			return &Event{"A Very Ambitious Rock", "A rock clears its throat. \"I want to be a brick when I grow up. Will you help me train?\"",
				[]EventChoice{
					{"Believe in it (next chamber: tougher bricks, double gold)", func(g *Game) string {
						g.run.Cursed = true
						return "It brings its friends to the next chamber. They've been training too."
					}, nil},
					{"Use it as a doorstop (+25 gold)", func(g *Game) string {
						g.run.Gold += 25
						return "It's a very good doorstop. A traveller pays you 25 gold for it. The rock seems happy."
					}, nil},
				}}
		},
		func() *Event {
			return &Event{"The Fortune Cookie", "A cookie the size of a shield lies cracked open on a pedestal. The fortune inside is still rolled up.",
				[]EventChoice{
					{"Read the fortune", func(g *Game) string {
						k := rng.Intn(3)
						if k == 2 && len(g.run.Scrolls) >= g.run.scrollCap() {
							k = 0 // no room for a scroll: the gold fortune instead
						}
						switch k {
						case 0:
							g.run.Gold += 40
							g.snd.Play("coinbig")
							return "\"You will find 40 gold.\" You do. Under the cookie."
						case 1:
							ward(1)
							return "\"Something will protect you.\" A barrier hums to life for the next chamber."
						}
						g.addScroll(randomScroll(rng))
						return "\"Learn to read.\" Rude. It was written on a scroll, though."
					}, nil},
					{"Eat the cookie (regain a life)", func(g *Game) string {
						g.run.Lives = min(g.run.Lives+1, g.run.maxLives())
						return "It's stale. It's been here since the last warden. You eat it anyway, for morale. (+1 life)"
					}, func(g *Game) bool { return g.run.Lives < g.run.maxLives() }},
					{"Leave it", func(g *Game) string { return "Some fortunes are better left unread." }, nil},
				}}
		},
	}
}

func init() {
	for k, v := range events3Glyphs {
		eventGlyphs[k] = v
	}
}
