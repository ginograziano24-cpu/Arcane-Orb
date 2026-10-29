package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Challenge runs: hand-made rule sets for a story run. Each one bends a
// part of the game, and finishing the story under it earns shards, an
// achievement and a seal on the Challenges screen. They open once you
// have beaten any Guardian.

type challenge struct {
	ID, Name, Rules, Glyph string
	Col                    color.RGBA
	Stats                  func(s *Stats) // applied on every stat recompute
	Setup                  func(g *Game)  // once, when the run starts
	Map                    func(m *ActMap, rng *rand.Rand)
}

var challenges = []challenge{
	{ID: "onelife", Name: "One Life", Glyph: "heart", Col: rgba(255, 110, 150, 255),
		Rules: "A single life for the whole run, and no hearts will fall. Gold +50%.",
		Stats: func(s *Stats) { s.GoldMult += 0.5 }},
	{ID: "glass", Name: "Glass Paddle", Glyph: "glass", Col: rgba(170, 230, 255, 255),
		Rules: "Your paddle is 40% narrower. Critical chance +15% and critical hits +100% damage.",
		Stats: func(s *Stats) { s.PaddleWidth -= 0.4; s.CritChance += 0.15; s.CritMult += 1 }},
	{ID: "hexed", Name: "Hexed", Glyph: "mask", Col: rgba(200, 60, 120, 255),
		Rules: "Start with three curses that can never be cleansed, and a Rare relic of your choice.",
		Setup: func(g *Game) {
			rng := rand.New(rand.NewSource(g.run.SeedN ^ 0x4e5))
			for i := 0; i < 3; i++ {
				g.addCurse(rng)
			}
			g.toasts = nil
			g.rewards = []rewardSet{{Title: "Hexed — choose a relic", Mods: g.run.Offer(3, true, Rare, rng), Skip: 0}}
		}},
	{ID: "speed", Name: "Speed Demon", Glyph: "comet", Col: rgba(255, 170, 60, 255),
		Rules: "Orbs fly 40% faster from the first launch. Gold +50% and +10% damage.",
		Stats: func(s *Stats) { s.SpeedMult += 0.4; s.GoldMult += 0.5; s.DmgMult *= 1.1 }},
	{ID: "noshops", Name: "Wanderer", Glyph: "feather", Col: rgba(150, 220, 170, 255),
		Rules: "No shops anywhere: strange encounters take their place. Every reward offers one more choice.",
		Stats: func(s *Stats) { s.ExtraChoices++ },
		Map: func(m *ActMap, rng *rand.Rand) {
			for _, n := range m.Nodes {
				if n.Type == NShop {
					n.Type = NEvent
				}
			}
		}},
	{ID: "pyro", Name: "Pyromania", Glyph: "flame", Col: rgba(255, 110, 40, 255),
		Rules: "Only fire, blast and plain items are ever offered. You start with Kindling.",
		Setup: func(g *Game) { g.take(modByID["kindling"]); g.toasts = nil }},
	{ID: "gauntlet", Name: "Elite Gauntlet", Glyph: "crown", Col: rgba(220, 160, 255, 255),
		Rules: "Every fight on the map is an elite chamber. Elites pay twice the gold.",
		Map: func(m *ActMap, rng *rand.Rand) {
			for _, n := range m.Nodes {
				if n.Type == NCombat && n.Row > 0 {
					n.Type = NElite
				}
			}
		}},
	{ID: "unadorned", Name: "Unadorned", Glyph: "ring", Col: rgba(230, 210, 170, 255),
		Rules: "No relics at all, from any source: only upgrades. Every reward offers one more choice.",
		Stats: func(s *Stats) { s.ExtraChoices++ }},
	{ID: "lightsout", Name: "Lights Out", Glyph: "moon", Col: rgba(130, 140, 230, 255),
		Rules: "The arena is dark. Only your orbs, your paddle and the blasts you make give light."},
	{ID: "omens", Name: "Omen Storm", Glyph: "eye", Col: rgba(255, 220, 120, 255),
		Rules: "Every fight carries an omen, for better and for worse. Challenges met pay 25 extra gold.",
		Stats: func(s *Stats) { s.ChallengeGold += 25 },
		Map: func(m *ActMap, rng *rand.Rand) {
			for _, n := range m.Nodes {
				if (n.Type == NCombat && n.Row > 0) || n.Type == NElite {
					n.Omen = omens[rng.Intn(len(omens))].ID
				}
			}
		}},
}

var challengeByID = map[string]*challenge{}

func init() {
	for i := range challenges {
		challengeByID[challenges[i].ID] = &challenges[i]
	}
}

func (r *Run) challenge() *challenge { return challengeByID[r.Challenge] }
func (r *Run) is(id string) bool     { return r.Challenge == id }

