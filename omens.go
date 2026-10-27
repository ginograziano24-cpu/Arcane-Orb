package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Omens mark some chambers on the map. A few change the chamber itself
// (gilded, frenzied, volatile...), others set an optional challenge with a
// prize. They're visible before you choose a path, so they shape routing.

type omen struct {
	ID, Name, Desc, Glyph string
	Col                   color.RGBA
	Challenge             bool
	Setup                 func(L *Level)
	Done                  func(L *Level) bool // challenge met (checked on clear)
	Status                func(L *Level) string
	Reward                func(g *Game, rng *rand.Rand) string
}

var omens = []omen{
	{ID: "gilded", Name: "Gilded", Desc: "Bricks shed twice the gold.", Glyph: "coinstack", Col: rgba(255, 210, 80, 255)},
	{ID: "frenzy", Name: "Frenzied", Desc: "Orbs fly 15% faster and hit 30% harder.", Glyph: "burst", Col: rgba(255, 100, 70, 255)},
	{ID: "fortified", Name: "Fortified", Desc: "Bricks have 40% more health. The reward is one rarity better.", Glyph: "shield", Col: rgba(180, 190, 220, 255)},
	{ID: "bountiful", Name: "Bountiful", Desc: "Power-ups drop twice as often.", Glyph: "clover", Col: rgba(120, 230, 140, 255)},
	{ID: "volatile", Name: "Volatile", Desc: "Many bricks are packed with blasting powder.", Glyph: "bomb", Col: rgba(255, 150, 60, 255),
		Setup: func(L *Level) { L.convertBricks(levelgen.Explosive, 0.14) }},
	{ID: "seam", Name: "Crystal Seam", Desc: "Gold-rich crystals grow among the bricks.", Glyph: "gem", Col: rgba(140, 220, 255, 255),
		Setup: func(L *Level) { L.convertBricks(levelgen.Crystal, 0.08) }},
	{ID: "twin", Name: "Twin Moons", Desc: "You serve with an extra orb.", Glyph: "twinorb", Col: rgba(220, 200, 255, 255)},
	{ID: "swift", Name: "Against the Clock", Desc: "Challenge: clear within 70 seconds for 50 gold.", Glyph: "hourglass", Col: rgba(255, 220, 140, 255), Challenge: true,
		Done:   func(L *Level) bool { return L.T <= 70 },
		Status: func(L *Level) string { return fmt.Sprintf("%.0fs left", math.Max(0, 70-L.T)) },
		Reward: func(g *Game, rng *rand.Rand) string { g.run.Gold += 50; return "+50 gold" }},
	{ID: "untouched", Name: "Untouchable", Desc: "Challenge: don't drop an orb to earn a scroll.", Glyph: "mirror", Col: rgba(170, 230, 255, 255), Challenge: true,
		Done: func(L *Level) bool { return L.lostOrbs == 0 },
		Status: func(L *Level) string {
			if L.lostOrbs == 0 {
				return "unbroken"
			}
			return "failed"
		},
		Reward: func(g *Game, rng *rand.Rand) string {
			id := randomScroll(rng)
			if g.addScroll(id) {
				return scrollByID[id].Name
			}
			g.run.Gold += 30
			return "+30 gold (scrolls full)"
		}},
	{ID: "chain", Name: "Chain Master", Desc: "Challenge: reach a 30-hit combo to win a relic.", Glyph: "chain", Col: rgba(255, 230, 120, 255), Challenge: true,
		Done:   func(L *Level) bool { return L.peakCombo >= 30 },
		Status: func(L *Level) string { return fmt.Sprintf("best %d / 30", min(L.peakCombo, 30)) },
		Reward: func(g *Game, rng *rand.Rand) string {
			if rel := pickRelic(g, rng, Common); rel != nil {
				g.take(rel)
				return rel.Name
			}
			g.run.Gold += 40
			return "+40 gold"
		}},
}

var omenByID = func() map[string]*omen {
	m := map[string]*omen{}
	for i := range omens {
		m[omens[i].ID] = &omens[i]
	}
	return m
}()

// placeOmens marks some fights on a fresh act map.
func placeOmens(m *ActMap, rng *rand.Rand, act int) {
	for _, n := range m.Nodes {
		if (n.Type == NCombat && n.Row > 0) || n.Type == NElite {
			p := 0.3 + 0.05*float64(act-1)
			if rng.Float64() < p {
				n.Omen = omens[rng.Intn(len(omens))].ID
			}
		}
	}
}

func (L *Level) hasOmen(id string) bool { return L.omen != nil && L.omen.ID == id }

// convertBricks turns a share of plain ground-floor bricks into kind k.
func (L *Level) convertBricks(k levelgen.Kind, share float64) {
	for _, b := range L.Bricks {
		if b.Alive && b.Kind == levelgen.Basic && b.Layer == 0 && b.Up == nil && !b.BossPart && L.rng.Float64() < share {
			if k == levelgen.Crystal {
				L.Crystals++
			}
			b.Kind = k
			b.Col = kindColor(k, b.Col)
			b.HP, b.MaxHP = 1, 1
		}
	}
}

// omenCleared settles a challenge when the chamber is cleared.
func (L *Level) omenCleared() {
	o := L.omen
	if o == nil || !o.Challenge {
		return
	}
	g := L.g
	rng := rand.New(rand.NewSource(L.propSeed ^ 0x0e1))
	if o.Done(L) {
		prize := o.Reward(g, rng)
		if b := int(g.run.Stats.ChallengeGold); b > 0 {
			g.run.Gold += b
			prize += fmt.Sprintf("  ·  +%d gold", b)
		}
		g.toastImg("Challenge met: "+o.Name, prize, o.Col, glyphImage(o.Glyph, o.Col))
		g.snd.PlayKeyed("good")
	} else {
		g.toastImg("Challenge failed: "+o.Name, "Better luck in the next chamber.", cDim, glyphImage(o.Glyph, cDim))
	}
}

// drawOmenTag shows the chamber's omen under the title plate.
func (g *Game) drawOmenTag(screen *ebiten.Image, L *Level, y float64) {
	o := L.omen
	if o == nil {
		return
	}
	u := g.ui
	s := o.Name
	if o.Challenge && o.Status != nil {
		s += "   ·   " + o.Status(L)
	}
	w := u.measure(s, fBold, 18) + 76
	u.frame(screen, 960-w/2, y, w, 42, 0.88)
	u.drawEmblem(screen, glyphImage(o.Glyph, o.Col), 960-w/2+30, y+21, 28, 1)
	u.text(screen, s, fBold, 18, 960-w/2+52, y+9, lighten(o.Col, 0.2), alignLeft)
	if u.hover(960-w/2, y, w, 42) {
		g.tooltip = o.Name + " — " + o.Desc
	}
}

// applyOmen sets a chamber's omen and prepares the arena for it.
func (L *Level) applyOmen(o *omen) {
	L.omen = o
	switch o.ID {
	case "fortified":
		for _, b := range L.Bricks {
			if b.Kind != levelgen.Steel && !b.BossPart {
				b.MaxHP = math.Ceil(b.MaxHP * 1.4)
				b.HP = b.MaxHP
			}
		}
	case "twin":
		r := L.g.run
		L.Balls = append(L.Balls, &Ball{R: L.ballR(), Stuck: true, Off: 0.9, X: L.PaddleX, Z: PaddleZ + 0.6, PierceLeft: r.Stats.Pierce})
	}
	if o.Setup != nil {
		o.Setup(L)
	}
}
