package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Stat changes on reward cards (ROADMAP-50 #33). Under each reward card the
// two biggest number changes that taking it would make are spelled out,
// "Orb damage 1.5 → 2.0", worked out by applying the card to a copy of the
// build (so synergies, fusions, the origin and the Sanctum all count). A
// card that changes behaviour rather than numbers shows nothing extra; its
// description already says what it does. A synergy the card would switch
// on is named too.

type statFmt int

const (
	sfNum   statFmt = iota // 1.5
	sfInt                  // 3
	sfPct                  // 14%   (a chance stored as 0.14)
	sfBase                 // 110%  (a multiplier around 1)
	sfBonus                // +30%  (a bonus stored as 0.3)
	sfTimes                // ×2.5
)

type statRow struct {
	Label string
	Get   func(s *Stats) float64
	Fmt   statFmt
}

// statRows lists the numbers worth showing, most important first. Every
// one of them is better when it goes up.
var statRows = []statRow{
	{"Orb damage", func(s *Stats) float64 { return s.Damage * s.DmgMult }, sfNum},
	{"Critical chance", func(s *Stats) float64 { return s.CritChance }, sfPct},
	{"Critical damage", func(s *Stats) float64 { return s.CritMult }, sfTimes},
	{"Extra orbs", func(s *Stats) float64 { return float64(s.ExtraBalls) }, sfInt},
	{"Pierce", func(s *Stats) float64 { return float64(s.Pierce) }, sfInt},
	{"Paddle width", func(s *Stats) float64 { return s.PaddleWidth }, sfBase},
	{"Orb speed", func(s *Stats) float64 { return s.SpeedMult }, sfBase},
	{"Orb size", func(s *Stats) float64 { return s.SizeMult }, sfBase},
	{"Burn chance", func(s *Stats) float64 { return s.BurnChance }, sfPct},
	{"Freeze chance", func(s *Stats) float64 { return s.FreezeChance }, sfPct},
	{"Lightning chance", func(s *Stats) float64 { return s.ChainChance }, sfPct},
	{"Explosion chance", func(s *Stats) float64 { return s.ExplodeChance }, sfPct},
	{"Void chance", func(s *Stats) float64 { return s.VoidChance }, sfPct},
	{"Split chance", func(s *Stats) float64 { return s.SplitChance }, sfPct},
	{"Lightning targets", func(s *Stats) float64 { return float64(s.ChainCount) }, sfInt},
	{"Explosion size", func(s *Stats) float64 { return s.ExplodeRadius }, sfNum},
	{"Elite & boss damage", func(s *Stats) float64 { return s.BossDmg }, sfBonus},
	{"Combo damage per hit", func(s *Stats) float64 { return s.ComboDmg }, sfBonus},
	{"Laser damage", func(s *Stats) float64 { return s.LaserDmg }, sfBonus},
	{"Barriers per chamber", func(s *Stats) float64 { return float64(s.BarrierPerLevel) }, sfInt},
	{"Extra lives", func(s *Stats) float64 { return float64(s.ExtraLives) }, sfInt},
	{"Gold gain", func(s *Stats) float64 { return s.GoldMult }, sfBase},
	{"Power-up drops", func(s *Stats) float64 { return s.Luck }, sfBase},
	{"Power-up time", func(s *Stats) float64 { return s.PowerDur }, sfBonus},
	{"Shop discount", func(s *Stats) float64 { return s.ShopDiscount }, sfBonus},
	{"Reward choices", func(s *Stats) float64 { return float64(3 + s.ExtraChoices) }, sfInt},
	{"Free rerolls", func(s *Stats) float64 { return float64(s.FreeRerolls) }, sfInt},
}

func fmtStat(v float64, f statFmt) string {
	trim := func(x float64, prec int) string {
		s := strconv.FormatFloat(x, 'f', prec, 64)
		if strings.Contains(s, ".") {
			s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		}
		return s
	}
	switch f {
	case sfInt:
		return itoa(int(math.Round(v)))
	case sfPct, sfBase:
		return trim(v*100, 0) + "%"
	case sfBonus:
		if v < 0 {
			return trim(v*100, 0) + "%"
		}
		return "+" + trim(v*100, 0) + "%"
	case sfTimes:
		return "×" + trim(v, 2)
	}
	return trim(v, 2)
}

// statChange is one line under a card.
type statChange struct {
	Label, From, To string
	Up              bool
	Synergy         bool // "Activates …" instead of a number
}

// previewAdd returns the stats the build would have with one more of the
// mod, and the synergies that would switch on, without touching the run.
func (r *Run) previewAdd(id string, meta *SaveData) (Stats, []string) {
	c := *r
	c.Mods = make(map[string]int, len(r.Mods)+1)
	for k, v := range r.Mods {
		c.Mods[k] = v
	}
	// Recompute writes into these slices: give the copy its own
	c.Order = append([]string(nil), r.Order...)
	c.Active = append([]string(nil), r.Active...)
	c.Fused = append([]string(nil), r.Fused...)
	c.NewFusions = nil
	c.Add(id, meta)
	had := map[string]bool{}
	for _, a := range r.Active {
		had[a] = true
	}
	var syn []string
	for _, a := range c.Active {
		if !had[a] {
			syn = append(syn, a)
		}
	}
	return c.Stats, syn
}

// statChanges lists what taking m would change, biggest first (by the
// order of statRows), at most n number lines plus any synergy.
func (r *Run) statChanges(m *Mod, meta *SaveData, n int) []statChange {
	if r == nil || m == nil || r.Arch == nil || !r.canTake(m) {
		return nil
	}
	after, syn := r.previewAdd(m.ID, meta)
	var out []statChange
	for _, row := range statRows {
		a, b := row.Get(&r.Stats), row.Get(&after)
		if math.Abs(b-a) < 1e-6 {
			continue
		}
		from, to := fmtStat(a, row.Fmt), fmtStat(b, row.Fmt)
		if from == to {
			continue // too small to show
		}
		out = append(out, statChange{Label: row.Label, From: from, To: to, Up: b > a})
		if len(out) >= n {
			break
		}
	}
	for _, s := range syn {
		out = append(out, statChange{Label: s, Synergy: true, Up: true})
	}
	return out
}

// statChangeCache keeps the preview while the build doesn't change: the
// card is drawn every frame.
type statChangeCache struct {
	key  string
	rows []statChange
}

func (g *Game) cardStatChanges(m *Mod) []statChange {
	r := g.run
	if r == nil {
		return nil
	}
	stacks := 0
	for _, v := range r.Mods {
		stacks += v
	}
	key := m.ID + "|" + itoa(len(r.Order)) + "|" + itoa(stacks) + "|" + itoa(r.Act)
	if g.statCache == nil {
		g.statCache = map[string]*statChangeCache{}
	}
	if c := g.statCache[m.ID]; c != nil && c.key == key {
		return c.rows
	}
	rows := r.statChanges(m, &g.save, 2)
	g.statCache[m.ID] = &statChangeCache{key: key, rows: rows}
	return rows
}

// drawStatChanges writes the lines centred under a card at y.
func (g *Game) drawStatChanges(s *ebiten.Image, m *Mod, cx, y float64) {
	u := g.ui
	rows := g.cardStatChanges(m)
	up, down := rgba(140, 240, 150, 255), rgba(255, 130, 120, 255)
	for _, c := range rows {
		if c.Synergy {
			u.text(s, fmt.Sprintf("Activates %s", c.Label), fBold, 18, cx, y, cGold, alignCenter)
			y += 24
			continue
		}
		col := up
		if !c.Up {
			col = down
		}
		vals := c.From + "  →  " + c.To
		lw := u.measure(c.Label, fReg, 18)
		vw := u.measure(vals, fBold, 18)
		x := cx - (lw+12+vw)/2
		u.text(s, c.Label, fReg, 18, x, y, cDim, alignLeft)
		u.text(s, vals, fBold, 18, x+lw+12, y, col, alignLeft)
		y += 24
	}
}

// kitStats is what the build would be with none of its items: the origin,
// the Sanctum, mastery, the challenge and the endless loop only. It is the
// "base" the stats panel compares against.
func (r *Run) kitStats(meta *SaveData) Stats {
	c := *r
	c.Mods = map[string]int{}
	c.Order, c.Active, c.Fused, c.NewFusions = nil, nil, nil, nil
	c.Recompute(meta)
	return c.Stats
}
