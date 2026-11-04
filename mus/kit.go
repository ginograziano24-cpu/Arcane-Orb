package mus

// Instrument presets shared by the songs.

func kickClub() inst {
	return inst{kind: iKick, f0: 200, f1: 50, pd: 0.03, decay: 0.25, click: 0.4, drive: 2.4, level: 1}
}
func kickBreak() inst {
	return inst{kind: iKick, f0: 155, f1: 56, pd: 0.022, decay: 0.19, click: 0.55, drive: 1.7, level: 1}
}
func kickHard() inst {
	return inst{kind: iKick, f0: 240, f1: 46, pd: 0.028, decay: 0.34, click: 0.45, drive: 3.2, level: 1}
}
func snareTight() inst {
	return inst{kind: iSnare, f0: 195, cut: 1900, tone: 4300, decay: 0.13, level: 0.8}
}
func snareFat() inst {
	return inst{kind: iSnare, f0: 175, cut: 1400, tone: 3600, decay: 0.2, level: 0.85}
}
func clapKit() inst { return inst{kind: iClap, tone: 1150, decay: 0.12, level: 0.75} }
func hatKit() inst {
	return inst{kind: iHat, f0: 410, tone: 10000, cut: 7000, decay: 0.034, bright: 0.45, level: 0.5}
}
func rideKit() inst {
	return inst{kind: iRide, f0: 380, tone: 6800, cut: 4200, decay: 0.85, bright: 0.35, level: 0.32}
}
func crashKit() inst {
	return inst{kind: iCrash, f0: 300, tone: 7200, cut: 3500, decay: 1.3, bright: 0.7, level: 0.55}
}
func shakerKit() inst { return inst{kind: iShaker, tone: 7600, decay: 0.045, level: 0.35} }
func tomKit() inst {
	return inst{kind: iTom, pd: 0.05, decay: 0.26, click: 0.25, drive: 1.4, level: 0.8}
}
func cowbellKit() inst { return inst{kind: iCowbell, f0: 560, tone: 2600, decay: 0.22, level: 0.4} }
func rimKit() inst     { return inst{kind: iRim, f0: 1750, tone: 3200, decay: 0.01, level: 0.45} }
func scratchKit() inst { return inst{kind: iScratch, f0: 150, level: 0.55} }

func bassFunk() inst {
	return inst{kind: iFunkBass, cut: 260, res: 1.1, env: 3.4, fdec: 0.1, a: 0.001, d: 0.35, s: 0.6, r: 0.04,
		glide: 0.03, sub: 0.3, level: 0.6}
}
func bassRoll() inst {
	return inst{kind: iAcid, cut: 240, res: 1.3, env: 3, fdec: 0.075, a: 0.001, d: 0.12, s: 0.5, r: 0.02,
		drive: 1.8, level: 0.55}
}
func bassAcid() inst {
	return inst{kind: iAcid, cut: 230, res: 7, env: 3.6, fdec: 0.17, a: 0.002, d: 0.3, s: 0.85, r: 0.015,
		glide: 0.055, drive: 2.4, level: 0.42}
}
func bassReese() inst {
	return inst{kind: iReese, cut: 520, res: 1.4, env: 1.2, fdec: 0.15, a: 0.003, d: 0.4, s: 1, r: 0.05,
		glide: 0.04, drive: 1.9, sub: 0.45, level: 0.5}
}
func bassSub() inst {
	return inst{kind: iSub, drive: 1.6, a: 0.003, d: 0.3, s: 0.9, r: 0.05, glide: 0.03, level: 0.6}
}

func superStab() inst {
	return inst{kind: iSuper, voices: 7, detune: 0.22, cut: 2300, res: 0.8, env: 1.6, fdec: 0.16,
		a: 0.001, d: 0.2, s: 0, r: 0.07, level: 0.42}
}
func superPad() inst {
	return inst{kind: iSuper, voices: 5, detune: 0.18, cut: 1500, res: 0.7, env: 0.4, fdec: 1.2,
		a: 0.45, d: 1.5, s: 0.85, r: 1.1, level: 0.22}
}
func superLead() inst {
	return inst{kind: iSuper, voices: 5, detune: 0.13, cut: 4200, res: 0.9, env: 1, fdec: 0.25,
		a: 0.004, d: 0.4, s: 0.8, r: 0.18, vib: 0.22, sub: 0.25, level: 0.34}
}
func squareLead() inst {
	return inst{kind: iLead, pw: 0.32, bright: 0.35, cut: 3000, res: 1, env: 1.3, fdec: 0.18,
		a: 0.003, d: 0.3, s: 0.8, r: 0.1, glide: 0.05, vib: 0.3, level: 0.36}
}
func pluckSynth() inst {
	return inst{kind: iPluck, cut: 650, res: 1.5, env: 3.1, fdec: 0.085, a: 0.001, d: 0.2, s: 0, r: 0.08, level: 0.34}
}
func ePiano() inst { return inst{kind: iEPiano, a: 0.002, d: 1.3, s: 0.25, r: 0.28, level: 0.5} }
func brassSec() inst {
	return inst{kind: iBrass, cut: 850, res: 1.0, env: 2.4, fdec: 0.22, a: 0.012, d: 0.25, s: 0.75, r: 0.11, level: 0.45}
}
func bellKit() inst  { return inst{kind: iBell, a: 0.001, d: 1.1, s: 0, r: 0.5, level: 0.3} }
func riserFX() inst  { return inst{kind: iRiser, tone: 1, level: 0.35} }
func downFX() inst   { return inst{kind: iDown, tone: 0.4, level: 0.32} }
func impactFX() inst { return inst{kind: iImpact, level: 0.7} }

// ---------------------------------------------------------------- helpers

// fill makes an n-bar list: the normal bar, with the fill as the last.
func fill(n int, normal, last bar) []bar {
	out := make([]bar, n)
	for i := range out {
		out[i] = normal
	}
	out[n-1] = last
	return out
}

// at places a single hit (crash, impact) at a step with a length.
func at(step, dur, key, vel float64) bar {
	return bar{{at: step, dur: dur, key: key, vel: vel}}
}

// firstOf puts b in the first of n bars and leaves the rest empty.
func firstOf(n int, b bar) []bar {
	out := make([]bar, n)
	out[0] = b
	for i := 1; i < n; i++ {
		out[i] = silence
	}
	return out
}

// lastOf puts b in the last of n bars.
func lastOf(n int, b bar) []bar {
	out := make([]bar, n)
	for i := range out {
		out[i] = silence
	}
	out[n-1] = b
	return out
}

// scr is a scratch: move 0 baby, 1 chirp, 2 transformer.
func scr(step, dur float64, move int) ev {
	return ev{at: step, dur: dur, key: float64(move), vel: 0.9}
}

// roll is a snare roll: hits every `every` steps from `from`, rising.
func roll(from, every int, v0, v1 float64) bar {
	var b bar
	n := (16 - from + every - 1) / every
	for i := 0; i < n; i++ {
		u := float64(i) / float64(max(1, n-1))
		b = append(b, ev{at: float64(from + i*every), dur: 1, vel: v0 + (v1-v0)*u})
	}
	return b
}

// cycle picks names from a fixed order by how often the song has moved on.
func (sp *songPlayer) cycle(order ...string) string {
	n := 0
	for _, k := range sp.plays {
		n += k
	}
	return order[n%len(order)]
}

var songs = map[string]*songDef{}

func register(s *songDef) { songs[s.name] = s }

// Names lists the songs.
func Names() []string {
	var out []string
	for k := range songs {
		out = append(out, k)
	}
	return out
}
