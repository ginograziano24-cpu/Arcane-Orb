package main

import (
	"image/color"
	"math"
	"math/rand"
	"sync"
)

// Procedural flora. Every tree, bush, hedge and topiary in the worlds is
// grown here rather than assembled from primitive shapes: wood is a set of
// tubes following bent paths, with bark ridges cut into the surface and
// roots splaying into the ground; foliage is made of individual leaves
// (each a folded two-tone blade with a midrib), needles, fronds or five
// petal flowers. Each species has a few seeded variants, built once and
// cached; distant plants use a lighter variant with fewer, larger leaves.
//
// Units match the old trees: an oak is ~7.5 tall at scale 1.

type floraSet struct {
	wood  *Mesh // trunk, roots, limbs (drawn with the bark texture)
	leaf  *Mesh // leaves, needles, petals (drawn plain, swaying)
	top   float64
	crown V3         // centre of the foliage, for culling
	plain bool       // wood keeps its own colours (no bark texture)
	impOf *floraSet  // set: draw as a picture card of that plant (impostor.go)
	cards []leafCard // set: the canopy as leaf cards (impostor.go), leaf mesh as fallback
}

var (
	floraMu   sync.Mutex
	floraSets = map[string]*floraSet{}
)

func floraGet(key string, build func() *floraSet) *floraSet {
	floraMu.Lock()
	s, ok := floraSets[key]
	floraMu.Unlock()
	if ok {
		return s
	}
	s = build()
	if s.wood == nil {
		s.wood = &Mesh{}
	}
	if s.leaf == nil {
		s.leaf = &Mesh{}
	}
	s.wood.topology()
	s.leaf.topology()
	floraMu.Lock()
	if old, ok := floraSets[key]; ok {
		s = old
	} else {
		floraSets[key] = s
	}
	floraMu.Unlock()
	return s
}

// floraPx is roughly how many pixels a plant of the given size (world
// units, about its height) at p spans on screen; 0 without a camera.
func floraPx(B *Batch, p V3, size float64) float64 {
	if B == nil || B.cam == nil {
		return 1e9
	}
	eye := B.cam.Eye
	if lodEye != (V3{}) && lodEye.Sub(eye).Len() < 3 {
		eye = lodEye
	}
	d := p.Sub(eye).Dot(B.cam.fwd)
	if d < 0.5 {
		return 1e9
	}
	return size * B.cam.f / d * floraDetail
}

// lodEye is the camera's position before shake and knock-back are added
// (set each frame by drawWorld). Detail choices measure from it, so a hit
// that jolts the camera can't flip plants near a threshold between their
// full, card and impostor forms (a visible pop).
var lodEye V3

// floraDetail scales how large plants count as on screen when choosing
// their detail: High 1, Medium 0.6, Low 0.35 (set each frame by drawWorld),
// so lower presets drop to the light variants and picture cards sooner.
var floraDetail = 1.0

// hiddenByTerrain reports whether a hill crest stands between the camera and
// the point top (so everything below it is out of sight too).
func hiddenByTerrain(B *Batch, top V3) bool {
	if B == nil || B.cam == nil {
		return false
	}
	if _, ok := terrainStyles[activeTerrain]; !ok {
		return false
	}
	eye := B.cam.Eye
	for k := 1; k <= 6; k++ {
		t := float64(k) / 7
		p := eye.Add(top.Sub(eye).Mul(t))
		if terrainY(activeTerrain, p.X, p.Z) > p.Y+0.3 {
			return true
		}
	}
	return false
}

// floraFar reports whether a plant at p is small enough on screen to use
// its light variant (a tree-sized plant under ~260 px tall).
func floraFar(B *Batch, p V3) bool { return floraPx(B, p, 6) < 260 }

// floraTiny: smaller still (under ~90 px): the light variant with its
// leaves' back faces stripped, which halves the visible leaf triangles.
func floraTiny(B *Batch, p V3) bool { return floraPx(B, p, 6) < 90 }

// stripBacks returns a copy of m without the back face of any two-sided
// pair (a triangle whose three corners repeat an earlier one's).
func stripBacks(m *Mesh) *Mesh {
	type key [3]V3
	sortK := func(t Tri) key {
		k := key{t.P[0], t.P[1], t.P[2]}
		less := func(a, b V3) bool {
			if a.X != b.X {
				return a.X < b.X
			}
			if a.Y != b.Y {
				return a.Y < b.Y
			}
			return a.Z < b.Z
		}
		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				if less(k[j], k[i]) {
					k[i], k[j] = k[j], k[i]
				}
			}
		}
		return k
	}
	seen := map[key]bool{}
	out := &Mesh{Tris: make([]Tri, 0, len(m.Tris))}
	for _, t := range m.Tris {
		k := sortK(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out.Tris = append(out.Tris, t)
	}
	return out
}

// ---------------------------------------------------------------- builder

type floraB struct {
	m        *Mesh
	r        *rand.Rand
	tint     bool // faces take the draw tint (Fixed=false)
	oneSided bool // leaves get no back faces (distant variants)
	emi      float32
	stats    int
}

func newFloraB(seed int64) *floraB {
	return &floraB{m: &Mesh{}, r: rand.New(rand.NewSource(seed))}
}

func (b *floraB) f(lo, hi float64) float64 { return lo + (hi-lo)*b.r.Float64() }

func (b *floraB) unit() V3 {
	for {
		v := V3{b.f(-1, 1), b.f(-1, 1), b.f(-1, 1)}
		if l := v.Len(); l > 0.05 && l <= 1 {
			return v.Mul(1 / l)
		}
	}
}

func (b *floraB) inBall() V3 {
	for {
		v := V3{b.f(-1, 1), b.f(-1, 1), b.f(-1, 1)}
		if v.Len() <= 1 {
			return v
		}
	}
}

// face adds a triangle whose front faces along n.
func (b *floraB) face(p0, p1, p2, n V3, c color.RGBA) {
	if p1.Sub(p0).Cross(p2.Sub(p0)).Dot(n) < 0 {
		p1, p2 = p2, p1
	}
	b.m.Tris = append(b.m.Tris, Tri{P: [3]V3{p0, p1, p2}, Col: c, Emi: b.emi, Fixed: !b.tint})
}

func perpTo(v V3) V3 {
	a := V3{0, 1, 0}
	if math.Abs(v.Y) > 0.9 {
		a = V3{1, 0, 0}
	}
	return v.Cross(a).Norm()
}

// rotAbout rotates v about unit axis k by angle a (Rodrigues).
func rotAbout(v, k V3, a float64) V3 {
	c, s := math.Cos(a), math.Sin(a)
	return v.Mul(c).Add(k.Cross(v).Mul(s)).Add(k.Mul(k.Dot(v) * (1 - c)))
}

// ---------------------------------------------------------------- wood

type barkStyle struct {
	ridge, groove color.RGBA
	moss          color.RGBA
	mossH         float64 // moss grows on the up/back faces below this height
	ridges        float64 // ridges around the trunk
	depth         float64 // how deep they cut (fraction of radius)
}

// tube sweeps rings along path. rad gives the radius at each point; the
// bark ridges wander along the length so they read as real bark plates.
func (b *floraB) tube(path []V3, rad []float64, sides int, bk barkStyle, phase float64) {
	n := len(path)
	if n < 2 {
		return
	}
	tan := make([]V3, n)
	for i := range path {
		a, c := path[max(0, i-1)], path[min(n-1, i+1)]
		tan[i] = c.Sub(a).Norm()
	}
	N := perpTo(tan[0])
	rings := make([][]V3, n)
	gs := make([][]float64, n)
	for i := 0; i < n; i++ {
		if i > 0 {
			N = N.Sub(tan[i].Mul(N.Dot(tan[i]))).Norm()
		}
		Bv := tan[i].Cross(N)
		rings[i] = make([]V3, sides)
		gs[i] = make([]float64, sides)
		for j := 0; j < sides; j++ {
			a := float64(j) / float64(sides) * 2 * math.Pi
			g := 0.5 + 0.5*math.Sin(a*bk.ridges+0.9*math.Sin(float64(i)*0.55+phase)+phase)
			d := 1 + bk.depth*(g*g-0.45)
			rings[i][j] = path[i].Add(N.Mul(math.Cos(a) * rad[i] * d)).Add(Bv.Mul(math.Sin(a) * rad[i] * d))
			gs[i][j] = g
		}
	}
	for i := 0; i+1 < n; i++ {
		ax := path[i].Add(path[i+1]).Mul(0.5)
		for j := 0; j < sides; j++ {
			j1 := (j + 1) % sides
			p00, p01, p11, p10 := rings[i][j], rings[i][j1], rings[i+1][j1], rings[i+1][j]
			ctr := p00.Add(p01).Add(p11).Add(p10).Mul(0.25)
			out := ctr.Sub(ax)
			if out.Len() < 1e-9 {
				continue
			}
			out = out.Norm()
			g := (gs[i][j] + gs[i][j1] + gs[i+1][j] + gs[i+1][j1]) / 4
			col := mixc(bk.groove, bk.ridge, g*g*1.2)
			col = mixc(col, lighten(col, 0.25), b.r.Float64()*0.25)
			if bk.mossH > 0 && ctr.Y < bk.mossH && out.Dot(V3{-0.2, 0.75, 0.6}.Norm()) > 0.3 {
				k := (1 - ctr.Y/bk.mossH) * (0.5 + 0.5*b.r.Float64())
				col = mixc(col, bk.moss, k*0.85)
			}
			b.face(p00, p10, p11, out, col)
			b.face(p00, p11, p01, out, darken(col, 0.96))
		}
	}
}

