package main

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Image prefetching. Decoding a baked floor or a painted backdrop takes
// around 100 ms, and the draw code asks for each one on the first frame
// that shows it, so a new world used to open with a stall. Images are now
// built (decoded, composed, uploaded) by whichever goroutine asks first:
// a background worker fed ahead of need (a run's next act, the menus'
// portraits), or the loader itself if it gets there before the worker.
// Either way each is built once; a loader that arrives while the worker is
// building waits for that build rather than starting its own.
//
// Only the building moves. Where a loader registers its image (the
// texture ids, whose order matters to the draw order) is unchanged.

type preImg struct {
	once sync.Once
	img  *ebiten.Image
	w, h int
	avg  color.RGBA // mean colour (baked textures use it for distant ground)
	aux  int        // builder-specific (the brick sheet: diffuse height)
}

var (
	preMu   sync.Mutex
	preImgs = map[string]*preImg{}
	preQ    chan string
)

func preEntry(key string) *preImg {
	preMu.Lock()
	defer preMu.Unlock()
	e := preImgs[key]
	if e == nil {
		e = &preImg{}
		preImgs[key] = e
	}
	return e
}

// loadImage returns the built image for key ("floor/meadow", "bg/neon_1",
// ...), building it here unless it is built or being built already. Its
// img is nil if the asset doesn't exist.
func loadImage(key string) *preImg {
	e := preEntry(key)
	e.once.Do(func() {
		src, aux := buildImage(key)
		if src == nil {
			return
		}
		b := src.Bounds()
		e.w, e.h, e.aux = b.Dx(), b.Dy(), aux
		e.avg = meanColour(src)
		e.img = ebiten.NewImageFromImage(src)
	})
	return e
}

// buildImage decodes (and for some, composes) the CPU image behind a key.
func buildImage(key string) (image.Image, int) {
	kind, name, _ := strings.Cut(key, "/")
	switch kind {
	case "floor":
		return decodeEmbedded(floorFS.ReadFile("assets/floor/" + name + ".jpg")), 0
	case "bricks":
		return composeBrickSheet()
	case "bg":
		return decodeEmbedded(backdropFS.ReadFile("assets/bg/" + name + ".jpg")), 0
	case "portrait":
		return composePortrait(name), 0
	case "hud":
		return decodeEmbedded(portraitFS.ReadFile("assets/portraits/hud/" + name + ".png")), 0
	case "scene":
		return decodeEmbedded(sceneFS.ReadFile("assets/scenes/" + name + ".jpg")), 0
	}
	return nil, 0
}

func decodeEmbedded(data []byte, err error) image.Image {
	if err != nil {
		return nil
	}
	im, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return im
}

// prefetch queues images for the background worker (started on first use).
// It never blocks: a full queue drops the rest, which then simply load
// when first drawn.
func prefetch(keys ...string) {
	preMu.Lock()
	if preQ == nil {
		preQ = make(chan string, 256)
		go func() {
			for k := range preQ {
				loadImage(k)
			}
		}()
	}
	q := preQ
	preMu.Unlock()
	for _, k := range keys {
		select {
		case q <- k:
		default:
			return
		}
	}
}

// prefetchWorld queues what a level in world w draws first: its floor (all
// four neon patterns), the ground around it and both backdrop paintings.
func prefetchWorld(w int) {
	var keys []string
	if n, ok := floorTexNames[w]; ok {
		keys = append(keys, "floor/"+n, "floor/ground_"+n)
	} else if worlds[w].Neon {
		keys = append(keys, "floor/neon_0", "floor/neon_1", "floor/neon_2", "floor/neon_3")
	}
	if n, ok := backdropNames[w]; ok {
		keys = append(keys, "bg/"+n+"_0", "bg/"+n+"_1")
	}
	prefetch(keys...)
}

// prefetchAct queues an act's worlds (its boss's first) and the brick sheet.
func prefetchAct(a int) {
	if a < 0 || a >= len(acts) {
		return
	}
	prefetch("bricks/sheet")
	prefetchWorld(acts[a].Signature)
	for _, w := range acts[a].Pool {
		if w != acts[a].Signature {
			prefetchWorld(w)
		}
	}
}

// prefetchTick queues images ahead of need: at start the origins'
// portraits and the set-piece scenes, and in a run its origin's HUD
// portraits and the worlds of the act it is in and the next.
func (g *Game) prefetchTick() {
	if !g.preBoot {
		g.preBoot = true
		for id := range classStyle {
			prefetch("portrait/" + id)
		}
		prefetch("scene/rest", "scene/shop", "scene/treasure", "scene/event")
		go prefetchModels()
	}
	r := g.run
	if r == nil || r.Arch == nil {
		return
	}
	key := r.Arch.ID + "/" + itoa(r.AI())
	if key == g.preKey {
		return
	}
	g.preKey = key
	for _, pose := range heroPoses {
		prefetch("hud/" + r.Arch.ID + "_" + pose)
	}
	prefetchAct(r.AI())
	prefetchAct(r.AI() + 1)
}

// prefetchModels parses every Blender model (and works out its shared
// corners) ahead of the first frame that draws it: each is small, but the
// first frame of a world used to parse a dozen of them under the asset
// lock, one side job at a time.
func prefetchModels() {
	ents, err := modelFS.ReadDir("assets/models")
	if err != nil {
		return
	}
	for _, e := range ents {
		if name, ok := strings.CutSuffix(e.Name(), ".orbm"); ok {
			model(name)
		}
	}
}
