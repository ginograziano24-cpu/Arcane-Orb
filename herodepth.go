package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

// Depth-tested hero rendering. The heroes are dense scanned meshes (hair
// over faces, eyelids over eyes, straps over clothes), and painter's
// sorting by triangle centre gets those overlaps wrong: hair pokes through
// cheeks, edges flicker as the idle sways. Ebiten has no depth buffer, so
// this builds one in three passes:
//
//  1. coarse depth: every triangle into an 8-bit image with a minimum blend
//     (the nearest surface wins), depth carried in the vertex colour;
//  2. fine depth: only fragments in the winning coarse step write the
//     remainder, again with a minimum blend, into a second image (about
//     14 bits in all);
//  3. colour: every triangle again with a shader that throws away the
//     fragments behind the stored depth and textures the rest.
//
// Depth is 1/z, which is linear across the screen, normalised to the
// hero's own depth range each frame.

const heroDepthCoarseSrc = `//kage:unit pixels
package main

func Fragment(dst vec4, src vec2, color vec4, custom vec4) vec4 {
	return vec4(custom.x, custom.x, custom.x, 1)
}
`

const heroDepthFineSrc = `//kage:unit pixels
package main

func Fragment(dst vec4, src vec2, color vec4, custom vec4) vec4 {
	p := dst.xy - imageDstOrigin() + imageSrc0Origin()
	q := imageSrc0UnsafeAt(p).r * 255
	d := custom.x * 255
	// generous window: the GPU's float-to-8-bit rounding is not exact
	if d > q+1.5 || d < q-1 {
		discard()
	}
	f := clamp((d-q+1)/2.5, 0, 1)
	return vec4(f, f, f, 1)
}
`

const heroDepthColorSrc = `//kage:unit pixels
package main

func texel(p vec2) vec4 {
	// bilinear, so the 2x supersample resolves smoothly
	p -= 0.5
	f := fract(p)
	b := floor(p) + 0.5
	a := imageSrc0At(b)
	c := imageSrc0At(b + vec2(1, 0))
	e := imageSrc0At(b + vec2(0, 1))
	g := imageSrc0At(b + vec2(1, 1))
	return mix(mix(a, c, f.x), mix(e, g, f.x), f.y)
}

func Fragment(dst vec4, src vec2, color vec4, custom vec4) vec4 {
	// images 1 and 2 are addressed through image 0's pixel space
	p := dst.xy - imageDstOrigin() + imageSrc0Origin()
	q := imageSrc1UnsafeAtFromSrc0Pos(p).r * 255
	d := custom.x * 255
	if d > q+1.5 {
		discard()
	}
	fs := imageSrc2UnsafeAtFromSrc0Pos(p).r
	f := clamp((d-q+1)/2.5, 0, 1)
	if f > fs+2/255.0 {
		discard()
	}
	t := texel(src)
	if t.a > 0 {
		t.rgb /= t.a
	}
	return vec4(t.rgb*color.rgb, 1)
}
`

type heroDepthKit struct {
	coarse, fine, color *ebiten.Shader
	failed              bool
	dC, dF              *ebiten.Image // depth targets, sized to the hero target
	white               *ebiten.Image
	vs                  []ebiten.Vertex
	is                  []uint16
	dumped              map[string]bool
}

var heroDK heroDepthKit

func (k *heroDepthKit) ready() bool {
	if k.failed {
		return false
	}
	if k.color != nil {
		return true
	}
	var err error
	if k.coarse, err = ebiten.NewShader([]byte(heroDepthCoarseSrc)); err != nil {
		k.failed = true
		return false
	}
	if k.fine, err = ebiten.NewShader([]byte(heroDepthFineSrc)); err != nil {
		k.failed = true
		return false
	}
	if k.color, err = ebiten.NewShader([]byte(heroDepthColorSrc)); err != nil {
		k.failed = true
		return false
	}
	k.white = ebiten.NewImage(4, 4)
	k.white.Fill(color.White)
	return true
}

var minBlend = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOne,
	BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationMin,
	BlendOperationAlpha:         ebiten.BlendOperationMin,
}