// limb grows a bent path from start along dir.
func (b *floraB) limb(start, dir V3, length float64, segs int, bend, up float64) []V3 {
	path := []V3{start}
	p := start
	d := dir.Norm()
	step := length / float64(segs)
	for k := 0; k < segs; k++ {
		d = d.Add(b.unit().Mul(bend)).Add(V3{0, up, 0}).Norm()
		p = p.Add(d.Mul(step))
		path = append(path, p)
	}
	return path
}

func taper(n int, r0, r1 float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / float64(max(1, n-1))
		out[i] = r0 + (r1-r0)*math.Pow(t, 0.8)
	}
	return out
}

// roots splay from the base of a trunk of radius r0 into the ground.
func (b *floraB) roots(base V3, r0 float64, count int, bk barkStyle) {
	for k := 0; k < count; k++ {
		a := float64(k)/float64(count)*2*math.Pi + b.f(-0.3, 0.3)
		out := V3{math.Cos(a), 0, math.Sin(a)}
		reach := r0 * b.f(2.4, 3.6)
		path := []V3{base.Add(V3{0, r0 * 0.9, 0}).Add(out.Mul(r0 * 0.3))}
		for s := 1; s <= 4; s++ {
			t := float64(s) / 4
			side := perpTo(out).Mul(b.f(-0.15, 0.15) * reach)
			y := r0*0.9*(1-t)*(1-t) - 0.12*t*t*r0*2
			path = append(path, base.Add(out.Mul(r0*0.3+reach*t)).Add(side.Mul(t)).Add(V3{0, y, 0}))
		}
		b.tube(path, taper(len(path), r0*0.5, r0*0.06), 6, bk, float64(k))
	}
}

// ---------------------------------------------------------------- leaves

type leafPal struct {
	dark, light, accent color.RGBA
	under               color.RGBA
	accentP             float64
}

// leaf adds a folded blade: base at p, pointing along dir, facing nrm.
// The two halves get slightly different shades so the midrib reads.
func (b *floraB) leaf(p, dir, nrm V3, L, W, fold float64, col, back color.RGBA, twoSided bool) {
	side := dir.Cross(nrm)
	if side.Len() < 1e-6 {
		return
	}
	side = side.Norm()
	nrm = side.Cross(dir).Norm()
	if nrm.Dot(nrm) == 0 {
		return
	}
	tip := p.Add(dir.Mul(L))
	mid := p.Add(dir.Mul(L * 0.42))
	l := mid.Add(side.Mul(W / 2)).Sub(nrm.Mul(fold * W))
	r := mid.Sub(side.Mul(W / 2)).Sub(nrm.Mul(fold * W))
	b.face(p, l, tip, nrm, col)
	b.face(p, tip, r, nrm, darken(col, 0.9))
	if twoSided {
		b.face(p, tip, l, nrm.Mul(-1), back)
		b.face(p, r, tip, nrm.Mul(-1), darken(back, 0.92))
	}
}

func (b *floraB) leafCol(pal leafPal, t float64) color.RGBA {
	if pal.accentP > 0 && b.r.Float64() < pal.accentP {
		return mixc(pal.accent, pal.light, b.r.Float64()*0.4)
	}
	c := mixc(pal.dark, pal.light, math.Max(0, math.Min(1, 0.25+0.6*t+b.f(-0.18, 0.18))))
	return c
}

// cluster scatters n leaves around c; cc is the crown centre (leaves face
// away from it so the canopy's outer surface catches the light).
func (b *floraB) cluster(c, cc V3, n int, rad, L, W float64, pal leafPal, lo, hi float64) {
	for k := 0; k < n; k++ {
		off := b.inBall().Mul(rad)
		off.Y *= 0.8
		p := c.Add(off)
		out := p.Sub(cc)
		if out.Len() < 1e-6 {
			out = V3{0, 1, 0}
		}
		nrm := out.Norm().Mul(0.55).Add(V3{0, 0.55, 0}).Add(b.unit().Mul(0.55)).Norm()
		d0 := off.Add(V3{0, -0.25 * rad, 0})
		if d0.Len() < 1e-6 {
			d0 = b.unit()
		}
		d0 = d0.Norm()
		dir := d0.Sub(nrm.Mul(d0.Dot(nrm)))
		if dir.Len() < 1e-4 {
			dir = perpTo(nrm)
		}
		dir = dir.Norm()
		t := (p.Y - lo) / math.Max(0.01, hi-lo)
		col := b.leafCol(pal, t)
		s := b.f(0.75, 1.2)
		b.leaf(p.Sub(dir.Mul(L*0.3*s)), dir, nrm, L*s, W*s, 0.18, col, pal.under, !b.oneSided)
	}
}

// flower is a five (or n) petal bloom facing nrm, cupped slightly.
func (b *floraB) flower(c, nrm V3, rad float64, petal, centre color.RGBA, petals int) {
	b.flowerX(c, nrm, rad, petal, centre, petals, true)
}

func (b *floraB) flowerX(c, nrm V3, rad float64, petal, centre color.RGBA, petals int, backs bool) {
	u := perpTo(nrm)
	v := nrm.Cross(u)
	rot := b.f(0, 2*math.Pi)
	hw := math.Pi / float64(petals) * 0.85
	for k := 0; k < petals; k++ {
		a := rot + float64(k)/float64(petals)*2*math.Pi
		dirAt := func(a float64) V3 { return u.Mul(math.Cos(a)).Add(v.Mul(math.Sin(a))) }
		tip := c.Add(dirAt(a).Mul(rad)).Add(nrm.Mul(rad * 0.28))
		s1 := c.Add(dirAt(a - hw).Mul(rad * 0.6)).Add(nrm.Mul(rad * 0.1))
		s2 := c.Add(dirAt(a + hw).Mul(rad * 0.6)).Add(nrm.Mul(rad * 0.1))
		pc := mixc(petal, lighten(petal, 0.35), b.r.Float64()*0.5)
		b.face(c, s1, tip, nrm, pc)
		b.face(c, tip, s2, nrm, darken(pc, 0.93))
		if backs {
			b.face(c, tip, s1, nrm.Mul(-1), darken(pc, 0.7))
			b.face(c, s2, tip, nrm.Mul(-1), darken(pc, 0.66))
		}
	}
	// the heart: a little raised star
	h := c.Add(nrm.Mul(rad * 0.3))
	for k := 0; k < 4; k++ {
		a := rot + float64(k)/4*2*math.Pi
		p0 := c.Add(u.Mul(math.Cos(a) * rad * 0.25)).Add(v.Mul(math.Sin(a) * rad * 0.25)).Add(nrm.Mul(rad * 0.12))
		a1 := a + math.Pi/2
		p1 := c.Add(u.Mul(math.Cos(a1) * rad * 0.25)).Add(v.Mul(math.Sin(a1) * rad * 0.25)).Add(nrm.Mul(rad * 0.12))
		b.face(p0, p1, h, nrm, mixc(centre, darken(centre, 0.8), float64(k%2)))
	}
}

// ---------------------------------------------------------------- species

var (
	barkOak = barkStyle{ridge: color.RGBA{150, 116, 88, 255}, groove: color.RGBA{70, 52, 40, 255},
		moss: color.RGBA{150, 196, 110, 255}, mossH: 1.4, ridges: 7, depth: 0.22}
	palOak = leafPal{dark: color.RGBA{44, 92, 38, 255}, light: color.RGBA{150, 200, 84, 255},
		accent: color.RGBA{214, 196, 82, 255}, under: color.RGBA{96, 128, 70, 255}, accentP: 0.04}
)

