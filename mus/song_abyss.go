package mus

// "Abyss Engine" — act 3. Dark, relentless techno-trance in B minor at 152
// BPM with phrygian turns: a galloping detuned bass, pounding kick and
// toms, ride and open hats, a plucked minor arpeggio, and a wide vibrato
// lead over the drop.

func init() {
	verse := []string{"Bm", "Bm", "C", "Bm", "Bm", "Bm", "G", "F#"}
	drop := []string{"Bm", "G", "C", "F#", "Bm", "G", "Em", "F#7"}

	kick := drums("X...x...x...x...")
	kickEnd := drums("X...x...x...x.xx")
	ohat := drums("..O...O...O...O.")
	hats := drums("x.xxx.xxx.xxx.xx")
	clap := drums("....X.......X...")
	ride := drums("x.x.x.x.x.x.x.x.")
	tomFill := toms("........x.x.x.xx", 52, 50, 47, 45, 40)
	tomGallop := toms("x..x..x.........", 40, 43, 45)

	gallop := "0! . 0 0 0! . 0 0 0! . 0 0 0! . 12 0"
	var bassV, bassD, arpV, arpD, padD []bar
	for _, c := range verse {
		bassV = append(bassV, bassT(gallop, c, 34))
		arpV = append(arpV, arp("0 1 2 1 3 1 2 1", c, 66, 1, 0.65))
	}
	for _, c := range drop {
		bassD = append(bassD, bassT(gallop, c, 34))
		arpD = append(arpD, arp("0 2 4 2 1 3 5 3", c, 66, 1, 0.7))
		padD = append(padD, pad(c, 62, 0.6))
	}
	lead := bars(
		seq("B4 - - - F#5 - - - D5 - - C#5 - - B4 -"),
		seq("D5 - - - B5 - - - G5 - - F#5 - - D5 -"),
		seq("E5 - - - G5 - - - C6 - - B5 - - G5 -"),
		seq("A#5 - - - - - - - C#6 - - - A#5 - F#5 -"),
		seq("B5 - - - F#5 - - - D6 - - C#6 - - B5 -"),
		seq("B5 - - - G5 - - - D6 - - B5 - - G5 -"),
		seq("G5 - - - E5 - - - B5 - - A5 - - G5 -"),
		seq("F#5 - - - E5 - - - C#5 - - - - - - -"),
	)
	crash := firstOf(8, at(0, 16, 0, 0.9))

	register(&songDef{
		name: "abyss", bpm: 152, room: 0.7, damp: 0.3, dlySteps: 3, dlyFb: 0.4, gain: 0.95,
		pumpDepth: 0.55, pumpRel: 0.085,
		parts: []*partDef{
			{name: "kick", in: kickHard(), gain: 0.33, trigger: true},
			{name: "clap", in: clapKit(), gain: 2.2, rev: 0.22, gate: 0.15},
			{name: "ohat", in: hatKit(), gain: 1.0, pan: 0.25},
			{name: "hats", in: hatKit(), gain: 0.95, pan: -0.3, gate: 0.3, pump: 0.3},
			{name: "ride", in: rideKit(), gain: 0.6, pan: 0.35, gate: 0.6, choke: true},
			{name: "toms", in: tomKit(), gain: 0.85, rev: 0.2},
			{name: "crash", in: crashKit(), gain: 0.9, rev: 0.25},
			{name: "snare", in: snareFat(), gain: 1.2, rev: 0.18},
			{name: "bass", in: bassReese(), gain: 1.3, mono: true, pump: 0.7},
			{name: "arp", in: pluckSynth(), gain: 0.95, pan: 0.2, rev: 0.2, dly: 0.3, pump: 0.5, gate: 0.25},
			{name: "pad", in: superPad(), gain: 1.0, rev: 0.4, pump: 0.6},
			{name: "lead", in: superLead(), gain: 1.45, rev: 0.32, dly: 0.25, pump: 0.3, gate: 0.42},
			{name: "riser", in: riserFX(), gain: 0.8, rev: 0.3},
			{name: "down", in: downFX(), gain: 0.8, rev: 0.4},
			{name: "impact", in: impactFX(), gain: 1.1, rev: 0.3},
		},
		first: "intro",
		secs: map[string]*secDef{
			"intro": {bars: 8,
				pat: map[string][]bar{
					"kick":   fill(8, kick, kickEnd),
					"toms":   bars(tomGallop),
					"bass":   bars(silence, silence, silence, silence, bassV[0], bassV[1], bassV[2], bassV[3]),
					"impact": firstOf(8, at(0, 16, 0, 0.9)),
					"riser":  lastOf(8, at(0, 16, 54, 0.8)),
				},
				sweep: map[string][2]float64{"*": {600, 18000}}},
			"A": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kickEnd),
					"clap":  bars(clap),
					"ohat":  bars(ohat),
					"hats":  bars(hats),
					"bass":  bassV,
					"arp":   arpV,
					"toms":  lastOf(8, tomFill),
					"crash": crash,
				}},
			"B": {bars: 4,
				pat: map[string][]bar{
					"kick":  bars(kick, kick, kick, silence),
					"ohat":  bars(ohat),
					"hats":  bars(hats),
					"bass":  bars(bassV[4], bassV[5], bassV[6], bassV[7]),
					"arp":   bars(arpV[4], arpV[5], arpV[6], arpV[7]),
					"snare": bars(roll(0, 4, 0.4, 0.55), roll(0, 2, 0.5, 0.7), roll(0, 1, 0.55, 0.85), roll(0, 1, 0.7, 1)),
					"riser": firstOf(4, at(0, 64, 54, 0.9)),
				},
				sweep: map[string][2]float64{"bass": {200, 3000}, "arp": {500, 9000}}},
			"C": {bars: 16,
				pat: map[string][]bar{
					"kick":   fill(8, kick, kickEnd),
					"clap":   bars(clap),
					"ohat":   bars(ohat),
					"hats":   bars(hats),
					"ride":   bars(ride),
					"bass":   bassD,
					"arp":    arpD,
					"pad":    padD,
					"lead":   lead,
					"toms":   lastOf(8, tomFill),
					"crash":  crash,
					"impact": firstOf(16, at(0, 16, 0, 0.9)),
				}},
			"D": {bars: 8,
				pat: map[string][]bar{
					"pad":   padD,
					"arp":   arpV,
					"lead":  bars(lead[0], lead[1], lead[2], lead[3], silence, silence, silence, silence),
					"toms":  bars(silence, silence, silence, silence, tomGallop, tomGallop, tomGallop, tomFill),
					"down":  firstOf(8, at(0, 32, 71, 0.8)),
					"riser": lastOf(8, at(0, 16, 54, 0.8)),
				},
				sweep: map[string][2]float64{"arp": {700, 5000}}},
		},
		next: func(sp *songPlayer) string {
			hot := sp.lvl >= 0.45
			switch sp.sec {
			case "intro":
				return "A"
			case "A":
				if hot {
					return "B"
				}
				if sp.lvl < 0.15 && sp.plays["A"]%2 == 0 {
					return "D"
				}
				return "A"
			case "B":
				return "C"
			case "C":
				if hot && sp.plays["C"]%3 != 0 {
					return "C"
				}
				return "D"
			case "D":
				if hot {
					return "B"
				}
				if sp.lvl < 0.15 {
					return "D"
				}
				return "A"
			}
			return "A"
		},
	})
}
