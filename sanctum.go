package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The front-end backdrop: a dark arcane sanctum drawn by one full-screen
// shader. Moonlit gothic windows and pillars on the back wall, light shafts,
// rolling violet mist, a stone floor that mirrors the orb, and a slowly
// turning magic circle beneath it. Floating runes are drawn on top in Go.

const sanctumSrc = `//kage:unit pixels
package main

var Time float
var Origin vec2
var Scale float
var Orb vec2   // orb centre, virtual px
var Circle float // 1 = draw the magic circle

func hash2(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func hash(p vec3) float {
	p = fract(p*0.3183099 + vec3(0.1, 0.2, 0.3))
	p *= 17.0
	return fract(p.x * p.y * p.z * (p.x + p.y + p.z))
}

func noise(x vec3) float {
	i := floor(x)
	f := fract(x)
	f = f * f * (3.0 - 2.0*f)
	a := mix(hash(i), hash(i+vec3(1, 0, 0)), f.x)
	b := mix(hash(i+vec3(0, 1, 0)), hash(i+vec3(1, 1, 0)), f.x)
	c := mix(hash(i+vec3(0, 0, 1)), hash(i+vec3(1, 0, 1)), f.x)
	d := mix(hash(i+vec3(0, 1, 1)), hash(i+vec3(1, 1, 1)), f.x)
	return mix(mix(a, b, f.y), mix(c, d, f.y), f.z)
}

func fbm(p vec3) float {
	v := 0.0
	a := 0.5
	for i := 0; i < 4; i++ {
		v += a * noise(p)
		p = p*2.03 + vec3(1.7, 9.2, 3.1)
		a *= 0.5
	}
	return v
}

// pointed gothic window: 1 inside, 0 outside (soft)
func window(p vec2, w float, top float, bot float) float {
	if p.y > bot || abs(p.x) > w {
		return 0.0
	}
	// arch: intersection of two circles centred on the opposite jambs
	ay := top + w*1.6
	if p.y < ay {
		r := w * 2.0
		d1 := length(p - vec2(-w, ay))
		d2 := length(p - vec2(w, ay))
		return smoothstep(r+1.5, r-1.5, max(d1, d2))
	}
	return smoothstep(w+1.5, w-1.5, abs(p.x)) * smoothstep(bot+1.0, bot-2.0, p.y)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	v := (dstPos.xy - Origin) / Scale
	t := Time
	hz := 700.0
	col := vec3(0)

	if v.y < hz {
		// ---- back wall
		wall := mix(vec3(0.010, 0.006, 0.022), vec3(0.035, 0.018, 0.065), smoothstep(0.0, hz, v.y))
		stone := fbm(vec3(v.x*0.02, v.y*0.035, 1.0))
		wall *= 0.7 + 0.6*stone
		col = wall
		// gothic windows with moonlit sky behind
		period := 330.0
		cx := floor((v.x+period*0.5)/period)*period
		lp := v - vec2(cx, 0)
		win := window(lp, 62.0, 150.0, 600.0)
		if win > 0.0 {
			sky := mix(vec3(0.15, 0.10, 0.32), vec3(0.04, 0.03, 0.11), smoothstep(150.0, 600.0, v.y))
			// clouds drifting past the glass
			cl := fbm(vec3(v.x*0.01+t*0.02, v.y*0.02, cx*0.01))
			sky += vec3(0.06, 0.04, 0.12) * smoothstep(0.5, 0.8, cl)
			// mullions and tracery
			mul := 1.0 - smoothstep(2.5, 1.0, abs(lp.x))
			mul *= 1.0 - smoothstep(2.5, 1.0, abs(lp.y-380.0))
			ring := abs(length(lp-vec2(0, 245.0)) - 34.0)
			mul *= smoothstep(1.0, 3.0, ring)
			col = mix(col, sky*mul, win)
		}
		// pillars between the windows
		px := abs(v.x - (floor(v.x/period)*period + period*0.0) )
		pd := min(px, period-px)
		pil := smoothstep(40.0, 36.0, pd)
		pcol := vec3(0.012, 0.008, 0.025) + vec3(0.07, 0.05, 0.12)*smoothstep(36.0, 30.0, pd)*smoothstep(26.0, 34.0, pd)
		col = mix(col, pcol, pil)
	} else {
		// ---- stone floor in perspective
		dy := (v.y - hz) / 380.0
		z := 1.0 / max(dy, 0.002)
		fx := (v.x - 960.0) * z / 420.0
		tile := vec2(fx, z*1.4)
		g := abs(fract(tile) - 0.5)
		line := smoothstep(0.47, 0.5, max(g.x, g.y)) * smoothstep(8.0, 1.0, z)
		st := fbm(vec3(tile*2.0, 2.0))
		col = vec3(0.016, 0.011, 0.028) * (0.6 + 0.8*st)
		col *= 1.0 - 0.6*line
		col *= smoothstep(0.0, 0.25, dy) * 0.7 + 0.3
		// mirror of the orb on the polished floor
		rx := (v.x - Orb.x) / 180.0
		ry := (v.y - (hz + (hz-Orb.y)*0.45)) / 260.0
		col += vec3(0.32, 0.14, 0.62) * exp(-(rx*rx*1.5 + ry*ry*4.0)) * 0.5 * (0.85 + 0.15*sin(t*2.3))
	}

	// ---- light shaft falling on the orb
	sx := (v.x - Orb.x + (v.y-Orb.y)*0.18) / (110.0 + v.y*0.18)
	shaft := exp(-sx*sx*2.0) * smoothstep(-80.0, 500.0, v.y) * smoothstep(1000.0, 650.0, v.y)
	shaft *= 0.7 + 0.3*fbm(vec3(sx*2.0, v.y*0.004-t*0.05, t*0.05))
	col += vec3(0.16, 0.10, 0.30) * shaft * 0.6
	// window shafts slanting into the hall
	period := 330.0
	wx := v.x - (v.y-150.0)*0.35
	wc := floor((wx+period*0.5)/period)*period
	wd := abs(wx-wc) / 60.0
	ws := exp(-wd*wd*1.6) * smoothstep(150.0, 400.0, v.y) * smoothstep(1050.0, 600.0, v.y)
	col += vec3(0.06, 0.05, 0.13) * ws * (0.6 + 0.4*fbm(vec3(wx*0.01, v.y*0.006-t*0.04, 3.0)))

	// ---- magic circle on the floor under the orb
	if Circle > 0.5 {
		cc := vec2(Orb.x, 880.0)
		q := (v - cc) / vec2(350.0, 86.0)
		r := length(q)
		if r < 1.25 {
			a := atan2(q.y, q.x)
			rot := t * 0.12
			w := 0.010
			ln := 0.0
			ln += exp(-abs(r-1.0)/w) + exp(-abs(r-0.9)/w)*0.8 + exp(-abs(r-0.62)/w)*0.7 + exp(-abs(r-0.3)/w)*0.6
			// rune band between the outer rings
			band := step(0.905, r) * step(r, 0.995)
			seg := fract((a+rot)*36.0/6.2831853)
			rn := hash2(vec2(floor((a+rot)*36.0/6.2831853), 3.0))
			glyph := step(0.25, seg) * step(seg, 0.75) * step(0.3, rn)
			gy := (r - 0.905) / 0.09
			glyph *= step(0.15, gy) * step(gy, 0.85) * (step(0.5, fract(gy*2.0+rn)) + step(0.42, abs(seg-0.5)))
			ln += glyph * band * 0.9
			// hexagram counter-rotating inside
			b := a - rot*1.6
			n3 := 2.0943951
			t1 := r*cos(mod(b, n3)-n3*0.5) - 0.31
			t2 := r*cos(mod(b+1.0471976, n3)-n3*0.5) - 0.31
			ln += (exp(-abs(t1)/w) + exp(-abs(t2)/w)) * step(r, 0.62) * 0.8
			// small orbiting nodes
			for k := 0; k < 6; k++ {
				ka := float(k)*1.0471976 + rot*2.0
				np := vec2(cos(ka), sin(ka)) * 0.76
				ln += exp(-length(q-np)/0.025) * 0.9
			}
			pulse := 0.75 + 0.25*sin(t*1.7)
			cc2 := mix(vec3(0.55, 0.3, 1.0), vec3(1.0, 0.6, 0.95), 0.5+0.5*sin(a*2.0+t))
			col += cc2 * ln * 0.55 * pulse
			col += vec3(0.25, 0.1, 0.5) * exp(-r*r*2.5) * 0.25 * pulse
		}
	}

	// ---- rolling mist, thickest along the floor
	m1 := fbm(vec3(v.x*0.0025-t*0.025, v.y*0.006, t*0.03))
	m2 := fbm(vec3(v.x*0.006+t*0.04, v.y*0.012, 7.0+t*0.05))
	mist := smoothstep(0.4, 0.85, m1*0.7+m2*0.5)
	mw := smoothstep(450.0, 760.0, v.y) * smoothstep(1100.0, 820.0, v.y)
	orbTint := exp(-pow((v.x-Orb.x)/420.0, 2.0))
	col += mix(vec3(0.05, 0.04, 0.09), vec3(0.16, 0.08, 0.3), orbTint) * mist * (0.25 + 0.75*mw)

	// ---- dust in the air
	cell := floor(v / 7.0)
	h := hash2(cell)
	if h > 0.9975 {
		tw := 0.5 + 0.5*sin(t*(1.0+h*3.0)+h*40.0)
		col += vec3(0.5, 0.42, 0.8) * tw * (0.3 + 0.7*shaft)
	}

	// vignette
	nv := (v - vec2(960.0, 560.0)) / vec2(1150.0, 760.0)
	col *= clamp(1.0-dot(nv, nv)*0.9, 0.0, 1.0)
	return vec4(col, 1.0)
}
`

