package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ================================================================ themes
//
// One UI system for the whole game: the same typography, layout, widgets
// and icons everywhere. Only the framing changes, subtly, with the world:
// holographic glass in the Neon Grid, walnut and brass in the meadows,
// faceted crystal in the caverns, carved stone with gold inlay in the
// ruins, ivory and gilt above the clouds, riveted steel in the machine.

type frameStyle int

const (
	styleHolo frameStyle = iota
	styleBrass
	styleCrystal
	styleStone
	styleCloud
	styleMetal
)

type Theme struct {
	Style        frameStyle
	BgTop, BgBot color.RGBA // panel body gradient
	Edge         color.RGBA // outer frame
	Edge2        color.RGBA // inner hairline
	Accent       color.RGBA // highlights and glows
}

func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

// themeArcane frames the front end (title, origins, codex...).
const themeArcane = 6

var themes = [7]Theme{
	WNeon:       {styleHolo, rgba(26, 12, 52, 228), rgba(8, 4, 24, 240), rgba(255, 84, 214, 255), rgba(96, 232, 255, 140), rgba(96, 232, 255, 255)},
	WMeadow:     {styleBrass, rgba(58, 41, 28, 234), rgba(31, 21, 14, 244), rgba(224, 180, 98, 255), rgba(255, 232, 180, 80), rgba(164, 226, 112, 255)},
	WCaverns:    {styleCrystal, rgba(18, 28, 56, 228), rgba(7, 11, 30, 240), rgba(132, 216, 255, 255), rgba(204, 164, 255, 120), rgba(172, 132, 255, 255)},
	WRuins:      {styleStone, rgba(60, 51, 41, 236), rgba(33, 27, 21, 244), rgba(234, 188, 94, 255), rgba(178, 158, 126, 110), rgba(92, 208, 194, 255)},
	WClouds:     {styleCloud, rgba(46, 60, 100, 222), rgba(22, 30, 62, 236), rgba(250, 244, 228, 255), rgba(242, 206, 120, 170), rgba(255, 214, 128, 255)},
	WMachine:    {styleMetal, rgba(46, 48, 54, 240), rgba(21, 22, 26, 246), rgba(158, 164, 176, 255), rgba(255, 172, 48, 140), rgba(255, 158, 44, 255)},
	themeArcane: {styleStone, rgba(22, 13, 38, 232), rgba(8, 5, 17, 244), rgba(196, 160, 92, 255), rgba(168, 120, 255, 110), rgba(184, 140, 255, 255)},
}

// themeWorld picks which world's framing the current screen uses.
func (g *Game) themeWorld() int {
	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp, StCodex:
		return themeArcane
	case StLevel, StPause:
		if g.level != nil {
			return g.level.Biome.ID
		}
	case StSettings:
		if g.prevState == StPause && g.level != nil {
			return g.level.Biome.ID
		}
		if g.run == nil || g.prevState == StMenu {
			return themeArcane
		}
	}
	if g.run != nil {
		return biomes[g.run.AI()].ID
	}
	return WNeon
}

func (u *UI) theme() *Theme {
	if u.Th == nil {
		return &themes[WNeon]
	}
	return u.Th
}

// ================================================================ painting

var cBlack = rgba(0, 0, 0, 255)

// ca scales a colour's (straight) alpha.
func ca(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, uint8(math.Max(0, math.Min(1, a)) * float64(c.A))}
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func ease(t float64) float64 { t = clamp01(t); return 1 - (1-t)*(1-t)*(1-t) }

// grad is a linear gradient axis in screen space.
type grad struct{ x0, y0, x1, y1 float64 }

func vgrad(y0, y1 float64) grad { return grad{0, y0, 0, y1} }
func hgrad(x0, x1 float64) grad { return grad{x0, 0, x1, 0} }

// Vector-path helpers, used only to bake icons into cached textures (see
// iconImage). Live interface shapes are drawn with the SDF renderer in sdf.go.

func vPaint(dst *ebiten.Image, vs []ebiten.Vertex, is []uint16, c0, c1 color.RGBA, gr grad) {
	for i := range vs {
		c := gradCol(c0, c1, gr, float64(vs[i].DstX), float64(vs[i].DstY))
		vs[i].SrcX, vs[i].SrcY = 4, 4
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, atlas, &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero})
}

func vFill(dst *ebiten.Image, p *vector.Path, c0, c1 color.RGBA, gr grad) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	vPaint(dst, vs, is, c0, c1, gr)
}

func vStroke(dst *ebiten.Image, p *vector.Path, w float64, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(w), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	vPaint(dst, vs, is, c, c, grad{})
}

