package main

import "math"

// H accumulates the closest surface and its material.
type H struct {
	d float64
	m int
}

func newH() H { return H{1e9, 0} }

func (h *H) add(d float64, m int) {
	if d < h.d {
		h.d, h.m = d, m
	}
}

// blend smooth-unions d into h, taking d's material where it is nearer.
func (h *H) blend(d float64, m int, k float64) {
	if d < h.d {
		h.m = m
	}
	h.d = smin(h.d, d, k)
}

// cut carves d out of the current shape.
func (h *H) cut(d, k float64) { h.d = smax(h.d, -d, k) }

// ---------------------------------------------------------------- body

// Arm describes one arm pose in character space.
type Arm struct {
	S, E, W V // shoulder, elbow, wrist
	Hand    V // hand centre (direction wrist -> hand sets the hand frame)
	Fist    bool
	Flip    bool    // palm facing the other way
	Curl    float64 // relaxed finger curl for open hands
}

// limb returns the distance to an arm (upper + forearm), radius tapering.
func limb(q V, a Arm, r0, r1 float64) float64 {
	d := sdCone(q, a.S, a.E, r0, mix(r0, r1, 0.5))
	return smin(d, sdCone(q, a.E, a.W, mix(r0, r1, 0.5), r1), 0.02)
}

// hand models a gloved hand in a frame built from the forearm direction:
// a palm block, four fingers (curled into a fist or relaxed and slightly
// spread) and a thumb.
func hand(q V, a Arm) float64 {
	f := a.Hand.Sub(a.W).Norm()
	up := V{0, 1, 0}
	if math.Abs(f.Y) > 0.9 {
		up = V{0, 0, 1}
	}
	sd := f.Cross(up).Norm() // across the knuckles
	n := sd.Cross(f)         // back of the hand
	side := 1.0
	if a.S.X < 0 {
		side = -1
	}
	if a.Flip {
		n = n.Mul(-1)
	}
	o := q.Sub(a.W)
	l := V{o.Dot(sd), o.Dot(n), o.Dot(f)}
	if l.Len() > 0.2 {
		return l.Len() - 0.1 // cheap bound
	}
	palm := sdBox(l.Sub(V{0, 0, 0.04}), V{0.032, 0.012, 0.036}, 0.011)
	palm = smin(palm, sdCone(l, V{0, 0, -0.01}, V{0, 0, 0.02}, 0.028, 0.03), 0.015)
	d := palm
	for i := 0; i < 4; i++ {
		x := (float64(i) - 1.5) * 0.017
		ln := 0.05 - 0.008*math.Abs(float64(i)-1.3)
		b := V{x, 0.002, 0.074}
		var fd float64
		if a.Fist {
			// curled: forward then down and back under the palm
			m := b.Add(V{0, -0.016, 0.014})
			e := b.Add(V{0, -0.03, -0.006})
			fd = math.Min(sdCone(l, b, m, 0.0095, 0.009), sdCone(l, m, e, 0.009, 0.008))
		} else {
			spread := x * 0.5
			m := b.Add(V{spread, -0.006 - a.Curl*0.01, ln * 0.55})
			e := m.Add(V{spread * 0.6, -0.008 - a.Curl*0.02, ln * 0.45})
			fd = math.Min(sdCone(l, b, m, 0.0085, 0.0078), sdCone(l, m, e, 0.0078, 0.0068))
		}
		d = smin(d, fd, 0.006)
	}
	// thumb on the inner side
	tb := V{-side * 0.026, -0.008, 0.02}
	tm := tb.Add(V{-side * 0.018, -0.012, 0.024})
	te := tm.Add(V{side * 0.004, -0.006, 0.024})
	if a.Fist {
		te = tm.Add(V{side * 0.02, -0.004, 0.012})
	}
	d = smin(d, math.Min(sdCone(l, tb, tm, 0.011, 0.0095), sdCone(l, tm, te, 0.0095, 0.008)), 0.008)
	return d
}

// headTilt rotates points around the neck so the head can tilt and turn
// (roll about Z, nod about X, turn about Y), only near the head.
func headTilt(q V, neck V, roll, nod, turn float64) V {
	w := smooth(neck.Y-0.04, neck.Y+0.06, q.Y) * smooth(0.3, 0.18, math.Hypot(q.X-neck.X, q.Z-neck.Z))
	if w <= 0 {
		return q
	}
	o := q.Sub(neck)
	o = rotY(o, -turn*w)
	o = rotX(o, -nod*w)
	o = rotZ(o, -roll*w)
	return o.Add(neck)
}

// sway bends cloth below y0 sideways like a breeze.
func sway(q V, y0, amt float64) V {
	w := smooth(y0, y0-1.0, q.Y)
	q.X -= amt * w * w
	q.Z += amt * 0.3 * w * w * math.Sin(q.Y*4)
	return q
}

