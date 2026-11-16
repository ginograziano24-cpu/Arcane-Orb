package main

import (
	"math"

	"orb/levelgen"
)

// Mid-chamber autosave (2026-11-11 list, item 20). Every 15 seconds of play
// the run is saved with the chamber's progress: which bricks are broken and
// how worn the rest are, the Guardian's health, the barrier and the clock.
// After a crash or a power cut, Continue rebuilds the same chamber from its
// seed and lays that progress back over it, so at most a few seconds are
// lost (the orbs come back to the paddle, as after a lost orb).
//
// Only the first formation of a chamber is kept this way: once a later
// wave has landed, the save falls back to the chamber's start, as before.

const chamberSaveEvery = 15.0

type savedChamber struct {
	Node      int       `json:"node"`
	N         int       `json:"n"`      // bricks when the chamber was built
	Frac      []float32 `json:"frac"`   // per brick: health left (0..1), -1 broken
	T         float64   `json:"t"`      // chamber clock
	BossHP    float64   `json:"boss,omitempty"`
	Barrier   int       `json:"barrier,omitempty"`
	LastWorld int       `json:"lw"` // the world before this chamber (the resume re-picks it)
}

// chamberSnap is the progress worth saving now, or nil.
func (g *Game) chamberSnap() *savedChamber {
	L, r := g.level, g.run
	if L == nil || r == nil || L.Cleared || L.introT > 0 || L.wave > 1 || r.Cur < 0 || L.initBricks == 0 {
		return nil
	}
	c := &savedChamber{Node: r.Cur, N: L.initBricks, T: L.T, Barrier: L.Barrier, LastWorld: L.prevWorld}
	n := min(L.initBricks, len(L.Bricks))
	c.Frac = make([]float32, n)
	for i := 0; i < n; i++ {
		b := L.Bricks[i]
		switch {
		case !b.Alive:
			c.Frac[i] = -1
		case b.MaxHP > 0:
			c.Frac[i] = float32(clamp01(b.HP / b.MaxHP))
		default:
			c.Frac[i] = 1
		}
	}
	if L.Boss != nil {
		c.BossHP = L.Boss.HP / math.Max(1, L.Boss.MaxHP)
	}
	return c
}

// tickChamberSave saves the chamber every 15 s of play (from Update).
func (g *Game) tickChamberSave() {
	L := g.level
	if L == nil || noPersist || g.duel != nil || L.T < L.nextSave {
		return
	}
	L.nextSave = L.T + chamberSaveEvery
	if c := g.chamberSnap(); c != nil {
		g.pendingChamber = c
		g.saveRun("chamber", c.Node)
		g.pendingChamber = nil
	}
}

// resumeChamber rebuilds the saved chamber and lays its progress back on.
func (g *Game) resumeChamber(c *savedChamber) bool {
	r := g.run
	if c == nil || c.Node < 0 || c.Node >= len(r.Map.Nodes) {
		return false
	}
	// newLevel counts the chamber and picks its world again: undo both, so
	// it is the same chamber in the same world
	r.Depth = max(0, r.Depth-1)
	r.ActDepth = max(0, r.ActDepth-1)
	r.LastWorld = c.LastWorld
	g.enterNode(c.Node)
	L := g.level
	if L == nil || L.initBricks != c.N {
		return true // the chamber came out different: start it fresh
	}
	for i, f := range c.Frac {
		if i >= len(L.Bricks) {
			break
		}
		b := L.Bricks[i]
		if !b.Alive {
			continue
		}
		if f < 0 {
			b.Alive = false
			L.removeFromColumn(b)
			if b.Kind.Required() && !b.BossPart {
				L.Left--
			}
			if b.Kind == levelgen.Crystal {
				L.Crystals--
			}
			continue
		}
		b.HP = math.Max(0.1, b.MaxHP*float64(f))
	}
	if L.Boss != nil && c.BossHP > 0 {
		L.Boss.HP = L.Boss.MaxHP * c.BossHP
	}
	L.Barrier = max(L.Barrier, c.Barrier)
	L.T = c.T
	L.nextSave = L.T + chamberSaveEvery
	g.toast("Back where you were", "The chamber was saved mid-play; the orb returns to your paddle.", cGreen, icStar)
	return true
}
