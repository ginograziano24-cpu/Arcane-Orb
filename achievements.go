package main

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Achievements live in the save file and, when the game runs through
// Steam, are mirrored to Steam under the same API names (see STEAM.md).

type achievement struct {
	ID, Name, Desc, Glyph string
	Hidden                bool
	// Prog returns progress toward a counted goal (cur, goal); nil = binary.
	Prog func(g *Game) (int, int)
}

var achievements = []achievement{
	{ID: "ACH_FIRST_CHAMBER", Name: "First Light", Desc: "Clear your first chamber.", Glyph: "star"},
	{ID: "ACH_GUARDIAN_1", Name: "Shattered Prism", Desc: "Defeat the Prism Colossus.", Glyph: "gem"},
	{ID: "ACH_GUARDIAN_2", Name: "The Forge Goes Cold", Desc: "Defeat the Crucible Golem.", Glyph: "anvil"},
	{ID: "ACH_GUARDIAN_3", Name: "Silence the Bell", Desc: "Defeat the Crystal Warden.", Glyph: "iceheart"},
	{ID: "ACH_WIN", Name: "The Last Warden", Desc: "Finish the story.", Glyph: "halo"},
	{ID: "ACH_WIN_GUARDIAN", Name: "Shield of the Wardens", Desc: "Finish the story as the Guardian.", Glyph: "shield"},
	{ID: "ACH_WIN_BERSERKER", Name: "Blood and Thunder", Desc: "Finish the story as the Berserker.", Glyph: "axe"},
	{ID: "ACH_WIN_ARCANE", Name: "Weaver's Heir", Desc: "Finish the story as the Arcanist.", Glyph: "orb"},
	{ID: "ACH_WIN_ENGINEER", Name: "Machine Spirit", Desc: "Finish the story as the Engineer.", Glyph: "gear"},
	{ID: "ACH_WIN_LUCKY", Name: "House Always Loses", Desc: "Finish the story as the Gambler.", Glyph: "clover"},
	{ID: "ACH_WIN_PYRO", Name: "Everything Burns", Desc: "Finish the story as the Pyromancer.", Glyph: "flame"},
	{ID: "ACH_WIN_STORM", Name: "The Sky Remembers", Desc: "Finish the story as the Stormcaller.", Glyph: "storm"},
	{ID: "ACH_WIN_ASC5", Name: "Ascendant", Desc: "Finish the story at Ascension 5 or higher.", Glyph: "crown"},
	{ID: "ACH_WIN_ASC10", Name: "Beyond the Veil", Desc: "Finish the story at Ascension 10.", Glyph: "dragonheart"},
	{ID: "ACH_FLAWLESS", Name: "Untouchable", Desc: "Defeat a Guardian without dropping an orb.", Glyph: "mirror"},
	{ID: "ACH_ENDLESS_3", Name: "Looping the Lattice", Desc: "Reach loop 3 in Endless Mode.", Glyph: "infinity",
		Prog: func(g *Game) (int, int) { return g.save.BestLoop, 3 }},
	{ID: "ACH_ENDLESS_10", Name: "Eternal Warden", Desc: "Reach loop 10 in Endless Mode.", Glyph: "blackhole",
		Prog: func(g *Game) (int, int) { return g.save.BestLoop, 10 }},
	{ID: "ACH_COMBO_50", Name: "Chain Reaction", Desc: "Reach a 50-hit combo.", Glyph: "chain",
		Prog: func(g *Game) (int, int) { return g.save.MaxCombo, 50 }},
	{ID: "ACH_COMBO_150", Name: "Unstoppable", Desc: "Reach a 150-hit combo.", Glyph: "burst",
		Prog: func(g *Game) (int, int) { return g.save.MaxCombo, 150 }},
	{ID: "ACH_FUSION", Name: "Alchemy", Desc: "Forge your first fusion.", Glyph: "flask"},
	{ID: "ACH_FUSION_ALL", Name: "Grand Alchemist", Desc: "Forge every fusion.", Glyph: "volcano",
		Prog: func(g *Game) (int, int) { return len(g.save.Fusions), len(fusions) }},
	{ID: "ACH_SYNERGY", Name: "In Harmony", Desc: "Awaken a synergy.", Glyph: "ring"},
	{ID: "ACH_ELITES_25", Name: "Elite Hunter", Desc: "Defeat 25 elite bricks.", Glyph: "fang",
		Prog: func(g *Game) (int, int) { return g.save.Elites + g.runElites(), 25 }},
	{ID: "ACH_BRICKS_10K", Name: "Lattice Breaker", Desc: "Break 10,000 bricks.", Glyph: "shatter",
		Prog: func(g *Game) (int, int) { return g.save.Bricks + g.runBricks(), 10000 }},
	{ID: "ACH_RANK_10", Name: "Adept", Desc: "Reach Arcane Rank 10.", Glyph: "book",
		Prog: func(g *Game) (int, int) { return rankOf(g.save.XP), 10 }},
	{ID: "ACH_RANK_25", Name: "Archmage", Desc: "Reach Arcane Rank 25.", Glyph: "moon",
		Prog: func(g *Game) (int, int) { return rankOf(g.save.XP), 25 }},
	{ID: "ACH_SANCTUM", Name: "Consecrated", Desc: "Fully empower one Sanctum blessing.", Glyph: "urn"},
	{ID: "ACH_ORIGINS", Name: "Every Path", Desc: "Unlock every origin.", Glyph: "mask",
		Prog: func(g *Game) (int, int) {
			n, have := 0, 0
			for _, a := range archetypes {
				if a.Cost > 0 {
					n++
					if g.save.Unlocked[a.ID] {
						have++
					}
				}
			}
			return have, n
		}},
	{ID: "ACH_GOLD_500", Name: "Dragon's Hoard", Desc: "Carry 500 gold at once.", Glyph: "coinstack"},
	{ID: "ACH_CODEX", Name: "Archivist", Desc: "Discover every upgrade and relic.", Glyph: "lens",
		Prog: func(g *Game) (int, int) {
			// signature items are mastery rewards, not part of the hunt
			n, seen := 0, 0
			for _, m := range modList {
				if m.Origin == "" && !m.Retired {
					n++
					if g.save.Seen[m.ID] {
						seen++
					}
				}
			}
			return seen, n
		}},
	{ID: "ACH_DAILY", Name: "Daily Devotion", Desc: "Complete a Daily Challenge run.", Glyph: "hourglass"},
	{ID: "ACH_CHALLENGE", Name: "Tested", Desc: "Finish a challenge run.", Glyph: "target"},
	{ID: "ACH_CHALLENGE_5", Name: "Proven", Desc: "Finish five different challenge runs.", Glyph: "crown"},
	{ID: "ACH_CHALLENGE_ALL", Name: "Master of Trials", Desc: "Finish every challenge run.", Glyph: "dragonheart"},
	{ID: "ACH_RAMPAGE", Name: "Rampage", Desc: "Fill the Berserker's Rage to the brim.", Glyph: "fist", Hidden: true},
	{ID: "ACH_SUPERCHARGE", Name: "Living Lightning", Desc: "Supercharge the orb as the Stormcaller.", Glyph: "bolt", Hidden: true},
}

