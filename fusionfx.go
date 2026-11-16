package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The forge: when two items fuse, the game stops for a short show instead
// of a banner. The two items' emblems fly in from either side and spiral
// round each other, collide in a flash, and the fusion's emblem rises out
// of the light with its name. Then the payoff is spelled out: every number
// the fusion itself adds ("Pierce 1 → 3"), counting up one line at a time,
// or, for a fusion that changes how things work rather than a number, a
// "NEW PASSIVE" tag with what it does. A click (or Space/Enter/A) skips
// to the end; the next click closes it. Several fusions at once play one
// after another.

const (
	forgeIn      = 0.30 // the screen darkens
	forgeCollide = 1.15 // the two emblems meet
	forgeName    = 1.45
	forgeGains   = 1.85 // first stat line
	forgeStep    = 0.22 // between stat lines
	forgeHold    = 1.6  // after the last line, before it closes by itself
	forgeOut     = 0.35 // fade out
)

type forgeAnim struct {
	f      *Fusion
	t      float64
	gains  []statChange // what the fusion itself adds
	ticked int          // stat lines whose sound has played
	boomed bool
	out    float64 // > 0 while closing
	skip   bool    // jumped to the end
}

// forgeQueue holds the fusions waiting for their show.
func (g *Game) queueForge(f *Fusion) {
	g.forges = append(g.forges, &forgeAnim{f: f, gains: g.run.fusionGains(f, &g.save, 4)})
}

// cinematic reports whether this run plays presentation moments (not the
// attract-mode demo, a duel, the trailer, a benchmark or a simulation).
func (g *Game) cinematic() bool {
	r := g.run
	if r == nil || r.Duel || bench != nil || inputHook != nil {
		return false
	}
	for _, code := range noShowRuns {
		if r.SeedCode == code {
			return false
		}
	}
	return true
}

// noShowRuns are the run codes of demos, tests and captures.
var noShowRuns = strings.Split("MENU,SIM,CAMP,TRAILER,LIFE,LIFE2,SAVEME,WAVES,BENCH,CHAMBER-REVIEW,HUD-REVIEW", ",")

// fusionGains is what fusion f adds on its own to the run's stats: the
// stats now, against the same build with only that fusion left out.
func (r *Run) fusionGains(f *Fusion, meta *SaveData, n int) []statChange {
	if r == nil || f == nil || r.Arch == nil {
		return nil
	}
	c := *r
	c.Mods = make(map[string]int, len(r.Mods))
	for k, v := range r.Mods {
		c.Mods[k] = v
	}
	c.Order = append([]string(nil), r.Order...)
	c.Active = append([]string(nil), r.Active...)
	c.Fused = append([]string(nil), r.Fused...)
	c.NewFusions = nil
	c.skipFusion = f.ID
	c.Recompute(meta)
	var out []statChange
	for _, row := range statRows {
		a, b := row.Get(&c.Stats), row.Get(&r.Stats)
		from, to := fmtStat(a, row.Fmt), fmtStat(b, row.Fmt)
		if from == to {
			continue
		}
		out = append(out, statChange{Label: row.Label, From: from, To: to, Up: b > a})
		if len(out) >= n {
			break
		}
	}
	return out
}

// forgeLen is when the current forge closes by itself.
func (fa *forgeAnim) end() float64 {
	n := max(1, len(fa.gains))
	return forgeGains + float64(n)*forgeStep + forgeHold
}

