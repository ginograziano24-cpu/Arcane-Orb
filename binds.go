package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Remappable controls. Each action has up to two keyboard keys and, for
// the actions that use one, a gamepad button. The mouse always works
// (move to steer, left click to launch, right click for the ability) and
// Escape always pauses and backs out, so a bad binding can't lock anyone
// out.

type bindAction struct {
	ID, Name string
	Keys     [2]ebiten.Key
	Pad      ebiten.StandardGamepadButton
	HasPad   bool
}

const keyNone = ebiten.Key(-1)

var bindDefaults = []bindAction{
	{ID: "left", Name: "Move left", Keys: [2]ebiten.Key{ebiten.KeyA, ebiten.KeyArrowLeft}},
	{ID: "right", Name: "Move right", Keys: [2]ebiten.Key{ebiten.KeyD, ebiten.KeyArrowRight}},
	{ID: "fast", Name: "Dash (hold)", Keys: [2]ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyShiftRight}},
	{ID: "launch", Name: "Launch orb", Keys: [2]ebiten.Key{ebiten.KeySpace, keyNone}, Pad: ebiten.StandardGamepadButtonRightBottom, HasPad: true},
	{ID: "ability", Name: "Class ability", Keys: [2]ebiten.Key{ebiten.KeyE, keyNone}, Pad: ebiten.StandardGamepadButtonRightLeft, HasPad: true},
	{ID: "scroll1", Name: "Read scroll 1", Keys: [2]ebiten.Key{ebiten.Key1, keyNone}},
	{ID: "scroll2", Name: "Read scroll 2", Keys: [2]ebiten.Key{ebiten.Key2, keyNone}},
	{ID: "scroll3", Name: "Read scroll 3", Keys: [2]ebiten.Key{ebiten.Key3, keyNone}},
	{ID: "readsel", Name: "Read selected scroll", HasPad: true, Pad: ebiten.StandardGamepadButtonRightTop, Keys: [2]ebiten.Key{keyNone, keyNone}},
	{ID: "pause", Name: "Pause", Keys: [2]ebiten.Key{ebiten.KeyP, keyNone}, Pad: ebiten.StandardGamepadButtonCenterRight, HasPad: true},
	{ID: "restart", Name: "Quick restart (hold)", Keys: [2]ebiten.Key{ebiten.KeyR, keyNone}},
}

// savedBind is how a binding is stored in the settings file.
type savedBind struct {
	Keys [2]string `json:"keys"`
	Pad  int       `json:"pad"`
}

// binds returns the live binding for an action.
func (g *Game) bind(id string) *bindAction {
	if g.bindCache == nil {
		g.rebuildBinds()
	}
	return g.bindCache[id]
}

func (g *Game) rebuildBinds() {
	g.bindCache = map[string]*bindAction{}
	for _, d := range bindDefaults {
		b := d
		if sb, ok := g.settings.Binds[d.ID]; ok {
			for i, k := range sb.Keys {
				if k == "" {
					b.Keys[i] = keyNone
					continue
				}
				var kk ebiten.Key
				if kk.UnmarshalText([]byte(k)) == nil {
					b.Keys[i] = kk
				}
			}
			if d.HasPad && sb.Pad >= 0 && sb.Pad <= int(ebiten.StandardGamepadButtonMax) {
				b.Pad = ebiten.StandardGamepadButton(sb.Pad)
			}
		}
		g.bindCache[d.ID] = &b
	}
}

// setBind stores a changed binding.
func (g *Game) setBind(b *bindAction) {
	if g.settings.Binds == nil {
		g.settings.Binds = map[string]savedBind{}
	}
	var sb savedBind
	for i, k := range b.Keys {
		if k != keyNone {
			t, _ := k.MarshalText()
			sb.Keys[i] = string(t)
		}
	}
	sb.Pad = int(b.Pad)
	g.settings.Binds[b.ID] = sb
	g.rebuildBinds()
}

func (g *Game) resetBinds() {
	g.settings.Binds = nil
	g.rebuildBinds()
}