// folds displaces cloth radially for vertical drapery.
func folds(q V, n, amp, y0, y1 float64) float64 {
	a := math.Atan2(q.X, q.Z)
	w := smooth(y0, y1, q.Y)
	// warp the fold phase so creases wander like real drapery
	a += 0.25*math.Sin(q.Y*3.1+a*2) + 0.12*math.Sin(q.Y*7.3)
	s1 := 1 - math.Abs(math.Sin(a*n*0.5+q.Y*1.3)) // sharp creases
	s2 := math.Sin(a*n*1.7 + 1.7 + q.Y*2.9)
	return amp * w * (1.4*s1*s1 - 0.5 + 0.35*s2)
}

// torso is a generic chest/abdomen/pelvis mass.
func torso(q V, broad float64) float64 {
	chest := sdEllipsoid(q.Sub(V{0, 1.37, 0}), V{0.19 * broad, 0.2, 0.125})
	abd := sdEllipsoid(q.Sub(V{0, 1.15, 0.0}), V{0.155 * broad, 0.17, 0.105})
	pel := sdEllipsoid(q.Sub(V{0, 0.97, 0}), V{0.17 * broad, 0.12, 0.11})
	d := smin(chest, abd, 0.06)
	d = smin(d, pel, 0.05)
	neck := sdCone(q, V{0, 1.52, -0.01}, V{0, 1.7, 0.0}, 0.065, 0.05)
	return smin(d, neck, 0.04)
}

func legs(q V, spread float64) float64 {
	l := sdCone(q, V{-0.09, 0.95, 0}, V{-0.1 - spread, 0.5, 0.02}, 0.085, 0.06)
	r := sdCone(q, V{0.09, 0.95, 0}, V{0.1 + spread, 0.5, -0.02}, 0.085, 0.06)
	l2 := sdCone(q, V{-0.1 - spread, 0.5, 0.02}, V{-0.11 - spread, 0.08, 0.0}, 0.06, 0.045)
	r2 := sdCone(q, V{0.1 + spread, 0.5, -0.02}, V{0.12 + spread, 0.08, -0.03}, 0.06, 0.045)
	return math.Min(math.Min(l, r), math.Min(l2, r2))
}

// hood is a pointed cowl around a head centred at c; returns shell and
// the opening's inner void (for the shadowed face).
func hood(q, c V, peak float64) (shell, void float64) {
	o := q.Sub(c)
	outer := sdEllipsoid(o.Sub(V{0, 0.02, -0.025}), V{0.135, 0.15, 0.15})
	tip := sdCone(o, V{0, 0.06, -0.06}, V{0, 0.06 + peak, -0.08 - peak*0.6}, 0.09, 0.01)
	outer = smin(outer, tip, 0.06)
	// drape down onto the shoulders
	drape := sdCone(o, V{0, -0.04, -0.03}, V{0, -0.2, -0.02}, 0.13, 0.2)
	outer = smin(outer, drape, 0.05)
	inner := sdEllipsoid(o.Sub(V{0, 0.0, -0.01}), V{0.11, 0.125, 0.13})
	shell = smax(outer, -inner, 0.01)
	// front opening
	open := sdEllipsoid(o.Sub(V{0, -0.02, 0.13}), V{0.062, 0.095, 0.09})
	shell = smax(shell, -open, 0.012)
	void = sdEllipsoid(o.Sub(V{0, -0.005, 0.0}), V{0.085, 0.11, 0.105})
	return
}

// cloak is an open shell hanging from the shoulders, wrapping behind.
func cloak(q V, top, bot V, rTop, rBot, front float64, nf float64) float64 {
	d := sdCone(q, top, bot, rTop, rBot)
	d += folds(q, nf, 0.012, 1.45, 0.3)
	d = math.Abs(d) - 0.011
	d = smax(d, q.Z-front, 0.02)
	return d
}

// gem / sigil helpers
func ring(q, c V, R, r float64, tiltX, tiltZ float64) float64 {
	o := rotZ(rotX(q.Sub(c), tiltX), tiltZ)
	return sdTorus(o, R, r)
}

func bandPattern(p V, alb V, y0, y1 float64, col V) V {
	if p.Y > y0 && p.Y < y1 {
		return col
	}
	return alb
}

// cowl is a deep hood with a pointed-arch opening and a drooping point.
func cowl(q, c V, droop float64) (shell, void, brim float64) { return cowlW(q, c, droop, 1) }

