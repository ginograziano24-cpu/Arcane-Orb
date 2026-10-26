package main

import "image/color"

// Fusions: owning two specific items forges a new, stronger item with its
// own effect, art and colour. Owning one half makes the partner appear
// more often, and its card is marked "FUSES WITH …" so players can plan.

type Fusion struct {
	ID, Name, Desc string
	A, B           string // the two mods that fuse
	Col            color.RGBA
	Apply          func(s *Stats)
}

var fusions = []Fusion{
	{"plasma", "Plasma Storm", "Lightning ignites what it strikes, and burning bricks discharge arcs when hit.", "kindling", "static", color.RGBA{255, 110, 220, 255}, func(s *Stats) { s.PlasmaStorm = true; s.ChainChance += 0.05 }},
	{"steam", "Steam Engine", "Bricks that are burning and frozen at once burst in scalding steam.", "kindling", "frost", color.RGBA{230, 236, 245, 255}, func(s *Stats) { s.SteamEngine = true; s.BurnChance += 0.05; s.FreezeChance += 0.05 }},
	{"cryo", "Cryo Charge", "Explosions flash-freeze every brick they hit.", "blast", "frost", color.RGBA{140, 220, 255, 255}, func(s *Stats) { s.CryoCharge = true }},
	{"lancer", "Lancer Formation", "Every orb pierces 2 extra bricks.", "twin", "pierce", color.RGBA{255, 240, 170, 255}, func(s *Stats) { s.Pierce += 2 }},
	{"goldrush", "Gold Rush", "Each coin you catch adds +2% damage for the rest of the chamber (up to +60%).", "goldpaddle", "magnet", color.RGBA{255, 214, 90, 255}, func(s *Stats) { s.GoldRush = true }},
	{"thunderclap", "Thunderclap", "Echo strikes call down a bolt from the sky that chains through 3 bricks.", "echo", "thunder", color.RGBA{170, 220, 255, 255}, func(s *Stats) { s.Thunderclap = true }},
	{"berserker", "Berserker Rage", "Below half health your orb burns red and deals another ×1.5 damage.", "glass", "bloodmoon", color.RGBA{255, 60, 60, 255}, func(s *Stats) { s.Berserker = true }},
	{"teslacoil", "Tesla Coil", "Laser hits chain lightning to 2 more bricks.", "laser", "static", color.RGBA{120, 240, 255, 255}, func(s *Stats) { s.TeslaCoil = true }},
	{"voidprism", "Void Prism", "Phantom orbs strike with the void: they ignore armour and deal double damage.", "voidtouch", "prism", color.RGBA{180, 110, 255, 255}, func(s *Stats) { s.VoidPrism = true }},
	{"brood", "Hydra Brood", "A new orb hatches every 15 bricks instead of 30.", "hydra", "split", color.RGBA{120, 255, 160, 255}, func(s *Stats) { s.Hydra = true; s.SplitChance += 0.05 }},
	{"perpetual", "Perpetual Motion", "Combo damage can reach +200%, and your combo lasts 2 seconds longer.", "momentum", "comboeng", color.RGBA{255, 230, 120, 255}, func(s *Stats) { s.ComboTime += 2; s.ComboDmg += 0.01 }},
	{"sanctuary", "Sanctuary", "Each barrier save heals 5 health, and barriers recharge.", "leech", "angel", color.RGBA{140, 255, 180, 255}, func(s *Stats) { s.Sanctuary = true; s.BarrierRegen = true }},
	{"cataclysm", "Cataclysm", "Every 12 impacts, a meteor slams into the bricks.", "meteor", "fuse", color.RGBA{255, 130, 40, 255}, func(s *Stats) { s.Cataclysm = true }},
	{"volcano", "Volcano", "Explosions ignite bricks, and burning bricks erupt when destroyed.", "molten", "wildfire", color.RGBA{255, 90, 30, 255}, func(s *Stats) { s.Volcano = true }},
	{"hawk", "Hawkstrike", "+15% critical chance; critical hits also strike the brick behind.", "sharp", "eagle", color.RGBA{255, 200, 120, 255}, func(s *Stats) { s.Hawkstrike = true; s.CritChance += 0.15 }},
	{"abszero", "Absolute Zero", "Frozen bricks take triple damage and shatter in a huge radius.", "frozenheart", "shatter", color.RGBA{200, 245, 255, 255}, func(s *Stats) { s.AbsoluteZero = true; s.Shatter = true }},
	{"haymaker", "Haymaker", "Charged paddle hits deal ×2.5 damage and add +3 combo.", "heavypad", "brutal", color.RGBA{255, 150, 90, 255}, func(s *Stats) { s.Haymaker = true }},
	{"hoard", "Dragon's Hoard", "Gold +25%, and interest can pay out up to 60.", "interest", "idol", color.RGBA{255, 190, 60, 255}, func(s *Stats) { s.GoldMult += 0.25; s.Interest += 0.05 }},
}

var fusionByID = func() map[string]*Fusion {
	m := map[string]*Fusion{}
	for i := range fusions {
		m[fusions[i].ID] = &fusions[i]
	}
	return m
}()

// fusionPartners lists, for a mod, the partners that would fuse with it.
func fusionPartners(id string) []string {
	var out []string
	for _, f := range fusions {
		if f.A == id {
			out = append(out, f.B)
		} else if f.B == id {
			out = append(out, f.A)
		}
	}
	return out
}

// fusionWith returns the fusion that mod id would complete for this run.
func (r *Run) fusionWith(id string) *Fusion {
	for i := range fusions {
		f := &fusions[i]
		if r.HasFusion(f.ID) {
			continue
		}
		if (f.A == id && r.Owned(f.B)) || (f.B == id && r.Owned(f.A)) {
			return f
		}
	}
	return nil
}

func (r *Run) HasFusion(id string) bool {
	for _, f := range r.Fused {
		if f == id {
			return true
		}
	}
	return false
}
