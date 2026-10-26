// Command portraits renders the class character portraits used on the
// class select screen. It is an offline signed-distance-field ray marcher:
// every character is modelled from smooth-blended primitives, lit with a
// warm key light, a coloured rim light and the glow of its own magic, then
// finished with bloom, embers, film grain and a soft alpha vignette.
//
//	go run ./tools/portraits            # all classes -> assets/portraits
//	go run ./tools/portraits pyro 0.5   # one class at half resolution
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// ---------------------------------------------------------------- vectors

type V struct{ X, Y, Z float64 }

func v3(x, y, z float64) V  { return V{x, y, z} }
func (a V) Add(b V) V       { return V{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a V) Sub(b V) V       { return V{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a V) Mul(s float64) V { return V{a.X * s, a.Y * s, a.Z * s} }
func (a V) MulV(b V) V      { return V{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a V) Dot(b V) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a V) Len() float64    { return math.Sqrt(a.Dot(a)) }
func (a V) Norm() V {
	l := a.Len()
	if l == 0 {
		return a
	}
	return a.Mul(1 / l)
}
func (a V) Abs() V                { return V{math.Abs(a.X), math.Abs(a.Y), math.Abs(a.Z)} }
func (a V) Cross(b V) V           { return V{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X} }
func (a V) Lerp(b V, t float64) V { return a.Add(b.Sub(a).Mul(t)) }
func (a V) MaxS(s float64) V      { return V{math.Max(a.X, s), math.Max(a.Y, s), math.Max(a.Z, s)} }

func clamp(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
func mix(a, b, t float64) float64   { return a + (b-a)*t }
func smooth(a, b, x float64) float64 {
	t := clamp((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}
func hexc(h uint32) V {
	return V{float64(h>>16&255) / 255, float64(h>>8&255) / 255, float64(h&255) / 255}.pow(2.2)
}
func (a V) pow(e float64) V { return V{math.Pow(a.X, e), math.Pow(a.Y, e), math.Pow(a.Z, e)} }

// rotY rotates p about the Y axis.
func rotY(p V, a float64) V {
	c, s := math.Cos(a), math.Sin(a)
	return V{c*p.X + s*p.Z, p.Y, -s*p.X + c*p.Z}
}
func rotX(p V, a float64) V {
	c, s := math.Cos(a), math.Sin(a)
	return V{p.X, c*p.Y - s*p.Z, s*p.Y + c*p.Z}
}
func rotZ(p V, a float64) V {
	c, s := math.Cos(a), math.Sin(a)
	return V{c*p.X - s*p.Y, s*p.X + c*p.Y, p.Z}
}

// ---------------------------------------------------------------- noise

func hash3(x, y, z int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(z)*2147483647
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0xffffff)
}

func vnoise(p V) float64 {
	xi, yi, zi := math.Floor(p.X), math.Floor(p.Y), math.Floor(p.Z)
	fx, fy, fz := p.X-xi, p.Y-yi, p.Z-zi
	ux, uy, uz := fx*fx*(3-2*fx), fy*fy*(3-2*fy), fz*fz*(3-2*fz)
	x, y, z := int(xi), int(yi), int(zi)
	a := mix(hash3(x, y, z), hash3(x+1, y, z), ux)
	b := mix(hash3(x, y+1, z), hash3(x+1, y+1, z), ux)
	c := mix(hash3(x, y, z+1), hash3(x+1, y, z+1), ux)
	d := mix(hash3(x, y+1, z+1), hash3(x+1, y+1, z+1), ux)
	return mix(mix(a, b, uy), mix(c, d, uy), uz)
}

func fbm(p V, oct int) float64 {
	s, a, n := 0.0, 0.5, 0.0
	for i := 0; i < oct; i++ {
		s += a * vnoise(p)
		n += a
		p = p.Mul(2.03).Add(V{1.7, 9.2, 3.1})
		a *= 0.5
	}
	return s / n
}

// ---------------------------------------------------------------- SDFs

func sdSphere(p V, r float64) float64 { return p.Len() - r }

func sdEllipsoid(p V, r V) float64 {
	k0 := V{p.X / r.X, p.Y / r.Y, p.Z / r.Z}.Len()
	k1 := V{p.X / (r.X * r.X), p.Y / (r.Y * r.Y), p.Z / (r.Z * r.Z)}.Len()
	if k1 == 0 {
		return -math.Min(r.X, math.Min(r.Y, r.Z))
	}
	return k0 * (k0 - 1) / k1
}

func sdCapsule(p, a, b V, r float64) float64 {
	pa, ba := p.Sub(a), b.Sub(a)
	h := clamp(pa.Dot(ba)/ba.Dot(ba), 0, 1)
	return pa.Sub(ba.Mul(h)).Len() - r
}

// sdCone is a capsule whose radius tapers from ra at a to rb at b.
func sdCone(p, a, b V, ra, rb float64) float64 {
	pa, ba := p.Sub(a), b.Sub(a)
	h := clamp(pa.Dot(ba)/ba.Dot(ba), 0, 1)
	return pa.Sub(ba.Mul(h)).Len() - mix(ra, rb, h)
}

func sdBox(p, b V, r float64) float64 {
	q := p.Abs().Sub(b).Add(V{r, r, r})
	return q.MaxS(0).Len() + math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0) - r
}

func sdTorus(p V, R, r float64) float64 {
	q := math.Hypot(p.X, p.Z) - R
	return math.Hypot(q, p.Y) - r
}

// sdCyl is a capped vertical cylinder (radius r, half height h, rounding rr).
func sdCyl(p V, r, h, rr float64) float64 {
	dx := math.Hypot(p.X, p.Z) - r + rr
	dy := math.Abs(p.Y) - h + rr
	return math.Min(math.Max(dx, dy), 0) + math.Hypot(math.Max(dx, 0), math.Max(dy, 0)) - rr
}

func smin(a, b, k float64) float64 {
	if k <= 0 {
		return math.Min(a, b)
	}
	h := clamp(0.5+0.5*(b-a)/k, 0, 1)
	return mix(b, a, h) - k*h*(1-h)
}
func smax(a, b, k float64) float64 { return -smin(-a, -b, k) }

// chain sweeps a tapering tube through points.
func chain(p V, pts []V, r0, r1 float64) float64 {
	d := 1e9
	n := len(pts) - 1
	for i := 0; i < n; i++ {
		ra := mix(r0, r1, float64(i)/float64(n))
		rb := mix(r0, r1, float64(i+1)/float64(n))
		d = math.Min(d, sdCone(p, pts[i], pts[i+1], ra, rb))
	}
	return d
}

// ---------------------------------------------------------------- scene

// Mat is a surface material.
type Mat struct {
	Alb     V
	Rough   float64
	Metal   float64
	Emit    V
	Cloth   bool    // wrap lighting + weave noise
	Grime   float64 // darkening noise amount
	Sheen   float64 // extra rim response
	Bump    float64 // micro surface relief strength
	BumpF   float64 // relief frequency
	Pattern func(p V, alb V) V
}

// Light is a point light (magic, embers).
type Light struct {
	P   V
	Col V
	R   float64 // influence radius scale
}

// Arc is a lightning bolt drawn in post.
type Arc struct {
	A, B  V
	Col   V
	Width float64
	Seed  int64
	Kids  int
}

// Spark is a glowing particle drawn in post (world-space).
type Spark struct {
	P    V
	Col  V
	Size float64
}

type Scene struct {
	Name    string
	SDF     func(p V) (float64, int)
	Mats    map[int]Mat
	Key     V // key light direction (towards light)
	KeyCol  V
	Rim     V
	RimCol  V
	Rim2    V
	Rim2Col V
	Fill    V
	FillCol V
	Amb     V
	Lights  []Light
	Glow    V // backdrop glow colour
	GlowPos [2]float64
	Mist    V
	Arcs    []Arc
	Sparks  []Spark
	Embers  V // colour of drifting embers (zero = dust)
	Bound   float64
	CamPos  V
	CamTgt  V
	FOV     float64
}

// ---------------------------------------------------------------- render

const (
	maxSteps = 200
	maxDist  = 9.0
)

type cam struct {
	pos, fw, rt, up V
	tanH            float64
	aspect          float64
}

func newCam(s *Scene, aspect float64) cam {
	fw := s.CamTgt.Sub(s.CamPos).Norm()
	rt := fw.Cross(V{0, 1, 0}).Norm()
	up := rt.Cross(fw)
	return cam{s.CamPos, fw, rt, up, math.Tan(s.FOV * math.Pi / 360), aspect}
}

func (c cam) ray(u, v float64) V { // u,v in -1..1 (v up)
	return c.fw.Add(c.rt.Mul(u * c.tanH * c.aspect)).Add(c.up.Mul(v * c.tanH)).Norm()
}

func (c cam) project(p V) (float64, float64, bool) {
	d := p.Sub(c.pos)
	z := d.Dot(c.fw)
	if z <= 0 {
		return 0, 0, false
	}
	u := d.Dot(c.rt) / (z * c.tanH * c.aspect)
	v := d.Dot(c.up) / (z * c.tanH)
	return u, v, true
}

func march(s *Scene, ro, rd V) (float64, int, bool) {
	// bounding sphere around the character
	oc := ro.Sub(V{0, 1.2, 0})
	b := oc.Dot(rd)
	c := oc.Dot(oc) - s.Bound*s.Bound
	if b*b-c < 0 {
		return 0, 0, false
	}
	t := math.Max(0, -b-math.Sqrt(b*b-c))
	tEnd := -b + math.Sqrt(b*b-c)
	for i := 0; i < maxSteps && t < tEnd; i++ {
		d, m := s.SDF(ro.Add(rd.Mul(t)))
		if d < 0.0004*t {
			return t, m, true
		}
		t += d * 0.85
	}
	return 0, 0, false
}

func normal(s *Scene, p V) V {
	const e = 0.0008
	k := []V{{1, -1, -1}, {-1, -1, 1}, {-1, 1, -1}, {1, 1, 1}}
	n := V{}
	for _, kk := range k {
		d, _ := s.SDF(p.Add(kk.Mul(e)))
		n = n.Add(kk.Mul(d))
	}
	return n.Norm()
}

func softShadow(s *Scene, ro, rd V, k float64) float64 {
	res := 1.0
	t := 0.01
	for i := 0; i < 48 && t < 2.5; i++ {
		h, _ := s.SDF(ro.Add(rd.Mul(t)))
		res = math.Min(res, k*h/t)
		if res < 0.002 {
			return 0
		}
		t += clamp(h, 0.006, 0.12)
	}
	return clamp(res, 0, 1)
}

func ao(s *Scene, p, n V) float64 {
	occ, sc := 0.0, 1.0
	for i := 0; i < 6; i++ {
		h := 0.01 + 0.035*float64(i)
		d, _ := s.SDF(p.Add(n.Mul(h)))
		occ += (h - d) * sc
		sc *= 0.82
	}
	return clamp(1-2.2*occ, 0, 1)
}

func shade(s *Scene, p, rd V, m int) V {
	mt, ok := s.Mats[m]
	if !ok {
		mt = Mat{Alb: V{0.5, 0.5, 0.5}, Rough: 0.6}
	}
	n := normal(s, p)
	if mt.Bump > 0 {
		f := mt.BumpF
		if f == 0 {
			f = 60
		}
		const e = 0.002
		b0 := fbm(p.Mul(f), 3)
		g := V{fbm(p.Add(V{e, 0, 0}).Mul(f), 3) - b0, fbm(p.Add(V{0, e, 0}).Mul(f), 3) - b0, fbm(p.Add(V{0, 0, e}).Mul(f), 3) - b0}.Mul(1 / e)
		g = g.Sub(n.Mul(g.Dot(n)))
		n = n.Sub(g.Mul(mt.Bump / f)).Norm()
	}
	alb := mt.Alb
	if mt.Pattern != nil {
		alb = mt.Pattern(p, alb)
	}
	if mt.Cloth {
		w := 0.88 + 0.12*fbm(p.Mul(90), 2)
		alb = alb.Mul(w)
	}
	if mt.Grime > 0 {
		g := fbm(p.Mul(7), 4)
		alb = alb.Mul(1 - mt.Grime*smooth(0.35, 0.75, g))
	}
	// micro roughness noise on metals breaks up highlights
	rough := mt.Rough
	if mt.Metal > 0 {
		rough = clamp(rough+0.18*(fbm(p.Mul(28), 3)-0.5), 0.08, 1)
	}
	v := rd.Mul(-1)
	occ := ao(s, p, n)
	nv := math.Max(n.Dot(v), 0)
	f0 := V{0.04, 0.04, 0.04}.Lerp(alb, mt.Metal)
	diffC := alb.Mul(1 - mt.Metal)

	col := V{}
	// ambient: dim sky from above, darker bounce from below
	hemi := 0.5 + 0.5*n.Y
	col = col.Add(diffC.MulV(s.Amb).Mul((0.35 + 0.65*hemi) * occ))
	// fake environment reflection so metals read: dim studio gradient
	// plus the coloured backlight and a warm softbox
	if mt.Metal > 0 || mt.Rough < 0.5 {
		r := rd.Sub(n.Mul(2 * rd.Dot(n)))
		env := V{0.02, 0.02, 0.025}.Lerp(s.Glow.Mul(0.18).Add(V{0.08, 0.08, 0.09}), smooth(-0.3, 0.9, r.Y))
		env = env.Add(s.RimCol.Mul(0.5 * math.Pow(math.Max(r.Dot(s.Rim), 0), 6)))
		env = env.Add(s.KeyCol.Mul(0.6 * math.Pow(math.Max(r.Dot(s.Key), 0), 10)))
		fe := f0.Add(V{1, 1, 1}.Sub(f0).Mul(math.Pow(1-nv, 5)))
		blur := 1 - 0.6*rough
		col = col.Add(fe.MulV(env).Mul(blur * occ * 1.4))
	}
	col = col.Add(f0.MulV(s.Amb).Mul(0.6 * occ * (0.3 + 0.7*hemi)))

	light := func(L, lc V, sh float64, wrap float64) {
		h := L.Add(v).Norm()
		ndl := n.Dot(L)
		d := math.Max((ndl+wrap)/(1+wrap), 0)
		if d <= 0 && mt.Sheen == 0 {
			return
		}
		nh := math.Max(n.Dot(h), 0)
		a := rough * rough
		a2 := a * a
		den := nh*nh*(a2-1) + 1
		D := a2 / (math.Pi * den * den)
		F := f0.Add(V{1, 1, 1}.Sub(f0).Mul(math.Pow(1-math.Max(h.Dot(v), 0), 5)))
		spec := F.Mul(D * 0.25 * math.Max(ndl, 0) / math.Max(0.25, nv*0.5+0.5))
		c := diffC.Mul(d / math.Pi * 3.2).Add(spec)
		col = col.Add(c.MulV(lc).Mul(sh))
	}
	pp := p.Add(n.Mul(0.003))
	w := 0.0
	if mt.Cloth {
		w = 0.35
	}
	ksh := softShadow(s, pp, s.Key, 10)
	light(s.Key, s.KeyCol, ksh*(0.3+0.7*occ), w)
	light(s.Fill, s.FillCol, occ, w)
	// rim: back light, wraps strongly so silhouettes glow
	fr := math.Pow(1-nv, 2.5)
	for _, rl := range [][2]V{{s.Rim, s.RimCol}, {s.Rim2, s.Rim2Col}} {
		if rl[1] == (V{}) {
			continue
		}
		rdl := math.Max(n.Dot(rl[0])+0.3, 0) / 1.3
		rimAmt := rdl*0.15 + fr*rdl*2.4
		rc := alb.Lerp(V{1, 1, 1}, 0.45+0.3*mt.Sheen)
		if mt.Metal > 0 {
			rc = alb.Mul(1.2)
		}
		col = col.Add(rl[1].MulV(rc).Mul(rimAmt * (0.4 + 0.6*occ)))
		light(rl[0], rl[1].Mul(0.7), 1, 0)
	}
	// point lights from the character's own magic
	for _, l := range s.Lights {
		dv := l.P.Sub(p)
		dist := dv.Len()
		L := dv.Mul(1 / dist)
		att := 1 / (1 + dist*dist/(l.R*l.R))
		light(L, l.Col.Mul(att), 0.35+0.65*occ, 0.25)
	}
	col = col.Add(mt.Emit)
	return col
}

func background(s *Scene, u, v float64, rd V) V {
	// deep vertical gradient
	top := V{0.004, 0.004, 0.007}
	mid := s.Mist.Mul(0.35)
	t := smooth(-1.1, 0.9, v)
	col := mid.Lerp(top, t)
	// cool sky light shaft from above-behind
	dx := u - s.GlowPos[0]
	dy := v - s.GlowPos[1]
	g := math.Exp(-(dx*dx*2.2 + dy*dy*1.4))
	col = col.Add(s.Glow.Mul(0.16 * g))
	shaft := math.Exp(-math.Pow((u-s.GlowPos[0]-0.25*(v-0.2))*2.6, 2)) * smooth(-0.6, 1.0, v)
	col = col.Add(s.Glow.Mul(0.05 * shaft))
	// drifting mist
	m := fbm(V{u*1.6 + 3, v * 3.2, 1.3}, 5)
	m2 := fbm(V{u*3.4 - 7, v*1.5 + 2, 4.1}, 4)
	col = col.Add(s.Mist.Mul(0.5 * smooth(0.45, 0.85, m) * smooth(0.6, -0.9, v)))
	col = col.Add(s.Glow.Mul(0.06 * smooth(0.5, 0.8, m2) * g))
	// ground haze
	col = col.Add(s.Mist.Mul(0.4 * smooth(-0.5, -1.1, v)))
	return col
}

// ---------------------------------------------------------------- post

type img struct {
	w, h int
	px   []V
}

func newImg(w, h int) *img { return &img{w, h, make([]V, w*h)} }
func (im *img) at(x, y int) V {
	x = max(0, min(im.w-1, x))
	y = max(0, min(im.h-1, y))
	return im.px[y*im.w+x]
}

func blur(im *img, r float64) *img {
	// separable gaussian
	k := int(math.Ceil(r * 3))
	ws := make([]float64, 2*k+1)
	sum := 0.0
	for i := -k; i <= k; i++ {
		ws[i+k] = math.Exp(-float64(i*i) / (2 * r * r))
		sum += ws[i+k]
	}
	for i := range ws {
		ws[i] /= sum
	}
	tmp := newImg(im.w, im.h)
	out := newImg(im.w, im.h)
	par(im.h, func(y int) {
		for x := 0; x < im.w; x++ {
			c := V{}
			for i := -k; i <= k; i++ {
				c = c.Add(im.at(x+i, y).Mul(ws[i+k]))
			}
			tmp.px[y*im.w+x] = c
		}
	})
	par(im.h, func(y int) {
		for x := 0; x < im.w; x++ {
			c := V{}
			for i := -k; i <= k; i++ {
				c = c.Add(tmp.at(x, y+i).Mul(ws[i+k]))
			}
			out.px[y*im.w+x] = c
		}
	})
	return out
}

func downsample(im *img, f int) *img {
	o := newImg(im.w/f, im.h/f)
	for y := 0; y < o.h; y++ {
		for x := 0; x < o.w; x++ {
			c := V{}
			for j := 0; j < f; j++ {
				for i := 0; i < f; i++ {
					c = c.Add(im.at(x*f+i, y*f+j))
				}
			}
			o.px[y*o.w+x] = c.Mul(1 / float64(f*f))
		}
	}
	return o
}

func sampleBilinear(im *img, x, y float64) V {
	x -= 0.5
	y -= 0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	a := im.at(int(x0), int(y0)).Lerp(im.at(int(x0)+1, int(y0)), fx)
	b := im.at(int(x0), int(y0)+1).Lerp(im.at(int(x0)+1, int(y0)+1), fx)
	return a.Lerp(b, fy)
}

func par(n int, f func(i int)) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	next := 0
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				i := next
				next++
				mu.Unlock()
				if i >= n {
					return
				}
				f(i)
			}
		}()
	}
	wg.Wait()
}

