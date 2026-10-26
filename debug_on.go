//go:build debug

package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"orb/levelgen"
)

// Developer shortcuts, only compiled with -tags debug.
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.run.Gold += 200
	}
}