// growTree builds a broadleaf tree. lite uses fewer, larger leaves.
func growTree(seed int64, bk barkStyle, pal leafPal, lite bool, blossom *color.RGBA) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 999)
	lean := V3{wb.f(-0.25, 0.25), 0, wb.f(-0.25, 0.25)}
	// trunk
	trunk := wb.limb(V3{0, -0.35, 0}, V3{0, 1, 0}.Add(lean.Mul(0.4)), 2.7, 5, 0.05, 0.05)
	tr := taper(len(trunk), 0.5, 0.32)
	tr[0], tr[1] = 0.66, 0.5
	wb.tube(trunk, tr, 9, bk, float64(seed))
	wb.roots(V3{0, -0.35, 0}, 0.48, map[bool]int{false: 6, true: 4}[lite], bk)
	type tip struct{ p, d V3 }
	var tips []tip
	var grow func(start, dir V3, length, r0 float64, depth int)
	grow = func(start, dir V3, length, r0 float64, depth int) {
		segs := []int{3, 3, 2}[min(depth, 2)]
		sides := []int{7, 5, 4}[min(depth, 2)]
		path := wb.limb(start, dir, length, segs, 0.18, 0.08)
		wb.tube(path, taper(len(path), r0, r0*0.45), sides, bk, float64(depth)+wb.f(0, 3))
		end := path[len(path)-1]
		d := end.Sub(path[len(path)-2]).Norm()
		if depth >= 2 || (lite && depth >= 1) {
			tips = append(tips, tip{end, d})
			if len(path) > 2 {
				tips = append(tips, tip{path[len(path)-2], d})
			}
			return
		}
		kids := 2 + wb.r.Intn(2)
		for k := 0; k < kids; k++ {
			t := 0.45 + 0.5*float64(k+1)/float64(kids)
			at := path[min(len(path)-1, int(t*float64(len(path)-1)+0.5))]
			ax := perpTo(d)
			ax = rotAbout(ax, d, wb.f(0, 2*math.Pi))
			cd := rotAbout(d, ax, wb.f(0.5, 0.95)).Add(V3{0, 0.15, 0}).Norm()
			grow(at, cd, length*wb.f(0.55, 0.7), r0*0.55, depth+1)
		}
		tips = append(tips, tip{end, d})
	}
	top := trunk[len(trunk)-1]
	limbs := 3 + wb.r.Intn(2)
	for k := 0; k < limbs; k++ {
		a := float64(seed)*1.3 + float64(k)/float64(limbs)*2*math.Pi + wb.f(-0.3, 0.3)
		th := wb.f(0.75, 1.1)
		dir := V3{math.Cos(a) * math.Sin(th), math.Cos(th), math.Sin(a) * math.Sin(th)}
		grow(top.Sub(V3{0, wb.f(0, 0.6), 0}), dir, wb.f(2.5, 3.1), 0.25, 0)
	}
	grow(top, V3{0, 1, 0}.Add(lean.Mul(0.3)), 1.9, 0.2, 1) // the leader
	// crown extents
	lo, hi := math.Inf(1), math.Inf(-1)
	var cc V3
	for _, t := range tips {
		cc = cc.Add(t.p)
		lo, hi = math.Min(lo, t.p.Y), math.Max(hi, t.p.Y)
	}
	cc = cc.Mul(1 / float64(len(tips)))
	n, L, W, rad := 15, 0.72, 0.43, 1.0
	if lite {
		n, L, W, rad = 11, 1.25, 0.75, 1.15
	}
	// the crown: an ellipsoid dome over the limbs. Leaf clusters sit in its
	// outer shell, each fed by a twig from the nearest branch tip, so the
	// canopy reads full from any side.
	var ext float64
	for _, t := range tips {
		ext = math.Max(ext, math.Hypot(t.p.X-cc.X, t.p.Z-cc.Z))
	}
	rx := ext + 0.7
	ry := math.Max(1.6, (hi-lo)*0.6+1.0)
	cen := V3{cc.X, (lo+hi)/2 + 0.5, cc.Z}
	clusters := 22
	if lite {
		clusters = 16
	}
	for k := 0; k < clusters; k++ {
		// golden spiral over the upper ~80% of the ellipsoid, pushed out
		v := (float64(k) + 0.5) / float64(clusters)
		th := math.Acos(1 - 1.7*v)
		a := float64(k) * 2.39996
		sh := lb.f(0.68, 0.95)
		p := cen.Add(V3{math.Sin(th) * math.Cos(a) * rx * sh, math.Cos(th) * ry * sh, math.Sin(th) * math.Sin(a) * rx * sh})
		lb.cluster(p, cen, n, rad, L, W, pal, cen.Y-ry, cen.Y+ry)
		if blossom != nil {
			for q := 0; q < n/6; q++ {
				fp := p.Add(lb.inBall().Mul(rad * 0.9))
				nrm := fp.Sub(cen).Norm().Add(V3{0, 0.6, 0}).Norm()
				lb.flowerX(fp, nrm, 0.2, *blossom, color.RGBA{255, 220, 120, 255}, 5, false)
			}
		}
		// twig from the nearest tip
		best, bd := tips[0].p, math.Inf(1)
		for _, t := range tips {
			if d := t.p.Sub(p).Len(); d < bd {
				best, bd = t.p, d
			}
		}
		if bd > 0.15 && !lite {
			mid := best.Add(p.Sub(best).Mul(0.5)).Add(V3{0, 0.12, 0})
			wb.tube([]V3{best, mid, p}, []float64{0.045, 0.03, 0.012}, 3, bk, 0)
		}
	}
	// the inner canopy: a sparser darker layer so you don't see through
	inner := pal
	inner.light = mixc(pal.dark, pal.light, 0.4)
	if !lite {
		for _, t := range tips {
			lb.cluster(t.p, cen, n/3, rad*0.85, L, W, inner, cen.Y-ry, cen.Y+ry)
		}
	}
	hi = cen.Y + ry
	return &floraSet{wood: wb.m, leaf: lb.m, top: hi + rad, crown: cc}
}

// growPine builds a conifer: a straight barked trunk with whorls of
// drooping branches, each lined with paired needle sprays.
func growPine(seed int64, lite bool, dark, light color.RGBA) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 77)
	bk := barkStyle{ridge: color.RGBA{140, 100, 76, 255}, groove: color.RGBA{66, 44, 34, 255},
		moss: color.RGBA{120, 150, 90, 255}, mossH: 1.0, ridges: 9, depth: 0.28}
	H := wb.f(8, 9.5)
	trunk := wb.limb(V3{0, -0.35, 0}, V3{0, 1, 0}, H, 8, 0.015, 0.02)
	wb.tube(trunk, taper(len(trunk), 0.42, 0.04), 9, bk, float64(seed))
	wb.roots(V3{0, -0.35, 0}, 0.38, 5, bk)
	step := 0.8
	if lite {
		step = 1.4
	}
	for y := 1.4; y < H-0.4; y += step {
		frac := (y - 1.4) / (H - 1.8)
		L := (1-frac)*2.9 + 0.45
		count := 6 + wb.r.Intn(2)
		at := V3{0, y, 0}
		for k := 0; k < count; k++ {
			a := wb.f(0, 0.4) + float64(k)/float64(count)*2*math.Pi + y*1.7
			out := V3{math.Cos(a), -0.22 - 0.15*frac*0, math.Sin(a)}.Norm()
			path := wb.limb(at, out, L, 3, 0.06, 0.06)
			wb.tube(path, []float64{0.07 * (1 - frac*0.6), 0.05, 0.03, 0.012}, 3, bk, 0)
			// needle sprays along the branch, paired, pointing forward and down
			sprays := int(L/0.4) + 2
			if lite {
				sprays = int(L/0.85) + 1
			}
			for s := 1; s <= sprays; s++ {
				t := float64(s) / float64(sprays)
				fi := t * float64(len(path)-1)
				i0 := min(len(path)-2, int(fi))
				p := path[i0].Add(path[i0+1].Sub(path[i0]).Mul(fi - float64(i0)))
				d := path[i0+1].Sub(path[i0]).Norm()
				sd := d.Cross(V3{0, 1, 0}).Norm()
				col := mixc(dark, light, 0.2+0.7*t*(1-frac*0.3)+lb.f(-0.12, 0.12))
				nl := (0.5 + 0.35*(1-t)) * (1 + 0.3*(1-frac))
				if lite {
					nl *= 1.4
				}
				for _, sg := range []float64{1, -1} {
					nd := d.Mul(0.55).Add(sd.Mul(sg * 0.7)).Add(V3{0, -0.35, 0}).Norm()
					nrm := V3{0, 1, 0}.Sub(nd.Mul(nd.Y)).Norm()
					lb.leaf(p, nd, nrm, nl, 0.17, 0.25, col, darken(col, 0.7), true)
				}
				// a third spray up the middle on the outer half
				if t > 0.65 {
					nd := d.Add(V3{0, 0.25, 0}).Norm()
					lb.leaf(p, nd, V3{0, 1, 0}, nl*0.8, 0.1, 0.2, lighten(col, 0.08), darken(col, 0.7), true)
				}
			}
		}
	}
	// the crown spike
	for k := 0; k < 6; k++ {
		a := float64(k) / 6 * 2 * math.Pi
		d := V3{math.Cos(a) * 0.5, 1, math.Sin(a) * 0.5}.Norm()
		lb.leaf(V3{0, H - 0.8, 0}, d, perpTo(d), 0.9, 0.12, 0.2, light, darken(light, 0.7), true)
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: H, crown: V3{0, H * 0.5, 0}}
}

