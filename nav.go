package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Keyboard and gamepad navigation for every menu.
//
// Each frame, widgets register the rectangles they occupy. The next frame,
// arrow keys / WASD / d-pad / left stick move a focus between those
// rectangles spatially, and Enter / Space / A activates the focused one.
// Moving the mouse hands control back to the pointer.

type navItem struct {
	id         string
	x, y, w, h float64
	horiz      bool // left/right adjust it instead of moving focus
}

type navState struct {
	items, prev []navItem
	focus       string
	mode        bool // true while navigating with keys or a pad
	confirm     bool
	adjust      int
	lastMX      float64
	lastMY      float64
}

// navInput is gathered in Update (latched) and consumed in Draw.
type navInput struct {
	dx, dy  int
	confirm bool
	back    bool
	pageL   bool
	pageR   bool
}

// ---------------------------------------------------------------- pads

func gamepads() []ebiten.GamepadID { return ebiten.AppendGamepadIDs(nil) }

// padJust reports a standard-layout button press on any pad this tick.
func padJust(b ebiten.StandardGamepadButton) bool {
	for _, id := range gamepads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && inpututil.IsStandardGamepadButtonJustPressed(id, b) {
			return true
		}
	}
	return false
}

func padDown(b ebiten.StandardGamepadButton) bool {
	for _, id := range gamepads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && ebiten.IsStandardGamepadButtonPressed(id, b) {
			return true
		}
	}
	return false
}

// padAxis returns the strongest value of a standard axis across pads.
func padAxis(a ebiten.StandardGamepadAxis) float64 {
	best := 0.0
	for _, id := range gamepads() {
		if !ebiten.IsStandardGamepadLayoutAvailable(id) {
			continue
		}
		v := ebiten.StandardGamepadAxisValue(id, a)
		if math.Abs(v) > math.Abs(best) {
			best = v
		}
	}
	return best
}

// anyPadInput reports whether a pad did anything this tick.
func anyPadInput() bool {
	for _, id := range gamepads() {
		if len(inpututil.AppendJustPressedStandardGamepadButtons(id, nil)) > 0 {
			return true
		}
	}
	return math.Abs(padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal)) > 0.5 ||
		math.Abs(padAxis(ebiten.StandardGamepadAxisLeftStickVertical)) > 0.5
}

// pollNav reads navigation presses for this tick, with key repeat for
// held directions and the stick.
func (g *Game) pollNav(dt float64) navInput {
	var in navInput
	dir := func(keys []ebiten.Key, pad ebiten.StandardGamepadButton) bool {
		for _, k := range keys {
			if inpututil.IsKeyJustPressed(k) {
				return true
			}
		}
		return padJust(pad)
	}
	if dir([]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyW}, ebiten.StandardGamepadButtonLeftTop) {
		in.dy = -1
	}
	if dir([]ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyS}, ebiten.StandardGamepadButtonLeftBottom) {
		in.dy = 1
	}
	if dir([]ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyA}, ebiten.StandardGamepadButtonLeftLeft) {
		in.dx = -1
	}
	if dir([]ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyD}, ebiten.StandardGamepadButtonLeftRight) {
		in.dx = 1
	}
	// analog stick with repeat
	sx := padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal)
	sy := padAxis(ebiten.StandardGamepadAxisLeftStickVertical)
	held := math.Abs(sx) > 0.55 || math.Abs(sy) > 0.55
	if held {
		if g.stickT <= 0 {
			if math.Abs(sx) > math.Abs(sy) {
				in.dx = int(math.Copysign(1, sx))
			} else {
				in.dy = int(math.Copysign(1, sy))
			}
			if g.stickHeld {
				g.stickT = 0.12
			} else {
				g.stickT = 0.38
			}
		}
		g.stickT -= dt
		g.stickHeld = true
	} else {
		g.stickT, g.stickHeld = 0, false
	}
	in.confirm = inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		padJust(ebiten.StandardGamepadButtonRightBottom)
	in.back = inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) ||
		padJust(ebiten.StandardGamepadButtonRightRight)
	in.pageL = inpututil.IsKeyJustPressed(ebiten.KeyQ) || padJust(ebiten.StandardGamepadButtonFrontTopLeft)
	in.pageR = inpututil.IsKeyJustPressed(ebiten.KeyE) || padJust(ebiten.StandardGamepadButtonFrontTopRight)
	return in
}

