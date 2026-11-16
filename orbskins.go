package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Collectible orb skins (roadmap #113), earned on the Challenges screen and
// chosen there. A skin changes only the orb's body: its glow still shows
// its element and power-ups, so play reads the same.

type orbSkin struct {
	ID, Name, Need string
	Core           color.RGBA
	Have           func(g *Game) bool
}

var orbSkins = []orbSkin{
	{ID: "", Name: "Classic", Need: "", Core: color.RGBA{240, 235, 255, 255}, Have: func(g *Game) bool { return true }},
	{ID: "cookie", Name: "Cookie", Need: "Seal any challenge.", Core: color.RGBA{214, 158, 96, 255},
		Have: func(g *Game) bool { return g.challengesDone() >= 1 }},
	{ID: "frog", Name: "Frog", Need: "Seal three challenges.", Core: color.RGBA{120, 214, 96, 255},
		Have: func(g *Game) bool { return g.challengesDone() >= 3 }},
	{ID: "glitter", Name: "Glitter", Need: "Win a weekly mutator run.", Core: color.RGBA{255, 170, 230, 255},
		Have: func(g *Game) bool {
			for _, v := range g.save.Weekly {
				if v > numActs {
					return true
				}
			}
			return false
		}},
}

func skinIndex(id string) int {
	for i, s := range orbSkins {
		if s.ID == id {
			return i
		}
	}
	return 0
}

// activeSkin is the skin the orbs wear (an earned one only).
func (g *Game) activeSkin() *orbSkin {
	s := &orbSkins[skinIndex(g.save.OrbSkin)]
	if s.ID == "" || !s.Have(g) {
		return &orbSkins[0]
	}
	return s
}

// skinCore is the orb body's colour under the skin; the classic skin keeps
// the colour the game chose (fire, meteor, storm).
func (g *Game) skinCore(def color.RGBA) color.RGBA {
	s := g.activeSkin()
	if s.ID == "" {
		return def
	}
	return s.Core
}

// drawOrbSkin adds the skin's details on top of the orb's body.
func (g *Game) drawOrbSkin(B *Batch, b *Ball, a float64) {
	s := g.activeSkin()
	if s.ID == "" {
		return
	}
	rot := (b.X + b.Z) * 0.9 / math.Max(0.1, b.R) // rolls as it travels
	at := func(ang, d, h float64) V3 {
		return V3{b.X + math.Cos(rot+ang)*b.R*d, b.R*(1+h), b.Z + math.Sin(rot+ang)*b.R*d*0.6}
	}
	switch s.ID {
	case "cookie":
		chip := color.RGBA{86, 48, 26, 255}
		for i, d := range [...]float64{0.45, 0.62, 0.3, 0.55, 0.15} {
			B.Billboard(at(float64(i)*1.37, d, 0.08), b.R*0.42, sprSphere, alpha(chip, 0.95*a), false)
		}
	case "frog":
		// two eyes on top, looking where it goes
		dir := math.Atan2(b.VZ, b.VX)
		for _, side := range [...]float64{-0.55, 0.55} {
			ex := b.X + math.Cos(dir+side)*b.R*0.55
			ez := b.Z + math.Sin(dir+side)*b.R*0.55*0.6
			B.Billboard(V3{ex, b.R * 1.75, ez}, b.R*0.8, sprSphere, alpha(color.RGBA{250, 250, 240, 255}, a), false)
			B.Billboard(V3{ex + math.Cos(dir)*b.R*0.12, b.R * 1.8, ez + math.Sin(dir)*b.R*0.07}, b.R*0.38, sprSphere, alpha(color.RGBA{20, 24, 18, 255}, a), false)
		}
	case "glitter":
		t := float64(time.Now().UnixNano()%1e10) / 1e9
		for i := 0; i < 6; i++ {
			tw := 0.5 + 0.5*math.Sin(t*7+float64(i)*2.1)
			B.Billboard(at(float64(i)*1.05, 0.7+0.25*math.Sin(t*2+float64(i)), 0.1+0.2*tw), b.R*(0.5+0.6*tw), sprSpark, alpha(color.RGBA{255, 255, 255, 255}, 0.9*tw*a), true)
		}
	}
}

// drawSkinPicker is the orb skin selector on the Challenges screen.
func (g *Game) drawSkinPicker(s *ebiten.Image, x, y float64) {
	u := g.ui
	cur := skinIndex(g.save.OrbSkin)
	sk := &orbSkins[cur]
	u.text(s, "Orb skin", fBold, 18, x+180, y-26, cDim, alignCenter)
	if u.button(s, "skinL", "‹", x, y, 48, 48, cAcc, true) {
		cur = (cur + len(orbSkins) - 1) % len(orbSkins)
	}
	if u.button(s, "skinR", "›", x+312, y, 48, 48, cAcc, true) {
		cur = (cur + 1) % len(orbSkins)
	}
	g.save.OrbSkin = orbSkins[cur].ID
	sk = &orbSkins[cur]
	have := sk.Have(g)
	u.frame(s, x+56, y, 248, 48, 0.85)
	// a small preview of the orb body
	cx, cy := x+84, y+24
	c := sk.Core
	if !have {
		c = rgba(70, 64, 84, 255)
	}
	fillPath(s, circlePath(u.X(cx), u.Y(cy), 14*u.S), lighten(c, 0.3), darken(c, 0.35), vgrad(u.Y(cy-14), u.Y(cy+14)), false)
	nc := cWhite
	if !have {
		nc = cDim
	}
	u.text(s, sk.Name, fBold, 18, x+190, y+12, nc, alignCenter)
	if u.hover(x+56, y, 248, 48) && sk.Need != "" {
		st := tr("Unlocked")
		if !have {
			st = fmt.Sprintf(tr("Locked: %s"), tr(sk.Need))
		}
		g.tooltip = st
	}
}
