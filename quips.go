package main

// Hero quips (roadmap step 2).
//
// Each hero has a line for a handful of big moments. The line pops up in a
// comic speech bubble beside the HUD portrait, with a short burst of
// synthesised "voice" babble in that hero's register (old JRPG style).
//
// Quips stay rare: the frequent moment (a big combo) waits out a long
// cooldown, and every quip waits a few seconds after the last one, so the
// heroes chime in about once every minute or two and never talk over each
// other. Settings > Audio turns them off. The lines are drafts from
// docs/quips.md; edit both together.

import (
	"image/color"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type quipMoment int

const (
	quipPick      quipMoment = iota // chosen on the origin screen
	quipStart                       // a run begins
	quipCombo                       // a big combo
	quipBoss                        // a Guardian falls
	quipLegendary                   // a legendary (or better) card is taken
	quipLastLife                    // down to the last life
	quipWin                         // the story is finished
	numQuipMoments
)

// heroQuips holds one line per moment for each origin, in quipMoment order.
var heroQuips = map[string][numQuipMoments]string{
	"guardian": {"Shield up. Snacks packed.", "Stay behind me. All of you.", "Nothing gets through!",
		"That's what you get for building walls.", "Finally, proper equipment.", "Steady... I've got this.", "Everyone home by bedtime."},
	"berserker": {"Point me at a wall.", "Nap's over. Smashing time.", "MORE BRICKS!",
		"Should've built it out of something tougher.", "Ooh. Shiny AND heavy.", "I'm not done hitting things!", "Now I can rebuild my house."},
	"arcane": {"Fascinating. Let us begin.", "Fire, frost, storm, void. In that order.", "Textbook! Literally, chapter four.",
		"I'll be writing a paper on that.", "I've only ever read about these!", "This was not in the syllabus.", "Class dismissed."},
	"engineer": {"Paddle's calibrated.", "Hold still. Aim. Fire.", "Running at peak efficiency!",
		"Shutting that one down.", "Ooh, I can upgrade this.", "Okay. Think. Think.", "Told you it would work."},
	"lucky": {"Feeling lucky? I always am.", "House rules: I win.", "Jackpot, baby!",
		"All in, and it paid off.", "Read 'em and weep.", "Double or nothing...", "Drinks are on Oriel."},
	"pyro": {"Let's light this up.", "Who brought the marshmallows?", "Burn, baby, burn!",
		"Ooh, that's gonna leave a scorch mark.", "It's even hotter than it looks.", "I'm not burning out yet!", "Best. Bonfire. Ever."},
	"storm": {"Let's get moving.", "Can't stop, won't stop.", "Feel that static!",
		"Struck down. Literally.", "Now THAT is electric.", "Keep moving, keep moving...", "Look up. The stars are back."},
}

// heroComboQuips are the lines for long combos, so a hot streak gets a
// running commentary instead of one repeated line: three that take turns at
// a 50-hit combo, two for 100 and one for 150.
var heroComboQuips = map[string][6]string{
	"guardian": {"Nothing gets through!", "That's it, nice and steady!", "Formation's holding. Good.",
		"A hundred! Somebody write this down!", "Okay. I'm actually impressed.", "I'm telling everyone about this at dinner."},
	"berserker": {"MORE BRICKS!", "Ha! Is that all you've got?", "Keep 'em coming!",
		"I could do this ALL DAY!", "Best. Day. Of my LIFE!", "Somebody stop me. Actually, don't."},
	"arcane": {"Textbook! Literally, chapter four.", "Mind the angles, mind the angles...", "A most elegant chain reaction.",
		"One hundred! I must take notes.", "Remarkable. Utterly remarkable.", "I'm rewriting the textbook. Again."},
	"engineer": {"Running at peak efficiency!", "Huh. Working better than planned.", "Okay, okay, the numbers love this.",
		"A hundred? I'm logging that.", "This is going straight in my notebook.", "Nobody touch anything. It's perfect."},
	"lucky": {"Jackpot, baby!", "Hot streak! Don't jinx it.", "Let it ride!",
		"The house is SWEATING.", "I'd bet on me. I DID bet on me.", "Somebody call my bookie!"},
	"pyro": {"Burn, baby, burn!", "Ooh, it's catching!", "Hehe. More. MORE.",
		"It's so BEAUTIFUL!", "Everything's on fire and I love it!", "Best. Bonfire. Ever. Again!"},
	"storm": {"Feel that static!", "Faster! Faster!", "Whoo! Don't stop now!",
		"I can see the sky from up here!", "Now we're flying!", "So THIS is what lightning feels like!"},
}

// heroVoiceIndex picks the voice recipe for an origin.
var heroVoiceIndex = map[string]int{"guardian": 0, "berserker": 1, "arcane": 2, "engineer": 3, "lucky": 4, "pyro": 5, "storm": 6}

const (
	quipShow      = 3.6  // seconds a bubble stays up
	quipGap       = 8.0  // minimum seconds between any two quips
	quipComboCool = 45.0 // the 50-hit combo lines wait this long between uses
)

// quipState is the bubble on screen and the cooldowns.
type quipState struct {
	text      string
	col       color.RGBA
	t         float64 // seconds the bubble has been up; < 0 when hidden
	since     float64 // seconds since the last quip
	comboCool float64
	run       *Run // the run whose start line has been said
	said      [numQuipMoments]bool
	comboN    int        // turns through the combo lines
	pending   quipMoment // a moment to say when play resumes, plus one (0 = none)
	// the bubble's layout, worked out once per line (and language)
	lines    []string
	width    float64
	layoutOf string
}

// quipsOn reports whether heroes may speak now.
func (g *Game) quipsOn() bool {
	r := g.run
	if g.settings == nil || g.settings.NoQuips || r == nil || r.Duel {
		return false
	}
	switch r.SeedCode {
	case "MENU", "SIM", "TRAILER", "LIFE", "SAVEME":
		return false
	}
	return true
}

// syncQuips resets the quip state when a new run starts.
func (g *Game) syncQuips() *quipState {
	q := &g.quips
	if q.run != g.run {
		*q = quipState{run: g.run, since: quipGap, t: -1}
	}
	return q
}

// queueQuip holds a moment that happens off the play screen (a card taken
// on the reward screen) until the next chamber is under way.
func (g *Game) queueQuip(m quipMoment) {
	if g.quipsOn() {
		g.syncQuips().pending = m + 1
	}
}

// pickQuip greets the player when they choose a hero on the origin screen.
func (g *Game) pickQuip(arch string) {
	if g.settings == nil || g.settings.NoQuips {
		return
	}
	lines, ok := heroQuips[arch]
	if !ok {
		return
	}
	name := arch
	if hl, ok := heroLore[arch]; ok {
		name = hl.Name
	}
	g.toast(name, lines[quipPick], classInfo[arch].Col, icStar)
	g.heroVoice(arch, lines[quipPick])
}

// quip lets the current hero say their line for a moment, if the rules allow.
func (g *Game) quip(m quipMoment) {
	if !g.quipsOn() {
		return
	}
	q := g.syncQuips()
	lines, ok := heroQuips[g.run.Arch.ID]
	if !ok {
		return
	}
	switch m {
	case quipCombo:
		g.comboQuip(0)
		return
	case quipStart, quipWin:
		if q.said[m] {
			return
		}
	}
	// the last life and the win always get through; everything else waits
	// its turn so lines never crowd each other
	if q.since < quipGap && m != quipLastLife && m != quipWin {
		return
	}
	q.said[m] = true
	g.say(q, lines[m])
}

// comboQuip comments on a long combo: tier 0 at 50 hits, 1 at 100, 2 at
// 150. The 50-hit lines take turns and wait out a cooldown; the rarer
// milestones only wait for the last line to finish.
func (g *Game) comboQuip(tier int) {
	if !g.quipsOn() {
		return
	}
	q := g.syncQuips()
	lines, ok := heroComboQuips[g.run.Arch.ID]
	if !ok || q.since < quipGap || (tier == 0 && q.comboCool > 0) {
		return
	}
	var line string
	switch tier {
	case 0:
		line = lines[q.comboN%3]
	case 1:
		line = lines[3+q.comboN%2]
	default:
		line = lines[5]
	}
	q.comboN++
	q.comboCool = quipComboCool
	g.say(q, line)
}

// say puts a line in the bubble and plays the hero's voice.
func (g *Game) say(q *quipState, line string) {
	q.text = line
	q.col = classInfo[g.run.Arch.ID].Col
	q.t = 0
	q.since = 0
	g.heroVoice(g.run.Arch.ID, line)
}

// heroVoice plays a babble burst sized to the line.
func (g *Game) heroVoice(arch, line string) {
	vi, ok := heroVoiceIndex[arch]
	if !ok || g.snd == nil {
		return
	}
	n := min(4, max(1, len(strings.Fields(line))/2))
	g.snd.PlayVol("voice."+itoa(vi)+"."+itoa(n), 0.75)
}

// updateQuips ages the bubble and the cooldowns.
func (g *Game) updateQuips(dt float64) {
	q := &g.quips
	q.since += dt
	q.comboCool = math.Max(0, q.comboCool-dt)
	if q.t >= 0 {
		q.t += dt
		if q.t > quipShow {
			q.t = -1
		}
	}
}

// drawQuip draws the speech bubble below the vitals plate, with a trail of
// little thought-dots leading back to the portrait.
func (g *Game) drawQuip(dst *ebiten.Image) {
	q := &g.quips
	if q.t < 0 || q.text == "" || q.run != g.run {
		return
	}
	u := g.ui
	in := clamp01(q.t / 0.18)
	out := clamp01((quipShow - q.t) / 0.35)
	a := in * out
	pop := 1 + 0.08*math.Sin(math.Min(1, q.t/0.25)*math.Pi)
	const fs = 19.0
	if key := curLang + "|" + q.text; q.layoutOf != key {
		q.layoutOf = key
		q.lines = u.wrap(q.text, fBold, fs, 330)
		q.width = 0
		raw := u.rawText
		u.rawText = true
		for _, l := range q.lines {
			q.width = math.Max(q.width, u.measure(l, fBold, fs))
		}
		u.rawText = raw
	}
	lines := q.lines
	w := q.width + 36
	h := float64(len(lines))*fs*1.3 + 24
	x, y := 186.0, 100.0
	cx, cy := x+w/2, y+h/2
	bw, bh := w*pop, h*pop
	bx, by := cx-bw/2, cy-bh/2
	// thought-dots from the portrait to the bubble
	for i, d := range []float64{7, 10} {
		dx := 160.0 + float64(i)*12
		dy := 112.0 + float64(i)*2
		u.rrect(dst, dx-d/2, dy-d/2, d, d, d/2, ca(rgba(250, 246, 236, 255), 0.92*a))
	}
	u.rrect(dst, bx, by, bw, bh, 16, ca(rgba(250, 246, 236, 255), 0.95*a))
	u.rstroke(dst, bx, by, bw, bh, 16, 2.5, ca(darken(q.col, 0.15), a))
	ty := by + 12*pop
	raw := u.rawText
	u.rawText = true
	for _, l := range lines {
		u.text(dst, l, fBold, fs, cx, ty, ca(rgba(38, 26, 48, 255), a), alignCenter)
		ty += fs * 1.3
	}
	u.rawText = raw
}

// ---------------------------------------------------------------- voices

// voiceSpec is a hero's register: base pitch (Hz), how far the pitch wanders
// per syllable, syllable length, breathiness, the wave and vibrato rate.
type voiceSpec struct {
	pitch, wander, syl, breath float64
	wave                       func(float64) float64
	vib                        float64
}

var heroVoices = []voiceSpec{
	{235, 0.10, 0.085, 0.10, wTri, 5.5}, // Oriel: warm, steady
	{132, 0.16, 0.095, 0.18, wSaw, 7.5}, // Brakka: low and gruff
	{178, 0.22, 0.075, 0.06, wSq, 4.0},  // Vell: reedy, fussy rises
	{290, 0.08, 0.060, 0.04, wSq, 0},    // Rivet: crisp, clipped
	{196, 0.28, 0.105, 0.08, wSaw, 3.0}, // Sol: smooth, sliding swagger
	{330, 0.20, 0.065, 0.06, wTri, 6.5}, // Cinder: bright and bouncy
	{248, 0.14, 0.080, 0.30, wTri, 5.0}, // Zephyr: airy, breathy
}

// vowel formants (F1, F2) for a, e, i, o, u
var vowelFormants = [][2]float64{{800, 1200}, {500, 1900}, {320, 2400}, {500, 900}, {350, 750}}

// syllableSrc is one syllable's buzzing source. Each formant filter gets its
// own copy, since the filters must not share oscillator state.
func syllableSrc(hv voiceSpec, f0, d float64, r *rand.Rand) func(float64) float64 {
	ph := 0.0
	return func(tt float64) float64 {
		vib := 1 + 0.012*math.Sin(2*math.Pi*hv.vib*tt)
		ph += f0 * vib * (1 - 0.06*tt/d) / sampleRate
		env := math.Max(0, math.Min(1, tt/0.012)*math.Min(1, (d-tt)/0.02))
		return (hv.wave(ph) + hv.breath*(r.Float64()*2-1)) * env
	}
}

func init() {
	// voice.hero.words : a babble burst of a few syllables
	recipes["voice"] = recipe{3, func(p []int, v int, r *rand.Rand) []byte {
		hi := arg(p, 0, 0)
		if hi < 0 || hi >= len(heroVoices) {
			hi = 0
		}
		hv := heroVoices[hi]
		n := 2 + max(0, arg(p, 1, 2))
		total := float64(n)*hv.syl*1.25 + 0.3
		return one(total, r.Int63(), verbSmall.with(0.42), func(b *sb) {
			t := 0.0
			for i := 0; i < n; i++ {
				vw := vowelFormants[r.Intn(len(vowelFormants))]
				f0 := hv.pitch * (1 + (r.Float64()*2-1)*hv.wander)
				if i == n-1 {
					f0 *= 0.9 // a friendly little fall at the end of the line
				}
				d := hv.syl * (0.8 + 0.4*r.Float64())
				b.filtered(t, d, syllableSrc(hv, f0, d, r), fBP, cst(vw[0]), 3.2, 0.9)
				b.filtered(t, d, syllableSrc(hv, f0, d, r), fBP, cst(vw[1]), 4.5, 0.5)
				t += d * 1.18
			}
		})
	}}
}
