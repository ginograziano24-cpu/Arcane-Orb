package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Item emblems. Every upgrade, relic, fusion and Sanctum upgrade has an
// emblem built from vector shapes: outlined, gradient-filled and
// highlighted, tinted with the item's element colour. They are baked once
// into textures (with real anti-aliasing) and drawn scaled.

const emblemRes = 192

type ib struct {
	img        *ebiten.Image
	cx, cy, h  float64
	ox, oy, sc float64
	rot        float64
	c          color.RGBA
}

var (
	toneGold  = color.RGBA{255, 206, 84, 255}
	toneSteel = color.RGBA{196, 204, 222, 255}
	toneWood  = color.RGBA{150, 98, 56, 255}
	toneWhite = color.RGBA{248, 248, 255, 255}
	toneDark  = color.RGBA{34, 26, 48, 255}
	toneLava  = color.RGBA{255, 120, 30, 255}
	toneIce   = color.RGBA{170, 230, 255, 255}
	toneRed   = color.RGBA{232, 52, 72, 255}
	toneGreen = color.RGBA{110, 210, 90, 255}
)

func (b *ib) P(x, y float64) (float32, float32) {
	cs, sn := math.Cos(b.rot), math.Sin(b.rot)
	x, y = x*cs-y*sn, x*sn+y*cs
	x, y = x*b.sc+b.ox, y*b.sc+b.oy
	return float32(b.cx + x*b.h), float32(b.cy + y*b.h)
}

type pth struct {
	b *ib
	p *vector.Path
}

func (b *ib) np() *pth { return &pth{b, &vector.Path{}} }
func (q *pth) M(x, y float64) *pth {
	px, py := q.b.P(x, y)
	q.p.MoveTo(px, py)
	return q
}
func (q *pth) L(x, y float64) *pth {
	px, py := q.b.P(x, y)
	q.p.LineTo(px, py)
	return q
}
func (q *pth) C(x1, y1, x2, y2, x3, y3 float64) *pth {
	a, b := q.b.P(x1, y1)
	c, d := q.b.P(x2, y2)
	e, f := q.b.P(x3, y3)
	q.p.CubicTo(a, b, c, d, e, f)
	return q
}
func (q *pth) Q(x1, y1, x2, y2 float64) *pth {
	a, b := q.b.P(x1, y1)
	c, d := q.b.P(x2, y2)
	q.p.QuadTo(a, b, c, d)
	return q
}
func (q *pth) Z() *vector.Path    { q.p.Close(); return q.p }
func (q *pth) Open() *vector.Path { return q.p }

func (b *ib) poly(xy ...float64) *vector.Path {
	q := b.np().M(xy[0], xy[1])
	for i := 2; i+1 < len(xy); i += 2 {
		q.L(xy[i], xy[i+1])
	}
	return q.Z()
}

func (b *ib) ellipse(x, y, rx, ry float64) *vector.Path {
	q := b.np()
	n := 40
	for i := 0; i < n; i++ {
		a := float64(i) / float64(n) * 2 * math.Pi
		if i == 0 {
			q.M(x+math.Cos(a)*rx, y+math.Sin(a)*ry)
		} else {
			q.L(x+math.Cos(a)*rx, y+math.Sin(a)*ry)
		}
	}
	return q.Z()
}

func (b *ib) circle(x, y, r float64) *vector.Path { return b.ellipse(x, y, r, r) }

func (b *ib) rrect(x, y, w, h, r float64) *vector.Path {
	r = math.Min(r, math.Min(w, h)/2)
	q := b.np()
	seg := func(cx, cy, a0 float64) {
		for i := 0; i <= 6; i++ {
			a := a0 + float64(i)/6*math.Pi/2
			q.L(cx+math.Cos(a)*r, cy+math.Sin(a)*r)
		}
	}
	q.M(x+r, y)
	seg(x+w-r, y+r, -math.Pi/2)
	seg(x+w-r, y+h-r, 0)
	seg(x+r, y+h-r, math.Pi/2)
	seg(x+r, y+r, math.Pi)
	return q.Z()
}

func (b *ib) star(x, y, r1, r2 float64, n int, rot float64) *vector.Path {
	var xy []float64
	for i := 0; i < n*2; i++ {
		r := r1
		if i%2 == 1 {
			r = r2
		}
		a := rot - math.Pi/2 + float64(i)*math.Pi/float64(n)
		xy = append(xy, x+math.Cos(a)*r, y+math.Sin(a)*r)
	}
	return b.poly(xy...)
}

func (b *ib) unit() float64 { return b.h * b.sc }

// shape draws a dark outline then a gradient fill in the given tone.
func (b *ib) shape(p *vector.Path, tone color.RGBA) {
	u := b.unit()
	vStroke(b.img, p, 0.11*u, ca(darken(tone, 0.18), 0.95))
	vFill(b.img, p, lighten(tone, 0.32), darken(tone, 0.62), vgrad(b.cy-b.h, b.cy+b.h))
}

// flat fills without an outline (highlights, inner details).
func (b *ib) flat(p *vector.Path, col color.RGBA) { vFill(b.img, p, col, col, grad{}) }

// line strokes an outlined polyline.
func (b *ib) line(w float64, tone color.RGBA, xy ...float64) {
	q := b.np().M(xy[0], xy[1])
	for i := 2; i+1 < len(xy); i += 2 {
		q.L(xy[i], xy[i+1])
	}
	p := q.Open()
	u := b.unit()
	vStroke(b.img, p, (w+0.1)*u, ca(darken(tone, 0.18), 0.95))
	vStroke(b.img, p, w*u, tone)
}