// ---------------------------------------------------------------- focus

// beginNav runs at the start of Draw with the latched input.
func (u *UI) beginNav(in navInput) {
	n := &u.nav
	n.prev, n.items = n.items, n.prev[:0]
	n.confirm, n.adjust = false, 0
	if math.Abs(u.MX-n.lastMX)+math.Abs(u.MY-n.lastMY) > 3 {
		n.mode = false
	}
	n.lastMX, n.lastMY = u.MX, u.MY
	moved := in.dx != 0 || in.dy != 0
	if !moved && !in.confirm {
		return
	}
	cur := n.find(n.focus)
	if !n.mode {
		n.mode = true
		if cur == nil && len(n.prev) > 0 {
			n.focus = n.prev[0].id
			u.navSound()
		}
		if moved {
			return // the first press just wakes the focus
		}
	}
	if cur == nil {
		if len(n.prev) > 0 {
			n.focus = n.prev[0].id
			u.navSound()
		}
		return
	}
	if in.confirm {
		n.confirm = true
		return
	}
	if cur.horiz && in.dx != 0 && in.dy == 0 {
		n.adjust = in.dx
		return
	}
	if best := n.nearest(cur, in.dx, in.dy); best != nil {
		n.focus = best.id
		u.navSound()
	}
}

func (u *UI) navSound() {
	if u.snd != nil {
		u.snd.Play("hover")
	}
}

func (n *navState) find(id string) *navItem {
	for i := range n.prev {
		if n.prev[i].id == id {
			return &n.prev[i]
		}
	}
	return nil
}

// nearest picks the item best placed in direction (dx, dy).
func (n *navState) nearest(c *navItem, dx, dy int) *navItem {
	cx, cy := c.x+c.w/2, c.y+c.h/2
	var best *navItem
	bs := math.Inf(1)
	for i := range n.prev {
		it := &n.prev[i]
		if it.id == c.id {
			continue
		}
		ix, iy := it.x+it.w/2, it.y+it.h/2
		ox, oy := ix-cx, iy-cy
		var along, across float64
		if dx != 0 {
			along, across = ox*float64(dx), math.Abs(oy)
			// overlapping rows count as straight across
			if it.y < c.y+c.h && it.y+it.h > c.y {
				across *= 0.25
			}
		} else {
			along, across = oy*float64(dy), math.Abs(ox)
			if it.x < c.x+c.w && it.x+it.w > c.x {
				across *= 0.25
			}
		}
		if along <= 4 {
			continue
		}
		score := along + across*2.2
		if score < bs {
			bs, best = score, it
		}
	}
	return best
}

// navReg registers a focusable rectangle; it returns whether it has focus
// and whether it was activated by the keyboard or a pad this frame.
func (u *UI) navReg(id string, x, y, w, h float64, horiz bool) (focused, activated bool) {
	n := &u.nav
	n.items = append(n.items, navItem{id, x, y, w, h, horiz})
	focused = n.mode && n.focus == id
	if focused && n.confirm {
		n.confirm = false
		activated = true
	}
	return
}

// hit combines pointer hover/click with keyboard/pad focus for a widget.
func (u *UI) hit(id string, x, y, w, h float64) (hover, click bool) {
	f, act := u.navReg(id, x, y, w, h, false)
	mh := !u.nav.mode && u.hover(x, y, w, h)
	hover = mh || f
	click = (mh && u.Click) || act
	return
}

// focusRing draws the keyboard/pad focus outline around a rectangle.
func (u *UI) focusRing(dst *ebiten.Image, x, y, w, h, r float64) {
	S := u.S
	p := 0.6 + 0.4*math.Sin(u.time*6)
	acc := lighten(u.theme().Accent, 0.4)
	strokePath(dst, boxPath(u.X(x-6), u.Y(y-6), (w+12)*S, (h+12)*S, (r+6)*S, false), 2.5*S, ca(acc, 0.9*p), false)
}
