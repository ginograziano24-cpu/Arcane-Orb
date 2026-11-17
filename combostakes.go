package main

import (
	"fmt"
	"image/color"
)

// Combo stakes (docs/ENJOYMENT_1_5X.md, Tier 1 #5): a long combo is worth
// protecting. At 25, 50 and 100 hits in one chamber it drops a free good
// power-up, the HUD shows the next milestone under COMBO, and dropping the
// orb with a combo of 10 or more breaks it out loud.

var comboStakeAt = [...]int{25, 50, 100}

// nextComboStake is the next milestone this chamber hasn't paid yet (0: all paid).
func (L *Level) nextComboStake() int {
	for _, m := range comboStakeAt {
		if L.hasOmen("resonant") {
			m /= 2 // 12, 25, 50
		}
		if m > L.comboPaid {
			return m
		}
	}
	return 0
}

// comboStakes pays a milestone the moment the combo reaches it.
func (L *Level) comboStakes(x, z float64) {
	m := L.nextComboStake()
	if m == 0 || L.Combo < m {
		return
	}
	L.comboPaid = m
	L.dropPower(x, z, true)
	L.text(V3{x, 2.6, z}, tr("COMBO REWARD!"), color.RGBA{255, 214, 110, 255}, true)
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 3.2, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 214, 110, 255}})
	L.g.snd.PlayKeyed("good")
}

// comboBroken: losing an orb with a big combo running says so.
func (L *Level) comboBroken() {
	if L.Combo < 10 {
		return
	}
	L.text(V3{L.PaddleX, 2.4, PaddleZ + 2}, fmt.Sprintf(tr("COMBO BROKEN  ×%d"), L.Combo), color.RGBA{255, 110, 90, 255}, true)
	L.g.snd.PlayKeyed("bad")
}
