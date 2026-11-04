package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"orb/levelgen"
)

// Save and quit: the run in progress is written to run.json next to the
// main save at every safe point (the map, the moment a node is entered,
// and the reward screen after a chamber). Continuing restores that exact
// moment; a chamber that was in progress starts over from its beginning,
// the same seeded layout as before, so quitting can't reroll anything.

type savedNode struct {
	Row, Lane int
	Type      NodeType
	Next      []int
	Visited   bool
	Seed      int64
	Omen      string
}

type savedReward struct {
	Title string
	Mods  []string
	Skip  int
}

type savedRun struct {
	Version   int
	Phase     string // "map", "node" or "reward"
	Node      int
	Rewards   []savedReward
	Endless   bool // g.endlessMode
	DailyMode bool

	SeedCode                                string
	SeedN                                   int64
	Daily                                   bool
	Arch                                    string
	Act, Cur, Depth                         int
	ActDepth                                int
	MapRows                                 int
	Map                                     []savedNode
	HP                                      float64 // unused since health was removed; kept so old saves load
	Gold                                    int
	Mods                                    map[string]int
	Order                                   []string
	PhoenixUsed                             bool
	Lives, LifeFinds                        int
	Challenge                               string
	Cursed                                  bool
	LastWorld                               int
	Element                                 string
	Rerolls                                 int
	Fused, Scrolls                          []string
	Rank, Asc                               int
	RunEndless, EndlessStart                bool
	XPPaid, ShardsPaid                      int
	StatsPaid                               [3]int
	MetaLuck                                int
	Bricks, Bosses, Elites, MaxCombo, Score int
	Elapsed                                 float64
	Assisted                                bool
	Recap                                   *runRecap
	Carry                                   bool    // v2: the build carries over between acts
	Power                                   float64 // v2: measured power (power.go)
	PowerN                                  int
	PowerI                                  float64
	PowerItems                              int
	PowerFF                                 float64
	NextBarrier                             int // barriers waiting for the next chamber
}

func runPath() string { return filepath.Join(filepath.Dir(savePath()), "run.json") }

// saveRun snapshots the run at a safe point.
func (g *Game) saveRun(phase string, node int) {
	if noPersist {
		return
	}
	b := g.snapRun(phase, node)
	if b == nil {
		return
	}
	p := runPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if os.WriteFile(p+".tmp", b, 0o644) == nil {
		_ = os.Rename(p+".tmp", p)
	}
}