func vPoly(pts [][2]float64) *vector.Path {
	p := &vector.Path{}
	for i, q := range pts {
		if i == 0 {
			p.MoveTo(float32(q[0]), float32(q[1]))
		} else {
			p.LineTo(float32(q[0]), float32(q[1]))
		}
	}
	p.Close()
	return p
}

func vCircle(cx, cy, r float64) *vector.Path {
	n := max(24, int(r))
	pts := make([][2]float64, n)
	for i := range pts {
		a := float64(i) / float64(n) * 2 * math.Pi
		pts[i] = [2]float64{cx + math.Cos(a)*r, cy + math.Sin(a)*r}
	}
	return vPoly(pts)
}

func vDiamond(cx, cy, rx, ry float64) *vector.Path {
	return vPoly([][2]float64{{cx, cy - ry}, {cx + rx, cy}, {cx, cy + ry}, {cx - rx, cy}})
}

// ================================================================ frames

// styleShape returns the corner radius and corner type for a frame style.
func styleShape(st frameStyle) (float64, bool) {
	switch st {
	case styleHolo:
		return 16, true
	case styleMetal:
		return 12, true
	case styleCrystal:
		return 20, true
	case styleStone:
		return 5, false
	case styleCloud:
		return 22, false
	}
	return 12, false
}

// glowLine is a horizontal line that fades out at both ends (virtual coords).
func (u *UI) glowLine(dst *ebiten.Image, x0, x1, y, th float64, c color.RGBA, a float64) {
	S := u.S
	mid := (x0 + x1) / 2
	fillPath(dst, rectPath(u.X(x0), u.Y(y-th/2), (mid-x0)*S, th*S), ca(c, 0), ca(c, a), hgrad(u.X(x0), u.X(mid)), false)
	fillPath(dst, rectPath(u.X(mid), u.Y(y-th/2), (x1-mid)*S, th*S), ca(c, a), ca(c, 0), hgrad(u.X(mid), u.X(x1)), false)
}

// fadeLine runs from solid at x0 to transparent at x1 (either direction).
func (u *UI) fadeLine(dst *ebiten.Image, x0, x1, y, th float64, c color.RGBA, a float64) {
	lo, hi := math.Min(x0, x1), math.Max(x0, x1)
	fillPath(dst, rectPath(u.X(lo), u.Y(y-th/2), (hi-lo)*u.S, th*u.S), ca(c, a), ca(c, 0), hgrad(u.X(x0), u.X(x1)), false)
}

func (u *UI) gem(dst *ebiten.Image, cx, cy, rx, ry float64, c color.RGBA, a float64) {
	S := u.S
	p := diamondPath(u.X(cx), u.Y(cy), rx*S, ry*S)
	strokePath(dst, p, 2.5*S, ca(rgba(10, 6, 18, 255), 0.7*a), false)
	fillPath(dst, p, ca(lighten(c, 0.45), a), ca(darken(c, 0.55), a), vgrad(u.Y(cy-ry), u.Y(cy+ry)), false)
	// table facet highlight
	fillPath(dst, diamondPath(u.X(cx), u.Y(cy-ry*0.32), rx*0.48*S, ry*0.42*S), ca(cWhite, 0.45*a), ca(cWhite, 0.08*a), vgrad(u.Y(cy-ry*0.75), u.Y(cy+ry*0.1)), false)
}

func (u *UI) rivet(dst *ebiten.Image, cx, cy, r float64, c color.RGBA, a float64, hex bool) {
	S := u.S
	var p shp
	if hex {
		p = ngonPath(u.X(cx), u.Y(cy), r*S, 6, math.Pi/6)
	} else {
		p = circlePath(u.X(cx), u.Y(cy), r*S)
	}
	strokePath(dst, p, 2.2*S, ca(cBlack, 0.55*a), false)
	fillPath(dst, p, ca(lighten(c, 0.5), a), ca(darken(c, 0.5), a), vgrad(u.Y(cy-r), u.Y(cy+r)), false)
	if hex {
		strokePath(dst, linePath(u.X(cx-r*0.45), u.Y(cy+r*0.2), u.X(cx+r*0.45), u.Y(cy-r*0.2)), 1.4*S, ca(darken(c, 0.35), a), false)
	} else {
		fillPath(dst, circlePath(u.X(cx-r*0.3), u.Y(cy-r*0.3), r*0.32*S), ca(cWhite, 0.7*a), ca(cWhite, 0.7*a), grad{}, false)
	}
}

