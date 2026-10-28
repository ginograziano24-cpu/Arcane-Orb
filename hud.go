package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

var nodeColors = []color.RGBA{{150, 150, 170, 255}, {200, 90, 255, 255}, {255, 210, 80, 255}, {90, 220, 255, 255}, {120, 230, 150, 255}, {255, 150, 150, 255}, {255, 80, 60, 255}}

// bar draws a polished meter: recessed well, damage trail, gloss, tick
// marks every `tick` units of `maxv`, and a themed rim.
func (u *UI) bar(dst *ebiten.Image, x, y, w, h, frac, trail float64, col color.RGBA, maxv, tick float64) {
	S := u.S
	th := u.theme()
	frac = clamp01(frac)
	trail = clamp01(math.Max(trail, frac))
	r := h / 2
	fillPath(dst, boxPath(u.X(x-3), u.Y(y-3), (w+6)*S, (h+6)*S, (r+3)*S, false), rgba(0, 0, 0, 190), rgba(0, 0, 0, 120), vgrad(u.Y(y-3), u.Y(y+h+3)), false)
	well := boxPath(u.X(x), u.Y(y), w*S, h*S, r*S, false)
	fillPath(dst, well, ca(darken(col, 0.14), 0.95), ca(darken(col, 0.3), 0.95), vgrad(u.Y(y), u.Y(y+h)), false)
	if trail > frac+0.002 {
		tw := math.Max(h, w*trail)
		fillPath(dst, boxPath(u.X(x), u.Y(y), tw*S, h*S, r*S, false), rgba(255, 240, 210, 235), rgba(255, 176, 120, 235), vgrad(u.Y(y), u.Y(y+h)), false)
	}
	if frac > 0 {
		fw := math.Max(h, w*frac)
		fillPath(dst, boxPath(u.X(x), u.Y(y), fw*S, h*S, r*S, false), lighten(col, 0.28), darken(col, 0.68), vgrad(u.Y(y), u.Y(y+h)), false)
		if fw > 12 {
			fillPath(dst, boxPath(u.X(x+5), u.Y(y+2.5), (fw-10)*S, h*0.38*S, h*0.19*S, false), rgba(255, 255, 255, 90), rgba(255, 255, 255, 10), vgrad(u.Y(y+2), u.Y(y+2+h*0.38)), false)
		}
	}
	if tick > 0 && maxv/tick < 40 {
		for v := tick; v < maxv-0.01; v += tick {
			tx := x + w*v/maxv
			fillRect(dst, u.X(tx)-0.75*S, u.Y(y+3), 1.5*S, (h-6)*S, rgba(0, 0, 0, 80))
		}
	}
	strokePath(dst, well, 1.6*S, ca(th.Edge, 0.8), false)
}

// medallion is a framed round badge used for map nodes and the legend.
func (u *UI) medallion(dst *ebiten.Image, x, y, rad float64, c color.RGBA) {
	S := u.S
	th := u.theme()
	cx, cy := u.X(x), u.Y(y)
	fillPath(dst, circlePath(cx, cy+4*S, (rad+6)*S), ca(cBlack, 0.4), ca(cBlack, 0.4), grad{}, false)
	fillPath(dst, circlePath(cx, cy, (rad+6)*S), lighten(th.Edge, 0.35), darken(th.Edge, 0.45), vgrad(cy-(rad+6)*S, cy+(rad+6)*S), false)
	fillPath(dst, circlePath(cx, cy, (rad+2)*S), rgba(12, 9, 22, 255), rgba(12, 9, 22, 255), grad{}, false)
	fillPath(dst, circlePath(cx, cy, rad*S), lighten(c, 0.3), darken(c, 0.5), vgrad(cy-rad*S, cy+rad*S), false)
	fillPath(dst, circlePath(cx, cy-rad*0.35*S, rad*0.6*S), rgba(255, 255, 255, 60), rgba(255, 255, 255, 0), vgrad(cy-rad*S, cy), false)
}

