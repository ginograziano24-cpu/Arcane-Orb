package main

import (
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const frameCapDisplay = -1
const desktopFramePacing = runtime.GOOS != "js"

var frameCaps = [...]int{0, 30, 60, 120, 144, 165, 240, frameCapDisplay} // 30: battery saver (Steam Deck)
var frameCapLabels = []string{"Off", "30", "60", "120", "144", "165", "240", "Display"}

func frameCapIndex(cap int) int {
	for i, value := range frameCaps {
		if value == cap {
			return i
		}
	}
	return 0
}

type framePacer struct {
	interval time.Duration
	deadline time.Time
}

func (g *Game) paceFrame() {
	if !desktopFramePacing {
		return
	}
	cap, vsync := g.settings.FrameCap, g.settings.VSync
	if benchOn {
		if bench == nil {
			return
		}
		cap, vsync = benchFrameCap, benchVSync
	} else if !vsync && cap == 0 && g.duel == nil {
		switch g.state {
		case StMenu, StArch, StSeed, StSanctum, StKeepsakes, StHelp, StCodex, StChallenge, StPause:
			// Limit otherwise unbounded local menus while keeping input and animation responsive.
			cap = 120
		}
	}
	if vsync && cap != 30 {
		cap = 0 // VSync paces the frames, except for the 30 fps battery saver
	}
	switch cap {
	case frameCapDisplay:
		cap = ebiten.DisplayRefreshRate()
		if cap <= 0 {
			cap = 60
		}
	case 30, 60, 120, 144, 165, 240:
	default:
		g.framePacer = framePacer{}
		return
	}
	g.framePacer.wait(time.Second / time.Duration(cap))
}

// wait paces update starts. Rendering and presentation consume part of the
// interval, so only the remaining time is spent waiting.
func (p *framePacer) wait(interval time.Duration) {
	now := time.Now()
	if p.interval != interval || p.deadline.IsZero() {
		p.interval = interval
		p.deadline = now.Add(interval)
		return
	}
	// Discard a missed deadline after a stall to avoid catching up in bursts.
	if now.Sub(p.deadline) > interval/2 {
		p.deadline = now.Add(interval)
		return
	}
	// Sleep wakes late by up to a timer tick, so the last stretch is spun.
	// Keep the spin short at high refresh rates, where 400 us is a tenth of
	// the frame and burns a core for nothing.
	spinWindow := 400 * time.Microsecond
	if interval < 5*time.Millisecond {
		spinWindow = 250 * time.Microsecond
	}
	if delay := p.deadline.Sub(now) - spinWindow; delay > 0 {
		time.Sleep(delay)
	}
	for time.Now().Before(p.deadline) {
	}
	p.deadline = p.deadline.Add(interval)
}
