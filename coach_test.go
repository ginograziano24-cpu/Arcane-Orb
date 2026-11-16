package main

import "testing"

// Laser items "fire" lasers; that is not the Burn keyword (#37).
func TestGlossaryFireIsNotLasers(t *testing.T) {
	for _, d := range []string{"Your paddle fires twin lasers automatically.", "Lasers fire 25% faster."} {
		for _, term := range glossFor(d) {
			if term.Name == "Burn" {
				t.Errorf("%q explained as Burn", d)
			}
		}
	}
	hasBurn := false
	for _, term := range tooltipTerms("Burning bricks spread fire to their neighbours when destroyed.", 3) {
		hasBurn = hasBurn || term.Name == "Burn"
	}
	if !hasBurn {
		t.Error("a burn item lost its Burn keyword")
	}
	if n := len(tooltipTerms("Crits freeze, burn and explode with lightning combo void", 3)); n != 3 {
		t.Errorf("tooltipTerms keeps at most 3, got %d", n)
	}
}