// updateForge runs the forge; it returns true while the show holds the
// game (nothing else updates). It waits for a card pick to finish first.
func (g *Game) updateForge(dt float64) bool {
	if len(g.forges) == 0 {
		return false
	}
	if g.pick.on || !g.cinematic() {
		if !g.cinematic() {
			g.forges = nil
		}
		return false
	}
	fa := g.forges[0]
	speed := 1.0
	if g.settings.FastAnim {
		speed = 1.8
	}
	if fa.t == 0 {
		g.snd.PlayKeyed("fuse")
		g.snd.Duck(0.75, 3.2)
	}
	press := g.ui.Click || g.keyHit(ebiten.KeySpace) || g.keyHit(ebiten.KeyEnter) || g.keyHit(ebiten.KeyEscape) || g.ui.nav.confirm
	switch {
	case fa.out > 0:
		fa.out += dt * speed
		if fa.out >= forgeOut {
			g.forges = g.forges[1:]
		}
	case press && fa.t > 0.5 && !fa.skip && fa.t < forgeGains+float64(len(fa.gains))*forgeStep:
		// first press: jump to the payoff, all lines shown
		fa.skip = true
		fa.t = forgeGains + float64(len(fa.gains))*forgeStep
		fa.ticked = len(fa.gains)
		fa.boomed = true
	case press && fa.t > 0.5:
		fa.out = 1e-6
	default:
		fa.t += dt * speed
		if fa.t >= fa.end() {
			fa.out = 1e-6
		}
	}
	if !fa.boomed && fa.t >= forgeCollide {
		fa.boomed = true
		g.react(reactFusion)
	}
	for fa.ticked < len(fa.gains) && fa.t >= forgeGains+float64(fa.ticked)*forgeStep {
		g.snd.Play(fmt.Sprintf("statup.%d.%d", fa.ticked, g.snd.Key))
		fa.ticked++
	}
	g.consumeInput()
	return true
}