// drawHUD renders the in-level overlay.
func (g *Game) drawHUD(screen *ebiten.Image, L *Level) {
	if L.introT > 0 {
		g.drawBossIntro(screen, L)
		return
	}
	u := g.ui
	r := g.run
	dt := u.DT
	th := u.theme()
	S := u.S

	// ---- animated readouts
	hp := math.Max(0, r.HP)
	if g.hudLevel != L {
		g.hudLevel = L
		g.hudHP = hp
		g.hudGold = float64(r.Gold)
		g.lastGold = r.Gold
		g.hudBoss = -1
	}
	if g.hudHP < hp {
		g.hudHP = hp
	}
	g.hudHP += (hp - g.hudHP) * math.Min(1, dt*2.4)
	if r.Gold > g.lastGold {
		g.goldPop = 1
	}
	g.lastGold = r.Gold
	g.hudGold += (float64(r.Gold) - g.hudGold) * math.Min(1, dt*9)
	if math.Abs(g.hudGold-float64(r.Gold)) < 0.5 {
		g.hudGold = float64(r.Gold)
	}
	g.goldPop = math.Max(0, g.goldPop-dt*3)

	// on screens wider than 16:9 the corner groups hug the screen edges
	ox := u.OX
	defer func() { u.OX = ox }()
	edge := func(side int) {
		if g.settings.CenterHUD {
			return
		}
		switch side {
		case -1:
			u.OX = 0
		case 1:
			u.OX = u.SW - 1920*S
		default:
			u.OX = ox
		}
	}

	// ---- hero portrait and lives (top-left)
	edge(-1)
	g.drawHero(screen, L, 22, 18, 150, 186)
	g.drawLives(screen, L, 97, 224)
	// ---- vitals plate, beside the portrait
	const vx = 162.0
	u.frame(screen, 22+vx, 18, 440, 112, 0.92)
	frac := hp / r.Stats.MaxHP
	beat := 1.0
	if frac < 0.3 {
		beat = 1 + 0.14*math.Max(0, math.Sin(L.T*9))
		drawSprite(screen, sprSoft, u.X(vx+64-40), u.Y(56-40), 80*S, 80*S, ca(rgba(255, 60, 90, 255), 0.5*(beat-1)/0.14), true)
	}
	u.icon(screen, icHeart, vx+64, 57, 42*beat, 1)
	u.bar(screen, vx+100, 42, 338, 28, frac, g.hudHP/r.Stats.MaxHP, rgba(236, 62, 96, 255), r.Stats.MaxHP, 10)
	u.textOutlined(screen, fmt.Sprintf("%.0f / %.0f", hp, r.Stats.MaxHP), fBold, 18, vx+269, 44, cWhite, alignCenter)
	pop := g.goldPop
	u.icon(screen, icCoin, vx+64, 101, 30*(1+0.25*pop), 1)
	gs := 26 * (1 + 0.16*pop)
	u.textOutlined(screen, itoa(int(math.Round(g.hudGold))), fBold, gs, vx+92, 101-gs*0.64, mixc(cGold, cWhite, pop*0.6), alignLeft)
	if L.Barrier > 0 {
		x := vx + 432.0
		for i := 0; i < L.Barrier; i++ {
			u.icon(screen, icShield, x, 101, 26, 1)
			x -= 30
		}
		u.text(screen, "BARRIER", fBold, 14, x+8, 93, ca(rgba(190, 172, 255, 255), 0.9), alignRight)
	}

	// ---- location plaque (top-centre)
	edge(0)
	title := L.Biome.Name
	node := strings.ToUpper(nodeNames[L.Node])
	sub := L.Data.Name
	if L.Boss == nil {
		sub += "   ·   seed " + itoa(int(L.Data.Seed%100000000))
	}
	tW := u.measure(title, fSerif, 30)
	nw := u.measure(node, fBold, 14) + 26
	rowW := tW + 14 + nw
	pw := math.Max(440, math.Max(rowW, u.measure(sub, fReg, 16))+110)
	u.frame(screen, 960-pw/2, 16, pw, 82, 0.9)
	tx := 960 - rowW/2
	u.textOutlined(screen, title, fSerif, 30, tx, 22, cGold, alignLeft)
	nc := nodeColors[L.Node]
	chip := boxPath(u.X(tx+tW+14), u.Y(32), nw*S, 24*S, 12*S, false)
	fillPath(screen, chip, ca(nc, 0.32), ca(darken(nc, 0.5), 0.32), vgrad(u.Y(32), u.Y(56)), false)
	strokePath(screen, chip, 1.5*S, ca(nc, 0.9), false)
	u.text(screen, node, fBold, 14, tx+tW+14+nw/2, 35, lighten(nc, 0.4), alignCenter)
	u.text(screen, sub, fReg, 16, 960, 66, cDim, alignCenter)
	ty := 108.0
	if L.Boss == nil {
		ty = g.drawWaveStrip(screen, L, ty)
	}
	if L.Data.Objective == levelgen.ObjCrystals && L.Boss == nil {
		s := fmt.Sprintf("Shatter the crystals   ·   %d left", L.Crystals)
		w := u.measure(s, fBold, 18) + 72
		u.frame(screen, 960-w/2, ty, w, 42, 0.88)
		u.icon(screen, icShard, 960-w/2+28, ty+21, 24, 1)
		u.text(screen, s, fBold, 18, 960-w/2+48, ty+9, rgba(176, 222, 255, 255), alignLeft)
		g.drawOmenTag(screen, L, ty+48)
	} else {
		g.drawOmenTag(screen, L, ty)
	}

	// ---- combo (top-right)
	edge(1)
	if L.Combo >= 3 {
		g.comboShown = L.Combo
	}
	ck := u.anim("hudcombo", L.Combo >= 3, 10)
	if ck > 0.01 {
		heat := math.Min(1, float64(g.comboShown)/40)
		c := mixc(rgba(255, 232, 140, 255), rgba(255, 84, 60, 255), heat)
		w := 300.0
		x := 1898 - w + (1-ck)*80
		u.frame(screen, x, 18, w, 104, 0.92*ck)
		u.text(screen, "COMBO", fBold, 17, x+28, 34, ca(lighten(th.Accent, 0.2), ck), alignLeft)
		pulse := 1 + 0.07*math.Sin(L.T*14)*heat
		size := 56 * pulse
		num := itoa(g.comboShown)
		nW := u.measure(num, fTitle, size)
		u.textOutlined(screen, num, fTitle, size, x+w-26, 24-(size-56)*0.6, ca(c, ck), alignRight)
		u.textOutlined(screen, "×", fBold, 32, x+w-34-nW, 46, ca(c, ck), alignRight)
		bw := w - 56
		fillPath(screen, boxPath(u.X(x+28), u.Y(98), bw*S, 7*S, 3.5*S, false), rgba(0, 0, 0, uint8(150*ck)), rgba(0, 0, 0, uint8(150*ck)), grad{}, false)
		if L.Combo >= 3 {
			fw := math.Max(7, bw*clamp01(L.comboT/3))
			fillPath(screen, boxPath(u.X(x+28), u.Y(98), fw*S, 7*S, 3.5*S, false), ca(lighten(c, 0.3), ck), ca(darken(c, 0.6), ck), vgrad(u.Y(98), u.Y(105)), false)
		}
	}
	if L.Frenzy > 0 {
		s := fmt.Sprintf("FRENZY ×2   %.0fs", L.Frenzy)
		w := u.measure(s, fBold, 20) + 74
		y := 18.0
		if ck > 0.01 {
			y = 134
		}
		u.frame(screen, 1898-w, y, w, 46, 0.9)
		u.icon(screen, icBolt, 1898-w+28, y+23, 26, 1)
		u.text(screen, s, fBold, 20, 1898-w+50, y+10, rgba(255, 130, 222, 255), alignLeft)
	}

	// ---- boss bar
	edge(0)
	if L.Boss != nil {
		bb := L.Boss
		if g.hudBoss < bb.HP || g.hudBoss > bb.MaxHP {
			g.hudBoss = bb.HP
		}
		g.hudBoss += (bb.HP - g.hudBoss) * math.Min(1, dt*2.4)
		y := 110.0
		u.frame(screen, 470, y, 980, 80, 0.92)
		u.textOutlined(screen, bb.Name, fRune, 28, 960, y+8, rgba(255, 216, 194, 255), alignCenter)
		col := mixc(rgba(226, 50, 60, 255), cWhite, bb.Hurt*0.5)
		u.bar(screen, 504, y+46, 912, 20, bb.HP/bb.MaxHP, g.hudBoss/bb.MaxHP, col, 10, 1)
		u.gem(screen, 488, y+56, 8, 11, th.Edge, 1)
		u.gem(screen, 1432, y+56, 8, 11, th.Edge, 1)
	}

	// ---- class ability (bottom-left, above power-ups)
	edge(-1)
	if r.hasAbility() {
		g.drawAbility(screen, L)
	}
	g.drawPassive(screen, L)

	// ---- active power-ups (bottom-left)
	x := 24.0
	for p := Power(0); p < numPowers; p++ {
		if L.Timers[p] <= 0 || powerDefs[p].Dur == 0 {
			continue
		}
		d := powerDefs[p]
		k := clamp01(L.Timers[p] / d.Dur)
		w := u.measure(d.Name, fBold, 17) + 76
		y := 1014.0
		u.frame(screen, x, y, w, 48, 0.9)
		cx, cy := x+26, y+24
		strokePath(screen, circlePath(u.X(cx), u.Y(cy), 17*S), 3*S, rgba(0, 0, 0, 140), false)
		if k > 0.01 {
			strokePath(screen, arcPath(u.X(cx), u.Y(cy), 17*S, -math.Pi/2, -math.Pi/2+2*math.Pi*k), 3*S, lighten(d.Col, 0.2), false)
		}
		fillPath(screen, circlePath(u.X(cx), u.Y(cy), 13*S), lighten(d.Col, 0.35), darken(d.Col, 0.55), vgrad(u.Y(cy-13), u.Y(cy+13)), false)
		u.text(screen, d.Letter, fBold, 16, cx, cy-10, rgba(20, 16, 30, 255), alignCenter)
		tc := cWhite
		if !d.Good {
			tc = cRed
		}
		u.text(screen, d.Name, fBold, 17, x+52, y+13, tc, alignLeft)
		x += w + 10
	}

	// ---- relics (bottom-right)
	edge(1)
	rx := 1898.0
	for i := len(r.Order) - 1; i >= 0; i-- {
		m := modByID[r.Order[i]]
		if m == nil || !m.Relic {
			continue
		}
		rx -= 48
		if rx < 1240 {
			break
		}
		cx, cy := rx+22, 1038.0
		rc := rarityColors[m.Rarity]
		hex := ngonPath(u.X(cx), u.Y(cy), 22*S, 6, math.Pi/6)
		fillPath(screen, ngonPath(u.X(cx), u.Y(cy+4), 22*S, 6, math.Pi/6), ca(cBlack, 0.4), ca(cBlack, 0.4), grad{}, false)
		strokePath(screen, hex, 3.5*S, ca(cBlack, 0.55), false)
		fillPath(screen, hex, darken(rc, 0.5), darken(rc, 0.2), vgrad(u.Y(cy-22), u.Y(cy+22)), false)
		strokePath(screen, hex, 1.6*S, ca(th.Edge, 0.95), false)
		u.drawEmblem(screen, modEmblem(m), cx, cy, 36, 1)
	}

	g.drawScrolls(screen, L)

	// ---- launch prompt
	edge(0)
	if L.anyStuck() && !L.Cleared {
		a := 0.75 + 0.25*math.Sin(L.T*4)
		t2, kc := "or click to launch", "SPACE"
		if g.padActive {
			t2, kc = "to launch", "A"
		}
		w1 := u.measure(kc, fBold, 16) + 28
		w2 := u.measure(t2, fBold, 22)
		x := 960 - (w1+14+w2)/2
		u.keycap(screen, kc, x, 960, a)
		u.textOutlined(screen, t2, fBold, 22, x+w1+14, 962, ca(cWhite, a), alignLeft)
	}

	// ---- chamber cleared
	if L.Cleared && L.clearShown() > 0 {
		k := ease(L.clearShown() * 2.5)
		w, h := 760.0, 236.0
		y := 372 + (1-k)*36
		u.frame(screen, 960-w/2, y, w, h, 0.95*k)
		u.headingW(screen, "CHAMBER CLEARED", 58, y+28, rgba(255, 222, 128, 255), k, w/2-24)
		u.iconLabel(screen, icCoin, fmt.Sprintf("+%d gold", L.GoldEarned), fBold, 28, 960, y+116, ca(cGold, k), alignCenter)
		if L.clearShown() > 1.2 {
			u.text(screen, "Click to continue", fReg, 20, 960, y+176, ca(cDim, k*(0.6+0.4*math.Sin(L.T*4))), alignCenter)
		}
	}
	if g.settings.ShowFPS {
		u.text(screen, fmt.Sprintf("%.0f FPS", ebiten.ActualFPS()), fReg, 16, 1894, 990, cDim, alignRight)
	}
}

