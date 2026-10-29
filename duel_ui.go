package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// ---------------------------------------------------------------- the duel screen

func (g *Game) drawDuelScreen(s *ebiten.Image) {
	d := g.duel
	if d == nil {
		return
	}
	u := g.ui
	u.heading(s, "DUEL", 80, 60, cGold, 1)
	u.text(s, "Best of three races  ·  the same chambers  ·  the same origin and items", fReg, 22, 960, 170, cDim, alignCenter)
	switch d.phase {
	case "connect":
		g.drawDuelConnect(s)
	case "over":
		g.drawDuelOver(s)
	default:
		g.drawDuelLobby(s)
	}
}

// field draws a text box; it returns true when clicked.
func (g *Game) duelField(s *ebiten.Image, id, text, hint string, x, y, w float64, focused bool) bool {
	u := g.ui
	hv, clk := u.hit(id, x, y, w, 56)
	edge := rgba(120, 100, 170, 255)
	if focused {
		edge = cGold
	} else if hv {
		edge = lighten(edge, 0.3)
	}
	fillPath(s, boxPath(u.X(x), u.Y(y), w*u.S, 56*u.S, 10*u.S, false), rgba(8, 5, 18, 235), rgba(18, 12, 34, 235), vgrad(u.Y(y), u.Y(y+56)), false)
	strokePath(s, boxPath(u.X(x), u.Y(y), w*u.S, 56*u.S, 10*u.S, false), 2*u.S, edge, false)
	if text == "" && !focused {
		u.text(s, hint, fReg, 22, x+18, y+14, ca(cDim, 0.7), alignLeft)
	} else {
		cur := ""
		if focused && math.Mod(u.time, 1) < 0.55 {
			cur = "_"
		}
		u.text(s, text+cur, fBold, 26, x+18, y+11, cWhite, alignLeft)
	}
	return clk
}

