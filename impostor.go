package main

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Impostors: a plant far enough away to be under ~70 pixels tall is drawn
// as one camera-facing card showing a picture of it, instead of its
// thousands of triangles. Each plant variant is photographed from eight
// directions around it (a little from above, like the playing camera) into
// one strip texture, lit by the world's key light. Pictures are taken on
// the main goroutine at the start of a frame (a couple per frame); until a
// plant's is ready it is drawn in full.

const (
	impViews = 8
	impCell  = 160 // pixels per view
)

type impostor struct {
	tex        uint8
	w, h       float32
	ymin, ymax float64 // the plant's height span (unscaled)
	half       float64 // half the card's width (unscaled)
}

var (
	impMu    sync.Mutex
	impDone  = map[*floraSet]*impostor{}
	impQueue []*floraSet
	impWant  = map[*floraSet]bool{}
	impWorld = -1
	impStage struct {
		cam   Camera
		batch *Batch
	}
)

// impostorFor returns the plant's picture card, or queues it to be taken.
func impostorFor(s *floraSet) *impostor {
	impMu.Lock()
	defer impMu.Unlock()
	if im, ok := impDone[s]; ok {
		return im
	}
	if !impWant[s] {
		impWant[s] = true
		impQueue = append(impQueue, s)
	}
	return nil
}

// bakeImpostors takes up to n queued pictures. Call on the main goroutine
// before the frame's drawing jobs start; world is the world the light is
// for (pictures are retaken when it changes).
func bakeImpostors(n int, world int, light V3, lightCol, ambCol [3]float32) {
	impMu.Lock()
	if world != impWorld {
		// a new world, a new light: retake everything (old textures stay
		// registered; there are only a handful per world)
		impWorld = world
		impDone = map[*floraSet]*impostor{}
		impWant = map[*floraSet]bool{}
		impQueue = impQueue[:0]
	}
	var todo []*floraSet
	for len(todo) < n && len(impQueue) > 0 {
		todo = append(todo, impQueue[0])
		impQueue = impQueue[1:]
	}
	impMu.Unlock()
	for _, s := range todo {
		im := takeImpostor(s, light, lightCol, ambCol)
		impMu.Lock()
		impDone[s] = im
		impMu.Unlock()
	}
}

func plantBounds(s *floraSet) (ymin, ymax, half float64) {
	ymin, ymax = math.Inf(1), math.Inf(-1)
	for _, m := range []*Mesh{s.wood, s.leaf} {
		for _, t := range m.Tris {
			for _, p := range t.P {
				ymin, ymax = math.Min(ymin, p.Y), math.Max(ymax, p.Y)
				half = math.Max(half, math.Hypot(p.X, p.Z))
			}
		}
	}
	return
}

func takeImpostor(s *floraSet, light V3, lightCol, ambCol [3]float32) *impostor {
	ymin, ymax, half := plantBounds(s)
	if math.IsInf(ymin, 0) {
		return nil
	}
	hgt := ymax - ymin
	side := math.Max(hgt, 2*half) * 1.06 // a square cell holds the plant whichever way it turns
	img := ebiten.NewImage(impCell*impViews, impCell)
	tmp := ebiten.NewImage(impCell, impCell)
	defer tmp.Deallocate()
	st := &impStage
	if st.batch == nil {
		st.batch = newBatch(&st.cam)
	}
	own := aaOwn
	aaOwn = false
	defer func() { aaOwn = own }()
	mid := (ymin + ymax) / 2
	dist := 80.0
	for k := 0; k < impViews; k++ {
		a := float64(k) / impViews * 2 * math.Pi
		dir := V3{math.Sin(a), 0.27, math.Cos(a)}.Norm()
		st.cam = Camera{Eye: V3{0, mid, 0}.Add(dir.Mul(dist)), Target: V3{0, mid, 0}, FOV: 2 * math.Atan(side/2/dist)}
		st.cam.Setup(impCell, impCell)
		b := st.batch
		b.cam = &st.cam
		b.back = nil
		b.Reset()
		b.fogNear, b.fogFar = 1e6, 2e6
		b.SetLight(light, lightCol, ambCol)
		if len(s.wood.Tris) > 0 {
			b.drawShared(s.wood, V3{}, v3(1), identity, color.RGBA{}, 0)
		}
		if len(s.leaf.Tris) > 0 {
			b.drawShared(s.leaf, V3{}, v3(1), identity, color.RGBA{}, 0)
		}
		tmp.Clear()
		b.Flush(tmp)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(k*impCell), 0)
		img.DrawImage(tmp, op)
	}
	return &impostor{tex: registerTex(img), w: float32(impCell * impViews), h: impCell, ymin: mid - side/2, ymax: mid + side/2, half: side / 2}
}

