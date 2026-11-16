package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

// Visual style (roadmap #96) and the day-to-night shift (#90). Both act on
// the world's colour grade only, so menus, cards and text stay exactly as
// designed.
//
//   Classic       the worlds as painted.
//   High contrast stronger contrast and colour, less bloom haze and a lighter
//                 vignette, so bricks and orbs stand off the floor.
//   Calm          softer colour and contrast, gentler bloom, and half the
//                 screen shake: easier on the eyes over a long session.
//
// Day to night: an act starts in clear morning light, warms to a golden
// afternoon around its middle chambers and ends at dusk going on night by
// the Guardian. It is a grade on top of each world's own, kept mild so a
// world still reads as itself.

const (
	StyleClassic = iota
	StyleContrast
	StyleCalm
)

var styleNames = []string{"Classic", "High contrast", "Calm"}

// dayPhase is how far through the act's day this chamber is: 0 morning,
// 1 night (the Guardian).
func (g *Game) dayPhase(L *Level) float64 {
	if L == nil || g.run == nil || g.settings.NoDayNight {
		return 0
	}
	if L.Node == NBoss {
		return 1
	}
	return clamp01(float64(g.run.ActDepth-1) / float64(mapRows))
}

func lerp32(a, b float32, t float64) float32 { return a + (b-a)*float32(t) }

// worldGrade is the grade a chamber is drawn with: its world's, then the
// time of day, then the player's visual style. Without bloom (Low) the
// world's own grade is left out, as it always was there, and only the time
// of day and the style apply.
func (g *Game) worldGrade(L *Level) Grade {
	gr := Grade{Saturation: 1, Contrast: 1, Tint: [3]float32{1, 1, 1}}
	if !g.settings.NoBloom {
		gr = worldGrades[L.Biome.ID]
	}
	if p := g.dayPhase(L); p > 0 {
		// morning (neutral) -> golden afternoon at 0.55 -> dusk and night
		var tint, lift [3]float32
		var con, sat, vig float64
		if p < 0.55 {
			k := p / 0.55
			tint = [3]float32{lerp32(1, 1.07, k), lerp32(1, 1.0, k), lerp32(1, 0.88, k)}
			sat, con, vig = 1+0.06*k, 1, 0
		} else {
			k := (p - 0.55) / 0.45
			tint = [3]float32{lerp32(1.07, 0.86, k), lerp32(1.0, 0.9, k), lerp32(0.88, 1.12, k)}
			lift = [3]float32{0, lerp32(0, 0.006, k), lerp32(0, 0.022, k)}
			sat, con, vig = 1.06-0.14*k, 1+0.06*k, 0.18*k
		}
		for i := range gr.Tint {
			gr.Tint[i] = nz32(gr.Tint[i], 1) * tint[i]
			gr.Lift[i] += lift[i]
		}
		gr.Saturation *= float32(sat)
		gr.Contrast *= float32(con)
		gr.Vig += float32(vig)
	}
	switch g.settings.VisualStyle {
	case StyleContrast:
		gr.Contrast *= 1.22
		gr.Saturation *= 1.15
		gr.Bloom *= 0.55
		gr.Threshold = min(0.95, gr.Threshold+0.12)
		gr.Vig *= 0.45
	case StyleCalm:
		gr.Contrast *= 0.9
		gr.Saturation *= 0.78
		gr.Bloom *= 0.6
		gr.Vig *= 0.85
	}
	return gr
}

func nz32(v, d float32) float32 {
	if v == 0 {
		return d
	}
	return v
}

// shakeScale is the visual style's share of the screen shake.
func (g *Game) shakeScale() float64 {
	if g.settings.VisualStyle == StyleCalm {
		return 0.5
	}
	return 1
}

// gradeLite draws the scene with the parts of a grade a colour matrix can
// do (tint, lift, contrast, saturation), for when bloom and the grading
// shader are off: the visual style and the time of day still apply on Low.
func gradeLite(dst, scene *ebiten.Image, gr Grade) {
	var cm colorm.ColorM
	sat := float64(nz32(gr.Saturation, 1))
	if sat != 1 {
		cm.ChangeHSV(0, sat, 1)
	}
	c := float64(nz32(gr.Contrast, 1))
	tr, tg, tb := float64(nz32(gr.Tint[0], 1)), float64(nz32(gr.Tint[1], 1)), float64(nz32(gr.Tint[2], 1))
	// contrast about mid grey, then the tint and lift
	cm.Scale(c*tr, c*tg, c*tb, 1)
	off := 0.5 * (1 - c)
	cm.Translate(off*tr+float64(gr.Lift[0]), off*tg+float64(gr.Lift[1]), off*tb+float64(gr.Lift[2]), 0)
	colorm.DrawImage(dst, scene, cm, nil)
}

// gradeIsPlain reports a grade a plain copy draws exactly.
func gradeIsPlain(gr Grade) bool {
	one := func(v float32) bool { return v == 0 || v == 1 }
	return one(gr.Saturation) && one(gr.Contrast) && one(gr.Tint[0]) && one(gr.Tint[1]) && one(gr.Tint[2]) &&
		gr.Lift == [3]float32{}
}
