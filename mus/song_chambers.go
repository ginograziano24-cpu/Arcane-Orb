package mus

// Chamber songs. In Magic Ball the music changes from level to level; here
// each act has a family of three songs that take turns from chamber to
// chamber: the act's original song plus two siblings. Siblings share the
// act's key (so the musical brick sounds still sit in the song), stay
// within a few BPM of it and use the same instrument palette, but each has
// its own chord loop, groove, bass line and lead melody.
//
// All six siblings use one arrangement (intro, verse, build, drop,
// breakdown) driven by the same intensity rules as the act songs, so
// calm play stays in the verse and a climbing combo builds into the drop.

type chamberCfg struct {
	name, title  string
	bpm          float64
	verse, drop  []string // 8 chords each
	kick, kick2  string   // main kick and its fill bar
	hats, ohat   string
	clap, clap2  string
	ride         string // "" for none
	bassTmpl     string
	bassLow      int
	bass         func() inst
	bassMono     bool
	bassGain     float64
	stab         string // "" for none
	arpV, arpD   string
	arpCentre    int
	arpRate      int
	arpIn        func() inst
	lead         [8]string
	leadT        int
	leadIn       func() inst
	leadGain     float64
	kickIn       func() inst
	room, dlyFb  float64
	riserNote    float64
	swing, human float64
	gain         float64 // master gain (0 = 0.95, the act songs' level)
}

