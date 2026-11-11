package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"runtime/metrics"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Frame profiling: section timers, draw counters and allocation rates,
// shown by the F3 overlay and recorded by the benchmark tour (bench.go).
// Everything here is cheap enough to leave on in release builds.

const (
	secUpdate = iota
	secWorld  // building the 3D scene (drawWorld minus the flush)
	secFlush  // sorting and submitting the 3D quads
	secPost   // bloom, grade, upscale
	secHUD    // HUD and screen UI
	secDraw   // the whole Draw call
	numSecs
)

var secNames = [numSecs]string{"update", "world", "flush", "post", "hud", "draw"}

type perfFrame struct {
	frame                  time.Duration // wall time since the previous frame
	updateSeconds          float64       // elapsed time delivered to game updates during a benchmark
	discardedUpdateSeconds float64
	updateCount            int
	sec                    [numSecs]time.Duration
	quads                  int
	world                  int // 3D draw calls (DrawTriangles runs in Batch.flush)
	sdf                    int // SDF shape draws
	sprite                 int // drawSprite calls
	text                   int // text draws
	rect                   int // vector rect fills
	allocs                 uint64
	bytes                  uint64
}

type perfState struct {
	cur, last perfFrame
	hist      []perfFrame // ring of recent frames for the overlay
	hi        int
	t0        [numSecs]time.Time
	lastFrame time.Time
	show      bool
	samples   []metrics.Sample
	allocs0   uint64
	bytes0    uint64
}

var perf = &perfState{hist: make([]perfFrame, 240)}

// perfMarks counts the 3D quads by part of drawWorld: scenery, scenery plus
// arena (floor, walls, posts), and the rest (bricks, orbs, effects).
var perfMarks [3]int

func perfMark(i, n int) { perfMarks[i] = n }

// perfScreen is the size of the last frame drawn, in device pixels.
var perfScreen image.Point

func init() {
	perf.samples = []metrics.Sample{{Name: "/gc/heap/allocs:objects"}, {Name: "/gc/heap/allocs:bytes"}}
}

func perfBegin(s int) { perf.t0[s] = time.Now() }
func perfEnd(s int)   { perf.cur.sec[s] += time.Since(perf.t0[s]) }

// perfFrameEnd closes the frame's record; call once per Draw.
func perfFrameEnd() {
	now := time.Now()
	p := perf
	if !p.lastFrame.IsZero() {
		p.cur.frame = now.Sub(p.lastFrame)
	}
	p.lastFrame = now
	metrics.Read(p.samples)
	a, b := p.samples[0].Value.Uint64(), p.samples[1].Value.Uint64()
	if p.allocs0 != 0 {
		p.cur.allocs, p.cur.bytes = a-p.allocs0, b-p.bytes0
	}
	p.allocs0, p.bytes0 = a, b
	p.cur.quads = lastQuads
	if p.cur.sec[secWorld] >= p.cur.sec[secFlush] {
		p.cur.sec[secWorld] -= p.cur.sec[secFlush] // world is reported without its flush
	}
	p.last = p.cur
	p.hist[p.hi] = p.cur
	p.hi = (p.hi + 1) % len(p.hist)
	benchRecord(&p.cur)
	p.cur = perfFrame{}
}

// perfToggle flips the overlay on F3.
func perfToggle() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		perf.show = !perf.show
	}
}

// summary condenses frames into averages and the 1% low frame rate.
type perfSummary struct {
	n                                     int
	updateSeconds                         float64
	discardedUpdateSeconds                float64
	updateCount, zeroDrawIntervals        int
	fps, low1, frameMS                    float64
	low01                                 float64
	p95MS, p99MS, p999MS, maxMS, stddevMS float64
	over60, over120, over144, over33      int
	sec                                   [numSecs]float64 // ms
	quads, world, sdf                     float64
	sprite, text, rect                    float64
	allocs, kb                            float64
}