// drawHeroDepth renders the hero into st.rt with real per-pixel depth. It
// reports false (having drawn nothing) when the shaders are unavailable, so
// the caller can fall back to the painter's path.
func drawHeroDepth(st *heroStage, m *heroMesh, rot Mat3, rim color.RGBA, dim float64) bool {
	k := &heroDK
	if !k.ready() {
		return false
	}
	tex := texImages[m.tex]
	if tex == nil {
		return false
	}
	W, H := st.rt.Bounds().Dx(), st.rt.Bounds().Dy()
	if k.dC == nil || k.dC.Bounds().Dx() != W || k.dC.Bounds().Dy() != H {
		if k.dC != nil {
			k.dC.Deallocate()
			k.dF.Deallocate()
		}
		k.dC = ebiten.NewImage(W, H)
		k.dF = ebiten.NewImage(W, H)
	}
	cam := &st.cam
	eye := cam.Eye
	key := V3{-0.55, 0.7, -0.75}.Norm()
	back := V3{0.5, 0.35, 0.9}.Norm()
	fill := V3{0.7, 0.1, -0.4}.Norm()
	rimF := [3]float64{float64(rim.R) / 255, float64(rim.G) / 255, float64(rim.B) / 255}

	// one vertex per triangle corner, front faces only
	vs := k.vs[:0]
	invNear, invFar := 0.0, math.Inf(1)
	for i := 0; i+2 < len(m.pos); i += 3 {
		var p [3]V3
		for c := 0; c < 3; c++ {
			p[c] = rot.Mul(m.pos[i+c])
		}
		fn := p[1].Sub(p[0]).Cross(p[2].Sub(p[0]))
		if fn.Len() < 1e-12 {
			continue
		}
		// no back-face culling: the depth test keeps the nearest surface
		// whichever way a triangle is wound (some scans have flipped
		// patches, which culling would punch holes in or show from inside)
		ctr := p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3)
		flip := 1.0
		if fn.Dot(eye.Sub(ctr)) < 0 {
			flip = -1
		}
		var sh [3]float64
		rimAmt := 0.0
		for c := 0; c < 3; c++ {
			n := rot.Mul(m.nrm[i+c])
			if n.Dot(eye.Sub(p[c])) < 0 && flip < 0 {
				n = n.Mul(-1) // light the side we see
			}
			s := 0.5 + 0.55*math.Max(0, n.Dot(key)) + 0.16*math.Max(0, n.Dot(fill)) + 0.1*n.Y
			sh[c] = math.Min(1.15, s)
			rimAmt += math.Max(0, n.Dot(back)) / 3
		}
		r := rimAmt * rimAmt * 0.55
		tc := [3]float64{
			dim * math.Min(1, 1-r+r*rimF[0]*1.6),
			dim * math.Min(1, 1-r+r*rimF[1]*1.6),
			dim * math.Min(1, 1-r+r*rimF[2]*1.6)}
		for c := 0; c < 3; c++ {
			sx, sy, z := cam.Project(p[c])
			iz := 1 / z
			invNear = math.Max(invNear, iz)
			invFar = math.Min(invFar, iz)
			vs = append(vs, ebiten.Vertex{
				DstX: float32(sx), DstY: float32(sy),
				SrcX: m.uv[i+c][0], SrcY: m.uv[i+c][1],
				ColorR: float32(tc[0] * sh[c]), ColorG: float32(tc[1] * sh[c]), ColorB: float32(tc[2] * sh[c]), ColorA: 1,
				Custom0: float32(iz),
			})
		}
	}
	k.vs = vs
	if len(vs) == 0 {
		return true
	}
	// normalise depth: 0 nearest, 1 farthest, with a little margin
	span := invNear - invFar
	if span < 1e-9 {
		span = 1e-9
	}
	for i := range vs {
		d := (invNear - float64(vs[i].Custom0)) / span
		vs[i].Custom0 = float32(0.01 + 0.98*d)
	}
	// draw in chunks that fit the index range of one call
	const chunk = 30000 // a multiple of 3, well inside 16-bit indices
	if cap(k.is) < chunk {
		k.is = make([]uint16, chunk)
		for i := range k.is[:chunk] {
			k.is[i] = uint16(i)
		}
	}
	each := func(f func(v []ebiten.Vertex)) {
		for lo := 0; lo < len(vs); lo += chunk {
			hi := min(lo+chunk, len(vs))
			f(vs[lo:hi])
		}
	}
	k.dC.Fill(color.White)
	k.dF.Fill(color.White)
	// pass 1: coarse depth
	each(func(v []ebiten.Vertex) {
		op := &ebiten.DrawTrianglesShaderOptions{Blend: minBlend}
		k.dC.DrawTrianglesShader(v, k.is[:len(v)], k.coarse, op)
	})
	// pass 2: fine depth within the winning coarse step
	each(func(v []ebiten.Vertex) {
		op := &ebiten.DrawTrianglesShaderOptions{Blend: minBlend}
		op.Images[0] = k.dC
		k.dF.DrawTrianglesShader(v, k.is[:len(v)], k.fine, op)
	})
	// pass 3: colour, nearest surface only
	each(func(v []ebiten.Vertex) {
		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Images[0] = tex
		op.Images[1] = k.dC
		op.Images[2] = k.dF
		st.rt.DrawTrianglesShader(v, k.is[:len(v)], k.color, op)
	})
	if auditOn {
		heroDepthDump(k, st.rt, st.key.id)
	}
	return true
}

// heroDepthDump saves the depth images once per hero during the audit, so a
// broken depth pass can be seen rather than guessed at.
func heroDepthDump(k *heroDepthKit, rt *ebiten.Image, id string) {
	if k.dumped == nil {
		k.dumped = map[string]bool{}
	}
	if k.dumped[id] {
		return
	}
	k.dumped[id] = true
	save := func(img *ebiten.Image, name string) {
		r := img.Bounds()
		px := image.NewRGBA(r)
		img.ReadPixels(px.Pix)
		exe, _ := os.Executable()
		dir := filepath.Join(filepath.Dir(exe), "shots")
		_ = os.MkdirAll(dir, 0o755)
		if f, err := os.Create(filepath.Join(dir, name)); err == nil {
			_ = png.Encode(f, px)
			f.Close()
		}
	}
	save(k.dC, "dbg-depthc-"+id+".png")
	save(k.dF, "dbg-depthf-"+id+".png")
	save(rt, "dbg-color-"+id+".png")
}
