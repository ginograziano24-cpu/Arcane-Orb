package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The run recap: where the damage came from, how the run unfolded, and a
// short history of past runs in the Codex.

type recapEvent struct {
	Act   int    `json:"a"`
	Kind  string `json:"k"` // item, relic, fusion, boss, life, heart, curse, act
	Text  string `json:"t"`
	Glyph string `json:"g,omitempty"`
}

type runRecap struct {
	Dmg       map[string]float64 `json:"dmg"`
	Timeline  []recapEvent       `json:"tl"`
	LivesLost int                `json:"ll"`
	Hearts    int                `json:"hh"`
	Scrolls   int                `json:"sc"`
}

func newRecap() *runRecap { return &runRecap{Dmg: map[string]float64{}} }

func (r *Run) rc() *runRecap {
	if r.recap == nil {
		r.recap = newRecap()
	}
	return r.recap
}

// recordDmg books damage dealt by a source.
func (r *Run) recordDmg(src string, amt float64) {
	if amt <= 0 || r.SeedCode == "MENU" {
		return
	}
	if src == "" {
		src = "Orb"
	}
	r.rc().Dmg[src] += amt
}

// log adds a moment to the run's timeline.
func (r *Run) log(kind, text, glyph string) {
	rc := r.rc()
	if len(rc.Timeline) >= 80 {
		return
	}
	rc.Timeline = append(rc.Timeline, recapEvent{Act: r.Act, Kind: kind, Text: text, Glyph: glyph})
}

// ---------------------------------------------------------------- history

type runHistory struct {
	Date      string   `json:"date"`
	Arch      string   `json:"arch"`
	Challenge string   `json:"challenge,omitempty"`
	Act       int      `json:"act"`
	Won       bool     `json:"won"`
	Endless   bool     `json:"endless,omitempty"`
	Loop      int      `json:"loop,omitempty"`
	Score     int      `json:"score"`
	Depth     int      `json:"depth"`
	Asc       int      `json:"asc"`
	Seed      string   `json:"seed"`
	Top       string   `json:"top"` // biggest damage source
	Build     []string `json:"build"`
	Minutes   int      `json:"min"`
}

// recordHistory files the finished run (newest first, last 40 kept).
func (g *Game) recordHistory(r *Run, win bool) {
	top, _ := topSource(r)
	h := runHistory{Date: time.Now().Format("2006-01-02"), Arch: r.Arch.ID, Challenge: r.Challenge, Act: r.Act, Won: win,
		Endless: r.Endless, Loop: r.Loop(), Score: r.FinalScore, Depth: r.Depth, Asc: r.Asc, Seed: r.SeedCode, Top: top,
		Minutes: int(time.Since(r.Start).Minutes())}
	for i := len(r.Order) - 1; i >= 0 && len(h.Build) < 10; i-- {
		if m := modByID[r.Order[i]]; m != nil && !m.Curse {
			h.Build = append(h.Build, m.ID)
		}
	}
	g.save.History = append([]runHistory{h}, g.save.History...)
	if len(g.save.History) > 40 {
		g.save.History = g.save.History[:40]
	}
}

type dmgRow struct {
	Src string
	Amt float64
}

func dmgRows(r *Run) ([]dmgRow, float64) {
	var rows []dmgRow
	total := 0.0
	for k, v := range r.rc().Dmg {
		rows = append(rows, dmgRow{k, v})
		total += v
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Amt > rows[j].Amt })
	return rows, total
}

func topSource(r *Run) (string, float64) {
	rows, total := dmgRows(r)
	if len(rows) == 0 || total <= 0 {
		return "", 0
	}
	return rows[0].Src, rows[0].Amt / total
}

// dmgColor gives each damage source a recognisable colour.
func dmgColor(src string) color.RGBA {
	switch src {
	case "Orb":
		return rgba(200, 180, 255, 255)
	case "Burning", "Firestorm", "Steam Engine":
		return rgba(255, 130, 50, 255)
	case "Lightning", "Static Field", "Thunderstrike":
		return rgba(130, 200, 255, 255)
	case "Explosions", "Shrapnel", "Crown of Thorns":
		return rgba(255, 190, 80, 255)
	case "Shatter":
		return rgba(170, 230, 255, 255)
	case "Lasers":
		return rgba(255, 90, 100, 255)
	case "Event Horizon", "Attunement":
		return rgba(180, 110, 255, 255)
	case "Parry", "Hawkstrike", "Eagle Eye":
		return rgba(255, 225, 150, 255)
	case "Scrolls":
		return rgba(120, 230, 170, 255)
	}
	return rgba(210, 200, 230, 255)
}