func cowlW(q, c V, droop, ow float64) (shell, void, brim float64) {
	o := q.Sub(c)
	outer := sdEllipsoid(o.Sub(V{0, 0.02, -0.025}), V{0.118, 0.14, 0.142})
	tip := chain(o, []V{{0, 0.08, -0.06}, {0, 0.1, -0.13}, {0, 0.06 - droop*0.3, -0.19 - droop*0.5}, {0, -0.02 - droop, -0.22 - droop*0.6}}, 0.075, 0.008)
	outer = smin(outer, tip, 0.045)
	drape := sdCone(o, V{0, -0.06, -0.035}, V{0, -0.2, -0.03}, 0.11, 0.17)
	outer = smin(outer, drape, 0.05)
	a := math.Atan2(o.X, o.Z)
	outer += 0.005 * (1 - math.Abs(math.Sin(a*4+o.Y*14)))
	inner := sdEllipsoid(o.Sub(V{0, 0.0, -0.005}), V{0.1, 0.122, 0.128})
	shell = smax(outer, -inner, 0.006)
	// pointed-arch opening: intersection of two offset ellipses
	oa := o.Sub(V{0, -0.02, 0.12})
	e1 := sdEllipsoid(oa.Sub(V{0.022 * ow, 0, 0}), V{0.07 * ow, 0.11, 0.09})
	e2 := sdEllipsoid(oa.Sub(V{-0.022 * ow, 0, 0}), V{0.07 * ow, 0.11, 0.09})
	open := smax(e1, e2, 0.005)
	shell = smax(shell, -open, 0.006)
	brim = 1e9
	void = sdEllipsoid(o.Sub(V{0, -0.005, 0.0}), V{0.09, 0.11, 0.105})
	return
}

// capelet is a short shoulder cape with a scalloped hem.
func capelet(q V, top, bot float64, rBot, sq float64, n float64) float64 {
	c := V{q.X, q.Y, q.Z * sq}
	d := sdCone(c, V{0, top, -0.01}, V{0, bot, 0.0}, 0.075, rBot)
	a := math.Atan2(c.X, c.Z)
	d += folds(c, n, 0.008, top-0.05, bot+0.02)
	hem := bot + 0.03 + 0.018*math.Abs(math.Sin(a*n*0.5))
	d = smax(d, hem-q.Y, 0.006)
	return d * 0.8
}

// mask is a smooth sculpted face mask with eye slits; returns the mask and
// the glowing eye volume behind the slits.
func mask(q, c V) (m, eyes float64) {
	o := q.Sub(c)
	m = sdEllipsoid(o.Sub(V{0, -0.012, 0.035}), V{0.06, 0.086, 0.058})
	m = smin(m, sdEllipsoid(o.Sub(V{0, 0.022, 0.07}), V{0.054, 0.016, 0.025}), 0.015)
	m = smin(m, sdCone(o, V{0, 0.016, 0.085}, V{0, -0.026, 0.098}, 0.007, 0.013), 0.012)
	m = smin(m, sdEllipsoid(o.Sub(V{0, -0.07, 0.065}), V{0.03, 0.018, 0.02}), 0.02)
	eyes = 1e9
	for _, sx := range []float64{-0.025, 0.025} {
		slit := sdEllipsoid(rotZ(o.Sub(V{sx, 0.004, 0.09}), -sx*8), V{0.017, 0.0045, 0.04})
		m = smax(m, -slit, 0.003)
		eyes = math.Min(eyes, sdEllipsoid(o.Sub(V{sx, 0.004, 0.072}), V{0.014, 0.006, 0.006}))
	}
	return
}

// pauldron is a layered shoulder plate centred over shoulder s (side = ±1).
func pauldron(q, s V, side, size float64) float64 {
	o := q.Sub(s)
	o.X *= side
	o = rotZ(o, -0.35)
	d := 1e9
	for i := 0; i < 3; i++ {
		f := float64(i)
		c := V{0.01 * f, -0.045 * f, 0}
		r := V{0.12, 0.085, 0.11}.Mul(size * (1 - 0.06*f))
		sh := math.Abs(sdEllipsoid(o.Sub(c), r)) - 0.008*size
		sh = smax(sh, -(o.Y - c.Y + 0.02*size), 0.004) // open underneath
		d = math.Min(d, sh)
	}
	return d
}

// layer returns a thin shell hugging surface distance d at offset off.
func layer(d, off, thick float64) float64 { return math.Abs(d-off) - thick }

// band limits a distance to a horizontal band.
func bandY(d float64, q V, y0, y1 float64) float64 {
	return smax(d, math.Max(y0-q.Y, q.Y-y1), 0.004)
}

// Pose holds a character's yaw and head attitude.
type Pose struct {
	Yaw, Roll, Nod, Turn float64
	Neck                 V
}

// local maps a world sample point into the character's rest space.
func (z Pose) local(p V) V {
	q := rotY(p, -z.Yaw)
	if z.Roll != 0 || z.Nod != 0 || z.Turn != 0 {
		q = headTilt(q, z.Neck, z.Roll, z.Nod, z.Turn)
	}
	return q
}

// head maps a rest-space point on the head into world space.
func (z Pose) head(e V) V {
	o := e.Sub(z.Neck)
	o = rotZ(o, z.Roll)
	o = rotX(o, z.Nod)
	o = rotY(o, z.Turn)
	return rotY(o.Add(z.Neck), z.Yaw)
}

// body maps a rest-space body point into world space.
func (z Pose) body(e V) V { return rotY(e, z.Yaw) }