var sanctumShader *ebiten.Shader

// drawHall fills the screen with the arcane hall.
func (g *Game) drawHall(dst *ebiten.Image, orbX, orbY float64, circle bool) {
	c := 0.0
	if circle {
		c = 1
	}
	g.drawHallK(dst, orbX, orbY, c)
}

// drawHallK is drawHall with the magic circle at strength c (0..1).
func (g *Game) drawHallK(dst *ebiten.Image, orbX, orbY, c float64) {
	if sanctumShader == nil {
		sh, err := ebiten.NewShader([]byte(sanctumSrc))
		if err != nil {
			panic(err)
		}
		sanctumShader = sh
	}
	u := g.ui
	if sc := g.hallScale(); sc < 1 {
		// render the hall smaller and let linear filtering enlarge it; it
		// is all soft light and mist, so the saving costs nothing visible
		w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
		rw, rh := max(16, int(float64(w)*sc)), max(16, int(float64(h)*sc))
		if hallSmall == nil || hallSmall.Bounds().Dx() != rw || hallSmall.Bounds().Dy() != rh {
			if hallSmall != nil {
				hallSmall.Deallocate()
			}
			hallSmall = ebiten.NewImage(rw, rh)
		}
		op := &ebiten.DrawRectShaderOptions{}
		op.Uniforms = map[string]any{
			"Time":   float32(math.Mod(g.frontT, 2000)),
			"Origin": []float32{float32(u.X(0) * sc), float32(u.Y(0) * sc)},
			"Scale":  float32(u.S * sc),
			"Orb":    []float32{float32(orbX), float32(orbY)},
			"Circle": float32(c),
		}
		hallSmall.DrawRectShader(rw, rh, sanctumShader, op)
		dop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		dop.GeoM.Scale(float64(w)/float64(rw), float64(h)/float64(rh))
		dst.DrawImage(hallSmall, dop)
		return
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":   float32(math.Mod(g.frontT, 2000)),
		"Origin": []float32{float32(u.X(0)), float32(u.Y(0))},
		"Scale":  float32(u.S),
		"Orb":    []float32{float32(orbX), float32(orbY)},
		"Circle": float32(c),
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	dst.DrawRectShader(w, h, sanctumShader, op)
}

// ---------------------------------------------------------------- runes

type runeShape [][4]float64 // strokes in a 1 x 1.4 box

var runeShapes []runeShape

func makeRunes() {
	if runeShapes != nil {
		return
	}
	r := rand.New(rand.NewSource(31))
	for i := 0; i < 20; i++ {
		var sh runeShape
		// a stave plus two to three branches, in the spirit of old futhark
		x := 0.3 + 0.4*r.Float64()
		sh = append(sh, [4]float64{x, 0, x, 1.4})
		n := 2 + r.Intn(2)
		for k := 0; k < n; k++ {
			y0 := r.Float64() * 1.1
			x1 := float64(r.Intn(2))
			y1 := y0 + (r.Float64()-0.3)*0.6
			sh = append(sh, [4]float64{x, y0, x1, y1})
		}
		runeShapes = append(runeShapes, sh)
	}
}

// drawRunes floats faint glowing runes upward through the hall.
func (g *Game) drawRunes(dst *ebiten.Image, count int, cx, spread float64) {
	makeRunes()
	u := g.ui
	t := g.frontT
	for i := 0; i < count; i++ {
		f := float64(i)
		sp := 14 + 22*math.Mod(f*0.618, 1)
		life := math.Mod(t*sp/700+f*0.137, 1)
		y := 980 - life*760
		x := cx + (math.Mod(f*0.377, 1)-0.5)*spread + math.Sin(t*0.4+f)*20
		a := math.Sin(life*math.Pi) * (0.25 + 0.5*math.Mod(f*0.73, 1))
		size := 16 + 18*math.Mod(f*0.53, 1)
		rot := math.Sin(t*0.3+f) * 0.3
		sh := runeShapes[i%len(runeShapes)]
		col := mixc(rgba(170, 120, 255, 255), rgba(255, 190, 240, 255), math.Mod(f*0.29, 1))
		cs, sn := math.Cos(rot), math.Sin(rot)
		for _, st := range sh {
			p := func(px, py float64) (float64, float64) {
				lx, ly := (px-0.5)*size, (py-0.7)*size
				return u.X(x + lx*cs - ly*sn), u.Y(y + lx*sn + ly*cs)
			}
			x0, y0 := p(st[0], st[1])
			x1, y1 := p(st[2], st[3])
			ln := linePath(x0, y0, x1, y1)
			strokePath(dst, ln, 7*u.S, ca(col, 0.12*a), true)
			strokePath(dst, ln, 1.8*u.S, ca(lighten(col, 0.4), 0.8*a), true)
		}
	}
}

// ---------------------------------------------------------------- title

// arcaneTitle draws engraved Cinzel lettering with a violet aura.
func (g *Game) arcaneTitle(dst *ebiten.Image, s string, size, x, y float64) {
	u := g.ui
	s = tr(s)
	face := u.face(fRune, size)
	w, h := text.Measure(s, face, 0)
	key := "rune" + s + itoa(int(size*u.S))
	img, ok := g.titleCache[key]
	if !ok {
		img = ebiten.NewImage(int(w)+8, int(h)+8)
		op := &text.DrawOptions{}
		op.GeoM.Translate(4, 4)
		text.Draw(img, s, face, op)
		g.titleCache[key] = img
	}
	iw := float64(img.Bounds().Dx())
	sx, sy := u.X(x)-iw/2, u.Y(y)
	t := g.frontT
	draw := func(dx, dy float64, c color.RGBA, a float64, add bool) {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		if add {
			op.Blend = ebiten.BlendLighter
		}
		op.GeoM.Translate(sx+dx, sy+dy)
		op.ColorScale.ScaleWithColor(c)
		op.ColorScale.ScaleAlpha(float32(a))
		dst.DrawImage(img, op)
	}
	// aura
	pulse := 0.8 + 0.2*math.Sin(t*1.3)
	for i := 0; i < 16; i++ {
		a := float64(i) / 16 * 2 * math.Pi
		r := 9 * u.S
		draw(math.Cos(a)*r, math.Sin(a)*r, rgba(130, 70, 255, 255), 0.07*pulse, true)
	}
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		r := 3.5 * u.S
		draw(math.Cos(a)*r, math.Sin(a)*r, rgba(200, 150, 255, 255), 0.12*pulse, true)
	}
	// carved shadow then the face of the letters
	draw(0, 4*u.S, rgba(0, 0, 0, 255), 0.8, false)
	draw(0, 0, rgba(232, 222, 250, 255), 1, false)
	// slow moonlight sheen sweeping across
	k := math.Mod(t*0.12, 1.4) - 0.2
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter, Filter: ebiten.FilterLinear}
	op.GeoM.Translate(sx, sy)
	op.ColorScale.ScaleWithColor(rgba(255, 230, 200, 255))
	op.ColorScale.ScaleAlpha(float32(0.35 * math.Max(0, math.Sin(math.Min(1, math.Max(0, k))*math.Pi))))
	dst.DrawImage(img, op)
}

