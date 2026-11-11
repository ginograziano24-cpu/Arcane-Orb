// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"image"
	"image/color"
	"log"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

// The rule references in this file (N1, L4, …) point at the Unicode
// Bidirectional Algorithm; see https://www.unicode.org/reports/tr9/
// for the full text.

// hebrewInParensSample mixes English, Japanese, and Hebrew, with
// parens wrapping the Hebrew word. The parens' bidi level (and thus
// whether L4 mirrors them) depends on how rule N1 resolves the
// neutrals against the surrounding context — in this line that varies
// with the base direction.
const hebrewInParensSample = "Hello こんにちは (שלום) world!"

// latinInParensSample wraps an English word in parens that sit
// between two Hebrew words. The neutrals flanked by RTL on both
// sides resolve to an RTL level via rule N1 regardless of the line's
// base direction, so L4 mirrors the bracket glyphs in both base-LTR
// and base-RTL renderings here.
const latinInParensSample = "שלום (Hello) עולם"

var faceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	faceSource = s
}

type Game struct {
	debugui    debugui.DebugUI
	showCarets bool
}

func (g *Game) Update() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Text Bidi", image.Rect(10, 10, 320, 140), func(layout debugui.ContainerLayout) {
			ctx.Checkbox(&g.showCarets, "Show caret marks")
			ctx.Text("Caret marks are drawn via text.AdvanceAt. Each mark is a logical byte index projected to its visual horizontal position.")
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// drawLine renders one bidi sample plus its bounding box, label, and
// optional caret markers placed via AdvanceAt.
func (g *Game) drawLine(screen *ebiten.Image, label, s string, dir text.Direction, y float64) {
	f := &text.GoTextFace{
		Source:    faceSource,
		Direction: dir,
		Size:      24,
	}

	ebitenutil.DebugPrintAt(screen, label, 20, int(y)-18)

	w, h := text.Measure(s, f, 0)

	// For an RTL base, anchor the line to the right of the screen as the
	// reading order naturally starts there. Otherwise anchor to the left.
	var rectX float64
	var translateX float64
	switch dir {
	case text.DirectionRightToLeft:
		rectX = float64(screenWidth) - 20 - w
		translateX = float64(screenWidth) - 20
	default:
		rectX = 20
		translateX = 20
	}

	bg := color.RGBA{0x60, 0x60, 0x60, 0xff}
	vector.FillRect(screen, float32(rectX), float32(y), float32(w), float32(h), bg, false)

	op := &text.DrawOptions{}
	op.GeoM.Translate(translateX, y)
	text.Draw(screen, s, f, op)

	if g.showCarets {
		caret := color.RGBA{0xff, 0xff, 0x40, 0xff}
		// Walk every logical byte index (0 .. len) and mark the visual
		// horizontal position returned by AdvanceAt. AdvanceAt measures
		// from the leftmost edge of the line regardless of base
		// direction, so the on-screen position is rectX + AdvanceAt for
		// both LTR and RTL. Interior bytes of multi-byte runes share
		// the cluster-start's position (snap-to-prev), so duplicates
		// collapse.
		prev := -1.0
		for i := 0; i <= len(s); i++ {
			x := text.AdvanceAt(s, i, f)
			if x == prev {
				continue
			}
			prev = x
			caretX := rectX + x
			vector.StrokeLine(screen, float32(caretX), float32(y)-2, float32(caretX), float32(y)+float32(h)+2, 1, caret, false)
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawLine(screen, "LTR base: English + Japanese + Hebrew",
		hebrewInParensSample, text.DirectionLeftToRight, 200)
	g.drawLine(screen, "RTL base: same text",
		hebrewInParensSample, text.DirectionRightToLeft, 270)
	g.drawLine(screen, "RTL base: Latin in parens between Hebrew",
		latinInParensSample, text.DirectionRightToLeft, 340)
	g.drawLine(screen, "LTR base: same text",
		latinInParensSample, text.DirectionLeftToRight, 410)

	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text Bidi (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
