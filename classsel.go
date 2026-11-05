package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Class select: a list of origins on the left, the selected character's
// portrait in the middle, and its attributes and ability on the right.
// Portraits are rendered offline by tools/portraits.

//go:embed assets/portraits
var portraitFS embed.FS

var portraitCache = map[string]*ebiten.Image{}

func portrait(id string) *ebiten.Image {
	if img, ok := portraitCache[id]; ok {
		return img
	}
	img := loadImage("portrait/" + id).img
	portraitCache[id] = img
	return img
}

// composePortrait joins a portrait's colour (JPEG) and alpha (greyscale
// PNG), which are stored separately.
func composePortrait(id string) image.Image {
	c := decodeEmbedded(portraitFS.ReadFile("assets/portraits/" + id + ".jpg"))
	a := decodeEmbedded(portraitFS.ReadFile("assets/portraits/" + id + "_a.png"))
	if c == nil || a == nil {
		return nil
	}
	b := c.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := c.At(x, y).RGBA()
			av, _, _, _ := a.At(x, y).RGBA()
			i := out.PixOffset(x, y)
			out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = uint8(r>>8), uint8(g>>8), uint8(bl>>8), uint8(av>>8)
		}
	}
	return out
}

// classStyle is how an origin plays: two short tags and a difficulty (1-3).
var classStyle = map[string]struct {
	Tags [2]string
	Diff int
}{
	"guardian":  {[2]string{"Defence", "Control"}, 1},
	"berserker": {[2]string{"Aggression", "Momentum"}, 2},
	"arcane":    {[2]string{"Elements", "Area"}, 2},
	"engineer":  {[2]string{"Precision", "Patience"}, 2},
	"lucky":     {[2]string{"Economy", "Chance"}, 3},
	"pyro":      {[2]string{"Burning", "Area"}, 2},
	"storm":     {[2]string{"Chains", "Movement"}, 3},
}

// classQuote is each origin's creed, shown when it steps into a new world.
var classQuote = map[string]string{
	"guardian":  "Hold the line. Let nothing through.",
	"berserker": "Break it. Then break what stood behind it.",
	"arcane":    "Every element answers, if you ask it properly.",
	"engineer":  "A still hand and a clear shot. That's all it takes.",
	"lucky":     "The house always loses. Eventually.",
	"pyro":      "Everything burns. Some things just need encouragement.",
	"storm":     "The sky remembers me.",
}

var diffNames = []string{"", "Straightforward", "Rewards mastery", "For daring hands"}

// classInfo is the presentation data for each archetype.
var classInfo = map[string]struct {
	Lore  string
	Col   color.RGBA
	Glyph string
}{
	"guardian":  {"A sworn shield of the old Wardens. Where others fall, the Guardian endures.", rgba(255, 210, 122, 255), "shield"},
	"berserker": {"Raised in the forge-pits of the Deep Realms. Strikes first, and strikes hardest.", rgba(255, 70, 50, 255), "sword"},
	"arcane":    {"A scholar of the Weavers' lost art, attuned to whichever element answers.", rgba(150, 110, 255, 255), "orb"},
	"engineer":  {"A tinkerer from the Crucible's ruins who taught old machines to fire light.", rgba(255, 160, 64, 255), "gear"},
	"lucky":     {"Fortune favours the bold, and the Gambler has never once been timid.", rgba(70, 214, 154, 255), "coin"},
	"pyro":      {"Keeper of the Crucible's last ember. Everything burns, eventually.", rgba(255, 106, 26, 255), "flame"},
	"storm":     {"Born the night the sky split open. The lightning still remembers.", rgba(94, 196, 255, 255), "bolt"},
}

