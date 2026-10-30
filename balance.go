//go:build capture && js

package main

// A headless balance harness: every origin plays the same chambers with
// the same items under an imperfect autopilot, and we compare how fast
// they clear and how much they get hurt. Call from the page:
//
//	__balance('{"seeds":6,"maxT":150}')

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"syscall/js"
)

type balScenario struct {
	Name  string
	Act   int
	Depth int
	Gold  int
	Boss  bool
	Elite bool
	Mods  []string
}

var balScenarios = []balScenario{
	{Name: "a1-early", Act: 1, Depth: 2},
	{Name: "a1-late", Act: 1, Depth: 6, Mods: []string{"heavy", "sharp", "twin"}},
	{Name: "a1-boss", Act: 1, Depth: 8, Boss: true, Gold: 80, Mods: []string{"heavy", "sharp", "twin", "kindling"}},
	{Name: "a2-mid", Act: 2, Depth: 10, Gold: 150, Mods: []string{"heavy", "sharp", "twin", "kindling", "temper", "wide", "fort", "blast"}},
	{Name: "a3-mid", Act: 3, Depth: 16, Gold: 220, Mods: []string{"heavy", "sharp", "twin", "kindling", "temper", "wide", "fort", "blast",
		"static", "pierce", "brutal", "comboeng", "frost", "eagle", "afterburn"}},
}

type balResult struct {
	Arch, Scen       string
	Seeds, Cleared   int
	Time, HPLost     float64
	Orbs, Hits, Abil float64
	Left, Start      float64
	T50, T90, Bounce float64
	DPS, Combo       float64
	Caches, Waves    float64
	Taken, Crushes   float64
}

// sloppy is the autopilot with human limits: it re-reads the play only
// every so often, misjudges where the orb lands, and is not always fast.
type sloppy struct {
	rng     *rand.Rand
	target  float64
	think   float64
	stuckT  float64
	errSD   float64
	reactMS float64
}

func (a *sloppy) input(L *Level, dt float64) Input {
	in := Input{UseKeys: true}
	a.think -= dt
	if a.think <= 0 {
		a.think = a.reactMS / 1000 * (0.7 + 0.6*a.rng.Float64())
		zp := PaddleZ + PaddleD/2
		bt := math.Inf(1)
		x := L.PaddleX
		for _, b := range L.Balls {
			if b.Stuck || b.VZ >= 0 || b.Phantom > 0 {
				continue
			}
			t := (b.Z - zp - b.R) / -b.VZ
			if t < bt {
				bt = t
				x = foldX(b.X+b.VX*t, HalfW-b.R)
			}
		}
		// with nothing urgent, go and catch a falling power-up
		if bt > 1.2 && a.rng.Float64() < 0.7 {
			for _, d := range L.Drops {
				if d.Z < 12 && powerDefs[d.P].Good {
					x, bt = d.X, 0
					break
				}
			}
		}
		// the further away the orb, the bigger the misread
		err := a.rng.NormFloat64() * a.errSD * math.Min(1.6, 0.4+bt*0.5)
		a.target = x + err + (a.rng.Float64()*2-1)*L.PaddleW*0.25
	}
	d := a.target - L.PaddleX
	in.Fast = math.Abs(d) > 2.5
	sp := 30.0
	if in.Fast {
		sp = 46
	}
	in.KeyDir = math.Max(-1, math.Min(1, d/(sp*dt)))
	if L.anyStuck() && L.introT <= 0 {
		a.stuckT += dt
		if a.stuckT > 0.6 {
			in.Fire, a.stuckT = true, 0
		}
	}
	if L.g.run.hasAbility() && L.g.run.AbilT == 0 {
		in.Ability = true
	}
	return in
}

