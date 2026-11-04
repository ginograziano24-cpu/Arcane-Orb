package mus

// "Warden" — the boss theme. D harmonic minor at 158 BPM: a sixteenth-note
// supersaw ostinato accented 3-3-2, war toms, a sub bass on eighths, a
// choir-like pad and a heroic lead over the full section.

func init() {
	prog := []string{"Dm", "Bb", "Gm", "A", "Dm", "Bb", "Gm", "A7"}

	kick := drums("X...x...x...x...")
	kickEnd := drums("X...x...x..xx.xx")
	clap := drums("....X.......X...")
	hats := drums("xxxxxxxxxxxxxxxx")
	ohat := drums("..O...O...O...O.")
	war := toms("x..x..x...x..x..", 40, 43, 40, 45, 43)
	warFill := toms("x.x.x.x.xxxxxxxx", 52, 50, 47, 45, 43, 40)

	var ost, bass, padP, ostLow []bar
	for _, c := range prog {
		ns := chordNotes(c, 62)
		var b bar
		acc := []float64{1, 0.55, 0.6, 0.95, 0.55, 0.6, 0.95, 0.6}
		for i := 0; i < 16; i++ {
			n := ns[[]int{0, 1, 2, 1}[i%4]%len(ns)]
			b = append(b, ev{at: float64(i), dur: 0.8, key: float64(n + 12), vel: acc[i%8]})
		}
		ost = append(ost, b)
		ostLow = append(ostLow, stabs("X..X..X.X..X..X.", c, 57))
		bass = append(bass, bassT("0! . 0 . 0 . 0 . 0! . 0 . 0 . 12 .", c, 33))
		padP = append(padP, pad(c, 62, 0.65))
	}
	lead := bars(
		seq("D5 - - - A5 - - - F5 - E5 - D5 - C#5 -"),
		seq("D5 - - - F5 - - - Bb5 - A5 - G5 - F5 -"),
		seq("G5 - - - Bb5 - - - D6 - C6 - Bb5 - A5 -"),
		seq("C#6 - - - - - - - E6 - - - C#6 - A5 -"),
		seq("D6 - - - A5 - - - F6 - E6 - D6 - C#6 -"),
		seq("D6 - - - Bb5 - - - F6 - - - D6 - Bb5 -"),
		seq("G5 - - - Bb5 - D6 - G6 - F6 - E6 - D6 -"),
		seq("C#6 - - - E6 - - - A6 - - - - - - -"),
	)
	crash := firstOf(8, at(0, 16, 0, 0.9))

	register(&songDef{
		name: "warden", bpm: 158, room: 0.75, damp: 0.3, dlySteps: 3, dlyFb: 0.35, gain: 0.95,
		pumpDepth: 0.5, pumpRel: 0.08,
		parts: []*partDef{
			{name: "kick", in: kickHard(), gain: 0.33, trigger: true},
			{name: "clap", in: clapKit(), gain: 2.2, rev: 0.25},
			{name: "hats", in: hatKit(), gain: 0.85, pan: 0.3, gate: 0.5},
			{name: "ohat", in: hatKit(), gain: 0.95, pan: -0.25},
			{name: "toms", in: tomKit(), gain: 0.95, rev: 0.25},
			{name: "crash", in: crashKit(), gain: 0.95, rev: 0.25},
			{name: "snare", in: snareFat(), gain: 1.2, rev: 0.2},
			{name: "bass", in: bassSub(), gain: 0.45, mono: true, pump: 0.5},
			{name: "ost", in: superStab(), gain: 1.5, rev: 0.2, dly: 0.1, pump: 0.45},
			{name: "brass", in: brassSec(), gain: 0.63, pan: -0.15, rev: 0.25},
			{name: "pad", in: superPad(), gain: 1.0, rev: 0.45, pump: 0.4},
			{name: "lead", in: superLead(), gain: 1.4, rev: 0.32, dly: 0.25, pump: 0.25, gate: 0.4},
			{name: "riser", in: riserFX(), gain: 0.8, rev: 0.3},
			{name: "impact", in: impactFX(), gain: 1.2, rev: 0.3},
		},
		first: "intro",
		secs: map[string]*secDef{
			"intro": {bars: 4,
				pat: map[string][]bar{
					"impact": firstOf(4, at(0, 16, 0, 1)),
					"pad":    bars(padP[0], padP[1], padP[2], padP[3]),
					"toms":   bars(war, war, war, warFill),
					"bass":   bars(silence, silence, bass[2], bass[3]),
					"riser":  lastOf(4, at(0, 16, 50, 0.8)),
				}},
			"A": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kickEnd),
					"clap":  bars(clap),
					"ohat":  bars(ohat),
					"hats":  bars(hats),
					"bass":  bass,
					"ost":   ost,
					"toms":  bars(silence, silence, silence, war, silence, silence, silence, warFill),
					"crash": crash,
				}},
			"C": {bars: 16,
				pat: map[string][]bar{
					"kick":   fill(8, kick, kickEnd),
					"clap":   bars(clap),
					"ohat":   bars(ohat),
					"hats":   bars(hats),
					"bass":   bass,
					"ost":    ost,
					"brass":  ostLow,
					"pad":    padP,
					"lead":   lead,
					"toms":   lastOf(8, warFill),
					"crash":  crash,
					"impact": firstOf(16, at(0, 16, 0, 0.9)),
				}},
			"D": {bars: 4,
				pat: map[string][]bar{
					"pad":   bars(padP[4], padP[5], padP[6], padP[7]),
					"toms":  bars(war, war, war, warFill),
					"ost":   bars(silence, silence, ost[6], ost[7]),
					"snare": bars(silence, silence, roll(0, 2, 0.4, 0.7), roll(0, 1, 0.6, 1)),
					"riser": firstOf(4, at(0, 64, 50, 0.9)),
				},
				sweep: map[string][2]float64{"ost": {400, 8000}}},
		},
		next: func(sp *songPlayer) string {
			switch sp.sec {
			case "intro":
				return "A"
			case "A":
				return "C"
			case "C":
				if sp.plays["C"]%2 == 1 {
					return "C"
				}
				return "D"
			}
			return sp.cycle("A", "C")
		},
	})
}
