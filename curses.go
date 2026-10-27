package main

import "math/rand"

// Curses are the price of power: events and shrines offer strong rewards
// that come with one. They sit in your build in crimson until cleansed at
// a campfire, a shop or a kind spirit.

var curseList = []*Mod{
	{ID: "c_brittle", Icon: "paddle", Name: "Brittle Paddle", Desc: "Curse: your paddle is 15% narrower.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.PaddleWidth -= 0.15 }},
	{ID: "c_leaden", Icon: "anvil", Name: "Leaden Orb", Desc: "Curse: your orb deals 12% less damage.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.DmgMult *= 0.88 }},
	{ID: "c_haste", Icon: "comet", Name: "Hex of Haste", Desc: "Curse: orbs fly 12% faster.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.SpeedMult += 0.12 }},
	{ID: "c_purse", Icon: "coin", Name: "Thin Purse", Desc: "Curse: you find 30% less gold.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.GoldMult -= 0.3 }},
	{ID: "c_fragile", Icon: "shatter", Name: "Fragile Soul", Desc: "Curse: dropping the orb also costs 15 health.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.BallLossDmg += 15 }},
	{ID: "c_dim", Icon: "moon", Name: "Dimmed Fortune", Desc: "Curse: power-ups drop 40% less often.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.Luck -= 0.4 }},
	{ID: "c_weight", Icon: "urn", Name: "Lattice's Weight", Desc: "Curse: bricks have 15% more health.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.BrickHPMult += 0.15 }},
	{ID: "c_doubt", Icon: "mask", Name: "Shaken Resolve", Desc: "Curse: 15 less max health.", Curse: true, Tags: []string{"curse"},
		Apply: func(s *Stats) { s.MaxHP -= 15 }},
}

// curses lists the curses the run carries.
func (r *Run) curses() []string {
	var out []string
	for _, id := range r.Order {
		if m := modByID[id]; m != nil && m.Curse && r.Mods[id] > 0 {
			out = append(out, id)
		}
	}
	return out
}

// addCurse lays a random curse the run doesn't have yet.
func (g *Game) addCurse(rng *rand.Rand) *Mod {
	r := g.run
	var pool []*Mod
	for _, c := range curseList {
		if r.Mods[c.ID] == 0 {
			pool = append(pool, c)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	c := pool[rng.Intn(len(pool))]
	r.Add(c.ID, &g.save)
	g.toastImg("Cursed", c.Name+" — "+c.Desc[len("Curse: "):], tagColors["curse"], modEmblem(c))
	return c
}

// cleanse lifts one curse (the oldest); it reports which.
func (g *Game) cleanse() *Mod {
	r := g.run
	cs := r.curses()
	if len(cs) == 0 {
		return nil
	}
	id := cs[0]
	r.Mods[id] = 0
	for i, o := range r.Order {
		if o == id {
			r.Order = append(r.Order[:i], r.Order[i+1:]...)
			break
		}
	}
	r.Recompute(&g.save)
	baseMaxPenalty(r)
	g.snd.PlayKeyed("heal")
	return modByID[id]
}

// empowerable lists owned upgrades that a campfire can strengthen.
func (r *Run) empowerable() []*Mod {
	var out []*Mod
	for _, id := range r.Order {
		m := modByID[id]
		if m == nil || m.Relic || m.Curse || r.Mods[id] == 0 {
			continue
		}
		if m.Max > 0 && r.Mods[id] >= m.Max {
			continue
		}
		out = append(out, m)
	}
	return out
}
