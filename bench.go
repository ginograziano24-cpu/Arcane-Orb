package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The benchmark tour: ArcaneOrb.exe -bench plays through every kind of
// screen on its own (menus, the map, a packed chamber in each world, a
// multiball storm, a boss, the shop, a reward pick), records frame times,
// CPU section timings, draw counts and allocations, writes bench.txt (and a
// CPU profile) beside the exe, and quits. It never writes the save or the
// run in progress.
//
//	-bench             run the tour
//	-cpuprofile=FILE   write a CPU profile (the tour writes bench_cpu.pprof)
//	-benchtag=NAME     label for the report file (bench_report_NAME.txt)
//	-benchwin=WxH      run windowed at this size instead of fullscreen

var (
	noPersist  bool // never write saves (benchmark)
	benchOn    bool
	benchTag   string
	benchWinW  int
	benchWinH  int
	cpuProfile string
	profFile   *os.File
)

type benchScene struct {
	name  string
	setup func(g *Game)
	level bool // a chamber: the autopilot plays it
}

type benchRun struct {
	scenes []benchScene
	i      int
	start  time.Time
	frames []perfFrame
	lines  []string
	sums   []perfSummary
	names  []string
}

var bench *benchRun

const (
	benchWarm    = 2500 * time.Millisecond
	benchMeasure = 6 * time.Second
)

// benchParse reads the command line; call first thing in main.
func benchParse() {
	for _, a := range os.Args[1:] {
		switch {
		case a == "-bench" || a == "--bench":
			benchOn, noPersist = true, true
		case strings.HasPrefix(a, "-cpuprofile="):
			cpuProfile = strings.TrimPrefix(a, "-cpuprofile=")
		case strings.HasPrefix(a, "-benchtag="):
			benchTag = strings.TrimPrefix(a, "-benchtag=")
		case a == "-noruns":
			planRunsOn = false
		case a == "-notextcache":
			textOff = true
		case a == "-glowaside":
			glowAsideOn = true
		case a == "-ebitenaa":
			aaOwn = false
		case a == "-noparmerge":
			parMergeOn = false
		case a == "-nopieces":
			piecesOn = false
		case a == "-nopresort":
			presortOn = false
		case strings.HasPrefix(a, "-benchwin="):
			_, _ = fmt.Sscanf(strings.TrimPrefix(a, "-benchwin="), "%dx%d", &benchWinW, &benchWinH)
		}
	}
	if benchOn && cpuProfile == "" {
		cpuProfile = benchPath("cpu", ".pprof")
	}
	if cpuProfile != "" {
		if f, err := os.Create(cpuProfile); err == nil {
			profFile = f
			_ = pprof.StartCPUProfile(f)
		}
	}
}

func benchPath(kind, ext string) string {
	exe, _ := os.Executable()
	name := "bench_" + kind
	if benchTag != "" {
		name += "_" + benchTag
	}
	return filepath.Join(filepath.Dir(exe), name+ext)
}

func stopProfile() {
	if profFile != nil {
		pprof.StopCPUProfile()
		profFile.Close()
		profFile = nil
	}
}

func benchSetupGame(g *Game) {
	benchConfig(g)
	if benchWinW > 0 && benchWinH > 0 {
		// a fixed window in device pixels: Layout multiplies by the scale
		s := ebiten.Monitor().DeviceScaleFactor()
		ebiten.SetFullscreen(false)
		ebiten.SetWindowSize(int(float64(benchWinW)/s), int(float64(benchWinH)/s))
	} else {
		g.settings.Fullscreen = true
		ebiten.SetFullscreen(true)
	}
	ebiten.SetVsyncEnabled(false)
	bench = &benchRun{scenes: benchScenes()}
	bench.i = -1
}

