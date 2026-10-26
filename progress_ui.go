package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Screens for long-term progression: Sanctum, Codex, rank badge,
// ascension selector and the end-of-run summary with XP.

// ---------------------------------------------------------------- rank badge

func (g *Game) drawRankBadge(s *ebiten.Image, x, y float64) {
	u := g.ui
	rank := rankOf(g.save.XP)
	lo, hi := xpForRank(rank), xpForRank(rank+1)
	frac := float64(g.save.XP-lo) / float64(max(1, hi-lo))
	u.frame(s, x, y, 430, 112, 0.92)
	u.medallion(s, x+58, y+56, 34, rgba(150, 110, 255, 255))
	w := u.measure(itoa(rank), fTitle, 34)
	u.textOutlined(s, itoa(rank), fTitle, 34, x+58-w/2+w/2, y+34, cWhite, alignCenter)
	u.text(s, "ARCANE RANK", fBold, 15, x+112, y+20, ca(u.theme().Accent, 1), alignLeft)
	u.text(s, fmt.Sprintf("%d / %d XP", g.save.XP-lo, hi-lo), fReg, 15, x+404, y+20, cDim, alignRight)
	u.bar(s, x+112, y+46, 292, 16, frac, frac, rgba(150, 110, 255, 255), 0, 0)
	next := unlocksAt(rank + 1)
	label := "Next rank: more items join the pool"
	if len(next) > 0 {
		label = "Next rank unlocks: " + next[0].Name
		if len(next) > 1 {
			label += fmt.Sprintf(" +%d more", len(next)-1)
		}
	}
	u.text(s, label, fReg, 15, x+112, y+72, cDim, alignLeft)
}

// ---------------------------------------------------------------- sanctum

