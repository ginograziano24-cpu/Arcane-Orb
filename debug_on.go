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

var dbgGame *Game

func debugMenuKeys(g *Game) {
	dbgGame = g
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
		g.save.Bosses = max(1, g.save.Bosses)
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

var dbgPending []string

// debugPre runs every frame, even while a coach tip is up.
func debugPre(g *Game) {
	dbgGame = g
	keep := dbgPending[:0]
	for _, a := range dbgPending {
		if a == "mastery" {
			g.save.Mastery, g.save.MOpts = map[string]int{}, map[string]int{}
			for i, ar := range archetypes {
				g.save.Mastery[ar.ID] = []int{0, 900, 2500, 5000, 8100, 10000, 400}[i%7]
				g.save.MOpts[ar.ID] = 15
				g.save.Unlocked[ar.ID] = true
			}
			continue
		}
		if a == "notips" {
			g.settings.NoTips = true
			g.coachQ = nil
			continue
		}
		keep = append(keep, a)
	}
	dbgPending = keep
}

func debugKeys(g *Game) {
	acts := dbgPending
	dbgPending = nil
	want := func(name string) bool {
		for _, a := range acts {
			if a == name {
				return true
			}
		}
		return false
	}
	// "crescendo": Crescendo and a running combo; "finale": break every brick
	if (inpututil.IsKeyJustPressed(ebiten.KeyNumpad1) || want("crescendo")) && g.level != nil {
		g.take(modByID["crescendo"])
		g.level.Combo, g.level.comboT = 18, 30
	}
	if want("critters") && g.level != nil {
		L := g.level
		L.spawnCritter()
		L.spawnCritter()
		if len(L.Critters) > 0 {
			L.Critters[len(L.Critters)-1].Golden = true
		}
		for _, c := range L.Critters {
			c.Life = 60
		}
	}
	if want("nextworld") && g.level != nil {
		forceWorld = (forceWorld + 1) % NumWorlds
		g.level = g.newLevel(NCombat, g.rng.Int63())
		g.settings.NoTips = true
	}
	if want("bumps") && g.level != nil {
		e := &g.level.env
		e.bump = [3]float64{0.3, 0.3, 0.3}
	}
	if want("finalstate") && g.level != nil {
		e := &g.level.env
		e.prog, e.door, e.thump = 1, 1, 1
		for k := range e.lit {
			e.lit[k] = 1
		}
		g.level.wave, g.level.waves = 2, 2
	}
	if want("twowaves") && g.level != nil {
		g.level.waves = 2
		g.settings.NoTips = true
	}
	if want("advance") && g.level != nil {
		g.level.shiftTarget += CellD * 5
	}
	if want("cache") && g.level != nil {
		g.level.Drops = append(g.level.Drops, &Drop{X: g.level.PaddleX, Z: PaddleZ + 4, P: PCache})
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyNumpad2) || want("finale")) && g.level != nil && !g.level.Cleared {
		for _, b := range g.level.Bricks {
			if b.Alive && b.Kind.Required() {
				g.level.destroy(b, Hit{Dmg: 99})
			}
		}
	}
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) && g.level != nil {
		g.level.Drops = append(g.level.Drops, &Drop{X: g.level.PaddleX, Z: 14, P: PLife})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		g.run.Gold += 200
		for len(g.run.Scrolls) < g.run.scrollCap() {
			g.addScroll(randomScroll(g.rng))
		}
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
