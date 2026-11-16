package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Photo mode (also the store-page capture mode): from the pause screen,
// the chamber freezes, the HUD and menus go, and the camera can orbit,
// tilt and zoom around the board. Enter / A (or the button) saves a full
// resolution PNG to Pictures/Arcane Orb; Steam's own F12 works as well.
// Esc / B returns to the pause screen.

type photoMode struct {
	yaw, pitch, zoom float64 // radians, radians, distance factor
	hideHints        bool
	shoot            bool
	saved            string
	savedT           float64
	lastMX, lastMY   int
	dragging         bool
}

func (g *Game) startPhoto() {
	g.photo = &photoMode{zoom: 1}
	g.snd.Play("click")
}

// updatePhoto handles photo mode's input (from the pause state's update).
func (g *Game) updatePhoto(dt float64) {
	p := g.photo
	if backPressed() {
		g.photo = nil
		g.consumeInput()
		return
	}
	// mouse: drag to orbit, wheel to zoom
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if p.dragging {
			p.yaw += float64(mx-p.lastMX) * 0.005
			p.pitch += float64(my-p.lastMY) * 0.004
		}
		p.dragging = true
	} else {
		p.dragging = false
	}
	p.lastMX, p.lastMY = mx, my
	_, wy := ebiten.Wheel()
	p.zoom *= math.Pow(0.9, wy)
	// keys and pads
	ax := padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal)
	ay := padAxis(ebiten.StandardGamepadAxisLeftStickVertical)
	zy := padAxis(ebiten.StandardGamepadAxisRightStickVertical)
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		ax = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		ax = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		ay = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		ay = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		zy = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		zy = -1
	}
	if math.Abs(ax) > 0.15 {
		p.yaw += ax * dt * 1.2
	}
	if math.Abs(ay) > 0.15 {
		p.pitch += ay * dt * 0.9
	}
	if math.Abs(zy) > 0.15 {
		p.zoom *= math.Pow(1.8, zy*dt)
	}
	p.yaw = math.Max(-1.4, math.Min(1.4, p.yaw))
	p.pitch = math.Max(-0.55, math.Min(0.6, p.pitch))
	p.zoom = math.Max(0.35, math.Min(1.8, p.zoom))
	if inpututil.IsKeyJustPressed(ebiten.KeyH) || padJust(ebiten.StandardGamepadButtonRightTop) {
		p.hideHints = !p.hideHints
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || padJust(ebiten.StandardGamepadButtonRightBottom) {
		p.shoot = true
	}
	if p.savedT > 0 {
		p.savedT -= dt
	}
}

// photoCamera moves the frozen chamber's camera (drawWorld).
func (g *Game) photoCamera(cam *Camera) {
	p := g.photo
	if p == nil || g.state != StPause {
		return
	}
	off := cam.Eye.Sub(cam.Target)
	d := math.Sqrt(off.X*off.X+off.Y*off.Y+off.Z*off.Z) * p.zoom
	baseYaw := math.Atan2(off.X, off.Z)
	basePitch := math.Asin(off.Y / math.Max(1e-6, math.Sqrt(off.X*off.X+off.Y*off.Y+off.Z*off.Z)))
	yaw := baseYaw + p.yaw
	pitch := math.Max(0.08, math.Min(1.45, basePitch+p.pitch))
	cam.Eye = cam.Target.Add(V3{math.Sin(yaw) * math.Cos(pitch) * d, math.Sin(pitch) * d, math.Cos(yaw) * math.Cos(pitch) * d})
	cam.Roll = 0
}

// photoShoot saves the frame drawn so far (the world, before any overlay)
// when a picture was asked for.
func (g *Game) photoShoot(screen *ebiten.Image) {
	p := g.photo
	if p == nil || !p.shoot {
		return
	}
	p.shoot = false
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	dir := photoDir()
	name := filepath.Join(dir, "ArcaneOrb-"+time.Now().Format("2006-01-02-150405")+".png")
	p.saved, p.savedT = name, 3
	g.snd.Play("click")
	go func() { // encoding a 4K frame takes a moment: off the game thread
		_ = os.MkdirAll(dir, 0o755)
		if f, err := os.Create(name); err == nil {
			_ = png.Encode(f, img)
			f.Close()
		}
	}()
}

// photoDir is Pictures/Arcane Orb in the player's home.
func photoDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "photos"
	}
	return filepath.Join(home, "Pictures", "Arcane Orb")
}

// drawPhoto is photo mode's small hint bar (H hides it) and the saved note.
func (g *Game) drawPhoto(s *ebiten.Image) {
	p := g.photo
	u := g.ui
	if p.savedT > 0 {
		a := clamp01(p.savedT)
		u.rrect(s, 660, 40, 600, 56, 14, ca(rgba(8, 6, 18, 255), 0.8*a))
		u.rawText = true
		u.text(s, fmt.Sprintf(tr("Saved to %s"), filepath.Base(p.saved)), fBold, 18, 960, 52, ca(cGold, a), alignCenter)
		u.text(s, filepath.Dir(p.saved), fReg, 13, 960, 76, ca(cDim, a), alignCenter)
		u.rawText = false
	}
	if p.hideHints {
		return
	}
	u.rrect(s, 360, 1000, 1200, 52, 14, rgba(8, 6, 18, 190))
	t := "PHOTO MODE  ·  drag or WASD to orbit  ·  wheel or Q/E to zoom  ·  Enter to save  ·  H hides this  ·  Esc to return"
	if g.padActive {
		t = "PHOTO MODE  ·  left stick to orbit  ·  right stick to zoom  ·  " + padLabel(ebiten.StandardGamepadButtonRightBottom) + tr(" save  ·  ") +
			padLabel(ebiten.StandardGamepadButtonRightTop) + tr(" hide  ·  ") + padLabel(ebiten.StandardGamepadButtonRightRight) + tr(" return")
		u.rawText = true
	}
	u.text(s, t, fBold, 17, 960, 1014, cWhite, alignCenter)
	u.rawText = false
	if !g.padActive && u.button(s, "photosave", "Save picture", 1580, 1000, 220, 52, cGreen, true) {
		p.shoot = true
	}
}
