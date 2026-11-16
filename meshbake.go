package main

import (
	"image/color"
	"math"
	"sync"
	"sync/atomic"
)

// Baked meshes (ROADMAP section S): most models are drawn at the same
// place, size, turn and tint frame after frame (arena walls, posts,
// scenery). Their world-space corners, face normals and lit colours don't
// depend on the camera, so they're worked out once and kept; each frame
// only projects the corners and fills the quads. The output is
// bit-for-bit what drawShared produces (TestBakedMatchesShared).
//
// A placement is baked the second frame in a row it's seen, so moving
// things (critters, the paddle) never pay for it, and placements unused
// for a couple of seconds are dropped.

type bakeKey struct {
	m        *Mesh
	pos, sc  V3
	rot      Mat3
	tint     color.RGBA
	emissive float32
}

type bakedTri struct {
	ix  [3]int32
	n   V3 // unnormalised face normal, for the facing test
	ctr V3
	nn  V3 // unit normal, for lighting
	col color.RGBA
	e   float32
	ok  bool // false: a degenerate triangle drawShared skips
}

type bakeGeo struct {
	pos  []V3
	tris []bakedTri
	ctr  V3      // bounding sphere of the placed geometry
	rad  float64 // (whole-placement culling in drawBaked)
}

// bakeCols is a geometry's lit colours under one light.
type bakeCols struct {
	light            V3
	lightCol, ambCol [3]float32
	rgba             [][4]float32
}

type bakeEntry struct {
	last atomic.Int64 // bake frame last drawn
	geo  atomic.Pointer[bakeGeo]
	cols atomic.Pointer[bakeCols]
}

var (
	bakeCache sync.Map // bakeKey -> *bakeEntry
	bakeFrame atomic.Int64
	bakeOff   bool // tests and comparisons
)

// bakeTick advances the frame and drops stale placements (main thread,
// before the frame's drawing starts).
func bakeTick() {
	f := bakeFrame.Add(1)
	if f%120 != 0 {
		return
	}
	bakeCache.Range(func(k, v any) bool {
		if f-v.(*bakeEntry).last.Load() > 120 {
			bakeCache.Delete(k)
		}
		return true
	})
}

// bakeMesh works out a placement's world-space corners and faces exactly
// as drawShared does.
func bakeMesh(m *Mesh, pos, scale V3, rot Mat3, tint color.RGBA, emissive float32) *bakeGeo {
	t := m.topo
	g := &bakeGeo{pos: make([]V3, len(t.pos)), tris: make([]bakedTri, len(m.Tris))}
	for i, lp := range t.pos {
		q := V3{lp.X * scale.X, lp.Y * scale.Y, lp.Z * scale.Z}
		g.pos[i] = rot.Mul(q).Add(pos)
	}
	if n := len(g.pos); n > 0 {
		var c V3
		for _, p := range g.pos {
			c = c.Add(p)
		}
		c = c.Mul(1 / float64(n))
		r := 0.0
		for _, p := range g.pos {
			r = math.Max(r, p.Sub(c).Len())
		}
		g.ctr, g.rad = c, r+1e-6
	}
	useTint := tint.A > 0
	for ti := range m.Tris {
		tr := &m.Tris[ti]
		ix := t.tri[ti]
		a, bb, c := g.pos[ix[0]], g.pos[ix[1]], g.pos[ix[2]]
		bt := &g.tris[ti]
		bt.ix = ix
		bt.n = bb.Sub(a).Cross(c.Sub(a))
		bt.ctr = a.Add(bb).Add(c).Mul(1.0 / 3)
		l := bt.n.Len()
		if l < 1e-9 {
			continue
		}
		bt.ok = true
		bt.nn = V3{bt.n.X / l, bt.n.Y / l, bt.n.Z / l}
		col := tr.Col
		if useTint && !tr.Fixed {
			col = color.RGBA{uint8(int(col.R) * int(tint.R) / 255), uint8(int(col.G) * int(tint.G) / 255), uint8(int(col.B) * int(tint.B) / 255), col.A}
		}
		bt.col = col
		bt.e = 1 + tr.Emi + emissive
	}
	return g
}

// colors lights a baked geometry with b's light.
func (b *Batch) bakeColors(g *bakeGeo) *bakeCols {
	c := &bakeCols{light: b.light, lightCol: b.lightCol, ambCol: b.ambCol, rgba: make([][4]float32, len(g.tris))}
	for i := range g.tris {
		bt := &g.tris[i]
		if !bt.ok {
			continue
		}
		e, col := bt.e, bt.col
		dl := float32(math.Max(0, bt.nn.Dot(b.light)))
		al := float32(col.A) / 255
		r := float32(litChan(col.R, b.ambCol[0]+b.lightCol[0]*dl*0.85)) / 255 * e
		gg := float32(litChan(col.G, b.ambCol[1]+b.lightCol[1]*dl*0.85)) / 255 * e
		bl := float32(litChan(col.B, b.ambCol[2]+b.lightCol[2]*dl*0.85)) / 255 * e
		c.rgba[i] = [4]float32{r, gg, bl, al}
	}
	return c
}

