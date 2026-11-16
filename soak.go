package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Memory soak (roadmap #104): `-soak` plays ten whole runs back to back with
// the benchmark autopilot, drawing one real frame of every chamber, reward
// screen, map and the main menu between runs, and records the live heap
// after each run. A leak shows as a heap that keeps climbing run after run;
// a healthy game levels off after the first run or two. The report goes to
// bench_soak[_TAG].txt beside the exe and the game then quits.

var soakOn bool

const (
	soakRuns     = 10
	soakChambers = 8  // per run; the last is the act's Guardian
	soakSimSecs  = 25 // simulated seconds per chamber at most
)

type soakSample struct {
	heapMB, sysMB float64
	objects       uint64
	goroutines    int
	gcs           uint32
	caches        string
}

type soakState struct {
	run, chamber, step int
	rng                *rand.Rand
	start              time.Time
	samples            []soakSample
	base               soakSample
}

var soak *soakState

func soakSnapshot(g *Game) soakSample {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return soakSample{
		heapMB:     float64(m.HeapAlloc) / (1 << 20),
		sysMB:      float64(m.Sys) / (1 << 20),
		objects:    m.HeapObjects,
		goroutines: runtime.NumGoroutine(),
		gcs:        m.NumGC,
		caches:     fmt.Sprintf("titles %d, sounds %d", len(g.titleCache), soundCacheLen(g.snd)),
	}
}

func soundCacheLen(s *Sounds) int {
	if s == nil {
		return 0
	}
	s.cmu.Lock()
	defer s.cmu.Unlock()
	return len(s.cache)
}

// soakUpdate drives the soak; it returns true when it has taken the frame.
// Each call does one step and leaves the frame to draw, so every screen the
// soak visits is drawn at least once.
func soakUpdate(g *Game) bool {
	if !soakOn {
		return false
	}
	if soak == nil {
		soak = &soakState{rng: rand.New(rand.NewSource(1104)), start: time.Now()}
		g.settings.NoTips = true
		g.snd.SFX, g.snd.Music = 0, 0
		g.snd.updateVolumes()
		g.save.Hints = map[string]bool{"level": true, "waves": true, "critter": true, "map": true, "reward": true, "cache": true, "shop": true, "boss": true}
		soak.base = soakSnapshot(g)
	}
	s := soak
	g.coachQ = nil
	switch s.step {
	case 0: // a fresh run
		if s.run >= soakRuns {
			soakFinish(g)
			return true
		}
		g.run = NewRun("BENCH", &archetypes[s.run%len(archetypes)], &g.save, false)
		s.chamber = 0
		g.setState(StMap)
		s.step = 1
	case 1: // play a chamber to the end (or the time limit)
		node := NCombat
		switch {
		case s.chamber == soakChambers-1:
			node = NBoss
		case s.chamber == 4:
			node = NElite
		}
		g.run.Lives = 3
		g.level = g.newLevel(node, s.rng.Int63())
		g.level.introT = 0
		g.setState(StLevel)
		L := g.level
		const dt = 1.0 / 60
		for t := 0.0; t < soakSimSecs && !L.Cleared; t += dt {
			in := benchPilot(g)
			L.Update(dt, in)
			for L.pendingPick > 0 {
				L.pendingPick--
				if ms := g.run.Offer(1, false, Common, s.rng); len(ms) > 0 {
					g.run.Add(ms[0].ID, &g.save)
				}
			}
		}
		g.run.Depth++
		s.step = 2
	case 2: // the reward screen, then take one of its cards
		ms := g.run.Offer(3, false, Common, s.rng)
		g.rewards = []rewardSet{{Title: "Choose an upgrade", Mods: ms, Skip: 15}}
		g.setState(StReward)
		if len(ms) > 0 {
			g.run.Add(ms[s.rng.Intn(len(ms))].ID, &g.save)
		}
		g.run.Gold += 40
		s.step = 3
	case 3: // back to the map; next chamber or the end of the run
		g.rewards = nil
		g.setState(StMap)
		s.chamber++
		if s.chamber < soakChambers {
			s.step = 1
			if s.chamber == 3 { // the shop once per run
				for i, n := range g.run.Map.Nodes {
					if n.Type == NShop {
						g.enterNode(i)
						break
					}
				}
			}
		} else {
			s.step = 4
		}
	case 4: // the run ends: summary-free return to the main menu
		g.level, g.run = nil, nil
		g.setState(StMenu)
		s.step = 5
	case 5: // after a menu frame has drawn, measure
		s.samples = append(s.samples, soakSnapshot(g))
		s.run++
		s.step = 0
	}
	return false
}

func soakFinish(g *Game) {
	s := soak
	var sb strings.Builder
	fmt.Fprintf(&sb, "Arcane Orb memory soak  %s  %s\n", gameVersion, time.Now().Format("2006-01-02 15:04"))
	fmt.Fprintf(&sb, "%d runs x %d chambers (autopilot, up to %d simulated s each), one drawn frame per screen; %s total\n\n",
		soakRuns, soakChambers, soakSimSecs, time.Since(s.start).Round(time.Second))
	fmt.Fprintf(&sb, "%-8s %9s %9s %10s %6s %5s  %s\n", "after", "heap MB", "sys MB", "objects", "gorout", "GCs", "caches")
	row := func(name string, x soakSample) {
		fmt.Fprintf(&sb, "%-8s %9.1f %9.1f %10d %6d %5d  %s\n", name, x.heapMB, x.sysMB, x.objects, x.goroutines, x.gcs, x.caches)
	}
	row("start", s.base)
	for i, x := range s.samples {
		row(fmt.Sprintf("run %d", i+1), x)
	}
	if n := len(s.samples); n >= 3 {
		// growth over the later runs (the first runs fill caches on purpose)
		d := (s.samples[n-1].heapMB - s.samples[1].heapMB) / float64(n-2)
		verdict := "steady"
		if d > 1 {
			verdict = "GROWING - look for a leak"
		}
		fmt.Fprintf(&sb, "\nheap change per run after run 2: %+.2f MB (%s)\n", d, verdict)
	}
	exe, _ := os.Executable()
	name := "bench_soak"
	if benchTag != "" {
		name += "_" + benchTag
	}
	_ = os.WriteFile(filepath.Join(filepath.Dir(exe), name+".txt"), []byte(sb.String()), 0o644)
	g.quit = true
}
