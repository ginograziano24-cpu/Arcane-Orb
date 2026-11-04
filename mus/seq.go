package mus

import (
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
)

// The music engine's sequencer: songs are sections of bar patterns played
// by parts (an instrument plus its mix settings). The song picks its next
// section from the game's intensity, and parts that need more intensity
// wait until it's there, so the arrangement builds with the action. The
// player renders straight into the audio stream, block by block.

type ev struct {
	at, dur float64 // steps (sixteenths) from the bar start
	key     float64 // MIDI note (drums: 0 closed, 1 open, or a tom pitch)
	vel     float64
	slide   bool // glide from the previous note (mono parts)
}

type bar []ev

type partDef struct {
	name                      string
	in                        inst
	gain, pan, rev, dly, pump float64
	gate                      float64 // intensity the part needs to play
	mono, choke, trigger      bool    // trigger: drives the sidechain pump
}

type secDef struct {
	bars  int
	pat   map[string][]bar      // per part, cycled over the section's bars
	sweep map[string][2]float64 // lowpass sweep per part ("*" = master), Hz
}

type songDef struct {
	name               string
	bpm, swing, human  float64
	parts              []*partDef
	secs               map[string]*secDef
	first              string
	next               func(sp *songPlayer) string
	room, damp         float64
	dlySteps, dlyFb    float64
	gain               float64
	pumpDepth, pumpRel float64
}

type partRT struct {
	def    *partDef
	voices []*voice
	last   *voice
	on     bool
	lp     [2]svf
	lpOn   bool
	l, r   []float64
}

type pendEv struct {
	start int64
	p     *partRT
	e     ev
	durS  int
}

type songPlayer struct {
	def      *songDef
	parts    []*partRT
	byName   map[string]*partRT
	stepLen  float64
	barLen   int64
	pos      int64
	nextBar  int64
	sec      string
	secBar   int
	secStart int64
	plays    map[string]int // times each section has started
	pend     []pendEv
	inten    atomic.Uint64
	lvl      float64 // intensity sampled at the bar start
	rev      *reverb
	dly      *pingpong
	comp     *busComp
	pumps    []int64
	lastPump int64
	mlp      [2]svf
	mlpOn    bool
	rng      *rand.Rand
	sl, sr   []float64 // send buses
	ml, mr   []float64
	duck     []float64
	fade     float64
}

const musBlock = 256

// Solo, when set, plays only the named part (analysis).
var Solo string

func newSongPlayer(def *songDef) *songPlayer {
	sp := &songPlayer{def: def, byName: map[string]*partRT{}, plays: map[string]int{},
		rng: rand.New(rand.NewSource(int64(len(def.name)) * 977))}
	sp.stepLen = msr * 60 / def.bpm / 4
	sp.barLen = int64(math.Round(sp.stepLen * 16))
	for _, pd := range def.parts {
		p := &partRT{def: pd, l: make([]float64, musBlock), r: make([]float64, musBlock)}
		sp.parts = append(sp.parts, p)
		sp.byName[pd.name] = p
	}
	sp.rev = newReverb(def.room, def.damp)
	sp.dly = newPingPong(def.dlySteps*sp.stepLen/msr, def.dlyFb)
	sp.comp = newBusComp()
	sp.sl, sp.sr = make([]float64, musBlock), make([]float64, musBlock)
	sp.ml, sp.mr = make([]float64, musBlock), make([]float64, musBlock)
	sp.duck = make([]float64, musBlock)
	sp.secBar = -1
	sp.lastPump = -1 << 40
	return sp
}

func (sp *songPlayer) setIntensity(v float64) { sp.inten.Store(math.Float64bits(v)) }
func (sp *songPlayer) intensity() float64     { return math.Float64frombits(sp.inten.Load()) }

