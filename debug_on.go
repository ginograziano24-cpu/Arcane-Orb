//go:build debug

package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"orb/levelgen"
)

// Developer shortcuts, only compiled with -tags debug.
// iconSheet > 0 shows a page of Steam achievement icons for capture.
var iconSheet int

func debugMenuKeys(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		iconSheet = (iconSheet + 1) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) { // unlock the next achievement
		for _, a := range achievements {
			if !g.save.Ach[a.ID] {
				g.unlock(a.ID)
				break
			}
		}
	}
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) && g.run != nil {
		g.run.Act = 3
		g.run.Bosses = 3
		g.endRun(true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.run.Gold += 200
	}
}

// debugMapKeys: on the map, digits 1-7 jump straight into a node of that
// type (combat, elite, shop, treasure, event, rest, boss).
func debugMapKeys(g *Game) {
	for d := 0; d < 7; d++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(d)) {
			for i, n := range g.run.Map.Nodes {
				if int(n.Type) == d {
					g.enterNode(i)
					return
				}
			}
		}
	}
}

// debugOverlay draws the achievement icon sheet (256px cells on a grid of
// 7x4 at 20,20 with an 8px gap) so a capture script can crop Steam icons.
func debugOverlay(g *Game, dst *ebiten.Image) {
	if iconSheet == 0 {
		return
	}
	u := g.ui
	fillRect(dst, 0, 0, 4000, 4000, rgba(0, 0, 0, 255))
	for i := 0; i < 28; i++ {
		k := (iconSheet-1)*28 + i
		if k >= len(achievements) {
			break
		}
		x := 20 + float64(i%7)*264
		y := 20 + float64(i/7)*264
		drawAchIcon(u, dst, &achievements[k], x, y, 256)
	}
}