// growPalm builds a palm: a curved trunk ringed with leaf scars under a
// crown of pinnate fronds (a rachis lined with drooping leaflets).
func growPalm(seed int64, lite bool) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 31)
	bk := barkStyle{ridge: color.RGBA{172, 140, 104, 255}, groove: color.RGBA{104, 80, 58, 255}, ridges: 5, depth: 0.12}
	H := 7.8
	segs := 18
	var path []V3
	var rad []float64
	for i := 0; i <= segs; i++ {
		t := float64(i) / float64(segs)
		x := 1.4 * t * t
		path = append(path, V3{x, -0.35 + H*t, 0})
		r := 0.42 - 0.13*t
		if i%2 == 1 {
			r *= 1.12 // the scar rings
		}
		rad = append(rad, r)
	}
	rad[0] = 0.62
	wb.tube(path, rad, 10, bk, float64(seed))
	top := path[len(path)-1]
	fronds := 10
	for f := 0; f < fronds; f++ {
		a := float64(f)/float64(fronds)*2*math.Pi + wb.f(-0.2, 0.2)
		rise := wb.f(0.1, 0.6)
		out := V3{math.Cos(a), rise, math.Sin(a)}.Norm()
		L := wb.f(3.6, 4.4)
		var rp []V3
		p, d := top, out
		for k := 0; k < 7; k++ {
			d = d.Add(V3{0, -0.16, 0}).Norm()
			p = p.Add(d.Mul(L / 7))
			rp = append(rp, p)
		}
		rp = append([]V3{top}, rp...)
		wb.tube(rp, taper(len(rp), 0.09, 0.015), 3, bk, 0)
		n := 22
		if lite {
			n = 11
		}
		for k := 1; k <= n; k++ {
			t := float64(k) / float64(n+1)
			fi := t * float64(len(rp)-1)
			i0 := min(len(rp)-2, int(fi))
			q := rp[i0].Add(rp[i0+1].Sub(rp[i0]).Mul(fi - float64(i0)))
			dd := rp[i0+1].Sub(rp[i0]).Norm()
			sd := dd.Cross(V3{0, 1, 0}).Norm()
			ll := (1.25 - 0.8*t) * (0.9 + 0.2*lb.r.Float64())
			if lite {
				ll *= 1.2
			}
			col := mixc(color.RGBA{50, 110, 44, 255}, color.RGBA{150, 196, 82, 255}, 0.3+0.5*t+lb.f(-0.1, 0.1))
			for _, sg := range []float64{1, -1} {
				ld := dd.Mul(0.45).Add(sd.Mul(sg)).Add(V3{0, -0.55, 0}).Norm()
				nrm := V3{0, 1, 0}.Sub(ld.Mul(ld.Y)).Norm()
				w := 0.16
				if lite {
					w = 0.28
				}
				lb.leaf(q, ld, nrm, ll, w, 0.3, col, darken(col, 0.72), true)
			}
		}
	}
	// a cluster of coconuts, each a ridged husk, tucked under the crown
	for k := 0; k < 4; k++ {
		a := float64(k) * 1.7
		c := top.Add(V3{math.Cos(a) * 0.32, -0.35, math.Sin(a) * 0.32})
		var p []V3
		for s := 0; s <= 4; s++ {
			p = append(p, c.Add(V3{0, -0.3 + 0.15*float64(s), 0}))
		}
		wb.tube(p, []float64{0.02, 0.2, 0.24, 0.2, 0.02}, 7, barkStyle{ridge: color.RGBA{150, 116, 60, 255}, groove: color.RGBA{90, 66, 34, 255}, ridges: 3, depth: 0.15}, a)
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: H + 1, crown: top}
}

// growLumen builds the Glimmer Grove tree: a twisting trunk spiralling up
// into upswept limbs that drip strands of glowing teardrop leaves. The
// leaves are white and tintable so each tree glows in the biome's hues.
func growLumen(seed int64, lite bool) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 5)
	lb.tint = true
	lb.emi = 0.55
	bk := barkStyle{ridge: color.RGBA{120, 90, 170, 255}, groove: color.RGBA{36, 22, 60, 255}, ridges: 4, depth: 0.32}
	var trunk []V3
	for i := 0; i <= 10; i++ {
		t := float64(i) / 10
		trunk = append(trunk, V3{0.35 * math.Sin(t*4+float64(seed)), -0.4 + 5.4*t, 0.35 * math.Cos(t*3.3)})
	}
	tr := taper(len(trunk), 0.6, 0.3)
	tr[0] = 0.85
	wb.tube(trunk, tr, 9, bk, float64(seed))
	wb.roots(V3{0, -0.4, 0}, 0.6, 7, bk)
	top := trunk[len(trunk)-1]
	for k := 0; k < 5; k++ {
		a := float64(k)/5*2*math.Pi + float64(seed)
		dir := V3{math.Cos(a) * 0.8, 0.9, math.Sin(a) * 0.8}
		path := wb.limb(top, dir, 2.8, 4, 0.12, 0.1)
		wb.tube(path, taper(len(path), 0.22, 0.05), 6, bk, float64(k))
		// strands hang from points along the limb
		for _, pt := range path[2:] {
			strands := 3
			if lite {
				strands = 2
			}
			for s := 0; s < strands; s++ {
				p := pt.Add(V3{lb.f(-0.4, 0.4), 0, lb.f(-0.4, 0.4)})
				hang := lb.f(1.2, 2.4)
				m := 7
				if lite {
					m = 4
				}
				for q := 0; q < m; q++ {
					y := p.Y - hang*float64(q)/float64(m)
					a2 := lb.f(0, 2*math.Pi)
					d := V3{math.Cos(a2) * 0.4, -1, math.Sin(a2) * 0.4}.Norm()
					nrm := V3{math.Cos(a2 + 1.6), 0.2, math.Sin(a2 + 1.6)}
					v := uint8(205 + lb.r.Intn(50))
					col := color.RGBA{v, v, v, 255}
					lb.leaf(V3{p.X, y, p.Z}, d, nrm, 0.38, 0.22, 0.25, col, darken(col, 0.85), true)
				}
			}
		}
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: 8.5, crown: top, plain: true}
}

// growBush builds a shrub: stems fanning from the root crown, each forking
// into leafy sprays, dotted with flowers in the bloom colour.
func growBush(seed int64, bloom color.RGBA, pal leafPal, lite bool) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 3)
	bk := barkStyle{ridge: color.RGBA{170, 130, 96, 255}, groove: color.RGBA{86, 62, 44, 255}, ridges: 3, depth: 0.15}
	cc := V3{0, 0.55, 0}
	stems := 7
	for k := 0; k < stems; k++ {
		a := float64(k)/float64(stems)*2*math.Pi + wb.f(-0.3, 0.3)
		dir := V3{math.Cos(a) * 0.6, 1, math.Sin(a) * 0.6}
		path := wb.limb(V3{0, -0.6, 0}, dir, wb.f(1.2, 1.6), 3, 0.2, 0.0)
		wb.tube(path, taper(len(path), 0.07, 0.02), 4, bk, float64(k))
		for i := 1; i < len(path); i++ {
			n := 7
			if lite {
				n = 4
			}
			L, W := 0.32, 0.19
			if lite {
				L, W = 0.46, 0.27
			}
			lb.cluster(path[i], cc, n, 0.42, L, W, pal, -0.6, 1.2)
		}
		if bloom.A > 0 {
			for f := 0; f < 3; f++ {
				p := path[len(path)-1].Add(lb.inBall().Mul(0.45))
				nrm := p.Sub(cc).Norm().Add(V3{0, 0.8, 0}).Norm()
				lb.flower(p, nrm, 0.13, bloom, color.RGBA{255, 214, 90, 255}, 5)
			}
		}
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: 1.2, crown: cc}
}

// growTopiary builds a clipped garden form out of leaves: a dense outer
// shell of overlapping leaves over a darker inner shell, studded with
// blossoms. shape 0 = ball, 1 = cone spire, 2 = rolled hedge tile (unit
// length along z, rounded top, 1.2 wide, 1 tall).
func growTopiary(seed int64, shape int, pal leafPal, bloom color.RGBA, lite bool) *floraSet {
	lb := newFloraB(seed)
	lb.emi = topiaryGlow
	// surface sample: point and outward normal for parameters u, v in [0,1)
	surf := func(u, v float64) (V3, V3) {
		switch shape {
		case 1: // cone: radius 1 at y=0 to a point at y=3.2
			a := u * 2 * math.Pi
			y := v * 3.2
			r := 1 - v
			return V3{math.Cos(a) * r, y, math.Sin(a) * r}, V3{math.Cos(a), 0.31, math.Sin(a)}.Norm()
		case 2: // hedge tile: front/back walls and a rounded top
			z := u - 0.5
			if v < 0.3 {
				t := v / 0.3
				return V3{0.6, 0.62 * t, z}, V3{1, 0, 0}
			} else if v < 0.6 {
				t := (v - 0.3) / 0.3
				return V3{-0.6, 0.62 * t, z}, V3{-1, 0, 0}
			}
			a := (v - 0.6) / 0.4 * math.Pi
			return V3{0.6 * math.Cos(a), 0.62 + 0.38*math.Sin(a), z}, V3{math.Cos(a), math.Sin(a), 0}
		default: // ball
			th := math.Acos(1 - 2*v)
			a := u * 2 * math.Pi
			n := V3{math.Sin(th) * math.Cos(a), math.Cos(th), math.Sin(th) * math.Sin(a)}
			return n, n
		}
	}
	area := []float64{12.6, 10.6, 3.1}[shape]
	dens := []float64{0.95, 1.0, 1.3}[shape]
	L, W := 0.72, 0.45
	if shape == 2 {
		L, W = 0.5, 0.31
	}
	if lite {
		L, W = L*1.45, W*1.45
	}
	n := int(area * dens / (L * W * 0.5))
	inner := darken(pal.dark, 0.7)
	for pass := 0; pass < 2; pass++ {
		cnt := n
		sc := 1.0
		if pass == 0 {
			cnt = n / 2
			sc = 0.84
		}
		for k := 0; k < cnt; k++ {
			// golden-ratio spread so the shell covers evenly
			u := math.Mod(float64(k)*0.618034+lb.f(0, 0.04), 1)
			v := (float64(k) + 0.5) / float64(cnt)
			p, nrm := surf(u, v)
			if shape == 2 {
				p = V3{p.X * sc, p.Y * (0.9 + 0.1*sc), p.Z}
			} else {
				p = p.Mul(sc)
			}
			p = p.Add(lb.unit().Mul(0.04))
			nn := nrm.Add(lb.unit().Mul(0.45)).Norm()
			dir := lb.unit()
			dir = dir.Sub(nn.Mul(dir.Dot(nn)))
			if dir.Len() < 1e-3 {
				dir = perpTo(nn)
			}
			dir = dir.Norm()
			var col color.RGBA
			if pass == 0 {
				col = mixc(inner, pal.dark, lb.r.Float64())
			} else {
				col = lb.leafCol(pal, 0.4+0.5*nrm.Y)
			}
			s := lb.f(0.8, 1.15)
			lb.leaf(p.Sub(dir.Mul(L*0.4*s)), dir, nn, L*s, W*s, 0.15, col, pal.under, false)
		}
	}
	if bloom.A > 0 {
		nb := int(area * 0.85)
		for k := 0; k < nb; k++ {
			p, nrm := surf(lb.r.Float64(), lb.f(0.1, 1))
			if shape == 2 && nrm.Y < 0.2 && lb.r.Float64() < 0.6 {
				continue // most blooms on the top roll
			}
			p = p.Add(nrm.Mul(0.05))
			lb.flowerX(p, nrm, 0.17, bloom, color.RGBA{255, 230, 140, 255}, 5, false)
		}
	}
	return &floraSet{wood: &Mesh{}, leaf: lb.m, top: 1, crown: V3{0, 0.5, 0}}
}