// startBar advances to the next bar, moving on to a new section when the
// current one ends, and queues the bar's notes.
func (sp *songPlayer) startBar(at int64) {
	sp.lvl = sp.intensity()
	sp.secBar++
	if sp.sec == "" || sp.secBar >= sp.def.secs[sp.sec].bars {
		next := sp.def.first
		if sp.sec != "" {
			next = sp.def.next(sp)
		}
		if sp.def.secs[next] == nil {
			next = sp.def.first
		}
		sp.sec, sp.secBar, sp.secStart = next, 0, at
		sp.plays[next]++
	}
	sec := sp.def.secs[sp.sec]
	for _, p := range sp.parts {
		pats := sec.pat[p.def.name]
		g := p.def.gate
		if p.on {
			g -= 0.08 // hysteresis
		}
		p.on = len(pats) > 0 && sp.lvl >= g
		if !p.on {
			continue
		}
		b := pats[sp.secBar%len(pats)]
		for _, e := range b {
			off := e.at * sp.stepLen
			if fi := math.Floor(e.at); e.at == fi && int(fi)%2 == 1 {
				off += sp.def.swing * sp.stepLen
			}
			if sp.def.human > 0 {
				off += (sp.rng.Float64()*2 - 1) * sp.def.human * msr / 1000
			}
			e2 := e
			if sp.def.human > 0 {
				e2.vel *= 1 + (sp.rng.Float64()*2-1)*0.06
			}
			sp.pend = append(sp.pend, pendEv{start: at + int64(math.Max(0, off)), p: p, e: e2,
				durS: max(1, int(e.dur*sp.stepLen))})
		}
	}
	sort.SliceStable(sp.pend, func(i, j int) bool { return sp.pend[i].start < sp.pend[j].start })
}

// barsLeft is how many bars remain in the current section.
func (sp *songPlayer) barsLeft() int { return sp.def.secs[sp.sec].bars - sp.secBar - 1 }

func (sp *songPlayer) trigger(pe pendEv) {
	p := pe.p
	in := &p.def.in
	e := pe.e
	if p.def.trigger {
		sp.pumps = append(sp.pumps, pe.start)
	}
	if p.def.mono && p.last != nil && !p.last.dead && e.slide {
		p.last.legato(e.key, e.vel, pe.durS, true)
		return
	}
	if (p.def.mono || p.def.choke) && p.last != nil && !p.last.dead {
		p.last.choke()
	}
	if len(p.voices) > 24 { // runaway safety: retire the oldest
		p.voices[0].choke()
	}
	ins := in
	if in.kind == iHat && e.key >= 1 {
		open := *in
		open.decay *= 6.5
		ins = &open
	}
	v := newVoice(ins, e.key, e.vel, pe.durS)
	v.delay = int(pe.start - sp.pos)
	p.voices = append(p.voices, v)
	p.last = v
}