// frame draws a themed panel in virtual coordinates; a is its opacity.
func (u *UI) frame(dst *ebiten.Image, x, y, w, h, a float64) {
	if a <= 0.01 {
		return
	}
	th := u.theme()
	S := u.S
	r, ch := styleShape(th.Style)
	r = math.Min(r, math.Min(w, h)/2)
	sx, sy, sw, sh := u.X(x), u.Y(y), w*S, h*S
	// soft two-layer drop shadow
	fillPath(dst, boxPath(sx-4*S, sy+2*S, sw+8*S, sh+14*S, (r+6)*S, ch), ca(cBlack, 0.14*a), ca(cBlack, 0.2*a), grad{}, false)
	fillPath(dst, boxPath(sx, sy+5*S, sw, sh, r*S, ch), ca(cBlack, 0.3*a), ca(cBlack, 0.3*a), grad{}, false)
	body := boxPath(sx, sy, sw, sh, r*S, ch)
	fillPath(dst, body, ca(th.BgTop, a), ca(th.BgBot, a), vgrad(sy, sy+sh), false)
	// glass sheen over the upper part
	fillPath(dst, body, rgba(255, 255, 255, uint8(20*a)), rgba(255, 255, 255, 0), vgrad(sy, sy+math.Min(sh, 110*S)), false)
	strokePath(dst, body, 2*S, ca(th.Edge, 0.92*a), false)
	in := 5.0
	if w > 2*in+4 && h > 2*in+4 {
		strokePath(dst, boxPath(sx+in*S, sy+in*S, sw-2*in*S, sh-2*in*S, math.Max(0, r-in)*S, ch), math.Max(1, S), ca(th.Edge2, a), false)
	}
	u.ornaments(dst, th, x, y, w, h, a)
}

// ornaments adds the per-world detailing to a frame.
func (u *UI) ornaments(dst *ebiten.Image, th *Theme, x, y, w, h, a float64) {
	S := u.S
	small := h < 64 || w < 150
	switch th.Style {
	case styleHolo:
		L := math.Min(26, math.Min(w, h)*0.3)
		o := 4.0
		for _, c := range [][4]float64{{x - o, y - o, 1, 1}, {x + w + o, y - o, -1, 1}, {x - o, y + h + o, 1, -1}, {x + w + o, y + h + o, -1, -1}} {
			hx, hy := u.X(c[0]), u.Y(c[1])
			arms := []shp{linePath(u.X(c[0]+c[2]*L), hy, hx, hy), linePath(hx, hy, hx, u.Y(c[1]+c[3]*L))}
			for _, p := range arms {
				strokePath(dst, p, 7*S, ca(th.Accent, 0.16*a), true)
			}
			for _, p := range arms {
				strokePath(dst, p, 2.4*S, ca(lighten(th.Accent, 0.35), 0.95*a), false)
			}
		}
		if !small {
			u.glowLine(dst, x+w*0.18, x+w*0.82, y+1, 2.5, lighten(th.Edge, 0.3), 0.9*a)
		}
	case styleBrass:
		if !small {
			for yy := y + 26; yy < y+h-16; yy += 28 {
				fillRect(dst, u.X(x+14), u.Y(yy), (w-28)*S, math.Max(1, 1.2*S), prem(rgba(0, 0, 0, 255), 0.07*a))
			}
		}
		in := 13.0
		if small {
			in = 9
		}
		rr := 5.0
		if small {
			rr = 3.5
		}
		for _, c := range [][2]float64{{x + in, y + in}, {x + w - in, y + in}, {x + in, y + h - in}, {x + w - in, y + h - in}} {
			u.rivet(dst, c[0], c[1], rr, th.Edge, a, false)
		}
	case styleCrystal:
		r, _ := styleShape(th.Style)
		r = math.Min(r, math.Min(w, h)/2)
		g := 6.0
		if small {
			g = 4
		}
		for _, c := range [][2]float64{{x + r/2, y + r/2}, {x + w - r/2, y + r/2}, {x + r/2, y + h - r/2}, {x + w - r/2, y + h - r/2}} {
			drawSprite(dst, sprSoft, u.X(c[0]-18), u.Y(c[1]-18), 36*S, 36*S, ca(th.Accent, 0.45*a), true)
			u.gem(dst, c[0], c[1], g, g*1.3, th.Edge, a)
		}
		if !small {
			drawSprite(dst, sprSoft, u.X(x+w/2-30), u.Y(y-30), 60*S, 60*S, ca(th.Accent, 0.5*a), true)
			u.gem(dst, x+w/2, y, 9, 14, lighten(th.Accent, 0.2), a)
		}
	case styleStone:
		g := 8.0
		if small {
			g = 5
		}
		for _, c := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
			u.gem(dst, c[0], c[1], g, g, th.Edge, a)
		}
		if !small && w > 180 {
			cx := x + w/2
			p := boxPath(u.X(cx-26), u.Y(y-6), 52*S, 15*S, 6*S, true)
			strokePath(dst, p, 2.5*S, ca(rgba(10, 6, 4, 255), 0.6*a), false)
			fillPath(dst, p, ca(lighten(th.Edge, 0.35), a), ca(darken(th.Edge, 0.55), a), vgrad(u.Y(y-5), u.Y(y+9)), false)
			u.gem(dst, cx, y+2, 4, 5, th.Accent, a)
		}
	case styleCloud:
		in := 11.0
		if small {
			in = 8
		}
		for _, c := range [][2]float64{{x + in, y + in}, {x + w - in, y + in}, {x + in, y + h - in}, {x + w - in, y + h - in}} {
			fillPath(dst, circlePath(u.X(c[0]), u.Y(c[1]), 3*S), ca(th.Edge2, a), ca(th.Edge2, a), grad{}, false)
		}
		if !small {
			cx := x + w/2
			u.fadeLine(dst, cx-14, cx-14-math.Min(120, w*0.3), y+1, 2, th.Edge2, a)
			u.fadeLine(dst, cx+14, cx+14+math.Min(120, w*0.3), y+1, 2, th.Edge2, a)
			u.gem(dst, cx, y+1, 8, 8, rgba(255, 220, 140, 255), a)
		}
	case styleMetal:
		in := 13.0
		if small {
			in = 9
		}
		rr := 6.0
		if small {
			rr = 4
		}
		for _, c := range [][2]float64{{x + in, y + in}, {x + w - in, y + in}, {x + in, y + h - in}, {x + w - in, y + h - in}} {
			u.rivet(dst, c[0], c[1], rr, th.Edge, a, true)
		}
		if w >= 220 && h >= 90 {
			// hazard trim along the bottom edge
			bx0, bx1 := x+34, x+w-34
			by0, by1 := y+h-15, y+h-9
			fillPath(dst, rectPath(u.X(bx0), u.Y(by0), (bx1-bx0)*S, (by1-by0)*S), ca(rgba(18, 16, 12, 255), a), ca(rgba(18, 16, 12, 255), a), grad{}, false)
			for sx := bx0 + 4; sx+8 < bx1; sx += 16 {
				strokePath(dst, linePath(u.X(sx), u.Y(by1-1), u.X(sx+5), u.Y(by0+1)), 4*S, ca(th.Accent, 0.85*a), false)
			}
		}
	}
}

