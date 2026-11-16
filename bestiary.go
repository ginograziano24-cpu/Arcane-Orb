package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// The bestiary (ROADMAP-50 #109): a Codex page for everything on the board
// that isn't you. Critters and special bricks are listed from the start;
// each Guardian appears once a run has reached its act.

type beast struct {
	Name, Desc, Glyph string
	Col               [3]uint8
	Act               int // a Guardian: shown once a run reaches this act (0: always)
}

var bestiary = []beast{
	{"Wanderer", "Pop it with the orb and it bursts into coins. It was only out for a walk.", "feather", [3]uint8{170, 230, 150}, 0},
	{"Golden Wanderer", "A wanderer that rolled in gold. Rare, shiny, and worth a lot more when it pops.", "coin", [3]uint8{255, 214, 90}, 0},
	{"Pilferer", "Runs down falling power-ups and makes off with them. Hit it to get your loot back.", "fist", [3]uint8{200, 150, 255}, 0},
	{"Mason", "Lays fresh bricks in front of the formation. Very hardworking. Very annoying.", "hive", [3]uint8{220, 180, 130}, 0},
	{"Bomber Bird", "Flies over the bricks with a bomb that drops wherever the orb hits it. Aim well and it's on your side.", "bomb", [3]uint8{255, 150, 80}, 0},
	{"Shield Brick", "Protects every brick around it until it breaks. Void hits walk right past. Break it first.", "halo", [3]uint8{120, 200, 255}, 0},
	{"Armoured Brick", "Shrugs off plain hits. Crits, void, blasts and elements get through, and six plain hits crack it.", "shield", [3]uint8{200, 200, 215}, 0},
	{"Blast Brick", "Packed with blasting powder. Hit it and its neighbours go with it.", "burst", [3]uint8{255, 120, 60}, 0},
	{"Crystal", "Gold grows inside. Break it for a handful of coins.", "gem", [3]uint8{140, 220, 255}, 0},
	{"Prism Colossus", "A giant of coloured glass, walling up the bright lands. Thinks it's very pretty. It is, a bit.", "gem", [3]uint8{255, 140, 220}, 1},
	{"Crucible Golem", "A huge iron worker who forges the bricks for everyone else. Has never taken a day off and is proud of it.", "anvil", [3]uint8{255, 150, 70}, 2},
	{"Crystal Warden", "The boss of the bosses, high in the Crystal Caverns. Its walls are the toughest of all, and so is its attitude.", "iceheart", [3]uint8{150, 230, 255}, 3},
}

func (g *Game) drawBestiaryTab(s *ebiten.Image, y float64) {
	u := g.ui
	cols, w, h, gap := 3, 520.0, 150.0, 18.0
	x0 := 960 - (float64(cols)*w+float64(cols-1)*gap)/2
	for i, b := range bestiary {
		x := x0 + float64(i%cols)*(w+gap)
		yy := y + float64(i/cols)*(h+gap)
		col := rgba(b.Col[0], b.Col[1], b.Col[2], 255)
		known := b.Act == 0 || g.save.BestAct >= b.Act
		u.panel(s, x, yy, w, h, 0.88)
		if !known {
			u.drawEmblem(s, glyphImage("mask", ca(cDim, 0.5)), x+70, yy+h/2, 80, 1)
			u.text(s, "? ? ?", fSerif, 26, x+130, yy+30, ca(cDim, 0.6), alignLeft)
			u.paragraph(s, "A Guardian you haven't met yet.", fReg, 17, x+130, yy+72, w-150, ca(cDim, 0.6), alignLeft)
			continue
		}
		bob := 0.0
		if u.hover(x, yy, w, h) {
			bob = -4 + 2*float64(int(u.time*6)%2) // a happy little hop
		}
		drawSprite(s, sprSoft, u.X(x+70-60), u.Y(yy+h/2-60), 120*u.S, 120*u.S, ca(col, 0.3), true)
		u.drawEmblem(s, glyphImage(b.Glyph, col), x+70, yy+h/2+bob, 80, 1)
		u.text(s, b.Name, fSerif, 26, x+130, yy+18, lighten(col, 0.2), alignLeft)
		if b.Act > 0 {
			u.text(s, fmt.Sprintf("GUARDIAN  ·  ACT %s", roman(b.Act)), fBold, 13, x+130, yy+52, ca(cDim, 0.9), alignLeft)
			u.paragraph(s, b.Desc, fReg, 16, x+130, yy+72, w-150, cDim, alignLeft)
		} else {
			u.paragraph(s, b.Desc, fReg, 17, x+130, yy+56, w-150, cDim, alignLeft)
		}
	}
}
