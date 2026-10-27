package main

import (
	"bytes"
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
	// colour (JPEG) and alpha (greyscale PNG) are stored separately
	var img *ebiten.Image
	cdata, err1 := portraitFS.ReadFile("assets/portraits/" + id + ".jpg")
	adata, err2 := portraitFS.ReadFile("assets/portraits/" + id + "_a.png")
	if err1 == nil && err2 == nil {
		c, _, e1 := image.Decode(bytes.NewReader(cdata))
		a, _, e2 := image.Decode(bytes.NewReader(adata))
		if e1 == nil && e2 == nil {
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
			img = ebiten.NewImageFromImage(out)
		}
	}
	portraitCache[id] = img
	return img
}

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
	if g.keyHit(ebiten.KeyArrowUp) || g.keyHit(ebiten.KeyW) {
		g.archSel = (g.archSel + n - 1) % n
		g.snd.Play("hover")
	}
	if g.keyHit(ebiten.KeyArrowDown) || g.keyHit(ebiten.KeyS) {
		g.archSel = (g.archSel + 1) % n
		g.snd.Play("hover")
	}
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
		xf := portraitXform{px, py, pw, ph, 1 + 0.003*br, 1 + 0.009*br, g.archPar[0], g.archPar[1] + dy}
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
	u.textOutlined(s, "Choose your Origin", fRune, 46, 70, 40, rgba(232, 214, 170, 255), alignLeft)
	u.text(s, strings.ToUpper(sub), fBold, 17, 72, 104, ca(lighten(u.theme().Accent, 0.3), 0.9), alignLeft)
	u.glowLine(s, 70, 520, 136, 1.5, cGold, 0.7)

	// ---- origin list (left)
	ly := 170.0
	for i := range archetypes {
		c := &archetypes[i]
		ci := classInfo[c.ID]
		y := ly + float64(i)*86
		hv := u.hover(60, y, 470, 78)
		if hv && u.lastHover != "arch"+c.ID {
			g.snd.Play("hover")
		}
		if hv {
			u.hoverID = "arch" + c.ID
			if u.Click || g.archSel != i {
				g.archSel = i
			}
		}
		k := u.anim("archrow"+c.ID, g.archSel == i, 12)
		if k > 0.01 {
			fillPath(s, boxPath(u.X(60), u.Y(y), 470*S, 78*S, 6*S, false), ca(ci.Col, 0.2*k), ca(ci.Col, 0.02*k), hgrad(u.X(60), u.X(530)), false)
			fillRect(s, u.X(60), u.Y(y+8), 4*S, 62*S, ca(lighten(ci.Col, 0.3), k))
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
	}

	// ---- detail panel (right)
	rx, rw := 1340.0, 520.0
	ry := 60.0
	u.frame(s, rx, ry, rw, 820, 0.88)
	cx := rx + 36
	u.textOutlined(s, a.Name, fRune, 50, cx, ry+24, lighten(col, 0.35), alignLeft)
	y := ry + 100
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
		{"Vigor", (st.MaxHP - 60) / 80, fmt.Sprintf("%.0f", st.MaxHP), rgba(236, 70, 100, 255)},
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
	y = u.paragraph(s, a.Desc, fReg, 18, cx, y, rw-72, cWhite, alignLeft) + 12
	if a.Ability != "" {
		bx, bw, bh := cx, rw-72, 104.0
		fillPath(s, boxPath(u.X(bx), u.Y(y), bw*S, bh*S, 10*S, false), ca(darken(col, 0.3), 0.6), ca(darken(col, 0.12), 0.6), vgrad(u.Y(y), u.Y(y+bh)), false)
		strokePath(s, boxPath(u.X(bx), u.Y(y), bw*S, bh*S, 10*S, false), 1.4*S, ca(col, 0.85), false)
		u.drawEmblem(s, glyphImage(info.Glyph, col), bx+44, y+bh/2, 56, 1)
		u.text(s, "ACTIVE  ·  "+strings.ToUpper(a.Ability), fBold, 18, bx+86, y+14, lighten(col, 0.45), alignLeft)
		u.paragraph(s, a.AbilityDesc, fReg, 16, bx+86, y+40, bw-100, rgba(226, 220, 236, 255), alignLeft)
		u.text(s, fmt.Sprintf("Right-click or E  ·  %.0fs cooldown", a.Cooldown), fReg, 14, bx+86, y+78, cDim, alignLeft)
		y += bh
	} else {
		u.text(s, "No active ability", fReg, 17, cx, y, ca(cDim, 0.7), alignLeft)
	}

	// begin / unlock
	by := ry + 820 - 96
	if locked {
		can := g.save.Shards >= a.Cost
		if u.button(s, "unlock"+a.ID, fmt.Sprintf("Unlock  ·  %d shards", a.Cost), cx, by, rw-72, 64, cGold, can) {
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
		if u.button(s, "begin", "Begin", cx, by, rw-72, 64, cGreen, true) || g.keyHit(ebiten.KeyEnter) {
			g.startRun(a)
			return
		}
	}

	// ascension (compact, under the panel)
	g.drawAscCompact(s, rx, 900, rw)
	if u.button(s, "back", "Back", 60, 990, 200, 56, cAcc, true) {
		g.setState(StMenu)
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