// heading draws a screen title with themed ornament rules on both sides.
func (u *UI) heading(dst *ebiten.Image, s string, size, y float64, col color.RGBA, a float64) {
	u.headingW(dst, s, size, y, col, a, 960)
}

// headingW is heading with the ornament rules kept within maxHalf of centre.
func (u *UI) headingW(dst *ebiten.Image, s string, size, y float64, col color.RGBA, a, maxHalf float64) {
	th := u.theme()
	w := u.measure(s, fRune, size)
	u.textOutlined(dst, s, fRune, size, 960, y, ca(col, a), alignCenter)
	cy := y + size*0.64
	ln := math.Min(math.Min(220, 40+size*2.4), maxHalf-(w/2+28+12))
	for _, sg := range []float64{-1, 1} {
		x0 := 960 + sg*(w/2+28)
		if ln > 16 {
			u.fadeLine(dst, x0+sg*12, x0+sg*(12+ln), cy, 2.4, th.Edge, 0.9*a)
			u.fadeLine(dst, x0+sg*12, x0+sg*(12+ln*0.55), cy+6, 1.2, th.Edge2, 0.8*a)
		}
		u.gem(dst, x0, cy, 7, 9, th.Edge, a)
	}
}

// meter is a slim stat bar.
func (u *UI) meter(dst *ebiten.Image, x, y, w, frac float64, col color.RGBA) {
	S := u.S
	h := 10.0
	fillPath(dst, boxPath(u.X(x), u.Y(y), w*S, h*S, h/2*S, false), rgba(0, 0, 0, 160), rgba(20, 16, 30, 160), vgrad(u.Y(y), u.Y(y+h)), false)
	fw := math.Max(h, w*clamp01(frac))
	fillPath(dst, boxPath(u.X(x), u.Y(y), fw*S, h*S, h/2*S, false), lighten(col, 0.3), darken(col, 0.6), vgrad(u.Y(y), u.Y(y+h)), false)
	for i := 1; i < 5; i++ {
		fillRect(dst, u.X(x+w*float64(i)/5)-0.6*S, u.Y(y+2), 1.2*S, (h-4)*S, rgba(0, 0, 0, 90))
	}
}

// ================================================================ icons

type iconKind int

const (
	icHeart iconKind = iota
	icCoin
	icShield
	icShard
	icBolt
	icStar
)

type iconStyle struct{ top, bot, line color.RGBA }

