package main

import (
	"image/color"
	"math"
)

// The Guardian's figure (Tripo-made models): a giant standing beyond the far
// wall during the boss fight, so the brick formation reads as its armour.
// It breathes slowly, leans in when hurt, and sinks into the ground when
// it falls.
var guardianFigures = map[string]string{
	"The Prism Colossus": "guardian_1",
	"The Crucible Golem": "guardian_2",
	"The Crystal Warden": "guardian_3",
}

// guardianLight: the backlight that separates each figure from its sky,
// and how much it glows itself (the iron Golem needs the most help).
var guardianLight = map[string]struct {
	col  color.RGBA
	a    float64
	self float32
}{
	"The Prism Colossus": {color.RGBA{200, 120, 255, 255}, 0.18, 0.05},
	"The Crucible Golem": {color.RGBA{255, 130, 40, 255}, 0.42, 0.22},
	"The Crystal Warden": {color.RGBA{140, 210, 255, 255}, 0.2, 0.05},
}

func (g *Game) drawGuardianFigure(B *Batch, L *Level, t float64) {
	bo := L.Boss
	if bo == nil {
		return
	}
	m := model(guardianFigures[bo.Name])
	if m == nil {
		return
	}
	sink := 0.0
	if bo.Dead {
		sink = math.Min(1, bo.deathT/2.5)
		if sink >= 1 {
			return
		}
	}
	// framed so its head and shoulders rise just above the brick formation
	// and stay below the boss bar (the models are 24 units tall)
	const k = 0.66
	breathe := 1 + 0.012*math.Sin(t*1.3)
	lean := 0.05*bo.Hurt + 0.02*math.Sin(t*0.7)
	y := -2.4 - sink*sink*16
	pos := V3{0, y, FarZ + 8}
	gl := guardianLight[bo.Name]
	B.Draw(m, pos, V3{k, k * breathe, k}, rotXYZ(-lean, 0.03*math.Sin(t*0.4), 0), color.RGBA{}, gl.self+float32(0.15*bo.Hurt))
	// a glow behind it so the silhouette separates from the sky
	pulse := 1 + 0.15*math.Sin(t*1.7)
	B.Billboard(V3{0, y + 11, FarZ + 11}, 26*pulse, sprSoft, alpha(gl.col, gl.a*(1-sink)), true)
	B.Billboard(V3{0, y + 13, FarZ + 12}, 14, sprSoft, alpha(lighten(gl.col, 0.3), gl.a*0.6*(1-sink)), true)
}
