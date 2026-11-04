package mus

// "Arcane Groove" — the title and menu theme. A swung funk vamp in D
// dorian at 110 BPM: breakbeat drums with ghost notes, a slap bass with
// octave pops, electric-piano comping, a brass hook and turntable
// scratches, opening with a filter sweep.

func init() {
	const (
		ep = 64 // electric piano centre (E4)
		br = 67 // brass centre (G4)
	)
	A := []string{"Dm9", "Dm9", "G13", "G13", "Dm9", "Dm9", "G13", "A7#9"}
	B := []string{"Bbmaj7", "A7", "Dm9", "Dm9", "Bbmaj7", "A7", "Gm9", "A7#9"}

	kick := drums("X......x..x.....")
	kick2 := drums("X......x..x...o.")
	snare := drums("....X..g.g..X..g")
	snareFill := drums("....X..g.gX.xXXX")
	hats := drums("x.x.x.xox.x.x.O.")
	hats2 := drums("xgx.xgxoxgx.xgO.")
	shaker := drums("o.x.o.x.o.x.o.x.")
	cow := drums("..x.....x.x..x..")
	rim := drums("...x.....x....x.")

	bassA := bars(
		seq("D2! . . D3 . D2 . C3 D2 . F2 G2 A2 . C3 D3'"),
		seq("D2! . . D3 . D2 . C3 D2 . . A2 . G2 F2 E2"),
		seq("G1! . . G2 . G1 . F2 G1 . B1 C2 D2 . F2 G2'"),
		seq("G1! . . G2 . G1 . F2 G1 . . D2 . E2 F2 C#2"),
		seq("D2! . . D3 . D2 . C3 D2 . F2 G2 A2 . C3 D3'"),
		seq("D2! . . D3 . D2 . C3 D2 . . A2 . C3 D3 A2"),
		seq("G1! . . G2 . G1 . F2 G1 . B1 C2 D2 . F2 G2'"),
		seq("A1! . . A2 . A1 . G2 A1 . C#2 E2 G2 . E2 C#2"),
	)
	bassB := bars(
		seq("Bb1! . . Bb2 . Bb1 . A2 Bb1 . D2 F2 A2 . F2 D2"),
		seq("A1! . . A2 . A1 . G2 A1 . C#2 E2 G2 . E2 C#2"),
		seq("D2! . . D3 . D2 . C3 D2 . F2 G2 A2 . C3 D3'"),
		seq("D2! . . D3 . D2 . C3 D2 . . A2 . G2 F2 A1"),
		seq("Bb1! . . Bb2 . Bb1 . A2 Bb1 . D2 F2 A2 . F2 D2"),
		seq("A1! . . A2 . A1 . G2 A1 . C#2 E2 G2 . E2 C#2"),
		seq("G1! . . G2 . G1 . F2 G1 . Bb1 D2 F2 . D2 Bb1"),
		seq("A1! . . A2 . A1 . G2 A1 . C#2 E2 G2 E2 C#2 A1"),
	)
	var keysA, keysB, brassA, brassB, padA []bar
	for i, c := range A {
		keysA = append(keysA, stabs("..X-...x-.x-....", c, ep))
		padA = append(padA, stabs("X-------........", c, ep))
		if i == 3 || i == 7 {
			brassA = append(brassA, stabs("..........X.x.X-", c, br))
		} else {
			brassA = append(brassA, silence)
		}
	}
	for i, c := range B {
		keysB = append(keysB, stabs(".X-..x-..X-..x-.", c, ep))
		if i%2 == 0 {
			brassB = append(brassB, stabs("X-.x......X-....", c, br))
		} else {
			brassB = append(brassB, stabs("X-.x....x.x.X---", c, br))
		}
	}
	hook := bars(
		seq("A4! . . A4 . G4 . A4 . D5 . . C5 . A4 ."),
		seq("G4 . . G4 . E4 . C#4 . E4 - - - - . ."),
		seq("A4! . . A4 . G4 . A4 . D5 . . E5 . F5 ."),
		seq("E5 . D5 . C5 . A4 - - - - . G4 . A4 ."),
		seq("A4! . . A4 . G4 . A4 . D5 . . C5 . A4 ."),
		seq("G4 . . G4 . E4 . C#4 . E4 - - - - . ."),
		seq("Bb4 . . Bb4 . A4 . G4 . F4 . . G4 . A4 ."),
		seq("C#5 . . E5 . . . D5 - - - - - - - ."),
	)
	scratchA := bar{scr(8, 2, 0), scr(11, 1, 1), scr(12, 2, 0), scr(14, 2, 2)}
	scratchB := bar{scr(0, 2, 1), scr(3, 1, 1), scr(4, 4, 2), scr(10, 2, 0), scr(13, 3, 0)}

	register(&songDef{
		name: "groove", bpm: 110, swing: 0.16, human: 3,
		room: 0.55, damp: 0.35, dlySteps: 3, dlyFb: 0.3, gain: 1.0,
		pumpDepth: 0.18, pumpRel: 0.12,
		parts: []*partDef{
			{name: "kick", in: kickBreak(), gain: 0.7, trigger: true},
			{name: "snare", in: snareTight(), gain: 1.15, rev: 0.14},
			{name: "hats", in: hatKit(), gain: 1.05, pan: 0.3, choke: true},
			{name: "shaker", in: shakerKit(), gain: 0.7, pan: -0.4},
			{name: "cow", in: cowbellKit(), gain: 0.7, pan: 0.45, rev: 0.1},
			{name: "rim", in: rimKit(), gain: 1.3, pan: -0.35, rev: 0.1},
			{name: "bass", in: bassFunk(), gain: 0.72, mono: true, pump: 0.3},
			{name: "keys", in: ePiano(), gain: 0.62, pan: -0.2, rev: 0.25, dly: 0.1, pump: 0.4},
			{name: "brass", in: brassSec(), gain: 0.72, pan: 0.2, rev: 0.2},
			{name: "hook", in: squareLead(), gain: 0.72, mono: true, rev: 0.2, dly: 0.25},
			{name: "scratch", in: scratchKit(), gain: 0.72, pan: 0.15, rev: 0.05},
			{name: "crash", in: crashKit(), gain: 0.38, rev: 0.2},
			{name: "riser", in: riserFX(), gain: 0.32, rev: 0.3},
		},
		first: "open",
		secs: map[string]*secDef{
			"open": {bars: 4,
				pat: map[string][]bar{
					"kick":    bars(kick, kick, kick, kick2),
					"snare":   bars(snare, snare, snare, snareFill),
					"hats":    bars(hats),
					"bass":    bars(silence, silence, bassA[0], bassA[1]),
					"scratch": bars(silence, scratchA, silence, scratchB),
					"riser":   lastOf(4, at(0, 16, 62, 0.9)),
					"keys":    bars(silence, silence, keysA[0], keysA[1]),
				},
				sweep: map[string][2]float64{"*": {350, 18000}}},
			"A": {bars: 8,
				pat: map[string][]bar{
					"kick":    bars(kick, kick, kick, kick2, kick, kick, kick, kick2),
					"snare":   fill(8, snare, snareFill),
					"hats":    bars(hats, hats, hats, hats2),
					"shaker":  bars(shaker),
					"rim":     bars(rim),
					"bass":    bassA,
					"keys":    keysA,
					"brass":   brassA,
					"crash":   firstOf(8, at(0, 16, 0, 0.8)),
					"scratch": bars(silence, silence, silence, scratchA, silence, silence, silence, silence),
				}},
			"B": {bars: 8,
				pat: map[string][]bar{
					"kick":   bars(kick, kick, kick, kick2),
					"snare":  fill(8, snare, snareFill),
					"hats":   bars(hats2),
					"shaker": bars(shaker),
					"cow":    bars(cow),
					"bass":   bassB,
					"keys":   keysB,
					"brass":  brassB,
					"hook":   hook,
					"crash":  firstOf(8, at(0, 16, 0, 0.9)),
				}},
			"C": {bars: 4,
				pat: map[string][]bar{
					"kick":    bars(kick, kick, kick, silence),
					"hats":    bars(hats),
					"rim":     bars(rim),
					"bass":    bars(bassA[0], bassA[1], bassA[0], bassA[1]),
					"keys":    bars(padA[0], silence, padA[0], silence),
					"scratch": bars(scratchA, scratchB, scratchA, scratchB),
					"snare":   bars(silence, silence, silence, roll(0, 2, 0.35, 0.95)),
					"riser":   lastOf(4, at(0, 16, 62, 0.9)),
				},
				sweep: map[string][2]float64{"bass": {300, 6000}}},
		},
		next: func(sp *songPlayer) string { return sp.cycle("A", "B", "A", "C", "B", "B") },
	})
}
