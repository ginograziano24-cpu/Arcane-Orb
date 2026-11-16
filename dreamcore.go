package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Dreamcore: an experimental art direction in the style of late-90s CGI
// fantasy as remixed by today's AI video trend. Simple primitives (spheres,
// cones, boxes) wrapped in tiled, embossed textures; galaxy skies; a sun
// with a face; and a sharpen-and-saturate pass over the 3D view.
//
// It is switched on with -dreamcore (or ARCANE_DREAMCORE=1) and leaves the
// normal look untouched otherwise. The first world built in this style is
// Sunking's Road, which replaces Sunset Meadow while the switch is on.
//
//go:embed assets/dream/*.png
var dreamFS embed.FS

var dreamcore bool

// playWorld is the -world=N launch flag (-1: off).
var playWorld = -1

// dreamSunAt is where drawDreamSky placed the sun this frame: screen x, y and radius.
var dreamSunAt [3]float64

// test switches for isolating the dreamcore passes
var dreamNoSun, dreamNoBloom = os.Getenv("ARCANE_DC_NOSUN") == "1", os.Getenv("ARCANE_DC_NOBLOOM") == "1"

func init() {
	if os.Getenv("ARCANE_DREAMCORE") == "1" {
		dreamcore = true
	}
	for _, a := range os.Args[1:] {
		if a == "-dreamcore" || a == "--dreamcore" {
			dreamcore = true
		}
		// -world=N plays every chamber in world N (playtesting one world):
		// 0 Glimmer Grove, 1 Sunset Meadow, 2 Caverns, 3 Ruins, 4 Clouds,
		// 5 Machine Works, 6 Sunking's Road
		if strings.HasPrefix(a, "-world=") {
			if n, err := strconv.Atoi(a[7:]); err == nil && n >= 0 && n < NumWorlds {
				playWorld = n
			}
		}
	}
	applyDreamTheme()
}

// dreamWorld reports whether a world is drawn in the dreamcore style.
func dreamWorld(id int) bool { return id == WDream }

// applyDreamTheme finishes Sunking's Road (WDream). The -dreamcore flag
// only forces every chamber onto it for testing.
func applyDreamTheme() {
	w := &worlds[WDream]
	w.Name, w.Subtitle = "Sunking's Road", "the sun is watching you"
	w.Hues = [6]color.RGBA{{255, 60, 170, 255}, {60, 200, 255, 255}, {255, 220, 40, 255}, {110, 240, 80, 255}, {255, 120, 40, 255}, {170, 90, 255, 255}}
	w.Floor, w.FloorAlt = color.RGBA{170, 130, 255, 255}, color.RGBA{130, 90, 220, 255}
	w.Outer = color.RGBA{40, 140, 50, 255}
	w.Wall, w.Trim = color.RGBA{40, 110, 50, 255}, color.RGBA{255, 120, 220, 255}
	w.Fog = color.RGBA{18, 34, 30, 255} // fades into the painted hills
	w.SkyTop, w.SkyBottom = color.RGBA{10, 6, 40, 255}, color.RGBA{170, 50, 170, 255}
	w.LightDir = V3{-0.45, 0.8, -0.35}
	w.LightCol = [3]float32{0.8, 0.72, 1.0}
	w.AmbCol = [3]float32{0.46, 0.38, 0.7}
	w.FogNear, w.FogFar = 70, 240
	w.PaddleBody, w.PaddleCap = color.RGBA{60, 40, 130, 255}, color.RGBA{255, 120, 220, 255}
	gs := grassStyles[WMeadow]
	gs.base, gs.tip = color.RGBA{14, 60, 30, 255}, color.RGBA{50, 150, 70, 255}
	gs.bloom = []color.RGBA{{255, 80, 200, 255}, {255, 230, 80, 255}, {120, 220, 255, 255}, {255, 255, 255, 255}}
	gs.glow = 0.25
	grassStyles[WDream] = gs
	if dreamcore && forceWorld < 0 {
		forceWorld = WDream // the test build plays every chamber on the new world
	}
	if playWorld >= 0 {
		forceWorld = playWorld
	}
}

