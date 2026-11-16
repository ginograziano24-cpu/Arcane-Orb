package main

// Screenshot tour: `ArcaneOrb.exe -shots` runs the benchmark tour plus a
// few extra screens, three seconds each, and saves one picture of every
// scene to a "shots" folder beside the exe (JPEG, at most 1600 px wide).
// It is a review tool: like the benchmark it never writes saves.

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	shotsOn   bool
	shotTaken = map[string]bool{}
)

const shotAt = 2300 * time.Millisecond

// shotsParse: called from benchParse for "-shots".
func shotsParse() {
	shotsOn, benchOn, noPersist = true, true, true
	benchWarm, benchMeasure = 500*time.Millisecond, 2500*time.Millisecond
	if benchTag == "" {
		benchTag = "shots"
	}
}

// shotScenes are the extra screens the plain benchmark does not visit.
func shotScenes() []benchScene {
	node := func(t NodeType) func(g *Game) {
		return func(g *Game) {
			benchRunFor(g)
			for i, n := range g.run.Map.Nodes {
				if n.Type == t {
					g.enterNode(i)
					return
				}
			}
			g.setState(StMap)
		}
	}
	boss := func(act int) func(g *Game) {
		return func(g *Game) {
			benchRunFor(g)
			old := g.run.Act
			g.run.Act = act
			g.level = g.newLevel(NBoss, int64(770+act))
			g.level.introT = 0
			g.run.Act = old
			g.setState(StLevel)
		}
	}
	return []benchScene{
		{name: "boss2", setup: boss(2), level: true},
		{name: "boss3", setup: boss(3), level: true},
		{name: "elite", setup: func(g *Game) {
			benchRunFor(g)
			forceWorld = WNeon
			g.level = g.newLevel(NElite, 991)
			forceWorld = -1
			g.level.introT = 0
			g.setState(StLevel)
		}, level: true},
		{name: "event", setup: node(NEvent)},
		{name: "rest", setup: node(NRest)},
		{name: "treasure", setup: node(NTreasure)},
		{name: "keepsakes", setup: func(g *Game) {
			g.save.Shards = 400
			g.save.KeepRanks = map[string]int{"coin": 3, "feather": 2, "token": 1}
			g.save.Keepsake = "coin"
			g.openKeepsakes(StMenu)
		}},
		{name: "trials", setup: func(g *Game) {
			g.setState(StArch)
			g.archPanel = 2
		}},
		{name: "settings", setup: func(g *Game) {
			g.prevState = StMenu
			g.setTab = 2
			g.setState(StSettings)
		}},
		{name: "codex", setup: func(g *Game) { g.codexTab = 0; g.setState(StCodex) }},
		{name: "sanctum", setup: func(g *Game) { g.setState(StSanctum) }},
		{name: "challenge", setup: func(g *Game) { g.setState(StChallenge) }},
		{name: "seed", setup: func(g *Game) { g.setState(StSeed) }},
		{name: "help", setup: func(g *Game) { g.setState(StHelp) }},
		{name: "actintro", setup: func(g *Game) { benchRunFor(g); g.setState(StActIntro) }},
		{name: "pause", setup: func(g *Game) {
			benchRunFor(g)
			forceWorld = WMeadow
			g.level = g.newLevel(NCombat, 4243)
			forceWorld = -1
			g.level.introT = 0
			g.setState(StLevel)
			g.setState(StPause)
		}},
	}
}

// shotCapture saves the finished frame once per scene (from Draw).
func shotCapture(g *Game, screen *ebiten.Image) {
	b := bench
	if !shotsOn || b == nil || b.i < 0 || b.i >= len(b.scenes) {
		return
	}
	name := b.scenes[b.i].name
	if shotTaken[name] || time.Since(b.start) < shotAt {
		return
	}
	shotTaken[name] = true
	r := screen.Bounds()
	src := image.NewRGBA(r)
	screen.ReadPixels(src.Pix)
	img := shotShrink(src, 1600)
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "shots")
	_ = os.MkdirAll(dir, 0o755)
	if f, err := os.Create(filepath.Join(dir, name+".jpg")); err == nil {
		_ = jpeg.Encode(f, img, &jpeg.Options{Quality: 88})
		f.Close()
	}
}

// shotShrink box-filters a picture down by a whole factor until it is at
// most maxW wide.
func shotShrink(src *image.RGBA, maxW int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	k := (w + maxW - 1) / maxW
	if k <= 1 {
		return src
	}
	dw, dh := w/k, h/k
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var s [3]int
			for j := 0; j < k; j++ {
				o := (y*k+j)*src.Stride + x*k*4
				for i := 0; i < k; i++ {
					s[0] += int(src.Pix[o])
					s[1] += int(src.Pix[o+1])
					s[2] += int(src.Pix[o+2])
					o += 4
				}
			}
			d := y*dst.Stride + x*4
			n := k * k
			dst.Pix[d], dst.Pix[d+1], dst.Pix[d+2], dst.Pix[d+3] = uint8(s[0]/n), uint8(s[1]/n), uint8(s[2]/n), 255
		}
	}
	return dst
}