func aces(x float64) float64 {
	return clamp((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14), 0, 1)
}

// glowDot splats a soft additive dot.
func glowDot(im *img, x, y, r float64, c V) {
	k := int(r*3) + 1
	for j := -k; j <= k; j++ {
		for i := -k; i <= k; i++ {
			px, py := int(x)+i, int(y)+j
			if px < 0 || py < 0 || px >= im.w || py >= im.h {
				continue
			}
			dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
			g := math.Exp(-(dx*dx + dy*dy) / (2 * r * r))
			im.px[py*im.w+px] = im.px[py*im.w+px].Add(c.Mul(g))
		}
	}
}

func glowLine(im *img, x0, y0, x1, y1, r float64, c V) {
	l := math.Hypot(x1-x0, y1-y0)
	n := int(l/(r*0.5)) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		glowDot(im, mix(x0, x1, t), mix(y0, y1, t), r, c.Mul(r*0.5/math.Max(r*0.5, 1)*0.6))
	}
}

// ---------------------------------------------------------------- main

func render(s *Scene, W, H, spp int) *image.NRGBA {
	start := time.Now()
	c := newCam(s, float64(W)/float64(H))
	hdr := newImg(W, H)
	mask := make([]float64, W*H)
	grid := int(math.Sqrt(float64(spp)))
	par(H, func(y int) {
		rng := rand.New(rand.NewSource(int64(y)*7919 + 1))
		for x := 0; x < W; x++ {
			acc := V{}
			cov := 0.0
			for sy := 0; sy < grid; sy++ {
				for sx := 0; sx < grid; sx++ {
					jx := (float64(sx) + rng.Float64()) / float64(grid)
					jy := (float64(sy) + rng.Float64()) / float64(grid)
					u := (float64(x)+jx)/float64(W)*2 - 1
					v := 1 - (float64(y)+jy)/float64(H)*2
					rd := c.ray(u, v)
					t, m, hit := march(s, c.pos, rd)
					if hit {
						p := c.pos.Add(rd.Mul(t))
						col := shade(s, p, rd, m)
						// thin atmospheric haze
						fog := 1 - math.Exp(-math.Max(0, t-3.0)*0.12)
						col = col.Lerp(s.Mist.Mul(0.6), fog)
						acc = acc.Add(col)
						cov++
					} else {
						acc = acc.Add(background(s, u, v, rd))
					}
				}
			}
			n := float64(grid * grid)
			hdr.px[y*W+x] = acc.Mul(1 / n)
			mask[y*W+x] = cov / n
		}
		if y%100 == 0 {
			fmt.Printf("  %s row %d/%d\n", s.Name, y, H)
		}
	})
	fmt.Printf("  %s traced in %v\n", s.Name, time.Since(start).Round(time.Second))

	// ---- post effects in HDR
	sc := float64(H) / 1200
	pr := rand.New(rand.NewSource(int64(len(s.Name)) * 31))
	// world-space sparks (occluded crudely by the character mask)
	for _, sp := range s.Sparks {
		u, v, ok := c.project(sp.P)
		if !ok {
			continue
		}
		x := (u + 1) / 2 * float64(W)
		y := (1 - v) / 2 * float64(H)
		glowDot(hdr, x, y, sp.Size*sc, sp.Col)
		glowDot(hdr, x, y, sp.Size*sc*0.35, sp.Col.Mul(2))
	}
	// lightning
	for _, a := range s.Arcs {
		drawBolt(hdr, c, a, W, H, sc, rand.New(rand.NewSource(a.Seed)), a.Kids)
	}
	// drifting embers / dust in screen space
	ec := s.Embers
	dust := ec == (V{})
	for i := 0; i < 140; i++ {
		x := pr.Float64() * float64(W)
		y := pr.Float64() * float64(H)
		depth := pr.Float64()
		r := (0.8 + 2.6*depth*depth) * sc
		col := ec
		if dust {
			col = V{0.5, 0.48, 0.45}.Mul(0.25)
		}
		bright := 0.15 + 0.85*pr.Float64()*pr.Float64()
		if depth > 0.85 { // close, out of focus bokeh
			r *= 3
			bright *= 0.25
		}
		glowDot(hdr, x, y, r, col.Mul(bright*1.4))
	}
	// bloom
	bright := newImg(W, H)
	for i, p := range hdr.px {
		l := 0.2126*p.X + 0.7152*p.Y + 0.0722*p.Z
		k := smooth(0.9, 3.5, l)
		bright.px[i] = p.Mul(k)
	}
	b1 := blur(downsample(bright, 2), 4*sc)
	b2 := blur(downsample(bright, 4), 8*sc)
	b3 := blur(downsample(bright, 8), 12*sc)
	out := image.NewNRGBA(image.Rect(0, 0, W, H))
	gr := rand.New(rand.NewSource(99))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			p := hdr.px[y*W+x]
			p = p.Add(sampleBilinear(b1, fx/2, fy/2).Mul(0.35))
			p = p.Add(sampleBilinear(b2, fx/4, fy/4).Mul(0.3))
			p = p.Add(sampleBilinear(b3, fx/8, fy/8).Mul(0.25))
			// exposure + filmic curve
			p = p.Mul(1.0)
			r, g, b := aces(p.X), aces(p.Y), aces(p.Z)
			// gentle grade: lift blacks toward the mist colour, desaturate highs a touch
			lum := 0.3*r + 0.59*g + 0.11*b
			r, g, b = mix(lum, r, 1.08), mix(lum, g, 1.08), mix(lum, b, 1.08)
			// vignette alpha: soft fade to transparent at the frame edges
			nx, ny := fx/float64(W)*2-1, fy/float64(H)*2-1
			ax := smooth(1.0, 0.62, math.Abs(nx))
			ayTop := smooth(-1.0, -0.82, ny)
			ayBot := smooth(1.0, 0.55, ny)
			alpha := ax * ayTop * ayBot
			// keep the figure itself solid a little further out
			alpha = math.Max(alpha, mask[y*W+x]*ax*smooth(1.0, 0.8, ny)*0.97)
			dark := 1 - 0.35*(nx*nx*0.6+ny*ny*0.4)
			r, g, b = r*dark, g*dark, b*dark
			gn := (gr.Float64() - 0.5) * 0.008
			to8 := func(c float64) uint8 {
				c = math.Pow(clamp(c+gn, 0, 1), 1/2.2)
				return uint8(clamp(c*255+0.5, 0, 255))
			}
			out.SetNRGBA(x, y, color.NRGBA{to8(r), to8(g), to8(b), uint8(clamp(alpha*255+0.5, 0, 255))})
		}
	}
	return out
}

