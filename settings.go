package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------- performance

// Quality presets trade looks for speed. Low and Medium render the 3D world
// below native resolution and upscale it with a sharpening filter (post.go);
// dynamic resolution steps it down further only while frames run long.
var qualityNames = []string{"Low", "Medium", "High", "Custom"}

func (g *Game) applyQuality(q int) {
	st := g.settings
	st.Quality = q
	switch q {
	case 0:
		st.RenderRes, st.NoBloom, st.Particles = 0.67, true, 0
	case 1:
		st.RenderRes, st.NoBloom, st.Particles = 0.85, false, 1
	case 2:
		st.RenderRes, st.NoBloom, st.Particles = 1, false, 1
	}
}

// lowFX turns off the costliest decorative layers (sky effects, ambience).
func (g *Game) lowFX() bool { return g.settings.Quality == 0 }

// renderScale is the 3D resolution for this frame.
func (g *Game) renderScale(ddt float64) float64 {
	st := g.settings
	base := st.RenderRes
	if base <= 0 || base > 1 {
		base = 1
	}
	base = math.Max(0.5, base)
	if g.dynScale == 0 {
		g.dynScale = 1
	}
	if !st.DynRes {
		g.dynScale = 1
		return base
	}
	// smoothed frame time; adjust in small steps so it never pumps
	if ddt > 0 && ddt < 0.25 {
		g.frameAvg += (ddt - g.frameAvg) * 0.08
	}
	g.dynT += ddt
	switch {
	case g.frameAvg > 1.0/50 && g.dynT > 0.5 && g.dynScale > 0.6:
		g.dynScale = math.Max(0.6, g.dynScale-0.05)
		g.dynT = 0
	case g.frameAvg < 1.0/58 && g.dynT > 1.5 && g.dynScale < 1:
		g.dynScale = math.Min(1, g.dynScale+0.05)
		g.dynT = 0
	}
	return math.Max(0.4, base*g.dynScale)
}

// ---------------------------------------------------------------- screen

var settingTabs = []string{"Audio", "Display", "Effects", "Controls", "Access"}

