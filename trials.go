package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Trials: custom difficulty. Each trial is a rule worth 1 or 2 points that
// stacks with Ascension; every point adds 6% to the shards a run earns.
// Chosen on the origin screen; normal and endless runs only (daily, weekly
// and challenge runs keep their own rules).

type trial struct {
	ID, Name, Desc, Glyph string
	Pts                   int
	Apply                 func(s *Stats)
}

var trials = []trial{
	{ID: "hp", Name: "Tough Bricks", Glyph: "shield", Pts: 1, Desc: "Bricks have 15% more health.",
		Apply: func(s *Stats) { s.BrickHPMult += 0.15 }},
	{ID: "swift", Name: "Swift Orbs", Glyph: "bolt", Pts: 1, Desc: "Orbs fly 10% faster.",
		Apply: func(s *Stats) { s.SpeedMult += 0.10 }},
	{ID: "narrow", Name: "Narrow Paddle", Glyph: "feather", Pts: 1, Desc: "Your paddle is 10% narrower.",
		Apply: func(s *Stats) { s.PaddleWidth -= 0.10 }},
	{ID: "purse", Name: "Lean Purse", Glyph: "coin", Pts: 1, Desc: "20% less gold from every source.",
		Apply: func(s *Stats) { s.GoldMult -= 0.20 }},
	{ID: "drought", Name: "Dry Spell", Glyph: "drop", Pts: 1, Desc: "Power-ups drop 25% less often.",
		Apply: func(s *Stats) { s.Luck -= 0.25 }},
	{ID: "choices", Name: "Fewer Choices", Glyph: "book", Pts: 2, Desc: "Every reward offers one card fewer.",
		Apply: func(s *Stats) { s.ExtraChoices-- }},
	{ID: "fragile", Name: "Fragile", Glyph: "heart", Pts: 2, Desc: "One fewer life.",
		Apply: func(s *Stats) { s.ExtraLives-- }},
	{ID: "guardians", Name: "Relentless Guardians", Glyph: "mask", Pts: 2, Desc: "Guardians have 30% more health.",
		Apply: func(s *Stats) { s.BossHPMult += 0.30 }},
}

var trialByID = func() map[string]*trial {
	m := map[string]*trial{}
	for i := range trials {
		m[trials[i].ID] = &trials[i]
	}
	return m
}()

const trialShardPct = 6 // shards per point, in percent

func (r *Run) applyTrials(s *Stats) {
	for _, id := range r.Trials {
		if t := trialByID[id]; t != nil {
			t.Apply(s)
		}
	}
}

// trialPoints is a run's (or a selection's) total.
func trialPoints(ids []string) int {
	n := 0
	for _, id := range ids {
		if t := trialByID[id]; t != nil {
			n += t.Pts
		}
	}
	return n
}

// trialShardMult is the shard bonus for a run's trials.
func (r *Run) trialShardMult() float64 {
	return 1 + float64(trialPoints(r.Trials)*trialShardPct)/100
}

// selectedTrials is the save's chosen trials, in table order.
func (g *Game) selectedTrials() []string {
	var out []string
	for _, t := range trials {
		if g.save.Trials[t.ID] {
			out = append(out, t.ID)
		}
	}
	return out
}

func (g *Game) trialsLabel() string {
	n := trialPoints(g.selectedTrials())
	if n == 0 {
		return tr("Trials: off")
	}
	return fmt.Sprintf(tr("Trials: %d"), n)
}

// drawTrialsPanel is the trials overlay on the origin screen.
func (g *Game) drawTrialsPanel(s *ebiten.Image) {
	u := g.ui
	fillRect(s, 0, 0, u.SW, u.SH, rgba(4, 2, 10, 170))
	x, y, w, h := 410.0, 130.0, 1100.0, 820.0
	g.trialsHeatBack(s, x, y, w, h)
	u.panel(s, x, y, w, h, 0.96)
	u.textOutlined(s, "Trials", fTitle, 40, 960, y+26, rgba(255, 140, 120, 255), alignCenter)
	u.text(s, "Turn on any rules you like. Each point adds 6% shards. They stack with Ascension.", fReg, 18, 960, y+84, cDim, alignCenter)
	if g.save.Trials == nil {
		g.save.Trials = map[string]bool{}
	}
	for i, t := range trials {
		ty := y + 128 + float64(i)*72
		on := g.save.Trials[t.ID]
		u.frame(s, x+40, ty, w-80, 64, 0.9)
		if on {
			g.trialRowGlow(s, x+40, ty, w-80, 64, i)
		}
		col := rgba(150, 140, 170, 255)
		if on {
			col = rgba(255, 140, 120, 255)
		}
		u.drawEmblem(s, glyphImage(t.Glyph, col), x+80, ty+32, 36, 1)
		nc := cDim
		if on {
			nc = cWhite
		}
		u.text(s, t.Name, fBold, 20, x+112, ty+8, nc, alignLeft)
		u.text(s, t.Desc, fReg, 15, x+112, ty+36, cDim, alignLeft)
		u.rawText = true
		u.text(s, "+"+itoa(t.Pts), fBold, 22, x+w-150, ty+18, col, alignRight)
		u.rawText = false
		if u.switchCtl(s, "trial"+t.ID, x+w-130, ty+12, on) {
			g.save.Trials[t.ID] = !on
			if !on {
				g.uiBurst(x+w-100, ty+32, 10, cTrial, true)
				g.snd.PlayVol("trialon", 0.55)
			} else {
				g.snd.PlayVol("trialoff", 0.5)
			}
		}
	}
	n := trialPoints(g.selectedTrials())
	u.rawText = true
	u.text(s, fmt.Sprintf(tr("Trial %d  ·  +%d%% shards"), n, n*trialShardPct), fBold, 22, 960, y+h-118, trialHeat(n), alignCenter)
	u.rawText = false
	if u.button(s, "trialsoff", "Clear all", 960-220, y+h-74, 200, 56, cAcc, n > 0) {
		g.save.Trials = map[string]bool{}
		g.uiBurst(960-120, y+h-46, 16, rgba(150, 140, 170, 255), true)
	}
	g.drawUISparks(s)
	if u.button(s, "trialsdone", "Done", 960+20, y+h-74, 200, 56, cGreen, true) || backPressed() {
		g.archPanel = 0
		writeSave(&g.save)
		g.consumeInput()
	}
}

// trialNames lists a run's trials for the recap.
func trialNames(ids []string) string {
	var n []string
	for _, id := range ids {
		if t := trialByID[id]; t != nil {
			n = append(n, tr(t.Name))
		}
	}
	return strings.Join(n, listSep)
}
