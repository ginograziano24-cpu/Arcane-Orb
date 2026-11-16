package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Controller button names that match the pad in the player's hands
// (Steam release readiness): Xbox letters by default, PlayStation shapes,
// Nintendo letters (which sit in different places) and the Steam Deck's
// own names. The family follows the last pad that was used.

const (
	padXbox = iota
	padPlayStation
	padNintendo
	padDeck
)

var padFamily = padXbox

var padFamilyNames = [...]map[ebiten.StandardGamepadButton]string{
	padPlayStation: {
		ebiten.StandardGamepadButtonRightBottom:      "Cross",
		ebiten.StandardGamepadButtonRightRight:       "Circle",
		ebiten.StandardGamepadButtonRightLeft:        "Square",
		ebiten.StandardGamepadButtonRightTop:         "Triangle",
		ebiten.StandardGamepadButtonFrontTopLeft:     "L1",
		ebiten.StandardGamepadButtonFrontTopRight:    "R1",
		ebiten.StandardGamepadButtonFrontBottomLeft:  "L2",
		ebiten.StandardGamepadButtonFrontBottomRight: "R2",
		ebiten.StandardGamepadButtonCenterLeft:       "Create",
		ebiten.StandardGamepadButtonCenterRight:      "Options",
	},
	padNintendo: {
		// the standard layout is positional: the bottom face button is B
		// on a Nintendo pad, the right one A, and so on
		ebiten.StandardGamepadButtonRightBottom:      "B",
		ebiten.StandardGamepadButtonRightRight:       "A",
		ebiten.StandardGamepadButtonRightLeft:        "Y",
		ebiten.StandardGamepadButtonRightTop:         "X",
		ebiten.StandardGamepadButtonFrontTopLeft:     "L",
		ebiten.StandardGamepadButtonFrontTopRight:    "R",
		ebiten.StandardGamepadButtonFrontBottomLeft:  "ZL",
		ebiten.StandardGamepadButtonFrontBottomRight: "ZR",
		ebiten.StandardGamepadButtonCenterLeft:       "−",
		ebiten.StandardGamepadButtonCenterRight:      "+",
	},
	padDeck: {
		ebiten.StandardGamepadButtonFrontTopLeft:     "L1",
		ebiten.StandardGamepadButtonFrontTopRight:    "R1",
		ebiten.StandardGamepadButtonFrontBottomLeft:  "L2",
		ebiten.StandardGamepadButtonFrontBottomRight: "R2",
		ebiten.StandardGamepadButtonCenterLeft:       "View",
		ebiten.StandardGamepadButtonCenterRight:      "Menu",
	},
}

// padFamilyOf reads a pad's family from its name.
func padFamilyOf(name string) int {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "steam deck"), strings.Contains(n, "valve"):
		return padDeck
	case strings.Contains(n, "dualsense"), strings.Contains(n, "dualshock"), strings.Contains(n, "playstation"),
		strings.Contains(n, "ps4"), strings.Contains(n, "ps5"), strings.Contains(n, "sony"), n == "wireless controller":
		return padPlayStation
	case strings.Contains(n, "nintendo"), strings.Contains(n, "switch"), strings.Contains(n, "joy-con"), strings.Contains(n, "pro controller"):
		return padNintendo
	}
	return padXbox
}

// notePadFamily follows the pad that was just used (latchInput, on pad
// input only).
func notePadFamily() {
	if steamOnDeck() {
		padFamily = padDeck
		return
	}
	for _, id := range gamepads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && anyButtonOn(id) {
			padFamily = padFamilyOf(ebiten.GamepadName(id))
			return
		}
	}
}

func anyButtonOn(id ebiten.GamepadID) bool {
	for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
		if ebiten.IsStandardGamepadButtonPressed(id, b) {
			return true
		}
	}
	return false
}

// familyLabel is a button's name on the current pad family, or "" for the
// default (Xbox) name.
func familyLabel(b ebiten.StandardGamepadButton) string {
	if padFamily == padXbox {
		return ""
	}
	return padFamilyNames[padFamily][b]
}
