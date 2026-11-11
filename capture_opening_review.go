//go:build heroreview && !js

package main

import (
	"encoding/json"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var reviewOpening, reviewOpeningPlayback bool
var openingReviewStarted bool
var openingControlsChecked bool
var openingReviewFrame int
var openingReviewLast time.Time
var openingFrameMS []float64
var openingFPS, openingTPS []float64
var openingReviewTimes = []float64{1, 5.8, 13.2, 16.5, 20.5, 24.8, 30.8, 35.1, 39.6, 41.6, 44.4, 50.2, 56.6, 58.6, 60.6, 62.6, 64.6, 66.6, 68.6, 70.2, 72.4, 74.4, 79}

func init() {
	for _, a := range os.Args[1:] {
		if a == "-review-opening" {
			reviewOpening = true
		}
		if a == "-review-opening-playback" {
			reviewOpening = true
			reviewOpeningPlayback = true
		}
	}
}
func openingReviewDir() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "Intro-Native-Review")
}
func captureOpeningUpdate(g *Game) bool {
	if !openingReviewStarted {
		openingReviewStarted = true
		g.save = defaultSave()
		g.settings = &g.save.Settings
		benchConfig(g)
		if benchWinW > 0 && benchWinH > 0 {
			scale := ebiten.Monitor().DeviceScaleFactor()
			ebiten.SetFullscreen(false)
			ebiten.SetWindowSize(int(float64(benchWinW)/scale), int(float64(benchWinH)/scale))
		}
		g.snd.Music = .7
		g.snd.SFX = .7
		g.setState(StMenu)
		g.startOpening()
		g.opening.silent = !reviewOpeningPlayback
	}
	if g.opening == nil {
		finishOpeningReview(g)
		return true
	}
	if g.opening.err != "" {
		panic(g.opening.err)
	}
	if g.opening.images != nil && reviewOpeningPlayback && !openingControlsChecked {
		o := g.opening
		o.pause()
		if o.player.IsPlaying() {
			panic("Pause did not stop audio")
		}
		o.seek(37.5)
		if math.Abs(o.player.Position().Seconds()-37.5) > .02 {
			panic("Seek drift")
		}
		o.mix.setGains(.4, .2, .6)
		if o.mix.gains() != [3]float64{.4, .2, .6} {
			panic("Independent audio controls failed")
		}
		o.mix.setGains(.7, .7, .7)
		o.seek(0)
		o.pause()
		if !o.player.IsPlaying() {
			panic("Resume failed")
		}
		openingControlsChecked = true
	}
	if g.opening.images != nil && !reviewOpeningPlayback {
		g.opening.review = true
		i := openingReviewFrame / 4
		if i >= len(openingReviewTimes) {
			g.closeOpening()
			finishOpeningReview(g)
			return true
		}
		g.opening.t = openingReviewTimes[i]
	}
	return false
}
func captureOpeningAfterDraw(g *Game, dst *ebiten.Image) {
	o := g.opening
	if o == nil || o.images == nil {
		return
	}
	if reviewOpeningPlayback {
		now := time.Now()
		if o.t > 2 && o.t < 80 && !openingReviewLast.IsZero() {
			openingFrameMS = append(openingFrameMS, float64(now.Sub(openingReviewLast))/float64(time.Millisecond))
			openingFPS = append(openingFPS, ebiten.ActualFPS())
			openingTPS = append(openingTPS, ebiten.ActualTPS())
		}
		openingReviewLast = now
		return
	}
	if openingReviewFrame%4 == 3 {
		_ = os.MkdirAll(openingReviewDir(), 0755)
		im := image.NewRGBA(dst.Bounds())
		dst.ReadPixels(im.Pix)
		f, e := os.Create(filepath.Join(openingReviewDir(), "shot-"+itoa(openingReviewFrame/4)+".png"))
		if e != nil {
			panic(e)
		}
		e = png.Encode(f, im)
		_ = f.Close()
		if e != nil {
			panic(e)
		}
	}
	openingReviewFrame++
}
func finishOpeningReview(g *Game) {
	_ = os.MkdirAll(openingReviewDir(), 0755)
	r := map[string]interface{}{"saveWritesDisabled": noPersist, "playback": reviewOpeningPlayback, "captureTimes": openingReviewTimes, "window": []int{benchWinW, benchWinH}, "scope": "Native illustrated player only; not the separate Three/Blender browser preview."}
	r["pauseSeekResumeAndIndependentGainsVerified"] = openingControlsChecked
	if reviewOpeningPlayback {
		if len(openingFrameMS) < 100 {
			panic("Not enough native playback frames")
		}
		sorted := append([]float64(nil), openingFrameMS...)
		sort.Float64s(sorted)
		sum := 0.0
		slow := 0
		for _, v := range sorted {
			sum += v
			if v > 25 {
				slow++
			}
		}
		r["frameSamples"] = len(sorted)
		r["meanFrameMS"] = sum / float64(len(sorted))
		r["p95FrameMS"] = sorted[int(float64(len(sorted)-1)*.95)]
		r["p99FrameMS"] = sorted[int(float64(len(sorted)-1)*.99)]
		r["maxFrameMS"] = sorted[len(sorted)-1]
		r["framesOver25MS"] = slow
		r["actualFPS"] = openingFPS
		r["actualTPS"] = openingTPS
		r["limitation"] = "Frame intervals measured across native game Draw callbacks after two-second warmup. No GPU timer; hardware-specific result. Full system and display configuration not certified."
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	name := "captures.json"
	if reviewOpeningPlayback {
		name = "playback-" + itoa(benchWinW) + "x" + itoa(benchWinH) + ".json"
	}
	_ = os.WriteFile(filepath.Join(openingReviewDir(), name), b, 0644)
	g.quit = true
}
