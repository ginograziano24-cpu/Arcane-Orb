package mus

import (
	"sort"
	"strconv"
	"strings"
)

// A compact notation for writing the songs.
//
// Drum lines are one character per sixteenth: X accent, x hit, o soft,
// g ghost, O open hat, . rest. Spaces and | are ignored.
//
// Note lines are whitespace-separated tokens, one per sixteenth: a note
// name with octave (A3, C#4, Bb2), "-" to hold the previous note, "." for
// a rest. Suffix ! accents a note, ' softens it; prefix ~ slides into it
// from the previous note (mono parts).
//
// Chords are symbols (Am, Fmaj7, G13, A7#9 ...) voiced close around a
// centre note, so progressions move smoothly.

var drumVel = map[rune]float64{'X': 1, 'x': 0.8, 'o': 0.55, 'g': 0.3, 'O': 0.75}

func drums(s string) bar {
	var b bar
	step := 0.0
	for _, c := range s {
		if c == ' ' || c == '|' {
			continue
		}
		if v, ok := drumVel[c]; ok {
			k := 0.0
			if c == 'O' {
				k = 1
			}
			b = append(b, ev{at: step, dur: 1, key: k, vel: v})
		}
		step++
	}
	return b
}

// toms plays a drum line where each hit takes the next pitch from keys.
func toms(s string, keys ...int) bar {
	b := drums(s)
	for i := range b {
		b[i].key = float64(keys[i%len(keys)])
	}
	return b
}

var noteBase = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// noteNum parses "C#4" / "Bb2" into a MIDI note (C4 = 60).
func noteNum(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n, ok := noteBase[s[0]]
	if !ok {
		return 0, false
	}
	i := 1
	for i < len(s) && (s[i] == '#' || s[i] == 'b') {
		if s[i] == '#' {
			n++
		} else {
			n--
		}
		i++
	}
	o, err := strconv.Atoi(s[i:])
	if err != nil {
		return 0, false
	}
	return n + 12*(o+1), true
}

func tokens(s string) []string {
	var out []string
	for _, t := range strings.Fields(s) {
		if t != "|" {
			out = append(out, t)
		}
	}
	return out
}

// seq parses a note line; transpose shifts every note.
func seq(s string) bar { return seqT(s, 0) }

func seqT(s string, transpose int) bar {
	var b bar
	for i, t := range tokens(s) {
		switch t {
		case ".":
			continue
		case "-":
			if len(b) > 0 && b[len(b)-1].at+b[len(b)-1].dur == float64(i) {
				b[len(b)-1].dur++
			}
			continue
		}
		e := ev{at: float64(i), dur: 1, vel: 0.8}
		if strings.HasPrefix(t, "~") {
			e.slide = true
			t = t[1:]
		}
		for strings.HasSuffix(t, "!") || strings.HasSuffix(t, "'") {
			if t[len(t)-1] == '!' {
				e.vel = 1
			} else {
				e.vel = 0.55
			}
			t = t[:len(t)-1]
		}
		n, ok := noteNum(t)
		if !ok {
			panic("mus: bad note " + t)
		}
		e.key = float64(n + transpose)
		b = append(b, e)
	}
	return b
}

// ---------------------------------------------------------------- chords

var chordIv = map[string][]int{
	"": {0, 4, 7}, "m": {0, 3, 7}, "5": {0, 7, 12}, "7": {0, 4, 7, 10},
	"m7": {0, 3, 7, 10}, "maj7": {0, 4, 7, 11}, "m9": {0, 3, 7, 10, 14},
	"9": {0, 4, 7, 10, 14}, "13": {0, 4, 10, 14, 21}, "7#9": {0, 4, 10, 15},
	"7b9": {0, 4, 7, 10, 13}, "sus4": {0, 5, 7}, "sus2": {0, 2, 7},
	"dim": {0, 3, 6}, "add9": {0, 4, 7, 14}, "madd9": {0, 3, 7, 14},
	"6": {0, 4, 7, 9}, "m6": {0, 3, 7, 9}, "m11": {0, 3, 7, 10, 17},
	"maj9": {0, 4, 7, 11, 14}, "7sus4": {0, 5, 7, 10},
}

// chordRoot returns the chord's root pitch class and its quality suffix.
func chordRoot(sym string) (int, string) {
	n := noteBase[sym[0]]
	i := 1
	for i < len(sym) && (sym[i] == '#' || sym[i] == 'b') {
		if sym[i] == '#' {
			n++
		} else {
			n--
		}
		i++
	}
	return (n + 12) % 12, sym[i:]
}