var iconStyles = []iconStyle{
	icHeart:  {rgba(255, 132, 154, 255), rgba(204, 26, 62, 255), rgba(46, 8, 20, 255)},
	icCoin:   {rgba(255, 234, 134, 255), rgba(222, 142, 28, 255), rgba(72, 40, 6, 255)},
	icShield: {rgba(188, 164, 255, 255), rgba(88, 62, 214, 255), rgba(22, 12, 60, 255)},
	icShard:  {rgba(216, 188, 255, 255), rgba(118, 72, 236, 255), rgba(28, 12, 64, 255)},
	icBolt:   {rgba(255, 244, 150, 255), rgba(255, 164, 28, 255), rgba(72, 40, 0, 255)},
	icStar:   {rgba(255, 240, 160, 255), rgba(240, 160, 30, 255), rgba(70, 40, 0, 255)},
}

func iconPath(k iconKind, cx, cy, h float64) *vector.Path {
	f := func(v float64) float32 { return float32(v) }
	P := func(x, y float64) [2]float64 { return [2]float64{cx + x*h, cy + y*h} }
	switch k {
	case icHeart:
		p := &vector.Path{}
		cub := func(x1, y1, x2, y2, x3, y3 float64) {
			p.CubicTo(f(cx+x1*h), f(cy+y1*h), f(cx+x2*h), f(cy+y2*h), f(cx+x3*h), f(cy+y3*h))
		}
		p.MoveTo(f(cx), f(cy+0.86*h))
		cub(-0.55, 0.45, -1.0, 0.12, -0.96, -0.3)
		cub(-0.9, -0.78, -0.3, -0.94, 0, -0.46)
		cub(0.3, -0.94, 0.9, -0.78, 0.96, -0.3)
		cub(1.0, 0.12, 0.55, 0.45, 0, 0.86)
		p.Close()
		return p
	case icCoin:
		return vCircle(cx, cy, 0.92*h)
	case icShield:
		p := &vector.Path{}
		p.MoveTo(f(cx-0.82*h), f(cy-0.72*h))
		p.QuadTo(f(cx), f(cy-1.02*h), f(cx+0.82*h), f(cy-0.72*h))
		p.LineTo(f(cx+0.82*h), f(cy-0.02*h))
		p.CubicTo(f(cx+0.82*h), f(cy+0.55*h), f(cx+0.35*h), f(cy+0.84*h), f(cx), f(cy+1.0*h))
		p.CubicTo(f(cx-0.35*h), f(cy+0.84*h), f(cx-0.82*h), f(cy+0.55*h), f(cx-0.82*h), f(cy-0.02*h))
		p.Close()
		return p
	case icShard:
		return vPoly([][2]float64{P(0, -1), P(0.62, -0.34), P(0.46, 0.7), P(0, 1), P(-0.46, 0.7), P(-0.62, -0.34)})
	case icBolt:
		return vPoly([][2]float64{P(0.22, -1), P(-0.62, 0.14), P(-0.06, 0.14), P(-0.26, 1), P(0.62, -0.2), P(0.06, -0.2)})
	case icStar:
		pts := make([][2]float64, 10)
		for i := range pts {
			r := 1.0
			if i%2 == 1 {
				r = 0.44
			}
			a := -math.Pi/2 + float64(i)*math.Pi/5
			pts[i] = P(math.Cos(a)*r, math.Sin(a)*r+0.06)
		}
		return vPoly(pts)
	}
	return vCircle(cx, cy, h)
}

// Icons are baked once into textures at high resolution (with true
// anti-aliasing) and then drawn scaled, so they cost one textured quad.
const iconRes = 160

var iconCache = map[iconKind]*ebiten.Image{}

func iconImage(k iconKind) *ebiten.Image {
	if img, ok := iconCache[k]; ok {
		return img
	}
	img := ebiten.NewImage(iconRes, iconRes)
	cx, cy := float64(iconRes)/2, float64(iconRes)/2
	h := float64(iconRes) * 0.42
	st := iconStyles[k]
	p := iconPath(k, cx, cy, h)
	vStroke(img, p, 0.24*h, ca(st.line, 0.92))
	vFill(img, p, st.top, st.bot, vgrad(cy-h, cy+h))
	switch k {
	case icCoin:
		vStroke(img, vCircle(cx, cy, 0.62*h), 0.11*h, ca(rgba(178, 104, 16, 255), 0.8))
		vFill(img, vDiamond(cx, cy, 0.26*h, 0.36*h), rgba(255, 246, 190, 255), rgba(240, 190, 80, 255), vgrad(cy-0.36*h, cy+0.36*h))
	case icShard:
		vFill(img, vPoly([][2]float64{{cx, cy - h}, {cx - 0.62*h, cy - 0.34*h}, {cx - 0.46*h, cy + 0.7*h}, {cx, cy + h}}), ca(cWhite, 0.28), ca(cWhite, 0.06), vgrad(cy-h, cy+h))
	case icShield:
		vFill(img, iconPath(icShield, cx, cy-0.08*h, h*0.56), ca(cWhite, 0.3), ca(cWhite, 0.02), vgrad(cy-0.7*h, cy+0.5*h))
	}
	if k != icShield && k != icShard {
		vFill(img, vCircle(cx-0.4*h, cy-0.38*h, 0.16*h), ca(cWhite, 0.6), ca(cWhite, 0.6), grad{})
	}
	iconCache[k] = img
	return img
}

