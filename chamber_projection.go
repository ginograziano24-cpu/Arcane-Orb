package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"math"
)

// The route is projected from an orb onto an engraved spell chart. Graph rules stay in Run.
func (g *Game) drawChamberProjection(s *ebiten.Image) {
	u := g.ui
	t := g.stateT
	fillRect(s, u.X(425), u.Y(126), 1030*u.S, 865*u.S, rgba(12, 7, 30, 190))
	drawSprite(s, sprSoft, u.X(420), u.Y(20), 1080*u.S, 1120*u.S, rgba(0, 0, 0, 215), false)
	for _, rx := range []float64{530, 575, 610} {
		for i := 0; i < 64; i++ {
			a, b := float64(i)*math.Pi/32, float64(i+1)*math.Pi/32
			strokeLine(s, u.X(960+math.Cos(a)*rx), u.Y(1010+math.Sin(a)*rx*.19), u.X(960+math.Cos(b)*rx), u.Y(1010+math.Sin(b)*rx*.19), 1.5*u.S, rgba(148, 113, 194, 90))
		}
	}
	for i := 0; i < 24; i++ {
		a := float64(i) * math.Pi / 12
		strokeLine(s, u.X(960+math.Cos(a)*540), u.Y(1010+math.Sin(a)*103), u.X(960+math.Cos(a)*558), u.Y(1010+math.Sin(a)*106), 2*u.S, rgba(221, 190, 135, 130))
	}
	// Fixed architectural rails give the chart a physical frame, rather than a floating menu graph.
	for _, x := range []float64{440, 1480} {
		strokeLine(s, u.X(x), u.Y(145), u.X(x), u.Y(930), 1.5*u.S, rgba(185, 150, 220, 55))
	}
	drawSprite(s, sprSoft, u.X(840), u.Y(900), 240*u.S, 210*u.S, rgba(178, 117, 255, 95), true)
	g.drawUIIcon(s, chamberSigil(NCombat), 960, 1014, 86, 1)
	u.text(s, "Where to next, hero?", fReg, 20, 470, 137, cDim, alignLeft)
	u.text(s, "The glowing seals are yours to pick", fReg, 17, 960, 1053, cDim, alignCenter)
	// Quiet light rises from the projector, behind all interactive seals.
	for i := 0; i < 14; i++ {
		f := math.Mod(t*.11+float64(i)/14, 1)
		x := 960 + math.Sin(float64(i)*2.4)*260*f
		y := 985 - f*750
		drawSprite(s, sprSoft, u.X(x-3), u.Y(y-3), 6*u.S, 6*u.S, ca(cGold, .13*math.Sin(f*math.Pi)), true)
	}
}

func (g *Game) drawChamberLeyLine(s *ebiten.Image, x0, y0, x1, y1, bend float64) {
	u := g.ui
	c := rgba(255, 222, 140, 255)
	point := func(t float64) (float64, float64) {
		return x0 + (x1-x0)*t, y0 + (y1-y0)*t + 4*bend*t*(1-t)
	}
	segments := 1
	if bend != 0 {
		segments = 12
	}
	ax, ay := x0, y0
	for i := 1; i <= segments; i++ {
		bx, by := point(float64(i) / float64(segments))
		strokeLine(s, u.X(ax), u.Y(ay), u.X(bx), u.Y(by), 12*u.S, prem(c, 0.18))
		strokeLine(s, u.X(ax), u.Y(ay), u.X(bx), u.Y(by), 4*u.S, prem(c, 0.95))
		ax, ay = bx, by
	}
	if g.settings.Particles == 0 {
		return
	}
	for i := range 3 {
		f := math.Mod(g.stateT*0.6+float64(i)/3, 1)
		x, y := point(f)
		drawSprite(s, sprSoft, u.X(x-14), u.Y(y-14), 28*u.S, 28*u.S, ca(c, 0.9*math.Sin(f*math.Pi)), true)
	}
}

