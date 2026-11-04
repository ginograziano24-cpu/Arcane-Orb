package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Text image cache. text.Draw lays a string out and draws every glyph as
// its own image: a dozen draw calls for a short label, nine times that for
// outlined text, each walking Ebitengine's image list, plus the layout's
// allocations. The benchmark profile put text at an eighth of all CPU time
// and over half of all allocation. A string seen twice is drawn once into
// a small image of its own (glyphs at their whole-pixel layout offsets);
// later draws are a single image draw with the same colour scale and
// origin, so the result is the same pixels. Strings that change every frame
// (timers, counters) never get past one sighting and draw as before;
// entries unused for a few seconds are freed.

type textKey struct {
	s    string
	face text.Face
	al   text.Align
}

type textEntry struct {
	img    *ebiten.Image // nil until seen twice, or if the string has no ink
	ox, oy float64       // where the image's corner sits from the draw origin
	seen   int
	last   int
	built  bool
}

var (
	textCache = map[textKey]*textEntry{}
	textFrame int
	textOff   bool // benchmarks and tests: draw every string directly
)

const (
	textKeepFrames = 240  // unused this long: freed
	textMaxEntries = 3000 // past this, a sweep keeps only the last second
)

// drawTextFace draws s at (sx, sy) with the given alignment and colour,
// through the cache.
func drawTextFace(dst *ebiten.Image, s string, face text.Face, al text.Align, sx, sy float64, cs ebiten.ColorScale) {
	if s == "" {
		return
	}
	if textOff {
		drawTextDirect(dst, s, face, al, sx, sy, cs)
		return
	}
	k := textKey{s, face, al}
	e := textCache[k]
	if e == nil {
		e = &textEntry{}
		textCache[k] = e
	}
	e.last = textFrame
	e.seen++
	if !e.built {
		if e.seen < 2 {
			drawTextDirect(dst, s, face, al, sx, sy, cs)
			return
		}
		e.build(s, face, al)
	}
	if e.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{ColorScale: cs}
	op.GeoM.Translate(e.ox+sx, e.oy+sy)
	dst.DrawImage(e.img, op)
}

func drawTextDirect(dst *ebiten.Image, s string, face text.Face, al text.Align, sx, sy float64, cs ebiten.ColorScale) {
	op := &text.DrawOptions{}
	op.PrimaryAlign = al
	op.GeoM.Translate(sx, sy)
	op.ColorScale = cs
	text.Draw(dst, s, face, op)
}

// build renders the string's glyphs, uncoloured, into the entry's image.
func (e *textEntry) build(s string, face text.Face, al text.Align) {
	e.built = true
	var lo text.LayoutOptions
	lo.PrimaryAlign = al
	gs := text.AppendGlyphs(nil, s, face, &lo)
	first := true
	var x0, y0, x1, y1 float64
	for _, g := range gs {
		if g.Image == nil {
			continue
		}
		b := g.Image.Bounds()
		gx0, gy0 := g.X, g.Y
		gx1, gy1 := gx0+float64(b.Dx()), gy0+float64(b.Dy())
		if first {
			x0, y0, x1, y1, first = gx0, gy0, gx1, gy1, false
			continue
		}
		x0, y0 = min(x0, gx0), min(y0, gy0)
		x1, y1 = max(x1, gx1), max(y1, gy1)
	}
	if first {
		return // nothing to draw (spaces)
	}
	w, h := int(x1-x0), int(y1-y0)
	if w <= 0 || h <= 0 || w > 4096 || h > 1024 {
		e.built = false // too big to be worth it: keep drawing directly
		e.seen = -1 << 30
		return
	}
	img := ebiten.NewImage(w, h)
	for _, g := range gs {
		if g.Image == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.X-x0, g.Y-y0)
		img.DrawImage(g.Image, op)
	}
	e.img, e.ox, e.oy = img, x0, y0
}

// textFrameEnd ages the cache; call once per drawn frame.
func textFrameEnd() {
	textFrame++
	if textFrame%60 != 0 && len(textCache) <= textMaxEntries {
		return
	}
	keep := textKeepFrames
	if len(textCache) > textMaxEntries {
		keep = 60
	}
	for k, e := range textCache {
		if textFrame-e.last > keep {
			if e.img != nil {
				e.img.Deallocate()
			}
			delete(textCache, k)
		}
	}
}