func (b *ib) shine(x, y, r float64) {
	b.flat(b.ellipse(x, y, r, r*0.7), ca(toneWhite, 0.55))
}

// ---------------------------------------------------------------- glyphs

var glyphs map[string]func(b *ib)

func init() {
	heart := func(b *ib) *vector.Path {
		return b.np().M(0, 0.86).C(-0.55, 0.45, -1.0, 0.12, -0.96, -0.3).C(-0.9, -0.78, -0.3, -0.94, 0, -0.46).
			C(0.3, -0.94, 0.9, -0.78, 0.96, -0.3).C(1.0, 0.12, 0.55, 0.45, 0, 0.86).Z()
	}
	flame := func(b *ib, s, oy float64, tone color.RGBA) {
		p := b.np().M(0, -0.95*s+oy).C(0.35*s, -0.45*s+oy, 0.8*s, -0.1*s+oy, 0.62*s, 0.42*s+oy).C(0.5*s, 0.82*s+oy, 0.18*s, 0.95*s+oy, 0, 0.95*s+oy).
			C(-0.18*s, 0.95*s+oy, -0.5*s, 0.82*s+oy, -0.62*s, 0.42*s+oy).C(-0.8*s, -0.1*s+oy, -0.2*s, -0.4*s+oy, 0, -0.95*s+oy).Z()
		b.shape(p, tone)
	}
	bolt := func(b *ib, tone color.RGBA) {
		b.shape(b.poly(0.25, -0.95, -0.55, 0.1, -0.05, 0.1, -0.3, 0.95, 0.6, -0.15, 0.08, -0.15), tone)
	}
	snow := func(b *ib, tone color.RGBA, s float64) {
		for k := 0; k < 6; k++ {
			a := float64(k) * math.Pi / 3
			cs, sn := math.Cos(a), math.Sin(a)
			b.line(0.13*s, lighten(tone, 0.35), 0, 0, cs*0.86*s, sn*0.86*s)
			for _, d := range []float64{-0.7, 0.7} {
				bx, by := cs*0.5*s, sn*0.5*s
				b.line(0.1*s, lighten(tone, 0.35), bx, by, bx+math.Cos(a+d)*0.26*s, by+math.Sin(a+d)*0.26*s)
			}
		}
		b.shape(b.star(0, 0, 0.22*s, 0.13*s, 6, 0), toneWhite)
	}
	coin := func(b *ib, x, y, r float64) {
		b.shape(b.circle(x, y, r), toneGold)
		vStroke(b.img, b.circle(x, y, r*0.7), 0.07*b.unit(), ca(darken(toneGold, 0.6), 0.8))
		b.flat(b.star(x, y, r*0.42, r*0.18, 4, 0), lighten(toneGold, 0.6))
		b.shine(x-r*0.35, y-r*0.4, r*0.2)
	}
	gem := func(b *ib, tone color.RGBA) {
		b.shape(b.poly(-0.78, -0.22, -0.42, -0.7, 0.42, -0.7, 0.78, -0.22, 0, 0.88), tone)
		b.flat(b.poly(-0.42, -0.7, 0.42, -0.7, 0.24, -0.22, -0.24, -0.22), ca(toneWhite, 0.35))
		b.flat(b.poly(-0.24, -0.22, 0.24, -0.22, 0, 0.88), ca(toneWhite, 0.18))
		b.line(0.04, ca(darken(tone, 0.4), 0.8), -0.78, -0.22, 0.78, -0.22)
	}
	shield := func(b *ib, tone color.RGBA) {
		p := b.np().M(-0.75, -0.7).Q(0, -1.0, 0.75, -0.7).L(0.75, -0.05).C(0.75, 0.5, 0.32, 0.8, 0, 0.95).C(-0.32, 0.8, -0.75, 0.5, -0.75, -0.05).Z()
		b.shape(p, tone)
		q := b.np().M(-0.45, -0.5).Q(0, -0.68, 0.45, -0.5).L(0.45, -0.05).C(0.45, 0.32, 0.2, 0.55, 0, 0.65).C(-0.2, 0.55, -0.45, 0.32, -0.45, -0.05).Z()
		b.flat(q, ca(toneWhite, 0.22))
	}
	bomb := func(b *ib, fuseLen float64) {
		b.shape(b.circle(-0.05, 0.18, 0.66), color.RGBA{70, 70, 90, 255})
		b.shine(-0.3, -0.08, 0.16)
		b.shape(b.rrect(0.18, -0.55, 0.3, 0.26, 0.06), toneSteel)
		b.line(0.08, toneWood, 0.36, -0.55, 0.45, -0.75-fuseLen*0.1, 0.6, -0.8-fuseLen*0.08)
		b.shape(b.star(0.66, -0.82, 0.22, 0.09, 8, 0.2), toneGold)
	}
	swirl := func(b *ib, tone color.RGBA, s float64) {
		var xy []float64
		for t := 0.0; t < 3.4*math.Pi; t += 0.15 {
			r := (0.06 + 0.24*t/math.Pi) * s
			xy = append(xy, math.Cos(t)*r, math.Sin(t)*r)
		}
		b.line(0.14*s, tone, xy...)
		b.shape(b.circle(0, 0, 0.12*s), lighten(tone, 0.5))
	}
	glyphs = map[string]func(b *ib){
		"anvil": func(b *ib) {
			b.shape(b.poly(-0.9, -0.38, 0.55, -0.38, 0.92, -0.2, 0.55, -0.05, 0.3, -0.05, 0.26, 0.28, 0.55, 0.58, -0.55, 0.58, -0.26, 0.28, -0.3, -0.05, -0.62, -0.05), toneSteel)
			b.flat(b.poly(-0.85, -0.36, 0.5, -0.36, 0.6, -0.3, -0.85, -0.3), ca(toneWhite, 0.6))
			b.shape(b.star(0.15, -0.62, 0.2, 0.07, 6, 0), b.c)
		},
		"sword": func(b *ib) {
			b.rot = -math.Pi / 4
			b.sc *= 1.12
			b.shape(b.poly(-0.08, 0.25, -0.08, -0.72, 0, -0.95, 0.08, -0.72, 0.08, 0.25), toneSteel)
			b.flat(b.poly(0, -0.9, 0.06, -0.72, 0.06, 0.22, 0, 0.22), ca(toneWhite, 0.5))
			b.shape(b.rrect(-0.34, 0.2, 0.68, 0.13, 0.05), b.c)
			b.shape(b.rrect(-0.06, 0.33, 0.12, 0.38, 0.04), toneWood)
			b.shape(b.circle(0, 0.78, 0.1), b.c)
		},
		"paddle": func(b *ib) {
			b.shape(b.rrect(-0.86, 0.28, 1.72, 0.34, 0.16), b.c)
			b.shape(b.rrect(-0.92, 0.22, 0.32, 0.46, 0.1), toneGold)
			b.shape(b.rrect(0.6, 0.22, 0.32, 0.46, 0.1), toneGold)
			b.shape(b.circle(0, -0.32, 0.3), toneWhite)
			b.shine(-0.1, -0.42, 0.08)
			b.line(0.05, ca(toneWhite, 0.7), -0.5, -0.65, -0.35, -0.5)
			b.line(0.05, ca(toneWhite, 0.7), 0.5, -0.65, 0.35, -0.5)
		},
		"comet": func(b *ib) {
			for i, off := range []float64{-0.28, 0, 0.28} {
				w := []float64{0.1, 0.16, 0.1}[i]
				x0, y0 := -0.3+off*0.7, 0.3+off*0.7
				vStroke(b.img, b.np().M(x0, y0).L(x0+0.95, y0-0.95).Open(), w*b.unit(), ca(lighten(b.c, 0.45), 0.5-0.1*math.Abs(off)*3))
			}
			b.flat(b.circle(-0.3, 0.3, 0.62), ca(lighten(b.c, 0.5), 0.25))
			b.shape(b.circle(-0.3, 0.3, 0.44), b.c)
			b.flat(b.circle(-0.3, 0.3, 0.2), ca(toneWhite, 0.55))
			b.shine(-0.45, 0.12, 0.13)
		},
		"leaf": func(b *ib) {
			b.rot = 0.5
			b.shape(b.np().M(0, -0.92).C(0.72, -0.6, 0.76, 0.3, 0, 0.86).C(-0.76, 0.3, -0.72, -0.6, 0, -0.92).Z(), b.c)
			b.line(0.05, darken(b.c, 0.5), 0, -0.75, 0, 0.95)
			for _, y := range []float64{-0.4, -0.05, 0.3} {
				b.line(0.035, darken(b.c, 0.5), 0, y, 0.38, y-0.22)
				b.line(0.035, darken(b.c, 0.5), 0, y, -0.38, y-0.22)
			}
		},
		"flame": func(b *ib) {
			flame(b, 1, 0, b.c)
			flame(b, 0.55, 0.36, lighten(b.c, 0.45))
			b.flat(b.circle(0, 0.62, 0.14), ca(toneWhite, 0.85))
		},
		"blaze": func(b *ib) {
			flame(b, 0.9, 0.05, b.c)
			flame(b, 0.5, 0.38, lighten(b.c, 0.45))
			for _, p := range [][3]float64{{-0.75, -0.55, 0.1}, {0.72, -0.35, 0.12}, {0.55, -0.85, 0.08}} {
				b.shape(b.star(p[0], p[1], p[2]*1.6, p[2]*0.6, 4, 0), toneGold)
			}
		},
		"bolt": func(b *ib) { bolt(b, b.c) },
		"snow": func(b *ib) { snow(b, b.c, 1) },
		"heart": func(b *ib) {
			b.shape(heart(b), b.c)
			b.shine(-0.42, -0.4, 0.17)
		},
		"orb": func(b *ib) {
			b.shape(b.circle(0, 0, 0.8), b.c)
			vStroke(b.img, b.circle(0, 0, 0.56), 0.05*b.unit(), ca(toneWhite, 0.35))
			b.shine(-0.32, -0.34, 0.2)
		},
		"shield": func(b *ib) { shield(b, b.c) },
		"split": func(b *ib) {
			b.line(0.09, ca(lighten(b.c, 0.5), 0.9), 0, 0.45, -0.42, -0.3)
			b.line(0.09, ca(lighten(b.c, 0.5), 0.9), 0, 0.45, 0.42, -0.3)
			for _, p := range [][2]float64{{-0.45, -0.42}, {0.45, -0.42}, {0, 0.52}} {
				b.shape(b.circle(p[0], p[1], 0.3), b.c)
				b.shine(p[0]-0.1, p[1]-0.1, 0.08)
			}
		},
		"twinorb": func(b *ib) {
			b.shape(b.circle(-0.28, 0.16, 0.5), b.c)
			b.shine(-0.44, -0.02, 0.13)
			b.shape(b.circle(0.36, -0.22, 0.4), lighten(b.c, 0.35))
			b.shine(0.24, -0.36, 0.1)
		},
		"fist": func(b *ib) {
			b.shape(b.rrect(-0.6, -0.5, 1.15, 1.0, 0.28), b.c)
			for _, x := range []float64{-0.32, -0.02, 0.28} {
				b.line(0.04, darken(b.c, 0.45), x, -0.5, x, -0.1)
			}
			b.shape(b.rrect(-0.72, -0.05, 0.62, 0.32, 0.15), lighten(b.c, 0.15))
			b.shine(-0.3, -0.36, 0.12)
		},
		"bomb": func(b *ib) { bomb(b, 1) },
		"fuse": func(b *ib) { bomb(b, 3) },
		"arrow": func(b *ib) {
			b.line(0.1, toneWood, -0.72, 0.72, 0.5, -0.5)
			b.shape(b.poly(0.85, -0.85, 0.3, -0.72, 0.72, -0.3), b.c)
			b.shape(b.poly(-0.72, 0.72, -0.82, 0.38, -0.6, 0.6), toneWhite)
			b.shape(b.poly(-0.72, 0.72, -0.38, 0.82, -0.6, 0.6), toneWhite)
		},
		"swirl": func(b *ib) { swirl(b, b.c, 1) },
		"chain": func(b *ib) {
			for i, p := range [][2]float64{{-0.32, 0.2}, {0.32, -0.2}} {
				b.rot = []float64{0.6, 0.6}[i]
				vStroke(b.img, b.rrect(p[0]-0.42, p[1]-0.24, 0.84, 0.48, 0.24), 0.26*b.unit(), ca(darken(b.c, 0.2), 0.95))
				vStroke(b.img, b.rrect(p[0]-0.42, p[1]-0.24, 0.84, 0.48, 0.24), 0.15*b.unit(), lighten(b.c, 0.2))
			}
			b.rot = 0
		},
		"shatter": func(b *ib) {
			gem(b, b.c)
			b.line(0.04, toneWhite, -0.1, -0.7, 0.05, -0.3, -0.12, 0.05, 0.08, 0.4)
			b.line(0.035, toneWhite, 0.05, -0.3, 0.4, -0.2)
		},
		"gear": func(b *ib) {
			var xy []float64
			n := 40
			for i := 0; i < n; i++ {
				a := float64(i) / float64(n) * 2 * math.Pi
				r := 0.68
				if (i/2)%2 == 0 {
					r = 0.88
				}
				xy = append(xy, math.Cos(a)*r, math.Sin(a)*r)
			}
			b.shape(b.poly(xy...), b.c)
			b.shape(b.circle(0, 0, 0.3), toneDark)
			b.shine(-0.35, -0.42, 0.12)
		},
		"clover": func(b *ib) {
			b.line(0.09, darken(toneGreen, 0.2), 0.05, 0.1, 0.35, 0.92)
			for _, p := range [][2]float64{{-0.3, -0.3}, {0.3, -0.3}, {-0.3, 0.26}, {0.3, 0.26}} {
				b.shape(b.circle(p[0], p[1], 0.34), b.c)
			}
			b.shape(b.circle(0, -0.02, 0.12), lighten(b.c, 0.4))
		},
		"coinstack": func(b *ib) {
			for i, y := range []float64{0.55, 0.32, 0.09} {
				_ = i
				b.shape(b.rrect(-0.6, y-0.12, 1.2, 0.26, 0.13), toneGold)
			}
			coin(b, 0.35, -0.45, 0.42)
		},
		"coin": func(b *ib) { coin(b, 0, 0, 0.82) },
		"laser": func(b *ib) {
			b.flat(b.rrect(-0.1, -0.1, 1.0, 0.2, 0.1), ca(lighten(b.c, 0.5), 0.6))
			b.flat(b.rrect(-0.1, -0.04, 1.0, 0.08, 0.04), toneWhite)
			b.shape(b.rrect(-0.85, -0.3, 0.8, 0.6, 0.12), toneSteel)
			b.shape(b.rrect(-0.2, -0.18, 0.2, 0.36, 0.06), b.c)
			b.shape(b.star(0.88, 0, 0.22, 0.08, 4, 0), toneWhite)
		},
		"eye": func(b *ib) {
			b.shape(b.np().M(-0.92, 0).Q(0, -0.85, 0.92, 0).Q(0, 0.85, -0.92, 0).Z(), toneWhite)
			b.shape(b.circle(0, 0, 0.38), b.c)
			b.flat(b.circle(0, 0, 0.16), toneDark)
			b.shine(-0.12, -0.14, 0.08)
		},
		"feather": func(b *ib) {
			b.rot = 0.6
			b.shape(b.np().M(0, -0.95).C(0.42, -0.6, 0.38, 0.35, 0.05, 0.7).L(-0.05, 0.7).C(-0.38, 0.35, -0.42, -0.6, 0, -0.95).Z(), b.c)
			b.line(0.05, lighten(b.c, 0.6), 0, -0.85, 0, 0.95)
			for _, y := range []float64{-0.5, -0.2, 0.1, 0.4} {
				b.line(0.025, darken(b.c, 0.4), 0, y, 0.28, y-0.18)
				b.line(0.025, darken(b.c, 0.4), 0, y, -0.28, y-0.18)
			}
			b.rot = 0
		},
		"volcano": func(b *ib) {
			b.shape(b.poly(-0.92, 0.8, -0.25, -0.25, 0.25, -0.25, 0.92, 0.8), color.RGBA{120, 80, 70, 255})
			b.shape(b.poly(-0.25, -0.25, 0.25, -0.25, 0.32, -0.05, 0.1, 0.25, -0.05, 0.05, -0.18, 0.3, -0.32, -0.05), toneLava)
			for _, p := range [][3]float64{{-0.3, -0.65, 0.12}, {0.15, -0.8, 0.14}, {0.45, -0.55, 0.1}} {
				b.shape(b.circle(p[0], p[1], p[2]), lighten(toneLava, 0.3))
			}
		},
		"glass": func(b *ib) {
			b.shape(b.circle(0, 0, 0.8), ca(lighten(b.c, 0.3), 0.85))
			b.line(0.05, toneWhite, -0.1, -0.78, 0.05, -0.3, -0.2, 0.05, 0.1, 0.45, 0, 0.78)
			b.line(0.04, toneWhite, 0.05, -0.3, 0.5, -0.2)
			b.line(0.04, toneWhite, -0.2, 0.05, -0.6, 0.2)
			b.shine(-0.35, -0.4, 0.15)
		},
		"magnet": func(b *ib) {
			var xy []float64
			for a := 0.0; a <= math.Pi+0.01; a += math.Pi / 16 {
				xy = append(xy, math.Cos(a)*0.5, 0.05+math.Sin(a)*0.5)
			}
			// draw as a thick U opening upward
			for i := 1; i < len(xy); i += 2 {
				xy[i] = -xy[i] + 0.1
			}
			_ = xy
			q := b.np().M(-0.75, -0.7)
			for a := math.Pi; a >= 0; a -= math.Pi / 16 {
				q.L(math.Cos(a)*0.75, 0.05+math.Sin(a)*0.75)
			}
			q.L(0.75, -0.7).L(0.35, -0.7)
			for a := 0.0; a <= math.Pi+0.01; a += math.Pi / 16 {
				q.L(math.Cos(a)*0.35, 0.05+math.Sin(a)*0.35)
			}
			q.L(-0.35, -0.7)
			b.shape(q.Z(), toneRed)
			b.shape(b.rrect(-0.77, -0.85, 0.44, 0.28, 0.04), toneSteel)
			b.shape(b.rrect(0.33, -0.85, 0.44, 0.28, 0.04), toneSteel)
		},
		"wave": func(b *ib) {
			for i, r := range []float64{0.3, 0.58, 0.86} {
				var xy []float64
				for a := -0.9; a <= 0.9; a += 0.1 {
					xy = append(xy, -0.55+math.Cos(a)*r, math.Sin(a)*r)
				}
				b.line(0.12-0.02*float64(i), lighten(b.c, 0.15*float64(i)), xy...)
			}
			b.shape(b.circle(-0.62, 0, 0.14), b.c)
		},
		"crown": func(b *ib) {
			b.shape(b.poly(-0.82, 0.48, -0.82, -0.3, -0.45, 0.05, 0, -0.62, 0.45, 0.05, 0.82, -0.3, 0.82, 0.48), toneGold)
			b.shape(b.rrect(-0.82, 0.32, 1.64, 0.22, 0.04), lighten(toneGold, 0.2))
			for _, x := range []float64{-0.45, 0, 0.45} {
				b.shape(b.circle(x, 0.43, 0.08), b.c)
			}
			b.shape(b.circle(0, -0.66, 0.1), b.c)
		},
		"moon": func(b *ib) {
			// crescent: outer arc on the left, inner arc of an offset circle
			const r1, r2, d = 0.84, 0.7, 0.4
			ix := (r1*r1 - r2*r2 + d*d) / (2 * d)
			iy := math.Sqrt(r1*r1 - ix*ix)
			ao := math.Atan2(iy, ix)
			ai := math.Atan2(iy, ix-d)
			q := b.np()
			n := 32
			for i := 0; i <= n; i++ {
				a := ao + float64(i)/float64(n)*(2*math.Pi-2*ao)
				if i == 0 {
					q.M(math.Cos(a)*r1, math.Sin(a)*r1)
				} else {
					q.L(math.Cos(a)*r1, math.Sin(a)*r1)
				}
			}
			for i := 0; i <= n; i++ {
				a := (2*math.Pi - ai) - float64(i)/float64(n)*(2*math.Pi-2*ai)
				q.L(d+math.Cos(a)*r2, math.Sin(a)*r2)
			}
			b.rot = -0.5
			b.shape(q.Z(), toneRed)
			b.rot = 0
			b.shape(b.star(0.45, -0.45, 0.16, 0.06, 4, 0), toneWhite)
		},
		"storm": func(b *ib) {
			cl := [][3]float64{{-0.42, -0.2, 0.32}, {0.0, -0.38, 0.4}, {0.42, -0.18, 0.32}, {0, -0.05, 0.36}}
			for _, c := range cl {
				vStroke(b.img, b.circle(c[0], c[1], c[2]), 0.11*b.unit(), ca(darken(toneSteel, 0.3), 0.95))
			}
			for _, c := range cl {
				vFill(b.img, b.circle(c[0], c[1], c[2]), lighten(toneSteel, 0.4), darken(toneSteel, 0.85), vgrad(b.cy-b.h, b.cy+b.h*0.2))
			}
			b.sc *= 0.55
			b.oy += 0.42
			bolt(b, b.c)
		},
		"gem": func(b *ib) { gem(b, b.c) },
		"iceheart": func(b *ib) {
			b.shape(heart(b), toneIce)
			b.sc *= 0.45
			b.oy -= 0.05
			snow(b, color.RGBA{90, 170, 230, 255}, 1)
		},
		"halo": func(b *ib) {
			vStroke(b.img, b.ellipse(0, -0.35, 0.72, 0.26), 0.26*b.unit(), ca(darken(toneGold, 0.2), 0.95))
			vStroke(b.img, b.ellipse(0, -0.35, 0.72, 0.26), 0.16*b.unit(), lighten(toneGold, 0.2))
			b.shape(b.star(0, 0.35, 0.42, 0.16, 4, 0), b.c)
		},
		"hydra": func(b *ib) {
			heads := [][2]float64{{-0.58, -0.45}, {0, -0.68}, {0.58, -0.45}}
			for _, h := range heads {
				b.line(0.18, b.c, 0, 0.85, h[0]*0.6, 0.2, h[0], h[1])
			}
			for _, h := range heads {
				b.shape(b.ellipse(h[0], h[1], 0.2, 0.15), lighten(b.c, 0.15))
				b.flat(b.circle(h[0]+0.07, h[1]-0.03, 0.04), toneGold)
			}
		},
		"fang": func(b *ib) {
			b.shape(b.rrect(-0.75, -0.75, 1.5, 0.32, 0.14), b.c)
			b.shape(b.poly(-0.55, -0.45, -0.08, -0.45, -0.32, 0.8), toneWhite)
			b.shape(b.poly(0.08, -0.45, 0.55, -0.45, 0.32, 0.8), toneWhite)
		},
		"urn": func(b *ib) {
			b.shape(b.np().M(-0.35, -0.45).L(0.35, -0.45).C(0.78, -0.15, 0.72, 0.62, 0.3, 0.82).L(-0.3, 0.82).C(-0.72, 0.62, -0.78, -0.15, -0.35, -0.45).Z(), b.c)
			b.shape(b.rrect(-0.42, -0.6, 0.84, 0.18, 0.06), toneGold)
			b.shape(b.rrect(-0.5, 0.1, 1.0, 0.12, 0.04), toneGold)
			b.sc *= 0.35
			b.oy -= 0.85 / 0.35 * 0.35
			flame(b, 1, 0, toneLava)
		},
		"battery": func(b *ib) {
			b.shape(b.rrect(-0.45, -0.62, 0.9, 1.45, 0.12), toneSteel)
			b.shape(b.rrect(-0.18, -0.8, 0.36, 0.2, 0.05), toneSteel)
			b.sc *= 0.55
			b.oy += 0.1
			bolt(b, b.c)
		},
		"mask": func(b *ib) {
			b.shape(b.np().M(0, -0.9).C(0.75, -0.9, 0.8, 0.1, 0.45, 0.6).Q(0, 1.0, -0.45, 0.6).C(-0.8, 0.1, -0.75, -0.9, 0, -0.9).Z(), b.c)
			b.flat(b.poly(-0.55, -0.2, -0.15, -0.28, -0.2, -0.08, -0.5, -0.05), toneDark)
			b.flat(b.poly(0.55, -0.2, 0.15, -0.28, 0.2, -0.08, 0.5, -0.05), toneDark)
			b.flat(b.rrect(-0.2, 0.35, 0.4, 0.1, 0.05), toneDark)
			b.shape(b.star(0, -0.62, 0.14, 0.06, 4, 0), toneGold)
		},
		"mirror": func(b *ib) {
			b.line(0.14, toneGold, 0, 0.45, 0, 0.92)
			b.shape(b.ellipse(0, -0.2, 0.55, 0.68), toneGold)
			b.flat(b.ellipse(0, -0.2, 0.43, 0.56), lighten(b.c, 0.45))
			b.line(0.07, ca(toneWhite, 0.8), -0.2, -0.55, 0.15, -0.1)
		},
		"blackhole": func(b *ib) {
			b.rot = -0.35
			vStroke(b.img, b.ellipse(0, 0, 0.92, 0.36), 0.24*b.unit(), ca(toneLava, 0.9))
			vStroke(b.img, b.ellipse(0, 0, 0.92, 0.36), 0.1*b.unit(), lighten(toneGold, 0.4))
			b.rot = 0
			b.shape(b.circle(0, 0, 0.44), toneDark)
			vStroke(b.img, b.circle(0, 0, 0.47), 0.06*b.unit(), b.c)
			b.rot = -0.35
			b.flat(b.poly(-0.92, 0, 0.92, 0, 0.88, 0.12, -0.88, 0.12), ca(lighten(toneGold, 0.3), 0.7))
			b.rot = 0
		},
		"ricochet": func(b *ib) {
			b.line(0.12, b.c, -0.82, 0.6, -0.3, -0.45, 0.2, 0.42, 0.6, -0.5)
			b.shape(b.poly(0.85, -0.8, 0.42, -0.62, 0.72, -0.32), lighten(b.c, 0.3))
			b.shape(b.circle(-0.82, 0.62, 0.17), toneWhite)
		},
		"drop": func(b *ib) {
			b.shape(b.np().M(0, -0.92).C(0.35, -0.4, 0.68, -0.05, 0.62, 0.35).C(0.55, 0.78, -0.55, 0.78, -0.62, 0.35).C(-0.68, -0.05, -0.35, -0.4, 0, -0.92).Z(), b.c)
			b.shine(-0.25, 0.15, 0.13)
		},
		"flask": func(b *ib) {
			b.shape(b.circle(0, 0.32, 0.58), ca(lighten(b.c, 0.65), 0.95))
			b.flat(b.np().M(-0.56, 0.42).C(-0.4, 0.95, 0.4, 0.95, 0.56, 0.42).Z(), b.c)
			b.shape(b.rrect(-0.17, -0.72, 0.34, 0.55, 0.06), lighten(b.c, 0.65))
			b.shape(b.rrect(-0.24, -0.85, 0.48, 0.16, 0.06), toneWood)
			b.shine(-0.25, 0.12, 0.1)
		},
		"axe": func(b *ib) {
			b.line(0.12, toneWood, -0.6, 0.85, 0.35, -0.6)
			b.shape(b.np().M(0.05, -0.5).C(0.5, -0.95, 0.95, -0.6, 0.9, -0.05).C(0.6, -0.2, 0.4, -0.15, 0.25, -0.15).Z(), toneSteel)
			b.flat(b.np().M(0.85, -0.1).C(0.9, -0.5, 0.6, -0.8, 0.35, -0.75).L(0.38, -0.68).C(0.6, -0.7, 0.82, -0.45, 0.78, -0.1).Z(), ca(toneWhite, 0.7))
		},
		"hourglass": func(b *ib) {
			b.shape(b.poly(-0.5, -0.72, 0.5, -0.72, 0.08, 0, 0.5, 0.72, -0.5, 0.72, -0.08, 0), ca(lighten(toneIce, 0.3), 0.9))
			b.flat(b.poly(-0.3, -0.45, 0.3, -0.45, 0.05, -0.08, -0.05, -0.08), b.c)
			b.flat(b.poly(0, 0.3, 0.42, 0.68, -0.42, 0.68), b.c)
			b.shape(b.rrect(-0.68, -0.9, 1.36, 0.2, 0.06), toneGold)
			b.shape(b.rrect(-0.68, 0.7, 1.36, 0.2, 0.06), toneGold)
		},
		"tag": func(b *ib) {
			b.rot = -0.5
			b.shape(b.poly(-0.82, 0, -0.42, -0.5, 0.78, -0.5, 0.78, 0.5, -0.42, 0.5), b.c)
			b.flat(b.circle(-0.38, 0, 0.12), toneDark)
			b.rot = 0
			b.sc *= 0.36
			b.ox += 0.18
			coin(b, 0, 0, 0.9)
		},
		"lens": func(b *ib) {
			b.line(0.2, toneWood, 0.3, 0.3, 0.82, 0.82)
			b.flat(b.circle(-0.15, -0.15, 0.55), ca(lighten(b.c, 0.5), 0.6))
			vStroke(b.img, b.circle(-0.15, -0.15, 0.58), 0.25*b.unit(), ca(darken(toneSteel, 0.3), 0.95))
			vStroke(b.img, b.circle(-0.15, -0.15, 0.58), 0.14*b.unit(), toneSteel)
			b.shine(-0.38, -0.38, 0.12)
		},
		"target": func(b *ib) {
			b.shape(b.circle(0, 0, 0.84), b.c)
			b.flat(b.circle(0, 0, 0.6), toneWhite)
			b.flat(b.circle(0, 0, 0.38), b.c)
			b.flat(b.circle(0, 0, 0.15), toneWhite)
			b.line(0.05, toneDark, -0.95, 0, -0.5, 0)
			b.line(0.05, toneDark, 0.5, 0, 0.95, 0)
			b.line(0.05, toneDark, 0, -0.95, 0, -0.5)
			b.line(0.05, toneDark, 0, 0.5, 0, 0.95)
		},
		"burst": func(b *ib) {
			b.shape(b.star(0, 0, 0.92, 0.42, 10, 0.1), b.c)
			b.shape(b.circle(0, 0, 0.3), lighten(b.c, 0.5))
		},
		"hive": func(b *ib) {
			hex := func(x, y, r float64) *vector.Path {
				var xy []float64
				for i := 0; i < 6; i++ {
					a := float64(i)*math.Pi/3 + math.Pi/6
					xy = append(xy, x+math.Cos(a)*r, y+math.Sin(a)*r)
				}
				return b.poly(xy...)
			}
			for i, p := range [][2]float64{{0, 0}, {0.52, -0.3}, {-0.52, -0.3}, {0, -0.6}, {0.52, 0.3}, {-0.52, 0.3}, {0, 0.6}} {
				t := b.c
				if i == 0 {
					t = toneGold
				}
				b.shape(hex(p[0], p[1], 0.3), t)
			}
		},
		"book": func(b *ib) {
			b.shape(b.poly(-0.9, -0.55, 0, -0.42, 0.9, -0.55, 0.9, 0.62, 0, 0.75, -0.9, 0.62), b.c)
			b.shape(b.poly(-0.8, -0.62, 0, -0.5, 0, 0.62, -0.8, 0.5), toneWhite)
			b.shape(b.poly(0.8, -0.62, 0, -0.5, 0, 0.62, 0.8, 0.5), toneWhite)
			for _, y := range []float64{-0.3, -0.1, 0.1, 0.3} {
				b.flat(b.poly(-0.65, y, -0.15, y+0.06, -0.15, y+0.09, -0.65, y+0.03), ca(toneDark, 0.4))
				b.flat(b.poly(0.65, y, 0.15, y+0.06, 0.15, y+0.09, 0.65, y+0.03), ca(toneDark, 0.4))
			}
		},
		"note": func(b *ib) {
			b.shape(b.poly(-0.42, -0.62, 0.62, -0.86, 0.62, -0.6, -0.42, -0.36), toneGold)
			b.line(0.1, toneGold, -0.42, 0.5, -0.42, -0.5)
			b.line(0.1, toneGold, 0.62, 0.26, 0.62, -0.74)
			b.shape(b.ellipse(-0.62, 0.55, 0.3, 0.22), b.c)
			b.shape(b.ellipse(0.42, 0.31, 0.3, 0.22), b.c)
			b.shine(-0.7, 0.48, 0.07)
			b.shine(0.34, 0.24, 0.07)
		},
		"drum": func(b *ib) {
			b.shape(b.rrect(-0.7, -0.3, 1.4, 0.95, 0.12), b.c)
			b.line(0.05, toneGold, -0.7, -0.15, -0.3, 0.6, 0.1, -0.15, 0.5, 0.6)
			b.shape(b.ellipse(0, -0.3, 0.7, 0.2), toneWhite)
			b.line(0.07, toneWood, -0.2, -0.5, -0.75, -0.95)
			b.line(0.07, toneWood, 0.2, -0.5, 0.75, -0.95)
		},
		"ring": func(b *ib) {
			vStroke(b.img, b.ellipse(0, 0.2, 0.62, 0.58), 0.28*b.unit(), ca(darken(toneGold, 0.2), 0.95))
			vStroke(b.img, b.ellipse(0, 0.2, 0.62, 0.58), 0.17*b.unit(), lighten(toneGold, 0.2))
			gemB := *b
			gemB.sc *= 0.38
			gemB.oy -= 0.55
			gem(&gemB, b.c)
		},
		"infinity": func(b *ib) {
			var xy []float64
			for t := 0.0; t <= 2*math.Pi+0.01; t += 0.08 {
				d := 1 + math.Sin(t)*math.Sin(t)
				xy = append(xy, 0.85*math.Cos(t)/d, 0.85*math.Sin(t)*math.Cos(t)/d)
			}
			b.line(0.16, b.c, xy...)
		},
		"dragonheart": func(b *ib) {
			b.shape(heart(b), toneRed)
			b.shine(-0.42, -0.4, 0.15)
			b.sc *= 0.45
			b.oy += 0.08
			flame(b, 1, 0, toneLava)
		},
		"star": func(b *ib) {
			b.shape(b.star(0, 0.05, 0.92, 0.4, 5, 0), b.c)
			b.shine(-0.15, -0.25, 0.12)
		},
	}
}