func runBalance(g *Game, seeds int, maxT, errSD, react float64, scens []balScenario, archs []string) []balResult {
	if len(scens) == 0 {
		scens = balScenarios
	}
	rec := capState.recording
	capState.recording = false
	defer func() { capState.recording = rec }()
	var out []balResult
	for ai := range archetypes {
		arch := &archetypes[ai]
		if len(archs) > 0 && !contains(archs, arch.ID) {
			continue
		}
		for _, sc := range scens {
			res := balResult{Arch: arch.ID, Scen: sc.Name, Seeds: seeds}
			for s := 0; s < seeds; s++ {
				sv := defaultSave()
				r := NewRun("SIM", arch, &sv, false)
				r.Act = sc.Act
				for _, id := range sc.Mods {
					r.Add(id, &sv)
				}
				r.Gold = sc.Gold
				r.Depth = sc.Depth - 1
				r.ActDepth = sc.Depth - 1
				r.Lives = 99 // the harness measures pace, not survival
				g.run = r
				node := NCombat
				if sc.Boss {
					node = NBoss
				}
				if sc.Elite {
					node = NElite
				}
				seed := int64(1000*sc.Act + 37*s + sc.Depth)
				L := g.newLevel(node, seed)
				g.level, g.state = L, StLevel
				r.HP = r.Stats.MaxHP
				ap := &sloppy{rng: rand.New(rand.NewSource(seed * 7)), errSD: errSD, reactMS: react}
				const dt = 1.0 / 60
				hp0 := r.HP
				res.Start += float64(L.Left)
				lost, abil := 0, 0
				t := 0.0
				t50, t90 := maxT, maxT
				pads := 0
				for ; t < maxT && !L.Cleared; t += dt {
					if float64(L.Left) <= float64(L.startLeft)*0.5 && t50 == maxT {
						t50 = t
					}
					if float64(L.Left) <= float64(L.startLeft)*0.1 && t90 == maxT {
						t90 = t
					}
					for _, b := range L.Balls {
						if !b.Stuck && b.VZ < 0 && b.Z < PaddleZ+PaddleD/2+b.R+0.3 {
							pads++
						}
					}
					lb := L.lostOrbs
					in := ap.input(L, dt)
					if in.Ability {
						abil++
					}
					hpB := r.HP
					L.Update(dt, in)
					if r.HP < hpB {
						res.Taken += hpB - r.HP
					}
					for L.pendingPick > 0 { // the autopilot takes a random cache item
						L.pendingPick--
						res.Caches++
						if ms := r.Offer(1, false, Common, ap.rng); len(ms) > 0 {
							r.Add(ms[0].ID, &sv)
						}
					}
					lost += L.lostOrbs - lb
					if r.HP < 1e5 { // never die in the harness; count the damage
						hp0 += 1e6
						r.HP += 1e6
					}
				}
				if L.Cleared {
					res.Cleared++
				}
				res.Left += float64(L.Left)
				res.T50 += math.Min(t50, t)
				res.T90 += math.Min(t90, t)
				res.Bounce += float64(pads) / t
				hits := 0
				for _, b := range L.Bricks {
					if b.Alive && b.Kind.Required() {
						hits++
					}
				}
				res.Hits += float64(hits)
				tot := 0.0
				for _, v := range r.rc().Dmg {
					tot += v
				}
				res.DPS += tot / t
				res.Combo += float64(L.peakCombo)
				res.Waves += float64(L.waves)
				res.Crushes += float64(L.crushes)
				res.Time += t
				res.HPLost += hp0 - r.HP
				res.Orbs += float64(lost)
				res.Abil += float64(abil)
			}
			n := float64(seeds)
			res.Time /= n
			res.HPLost /= n
			res.Orbs /= n
			res.Abil /= n
			res.Left /= n
			res.T50 /= n
			res.T90 /= n
			res.Bounce /= n
			res.Start /= n
			res.Hits /= n
			res.DPS /= n
			res.Combo /= n
			res.Caches /= n
			res.Taken /= n
			res.Crushes /= n
			res.Waves /= n
			out = append(out, res)
		}
	}
	g.state = StMenu
	return out
}