func (g *Game) drawDuelConnect(s *ebiten.Image) {
	d := g.duel
	u := g.ui
	// left: Steam (or the test relay in browser builds)
	lx, rx, py, pw, ph := 250.0, 990.0, 230.0, 680.0, 640.0
	u.panel(s, lx, py, pw, ph, 0.9)
	u.panel(s, rx, py, pw, ph, 0.9)
	switch {
	case relayBuild:
		u.text(s, "TEST RELAY", fBold, 20, lx+pw/2, py+24, cGold, alignCenter)
		u.paragraph(s, "Browser builds connect through a local test relay. Pick any room code and share it.", fReg, 19, lx+pw/2, py+64, pw-80, cDim, alignCenter)
		if g.duelField(s, "dcode1", d.inputs[0], "Room code (e.g. TEST)", lx+40, py+150, pw-80, d.focus == 1) {
			d.focus = 1
		}
		if u.button(s, "drhost", "Host this room", lx+40, py+230, 290, 60, cGreen, d.link == nil && d.inputs[0] != "") {
			d.hostRelay(d.inputs[0])
		}
		if u.button(s, "drjoin", "Join this room", lx+pw-330, py+230, 290, 60, cAcc, d.link == nil && d.inputs[0] != "") {
			d.joinRelay(d.inputs[0])
		}
	case steamDuelAvailable():
		u.text(s, "STEAM", fBold, 20, lx+pw/2, py+24, cGold, alignCenter)
		u.paragraph(s, "Rooms on Steam work anywhere with no port forwarding. Host one and share the code, or join a friend's.", fReg, 19, lx+pw/2, py+64, pw-80, cDim, alignCenter)
		if d.steamCode != "" && sNetHost() {
			u.text(s, "YOUR ROOM CODE", fBold, 16, lx+pw/2, py+140, cDim, alignCenter)
			u.textOutlined(s, d.steamCode, fTitle, 64, lx+pw/2, py+160, cGold, alignCenter)
			u.text(s, "Waiting for your friend to join…", fReg, 19, lx+pw/2, py+246, ca(cDim, 0.6+0.4*math.Sin(u.time*3)), alignCenter)
		} else if u.button(s, "dshost", "Host a room", lx+pw/2-170, py+150, 340, 64, cGreen, d.link == nil) {
			d.steamWant = true
			steamDuelHost()
			d.status = "Opening a room on Steam…"
		}
		if g.duelField(s, "dcode1", d.inputs[0], "Friend's room code", lx+40, py+300, pw-280, d.focus == 1) {
			d.focus = 1
		}
		if u.button(s, "dsjoin", "Join", lx+pw-220, py+298, 180, 60, cAcc, d.link == nil) {
			g.duelJoinField(1)
		}
		u.text(s, "OPEN ROOMS", fBold, 16, lx+40, py+390, cDim, alignLeft)
		if u.button(s, "dsrefresh", "Refresh", lx+pw-180, py+380, 140, 40, cAcc, true) {
			d.steamWant = true
			steamDuelList("")
		}
		y := py + 432
		if len(d.steamRooms) == 0 {
			u.text(s, "No open rooms yet. Refresh to look again.", fReg, 18, lx+40, y+6, ca(cDim, 0.7), alignLeft)
		}
		for i, rm := range d.steamRooms {
			if i >= 4 {
				break
			}
			u.text(s, fallback(rm.Name, "Room")+"  ·  "+rm.Code, fBold, 20, lx+40, y+8, cWhite, alignLeft)
			if u.button(s, "dsroom"+itoa(i), "Join", lx+pw-160, y, 120, 44, cAcc, d.link == nil) {
				d.steamWant = true
				steamDuelJoin(rm.id)
				d.status = "Joining " + fallback(rm.Name, "the room") + "…"
			}
			y += 52
		}
	default:
		u.text(s, "STEAM", fBold, 20, lx+pw/2, py+24, cGold, alignCenter)
		u.paragraph(s, "Start Steam, then reopen this screen to host or join rooms with a short code: no port forwarding needed. You can still connect directly on the right.", fReg, 20, lx+pw/2, py+120, pw-100, cDim, alignCenter)
	}
	// right: direct / LAN
	u.text(s, "DIRECT  ·  LAN", fBold, 20, rx+pw/2, py+24, cGold, alignCenter)
	if relayBuild {
		u.paragraph(s, "Direct connections need the desktop build.", fReg, 20, rx+pw/2, py+120, pw-100, cDim, alignCenter)
	} else {
		if d.server == nil {
			u.paragraph(s, "Play on the same network, or over the internet if the host forwards port 7777.", fReg, 19, rx+pw/2, py+64, pw-80, cDim, alignCenter)
			if u.button(s, "ddhost", "Host directly", rx+pw/2-170, py+150, 340, 64, cGreen, d.link == nil) {
				d.hostDirect()
			}
		} else {
			u.text(s, "LAN CODE", fBold, 16, rx+40, py+80, cDim, alignLeft)
			u.textOutlined(s, fallback(d.lanCode, "—"), fTitle, 40, rx+40, py+100, cGold, alignLeft)
			u.text(s, "INTERNET CODE (port 7777 forwarded)", fBold, 16, rx+40, py+168, cDim, alignLeft)
			wc := d.wanCode
			if wc == "" && d.wanCh != nil {
				wc = "looking up…"
			}
			u.textOutlined(s, fallback(wc, "unavailable"), fTitle, 32, rx+40, py+188, lighten(cGold, 0.2), alignLeft)
		}
		if g.duelField(s, "dcode2", d.inputs[1], "Room code or IP address", rx+40, py+300, pw-280, d.focus == 2) {
			d.focus = 2
		}
		if u.button(s, "ddjoin", "Join", rx+pw-220, py+298, 180, 60, cAcc, d.link == nil && d.dialer == nil) {
			g.duelJoinField(2)
		}
		u.text(s, "ON YOUR NETWORK", fBold, 16, rx+40, py+390, cDim, alignLeft)
		y := py + 432
		rooms := d.browser.Rooms()
		if len(rooms) == 0 {
			msg := "Nobody is hosting on this network right now."
			if d.browser == nil {
				msg = "Room discovery is busy (another copy is open)."
			}
			u.text(s, msg, fReg, 18, rx+40, y+6, ca(cDim, 0.7), alignLeft)
		}
		for i, rm := range rooms {
			if i >= 4 {
				break
			}
			u.text(s, fallback(rm.Name, "Room")+"  ·  "+rm.Addr, fBold, 20, rx+40, y+8, cWhite, alignLeft)
			if u.button(s, "dlroom"+itoa(i), "Join", rx+pw-160, y, 120, 44, cAcc, d.link == nil && d.dialer == nil) {
				d.joinDirect(rm.Addr)
			}
			y += 52
		}
	}
	if d.status != "" {
		u.text(s, d.status, fBold, 22, 960, 900, rgba(255, 214, 160, 255), alignCenter)
	}
	if u.button(s, "dback", "Back", 60, 990, 200, 56, cAcc, true) {
		g.leaveDuel()
	}
}