// ---------------------------------------------------------------- draw

// drawFlora draws a plant at base pos. wood is drawn with the bark
// texture; the foliage sways about the base by a fraction of a degree.
func drawFlora(B *Batch, s *floraSet, pos V3, scale V3, ry float64, tint color.RGBA, t, sway float64) {
	// Static on purpose: a plant drawn at the same placement every frame
	// stays in the renderer's baked-mesh cache, which is many times cheaper
	// than re-projecting thousands of leaves (sway and the textured bark
	// path both defeated it). The bark detail is in the geometry: ridged,
	// grooved and mossy faces.
	_, _ = t, sway
	if s.impOf != nil {
		if im := impostorFor(s.impOf); im != nil && B.cam != nil {
			drawImpostor(B, im, pos, scale, ry, tint)
			return
		}
		s = s.impOf // not photographed yet: draw it in full meanwhile
	}
	if B.cam != nil && hiddenByTerrain(B, pos.Add(V3{0, s.top * scale.Y, 0})) {
		return // behind a hill
	}
	if B.cam != nil {
		// whole-plant culling: skip it when its bounds are off screen or it
		// would be a speck a few pixels across
		sc := math.Max(scale.X, math.Max(scale.Y, scale.Z))
		c := pos.Add(V3{s.crown.X * scale.X, s.crown.Y * scale.Y, s.crown.Z * scale.Z})
		rad := math.Max(1, s.top) * sc * 0.75
		sx, sy, d := B.cam.Project(c)
		if d > 0.5 {
			rp := rad * B.cam.f / d
			if rp < 2.5 || sx+rp < 0 || sx-rp > B.cam.W || sy+rp < 0 || sy-rp > B.cam.H {
				return
			}
		}
	}
	rot := rotXYZ(0, ry, 0)
	if len(s.wood.Tris) > 0 {
		B.Draw(s.wood, pos, scale, rot, color.RGBA{}, 0)
	}
	if len(s.cards) > 0 && tint.A == 0 {
		if tex := leafCardTex(); tex != 0 {
			drawCards(B, s.cards, tex, pos, scale, rot)
			return
		}
	}
	if len(s.leaf.Tris) > 0 {
		B.Draw(s.leaf, pos, scale, rot, tint, 0)
	}
}

func treeVariant(name string, seed int64, far bool, build func(seed int64, lite bool) *floraSet) *floraSet {
	return treeVariantLOD(name, seed, map[bool]int{false: 0, true: 1}[far], build)
}

// treeVariantLOD picks a cached variant by level of detail: 0 full, 1 light,
// 2 light with single-sided leaves, 3 a picture card (impostor).
func treeVariantLOD(name string, seed int64, lod int, build func(seed int64, lite bool) *floraSet) *floraSet {
	if lod == 3 {
		lite := treeVariantLOD(name, seed, 1, build)
		return floraGet(name+itoa(int(seed%4))+"I", func() *floraSet {
			return &floraSet{impOf: lite, top: lite.top, crown: lite.crown}
		})
	}
	key := name + itoa(int(seed%4)) + []string{"", "L", "T"}[lod]
	return floraGet(key, func() *floraSet {
		s := build(seed%4+11, lod > 0)
		if lod >= 1 && floraCardTrees[name] {
			// mid-distance broadleaf canopies as leaf cards
			s = &floraSet{wood: s.wood, leaf: s.leaf, top: s.top, crown: s.crown, plain: s.plain, cards: cardsFromLeaves(s.leaf, s.crown)}
		}
		if lod == 2 {
			s = &floraSet{wood: s.wood, leaf: stripBacks(s.leaf), top: s.top, crown: s.crown, plain: s.plain, cards: s.cards}
		}
		return s
	})
}

// floraCardTrees are the species whose light variants use leaf cards.
var floraCardTrees = map[string]bool{"oak": true, "blossom": true}

// floraLOD is the level of detail for a plant at p.
func floraLOD(B *Batch, p V3) int {
	switch {
	case floraPx(B, p, 6) < 70:
		return 3
	case floraTiny(B, p):
		return 2
	case floraFar(B, p):
		return 1
	}
	return 0
}

// topiaryGlow lifts the Dreamscape garden's leaves a little, so the beds
// keep the world's soft, lit-from-within look.
var topiaryGlow float32 = 0

var (
	palDream = leafPal{dark: color.RGBA{22, 78, 46, 255}, light: color.RGBA{84, 168, 92, 255},
		accent: color.RGBA{230, 150, 200, 255}, under: color.RGBA{40, 90, 60, 255}, accentP: 0.04}
	palBlossom = leafPal{dark: color.RGBA{226, 128, 176, 255}, light: color.RGBA{255, 226, 240, 255},
		accent: color.RGBA{255, 255, 255, 255}, under: color.RGBA{196, 110, 158, 255}, accentP: 0.12}
	barkDream = barkStyle{ridge: color.RGBA{176, 128, 122, 255}, groove: color.RGBA{92, 60, 62, 255},
		moss: color.RGBA{180, 240, 190, 255}, mossH: 1.2, ridges: 6, depth: 0.2}
)

// growStem is a clipped topiary's single barked stem, height h.
func growStem(seed int64, h float64) *floraSet {
	wb := newFloraB(seed)
	p := wb.limb(V3{0, -0.1, 0}, V3{0, 1, 0}, h, 4, 0.04, 0)
	wb.tube(p, taper(len(p), 0.16, 0.1), 7, barkDream, float64(seed))
	wb.roots(V3{0, -0.1, 0}, 0.16, 4, barkDream)
	return &floraSet{wood: wb.m, leaf: &Mesh{}, top: h}
}

// mossTuft scatters short, splayed blades over an oval of w x d at c,
// facing up: moss and low grass on stone.
func (b *floraB) mossTuft(c V3, w, d float64, dark, light color.RGBA, n int) {
	for k := 0; k < n; k++ {
		p := c.Add(V3{b.f(-1, 1) * w, 0, b.f(-1, 1) * d})
		a := b.f(0, 2*math.Pi)
		dir := V3{math.Cos(a), b.f(0.5, 1.4), math.Sin(a)}.Norm()
		col := mixc(dark, light, b.r.Float64())
		b.leaf(p, dir, perpTo(dir), b.f(0.14, 0.28), b.f(0.05, 0.08), 0.2, col, darken(col, 0.75), true)
	}
}

// growFlowerBed: a low bed of leaf rosettes with stemmed five-petal
// flowers in mixed colours.
func growFlowerBed(seed int64) *Mesh {
	b := newFloraB(seed)
	pal := []color.RGBA{{250, 236, 200, 255}, {248, 176, 70, 255}, {236, 108, 112, 255}, {196, 150, 240, 255}, {255, 214, 90, 255}}
	leafP := leafPal{dark: color.RGBA{50, 100, 40, 255}, light: color.RGBA{130, 186, 80, 255}, under: color.RGBA{80, 110, 60, 255}}
	for k := 0; k < 60; k++ {
		a := b.f(0, 2*math.Pi)
		d := math.Sqrt(b.r.Float64())
		p := V3{math.Cos(a) * d * 1.5, 0.08 + 0.18*(1-d*d), math.Sin(a) * d * 1.0}
		dir := V3{math.Cos(a + b.f(-1, 1)), b.f(0.2, 0.7), math.Sin(a + b.f(-1, 1))}.Norm()
		col := b.leafCol(leafP, b.r.Float64())
		b.leaf(p, dir, V3{0, 1, 0}.Sub(dir.Mul(dir.Y)).Norm(), 0.3, 0.16, 0.2, col, leafP.under, true)
	}
	bk := barkStyle{ridge: color.RGBA{90, 140, 60, 255}, groove: color.RGBA{60, 100, 40, 255}, ridges: 2, depth: 0.05}
	for k := 0; k < 14; k++ {
		a := b.f(0, 2*math.Pi)
		d := math.Sqrt(b.r.Float64()) * 0.9
		base := V3{math.Cos(a) * d * 1.4, 0.1, math.Sin(a) * d * 0.9}
		h := b.f(0.35, 0.6)
		top := base.Add(V3{b.f(-0.08, 0.08), h, b.f(-0.08, 0.08)})
		b.tube([]V3{base, base.Add(top.Sub(base).Mul(0.5)), top}, []float64{0.022, 0.018, 0.014}, 3, bk, 0)
		b.flower(top, V3{b.f(-0.3, 0.3), 1, b.f(-0.5, 0)}.Norm(), b.f(0.12, 0.17), pal[b.r.Intn(len(pal))], color.RGBA{255, 210, 80, 255}, 5)
	}
	return b.m
}