func (g *Game) drawSettings(s *ebiten.Image) {
	u := g.ui
	st := g.settings
	u.panel(s, 460, 80, 1000, 900, 0.92)
	u.heading(s, "SETTINGS", 64, 100, cGold, 1)
	if g.navLatch.pageL {
		g.setTab = (g.setTab + len(settingTabs) - 1) % len(settingTabs)
	}
	if g.navLatch.pageR {
		g.setTab = (g.setTab + 1) % len(settingTabs)
	}
	for i, t := range settingTabs {
		acc := cAcc
		if g.setTab == i {
			acc = cGold
		}
		tx := 960 - 470 + float64(i)*190
		if u.button(s, "stab"+itoa(i), t, tx, 196, 180, 50, acc, true) {
			g.setTab = i
			g.rebinding = ""
		}
		if g.setTab == i {
			u.glowLine(s, tx+20, tx+160, 254, 3, cGold, 1)
		}
	}
	y := 300.0
	toggle := func(id, label string, v bool) bool {
		u.text(s, label, fBold, 22, 520, y+12, cWhite, alignLeft)
		if u.switchCtl(s, id, 1260, y+4, v) {
			v = !v
		}
		y += 62
		return v
	}
	note := func(t string) {
		u.paragraph(s, t, fReg, 18, 960, y, 860, cDim, alignCenter)
		y += 50
	}
	choice := func(id, label string, names []string, cur int, bw float64) int {
		u.text(s, label, fBold, 22, 520, y+12, cWhite, alignLeft)
		x := 1400 - float64(len(names))*(bw+10) + 10
		for i, n := range names {
			acc := cAcc
			if cur == i {
				acc = cGold
			}
			if u.button(s, id+itoa(i), n, x+float64(i)*(bw+10), y, bw, 48, acc, true) {
				cur = i
			}
		}
		y += 70
		return cur
	}
	switch g.setTab {
	case 0:
		st.Master = u.slider(s, "Master volume", 520, y, 880, st.Master, g.mouseHeld)
		st.Music = u.slider(s, "Music", 520, y+64, 880, st.Music, g.mouseHeld)
		st.SFX = u.slider(s, "Sound effects", 520, y+128, 880, st.SFX, g.mouseHeld)
		g.snd.SFX = st.SFX * st.Master
		g.snd.Music = st.Music * st.Master
		g.snd.updateVolumes()
	case 1:
		// quality preset as a row of choices
		u.text(s, "Graphics quality", fBold, 22, 520, y+12, cWhite, alignLeft)
		for i, n := range qualityNames {
			acc := cAcc
			if st.Quality == i {
				acc = cGold
			}
			if u.button(s, "q"+itoa(i), n, 790+float64(i)*152, y, 140, 48, acc, i < 3) {
				g.applyQuality(i)
			}
		}
		y += 76
		res := (math.Max(0.5, nz(st.RenderRes, 1)) - 0.5) / 0.5
		nr := u.sliderFmt(s, "3D resolution", 520, y, 880, res, g.mouseHeld, func(v float64) string {
			return fmt.Sprintf("%.0f%%", (0.5+0.5*v)*100)
		})
		if math.Abs(nr-res) > 1e-6 {
			st.RenderRes, st.Quality = 0.5+0.5*nr, 3
		}
		y += 70
		st.DynRes = toggle("dynres", "Dynamic resolution (holds the frame rate on busy moments)", st.DynRes)
		wm := 2
		switch {
		case st.Fullscreen:
			wm = 0
		case st.Borderless:
			wm = 1
		}
		nwm := choice("win", "Display mode (F11)", []string{"Fullscreen", "Borderless", "Windowed"}, wm, 150)
		vs := toggle("vs", "VSync", st.VSync)
		st.ShowFPS = toggle("fps", "Show FPS", st.ShowFPS)
		st.CenterHUD = !toggle("edgehud", "Ultrawide: spread the HUD to the screen edges", !st.CenterHUD)
		if nwm != wm || vs != st.VSync {
			st.Fullscreen, st.Borderless, st.VSync = nwm == 0, nwm == 1, vs
			g.applySettings()
		}
		note("Below 100% the 3D scene renders at a lower resolution and is sharpened back up to your screen; menus and text always stay at full resolution.")
	case 2:
		was := [3]bool{st.NoBloom, st.Particles == 1, st.Classic}
		st.Shake = toggle("shake", "Screen shake", st.Shake)
		st.Flashes = toggle("flash", "Flash effects", st.Flashes)
		p := toggle("parts", "Full particles", st.Particles == 1)
		st.Particles = 0
		if p {
			st.Particles = 1
		}
		st.NoBloom = !toggle("bloom", "Bloom & colour grading", !st.NoBloom)
		st.Classic = toggle("classic", "Classic backdrop (painted picture, no 3D scenery)", st.Classic)
		st.FastAnim = toggle("fastanim", "Fast animations (no slow motion or hit-stop, quicker screens)", st.FastAnim)
		if was != [3]bool{st.NoBloom, st.Particles == 1, st.Classic} && st.Quality != 3 {
			st.Quality = 3
		}
	case 3:
		g.drawControls(s, y)
	case 4:
		ts := (g.textScale() - 0.9) / 0.45
		nt := u.sliderFmt(s, "Text size", 520, y, 880, ts, g.mouseHeld, func(v float64) string {
			return fmt.Sprintf("%.0f%%", (0.9+0.45*v)*100)
		})
		if math.Abs(nt-ts) > 1e-6 {
			st.TextScale = 0.9 + 0.45*nt
		}
		y += 70
		gs := (g.gameSpeed() - 0.5) / 0.5
		ng := u.sliderFmt(s, "Game speed", 520, y, 880, gs, g.mouseHeld, func(v float64) string {
			return fmt.Sprintf("%.0f%%", (0.5+0.5*v)*100)
		})
		if math.Abs(ng-gs) > 1e-6 {
			st.GameSpeed = 0.5 + 0.5*math.Round(ng*10)/10
		}
		y += 70
		// language: ‹ name ›, each name written in its own script
		li := langIndex(st.Lang)
		u.text(s, "Language", fBold, 22, 520, y+12, cWhite, alignLeft)
		if u.button(s, "langL", "‹", 940, y, 56, 48, cAcc, true) {
			g.setLang(languages[(li+len(languages)-1)%len(languages)].Code)
		}
		if u.button(s, "langR", "›", 1344, y, 56, 48, cAcc, true) {
			g.setLang(languages[(li+1)%len(languages)].Code)
		}
		u.frame(s, 1006, y, 328, 48, 0.8)
		u.langName(s, languages[langIndex(st.Lang)].Name, 1170, y+11, cGold)
		y += 70
		st.ColorBlind = choice("cb", "Colour filter", colorBlindNames, st.ColorBlind, 170)
		st.NoTips = !toggle("tips", "First-time tips", !st.NoTips)
		if u.button(s, "tipsreset", "Show all tips again", 520, y, 300, 48, cAcc, len(g.save.Hints) > 0) {
			g.save.Hints = nil
		}
		y += 76
		note("A slower game speed makes every chamber easier to read and react to. Endless runs played below 100% are kept off the global leaderboard.")
	}
	if u.button(s, "setback", "Back", 760, 900, 400, 60, cGreen, true) {
		writeSave(&g.save)
		g.setState(g.prevState)
	}
}

func nz(v, d float64) float64 {
	if v == 0 {
		return d
	}
	return v
}
