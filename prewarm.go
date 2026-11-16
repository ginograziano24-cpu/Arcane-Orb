package main

// Getting the next chamber ready while the player chooses (2026-11-11 #18).
// The worlds' textures are already fetched ahead per act (prefetch.go);
// what a new world still built on entry was its music-keyed contact sounds
// (bricks, walls, the paddle), which are rendered for the current world and
// key only. On the map and the reward screen this warms those sounds for
// every world a reachable chamber will use, and bumps those worlds'
// textures to the front of the queue.

// nextWorlds are the worlds of the chambers reachable from where the run is.
func (r *Run) nextWorlds() []int {
	if r == nil || r.Map == nil {
		return nil
	}
	var out []int
	seen := map[int]bool{}
	for _, i := range r.Reachable() {
		n := r.Map.Nodes[i]
		w := -1
		switch n.Type {
		case NBoss:
			w = acts[r.AI()].Signature
		case NCombat, NElite:
			w = r.pickWorld(r.NodeSeed(i))
		}
		if w >= 0 && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// prewarmNext readies the next chambers' worlds (once per screen visit).
func (g *Game) prewarmNext() {
	r := g.run
	if r == nil || r.Duel || g.snd == nil {
		return
	}
	for _, w := range r.nextWorlds() {
		prefetchWorld(w)
		g.snd.warmWorld(w)
	}
}

// warmWorld queues world w's contact sounds in the current key: what the
// first brick, wall and paddle hits of a chamber there will play.
func (s *Sounds) warmWorld(w int) {
	if w == s.World {
		return // warmKeyed already has these
	}
	ws, k := itoa(w), itoa(s.Key)
	names := []string{"tuf." + ws + "." + k, "deb." + ws, "wall." + ws}
	for st := 0; st < 15; st++ {
		names = append(names, "brk."+ws+"."+k+"."+itoa(st))
	}
	s.warm(names...)
}
