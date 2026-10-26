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

type Post struct {
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

// Apply composites the scene with bloom and grading onto dst.
func (p *Post) Apply(dst *ebiten.Image, g Grade, enabled bool) {
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
