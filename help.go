package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// How to Play: an illustrated guide in seven pages. Each page pairs a
// small animated illustration with a short, scannable explanation.
// Arrows / Q E / LB RB turn pages.

var helpPages = []string{"The Orb & the Paddle", "A Run in Three Acts", "Know Your Bricks", "Power-ups",
	"Upgrades, Relics & Fusions", "Origins", "Beyond the Run"}

var powerDesc = [numPowers]string{
	PMulti:   "Splits your orb into three",
	PExpand:  "A wider paddle for a while",
	PShrink:  "Curse: a narrower paddle",
	PSlow:    "Slows every orb down",
	PHaste:   "Curse: orbs speed up",
	PFire:    "Orbs burn through bricks, double damage",
	PLaser:   "Paddle cannons fire on their own",
	PBarrier: "A one-time safety net under the paddle",
	PGlue:    "The orb sticks; click to aim and release",
	PMagnet:  "Pulls coins and capsules to you",
}

var nodeDesc = []string{
	"A chamber of bricks. Clear it for an upgrade.",
	"Tougher bricks and an elite. Pays out a relic.",
	"Spend gold on upgrades, relics and healing.",
	"A free relic, no fight required.",
	"A strange encounter: a gamble, a bargain, a gift.",
	"Heal, or train for a stronger upgrade.",
	"The act's Guardian. Defeat it to move on.",
}

