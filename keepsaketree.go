package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The Keepsakes menu: an upgrade tree bought with shards, opened from the
// main menu or the origin screen's Keepsake button. Every keepsake has a
// few small ranks; deeper keepsakes need two ranks of the one above them.
// One keepsake is equipped and carried into each run.

const (
	ktNodeW, ktNodeH = 300.0, 150.0
	ktX0, ktY0       = 110.0, 190.0
	ktColGap         = 70.0
	ktRowGap         = 52.0
)

func ktNodePos(k *keepsake) (float64, float64) {
	return ktX0 + float64(k.Col)*(ktNodeW+ktColGap), ktY0 + float64(k.Row)*(ktNodeH+ktRowGap)
}

// openKeepsakes shows the Keepsakes menu, returning to from on Back.
func (g *Game) openKeepsakes(from State) {
	g.keepFrom = from
	g.kt.popT, g.kt.linkT, g.kt.equipT, g.kt.shardsOK = 9, 9, 9, false
	if g.keepSel == "" {
		g.keepSel = keepsakes[0].ID
		if keepsakeByID[g.save.Keepsake] != nil {
			g.keepSel = g.save.Keepsake
		}
	}
	g.setState(StKeepsakes)
}

func (g *Game) drawKeepsakeTree(s *ebiten.Image) {
	u := g.ui
	S := u.S
	sv := &g.save
	if sv.KeepRanks == nil {
		sv.KeepRanks = map[string]int{}
	}
	g.ktTick()
	g.ktStars(s)
	g.treeComplete(s)
	u.heading(s, "KEEPSAKES", 64, 34, cGold, 1)
	if g.keepFrom == StMenu {
		g.sanctumTabs(s, 1) // opened as the Sanctum's second tab
	}
	g.ktShardFlash(s)
	u.iconLabel(s, icShard, fmt.Sprintf("%d shards", sv.Shards), fBold, 28, 960, 124, rgba(200, 180, 255, 255), alignCenter)

	// links first, under the nodes
	for i := range keepsakes {
		k := &keepsakes[i]
		if k.Parent == "" {
			continue
		}
		p := keepsakeByID[k.Parent]
		px, py := ktNodePos(p)
		cx, cy := ktNodePos(k)
		x0, y0 := px+ktNodeW/2, py+ktNodeH
		x1, y1 := cx+ktNodeW/2, cy
		col := rgba(70, 62, 92, 255)
		if sv.keepOpen(k) {
			col = ca(cGold, 0.8)
		}
		strokePath(s, linePath(u.X(x0), u.Y(y0), u.X(x1), u.Y(y1)), 3*S, col, false)
		if sv.keepOpen(k) {
			g.ktLinkFlow(s, x0, y0, x1, y1)
		}
		g.ktLinkPulse(s, k, x0, y0, x1, y1)
	}

	for i := range keepsakes {
		k := &keepsakes[i]
		x, y := ktNodePos(k)
		rank := sv.keepRank(k.ID)
		open := sv.keepOpen(k)
		hv, clk := u.hit("kt"+k.ID, x, y, ktNodeW, ktNodeH)
		sel := g.keepSel == k.ID
		if hv || sel {
			a := 0.14
			if sel {
				a = 0.26
			}
			drawSprite(s, sprSoft, u.X(x-24), u.Y(y-20), (ktNodeW+48)*S, (ktNodeH+40)*S, ca(cGold, a), true)
		}
		if open && rank == 0 {
			g.ktInvite(s, x, y)
		}
		g.ktNodePop(s, k, x, y)
		u.frame(s, x, y, ktNodeW, ktNodeH, 0.92)
		if rank >= k.Ranks {
			g.ktMaxedSheen(s, x, y, i)
		}
		if open && rank < k.Ranks && sv.Shards >= k.cost(rank) {
			g.ktAffordable(s, x, y)
		}
		if hv && !open {
			g.ktParentHint(s, k)
		}
		if hv && u.nav.mode {
			u.focusRing(s, x, y, ktNodeW, ktNodeH, 10)
		}
		col := cGold
		switch {
		case !open:
			col = rgba(90, 82, 110, 255)
		case rank == 0:
			col = rgba(180, 160, 220, 255)
		}
		if sv.Keepsake == k.ID && rank > 0 {
			u.glowLine(s, x+14, x+ktNodeW-14, y+ktNodeH-3, 3, cGold, 1)
		}
		u.drawEmblem(s, glyphImage(k.Glyph, col), x+52, y+ktNodeH/2-6, 60, 1)
		if !open {
			g.lockedCharm(s, x+76, y+ktNodeH/2+18)
		}
		nc := cWhite
		if !open {
			nc = cDim
		}
		u.text(s, k.Name, fBold, 20, x+98, y+24, nc, alignLeft)
		for p := 0; p < k.Ranks; p++ {
			c := rgba(60, 54, 80, 255)
			if p < rank {
				c = cGold
			}
			u.gem(s, x+108+float64(p)*24, y+78, 7, 9, c, 1)
		}
		g.ktGemFlash(s, k, x, y)
		if sv.Keepsake == k.ID && rank > 0 {
			g.keepsakeRunes(s, x+52, y+ktNodeH/2-6)
		}
		if clk {
			g.keepSel = k.ID
			g.snd.Play("click")
		}
	}
	g.drawKeepsakeDetail(s)
	g.drawUISparks(s)
	if u.button(s, "ktback", "Back", 860, 990, 200, 56, cAcc, true) {
		writeSave(sv)
		g.setState(g.keepFrom)
	}
}

