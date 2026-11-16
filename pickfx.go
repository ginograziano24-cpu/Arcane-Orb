package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Choosing a card should feel like a moment: the pick punches forward in a
// burst of its rarity's light while the others fall away, a banner names
// what you took, and the card collapses into a streak that flies up into
// your build. Shop purchases get a shorter flash of the same.

const pickDur = 1.05

type pickAnim struct {
	on          bool
	t           float64
	idx         int
	mods        []*Mod
	x0, y, w, h float64
	gap         float64
	relic       bool
	shook       bool
	flew        bool         // the fly-up sound has played
	gains       []statChange // what the card changed (shown as chips)
}

// dur is how long the pick plays: a little longer when there are stat
// chips to read.
func (p *pickAnim) dur() float64 {
	if len(p.gains) > 0 {
		return pickDur + 0.35
	}
	return pickDur
}

type buyFX struct {
	x, y, w, h, t float64
	m             *Mod
}

// startPick begins the reward pick animation for card idx of the set.
func (g *Game) startPick(mods []*Mod, idx int, x0, y, w, h, gap float64, gains []statChange) {
	g.pick = pickAnim{on: true, idx: idx, mods: mods, x0: x0, y: y, w: w, h: h, gap: gap, relic: mods[idx].Relic, gains: gains}
}

// withScale draws f with the UI scaled by k about the virtual point
// (cx, cy), so any widget can be punched or shrunk in place.
func (u *UI) withScale(k, cx, cy, dy float64, f func()) {
	ox, oy, s := u.OX, u.OY, u.S
	u.OX = ox + cx*s*(1-k)
	u.OY = oy + cy*s*(1-k) + dy*s
	u.S = s * k
	f()
	u.OX, u.OY, u.S = ox, oy, s
}

// faded draws f into an offscreen layer and composites it with alpha a.
func (g *Game) faded(dst *ebiten.Image, a float64, f func(*ebiten.Image)) {
	if a <= 0.01 {
		return
	}
	b := dst.Bounds()
	if g.fadeBuf == nil || g.fadeBuf.Bounds().Dx() != b.Dx() || g.fadeBuf.Bounds().Dy() != b.Dy() {
		g.fadeBuf = ebiten.NewImage(b.Dx(), b.Dy())
	}
	g.fadeBuf.Clear()
	f(g.fadeBuf)
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(g.fadeBuf, op)
}