var achByID = func() map[string]*achievement {
	m := map[string]*achievement{}
	for i := range achievements {
		m[achievements[i].ID] = &achievements[i]
	}
	return m
}()

// liveRun is the player's run in progress (not the attract-mode demo).
func (g *Game) liveRun() *Run {
	if g.run == nil || g.run.SeedCode == "MENU" {
		return nil
	}
	return g.run
}

func (g *Game) runElites() int {
	if r := g.liveRun(); r != nil {
		return r.Elites - r.statsPaid[2]
	}
	return 0
}

func (g *Game) runBricks() int {
	if r := g.liveRun(); r != nil {
		return r.Bricks - r.statsPaid[0]
	}
	return 0
}

// unlock awards an achievement once.
func (g *Game) unlock(id string) {
	if g.save.Ach == nil {
		g.save.Ach = map[string]bool{}
	}
	if g.save.Ach[id] || achByID[id] == nil {
		return
	}
	g.save.Ach[id] = true
	writeSave(&g.save)
	steamUnlock(id)
	if !steamActive() { // Steam shows its own popup
		g.achQueue = append(g.achQueue, id)
	}
}

// checkAchievements tests the progress-style goals; cheap, run each tick.
func (g *Game) checkAchievements() {
	g.achTick++
	if g.achTick%20 != 0 {
		return
	}
	for i := range achievements {
		a := &achievements[i]
		if a.Prog != nil && !g.save.Ach[a.ID] {
			if cur, goal := a.Prog(g); goal > 0 && cur >= goal {
				g.unlock(a.ID)
			}
		}
	}
	if r := g.liveRun(); r != nil {
		if r.MaxCombo >= 50 {
			g.unlock("ACH_COMBO_50")
		}
		if r.MaxCombo >= 150 {
			g.unlock("ACH_COMBO_150")
		}
		if len(r.Active) > 0 {
			g.unlock("ACH_SYNERGY")
		}
		if r.Gold >= 500 {
			g.unlock("ACH_GOLD_500")
		}
		if len(r.Fused) > 0 {
			g.unlock("ACH_FUSION")
		}
		if r.Endless && r.Loop() >= 3 {
			g.unlock("ACH_ENDLESS_3")
		}
		if r.Endless && r.Loop() >= 10 {
			g.unlock("ACH_ENDLESS_10")
		}
	}
	for _, m := range metaUpgrades {
		if g.save.Upg[m.ID] >= m.Max {
			g.unlock("ACH_SANCTUM")
			break
		}
	}
}

