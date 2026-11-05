package main

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The hero Lore page: each origin's name and short story, opened from
// class select ("Lore") or by clicking an origin card in the codex.
// The story itself lives in docs/lore.md; keep the two in step.

type heroStory struct {
	Name    string   // the hero's own name
	Paras   []string // their story, a few short paragraphs
	Friends string   // one line on how they get on with the others
}

var heroLore = map[string]heroStory{
	"guardian": {"Dame Oriel", []string{
		"Oriel swore an oath to protect people. For years that meant guarding one quiet temple door. Then the walls came, and suddenly there was a lot more to protect.",
		"Catch the Orb dead centre and her shield answers with a wave of holy light. She is the responsible one: she packs the snacks, checks the map and knows everyone's bedtime.",
	}, "Thinks Brakka is reckless. Thinks Sol is a bad influence. Is right about both."},
	"berserker": {"Brakka", []string{
		"Brakka lay down for a nap one afternoon. When she woke up, a brick wall ran straight through the middle of her house.",
		"She has been hitting bricks ever since. Every brick she breaks makes her stronger, so she never slows down. Under all the noise she has the softest heart in the group, and she rebuilds every house she smashes through.",
	}, "Loves sparring with Oriel. Loves setting things on fire with Cinder even more."},
	"arcane": {"Professor Vell", []string{
		"Vell has read every spellbook in the land, some of them twice. He could never choose a favourite element, so his Orb cycles through all four: fire, frost, storm and void.",
		"He reads aloud to the group at camp every night. Nobody has ever asked him to.",
	}, "Argues with Rivet about magic versus machines. Loses about half the time."},
	"engineer": {"Rivet", []string{
		"Rivet grew up in the Machine Works, where the Crucible Golem forges every brick. She knows bricks the way bakers know bread.",
		"She built her laser paddle out of spare parts. It hits hardest when she holds still and thinks, which is her favourite thing to do anyway.",
	}, "Best friends with Zephyr, who has never held still in his life."},
	"lucky": {"Lucky Sol", []string{
		"Sol was halfway through a card game on a sky barge when the Orb fell out of the sky and landed on the table. He looked at it, looked at his cards, and declared it part of his winning hand.",
		"The more gold he carries, the harder he hits, so every run comes down to one question: spend it or keep it? He says luck is a skill.",
	}, "Owes everyone in the group money. Including the Orb."},
	"pyro": {"Cinder", []string{
		"Cinder grew up in a town full of forges and loves fire more than anything in the world. Bricks burn surprisingly well, if you ask her.",
		"Every few hits she sets a brick alight, and every hit on a burning brick makes it burn hotter. She says she is in it for the explosions. She is.",
	}, "Vell's worst student, and his favourite one. Brakka's partner in crime."},
	"storm": {"Zephyr", []string{
		"Zephyr grew up in the Cloud Kingdom, where you could see every star in the sky. Then the walls covered it, and he has not sat still since.",
		"Keep the paddle moving to build up static. When it is full, the Orb turns into a lightning storm, and for a few seconds Zephyr is very happy.",
	}, "Best friends with Rivet, the one hero who loves standing still."},
}

// openLore shows the Lore page for archetypes[i].
func (g *Game) openLore(i int) {
	g.loreSel = i
	g.loreOpen = true
	g.snd.Play("click")
	g.consumeInput()
}

func (g *Game) drawLore(s *ebiten.Image) {
	u := g.ui
	S := u.S
	n := len(archetypes)
	if g.loreSel < 0 || g.loreSel >= n {
		g.loreSel = 0
	}
	flip := func(d int) {
		g.loreSel = (g.loreSel + n + d) % n
		g.snd.Play("hover")
	}
	if g.navLatch.pageL {
		flip(-1)
	}
	if g.navLatch.pageR {
		flip(1)
	}
	a := &archetypes[g.loreSel]
	info := classInfo[a.ID]
	col := info.Col
	story := heroLore[a.ID]
	locked := a.Cost > 0 && !g.save.Unlocked[a.ID]

	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	fillRect(s, 0, 0, sw, sh, rgba(3, 2, 6, 190))
	drawSprite(s, sprSoft, u.X(500-520), u.Y(80), 1040*S, 1000*S, ca(col, 0.22), true)

	// portrait, standing on the bottom edge, breathing gently
	if img := portrait(a.ID); img != nil {
		ph := 960.0
		pw := ph * 0.75
		br := 1 + 0.004*math.Sin(g.stateT*1.7)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(pw*S/float64(img.Bounds().Dx()), ph*S/float64(img.Bounds().Dy()))
		op.GeoM.Translate(-pw*S/2, -ph*S)
		op.GeoM.Scale(br, br)
		op.GeoM.Translate(u.X(500), u.Y(1086))
		if locked {
			op.ColorScale.Scale(0.55, 0.55, 0.55, 1)
		}
		s.DrawImage(img, op)
	}

	// the story
	rx, ry, rw, rh := 900.0, 90.0, 900.0, 840.0
	u.frame(s, rx, ry, rw, rh, 0.9)
	cx := rx + 50
	u.text(s, strings.ToUpper(a.Name), fBold, 18, cx, ry+36, ca(cGold, 0.9), alignLeft)
	u.textOutlined(s, story.Name, fRune, 60, cx, ry+64, lighten(col, 0.35), alignLeft)
	y := ry + 150
	u.glowLine(s, cx, rx+rw-50, y, 1.5, ca(col, 0.8), 1)
	y += 30
	for _, p := range story.Paras {
		y = u.paragraph(s, p, fReg, 25, cx, y, rw-100, rgba(226, 220, 236, 255), alignLeft) + 22
	}
	if story.Friends != "" {
		y += 6
		u.glowLine(s, cx, cx+120, y, 1, ca(cGold, 0.7), 1)
		y += 22
		u.text(s, "FRIENDS & RIVALS", fBold, 15, cx, y, ca(cGold, 0.9), alignLeft)
		y += 28
		u.paragraph(s, story.Friends, fSerif, 23, cx, y, rw-100, rgba(236, 222, 196, 255), alignLeft)
	}
	if q := classQuote[a.ID]; q != "" {
		u.paragraph(s, "“"+tr(q)+"”", fSerif, 22, rx+rw/2, ry+rh-74, rw-120, ca(lighten(col, 0.4), 0.85), alignCenter)
	}

	// flip between heroes, and back
	if u.button(s, "loreprev", "‹", rx, ry+rh+30, 72, 60, cAcc, true) {
		flip(-1)
	}
	u.text(s, itoa(g.loreSel+1)+" / "+itoa(n), fBold, 20, rx+rw/2, ry+rh+46, cDim, alignCenter)
	if u.button(s, "lorenext", "›", rx+rw-72, ry+rh+30, 72, 60, cAcc, true) {
		flip(1)
	}
	if g.padActive {
		u.keycap(s, "LB", rx+86, ry+rh+44, 0.9)
		u.keycap(s, "RB", rx+rw-130, ry+rh+44, 0.9)
	}
	if u.button(s, "loreback", "Back", 60, 990, 200, 56, cAcc, true) {
		g.loreOpen = false
		g.consumeInput()
	}
}