// drawImpostor draws the card for a plant at pos, turned ry, scaled.
func drawImpostor(B *Batch, im *impostor, pos, scale V3, ry float64, tint color.RGBA) {
	cam := B.cam
	v := cam.Eye.Sub(pos)
	v.Y = 0
	if v.Len() < 1e-6 {
		return
	}
	// the view direction in the plant's own frame picks the picture
	r := rotXYZ(0, ry, 0)
	lv := V3{r[0].X*v.X + r[1].X*v.Y + r[2].X*v.Z, r[0].Y*v.X + r[1].Y*v.Y + r[2].Y*v.Z, r[0].Z*v.X + r[1].Z*v.Y + r[2].Z*v.Z}
	a := math.Atan2(lv.X, lv.Z)
	k := int(math.Round(a/(2*math.Pi)*impViews)) % impViews
	if k < 0 {
		k += impViews
	}
	right := V3{cam.right.X, 0, cam.right.Z}
	if right.Len() < 1e-6 {
		right = V3{1, 0, 0}
	}
	right = right.Norm().Mul(im.half * scale.X)
	y0 := pos.Y + im.ymin*scale.Y
	y1 := pos.Y + im.ymax*scale.Y
	c := V3{pos.X, 0, pos.Z}
	p := [4]V3{c.Sub(right).Add(V3{0, y1, 0}), c.Add(right).Add(V3{0, y1, 0}), c.Add(right).Add(V3{0, y0, 0}), c.Sub(right).Add(V3{0, y0, 0})}
	u0 := float32(k*impCell) + 0.5
	u1 := float32((k+1)*impCell) - 0.5
	uv := [4][2]float32{{u0, 0.5}, {u1, 0.5}, {u1, impCell - 0.5}, {u0, impCell - 0.5}}
	col := color.RGBA{255, 255, 255, 255}
	if tint.A > 0 {
		col = tint
	}
	B.QuadUV(p, im.tex, uv, col, [4]float32{1, 1, 1, 1}, false)
}

// ---------------------------------------------------------------- leaf cards
//
// Mid-distance broadleaf trees draw their canopy as leaf cards: each cluster
// of leaves is two crossed quads showing a picture of a real leaf spray
// (white, so the card's colour tints it), instead of its dozens of leaf
// triangles. The spray is drawn once from the same folded leaves the full
// trees use.

const cardTexSize = 256

var (
	cardTex    uint8
	cardWanted bool
)

// leafCardTex returns the spray texture, or 0 (and asks for it) if it is not
// drawn yet.
func leafCardTex() uint8 {
	impMu.Lock()
	defer impMu.Unlock()
	if cardTex == 0 {
		cardWanted = true
	}
	return cardTex
}