// ---------------------------------------------------------------- summary

func (g *Game) drawSummary(s *ebiten.Image) {
	u := g.ui
	S := u.S
	r := g.run
	title, col := "RUN OVER", cRed
	if g.victory {
		title, col = "VICTORY!", cGold
	} else if r.Endless {
		title, col = fmt.Sprintf("ENDLESS  ·  LOOP %d", r.Loop()), rgba(255, 140, 120, 255)
	}
	// the origin stands beside the tally: lit in victory, dimmed in defeat
	if img := portrait(r.Arch.ID); img != nil {
		ph := 760.0
		pw := ph * 0.75
		px, py := 190-pw/2, 1080-ph+10
		k := ease(clamp01(g.stateT * 1.5))
		gc := classInfo[r.Arch.ID].Col
		if !g.victory && !r.Endless {
			gc = rgba(120, 40, 50, 255)
		}
		drawSprite(s, sprSoft, u.X(px-60), u.Y(py+80), (pw+120)*S, ph*S, ca(gc, 0.25*k), true)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(pw*S/float64(img.Bounds().Dx()), ph*S/float64(img.Bounds().Dy()))
		op.GeoM.Translate(u.X(px), u.Y(py+(1-k)*20))
		if !g.victory && !r.Endless {
			op.ColorScale.Scale(0.5, 0.4, 0.45, 1)
		}
		op.ColorScale.ScaleAlpha(float32(k))
		s.DrawImage(img, op)
		fillPath(s, boxPath(u.X(-10), u.Y(py-10), 500*S, 330*S, 0, false), rgba(4, 3, 10, 235), rgba(4, 3, 10, 0), vgrad(u.Y(py-10), u.Y(py+320)), false)
	}
	u.panel(s, 400, 40, 1490, 1000, 0.93)
	u.heading(s, title, 84, 56, col, 1)

	// ---- left column: the tally
	lx := 790.0
	sub := fmt.Sprintf("%s   ·   Seed %s", r.Arch.Name, r.SeedCode)
	if c := r.challenge(); c != nil {
		sub += "   ·   Challenge: " + c.Name
	} else if r.Asc > 0 {
		sub += fmt.Sprintf("   ·   Ascension %d", r.Asc)
	}
	mins := int(time.Since(r.Start).Minutes())
	lines := []string{
		sub,
		fmt.Sprintf("Reached %s (Act %d), %s  ·  %d min", biomes[r.AI()].Name, r.Act, plural(r.Depth, "chamber"), mins),
		fmt.Sprintf("Bricks %d  ·  Elites %d  ·  Guardians %d  ·  Best combo ×%d", r.Bricks, r.Elites, r.Bosses, r.MaxCombo),
		fmt.Sprintf("Lives lost %d  ·  Hearts caught %d  ·  Final score %d", r.rc().LivesLost, r.rc().Hearts, r.FinalScore),
	}
	y := 190.0
	for i, l := range lines {
		c := cWhite
		if i > 0 {
			c = rgba(214, 206, 236, 255)
		}
		u.text(s, l, fReg, 21, lx, y, c, alignCenter)
		y += 36
	}
	// build strip
	n := min(len(r.Order), 12)
	bx := lx - float64(n)*28
	for i := 0; i < n; i++ {
		if m := modByID[r.Order[len(r.Order)-n+i]]; m != nil {
			cx := bx + float64(i)*56 + 28
			u.drawEmblem(s, modEmblem(m), cx, y+30, 50, 1)
			if u.hover(cx-26, y+4, 52, 52) {
				g.tooltip = m.Name + " — " + m.Desc
			}
		}
	}
	y += 80
	// XP fills over a couple of seconds
	t := ease(clamp01((g.stateT - 0.4) / 2.0))
	xp := g.xpBefore + int(float64(g.xpGained)*t)
	rank := rankOf(xp)
	lo, hi := xpForRank(rank), xpForRank(rank+1)
	u.text(s, fmt.Sprintf("+%d XP", int(float64(g.xpGained)*t)), fBold, 24, lx, y, rgba(190, 160, 255, 255), alignCenter)
	u.medallion(s, lx-320, y+62, 28, rgba(150, 110, 255, 255))
	u.textOutlined(s, itoa(rank), fTitle, 28, lx-320, y+44, cWhite, alignCenter)
	u.bar(s, lx-270, y+52, 620, 20, float64(xp-lo)/float64(max(1, hi-lo)), 0, rgba(150, 110, 255, 255), 0, 0)
	y += 108
	// unlocks are revealed one at a time
	var news []*Mod
	if rank > g.rankBefore {
		pulse := 0.8 + 0.2*math.Sin(g.stateT*6)
		u.textOutlined(s, fmt.Sprintf("Rank Up!  Arcane Rank %d", rank), fRune, 36, lx, y, ca(cGold, pulse), alignCenter)
		y += 52
		for rk := g.rankBefore + 1; rk <= rank; rk++ {
			news = append(news, unlocksAt(rk)...)
		}
	}
	if len(news) > 0 {
		u.text(s, "New items join the pool:", fReg, 19, lx, y, cDim, alignCenter)
		y += 30
		nx := lx - float64(min(len(news), 6))*55
		for i, m := range news {
			if i >= 6 {
				break
			}
			k := ease(clamp01((g.stateT - 2.6 - float64(i)*0.35) / 0.4))
			if k <= 0 {
				continue
			}
			if k < 1 && g.stateT-2.6-float64(i)*0.35 < 1/60.0*2 {
				g.snd.PlayVol("pick.2.0", 0.4)
			}
			cx := nx + float64(i)*110 + 55
			drawSprite(s, sprSoft, u.X(cx-50), u.Y(y-10), 100*S, 100*S, ca(rarityColors[m.Rarity], 0.35*(1-k)+0.1), true)
			u.drawEmblem(s, modEmblem(m), cx, y+30+(1-k)*14, 58*k, k)
			u.text(s, m.Name, fReg, 14, cx, y+66, ca(cDim, k), alignCenter)
		}
		y += 96
	}
	u.iconLabel(s, icShard, fmt.Sprintf("+%d shards", g.shardsWon), fBold, 26, lx, y+4, rgba(200, 180, 255, 255), alignCenter)
	y += 40
	if g.ascUnlocked {
		u.textOutlined(s, fmt.Sprintf("Ascension %d unlocked!", g.save.AscMax), fBold, 24, lx, y, rgba(255, 140, 120, 255), alignCenter)
		y += 36
	}
	if r.Endless && !g.victory {
		if g.boardRank == 1 {
			pulse := 0.75 + 0.25*math.Sin(g.stateT*6)
			u.textOutlined(s, "New Endless Record!", fRune, 34, lx, y, ca(cGold, pulse), alignCenter)
		} else if g.boardRank > 0 {
			u.textOutlined(s, fmt.Sprintf("Endless leaderboard  ·  #%d", g.boardRank), fBold, 24, lx, y, rgba(255, 170, 130, 255), alignCenter)
		}
	}

	// ---- right column: damage and the story of the run
	rx, rw := 1210.0, 640.0
	u.text(s, "WHERE THE DAMAGE CAME FROM", fBold, 18, rx+rw/2, 190, lighten(u.theme().Accent, 0.3), alignCenter)
	yy := g.drawDmgBars(s, r, rx, 226, rw, 8, ease(clamp01(g.stateT/1.4)))
	yy = math.Max(yy+18, 500)
	u.text(s, "THE RUN", fBold, 18, rx+rw/2, yy, lighten(u.theme().Accent, 0.3), alignCenter)
	yy += 34
	g.drawTimeline(s, r, rx, yy, rw, 900-yy)

	// ---- buttons
	ready := g.stateT > 0.8
	playAgain := func(x float64) {
		if u.button(s, "again", "Play again", x, 960, 340, 60, cGreen, ready) {
			arch, ch := r.Arch, r.Challenge
			endless := r.EndlessStart
			g.run, g.level = nil, nil
			g.seedText, g.daily, g.endlessMode = "", false, endless
			g.startRunWith(arch, ch)
		}
	}
	if g.victory && !r.Endless {
		pulse := 0.5 + 0.5*math.Sin(g.stateT*4)
		drawSprite(s, sprSoft, u.X(1430), u.Y(930), 440*S, 120*S, ca(rgba(255, 120, 90, 255), 0.2+0.15*pulse), true)
		if u.button(s, "endless", "Continue: Endless", 1460, 960, 380, 60, rgba(255, 120, 90, 255), ready) {
			r.Endless = true
			r.NextAct(&g.save)
			g.beginWorld()
			g.victory = false
			g.setState(StActIntro)
			return
		}
		playAgain(975)
	} else if !r.Daily {
		playAgain(1500)
	}
	if u.button(s, "summenu", "Main menu", 450, 960, 340, 60, cAcc, ready) {
		g.run = nil
		g.level = nil
		g.setState(StMenu)
	}
}

