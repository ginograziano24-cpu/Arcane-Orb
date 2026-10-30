package main

// retiredMods are near-duplicates of other items (the same effect at a
// different strength): they no longer turn up in rewards or shops.
var retiredMods = []string{"voidbounce", "mirror", "greed", "shrapnel", "slayer", "overcharge", "nitro", "ricochet", "hive", "extended", "secondwind", "bargain", "scavenger", "scholar", "patience", "inferno", "deepfreeze", "temper", "barrier", "echochip", "gambit"}

func init() {
	for _, id := range retiredMods {
		for _, m := range modList {
			if m.ID == id {
				m.Retired = true
			}
		}
	}
}
