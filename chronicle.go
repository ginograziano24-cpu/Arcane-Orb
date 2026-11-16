package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Fumbleton's Chronicle: a Codex tab of short pages about the worlds and
// the Guardians, "written" by Archmage Fumbleton. A world's page opens the
// first time a run enters it (or once a run has reached an act that holds
// it); a Guardian's page opens when you first face it.

type chroniclePage struct {
	Key, Title, Glyph string
	World, Act        int // a world page (World >= 0) or a Guardian page (Act > 0); both unset = the foreword
	Paras             []string
}

var chronicle = []chroniclePage{
	{"intro", "A Foreword, of Sorts", "book", -1, 0, []string{
		"This is my chronicle of the walls, the worlds they cover and the Guardians who keep them. I am writing it so that historians will know the truth.",
		"The truth is that the walls are only a little bit my fault. I have underlined that part twice.",
	}},
	{"w0", "Glimmer Grove", "swirl", WNeon, 0, []string{
		"A garden that glows all night, so the flowers never have to sleep. They are very grumpy about it.",
		"The bricks here hum when you break them. I tried to write the tune down. Mirabel says my notes look like a spider fell in the ink.",
	}},
	{"w1", "Sunset Meadow", "leaf", WMeadow, 0, []string{
		"It is always the end of the day in the Meadow. The sun got stuck there years ago and nobody had the heart to move it.",
		"Excellent for picnics. Terrible for anyone trying to get to bed on time.",
	}},
	{"w2", "Crystal Caverns", "gem", WCaverns, 0, []string{
		"Deep underground, the stones sing. Each crystal knows exactly one note and is very proud of it.",
		"Do not hum along. They take it as a challenge.",
	}},
	{"w3", "Ancient Ruins", "urn", WRuins, 0, []string{
		"Somebody lived here long before any of us, and built everything out of bricks. I have a theory about who it was.",
		"My theory is: people who liked bricks. I am almost certain.",
	}},
	{"w4", "Cloud Kingdom", "feather", WClouds, 0, []string{
		"Castles drift above the sky here, tied down with very long ropes. Zephyr grew up on one of them and still waves at it.",
		"Before the walls, you could count every star from the castle roofs. Zephyr says he will count them again when we are done.",
	}},
	{"w5", "Machine Works", "gear", WMachine, 0, []string{
		"The whole place is one enormous engine, and every part of it makes bricks. Rivet grew up here, between the gears.",
		"I asked the engine what it was for. It made a brick. I think that was the answer.",
	}},
	{"g1", "The Prism Colossus", "gem", -1, 1, []string{
		"A giant of coloured glass who guards the Outer Realms. It spends most of its day admiring its own reflection.",
		"When it breaks it makes the loveliest sound. I am told it does not agree.",
	}},
	{"g2", "The Crucible Golem", "anvil", -1, 2, []string{
		"An iron worker who forges the bricks for everyone else. It has never taken a day off, and it will tell you so.",
		"I once suggested a holiday. It made me a brick with the word NO stamped on it. I keep it on my desk.",
	}},
	{"g3", "The Crystal Warden", "iceheart", -1, 3, []string{
		"The highest wall of all, and the one who keeps the other Guardians in line. Cold, clever and very, very tidy.",
		"I will say this for the Warden: it never once blamed me for the walls. I have decided to like it for that.",
	}},
}

// chronicleOpen reports whether page p can be read.
func (g *Game) chronicleOpen(p *chroniclePage) bool {
	switch {
	case p.World < 0 && p.Act == 0:
		return true
	case g.save.Chronicle[p.Key]:
		return true
	case p.Act > 0:
		return g.save.BestAct > p.Act // reached the act after it: beaten
	}
	for a := 0; a < numActs && a < g.save.BestAct; a++ {
		for _, w := range acts[a].Pool {
			if w == p.World {
				return true
			}
		}
	}
	return false
}

// noteChronicle opens the pages for a chamber about to be played
// (newLevel): its world's, and its Guardian's on a Guardian fight.
func (g *Game) noteChronicle(world int, boss bool) {
	r := g.run
	if r == nil {
		return
	}
	for _, code := range noShowRuns {
		if r.SeedCode == code {
			return
		}
	}
	if g.save.Chronicle == nil {
		g.save.Chronicle = map[string]bool{}
	}
	g.save.Chronicle["w"+itoa(world)] = true
	if boss {
		g.save.Chronicle["g"+itoa(r.AI()+1)] = true
	}
}

// drawChronicleTab lists the pages on the left and reads the chosen one on
// the right; locked pages show only their place.
func (g *Game) drawChronicleTab(s *ebiten.Image, y float64) {
	u := g.ui
	lx, lw := 230.0, 420.0
	for i := range chronicle {
		p := &chronicle[i]
		open := g.chronicleOpen(p)
		yy := y + float64(i)*66
		hv, clk := u.hit("chron"+itoa(i), lx, yy, lw, 58)
		sel := g.chronSel == i
		u.frame(s, lx, yy, lw, 58, 0.9)
		if sel {
			u.glowLine(s, lx+12, lx+lw-12, yy+56, 3, cGold, 1)
		}
		if hv && u.nav.mode {
			u.focusRing(s, lx, yy, lw, 58, 10)
		}
		col := cGold
		if !open {
			col = rgba(110, 100, 130, 255)
		}
		u.drawEmblem(s, glyphImage(p.Glyph, col), lx+34, yy+29, 34, 1)
		if open {
			u.text(s, p.Title, fBold, 20, lx+64, yy+16, cWhite, alignLeft)
		} else {
			u.text(s, "Not yet written", fBold, 20, lx+64, yy+16, cDim, alignLeft)
		}
		if clk && open {
			g.chronSel = i
			g.snd.Play("click")
		}
	}
	p := &chronicle[min(max(g.chronSel, 0), len(chronicle)-1)]
	if !g.chronicleOpen(p) {
		p = &chronicle[0]
	}
	px, pw := 690.0, 1000.0
	ph := float64(len(chronicle))*66 - 8
	u.panel(s, px, y, pw, ph, 0.92)
	u.drawEmblem(s, glyphImage(p.Glyph, cGold), px+70, y+70, 64, 1)
	u.textOutlined(s, p.Title, fTitle, 36, px+120, y+44, cGold, alignLeft)
	ty := y + 140
	for _, para := range p.Paras {
		for _, l := range u.wrap(tr(para), fSerif, 24, pw-120) {
			u.rawText = true
			u.text(s, l, fSerif, 24, px+60, ty, cWhite, alignLeft)
			u.rawText = false
			ty += 36
		}
		ty += 18
	}
	u.text(s, "— Archmage Fumbleton", fSerif, 22, px+pw-60, y+ph-60, cDim, alignRight)
}
