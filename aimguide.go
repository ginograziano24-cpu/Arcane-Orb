package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Aim guide (ROADMAP-50 #8, Settings > Effects, off by default).
//
// While an orb is coming down toward the paddle, a faint dotted ray shows
// where it would leave the paddle if it landed where the paddle is now.
// It only appears for the orb that will arrive first, only in the last
// stretch of its fall, and only when it would actually land on the
// paddle, so it teaches the edge angles without turning into a laser
// sight. The ray uses the same angle rule as collidePaddle, including the
// last-brick lean.

const (
	aimGuideReach = 9.0 // the guide appears once the orb is this close (world units, along Z)
	aimGuideLen   = 6.0 // length of the outgoing ray
	aimGuideDots  = 7
)

// predictPaddleHit folds the orb's straight flight between the side walls
// until it reaches the paddle's face. It returns the contact X and the time
// to get there, or ok=false if the orb isn't heading for the paddle.
func predictPaddleHit(x, z, vx, vz, r float64) (cx, t float64, ok bool) {
	face := PaddleZ + PaddleD/2 + r
	if vz >= 0 || z <= face {
		return 0, 0, false
	}
	t = (z - face) / -vz
	lo, hi := -HalfW+r, HalfW-r
	w := hi - lo
	if w <= 0 {
		return 0, 0, false
	}
	// unfold the bounces: reflect the free path into [lo, hi]
	p := math.Mod(x+vx*t-lo, 2*w)
	if p < 0 {
		p += 2 * w
	}
	if p > w {
		p = 2*w - p
	}
	return lo + p, t, true
}

// paddleExitAngle is the angle (radians from straight up the board) an orb
// leaves the paddle at when it lands at contact X cx; the same rule as
// collidePaddle.
func (L *Level) paddleExitAngle(b *Ball, cx float64) float64 {
	half := L.PaddleW / 2
	t := math.Max(-1, math.Min(1, (cx-L.PaddleX)/half))
	ang := t * 62 * math.Pi / 180
	probe := *b
	probe.X, probe.Z = cx, PaddleZ+PaddleD/2+b.R
	if a, ok := L.lastBrickAim(&probe); ok {
		ang += (a - ang) * 0.55
	}
	return ang
}

func (L *Level) drawAimGuide(screen *ebiten.Image, proj func(V3) (float64, float64)) {
	if !L.g.settings.AimGuide || L.Cleared || L.introT > 0 {
		return
	}
	var best *Ball
	bestT, bestX := math.Inf(1), 0.0
	for _, b := range L.Balls {
		if b.Stuck || b.Phantom > 0 || b.Z-PaddleZ > aimGuideReach {
			continue
		}
		cx, t, ok := predictPaddleHit(b.X, b.Z, b.VX, b.VZ, b.R)
		if ok && t < bestT {
			best, bestT, bestX = b, t, cx
		}
	}
	if best == nil {
		return
	}
	half := L.PaddleW / 2
	if math.Abs(bestX-L.PaddleX) > half+best.R {
		return // it would miss: no promise the paddle can't keep
	}
	ang := L.paddleExitAngle(best, bestX)
	// fade in over the fall, so the guide never pops in
	fade := clamp01((aimGuideReach - (best.Z - PaddleZ)) / 3)
	ink := color.RGBA{255, 244, 214, 255}
	sx, sz := math.Sin(ang), math.Cos(ang)
	z0 := PaddleZ + PaddleD/2 + best.R
	size := max(1.5, 2.2*L.g.ui.S)
	for i := 1; i <= aimGuideDots; i++ {
		f := float64(i) / aimGuideDots
		px, pz := bestX+sx*aimGuideLen*f, z0+sz*aimGuideLen*f
		if px < -HalfW || px > HalfW {
			break
		}
		x, y := proj(V3{px, best.R, pz})
		a := fade * 0.55 * (1 - f*0.8)
		fillCircle(screen, x, y, size, prem(ink, a))
	}
}
