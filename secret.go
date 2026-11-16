package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Secret chambers (roadmap #106). Clearing a chamber without dropping an orb
// sometimes reveals a hidden door on the map: a vault off to the side of the
// next row, reachable from where you stand. It takes the place of one step
// of the route (it leads on to the same chambers the next row would), so
// taking it costs nothing but the choice. Inside: a chamber thick with gold
// crystals, and a relic of Rare or better at the end. One per act.

const secretChance = 0.4

// vaultOmen marks a secret chamber; it is never placed on a map at random.
var vaultOmen = omen{ID: "vault", Name: "Secret Vault", Desc: "A hidden vault: gold crystals grow everywhere and every brick sheds twice the gold. A relic of Rare or better waits at the end.",
	Glyph: "crown", Col: rgba(255, 214, 110, 255), Secret: true,
	Setup: func(L *Level) { L.convertBricks(levelgen.Crystal, 0.22) }}

func init() {
	omens = append(omens, vaultOmen)
	omenByID["vault"] = &omens[len(omens)-1]
}

// maybeSecretDoor opens a vault door after a flawless clear (afterLevel).
func (g *Game) maybeSecretDoor(rng *rand.Rand) bool {
	r, L := g.run, g.level
	if r == nil || L == nil || r.Duel || !g.flawless || r.Map == nil || r.Cur < 0 {
		return false
	}
	for _, code := range noShowRuns {
		if r.SeedCode == code {
			return false // demos, tests, benchmarks
		}
	}
	if L.Node != NCombat && L.Node != NElite || L.hasOmen("vault") {
		return false
	}
	return r.Map.openVault(r.Cur, rng)
}

// openVault adds the vault beside the row after node cur; it reports
// whether one was added. Only one per act, never on the Guardian's row.
func (m *ActMap) openVault(cur int, rng *rand.Rand) bool {
	for _, n := range m.Nodes {
		if n.Omen == "vault" {
			return false
		}
	}
	c := m.Nodes[cur]
	if c.Row+1 >= mapRows || len(c.Next) == 0 || rng.Float64() >= secretChance {
		return false
	}
	// it leads on to everything the next row's chambers lead to
	seen := map[int]bool{}
	var next []int
	for _, j := range c.Next {
		for _, k := range m.Nodes[j].Next {
			if !seen[k] {
				seen[k] = true
				next = append(next, k)
			}
		}
	}
	if len(next) == 0 {
		return false
	}
	lane := 4 // off the right edge of the map...
	if c.Lane < 2 {
		lane = -1 // ...or the left, whichever side you're on
	}
	m.Nodes = append(m.Nodes, &MapNode{Row: c.Row + 1, Lane: lane, Type: NCombat, Next: next, Seed: rng.Int63(), Omen: "vault"})
	c.Next = append(c.Next, len(m.Nodes)-1)
	return true
}

// announceVault is the toast when the door appears.
func (g *Game) announceVault() {
	o := omenByID["vault"]
	g.toastImg("A hidden door!", "A secret vault has opened on the map, beside the next row.", o.Col, glyphImage(o.Glyph, o.Col))
	g.snd.PlayKeyed("omenmet")
}

// drawVaultGlow is the vault door's shimmer on the map.
func (g *Game) drawVaultGlow(s *ebiten.Image, n *MapNode, x, y, rad float64) {
	if n.Omen != "vault" || n.Visited {
		return
	}
	u := g.ui
	o := omenByID["vault"]
	p := 0.5 + 0.5*math.Sin(g.stateT*3)
	drawSprite(s, sprSoft, u.X(x-rad*3), u.Y(y-rad*3), rad*6*u.S, rad*6*u.S, ca(o.Col, 0.25+0.25*p), true)
	for k := 0; k < 6; k++ {
		a := g.stateT*1.4 + float64(k)*1.047
		sx, sy := x+math.Cos(a)*(rad+26), y+math.Sin(a)*(rad+26)
		drawSprite(s, sprSpark, u.X(sx-7), u.Y(sy-7), 14*u.S, 14*u.S, ca(lighten(o.Col, 0.3), 0.8), true)
	}
}
