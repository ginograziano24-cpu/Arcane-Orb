package mus

// "Orb Rush" — act 1. Driving electro-trance in A minor at 140 BPM: four-
// on-the-floor with a rolling sixteenth bass, syncopated supersaw stabs, a
// plucked arpeggio, and a soaring lead in the drop. Calm play keeps it in
// the verse; a climbing combo builds into the drop.

func init() {
	verse := []string{"Am", "Am", "F", "G", "Am", "Am", "F", "E"}
	drop := []string{"Am", "F", "C", "G", "Am", "F", "C", "G"}

	kick := drums("X...x...x...x...")
	kickEnd := drums("X...x...x...x.x.")
	ohat := drums("..O...O...O...O.")
	hats := drums("xgxgxgxgxgxgxgxg")
	clap := drums("....x.......x...")
	clapFill := drums("....x.......x.xx")
	ride := drums("x.o.x.o.x.o.x.o.")

	rolling := "  . 0! 0 0   . 0 12 0   . 0! 0 0   . 0 12 7 "
	var bassV, bassD, stabV, stabD, arpV, arpD, padD, padV []bar
	for _, c := range verse {
		bassV = append(bassV, bassT(rolling, c, 36))
		stabV = append(stabV, stabs("X..x..x...x..x..", c, 64))
		arpV = append(arpV, arp("0 1 2 3 4 3 2 1", c, 69, 1, 0.7))
		padV = append(padV, pad(c, 60, 0.7))
	}
	for _, c := range drop {
		bassD = append(bassD, bassT(rolling, c, 36))
		stabD = append(stabD, stabs("X..x..x...x..x..", c, 64))
		arpD = append(arpD, arp("0 2 1 3 2 4 3 5", c, 69, 1, 0.7))
		padD = append(padD, pad(c, 60, 0.75))
	}
	lead := bars(
		seqT("A5 - - E5 - - A5 - B5 - C6 - - - B5 -", -12),
		seqT("A5 - - - - - . . F5 - G5 - A5 - C6 -", -12),
		seqT("G5 - - E5 - - G5 - C6 - B5 - - - A5 -", -12),
		seqT("B5 - - - - - . . G5 - A5 - B5 - D6 -", -12),
		seqT("E6 - - C6 - - A5 - B5 - C6 - - - D6 -", -12),
		seqT("C6 - - - A5 - - - F5 - G5 - A5 - - -", -12),
		seqT("G5 - - - E5 - G5 - C6 - D6 - E6 - - -", -12),
		seqT("D6 - - - B5 - - - G5 - - - . . . .", -12),
	)
	crash := firstOf(8, at(0, 16, 0, 0.9))

	register(&songDef{
		name: "rush", bpm: 140, room: 0.6, damp: 0.3, dlySteps: 3, dlyFb: 0.35, gain: 0.95,
		pumpDepth: 0.6, pumpRel: 0.1,
		parts: []*partDef{
			{name: "kick", in: kickClub(), gain: 0.5, trigger: true},
			{name: "clap", in: clapKit(), gain: 2.6, rev: 0.18, gate: 0.12},
			{name: "ohat", in: hatKit(), gain: 1.0, pan: 0.25},
			{name: "hats", in: hatKit(), gain: 1.1, pan: -0.3, gate: 0.3, pump: 0.3},
			{name: "ride", in: rideKit(), gain: 0.62, pan: 0.35, gate: 0.6, choke: true},
			{name: "crash", in: crashKit(), gain: 0.9, rev: 0.2},
			{name: "snare", in: snareTight(), gain: 1.3, rev: 0.15},
			{name: "bass", in: bassRoll(), gain: 0.78, pump: 0.75},
			{name: "stab", in: superStab(), gain: 1.3, rev: 0.2, dly: 0.12, pump: 0.6, gate: 0.25},
			{name: "arp", in: pluckSynth(), gain: 0.95, pan: 0.2, rev: 0.15, dly: 0.3, pump: 0.5, gate: 0.5},
			{name: "pad", in: superPad(), gain: 1.1, rev: 0.35, pump: 0.55},
			{name: "lead", in: superLead(), gain: 1.5, rev: 0.28, dly: 0.22, pump: 0.3, gate: 0.42},
			{name: "riser", in: riserFX(), gain: 0.8, rev: 0.3},
			{name: "down", in: downFX(), gain: 0.8, rev: 0.4},
			{name: "impact", in: impactFX(), gain: 1.0, rev: 0.25},
		},
		first: "intro",
		secs: map[string]*secDef{
			"intro": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kickEnd),
					"ohat":  bars(ohat),
					"bass":  bars(silence, silence, silence, silence, bassV[0], bassV[1], bassV[2], bassV[3]),
					"riser": lastOf(8, at(0, 16, 57, 0.8)),
				},
				sweep: map[string][2]float64{"*": {500, 18000}}},
			"A": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kickEnd),
					"clap":  fill(8, clap, clapFill),
					"ohat":  bars(ohat),
					"hats":  bars(hats),
					"bass":  bassV,
					"stab":  stabV,
					"arp":   arpV,
					"crash": crash,
				}},
			"B": {bars: 4,
				pat: map[string][]bar{
					"kick":  bars(kick, kick, kick, silence),
					"ohat":  bars(ohat),
					"hats":  bars(hats),
					"bass":  bars(bassV[4], bassV[5], bassV[6], bassV[7]),
					"stab":  bars(stabV[4], stabV[5], stabV[6], stabV[7]),
					"snare": bars(roll(0, 4, 0.4, 0.55), roll(0, 2, 0.5, 0.7), roll(0, 1, 0.55, 0.85), roll(0, 1, 0.7, 1)),
					"riser": firstOf(4, at(0, 64, 57, 0.9)),
				},
				sweep: map[string][2]float64{"bass": {250, 5000}, "stab": {400, 9000}}},
			"C": {bars: 16,
				pat: map[string][]bar{
					"kick":   fill(8, kick, kickEnd),
					"clap":   fill(8, clap, clapFill),
					"ohat":   bars(ohat),
					"hats":   bars(hats),
					"ride":   bars(ride),
					"bass":   bassD,
					"stab":   stabD,
					"arp":    arpD,
					"lead":   lead,
					"crash":  crash,
					"impact": firstOf(16, at(0, 16, 0, 0.9)),
				}},
			"D": {bars: 8,
				pat: map[string][]bar{
					"pad":   padV,
					"arp":   arpV,
					"lead":  bars(lead[0], lead[1], lead[2], lead[3], lead[4], lead[5], lead[6], silence),
					"bass":  bars(silence, silence, silence, silence, bassV[4], bassV[5], bassV[6], bassV[7]),
					"down":  firstOf(8, at(0, 32, 69, 0.8)),
					"riser": lastOf(8, at(0, 16, 57, 0.8)),
					"ohat":  bars(silence, silence, silence, silence, silence, silence, ohat, ohat),
				},
				sweep: map[string][2]float64{"arp": {800, 6000}}},
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