// drawTimeline lists the run's key moments, grouped by act.
func (g *Game) drawTimeline(s *ebiten.Image, r *Run, x, y, w, h float64) {
	u := g.ui
	tl := r.rc().Timeline
	// keep the most telling moments if there are too many to show
	maxRows := int(h / 30)
	var show []recapEvent
	for _, e := range tl {
		if e.Kind != "item" || len(tl) <= maxRows {
			show = append(show, e)
		}
	}
	if len(show) > maxRows {
		show = show[len(show)-maxRows:]
	}
	lastAct := 0
	for _, e := range show {
		if e.Act != lastAct {
			lastAct = e.Act
			if y+28 > y+h {
				break
			}
			u.text(s, fmt.Sprintf("ACT %s", roman(e.Act)), fBold, 14, x, y+6, ca(cGold, 0.85), alignLeft)
			u.glowLine(s, x+70, x+w, y+14, 1, ca(cGold, 0.5), 0.6)
			y += 26
		}
		c := map[string]color.RGBA{"relic": rgba(255, 200, 110, 255), "fusion": rgba(255, 140, 220, 255), "boss": rgba(255, 110, 110, 255),
			"life": rgba(255, 90, 110, 255), "heart": rgba(255, 150, 190, 255), "curse": tagColors["curse"], "item": cDim}[e.Kind]
		if c.A == 0 {
			c = cDim
		}
		gl := e.Glyph
		if gl == "" {
			gl = "star"
		}
		u.drawEmblem(s, glyphImage(gl, c), x+14, y+11, 22, 1)
		u.text(s, e.Text, fReg, 17, x+34, y, c, alignLeft)
		y += 28
	}
}