// dreamTex loads the tiled textures the dreamcore props are wrapped in.
// Called before the scenery jobs start so they register in a fixed order.
type dreamTexSet struct {
	flowers, candy, brick, roof, grass, cobble, gem, sky *bakedTex
}

func dreamTextures() dreamTexSet {
	return dreamTexSet{
		flowers: bakedTexture("dream_flowers"),
		candy:   bakedTexture("dream_candy"),
		brick:   bakedTexture("dream_brick"),
		roof:    bakedTexture("dream_roof"),
		grass:   bakedTexture("ground_dream_meadow"),
		cobble:  bakedTexture("dream_meadow"),
		gem:     bakedTexture("dream_gem"),
		sky:     dreamSkyTex(),
	}
}

var (
	dreamSkyOnce sync.Once
	dreamSkyT    *bakedTex
)

// dreamSkyTex registers the galaxy painting so the lake can mirror it.
func dreamSkyTex() *bakedTex {
	dreamSkyOnce.Do(func() {
		if img := backdrop(WDream, 0); img != nil {
			b := img.Bounds()
			dreamSkyT = &bakedTex{id: registerTex(img), w: float32(b.Dx()), h: float32(b.Dy())}
		}
	})
	return dreamSkyT
}

// ---------------------------------------------------------------- textured meshes

const (
	uvBox = iota // each face takes the two axes it faces least along
	uvCyl        // wrapped around the Y axis (stripes spiral up cones and posts)
)

// DrawTex draws a mesh with a tiled texture. UVs come from the mesh's own
// scaled coordinates, so a texture keeps one size in world units whatever
// the mesh is scaled to; tile is the world size of one repeat. Face colours
// tint the texture and are lit as Draw lights them.
func (b *Batch) DrawTex(m *Mesh, pos, scale V3, rot Mat3, tex *bakedTex, tile float64, mode int, emissive float32) {
	b.DrawTexTint(m, pos, scale, rot, tex, tile, mode, color.RGBA{}, emissive)
}

// DrawTexTint is DrawTex with the face colours multiplied by tint (when tint.A > 0).
func (b *Batch) DrawTexTint(m *Mesh, pos, scale V3, rot Mat3, tex *bakedTex, tile float64, mode int, tint color.RGBA, emissive float32) {
	if b.obj.n > 0 {
		if b.obj.k > 0 {
			pos, scale = b.obj.apply(pos, scale)
		}
	} else {
		var ok bool
		if pos, scale, ok = b.autoClear(m, pos, scale, rot); !ok {
			return
		}
	}
	if tex == nil {
		b.Draw(m, pos, scale, rot, tint, emissive)
		return
	}
	if b.cut.on {
		if !b.keep(0.5 * float64(len(m.Tris))) {
			return
		}
		cut := b.cut
		b.cut.on = false
		defer func() { b.cut = cut }()
	}
	eye := b.cam.Eye
	tw, th := float64(tex.w), float64(tex.h)
	for _, t := range m.Tris {
		var q, p [3]V3
		for i := 0; i < 3; i++ {
			q[i] = V3{t.P[i].X * scale.X, t.P[i].Y * scale.Y, t.P[i].Z * scale.Z}
			p[i] = rot.Mul(q[i]).Add(pos)
		}
		n := p[1].Sub(p[0]).Cross(p[2].Sub(p[0]))
		if n.Len() < 1e-9 {
			continue
		}
		n = n.Norm()
		c := p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3)
		if n.Dot(eye.Sub(c)) <= 0 {
			continue
		}
		var uv [3][2]float32
		switch mode {
		case uvCyl:
			for i := 0; i < 3; i++ {
				a := math.Atan2(q[i].Z, q[i].X) / (2 * math.Pi)
				uv[i] = [2]float32{float32(a * tw * 2), float32(q[i].Y / tile * th)}
			}
			// mend the seam: keep the three u's within half a wrap of each other
			for i := 1; i < 3; i++ {
				for uv[i][0]-uv[0][0] > float32(tw) {
					uv[i][0] -= float32(tw * 2)
				}
				for uv[0][0]-uv[i][0] > float32(tw) {
					uv[i][0] += float32(tw * 2)
				}
			}
		default:
			ln := q[1].Sub(q[0]).Cross(q[2].Sub(q[0]))
			ax, ay, az := math.Abs(ln.X), math.Abs(ln.Y), math.Abs(ln.Z)
			for i := 0; i < 3; i++ {
				var u, v float64
				switch {
				case ax >= ay && ax >= az:
					u, v = q[i].Z, -q[i].Y
				case ay >= az:
					u, v = q[i].X, q[i].Z
				default:
					u, v = q[i].X, -q[i].Y
				}
				uv[i] = [2]float32{float32((u/tile + 0.5) * tw), float32((v/tile + 0.5) * th)}
			}
		}
		col := t.Col
		if tint.A > 0 {
			col = color.RGBA{uint8(int(col.R) * int(tint.R) / 255), uint8(int(col.G) * int(tint.G) / 255), uint8(int(col.B) * int(tint.B) / 255), col.A}
		}
		e := 1 + t.Emi + emissive
		b.QuadUV([4]V3{p[0], p[1], p[2], p[2]}, tex.id, [4][2]float32{uv[0], uv[1], uv[2], uv[2]},
			b.lit(col, n), [4]float32{e, e, e, e}, false)
	}
}

