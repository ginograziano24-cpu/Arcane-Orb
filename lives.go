package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Lives: dropping every orb (or running out of health) costs a life and
// blows the paddle apart. A run starts with two, every act tops them back
// up to two, and a rare falling heart can restore a lost one, at most
// three times per run.

const (
	baseLives     = 2
	maxLifeFinds  = 3
	lifeDropP     = 0.0035 // per broken brick while a life is missing
	paddleReform  = 1.15   // seconds the paddle takes to reform
	gameOverDelay = 1.9
)

func (r *Run) maxLives() int {
	if r.Challenge == "onelife" {
		return 1
	}
	return baseLives
}

// lifeDropFalling reports whether a heart is already on its way down.
func (L *Level) lifeDropFalling() bool {
	for _, d := range L.Drops {
		if d.P == PLife {
			return true
		}
	}
	return false
}

// maybeDropLife rolls the rare heart when a brick breaks.
func (L *Level) maybeDropLife(x, z float64) {
	r := L.g.run
	if r.Lives >= r.maxLives() || r.LifeFinds >= maxLifeFinds || L.lifeDropFalling() || L.Cleared {
		return
	}
	if L.rng.Float64() < lifeDropP*r.Stats.Luck {
		L.Drops = append(L.Drops, &Drop{X: x, Z: z, P: PLife})
		L.text(V3{x, 2.2, z}, "A HEART FALLS!", color.RGBA{255, 120, 160, 255}, true)
		L.g.snd.PlayVol("abilready", 0.8)
	}
}

// gainLife is the heart being caught.
func (L *Level) gainLife() {
	g := L.g
	r := g.run
	if r.Lives >= r.maxLives() {
		return
	}
	r.Lives++
	r.LifeFinds++
	r.rc().Hearts++
	r.log("heart", "Caught a falling heart", "heart")
	L.lifeFlash = 1
	L.text(V3{L.PaddleX, 2.4, PaddleZ + 2}, "LIFE RESTORED", color.RGBA{255, 140, 180, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 4, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 130, 170, 255}})
	L.sparks(V3{L.PaddleX, 1, PaddleZ}, 30, color.RGBA{255, 150, 190, 255}, 8)
	g.snd.PlayKeyed("heal")
	g.snd.Duck(0.5, 1.5)
	g.react(reactLife)
}

// loseLife spends a life; with none left the run ends after the blast.
func (L *Level) loseLife(why string) {
	g := L.g
	r := g.run
	r.Lives--
	r.rc().LivesLost++
	r.log("life", "Lost a life ("+why+")", "heart")
	L.explodePaddle()
	g.react(reactLifeLost)
	if r.Lives <= 0 {
		r.Lives = 0
		L.gameOverT = gameOverDelay
		L.Balls = nil
		L.text(V3{0, 3, 12}, "THE ORB GOES DARK", color.RGBA{255, 90, 110, 255}, true)
		g.snd.PlayKeyed("over")
		g.snd.Duck(0.8, 3)
		return
	}
	g.coach("life")
	s := fmt.Sprintf("%s  ·  %d LIFE LEFT", why, r.Lives)
	if r.Lives > 1 {
		s = fmt.Sprintf("%s  ·  %d LIVES LEFT", why, r.Lives)
	}
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2.5}, s, color.RGBA{255, 110, 120, 255}, true)
	g.snd.Play("lose")
}

// explodePaddle blows the paddle into shards and sparks.
func (L *Level) explodePaddle() {
	g := L.g
	x, z := L.PaddleX, PaddleZ
	body, trim := L.Biome.PaddleBody, L.Biome.Trim
	L.paddleGone = paddleReform
	L.Shake = math.Max(L.Shake, 1.1)
	L.Flash = math.Max(L.Flash, 0.55)
	L.hitStop = math.Max(L.hitStop, 0.12)
	L.hurtFlash = 1
	for i := 0; i < 3; i++ {
		L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 3 + 2.5*float64(i), Life: 0.35 + 0.15*float64(i), Max: 0.35 + 0.15*float64(i), Col: mixc(color.RGBA{255, 120, 70, 255}, trim, float64(i)/2)})
	}
	half := L.PaddleW / 2
	for i := 0; i < 26; i++ {
		px := x + (L.rng.Float64()*2-1)*half
		v := V3{(px - x) * 3.2, 6 + L.rng.Float64()*9, (L.rng.Float64()*2 - 1) * 5}
		c := body
		if i%3 == 0 {
			c = trim
		}
		L.Parts = append(L.Parts, &Particle{P: V3{px, 0.4, z}, V: v, Life: 1.1, Max: 1.4, Size: 0.22 + L.rng.Float64()*0.3, Col: c, Kind: pDebris, Grav: 26,
			Rot: L.rng.Float64() * 6, Spin: (L.rng.Float64()*2 - 1) * 14})
	}
	L.sparks(V3{x, 0.6, z}, 60, color.RGBA{255, 190, 90, 255}, 14)
	L.sparks(V3{x, 0.6, z}, 30, lighten(trim, 0.4), 9)
	g.snd.Play("boom")
	g.snd.Play("paddleboom")
}