// drawAbility renders the class ability medallion with its cooldown ring.
func (g *Game) drawAbility(screen *ebiten.Image, L *Level) {
	u := g.ui
	r := g.run
	S := u.S
	th := u.theme()
	a := r.Arch
	col, gl := rgba(255, 140, 60, 255), "flame"
	if a.ID == "storm" {
		col, gl = rgba(140, 200, 255, 255), "bolt"
	}
	frac := r.abilityFrac()
	ready := frac == 0
	cx, cy, rad := 74.0, 934.0, 40.0
	w := 250.0
	u.frame(screen, 22, cy-46, w, 92, 0.9)
	if ready {
		p := 0.5 + 0.5*math.Sin(L.T*5)
		drawSprite(screen, sprSoft, u.X(cx-rad*2), u.Y(cy-rad*2), rad*4*S, rad*4*S, ca(col, 0.25+0.2*p+0.5*L.abilReadyT), true)
	}
	if L.abilFlash > 0 {
		drawSprite(screen, sprSoft, u.X(cx-rad*3), u.Y(cy-rad*3), rad*6*S, rad*6*S, ca(cWhite, 0.6*L.abilFlash), true)
	}
	fillPath(screen, circlePath(u.X(cx), u.Y(cy+3), (rad+4)*S), ca(cBlack, 0.45), ca(cBlack, 0.45), grad{}, false)
	fillPath(screen, circlePath(u.X(cx), u.Y(cy), (rad+4)*S), lighten(th.Edge, 0.3), darken(th.Edge, 0.45), vgrad(u.Y(cy-rad-4), u.Y(cy+rad+4)), false)
	inner := darken(col, 0.3)
	if !ready {
		inner = darken(col, 0.12)
	}
	fillPath(screen, circlePath(u.X(cx), u.Y(cy), rad*S), inner, darken(inner, 0.5), vgrad(u.Y(cy-rad), u.Y(cy+rad)), false)
	// cooldown: dark sweep shrinking clockwise, bright ring filling
	strokePath(screen, circlePath(u.X(cx), u.Y(cy), (rad-3)*S), 5*S, rgba(0, 0, 0, 150), false)
	prog := 1 - frac
	if prog > 0.01 {
		strokePath(screen, arcPath(u.X(cx), u.Y(cy), (rad-3)*S, -math.Pi/2, -math.Pi/2+2*math.Pi*prog), 5*S, lighten(col, 0.25), false)
	}
	ea := 1.0
	if !ready {
		ea = 0.45
	}
	u.drawEmblem(screen, glyphImage(gl, col), cx, cy, 52, ea)
	if !ready {
		u.textOutlined(screen, fmt.Sprintf("%.0f", math.Ceil(r.AbilT)), fTitle, 30, cx, cy-20, cWhite, alignCenter)
	}
	tx := cx + rad + 16
	u.text(screen, strings.ToUpper(a.Ability), fBold, 18, tx, cy-30, lighten(col, 0.4), alignLeft)
	if ready {
		if g.padActive {
			kw := u.keycap(screen, "X", tx, cy-2, 1)
			u.text(screen, "or RB", fReg, 15, tx+kw+8, cy+6, cDim, alignLeft)
		} else {
			kw := u.keycap(screen, "E", tx, cy-2, 1)
			u.text(screen, "or right-click", fReg, 15, tx+kw+8, cy+6, cDim, alignLeft)
		}
	} else {
		u.text(screen, "recharging", fReg, 15, tx, cy+6, cDim, alignLeft)
	}
}

