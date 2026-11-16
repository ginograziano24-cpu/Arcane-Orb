package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Banish and lock: two small ways to steer a build without making it
// stronger. A banished item never shows up again this run (3 a run, more
// with the Banisher's Ring); a locked shop item stays put through a reroll
// (2 at a time).

const (
	baseBanishes = 3
	maxShopLocks = 2
)

// banishesLeft this run.
func (r *Run) banishesLeft() int {
	return max(0, baseBanishes+r.Stats.ExtraBanishes-r.BanishUsed)
}

// banish takes an item out of the run's offers for good.
func (r *Run) banish(id string) bool {
	if r.banishesLeft() <= 0 || r.Banished[id] {
		return false
	}
	if r.Banished == nil {
		r.Banished = map[string]bool{}
	}
	r.Banished[id] = true
	r.BanishUsed++
	return true
}

// replacementCard finds a card for slot i that isn't already in the hand.
func (r *Run) replacementCard(hand []*Mod, i int, rng *rand.Rand) *Mod {
	relic := hand[i].Relic
	minR := hand[i].Rarity
	for _, m := range hand {
		minR = min(minR, m.Rarity)
	}
	for try := 0; try < 6; try++ {
		for _, m := range r.Offer(len(hand)+1, relic, minR, rng) {
			dup := false
			for _, h := range hand {
				if h.ID == m.ID {
					dup = true
				}
			}
			if !dup {
				return m
			}
		}
	}
	return nil
}

// banishCard banishes card i of the reward on screen and deals another in
// its place (or drops the card when nothing is left to offer).
func (g *Game) banishCard(i int) {
	g.banishing = false
	rs := &g.rewards[0]
	m := rs.Mods[i]
	if !g.run.banish(m.ID) {
		g.snd.Play("deny")
		return
	}
	rng := rand.New(rand.NewSource(g.rng.Int63()))
	if nm := g.run.replacementCard(rs.Mods, i, rng); nm != nil {
		rs.Mods[i] = nm
	} else if len(rs.Mods) > 1 {
		rs.Mods = append(rs.Mods[:i:i], rs.Mods[i+1:]...)
	}
	g.snd.Play("shatter")
	g.snd.PlayVol("banish", 0.5)
	g.stateT = 0.31 // no re-deal, and the new card can be picked at once
}

// drawBanishButton toggles banish mode on a reward screen.
func (g *Game) drawBanishButton(s *ebiten.Image) {
	u := g.ui
	left := g.run.banishesLeft()
	if left <= 0 && !g.banishing {
		return
	}
	g.ringOnBanish(s)
	label := trJoin(tr("Banish"), "  (", itoa(left), ")")
	acc := cAcc
	if g.banishing {
		label = tr("Cancel banish")
		acc = cRed
	}
	u.rawText = true
	if u.button(s, "banish", label, 480, 900, 260, 60, acc, true) && g.stateT > 0.3 {
		g.banishing = !g.banishing
		if g.banishing {
			g.uiBurst(610, 930, 12, cBanish, false)
		}
	}
	u.rawText = false
	if !g.banishing {
		g.banishPips(s, 610, 974)
		g.banishPipPop(s)
	}
	if u.hover(480, 900, 260, 60) {
		g.tooltip = tr("Banish a card: it never appears again this run, and another takes its place.")
	}
	if g.banishing {
		u.text(s, "Pick a card to banish", fBold, 22, 960, 214, cRed, alignCenter)
	}
}

// drawBanishMark tints the cards while banish mode is on.
func (g *Game) drawBanishMark(s *ebiten.Image, x, y, w, h float64) {
	u := g.ui
	fillRect(s, u.X(x), u.Y(y), w*u.S, h*u.S, ca(cRed, 0.10+0.05*pulse01(g.stateT*4)))
}

func pulse01(t float64) float64 { return 0.5 + 0.5*math.Sin(t) }

// lockedStock lists the shop's locked, unsold items by slot.
func (g *Game) lockedStock() map[int]shopItem {
	out := map[int]shopItem{}
	for i, it := range g.shop {
		if it.Kind == "mod" && it.Locked && !it.Sold {
			out[i] = it
		}
	}
	return out
}

// restoreLocked puts locked items back into their slots after a reroll, and
// swaps out any new copy of a locked item.
func (g *Game) restoreLocked(kept map[int]shopItem) {
	for i, it := range kept {
		if i < len(g.shop) && g.shop[i].Kind == "mod" {
			g.shop[i] = it
		}
	}
	for i, it := range g.shop {
		if it.Kind != "mod" || it.Locked {
			continue
		}
		for _, k := range kept {
			if k.Mod.ID == it.Mod.ID {
				g.shop[i].Sold = true // no double offer of a locked item
			}
		}
	}
}

func (g *Game) shopLocks() int {
	n := 0
	for _, it := range g.shop {
		if it.Locked && !it.Sold {
			n++
		}
	}
	return n
}

// drawLockButton is the small lock toggle under a shop item.
func (g *Game) drawLockButton(s *ebiten.Image, i int, x, y, w float64) {
	u := g.ui
	it := &g.shop[i]
	label := "Lock"
	acc := cAcc
	if it.Locked {
		label = "Locked"
		acc = cGold
	}
	can := it.Locked || g.shopLocks() < maxShopLocks
	if u.button(s, "lock"+itoa(i), label, x+w/2-70, y, 140, 44, acc, can) {
		it.Locked = !it.Locked
		if it.Locked {
			g.snd.PlayVol("lock", 0.7)
		} else {
			g.snd.PlayVol("unlock", 0.6)
		}
		if it.Locked {
			g.uiBurst(x+w/2, y+22, 10, cGold, false)
		}
	}
	if u.hover(x+w/2-70, y, 140, 44) {
		g.tooltip = tr("Locked items stay in the shop when you reroll (two at a time).")
	}
}
