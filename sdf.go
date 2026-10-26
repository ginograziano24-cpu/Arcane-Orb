package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Signed-distance-field shape renderer for the interface.
//
// Every rounded or chamfered box, circle, outline, soft glow, capsule line
// and progress arc is one quad drawn with a tiny shader that computes
// exact, smoothly anti-aliased coverage per pixel. Unlike path-based
// anti-aliasing this needs no offscreen buffers, stays crisp at any
// resolution, and consecutive shapes batch into a single draw call.

const sdfSrc = `//kage:unit pixels
package main

func sdBox(p vec2, b vec2, r float) float {
	q := abs(p) - b + vec2(r)
	return length(max(q, vec2(0))) + min(max(q.x, q.y), 0.0) - r
}

func sdChamfer(p vec2, b vec2, r float) float {
	a := abs(p)
	q := a - b
	d := max(q.x, q.y)
	c := (a.x + a.y - (b.x + b.y - r)) * 0.70710678
	return max(d, c)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	kind := floor(custom.w/1000.0 + 0.0001)
	w := custom.w - kind*1000.0
	p := srcPos
	if kind > 7.5 {
		ang := atan2(p.x, -p.y)
		if ang < 0.0 {
			ang += 6.2831853
		}
		dr := abs(length(p)-custom.x) - w*0.5
		cov := clamp(0.5-dr, 0.0, 1.0)
		cov *= clamp((custom.y*6.2831853-ang)*custom.x+0.5, 0.0, 1.0)
		return color * cov
	}
	d := 0.0
	if mod(kind, 2.0) > 0.5 {
		d = sdChamfer(p, custom.xy, custom.z)
	} else {
		d = sdBox(p, custom.xy, custom.z)
	}
	if kind > 3.5 {
		g := clamp(1.0-d/max(w, 1.0), 0.0, 1.0)
		return color * g * g
	}
	if kind > 1.5 {
		return color * clamp(0.5-(abs(d)-w*0.5), 0.0, 1.0)
	}
	return color * clamp(0.5-d, 0.0, 1.0)
}
`

var (
	sdfShader *ebiten.Shader
	sdfVerts  = make([]ebiten.Vertex, 4)
	sdfIdx    = []uint16{0, 1, 2, 0, 2, 3}
)

// shp describes a shape in screen pixels.
type shp struct {
	cx, cy    float64 // centre
	hw, hh, r float64 // half extents, corner radius
	rot       float64 // rotation (radians)
	ch        bool    // chamfered corners instead of rounded
	arc       bool    // progress ring: r = radius, hw = fraction 0..1
	line      bool    // segment from (x0,y0) to (x1,y1), stroked as a capsule
	x0, y0    float64
	x1, y1    float64
}

const (
	sdfFill = 0
	sdfLine = 2
	sdfGlow = 4
	sdfArc  = 8
	sdfMaxW = 999
)

func gradCol(c0, c1 color.RGBA, gr grad, x, y float64) color.RGBA {
	dx, dy := gr.x1-gr.x0, gr.y1-gr.y0
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = ((x-gr.x0)*dx + (y-gr.y0)*dy) / l2
	}
	return prem(mixc(c0, c1, t), 1)
}