// ---------------------------------------------------------------- stone

func hash3(x, y, z int, seed uint32) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(z)*2147483647 + seed*144665
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0xffffff)
}

func noise3(p V3, seed uint32) float64 {
	x0, y0, z0 := math.Floor(p.X), math.Floor(p.Y), math.Floor(p.Z)
	fx, fy, fz := p.X-x0, p.Y-y0, p.Z-z0
	sm := func(t float64) float64 { return t * t * (3 - 2*t) }
	ux, uy, uz := sm(fx), sm(fy), sm(fz)
	ix, iy, iz := int(x0), int(y0), int(z0)
	l := func(a, b, t float64) float64 { return a + (b-a)*t }
	c := func(dx, dy, dz int) float64 { return hash3(ix+dx, iy+dy, iz+dz, seed) }
	return l(l(l(c(0, 0, 0), c(1, 0, 0), ux), l(c(0, 1, 0), c(1, 1, 0), ux), uy),
		l(l(c(0, 0, 1), c(1, 0, 1), ux), l(c(0, 1, 1), c(1, 1, 1), ux), uy), uz)
}

func fbm3(p V3, oct int, seed uint32) float64 {
	s, a, n := 0.0, 0.5, 0.0
	for o := 0; o < oct; o++ {
		s += a * noise3(p, seed+uint32(o)*31)
		n += a
		a *= 0.5
		p = p.Mul(2.03)
	}
	return s / n
}

// growRock builds a weathered boulder about 2 across: a subdivided ball
// pushed out by layered noise, cut by a few chisel planes into facets,
// flattened where it sits, with darker crevices, paler weathered tops and
// (optionally) moss on its upper faces. Faces are tintable (the caller's
// stone colour); moss keeps its own green.
func growRock(seed int64, moss bool) *Mesh {
	base := Icosphere(2, 0, seed, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255})
	r := rand.New(rand.NewSource(seed))
	sd := uint32(seed*7 + 3)
	type plane struct {
		n V3
		d float64
	}
	var cuts []plane
	for k := 0; k < 9; k++ {
		n := V3{r.Float64()*2 - 1, r.Float64()*1.4 - 0.3, r.Float64()*2 - 1}.Norm()
		cuts = append(cuts, plane{n, 0.66 + 0.2*r.Float64()})
	}
	disp := map[V3]V3{}
	shape := func(v V3) V3 {
		if p, ok := disp[v]; ok {
			return p
		}
		d := v.Norm()
		rad := 1 + 0.4*(fbm3(d.Mul(1.4), 3, sd)-0.5) + 0.16*(fbm3(d.Mul(5), 2, sd+9)-0.5)
		p := d.Mul(rad)
		for _, c := range cuts {
			if e := p.Dot(c.n) - c.d; e > 0 {
				p = p.Sub(c.n.Mul(e * 0.97))
			}
		}
		if p.Y < -0.35 {
			p.Y = -0.35 + (p.Y+0.35)*0.25
		}
		p.Y *= 0.78
		disp[v] = p
		return p
	}
	m := &Mesh{}
	for _, t := range base.Tris {
		q := [3]V3{shape(t.P[0]), shape(t.P[1]), shape(t.P[2])}
		n := q[1].Sub(q[0]).Cross(q[2].Sub(q[0]))
		if n.Len() < 1e-9 {
			continue
		}
		n = n.Norm()
		c := q[0].Add(q[1]).Add(q[2]).Mul(1.0 / 3)
		// Icosphere winds every face inward; turn them all out
		q[1], q[2] = q[2], q[1]
		n = n.Mul(-1)
		// crevices darker, weathered tops paler, a little speckle
		cav := fbm3(c.Mul(3.2), 2, sd+17)
		v := 0.45 + 0.5*cav + 0.15*math.Max(0, n.Y) + 0.1*(r.Float64()-0.5)
		if band := math.Sin(c.Y*9 + cav*4); band > 0.85 {
			v *= 0.82 // a darker stratum line
		}
		v = math.Max(0.35, math.Min(1, v))
		g := uint8(255 * v)
		col := color.RGBA{g, uint8(float64(g) * 0.98), uint8(float64(g) * 0.95), 255}
		fixed := false
		if moss && n.Y > 0.55 && fbm3(c.Mul(2.2), 2, sd+40) > 0.45 {
			k := math.Min(1, (n.Y-0.55)*2.5)
			col = mixc(color.RGBA{96, 128, 58, 255}, color.RGBA{150, 182, 84, 255}, r.Float64()*k)
			fixed = true
		}
		m.Tris = append(m.Tris, Tri{P: q, Col: col, Fixed: fixed})
	}
	return m
}

// growStalagmite builds a dripstone column of unit radius and height:
// flowstone ridges running down a knobbly taper, with drip rings, darker
// and wet-looking low down. Faces are tintable.
func growStalagmite(seed int64) *Mesh {
	b := newFloraB(seed)
	b.tint = true
	bk := barkStyle{ridge: color.RGBA{240, 240, 248, 255}, groove: color.RGBA{120, 124, 150, 255}, ridges: 6, depth: 0.3}
	var path []V3
	var rad []float64
	segs := 14
	for i := 0; i <= segs; i++ {
		t := float64(i) / float64(segs)
		path = append(path, V3{0.12 * math.Sin(t*5+float64(seed)), t, 0.1 * math.Cos(t*4)})
		rr := math.Pow(1-t, 1.4)*0.95 + 0.02
		rr *= 1 + 0.12*math.Sin(t*31+float64(seed)) // drip rings
		rad = append(rad, rr)
	}
	b.tube(path, rad, 9, bk, float64(seed))
	return b.m
}

// growHedgerow is a long field hedge tile (length 4 along z, 1.4 wide,
// 1.6 tall) for the far hills: woody stems showing at the foot, a dense
// shell of leaves above, a scatter of white hawthorn blossom.
func growHedgerow(seed int64) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 1)
	bk := barkStyle{ridge: color.RGBA{170, 136, 104, 255}, groove: color.RGBA{80, 60, 44, 255}, ridges: 3, depth: 0.15}
	for k := 0; k < 9; k++ {
		z := -2 + 4*(float64(k)+0.5)/9
		p := wb.limb(V3{wb.f(-0.3, 0.3), -0.1, z}, V3{wb.f(-0.3, 0.3), 1, wb.f(-0.3, 0.3)}, 0.9, 2, 0.2, 0)
		wb.tube(p, []float64{0.07, 0.05, 0.03}, 4, bk, float64(k))
	}
	pal := leafPal{dark: color.RGBA{38, 80, 34, 255}, light: color.RGBA{120, 170, 72, 255}, under: color.RGBA{70, 100, 50, 255}, accent: color.RGBA{190, 200, 90, 255}, accentP: 0.05}
	n := 100
	for k := 0; k < n; k++ {
		u := lb.r.Float64()
		v := lb.r.Float64()
		var p, nrm V3
		z := -2 + 4*u
		switch {
		case v < 0.3:
			p, nrm = V3{0.7, 0.3 + 0.9*v/0.3, z}, V3{1, 0, 0}
		case v < 0.6:
			p, nrm = V3{-0.7, 0.3 + 0.9*(v-0.3)/0.3, z}, V3{-1, 0, 0}
		default:
			a := (v - 0.6) / 0.4 * math.Pi
			p, nrm = V3{0.7 * math.Cos(a), 1.2 + 0.4*math.Sin(a), z}, V3{math.Cos(a), math.Sin(a), 0}
		}
		p = p.Add(lb.inBall().Mul(0.12))
		nn := nrm.Add(lb.unit().Mul(0.5)).Norm()
		dir := lb.unit()
		dir = dir.Sub(nn.Mul(dir.Dot(nn)))
		if dir.Len() < 1e-3 {
			dir = perpTo(nn)
		}
		dir = dir.Norm()
		col := lb.leafCol(pal, 0.3+0.6*nrm.Y)
		lb.leaf(p.Sub(dir.Mul(0.35)), dir, nn, 0.95, 0.57, 0.15, col, pal.under, false)
	}
	for k := 0; k < 10; k++ {
		a := lb.f(0, math.Pi)
		p := V3{0.72 * math.Cos(a), 1.22 + 0.42*math.Sin(a), lb.f(-1.9, 1.9)}
		lb.flowerX(p, V3{math.Cos(a), math.Sin(a), 0}, 0.14, color.RGBA{250, 246, 236, 255}, color.RGBA{230, 200, 120, 255}, 5, false)
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: 1.6, crown: V3{0, 1, 0}}
}

