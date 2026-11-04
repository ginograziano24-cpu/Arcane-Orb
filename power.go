package main

import "math"

// Builds carry over between acts, and the chambers scale to the build.
//
// The game watches how fast the build actually clears. After each combat
// or elite chamber it compares the seconds of active play (orbs in flight,
// no intros or waves dropping in) with a target for the act, and moves the
// run's brick-health scale part of the way toward what would have hit it:
// scale *= (target / actual)^powerGain. The scale becomes heat, which
// already drives brick health, the armour share, the advance and the
// bosses' and critters' toughness. The partial step and targets that fall a
// little each act keep a stronger build feeling stronger; the scale never
// drops below the plain per-act curve and is capped per act. Measuring
// time rather than modelling damage catches every synergy, however odd.

const (
	// powerGain is how much of the gap to the target one chamber closes;
	// powerInteg weighs the running sum of past gaps.
	powerGain  = 0.6
	powerInteg = 0.4
	// powerMinActive is the least active play a chamber needs to count.
	powerMinActive = 15.0
	// shopInflation raises prices per act once gold carries over.
	shopInflation = 0.2
	// powerPerItem is the feed-forward: every item (a fusion counts two)
	// taken since the last measured chamber raises the scale this much at
	// once, so the bricks keep up with a build that grows every chamber
	// instead of trailing it; the measurement then corrects the guess.
	powerPerItem = 1.12
)

// powerTarget is the active seconds a chamber should take in an act.
// (Acts 2 and 3 aim a little long to offset the lag of chasing a build
// that strengthens every chamber; seconds per brick still fall.)
func (r *Run) powerTarget() float64 {
	return [3]float64{140, 135, 125}[min(r.Act-1, 2)]
}

// powerOff pins the scale at 1 (balance harness experiments).
var powerOff bool

// carries reports whether this run keeps its build between acts.
func (r *Run) carries() bool { return r.Carry && !r.Duel }

// powerCapScale is the most the scale may reach in the current act.
func (r *Run) powerCapScale() float64 {
	return [3]float64{4, 40, 150}[min(r.Act-1, 2)] * math.Pow(2, float64(r.Loop()))
}

// powerScale is the brick-health scale the next chamber gets: the measured
// scale, raised by the feed-forward for items taken since, capped per act.
func (r *Run) powerScale() float64 {
	if powerOff {
		return 1
	}
	s := math.Max(1, r.Power)
	if r.Act > 1 {
		// act 1's own curve already expects a growing build
		s *= math.Pow(powerPerItem, float64(r.buildSize()-r.PowerItems))
	}
	return math.Max(1, math.Min(r.powerCapScale(), s))
}

// powerHeat turns the run's scale into heat (heatHP of brick health each).
func (r *Run) powerHeat() int {
	return int(math.Round((r.powerScale() - 1) / heatHP))
}

// buildSize counts the build: every item copy, two per fusion.
func (r *Run) buildSize() int {
	n := 0
	for id, k := range r.Mods {
		if m := modByID[id]; m != nil && !m.Curse && id[0] != '_' {
			n += k
		}
	}
	return n + 2*len(r.Fused)
}

// itemPower estimates a scale from the build's size, the way heat used to
// count it, for runs saved before the scale was measured.
func (r *Run) itemPower() float64 {
	return 1 + heatHP*float64(r.buildSize())
}

// notePower moves the scale after a finished chamber: a proportional step
// toward the target time plus an integral of past misses, so steady growth
// (a build that gets stronger every chamber) is met rather than chased.
func (r *Run) notePower(L *Level) {
	if !r.carries() || L.Boss != nil || L.Node != NCombat {
		return // elites and bosses play at their own share and don't steer it
	}
	if L.activeT < powerMinActive {
		return
	}
	target := r.powerTarget()
	e := math.Log(target / L.activeT)
	r.PowerI = math.Max(-1.5, math.Min(1.5, r.PowerI+e))
	step := math.Max(math.Log(0.6), math.Min(math.Log(2.2), powerGain*e+powerInteg*r.PowerI))
	used := L.powerUsed // the scale this chamber was built with
	if used <= 0 {
		used = r.powerScale()
	}
	s := used * math.Exp(step)
	r.Power = math.Max(1, math.Min(r.powerCapScale(), s))
	r.PowerItems = r.buildSize() // the feed-forward starts again from here
	if r.Power <= 1 && r.PowerI < 0 {
		r.PowerI = 0 // at the floor, misses on the slow side don't pile up
	}
	r.PowerN++
}

// powerRate is the chamber's brick health removed per second of active
// play in base-health units (for the balance harness).
func (L *Level) powerRate() float64 {
	if L.activeT <= 0 {
		return 0
	}
	return L.work / L.hpMul() / L.activeT
}

// critterPace speeds critter visits up in later acts of a carried run.
func (r *Run) critterPace() float64 {
	if !r.carries() {
		return 1
	}
	return 1 + 0.15*float64(min(r.Act-1, 3))
}

// ballsInPlay reports whether any real orb is in flight.
func (L *Level) ballsInPlay() bool {
	for _, b := range L.Balls {
		if !b.Stuck && b.Phantom <= 0 {
			return true
		}
	}
	return false
}