// benchConfig puts the game in the fixed state the tour and the golden
// captures measure: High quality, no tips, muted, the autopilot playing.
func benchConfig(g *Game) {
	st := g.settings
	g.applyQuality(2) // High: full resolution, bloom, particles
	st.DynRes = false
	st.NoTips = true
	g.snd.SFX, g.snd.Music = 0, 0
	g.snd.updateVolumes()
	g.coachQ = nil
	g.save.Hints = map[string]bool{"level": true, "waves": true, "critter": true, "map": true, "reward": true, "cache": true}
	inputHook = benchPilot
}

// benchScenes is the tour, shared with the golden captures.
func benchScenes() []benchScene {
	chamber := func(world int, extraBalls int, classic bool) func(g *Game) {
		return func(g *Game) {
			benchRunFor(g)
			g.settings.Classic = classic
			forceWorld = world
			g.level = g.newLevel(NCombat, int64(4242+world))
			forceWorld = -1
			g.setState(StLevel)
			L := g.level
			L.introT = 0
			for i := 0; i < extraBalls; i++ {
				sp := L.speed()
				a := float64(i) * 0.7
				L.Balls = append(L.Balls, &Ball{X: math.Sin(a) * 6, Z: 9 + float64(i%3), R: L.ballR(), VX: sp * math.Sin(a+0.4), VZ: sp * math.Cos(a+0.4)})
			}
		}
	}
	return []benchScene{
		{name: "menu", setup: func(g *Game) { g.settings.Classic = false; g.setState(StMenu) }},
		{name: "classsel", setup: func(g *Game) { g.setState(StArch) }},
		{name: "map", setup: func(g *Game) { benchRunFor(g); g.setState(StMap) }},
		{name: "world1", setup: chamber(0, 0, false), level: true},
		{name: "world2", setup: chamber(1, 0, false), level: true},
		{name: "world3", setup: chamber(2, 0, false), level: true},
		{name: "world4", setup: chamber(3, 0, false), level: true},
		{name: "world5", setup: chamber(4, 0, false), level: true},
		{name: "world6", setup: chamber(5, 0, false), level: true},
		{name: "multiball", setup: chamber(3, 8, false), level: true},
		// Classic backdrop: the painted picture, and the modelled bricks
		{name: "classic2", setup: chamber(1, 0, true), level: true},
		{name: "classic5", setup: chamber(4, 0, true), level: true},
		{name: "boss", setup: func(g *Game) {
			benchRunFor(g)
			g.settings.Classic = false
			g.level = g.newLevel(NBoss, 777)
			g.level.introT = 0
			g.setState(StLevel)
		}, level: true},
		{name: "shop", setup: func(g *Game) {
			benchRunFor(g)
			for i, n := range g.run.Map.Nodes {
				if n.Type == NShop {
					g.enterNode(i)
					return
				}
			}
			g.setState(StMap)
		}},
		{name: "reward", setup: func(g *Game) {
			benchRunFor(g)
			g.rewards = []rewardSet{{Title: "Choose an upgrade", Mods: []*Mod{modByID["blast"], modByID["excalibur"], modByID["hive"]}, Skip: 15}}
			g.setState(StReward)
		}},
	}
}

// benchRunFor gives the tour a fresh run that touches no files.
func benchRunFor(g *Game) {
	if g.run == nil {
		g.run = NewRun("BENCH", &archetypes[0], &g.save, false)
	}
	g.run.Lives = 3
}

// benchPilot follows the lowest orb heading for the paddle.
func benchPilot(g *Game) Input {
	L := g.level
	in := Input{UseKeys: true}
	if L == nil {
		return in
	}
	target, best := 0.0, 1e9
	for _, b := range L.Balls {
		if b.VZ < 0 && b.Z < best {
			best, target = b.Z, b.X
		}
	}
	if math.Abs(target-L.PaddleX) > 0.3 {
		in.KeyDir = math.Copysign(1, target-L.PaddleX)
	}
	if L.anyStuck() {
		L.release()
	}
	if g.run != nil {
		g.run.Lives = 3 // topped up every frame, so the tour never ends early
	}
	return in
}