func fallback(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func (g *Game) drawDuelLobby(s *ebiten.Image) {
	d := g.duel
	u := g.ui
	if d.oppName == "" {
		u.text(s, d.status, fBold, 24, 960, 500, ca(cDim, 0.6+0.4*math.Sin(u.time*3)), alignCenter)
		if u.button(s, "dleave", "Leave", 60, 990, 200, 56, cAcc, true) {
			g.leaveDuel()
		}
		return
	}
	// the two players
	card := func(x float64, name, tag string, ready bool, me bool) {
		u.panel(s, x, 240, 520, 220, 0.9)
		u.text(s, tag, fBold, 16, x+260, 262, cDim, alignCenter)
		u.textOutlined(s, name, fSerif, 40, x+260, 290, cGold, alignCenter)
		st, c := "Choosing…", cDim
		if ready {
			st, c = "READY", cGreen
		}
		if (me && d.host) || (!me && !d.host) {
			st, c = "HOST", rgba(255, 214, 140, 255)
		}
		u.text(s, st, fBold, 24, x+260, 372, c, alignCenter)
	}
	card(320, d.myName, "YOU", d.ready, true)
	card(1080, d.oppName, "OPPONENT", d.oppReady, false)
	u.textOutlined(s, "VS", fTitle, 64, 960, 310, rgba(255, 120, 160, 255), alignCenter)
	u.text(s, fmt.Sprintf("%s  ·  %.0f ms", strings.ToUpper(d.via), d.ping), fReg, 18, 960, 400, cDim, alignCenter)
	// origin (the host chooses for both)
	a := &archetypes[d.archIdx]
	u.panel(s, 560, 490, 800, 300, 0.9)
	u.text(s, "ORIGIN FOR BOTH PLAYERS", fBold, 16, 960, 510, cDim, alignCenter)
	u.textOutlined(s, a.Name, fSerif, 44, 960, 536, cWhite, alignCenter)
	u.paragraph(s, a.Passive+": "+a.Desc, fReg, 18, 960, 600, 700, cDim, alignCenter)
	if d.host {
		if u.button(s, "dprev", "‹", 590, 540, 70, 60, cAcc, true) {
			d.archIdx = (d.archIdx + len(archetypes) - 1) % len(archetypes)
			d.setupAt = time.Time{}
		}
		if u.button(s, "dnext", "›", 1260, 540, 70, 60, cAcc, true) {
			d.archIdx = (d.archIdx + 1) % len(archetypes)
			d.setupAt = time.Time{}
		}
	}
	// the kit
	var kit []string
	for i, ids := range duelKit {
		var names []string
		for _, id := range ids {
			if m := modByID[id]; m != nil {
				names = append(names, m.Name)
			}
		}
		kit = append(kit, fmt.Sprintf("Round %d: +%s", i+1, strings.Join(names, ", ")))
	}
	u.text(s, strings.Join(kit, "    "), fReg, 18, 960, 740, rgba(220, 210, 240, 255), alignCenter)
	// controls
	if d.host {
		label := "Waiting for " + d.oppName + " to ready up"
		if d.oppReady {
			label = "Start the duel"
		}
		if u.button(s, "dstart", label, 660, 850, 600, 70, cGreen, d.oppReady) {
			d.wins = [2]int{}
			d.times = [duelRounds][2]float64{}
			d.note = ""
			seed := randSeed()
			d.send(duelMsg{T: "start", Rd: 1, S: seed, A: archetypes[d.archIdx].ID}, true)
			g.duelBeginRound(1, seed)
			return
		}
	} else {
		label := "Ready"
		if d.ready {
			label = "Ready  ✓  (click to cancel)"
		}
		if u.button(s, "dready", label, 660, 850, 600, 70, cGreen, true) {
			d.ready = !d.ready
			d.send(duelMsg{T: "ready", R: d.ready}, true)
		}
	}
	if u.button(s, "dleave", "Leave", 60, 990, 200, 56, cAcc, true) {
		g.leaveDuel()
	}
}

func (g *Game) drawDuelOver(s *ebiten.Image) {
	d := g.duel
	u := g.ui
	me, op := d.me(), 1-d.me()
	won := d.wins[me] >= duelToWin || (d.wins[me] > d.wins[op] && d.lost)
	title, col := "DEFEAT", rgba(255, 110, 120, 255)
	if won {
		title, col = "VICTORY", cGold
	}
	pulse := 0.5 + 0.5*math.Sin(u.time*2)
	drawSprite(s, sprSoft, u.X(960-500), u.Y(200), 1000*u.S, 300*u.S, ca(col, 0.15+0.1*pulse), true)
	u.textOutlined(s, title, fTitle, 110, 960, 230, col, alignCenter)
	u.text(s, fmt.Sprintf("%s  %d  –  %d  %s", d.myName, d.wins[me], d.wins[op], fallback(d.oppName, "Opponent")), fBold, 30, 960, 380, cWhite, alignCenter)
	if d.note != "" {
		u.text(s, d.note, fReg, 22, 960, 430, cDim, alignCenter)
	}
	u.panel(s, 610, 480, 700, 250, 0.9)
	for i := 0; i < duelRounds; i++ {
		y := 510 + float64(i)*66
		t := d.times[i]
		if t[0] == 0 && t[1] == 0 {
			u.text(s, fmt.Sprintf("Round %d  ·  not played", i+1), fReg, 22, 960, y, ca(cDim, 0.6), alignCenter)
			continue
		}
		who := d.winners[i]
		name := d.myName
		if who != me {
			name = fallback(d.oppName, "Opponent")
		}
		u.text(s, fmt.Sprintf("Round %d  ·  %s cleared it in %.1f s", i+1, name, t[who]), fBold, 22, 960, y, cWhite, alignCenter)
	}
	if !d.lost && d.link != nil {
		label := "Rematch"
		switch {
		case d.rematch && !d.oppRematch:
			label = "Waiting for " + fallback(d.oppName, "them") + "…"
		case d.oppRematch && !d.rematch:
			label = "Rematch  (" + fallback(d.oppName, "they") + " wants one)"
		}
		if u.button(s, "drematch", label, 660, 790, 600, 70, cGreen, !d.rematch) {
			d.rematch = true
			d.send(duelMsg{T: "rematch", R: true}, true)
		}
	}
	if u.button(s, "dmenu", "Main menu", 760, 890, 400, 60, cAcc, true) {
		g.leaveDuel()
	}
}

// ---------------------------------------------------------------- in-chamber overlays

// drawDuelHUD adds the race bar, the opponent's board and the round
// banners over a duel chamber.
func (g *Game) drawDuelHUD(s *ebiten.Image, L *Level) {
	d := g.duel
	if d == nil {
		return
	}
	u := g.ui
	S := u.S
	me, op := d.me(), 1-d.me()
	// race bar under the wave strip
	myP := duelProgress(L.wave, L.waves, L.Left, L.startLeft, L.Cleared)
	o := &d.opp
	opP := duelProgress(o.wave, o.waves, o.left, o.start, o.done)
	bx, by, bw := 960.0-300, 156.0, 600.0
	u.frame(s, bx-20, by-8, bw+40, 74, 0.88)
	bar := func(y, p float64, c color.RGBA, label string) {
		fillPath(s, boxPath(u.X(bx+110), u.Y(y), (bw-170)*S, 14*S, 7*S, false), rgba(0, 0, 0, 170), rgba(20, 14, 30, 170), grad{}, false)
		if p > 0.01 {
			fillPath(s, boxPath(u.X(bx+110), u.Y(y), (bw-170)*p*S, 14*S, 7*S, false), lighten(c, 0.25), darken(c, 0.3), grad{}, false)
		}
		u.text(s, label, fBold, 15, bx, y-3, c, alignLeft)
		u.text(s, fmt.Sprintf("%.0f%%", p*100), fBold, 15, bx+bw, y-3, c, alignRight)
	}
	bar(by+4, myP, rgba(120, 220, 255, 255), "YOU")
	bar(by+32, opP, rgba(255, 130, 170, 255), strings.ToUpper(trimTo(fallback(d.oppName, "RIVAL"), 8)))
	// round pips
	for i := 0; i < duelToWin; i++ {
		on := func(w int) color.RGBA {
			if i < w {
				return cGold
			}
			return rgba(60, 50, 80, 255)
		}
		u.icon(s, icStar, bx-60+float64(i)*22, by+11, 18, 1)
		fillPath(s, circlePath(u.X(bx-60+float64(i)*22), u.Y(by+11), 7*S), on(d.wins[me]), on(d.wins[me]), grad{}, false)
		fillPath(s, circlePath(u.X(bx-60+float64(i)*22), u.Y(by+39), 7*S), on(d.wins[op]), on(d.wins[op]), grad{}, false)
	}
	u.text(s, fmt.Sprintf("ROUND %d  ·  %.1f s", d.round, d.elapsed), fBold, 15, 960, by+52, cDim, alignCenter)
	g.drawGhostBoard(s, 1556, 250, 334, 560)
	// countdown
	switch d.phase {
	case "countdown":
		n := int(math.Ceil(d.t))
		f := d.t - math.Floor(d.t)
		sz := 160 + 80*f
		u.textOutlined(s, itoa(n), fTitle, sz, 960, 420-sz/2, ca(cGold, 0.4+0.6*f), alignCenter)
		u.text(s, fmt.Sprintf("Round %d of %d", d.round, duelRounds), fSerif, 30, 960, 600, cWhite, alignCenter)
	case "play":
		if d.elapsed < 0.8 {
			a := 1 - d.elapsed/0.8
			u.textOutlined(s, "GO!", fTitle, 150+60*(1-a), 960, 340, ca(cGold, a), alignCenter)
		}
		if o.done && !d.finished {
			u.textOutlined(s, fallback(d.oppName, "Your rival")+" has cleared the chamber!", fBold, 26, 960, 860, ca(rgba(255, 130, 170, 255), 0.7+0.3*math.Sin(u.time*6)), alignCenter)
		}
	case "roundover":
		who := d.winners[max(0, d.round-1)]
		msg, c := "ROUND TO "+strings.ToUpper(fallback(d.oppName, "RIVAL")), rgba(255, 130, 170, 255)
		if who == me {
			msg, c = "ROUND WON", cGold
		}
		fillRect(s, 0, 0, u.SW, u.SH, color.RGBA{6, 4, 14, 120})
		u.textOutlined(s, msg, fTitle, 96, 960, 380, c, alignCenter)
		u.text(s, fmt.Sprintf("%.1f s", d.times[max(0, d.round-1)][who]), fBold, 32, 960, 500, cWhite, alignCenter)
		u.text(s, fmt.Sprintf("%d  –  %d", d.wins[me], d.wins[op]), fTitle, 54, 960, 550, cWhite, alignCenter)
	}
	if d.menuOpen {
		fillRect(s, 0, 0, u.SW, u.SH, color.RGBA{6, 4, 14, 170})
		u.heading(s, "DUEL", 72, 300, cGold, 1)
		u.text(s, "The race keeps running while this is open.", fReg, 22, 960, 400, cDim, alignCenter)
		if u.button(s, "dresume", "Back to the race", 760, 460, 400, 64, cGreen, true) {
			d.menuOpen = false
		}
		if u.button(s, "dff", "Forfeit the match", 760, 550, 400, 64, cRed, true) {
			d.send(duelMsg{T: "ff"}, true)
			d.menuOpen = false
			d.note = "You forfeited."
			d.wins[op] = duelToWin
			g.duelMatchOver()
		}
	}
}

func trimTo(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// drawGhostBoard draws the opponent's arena from above.
func (g *Game) drawGhostBoard(s *ebiten.Image, x, y, w, h float64) {
	d := g.duel
	u := g.ui
	S := u.S
	o := &d.opp
	u.frame(s, x, y, w, h, 0.9)
	u.text(s, trimTo(fallback(d.oppName, "Rival"), 14), fSerif, 24, x+w/2, y+14, rgba(255, 170, 200, 255), alignCenter)
	u.text(s, fmt.Sprintf("WAVE %d / %d  ·  COMBO %d", max(1, o.wave), max(1, o.waves), o.combo), fBold, 16, x+w/2, y+46, cDim, alignCenter)
	ax, ay, aw, ah := x+16, y+74, w-32, h-90
	fillPath(s, boxPath(u.X(ax), u.Y(ay), aw*S, ah*S, 6*S, false), rgba(10, 8, 22, 230), rgba(20, 14, 38, 230), vgrad(u.Y(ay), u.Y(ay+ah)), false)
	if o.pulse > 0 {
		strokePath(s, boxPath(u.X(ax), u.Y(ay), aw*S, ah*S, 6*S, false), 3*S, ca(cGold, o.pulse), false)
	}
	const zTop, zBot = FarZ, 0.4
	px := func(wx float64) float64 { return ax + (wx+HalfW)/(2*HalfW)*aw }
	pz := func(wz float64) float64 { return ay + (zTop-wz)/(zTop-zBot)*ah }
	cw := CellW / (2 * HalfW) * aw
	cd := CellD / (zTop - zBot) * ah
	for r := 0; r < levelgen.MaxRows; r++ {
		for c := 0; c < levelgen.Cols; c++ {
			n := o.cells[r*levelgen.Cols+c]
			if n == 0 {
				continue
			}
			col := o.colors[r*levelgen.Cols+c]
			if col.A == 0 {
				col = rgba(170, 150, 210, 255)
			}
			col = lighten(col, math.Min(0.4, 0.1*float64(n-1)))
			cx := px(cellXRaw(c))
			cz := pz(GridTopZ - o.shift - (float64(r)+0.5)*CellD)
			fillPath(s, boxPath(u.X(cx-cw/2+1), u.Y(cz-cd/2+1), (cw-2)*S, (cd-2)*S, 2*S, false), col, darken(col, 0.3), grad{}, false)
		}
	}
	// danger line, paddle, orbs
	strokePath(s, linePath(u.X(ax), u.Y(pz(dangerZ)), u.X(ax+aw), u.Y(pz(dangerZ))), 1.5*S, ca(rgba(255, 80, 70, 255), 0.5), false)
	if o.pw > 0 {
		fillPath(s, boxPath(u.X(px(o.px-o.pw/2)), u.Y(pz(PaddleZ)-3), (o.pw/(2*HalfW)*aw)*S, 6*S, 3*S, false), rgba(255, 220, 160, 255), rgba(200, 140, 80, 255), grad{}, false)
	}
	for i := 0; i+1 < len(o.balls); i += 2 {
		bx, bz := px(float64(o.balls[i])), pz(float64(o.balls[i+1]))
		drawSprite(s, sprSoft, u.X(bx-10), u.Y(bz-10), 20*S, 20*S, ca(rgba(190, 170, 255, 255), 0.8), true)
		fillPath(s, circlePath(u.X(bx), u.Y(bz), 3.5*S), cWhite, cWhite, grad{}, false)
	}
	if o.done {
		fillPath(s, boxPath(u.X(ax), u.Y(ay+ah/2-40), aw*S, 80*S, 0, false), rgba(0, 0, 0, 170), rgba(0, 0, 0, 170), grad{}, false)
		u.textOutlined(s, "CLEARED", fTitle, 40, x+w/2, ay+ah/2-26, cGold, alignCenter)
	}
}

// cellXRaw is a column's centre (columns never shift).
func cellXRaw(c int) float64 { return -HalfW + (float64(c)+0.5)*CellW }

// drawDuelVitals replaces the health plate during a duel.
func (g *Game) drawDuelVitals(s *ebiten.Image, x, y float64) {
	d := g.duel
	u := g.ui
	u.frame(s, x, y, 440, 112, 0.92)
	u.text(s, fmt.Sprintf("DUEL  ·  ROUND %d OF %d", d.round, duelRounds), fBold, 18, x+30, y+18, cGold, alignLeft)
	u.text(s, "Clear both waves first to take the round.", fReg, 16, x+30, y+50, cDim, alignLeft)
	u.text(s, "Esc: forfeit menu (the race keeps going)", fReg, 16, x+30, y+78, ca(cDim, 0.7), alignLeft)
}