func init() {
	js.Global().Set("__balance", js.FuncOf(func(this js.Value, args []js.Value) any {
		p := struct {
			Seeds int
			MaxT  float64
			Err   float64
			React float64
			Scens []balScenario
			Archs []string
		}{6, 150, 1.4, 180, nil, nil}
		if len(args) > 0 {
			_ = json.Unmarshal([]byte(args[0].String()), &p)
		}
		b, _ := json.Marshal(runBalance(theGame, p.Seeds, p.MaxT, p.Err, p.React, p.Scens, p.Archs))
		return string(b)
	}))
}

func init() {
	js.Global().Set("__levelStats", js.FuncOf(func(this js.Value, args []js.Value) any {
		g := theGame
		type row struct {
			Act, Depth           int
			Bricks, HP, Designed float64
			Min, Max             float64
		}
		var out []row
		for act := 1; act <= 3; act++ {
			for d := 1; d <= 14; d += 1 {
				rw := row{Act: act, Depth: d, Min: 1e9}
				n := 40
				for s := 0; s < n; s++ {
					sv := defaultSave()
					r := NewRun("SIM", &archetypes[0], &sv, false)
					r.Act, r.Depth = act, d-1
					g.run = r
					L := g.newLevel(NCombat, int64(s*7919+d*31+act))
					hp := 0.0
					for _, b := range L.Bricks {
						if b.Kind.Required() {
							hp += b.HP
						}
					}
					rw.Bricks += float64(L.Left)
					rw.HP += hp
					rw.Min = math.Min(rw.Min, float64(L.Left))
					rw.Max = math.Max(rw.Max, float64(L.Left))
					if L.Designed {
						rw.Designed++
					}
				}
				rw.Bricks /= float64(n)
				rw.HP /= float64(n)
				out = append(out, rw)
			}
		}
		b, _ := json.Marshal(out)
		return string(b)
	}))
}