// unitBox is a cube from -1 to 1 with outward faces, coloured white so a
// texture shows true.
func unitBox() *Mesh {
	return cached("dreamBox", func() *Mesh {
		m := &Mesh{}
		X, Y, Z := V3{1, 0, 0}, V3{0, 1, 0}, V3{0, 0, 1}
		for _, f := range [][3]V3{{X, Y, Z}, {X.Mul(-1), Z, Y}, {Y, Z, X}, {Y.Mul(-1), X, Z}, {Z, X, Y}, {Z.Mul(-1), Y, X}} {
			n, u, v := f[0], f[1], f[2]
			q0, q1 := n.Sub(u).Sub(v), n.Add(u).Sub(v)
			q2, q3 := n.Add(u).Add(v), n.Sub(u).Add(v)
			w := color.RGBA{255, 255, 255, 255}
			m.Tris = append(m.Tris, Tri{P: [3]V3{q0, q1, q2}, Col: w}, Tri{P: [3]V3{q0, q2, q3}, Col: w})
		}
		return m
	})
}

func dreamSphere() *Mesh {
	return cached("dreamSphere", func() *Mesh {
		return Icosphere(1, 0.07, 11, color.RGBA{255, 255, 255, 255}, color.RGBA{214, 206, 224, 255})
	})
}

func dreamCone(sides int) *Mesh {
	return cached("dreamCone"+itoa(sides), func() *Mesh {
		return Lathe([][2]float64{{1, 0}, {0, 1}}, sides, []color.RGBA{{255, 255, 255, 255}}, false)
	})
}

func dreamCylinder(sides int) *Mesh {
	return cached("dreamCyl"+itoa(sides), func() *Mesh {
		// closed at both ends, so a rolled hedge never shows a hollow end
		return Lathe([][2]float64{{0, 0}, {1, 0}, {1, 1}}, sides, []color.RGBA{{255, 255, 255, 255}}, true)
	})
}

// ---------------------------------------------------------------- the sun with a face

var (
	dreamSunOnce sync.Once
	dreamSunImg  *ebiten.Image
)

func dreamSun() *ebiten.Image {
	dreamSunOnce.Do(func() {
		data, err := dreamFS.ReadFile("assets/dream/dream_sun.png")
		if err != nil {
			return
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return
		}
		dreamSunImg = ebiten.NewImageFromImage(img)
	})
	return dreamSunImg
}