// drawPick plays the pick animation; it returns true once it has finished.
func (g *Game) drawPick(s *ebiten.Image) bool {
	u := g.ui
	S := u.S
	p := &g.pick
	p.t += u.DT
	t := p.t
	m := p.mods[p.idx]
	rc := rarityColors[m.Rarity]
	cardX := func(i int) float64 { return p.x0 + float64(i)*(p.w+p.gap) }
	// the others drop away and fade
	k := smootherstep(t / 0.38)
	for i, om := range p.mods {
		if i == p.idx {
			continue
		}
		x := cardX(i)
		g.faded(s, 1-k, func(dst *ebiten.Image) {
			u.withScale(1-0.12*k, x+p.w/2, p.y+p.h/2, 90*k, func() {
				g.drawCard(dst, om, x, p.y, p.w, p.h, 0, false)
			})
		})
	}
	cx, cy := cardX(p.idx)+p.w/2, p.y+p.h/2
	// screen flash and a little kick
	if t < 0.25 {
		fl := 1 - t/0.25
		fillRectA(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), rc, 0.18*fl*fl)
	}
	p.shook = true
	// rarity light behind the card
	gl := math.Min(1, t/0.12)
	if t > 0.6 {
		gl = math.Max(0, 1-(t-0.6)/0.3)
	}
	drawSprite(s, sprSoft, u.X(cx-p.w*1.1), u.Y(cy-p.h*0.9), p.w*2.2*S, p.h*1.8*S, ca(rc, 0.55*gl), true)
	// rarer cards get turning god rays, and a third ring from Epic up
	if m.Rarity >= Rare && t < 0.75 {
		ra := math.Min(1, t/0.15) * math.Max(0, 1-math.Max(0, t-0.5)/0.25)
		n := 10 + 4*m.Rarity
		for i := 0; i < n; i++ {
			a := t*0.9 + float64(i)/float64(n)*2*math.Pi
			r0, r1 := p.h*0.35, p.h*(0.75+0.08*float64(m.Rarity))+30*math.Sin(t*6+float64(i))
			strokeLine(s, u.X(cx+math.Cos(a)*r0), u.Y(cy+math.Sin(a)*r0), u.X(cx+math.Cos(a)*r1), u.Y(cy+math.Sin(a)*r1), 14*S, prem(lighten(rc, 0.3), 0.13*ra))
		}
	}
	if m.Rarity >= Epic {
		e := (t - 0.24) / 0.7
		if e > 0 && e < 1 {
			rr := 100 + 760*ease(e)
			drawSprite(s, sprRing, u.X(cx-rr), u.Y(cy-rr), 2*rr*S, 2*rr*S, ca(cGold, 0.6*(1-e)), true)
		}
	}
	// two shockwave rings
	for j, d := range []float64{0, 0.12} {
		e := (t - d) / 0.6
		if e > 0 && e < 1 {
			rr := 80 + 520*ease(e)
			drawSprite(s, sprRing, u.X(cx-rr), u.Y(cy-rr), 2*rr*S, 2*rr*S, ca(lighten(rc, 0.3*float64(j)), 0.85*(1-e)), true)
		}
	}
	// the card itself: punch, hold, then collapse upward
	sc, dy, a := 1.0, 0.0, 1.0
	switch {
	case t < 0.14:
		e := t / 0.14
		sc = 1 + 0.14*math.Sin(e*math.Pi*0.5)*1.2
	case t < 0.58:
		e := (t - 0.14) / 0.44
		sc = 1.168 - 0.06*ease(e)
	default:
		e := smootherstep((t - 0.58) / 0.36)
		sc = 1.108 * (1 - 0.92*e)
		dy = -(cy - 160) * e
		a = 1 - e*e
	}
	if a > 0.01 {
		g.faded(s, a, func(dst *ebiten.Image) {
			u.withScale(sc, cx, cy, dy, func() {
				g.drawCard(dst, m, cardX(p.idx), p.y, p.w, p.h, 0, false)
			})
		})
		// a white sheen sweeping across it on the punch
		if t < 0.5 {
			sw := (t / 0.5)
			x := cx - p.w*sc/2 + p.w*sc*sw
			drawSprite(s, sprSoft, u.X(x-70), u.Y(cy-p.h*sc/2+dy), 140*S, p.h*sc*S, ca(cWhite, 0.45*(1-sw)), true)
		}
	}
	// the streak flying up into the build
	if t > 0.56 && !p.flew {
		p.flew = true
		g.snd.PlayKeyed("cardfly." + itoa(m.Rarity))
	}
	if t > 0.58 && t < pickDur {
		e := smootherstep((t - 0.58) / 0.36)
		hy := cy + dy
		drawSprite(s, sprSoft, u.X(cx-26), u.Y(hy-26), 52*S, (52+(cy-hy)*0.6)*S, ca(lighten(rc, 0.4), 0.7*(1-e*0.5)), true)
		if e > 0.92 {
			f := (e - 0.92) / 0.08
			drawSprite(s, sprSoft, u.X(cx-120), u.Y(160-120), 240*S, 240*S, ca(lighten(rc, 0.5), 0.8*(1-f)), true)
		}
	}
	// sparks
	if t > 0.02 {
		for i := 0; i < 46; i++ {
			fi := float64(i)
			an := fi/46*2*math.Pi + math.Sin(fi*12.9)*0.3
			sp := 260 + 520*math.Mod(fi*0.618, 1)
			tt := t - 0.02
			x := cx + math.Cos(an)*sp*tt
			y := cy + math.Sin(an)*sp*tt*0.8 + 420*tt*tt
			al := math.Max(0, 1-tt/0.8)
			sz := (3 + 6*math.Mod(fi*0.37, 1)) * S * (1 - tt*0.6)
			if al > 0 && sz > 0 {
				drawSprite(s, sprSoft, u.X(x)-sz*2, u.Y(y)-sz*2, sz*4, sz*4, ca(lighten(rc, 0.3+0.4*math.Mod(fi*0.21, 1)), al), true)
			}
		}
	}
	// the banner
	bEnd := p.dur() - 0.13
	if t > 0.1 && t < bEnd {
		bk := ease((t - 0.1) / 0.16)
		ba := math.Min(1, (bEnd-t)/0.2)
		label := "UPGRADE GET!"
		if p.relic {
			label = "RELIC CLAIMED"
		}
		size := 40 * (1.5 - 0.5*bk)
		by := p.y + p.h + 70
		drawSprite(s, sprSoft, u.X(960-380), u.Y(by-20), 760*S, 110*S, ca(rc, 0.3*ba*bk), true)
		u.textOutlined(s, label, fRune, size, 960, by-size*0.1, ca(lighten(rc, 0.45), ba*bk), alignCenter)
		u.text(s, m.Name, fBold, 26, 960, by+48, ca(cWhite, ba*bk), alignCenter)
		g.drawPickChips(s, p, cx, cy, by+96, t, ba)
	}
	return t >= p.dur()
}