// bakeLeafCard draws the spray if it was asked for. Main goroutine only.
func bakeLeafCard() {
	impMu.Lock()
	want := cardWanted && cardTex == 0
	impMu.Unlock()
	if !want {
		return
	}
	fb := newFloraB(4242)
	pal := leafPal{dark: color.RGBA{150, 150, 150, 255}, light: color.RGBA{255, 255, 255, 255}, under: color.RGBA{120, 120, 120, 255}}
	for k := 0; k < 30; k++ {
		a := fb.f(0, 2*math.Pi)
		r := math.Sqrt(fb.r.Float64()) * 0.75
		p := V3{math.Cos(a) * r, math.Sin(a) * r, fb.f(-0.1, 0.1)}
		dir := V3{math.Cos(a + fb.f(-0.6, 0.6)), math.Sin(a + fb.f(-0.6, 0.6)), 0}.Norm()
		col := fb.leafCol(pal, fb.r.Float64())
		fb.leaf(p.Sub(dir.Mul(0.12)), dir, V3{fb.f(-0.3, 0.3), fb.f(-0.3, 0.3), -1}.Norm(), 0.42, 0.25, 0.15, col, pal.under, true)
	}
	fb.m.topology()
	img := ebiten.NewImage(cardTexSize, cardTexSize)
	st := &impStage
	if st.batch == nil {
		st.batch = newBatch(&st.cam)
	}
	own := aaOwn
	aaOwn = false
	st.cam = Camera{Eye: V3{0, 0, -30}, Target: V3{}, FOV: 2 * math.Atan(1.0/30)}
	st.cam.Setup(cardTexSize, cardTexSize)
	b := st.batch
	b.cam = &st.cam
	b.back = nil
	b.Reset()
	b.fogNear, b.fogFar = 1e6, 2e6
	b.SetLight(V3{-0.3, 0.6, -0.75}.Norm(), [3]float32{0.75, 0.75, 0.75}, [3]float32{0.45, 0.45, 0.45})
	b.drawShared(fb.m, V3{}, v3(1), identity, color.RGBA{}, 0)
	b.Flush(img)
	aaOwn = own
	impMu.Lock()
	cardTex = registerTex(img)
	impMu.Unlock()
}

type leafCard struct {
	c   V3
	n   V3 // facing (outward from the crown)
	r   float64
	col color.RGBA
}

// cardsFromLeaves groups a canopy's leaf triangles into cells about one
// cluster across and makes a card per cell: at the cell's centre, as big as
// its spread, in its average colour, facing out from the crown.
func cardsFromLeaves(m *Mesh, crown V3) []leafCard {
	type acc struct {
		sum     V3
		n       int
		r, g, b float64
		pts     []V3
	}
	cells := map[[3]int]*acc{}
	const cs = 1.1
	for _, t := range m.Tris {
		c := t.P[0].Add(t.P[1]).Add(t.P[2]).Mul(1.0 / 3)
		k := [3]int{int(math.Floor(c.X / cs)), int(math.Floor(c.Y / cs)), int(math.Floor(c.Z / cs))}
		a := cells[k]
		if a == nil {
			a = &acc{}
			cells[k] = a
		}
		a.sum = a.sum.Add(c)
		a.n++
		a.r += float64(t.Col.R)
		a.g += float64(t.Col.G)
		a.b += float64(t.Col.B)
		if len(a.pts) < 64 {
			a.pts = append(a.pts, c)
		}
	}
	var out []leafCard
	for _, a := range cells {
		if a.n < 3 {
			continue
		}
		c := a.sum.Mul(1 / float64(a.n))
		r := 0.0
		for _, p := range a.pts {
			r = math.Max(r, p.Sub(c).Len())
		}
		n := c.Sub(crown)
		if n.Len() < 1e-3 {
			n = V3{0, 1, 0}
		}
		f := 1 / float64(a.n)
		out = append(out, leafCard{c: c, n: n.Norm(), r: math.Max(0.45, r*1.25),
			col: color.RGBA{uint8(a.r * f), uint8(a.g * f), uint8(a.b * f), 255}})
	}
	return out
}

// drawCards draws a canopy's cards: two crossed quads each, lit by their
// facing.
func drawCards(B *Batch, cards []leafCard, tex uint8, pos, scale V3, rot Mat3) {
	s := float32(cardTexSize)
	uv := [4][2]float32{{0.5, 0.5}, {s - 0.5, 0.5}, {s - 0.5, s - 0.5}, {0.5, s - 0.5}}
	tr := func(p V3) V3 { return rot.Mul(V3{p.X * scale.X, p.Y * scale.Y, p.Z * scale.Z}).Add(pos) }
	for i := range cards {
		cd := &cards[i]
		n := rot.Mul(cd.n)
		u := perpTo(n)
		v := n.Cross(u)
		r := cd.r
		col := B.lit(cd.col, n)
		for _, ax := range [2][2]V3{{u, v}, {u.Add(n).Norm(), v}} {
			a, b := ax[0].Mul(r), ax[1].Mul(r)
			c := tr(cd.c)
			p := [4]V3{c.Sub(a).Add(b), c.Add(a).Add(b), c.Add(a).Sub(b), c.Sub(a).Sub(b)}
			B.QuadUV(p, tex, uv, col, [4]float32{1, 1, 1, 1}, false)
		}
	}
}
