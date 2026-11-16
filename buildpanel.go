package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The Build panel, version B2 (ROADMAP-50 #32, approved 2026-11-05):
// items grouped by rarity (fusions first, then Mythic down to Common, then
// curses) with a count per group; hovering an item shows its full card
// beside the panel; a "Your stats" box at the bottom lists each stat the
// items have moved, base value in dim grey and the value now in green.
// The panel keeps its place and look on the map and the pause screen.

// buildGroup is one heading of the Build panel.
type buildGroup struct {
	Title string
	Col   color.RGBA
	Mods  []*Mod
}

// buildGroups sorts the run's items for the panel.
func buildGroups(r *Run) []buildGroup {
	var out []buildGroup
	for rar := Mythic; rar >= Common; rar-- {
		gr := buildGroup{Title: rarityNames[rar], Col: rarityColors[rar]}
		for _, id := range r.Order {
			if m := modByID[id]; m != nil && r.Mods[id] > 0 && !m.Curse && m.Rarity == rar {
				gr.Mods = append(gr.Mods, m)
			}
		}
		if len(gr.Mods) > 0 {
			out = append(out, gr)
		}
	}
	curses := buildGroup{Title: "Curses", Col: tagColors["curse"]}
	for _, id := range r.Order {
		if m := modByID[id]; m != nil && r.Mods[id] > 0 && m.Curse {
			curses.Mods = append(curses.Mods, m)
		}
	}
	if len(curses.Mods) > 0 {
		out = append(out, curses)
	}
	return out
}

// drawBuildGroups lists fusions and items from y down to bottom.
func (g *Game) drawBuildGroups(s *ebiten.Image, r *Run, y, bottom float64) {
	u := g.ui
	var hover *Mod
	hoverY := 0.0
	header := func(title string, n int, col color.RGBA) {
		u.text(s, fmt.Sprintf("%s  ·  %d", strings.ToUpper(tr(title)), n), fBold, 14, 1550, y+4, ca(lighten(col, 0.2), 0.9), alignLeft)
		u.glowLine(s, 1550, 1870, y+24, 1, ca(col, 0.5), 0.5)
		y += 30
	}
	full := func() bool { return y+28 > bottom }
	more := func(n int) {
		u.text(s, fmt.Sprintf("+%d more", n), fReg, 16, 1576, y, ca(cDim, 0.8), alignLeft)
		y += 26
	}
	if len(r.Fused) > 0 {
		header("Fusions", len(r.Fused), rgba(255, 140, 220, 255))
		for i, id := range r.Fused {
			if full() {
				more(len(r.Fused) - i)
				break
			}
			f := fusionByID[id]
			u.drawEmblem(s, fusionEmblem(f), 1562, y+13, 30, 1)
			u.text(s, f.Name, fBold, 19, 1582, y, f.Col, alignLeft)
			if u.hover(1540, y-4, 340, 28) {
				g.tooltip = "FUSION — " + f.Name + ": " + f.Desc
			}
			y += 30
		}
	}
	// fusions one item away, as faint outlines (ROADMAP-50 #58)
	if near := nearFusions(r); len(near) > 0 && y+70 < bottom {
		header("One item away", len(near), rgba(200, 160, 220, 255))
		for _, nf := range near {
			if full() {
				break
			}
			pulse := 0.55 + 0.15*math.Sin(u.time*2.5)
			u.drawEmblem(s, fusionEmblem(nf.F), 1562, y+13, 30, pulse)
			u.text(s, nf.F.Name, fBold, 17, 1582, y-2, ca(nf.F.Col, pulse), alignLeft)
			if m := modByID[nf.Need]; m != nil {
				u.text(s, trJoin("needs ", m.Name), fReg, 14, 1582, y+16, ca(cDim, 0.8), alignLeft)
			}
			if u.hover(1540, y-4, 340, 34) {
				g.tooltip = trJoin(nf.F.Name, " — ", nf.F.Desc)
			}
			y += 38
		}
	}
	groups := buildGroups(r)
	for gi, gr := range groups {
		if y+58 > bottom {
			// no room for another heading and a line: count what's left
			left := 0
			for _, rest := range groups[gi:] {
				left += len(rest.Mods)
			}
			if left > 0 {
				more(left)
			}
			break
		}
		n := 0
		for _, m := range gr.Mods {
			n += r.Mods[m.ID]
		}
		header(gr.Title, n, gr.Col)
		for i, m := range gr.Mods {
			if full() {
				more(len(gr.Mods) - i)
				break
			}
			label := m.Name
			if r.Mods[m.ID] > 1 {
				label = tr(m.Name) + " " + roman(r.Mods[m.ID]) // its level: Heavy Core III
			}
			u.drawEmblem(s, modEmblem(m), 1560, y+12, 26, 1)
			u.text(s, label, fBold, 19, 1576, y, gr.Col, alignLeft)
			if u.hover(1540, y-4, 340, 28) {
				hover, hoverY = m, y
			}
			y += 28
		}
	}
	if hover != nil {
		// the item's own card, beside the panel
		w, h := 330.0, 440.0
		cy := math.Max(40, math.Min(1040-h, hoverY-h/2))
		g.drawCard(s, hover, 1530-w-18, cy, w, h, 0, true)
	}
}

// drawStatsBox is the B2 "Your stats" box: every stat the items have
// moved, base in dim grey, now in green (red if an item lowered it).
// Orb damage and critical chance always show.
func (g *Game) drawStatsBox(s *ebiten.Image, x, y, w, h float64) {
	u := g.ui
	r := g.run
	if r == nil || r.Arch == nil {
		return
	}
	fillPath(s, boxPath(u.X(x), u.Y(y), w*u.S, h*u.S, 10*u.S, false), ca(cBlack, 0.28), ca(cBlack, 0.18), vgrad(u.Y(y), u.Y(y+h)), false)
	strokePath(s, boxPath(u.X(x), u.Y(y), w*u.S, h*u.S, 10*u.S, false), 1.2*u.S, ca(u.theme().Edge2, 0.8), false)
	u.text(s, "Your stats", fRune, 22, x+14, y+10, cGold, alignLeft)
	base := r.kitStats(&g.save)
	rows := 0
	type line struct {
		label, a, b string
		up, moved   bool
	}
	var lines []line
	for i, row := range statRows {
		a, b := fmtStat(row.Get(&base), row.Fmt), fmtStat(row.Get(&r.Stats), row.Fmt)
		if i >= 2 && a == b {
			continue
		}
		lines = append(lines, line{row.Label, a, b, row.Get(&r.Stats) >= row.Get(&base), a != b})
	}
	room := int((h - 52) / 25)
	ly := y + 48
	for i, l := range lines {
		if rows == room-1 && len(lines)-i > 1 {
			u.text(s, fmt.Sprintf("+%d more", len(lines)-i), fReg, 15, x+14, ly, ca(cDim, 0.8), alignLeft)
			break
		}
		u.text(s, l.label, fReg, 15, x+14, ly, cDim, alignLeft)
		now := cWhite
		if l.moved {
			now = rgba(140, 240, 150, 255)
			if !l.up {
				now = rgba(255, 130, 120, 255)
			}
			u.text(s, l.a, fReg, 15, x+w-92, ly, ca(cDim, 0.6), alignRight)
			u.text(s, "›", fReg, 15, x+w-80, ly, ca(cDim, 0.5), alignCenter)
		}
		u.text(s, l.b, fBold, 15, x+w-14, ly, now, alignRight)
		ly += 25
		rows++
	}
}

// nearFusion is a fusion the run holds one half of.
type nearFusion struct {
	F    *Fusion
	Need string // the missing half
}

// nearFusions lists up to three fusions the build is one item away from.
func nearFusions(r *Run) []nearFusion {
	var out []nearFusion
	for i := range fusions {
		f := &fusions[i]
		if r.HasFusion(f.ID) {
			continue
		}
		a, b := r.Owned(f.A), r.Owned(f.B)
		switch {
		case a && !b:
			out = append(out, nearFusion{f, f.B})
		case b && !a:
			out = append(out, nearFusion{f, f.A})
		}
		if len(out) == 3 {
			break
		}
	}
	return out
}
