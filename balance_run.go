//go:build capture && js

package main

// The whole-run balance harness: each origin plays acts 1-3 as one run,
// carrying its build, under the same imperfect autopilot as the chamber
// harness (balance.go). An act is five combats, an elite and the boss; a
// random upgrade follows every combat and a relic every elite and boss.
// Call from the page:
//
//	__campaign('{"seeds":3,"maxT":400,"carry":true}')

import (
	"encoding/json"
	"math"
	"math/rand"
	"syscall/js"
)

type campRow struct {
	Arch     string
	Seed     int
	Act      int
	Node     string
	Idx      int
	Time     float64
	Cleared  bool
	Bricks   int     // bricks broken in the chamber
	PerBrick float64 // seconds per brick broken
	Power    float64 // the run's power after the chamber
	Heat     int     // the chamber's heat
	Items    int
	Lost     int     // orbs dropped
	Rate     float64 // brick health removed per second of active play
	Speed    float64 // orb speed at the end of the chamber
	Front    float64 // mean distance from the paddle to the formation's front
	Crush    int     // bricks that reached the danger line
	PadW     float64 // paddle width
}

var campPath = []NodeType{NCombat, NCombat, NCombat, NElite, NCombat, NCombat, NBoss}

func runCampaign(g *Game, seeds int, maxT float64, archs []string, carry bool, acts int) []campRow {
	var out []campRow
	const dt = 1.0 / 60
	for ai := range archetypes {
		arch := &archetypes[ai]
		if len(archs) > 0 && !contains(archs, arch.ID) {
			continue
		}
		for s := 0; s < seeds; s++ {
			sv := defaultSave()
			sv.Settings.FreshActs = !carry
			r := NewRun("CAMP", arch, &sv, false)
			r.Lives = 99
			g.run = r
			pick := rand.New(rand.NewSource(int64(1000*ai + 77*s + 5)))
			for act := 1; act <= acts; act++ {
				if act > 1 {
					r.NextAct(&sv)
					r.Lives = 99
				}
				for i, node := range campPath {
					seed := int64(100000*act + 1000*s + 37*i + ai)
					L := g.newLevel(node, seed)
					g.level, g.state = L, StLevel
					ap := &sloppy{rng: rand.New(rand.NewSource(seed * 7)), errSD: 1.4, reactMS: 180}
					bricks0 := r.Bricks
					lost := 0
					t := 0.0
					fsum, fn := 0.0, 0
					for ; t < maxT && !L.Cleared; t += dt {
						if fz := L.frontZ(); fz < FarZ {
							fsum += fz - PaddleZ
							fn++
						}
						lb := L.lostOrbs
						L.Update(dt, ap.input(L, dt))
						lost += L.lostOrbs - lb
						for L.pendingPick > 0 {
							L.pendingPick--
							if ms := r.Offer(1, false, Common, pick); len(ms) > 0 {
								r.Add(ms[0].ID, &sv)
							}
						}
						r.Lives = 99
					}
					if !L.Cleared {
						r.notePower(L) // a real chamber ends eventually and is measured then
					}
					nb := r.Bricks - bricks0
					row := campRow{Arch: arch.ID, Seed: s, Act: act, Node: campNodeNames[node], Idx: i, Time: t, Cleared: L.Cleared,
						Bricks: nb, Power: r.Power, Heat: L.heat, Items: len(r.Order), Lost: lost}
					if nb > 0 {
						row.PerBrick = t / float64(nb)
					}
					row.Rate = L.powerRate()
					row.Speed, row.PadW = L.speed(), L.PaddleW
					row.Front, row.Crush = fsum/math.Max(1, float64(fn)), L.crushes
					out = append(out, row)
					// the rewards a typical path brings
					relic := node != NCombat
					if ms := r.Offer(1, relic, Common, pick); len(ms) > 0 {
						r.Add(ms[0].ID, &sv)
					}
				}
			}
		}
	}
	g.state = StMenu
	return out
}

var campNodeNames = map[NodeType]string{NCombat: "combat", NElite: "elite", NBoss: "boss"}

func init() {
	js.Global().Set("__campaign", js.FuncOf(func(this js.Value, args []js.Value) any {
		p := struct {
			Seeds   int
			MaxT    float64
			Archs   []string
			Carry   bool
			Acts    int
			NoScale bool
		}{3, 400, nil, true, 3, false}
		if len(args) > 0 {
			_ = json.Unmarshal([]byte(args[0].String()), &p)
		}
		powerOff = p.NoScale
		b, _ := json.Marshal(runCampaign(theGame, p.Seeds, p.MaxT, p.Archs, p.Carry, p.Acts))
		powerOff = false
		return string(b)
	}))
}
