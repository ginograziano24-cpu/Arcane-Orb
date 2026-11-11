package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
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
//	-benchpreset=deck  Medium, 1280x800 window and 60 FPS by default

var (
	noPersist       bool // never write saves (benchmark)
	benchOn         bool
	benchTag        string
	benchWinW       int
	benchWinH       int
	cpuProfile      string
	profFile        *os.File
	benchAudio      bool
	benchVSync      bool
	benchFrameCap   int
	benchQuality    = 2
	benchTrace      bool
	benchTraceScene = "multiball"
	benchTraceFile  *os.File
	benchSceneNames string
)

type benchScene struct {
	name  string
	setup func(g *Game)
	level bool // a chamber: the autopilot plays it
}

type benchRun struct {
	scenes                         []benchScene
	i                              int
	start                          time.Time
	frames                         []perfFrame
	lines                          []string
	sums                           []perfSummary
	names                          []string
	warmFrames                     int
	reserved                       bool
	measureStart                   time.Time
	setupTime, firstFrame, warmMax time.Duration
	transitions                    []benchTransition
}

type benchTransition struct {
	setup, first, warmMax time.Duration
	warmFrames            int
}

var bench *benchRun

var (
	benchWarm    = 2500 * time.Millisecond
	benchMeasure = 6 * time.Second
)

func benchDurationArg(value string, min, max time.Duration) (time.Duration, bool) {
	d, err := time.ParseDuration(value)
	return d, err == nil && d >= min && d <= max
}