// growSunflowerStem is a sunflower's hairy stalk (to y=3.3 from y=0) with
// broad heart-shaped leaves alternating up it; the head is drawn on top.
func growSunflowerStem(seed int64) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 2)
	bk := barkStyle{ridge: color.RGBA{120, 170, 80, 255}, groove: color.RGBA{70, 110, 46, 255}, ridges: 4, depth: 0.1}
	p := wb.limb(V3{0, 0, 0}, V3{0, 1, 0}, 3.3, 5, 0.03, 0.02)
	wb.tube(p, taper(len(p), 0.08, 0.05), 5, bk, float64(seed))
	for k := 0; k < 5; k++ {
		y := 0.5 + float64(k)*0.5
		a := float64(k)*2.4 + float64(seed)
		d := V3{math.Cos(a), 0.15, math.Sin(a)}.Norm()
		col := mixc(color.RGBA{50, 100, 36, 255}, color.RGBA{110, 160, 60, 255}, lb.r.Float64())
		base := V3{0, y, 0}
		// a stalk then the blade, drooping
		lb.leaf(base, d, V3{0, 1, 0}.Sub(d.Mul(d.Y)).Norm(), 0.25, 0.05, 0, darken(col, 0.9), darken(col, 0.7), true)
		bd := d.Add(V3{0, -0.35, 0}).Norm()
		lb.leaf(base.Add(d.Mul(0.22)), bd, V3{0, 1, 0}.Sub(bd.Mul(bd.Y)).Norm(), 0.95-float64(k)*0.07, 0.66-float64(k)*0.05, 0.12, col, darken(col, 0.72), true)
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: 3.3}
}

// growPumpkinVine is a ground vine winding out from a pumpkin, with big
// lobed leaves on stalks and a few curling tendrils.
func growPumpkinVine(seed int64) *floraSet {
	wb := newFloraB(seed)
	lb := newFloraB(seed + 4)
	bk := barkStyle{ridge: color.RGBA{120, 160, 70, 255}, groove: color.RGBA{70, 100, 40, 255}, ridges: 3, depth: 0.1}
	for v := 0; v < 2; v++ {
		a := wb.f(0, 2*math.Pi)
		var p []V3
		pos := V3{0, 0.05, 0}
		for k := 0; k < 7; k++ {
			a += wb.f(-0.6, 0.6)
			pos = pos.Add(V3{math.Cos(a) * 0.45, 0, math.Sin(a) * 0.45})
			p = append(p, V3{pos.X, 0.05 + 0.04*math.Sin(float64(k)), pos.Z})
		}
		p = append([]V3{{0, 0.25, 0}}, p...)
		wb.tube(p, taper(len(p), 0.05, 0.025), 4, bk, 0)
		for k := 2; k < len(p); k += 2 {
			at := p[k]
			up := V3{wb.f(-0.3, 0.3), 1, wb.f(-0.3, 0.3)}.Norm()
			top := at.Add(up.Mul(0.45))
			wb.tube([]V3{at, at.Add(up.Mul(0.25)), top}, []float64{0.02, 0.018, 0.015}, 3, bk, 0)
			// a lobed leaf: three overlapping blades fanned from the stalk tip
			col := mixc(color.RGBA{46, 96, 34, 255}, color.RGBA{108, 156, 60, 255}, lb.r.Float64())
			fa := lb.f(0, 2*math.Pi)
			for l := -1; l <= 1; l++ {
				d := V3{math.Cos(fa + float64(l)*0.7), -0.1, math.Sin(fa + float64(l)*0.7)}.Norm()
				lb.leaf(top, d, V3{0, 1, 0}.Sub(d.Mul(d.Y)).Norm(), 0.5-0.08*math.Abs(float64(l)), 0.42, 0.15, col, darken(col, 0.7), true)
			}
		}
	}
	return &floraSet{wood: wb.m, leaf: lb.m, top: 0.8}
}

// growReed is a clump of tall, arching reed blades of unit height with a
// seed head on the tallest.
func growReed(seed int64) *Mesh {
	b := newFloraB(seed)
	for k := 0; k < 7; k++ {
		a := b.f(0, 2*math.Pi)
		h := b.f(0.55, 1)
		d := V3{math.Cos(a) * 0.25, 1, math.Sin(a) * 0.25}.Norm()
		col := mixc(color.RGBA{50, 140, 100, 255}, color.RGBA{140, 230, 170, 255}, b.r.Float64())
		// two segments, the upper bending further out
		p0 := V3{b.f(-0.05, 0.05), 0, b.f(-0.05, 0.05)}
		b.leaf(p0, d, perpTo(d), h*0.55, 0.06, 0.1, col, darken(col, 0.75), true)
		d2 := d.Add(V3{math.Cos(a) * 0.5, 0, math.Sin(a) * 0.5}).Norm()
		b.leaf(p0.Add(d.Mul(h*0.5)), d2, perpTo(d2), h*0.5, 0.05, 0.1, lighten(col, 0.1), darken(col, 0.75), true)
	}
	return b.m
}

// growFern is a low fern: arching fronds from a crown, each a rachis lined
// with paired leaflets that shrink toward the tip. Tintable, so a cave
// fern can glow in the biome's hue.
func growFern(seed int64, glow bool) *Mesh {
	b := newFloraB(seed)
	b.tint = glow
	if glow {
		b.emi = 0.45
	}
	fronds := 9
	for f := 0; f < fronds; f++ {
		a := float64(f)/float64(fronds)*2*math.Pi + b.f(-0.2, 0.2)
		out := V3{math.Cos(a), b.f(1.4, 2.2), math.Sin(a)}.Norm()
		L := b.f(1.0, 1.4)
		var rp []V3
		p, d := V3{}, out
		for k := 0; k < 9; k++ {
			d = d.Add(V3{0, -0.09, 0}).Norm()
			p = p.Add(d.Mul(L / 9))
			rp = append(rp, p)
		}
		for k := 0; k < len(rp)-1; k++ {
			q := rp[k]
			dd := rp[k+1].Sub(q).Norm()
			sd := dd.Cross(V3{0, 1, 0})
			if sd.Len() < 1e-3 {
				continue
			}
			sd = sd.Norm()
			t := float64(k) / float64(len(rp)-1)
			var col color.RGBA
			if glow {
				v := uint8(170 + 85*t)
				col = color.RGBA{v, v, v, 255}
			} else {
				col = mixc(color.RGBA{40, 90, 40, 255}, color.RGBA{120, 176, 70, 255}, 0.3+0.5*t+b.f(-0.1, 0.1))
			}
			for _, sg := range []float64{1, -1} {
				ld := dd.Mul(0.5).Add(sd.Mul(sg)).Add(V3{0, -0.1, 0}).Norm()
				nrm := V3{0, 1, 0}.Sub(ld.Mul(ld.Y)).Norm()
				b.leaf(q, ld, nrm, 0.32*(1-t*0.6), 0.11, 0.2, col, darken(col, 0.7), true)
			}
		}
		b.leaf(rp[len(rp)-2], rp[len(rp)-1].Sub(rp[len(rp)-2]).Norm(), V3{0, 1, 0}, 0.2, 0.08, 0.1, lighten(b.m.Tris[len(b.m.Tris)-1].Col, 0.1), darken(b.m.Tris[len(b.m.Tris)-1].Col, 0.7), true)
	}
	return b.m
}

// growHangingRoots: roots dangling from under a floating island, each a
// thin wandering tube with a few side rootlets.
func growHangingRoots(seed int64) *Mesh {
	b := newFloraB(seed)
	bk := barkStyle{ridge: color.RGBA{160, 130, 104, 255}, groove: color.RGBA{80, 62, 48, 255}, ridges: 3, depth: 0.2}
	for k := 0; k < 9; k++ {
		a := b.f(0, 2*math.Pi)
		r := b.f(0.1, 0.65)
		start := V3{math.Cos(a) * r, -0.1, math.Sin(a) * r * 0.8}
		p := b.limb(start, V3{b.f(-0.3, 0.3), -1, b.f(-0.3, 0.3)}, b.f(0.6, 1.4), 4, 0.25, -0.05)
		b.tube(p, taper(len(p), 0.05, 0.008), 4, bk, float64(k))
		if len(p) > 3 {
			q := b.limb(p[2], V3{b.f(-1, 1), -0.6, b.f(-1, 1)}, 0.35, 2, 0.3, -0.05)
			b.tube(q, []float64{0.022, 0.014, 0.005}, 3, bk, 0)
		}
	}
	return b.m
}

var palNight = leafPal{dark: color.RGBA{18, 40, 60, 255}, light: color.RGBA{60, 120, 130, 255},
	accent: color.RGBA{120, 200, 220, 255}, under: color.RGBA{30, 50, 70, 255}, accentP: 0.06}

var palDry = leafPal{dark: color.RGBA{96, 104, 52, 255}, light: color.RGBA{176, 168, 98, 255},
	accent: color.RGBA{200, 150, 80, 255}, under: color.RGBA{110, 100, 70, 255}, accentP: 0.12}