// iconAt draws an icon centred on screen point (cx, cy); s is its size in px.
func iconAt(dst *ebiten.Image, k iconKind, cx, cy, s, a float64) {
	img := iconImage(k)
	sc := s / (float64(iconRes) * 0.84)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(iconRes)/2, -float64(iconRes)/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleAlpha(float32(clamp01(a)))
	dst.DrawImage(img, op)
}

// icon draws an icon centred at virtual coordinates.
func (u *UI) icon(dst *ebiten.Image, k iconKind, x, y, size, a float64) {
	iconAt(dst, k, u.X(x), u.Y(y), size*u.S, a)
}

// iconLabel draws an icon followed by text; y is the top of the text line.
func (u *UI) iconLabel(dst *ebiten.Image, k iconKind, s string, f fontID, size, x, y float64, c color.RGBA, al align) float64 {
	is := size * 1.05
	w := is + 8 + u.measure(s, f, size)
	x0 := x
	switch al {
	case alignCenter:
		x0 = x - w/2
	case alignRight:
		x0 = x - w
	}
	u.icon(dst, k, x0+is/2, y+size*0.64, is, float64(c.A)/255)
	u.text(dst, s, f, size, x0+is+8, y, c, alignLeft)
	return w
}

// keycap draws a keyboard key label; returns its width.
func (u *UI) keycap(dst *ebiten.Image, label string, x, y, a float64) float64 {
	S := u.S
	w := u.measure(label, fBold, 16) + 28
	h := 34.0
	fillPath(dst, boxPath(u.X(x), u.Y(y+4), w*S, h*S, 7*S, false), ca(rgba(40, 36, 60, 255), a), ca(rgba(20, 18, 30, 255), a), grad{}, false)
	body := boxPath(u.X(x), u.Y(y), w*S, h*S, 7*S, false)
	fillPath(dst, body, ca(rgba(252, 250, 255, 255), a), ca(rgba(196, 192, 214, 255), a), vgrad(u.Y(y), u.Y(y+h)), false)
	strokePath(dst, body, 1.5*S, ca(rgba(90, 86, 110, 255), a), false)
	u.drawText(dst, label, fBold, 16, u.X(x+w/2), u.Y(y+7), ca(rgba(30, 26, 44, 255), a), alignCenter)
	return w
}

// ================================================================ widgets

// anim eases a per-widget value toward 1 while on is true.
func (u *UI) anim(id string, on bool, speed float64) float64 {
	v := u.anims[id]
	t := 0.0
	if on {
		t = 1
	}
	v += (t - v) * math.Min(1, u.DT*speed)
	if math.Abs(t-v) < 0.002 {
		v = t
	}
	u.anims[id] = v
	return v
}