func drawBolt(im *img, c cam, a Arc, W, H int, sc float64, r *rand.Rand, kids int) {
	pts := []V{a.A, a.B}
	// midpoint displacement in 3D
	disp := a.A.Sub(a.B).Len() * 0.22
	for it := 0; it < 6; it++ {
		var np []V
		for i := 0; i < len(pts)-1; i++ {
			m := pts[i].Lerp(pts[i+1], 0.5)
			m = m.Add(V{r.Float64() - 0.5, r.Float64() - 0.5, r.Float64() - 0.5}.Mul(disp))
			np = append(np, pts[i], m)
		}
		np = append(np, pts[len(pts)-1])
		pts = np
		disp *= 0.55
	}
	proj := func(p V) (float64, float64) {
		u, v, _ := c.project(p)
		return (u + 1) / 2 * float64(W), (1 - v) / 2 * float64(H)
	}
	for i := 0; i < len(pts)-1; i++ {
		x0, y0 := proj(pts[i])
		x1, y1 := proj(pts[i+1])
		glowLine(im, x0, y0, x1, y1, a.Width*3*sc, a.Col.Mul(0.25))
		glowLine(im, x0, y0, x1, y1, a.Width*sc, a.Col.Lerp(V{1, 1, 1}, 0.6).Mul(1.6))
	}
	for k := 0; k < kids; k++ {
		i := r.Intn(len(pts)-2) + 1
		dir := a.B.Sub(a.A)
		end := pts[i].Add(dir.Mul(0.25 + 0.2*r.Float64())).Add(V{r.Float64() - 0.5, r.Float64() - 0.5, r.Float64() - 0.5}.Mul(dir.Len() * 0.35))
		drawBolt(im, c, Arc{pts[i], end, a.Col, a.Width * 0.6, r.Int63(), 0}, W, H, sc, r, 0)
	}
}

