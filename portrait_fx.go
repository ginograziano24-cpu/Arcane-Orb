package main

import (
	"encoding/json"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Live effects layered over the pre-rendered class portraits so they feel
// alive: breathing, parallax, pulsing eyes, flickering fire, crackling
// lightning, orbiting sparks, steam and glints. Anchor positions come from
// the portrait renderer (assets/portraits/<id>.json).

type pAnchor struct {
	Kind          string
	X, Y          float64
	R, G, B, Size float64
}

func (a *pAnchor) UnmarshalJSON(b []byte) error {
	var m struct {
		Kind          string
		X, Y          float64
		R, G, B, Size float64
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*a = pAnchor{m.Kind, m.X, m.Y, m.R, m.G, m.B, m.Size}
	return nil
}

func (a pAnchor) col() color.RGBA {
	return rgba(uint8(clamp01(a.R)*255), uint8(clamp01(a.G)*255), uint8(clamp01(a.B)*255), 255)
}

var anchorCache = map[string][]pAnchor{}

func portraitAnchors(id string) []pAnchor {
	if a, ok := anchorCache[id]; ok {
		return a
	}
	var out []pAnchor
	if b, err := portraitFS.ReadFile("assets/portraits/" + id + ".json"); err == nil {
		json.Unmarshal(b, &out)
	}
	anchorCache[id] = out
	return out
}

// boltState keeps a lightning bolt alive for a few frames.
type boltState struct {
	pts  [][2]float64
	life float64
	next float64
}

type portraitFX struct {
	bolts map[int]*boltState
	rng   *rand.Rand
}

var pfx = portraitFX{bolts: map[int]*boltState{}, rng: rand.New(rand.NewSource(7))}

// portraitXform maps normalised portrait coords to virtual screen coords,
// including breathing and parallax.
type portraitXform struct {
	px, py, pw, ph float64
	sx, sy, dx, dy float64
}

func (x portraitXform) at(nx, ny float64) (float64, float64) {
	cx, by := x.px+x.pw/2, x.py+x.ph
	return cx + (nx*x.pw-x.pw/2)*x.sx + x.dx, by - (x.ph-ny*x.ph)*x.sy + x.dy
}

func (g *Game) drawPortraitFX(s *ebiten.Image, id string, xf portraitXform, alpha float64) {
	if alpha <= 0.01 {
		return
	}
	u := g.ui
	S := u.S
	t := g.stateT
	glow := func(x, y, r float64, c color.RGBA, a float64) {
		if a <= 0 {
			return
		}
		drawSprite(s, sprSoft, u.X(x-r), u.Y(y-r), 2*r*S, 2*r*S, ca(c, math.Min(1, a*alpha)), true)
	}
	streak := func(x, y, w, h float64, c color.RGBA, a float64) {
		drawSprite(s, sprSoft, u.X(x-w), u.Y(y-h), 2*w*S, 2*h*S, ca(c, math.Min(1, a*alpha)), true)
	}
	flick := func(ph float64) float64 { // organic flicker 0..1
		return 0.5 + 0.22*math.Sin(t*11+ph) + 0.17*math.Sin(t*17.3+ph*2) + 0.11*math.Sin(t*29.1+ph*3)
	}
	sc := xf.ph / 1010
	for i, a := range portraitAnchors(id) {
		x, y := xf.at(a.X, a.Y)
		c := a.col()
		ph := float64(i) * 1.7
		sz := a.Size * sc
		switch a.Kind {
		case "eye":
			p := 0.55 + 0.45*math.Sin(t*1.9+ph*0.3)
			glow(x, y, 26*sz, c, 0.35*p)
			glow(x, y, 7*sz, lighten(c, 0.6), 0.5*p)
		case "orb":
			p := 0.75 + 0.25*math.Sin(t*3.1)
			glow(x, y, 120*sz, c, 0.35*p)
			glow(x, y, 46*sz, lighten(c, 0.5), 0.45*p)
			for k := 0; k < 12; k++ {
				fk := float64(k)
				an := t*(0.9+0.15*fk) + fk*0.52
				rx := (55 + 22*math.Sin(fk*1.3)) * sz
				ox, oy := math.Cos(an)*rx, math.Sin(an)*rx*0.38+math.Sin(t+fk)*6*sz
				front := math.Sin(an) > 0
				a2 := 0.5
				if !front {
					a2 = 0.22
				}
				glow(x+ox, y+oy, 6*sz, lighten(c, 0.4), a2)
			}
		case "core":
			p := 0.6 + 0.4*math.Sin(t*1.6+ph)
			glow(x, y, 80*sz, c, 0.3*p)
			glow(x, y, 26*sz, lighten(c, 0.6), 0.4*p)
		case "glint":
			// a star glint every few seconds
			k := math.Mod(t*0.37+ph*0.13, 1)
			if k < 0.08 {
				e := math.Sin(k / 0.08 * math.Pi)
				streak(x, y, 46*sz*e, 2.2*sz, lighten(c, 0.5), 0.9*e)
				streak(x, y, 2.2*sz, 30*sz*e, lighten(c, 0.5), 0.9*e)
				glow(x, y, 14*sz, cWhite, 0.8*e)
			}
		case "ember":
			f := flick(ph)
			glow(x, y, 34*sz, c, 0.4*f)
			for k := 0; k < 8; k++ {
				fk := float64(k)
				life := math.Mod(t*0.55+fk*0.137, 1)
				ex := x + math.Sin(fk*2.1+t*1.3)*12*sz*life
				ey := y - life*110*sz
				glow(ex, ey, 4*sz*(1-life*0.6), lighten(c, 0.3), 0.8*math.Sin(life*math.Pi))
			}
		case "fire":
			f := flick(ph)
			glow(x, y-20*sz, 150*sz, c, 0.28*f)
			glow(x, y, 60*sz, rgba(255, 200, 90, 255), 0.5*f)
			for k := 0; k < 26; k++ {
				fk := float64(k)
				life := math.Mod(t*(0.9+0.4*math.Mod(fk*0.37, 1))+fk*0.0731, 1)
				spread := (math.Mod(fk*0.618, 1) - 0.5) * 50 * sz * (1 - life)
				fx := x + spread + math.Sin(t*5+fk)*8*sz*life
				fy := y - 10*sz - life*170*sz
				fc := mixc(rgba(255, 230, 140, 255), rgba(255, 70, 20, 255), life)
				glow(fx, fy, (22-16*life)*sz, fc, 0.55*(1-life))
			}
		case "coin":
			p := 0.6 + 0.4*math.Sin(t*2.4)
			glow(x, y, 50*sz, c, 0.35*p)
			k := math.Mod(t*0.55, 1)
			if k < 0.12 {
				e := math.Sin(k / 0.12 * math.Pi)
				streak(x, y, 52*sz*e, 2.4*sz, lighten(c, 0.6), e)
				streak(x, y, 2.4*sz, 34*sz*e, lighten(c, 0.6), e)
			}
		case "steam":
			for k := 0; k < 7; k++ {
				fk := float64(k)
				life := math.Mod(t*0.22+fk/7, 1)
				sx := x + math.Sin(fk*1.9+t*0.7)*14*sz + life*30*sz
				sy := y - life*140*sz
				drawSprite(s, sprSoft, u.X(sx-(10+50*life)*sz), u.Y(sy-(10+50*life)*sz), (20+100*life)*sz*S, (20+100*life)*sz*S,
					ca(c, 0.18*math.Sin(life*math.Pi)*alpha), false)
			}
		case "bolt", "spark":
			big := a.Kind == "bolt"
			glow(x, y, 70*sz, c, 0.35*(0.7+0.3*math.Sin(t*7+ph)))
			b := pfx.bolts[i]
			if b == nil {
				b = &boltState{}
				pfx.bolts[i] = b
			}
			b.life -= u.DT
			b.next -= u.DT
			if b.next <= 0 {
				reach := 120.0
				if big {
					reach = 230
				}
				ang := -math.Pi/2 + (pfx.rng.Float64()-0.5)*2.6
				if !big {
					ang = pfx.rng.Float64() * 2 * math.Pi
				}
				l := (0.5 + 0.5*pfx.rng.Float64()) * reach * sz
				b.pts = jagged2(x, y, x+math.Cos(ang)*l, y+math.Sin(ang)*l, 5, pfx.rng)
				b.life = 0.14 + pfx.rng.Float64()*0.1
				b.next = 0.25 + pfx.rng.Float64()*1.4
				if !big {
					b.next = 0.15 + pfx.rng.Float64()*0.6
				}
			}
			if b.life > 0 && len(b.pts) > 1 {
				k := b.life / 0.2
				for j := 0; j+1 < len(b.pts); j++ {
					p0, p1 := b.pts[j], b.pts[j+1]
					ln := linePath(u.X(p0[0]), u.Y(p0[1]), u.X(p1[0]), u.Y(p1[1]))
					strokePath(s, ln, 9*sz*S, ca(c, 0.25*k*alpha), true)
					strokePath(s, ln, 2.6*sz*S, ca(lighten(c, 0.75), 0.95*math.Min(1, k)*alpha), true)
				}
				glow(x, y, 110*sz, c, 0.4*k)
			}
		}
	}
}

// jagged2 builds a 2D midpoint-displaced lightning polyline.
func jagged2(x0, y0, x1, y1 float64, depth int, r *rand.Rand) [][2]float64 {
	pts := [][2]float64{{x0, y0}, {x1, y1}}
	disp := math.Hypot(x1-x0, y1-y0) * 0.25
	for d := 0; d < depth; d++ {
		var np [][2]float64
		for i := 0; i+1 < len(pts); i++ {
			a, b := pts[i], pts[i+1]
			m := [2]float64{(a[0]+b[0])/2 + (r.Float64()-0.5)*disp, (a[1]+b[1])/2 + (r.Float64()-0.5)*disp}
			np = append(np, a, m)
		}
		np = append(np, pts[len(pts)-1])
		pts = np
		disp *= 0.55
	}
	return pts
}