// drawPassive shows the class passive's live state as a compact plate.
func (g *Game) drawPassive(screen *ebiten.Image, L *Level) {
	u := g.ui
	r := g.run
	st := &r.Stats
	c := &L.cls
	var label, val, gl string
	var frac float64
	var col color.RGBA
	ready := false
	switch {
	case st.ClassParry:
		label, gl, col = "PARRY", "shield", rgba(255, 215, 130, 255)
		frac = 1 - clamp01(c.parryCD/0.8)
		ready = c.parryCD <= 0
		val = "centre hit"
	case st.ClassRage:
		label, gl, col = "RAGE", "sword", rgba(255, 70, 50, 255)
		frac = float64(c.rage) / 20
		val = fmt.Sprintf("+%d%%", c.rage*4)
		ready = c.rage == 20
	case st.ClassAttune:
		col = attuneCols[c.attuneEl]
		label, gl = attuneNames[c.attuneEl], []string{"flame", "snow", "bolt", "orb"}[c.attuneEl]
		frac = 1 - c.attuneT/8
		val = fmt.Sprintf("%.0fs", 8-c.attuneT)
	case st.ClassOverclock:
		label, gl, col = "OVERCLOCK", "gear", rgba(255, 160, 64, 255)
		frac = clamp01(c.stillT / 1.5)
		ready = c.overclock
		val = "hold still"
		if ready {
			frac, val = 1, "READY"
		}
	case st.ClassHighRoller:
		label, gl, col = "HIGH ROLLER", "coin", rgba(255, 210, 90, 255)
		b := highRollerBonus(r.Gold)
		frac = b / 0.6
		val = fmt.Sprintf("+%.0f%%", b*100)
		ready = frac >= 1
	case st.ClassIgnite > 0:
		label, gl, col = "IGNITE", "flame", rgba(255, 106, 26, 255)
		frac = float64(L.igniteN%st.ClassIgnite) / float64(st.ClassIgnite-1)
		val = fmt.Sprintf("%d / %d", L.igniteN%st.ClassIgnite+1, st.ClassIgnite)
	case st.ClassStatic:
		label, gl, col = "STATIC", "bolt", rgba(120, 190, 255, 255)
		frac = c.static / 100
		val = fmt.Sprintf("%.0f%%", c.static)
		ready = c.static >= 100
		if ready {
			val = "bounce!"
		}
	default:
		return
	}
	y := 934.0
	if r.hasAbility() {
		y = 828
	}
	w := 250.0
	u.frame(screen, 22, y-26, w, 52, 0.88)
	if ready {
		p := 0.5 + 0.5*math.Sin(L.T*6)
		drawSprite(screen, sprSoft, u.X(22-20), u.Y(y-40), (w+40)*u.S, 80*u.S, ca(col, 0.15+0.15*p), true)
	}
	u.drawEmblem(screen, glyphImage(gl, col), 50, y, 34, 1)
	u.text(screen, label, fBold, 15, 74, y-19, lighten(col, 0.35), alignLeft)
	u.text(screen, val, fBold, 14, 22+w-16, y-19, cWhite, alignRight)
	u.meter(screen, 74, y+4, w-92, frac, col)
}

