package main

import "github.com/hajimehoshi/ebiten/v2"

// Near-plane clipping.
//
// Camera.Project clamps a point's depth to nearClip, so a corner behind
// the camera used to land at an enormous screen position and drag its
// quad across the whole view: scenery "zooming in" and smearing when the
// camera moved close to it (photo mode, boss intros, the finale push-in).
// A polygon with any corner at or behind the near plane is now cut at the
// plane in world space first. The cut is rare (only geometry that crosses
// the camera), so it takes the plain per-vertex path.

const nearClip = 0.05 // must match the clamp in Camera.Project

// depthOf is a point's distance in front of the camera (unclamped).
func (c *Camera) depthOf(p V3) float64 { return p.Sub(c.Eye).Dot(c.fwd) }

// clipVert is one corner of a polygon being clipped.
type clipVert struct {
	p     V3
	u, v  float32
	shade float32
}

// emitClipped clips the polygon pts against the near plane and adds what
// is left as quads (a fan, two triangles per quad). Returns false when
// nothing was in front of the camera.
func (b *Batch) emitClipped(pts []clipVert, col [4]float32, tex uint8, glow bool) bool {
	var buf [8]clipVert
	out := buf[:0]
	n := len(pts)
	for i := 0; i < n; i++ {
		a, c := pts[i], pts[(i+1)%n]
		da, dc := b.cam.depthOf(a.p)-nearClip*2, b.cam.depthOf(c.p)-nearClip*2
		if da >= 0 {
			out = append(out, a)
		}
		if (da >= 0) != (dc >= 0) {
			t := da / (da - dc)
			lerp := func(x, y float32) float32 { return x + (y-x)*float32(t) }
			out = append(out, clipVert{
				p: a.p.Add(c.p.Sub(a.p).Mul(t)),
				u: lerp(a.u, c.u), v: lerp(a.v, c.v), shade: lerp(a.shade, c.shade),
			})
		}
		if len(out) >= len(buf)-1 {
			break
		}
	}
	if len(out) < 3 {
		return false
	}
	for i := 1; i+1 < len(out); i += 2 {
		k := [4]int{0, i, i + 1, i + 2}
		if i+2 >= len(out) {
			k[3] = i + 1 // a lone triangle: repeat its last corner
		}
		q := quad{tex: tex}
		sum := 0.0
		for j, idx := range k {
			cv := out[idx]
			var d float64
			q.v[j], d = b.vtx(cv.p, cv.u, cv.v, col, cv.shade)
			sum += d
		}
		q.depth = sum / 4
		b.add(&q, glow)
	}
	return true
}

// clipQuad is the near-plane path for Quad and QuadUV: true when it
// handled the quad (some corner was behind the camera).
func (b *Batch) clipQuad(p *[4]V3, uv *[4][2]float32, col [4]float32, shades *[4]float32, tex uint8, glow bool) bool {
	behind := 0
	for i := range p {
		if b.cam.depthOf(p[i]) <= nearClip*2 {
			behind++
		}
	}
	if behind == 0 {
		return false
	}
	if behind < 4 {
		var pts [4]clipVert
		for i := range p {
			pts[i] = clipVert{p[i], uv[i][0], uv[i][1], shades[i]}
		}
		b.emitClipped(pts[:], col, tex, glow)
	}
	return true
}

// clipTri is the near-plane path for the mesh triangle fillers: the face
// colour (already lit, before fog) is applied per clipped corner.
func (b *Batch) clipTri(a, c, d V3, col [4]float32) {
	pts := [3]clipVert{
		{a, sprWhite.X0, sprWhite.Y0, 1},
		{c, sprWhite.X1, sprWhite.Y0, 1},
		{d, sprWhite.X1, sprWhite.Y1, 1},
	}
	b.emitClipped(pts[:], col, 0, false)
}

// triBehind reports whether a projected triangle touches the near plane
// (Project clamped one of its depths).
func triBehind(a, c, d *projected) bool {
	return a.d <= nearClip*2 || c.d <= nearClip*2 || d.d <= nearClip*2
}

var _ = ebiten.Vertex{}