func summarize(fs []perfFrame) perfSummary {
	var s perfSummary
	var times []float64
	var mean, m2 float64
	for _, f := range fs {
		// A coarse clock can give two draws the same timestamp. Their
		// updates still count toward simulation time.
		s.updateSeconds += f.updateSeconds
		s.discardedUpdateSeconds += f.discardedUpdateSeconds
		s.updateCount += f.updateCount
		if f.frame == 0 && f.updateCount > 0 {
			s.zeroDrawIntervals++
		}
		if f.frame <= 0 {
			continue
		}
		s.n++
		ms := f.frame.Seconds() * 1000
		times = append(times, ms)
		delta := ms - mean
		mean += delta / float64(s.n)
		m2 += delta * (ms - mean)
		if ms > 1000.0/60 {
			s.over60++
		}
		if ms > 1000.0/120 {
			s.over120++
		}
		if ms > 1000.0/144 {
			s.over144++
		}
		if ms > 1000.0/30 {
			s.over33++
		}
		s.frameMS += ms
		for i := range f.sec {
			s.sec[i] += f.sec[i].Seconds() * 1000
		}
		s.quads += float64(f.quads)
		s.world += float64(f.world)
		s.sdf += float64(f.sdf)
		s.sprite += float64(f.sprite)
		s.text += float64(f.text)
		s.rect += float64(f.rect)
		s.allocs += float64(f.allocs)
		s.kb += float64(f.bytes) / 1024
	}
	if s.n == 0 {
		return s
	}
	n := float64(s.n)
	s.frameMS /= n
	for i := range s.sec {
		s.sec[i] /= n
	}
	s.quads /= n
	s.world /= n
	s.sdf /= n
	s.sprite /= n
	s.text /= n
	s.rect /= n
	s.allocs /= n
	s.kb /= n
	s.fps = 1000 / s.frameMS
	sort.Float64s(times)
	s.stddevMS = math.Sqrt(m2 / n)
	s.p95MS, s.p99MS, s.p999MS = framePercentile(times, .95), framePercentile(times, .99), framePercentile(times, .999)
	s.maxMS = times[len(times)-1]
	// 1% low: the average of the slowest 1% of frames
	k := max(1, len(times)/100)
	w := 0.0
	for _, t := range times[len(times)-k:] {
		w += t
	}
	s.low1 = 1000 / (w / float64(k))
	k01 := max(1, len(times)/1000)
	w01 := 0.0
	for _, t := range times[len(times)-k01:] {
		w01 += t
	}
	s.low01 = 1000 / (w01 / float64(k01))
	return s
}

// framePercentile uses linear interpolation on an already sorted sample.
func framePercentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	at := math.Max(0, math.Min(1, q)) * float64(len(sorted)-1)
	i := int(at)
	if i == len(sorted)-1 {
		return sorted[i]
	}
	return sorted[i] + (sorted[i+1]-sorted[i])*(at-float64(i))
}

// drawPerf draws the F3 overlay in screen pixels at the top left.
func (g *Game) drawPerf(s *ebiten.Image) {
	if !perf.show {
		return
	}
	sm := summarize(perf.hist)
	w, h := s.Bounds().Dx(), s.Bounds().Dy()
	lines := []string{
		fmt.Sprintf("%.0f fps   1%% low %.0f   %.2f ms   %dx%d", sm.fps, sm.low1, sm.frameMS, w, h),
		fmt.Sprintf("cpu ms  upd %.2f  world %.2f  flush %.2f  post %.2f  hud %.2f  draw %.2f",
			sm.sec[secUpdate], sm.sec[secWorld], sm.sec[secFlush], sm.sec[secPost], sm.sec[secHUD], sm.sec[secDraw]),
		fmt.Sprintf("quads %.0f  3D calls %.0f  sdf %.0f  sprites %.0f  text %.0f  rects %.0f",
			sm.quads, sm.world, sm.sdf, sm.sprite, sm.text, sm.rect),
		fmt.Sprintf("allocs/frame %.0f (%.0f KB)  goroutines %d  %s", sm.allocs, sm.kb, runtime.NumGoroutine(), graphicsLib()),
	}
	sc := math.Max(1, float64(h)/1080)
	fillRect(s, 0, 0, 760*sc, (float64(len(lines))*22+12)*sc, color.RGBA{0, 0, 0, 190})
	for i, l := range lines {
		g.ui.textPx(s, l, 16*sc, 10*sc, (8+float64(i)*22)*sc, color.RGBA{180, 255, 190, 255})
	}
	// frame time graph: 240 frames, 1 px per 0.25 ms, red over 1/60 s
	gx, gy := 770*sc, 8*sc
	for i := range perf.hist {
		f := perf.hist[(perf.hi+i)%len(perf.hist)]
		ms := f.frame.Seconds() * 1000
		c := color.RGBA{120, 220, 140, 220}
		if ms > 16.7 {
			c = color.RGBA{255, 90, 80, 230}
		} else if ms > 8.4 {
			c = color.RGBA{255, 210, 90, 230}
		}
		hh := math.Min(80, ms*4) * sc
		fillRect(s, gx+float64(i)*sc, gy+80*sc-hh, sc, hh, c)
	}
}

func graphicsLib() string {
	var d ebiten.DebugInfo
	ebiten.ReadDebugInfo(&d)
	switch d.GraphicsLibrary {
	case ebiten.GraphicsLibraryDirectX:
		return "DirectX"
	case ebiten.GraphicsLibraryOpenGL:
		return "OpenGL"
	case ebiten.GraphicsLibraryMetal:
		return "Metal"
	}
	return "?"
}
