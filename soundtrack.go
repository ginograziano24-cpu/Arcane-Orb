package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/mus"
)

// The soundtrack (ROADMAP-50 #43): a Codex tab listing every song by name,
// where it plays, and a button to listen to it. An act's songs stay hidden
// until a run has reached that act, so the list doubles as a small reward
// for getting further.

// trackTitles names every track by number (see trackSongs).
func trackTitle(i int) string {
	switch i {
	case 0:
		return "Orb Rush"
	case 1:
		return "Acid Cascade"
	case 2:
		return "Abyss Engine"
	case 3:
		return "Arcane Groove"
	case trackBoss:
		return "The Warden"
	}
	if i >= 0 && i < len(trackSongs) {
		if t := mus.Title(trackSongs[i]); t != "" {
			return t
		}
	}
	return "?"
}

// soundtrackOrder is the order the Codex lists the tracks in: the menu
// theme, each act's family of three, then the Guardian theme.
var soundtrackOrder = []int{3, 0, 5, 6, 1, 7, 8, 2, 9, 10, trackBoss}

// trackAct is the act a track belongs to (0..2), -1 for the menu theme and
// 3 for the Guardian theme.
func trackAct(i int) int {
	switch {
	case i == 3:
		return -1
	case i == trackBoss:
		return 3
	case i < 3:
		return i
	}
	return (i - 5) / 2
}

// trackUnlocked: the menu theme always; an act's songs once a run has
// reached that act; the Guardian theme after the first run.
func (g *Game) trackUnlocked(i int) bool {
	switch a := trackAct(i); {
	case a < 0:
		return true
	case a == 3:
		return g.save.Runs > 0
	default:
		return g.save.BestAct >= a+1
	}
}

func trackWhere(i int) string {
	switch a := trackAct(i); {
	case a < 0:
		return "Main menu"
	case a == 3:
		return "Guardian fights"
	default:
		return fmt.Sprintf("Act %s  ·  %s", roman(a+1), biomes[a].Name)
	}
}

// drawSoundtrackTab draws the jukebox. g.jukebox is the track the player
// picked while g.jukeboxOn; otherwise the menu theme plays as usual.
func (g *Game) drawSoundtrackTab(s *ebiten.Image, y float64) {
	u := g.ui
	w := 1100.0
	x := 960 - w/2
	u.panel(s, x, y, w, 730, 0.92)
	u.text(s, "THE SOUNDTRACK", fBold, 22, 960, y+22, lighten(u.theme().Accent, 0.3), alignCenter)
	if g.settings.Music <= 0 || g.settings.Master <= 0 {
		u.text(s, "Music is turned off in Settings > Audio.", fReg, 18, 960, y+56, cRed, alignCenter)
	} else {
		u.text(s, "Every chamber plays the next song of its act's family, the way Magic Ball did.", fReg, 18, 960, y+56, cDim, alignCenter)
	}
	ry := y + 100
	rowH := 55.0
	lastAct := -2
	for _, ti := range soundtrackOrder {
		a := trackAct(ti)
		if a != lastAct && a >= 0 && a < 3 && lastAct != -2 {
			ry += 6 // a little air between the act families
		}
		lastAct = a
		open := g.trackUnlocked(ti)
		playing := g.jukeboxOn && g.jukebox == ti
		rx := x + 40
		if playing {
			fillPath(s, boxPath(u.X(rx-12), u.Y(ry-6), (w-56)*u.S, (rowH-4)*u.S, 10*u.S, false), ca(cGold, 0.16), ca(cGold, 0.06), vgrad(u.Y(ry-6), u.Y(ry+rowH-10)), false)
		}
		// a small equaliser while a track plays, a note glyph otherwise
		if playing {
			for k := 0; k < 4; k++ {
				hh := 6 + 14*math.Abs(math.Sin(g.stateT*(5+float64(k)*1.7)+float64(k)))
				fillRect(s, u.X(rx+float64(k)*7), u.Y(ry+30-hh), 5*u.S, hh*u.S, cGold)
			}
		} else {
			c := cDim
			if !open {
				c = ca(cDim, 0.4)
			}
			u.drawEmblem(s, glyphImage("note", c), rx+12, ry+20, 26, 1)
		}
		if !open {
			u.text(s, "? ? ?", fSerif, 24, rx+48, ry+2, ca(cDim, 0.5), alignLeft)
			hint := fmt.Sprintf("Reach Act %s to hear it", roman(a+1))
			if a == 3 {
				hint = "Finish a run to hear it"
			}
			u.text(s, hint, fReg, 16, x+w-40, ry+10, ca(cDim, 0.6), alignRight)
			ry += rowH
			continue
		}
		tc := cWhite
		if playing {
			tc = cGold
		}
		u.text(s, trackTitle(ti), fSerif, 24, rx+48, ry+2, tc, alignLeft)
		u.text(s, trackWhere(ti), fReg, 16, rx+340, ry+10, cDim, alignLeft)
		label := "Listen"
		if playing {
			label = "Stop"
		}
		if u.button(s, "trk"+itoa(ti), label, x+w-190, ry, 150, 42, cAcc, true) {
			g.jukebox, g.jukeboxOn = ti, !playing
		}
		ry += rowH
	}
}
