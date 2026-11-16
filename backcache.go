package main

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The background cache. Everything beyond the arena (hills, groves, far
// props, the far half of the scenery) changes slowly, but it was being
// projected, sorted and submitted every frame. The painter's sort draws far
// before near, so the frame can be cut by depth without changing it:
//
//	bottom  depth >= 5e8: the ground and the dream sun (always drawn live)
//	back    past the arena, or pushed beyond 3900: drawn into cached images
//	front   the arena and everything near it (always drawn live)
//
// The back layer is redrawn about 30 times a second (its animation runs at
// that rate), on any large camera move, or while the camera rolls. Between
// redraws the cached image is shifted by how far the camera's drift moves a
// point at the depth of the scenery, so it stays put as the camera follows
// the paddle. Additive glows keep their place, composited after everything
// as before, so a refresh frame is the same picture as drawing it all.

type backCache struct {
	on, refresh bool
	split       float64 // camera depth past which a quad is "back"
	img, glow   *ebiten.Image
	valid       bool
	eye, fwd    V3
	fov, w, h   float64
	aa          float64 // the supersample factor the images were drawn at
	hasGlow     bool
	rect        image.Rectangle // the region of the images in use
	worth       bool            // the last measure found enough background to pay for the cache
	measured    time.Time       // when that was
	ref         V3              // a point straight ahead at scenery depth, at the last refresh
	at          time.Time
	level       *Level
	world       int
	shiftX      float64
	shiftY      float64
	// partition scratch
	bot, back, front, gbot, gback, gfront []*quad
}

// worldBack is the world batch's background cache.
var worldBack backCache

// backCacheOn lets the bench compare (-nobackcache).
var backCacheOn = true

const backRefresh = time.Second / 30

// backWorth is how many background quads make caching them pay for the
// full-screen composite.
const backWorth = 9000

// class sorts a quad depth into 0 bottom, 1 back, 2 front.
func (c *backCache) class(d float64) int {
	switch {
	case d >= 5e8:
		return 0
	case (d > c.split && d < 900) || d >= 3900:
		return 1
	}
	return 2
}

// begin decides, before the frame is drawn, whether the back layer is
// redrawn this frame (all of it is drawn) or reused (back geometry may be
// skipped).
func (c *backCache) begin(g *Game, L *Level, cam *Camera) {
	c.on = backCacheOn && !g.settings.Classic && L.Boss == nil && L.introT <= 0 && L.finaleT <= 0 && g.photo == nil
	if c.on && c.level == L && !c.measured.IsZero() && !c.worth && time.Since(c.measured) < 2*time.Second {
		// a light background: drawing it straight is cheaper than caching
		// it; look again in a couple of seconds
		c.on = false
	}
	if !c.on {
		c.valid = false
		return
	}
	// the arena's far edge, at its farthest from the camera, plus a margin
	far := 0.0
	for _, x := range []float64{-HalfW - 2, HalfW + 2} {
		for _, y := range []float64{-1, 6} {
			p := V3{x, y, FarZ + 3}
			far = math.Max(far, p.Sub(cam.Eye).Dot(cam.fwd))
		}
	}
	c.split = far + 5
	now := time.Now()
	c.refresh = !c.valid || now.Sub(c.at) >= backRefresh || cam.Roll != 0 ||
		c.level != L || c.world != L.Biome.ID || c.w != cam.W || c.h != cam.H || c.fov != cam.FOV || c.aa != aaScale ||
		cam.Eye.Sub(c.eye).Len() > 0.8 || cam.fwd.Dot(c.fwd) < 0.9995
	if c.refresh {
		c.eye, c.fwd, c.fov, c.w, c.h, c.aa = cam.Eye, cam.fwd, cam.FOV, cam.W, cam.H, aaScale
		c.ref = cam.Eye.Add(cam.fwd.Mul(70))
		c.at, c.level, c.world = now, L, L.Biome.ID
		c.shiftX, c.shiftY = 0, 0
		return
	}
	sx, sy, _ := cam.Project(c.ref)
	c.shiftX, c.shiftY = sx-cam.W/2, sy-cam.H/2
}

// skipBack reports whether geometry wholly in the back layer can be left out
// of this frame (the cached image already has it). z is its depth along the
// camera axis, r its radius.
func (b *Batch) skipBack(z, r float64) bool {
	c := b.back
	if c == nil || !c.on || c.refresh {
		return false
	}
	return z-r > c.split && z+r < 900
}

func partitionQuads(c *backCache, segs [][]quad, bot, back, front []*quad) ([]*quad, []*quad, []*quad) {
	bot, back, front = bot[:0], back[:0], front[:0]
	for _, s := range segs {
		for i := range s {
			q := &s[i]
			switch c.class(q.depth) {
			case 0:
				bot = append(bot, q)
			case 1:
				back = append(back, q)
			default:
				front = append(front, q)
			}
		}
	}
	return bot, back, front
}

// planPtrs is planList for a ready list of quad pointers: sort far first
// and group into runs (always planned).
func (b *Batch) planPtrs(qs []*quad, list int) []quadRun {
	if cap(b.qtex) < len(qs) {
		b.qtex = make([]uint8, len(qs), len(qs)+len(qs)/4)
	}
	b.qtex = b.qtex[:len(qs)]
	order := b.sortFarFirst(qs, true)
	runs, switches := b.planRuns(qs, order, true)
	b.switches[list] = switches
	return runs
}

// flushBack is Flush with the background cache in play.
func (b *Batch) flushBack(dst *ebiten.Image) {
	c := b.back
	segQ := append(b.extQ, b.quads)
	segG := append(b.extG, b.glow)
	c.bot, c.back, c.front = partitionQuads(c, segQ, c.bot, c.back, c.front)
	c.gbot, c.gback, c.gfront = partitionQuads(c, segG, c.gbot, c.gback, c.gfront)
	lastQuads = len(c.bot) + len(c.front) + len(c.gbot) + len(c.gfront)
	b.prepMeta()
	tgt, sc := dst, float32(1)
	own := worldAA && aaOwn
	if own {
		tgt, sc = aaBegin(dst), float32(aaScale)
	}
	extB := b.extB
	b.extB = nil // the lists are re-cut: plan them with a plain sort
	draw := func(img *ebiten.Image, qs []*quad, list int, blend ebiten.Blend) {
		if len(qs) == 0 {
			return
		}
		runs := b.planPtrs(qs, list)
		b.drawRuns(img, qs, runs, blend, sc)
	}
	tw, th := tgt.Bounds().Dx(), tgt.Bounds().Dy()
	if !c.refresh && c.valid && (c.img == nil || c.img.Bounds().Dx() != tw || c.img.Bounds().Dy() != th) {
		c.valid = false // stale size: skip it this frame, the next one refreshes
	}
	if c.refresh {
		for _, im := range []**ebiten.Image{&c.img, &c.glow} {
			if *im == nil || (*im).Bounds().Dx() != tw || (*im).Bounds().Dy() != th {
				if *im != nil {
					(*im).Deallocate()
				}
				*im = ebiten.NewImage(tw, th)
			}
		}
		// only the screen region the background covers is drawn, cleared
		// and composited (usually the top part of the screen)
		x0, y0, x1, y1 := float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.Inf(-1))
		for _, l := range [][]*quad{c.back, c.gback} {
			for _, q := range l {
				for k := range q.v {
					v := &q.v[k]
					x0, x1 = min(x0, v.DstX), max(x1, v.DstX)
					y0, y1 = min(y0, v.DstY), max(y1, v.DstY)
				}
			}
		}
		r := image.Rect(0, 0, 0, 0)
		if x1 >= x0 {
			r = image.Rect(int(float32(math.Floor(float64(x0*sc))))-2, int(float32(math.Floor(float64(y0*sc))))-2,
				int(float32(math.Ceil(float64(x1*sc))))+2, int(float32(math.Ceil(float64(y1*sc))))+2).Intersect(image.Rect(0, 0, tw, th))
		}
		c.rect = r
		if !r.Empty() {
			sub := c.img.SubImage(r).(*ebiten.Image)
			sub.Clear()
			draw(sub, c.back, 0, ebiten.BlendSourceOver)
			c.hasGlow = len(c.gback) > 0
			if c.hasGlow {
				gs := c.glow.SubImage(r).(*ebiten.Image)
				gs.Clear()
				draw(gs, c.gback, 1, ebiten.BlendLighter)
			}
		}
		lastQuads += len(c.back) + len(c.gback)
		c.valid = !r.Empty()
		// is the cache worth its composites here?
		c.worth = len(c.back)+len(c.gback) >= backWorth
		c.measured = time.Now()
	}
	comp := func(img *ebiten.Image, blend ebiten.Blend) {
		op := &ebiten.DrawImageOptions{Blend: blend}
		op.GeoM.Translate(float64(c.rect.Min.X)+c.shiftX*float64(sc), float64(c.rect.Min.Y)+c.shiftY*float64(sc))
		if c.shiftX != 0 || c.shiftY != 0 {
			op.Filter = ebiten.FilterLinear
		}
		tgt.DrawImage(img.SubImage(c.rect).(*ebiten.Image), op)
	}
	draw(tgt, c.bot, 0, ebiten.BlendSourceOver)
	if c.valid {
		comp(c.img, ebiten.BlendSourceOver)
	}
	draw(tgt, c.front, 0, ebiten.BlendSourceOver)
	draw(tgt, c.gbot, 1, ebiten.BlendLighter)
	if c.valid && c.hasGlow {
		comp(c.glow, ebiten.BlendLighter)
	}
	draw(tgt, c.gfront, 1, ebiten.BlendLighter)
	b.extB = extB
	if own {
		aaEnd(dst)
	}
	b.extQ, b.extG, b.extB = b.extQ[:0], b.extG[:0], b.extB[:0]
}