func (g *Game) drawSanctum(s *ebiten.Image) {
	u := g.ui
	u.heading(s, "THE SANCTUM", 72, 40, cGold, 1)
	u.iconLabel(s, icShard, fmt.Sprintf("%d shards", g.save.Shards), fBold, 28, 960, 142, rgba(200, 180, 255, 255), alignCenter)
	cols, w, h, gap := 5, 330.0, 236.0, 22.0
	x0 := 960 - (float64(cols)*w+float64(cols-1)*gap)/2
	for i, m := range metaUpgrades {
		x := x0 + float64(i%cols)*(w+gap)
		y := 200 + float64(i/cols)*(h+gap)
		rank := g.save.Upg[m.ID]
		maxed := rank >= m.Max
		k := u.anim("meta"+m.ID, u.hover(x, y, w, h), 12)
		if k > 0.01 {
			drawSprite(s, sprSoft, u.X(x-40), u.Y(y-40), (w+80)*u.S, (h+80)*u.S, ca(u.theme().Accent, 0.22*k), true)
		}
		u.frame(s, x, y, w, h, 0.92)
		col := rgba(180, 150, 255, 255)
		if maxed {
			col = cGold
		}
		u.drawEmblem(s, glyphImage(m.Icon, col), x+62, y+70, 92, 1)
		u.text(s, m.Name, fBold, 23, x+118, y+26, cWhite, alignLeft)
		for p := 0; p < m.Max; p++ {
			px := x + 126 + float64(p)*26
			c := rgba(60, 54, 80, 255)
			if p < rank {
				c = cGold
			}
			u.gem(s, px, y+76, 8, 10, c, 1)
		}
		u.paragraph(s, m.Desc, fReg, 18, x+24, y+122, w-48, cDim, alignLeft)
		label, can := "Maxed", false
		if !maxed {
			cost := metaCost(m, rank)
			label = fmt.Sprintf("%d shards", cost)
			can = g.save.Shards >= cost
		}
		if u.button(s, "metab"+m.ID, label, x+24, y+h-62, w-48, 46, cGold, can) {
			g.save.Shards -= metaCost(m, rank)
			g.save.Upg[m.ID]++
			writeSave(&g.save)
			g.snd.Play("buy")
			g.snd.Pick(min(4, rank+1))
		}
	}
	if u.button(s, "sback", "Back", 860, 990, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

// ---------------------------------------------------------------- codex

func (g *Game) drawCodex(s *ebiten.Image) {
	u := g.ui
	u.heading(s, "THE CODEX", 72, 40, cGold, 1)
	tabs := []string{"Upgrades", "Relics", "Fusions", "Endless Records"}
	for i, t := range tabs {
		acc := cAcc
		if g.codexTab == i {
			acc = cGold
		}
		tx := 960 - 470 + float64(i)*240
		if u.button(s, "ctab"+itoa(i), t, tx, 140, 220, 50, acc, true) {
			g.codexTab = i
		}
		if g.codexTab == i {
			u.glowLine(s, tx+20, tx+200, 198, 3, cGold, 1)
		}
	}
	if g.codexTab == 3 {
		g.drawRecords(s, 236)
		if u.button(s, "cback", "Back", 60, 990, 200, 56, cAcc, true) {
			g.setState(StMenu)
		}
		return
	}
	rank := rankOf(g.save.XP)
	tile, gap, cols := 108.0, 14.0, 12
	x0 := 960 - (float64(cols)*tile+float64(cols-1)*gap)/2
	y0 := 236.0
	if g.codexTab < 2 {
		relic := g.codexTab == 1
		var list []*Mod
		for _, m := range modList {
			if m.Relic == relic {
				list = append(list, m)
			}
		}
		found := 0
		for i, m := range list {
			x := x0 + float64(i%cols)*(tile+gap)
			y := y0 + float64(i/cols)*(tile+gap)
			seen := g.save.Seen[m.ID]
			locked := m.Unlock > rank
			if seen {
				found++
			}
			hv := u.hover(x, y, tile, tile)
			rc := rarityColors[m.Rarity]
			bg := boxPath(u.X(x), u.Y(y), tile*u.S, tile*u.S, 14*u.S, false)
			fillPath(s, bg, rgba(26, 20, 44, 230), rgba(12, 9, 24, 240), vgrad(u.Y(y), u.Y(y+tile)), false)
			edge := ca(rc, 0.85)
			if !seen {
				edge = rgba(70, 64, 92, 255)
			}
			if hv {
				edge = lighten(edge, 0.4)
			}
			strokePath(s, bg, 2*u.S, edge, false)
			switch {
			case seen:
				u.drawEmblem(s, modEmblem(m), x+tile/2, y+tile/2, 86, 1)
			case locked:
				u.drawEmblemDark(s, modEmblem(m), x+tile/2, y+tile/2, 80, 0.9)
				u.text(s, "Rank "+itoa(m.Unlock), fBold, 15, x+tile/2, y+tile-26, cDim, alignCenter)
			default:
				u.drawEmblemDark(s, modEmblem(m), x+tile/2, y+tile/2, 80, 1)
				u.textOutlined(s, "?", fTitle, 40, x+tile/2, y+tile/2-26, rgba(120, 112, 150, 255), alignCenter)
			}
			if hv {
				switch {
				case seen:
					g.tooltip = m.Name + " (" + rarityNames[m.Rarity] + ") — " + m.Desc
					if ps := fusionPartners(m.ID); len(ps) > 0 {
						var names []string
						for _, p := range ps {
							names = append(names, modByID[p].Name)
						}
						g.tooltip += "  Fuses with: " + strings.Join(names, ", ") + "."
					}
				case locked:
					g.tooltip = fmt.Sprintf("Reach Arcane Rank %d to unlock this %s.", m.Unlock, map[bool]string{true: "relic", false: "upgrade"}[relic])
				default:
					g.tooltip = "Undiscovered. Take it in a run to record it here."
				}
			}
		}
		u.text(s, fmt.Sprintf("Discovered %d / %d", found, len(list)), fBold, 22, 960, 964, cDim, alignCenter)
	} else {
		w, h := 560.0, 112.0
		found := 0
		for i := range fusions {
			f := &fusions[i]
			x := 960 - (3*w+2*24)/2 + float64(i%3)*(w+24)
			y := 220 + float64(i/3)*(h+10)
			known := g.save.Fusions[f.ID]
			if known {
				found++
			}
			a, b := modByID[f.A], modByID[f.B]
			u.frame(s, x, y, w, h, 0.9)
			showA, showB := g.save.Seen[f.A], g.save.Seen[f.B]
			emb := func(m *Mod, show bool, ex float64) {
				if show {
					u.drawEmblem(s, modEmblem(m), ex, y+h/2, 64, 1)
				} else {
					u.drawEmblemDark(s, modEmblem(m), ex, y+h/2, 60, 1)
					u.textOutlined(s, "?", fTitle, 30, ex, y+h/2-20, rgba(120, 112, 150, 255), alignCenter)
				}
			}
			emb(a, showA, x+52)
			u.text(s, "+", fBold, 30, x+100, y+h/2-20, cDim, alignCenter)
			emb(b, showB, x+148)
			u.text(s, "=", fBold, 30, x+196, y+h/2-20, cDim, alignCenter)
			if known {
				drawSprite(s, sprSoft, u.X(x+252-60), u.Y(y+h/2-60), 120*u.S, 120*u.S, ca(f.Col, 0.45), true)
				u.drawEmblem(s, fusionEmblem(f), x+252, y+h/2, 82, 1)
				u.text(s, f.Name, fBold, 21, x+302, y+12, f.Col, alignLeft)
				u.paragraph(s, f.Desc, fReg, 15, x+302, y+40, w-320, cDim, alignLeft)
			} else {
				u.textOutlined(s, "?", fTitle, 44, x+252, y+h/2-30, rgba(120, 112, 150, 255), alignCenter)
				hint := "Own both items in one run to forge it."
				if showA && showB {
					hint = a.Name + " + " + b.Name
				}
				u.text(s, "Unknown fusion", fBold, 21, x+302, y+16, rgba(150, 142, 180, 255), alignLeft)
				u.paragraph(s, hint, fReg, 16, x+302, y+48, w-320, cDim, alignLeft)
			}
		}
		u.text(s, fmt.Sprintf("Fusions forged %d / %d", found, len(fusions)), fBold, 22, 960, 972, cDim, alignCenter)
	}
	if u.button(s, "cback", "Back", 60, 990, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

// ---------------------------------------------------------------- ascension selector

func (g *Game) drawAscSelector(s *ebiten.Image, y float64) {
	u := g.ui
	w := 820.0
	x := 960 - w/2
	u.frame(s, x, y, w, 96, 0.92)
	if g.save.AscMax == 0 {
		u.text(s, "ASCENSION", fBold, 16, 960, y+16, ca(u.theme().Accent, 1), alignCenter)
		u.text(s, "Win a run to unlock Ascension: harder runs with bigger rewards.", fReg, 21, 960, y+46, cDim, alignCenter)
		return
	}
	a := g.save.AscSel
	if u.button(s, "ascL", "‹", x+18, y+20, 56, 56, cAcc, a > 0) {
		g.save.AscSel--
		writeSave(&g.save)
	}
	if u.button(s, "ascR", "›", x+w-74, y+20, 56, 56, cAcc, a < g.save.AscMax) {
		g.save.AscSel++
		writeSave(&g.save)
	}
	a = g.save.AscSel
	if a == 0 {
		u.text(s, "ASCENSION 0  ·  STANDARD", fBold, 20, 960, y+18, cWhite, alignCenter)
		u.text(s, fmt.Sprintf("Unlocked up to Ascension %d", g.save.AscMax), fReg, 18, 960, y+52, cDim, alignCenter)
		return
	}
	asc := ascensions[a-1]
	u.text(s, fmt.Sprintf("ASCENSION %d  ·  %s", a, strings.ToUpper(asc.Name)), fBold, 20, 960, y+16, rgba(255, 140, 120, 255), alignCenter)
	u.text(s, asc.Desc+fmt.Sprintf("   (+%d%% shards and XP; includes all lower levels)", a*10), fReg, 17, 960, y+52, cDim, alignCenter)
}

// ---------------------------------------------------------------- summary

func (g *Game) drawSummary(s *ebiten.Image) {
	u := g.ui
	r := g.run
	title, col := "RUN OVER", cRed
	if g.victory {
		title, col = "VICTORY!", cGold
	} else if r.Endless {
		title, col = fmt.Sprintf("ENDLESS  ·  LOOP %d", r.Loop()), rgba(255, 140, 120, 255)
	}
	u.panel(s, 400, 60, 1120, 960, 0.92)
	u.heading(s, title, 90, 80, col, 1)
	lines := []string{
		fmt.Sprintf("%s   ·   Seed %s   ·   Ascension %d", r.Arch.Name, r.SeedCode, r.Asc),
		fmt.Sprintf("Reached %s (Act %d), %d chambers deep", biomes[r.AI()].Name, r.Act, r.Depth),
		fmt.Sprintf("Bricks %d   ·   Elites %d   ·   Bosses %d   ·   Best combo ×%d", r.Bricks, r.Elites, r.Bosses, r.MaxCombo),
		fmt.Sprintf("Final score  %d", r.FinalScore),
	}
	y := 220.0
	for _, l := range lines {
		u.text(s, l, fReg, 25, 960, y, cWhite, alignCenter)
		y += 42
	}
	// build strip
	bx := 960 - float64(min(len(r.Order), 16))*30
	for i, id := range r.Order {
		if i >= 16 {
			break
		}
		if m := modByID[id]; m != nil {
			u.drawEmblem(s, modEmblem(m), bx+float64(i)*60+30, y+34, 54, 1)
		}
	}
	y += 84
	// XP bar that fills up over a couple of seconds
	t := clamp01((g.stateT - 0.4) / 2.0)
	t = ease(t)
	xp := g.xpBefore + int(float64(g.xpGained)*t)
	rank := rankOf(xp)
	lo, hi := xpForRank(rank), xpForRank(rank+1)
	u.text(s, fmt.Sprintf("+%d XP", int(float64(g.xpGained)*t)), fBold, 26, 960, y, rgba(190, 160, 255, 255), alignCenter)
	u.medallion(s, 560, y+66, 30, rgba(150, 110, 255, 255))
	u.textOutlined(s, itoa(rank), fTitle, 30, 560, y+46, cWhite, alignCenter)
	u.bar(s, 610, y+56, 720, 20, float64(xp-lo)/float64(max(1, hi-lo)), 0, rgba(150, 110, 255, 255), 0, 0)
	y += 120
	if rank > g.rankBefore {
		pulse := 0.8 + 0.2*math.Sin(g.stateT*6)
		u.textOutlined(s, fmt.Sprintf("RANK UP!  Arcane Rank %d", rank), fTitle, 40, 960, y, ca(cGold, pulse), alignCenter)
		y += 58
		var news []*Mod
		for rk := g.rankBefore + 1; rk <= rank; rk++ {
			news = append(news, unlocksAt(rk)...)
		}
		if len(news) > 0 {
			u.text(s, "New items join the pool:", fReg, 20, 960, y, cDim, alignCenter)
			y += 34
			nx := 960 - float64(min(len(news), 8))*60
			for i, m := range news {
				if i >= 8 {
					break
				}
				u.drawEmblem(s, modEmblem(m), nx+float64(i)*120+60, y+34, 62, 1)
				u.text(s, m.Name, fReg, 14, nx+float64(i)*120+60, y+70, cDim, alignCenter)
			}
			y += 100
		}
	}
	u.iconLabel(s, icShard, fmt.Sprintf("+%d shards", g.shardsWon), fBold, 28, 960, y+6, rgba(200, 180, 255, 255), alignCenter)
	if g.ascUnlocked {
		u.textOutlined(s, fmt.Sprintf("Ascension %d unlocked!", g.save.AscMax), fBold, 26, 960, y+50, rgba(255, 140, 120, 255), alignCenter)
	}
	if r.Endless && !g.victory {
		ty := y + 50
		if g.ascUnlocked {
			ty += 40
		}
		if g.boardRank == 1 {
			pulse := 0.75 + 0.25*math.Sin(g.stateT*6)
			u.textOutlined(s, "NEW ENDLESS RECORD!", fTitle, 38, 960, ty, ca(cGold, pulse), alignCenter)
		} else if g.boardRank > 0 {
			u.textOutlined(s, fmt.Sprintf("Endless leaderboard  ·  #%d", g.boardRank), fBold, 26, 960, ty, rgba(255, 170, 130, 255), alignCenter)
		}
	}
	if g.victory && !r.Endless {
		pulse := 0.5 + 0.5*math.Sin(g.stateT*4)
		drawSprite(s, sprSoft, u.X(990-80), u.Y(930-60), 580*u.S, 184*u.S, ca(rgba(255, 120, 90, 255), 0.25+0.2*pulse), true)
		if u.button(s, "endless", "Continue: Endless Mode", 1000, 930, 420, 64, rgba(255, 120, 90, 255), g.stateT > 0.8) {
			r.Endless = true
			r.NextAct(&g.save)
			r.Recompute(&g.save)
			g.victory = false
			g.setState(StActIntro)
			return
		}
		if u.button(s, "summenu", "Main menu", 500, 930, 420, 64, cAcc, g.stateT > 0.8) {
			g.run = nil
			g.level = nil
			g.setState(StMenu)
		}
		return
	}
	if u.button(s, "summenu", "Main menu", 760, 930, 400, 64, cAcc, g.stateT > 0.8) {
		g.run = nil
		g.level = nil
		g.setState(StMenu)
	}
}

// fusionToast announces a newly forged fusion.
func (g *Game) announceFusions() {
	r := g.run
	for _, id := range r.NewFusions {
		f := fusionByID[id]
		if f == nil {
			continue
		}
		g.save.Fusions[id] = true
		g.banner, g.bannerDesc, g.bannerT = "FUSION: "+strings.ToUpper(f.Name), f.Desc, 4.5
		g.bannerIcon = fusionEmblem(f)
		g.bannerCol = f.Col
		g.snd.PlayKeyed("synergy")
		g.snd.Duck(0.7, 3.5)
	}
	r.NewFusions = nil
}

var _ = color.RGBA{}

// drawRecords shows the local Endless leaderboard.
func (g *Game) drawRecords(s *ebiten.Image, y float64) {
	u := g.ui
	w := 1100.0
	x := 960 - w/2
	u.panel(s, x, y, w, 700, 0.92)
	u.text(s, "ENDLESS RECORDS", fBold, 22, 960, y+24, lighten(u.theme().Accent, 0.3), alignCenter)
	if len(g.save.Board) == 0 {
		msg := "No Endless runs yet. Finish the story to unlock Endless Mode, then see how far you can go."
		if g.save.Wins > 0 {
			msg = "No Endless runs yet. Start one from the main menu and set the first record."
		}
		u.paragraph(s, msg, fReg, 24, 960, y+300, 760, cDim, alignCenter)
		return
	}
	cols := []struct {
		label string
		x     float64
		al    align
	}{{"#", x + 60, alignCenter}, {"SCORE", x + 260, alignRight}, {"LOOP", x + 380, alignCenter}, {"DEPTH", x + 500, alignCenter},
		{"CLASS", x + 600, alignLeft}, {"ASC", x + 840, alignCenter}, {"DATE", x + w - 50, alignRight}}
	hy := y + 76
	for _, c := range cols {
		u.text(s, c.label, fBold, 16, c.x, hy, cDim, c.al)
	}
	u.glowLine(s, x+30, x+w-30, hy+30, 1.5, u.theme().Edge, 0.8)
	for i, e := range g.save.Board {
		ry := hy + 50 + float64(i)*56
		col := cWhite
		if i < 3 {
			col = []color.RGBA{cGold, rgba(220, 226, 240, 255), rgba(230, 160, 110, 255)}[i]
			fillPath(s, boxPath(u.X(x+24), u.Y(ry-10), (w-48)*u.S, 48*u.S, 10*u.S, false), ca(col, 0.1), ca(col, 0.04), vgrad(u.Y(ry-10), u.Y(ry+38)), false)
		}
		vals := []string{itoa(i + 1), itoa(e.Score), itoa(e.Loop), itoa(e.Depth), e.Class, itoa(e.Asc), e.Date}
		for j, c := range cols {
			f := fReg
			if j < 2 {
				f = fBold
			}
			u.text(s, vals[j], f, 22, c.x, ry, col, c.al)
		}
	}
}