// drawCached is drawShared through the bake cache: true when it drew.
func (b *Batch) drawCached(m *Mesh, pos, scale V3, rot Mat3, tint color.RGBA, emissive float32) bool {
	if bakeOff {
		return false
	}
	k := bakeKey{m, pos, scale, rot, tint, emissive}
	f := bakeFrame.Load()
	v, ok := bakeCache.Load(k)
	if !ok {
		e := &bakeEntry{}
		e.last.Store(f)
		bakeCache.Store(k, e) // first sighting: just remember it
		return false
	}
	e := v.(*bakeEntry)
	prev := e.last.Swap(f)
	g := e.geo.Load()
	if g == nil {
		if prev == f {
			return false // drawn twice this frame but never before: wait
		}
		g = bakeMesh(m, pos, scale, rot, tint, emissive)
		e.geo.Store(g)
	}
	c := e.cols.Load()
	if c == nil || c.light != b.light || c.lightCol != b.lightCol || c.ambCol != b.ambCol {
		c = b.bakeColors(g)
		e.cols.Store(c)
	}
	b.drawBaked(g, c)
	return true
}

// drawBaked emits a baked placement's quads: drawShared's projection,
// culling and vertex fill, with the camera-free work already done.
func (b *Batch) drawBaked(g *bakeGeo, c *bakeCols) {
	eye := b.cam.Eye
	cam := b.cam
	if g.rad > 0 && sphereOffscreen(cam, g.ctr, g.rad) {
		return // every triangle would be culled off screen anyway
	}
	if b.back != nil && g.rad > 0 && b.skipBack(g.ctr.Sub(eye).Dot(cam.fwd), g.rad) {
		return // wholly in the cached background layer
	}
	if cap(b.proj) < len(g.pos) {
		b.proj = make([]projected, len(g.pos))
	}
	pr := b.proj[:len(g.pos)]
	span := b.fogFar - b.fogNear
	for i, p := range g.pos {
		sx, sy, d := cam.Project(p)
		pr[i] = projected{p: p, sx: float32(sx), sy: float32(sy), d: d, fog: float32(math.Max(0, math.Min(1, (d-b.fogNear)/span)))}
	}
	fc := b.fogColor
	fw, fh := float32(cam.W), float32(cam.H)
	for ti := range g.tris {
		bt := &g.tris[ti]
		ix := bt.ix
		a, bb, cc := &pr[ix[0]], &pr[ix[1]], &pr[ix[2]]
		ax, bx, cx := a.sx, bb.sx, cc.sx
		ay, by, cy := a.sy, bb.sy, cc.sy
		near := triBehind(a, bb, cc)
		if !near && ((ax < 0 && bx < 0 && cx < 0) || (ax > fw && bx > fw && cx > fw) ||
			(ay < 0 && by < 0 && cy < 0) || (ay > fh && by > fh && cy > fh)) {
			continue
		}
		if bt.n.Dot(eye.Sub(bt.ctr)) <= 0 || !bt.ok {
			continue
		}
		col := c.rgba[ti]
		r, gg, bl, al := col[0], col[1], col[2], col[3]
		if near {
			b.clipTri(a.p, bb.p, cc.p, col)
			continue
		}
		b.quads = append(b.quads, quad{})
		q := &b.quads[len(b.quads)-1]
		for k, v := range [3]*projected{a, bb, cc} {
			fog := v.fog
			vx := &q.v[k]
			vx.DstX, vx.DstY = v.sx, v.sy
			vx.SrcX, vx.SrcY = sprWhite.X0, sprWhite.Y0
			vx.ColorR = (r*(1-fog) + fc[0]*fog) * al
			vx.ColorG = (gg*(1-fog) + fc[1]*fog) * al
			vx.ColorB = (bl*(1-fog) + fc[2]*fog) * al
			vx.ColorA = al
		}
		q.v[3] = q.v[2]
		q.v[1].SrcX = sprWhite.X1
		q.v[2].SrcX, q.v[2].SrcY = sprWhite.X1, sprWhite.Y1
		q.v[3].SrcX, q.v[3].SrcY = sprWhite.X0, sprWhite.Y1
		sum := 0.0
		sum += a.d
		sum += bb.d
		sum += cc.d
		sum += cc.d
		q.depth = sum / 4
	}
}

// sphereOffscreen reports whether a sphere lies wholly in front of the
// camera and wholly outside one side of the view frustum, so that every
// triangle inside it fails the per-triangle off-screen test.
func sphereOffscreen(cam *Camera, c V3, r float64) bool {
	d := c.Sub(cam.Eye)
	z := d.Dot(cam.fwd)
	if z-r < 0.06 {
		return false // touches the near plane: leave it to the clipper
	}
	x, y := d.Dot(cam.right), d.Dot(cam.up)
	hw, hh := cam.W/2, cam.H/2
	nx := math.Hypot(cam.f, hw)
	ny := math.Hypot(cam.f, hh)
	return (cam.f*x+hw*z)/nx < -r || (-cam.f*x+hw*z)/nx < -r ||
		(cam.f*y+hh*z)/ny < -r || (-cam.f*y+hh*z)/ny < -r
}