func init() {
	// __waveTest exercises waves, the advance, crushing and rune caches
	js.Global().Set("__waveTest", js.FuncOf(func(this js.Value, args []js.Value) any {
		g := theGame
		var log []string
		say := func(f string, a ...any) { log = append(log, fmt.Sprintf(f, a...)) }
		sv := defaultSave()
		g.save = sv
		r := NewRun("WAVES", archByID("storm"), &g.save, false)
		r.ActDepth = 3
		g.run = r
		g.hero = heroPortrait{}
		L := g.newLevel(NCombat, 4242)
		g.level, g.state = L, StLevel
		say("waves=%d wave=%d left=%d heat=%d adv=%.1fs cacheAt=%.2f", L.waves, L.wave, L.Left, L.heat, L.advT, L.cacheAt)
		step := func(n int) {
			for i := 0; i < n; i++ {
				L.Update(1.0/60, Input{UseKeys: true})
			}
		}
		L.release()
		front := L.frontZ()
		L.shiftTarget = CellD * 28
		hp := r.HP
		step(400)
		say("advanced: shift=%.2f front %.1f->%.1f left=%d hp %.0f->%.0f", L.shift, front, L.frontZ(), L.Left, hp, r.HP)
		// break most of wave 1: it should break early and bring wave 2
		n := 0
		for _, b := range L.Bricks {
			if b.Alive && b.Kind.Required() && float64(n) < float64(L.startLeft)*0.9 {
				L.destroy(b, Hit{Dmg: 99})
				n++
			}
		}
		step(2)
		say("after wave 1: wave=%d left=%d startLeft=%d shift=%.1f balls=%d cleared=%v caches=%d drops=%d", L.wave, L.Left, L.startLeft, L.shift, len(L.Balls), L.Cleared, L.caches, len(L.Drops))
		// a cache at the paddle
		L.Drops = append(L.Drops, &Drop{X: L.PaddleX, Z: PaddleZ + 1, P: PCache})
		step(30)
		say("cache caught: pending=%d", L.pendingPick)
		mods := len(r.Order)
		g.openCache()
		say("cache state=%v offers=%d", g.state == StCache, len(g.cacheOffers))
		g.take(g.cacheOffers[0])
		L.pendingPick = 0
		g.state = StLevel
		say("items %d->%d heat now %d", mods, len(r.Order), r.heat())
		for pass := 0; pass < 4; pass++ { // split bricks spawn more as they break
			for _, b := range append([]*Brick(nil), L.Bricks...) {
				if b.Alive && b.Kind.Required() {
					L.destroy(b, Hit{Dmg: 99})
				}
			}
		}
		step(2)
		say("final wave cleared=%v finale=%.2f left=%d crystals=%d obj=%v", L.Cleared, L.finaleT, L.Left, L.Crystals, L.Data.Objective)
		// a new world resets the build
		r.Gold = 999
		r.NextAct(&g.save)
		say("world %d: items=%d gold=%d hp=%.0f/%.0f heat=%d actDepth=%d", r.Act, len(r.Order), r.Gold, r.HP, r.Stats.MaxHP, r.heat(), r.ActDepth)
		g.run = nil
		g.state = StMenu
		return strings.Join(log, "\n")
	}))
	// __saveTest round-trips a run through the save file format
	js.Global().Set("__saveTest", js.FuncOf(func(this js.Value, args []js.Value) any {
		g := theGame
		var log []string
		say := func(f string, a ...any) { log = append(log, fmt.Sprintf(f, a...)) }
		sv := defaultSave()
		g.save = sv
		r := NewRun("SAVEME", archByID("ranger"), &g.save, false)
		g.run = r
		g.hero = heroPortrait{}
		rng := rand.New(rand.NewSource(5))
		for i := 0; i < 6; i++ {
			for _, m := range r.Offer(1, i%3 == 0, Common, rng) {
				g.take(m)
			}
		}
		r.Gold, r.Depth, r.Lives, r.LifeFinds = 137, 3, 1, 1
		r.recordDmg("Lightning", 420)
		r.recordDmg("", 1200)
		r.Map.Nodes[0].Visited = true
		r.Cur = 0
		before := fmt.Sprintf("arch=%s act=%d depth=%d gold=%d hp=%.1f lives=%d/%d mods=%d dmg=%.2f crit=%.3f speed=%.2f cur=%d nodes=%d visited0=%v recap=%d/%d",
			r.Arch.ID, r.Act, r.Depth, r.Gold, r.HP, r.Lives, r.LifeFinds, len(r.Order), r.Stats.Damage, r.Stats.CritChance, r.Stats.SpeedMult, r.Cur, len(r.Map.Nodes), r.Map.Nodes[0].Visited, len(r.rc().Timeline), int(r.rc().Dmg["Orb"]))
		say("before %s", before)
		// map phase
		b := g.snapRun("map", -1)
		say("bytes=%d", len(b))
		s := parseSavedRun(b)
		g.continueRun(s)
		r2 := g.run
		after := fmt.Sprintf("arch=%s act=%d depth=%d gold=%d hp=%.1f lives=%d/%d mods=%d dmg=%.2f crit=%.3f speed=%.2f cur=%d nodes=%d visited0=%v recap=%d/%d",
			r2.Arch.ID, r2.Act, r2.Depth, r2.Gold, r2.HP, r2.Lives, r2.LifeFinds, len(r2.Order), r2.Stats.Damage, r2.Stats.CritChance, r2.Stats.SpeedMult, r2.Cur, len(r2.Map.Nodes), r2.Map.Nodes[0].Visited, len(r2.rc().Timeline), int(r2.rc().Dmg["Orb"]))
		say("after  %s", after)
		say("match=%v state=%v (map=%v)", before == after, g.state, g.state == StMap)
		// node phase: same chamber layout on resume
		var next int
		for _, n := range r2.Map.Nodes[0].Next {
			next = n
		}
		b = g.snapRun("node", next) // what enterNode writes first
		g.enterNode(next)
		bricks1, seed1 := len(g.level.Bricks), g.level.Data.Seed
		g.state = StMenu
		g.continueRun(parseSavedRun(b))
		say("node phase: state=%v level=%v bricks %d→%d seed same=%v", g.state, g.state == StLevel, bricks1, len(g.level.Bricks), seed1 == g.level.Data.Seed)
		// reward phase
		g.rewards = []rewardSet{{Title: "Choose an upgrade", Mods: g.run.Offer(3, false, Common, rng), Skip: 15}}
		ids := ""
		for _, m := range g.rewards[0].Mods {
			ids += m.ID + " "
		}
		b = g.snapRun("reward", -1)
		g.rewards = nil
		g.continueRun(parseSavedRun(b))
		ids2 := ""
		for _, m := range g.rewards[0].Mods {
			ids2 += m.ID + " "
		}
		say("reward phase: state=%v reward=%v offers same=%v (%s)", g.state, g.state == StReward, ids == ids2, ids2)
		say("garbage parse nil=%v", parseSavedRun([]byte("{bad")) == nil)
		g.run = nil
		g.state = StMenu
		return strings.Join(log, "\n")
	}))
	// __lifeTest exercises lives end to end without rendering
	js.Global().Set("__lifeTest", js.FuncOf(func(this js.Value, args []js.Value) any {
		g := theGame
		rec := capState.recording
		capState.recording = false
		defer func() { capState.recording = rec }()
		var log []string
		say := func(f string, a ...any) { log = append(log, fmt.Sprintf(f, a...)) }
		sv := defaultSave()
		r := NewRun("LIFE", archByID("berserker"), &sv, false)
		g.run, g.save = r, sv
		g.hero = heroPortrait{}
		L := g.newLevel(NCombat, 99)
		g.level, g.state = L, StLevel
		say("start lives=%d hp=%.0f", r.Lives, r.HP)
		step := func(n int, in Input) {
			for i := 0; i < n && g.state == StLevel; i++ {
				L.Update(1.0/60, in)
			}
		}
		// drop every orb
		L.release()
		for _, b := range L.Balls {
			b.X, b.Z, b.VX, b.VZ = 0, 1, 0, -10
		}
		step(30, Input{UseKeys: true})
		say("after drop lives=%d paddleGone=%.2f state=%v", r.Lives, L.paddleGone, g.state)
		in := Input{Fire: true, UseKeys: true}
		step(5, in)
		say("launch while reforming: stuck=%v", L.anyStuck())
		step(90, Input{UseKeys: true})
		// a heart lands on the paddle
		L.Drops = append(L.Drops, &Drop{X: L.PaddleX, Z: PaddleZ + 3, P: PLife})
		say("heart at x=%.1f z=%.1f paddle=%.1f w=%.1f balls=%d stuck=%v", L.Drops[0].X, L.Drops[0].Z, L.PaddleX, L.PaddleW, len(L.Balls), L.anyStuck())
		step(10, Input{UseKeys: true})
		if len(L.Drops) > 0 {
			say("10 frames: heart z=%.2f", L.Drops[0].Z)
		} else {
			say("10 frames: drops empty, lives=%d", r.Lives)
		}
		step(110, Input{UseKeys: true})
		say("after heart lives=%d finds=%d", r.Lives, r.LifeFinds)
		// health runs dry
		L.hurt(r.HP+5, "TEST")
		say("after hp loss lives=%d hp=%.0f", r.Lives, r.HP)
		// last life (once the paddle has reformed from the health loss)
		step(90, Input{UseKeys: true})
		L.release()
		for _, b := range L.Balls {
			b.X, b.Z, b.VX, b.VZ = 0, 1, 0, -10
		}
		step(30, Input{UseKeys: true})
		say("last drop lives=%d gameOverT=%.2f state=%v", r.Lives, L.gameOverT, g.state)
		step(200, Input{UseKeys: true})
		say("after blast state=%v (summary=%v)", g.state, g.state == StSummary)
		// act refill
		r2 := NewRun("LIFE2", archByID("guardian"), &sv, false)
		r2.Lives = 0
		r2.NextAct(&sv)
		say("act refill lives=%d", r2.Lives)
		g.state = StMenu
		return strings.Join(log, "\n")
	}))
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