// onRunEnd awards the end-of-run achievements.
func (g *Game) onRunEnd(r *Run, win bool) {
	if r.Daily {
		g.unlock("ACH_DAILY")
	}
	if !win {
		return
	}
	g.unlock("ACH_WIN")
	g.unlock("ACH_WIN_" + strings.ToUpper(r.Arch.ID))
	if r.Asc >= 5 {
		g.unlock("ACH_WIN_ASC5")
	}
	if r.Asc >= 10 {
		g.unlock("ACH_WIN_ASC10")
	}
}

// drawAchievementPopup shows unlocks (when Steam isn't showing its own).
func (g *Game) drawAchievementPopup(dst *ebiten.Image) {
	if len(g.achQueue) == 0 {
		return
	}
	u := g.ui
	S := u.S
	g.achT += u.DT
	const dur = 3.6
	if g.achT > dur {
		g.achT = 0
		g.achQueue = g.achQueue[1:]
		return
	}
	if g.achT-u.DT <= 0 {
		g.snd.Play("achievement")
	}
	a := achByID[g.achQueue[0]]
	k := math.Min(ease(g.achT/0.35), ease((dur-g.achT)/0.4))
	w, h := 620.0, 104.0
	x, y := 960-w/2, 1080-h-40+(1-k)*(h+60)
	gold := rgba(255, 210, 110, 255)
	drawSprite(dst, sprSoft, u.X(x-60), u.Y(y-40), (w+120)*S, (h+80)*S, ca(gold, 0.18*k), true)
	u.frame(dst, x, y, w, h, 0.97*k)
	cx, cy := x+62, y+h/2
	fillPath(dst, circlePath(u.X(cx), u.Y(cy), 38*S), ca(rgba(40, 30, 60, 255), k), ca(rgba(14, 10, 24, 255), k), vgrad(u.Y(cy-38), u.Y(cy+38)), false)
	strokePath(dst, circlePath(u.X(cx), u.Y(cy), 38*S), 3*S, ca(gold, k), false)
	u.drawEmblem(dst, glyphImage(a.Glyph, gold), cx, cy, 52, k)
	u.text(dst, "ACHIEVEMENT UNLOCKED", fBold, 14, x+118, y+14, ca(gold, k), alignLeft)
	u.text(dst, a.Name, fSerif, 28, x+118, y+32, ca(cWhite, k), alignLeft)
	u.text(dst, a.Desc, fReg, 16, x+118, y+70, ca(cDim, k*0.9), alignLeft)
}