func (g *Game) drawArch(s *ebiten.Image) {
	u := g.ui
	S := u.S
	n := len(archetypes)
	if g.archSel < 0 || g.archSel >= n {
		g.archSel = 0
	}
	// keyboard
	defer func() { g.archActivate = false }()
	a := &archetypes[g.archSel]
	info := classInfo[a.ID]
	col := info.Col
	locked := a.Cost > 0 && !g.save.Unlocked[a.ID]

	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	fillRect(s, 0, 0, sw, sh, rgba(3, 2, 6, 170))
	// class-tinted light behind the figure
	g.archGlow = mixc(g.archGlow, col, math.Min(1, u.DT*4))
	if g.archGlow.A == 0 {
		g.archGlow = col
	}
	gc := g.archGlow
	drawSprite(s, sprSoft, u.X(960-560), u.Y(60), 1120*S, 1000*S, ca(gc, 0.22), true)
	drawSprite(s, sprSoft, u.X(960-380), u.Y(860), 760*S, 200*S, ca(gc, 0.3), true)

	// ---- portrait with cross-fade
	if g.archShown != g.archSel {
		g.archPrev = g.archShown
		g.archShown = g.archSel
		g.archFade = 0
	}
	g.archFade = math.Min(1, g.archFade+u.DT*3.5)
	g.archLockT = math.Max(0, g.archLockT-u.DT*1.8)
	ph := 1010.0
	pw := ph * 0.75
	px, py := 960-pw/2, 1080-ph+6
	// breathing (scale about the feet) and a little mouse parallax
	br := math.Sin(g.stateT * 1.7)
	g.archPar[0] += ((u.MX-960)/960*-12 - g.archPar[0]) * math.Min(1, u.DT*3)
	g.archPar[1] += ((u.MY-540)/540*-7 - g.archPar[1]) * math.Min(1, u.DT*3)
	drawP := func(id string, alpha, dy float64) {
		img := portrait(id)
		if img == nil || alpha <= 0 {
			return
		}
		punch := 0.0
		if id == a.ID {
			lt := g.archLockT
			punch = 0.035 * lt * lt
		}
		xf := portraitXform{px, py, pw, ph, 1 + 0.003*br + punch, 1 + 0.009*br + punch, g.archPar[0], g.archPar[1] + dy}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(pw*S/float64(img.Bounds().Dx()), ph*S/float64(img.Bounds().Dy()))
		op.GeoM.Translate(-pw*S/2, -ph*S)
		op.GeoM.Scale(xf.sx, xf.sy)
		op.GeoM.Translate(u.X(px+pw/2+xf.dx), u.Y(py+ph+xf.dy))
		op.ColorScale.ScaleAlpha(float32(alpha))
		dim := 1.0
		if locked && id == a.ID {
			op.ColorScale.Scale(0.55, 0.55, 0.55, 1)
			dim = 0.5
		}
		s.DrawImage(img, op)
		g.drawPortraitFX(s, id, xf, alpha*dim)
	}
	if g.archFade < 1 && g.archPrev >= 0 && g.archPrev < n {
		drawP(archetypes[g.archPrev].ID, 1-ease(g.archFade), 0)
	}
	drawP(a.ID, ease(g.archFade), (1-ease(g.archFade))*14)
	if lt := g.archLockT; lt > 0 {
		// lock-in: a flash through the figure and a ring of light at its feet
		e := 1 - lt
		drawSprite(s, sprSoft, u.X(960-420), u.Y(140), 840*S, 900*S, ca(lighten(col, 0.4), 0.35*lt*lt), true)
		rr := 120 + 520*e
		drawSprite(s, sprRing, u.X(960-rr), u.Y(1000-rr*0.28), 2*rr*S, 2*rr*0.28*S, ca(lighten(col, 0.3), 0.9*lt), true)
		rr2 := 80 + 360*e
		drawSprite(s, sprRing, u.X(960-rr2), u.Y(1000-rr2*0.28), 2*rr2*S, 2*rr2*0.28*S, ca(cWhite, 0.6*lt), true)
		for i := 0; i < 18; i++ {
			f := float64(i)
			an := f / 18 * 2 * math.Pi
			d := 60 + 420*e
			x := 960 + math.Cos(an)*d
			y := 980 - math.Abs(math.Sin(an))*d*0.9 - 120*e
			sz := (6 + 6*math.Mod(f*0.37, 1)) * S
			drawSprite(s, sprSoft, u.X(x)-sz, u.Y(y)-sz, 2*sz, 2*sz, ca(lighten(col, 0.5), lt), true)
		}
	}
	// live embers drifting up through the portrait
	for i := 0; i < 46; i++ {
		f := float64(i)
		sp := 18 + 40*math.Mod(f*0.618, 1)
		ey := 1080 - math.Mod(g.stateT*sp+f*97, 1000)
		ex := 960 + (math.Mod(f*0.377, 1)-0.5)*720 + math.Sin(g.stateT*0.8+f)*18
		life := (1080 - ey) / 1000
		al := math.Sin(life*math.Pi) * (0.35 + 0.65*math.Mod(f*0.73, 1))
		sz := (3 + 6*math.Mod(f*0.53, 1)) * S
		drawSprite(s, sprSoft, u.X(ex)-sz, u.Y(ey)-sz, sz*2, sz*2, ca(lighten(gc, 0.3), al*0.8), true)
	}

	// ---- header
	sub := "Story"
	if g.endlessMode {
		sub = "Endless Mode"
	}
	if g.daily {
		sub = "Daily Challenge  ·  " + dailyCode()
	} else if g.seedText != "" {
		sub = "Seeded run  ·  " + g.seedText
	}
	if c := challengeByID[g.challengeSel]; c != nil {
		sub = "Challenge  ·  " + c.Name
	}
	u.textOutlined(s, "Choose your Origin", fRune, 46, 70, 40, rgba(232, 214, 170, 255), alignLeft)
	u.text(s, strings.ToUpper(sub), fBold, 17, 72, 104, ca(lighten(u.theme().Accent, 0.3), 0.9), alignLeft)
	u.glowLine(s, 70, 520, 136, 1.5, cGold, 0.7)

	// ---- origin list (left)
	ly := 170.0
	for i := range archetypes {
		c := &archetypes[i]
		ci := classInfo[c.ID]
		y := ly + float64(i)*86
		// pointer: hovering only highlights, a click locks the origin in;
		// pad/keys: focus selects, A begins
		f, clk := u.navReg("arch"+c.ID, 60, y, 470, 78, false)
		mh := !u.nav.mode && u.hover(60, y, 470, 78)
		if mh && u.lastHover != "arch"+c.ID {
			g.snd.Play("hover")
		}
		if mh || f {
			u.hoverID = "arch" + c.ID
		}
		if f && g.archSel != i {
			g.archSel = i
			g.archLockT = 1
			g.archLockRow = i
		}
		if mh && u.Click {
			if g.archSel != i {
				g.archSel = i
				g.snd.Play("pick.2." + itoa(g.snd.Key))
			} else {
				g.snd.Play("click")
			}
			g.archLockT = 1
			g.archLockRow = i
		}
		if clk {
			g.archActivate = true
		}
		hk := u.anim("archhov"+c.ID, mh && g.archSel != i, 14)
		if hk > 0.01 {
			fillPath(s, boxPath(u.X(60), u.Y(y), 470*S, 78*S, 6*S, false), ca(ci.Col, 0.09*hk), ca(ci.Col, 0.0), hgrad(u.X(60), u.X(530)), false)
			strokePath(s, boxPath(u.X(60), u.Y(y), 470*S, 78*S, 6*S, false), 1.2*S, ca(ci.Col, 0.35*hk), false)
		}
		k := u.anim("archrow"+c.ID, g.archSel == i, 12)
		if g.archLockRow == i && g.archLockT > 0 {
			// the row flashes and a sweep of light runs across it
			lt := g.archLockT
			fillPath(s, boxPath(u.X(60), u.Y(y), 470*S, 78*S, 6*S, false), ca(lighten(ci.Col, 0.5), 0.5*lt*lt), ca(ci.Col, 0.1*lt), hgrad(u.X(60), u.X(530)), true)
			sx := 60 + 470*(1-lt)
			drawSprite(s, sprSoft, u.X(sx-60), u.Y(y-10), 120*S, 98*S, ca(cWhite, 0.5*lt), true)
		}
		if g.archSel == i {
			// a lit check seal marks the locked origin
			cx2, cy2 := 506.0, y+39
			p := 1 + 0.35*g.archLockT*g.archLockT
			drawSprite(s, sprSoft, u.X(cx2-22*p), u.Y(cy2-22*p), 44*p*S, 44*p*S, ca(ci.Col, 0.55), true)
			fillPath(s, circlePath(u.X(cx2), u.Y(cy2), 12*p*S), lighten(ci.Col, 0.35), darken(ci.Col, 0.2), vgrad(u.Y(cy2-12), u.Y(cy2+12)), false)
			strokeLine(s, u.X(cx2-5*p), u.Y(cy2), u.X(cx2-1*p), u.Y(cy2+4*p), 2.4*S, cWhite)
			strokeLine(s, u.X(cx2-1*p), u.Y(cy2+4*p), u.X(cx2+6*p), u.Y(cy2-4*p), 2.4*S, cWhite)
		}
		if k > 0.01 {
			fillPath(s, boxPath(u.X(60), u.Y(y), 470*S, 78*S, 6*S, false), ca(ci.Col, 0.2*k), ca(ci.Col, 0.02*k), hgrad(u.X(60), u.X(530)), false)
			fillRectA(s, u.X(60), u.Y(y+8), 4*S, 62*S, lighten(ci.Col, 0.3), k)
			u.glowLine(s, 60, 60+470*k, y+77, 1, ca(ci.Col, 0.8), k)
		}
		cl := c.Cost > 0 && !g.save.Unlocked[c.ID]
		ea := 0.55 + 0.45*k
		if cl {
			ea *= 0.5
		}
		u.drawEmblem(s, glyphImage(ci.Glyph, ci.Col), 104, y+39, 52, ea)
		nc := mixc(rgba(200, 194, 214, 255), cWhite, k)
		if cl {
			nc = rgba(130, 124, 146, 255)
		}
		u.text(s, c.Name, fSerif, 30, 146+10*k, y+8, nc, alignLeft)
		tag := c.Passive
		if c.Ability != "" {
			tag += "  ·  " + c.Ability
		}
		if cl {
			tag = fmt.Sprintf("Locked · %d shards", c.Cost)
		}
		tc := ca(cDim, 0.85)
		if cl {
			tc = ca(rgba(200, 180, 255, 255), 0.8)
		}
		u.text(s, tag, fReg, 16, 148+10*k, y+48, tc, alignLeft)
		if lv, _ := g.save.masteryOf(c.ID); lv > 1 {
			mc := ci.Col
			if fc, ok := frameCol(lv); ok {
				mc = fc
			}
			chip := boxPath(u.X(470), u.Y(y+22), 50*S, 30*S, 15*S, false)
			fillPath(s, chip, ca(darken(mc, 0.5), 0.5*ea+0.2), ca(darken(mc, 0.75), 0.5*ea+0.2), grad{}, false)
			strokePath(s, chip, 1.4*S, ca(mc, ea), false)
			u.text(s, itoa(lv), fBold, 18, 495, y+26, ca(lighten(mc, 0.4), ea+0.2), alignCenter)
		}
	}
	g.drawMasteryPanel(s, a, 60, 776, 470)

	// ---- detail panel (right)
	rx, rw := 1340.0, 520.0
	ry := 60.0
	u.frame(s, rx, ry, rw, 820, 0.88)
	cx := rx + 36
	u.textOutlined(s, a.Name, fRune, 50, cx, ry+24, lighten(col, 0.35), alignLeft)
	// playstyle chips and difficulty pips under the name
	if cs, ok := classStyle[a.ID]; ok {
		tx := cx
		for _, tg := range cs.Tags {
			lbl := strings.ToUpper(tr(tg))
			tw := u.measure(lbl, fBold, 14) + 22
			chip := boxPath(u.X(tx), u.Y(ry+86), tw*S, 26*S, 13*S, false)
			fillPath(s, chip, ca(darken(col, 0.35), 0.75), ca(darken(col, 0.6), 0.75), vgrad(u.Y(ry+86), u.Y(ry+112)), false)
			strokePath(s, chip, 1.2*S, ca(col, 0.8), false)
			u.text(s, lbl, fBold, 14, tx+tw/2, ry+90, lighten(col, 0.55), alignCenter)
			tx += tw + 8
		}
		dx := rx + rw - 36 - 3*20
		for i := 0; i < 3; i++ {
			on := i < cs.Diff
			c := ca(cDim, 0.35)
			if on {
				c = lighten(col, 0.2)
			}
			fillPath(s, diamondPath(u.X(dx+float64(i)*20+8), u.Y(ry+99), 7*S, 9*S), c, darken(c, 0.3), grad{}, false)
		}
		if u.hover(dx-6, ry+86, 66, 28) {
			g.tooltip = diffNames[cs.Diff]
		}
	}
	y := ry + 126
	y = u.paragraph(s, info.Lore, fReg, 19, cx, y, rw-72, rgba(214, 206, 228, 255), alignLeft) + 10
	u.glowLine(s, cx, rx+rw-36, y, 1, ca(u.theme().Edge, 0.9), 1)
	y += 20
	u.text(s, "ATTRIBUTES", fBold, 15, cx, y, ca(cGold, 0.9), alignLeft)
	y += 30
	st := baseStats()
	a.Apply(&st)
	rows := []struct {
		label string
		v     float64
		val   string
		col   color.RGBA
	}{
		{"Precision", (st.CritChance - 0.04) / 0.18, fmt.Sprintf("%.0f%%", st.CritChance*100), rgba(236, 70, 100, 255)},
		{"Reach", (st.PaddleWidth - 0.6) / 0.8, fmt.Sprintf("%.0f%%", st.PaddleWidth*100), rgba(110, 170, 255, 255)},
		{"Might", (st.Damage*st.DmgMult - 0.4) / 1.6, fmt.Sprintf("%.1f", st.Damage*st.DmgMult), rgba(255, 170, 70, 255)},
		{"Fortune", (st.GoldMult - 0.8) / 0.8, fmt.Sprintf("%.0f%%", st.GoldMult*100), rgba(255, 214, 90, 255)},
	}
	for _, r := range rows {
		u.text(s, r.label, fBold, 19, cx, y, rgba(220, 214, 232, 255), alignLeft)
		u.meter(s, cx+120, y+8, rw-250, r.v, r.col)
		u.text(s, r.val, fBold, 19, rx+rw-36, y, cWhite, alignRight)
		y += 36
	}
	y += 8
	u.glowLine(s, cx, rx+rw-36, y, 1, ca(u.theme().Edge, 0.9), 1)
	y += 20
	u.text(s, "PASSIVE  ·  "+strings.ToUpper(a.Passive), fBold, 15, cx, y, ca(cGold, 0.9), alignLeft)
	y += 28
	y = u.paragraph(s, a.Desc, fReg, 18, cx, y, rw-72, cWhite, alignLeft) + 8
	if lv, _ := g.save.masteryOf(a.ID); lv >= 5 && g.save.masteryOpt(a.ID, mOptAspect) {
		o := originMastery[a.ID]
		u.text(s, "ASPECT OF "+strings.ToUpper(o.Aspect), fBold, 15, cx, y, lighten(col, 0.4), alignLeft)
		y = u.paragraph(s, o.AspectDesc, fReg, 17, cx, y+24, rw-72, rgba(226, 220, 236, 255), alignLeft) + 8
	}
	y += 4
	if a.Ability != "" {
		bx, bw := cx, rw-72
		nl := float64(len(u.wrap(a.AbilityDesc, fReg, 16, bw-100)))
		bh := math.Max(104, 48+nl*16*1.35+28)
		fillPath(s, boxPath(u.X(bx), u.Y(y), bw*S, bh*S, 10*S, false), ca(darken(col, 0.3), 0.6), ca(darken(col, 0.12), 0.6), vgrad(u.Y(y), u.Y(y+bh)), false)
		strokePath(s, boxPath(u.X(bx), u.Y(y), bw*S, bh*S, 10*S, false), 1.4*S, ca(col, 0.85), false)
		u.drawEmblem(s, glyphImage(info.Glyph, col), bx+44, y+bh/2, 56, 1)
		u.text(s, "ACTIVE  ·  "+strings.ToUpper(a.Ability), fBold, 18, bx+86, y+14, lighten(col, 0.45), alignLeft)
		u.paragraph(s, a.AbilityDesc, fReg, 16, bx+86, y+40, bw-100, rgba(226, 220, 236, 255), alignLeft)
		u.text(s, fmt.Sprintf("Right-click or E  ·  %.0fs cooldown", a.Cooldown), fReg, 14, bx+86, y+bh-26, cDim, alignLeft)
		y += bh
	} else {
		u.text(s, "No active ability", fReg, 17, cx, y, ca(cDim, 0.7), alignLeft)
	}

	// begin / unlock
	by := ry + 820 - 96
	if locked {
		can := g.save.Shards >= a.Cost
		if u.button(s, "unlock"+a.ID, fmt.Sprintf("Unlock  ·  %d shards", a.Cost), cx, by, rw-72, 64, cGold, can) || (g.archActivate && can) {
			g.save.Shards -= a.Cost
			g.save.Unlocked[a.ID] = true
			writeSave(&g.save)
			g.snd.Play("buy")
		}
		if !can {
			u.text(s, fmt.Sprintf("You have %d shards", g.save.Shards), fReg, 15, rx+rw/2, by-26, cDim, alignCenter)
		}
	} else {
		pulse := 0.5 + 0.5*math.Sin(g.stateT*3)
		drawSprite(s, sprSoft, u.X(cx-40), u.Y(by-30), (rw-72+80)*S, 124*S, ca(col, 0.12+0.1*pulse), true)
		if u.button(s, "begin", "Begin", cx, by, rw-72, 64, cGreen, true) || g.archActivate {
			g.archActivate = false
			g.startRun(a)
			return
		}
	}

	// ascension (compact, under the panel); challenges set their own rules
	if g.challengeSel == "" {
		g.drawAscCompact(s, rx, 900, rw)
	} else {
		g.challengeBanner(s)
	}
	if u.button(s, "back", "Back", 60, 990, 200, 56, cAcc, true) {
		if g.challengeSel != "" {
			g.setState(StChallenge)
		} else {
			g.setState(StMenu)
		}
	}
}

