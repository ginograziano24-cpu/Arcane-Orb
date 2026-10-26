package main

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The story frames the three-act run: a prologue, a card between acts
// and an epilogue after the final Guardian. Endless and daily runs skip it.

type storyCard struct {
	Label, Title string
	Paras        []string
}

var storyCards = []storyCard{
	{"PROLOGUE", "The Last Warden", []string{
		"Long ago the Weavers gathered the world's magic into a single sphere of living light, and called it the Orb.",
		"Then the Weavers vanished. Their constructs did not. Brick by brick the Lattice rose: walls of stone, steel and crystal that spread across every realm and choked the light out of the sky.",
		"You are the last Warden of the Orb. Where it strikes, the Lattice breaks.",
		"Three Guardians hold the Lattice together. Break them, and the realms will breathe again.",
	}},
	{"CHAPTER II", "Into the Deep", []string{
		"The Prism Colossus bursts into a rain of coloured glass. For one night, the Outer Realms can see the stars.",
		"But the Lattice is still growing. Its roots run down into the Deep Realms, where the Crucible Golem keeps the forges burning that cast every brick.",
		"Descend. Put out the fire.",
	}},
	{"CHAPTER III", "The Ringing Above", []string{
		"The Crucible goes cold and the great hammers stop. In the silence you hear something far above you: a long, clear note, like a struck bell.",
		"The Crystal Warden was the Weavers' first creation and their last mistake. It is the heart of the Lattice, and it is singing.",
		"Climb.",
	}},
	{"EPILOGUE", "Light Returns", []string{
		"The Warden's heart cracks, and the Lattice cracks with it. Across every realm the walls crumble to dust, and light pours back into the sky.",
		"The Orb dims in your hand, tired and warm.",
		"But the Weavers built deep. Somewhere beneath the rubble, a single brick shifts... and the Lattice begins to grow again.",
	}},
}

const (
	storyParaStart = 0.9
	storyParaGap   = 1.7
	storyFade      = 0.9
)

func (r *Run) storyOn() bool { return !r.Daily && !r.Endless }

func (g *Game) startStory(i int) {
	g.storyIdx = i
	g.storySkip = 0
	g.storyTick = [8]bool{}
	g.setState(StStory)
	g.snd.PlayKeyed("story")
	g.snd.Duck(0.45, 2.5)
}

// finishStory moves on from the current card.
func (g *Game) finishStory() {
	if g.storyIdx >= len(storyCards)-1 {
		g.endRun(true)
		return
	}
	g.setState(StActIntro)
}

func (g *Game) drawStory(s *ebiten.Image) {
	u := g.ui
	S := u.S
	c := storyCards[g.storyIdx]
	t := g.stateT + g.storySkip
	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	fillRect(s, 0, 0, sw, sh, rgba(4, 3, 10, 150))
	// cinematic bars
	bar := 96 * ease(math.Min(1, g.stateT*1.4))
	fillRect(s, 0, 0, sw, bar*S, rgba(0, 0, 0, 235))
	fillRect(s, 0, sh-bar*S, sw, bar*S, rgba(0, 0, 0, 235))
	acc := lighten(u.theme().Accent, 0.3)
	// soft dark wash behind the text so it reads over any backdrop
	drawSprite(s, sprSoft, u.X(960-1100), u.Y(120), 2200*S, 820*S, ca(cBlack, 0.75*ease(clamp01(g.stateT*2))), false)

	a := ease(clamp01(t * 1.6))
	drawSprite(s, sprSoft, u.X(960-700), u.Y(140), 1400*S, 300*S, ca(acc, 0.12*a), true)
	u.textOutlined(s, c.Label, fBold, 24, 960, 186-(1-a)*14, ca(acc, a), alignCenter)
	u.heading(s, strings.ToUpper(c.Title), 84, 226, cGold, a)
	u.glowLine(s, 960-260*a, 960+260*a, 336, 2, cGold, a)

	y := 392.0
	last := 0.0
	for i, p := range c.Paras {
		st := storyParaStart + float64(i)*storyParaGap
		k := clamp01((t - st) / storyFade)
		last = st + storyFade
		lines := float64(len(u.wrap(p, fReg, 29, 1180)))
		if k > 0 {
			if !g.storyTick[i] {
				g.storyTick[i] = true
				if i > 0 && g.storySkip == 0 {
					g.snd.PlayKeyed("storyline")
				}
			}
			e := ease(k)
			col := cWhite
			if i == len(c.Paras)-1 {
				col = rgba(255, 226, 170, 255)
			}
			u.paragraph(s, p, fReg, 29, 960, y+(1-e)*16, 1180, ca(col, e), alignCenter)
		}
		y += lines*29*1.35 + 26
	}
	if g.storyIdx == len(storyCards)-1 && g.save.Wins == 0 && t > last {
		k := ease(clamp01(t - last))
		u.textOutlined(s, "THE END", fTitle, 40, 960, y+20, ca(cGold, k), alignCenter)
		u.text(s, "Endless Mode is now unlocked: how far can you go?", fBold, 22, 960, y+78, ca(rgba(255, 140, 120, 255), k), alignCenter)
	}

	done := t >= last
	adv := u.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	if g.stateT > 0.3 && adv {
		if !done {
			g.storySkip = last - g.stateT // reveal everything
		} else {
			g.finishStory()
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.finishStory()
		return
	}
	if done {
		pa := 0.6 + 0.4*math.Sin(g.stateT*4)
		u.text(s, "Click to continue", fReg, 22, 960, 1080-bar-60, ca(cDim, pa), alignCenter)
	} else {
		u.text(s, "Click to reveal   ·   Esc to skip", fReg, 18, 960, 1080-bar-52, ca(cDim, 0.5), alignCenter)
	}
}
