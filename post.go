package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Post-processing: bright-pass bloom, colour grading and a vignette.

const brightSrc = `//kage:unit pixels
package main

var Threshold float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src).rgb
	l := dot(c, vec3(0.299, 0.587, 0.114))
	k := clamp((l-Threshold)/(1.0-Threshold), 0.0, 1.0)
	return vec4(c*k*k, 1.0)
}
`

const blurSrc = `//kage:unit pixels
package main

var Dir vec2

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src) * 0.227027
	c += imageSrc0At(src+Dir*1.0) * 0.1945946
	c += imageSrc0At(src-Dir*1.0) * 0.1945946
	c += imageSrc0At(src+Dir*2.0) * 0.1216216
	c += imageSrc0At(src-Dir*2.0) * 0.1216216
	c += imageSrc0At(src+Dir*3.0) * 0.054054
	c += imageSrc0At(src-Dir*3.0) * 0.054054
	c += imageSrc0At(src+Dir*4.0) * 0.016216
	c += imageSrc0At(src-Dir*4.0) * 0.016216
	return vec4(c.rgb, 1.0)
}
`

const gradeSrc = `//kage:unit pixels
package main

var Tint vec3
var Saturation float
var Contrast float
var Vignette float
var Lift vec3

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src).rgb + imageSrc1At(src).rgb
	l := dot(c, vec3(0.299, 0.587, 0.114))
	c = mix(vec3(l), c, Saturation)
	c = (c-0.5)*Contrast + 0.5
	c = c*Tint + Lift
	uv := (src - imageSrc0Origin()) / imageSrc0Size()
	d := distance(uv, vec2(0.5, 0.5))
	c *= 1.0 - Vignette*smoothstep(0.38, 0.9, d)
	return vec4(clamp(c, 0.0, 1.0), 1.0)
}
`

// upscaleSrc enlarges the scene when it renders below native resolution:
// a 16-tap Catmull-Rom filter, clamped to the local min/max of the four
// nearest texels so edges stay crisp without halos (in the spirit of AMD's
// FSR 1 spatial upscaler).
const upscaleSrc = `//kage:unit pixels
package main

var Sharp float

func wts(f float) vec4 {
	return vec4(
		f*(-0.5+f*(1.0-0.5*f)),
		1.0+f*f*(-2.5+1.5*f),
		f*(0.5+f*(2.0-1.5*f)),
		f*f*(-0.5+0.5*f))
}

func tap(p vec2) vec3 {
	o := imageSrc0Origin()
	sz := imageSrc0Size()
	q := clamp(p, o+0.5, o+sz-0.5)
	return imageSrc0UnsafeAt(q).rgb
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	t := src - 0.5
	b := floor(t)
	f := t - b
	c0 := b + 0.5
	wx := wts(f.x)
	wy := wts(f.y)
	acc := vec3(0)
	for j := 0; j < 4; j++ {
		row := vec3(0)
		y := c0.y + float(j) - 1.0
		row += tap(vec2(c0.x-1.0, y)) * wx.x
		row += tap(vec2(c0.x, y)) * wx.y
		row += tap(vec2(c0.x+1.0, y)) * wx.z
		row += tap(vec2(c0.x+2.0, y)) * wx.w
		if j == 0 {
			acc += row * wy.x
		} else if j == 1 {
			acc += row * wy.y
		} else if j == 2 {
			acc += row * wy.z
		} else {
			acc += row * wy.w
		}
	}
	a := tap(c0)
	bb := tap(c0 + vec2(1, 0))
	cc := tap(c0 + vec2(0, 1))
	dd := tap(c0 + vec2(1, 1))
	mn := min(min(a, bb), min(cc, dd))
	mx := max(max(a, bb), max(cc, dd))
	lin := mix(mix(a, bb, f.x), mix(cc, dd, f.x), f.y)
	c := mix(lin, clamp(acc, mn, mx), Sharp)
	return vec4(c, 1.0)
}
`

type Post struct {
	upscale             *ebiten.Shader
	graded              *ebiten.Image
	bright, blur, grade *ebiten.Shader
	scene               *ebiten.Image
	small, small2       *ebiten.Image
	bloomFull           *ebiten.Image
	ok                  bool
}

type Grade struct {
	Bloom, Threshold          float32
	Tint, Lift                [3]float32
	Saturation, Contrast, Vig float32
}

func newPost() *Post {
	p := &Post{}
	var err error
	if p.bright, err = ebiten.NewShader([]byte(brightSrc)); err != nil {
		return p
	}
	if p.blur, err = ebiten.NewShader([]byte(blurSrc)); err != nil {
		return p
	}
	if p.grade, err = ebiten.NewShader([]byte(gradeSrc)); err != nil {
		return p
	}
	if p.upscale, err = ebiten.NewShader([]byte(upscaleSrc)); err != nil {
		p.upscale = nil // fall back to bilinear
	}
	p.ok = true
	return p
}