func sdfDraw(dst *ebiten.Image, s shp, kind int, w float64, c0, c1 color.RGBA, gr grad, additive bool) {
	if sdfShader == nil {
		sh, err := ebiten.NewShader([]byte(sdfSrc))
		if err != nil {
			panic(err)
		}
		sdfShader = sh
	}
	w = math.Min(w, sdfMaxW)
	pad := 1.5
	var cust [4]float32
	ex, ey := s.hw, s.hh
	switch {
	case kind == sdfArc:
		pad += w / 2
		ex, ey = s.r, s.r
		cust = [4]float32{float32(s.r), float32(clamp01(s.hw)), 0, float32(sdfArc*1000 + w)}
	default:
		if kind == sdfLine {
			pad += w / 2
		} else if kind == sdfGlow {
			pad += w
		}
		k := kind
		if s.ch {
			k++
		}
		cust = [4]float32{float32(s.hw), float32(s.hh), float32(math.Min(s.r, math.Min(s.hw, s.hh))), float32(float64(k*1000) + w)}
	}
	ex += pad
	ey += pad
	cs, sn := math.Cos(s.rot), math.Sin(s.rot)
	for i, c := range [4][2]float64{{-ex, -ey}, {ex, -ey}, {ex, ey}, {-ex, ey}} {
		x := s.cx + c[0]*cs - c[1]*sn
		y := s.cy + c[0]*sn + c[1]*cs
		col := gradCol(c0, c1, gr, x, y)
		sdfVerts[i] = ebiten.Vertex{
			DstX: float32(x), DstY: float32(y), SrcX: float32(c[0]), SrcY: float32(c[1]),
			ColorR: float32(col.R) / 255, ColorG: float32(col.G) / 255, ColorB: float32(col.B) / 255, ColorA: float32(col.A) / 255,
			Custom0: cust[0], Custom1: cust[1], Custom2: cust[2], Custom3: cust[3],
		}
	}
	op := &ebiten.DrawTrianglesShaderOptions{}
	if additive {
		op.Blend = ebiten.BlendLighter
	}
	dst.DrawTrianglesShader(sdfVerts, sdfIdx, sdfShader, op)
}

// ---------------------------------------------------------------- shape constructors

func boxPath(x, y, w, h, r float64, chamfer bool) shp {
	r = math.Max(0, math.Min(r, math.Min(w, h)/2))
	return shp{cx: x + w/2, cy: y + h/2, hw: w / 2, hh: h / 2, r: r, ch: chamfer && r > 0.5}
}

func rectPath(x, y, w, h float64) shp { return boxPath(x, y, w, h, 0, false) }

func circlePath(cx, cy, r float64) shp { return shp{cx: cx, cy: cy, hw: r, hh: r, r: r} }

// ngonPath approximates a badge polygon with a chamfered (octagonal) shape.
func ngonPath(cx, cy, r float64, n int, rot float64) shp {
	return shp{cx: cx, cy: cy, hw: r * 0.9, hh: r * 0.9, r: r * 0.42, ch: true}
}

// diamondPath is a cut gem: a perfect diamond when rx == ry, an elongated
// hexagonal cut otherwise.
func diamondPath(cx, cy, rx, ry float64) shp {
	return shp{cx: cx, cy: cy, hw: rx, hh: ry, r: math.Min(rx, ry), ch: true}
}

// arcPath is a progress ring starting at a0 (expected -π/2, i.e. 12 o'clock).
func arcPath(cx, cy, r, a0, a1 float64) shp {
	return shp{cx: cx, cy: cy, r: r, hw: (a1 - a0) / (2 * math.Pi), arc: true}
}

func linePath(x0, y0, x1, y1 float64) shp {
	return shp{line: true, x0: x0, y0: y0, x1: x1, y1: y1}
}

// ---------------------------------------------------------------- painting

func fillPath(dst *ebiten.Image, s shp, c0, c1 color.RGBA, gr grad, additive bool) {
	sdfDraw(dst, s, sdfFill, 0, c0, c1, gr, additive)
}

func strokePath(dst *ebiten.Image, s shp, w float64, c color.RGBA, additive bool) {
	switch {
	case s.arc:
		sdfDraw(dst, s, sdfArc, w, c, c, grad{}, additive)
	case s.line:
		dx, dy := s.x1-s.x0, s.y1-s.y0
		l := math.Hypot(dx, dy)
		cp := shp{cx: (s.x0 + s.x1) / 2, cy: (s.y0 + s.y1) / 2, hw: l/2 + w/2, hh: w / 2, r: w / 2, rot: math.Atan2(dy, dx)}
		sdfDraw(dst, cp, sdfFill, 0, c, c, grad{}, additive)
	default:
		sdfDraw(dst, s, sdfLine, w, c, c, grad{}, additive)
	}
}

// glowPath paints a soft halo that fades out over `blur` pixels.
func glowPath(dst *ebiten.Image, s shp, blur float64, c color.RGBA) {
	sdfDraw(dst, s, sdfGlow, blur, c, c, grad{}, true)
}
