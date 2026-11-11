package main

import (
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const frameCapDisplay = -1
const desktopFramePacing = runtime.GOOS != "js"

var frameCaps = [...]int{0, 60, 120, 144, frameCapDisplay}
var frameCapLabels = []string{"Off", "60", "120", "144", "Display"}

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
		case StMenu, StArch, StSeed, StSanctum, StHelp, StCodex, StChallenge, StPause:
			// Limit otherwise unbounded local menus while keeping input and animation responsive.
			cap = 120
		}
	}
	if vsync {
		cap = 0
	}
	switch cap {
	case frameCapDisplay:
		cap = ebiten.DisplayRefreshRate()
		if cap <= 0 {
			cap = 60
		}
	case 60, 120, 144:
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
	const spinWindow = 400 * time.Microsecond
	if delay := p.deadline.Sub(now) - spinWindow; delay > 0 {
		time.Sleep(delay)
	}
	for time.Now().Before(p.deadline) {
	}
	p.deadline = p.deadline.Add(interval)
}