// ---------------------------------------------------------------- baking

var emblemCache = map[string]*ebiten.Image{}

func bakeEmblem(key string, draw func(img *ebiten.Image, b *ib)) *ebiten.Image {
	if img, ok := emblemCache[key]; ok {
		return img
	}
	img := ebiten.NewImage(emblemRes, emblemRes)
	b := &ib{img: img, cx: emblemRes / 2, cy: emblemRes / 2, h: emblemRes * 0.42, sc: 1}
	draw(img, b)
	emblemCache[key] = img
	return img
}

// glyphImage bakes a glyph tinted with col.
func glyphImage(glyph string, col color.RGBA) *ebiten.Image {
	return bakeEmblem(glyph+colKey(col), func(img *ebiten.Image, b *ib) {
		b.c = col
		if f, ok := glyphs[glyph]; ok {
			f(b)
		} else {
			glyphs["orb"](b)
		}
	})
}

// modColor is the colour an item's emblem is tinted with.
func modColor(m *Mod) color.RGBA {
	if len(m.Tags) > 0 {
		if c, ok := tagColors[m.Tags[0]]; ok {
			return c
		}
	}
	return rarityColors[m.Rarity]
}

func modEmblem(m *Mod) *ebiten.Image { return glyphImage(m.Icon, modColor(m)) }