// render fills l and r with n frames.
func (sp *songPlayer) render(l, r []float64, n int) {
	end := sp.pos + int64(n)
	for sp.nextBar < end {
		sp.startBar(sp.nextBar)
		sp.nextBar += sp.barLen
	}
	k := 0
	for k < len(sp.pend) && sp.pend[k].start < end {
		pe := sp.pend[k]
		if pe.start < sp.pos {
			pe.start = sp.pos
		}
		sp.trigger(pe)
		k++
	}
	sp.pend = sp.pend[:copy(sp.pend, sp.pend[k:])]

	sec := sp.def.secs[sp.sec]
	prog := float64(sp.pos-sp.secStart) / float64(int64(sec.bars)*sp.barLen)
	for i := 0; i < n; i++ {
		sp.sl[i], sp.sr[i], sp.ml[i], sp.mr[i] = 0, 0, 0, 0
	}
	// the sidechain pump, per sample
	depth := sp.def.pumpDepth
	duck := sp.duck[:n]
	pi := 0
	for i := 0; i < n; i++ {
		at := sp.pos + int64(i)
		for pi < len(sp.pumps) && sp.pumps[pi] <= at {
			sp.lastPump = sp.pumps[pi]
			pi++
		}
		t := float64(at-sp.lastPump) / msr
		duck[i] = depth * math.Min(1, t/0.004) * fexp(-t/math.Max(0.01, sp.def.pumpRel))
	}
	sp.pumps = sp.pumps[:copy(sp.pumps, sp.pumps[pi:])]

	for _, p := range sp.parts {
		if len(p.voices) == 0 {
			continue
		}
		pl, pr := p.l[:n], p.r[:n]
		for i := range pl {
			pl[i], pr[i] = 0, 0
		}
		live := p.voices[:0]
		for _, v := range p.voices {
			if v.render(pl, pr, n) {
				live = append(live, v)
			}
		}
		for i := len(live); i < len(p.voices); i++ {
			p.voices[i] = nil
		}
		p.voices = live
		if sw, ok := sec.sweep[p.def.name]; ok {
			if !p.lpOn {
				p.lp = [2]svf{}
				p.lpOn = true
			}
			c := sw[0] * math.Pow(sw[1]/sw[0], math.Min(1, prog))
			p.lp[0].set(c, 0.9)
			p.lp[1].set(c, 0.9)
			for i := range pl {
				pl[i] = p.lp[0].lp(pl[i])
				pr[i] = p.lp[1].lp(pr[i])
			}
		} else {
			p.lpOn = false
		}
		d := p.def
		if Solo != "" && d.name != Solo {
			continue
		}
		gl := d.gain * math.Min(1, 1-d.pan)
		gr := d.gain * math.Min(1, 1+d.pan)
		for i := 0; i < n; i++ {
			g := 1 - d.pump*duck[i]
			a, b := pl[i]*gl*g, pr[i]*gr*g
			sp.ml[i] += a
			sp.mr[i] += b
			m := (a + b) * 0.5
			sp.sl[i] += m * d.rev
			sp.sr[i] += m * d.dly
		}
	}
	sw, msw := sec.sweep["*"]
	if msw && !sp.mlpOn {
		sp.mlp = [2]svf{}
	}
	sp.mlpOn = msw
	if msw {
		c := sw[0] * math.Pow(sw[1]/sw[0], math.Min(1, prog))
		sp.mlp[0].set(c, 0.8)
		sp.mlp[1].set(c, 0.8)
	}
	g := sp.def.gain
	for i := 0; i < n; i++ {
		rl, rr := sp.rev.proc(sp.sl[i])
		dl, dr := sp.dly.proc(sp.sr[i])
		a := sp.ml[i] + rl*3 + dl
		b := sp.mr[i] + rr*3 + dr
		if msw {
			a, b = sp.mlp[0].lp(a), sp.mlp[1].lp(b)
		}
		a, b = sp.comp.proc(a*g, b*g)
		if sp.fade < 1 {
			sp.fade = math.Min(1, sp.fade+1/(0.05*msr))
		}
		l[i], r[i] = a*sp.fade, b*sp.fade
	}
	sp.pos = end
}

// Stream plays a song as a 16-bit little-endian stereo PCM stream.
type Stream struct {
	sp   *songPlayer
	l, r []float64
}

// NewStream starts the named song (nil if there is no such song).
func NewStream(name string) *Stream {
	def := songs[name]
	if def == nil {
		return nil
	}
	return &Stream{sp: newSongPlayer(def), l: make([]float64, musBlock), r: make([]float64, musBlock)}
}

// SetIntensity sets how hard the music should play, 0..1. Safe to call
// from any goroutine; it takes effect at the next bar.
func (m *Stream) SetIntensity(v float64) { m.sp.setIntensity(v) }

// Section is the section playing now (for debugging).
func (m *Stream) Section() string { return m.sp.sec }

func (m *Stream) Read(p []byte) (int, error) {
	frames := len(p) / 4
	done := 0
	for done < frames {
		n := min(musBlock, frames-done)
		m.sp.render(m.l, m.r, n)
		for i := 0; i < n; i++ {
			o := (done + i) * 4
			binary.LittleEndian.PutUint16(p[o:], uint16(int16(clamp16(m.l[i]))))
			binary.LittleEndian.PutUint16(p[o+2:], uint16(int16(clamp16(m.r[i]))))
		}
		done += n
	}
	return frames * 4, nil
}

func clamp16(v float64) float64 {
	return math.Max(-32767, math.Min(32767, v*32767))
}

// Render renders secs of the named song offline (previews and analysis),
// with the intensity following inten(t).
func Render(name string, secs float64, inten func(t float64) float64) (l, r []float64) {
	sp := newSongPlayer(songs[name])
	sp.fade = 1
	total := int(secs * msr)
	l, r = make([]float64, total), make([]float64, total)
	for at := 0; at < total; at += musBlock {
		n := min(musBlock, total-at)
		sp.setIntensity(inten(float64(at) / msr))
		sp.render(l[at:at+n], r[at:at+n], n)
	}
	return
}