// drawForge draws the current forge over everything.
func (g *Game) drawForge(s *ebiten.Image) {
	if len(g.forges) == 0 || g.pick.on {
		return
	}
	fa := g.forges[0]
	f := fa.f
	u := g.ui
	S := u.S
	t := fa.t
	fade := 1.0
	if fa.out > 0 {
		fade = 1 - smootherstep(fa.out/forgeOut)
	}
	col := f.Col
	A, B := modByID[f.A], modByID[f.B]
	cx, cy := 960.0, 400.0
	// the stage: a dark veil and a slow sigil
	veil := smootherstep(t/forgeIn) * fade
	fillRectA(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), rgba(4, 3, 10, 255), 0.78*veil)
	g.drawSigil(s, cx, cy, 250, t*0.18, col, 0.35*veil)
	drawSprite(s, sprSoft, u.X(cx-520), u.Y(cy-380), 1040*S, 760*S, ca(col, 0.14*veil), true)
	u.text(s, "FUSION", fBold, 22, 960, 120, ca(lighten(col, 0.3), veil*smootherstep((t-0.1)/0.3)), alignCenter)

	// 1. the two halves spiral in and meet
	if t < forgeCollide+0.05 {
		// pos is where a half is at progress e: both orbit the centre on
		// opposite sides, a quarter turn and more, closing in as they go
		pos := func(side int, e float64) (float64, float64) {
			r := 520 * (1 - e)
			a := math.Pi*float64(1-side) + e*math.Pi*1.25
			return cx + math.Cos(a)*r, cy + math.Sin(a)*r*0.35
		}
		prog := func(t float64) float64 { return smootherstep((t - 0.15) / (forgeCollide - 0.15)) }
		for side, m := range []*Mod{A, B} {
			if m == nil {
				continue
			}
			mc := modColor(m)
			// a comet tail behind each emblem
			for k := 6; k >= 1; k-- {
				xk, yk := pos(side, prog(t-float64(k)*0.03))
				sz := (40 - float64(k)*5) * S
				drawSprite(s, sprSoft, u.X(xk)-sz, u.Y(yk)-sz, 2*sz, 2*sz, ca(mc, 0.35*veil*(1-float64(k)/7)), true)
			}
			x, y := pos(side, prog(t))
			in := smootherstep(t / 0.25)
			drawSprite(s, sprSoft, u.X(x-80), u.Y(y-80), 160*S, 160*S, ca(mc, 0.5*in), true)
			u.drawEmblem(s, modEmblem(m), x, y, 110*(0.8+0.2*in), in*fade)
			if t < 0.9 {
				u.text(s, m.Name, fBold, 20, x, y+66, ca(cWhite, in*(1-smootherstep((t-0.6)/0.3))), alignCenter)
			}
		}
		// the gap crackles as they close in
		if t > 0.7 {
			k := (t - 0.7) / (forgeCollide - 0.7)
			drawSprite(s, sprSoft, u.X(cx-60-120*k), u.Y(cy-60-120*k), (120+240*k)*S, (120+240*k)*S, ca(lighten(col, 0.5), 0.5*k), true)
		}
	}

	// 2. the collision: a white flash, rings and sparks
	if d := t - forgeCollide; d >= 0 {
		if d < 0.35 {
			fl := 1 - d/0.35
			fillRectA(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), lighten(col, 0.6), 0.55*fl*fl*fade)
		}
		for j, dd := range []float64{0, 0.1, 0.22} {
			e := (d - dd) / 0.8
			if e > 0 && e < 1 {
				rr := 60 + 700*ease(e)
				drawSprite(s, sprRing, u.X(cx-rr), u.Y(cy-rr), 2*rr*S, 2*rr*S, ca(lighten(col, 0.25*float64(j)), 0.9*(1-e)*fade), true)
			}
		}
		if d < 1.2 {
			for i := 0; i < 64; i++ {
				fi := float64(i)
				an := fi/64*2*math.Pi + math.Sin(fi*7.3)*0.2
				sp := 300 + 700*math.Mod(fi*0.618, 1)
				x := cx + math.Cos(an)*sp*d
				y := cy + math.Sin(an)*sp*d*0.75 + 380*d*d
				al := math.Max(0, 1-d/1.1) * fade
				sz := (3 + 7*math.Mod(fi*0.37, 1)) * S * (1 - d*0.5)
				if al > 0 && sz > 0 {
					drawSprite(s, sprSoft, u.X(x)-sz*2, u.Y(y)-sz*2, sz*4, sz*4, ca(lighten(col, 0.2+0.5*math.Mod(fi*0.21, 1)), al), true)
				}
			}
		}
		// god rays turning behind the new emblem
		ra := smootherstep(d/0.4) * fade
		for i := 0; i < 16; i++ {
			a := t*0.35 + float64(i)/16*2*math.Pi
			r0, r1 := 90.0, 300+40*math.Sin(t*2+float64(i))
			strokeLine(s, u.X(cx+math.Cos(a)*r0), u.Y(cy+math.Sin(a)*r0), u.X(cx+math.Cos(a)*r1), u.Y(cy+math.Sin(a)*r1), 10*S, prem(col, 0.10*ra))
		}
		// 3. the fusion's emblem rises out of the light, overshooting a little
		e := clamp01(d / 0.5)
		pop := 1 + 0.25*math.Sin(e*math.Pi)*(1-e)
		size := 190 * ease(e) * pop
		bob := math.Sin(t*2.2) * 5
		drawSprite(s, sprSoft, u.X(cx-170), u.Y(cy-170+bob), 340*S, 340*S, ca(col, 0.55*fade), true)
		u.drawEmblem(s, fusionEmblem(f), cx, cy+bob, size, fade)
		// a sheen across it
		if d < 0.9 {
			sw := d / 0.9
			x := cx - 110 + 220*sw
			drawSprite(s, sprSoft, u.X(x-30), u.Y(cy-100+bob), 60*S, 200*S, ca(cWhite, 0.5*(1-sw)*fade), true)
		}
	}

	// 4. the name and what it does
	y := 730.0
	if d := t - forgeName; d >= 0 {
		k := ease(d / 0.3)
		size := 54 * (1.25 - 0.25*k)
		u.textOutlined(s, strings.ToUpper(tr(f.Name)), fRune, size, 960, 545+(1-k)*12, ca(lighten(col, 0.35), k*fade), alignCenter)
		by := u.paragraph(s, f.Desc, fReg, 22*g.textScale(), 960, 615, 900, ca(cWhite, k*fade), alignCenter)
		if A != nil && B != nil {
			u.text(s, trJoin(A.Name, "  +  ", B.Name), fReg, 18, 960, by+8, ca(cDim, k*fade), alignCenter)
			by += 36
		}
		y = math.Max(y, by+24)
	}

	// 5. the payoff: each number it adds, counting up, or the new passive
	if len(fa.gains) == 0 {
		if d := t - forgeGains; d >= 0 {
			k := ease(d / 0.3)
			w := u.measure("NEW PASSIVE", fBold, 22) + 60
			fillPath(s, boxPath(u.X(960-w/2), u.Y(y), w*S, 44*S, 22*S, false), ca(col, 0.35*k*fade), ca(darken(col, 0.4), 0.35*k*fade), vgrad(u.Y(y), u.Y(y+44)), false)
			strokePath(s, boxPath(u.X(960-w/2), u.Y(y), w*S, 44*S, 22*S, false), 2*S, ca(lighten(col, 0.3), k*fade), false)
			u.text(s, "NEW PASSIVE", fBold, 22, 960, y+9, ca(cWhite, k*fade), alignCenter)
		}
	}
	for i, c := range fa.gains {
		d := t - forgeGains - float64(i)*forgeStep
		if d < 0 {
			break
		}
		k := ease(d / 0.25)
		ly := y + float64(i)*50 + (1-k)*18
		// a plate per line
		w := 560.0
		fillPath(s, boxPath(u.X(960-w/2), u.Y(ly), w*S, 42*S, 12*S, false), ca(rgba(20, 30, 20, 255), 0.6*k*fade), ca(rgba(10, 16, 10, 255), 0.6*k*fade), vgrad(u.Y(ly), u.Y(ly+42)), false)
		strokePath(s, boxPath(u.X(960-w/2), u.Y(ly), w*S, 42*S, 12*S, false), 1.5*S, ca(rgba(140, 240, 150, 255), 0.6*k*fade), false)
		u.text(s, c.Label, fReg, 21, 960-w/2+20, ly+9, ca(cDim, k*fade), alignLeft)
		to := countUp(c.From, c.To, clamp01(d/0.45))
		vc := rgba(140, 240, 150, 255)
		if !c.Up {
			vc = rgba(255, 130, 120, 255)
		}
		// the new value lands with a little pop
		ps := 21 * (1 + 0.3*math.Max(0, 1-d/0.2))
		u.text(s, c.From+"  →", fReg, 21, 960+w/2-120, ly+9, ca(cDim, 0.7*k*fade), alignRight)
		u.text(s, to, fBold, ps, 960+w/2-20, ly+9-(ps-21)*0.5, ca(vc, k*fade), alignRight)
	}
	// how to move on
	if t > 0.8 && fa.out == 0 {
		hint := "Click to continue"
		if t < forgeGains+float64(len(fa.gains))*forgeStep {
			hint = "Click to skip"
		}
		a := 0.45 + 0.2*math.Sin(t*3)
		u.text(s, hint, fReg, 17, 960, 1010, ca(cDim, a*fade), alignCenter)
	}
}