// button draws a themed button and returns true when clicked. accent is
// the semantic colour (confirm, danger, gold, neutral).
func (u *UI) button(dst *ebiten.Image, id, label string, x, y, w, h float64, accent color.RGBA, enabled bool) bool {
	th := u.theme()
	var hv, clk bool
	if enabled {
		hv, clk = u.hit(id, x, y, w, h)
	}
	if hv && u.nav.mode {
		u.focusRing(dst, x, y, w, h, math.Min(h/2, 14))
	}
	if hv && !u.nav.mode {
		u.hoverID = id
		if u.lastHover != id && u.snd != nil {
			u.snd.Play("hover")
		}
	}
	k := u.anim("btn"+id, hv, 14)
	if hv && u.Held {
		y += 2
	}
	S := u.S
	r, ch := styleShape(th.Style)
	r = math.Min(r*0.7, h/2)
	sx, sy, sw, sh := u.X(x), u.Y(y), w*S, h*S
	a := 1.0
	if !enabled {
		a = 0.6
	}
	fillPath(dst, boxPath(sx, sy+5*S, sw, sh, r*S, ch), ca(cBlack, 0.35*a), ca(cBlack, 0.35*a), grad{}, false)
	if k > 0.01 {
		drawSprite(dst, sprSoft, sx-30*S, sy-24*S, sw+60*S, sh+48*S, ca(accent, 0.22*k), true)
	}
	body := boxPath(sx, sy, sw, sh, r*S, ch)
	top := mixc(th.BgTop, accent, 0.16+0.24*k)
	bot := mixc(th.BgBot, accent, 0.05+0.12*k)
	top.A, bot.A = 245, 250
	if !enabled {
		top, bot = rgba(42, 40, 50, 220), rgba(24, 22, 30, 230)
	}
	fillPath(dst, body, top, bot, vgrad(sy, sy+sh), false)
	fillPath(dst, body, rgba(255, 255, 255, uint8(22+22*k)), rgba(255, 255, 255, 0), vgrad(sy, sy+sh*0.55), false)
	edge := mixc(th.Edge, lighten(accent, 0.35), 0.35+0.65*k)
	if !enabled {
		edge = rgba(90, 88, 100, 255)
	}
	strokePath(dst, body, 2*S, ca(edge, 0.95), false)
	if enabled {
		uw := w * (0.3 + 0.4*k)
		u.glowLine(dst, x+w/2-uw/2, x+w/2+uw/2, y+h-6, 3, lighten(accent, 0.2), 0.95)
	}
	tc := cWhite
	if !enabled {
		tc = rgba(130, 126, 142, 255)
	}
	size := math.Min(26, h*0.42)
	u.text(dst, label, fBold, size, x+w/2, y+h/2-size*0.64, tc, alignCenter)
	if clk {
		if u.snd != nil {
			u.snd.Play("click")
		}
		return true
	}
	return false
}

// slider returns the new value (0..1).
func (u *UI) slider(dst *ebiten.Image, label string, x, y, w float64, v float64, held bool) float64 {
	return u.sliderFmt(dst, label, x, y, w, v, held, func(v float64) string { return itoa(int(math.Round(v*100))) + "%" })
}

// sliderFmt is a slider whose readout is formatted by show.
func (u *UI) sliderFmt(dst *ebiten.Image, label string, x, y, w float64, v float64, held bool, show func(float64) string) float64 {
	th := u.theme()
	S := u.S
	if f, _ := u.navReg("sl"+label, x, y-6, w, 44, true); f {
		u.focusRing(dst, x-10, y-6, w+20, 44, 12)
		if u.nav.adjust != 0 {
			v = math.Max(0, math.Min(1, math.Round(v*20+float64(u.nav.adjust))/20))
			if u.snd != nil {
				u.snd.Play("hover")
			}
		}
	}
	u.text(dst, label, fBold, 22, x, y, cWhite, alignLeft)
	bx, by, bw, bh := x+300, y+10, w-380, 12.0
	track := boxPath(u.X(bx), u.Y(by), bw*S, bh*S, bh/2*S, false)
	fillPath(dst, track, rgba(0, 0, 0, 170), rgba(30, 26, 40, 170), vgrad(u.Y(by), u.Y(by+bh)), false)
	fw := math.Max(bh, bw*v)
	fillPath(dst, boxPath(u.X(bx), u.Y(by), fw*S, bh*S, bh/2*S, false), lighten(th.Accent, 0.3), darken(th.Accent, 0.6), vgrad(u.Y(by), u.Y(by+bh)), false)
	strokePath(dst, track, 1.2*S, ca(th.Edge, 0.6), false)
	kx, ky := u.X(bx+bw*v), u.Y(by+bh/2)
	drawSprite(dst, sprSoft, kx-26*S, ky-26*S, 52*S, 52*S, ca(th.Accent, 0.35), true)
	knob := circlePath(kx, ky, 14*S)
	strokePath(dst, knob, 2.5*S, ca(cBlack, 0.6), false)
	fillPath(dst, knob, lighten(th.Edge, 0.55), darken(th.Edge, 0.55), vgrad(ky-14*S, ky+14*S), false)
	fillPath(dst, circlePath(kx, ky, 5*S), th.Accent, th.Accent, grad{}, false)
	u.text(dst, show(v), fBold, 20, x+w, y+1, cDim, alignRight)
	if held && u.MY > y-10 && u.MY < y+40 && u.MX > bx-20 && u.MX < bx+bw+20 {
		v = math.Max(0, math.Min(1, (u.MX-bx)/bw))
	}
	return v
}