// Target returns the offscreen image the 3D scene should be drawn into.
func (p *Post) Target(w, h int) *ebiten.Image {
	if p.scene == nil || p.scene.Bounds().Dx() != w || p.scene.Bounds().Dy() != h {
		if p.scene != nil {
			p.scene.Deallocate()
			p.small.Deallocate()
			p.small2.Deallocate()
			p.bloomFull.Deallocate()
			if p.graded != nil {
				p.graded.Deallocate()
				p.graded = nil
			}
		}
		p.scene = ebiten.NewImage(w, h)
		sw, sh := max(1, w/4), max(1, h/4)
		p.small = ebiten.NewImage(sw, sh)
		p.small2 = ebiten.NewImage(sw, sh)
		p.bloomFull = ebiten.NewImage(w, h)
	}
	p.scene.Clear()
	return p.scene
}

// Apply composites the scene with bloom and grading onto dst, upscaling
// it when the scene was rendered below dst's resolution.
func (p *Post) Apply(dst *ebiten.Image, g Grade, enabled bool) {
	w, h := p.scene.Bounds().Dx(), p.scene.Bounds().Dy()
	dw, dh := dst.Bounds().Dx(), dst.Bounds().Dy()
	out := dst
	if w != dw || h != dh {
		if p.graded == nil {
			p.graded = ebiten.NewImage(w, h)
		}
		out = p.graded
	}
	p.applyAt(out, g, enabled)
	if out != dst {
		p.upscaleTo(dst, out)
	}
}

func (p *Post) upscaleTo(dst, src *ebiten.Image) {
	sw, sh := float32(src.Bounds().Dx()), float32(src.Bounds().Dy())
	dw, dh := float32(dst.Bounds().Dx()), float32(dst.Bounds().Dy())
	if p.upscale == nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(float64(dw/sw), float64(dh/sh))
		dst.DrawImage(src, op)
		return
	}
	vs := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: dw, DstY: 0, SrcX: sw, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: dw, DstY: dh, SrcX: sw, SrcY: sh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: dh, SrcX: 0, SrcY: sh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	op := &ebiten.DrawTrianglesShaderOptions{Images: [4]*ebiten.Image{src}, Uniforms: map[string]any{"Sharp": float32(0.85)}}
	dst.DrawTrianglesShader(vs, []uint16{0, 1, 2, 0, 2, 3}, p.upscale, op)
}

func (p *Post) applyAt(dst *ebiten.Image, g Grade, enabled bool) {
	if !p.ok || !enabled {
		dst.DrawImage(p.scene, nil)
		return
	}
	w, h := p.scene.Bounds().Dx(), p.scene.Bounds().Dy()
	sw, sh := p.small.Bounds().Dx(), p.small.Bounds().Dy()
	// downsample
	p.small2.Clear()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(sw)/float64(w), float64(sh)/float64(h))
	p.small2.DrawImage(p.scene, op)
	// bright pass
	p.small.Clear()
	p.small.DrawRectShader(sw, sh, p.bright, &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{p.small2}, Uniforms: map[string]any{"Threshold": g.Threshold}})
	// blur (two separable passes, twice)
	for i := 0; i < 2; i++ {
		p.small2.Clear()
		p.small2.DrawRectShader(sw, sh, p.blur, &ebiten.DrawRectShaderOptions{
			Images: [4]*ebiten.Image{p.small}, Uniforms: map[string]any{"Dir": []float32{1.5, 0}}})
		p.small.Clear()
		p.small.DrawRectShader(sw, sh, p.blur, &ebiten.DrawRectShaderOptions{
			Images: [4]*ebiten.Image{p.small2}, Uniforms: map[string]any{"Dir": []float32{0, 1.5}}})
	}
	// upscale bloom
	p.bloomFull.Clear()
	op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(w)/float64(sw), float64(h)/float64(sh))
	op.ColorScale.Scale(g.Bloom, g.Bloom, g.Bloom, 1)
	p.bloomFull.DrawImage(p.small, op)
	// grade + composite
	dst.DrawRectShader(w, h, p.grade, &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{p.scene, p.bloomFull},
		Uniforms: map[string]any{
			"Tint": g.Tint[:], "Lift": g.Lift[:], "Saturation": g.Saturation,
			"Contrast": g.Contrast, "Vignette": g.Vig,
		}})
}

var worldGrades = map[int]Grade{
	WNeon:    {Bloom: 1.1, Threshold: 0.55, Tint: [3]float32{1.02, 0.98, 1.06}, Saturation: 1.12, Contrast: 1.06, Vig: 0.55},
	WMeadow:  {Bloom: 0.45, Threshold: 0.72, Tint: [3]float32{1.05, 1.0, 0.92}, Lift: [3]float32{0.02, 0.01, 0}, Saturation: 1.08, Contrast: 1.04, Vig: 0.38},
	WCaverns: {Bloom: 1.0, Threshold: 0.5, Tint: [3]float32{0.94, 1.0, 1.08}, Saturation: 1.1, Contrast: 1.1, Vig: 0.62},
	WRuins:   {Bloom: 0.4, Threshold: 0.78, Tint: [3]float32{1.06, 1.0, 0.9}, Lift: [3]float32{0.02, 0.015, 0}, Saturation: 1.05, Contrast: 1.08, Vig: 0.42},
	WClouds:  {Bloom: 0.3, Threshold: 0.9, Tint: [3]float32{0.97, 0.98, 1.02}, Saturation: 1.1, Contrast: 1.06, Vig: 0.3},
	WMachine: {Bloom: 0.9, Threshold: 0.55, Tint: [3]float32{1.06, 0.98, 0.9}, Saturation: 1.0, Contrast: 1.12, Vig: 0.6},
}

var _ = color.RGBA{}