// countUp moves a stat label from one value to another as k runs 0..1,
// keeping its format ("+30%", "×2.5", "110%"). Values it can't read jump.
func countUp(from, to string, k float64) string {
	if k >= 1 {
		return to
	}
	pre, a, suf, ok1 := splitNum(from)
	pre2, b, suf2, ok2 := splitNum(to)
	if !ok1 || !ok2 || pre != pre2 || suf != suf2 {
		if k < 0.5 {
			return from
		}
		return to
	}
	if k <= 0 {
		return from
	}
	v := a + (b-a)*smootherstep(k)
	s := strconv.FormatFloat(v, 'f', max(decimals(from, suf), decimals(to, suf)), 64)
	return pre + s + suf
}

// decimals counts the digits after the point in a stat label.
func decimals(s, suf string) int {
	s = strings.TrimSuffix(s, suf)
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		return len(s) - i - 1
	}
	return 0
}

// splitNum cuts "×2.5" into "×", 2.5, "" and "+30%" into "+", 30, "%".
func splitNum(s string) (pre string, v float64, suf string, ok bool) {
	i := strings.IndexAny(s, "0123456789")
	if i < 0 {
		return "", 0, "", false
	}
	j := i
	for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
		j++
	}
	if _, err := fmt.Sscanf(s[i:j], "%g", &v); err != nil {
		return "", 0, "", false
	}
	pre, suf = s[:i], s[j:]
	if strings.HasSuffix(pre, "-") {
		pre, v = strings.TrimSuffix(pre, "-"), -v
	}
	return pre, v, suf, true
}