// benchUpdate advances the tour; it returns true when it has taken over
// the frame (switching scenes).
func benchUpdate(g *Game) bool {
	if !benchOn {
		return false
	}
	if bench == nil {
		benchSetupGame(g)
	}
	b := bench
	g.settings.NoTips = true
	g.coachQ = nil
	if b.i >= 0 && time.Since(b.start) < benchWarm+benchMeasure {
		return false
	}
	if b.i >= 0 {
		s := summarize(b.frames)
		b.sums = append(b.sums, s)
		b.names = append(b.names, b.scenes[b.i].name)
	}
	b.i++
	if b.i >= len(b.scenes) {
		benchFinish(g)
		return true
	}
	b.frames = b.frames[:0]
	b.scenes[b.i].setup(g)
	b.start = time.Now()
	return true
}

// benchRecord keeps a frame once the scene has warmed up.
func benchRecord(f *perfFrame) {
	if bench == nil || bench.i < 0 || bench.i >= len(bench.scenes) {
		return
	}
	if time.Since(bench.start) >= benchWarm {
		bench.frames = append(bench.frames, *f)
	}
}

func benchFinish(g *Game) {
	stopProfile()
	if f, err := os.Create(benchPath("allocs", ".pprof")); err == nil {
		_ = pprof.Lookup("allocs").WriteTo(f, 0) // where the per-frame garbage comes from
		f.Close()
	}
	b := bench
	var sb strings.Builder
	w, h := ebiten.Monitor().Size()
	fmt.Fprintf(&sb, "Arcane Orb benchmark  %s  %s\n", gameVersion, time.Now().Format("2006-01-02 15:04"))
	ww, wh := ebiten.WindowSize()
	fmt.Fprintf(&sb, "graphics %s  monitor %dx%d  scale %.2f  fullscreen %v  window %dx%d  drawn %dx%d  vsync off  quality High (render 100%%, bloom on)\n",
		graphicsLib(), w, h, ebiten.Monitor().DeviceScaleFactor(), ebiten.IsFullscreen(), ww, wh, perfScreen.X, perfScreen.Y)
	fmt.Fprintf(&sb, "%s  GOMAXPROCS %d  cpus %d  EBITENGINE_DIRECTX=%q  (classic2/classic5: Classic backdrop with modelled bricks)\n\n",
		runtime.Version(), runtime.GOMAXPROCS(0), runtime.NumCPU(), os.Getenv("EBITENGINE_DIRECTX"))
	fmt.Fprintf(&sb, "%-10s %7s %7s %7s | %6s %6s %6s %6s %6s %6s %6s | %6s %5s %5s %5s %5s %5s | %7s %7s\n",
		"scene", "fps", "1%low", "ms", "upd", "world", "flush", "post", "hud", "draw", "other",
		"quads", "3D", "sdf", "spr", "text", "rect", "allocs", "KB")
	for i, s := range b.sums {
		other := s.frameMS - s.sec[secUpdate] - s.sec[secDraw]
		fmt.Fprintf(&sb, "%-10s %7.0f %7.0f %7.2f | %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f | %6.0f %5.0f %5.0f %5.0f %5.0f %5.0f | %7.0f %7.0f\n",
			b.names[i], s.fps, s.low1, s.frameMS,
			s.sec[secUpdate], s.sec[secWorld], s.sec[secFlush], s.sec[secPost], s.sec[secHUD], s.sec[secDraw], other,
			s.quads, s.world, s.sdf, s.sprite, s.text, s.rect, s.allocs, s.kb)
	}
	sb.WriteString("\nms columns are CPU time per frame on the game thread; 'other' is the rest of the frame\n" +
		"(Ebitengine submitting to the driver, presenting and waiting on the GPU).\n")
	_ = os.WriteFile(benchPath("report", ".txt"), []byte(sb.String()), 0o644)
	g.quit = true
}