// arcaneItem is a front-end menu entry: Cinzel lettering that lights up
// with a soft violet band and a small diamond when hovered.
func (g *Game) arcaneItem(dst *ebiten.Image, id, label string, x, y float64, enabled bool) bool {
	u := g.ui
	size := 31.0
	w := u.measure(label, fSerif, size) + 60
	var hv, clk bool
	if enabled {
		hv, clk = u.hit(id, x-30, y-4, w+40, 52)
	}
	k := u.anim("ai"+id, hv, 12)
	if hv && !u.nav.mode {
		u.hoverID = id
		if u.lastHover != id {
			g.snd.Play("hover")
		}
	}
	if k > 0.01 {
		S := u.S
		fillPath(dst, boxPath(u.X(x-34), u.Y(y), (w+120)*S, 46*S, 0, false), rgba(150, 100, 255, uint8(70*k)), rgba(150, 100, 255, 0), hgrad(u.X(x-34), u.X(x+w+86)), true)
		u.glowLine(dst, x-30, x-30+(w+60)*k, y+46, 1, rgba(200, 160, 255, 255), 0.8*k)
		dp := diamondPath(u.X(x-22+6*k), u.Y(y+23), 6*S, 9*S)
		fillPath(dst, dp, ca(rgba(230, 200, 255, 255), k), ca(rgba(150, 100, 255, 255), k), grad{}, false)
	}
	c := mixc(rgba(196, 188, 214, 255), rgba(255, 250, 255, 255), k)
	if !enabled {
		c = rgba(110, 104, 128, 255)
	}
	u.text(dst, label, fSerif, size, x+2, y+8+2, ca(cBlack, 0.7), alignLeft)
	u.text(dst, label, fSerif, size, x+12*k, y+8, c, alignLeft)
	if clk {
		g.snd.Play("click")
		return true
	}
	return false
}

// vignette darkens the screen edges with four soft gradients.
func vignette(dst *ebiten.Image, sw, sh float64, a uint8) {
	e := sw * 0.22
	clr := color.RGBA{0, 0, 0, 0}
	blk := color.RGBA{0, 0, 0, a}
	fillPath(dst, boxPath(0, 0, e, sh, 0, false), blk, clr, hgrad(0, e), false)
	fillPath(dst, boxPath(sw-e, 0, e, sh, 0, false), clr, blk, hgrad(sw-e, sw), false)
	ev := sh * 0.25
	fillPath(dst, boxPath(0, 0, sw, ev, 0, false), blk, clr, vgrad(0, ev), false)
	fillPath(dst, boxPath(0, sh-ev, sw, ev, 0, false), clr, blk, vgrad(sh-ev, sh), false)
}

var hallSmall *ebiten.Image

// hallScale is the resolution the title hall renders at.
func (g *Game) hallScale() float64 {
	switch g.settings.Quality {
	case 0:
		return 0.5
	case 1:
		return 0.75
	}
	return 1
}