// snapRun serialises the run (nil when there's nothing worth keeping).
func (g *Game) snapRun(phase string, node int) []byte {
	r := g.run
	if r == nil || r.Duel || r.SeedCode == "MENU" || r.SeedCode == "SIM" || r.SeedCode == "TRAILER" || r.SeedCode == "LIFE" || r.FinalScore != 0 && !r.Endless {
		return nil
	}
	s := savedRun{Version: runSaveVersion, Phase: phase, Node: node, Endless: g.endlessMode, DailyMode: g.daily,
		SeedCode: r.SeedCode, SeedN: r.SeedN, Daily: r.Daily, Arch: r.Arch.ID, Act: r.Act, Cur: r.Cur, Depth: r.Depth, ActDepth: r.ActDepth,
		MapRows: r.Map.Rows, Gold: r.Gold, Mods: r.Mods, Order: r.Order, PhoenixUsed: r.PhoenixUsed,
		Lives: r.Lives, LifeFinds: r.LifeFinds, Challenge: r.Challenge, Cursed: r.Cursed, LastWorld: r.LastWorld,
		Element: r.Element, Rerolls: r.Rerolls, Fused: r.Fused, Scrolls: r.Scrolls, Rank: r.Rank, Asc: r.Asc,
		RunEndless: r.Endless, EndlessStart: r.EndlessStart, XPPaid: r.xpPaid, ShardsPaid: r.shardsPaid,
		StatsPaid: r.statsPaid, MetaLuck: r.metaLuck, Bricks: r.Bricks, Bosses: r.Bosses, Elites: r.Elites,
		MaxCombo: r.MaxCombo, Score: r.Score, Elapsed: time.Since(r.Start).Seconds(), Recap: r.recap, Assisted: r.Assisted,
		Carry: r.Carry, Power: r.Power, PowerN: r.PowerN, PowerI: r.PowerI, PowerItems: r.PowerItems, PowerFF: r.PowerFF, NextBarrier: r.NextBarrier}
	for _, n := range r.Map.Nodes {
		s.Map = append(s.Map, savedNode{n.Row, n.Lane, n.Type, n.Next, n.Visited, n.Seed, n.Omen})
	}
	if phase == "reward" {
		for _, rs := range g.rewards {
			sr := savedReward{Title: rs.Title, Skip: rs.Skip}
			for _, m := range rs.Mods {
				sr.Mods = append(sr.Mods, m.ID)
			}
			s.Rewards = append(s.Rewards, sr)
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return b
}

// clearRun forgets the saved run (death, victory or abandoning).
func clearRun() {
	if !noPersist {
		_ = os.Remove(runPath())
	}
}

// loadSavedRun reads the saved run, if there is a usable one.
func loadSavedRun() *savedRun {
	b, err := os.ReadFile(runPath())
	if err != nil {
		return nil
	}
	return parseSavedRun(b)
}

func parseSavedRun(b []byte) *savedRun {
	var s savedRun
	if json.Unmarshal(b, &s) != nil || s.Version < 1 || s.Version > runSaveVersion || archByIDOK(s.Arch) == nil || len(s.Map) == 0 {
		return nil
	}
	if s.Version == 1 {
		// saved before builds carried over: keep the build from the next
		// act on, starting from the power its size suggests
		s.Carry, s.Power, s.PowerN = true, 0, 0
	}
	return &s
}

// runSaveVersion 2 adds the carried build and its measured power.
const runSaveVersion = 2

func archByIDOK(id string) *Archetype {
	for i := range archetypes {
		if archetypes[i].ID == id {
			return &archetypes[i]
		}
	}
	return nil
}

// continueRun restores a saved run and puts the player back where they
// left off.
func (g *Game) continueRun(s *savedRun) {
	arch := archByIDOK(s.Arch)
	r := &Run{SeedCode: s.SeedCode, SeedN: s.SeedN, Daily: s.Daily, Arch: arch, Act: s.Act, Cur: s.Cur, Depth: s.Depth, ActDepth: s.ActDepth,
		Gold: s.Gold, Mods: s.Mods, Order: s.Order, PhoenixUsed: s.PhoenixUsed, Lives: s.Lives, LifeFinds: s.LifeFinds,
		Challenge: s.Challenge, Cursed: s.Cursed, LastWorld: s.LastWorld, Element: s.Element, Rerolls: s.Rerolls,
		Fused: s.Fused, Scrolls: s.Scrolls, Rank: s.Rank, Asc: s.Asc, Endless: s.RunEndless, EndlessStart: s.EndlessStart,
		xpPaid: s.XPPaid, shardsPaid: s.ShardsPaid, statsPaid: s.StatsPaid, metaLuck: s.MetaLuck,
		Bricks: s.Bricks, Bosses: s.Bosses, Elites: s.Elites, MaxCombo: s.MaxCombo, Score: s.Score,
		Start: time.Now().Add(-time.Duration(s.Elapsed * float64(time.Second))), recap: s.Recap, Assisted: s.Assisted}
	if r.Mods == nil {
		r.Mods = map[string]int{}
	}
	if r.recap == nil {
		r.recap = newRecap()
	}
	r.Carry, r.Power, r.PowerN, r.PowerI, r.PowerItems = s.Carry, s.Power, s.PowerN, s.PowerI, s.PowerItems
	r.PowerFF = s.PowerFF
	r.NextBarrier = s.NextBarrier
	if r.Power <= 0 {
		r.Power = r.itemPower()
	}
	if s.PowerItems == 0 {
		r.PowerItems = r.buildSize() // no feed-forward jump on loading an older save
	}
	r.Rng = rand.New(rand.NewSource(levelgen.Mix(r.SeedN, int64(r.Depth), int64(r.Act), 31)))
	r.Map = &ActMap{Rows: s.MapRows}
	for _, n := range s.Map {
		r.Map.Nodes = append(r.Map.Nodes, &MapNode{Row: n.Row, Lane: n.Lane, Type: n.Type, Next: n.Next, Visited: n.Visited, Seed: n.Seed, Omen: n.Omen})
	}
	r.Recompute(&g.save)
	r.NewFusions = nil
	g.run = r
	g.hero = heroPortrait{}
	g.endlessMode, g.daily, g.seedText = s.Endless, s.DailyMode, ""
	g.toasts = nil
	switch s.Phase {
	case "node":
		if s.Node >= 0 && s.Node < len(r.Map.Nodes) {
			g.enterNode(s.Node)
			return
		}
	case "reward":
		g.rewards = nil
		for _, sr := range s.Rewards {
			rs := rewardSet{Title: sr.Title, Skip: sr.Skip}
			for _, id := range sr.Mods {
				if m := modByID[id]; m != nil {
					rs.Mods = append(rs.Mods, m)
				}
			}
			g.rewards = append(g.rewards, rs)
		}
		if len(g.rewards) > 0 {
			g.setState(StReward)
			return
		}
		g.finishRewards()
		return
	}
	g.setState(StMap)
}

// saveAndQuit leaves for the main menu with the run kept on disk. The
// snapshot taken at the last safe point is the one that's resumed.
func (g *Game) saveAndQuit() {
	if g.state == StMap {
		g.saveRun("map", -1)
	}
	g.run = nil
	g.setState(StMenu)
}
