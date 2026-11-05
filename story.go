package main

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The story frames the three-act run: a prologue, a card between acts
// and an epilogue after the final boss. Endless and daily runs skip it.

type storyCard struct {
	Label, Title string
	Paras        []string
}

var storyCards = []storyCard{
	{"PROLOGUE", "The Orb", []string{
		"Three giant bosses are burying the world in bricks. Every day, the sky gets a little darker.",
		"But there is one thing no wall can stop: the Orb. Whatever it hits, it breaks.",
		"Seven heroes have taken up the Orb. Today, it's your turn.",
		"Smash the walls. Beat the bosses. Bring back the sky.",
	}},
	{"CHAPTER II", "Into the Machine", []string{
		"The Prism Colossus shatters into a rain of coloured glass. For the first time in years, you can see the stars.",
		"But someone is still making the bricks. Deep in the Machine Works, the Crucible Golem keeps the forges burning.",
		"Go down there. Put out the fire.",
	}},
	{"CHAPTER III", "The Top of the Wall", []string{
		"The forges go cold. The hammers stop.",
		"High above, in the Crystal Caverns, the last boss is waiting: the Crystal Warden, the toughest of them all.",
		"Climb.",
	}},
	{"EPILOGUE", "The Sky Is Back", []string{
		"The Crystal Warden cracks, and every wall in the world crumbles with it. Sunlight pours back in.",
		"The heroes are tired, a bit singed, and very proud.",
		"Then, somewhere, a single brick wiggles... Looks like the Orb isn't done yet.",
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
		u.textOutlined(s, "The End", fRune, 44, 960, y+20, ca(cGold, k), alignCenter)
		u.text(s, "Endless Mode is now unlocked: how far can you go?", fBold, 22, 960, y+78, ca(rgba(255, 140, 120, 255), k), alignCenter)
	}

	done := t >= last
	adv := u.Click || g.keyHit(ebiten.KeySpace) || g.keyHit(ebiten.KeyEnter)
	if g.stateT > 0.3 && adv {
		if !done {
			g.storySkip = last - g.stateT // reveal everything
			g.consumeInput()
		} else {
			g.finishStory()
			return
		}
	}
	if g.keyHit(ebiten.KeyEscape) {
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