// drawDreamSky draws the grinning sun over the painted galaxy. It hangs in
// the upper left, sways and slowly turns, and leans in grinning wider when
// the chamber shakes.
func (g *Game) drawDreamSky(dst *ebiten.Image, L *Level, horizonY, t float64) {
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	kick := math.Min(1, L.Shake*0.8)
	size := sh * (0.36 + 0.05*kick) * (1 + 0.018*math.Sin(t*1.7))
	cx := sw*0.25 + math.Sin(t*0.31)*sw*0.01 + skyPanX
	cy := math.Min(horizonY-sh*0.06, sh*0.2) + math.Sin(t*0.9)*sh*0.008
	// a warm halo and slowly turning lens glints behind the face
	if halo := dreamHalo(); halo != nil {
		hw := float64(halo.Bounds().Dx())
		for i, k := range []float64{2.1, 1.3} {
			hop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
			hop.GeoM.Translate(-hw/2, -hw/2)
			hop.GeoM.Scale(size*k/hw, size*k/hw)
			hop.GeoM.Translate(cx, cy)
			a := float32(0.55 + 0.1*math.Sin(t*1.3+float64(i)))
			if i == 0 {
				hop.ColorScale.Scale(1*a, 0.42*a, 0.16*a, a)
			} else {
				hop.ColorScale.Scale(1*a, 0.7*a, 0.3*a, a)
			}
			dst.DrawImage(halo, hop)
		}
		for i := 0; i < 2; i++ {
			sop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
			sop.GeoM.Translate(-hw/2, -hw/2)
			sop.GeoM.Scale(size*3.2/hw, size*0.05/hw)
			sop.GeoM.Rotate(t*0.05 + float64(i)*math.Pi/2 + 0.4)
			sop.GeoM.Translate(cx, cy)
			sop.ColorScale.Scale(0.55, 0.45, 0.3, 0.5)
			dst.DrawImage(halo, sop)
		}
	}
	// the Sunking himself, a low-poly 3D model, joins the world batch
	// (drawWorld calls drawSunking with this placement)
	dreamSunAt = [3]float64{cx, cy, size / 2}
}

var (
	dreamHaloOnce sync.Once
	dreamHaloImg  *ebiten.Image
)

// dreamHalo is a soft white radial glow, built once.
func dreamHalo() *ebiten.Image {
	dreamHaloOnce.Do(func() {
		const n = 256
		img := image.NewRGBA(image.Rect(0, 0, n, n))
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				d := math.Hypot(float64(x)-n/2+0.5, float64(y)-n/2+0.5) / (n / 2)
				a := math.Max(0, 1-d)
				a = a * a * a
				v := uint8(a * 255)
				img.SetRGBA(x, y, color.RGBA{v, v, v, v})
			}
		}
		dreamHaloImg = ebiten.NewImageFromImage(img)
	})
	return dreamHaloImg
}

// ---------------------------------------------------------------- the look

const dreamPostSrc = `//kage:unit pixels
package main

var Amount float
var Sat float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	n := imageSrc0At(src+vec2(1, 0)) + imageSrc0At(src-vec2(1, 0)) + imageSrc0At(src+vec2(0, 1)) + imageSrc0At(src-vec2(0, 1))
	d := imageSrc0At(src+vec2(1, 1)) + imageSrc0At(src-vec2(1, 1)) + imageSrc0At(src+vec2(1, -1)) + imageSrc0At(src+vec2(-1, 1))
	blur := (c*4 + n*2 + d) / 16
	s := c.rgb + (c.rgb-blur.rgb)*Amount
	l := dot(s, vec3(0.299, 0.587, 0.114))
	s = mix(vec3(l), s, Sat)
	s = (s-0.5)*1.07 + 0.5
	return vec4(clamp(s, 0, 1)*c.a, c.a)
}
`

const dreamBrightSrc = `//kage:unit pixels
package main

var Threshold float
var Boost float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	m := max(c.r, max(c.g, c.b))
	n := min(c.r, min(c.g, c.b))
	sat := m - n
	// bright colours glow, and saturated ones glow sooner, as in the style
	w := smoothstep(Threshold, 1.0, m+sat*0.35)
	return vec4(c.rgb*w*Boost, 1)
}
`

