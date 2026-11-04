package mus

// "Acid Cascade" — act 2. Electro breaks in G minor at 146 BPM: a squelching
// resonant bass line with slides and accents, broken kick patterns, sixteenth
// hats, chord stabs and a square-wave riff that takes over in the drop.

func init() {
	verse := []string{"Gm", "Gm", "Eb", "F", "Gm", "Gm", "Eb", "D7"}
	drop := []string{"Gm", "Eb", "Bb", "F", "Gm", "Eb", "Cm", "D7"}

	kick := drums("X.....x...X.....")
	kick2 := drums("X.....x...X..x..")
	kick4 := drums("X...x...x...x...")
	snare := drums("....X.......X...")
	snareFill := drums("....X.......X.xx")
	hats := drums("xxoxxxoxxxoxxxox")
	ohat := drums("..O...O...O...O.")
	clap := drums("....X.......X...")
	tomFill := toms("..........x.x.xx", 50, 47, 43, 40)

	acid1 := "0! 0 12 0 ~3 0 10! 0 0 ~12 0 7 ~10 0 5! 3"
	acid2 := "0! . 12 ~10 0 0 7! 0 ~3 5 0 12! 0 ~7 10 12"
	var bassV, bassD, stabD, padV []bar
	for i, c := range verse {
		t := acid1
		if i%2 == 1 {
			t = acid2
		}
		bassV = append(bassV, bassT(t, c, 36))
		padV = append(padV, pad(c, 62, 0.7))
	}
	for i, c := range drop {
		t := acid1
		if i%2 == 1 {
			t = acid2
		}
		bassD = append(bassD, bassT(t, c, 36))
		stabD = append(stabD, stabs("X-.X-.X-..X-.X-.", c, 65))
	}
	riff := bars(
		seq("D5 . G5 . Bb5 . G5 . D6 . C6 . Bb5 . A5 ."),
		seq("G5 . Eb5 . G5 . Bb5 . Eb6 . D6 . Bb5 . G5 ."),
		seq("F5 . Bb5 . D6 . Bb5 . F6 . D6 . C6 . Bb5 ."),
		seq("A5 . C6 . F6 . C6 . A5 . F5 . G5 . A5 ."),
		seq("D5 . G5 . Bb5 . G5 . D6 . C6 . Bb5 . A5 ."),
		seq("G5 . Eb5 . G5 . Bb5 . Eb6 . D6 . Bb5 . G5 ."),
		seq("Eb5 . G5 . C6 . G5 . Eb6 . D6 . C6 . Bb5 ."),
		seq("A5 - - - F#5 - - - D5 - - - . . . ."),
	)
	stabV := func() []bar {
		var out []bar
		for _, c := range verse {
			out = append(out, stabs("X.......X.......", c, 65))
		}
		return out
	}()
	crash := firstOf(8, at(0, 16, 0, 0.9))

	register(&songDef{
		name: "acid", bpm: 146, room: 0.5, damp: 0.35, dlySteps: 3, dlyFb: 0.38, gain: 0.95,
		pumpDepth: 0.45, pumpRel: 0.09,
		parts: []*partDef{
			{name: "kick", in: kickClub(), gain: 0.5, trigger: true},
			{name: "snare", in: snareFat(), gain: 1.1, rev: 0.16},
			{name: "clap", in: clapKit(), gain: 1.6, rev: 0.2, gate: 0.4},
			{name: "hats", in: hatKit(), gain: 0.95, pan: 0.3, gate: 0.2},
			{name: "ohat", in: hatKit(), gain: 0.9, pan: -0.25, gate: 0.55},
			{name: "toms", in: tomKit(), gain: 0.8, rev: 0.15},
			{name: "crash", in: crashKit(), gain: 0.9, rev: 0.2},
			{name: "acid", in: bassAcid(), gain: 0.85, mono: true, dly: 0.08, pump: 0.4},
			{name: "sub", in: bassSub(), gain: 0.55, mono: true, pump: 0.6},
			{name: "stab", in: superStab(), gain: 1.15, pan: 0.1, rev: 0.22, dly: 0.15, pump: 0.6, gate: 0.3},
			{name: "pad", in: superPad(), gain: 1.0, rev: 0.4, pump: 0.5},
			{name: "riff", in: squareLead(), gain: 1.05, mono: true, rev: 0.2, dly: 0.28, gate: 0.45},
			{name: "riser", in: riserFX(), gain: 0.8, rev: 0.3},
			{name: "impact", in: impactFX(), gain: 1.0, rev: 0.25},
		},
		first: "intro",
		secs: map[string]*secDef{
			"intro": {bars: 8,
				pat: map[string][]bar{
					"acid":  bassV,
					"hats":  bars(silence, silence, silence, silence, hats, hats, hats, hats),
					"kick":  bars(silence, silence, silence, silence, kick, kick, kick, kick2),
					"riser": lastOf(8, at(0, 16, 55, 0.8)),
				},
				sweep: map[string][2]float64{"acid": {150, 1.0e4}}},
			"A": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kick2),
					"snare": fill(8, snare, snareFill),
					"hats":  bars(hats),
					"acid":  bassV,
					"sub":   bassV,
					"stab":  stabV,
					"toms":  lastOf(8, tomFill),
					"crash": crash,
				}},
			"B": {bars: 4,
				pat: map[string][]bar{
					"kick":  bars(kick4, kick4, kick4, silence),
					"hats":  bars(hats),
					"acid":  bars(bassV[4], bassV[5], bassV[6], bassV[7]),
					"snare": bars(roll(0, 4, 0.4, 0.55), roll(0, 2, 0.5, 0.7), roll(0, 1, 0.55, 0.85), roll(0, 1, 0.7, 1)),
					"riser": firstOf(4, at(0, 64, 55, 0.9)),
				},
				sweep: map[string][2]float64{"acid": {300, 1.2e4}}},
			"C": {bars: 16,
				pat: map[string][]bar{
					"kick":   fill(8, kick4, kick4),
					"clap":   bars(clap),
					"snare":  lastOf(8, snareFill),
					"hats":   bars(hats),
					"ohat":   bars(ohat),
					"acid":   bassD,
					"sub":    bassD,
					"stab":   stabD,
					"riff":   riff,
					"crash":  crash,
					"impact": firstOf(16, at(0, 16, 0, 0.9)),
					"toms":   lastOf(16, tomFill),
				}},
			"D": {bars: 8,
				pat: map[string][]bar{
					"pad":   padV,
					"acid":  bars(silence, silence, silence, silence, bassV[4], bassV[5], bassV[6], bassV[7]),
					"riff":  bars(riff[0], silence, riff[2], silence, riff[4], silence, riff[6], riff[7]),
					"hats":  bars(silence, silence, silence, silence, hats, hats, hats, hats),
					"riser": lastOf(8, at(0, 16, 55, 0.8)),
				},
				sweep: map[string][2]float64{"acid": {200, 3000}, "riff": {700, 5000}}},
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