// fusionEmblem overlaps both parents' glyphs under a sparkle.
func fusionEmblem(f *Fusion) *ebiten.Image {
	a, bm := modByID[f.A], modByID[f.B]
	return bakeEmblem("fusion:"+f.ID, func(img *ebiten.Image, b *ib) {
		b.c = f.Col
		if a != nil {
			b.sc, b.ox, b.oy = 0.66, -0.3, -0.22
			glyphs[a.Icon](b)
			b.rot = 0
		}
		if bm != nil {
			b.sc, b.ox, b.oy = 0.66, 0.3, 0.26
			b.c = modColor(bm)
			glyphs[bm.Icon](b)
			b.rot = 0
		}
		b.sc, b.ox, b.oy = 1, 0, 0
		b.shape(b.star(0.62, -0.68, 0.26, 0.09, 4, 0), f.Col)
	})
}

// drawEmblemDark draws an emblem as a dark silhouette (undiscovered items).
func (u *UI) drawEmblemDark(dst *ebiten.Image, img *ebiten.Image, x, y, size, a float64) {
	sc := size * u.S / float64(emblemRes)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-emblemRes/2, -emblemRes/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(u.X(x), u.Y(y))
	op.ColorScale.Scale(0.16, 0.14, 0.24, 1)
	op.ColorScale.ScaleAlpha(float32(clamp01(a)))
	dst.DrawImage(img, op)
}

// drawEmblem draws an emblem image centred at virtual (x, y) with size px.
func (u *UI) drawEmblem(dst *ebiten.Image, img *ebiten.Image, x, y, size, a float64) {
	sc := size * u.S / float64(emblemRes)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-emblemRes/2, -emblemRes/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(u.X(x), u.Y(y))
	op.ColorScale.ScaleAlpha(float32(clamp01(a)))
	dst.DrawImage(img, op)
}
