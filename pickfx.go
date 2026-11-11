package main

import (
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
}

type buyFX struct {
	x, y, w, h, t float64
	m             *Mod
}

// startPick begins the reward pick animation for card idx of the set.
func (g *Game) startPick(mods []*Mod, idx int, x0, y, w, h, gap float64) {
	g.pick = pickAnim{on: true, idx: idx, mods: mods, x0: x0, y: y, w: w, h: h, gap: gap, relic: mods[idx].Relic}
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
	if !p.shook {
		p.shook = true
		g.snd.Play("synergy.0")
	}
	// rarity light behind the card
	gl := math.Min(1, t/0.12)
	if t > 0.6 {
		gl = math.Max(0, 1-(t-0.6)/0.3)
	}
	drawSprite(s, sprSoft, u.X(cx-p.w*1.1), u.Y(cy-p.h*0.9), p.w*2.2*S, p.h*1.8*S, ca(rc, 0.55*gl), true)
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
	if t > 0.1 && t < 0.92 {
		bk := ease((t - 0.1) / 0.16)
		ba := math.Min(1, (0.92-t)/0.2)
		label := "UPGRADE GET!"
		if p.relic {
			label = "RELIC CLAIMED"
		}
		size := 40 * (1.5 - 0.5*bk)
		by := p.y + p.h + 70
		drawSprite(s, sprSoft, u.X(960-380), u.Y(by-20), 760*S, 110*S, ca(rc, 0.3*ba*bk), true)
		u.textOutlined(s, label, fRune, size, 960, by-size*0.1, ca(lighten(rc, 0.45), ba*bk), alignCenter)
		u.text(s, m.Name, fBold, 26, 960, by+48, ca(cWhite, ba*bk), alignCenter)
	}
	return t >= pickDur
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
