package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Brick badges: every special brick wears a round, high-contrast badge that
// floats above it and always faces the camera, so its behaviour can be read
// at a glance. Badges are baked once into the free right half of the 3D
// atlas from the same vector emblems used on cards.

type badgeDef struct {
	Glyph string
	Col   color.RGBA
	Text  string // drawn instead of a glyph when set
}

var badgeDefs = map[levelgen.Kind]badgeDef{
	levelgen.Armored:    {"anvil", rgba(176, 182, 196, 255), ""},
	levelgen.Explosive:  {"bomb", rgba(255, 110, 50, 255), ""},
	levelgen.Chain:      {"bolt", rgba(110, 200, 255, 255), ""},
	levelgen.Split:      {"split", rgba(120, 230, 140, 255), ""},
	levelgen.Regen:      {"heart", rgba(255, 110, 160, 255), ""},
	levelgen.Moving:     {"arrow", rgba(90, 230, 210, 255), ""},
	levelgen.Shield:     {"shield", rgba(130, 200, 255, 255), ""},
	levelgen.Teleport:   {"swirl", rgba(190, 120, 255, 255), ""},
	levelgen.Spawner:    {"leaf", rgba(170, 240, 90, 255), ""},
	levelgen.Multiplier: {"star", rgba(255, 120, 220, 255), ""},
	levelgen.Crystal:    {"gem", rgba(140, 225, 255, 255), ""},
	levelgen.Mystery:    {"", rgba(255, 196, 70, 255), "?"},
	levelgen.Elite:      {"crown", rgba(255, 210, 90, 255), ""},
}

// badgeBlurb is the one-line rule shown in How to Play.
var badgeBlurb = map[levelgen.Kind]string{
	levelgen.Basic:      "Gold trim means extra hits",
	levelgen.Steel:      "Unbreakable - plan around it",
	levelgen.Armored:    "Only crits, blasts, elements or void hurt it",
	levelgen.Explosive:  "Blows up its neighbours",
	levelgen.Chain:      "Arcs lightning when it breaks",
	levelgen.Split:      "Splits into smaller bricks",
	levelgen.Regen:      "Heals if left alone",
	levelgen.Moving:     "Slides between free cells",
	levelgen.Shield:     "Protects its neighbours",
	levelgen.Teleport:   "Hops to a random cell",
	levelgen.Spawner:    "Grows new bricks around it",
	levelgen.Multiplier: "Frenzy: double gold and score",
	levelgen.Crystal:    "Bursts into gold",
	levelgen.Mystery:    "Anything can happen",
	levelgen.Elite:      "Very tough, very rewarding",
}

var (
	badgeSpr   = map[levelgen.Kind]Sprite{}
	badgeImgs  = map[levelgen.Kind]*ebiten.Image{}
	badgeBaked bool
)

const badgeRes = 128

// bakeBadges renders every badge once (must run after the game starts).
func (g *Game) bakeBadges() {
	if badgeBaked || atlas == nil {
		return
	}
	badgeBaked = true
	i := 0
	for k := levelgen.Kind(0); k < levelgen.NumKinds; k++ {
		d, ok := badgeDefs[k]
		if !ok {
			continue
		}
		img := g.renderBadge(d, badgeRes)
		badgeImgs[k] = img
		ox, oy := 512+badgeRes*(i%4), badgeRes*(i/4)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ox), float64(oy))
		op.Blend = ebiten.BlendCopy
		atlas.DrawImage(img, op)
		badgeSpr[k] = Sprite{float32(ox) + 1, float32(oy) + 1, float32(ox+badgeRes) - 1, float32(oy+badgeRes) - 1}
		i++
	}
}

// renderBadge draws one badge into a fresh image of the given size.
func (g *Game) renderBadge(d badgeDef, res int) *ebiten.Image {
	img := ebiten.NewImage(res, res)
	c := float64(res) / 2
	R := float64(res) * 0.44
	// soft drop shadow, dark disc, coloured rim, glint
	fillPath(img, circlePath(c, c+R*0.06, R*1.02), rgba(0, 0, 0, 140), rgba(0, 0, 0, 140), grad{}, false)
	fillPath(img, circlePath(c, c, R), mixc(rgba(34, 26, 50, 250), d.Col, 0.22), mixc(rgba(10, 8, 18, 250), d.Col, 0.1), vgrad(c-R, c+R), false)
	strokePath(img, circlePath(c, c, R*0.94), R*0.13, d.Col, false)
	strokePath(img, circlePath(c, c, R*0.8), R*0.025, ca(lighten(d.Col, 0.5), 0.6), false)
	fillPath(img, circlePath(c, c-R*0.45, R*0.55), rgba(255, 255, 255, 34), rgba(255, 255, 255, 0), vgrad(c-R, c), false)
	if d.Text != "" {
		face := g.ui.rawFace(fTitle, R*1.25)
		g.ui.drawRawText(img, d.Text, face, c, c-R*0.66, lighten(d.Col, 0.35))
	} else {
		em := glyphImage(d.Glyph, d.Col)
		sz := R * 1.25
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		sc := sz / float64(emblemRes)
		op.GeoM.Translate(-emblemRes/2, -emblemRes/2)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(c, c)
		img.DrawImage(em, op)
	}
	return img
}

// drawBadge places a brick's badge above it.
func (g *Game) drawBadge(B *Batch, b *Brick, x, top, z, t float64) {
	spr, ok := badgeSpr[b.Kind]
	if !ok || b.Up != nil || b.BossPart || b.Core {
		return
	}
	bob := 0.06 * math.Sin(t*2.2+x*0.7+z*0.3)
	size := 1.55
	if b.Kind == levelgen.Elite {
		size = 1.95
	}
	a := 1.0
	if b.Appear < 1 {
		a = b.Appear
	}
	p := V3{x, top + 0.85 + bob, z - 0.1}
	B.Billboard(p, size*1.7, sprSoft, alpha(badgeDefs[b.Kind].Col, 0.35*a), true)
	B.Billboard(p, size, spr, alpha(cWhite, a), false)
}