func (g *Game) drawChamberSeal(s *ebiten.Image, n *MapNode, x, y, rad float64, reachable bool) {
	u := g.ui
	c := nodeColors[n.Type]
	if !reachable {
		c = darken(c, .65)
	}
	fillPath(s, circlePath(u.X(x), u.Y(y), rad*1.25*u.S), rgba(9, 7, 22, 240), rgba(25, 16, 45, 240), vgrad(u.Y(y-rad), u.Y(y+rad)), false)
	strokePath(s, circlePath(u.X(x), u.Y(y), rad*1.24*u.S), 1.3*u.S, ca(c, .8), false)
	if reachable {
		strokePath(s, circlePath(u.X(x), u.Y(y), (rad+18)*u.S), 1*u.S, ca(c, .4), false)
		for k := 0; k < 3; k++ {
			a := g.stateT*.4 + float64(k)*math.Pi*2/3
			xx, yy := x+math.Cos(a)*(rad+18), y+math.Sin(a)*(rad+18)
			drawSprite(s, sprSoft, u.X(xx-5), u.Y(yy-5), 10*u.S, 10*u.S, ca(c, .9), true)
		}
	}
}

func (g *Game) drawMapSidePanel(s *ebiten.Image, index int, reachable bool) {
	u := g.ui
	chamberAccent, buildAccent := cGold, cAcc
	if g.mapShowBuild {
		chamberAccent, buildAccent = cAcc, cGold
	}
	if u.button(s, "mapchamber", "Chamber", 1530, 30, 174, 50, chamberAccent, true) {
		g.mapShowBuild = false
	}
	if u.button(s, "mapbuild", "Build", 1716, 30, 174, 50, buildAccent, true) {
		g.mapShowBuild = true
	}
	if g.mapShowBuild {
		g.drawRunBuild(s, 94)
		return
	}
	g.drawChamberPreview(s, index, reachable)
}

func (g *Game) drawChamberPreview(s *ebiten.Image, index int, reachable bool) {
	if index < 0 || index >= len(g.run.Map.Nodes) {
		return
	}
	u := g.ui
	n := g.run.Map.Nodes[index]
	x, y, w := 1530.0, 94.0, 360.0
	c := nodeColors[n.Type]
	u.panel(s, x, y, w, 620, .93)
	u.text(s, "CHAMBER ATTUNEMENT", fRune, 19, x+22, y+23, cGold, alignLeft)
	g.drawChamberGlyph(s, n, x+w/2, y+112, 62, true)
	u.text(s, nodeNames[n.Type], fRune, 32, x+w/2, y+169, c, alignCenter)
	descriptions := []string{
		"Break the formation, then pick a shiny new upgrade.",
		"A tougher brawl. Win it for a relic and an upgrade.",
		"A merchant with a cart full of goodies. Bring gold.",
		"A treasure chest, just sitting there. Pick a relic.",
		"Something strange is afoot. Read the choices before you leap.",
		"A crackling campfire. Catch your breath before the next fight.",
		"The realm's Guardian. Beat it for a rare-or-better relic.",
	}
	bottom := u.paragraph(s, descriptions[n.Type], fReg, 21, x+24, y+226, w-48, cWhite, alignLeft)
	if o := omenByID[n.Omen]; o != nil && !n.Visited {
		yy := math.Max(y+330, bottom+20)
		u.text(s, o.Name, fBold, 22, x+24, yy, o.Col, alignLeft)
		u.paragraph(s, o.Desc, fReg, 19, x+24, yy+35, w-48, cDim, alignLeft)
	}
	status := "Out of reach. Follow the glowing lines to get here."
	if n.Visited {
		status = "Been there, smashed that."
	}
	if reachable {
		status = "Within reach. Choose this seal to step inside."
	}
	u.paragraph(s, status, fReg, 18, x+24, y+520, w-48, ca(cWhite, .8), alignLeft)
}

func (g *Game) drawChamberGlyph(s *ebiten.Image, n *MapNode, x, y, rad float64, reachable bool) {
	b := .42
	if reachable {
		b = 1
	}
	if n.Visited {
		b = .65
	}
	im := chamberSigil(n.Type)
	if im == nil {
		im = uiIcon(nodeIconFile[n.Type])
	}
	if im != nil {
		g.drawUIIconTint(s, im, x, y, rad*1.8, b, b, b, 1)
		return
	}
	g.ui.drawEmblemDark(s, glyphImage(nodeGlyphs[n.Type], ca(cWhite, b)), x, y, rad*1.2, 1)
}
