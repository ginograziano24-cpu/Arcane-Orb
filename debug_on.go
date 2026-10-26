//go:build debug

package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"orb/levelgen"
)

// Developer shortcuts, only compiled with -tags debug.
func debugMenuKeys(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.save.XP += 3000
		g.save.Shards += 900
		g.save.AscMax = 3
		for i, m := range modList {
			if i%3 != 2 {
				g.save.Seen[m.ID] = true
			}
		}
		for i, f := range fusions {
			if i%2 == 0 {
				g.save.Fusions[f.ID] = true
			}
		}
	}
}

func debugKeys(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) && g.level != nil && !g.level.Cleared {
		if g.level.Boss != nil {
			g.level.killBoss()
			g.level.Boss.deathT = 3
		} else {
			g.level.clear()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		m := g.run.Offer(1, true, 0, rand.New(rand.NewSource(g.rng.Int63())))
		if len(m) > 0 {
			g.take(m[0])
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) && g.level != nil {
		forceWorld = (forceWorld + 1) % NumWorlds
		g.run.Depth += 2
		g.level = g.newLevel(NCombat, g.rng.Int63())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) && g.level != nil {
		levelgen.DesignChance = 1
		g.run.Depth += 2
		g.level = g.newLevel(NCombat, g.rng.Int63())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) && g.level != nil {
		g.run.Depth += 3
		g.level = g.newLevel(NCombat, g.rng.Int63())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) && g.run != nil {
		// force a fusion: take both halves of Plasma Storm
		g.take(modByID["kindling"])
		g.take(modByID["static"])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) && g.run != nil {
		// showcase reward: one card completes a fusion with an owned item
		g.take(modByID["frost"])
		g.rewards = []rewardSet{{Title: "Choose an upgrade", Mods: []*Mod{modByID["blast"], modByID["excalibur"], modByID["hive"]}, Skip: 15}}
		g.setState(StReward)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.run.Gold += 200
	}
}