// drawAscCompact is a narrow ascension selector.
func (g *Game) drawAscCompact(s *ebiten.Image, x, y, w float64) {
	u := g.ui
	u.frame(s, x, y, w, 120, 0.88)
	cx := x + w/2
	if g.save.AscMax == 0 {
		u.text(s, "ASCENSION", fBold, 15, cx, y+18, ca(u.theme().Accent, 1), alignCenter)
		u.paragraph(s, "Win a run to unlock harder runs with bigger rewards.", fReg, 17, cx, y+48, w-60, cDim, alignCenter)
		return
	}
	if u.button(s, "ascL", "‹", x+16, y+32, 52, 52, cAcc, g.save.AscSel > 0) {
		g.save.AscSel--
		writeSave(&g.save)
	}
	if u.button(s, "ascR", "›", x+w-68, y+32, 52, 52, cAcc, g.save.AscSel < g.save.AscMax) {
		g.save.AscSel++
		writeSave(&g.save)
	}
	a := g.save.AscSel
	if a == 0 {
		u.text(s, "ASCENSION 0  ·  STANDARD", fBold, 18, cx, y+20, cWhite, alignCenter)
		u.text(s, fmt.Sprintf("Unlocked up to %d", g.save.AscMax), fReg, 16, cx, y+56, cDim, alignCenter)
		return
	}
	asc := ascensions[a-1]
	u.text(s, fmt.Sprintf("ASCENSION %d  ·  %s", a, strings.ToUpper(asc.Name)), fBold, 18, cx, y+14, rgba(255, 140, 120, 255), alignCenter)
	u.paragraph(s, asc.Desc, fReg, 15, cx, y+44, w-170, cDim, alignCenter)
	u.text(s, fmt.Sprintf("+%d%% shards and XP", a*10), fBold, 14, cx, y+92, rgba(255, 190, 150, 255), alignCenter)
}