// benchParse reads the command line; call first thing in main.
func benchParse() {
	var deckPreset, explicitCap bool
	for _, a := range os.Args[1:] {
		switch {
		case a == "-bench" || a == "--bench":
			benchOn, noPersist = true, true
		case a == "-nosave":
			noPersist = true
		case a == "-livemenu":
			liveMenuSynthesis = true
		case strings.HasPrefix(a, "-benchtrace="):
			benchTrace = true
			benchTraceScene = strings.TrimPrefix(a, "-benchtrace=")
		case a == "-benchtrace":
			benchTrace = true
		case a == "-benchaudio":
			benchAudio = true
		case a == "-benchvsync":
			benchVSync = true
		case a == "-benchpreset=deck":
			deckPreset = true
		case strings.HasPrefix(a, "-benchcap="):
			explicitCap = true
			switch strings.TrimPrefix(a, "-benchcap=") {
			case "display":
				benchFrameCap = frameCapDisplay
			case "60":
				benchFrameCap = 60
			case "120":
				benchFrameCap = 120
			case "144":
				benchFrameCap = 144
			case "off":
				benchFrameCap = 0
			default:
				fmt.Fprintln(os.Stderr, "Invalid -benchcap (use off, 60, 120, 144, or display)")
				os.Exit(2)
			}
		case strings.HasPrefix(a, "-benchscenes="):
			benchSceneNames = strings.TrimPrefix(a, "-benchscenes=")
		case strings.HasPrefix(a, "-benchduration="):
			if d, ok := benchDurationArg(strings.TrimPrefix(a, "-benchduration="), time.Second, 10*time.Minute); ok {
				benchMeasure = d
			} else {
				fmt.Fprintln(os.Stderr, "Invalid -benchduration (use 1s through 10m); retaining default")
			}
		case strings.HasPrefix(a, "-benchwarmup="):
			if d, ok := benchDurationArg(strings.TrimPrefix(a, "-benchwarmup="), 0, time.Minute); ok {
				benchWarm = d
			} else {
				fmt.Fprintln(os.Stderr, "Invalid -benchwarmup (use 0s through 1m); retaining default")
			}
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
	if deckPreset {
		benchQuality = 1
		if benchWinW <= 0 || benchWinH <= 0 {
			benchWinW, benchWinH = 1280, 800
		}
		if !explicitCap {
			benchFrameCap = 60
		}
	}
	if benchSceneNames != "" {
		if _, err := selectBenchScenes(benchScenes(), benchSceneNames); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if benchOn && !benchTrace && cpuProfile == "" {
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
	ebiten.SetVsyncEnabled(benchVSync)
	bench = &benchRun{scenes: benchScenes()}
	if benchSceneNames != "" {
		bench.scenes, _ = selectBenchScenes(bench.scenes, benchSceneNames)
	}
	bench.i = -1
}

// benchConfig puts the game in the fixed state the tour and the golden
// captures measure: fixed graphics quality, no tips and autopilot.
func benchConfig(g *Game) {
	st := g.settings
	g.applyQuality(benchQuality)
	st.DynRes = false
	st.NoTips = true
	if benchAudio {
		g.snd.SFX, g.snd.Music = .8, .6
	} else {
		g.snd.SFX, g.snd.Music = 0, 0
	}
	g.snd.updateVolumes()
	g.coachQ = nil
	g.save.Hints = map[string]bool{"level": true, "waves": true, "critter": true, "map": true, "reward": true, "cache": true}
	inputHook = benchPilot
}

func selectBenchScenes(scenes []benchScene, names string) ([]benchScene, error) {
	var selected []benchScene
	seen := map[string]bool{}
	for _, part := range strings.Split(names, ",") {
		name := strings.TrimSpace(part)
		if name == "" || seen[name] {
			return nil, fmt.Errorf("invalid or repeated benchmark scene %q", name)
		}
		found := false
		for _, scene := range scenes {
			if scene.name == name {
				selected = append(selected, scene)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown benchmark scene %q", name)
		}
		seen[name] = true
	}
	return selected, nil
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
	if benchTraceFile != nil && b.warmFrames > 0 {
		trace.Stop()
		benchTraceFile.Close()
		benchTraceFile = nil
	}
	g.settings.NoTips = true
	g.coachQ = nil
	if b.i >= 0 && (b.measureStart.IsZero() || time.Since(b.measureStart) < benchMeasure) {
		return false
	}
	if b.i >= 0 {
		s := summarize(b.frames)
		b.sums = append(b.sums, s)
		b.names = append(b.names, b.scenes[b.i].name)
		b.transitions = append(b.transitions, benchTransition{b.setupTime, b.firstFrame, b.warmMax, b.warmFrames})
	}
	b.i++
	if b.i >= len(b.scenes) {
		benchFinish(g)
		return true
	}
	b.frames = b.frames[:0]
	b.warmFrames, b.reserved = 0, false
	b.measureStart = time.Time{}
	b.firstFrame, b.warmMax = 0, 0
	if benchTrace && b.scenes[b.i].name == benchTraceScene {
		f, err := os.Create(benchPath("trace", ".out"))
		if err == nil {
			err = trace.Start(f)
			if err == nil {
				benchTraceFile = f
			} else {
				f.Close()
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Transition trace:", err)
		}
	}
	setupStart := time.Now()
	b.scenes[b.i].setup(g)
	b.start = time.Now()
	b.setupTime = b.start.Sub(setupStart)
	return true
}

// benchRecord keeps a frame once the scene has warmed up.
func benchRecord(f *perfFrame) {
	if bench == nil || bench.i < 0 || bench.i >= len(bench.scenes) {
		return
	}
	benchCollectFrame(bench, f, time.Now())
}

func benchCollectFrame(b *benchRun, f *perfFrame, now time.Time) {
	if !b.reserved {
		if b.warmFrames == 0 {
			b.firstFrame = f.frame
		}
		b.warmFrames++
		b.warmMax = max(b.warmMax, f.frame)
		if now.Sub(b.start) < benchWarm {
			return
		}
		// Reserve using the observed warmup rate before collecting timings.
		// Skip this frame AND the following frame, whose elapsed time/alloc
		// counters can include this allocation, so the recorder does not
		// manufacture the first measured stutter itself.
		rate := 1000.0
		if elapsed := now.Sub(b.start).Seconds(); elapsed > .01 {
			rate = float64(b.warmFrames) / elapsed
		}
		needed := max(1024, int(rate*benchMeasure.Seconds()*1.25)+16)
		if cap(b.frames) < needed {
			b.frames = make([]perfFrame, 0, needed)
		}
		b.reserved = true
		return
	}
	if b.measureStart.IsZero() {
		b.measureStart = now
		return
	}
	b.frames = append(b.frames, *f)
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
	fmt.Fprintf(&sb, "graphics %s  monitor %dx%d  scale %.2f  refresh %d Hz  fullscreen %v  window %dx%d  drawn %dx%d  vsync %v  frame cap %s  quality %s (render %.0f%%, bloom %v)\n",
		graphicsLib(), w, h, ebiten.Monitor().DeviceScaleFactor(), ebiten.DisplayRefreshRate(), ebiten.IsFullscreen(), ww, wh, perfScreen.X, perfScreen.Y, benchVSync, frameCapLabels[frameCapIndex(benchFrameCap)], qualityNames[g.settings.Quality], g.settings.RenderRes*100, !g.settings.NoBloom)
	fmt.Fprintf(&sb, "%s  GOMAXPROCS %d  cpus %d  EBITENGINE_DIRECTX=%q  (classic2/classic5: Classic backdrop with modelled bricks)\n\n",
		runtime.Version(), runtime.GOMAXPROCS(0), runtime.NumCPU(), os.Getenv("EBITENGINE_DIRECTX"))
	fmt.Fprintf(&sb, "warmup %s per scene, measurement %s per scene; audio enabled %v; -livemenu flag %v (browser builds always use live synthesis)\n", benchWarm, benchMeasure, benchAudio, liveMenuSynthesis)
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
	sb.WriteString("\nFrame pacing (milliseconds; over-budget counts use strict >; sample count excludes warmup):\n")
	fmt.Fprintf(&sb, "%-10s %7s %7s %7s %7s %7s %7s %7s | %7s %7s %7s %7s\n", "scene", "samples", "0.1%low", "p95", "p99", "p99.9", "max", "stddev", "miss60", "miss120", "miss144", ">33.3ms")
	for i, s := range b.sums {
		fmt.Fprintf(&sb, "%-10s %7d %7.1f %7.3f %7.3f %7.3f %7.3f %7.3f | %7d %7d %7d %7d\n", b.names[i], s.n, s.low01, s.p95MS, s.p99MS, s.p999MS, s.maxMS, s.stddevMS, s.over60, s.over120, s.over144, s.over33)
	}
	sb.WriteString("0.1% low is FPS from the mean of the slowest 0.1% frame times (minimum one frame). Short samples are descriptive, not reliable tail estimates.\nPercentiles use linear interpolation. Frame times include presentation/waiting, not GPU timestamp measurements.\n")
	sb.WriteString("\nElapsed update time versus frame time (seconds):\n")
	for i, s := range b.sums {
		fmt.Fprintf(&sb, "%-10s updates %.4f, frames %.4f, %d updates, %.4f discarded seconds, %d zero draw intervals\n", b.names[i], s.updateSeconds, s.frameMS*float64(s.n)/1000, s.updateCount, s.discardedUpdateSeconds, s.zeroDrawIntervals)
	}
	sb.WriteString("\nTransition/warmup observations, separate from steady-state results (ms):\n")
	for i, t := range b.transitions {
		fmt.Fprintf(&sb, "%-10s setup %.3f, first draw interval %.3f, warmup max interval %.3f, warmup frames %d\n", b.names[i], float64(t.setup)/float64(time.Millisecond), float64(t.first)/float64(time.Millisecond), float64(t.warmMax)/float64(time.Millisecond), t.warmFrames)
	}
	sb.WriteString("Setup is the scene setup call only, not process cold-start time. First/warmup intervals can include the preceding frame/present. Recorder reserves storage before measurement and excludes the following settling interval.\n")
	sb.WriteString("\nms columns are CPU time per frame on the game thread; 'other' is the rest of the frame\n" +
		"(Ebitengine submitting to the driver, presenting and waiting on the GPU).\n")
	_ = os.WriteFile(benchPath("report", ".txt"), []byte(sb.String()), 0o644)
	g.quit = true
}