func (g *Game) drawHelp(s *ebiten.Image) {
	u := g.ui
	n := len(helpPages)
	if g.navLatch.pageL || g.keyHit(ebiten.KeyArrowLeft) && !u.nav.mode {
		g.helpPage = (g.helpPage + n - 1) % n
		g.snd.Play("cardhover")
	}
	if g.navLatch.pageR || g.keyHit(ebiten.KeyArrowRight) && !u.nav.mode {
		g.helpPage = (g.helpPage + 1) % n
		g.snd.Play("cardhover")
	}
	if g.helpPage != g.helpShown {
		g.helpShown = g.helpPage
		g.helpT = 0
	}
	g.helpT += u.DT
	pg := g.helpPage
	a := ease(clamp01(g.helpT * 3))

	u.heading(s, "How to Play", 60, 40, cGold, 1)
	u.text(s, strings.ToUpper(helpPages[pg]), fSerif, 26, 960, 130, ca(lighten(u.theme().Accent, 0.3), a), alignCenter)

	switch pg {
	case 0:
		g.helpControls(s, a)
	case 1:
		g.helpRun(s, a)
	case 2:
		g.helpBricks(s, a)
	case 3:
		g.helpPowers(s, a)
	case 4:
		g.helpItems(s, a)
	case 5:
		g.helpOrigins(s, a)
	case 6:
		g.helpMeta(s, a)
	}

	// page dots and arrows
	for i := 0; i < n; i++ {
		x := 960 + (float64(i)-float64(n-1)/2)*30
		c := rgba(90, 80, 120, 255)
		if i == pg {
			c = cGold
		}
		u.gem(s, x, 978, 6, 8, c, 1)
	}
	if u.button(s, "hprev", "‹  Previous", 560, 950, 240, 56, cAcc, true) {
		g.helpPage = (pg + n - 1) % n
	}
	if u.button(s, "hnext", "Next  ›", 1120, 950, 240, 56, cAcc, true) {
		g.helpPage = (pg + 1) % n
	}
	if u.button(s, "hback", "Back", 60, 990, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
	u.text(s, "Q / E  or  LB / RB  to turn pages", fReg, 15, 1860, 1050, ca(cDim, 0.6), alignRight)
}

// illus frames the left illustration area.
func (g *Game) illus(s *ebiten.Image, x, y, w, h float64) {
	u := g.ui
	u.frame(s, x, y, w, h, 0.9)
	fillPath(s, boxPath(u.X(x+14), u.Y(y+14), (w-28)*u.S, (h-28)*u.S, 10*u.S, false), rgba(6, 4, 14, 230), rgba(14, 9, 28, 230), vgrad(u.Y(y), u.Y(y+h)), false)
}

// textBlock writes a title + body pair and returns the next y.
func (g *Game) textBlock(s *ebiten.Image, title, body string, x, y, w, a float64) float64 {
	u := g.ui
	if title != "" {
		u.text(s, title, fBold, 22, x, y, ca(cGold, a), alignLeft)
		y += 34
	}
	return u.paragraph(s, body, fReg, 20, x, y, w, ca(rgba(222, 216, 236, 255), a), alignLeft) + 18
}

// ---------------------------------------------------------------- page 1

func (g *Game) helpControls(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	ix, iy, iw, ih := 120.0, 180.0, 820.0, 740.0
	g.illus(s, ix, iy, iw, ih)
	// a tiny arena: bricks along the top, the orb bouncing, the paddle following
	ax0, ay0, ax1, ay1 := ix+60, iy+60, ix+iw-60, iy+ih-80
	t := g.helpT
	vx, vy := 330.0, 410.0
	tri := func(v, lo, hi float64) float64 {
		span := hi - lo
		p := math.Mod(v, 2*span)
		if p < 0 {
			p += 2 * span
		}
		if p > span {
			p = 2*span - p
		}
		return lo + p
	}
	bx := tri(t*vx+120, ax0+14, ax1-14)
	by := tri(t*vy+200, ay0+140, ay1-40)
	cols := []color.RGBA{{255, 110, 90, 255}, {255, 200, 80, 255}, {120, 220, 140, 255}, {110, 180, 255, 255}, {200, 120, 255, 255}}
	bw := (ax1 - ax0) / 8
	for r := 0; r < 3; r++ {
		for c := 0; c < 8; c++ {
			x, y := ax0+float64(c)*bw+4, ay0+float64(r)*38
			col := cols[(r*3+c)%len(cols)]
			hit := math.Abs(bx-(x+bw/2)) < bw/2 && by < ay0+150 && r == 2
			if hit {
				col = lighten(col, 0.6)
			}
			fillPath(s, boxPath(u.X(x), u.Y(y), (bw-8)*S, 30*S, 6*S, false), lighten(col, 0.15), darken(col, 0.6), vgrad(u.Y(y), u.Y(y+30)), false)
		}
	}
	// paddle chases the orb a little behind
	px := math.Max(ax0+70, math.Min(ax1-70, bx+math.Sin(t*2)*30))
	py := ay1
	fillPath(s, boxPath(u.X(px-70), u.Y(py), 140*S, 18*S, 9*S, false), rgba(160, 130, 255, 255), rgba(70, 50, 150, 255), vgrad(u.Y(py), u.Y(py+18)), false)
	fillPath(s, boxPath(u.X(px-74), u.Y(py-2), 18*S, 22*S, 6*S, false), cGold, darken(cGold, 0.5), grad{}, false)
	fillPath(s, boxPath(u.X(px+56), u.Y(py-2), 18*S, 22*S, 6*S, false), cGold, darken(cGold, 0.5), grad{}, false)
	// orb + trail
	for k := 8; k > 0; k-- {
		tt := t - float64(k)*0.018
		ox, oy := tri(tt*vx+120, ax0+14, ax1-14), tri(tt*vy+200, ay0+140, ay1-40)
		drawSprite(s, sprSoft, u.X(ox-14), u.Y(oy-14), 28*S, 28*S, ca(rgba(170, 130, 255, 255), 0.06*float64(9-k)), true)
	}
	drawSprite(s, sprSoft, u.X(bx-30), u.Y(by-30), 60*S, 60*S, ca(rgba(170, 130, 255, 255), 0.5), true)
	fillPath(s, circlePath(u.X(bx), u.Y(by), 11*S), rgba(255, 255, 255, 255), rgba(200, 190, 255, 255), vgrad(u.Y(by-11), u.Y(by+11)), false)
	// angle guide under the paddle
	for i := -1; i <= 1; i++ {
		an := float64(i) * 0.9
		x1, y1 := px+math.Sin(an)*120, py-math.Cos(an)*120
		strokePath(s, linePath(u.X(px+float64(i)*50), u.Y(py-6), u.X(x1+float64(i)*50), u.Y(y1)), 2*S, ca(cGold, 0.35), true)
	}
	u.text(s, "edges angle the orb wide", fReg, 17, px, py+28, ca(cDim, 0.9), alignCenter)

	// controls
	tx := 1010.0
	y := 190.0
	row := func(keys []string, what string) {
		x := tx
		for i, k := range keys {
			if i > 0 {
				u.text(s, "/", fReg, 18, x+6, y+6, cDim, alignLeft)
				x += 22
			}
			x += u.keycap(s, k, x, y, a) + 6
		}
		u.text(s, what, fReg, 20, tx+330, y+5, ca(cWhite, a), alignLeft)
		y += 46
	}
	u.text(s, "KEYBOARD & MOUSE", fBold, 18, tx, y, ca(cGold, a), alignLeft)
	y += 36
	row([]string{"Mouse", "A D", "← →"}, "Move the paddle (Shift: faster)")
	row([]string{"Click", "Space"}, "Launch the orb")
	row([]string{"Right-click", "E"}, "Your origin's ability")
	row([]string{"Esc", "P"}, "Pause")
	y += 14
	u.text(s, "CONTROLLER", fBold, 18, tx, y, ca(cGold, a), alignLeft)
	y += 36
	row([]string{"Left stick", "D-pad"}, "Move the paddle")
	row([]string{"A"}, "Launch / select")
	row([]string{"X", "RB"}, "Ability")
	row([]string{"Start"}, "Pause")
	row([]string{"B"}, "Back")
	y += 10
	g.textBlock(s, "", "Losing an orb costs health. Health carries between chambers, so every save matters. Where the orb meets the paddle decides where it goes.", tx, y, 790, a)
}

// ---------------------------------------------------------------- page 2

func (g *Game) helpRun(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	ix, iy, iw, ih := 120.0, 180.0, 820.0, 740.0
	g.illus(s, ix, iy, iw, ih)
	// a small branching map
	type nd struct {
		x, y float64
		t    NodeType
	}
	nodes := []nd{
		{530, 260, NBoss},
		{400, 380, NRest}, {660, 380, NShop},
		{330, 500, NElite}, {530, 500, NEvent}, {730, 500, NTreasure},
		{400, 620, NCombat}, {660, 620, NCombat},
		{530, 760, NCombat},
	}
	edges := [][2]int{{8, 6}, {8, 7}, {6, 3}, {6, 4}, {7, 4}, {7, 5}, {3, 1}, {4, 1}, {4, 2}, {5, 2}, {1, 0}, {2, 0}}
	path := []int{8, 7, 4, 1, 0}
	onPath := func(i, j int) bool {
		for k := 0; k+1 < len(path); k++ {
			if (path[k] == i && path[k+1] == j) || (path[k] == j && path[k+1] == i) {
				return true
			}
		}
		return false
	}
	for _, e := range edges {
		p, q := nodes[e[0]], nodes[e[1]]
		c := ca(rgba(150, 140, 190, 255), 0.4)
		w := 2.5
		if onPath(e[0], e[1]) {
			c, w = ca(cGold, 0.9), 4
		}
		strokePath(s, linePath(u.X(p.x), u.Y(p.y), u.X(q.x), u.Y(q.y)), w*S, c, false)
	}
	// a mote travelling the chosen route
	f := math.Mod(g.helpT*0.35, 1) * float64(len(path)-1)
	k := int(f)
	if k < len(path)-1 {
		p, q := nodes[path[k]], nodes[path[k+1]]
		ff := f - float64(k)
		mx, my := p.x+(q.x-p.x)*ff, p.y+(q.y-p.y)*ff
		drawSprite(s, sprSoft, u.X(mx-24), u.Y(my-24), 48*S, 48*S, ca(cGold, 0.95), true)
	}
	for _, n := range nodes {
		rad := 32.0
		if n.t == NBoss {
			rad = 46
		}
		u.medallion(s, n.x, n.y, rad, nodeColors[n.t])
		g.nodeIcon(s, n.t, n.x, n.y, rad*1.1)
	}
	u.text(s, "ACT I  ·  II  ·  III", fSerif, 22, ix+iw/2, iy+ih-50, ca(cDim, 0.9), alignCenter)

	tx, y := 1010.0, 190.0
	y = g.textBlock(s, "", "Each act is a map of chambers. Pick a route from the bottom to the act's Guardian at the top: risk an elite for a relic, or detour past a shop and a campfire.", tx, y, 800, a)
	for i, nm := range nodeNames {
		u.medallion(s, tx+20, y+16, 16, nodeColors[i])
		g.nodeIcon(s, NodeType(i), tx+20, y+16, 18)
		u.text(s, nm, fBold, 20, tx+52, y+2, ca(cWhite, a), alignLeft)
		u.text(s, nodeDesc[i], fReg, 18, tx+180, y+4, ca(cDim, a), alignLeft)
		y += 50
	}
	g.textBlock(s, "", "Beat the third Guardian to win the story, then try Endless Mode.", tx, y+10, 800, a)
}

// ---------------------------------------------------------------- page 3

func (g *Game) helpBricks(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	kinds := []levelgen.Kind{levelgen.Basic, levelgen.Steel, levelgen.Armored, levelgen.Explosive, levelgen.Chain,
		levelgen.Split, levelgen.Regen, levelgen.Moving, levelgen.Shield, levelgen.Teleport, levelgen.Spawner,
		levelgen.Multiplier, levelgen.Crystal, levelgen.Mystery, levelgen.Elite}
	u.text(s, "Special bricks wear a badge so you can read them at a glance.", fReg, 21, 960, 180, ca(cDim, a), alignCenter)
	for i, k := range kinds {
		col, row := i%3, i/3
		x := 140 + float64(col)*560
		y := 240 + float64(row)*134
		bob := math.Sin(g.helpT*2+float64(i)) * 3
		// the brick itself
		bc := []color.RGBA{{240, 120, 90, 255}, {150, 156, 170, 255}}[0]
		if k == levelgen.Steel {
			bc = rgba(150, 156, 170, 255)
		} else if d, ok := badgeDefs[k]; ok {
			bc = darken(d.Col, 0.75)
		}
		fillPath(s, boxPath(u.X(x), u.Y(y+40), 110*S, 46*S, 8*S, false), lighten(bc, 0.15), darken(bc, 0.55), vgrad(u.Y(y+40), u.Y(y+86)), false)
		if k == levelgen.Basic {
			fillPath(s, boxPath(u.X(x), u.Y(y+58), 110*S, 8*S, 0, false), cGold, darken(cGold, 0.4), grad{}, false)
		}
		if img := badgeImgs[k]; img != nil {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			sc := 76 * S / float64(badgeRes)
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(u.X(x+55-38), u.Y(y-14+bob))
			op.ColorScale.ScaleAlpha(float32(a))
			s.DrawImage(img, op)
		}
		u.text(s, levelgen.KindNames[k], fBold, 22, x+132, y+22, ca(cWhite, a), alignLeft)
		u.paragraph(s, badgeBlurb[k], fReg, 18, x+132, y+52, 380, ca(cDim, a), alignLeft)
	}
}

// ---------------------------------------------------------------- page 4

func (g *Game) helpPowers(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	ix, iy, iw, ih := 120.0, 180.0, 620.0, 740.0
	g.illus(s, ix, iy, iw, ih)
	// capsules drifting down toward a paddle
	for i := 0; i < int(numPowers); i++ {
		p := powerDefs[i]
		f := math.Mod(g.helpT*0.22+float64(i)*0.137, 1)
		x := ix + 80 + math.Mod(float64(i)*173, iw-160)
		y := iy + 60 + f*(ih-180)
		al := math.Sin(f * math.Pi)
		fillPath(s, boxPath(u.X(x-34), u.Y(y-16), 68*S, 32*S, 16*S, false), ca(lighten(p.Col, 0.2), al), ca(darken(p.Col, 0.5), al), vgrad(u.Y(y-16), u.Y(y+16)), false)
		u.text(s, p.Letter, fBold, 20, x, y-13, ca(rgba(20, 16, 30, 255), al), alignCenter)
	}
	px := ix + iw/2 + math.Sin(g.helpT*1.3)*180
	fillPath(s, boxPath(u.X(px-70), u.Y(iy+ih-70), 140*S, 18*S, 9*S, false), rgba(160, 130, 255, 255), rgba(70, 50, 150, 255), grad{}, false)

	tx, y := 800.0, 190.0
	y = g.textBlock(s, "", "Broken bricks sometimes drop capsules. Catch them with the paddle. Most help; the red ones are curses, so dodge them.", tx, y, 980, a)
	for i := 0; i < int(numPowers); i++ {
		p := powerDefs[i]
		x := tx + float64(i%2)*500
		yy := y + float64(i/2)*92
		fillPath(s, boxPath(u.X(x), u.Y(yy), 60*S, 32*S, 16*S, false), lighten(p.Col, 0.2), darken(p.Col, 0.5), grad{}, false)
		u.text(s, p.Letter, fBold, 20, x+30, yy+3, rgba(20, 16, 30, 255), alignCenter)
		c := cWhite
		if !p.Good {
			c = rgba(255, 120, 120, 255)
		}
		u.text(s, p.Name, fBold, 20, x+76, yy+2, ca(c, a), alignLeft)
		u.paragraph(s, powerDesc[i], fReg, 17, x+76, yy+32, 400, ca(cDim, a), alignLeft)
	}
}

// ---------------------------------------------------------------- page 5

func (g *Game) helpItems(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	ix, iy, iw, ih := 120.0, 180.0, 820.0, 740.0
	g.illus(s, ix, iy, iw, ih)
	// three tiny cards fanning
	ids := []string{"kindling", "static", "excalibur"}
	for i, id := range ids {
		m := modByID[id]
		if m == nil {
			continue
		}
		rc := rarityColors[m.Rarity]
		x := ix + 110 + float64(i)*210
		y := iy + 60 + math.Sin(g.helpT*1.5+float64(i))*6
		fillPath(s, boxPath(u.X(x), u.Y(y), 180*S, 240*S, 14*S, false), mixc(rgba(30, 22, 50, 250), rc, 0.25), rgba(10, 8, 18, 250), vgrad(u.Y(y), u.Y(y+240)), false)
		strokePath(s, boxPath(u.X(x), u.Y(y), 180*S, 240*S, 14*S, false), 2.5*S, rc, false)
		u.drawEmblem(s, modEmblem(m), x+90, y+90, 96, 1)
		u.text(s, m.Name, fSerif, 20, x+90, y+160, cWhite, alignCenter)
		u.text(s, strings.ToUpper(rarityNames[m.Rarity]), fBold, 13, x+90, y+196, rc, alignCenter)
	}
	// fusion equation
	cy := iy + 470.0
	if f := fusionByID["plasma"]; f != nil {
		a1, b1 := modByID[f.A], modByID[f.B]
		pulse := 0.6 + 0.4*math.Sin(g.helpT*3)
		u.drawEmblem(s, modEmblem(a1), ix+170, cy, 110, 1)
		u.text(s, "+", fBold, 50, ix+290, cy-34, cDim, alignCenter)
		u.drawEmblem(s, modEmblem(b1), ix+410, cy, 110, 1)
		u.text(s, "=", fBold, 50, ix+530, cy-34, cDim, alignCenter)
		drawSprite(s, sprSoft, u.X(ix+650-110), u.Y(cy-110), 220*S, 220*S, ca(f.Col, 0.45*pulse), true)
		u.drawEmblem(s, fusionEmblem(f), ix+650, cy, 140, 1)
		u.text(s, f.Name, fRune, 30, ix+650, cy+84, f.Col, alignCenter)
		u.text(s, a1.Name+" + "+b1.Name, fReg, 17, ix+290, cy+76, cDim, alignCenter)
	}

	tx, y := 1010.0, 190.0
	y = g.textBlock(s, "Upgrades and relics", "After each chamber, choose one of three cards. Upgrades stack; relics are one-of-a-kind. Rarer cards are stronger: common, uncommon, rare, epic, legendary and mythic.", tx, y, 800, a)
	y = g.textBlock(s, "Synergies", "Every card has tags such as fire, storm or guard. Own three cards that share a tag and a SYNERGY awakens with a bonus of its own.", tx, y, 800, a)
	y = g.textBlock(s, "Fusions", "Some pairs forge into something new. Own both halves and they fuse on the spot into a powerful item with its own art and effect. Cards tell you when they complete a fusion.", tx, y, 800, a)
	g.textBlock(s, "Gold", "Gold buys cards, relics and healing in shops, and pays for gambles at events.", tx, y, 800, a)
}

// ---------------------------------------------------------------- page 6

func (g *Game) helpOrigins(s *ebiten.Image, a float64) {
	u := g.ui
	S := u.S
	n := len(archetypes)
	pw, ph := 236.0, 315.0
	x0 := 960 - (float64(n)*pw+float64(n-1)*12)/2
	for i := range archetypes {
		ar := &archetypes[i]
		ci := classInfo[ar.ID]
		x := x0 + float64(i)*(pw+12)
		y := 180.0
		fillPath(s, boxPath(u.X(x), u.Y(y), pw*S, ph*S, 12*S, false), rgba(10, 7, 18, 230), mixc(rgba(10, 7, 18, 230), ci.Col, 0.25), vgrad(u.Y(y), u.Y(y+ph)), false)
		if img := portrait(ar.ID); img != nil {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(pw*S/float64(img.Bounds().Dx()), ph*S/float64(img.Bounds().Dy()))
			op.GeoM.Translate(u.X(x), u.Y(y))
			op.ColorScale.ScaleAlpha(float32(a))
			s.DrawImage(img, op)
		}
		strokePath(s, boxPath(u.X(x), u.Y(y), pw*S, ph*S, 12*S, false), 2*S, ca(ci.Col, 0.8), false)
		u.text(s, ar.Name, fSerif, 22, x+pw/2, y+ph+10, ca(cWhite, a), alignCenter)
		u.text(s, strings.ToUpper(ar.Passive), fBold, 14, x+pw/2, y+ph+42, ca(lighten(ci.Col, 0.3), a), alignCenter)
		if ar.Ability != "" {
			u.text(s, "+ "+ar.Ability, fReg, 14, x+pw/2, y+ph+64, ca(cDim, a), alignCenter)
		}
	}
	tx, y := 300.0, 620.0
	y = g.textBlock(s, "", "Your origin decides how you play. Every origin has a passive that changes the rules, and some carry an active ability on right-click / E (X on a controller) that recharges over time.", tx, y, 1320, a)
	g.textBlock(s, "", "Four origins are locked at first. Unlock them with shards on the Choose your Origin screen. Each origin's passive also shows as a live meter in the corner of the screen.", tx, y, 1320, a)
}

// ---------------------------------------------------------------- page 7

func (g *Game) helpMeta(s *ebiten.Image, a float64) {
	u := g.ui
	items := []struct{ glyph, title, body string }{
		{"star", "Arcane Rank", "Every run earns experience. Ranking up unlocks new upgrades and relics into the pool, so the game keeps growing."},
		{"gem", "The Sanctum", "Shards earned on every run, won or lost, buy permanent boosts: more health, more gold, extra choices and more."},
		{"halo", "Ascension", "Win a run to unlock Ascension: ten tiers of harder runs with bigger rewards."},
		{"infinity", "Endless Mode", "Beat the story and the worlds loop forever, growing tougher each loop. Chase your best score on the local leaderboard."},
		{"book", "The Codex", "Every upgrade, relic and fusion you discover is recorded, along with your Endless records."},
		{"hourglass", "Daily Challenge & Seeds", "Everyone gets the same daily run. Every level shows its seed, so you can share and replay a run."},
	}
	for i, it := range items {
		col, row := i%2, i/2
		x := 160 + float64(col)*820
		y := 200 + float64(row)*230
		u.frame(s, x, y, 780, 200, 0.85)
		pulse := 0.5 + 0.5*math.Sin(g.helpT*2+float64(i))
		drawSprite(s, sprSoft, u.X(x+90-70), u.Y(y+100-70), 140*u.S, 140*u.S, ca(u.theme().Accent, 0.2+0.15*pulse), true)
		u.drawEmblem(s, glyphImage(it.glyph, u.theme().Accent), x+90, y+100, 100, a)
		u.text(s, it.title, fSerif, 26, x+170, y+30, ca(cGold, a), alignLeft)
		u.paragraph(s, it.body, fReg, 19, x+170, y+76, 580, ca(rgba(222, 216, 236, 255), a), alignLeft)
	}
}
