package main

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Benchmark from the menu (roadmap #105): Settings > Display > Run
// benchmark plays a scripted 60-second tour of six chambers with the
// player's own settings (the autopilot plays), then shows the average frame
// rate, the 1% low and the worst frame of each, so a player can check a
// setting change without the command line.

const (
	mbScenes  = 6
	mbWarm    = 2.0 // seconds per scene before frames count
	mbPerSecs = 10.0
)

type mbResult struct {
	name              string
	avgFPS, low1, worstMS float64
}

type menuBench struct {
	scene   int
	sceneT  float64
	last    time.Time
	frames  []float64 // frame times, ms, this scene
	results []mbResult
	noTips  bool
	hints   map[string]bool
}

// mbName is a scene's name: its world, or the first Guardian.
func mbName(i int) string {
	if i == mbScenes-1 {
		return acts[0].BossName
	}
	return worlds[i%len(worlds)].Name
}

// startMenuBench begins the tour (from Settings, on the main menu only).
func (g *Game) startMenuBench() {
	mb := &menuBench{noTips: g.settings.NoTips, hints: g.save.Hints}
	g.mbench, g.mbenchDone = mb, nil
	g.settings.NoTips = true
	h := map[string]bool{}
	for k, v := range g.save.Hints {
		h[k] = v
	}
	for _, k := range []string{"level", "waves", "critter", "boss", "cache"} {
		h[k] = true
	}
	g.save.Hints = h
	inputHook = benchPilot
	g.mbScene(0)
}

func (g *Game) mbScene(i int) {
	mb := g.mbench
	if i != mb.scene || mb.sceneT == 0 {
		mb.scene, mb.sceneT, mb.frames = i, 0, mb.frames[:0]
	}
	mb.last = time.Time{} // the frame that sets a chamber up is not timed
	g.run = NewRun("BENCH", &archetypes[0], &g.save, false)
	g.run.Lives = g.run.maxLives()
	if i == mbScenes-1 {
		g.run.Act = 1
		g.level = g.newLevel(NBoss, 777)
	} else {
		forceWorld = i % NumWorlds
		g.level = g.newLevel(NCombat, int64(4242+i))
		forceWorld = -1
	}
	g.level.introT = 0
	g.setState(StLevel)
}

// tickMenuBench times the frame and moves the tour on; it returns true when
// it has taken the frame (cancelled or finished).
func (g *Game) tickMenuBench() bool {
	mb := g.mbench
	if mb == nil {
		return false
	}
	now := time.Now()
	g.coachQ = nil
	if backPressed() {
		g.endMenuBench(false)
		return true
	}
	if !mb.last.IsZero() {
		dt := now.Sub(mb.last).Seconds()
		mb.sceneT += dt
		if mb.sceneT > mbWarm {
			mb.frames = append(mb.frames, dt*1000)
		}
	}
	mb.last = now
	// a chamber the autopilot is about to clear (or anything that left the
	// chamber) is set up again, so a benchmark never finishes a chamber and
	// nothing it does reaches the player's records
	if g.state != StLevel || g.level == nil || g.level.Cleared || g.level.Left <= max(3, g.level.startLeft/8) {
		g.mbScene(mb.scene)
		return true
	}
	if mb.sceneT >= mbPerSecs {
		mb.results = append(mb.results, mbSummarize(mbName(mb.scene), mb.frames))
		if mb.scene+1 >= mbScenes {
			g.endMenuBench(true)
			return true
		}
		g.mbScene(mb.scene + 1)
		return true
	}
	return false
}

func mbSummarize(name string, ms []float64) mbResult {
	r := mbResult{name: name}
	if len(ms) == 0 {
		return r
	}
	sum := 0.0
	for _, v := range ms {
		sum += v
	}
	s := append([]float64(nil), ms...)
	sort.Float64s(s)
	r.avgFPS = 1000 * float64(len(s)) / sum
	r.worstMS = s[len(s)-1]
	// 1% low: the frame rate of the slowest 1% of frames
	n := max(1, len(s)/100)
	slow := 0.0
	for _, v := range s[len(s)-n:] {
		slow += v
	}
	r.low1 = 1000 * float64(n) / slow
	return r
}

func (g *Game) endMenuBench(done bool) {
	mb := g.mbench
	g.mbench = nil
	inputHook = nil
	g.settings.NoTips = mb.noTips
	g.save.Hints = mb.hints
	g.level, g.run = nil, nil
	if done {
		g.mbenchDone = mb.results
	}
	g.setState(StMenu)
	g.setState(StSettings) // back where the player started it
	g.setTab = 1
}

// drawMenuBench is the tour's banner while it runs, and the results card
// after it.
func (g *Game) drawMenuBench(s *ebiten.Image) {
	u := g.ui
	if mb := g.mbench; mb != nil {
		left := (mbScenes-mb.scene-1)*int(mbPerSecs) + int(math.Ceil(mbPerSecs-mb.sceneT))
		u.rrect(s, 660, 18, 600, 52, 14, rgba(8, 6, 18, 210))
		u.text(s, fmt.Sprintf(tr("Benchmark: %s (%d/%d), %d s left"), tr(mbName(mb.scene)), mb.scene+1, mbScenes, left), fBold, 20, 960, 28, cWhite, alignCenter)
		u.text(s, "Esc to stop", fReg, 14, 960, 52, cDim, alignCenter)
		return
	}
	res := g.mbenchDone
	if res != nil && g.state != StSettings {
		g.mbenchDone = nil // left Settings with the card still up
		return
	}
	if res == nil {
		return
	}
	fillRect(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), rgba(0, 0, 0, 150))
	x, y, w, h := 560.0, 230.0, 800.0, 560.0
	u.panel(s, x, y, w, h, 1)
	u.textOutlined(s, "Benchmark results", fTitle, 34, 960, y+30, cGold, alignCenter)
	u.rawText = true
	u.text(s, tr(qualityLabel(g))+listSep+frameCapLabel(g), fReg, 16, 960, y+78, cDim, alignCenter)
	u.rawText = false
	cols := [...]float64{x + 60, x + 400, x + 540, x + 680}
	ty := y + 120
	u.text(s, "Chamber", fBold, 18, cols[0], ty, cDim, alignLeft)
	u.text(s, "Average", fBold, 18, cols[1], ty, cDim, alignCenter)
	u.text(s, "1% low", fBold, 18, cols[2], ty, cDim, alignCenter)
	u.text(s, "Worst", fBold, 18, cols[3], ty, cDim, alignCenter)
	ty += 40
	tot, worst, low := 0.0, 0.0, math.Inf(1)
	for _, r := range res {
		u.text(s, r.name, fBold, 20, cols[0], ty, cWhite, alignLeft)
		u.rawText = true
		u.text(s, fpsText(r.avgFPS), fBold, 20, cols[1], ty, fpsColor(r.avgFPS), alignCenter)
		u.text(s, fmt.Sprint(math.Round(r.low1)), fReg, 20, cols[2], ty, fpsColor(r.low1), alignCenter)
		u.text(s, msText(r.worstMS), fReg, 20, cols[3], ty, cWhite, alignCenter)
		u.rawText = false
		tot += r.avgFPS
		worst = math.Max(worst, r.worstMS)
		low = math.Min(low, r.low1)
		ty += 44
	}
	if n := float64(len(res)); n > 0 {
		ty += 8
		u.text(s, "Overall", fBold, 22, cols[0], ty, cGold, alignLeft)
		u.rawText = true
		u.text(s, fpsText(tot/n), fBold, 22, cols[1], ty, fpsColor(tot/n), alignCenter)
		u.text(s, fmt.Sprint(math.Round(low)), fBold, 22, cols[2], ty, fpsColor(low), alignCenter)
		u.text(s, msText(worst), fBold, 22, cols[3], ty, cWhite, alignCenter)
		u.rawText = false
	}
	if u.button(s, "mbok", "OK", 960-90, y+h-76, 180, 52, cAcc, true) {
		g.mbenchDone = nil
	}
}

func fpsColor(f float64) color.RGBA {
	switch {
	case f >= 110:
		return rgba(120, 230, 140, 255)
	case f >= 58:
		return rgba(240, 220, 120, 255)
	}
	return rgba(255, 110, 100, 255)
}

func qualityLabel(g *Game) string { return qualityNames[g.settings.Quality] }

func frameCapLabel(g *Game) string {
	if g.settings.VSync {
		return "VSync"
	}
	return tr("Frame limit") + " " + tr(frameCapLabels[frameCapIndex(g.settings.FrameCap)])
}