// drawPickChips: the stat changes the card made fly out of it and line up
// under the banner, each value counting up to its new number.
func (g *Game) drawPickChips(s *ebiten.Image, p *pickAnim, fromX, fromY, y, t, alpha float64) {
	if len(p.gains) == 0 {
		return
	}
	u := g.ui
	S := u.S
	type chip struct {
		text, val string
		w         float64
		col       color.RGBA
	}
	var chips []chip
	total := 0.0
	for i, c := range p.gains {
		d := t - 0.3 - float64(i)*0.09
		if d < 0 {
			break
		}
		if c.Synergy {
			txt := fmt.Sprintf("Activates %s", c.Label)
			w := u.measure(txt, fBold, 20) + 36
			chips = append(chips, chip{txt, "", w, cGold})
			total += w + 14
			continue
		}
		val := countUp(c.From, c.To, clamp01(d/0.4))
		w := u.measure(c.Label, fReg, 20) + u.measure(c.From+" → "+c.To, fBold, 20) + 52
		col := rgba(140, 240, 150, 255)
		if !c.Up {
			col = rgba(255, 130, 120, 255)
		}
		chips = append(chips, chip{c.Label, c.From + " → " + val, w, col})
		total += w + 14
	}
	x := 960 - (total-14)/2
	for i, c := range chips {
		d := t - 0.3 - float64(i)*0.09
		k := smootherstep(d / 0.32)
		// fly from the card to its place in the row
		tx, ty := x+c.w/2, y
		px, py := fromX+(tx-fromX)*k, fromY+(ty-fromY)*k-math.Sin(k*math.Pi)*60
		a := alpha * math.Min(1, d/0.12)
		sc := 0.6 + 0.4*k + 0.12*math.Sin(math.Min(1, d/0.45)*math.Pi)
		u.withScale(sc, px, py+18, 0, func() {
			bx := px - c.w/2
			fillPath(s, boxPath(u.X(bx), u.Y(py), c.w*u.S, 36*u.S, 18*u.S, false), ca(rgba(14, 22, 16, 255), 0.85*a), ca(rgba(8, 12, 9, 255), 0.85*a), vgrad(u.Y(py), u.Y(py+36)), false)
			strokePath(s, boxPath(u.X(bx), u.Y(py), c.w*u.S, 36*u.S, 18*u.S, false), 1.6*u.S, ca(c.col, 0.8*a), false)
			if c.val == "" {
				u.text(s, c.text, fBold, 20, px, py+6, ca(c.col, a), alignCenter)
			} else {
				lw := u.measure(c.text, fReg, 20)
				u.text(s, c.text, fReg, 20, bx+18, py+6, ca(cDim, a), alignLeft)
				u.text(s, c.val, fBold, 20, bx+18+lw+16, py+6, ca(c.col, a), alignLeft)
			}
		})
		if k < 1 {
			drawSprite(s, sprSoft, u.X(px-30), u.Y(py-12), 60*S, 60*S, ca(c.col, 0.4*a*(1-k)), true)
		}
		x += c.w + 14
	}
}

// addBuyFX flashes a purchased card in the shop.
func (g *Game) addBuyFX(m *Mod, x, y, w, h float64) {
	g.buys = append(g.buys, buyFX{x, y, w, h, 0, m})
}