// drawKeepsakeDetail is the right-hand panel: the selected keepsake's
// current and next rank, Upgrade and Equip.
func (g *Game) drawKeepsakeDetail(s *ebiten.Image) {
	u := g.ui
	sv := &g.save
	k := keepsakeByID[g.keepSel]
	if k == nil {
		return
	}
	x, y, w, h := 1250.0, 190.0, 590.0, 750.0
	u.panel(s, x, y, w, h, 0.94)
	rank := sv.keepRank(k.ID)
	open := sv.keepOpen(k)
	g.ktDetailGlow(s, x+w/2, y+90, rank >= k.Ranks)
	u.drawEmblem(s, glyphImage(k.Glyph, cGold), x+w/2, y+90, 110, 1)
	u.textOutlined(s, k.Name, fTitle, 34, x+w/2, y+160, cGold, alignCenter)
	u.rawText = true
	u.text(s, fmt.Sprintf(tr("Rank %d / %d"), rank, k.Ranks), fBold, 20, x+w/2, y+212, cDim, alignCenter)
	u.rawText = false

	ty := y + 262.0
	u.text(s, "Now", fBold, 18, x+36, ty, rgba(200, 180, 255, 255), alignLeft)
	now := tr("Not upgraded yet.")
	if rank > 0 {
		now = k.Desc(rank)
	}
	u.rawText = true
	ty = u.paragraph(s, now, fReg, 19, x+36, ty+30, w-72, cWhite, alignLeft) + 18
	u.rawText = false
	if rank < k.Ranks {
		if rank > 0 {
			g.ktNextArrow(s, x+w/2, ty-8)
		}
		u.text(s, "Next rank", fBold, 18, x+36, ty, rgba(200, 180, 255, 255), alignLeft)
		u.rawText = true
		ty = u.paragraph(s, k.Desc(rank+1), fReg, 19, x+36, ty+30, w-72, cGold, alignLeft) + 18
		u.rawText = false
	}
	if !open {
		p := keepsakeByID[k.Parent]
		u.rawText = true
		u.paragraph(s, fmt.Sprintf(tr("Needs %d ranks of %s."), keepsakeParentRanks, tr(p.Name)), fReg, 18, x+36, ty, w-72, rgba(255, 150, 140, 255), alignLeft)
		u.rawText = false
	}

	// upgrade
	by := y + h - 170
	label, can := tr("Max rank"), false
	if rank < k.Ranks {
		cost := k.cost(rank)
		label = fmt.Sprintf(tr("Upgrade  ·  %d shards"), cost)
		can = open && sv.Shards >= cost
	}
	u.rawText = true
	if u.button(s, "ktup", label, x+36, by, w-72, 60, cGold, can) {
		sv.Shards -= k.cost(rank)
		sv.KeepRanks[k.ID] = rank + 1
		if sv.Keepsake == "" {
			sv.Keepsake = k.ID // the first keepsake you buy is worn at once
		}
		writeSave(sv)
		g.snd.Play("buy")
		g.snd.Pick(min(4, rank+1))
		g.snd.PlayVol("ktrank."+itoa(rank+1), 0.6)
		kx, ky := ktNodePos(k)
		g.uiBurst(kx+52, ky+ktNodeH/2, 16, cGold, false)
		g.ktUpgraded(k, rank+1)
	}
	u.rawText = false
	// equip
	eq := sv.Keepsake == k.ID
	el := "Equip"
	if eq {
		el = "Unequip"
	}
	if u.button(s, "kteq", el, x+36, by+76, w-72, 56, cAcc, rank > 0) {
		if eq {
			sv.Keepsake = ""
		} else {
			sv.Keepsake = k.ID
			g.kt.equipT = 0
		}
		writeSave(sv)
		if eq {
			g.snd.PlayVol("ktuneq", 0.6)
		} else {
			g.snd.PlayVol("kteq", 0.7)
		}
	}
	if eq {
		p := 0.6 + 0.4*math.Sin(g.stateT*3)
		u.text(s, "Equipped", fBold, 18, x+w/2, by-34, ca(cGold, p), alignCenter)
	}
}