// updateLives runs the reform timer; it returns true while the run is
// ending so the level freezes on the explosion.
func (L *Level) updateLives(dt float64) bool {
	L.lifeFlash = math.Max(0, L.lifeFlash-dt*1.5)
	if L.gameOverT > 0 {
		L.gameOverT -= dt
		L.updateEffects(dt)
		L.Shake = math.Max(0, L.Shake-dt)
		if L.gameOverT <= 0 {
			L.g.endRun(false)
		}
		return true
	}
	if L.paddleGone > 0 {
		was := L.paddleGone
		L.paddleGone -= dt
		if was > 0.25 && L.paddleGone <= 0.25 {
			// the paddle reforms in a swirl of its world's light
			L.Rings = append(L.Rings, &Ring{X: L.PaddleX, Z: PaddleZ, MaxR: 2.4, Life: 0.3, Max: 0.3, Col: lighten(L.Biome.Trim, 0.3)})
			L.sparks(V3{L.PaddleX, 0.5, PaddleZ}, 24, lighten(L.Biome.Trim, 0.4), 4)
			L.g.snd.PlayVol("warp", 0.7)
		}
	}
	return false
}

// paddleShown is how much of the paddle is drawn (0 while it's in pieces).
func (L *Level) paddleShown() float64 {
	if L.gameOverT > 0 {
		return 0
	}
	if L.paddleGone <= 0 {
		return 1
	}
	return clamp01(1 - L.paddleGone/0.25)
}

// drawLives shows the remaining lives as glowing orbs, centred on x.
func (g *Game) drawLives(dst *ebiten.Image, L *Level, x, y float64) {
	u := g.ui
	S := u.S
	r := g.run
	n := max(r.maxLives(), r.Lives)
	gap := 36.0
	x0 := x - float64(n-1)*gap/2
	for i := 0; i < n; i++ {
		cx := x0 + float64(i)*gap
		rad := 13.0
		full := i < r.Lives
		if full && L != nil && L.lifeFlash > 0 && i == r.Lives-1 {
			rad *= 1 + 0.45*L.lifeFlash
		}
		fillPath(dst, circlePath(u.X(cx), u.Y(y+2), (rad+3)*S), ca(cBlack, 0.45), ca(cBlack, 0.45), grad{}, false)
		if full {
			pulse := 0.8 + 0.2*math.Sin(u.time*3+float64(i))
			drawSprite(dst, sprSoft, u.X(cx-rad*2.2), u.Y(y-rad*2.2), rad*4.4*S, rad*4.4*S, ca(rgba(255, 120, 170, 255), 0.4*pulse), true)
			fillPath(dst, circlePath(u.X(cx), u.Y(y), rad*S), rgba(255, 214, 236, 255), rgba(214, 40, 110, 255), vgrad(u.Y(y-rad), u.Y(y+rad)), false)
			fillPath(dst, circlePath(u.X(cx-rad*0.35), u.Y(y-rad*0.35), rad*0.32*S), ca(cWhite, 0.85), ca(cWhite, 0.85), grad{}, false)
		} else {
			fillPath(dst, circlePath(u.X(cx), u.Y(y), rad*S), rgba(40, 24, 40, 255), rgba(14, 8, 16, 255), vgrad(u.Y(y-rad), u.Y(y+rad)), false)
		}
		strokePath(dst, circlePath(u.X(cx), u.Y(y), rad*S), 1.6*S, ca(rgba(255, 200, 226, 255), 0.75), false)
	}
}
