package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The title screen: a synthwave horizon with an endless scrolling grid,
// a hovering chrome orb and slowly tumbling brick stacks.

type menuScene struct {
	t     float64
	biome *Biome
	sky   *ebiten.Image
	cubes []menuCube
	seed  int64
}

type menuCube struct {
	x, y, z, s, spin, bob float64
	col                   color.RGBA
	stack                 int
}

func newMenuScene(seed int64) *menuScene {
	r := rand.New(rand.NewSource(seed))
	b, _ := levelLook(&worlds[WNeon], seed)
	m := &menuScene{biome: b, seed: seed}
	if img := backdrop(b.SkyKind, int(uint64(seed)%2)); img != nil {
		m.sky = img
	} else {
		m.sky = makeSky(b, seed)
	}
	for i := 0; i < 12; i++ {
		side := float64(r.Intn(2)*2 - 1)
		m.cubes = append(m.cubes, menuCube{
			x: side * (9 + r.Float64()*18), y: 1 + r.Float64()*5, z: 12 + r.Float64()*45,
			s: 1.2 + r.Float64()*1.6, spin: (r.Float64() - 0.5) * 0.8, bob: r.Float64() * 6,
			col: b.Hues[r.Intn(6)], stack: 1 + r.Intn(3),
		})
	}
	return m
}

func (g *Game) drawMenuScene(screen *ebiten.Image, dt float64) {
	if g.menu == nil {
		g.menu = newMenuScene(g.rng.Int63())
	}
	m := g.menu
	m.t += dt
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	b := m.biome
	cam := &g.cam
	cam.Roll = 0
	cam.Eye = V3{math.Sin(m.t*0.15) * 1.5, 4.2, -8}
	cam.Target = V3{0, 3.4, 30}
	cam.FOV = 62 * math.Pi / 180
	cam.Setup(sw, sh)
	_, hy, _ := cam.Project(V3{cam.Eye.X, 0, cam.Eye.Z + 4000})
	fillRect(screen, 0, 0, sw, sh, prem(b.SkyTop, 1))
	bw, bh := float64(m.sky.Bounds().Dx()), float64(m.sky.Bounds().Dy())
	sc := sw / bw
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(0, hy-bh*skyHorizon*sc)
	screen.DrawImage(m.sky, op)

	B := g.batch
	B.Reset()
	B.fogColor = [3]float32{float32(b.Fog.R) / 255, float32(b.Fog.G) / 255, float32(b.Fog.B) / 255}
	B.fogNear, B.fogFar = 20, 150
	B.Quad([4]V3{{-400, 0, 600}, {400, 0, 600}, {400, 0, -20}, {-400, 0, -20}}, sprWhite, b.Outer, [4]float32{1, 1, 1, 1}, false)
	B.quads[len(B.quads)-1].depth = 1e9
	scroll := math.Mod(m.t*6, 5)
	for x := -150.0; x <= 150; x += 5 {
		B.Quad([4]V3{{x - 0.07, 0.01, 600}, {x + 0.07, 0.01, 600}, {x + 0.07, 0.01, -20}, {x - 0.07, 0.01, -20}}, sprWhite, alpha(b.Floor, 0.8), [4]float32{1, 1, 1, 1}, true)
	}
	for z := -15.0 - scroll; z < 400; z += 5 {
		B.Quad([4]V3{{-400, 0.01, z + 0.09}, {400, 0.01, z + 0.09}, {400, 0.01, z - 0.09}, {-400, 0.01, z - 0.09}}, sprWhite, alpha(b.FloorAlt, 0.7), [4]float32{1, 1, 1, 1}, true)
	}
	// floating brick stacks
	for _, c := range m.cubes {
		y := c.y + math.Sin(m.t*0.9+c.bob)*0.6
		for k := 0; k < c.stack; k++ {
			col := hueShift(c.col, float64(k)*25)
			fp := rotQuad(c.x, c.z, c.s*1.6, c.s, m.t*c.spin+c.bob)
			y0 := y + float64(k)*c.s*0.62
			B.Prism(fp, y0, y0+c.s*0.56, 1, col, 0.15)
		}
		B.DecalBelow(c.x, c.z, 0.02, c.s*3, c.s*2, sprSoft, alpha(c.col, 0.25))
	}
	// sparkles
	pr := rand.New(rand.NewSource(m.seed))
	for i := 0; i < 70; i++ {
		x := (pr.Float64() - 0.5) * 60
		z := 5 + pr.Float64()*60
		sp := 0.4 + pr.Float64()*0.8
		y := math.Mod(pr.Float64()*12+m.t*sp, 12)
		B.Billboard(V3{x, y, z}, 0.3, sprSoft, alpha(b.Hues[i%6], 0.8*math.Sin(y/12*math.Pi)), true)
	}
	B.Flush(screen)
	// scanlines + vignette for a retro CRT touch
	for y := 0.0; y < sh; y += 4 * g.ui.S {
		fillRect(screen, 0, y, sw, math.Max(1, g.ui.S), color.RGBA{0, 0, 0, 22})
	}
}