// growIsland builds a floating island about 2 across (top at y=0): a
// grassy, tufted top over a rocky underside that tapers to a craggy point,
// with earth showing at the rim. top/under tint the grass and the stone.
func growIsland(seed int64, top, under color.RGBA) *Mesh {
	base := Icosphere(2, 0, seed, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 255, 255})
	r := rand.New(rand.NewSource(seed))
	sd := uint32(seed*13 + 5)
	disp := map[V3]V3{}
	shape := func(v V3) V3 {
		if p, ok := disp[v]; ok {
			return p
		}
		d := v.Norm()
		rad := 1 + 0.3*(fbm3(d.Mul(1.6), 3, sd)-0.5) + 0.12*(fbm3(d.Mul(5), 2, sd+9)-0.5)
		p := d.Mul(rad)
		if p.Y > 0 {
			p.Y = 0.08 * p.Y // a nearly flat, gently domed top
		} else {
			k := -p.Y
			p.Y = -k * 2.2
			sx := 1 - 0.55*k*k
			p.X *= sx
			p.Z *= sx
		}
		disp[v] = p
		return p
	}
	m := &Mesh{}
	for _, t := range base.Tris {
		q := [3]V3{shape(t.P[0]), shape(t.P[1]), shape(t.P[2])}
		n := q[1].Sub(q[0]).Cross(q[2].Sub(q[0]))
		if n.Len() < 1e-9 {
			continue
		}
		n = n.Norm()
		c := q[0].Add(q[1]).Add(q[2]).Mul(1.0 / 3)
		q[1], q[2] = q[2], q[1] // Icosphere winds inward
		n = n.Mul(-1)
		var col color.RGBA
		switch {
		case n.Y > 0.6:
			col = mixc(darken(top, 0.85), lighten(top, 0.12), r.Float64())
		case c.Y > -0.3:
			col = mixc(color.RGBA{110, 84, 60, 255}, color.RGBA{140, 110, 80, 255}, r.Float64()) // earth at the rim
		default:
			cav := fbm3(c.Mul(3), 2, sd+3)
			col = mixc(darken(under, 0.6), lighten(under, 0.1), cav+0.15*(r.Float64()-0.5))
		}
		m.Tris = append(m.Tris, Tri{P: q, Col: col, Fixed: true})
	}
	fb := newFloraB(seed + 1)
	for k := 0; k < 6; k++ {
		a := fb.f(0, 2*math.Pi)
		d := fb.f(0, 0.75)
		fb.mossTuft(V3{math.Cos(a) * d, 0.06, math.Sin(a) * d}, 0.25, 0.25, darken(top, 0.7), lighten(top, 0.2), 14)
	}
	m.Tris = append(m.Tris, fb.m.Tris...)
	return m
}

// bevelBox is a unit box (-1..1) with chamfered edges and corners: a candy
// tile with no texture or symbol. Faces are white and tintable; the
// chamfers are a shade lighter on top so the edges catch the light.
func bevelBox(b float64) *Mesh {
	return cached("bevelBox"+itoa(int(b*100)), func() *Mesh {
		fb := newFloraB(1)
		fb.tint = true
		i := 1 - b
		col := func(n V3) color.RGBA {
			v := 236.0
			if n.Y > 0.3 {
				v = 255
			} else if n.Y < -0.3 {
				v = 200
			}
			if math.Abs(n.X)+math.Abs(n.Y)+math.Abs(n.Z) > 1.2 { // a chamfer
				v = math.Min(255, v*1.06)
			}
			return color.RGBA{uint8(v), uint8(v), uint8(v), 255}
		}
		add := func(p ...V3) {
			c := V3{}
			for _, q := range p {
				c = c.Add(q)
			}
			n := c.Norm()
			fb.face(p[0], p[1], p[2], n, col(n))
			if len(p) == 4 {
				fb.face(p[0], p[2], p[3], n, col(n))
			}
		}
		// axis helpers: build a point from (axis value, u, v) per axis order
		pt := func(ax int, a, u, v float64) V3 {
			switch ax {
			case 0:
				return V3{a, u, v}
			case 1:
				return V3{v, a, u}
			}
			return V3{u, v, a}
		}
		for ax := 0; ax < 3; ax++ {
			for _, s := range []float64{-1, 1} {
				add(pt(ax, s, -i, -i), pt(ax, s, i, -i), pt(ax, s, i, i), pt(ax, s, -i, i))
			}
		}
		// edge chamfers: between faces of two axes
		for _, sx := range []float64{-1, 1} {
			for _, sy := range []float64{-1, 1} {
				add(V3{sx, sy * i, -i}, V3{sx * i, sy, -i}, V3{sx * i, sy, i}, V3{sx, sy * i, i}) // x-y edges
				add(V3{sx, -i, sy * i}, V3{sx * i, -i, sy}, V3{sx * i, i, sy}, V3{sx, i, sy * i}) // x-z edges
				add(V3{-i, sx, sy * i}, V3{-i, sx * i, sy}, V3{i, sx * i, sy}, V3{i, sx, sy * i}) // y-z edges
				for _, sz := range []float64{-1, 1} {
					_ = sz
				}
			}
		}
		for _, sx := range []float64{-1, 1} {
			for _, sy := range []float64{-1, 1} {
				for _, sz := range []float64{-1, 1} {
					add(V3{sx, sy * i, sz * i}, V3{sx * i, sy, sz * i}, V3{sx * i, sy * i, sz})
				}
			}
		}
		return fb.m
	})
}

// candyMesh is a candy-striped cone (cone=true, radius 1 at y=0 to a point
// at y=1) or cylinder (radius 1, height 1, capped), the spiral stripes cut
// into its faces' colours rather than a texture, so it draws in the same
// untextured batch as the scenery around it.
func candyMesh(cone bool) *Mesh {
	key := "candyCyl"
	if cone {
		key = "candyCone"
	}
	return cached(key, func() *Mesh {
		b := newFloraB(3)
		pink, white := color.RGBA{255, 104, 180, 255}, color.RGBA{255, 244, 250, 255}
		const sides, rings, stripes = 16, 16, 4
		pt := func(i, j int) V3 {
			y := float64(i) / rings
			r := 1.0
			if cone {
				r = 1 - y
			}
			a := float64(j) / sides * 2 * math.Pi
			return V3{math.Cos(a) * r, y, math.Sin(a) * r}
		}
		for i := 0; i < rings; i++ {
			for j := 0; j < sides; j++ {
				p00, p01, p11, p10 := pt(i, j), pt(i, j+1), pt(i+1, j+1), pt(i+1, j)
				a := (float64(j) + 0.5) / sides
				y := (float64(i) + 0.5) / rings
				col := white
				if int(math.Floor((a+y*0.9)*stripes*2))%2 == 0 {
					col = pink
				}
				out := p00.Add(p11).Mul(0.5)
				out = V3{out.X, 0.35, out.Z}
				if cone && i == rings-1 {
					b.face(p00, p01, p11, out, col)
					continue
				}
				b.face(p00, p10, p11, out, col)
				b.face(p00, p11, p01, out, darken(col, 0.97))
			}
		}
		if !cone {
			for j := 0; j < sides; j++ {
				b.face(V3{0, 1, 0}, pt(rings, j), pt(rings, j+1), V3{0, 1, 0}, white)
			}
		}
		return b.m
	})
}

// ---------------------------------------------------------------- merged statics
//
// Groups of plants that never move in a chamber (the Dreamscape topiary
// beds, the meadow hedgerows) are merged into one mesh the first time the
// chamber draws them, so each frame makes one draw instead of dozens. The
// detail level each plant gets is chosen when the group is built.

type mergedGroup struct {
	m *Mesh
}

var (
	mergeMu     sync.Mutex
	mergeGroups = map[string]*mergedGroup{}
	mergeLevel  *Level
)

// mergedStatic draws group key (built by fill, which calls add for every
// plant) for chamber L.
func mergedStatic(B *Batch, L *Level, key string, fill func(add func(s *floraSet, pos, scale V3, ry float64))) {
	mergeMu.Lock()
	if mergeLevel != L {
		mergeLevel = L
		mergeGroups = map[string]*mergedGroup{}
	}
	g := mergeGroups[key]
	mergeMu.Unlock()
	if g == nil {
		m := &Mesh{}
		fill(func(s *floraSet, pos, scale V3, ry float64) {
			if s.impOf != nil {
				s = s.impOf
			}
			rot := rotXYZ(0, ry, 0)
			for _, src := range []*Mesh{s.wood, s.leaf} {
				for _, t := range src.Tris {
					var q [3]V3
					for k := 0; k < 3; k++ {
						p := t.P[k]
						q[k] = rot.Mul(V3{p.X * scale.X, p.Y * scale.Y, p.Z * scale.Z}).Add(pos)
					}
					m.Tris = append(m.Tris, Tri{P: q, Col: t.Col, Emi: t.Emi, Fixed: t.Fixed})
				}
			}
		})
		m.topology()
		g = &mergedGroup{m: m}
		mergeMu.Lock()
		if old := mergeGroups[key]; old != nil {
			g = old
		} else {
			mergeGroups[key] = g
		}
		mergeMu.Unlock()
	}
	if len(g.m.Tris) > 0 {
		B.Draw(g.m, V3{}, v3(1), identity, color.RGBA{}, 0)
	}
}