// switchCtl draws an on/off switch at (x, y) and returns true when clicked.
func (u *UI) switchCtl(dst *ebiten.Image, id string, x, y float64, on bool) bool {
	th := u.theme()
	S := u.S
	w, h := 92.0, 42.0
	hv, clk := u.hit(id, x-460, y-6, w+460+60, h+12)
	if hv && u.nav.mode {
		u.focusRing(dst, x-760+60, y-6, 760+w-20, h+12, 12)
	}
	if hv && !u.nav.mode {
		u.hoverID = id
		if u.lastHover != id && u.snd != nil {
			u.snd.Play("hover")
		}
	}
	k := u.anim("sw"+id, on, 16)
	hk := u.anim("swh"+id, hv, 14)
	body := boxPath(u.X(x), u.Y(y), w*S, h*S, h/2*S, false)
	off0, off1 := rgba(30, 26, 42, 240), rgba(14, 12, 22, 240)
	on0, on1 := lighten(th.Accent, 0.1), darken(th.Accent, 0.45)
	fillPath(dst, body, mixc(off0, on0, k), mixc(off1, on1, k), vgrad(u.Y(y), u.Y(y+h)), false)
	strokePath(dst, body, 2*S, ca(mixc(th.Edge, cWhite, hk*0.4), 0.9), false)
	kx := x + h/2 + (w-h)*k
	ky := y + h/2
	knob := circlePath(u.X(kx), u.Y(ky), (h/2-5)*S)
	strokePath(dst, knob, 2*S, ca(cBlack, 0.5), false)
	fillPath(dst, knob, rgba(255, 255, 255, 255), rgba(200, 198, 220, 255), vgrad(u.Y(ky-h/2), u.Y(ky+h/2)), false)
	lbl := "OFF"
	lx := x + w + 16
	if on {
		lbl = "ON"
	}
	u.text(dst, lbl, fBold, 18, lx, y+10, mixc(cDim, cWhite, k), alignLeft)
	if clk {
		if u.snd != nil {
			u.snd.Play("click")
		}
		return true
	}
	return false
}

// ================================================================ toasts & transitions

type toastMsg struct {
	Title, Desc string
	Col         color.RGBA
	Icon        iconKind
	Img         *ebiten.Image // optional item emblem instead of an icon
	T           float64
}

const toastDur = 3.6

func (g *Game) toast(title, desc string, col color.RGBA, ic iconKind) {
	g.toasts = append(g.toasts, toastMsg{Title: title, Desc: desc, Col: col, Icon: ic})
	if len(g.toasts) > 4 {
		g.toasts = g.toasts[len(g.toasts)-4:]
	}
}

func (g *Game) toastImg(title, desc string, col color.RGBA, img *ebiten.Image) {
	g.toast(title, desc, col, icStar)
	g.toasts[len(g.toasts)-1].Img = img
}

func (g *Game) drawToasts(dst *ebiten.Image, dt float64) {
	u := g.ui
	keep := g.toasts[:0]
	for _, t := range g.toasts {
		t.T += dt
		if t.T < toastDur {
			keep = append(keep, t)
		}
	}
	g.toasts = keep
	y := 140.0
	for _, t := range g.toasts {
		k := math.Min(ease(t.T/0.3), ease((toastDur-t.T)/0.35))
		w, h := 410.0, 84.0
		x := 1896 - w + (1-k)*(w+60)
		u.frame(dst, x, y, w, h, k)
		fillPath(dst, boxPath(u.X(x+12), u.Y(y+16), 5*u.S, (h-32)*u.S, 2.5*u.S, false), ca(t.Col, k), ca(darken(t.Col, 0.6), k), vgrad(u.Y(y+16), u.Y(y+h-16)), false)
		if t.Img != nil {
			u.drawEmblem(dst, t.Img, x+50, y+h/2, 54, k)
		} else {
			u.icon(dst, t.Icon, x+50, y+h/2, 40, k)
		}
		u.text(dst, t.Title, fBold, 21, x+84, y+14, ca(lighten(t.Col, 0.2), k), alignLeft)
		lines := u.wrap(t.Desc, fReg, 17, w-104)
		line := ""
		if len(lines) > 0 {
			line = lines[0]
			if len(lines) > 1 {
				line += "…"
			}
		}
		u.text(dst, line, fReg, 17, x+84, y+46, ca(cDim, k), alignLeft)
		y += (h + 12) * k
	}
}

// drawTransition fades each new screen in from dark with a light sweep.
func (g *Game) drawTransition(dst *ebiten.Image, dt float64) {
	if g.fadeT <= 0 {
		return
	}
	g.fadeT -= dt / 0.34
	k := clamp01(g.fadeT)
	k = k * k * (3 - 2*k)
	u := g.ui
	fillRect(dst, 0, 0, u.SW, u.SH, prem(rgba(6, 4, 14, 255), 0.92*k))
	half := 960 * (1 - k)
	if half > 4 {
		u.glowLine(dst, 960-half-40, 960+half+40, 540, 2, u.theme().Accent, k)
	}
}
func smootherstep(x float64) float64 { x = clamp01(x); return x * x * x * (x*(x*6-15) + 10) }