func (g *Game) actDown(id string) bool {
	b := g.bind(id)
	for _, k := range b.Keys {
		if k != keyNone && ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return b.HasPad && padDown(b.Pad)
}

func (g *Game) actJust(id string) bool {
	b := g.bind(id)
	for _, k := range b.Keys {
		if k != keyNone && inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return b.HasPad && padJust(b.Pad)
}

// keyLabel is a short, readable name for a key.
func keyLabel(k ebiten.Key) string {
	if k == keyNone {
		return "—"
	}
	s := k.String()
	switch {
	case strings.HasPrefix(s, "Digit"):
		return s[5:]
	case strings.HasPrefix(s, "Arrow"):
		return map[string]string{"Left": "←", "Right": "→", "Up": "↑", "Down": "↓"}[s[5:]]
	case strings.HasPrefix(s, "Numpad"):
		return "Num " + s[6:]
	case strings.HasSuffix(s, "Left") && len(s) > 4:
		return "L-" + s[:len(s)-4]
	case strings.HasSuffix(s, "Right") && len(s) > 5:
		return "R-" + s[:len(s)-5]
	}
	return s
}

var padNames = map[ebiten.StandardGamepadButton]string{
	ebiten.StandardGamepadButtonRightBottom:      "A",
	ebiten.StandardGamepadButtonRightRight:       "B",
	ebiten.StandardGamepadButtonRightLeft:        "X",
	ebiten.StandardGamepadButtonRightTop:         "Y",
	ebiten.StandardGamepadButtonFrontTopLeft:     "LB",
	ebiten.StandardGamepadButtonFrontTopRight:    "RB",
	ebiten.StandardGamepadButtonFrontBottomLeft:  "LT",
	ebiten.StandardGamepadButtonFrontBottomRight: "RT",
	ebiten.StandardGamepadButtonCenterLeft:       "Back",
	ebiten.StandardGamepadButtonCenterRight:      "Start",
	ebiten.StandardGamepadButtonLeftStick:        "L3",
	ebiten.StandardGamepadButtonRightStick:       "R3",
	ebiten.StandardGamepadButtonLeftTop:          "D-up",
	ebiten.StandardGamepadButtonLeftBottom:       "D-down",
	ebiten.StandardGamepadButtonLeftLeft:         "D-left",
	ebiten.StandardGamepadButtonLeftRight:        "D-right",
}

func padLabel(b ebiten.StandardGamepadButton) string {
	if n, ok := padNames[b]; ok {
		return n
	}
	return "Btn " + itoa(int(b))
}

// bindText describes an action's controls for help text and tips.
func (g *Game) bindText(id string) string {
	pad := g.padActive
	switch id {
	case "move":
		if pad {
			return "the left stick"
		}
		return "the mouse or " + keyLabel(g.bind("left").Keys[0]) + " / " + keyLabel(g.bind("right").Keys[0])
	case "launch":
		if pad {
			return padLabel(g.bind("launch").Pad)
		}
		return "a click or " + keyLabel(g.bind("launch").Keys[0])
	case "ability":
		if pad {
			return padLabel(g.bind("ability").Pad) + ", RB or RT"
		}
		return "right click or " + keyLabel(g.bind("ability").Keys[0])
	case "scrolls":
		if pad {
			return "LB to choose and " + padLabel(g.bind("readsel").Pad) + " to read"
		}
		return keyLabel(g.bind("scroll1").Keys[0]) + ", " + keyLabel(g.bind("scroll2").Keys[0]) + " or " + keyLabel(g.bind("scroll3").Keys[0])
	}
	return id
}

// captureBind waits for the key or button that a slot is being set to.
func (g *Game) captureBind() {
	id, slot, _ := strings.Cut(g.rebinding, ":")
	b := *g.bind(id)
	done := func() {
		g.rebinding = ""
		g.snd.Play("click")
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		done()
		return
	}
	if slot == "pad" {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			done()
			return
		}
		for bt := ebiten.StandardGamepadButton(0); bt <= ebiten.StandardGamepadButtonMax; bt++ {
			if padJust(bt) {
				b.Pad = bt
				g.setBind(&b)
				done()
				return
			}
		}
		return
	}
	i := 0
	if slot == "1" {
		i = 1
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		switch k {
		case ebiten.KeyEscape:
			done()
			return
		case ebiten.KeyBackspace, ebiten.KeyDelete:
			b.Keys[i] = keyNone
		default:
			// a key does one thing: take it off any other action first
			for _, d := range bindDefaults {
				if d.ID == id {
					continue
				}
				o := *g.bind(d.ID)
				changed := false
				for j := range o.Keys {
					if o.Keys[j] == k {
						o.Keys[j], changed = keyNone, true
					}
				}
				if changed {
					g.setBind(&o)
				}
			}
			b.Keys[i] = k
		}
		g.setBind(&b)
		done()
		return
	}
}

// drawControls is the Controls tab: every action with two key slots and,
// where it has one, a gamepad button.
func (g *Game) drawControls(s *ebiten.Image, y float64) {
	u := g.ui
	u.text(s, "KEY", fBold, 16, 1000, y-6, cDim, alignCenter)
	u.text(s, "ALTERNATE", fBold, 16, 1160, y-6, cDim, alignCenter)
	u.text(s, "PAD", fBold, 16, 1320, y-6, cDim, alignCenter)
	y += 18
	slot := func(id, label string, x, y float64, on bool) {
		acc := cAcc
		if g.rebinding == id {
			acc = cGold
			label = "press…"
		}
		if u.button(s, "bind"+id, label, x-70, y, 140, 40, acc, on) {
			g.rebinding = id
			g.snd.Play("click")
		}
	}
	for _, d := range bindDefaults {
		b := g.bind(d.ID)
		u.text(s, d.Name, fReg, 20, 520, y+8, cWhite, alignLeft)
		slot(d.ID+":0", keyLabel(b.Keys[0]), 1000, y, true)
		slot(d.ID+":1", keyLabel(b.Keys[1]), 1160, y, true)
		if d.HasPad {
			slot(d.ID+":pad", padLabel(b.Pad), 1320, y, true)
		}
		y += 46
	}
	if g.rebinding != "" {
		u.text(s, "Press a key (Backspace clears, Escape cancels)", fReg, 18, 960, y+6, cGold, alignCenter)
	} else {
		u.text(s, "The mouse always steers, left click launches and right click uses your ability. Escape always pauses.", fReg, 17, 960, y+6, cDim, alignCenter)
	}
	if u.button(s, "bindreset", "Reset to defaults", 520, 900, 220, 60, cAcc, g.settings.Binds != nil) {
		g.resetBinds()
	}
}