// drawBossIntro is the cinematic overlay while a Guardian assembles.
func (g *Game) drawBossIntro(screen *ebiten.Image, L *Level) {
	u := g.ui
	S := u.S
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	k := 1 - L.introT/L.introMax
	out := clamp01(L.introT / 0.45)
	bh := 118 * ease(clamp01(k*4)) * out
	fillRect(screen, 0, 0, sw, bh*S+u.Y(0), rgba(0, 0, 0, 255))
	fillRect(screen, 0, u.Y(1080-bh), sw, sh-u.Y(1080-bh), rgba(0, 0, 0, 255))
	a := ease(clamp01((k-0.32)*3.5)) * out
	if a > 0.01 {
		acc := lighten(L.Biome.Trim, 0.25)
		g.drawSigil(screen, 960, 800, 170+20*k, L.T*0.2, acc, 0.6*a)
		drawSprite(screen, sprSoft, u.X(960-520), u.Y(800-150), 1040*S, 300*S, ca(rgba(0, 0, 0, 255), 0.55*a), false)
		act := g.run.AI()
		cap := "GUARDIAN OF " + strings.ToUpper(acts[act].Name)
		u.text(screen, cap, fSerif, 24, 960, 718-(1-a)*10, ca(acc, a), alignCenter)
		u.headingW(screen, L.Boss.Name, 86, 752+(1-a)*14, rgba(255, 226, 196, 255), a, 940)
	}
	if k > 0.1 && out > 0.5 {
		if !hideHUD {
			u.text(screen, "Click to skip", fSerif, 18, 1880, 1080-bh+(bh-30)/2, ca(cDim, 0.6*out), alignRight)
		}
	}
}
