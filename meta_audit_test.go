package main

import "testing"

// Every ascension tier adds one clear rule, and the rule is real (#21).
func TestAscensionTiersDoWhatTheySay(t *testing.T) {
	if len(ascensions) != 10 {
		t.Fatalf("%d ascension tiers", len(ascensions))
	}
	names := map[string]bool{}
	for _, o := range omens {
		names[o.Name] = true
	}
	for i, a := range ascensions {
		if names[a.Name] {
			t.Errorf("ascension %d shares the name %q with an omen", i+1, a.Name)
		}
		names[a.Name] = true
	}
	// Final Form: bosses attack faster, not only tougher
	if bossAttackGap(10) >= bossAttackGap(9) {
		t.Error("Final Form doesn't speed up boss attacks")
	}
	if ascHP(10, false, true) <= ascHP(9, false, true) || ascHP(7, false, true) <= ascHP(6, false, true) {
		t.Error("Titans / Final Form don't raise boss health")
	}
	if ascHP(1, true, false) <= ascHP(0, true, false) || ascHP(4, false, false) <= ascHP(3, false, false) || ascHP(9, false, false) <= ascHP(8, false, false) {
		t.Error("a brick-health tier does nothing")
	}
}