func (g *Game) drawBuyFX(s *ebiten.Image) {
	u := g.ui
	S := u.S
	keep := g.buys[:0]
	for _, b := range g.buys {
		b.t += u.DT
		if b.t > 0.8 {
			continue
		}
		keep = append(keep, b)
		rc := rarityColors[b.m.Rarity]
		cx, cy := b.x+b.w/2, b.y+b.h/2
		k := b.t / 0.8
		drawSprite(s, sprSoft, u.X(cx-b.w), u.Y(cy-b.h*0.8), 2*b.w*S, 1.6*b.h*S, ca(rc, 0.6*(1-k)), true)
		fillRectA(s, u.X(b.x), u.Y(b.y), b.w*S, b.h*S, lighten(rc, 0.5), 0.5*(1-k)*(1-k))
		rr := 60 + 300*ease(k)
		drawSprite(s, sprRing, u.X(cx-rr), u.Y(cy-rr), 2*rr*S, 2*rr*S, ca(rc, 0.9*(1-k)), true)
		for i := 0; i < 24; i++ {
			fi := float64(i)
			an := fi / 24 * 2 * math.Pi
			sp := 200 + 260*math.Mod(fi*0.618, 1)
			x := cx + math.Cos(an)*sp*b.t
			y := cy + math.Sin(an)*sp*b.t + 300*b.t*b.t
			sz := (3 + 4*math.Mod(fi*0.37, 1)) * S
			drawSprite(s, sprSoft, u.X(x)-sz*2, u.Y(y)-sz*2, sz*4, sz*4, ca(lighten(rc, 0.4), 1-k), true)
		}
	}
	g.buys = keep
}

// dealIn is card i's entrance progress (0..1) at screen time t.
func dealIn(t float64, i int) float64 {
	return clamp01((t - 0.05 - 0.09*float64(i)) / 0.5)
}

// fadedFlip is faded with the layer squeezed horizontally about virtual x
// cx by sx (0..1), for cards turning over.
func (g *Game) fadedFlip(dst *ebiten.Image, a, sx, cx float64, f func(*ebiten.Image)) {
	if a <= 0.01 || sx <= 0.01 {
		return
	}
	b := dst.Bounds()
	if g.fadeBuf == nil || g.fadeBuf.Bounds().Dx() != b.Dx() || g.fadeBuf.Bounds().Dy() != b.Dy() {
		g.fadeBuf = ebiten.NewImage(b.Dx(), b.Dy())
	}
	g.fadeBuf.Clear()
	f(g.fadeBuf)
	sx0 := g.ui.X(cx)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-sx0, 0)
	op.GeoM.Scale(sx, 1)
	op.GeoM.Translate(sx0, 0)
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(g.fadeBuf, op)
}

// drawCardBack is a reward card face down (ROADMAP-50 #92 and #57): the
// back is tinted by the card's rarity, so a rare card glows before it turns.
func (g *Game) drawCardBack(s *ebiten.Image, m *Mod, x, y, w, h float64) {
	u := g.ui
	S := u.S
	rc := rarityColors[m.Rarity]
	r, ch := styleShape(u.theme().Style)
	r = math.Min(r+4, 26)
	sx, sy, sw, sh := u.X(x), u.Y(y), w*S, h*S
	body := boxPath(sx, sy, sw, sh, r*S, ch)
	fillPath(s, boxPath(sx+6*S, sy+12*S, sw, sh, r*S, ch), ca(cBlack, 0.42), ca(cBlack, 0.42), grad{}, false)
	fillPath(s, body, darken(rc, 0.55), darken(rc, 0.8), vgrad(sy, sy+sh), false)
	strokePath(s, body, 3*S, lighten(rc, 0.2), false)
	strokePath(s, boxPath(sx+14*S, sy+14*S, sw-28*S, sh-28*S, math.Max(0, r-8)*S, ch), 1.5*S, ca(lighten(rc, 0.3), 0.6), false)
	g.drawSigil(s, x+w/2, y+h/2, w*0.36, g.stateT*0.6, lighten(rc, 0.3), 0.7)
	drawSprite(s, sprSoft, u.X(x+w/2-w*0.4), u.Y(y+h/2-w*0.4), w*0.8*S, w*0.8*S, ca(rc, 0.35+0.1*float64(m.Rarity)), true)
	u.drawEmblem(s, glyphImage("orb", lighten(rc, 0.4)), x+w/2, y+h/2, w*0.3, 1)
}
