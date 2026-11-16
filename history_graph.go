package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The run history graph (Codex > Run History): the last 20 runs, oldest on
// the left. Each bar is how far the run got (a gold bar is a win); the line
// is the damage it dealt.

func runDamage(r *Run) float64 {
	_, total := dmgRows(r)
	return total
}

func (g *Game) drawHistoryGraph(s *ebiten.Image, x, y, w, h float64) {
	u := g.ui
	hist := g.save.History
	n := min(20, len(hist))
	if n < 2 {
		return
	}
	// oldest first
	runs := make([]runHistory, n)
	for i := 0; i < n; i++ {
		runs[i] = hist[n-1-i]
	}
	u.text(s, "Last runs: how far each got (bars) and the damage it dealt (line)", fBold, 16, x, y, cDim, alignLeft)
	gy, gh := y+28, h-50
	fillPath(s, boxPath(u.X(x), u.Y(gy), w*u.S, gh*u.S, 8*u.S, false), rgba(0, 0, 0, 90), rgba(0, 0, 0, 60), vgrad(u.Y(gy), u.Y(gy+gh)), false)
	maxAct := float64(numActs + 1)
	maxDmg := 1.0
	for _, r := range runs {
		maxDmg = math.Max(maxDmg, float64(r.Dmg))
		maxAct = math.Max(maxAct, float64(r.Act))
	}
	step := w / float64(n)
	bw := math.Min(42, step*0.62)
	var px, py float64
	for i, r := range runs {
		cx := x + step*(float64(i)+0.5)
		reach := float64(r.Act)
		col := ca(classInfo[r.Arch].Col, 0.75)
		if r.Won {
			reach, col = maxAct, ca(cGold, 0.9)
		}
		bh := gh * 0.9 * reach / maxAct
		fillPath(s, boxPath(u.X(cx-bw/2), u.Y(gy+gh-bh), bw*u.S, bh*u.S, 5*u.S, false), lighten(col, 0.15), darken(col, 0.4), vgrad(u.Y(gy+gh-bh), u.Y(gy+gh)), false)
		if r.Dmg > 0 {
			dy := gy + gh - gh*0.9*float64(r.Dmg)/maxDmg
			if i > 0 && py > 0 {
				strokeLine(s, u.X(px), u.Y(py), u.X(cx), u.Y(dy), 3*u.S, prem(rgba(130, 220, 255, 255), 0.9))
			}
			fillPath(s, circlePath(u.X(cx), u.Y(dy), 5*u.S), rgba(170, 235, 255, 255), rgba(90, 170, 220, 255), grad{}, false)
			px, py = cx, dy
		}
		if u.hover(cx-step/2, gy, step, gh) {
			res := fmt.Sprintf(tr("Fell in Act %d"), r.Act)
			if r.Won {
				res = tr("Victory")
			}
			g.tooltip = histTip(r.Date, res, fmtBig(float64(r.Dmg)), tr("damage"))
		}
	}
	u.text(s, fmt.Sprintf(tr("Best damage: %s"), fmtBig(maxDmg)), fReg, 14, x+w, y+2, rgba(150, 220, 255, 255), alignRight)
}

// fmtBig shortens big numbers: 950, 12.3k, 4.1M.
