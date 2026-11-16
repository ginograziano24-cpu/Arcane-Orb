package main

import (
	"math"
	"sync"
)

// Camera clearance. Scenery placed for the play view can end up right in
// front of the lens once the camera moves (photo mode's orbit, the boss
// intro, the finale push-in, trailer shots): a gear or crystal fills the
// screen, or the near plane slices through it. Each prop opens a scope
// with its centre and radius; while the prop blocks the camera, everything
// it draws sinks and shrinks smoothly out of the way, and grows back once
// the camera has passed. The play camera never triggers it.

type objScope struct {
	c V3      // the prop's centre
	r float64 // its radius
	k float64 // 0 in place .. 1 fully cleared
	n int     // open scopes: a prop drawn inside another moves with it
}

// apply moves a part of the current prop for the clearance in progress.
func (o *objScope) apply(pos, scale V3) (V3, V3) {
	e := o.k * o.k * (3 - 2*o.k)
	f := 1 - e
	p := o.c.Add(pos.Sub(o.c).Mul(f))
	p.Y -= e * o.r * 0.8 // sink as it shrinks, so it reads as settling into the ground
	return p, scale.Mul(f)
}

// camBlock reports how much a sphere at c (radius r) is in the camera's way:
// 0 not at all, 1 hard against the lens or squarely between it and the board.
func camBlock(cam *Camera, c V3, r float64) float64 {
	eye, tgt := cam.Eye, cam.Target
	// right against the lens
	d := c.Sub(eye).Len()
	near := smoothstep(r+8, r+3, d)
	// inside the cone from the eye to the board, well in front of it
	v := tgt.Sub(eye)
	L2 := v.Dot(v)
	occ := 0.0
	if L2 > 1e-6 {
		t := c.Sub(eye).Dot(v) / L2
		if t > 0 && t < 0.72 {
			perp := c.Sub(eye.Add(v.Mul(t))).Len()
			reach := r + HalfW*0.85*t // the board's silhouette widens toward it
			occ = smoothstep(reach+1.5, reach-0.5, perp) * smoothstep(0.0, 0.08, t) * smoothstep(0.72, 0.55, t)
		}
	}
	return math.Max(near, occ)
}

// objBegin opens a prop's scope. It reports false when the prop is fully
// cleared this frame, so the caller can skip drawing it. Always pair with
// objEnd (defer is fine).
func (b *Batch) objBegin(x, y, z, r float64) bool {
	if b.obj.n > 0 {
		b.obj.n++
		return b.obj.k < 0.995
	}
	c := V3{x, y, z}
	k := 0.0
	if b.cam != nil {
		k = camBlock(b.cam, c, r)
	}
	b.obj = objScope{c: c, r: r, k: k, n: 1}
	return k < 0.995
}

// objEnd closes the current prop's scope.
func (b *Batch) objEnd() {
	if b.obj.n--; b.obj.n <= 0 {
		b.obj = objScope{}
	}
}

// Meshes drawn outside any prop scope are cleared one by one, by their own
// bounds, unless they belong to the board (bricks, paddle, walls, orb).

type meshBound struct {
	c V3      // centre, in the mesh's own space
	r float64 // radius around it
}

var meshBounds sync.Map // *Mesh -> meshBound

func boundOf(m *Mesh) meshBound {
	if v, ok := meshBounds.Load(m); ok {
		return v.(meshBound)
	}
	lo, hi := V3{math.Inf(1), math.Inf(1), math.Inf(1)}, V3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, t := range m.Tris {
		for _, p := range t.P {
			lo = V3{math.Min(lo.X, p.X), math.Min(lo.Y, p.Y), math.Min(lo.Z, p.Z)}
			hi = V3{math.Max(hi.X, p.X), math.Max(hi.Y, p.Y), math.Max(hi.Z, p.Z)}
		}
	}
	b := meshBound{}
	if len(m.Tris) > 0 {
		b.c = lo.Add(hi).Mul(0.5)
		b.r = hi.Sub(lo).Len() / 2
	}
	meshBounds.Store(m, b)
	return b
}

// onBoard reports whether a point is on or right beside the playing field.
func onBoard(p V3) bool {
	return math.Abs(p.X) <= HalfW+2.5 && p.Z >= LoseZ-2 && p.Z <= FarZ+3 && p.Y < 14
}

// autoClear clears a lone mesh (no prop scope open) that blocks the camera.
// It reports false when the mesh should not be drawn at all this frame.
func (b *Batch) autoClear(m *Mesh, pos, scale V3, rot Mat3) (V3, V3, bool) {
	if b.cam == nil || onBoard(pos) {
		return pos, scale, true
	}
	mb := boundOf(m)
	sc := math.Max(math.Abs(scale.X), math.Max(math.Abs(scale.Y), math.Abs(scale.Z)))
	r := mb.r * sc
	if r <= 0 || r > 30 {
		return pos, scale, true // empty, or terrain-sized: never in the way
	}
	c := rot.Mul(V3{mb.c.X * scale.X, mb.c.Y * scale.Y, mb.c.Z * scale.Z}).Add(pos)
	k := camBlock(b.cam, c, r)
	if k <= 0 {
		return pos, scale, true
	}
	if k >= 0.995 {
		return pos, scale, false
	}
	o := objScope{c: c, r: r, k: k}
	p, s := o.apply(pos, scale)
	return p, s, true
}