func roman(n int) string {
	if n >= 1 && n <= 10 {
		return []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}[n-1]
	}
	return itoa(n)
}

// ---------------------------------------------------------------- codex

func (g *Game) drawHistory(s *ebiten.Image, y float64) {
	u := g.ui
	S := u.S
	w := 1500.0
	x := 960 - w/2
	u.panel(s, x, y, w, 720, 0.92)
	if len(g.save.History) == 0 {
		u.paragraph(s, "No finished runs yet. Every run you end is recorded here: origin, how far you got, your build and what did the damage.", fReg, 22, 960, y+320, 760, cDim, alignCenter)
		return
	}
	_, wy := ebiten.Wheel()
	rowH := 66.0
	maxScroll := math.Max(0, float64(len(g.save.History))*rowH-600)
	g.histScroll = math.Max(0, math.Min(maxScroll, g.histScroll-wy*60))
	clip := s.SubImage(imageRect(u.X(x), u.Y(y+20), u.X(x+w), u.Y(y+700))).(*ebiten.Image)
	for i, h := range g.save.History {
		ry := y + 30 + float64(i)*rowH - g.histScroll
		if ry < y-rowH || ry > y+720 {
			continue
		}
		f, _ := u.navReg("hist"+itoa(i), x+20, math.Max(y+20, math.Min(ry, y+700-rowH)), w-40, rowH-8, false)
		if f && u.nav.mode {
			if ry < y+30 {
				g.histScroll = math.Max(0, g.histScroll-rowH)
			} else if ry > y+700-rowH {
				g.histScroll = math.Min(maxScroll, g.histScroll+rowH)
			}
		}
		ci := classInfo[h.Arch]
		bg := ca(ci.Col, 0.08)
		if h.Won {
			bg = ca(cGold, 0.12)
		}
		fillPath(clip, boxPath(u.X(x+20), u.Y(ry), (w-40)*S, (rowH-8)*S, 10*S, false), bg, ca(bg, 0.4), vgrad(u.Y(ry), u.Y(ry+rowH)), false)
		if f && u.nav.mode {
			u.focusRing(clip, x+20, ry, w-40, rowH-8, 10)
		}
		u.drawEmblem(clip, glyphImage(ci.Glyph, ci.Col), x+56, ry+29, 38, 1)
		res := fmt.Sprintf("Fell in Act %d", h.Act)
		rc := rgba(255, 140, 140, 255)
		switch {
		case h.Won:
			res, rc = "Victory", cGold
		case h.Endless:
			res, rc = fmt.Sprintf("Endless loop %d", h.Loop), rgba(255, 170, 130, 255)
		}
		name := strings.ToUpper(h.Arch)
		if a := archByIDOK(h.Arch); a != nil {
			name = a.Name
		}
		u.text(clip, name, fSerif, 22, x+90, ry+6, lighten(ci.Col, 0.3), alignLeft)
		mode := h.Date
		if c := challengeByID[h.Challenge]; c != nil {
			mode += "  ·  " + c.Name
		} else if h.Asc > 0 {
			mode += fmt.Sprintf("  ·  Ascension %d", h.Asc)
		}
		u.text(clip, mode, fReg, 15, x+90, ry+34, cDim, alignLeft)
		u.text(clip, res, fBold, 20, x+420, ry+18, rc, alignLeft)
		u.text(clip, fmt.Sprintf("%s  ·  %d min  ·  score %d", plural(h.Depth, "chamber"), h.Minutes, h.Score), fReg, 16, x+620, ry+20, cDim, alignLeft)
		if h.Top != "" {
			u.text(clip, "top damage: "+h.Top, fReg, 15, x+1000, ry+4, dmgColor(h.Top), alignLeft)
		}
		for j, id := range h.Build {
			if j >= 8 {
				break
			}
			if m := modByID[id]; m != nil {
				u.drawEmblem(clip, modEmblem(m), x+1015+float64(j)*52, ry+40, 30, 1)
			}
		}
	}
}