// chordNotes voices a chord close around centre (a MIDI note).
func chordNotes(sym string, centre int) []int {
	root, q := chordRoot(sym)
	iv, ok := chordIv[q]
	if !ok {
		panic("mus: bad chord " + sym)
	}
	seen := map[int]bool{}
	var out []int
	for _, x := range iv {
		pc := (root + x) % 12
		n := centre - 6 + ((pc-(centre-6))%12+12)%12
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// splitBar spreads the chords of "Am|F" evenly across a bar.
func splitBar(chords string) []string { return strings.Split(chords, "|") }

// stabs plays chords on a rhythm (drum characters; "-" extends a hit).
func stabs(rhythm, chords string, centre int) bar {
	cs := splitBar(chords)
	var b bar
	step := 0
	for _, c := range rhythm {
		if c == ' ' {
			continue
		}
		if c == '-' {
			for i := len(b) - 1; i >= 0 && b[i].at+b[i].dur == float64(step); i-- {
				b[i].dur++
			}
			step++
			continue
		}
		if v, ok := drumVel[c]; ok {
			sym := cs[min(len(cs)-1, step*len(cs)/16)]
			for _, n := range chordNotes(sym, centre) {
				b = append(b, ev{at: float64(step), dur: 1, key: float64(n), vel: v})
			}
		}
		step++
	}
	return b
}

// pad holds each chord of the bar for its whole share.
func pad(chords string, centre int, vel float64) bar {
	cs := splitBar(chords)
	var b bar
	w := 16 / float64(len(cs))
	for i, sym := range cs {
		for _, n := range chordNotes(sym, centre) {
			b = append(b, ev{at: float64(i) * w, dur: w, key: float64(n), vel: vel})
		}
	}
	return b
}

// bassT plays a line of semitone offsets from each chord's root, the root
// placed at or above the MIDI note low (tokens as in seq: "-", ".", !, ', ~).
func bassT(tmpl, chords string, low int) bar {
	cs := splitBar(chords)
	var b bar
	for i, t := range tokens(tmpl) {
		switch t {
		case ".":
			continue
		case "-":
			if len(b) > 0 && b[len(b)-1].at+b[len(b)-1].dur == float64(i) {
				b[len(b)-1].dur++
			}
			continue
		}
		e := ev{at: float64(i), dur: 1, vel: 0.8}
		if strings.HasPrefix(t, "~") {
			e.slide = true
			t = t[1:]
		}
		for strings.HasSuffix(t, "!") || strings.HasSuffix(t, "'") {
			if t[len(t)-1] == '!' {
				e.vel = 1
			} else {
				e.vel = 0.55
			}
			t = t[:len(t)-1]
		}
		off, err := strconv.Atoi(t)
		if err != nil {
			panic("mus: bad offset " + t)
		}
		root, _ := chordRoot(cs[min(len(cs)-1, i*len(cs)/16)])
		e.key = float64(low + ((root-low)%12+12)%12 + off)
		b = append(b, e)
	}
	return b
}

// arp walks the chord tones in the given order, one every rate steps.
func arp(order string, chords string, centre, rate int, vel float64) bar {
	cs := splitBar(chords)
	idx := tokens(order)
	var b bar
	k := 0
	for step := 0; step < 16; step += rate {
		sym := cs[min(len(cs)-1, step*len(cs)/16)]
		ns := chordNotes(sym, centre)
		t := idx[k%len(idx)]
		k++
		if t == "." {
			continue
		}
		j, _ := strconv.Atoi(t)
		oct := 0
		for j >= len(ns) {
			j -= len(ns)
			oct += 12
		}
		v := vel
		if step%4 == 0 {
			v = vel * 1.15
		}
		b = append(b, ev{at: float64(step), dur: float64(rate) * 0.9, key: float64(ns[j] + oct), vel: v})
	}
	return b
}

// one is a single note held for dur steps.
func one(note string, at, dur, vel float64) bar {
	n, _ := noteNum(note)
	return bar{{at: at, dur: dur, key: float64(n), vel: vel}}
}

// cat joins bars played together (layering several lines in one part).
func cat(bs ...bar) bar {
	var out bar
	for _, b := range bs {
		out = append(out, b...)
	}
	return out
}

// rep repeats a list of bars k times.
func rep(k int, bs ...bar) []bar {
	var out []bar
	for i := 0; i < k; i++ {
		out = append(out, bs...)
	}
	return out
}

// bars is a readable list constructor.
func bars(bs ...bar) []bar { return bs }

var silence = bar{}