// challengesOpen reports whether the Challenges menu is unlocked.
func (g *Game) challengesOpen() bool { return g.save.Bosses > 0 || g.save.Wins > 0 }

// applyChallengeMap reshapes a freshly generated act map.
func (r *Run) applyChallengeMap() {
	if c := r.challenge(); c != nil && c.Map != nil {
		c.Map(r.Map, rand.New(rand.NewSource(r.SeedN^int64(r.Act*977))))
	}
}

// offerAllowed filters item offers for challenges that restrict them.
func (r *Run) offerAllowed(m *Mod) bool {
	switch r.Challenge {
	case "pyro":
		for _, t := range m.Tags {
			if t == "frost" || t == "storm" || t == "void" {
				return false
			}
		}
	case "unadorned":
		return !m.Relic
	}
	return true
}

// beginChallenge starts a run under challenge id with origin arch.
func (g *Game) beginChallenge(arch *Archetype, id string) {
	g.seedText = ""
	g.daily, g.endlessMode = false, false
	g.challengeSel = ""
	asc := g.save.AscSel
	g.save.AscSel = 0 // challenges are their own difficulty
	g.startRunWith(arch, id)
	g.save.AscSel = asc
}

// onChallengeEnd records a finished challenge and pays out.
func (g *Game) onChallengeEnd(r *Run, win bool) {
	c := r.challenge()
	if c == nil {
		return
	}
	if g.save.Challenges == nil {
		g.save.Challenges = map[string]int{}
	}
	best := g.save.Challenges[c.ID]
	reached := r.Act
	if win {
		reached = numActs + 1
	}
	if reached > best {
		g.save.Challenges[c.ID] = reached
	}
	if win && best <= numActs {
		g.save.Shards += 150
		g.shardsWon += 150
		g.unlock("ACH_CHALLENGE")
		done := 0
		for _, ch := range challenges {
			if g.save.Challenges[ch.ID] > numActs {
				done++
			}
		}
		if done >= 5 {
			g.unlock("ACH_CHALLENGE_5")
		}
		if done >= len(challenges) {
			g.unlock("ACH_CHALLENGE_ALL")
		}
	}
}

// ---------------------------------------------------------------- screen