func registerChamber(c chamberCfg) {
	if c.gain == 0 {
		c.gain = 0.95
	}
	kick, kick2 := drums(c.kick), drums(c.kick2)
	hats, ohat := drums(c.hats), drums(c.ohat)
	clap, clap2 := drums(c.clap), drums(c.clap2)
	var bassV, bassD, stabV, stabD, arpV, arpD, padV []bar
	for _, ch := range c.verse {
		bassV = append(bassV, bassT(c.bassTmpl, ch, c.bassLow))
		arpV = append(arpV, arp(c.arpV, ch, c.arpCentre, c.arpRate, 0.7))
		padV = append(padV, pad(ch, 60, 0.7))
		if c.stab != "" {
			stabV = append(stabV, stabs(c.stab, ch, 64))
		}
	}
	for _, ch := range c.drop {
		bassD = append(bassD, bassT(c.bassTmpl, ch, c.bassLow))
		arpD = append(arpD, arp(c.arpD, ch, c.arpCentre, c.arpRate, 0.72))
		if c.stab != "" {
			stabD = append(stabD, stabs(c.stab, ch, 64))
		}
	}
	var lead []bar
	for _, l := range c.lead {
		lead = append(lead, seqT(l, c.leadT))
	}
	crash := firstOf(8, at(0, 16, 0, 0.9))

	parts := []*partDef{
		{name: "kick", in: c.kickIn(), gain: 0.45, trigger: true},
		{name: "clap", in: clapKit(), gain: 2.2, rev: 0.2, gate: 0.12},
		{name: "ohat", in: hatKit(), gain: 0.95, pan: 0.25},
		{name: "hats", in: hatKit(), gain: 1.0, pan: -0.3, gate: 0.25, pump: 0.3},
		{name: "ride", in: rideKit(), gain: 0.6, pan: 0.35, gate: 0.6, choke: true},
		{name: "crash", in: crashKit(), gain: 0.9, rev: 0.22},
		{name: "snare", in: snareTight(), gain: 1.25, rev: 0.15},
		{name: "bass", in: c.bass(), gain: c.bassGain, mono: c.bassMono, pump: 0.7},
		{name: "stab", in: superStab(), gain: 1.15, rev: 0.22, dly: 0.14, pump: 0.6, gate: 0.3},
		{name: "arp", in: c.arpIn(), gain: 0.95, pan: 0.2, rev: 0.18, dly: 0.3, pump: 0.5, gate: 0.4},
		{name: "pad", in: superPad(), gain: 1.05, rev: 0.38, pump: 0.55},
		{name: "lead", in: c.leadIn(), gain: c.leadGain, rev: 0.28, dly: 0.24, pump: 0.3, gate: 0.42},
		{name: "riser", in: riserFX(), gain: 0.8, rev: 0.3},
		{name: "down", in: downFX(), gain: 0.8, rev: 0.4},
		{name: "impact", in: impactFX(), gain: 1.0, rev: 0.25},
	}
	dropPat := map[string][]bar{
		"kick":   fill(8, kick, kick2),
		"clap":   fill(8, clap, clap2),
		"ohat":   bars(ohat),
		"hats":   bars(hats),
		"bass":   bassD,
		"arp":    arpD,
		"lead":   lead,
		"crash":  crash,
		"impact": firstOf(16, at(0, 16, 0, 0.9)),
	}
	versePat := map[string][]bar{
		"kick":  fill(8, kick, kick2),
		"clap":  fill(8, clap, clap2),
		"ohat":  bars(ohat),
		"hats":  bars(hats),
		"bass":  bassV,
		"arp":   arpV,
		"crash": crash,
	}
	if c.stab != "" {
		dropPat["stab"] = stabD
		versePat["stab"] = stabV
	}
	if c.ride != "" {
		dropPat["ride"] = bars(drums(c.ride))
	}
	register(&songDef{
		name: c.name, bpm: c.bpm, swing: c.swing, human: c.human,
		room: c.room, damp: 0.3, dlySteps: 3, dlyFb: c.dlyFb, gain: c.gain,
		pumpDepth: 0.55, pumpRel: 0.095,
		parts: parts,
		first: "intro",
		secs: map[string]*secDef{
			"intro": {bars: 8,
				pat: map[string][]bar{
					"kick":  fill(8, kick, kick2),
					"ohat":  bars(ohat),
					"arp":   bars(silence, silence, silence, silence, arpV[0], arpV[1], arpV[2], arpV[3]),
					"riser": lastOf(8, at(0, 16, c.riserNote, 0.8)),
				},
				sweep: map[string][2]float64{"*": {600, 18000}}},
			"A": {bars: 8, pat: versePat},
			"B": {bars: 4,
				pat: map[string][]bar{
					"kick":  bars(kick, kick, kick, silence),
					"hats":  bars(hats),
					"bass":  bars(bassV[4], bassV[5], bassV[6], bassV[7]),
					"arp":   bars(arpV[4], arpV[5], arpV[6], arpV[7]),
					"snare": bars(roll(0, 4, 0.4, 0.55), roll(0, 2, 0.5, 0.7), roll(0, 1, 0.55, 0.85), roll(0, 1, 0.7, 1)),
					"riser": firstOf(4, at(0, 64, c.riserNote, 0.9)),
				},
				sweep: map[string][2]float64{"bass": {250, 5000}, "arp": {500, 9000}}},
			"C": {bars: 16, pat: dropPat},
			"D": {bars: 8,
				pat: map[string][]bar{
					"pad":   padV,
					"arp":   arpV,
					"lead":  bars(lead[0], lead[1], lead[2], lead[3], silence, silence, silence, silence),
					"bass":  bars(silence, silence, silence, silence, bassV[4], bassV[5], bassV[6], bassV[7]),
					"down":  firstOf(8, at(0, 32, c.riserNote+12, 0.8)),
					"riser": lastOf(8, at(0, 16, c.riserNote, 0.8)),
				},
				sweep: map[string][2]float64{"arp": {700, 6000}}},
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
	chamberTitles[c.name] = c.title
}

// chamberTitles names the chamber songs (for a future soundtrack screen).
var chamberTitles = map[string]string{}

// Title returns a song's display name, or "" if it has none here.
func Title(name string) string { return chamberTitles[name] }

func init() {
	// ---- act 1, A minor, around 140 BPM (siblings of "Orb Rush") --------
	registerChamber(chamberCfg{
		name: "rush2", title: "Lantern Steps", bpm: 136, swing: 0.06,
		verse: []string{"Am", "C", "G", "F", "Am", "C", "G", "E"},
		drop:  []string{"F", "G", "Am", "Am", "F", "G", "C", "E"},
		kick:  "X...x...x...x...", kick2: "X...x...x...x.x.",
		hats: "xgxgxgxgxgxgxgxg", ohat: "..O...O...O...O.",
		clap: "....x.......x...", clap2: "....x.......x.xx",
		ride:     "x.o.x.o.x.o.x.o.",
		bassTmpl: "0! . 12 . 0 . 12 0 . 0 12 . 0 . 7 .", bassLow: 33, bass: bassFunk, bassGain: 0.85,
		stab: "X...x..x..x.x...",
		arpV: "0 2 1 2 0 2 1 2", arpD: "0 1 2 3 2 1 2 3", arpCentre: 69, arpRate: 1, arpIn: pluckSynth,
		lead: [8]string{
			"E5 - - - C5 - D5 - E5 - - - G5 - E5 -",
			"D5 - - - - - . . C5 - D5 - E5 - G5 -",
			"A5 - - - G5 - E5 - D5 - - - E5 - - -",
			"B4 - - - - - . . C5 - D5 - E5 - G#5 -",
			"A5 - - - G5 - - - E5 - G5 - A5 - - -",
			"G5 - - - E5 - D5 - C5 - - - D5 - E5 -",
			"F5 - - - E5 - D5 - C5 - D5 - E5 - G5 -",
			"E5 - - - - - - - G#5 - - - . . . .",
		},
		leadT: -12, leadIn: superLead, leadGain: 1.4, kickIn: kickClub,
		room: 0.62, dlyFb: 0.36, riserNote: 57,
	})
	registerChamber(chamberCfg{
		name: "rush3", title: "Kite Weather", bpm: 142,
		verse: []string{"Am", "Dm", "Am", "E", "Am", "Dm", "F", "E"},
		drop:  []string{"Dm", "Am", "F", "E", "Dm", "Am", "G", "E"},
		kick:  "X.....x...X.....", kick2: "X.....x...X..x..",
		hats: "xxoxxxoxxxoxxxox", ohat: "......O.......O.",
		clap: "....X.......X...", clap2: "....X.......X.xx",
		bassTmpl: "0! 0 . 0 ~12 . 0 . 0! 0 . 0 ~7 . 10 .", bassLow: 33, bass: bassSub, bassMono: true, bassGain: 0.7,
		stab: "X..x..x...x..x..",
		arpV: "0 1 2 1 0 1 2 1", arpD: "0 2 4 2 1 3 5 3", arpCentre: 69, arpRate: 1, arpIn: pluckSynth,
		lead: [8]string{
			"A5 - C6 - B5 - A5 - E5 - - - . . A5 -",
			"F5 - - - A5 - - - D6 - C6 - A5 - - -",
			"C6 - - - B5 - A5 - G#5 - - - B5 - - -",
			"E6 - - - - - - - D6 - C6 - B5 - G#5 -",
			"A5 - C6 - E6 - - - D6 - C6 - A5 - - -",
			"F5 - A5 - C6 - - - B5 - A5 - F5 - - -",
			"G5 - B5 - D6 - - - C6 - B5 - G5 - - -",
			"G#5 - - - B5 - - - E6 - - - . . . .",
		},
		leadT: -12, leadIn: squareLead, leadGain: 1.05, kickIn: kickBreak,
		room: 0.55, dlyFb: 0.38, riserNote: 57,
	})

	// ---- act 2, G minor, around 146 BPM (siblings of "Acid Cascade") ----
	registerChamber(chamberCfg{
		name: "acid2", title: "Copper Gears", bpm: 144,
		verse: []string{"Gm", "Bb", "F", "Gm", "Gm", "Bb", "Eb", "D7"},
		drop:  []string{"Eb", "F", "Gm", "Gm", "Eb", "F", "Bb", "D7"},
		kick:  "X...x...x...x...", kick2: "X...x...x...x.x.",
		hats: "x.xgx.xgx.xgx.xg", ohat: "..O...O...O...O.",
		clap: "....x.......x...", clap2: "....x.......x.xx",
		ride:     "x.o.x.o.x.o.x.o.",
		bassTmpl: "0! 0 12 0 ~3 0 10! 0 0 ~12 0 7 ~10 0 5! 3", bassLow: 31, bass: bassAcid, bassMono: true, bassGain: 0.8,
		stab: "X..x..x...x..x..",
		arpV: "0 1 2 1 0 1 2 3", arpD: "0 2 1 3 2 4 3 5", arpCentre: 67, arpRate: 1, arpIn: pluckSynth,
		lead: [8]string{
			"G5 - - Bb5 - - D6 - C6 - Bb5 - A5 - - -",
			"F5 - - - A5 - C6 - Bb5 - A5 - F5 - - -",
			"G5 - - - Bb5 - D6 - F6 - - - D6 - - -",
			"F#5 - - - A5 - - - D6 - C6 - A5 - - -",
			"Eb6 - - - D6 - C6 - Bb5 - - - C6 - D6 -",
			"F6 - - - D6 - C6 - A5 - - - . . F5 -",
			"G5 - Bb5 - D6 - - - Eb6 - D6 - Bb5 - - -",
			"A5 - - - F#5 - - - D6 - - - . . . .",
		},
		leadT: -12, leadIn: squareLead, leadGain: 1.0, kickIn: kickClub,
		room: 0.5, dlyFb: 0.36, riserNote: 55,
	})
	registerChamber(chamberCfg{
		name: "acid3", title: "Steam Choir", bpm: 148, human: 2,
		verse: []string{"Gm", "Cm", "Eb", "D7", "Gm", "Cm", "Bb", "D7"},
		drop:  []string{"Cm", "Gm", "Eb", "F", "Cm", "Gm", "Eb", "D7"},
		kick:  "X.....x...X.....", kick2: "X.....x...X..x..",
		hats: "xxoxxxoxxxoxxxox", ohat: "..O...O...O...O.",
		clap: "....X.......X...", clap2: "....X.......X.xx",
		bassTmpl: "0! . 0 12 . 0 10 . 0! . 0 7 . 5 3 .", bassLow: 31, bass: bassReese, bassMono: true, bassGain: 1.3,
		arpV: "0 1 2 3 2 1 2 3", arpD: "0 2 4 2 3 1 4 2", arpCentre: 67, arpRate: 2, arpIn: ePiano,
		lead: [8]string{
			"D6 - - - C6 - Bb5 - G5 - - - Bb5 - C6 -",
			"Eb6 - - - - - D6 - C6 - - - . . G5 -",
			"Bb5 - - - C6 - D6 - Eb6 - - - G6 - - -",
			"F#6 - - - - - D6 - A5 - - - . . . .",
			"G6 - - - F6 - Eb6 - D6 - - - C6 - - -",
			"Bb5 - - - C6 - D6 - G5 - - - . . . .",
			"Eb6 - - - D6 - C6 - Bb5 - - - A5 - - -",
			"D6 - - - C6 - A5 - F#5 - - - . . . .",
		},
		leadT: -12, leadIn: superLead, leadGain: 1.35, kickIn: kickBreak,
		room: 0.58, dlyFb: 0.4, riserNote: 55,
	})

	// ---- act 3, B minor, around 152 BPM (siblings of "Abyss Engine") ----
	registerChamber(chamberCfg{
		name: "abyss2", title: "Crystal Hollow", bpm: 150,
		verse: []string{"Bm", "G", "D", "A", "Bm", "G", "Em", "F#"},
		drop:  []string{"G", "A", "Bm", "Bm", "G", "A", "D", "F#"},
		kick:  "X...x...x...x...", kick2: "X...x...x...x.x.",
		hats: "xgxgxgxgxgxgxgxg", ohat: "..O...O...O...O.",
		clap: "....x.......x...", clap2: "....x.......x.xx",
		ride:     "x.o.x.o.x.o.x.o.",
		bassTmpl: "0! 0 0 0 0! 0 0 0 0! 0 0 0 0! 0 12 0", bassLow: 35, bass: bassReese, bassMono: true, bassGain: 1.2,
		arpV: "0 1 2 3 2 1 0 1", arpD: "0 2 4 6 4 2 1 3", arpCentre: 71, arpRate: 1, arpIn: bellKit,
		lead: [8]string{
			"F#5 - - - D5 - E5 - F#5 - - - A5 - F#5 -",
			"G5 - - - - - F#5 - E5 - D5 - B4 - - -",
			"A5 - - - F#5 - D5 - E5 - - - F#5 - - -",
			"E5 - - - - - C#5 - A4 - - - . . . .",
			"B5 - - - A5 - F#5 - G5 - - - A5 - B5 -",
			"D6 - - - B5 - A5 - F#5 - - - E5 - - -",
			"G5 - - - A5 - B5 - D6 - - - C#6 - - -",
			"A#5 - - - F#5 - - - C#6 - - - . . . .",
		},
		leadT: -12, leadIn: superLead, leadGain: 1.4, kickIn: kickHard,
		room: 0.72, dlyFb: 0.42, riserNote: 59,
	})
	registerChamber(chamberCfg{
		name: "abyss3", title: "Starfall Engine", bpm: 154,
		verse: []string{"Bm", "Em", "G", "F#", "Bm", "Em", "C", "F#"},
		drop:  []string{"Em", "Bm", "G", "F#", "Em", "Bm", "C", "F#7"},
		kick:  "X...x...x...x...", kick2: "X...x...x.x.x.x.",
		hats: "xxoxxxoxxxoxxxox", ohat: "..O...O...O...O.",
		clap: "....x.......x...", clap2: "....x.......xxxx",
		bassTmpl: "  . 0! 0 0   . 0 12 0   . 0! 0 0   . 0 12 7 ", bassLow: 35, bass: bassRoll, bassGain: 0.62,
		stab: "X..x..x...x..x..",
		arpV: "0 1 2 1 0 1 2 1", arpD: "0 2 1 3 2 4 3 5", arpCentre: 71, arpRate: 1, arpIn: pluckSynth,
		lead: [8]string{
			"B5 - - - D6 - - - F#6 - E6 - D6 - - -",
			"E6 - - - - - D6 - B5 - - - G5 - - -",
			"D6 - - - B5 - G5 - A5 - B5 - D6 - - -",
			"C#6 - - - - - A#5 - F#5 - - - . . . .",
			"F#6 - - - E6 - D6 - E6 - - - G6 - - -",
			"F#6 - - - D6 - B5 - D6 - - - E6 - - -",
			"E6 - - - D6 - C6 - B5 - - - A5 - G5 -",
			"F#5 - - - A#5 - - - C#6 - - - . . . .",
		},
		leadT: -12, leadIn: superLead, leadGain: 1.3, kickIn: kickHard,
		room: 0.7, dlyFb: 0.4, riserNote: 59, gain: 0.76,
	})
}