// drawAchievementsTab lists every achievement in the Codex.
func (g *Game) drawAchievementsTab(s *ebiten.Image, y0 float64) {
	u := g.ui
	S := u.S
	got := 0
	for _, a := range achievements {
		if g.save.Ach[a.ID] {
			got++
		}
	}
	u.text(s, itoa(got)+" / "+itoa(len(achievements))+" unlocked", fBold, 20, 960, y0-2, ca(cGold, 0.9), alignCenter)
	cols, w, h := 3, 560.0, 84.0
	x0 := 960 - (float64(cols)*w+float64(cols-1)*16)/2
	top, bot := y0+34, 966.0 // visible window; rows outside are clipped
	clip := s.SubImage(image.Rect(0, int(u.Y(top)), int(u.X(1920)), int(u.Y(bot)))).(*ebiten.Image)
	// scroll with the wheel, or follow the focused row
	_, wy := ebiten.Wheel()
	rows := (len(achievements) + cols - 1) / cols
	content := float64(rows)*(h+12) - 12
	maxScroll := math.Max(0, content-(bot-top))
	g.achScroll = math.Max(0, math.Min(maxScroll, g.achScroll-wy*60))
	for i, a := range achievements {
		x := x0 + float64(i%cols)*(w+16)
		y := top + float64(i/cols)*(h+12) - g.achScroll
		f, _ := u.navReg("ach"+a.ID, x, math.Max(top, math.Min(y, bot-h)), w, h, false)
		if f && u.nav.mode {
			if y < top {
				g.achScroll = math.Max(0, g.achScroll-math.Min(top-y, 40))
			}
			if y+h > bot {
				g.achScroll = math.Min(maxScroll, g.achScroll+math.Min(y+h-bot, 40))
			}
		}
		if y+h < top || y > bot {
			continue
		}
		s := clip
		have := g.save.Ach[a.ID]
		col := rgba(255, 210, 110, 255)
		bg0, bg1 := rgba(30, 22, 46, 235), rgba(12, 9, 22, 240)
		if !have {
			col = rgba(110, 100, 130, 255)
			bg0 = rgba(20, 16, 30, 220)
		}
		fillPath(s, boxPath(u.X(x), u.Y(y), w*S, h*S, 10*S, false), bg0, bg1, vgrad(u.Y(y), u.Y(y+h)), false)
		strokePath(s, boxPath(u.X(x), u.Y(y), w*S, h*S, 10*S, false), 1.5*S, ca(col, 0.7), false)
		if f && u.nav.mode {
			u.focusRing(s, x, y, w, h, 10)
		}
		if have {
			u.drawEmblem(s, glyphImage(a.Glyph, col), x+44, y+h/2, 54, 1)
		} else {
			u.drawEmblemDark(s, glyphImage(a.Glyph, col), x+44, y+h/2, 50, 1)
		}
		name, desc := a.Name, a.Desc
		if a.Hidden && !have {
			name, desc = "Hidden", "Keep playing to discover this one."
		}
		nc := cWhite
		if !have {
			nc = rgba(170, 162, 190, 255)
		}
		u.text(s, name, fSerif, 22, x+84, y+12, nc, alignLeft)
		u.text(s, desc, fReg, 16, x+84, y+44, cDim, alignLeft)
		if a.Prog != nil && !have {
			cur, goal := a.Prog(g)
			fr := clamp01(float64(cur) / float64(max(1, goal)))
			u.meter(s, x+w-170, y+20, 140, fr, rgba(170, 130, 255, 255))
			u.text(s, itoa(min(cur, goal))+" / "+itoa(goal), fBold, 15, x+w-30, y+36, cDim, alignRight)
		}
	}
	// scrollbar
	if maxScroll > 0 {
		bx, bh := x0+float64(cols)*(w+16)+4, bot-top
		fillPath(s, boxPath(u.X(bx), u.Y(top), 5*S, bh*S, 2.5*S, false), rgba(60, 50, 90, 160), rgba(60, 50, 90, 160), vgrad(u.Y(top), u.Y(bot)), false)
		tk := bh * (bh / content)
		ty := top + (bh-tk)*(g.achScroll/maxScroll)
		fillPath(s, boxPath(u.X(bx), u.Y(ty), 5*S, tk*S, 2.5*S, false), rgba(200, 170, 255, 230), rgba(140, 110, 220, 230), vgrad(u.Y(ty), u.Y(ty+tk)), false)
	}
}

// drawAchIcon draws an achievement's square medallion (the Steam icon art).
func drawAchIcon(u *UI, dst *ebiten.Image, a *achievement, x, y, sz float64) {
	S := u.S
	gold := rgba(255, 210, 110, 255)
	k := sz / 256
	cx, cy := x+sz/2, y+sz/2
	fillPath(dst, boxPath(u.X(x), u.Y(y), sz*S, sz*S, 0, false), rgba(46, 30, 82, 255), rgba(9, 6, 18, 255), vgrad(u.Y(y), u.Y(y+sz)), false)
	drawSprite(dst, sprSoft, u.X(cx-150*k), u.Y(cy-150*k), 300*k*S, 300*k*S, ca(rgba(150, 90, 255, 255), 0.55), true)
	// faint rune ring
	for i := 0; i < 24; i++ {
		an := float64(i) / 24 * 2 * math.Pi
		r := 116 * k
		fillPath(dst, circlePath(u.X(cx+math.Cos(an)*r), u.Y(cy+math.Sin(an)*r), 2.2*k*S), ca(gold, 0.45), ca(gold, 0.45), vgrad(0, 1), false)
	}
	fillPath(dst, circlePath(u.X(cx), u.Y(cy), 98*k*S), rgba(44, 32, 70, 255), rgba(12, 8, 24, 255), vgrad(u.Y(cy-98*k), u.Y(cy+98*k)), false)
	strokePath(dst, circlePath(u.X(cx), u.Y(cy), 98*k*S), 6*k*S, gold, false)
	strokePath(dst, circlePath(u.X(cx), u.Y(cy), 86*k*S), 1.5*k*S, ca(gold, 0.5), false)
	drawSprite(dst, sprSoft, u.X(cx-80*k), u.Y(cy-80*k), 160*k*S, 160*k*S, ca(gold, 0.25), true)
	u.drawEmblem(dst, glyphImage(a.Glyph, gold), cx, cy, 128*k, 1)
}
