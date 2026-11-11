package main

import "math"

// Anti-stall. An orb can fall into a loop between the walls and steel
// (unbreakable) or shielded bricks, never touching the paddle and never
// hurting anything, and the chamber just... waits. Once an orb has gone
// stallAfter seconds without real progress, each wall bounce turns it a
// little further towards the paddle, until it comes home or hits something
// that breaks. Normal play never notices: a real rally resets the clock on
// every paddle touch and every brick hurt.

const (
	stallAfter = 9.0  // seconds without progress before nudging
	stallTurn  = 0.16 // radians turned per wall bounce while stalled (about 9°)
)

// stallNudge bends a stalled orb's path towards the paddle on a wall bounce.
func (L *Level) stallNudge(b *Ball) {
	if b.Idle < stallAfter || b.Stuck {
		return
	}
	sp := math.Hypot(b.VX, b.VZ)
	if sp == 0 {
		return
	}
	// rotate the velocity so its z component (towards the far wall is +z)
	// becomes more negative, i.e. heads back down to the paddle
	a := math.Atan2(b.VX, b.VZ) // 0 = straight up the arena, ±π = straight down
	if a >= 0 {
		a = math.Min(math.Pi*0.8, a+stallTurn)
	} else {
		a = math.Max(-math.Pi*0.8, a-stallTurn)
	}
	b.VX, b.VZ = sp*math.Sin(a), sp*math.Cos(a)
	// fixAngle forbids near-flat paths (|vz| < 0.3 of the speed). Jump the
	// band towards the paddle rather than letting fixAngle push it back up.
	if math.Abs(b.VZ) < 0.3*sp {
		b.VZ = -0.3 * sp
		b.VX = math.Copysign(math.Sqrt(sp*sp-b.VZ*b.VZ), b.VX)
	}
}
