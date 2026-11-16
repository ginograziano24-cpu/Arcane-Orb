package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Build hints: after you take a card from a reward, a toast names up to
// three items that work well with it: the ones it fuses with first, then
// items of the same school (tag). Owned items are left out, and so are
// items this run can't be offered.

func (g *Game) buildHints(m *Mod) []*Mod {
	r := g.run
	if r == nil || m == nil || m.Curse {
		return nil
	}
	seen := map[string]bool{m.ID: true}
	var out []*Mod
	add := func(c *Mod) {
		if c == nil || seen[c.ID] || len(out) >= 3 || c.Curse || c.Retired || r.Mods[c.ID] > 0 || !r.offerAllowed(c) {
			return
		}
		if c.Origin != "" && (r.Arch == nil || c.Origin != r.Arch.ID) {
			return
		}
		seen[c.ID] = true
		out = append(out, c)
	}
	for _, f := range fusions {
		switch m.ID {
		case f.A:
			add(modByID[f.B])
		case f.B:
			add(modByID[f.A])
		}
	}
	if len(m.Tags) > 0 {
		for _, c := range modList {
			for _, t := range c.Tags {
				if t == m.Tags[0] && c.Relic == m.Relic {
					add(c)
					break
				}
			}
		}
	}
	return out
}

// hintAfterPick shows the hint toast for the card just taken.
func (g *Game) hintAfterPick(m *Mod) {
	hs := g.buildHints(m)
	if len(hs) == 0 {
		return
	}
	var names []string
	for _, h := range hs {
		names = append(names, tr(h.Name))
	}
	g.toastImg(trJoin("Pairs well with ", tr(m.Name)), strings.Join(names, listSep), modColor(m), modEmblem(hs[0]))
}

// drawPauseRules adds what bends this chamber and run to the pause screen:
// the chamber's omen (with a challenge's progress), the challenge or the
// week's mutators.
func (g *Game) drawPauseRules(s *ebiten.Image, x, y, w float64) {
	u := g.ui
	L, r := g.level, g.run
	type line struct {
		name, desc, glyph string
		col               color.RGBA
	}
	var lines []line
	if L != nil && L.omen != nil {
		o := L.omen
		d := tr(o.Desc)
		if o.Challenge && o.Status != nil {
			d += "  (" + o.Status(L) + ")"
		}
		lines = append(lines, line{o.Name, d, o.Glyph, o.Col})
	}
	if c := r.challenge(); c != nil {
		lines = append(lines, line{c.Name, tr(c.Rules), c.Glyph, c.Col})
	}
	for _, id := range r.Weekly {
		if m := mutatorByID[id]; m != nil {
			lines = append(lines, line{m.Name, tr(m.Rules), m.Glyph, rgba(120, 220, 255, 255)})
		}
	}
	if len(lines) == 0 {
		return
	}
	h := 56 + float64(len(lines))*66
	u.panel(s, x, y, w, h, 0.85)
	u.text(s, "In effect", fRune, 24, x+20, y+12, cGold, alignLeft)
	ly := y + 56
	for _, l := range lines {
		u.drawEmblem(s, glyphImage(l.glyph, l.col), x+38, ly+22, 30, 1)
		u.text(s, l.name, fBold, 17, x+64, ly, lighten(l.col, 0.3), alignLeft)
		u.rawText = true
		for i, t := range u.wrap(l.desc, fReg, 14, w-84) {
			if i >= 2 {
				break
			}
			u.text(s, t, fReg, 14, x+64, ly+22+float64(i)*18, cDim, alignLeft)
		}
		u.rawText = false
		ly += 66
	}
}

// ---------------------------------------------------------------- damage numbers

// srcNumber floats a hit's damage in its source's colour (Settings >
// Effects > Damage numbers): violet for the orb, orange for fire, blue for
// lightning, gold for blasts... crits are larger with a "!". Damage over
// time and tiny ticks stay quiet, and at most a dozen numbers start in a
// frame so a chain reaction doesn't bury the board.
func (L *Level) srcNumber(b *Brick, h Hit) {
	if h.Dmg < 0.5 || L.dmgNums >= 12 || len(L.Floats) > 40 {
		return
	}
	src := h.Src
	if src == "" {
		src = "Orb"
	}
	if src == "Burning" && !h.Crit {
		return // the burn ticks would be a constant drizzle
	}
	L.dmgNums++
	c := dmgColor(src)
	s := fmtDmg(h.Dmg)
	size := 18.0
	if h.Crit {
		s += "!"
		size = 26
		c = lighten(c, 0.25)
	}
	rng := L.fxr()
	L.Floats = append(L.Floats, &FloatText{P: V3{b.X() + (rng.Float64()-0.5)*0.8, 1.8, b.Z()}, S: s, Col: c, Life: 0.9, Scale: size})
}