func (g *Game) drawChallenges(s *ebiten.Image) {
	u := g.ui
	S := u.S
	u.heading(s, "CHALLENGES", 72, 40, cGold, 1)
	u.text(s, "Story runs under special rules. Finish one for 150 shards and its seal.", fReg, 22, 960, 140, cDim, alignCenter)
	cols, w, h, gap := 5, 330.0, 360.0, 22.0
	x0 := 960 - (float64(cols)*w+float64(cols-1)*gap)/2
	for i := range challenges {
		c := &challenges[i]
		x := x0 + float64(i%cols)*(w+gap)
		y := 200 + float64(i/cols)*(h+gap)
		hv, clk := u.hit("ch"+c.ID, x, y, w, h)
		k := u.anim("ch"+c.ID, hv, 10)
		yy := y - 6*k
		done := g.save.Challenges[c.ID] > numActs
		bg0, bg1 := rgba(34, 24, 54, 240), rgba(12, 8, 22, 245)
		box := boxPath(u.X(x), u.Y(yy), w*S, h*S, 14*S, false)
		if k > 0.01 {
			drawSprite(s, sprSoft, u.X(x-40), u.Y(yy-30), (w+80)*S, (h+60)*S, ca(c.Col, 0.22*k), true)
		}
		fillPath(s, box, bg0, bg1, vgrad(u.Y(yy), u.Y(yy+h)), false)
		strokePath(s, box, (1.6+1.4*k)*S, ca(lighten(c.Col, 0.2*k), 0.6+0.4*k), false)
		if hv && u.nav.mode {
			u.focusRing(s, x, yy, w, h, 14)
		}
		// emblem in a medallion
		cx, cy := x+w/2, yy+86
		fillPath(s, circlePath(u.X(cx), u.Y(cy), 56*S), rgba(44, 30, 70, 255), rgba(14, 9, 26, 255), vgrad(u.Y(cy-56), u.Y(cy+56)), false)
		strokePath(s, circlePath(u.X(cx), u.Y(cy), 56*S), 3*S, ca(c.Col, 0.9), false)
		drawSprite(s, sprSoft, u.X(cx-60), u.Y(cy-60), 120*S, 120*S, ca(c.Col, 0.25+0.2*k), true)
		u.drawEmblem(s, glyphImage(c.Glyph, c.Col), cx, cy, 76, 1)
		u.text(s, c.Name, fSerif, 28, cx, yy+156, lighten(c.Col, 0.35), alignCenter)
		u.paragraph(s, c.Rules, fReg, 18, cx, yy+200, w-40, cDim, alignCenter)
		// status line
		st, sc := "Not attempted", ca(cDim, 0.7)
		switch b := g.save.Challenges[c.ID]; {
		case done:
			st, sc = "✦ SEALED ✦", cGold
		case b > 0:
			st, sc = fmt.Sprintf("Best: reached Act %d", b), cAcc
		}
		u.text(s, st, fBold, 17, cx, yy+h-38, sc, alignCenter)
		if done {
			u.drawEmblem(s, glyphImage("star", cGold), x+w-30, yy+28, 30, 1)
		}
		if clk {
			g.challengeSel = c.ID
			g.daily, g.endlessMode, g.seedText = false, false, ""
			g.snd.Play("click")
			g.setState(StArch)
		}
	}
	if u.button(s, "chback", "Back", 60, 990, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

// challengeBanner marks the origin screen when a challenge is chosen.
func (g *Game) challengeBanner(s *ebiten.Image) {
	c := challengeByID[g.challengeSel]
	if c == nil {
		return
	}
	u := g.ui
	t := "CHALLENGE  ·  " + strings.ToUpper(c.Name)
	w := u.measure(t, fBold, 20) + 90
	x := 960 - w/2
	u.frame(s, x, 1000, w, 48, 0.95)
	u.drawEmblem(s, glyphImage(c.Glyph, c.Col), x+32, 1024, 30, 1)
	u.text(s, t, fBold, 20, x+56, 1012, lighten(c.Col, 0.3), alignLeft)
	if u.hover(x, 1000, w, 48) {
		g.tooltip = c.Name + " — " + c.Rules
	}
}

// ---------------------------------------------------------------- lights out

var darkImg *ebiten.Image

// drawDarkness covers the arena in night, with holes of light around the
// orbs, the paddle and anything burning or exploding.
func (g *Game) drawDarkness(screen *ebiten.Image, L *Level, k float64) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if darkImg == nil || darkImg.Bounds().Dx() != sw || darkImg.Bounds().Dy() != sh {
		darkImg = ebiten.NewImage(sw, sh)
	}
	darkImg.Fill(color.RGBA{2, 1, 8, 238})
	cam := &g.cam
	cam.Roll = 0
	S := g.ui.S
	hole := func(p V3, rad, a float64) {
		x, y, z := cam.Project(p)
		if z <= 0.06 {
			return
		}
		x, y = x*k, y*k
		r := rad * S
		spr := sprSoft
		vs := []ebiten.Vertex{
			{DstX: float32(x - r), DstY: float32(y - r), SrcX: spr.X0, SrcY: spr.Y0},
			{DstX: float32(x + r), DstY: float32(y - r), SrcX: spr.X1, SrcY: spr.Y0},
			{DstX: float32(x + r), DstY: float32(y + r), SrcX: spr.X1, SrcY: spr.Y1},
			{DstX: float32(x - r), DstY: float32(y + r), SrcX: spr.X0, SrcY: spr.Y1},
		}
		for i := range vs {
			vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(a), float32(a), float32(a), float32(a)
		}
		darkImg.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, atlas, &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendDestinationOut, Filter: ebiten.FilterLinear})
	}
	for _, b := range L.Balls {
		hole(V3{b.X, 0.5, b.Z}, 230, 1)
		hole(V3{b.X, 0.5, b.Z}, 120, 1)
	}
	if L.paddleShown() > 0 {
		hole(V3{L.PaddleX, 0.3, PaddleZ}, 260, 1)
	}
	for _, d := range L.Drops {
		hole(V3{d.X, 0.6, d.Z}, 90, 0.9)
	}
	for _, rg := range L.Rings {
		hole(V3{rg.X, 0.3, rg.Z}, 120+rg.MaxR*40, clamp01(rg.Life/rg.Max))
	}
	for _, b := range L.Bricks {
		if b.Alive && b.Burn > 0 {
			hole(V3{b.X(), 0.8, b.Z()}, 80, 0.6)
		}
	}
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(darkImg, op)
}

func (g *Game) challengesDone() int {
	n := 0
	for _, c := range challenges {
		if g.save.Challenges[c.ID] > numActs {
			n++
		}
	}
	return n
}

func init() {
	// progress meters for the challenge achievements (set here to avoid an
	// initialisation cycle between the two tables)
	for i := range achievements {
		switch achievements[i].ID {
		case "ACH_CHALLENGE_5":
			achievements[i].Prog = func(g *Game) (int, int) { return g.challengesDone(), 5 }
		case "ACH_CHALLENGE_ALL":
			achievements[i].Prog = func(g *Game) (int, int) { return g.challengesDone(), len(challenges) }
		}
	}
}