func imageRect(x0, y0, x1, y1 float64) image.Rectangle {
	return image.Rect(int(x0), int(y0), int(x1), int(y1))
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return itoa(n) + " " + word + "s"
}

// drawDmgBars lists where the run's damage came from as bars; it returns
// the y below the last row.
func (g *Game) drawDmgBars(s *ebiten.Image, r *Run, rx, yy, rw float64, maxRows int, grow float64) float64 {
	u := g.ui
	S := u.S
	rows, total := dmgRows(r)
	if total <= 0 {
		u.text(s, "No damage dealt yet.", fReg, 18, rx+rw/2, yy+10, cDim, alignCenter)
		return yy + 32
	}
	lw := math.Min(150, rw*0.36)
	for i, row := range rows {
		if i >= maxRows {
			break
		}
		f := row.Amt / total
		c := dmgColor(row.Src)
		u.text(s, row.Src, fBold, 17, rx, yy, cWhite, alignLeft)
		u.text(s, fmt.Sprintf("%.0f%%", f*100), fBold, 17, rx+rw, yy, c, alignRight)
		bw := (rw - lw - 50) * f * grow
		fillPath(s, boxPath(u.X(rx+lw), u.Y(yy+4), (rw-lw-50)*S, 14*S, 7*S, false), rgba(0, 0, 0, 150), rgba(20, 16, 30, 150), grad{}, false)
		if bw > 2 {
			fillPath(s, boxPath(u.X(rx+lw), u.Y(yy+4), bw*S, 14*S, 7*S, false), lighten(c, 0.25), darken(c, 0.45), vgrad(u.Y(yy+4), u.Y(yy+18)), false)
		}
		yy += 32
	}
	return yy
}