func main() {
	scale := 1.0
	only := ""
	if len(os.Args) > 1 {
		only = os.Args[1]
	}
	if len(os.Args) > 2 {
		scale, _ = strconv.ParseFloat(os.Args[2], 64)
	}
	outDir := "assets/portraits"
	if d := os.Getenv("OUT"); d != "" {
		outDir = d
	}
	os.MkdirAll(outDir, 0o755)
	W, H := int(900*scale), int(1200*scale)
	spp := 4
	if scale < 0.7 {
		spp = 1
	}
	for _, mk := range classes {
		s := mk()
		if only != "" && only != "all" && s.Name != only {
			continue
		}
		im := render(s, W, H, spp)
		// colour as JPEG + alpha as a greyscale PNG keeps the game small
		rgb := image.NewRGBA(im.Bounds())
		al := image.NewGray(im.Bounds())
		for i := 0; i < len(im.Pix); i += 4 {
			copy(rgb.Pix[i:i+3], im.Pix[i:i+3])
			rgb.Pix[i+3] = 255
			al.Pix[i/4] = im.Pix[i+3]
		}
		f, err := os.Create(filepath.Join(outDir, s.Name+".jpg"))
		if err != nil {
			panic(err)
		}
		jpeg.Encode(f, rgb, &jpeg.Options{Quality: 90})
		f.Close()
		f, _ = os.Create(filepath.Join(outDir, s.Name+"_a.png"))
		png.Encode(f, al)
		f.Close()
		fmt.Println("wrote", s.Name)
	}
}