const dreamBlurSrc = `//kage:unit pixels
package main

var Dir vec2

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	s := imageSrc0At(src) * 0.2270270
	s += (imageSrc0At(src+Dir*1.3846154) + imageSrc0At(src-Dir*1.3846154)) * 0.3162162
	s += (imageSrc0At(src+Dir*3.2307692) + imageSrc0At(src-Dir*3.2307692)) * 0.0702703
	return vec4(s.rgb, 1)
}
`

var (
	dreamPostShader   *ebiten.Shader
	dreamBrightShader *ebiten.Shader
	dreamBlurShader   *ebiten.Shader
	dreamPostBuf      *ebiten.Image
	dreamBloomA       *ebiten.Image
	dreamBloomB       *ebiten.Image
	dreamPostFailed   bool
)

// dreamFilter runs the 3D view through the crunchy sharpen-and-saturate pass
// that gives the trend its embossed, over-processed look, then blooms it:
// bright, saturated colours bleed soft light into their surroundings.
func dreamFilter(screen *ebiten.Image) {
	if dreamPostFailed {
		return
	}
	if dreamPostShader == nil {
		sh, err1 := ebiten.NewShader([]byte(dreamPostSrc))
		br, err2 := ebiten.NewShader([]byte(dreamBrightSrc))
		bl, err3 := ebiten.NewShader([]byte(dreamBlurSrc))
		if err1 != nil || err2 != nil || err3 != nil {
			dreamPostFailed = true
			return
		}
		dreamPostShader, dreamBrightShader, dreamBlurShader = sh, br, bl
	}
	b := screen.Bounds()
	w, h := b.Dx(), b.Dy()
	qw, qh := max(1, w/4), max(1, h/4)
	if dreamPostBuf == nil || dreamPostBuf.Bounds().Dx() != w || dreamPostBuf.Bounds().Dy() != h {
		for _, im := range []*ebiten.Image{dreamPostBuf, dreamBloomA, dreamBloomB} {
			if im != nil {
				im.Deallocate()
			}
		}
		dreamPostBuf = ebiten.NewImage(w, h)
		dreamBloomA = ebiten.NewImage(qw, qh)
		dreamBloomB = ebiten.NewImage(qw, qh)
	}
	dreamPostBuf.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-b.Min.X), float64(-b.Min.Y))
	dreamPostBuf.DrawImage(screen, op)

	sop := &ebiten.DrawRectShaderOptions{Uniforms: map[string]any{"Amount": float32(1.1), "Sat": float32(1.28)}}
	sop.Images[0] = dreamPostBuf
	sop.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))
	screen.DrawRectShader(w, h, dreamPostShader, sop)

	if dreamNoBloom {
		return
	}
	// bloom source: the frame at quarter size, bright parts only, blurred wide
	dreamBloomB.Clear()
	dop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	dop.GeoM.Scale(float64(qw)/float64(w), float64(qh)/float64(h))
	dreamBloomB.DrawImage(dreamPostBuf, dop)
	bop := &ebiten.DrawRectShaderOptions{Uniforms: map[string]any{"Threshold": float32(0.5), "Boost": float32(1.25)}}
	bop.Images[0] = dreamBloomB
	dreamBloomA.DrawRectShader(qw, qh, dreamBrightShader, bop)
	for _, step := range []float32{1, 2.2} {
		for _, d := range [][2]float32{{step, 0}, {0, step}} {
			src, dstImg := dreamBloomA, dreamBloomB
			if d[1] != 0 {
				src, dstImg = dreamBloomB, dreamBloomA
			}
			lop := &ebiten.DrawRectShaderOptions{Uniforms: map[string]any{"Dir": []float32{d[0], d[1]}}}
			lop.Images[0] = src
			dstImg.DrawRectShader(qw, qh, dreamBlurShader, lop)
		}
	}

	// add the glow back over the sharpened frame
	gop := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	gop.GeoM.Scale(float64(w)/float64(qw), float64(h)/float64(qh))
	gop.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))
	gop.ColorScale.Scale(0.75, 0.75, 0.75, 0.75)
	screen.DrawImage(dreamBloomA, gop)
}
