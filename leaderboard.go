package main

import (
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Endless leaderboards: the world's best and your friends' best runs,
// shown on the main menu. They come from Steam when it is running; without
// Steam the panel shows your own local records instead.

const lbName = "Endless Best Run"

type lbRow struct {
	Rank, Score, Loop, Arch int
	Name                    string
	You                     bool
}

var lb struct {
	asked          bool
	found          bool
	global         []lbRow
	friends        []lbRow
	refreshAt      time.Time
	pendingScore   int
	pendingDetails []int32
}

// lbTick drives the Steam requests; cheap to call every frame.
func (g *Game) lbTick() {
	if !steamLBAvailable() {
		return
	}
	if !lb.asked {
		lb.asked = true
		steamLBFind(lbName)
	}
	steamLBPoll(func() {
		lb.found = true
		lbRefresh()
		if lb.pendingScore > 0 {
			steamLBUpload(lb.pendingScore, lb.pendingDetails)
			lb.pendingScore = 0
		}
	}, func(tag string, rows []lbRow) {
		if tag == "global" {
			lb.global = rows
		} else {
			lb.friends = rows
		}
	}, func() {
		lbRefresh()
	})
	if lb.found && nowFn().After(lb.refreshAt) && g.state == StMenu {
		lbRefresh()
	}
}

func lbRefresh() {
	lb.refreshAt = nowFn().Add(90 * time.Second)
	steamLBDownload(0, 1, 5)   // world top 5
	steamLBDownload(2, 1, 100) // all friends, ranked
}

// lbSubmit sends an Endless run's score (kept only if it beats your best).
func lbSubmit(r *Run, score int) {
	arch := 0
	for i := range archetypes {
		if archetypes[i].ID == r.Arch.ID {
			arch = i
		}
	}
	det := []int32{int32(r.Loop()), int32(r.Depth), int32(arch), int32(r.Asc)}
	if lb.found {
		steamLBUpload(score, det)
		return
	}
	if score > lb.pendingScore {
		lb.pendingScore, lb.pendingDetails = score, det
	}
}

// drawLeaderboard is the main-menu panel.
func (g *Game) drawLeaderboard(s *ebiten.Image, x, y, w float64) {
	u := g.ui
	S := u.S
	gold := rgba(255, 214, 130, 255)
	steamOn := steamLBAvailable()
	type section struct {
		title string
		rows  []lbRow
		empty string
	}
	var secs []section
	if steamOn {
		friends := lb.friends
		if len(friends) > 4 {
			// your friends' top three, plus you if you're further down
			top := append([]lbRow{}, friends[:3]...)
			for _, r := range friends[3:] {
				if r.You {
					top = append(top, r)
				}
			}
			friends = top
		}
		secs = append(secs, section{"WORLD", lb.global, "No runs yet. Be the first."}, section{"FRIENDS", friends, "No friends have a run yet."})
	} else {
		var rows []lbRow
		for i, e := range g.save.Board {
			if i >= 4 {
				break
			}
			rows = append(rows, lbRow{Rank: i + 1, Score: e.Score, Loop: e.Loop, Name: e.Class, You: true})
		}
		secs = append(secs, section{"YOUR BEST", rows, "No Endless runs yet."})
	}
	rowH := 30.0
	h := 58.0
	for _, sc := range secs {
		h += 30 + math.Max(1, float64(len(sc.rows)))*rowH + 6
	}
	if !steamOn {
		h += 26
	}
	sealed := g.save.Wins == 0
	if sealed {
		h += 24
	}
	hv, clk := u.hit("lbpanel", x, y, w, h)
	a := 0.9
	if hv {
		a = 1
		if u.nav.mode {
			u.focusRing(s, x, y, w, h, 12)
		}
	}
	fillPath(s, boxPath(u.X(x), u.Y(y), w*S, h*S, 12*S, false), rgba(26, 18, 44, uint8(215*a)), rgba(10, 7, 20, uint8(230*a)), vgrad(u.Y(y), u.Y(y+h)), false)
	strokePath(s, boxPath(u.X(x), u.Y(y), w*S, h*S, 12*S, false), 1.5*S, ca(gold, 0.45*a), false)
	u.text(s, "ENDLESS  ·  BEST RUNS", fSerif, 19, x+w/2, y+14, gold, alignCenter)
	u.glowLine(s, x+30, x+w-30, y+44, 1, rgba(196, 160, 92, 255), 0.6)
	cy := y + 56
	for _, sc := range secs {
		u.text(s, sc.title, fBold, 13, x+18, cy+6, rgba(190, 160, 255, 255), alignLeft)
		cy += 30
		if len(sc.rows) == 0 {
			u.text(s, sc.empty, fReg, 15, x+18, cy+4, cDim, alignLeft)
			cy += rowH
		}
		for _, r := range sc.rows {
			col := cWhite
			if r.You {
				col = gold
				fillPath(s, boxPath(u.X(x+8), u.Y(cy-2), (w-16)*S, (rowH-2)*S, 6*S, false), rgba(255, 200, 90, 30), rgba(255, 200, 90, 18), vgrad(0, 1), false)
			}
			u.text(s, itoa(r.Rank), fBold, 15, x+30, cy+4, ca(col, 0.8), alignRight)
			name := r.Name
			if steamOn && r.Arch >= 0 && r.Arch < len(archetypes) {
				u.drawEmblem(s, glyphImage(classInfo[archetypes[r.Arch].ID].Glyph, col), x+48, cy+13, 20, 0.9)
			}
			if u.measure(name, fReg, 16) > w-180 {
				rs := []rune(name)
				for len(rs) > 1 && u.measure(string(rs)+"…", fReg, 16) > w-180 {
					rs = rs[:len(rs)-1]
				}
				name = strings.TrimSpace(string(rs)) + "…"
			}
			u.text(s, name, fReg, 16, x+62, cy+3, col, alignLeft)
			right := itoa(r.Score)
			if r.Loop > 0 {
				right = "L" + itoa(r.Loop) + "  " + right
			}
			u.text(s, right, fBold, 15, x+w-16, cy+4, col, alignRight)
			cy += rowH
		}
		cy += 6
	}
	if !steamOn {
		u.text(s, "World and friend ranks appear on Steam", fReg, 13, x+w/2, cy, cDim, alignCenter)
		cy += 26
	}
	if sealed {
		u.text(s, "Finish the story to unlock Endless", fReg, 13, x+w/2, cy, ca(gold, 0.75), alignCenter)
	}
	if clk {
		g.codexTab = 4
		g.setState(StCodex)
	}
}
