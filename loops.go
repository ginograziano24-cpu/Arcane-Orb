package main

import (
	"math"

	"orb/levelgen"
)

// Endless loop twists (ROADMAP-50 #31). Past the final Guardian every loop
// adds one named twist on top of the old ones (and on top of the usual
// tougher bricks and faster orbs), so each loop plays a little differently
// instead of only a little harder. The list alternates trouble with a
// helping hand; after the last twist it starts again and they stack.

type loopTwist struct {
	Name, Desc string
	Glyph      string
	Apply      func(L *Level) // once, when a chamber starts
}

var loopTwists = []loopTwist{
	{Name: "Powder Kegs", Desc: "Some bricks are packed with blasting powder.", Glyph: "bomb",
		Apply: func(L *Level) { L.convertBricks(levelgen.Explosive, 0.06) }},
	{Name: "Iron Bark", Desc: "Bricks grow 20% more health.", Glyph: "shield",
		Apply: func(L *Level) {
			for _, b := range L.Bricks {
				if b.Kind != levelgen.Steel && !b.BossPart {
					b.MaxHP = math.Ceil(b.MaxHP * 1.2)
					b.HP = b.MaxHP
				}
			}
		}},
	{Name: "Twin Moons", Desc: "You serve with an extra orb.", Glyph: "twinorb",
		Apply: func(L *Level) {
			r := L.g.run
			L.Balls = append(L.Balls, &Ball{R: L.ballR(), Stuck: true, Off: -0.9, X: L.PaddleX, Z: PaddleZ + 0.6, PierceLeft: r.Stats.Pierce})
		}},
	{Name: "Critter Season", Desc: "Critters turn up twice as often.", Glyph: "feather",
		Apply: func(L *Level) { L.critT = 4 }}, // the pace itself is in loopCritterPace
	{Name: "Crystal Bloom", Desc: "Gold-rich crystals grow among the bricks.", Glyph: "gem",
		Apply: func(L *Level) { L.convertBricks(levelgen.Crystal, 0.05) }},
}

// activeTwists lists the loop's twists in the order they arrived (loop 1
// has one, loop 2 two, ...).
func (r *Run) activeTwists() []*loopTwist {
	var out []*loopTwist
	for i := 0; i < r.Loop(); i++ {
		out = append(out, &loopTwists[i%len(loopTwists)])
	}
	return out
}

// newestTwist is the twist this loop added, or nil before Endless.
func (r *Run) newestTwist() *loopTwist {
	if l := r.Loop(); l > 0 {
		return &loopTwists[(l-1)%len(loopTwists)]
	}
	return nil
}

// loopCritterPace speeds critters up for each Critter Season in play.
func (r *Run) loopCritterPace() float64 {
	p := 1.0
	for _, t := range r.activeTwists() {
		if t.Name == "Critter Season" {
			p *= 2
		}
	}
	return p
}

// applyLoopTwists runs once as a chamber starts.
func (L *Level) applyLoopTwists() {
	r := L.g.run
	if r == nil || r.Duel {
		return
	}
	for _, t := range r.activeTwists() {
		t.Apply(L)
	}
}