// chromeTitle draws text with a retro chrome gradient and neon glow.
func (g *Game) chromeTitle(dst *ebiten.Image, s string, size, x, y float64, glow color.RGBA) {
	u := g.ui
	face := u.face(fTitle, size)
	w, h := text.Measure(s, face, 0)
	key := s + itoa(int(size*u.S))
	img, ok := g.titleCache[key]
	if !ok {
		img = ebiten.NewImage(int(w)+8, int(h)+8)
		op := &text.DrawOptions{}
		op.GeoM.Translate(4, 4)
		text.Draw(img, s, face, op)
		g.titleCache[key] = img
	}
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	sx := u.X(x) - float64(iw)/2
	sy := u.Y(y)
	// glow
	for i := 0; i < 12; i++ {
		a := float64(i) / 12 * 2 * math.Pi
		r := 7 * u.S
		op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
		op.GeoM.Translate(sx+math.Cos(a)*r, sy+math.Sin(a)*r)
		op.ColorScale.ScaleWithColor(prem(glow, 0.16))
		dst.DrawImage(img, op)
	}
	// dark outline
	for _, d := range [][2]float64{{-3, 0}, {3, 0}, {0, -3}, {0, 3}, {-2, -2}, {2, 2}, {-2, 2}, {2, -2}} {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx+d[0]*u.S, sy+d[1]*u.S+3*u.S)
		op.ColorScale.ScaleWithColor(color.RGBA{14, 6, 30, 255})
		dst.DrawImage(img, op)
	}
	// chrome gradient in horizontal strips
	stops := []struct {
		t float64
		c color.RGBA
	}{
		{0, color.RGBA{180, 240, 255, 255}}, {0.42, color.RGBA{255, 255, 255, 255}}, {0.5, color.RGBA{70, 40, 150, 255}},
		{0.56, color.RGBA{255, 80, 200, 255}}, {0.8, color.RGBA{255, 170, 90, 255}}, {1, color.RGBA{255, 240, 190, 255}},
	}
	grad := func(t float64) color.RGBA {
		for i := 1; i < len(stops); i++ {
			if t <= stops[i].t {
				f := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
				return mixc(stops[i-1].c, stops[i].c, f)
			}
		}
		return stops[len(stops)-1].c
	}
	// the glyph body sits roughly between 18% and 92% of the image height
	top, bot := float64(ih)*0.18, float64(ih)*0.92
	n := 40
	for i := 0; i < n; i++ {
		y0 := int(float64(ih) * float64(i) / float64(n))
		y1 := int(float64(ih) * float64(i+1) / float64(n))
		if y1 <= y0 {
			continue
		}
		t := math.Max(0, math.Min(1, (float64(y0+y1)/2-top)/(bot-top)))
		sub := img.SubImage(image.Rect(0, y0, iw, y1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx, sy+float64(y0))
		op.ColorScale.ScaleWithColor(grad(t))
		dst.DrawImage(sub, op)
	}
	// horizon highlight line
	ly := sy + top + (bot-top)*0.5
	fillRect(dst, sx+8*u.S, ly, float64(iw)-16*u.S, math.Max(1, 1.5*u.S), color.RGBA{255, 255, 255, 140})
}

// neonItem is a text-only menu entry with a glowing hover state.
func (g *Game) neonItem(dst *ebiten.Image, id, label string, cx, y float64, accent color.RGBA, enabled bool) bool {
	u := g.ui
	w := u.measure(label, fBold, 34) + 80
	x := cx - w/2
	hv := enabled && u.hover(x, y, w, 52)
	if hv {
		u.hoverID = id
		if u.lastHover != id {
			g.snd.Play("hover")
		}
		pulse := 0.6 + 0.4*math.Sin(g.stateT*8)
		u.rrect(dst, x, y+4, w, 46, 23, alpha(accent, 0.22*pulse+0.1))
		u.rstroke(dst, x, y+4, w, 46, 23, 2, alpha(lighten(accent, 0.3), 0.9))
		u.text(dst, "›", fBold, 34, x+22, y+2, lighten(accent, 0.4), alignCenter)
		u.text(dst, "‹", fBold, 34, x+w-22, y+2, lighten(accent, 0.4), alignCenter)
	}
	c := color.RGBA{225, 220, 255, 255}
	if hv {
		c = color.RGBA{255, 255, 255, 255}
	}
	if !enabled {
		c = color.RGBA{120, 115, 140, 255}
	}
	u.textOutlined(dst, label, fBold, 34, cx, y+5, c, alignCenter)
	if hv && u.Click {
		g.snd.Play("click")
		return true
	}
	return false
}
