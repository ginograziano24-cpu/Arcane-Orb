package main

import "strconv"

// hudNum builds a HUD label from a prefix, a whole number and a suffix, and
// remembers it, so labels that are redrawn every frame (combo, timers, gold)
// don't allocate a new string each frame. Each call site passes its own
// small cache.
type hudNum struct {
	v  int
	s  string
	ok bool
}

func (c *hudNum) get(pre string, v int, post string) string {
	if !c.ok || c.v != v {
		c.v, c.s, c.ok = v, pre+strconv.Itoa(v)+post, true
	}
	return c.s
}

var (
	hudCrystals, hudFrenzy, hudGold, hudAbil hudNum
)
