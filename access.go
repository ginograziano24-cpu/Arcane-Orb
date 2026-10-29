package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Accessibility: bigger body text, a slower game speed, and colour-blind
// filters that shift colours a viewer can't tell apart into ones they can.

func (g *Game) textScale() float64 {
	s := g.settings.TextScale
	if s <= 0 {
		return 1
	}
	return math.Max(0.9, math.Min(1.35, s))
}

// gameSpeed is the chamber speed assist (1 = normal).
func (g *Game) gameSpeed() float64 {
	if g.duel != nil {
		return 1 // both racers play at full speed
	}
	s := g.settings.GameSpeed
	if s <= 0 || s > 1 {
		return 1
	}
	return math.Max(0.5, s)
}

// assisted runs don't post to the leaderboards.
func (g *Game) assisted() bool { return g.gameSpeed() < 1 }

var colorBlindNames = []string{"Off", "Protanopia", "Deuteranopia", "Tritanopia"}

// Daltonize: simulate the deficiency in LMS space, then push the colour
// information that was lost into the channels that are still seen.
const daltonSrc = `//kage:unit pixels
package main

var Mode float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		c.rgb /= c.a
	}
	r := c.r
	gg := c.g
	b := c.b
	l := 17.8824*r + 43.5161*gg + 4.11935*b
	m := 3.45565*r + 27.1554*gg + 3.86714*b
	s := 0.0299566*r + 0.184309*gg + 1.46709*b
	if Mode < 1.5 {
		l = 2.02344*m - 2.52581*s
	} else if Mode < 2.5 {
		m = 0.494207*l + 1.24827*s
	} else {
		s = -0.395913*l + 0.801109*m
	}
	sr := 0.0809444479*l - 0.130504409*m + 0.116721066*s
	sg := -0.0102485335*l + 0.0540193266*m - 0.113614708*s
	sb := -0.000365296938*l - 0.00412161469*m + 0.693511405*s
	er := r - sr
	eg := gg - sg
	eb := b - sb
	out := vec3(r, gg+0.7*er+eg, b+0.7*er+eb)
	if Mode > 2.5 {
		out = vec3(r+0.7*eb+er, gg+0.7*eb+eg, b)
	}
	out = clamp(out, 0, 1)
	return vec4(out*c.a, c.a)
}
`

// drawFiltered runs the frame through the colour-blind filter when one is
// chosen; otherwise it draws straight to the screen.
func (g *Game) drawFiltered(screen *ebiten.Image, draw func(*ebiten.Image)) {
	mode := g.settings.ColorBlind
	if mode <= 0 || mode >= len(colorBlindNames) {
		draw(screen)
		return
	}
	if g.cbShader == nil {
		sh, err := ebiten.NewShader([]byte(daltonSrc))
		if err != nil {
			draw(screen)
			return
		}
		g.cbShader = sh
	}
	b := screen.Bounds()
	if g.cbBuf == nil || g.cbBuf.Bounds() != b {
		if g.cbBuf != nil {
			g.cbBuf.Deallocate()
		}
		g.cbBuf = ebiten.NewImage(b.Dx(), b.Dy())
	}
	g.cbBuf.Clear()
	draw(g.cbBuf)
	op := &ebiten.DrawRectShaderOptions{Uniforms: map[string]any{"Mode": float32(mode)}}
	op.Images[0] = g.cbBuf
	screen.DrawRectShader(b.Dx(), b.Dy(), g.cbShader, op)
}

// applyWindowMode switches between a framed window and a borderless one
// that fills the monitor (fullscreen is handled by SetFullscreen).
func applyWindowMode(s *Settings) {
	if s.Fullscreen {
		return
	}
	if s.Borderless {
		ebiten.SetWindowDecorated(false)
		w, h := ebiten.Monitor().Size()
		ebiten.SetWindowPosition(0, 0)
		ebiten.SetWindowSize(w, h)
		return
	}
	if !ebiten.IsWindowDecorated() {
		ebiten.SetWindowDecorated(true)
		mw, mh := ebiten.Monitor().Size()
		w, h := mw*3/4, mh*3/4
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowPosition((mw-w)/2, (mh-h)/2)
	}
}

// ---------------------------------------------------------------- quick restart

const restartHoldTime = 0.8

// updateRestartHold restarts the run (same class and challenge, a fresh
// seed) once the restart key has been held; it reports a restart.
func (g *Game) updateRestartHold(dt float64) bool {
	r := g.run
	if r == nil || r.Daily || g.daily || !g.actDown("restart") {
		g.restartHold = 0
		return false
	}
	g.restartHold += dt
	if g.restartHold < restartHoldTime {
		return false
	}
	g.restartHold = 0
	arch, ch, endless := r.Arch, r.Challenge, r.EndlessStart
	clearRun()
	g.run, g.level = nil, nil
	g.seedText, g.endlessMode = "", endless
	g.snd.Play("warp")
	g.startRunWith(arch, ch)
	return true
}

func (g *Game) drawRestartHold(s *ebiten.Image) {
	if g.restartHold <= 0.05 || g.state != StLevel {
		return
	}
	u := g.ui
	k := math.Min(1, g.restartHold/restartHoldTime)
	cx, cy := 960.0, 540.0
	fillPath(s, circlePath(u.X(cx), u.Y(cy), 46*u.S), ca(cBlack, 0.55), ca(cBlack, 0.55), grad{}, false)
	strokePath(s, arcPath(u.X(cx), u.Y(cy), 40*u.S, -math.Pi/2, -math.Pi/2+2*math.Pi*k), 6*u.S, cGold, false)
	u.textOutlined(s, "Restarting…", fBold, 22, cx, cy+58, cWhite, alignCenter)
}
