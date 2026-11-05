package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"orb/mus"
)

const sampleRate = 44100

type Sounds struct {
	ctx   *audio.Context
	SFX   float64
	Music float64

	cmu     sync.Mutex
	cache   map[string]*clipEntry // rendered clips, key "name#variant"
	warmQ   chan string           // clip names for the warming workers
	lastVar map[string]int
	last    map[string]time.Time
	vmu     sync.Mutex // guards voices (clips not yet rendered start from other goroutines)
	voices  []*audio.Player
	rng     *rand.Rand

	Key, World int // musical key (act) and current world, for musical SFX
	coinStep   int
	coinT      time.Time
	duck       float64
	duckT      time.Time
	duckLen    float64

	player, old    *audio.Player
	stream         *mus.Stream
	curTrack       int
	intensity      float64
	curVol, oldVol float64
	startT, oldT   time.Time
	punch          float64
	punchT         time.Time
}

func pcm(samples []float64) []byte {
	buf := make([]byte, len(samples)*4)
	for i, v := range samples {
		v = math.Max(-1, math.Min(1, v))
		s := uint16(int16(v * 32000))
		binary.LittleEndian.PutUint16(buf[i*4:], s)
		binary.LittleEndian.PutUint16(buf[i*4+2:], s)
	}
	return buf
}

func sine(f, t float64) float64 { return math.Sin(2 * math.Pi * f * t) }
func tri(f, t float64) float64 {
	p := math.Mod(f*t, 1)
	return 4*math.Abs(p-0.5) - 1
}
func sq(f, t float64) float64 {
	if math.Mod(f*t, 1) < 0.5 {
		return 1
	}
	return -1
}

func NewSounds() *Sounds {
	s := &Sounds{ctx: audio.NewContext(sampleRate), cache: map[string]*clipEntry{}, lastVar: map[string]int{}, last: map[string]time.Time{},
		SFX: 0.8, Music: 0.6, curTrack: -1, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	s.warmQ = make(chan string, 1024)
	for range max(1, min(4, runtime.NumCPU()-1)) {
		go func() {
			for n := range s.warmQ {
				base, _ := parseName(n)
				for v := 0; v < max(1, recipes[base].vars); v++ {
					s.clip(n, v)
				}
			}
		}()
	}
	s.warm(prewarmNames()...)
	return s
}

// sfxWarm enables warming clips ahead of use; the balance harnesses, which
// play thousands of silent levels, turn it off.
var sfxWarm = true

// clipEntry is one rendered clip; once guards its rendering.
type clipEntry struct {
	once sync.Once
	done atomic.Bool // data is set
	data []byte
}

// clip returns variant v of a named clip, rendering and caching it on first
// use. A clip being rendered elsewhere (a warming worker) is waited for,
// not rendered twice.
func (s *Sounds) clip(name string, v int) []byte {
	key := name + "#" + itoa(v)
	s.cmu.Lock()
	e := s.cache[key]
	if e == nil {
		e = &clipEntry{}
		s.cache[key] = e
	}
	s.cmu.Unlock()
	e.once.Do(func() {
		base, params := parseName(name)
		if rc, ok := recipes[base]; ok {
			e.data = rc.build(params, v, rand.New(rand.NewSource(hashStr(key))))
		}
		e.done.Store(true)
	})
	return e.data
}

// warm queues clips (every variant) for the warming workers, so they are
// rendered before they are first played instead of on the frame that
// plays them. It never blocks; a full queue drops the rest.
func (s *Sounds) warm(names ...string) {
	if !sfxWarm || s.SFX <= 0 {
		return
	}
	for _, n := range names {
		select {
		case s.warmQ <- n:
		default:
			return
		}
	}
}

// warmKeyed queues the musical brick sounds for the current world and key
// (the key follows the music, so each act and the boss have their own).
func (s *Sounds) warmKeyed() {
	w, k := itoa(s.World), itoa(s.Key)
	names := []string{"tuf." + w + "." + k, "deb." + w}
	for st := 0; st < 15; st++ {
		names = append(names, "brk."+w+"."+k+"."+itoa(st))
	}
	s.warm(names...)
}

// SetWorld sets the world the musical SFX play in.
func (s *Sounds) SetWorld(w int) {
	s.World = w
	s.warmKeyed()
}

// Play plays a named sound ("boom", "pick.3.1", ...) at full volume.
func (s *Sounds) Play(name string) { s.play(name, 1) }

// PlayVol plays a named sound at a relative volume.
func (s *Sounds) PlayVol(name string, vol float64) { s.play(name, vol) }

// PlayKeyed appends the current musical key to the name.
func (s *Sounds) PlayKeyed(name string) { s.play(name+"."+itoa(s.Key), 1) }

func (s *Sounds) play(name string, vol float64) {
	if s == nil || s.SFX <= 0 {
		return
	}
	base, _ := parseName(name)
	rc, ok := recipes[base]
	if !ok {
		return
	}
	now := nowFn()
	if t, ok := s.last[name]; ok && now.Sub(t) < 35*time.Millisecond {
		return
	}
	s.last[name] = now
	// voice management: drop finished players, cap the total
	s.vmu.Lock()
	live := s.voices[:0]
	for _, p := range s.voices {
		if p.IsPlaying() {
			live = append(live, p)
		}
	}
	s.voices = live
	nv := len(s.voices)
	s.vmu.Unlock()
	important := base != "brk" && base != "deb" && base != "wall" && base != "coin" && base != "tuf" && base != "thump"
	if nv >= 26 && !important {
		return
	}
	// pick a variant different from the last one
	v := 0
	if rc.vars > 1 {
		v = s.rng.Intn(rc.vars)
		if v == s.lastVar[name] {
			v = (v + 1) % rc.vars
		}
	}
	s.lastVar[name] = v
	if sfxHook != nil && sfxHook(name, v, vol) {
		return
	}
	if pk := punchFor[base]; pk > 0 {
		s.Punch(pk * vol)
	}
	gain := s.SFX * vol
	if !s.rendered(name, v) {
		// not ready (nor warmed in time): render it on another goroutine
		// and start it from there, a few milliseconds late, rather than
		// stall this frame
		go func() { s.start(s.clip(name, v), gain) }()
		return
	}
	s.start(s.clip(name, v), gain)
}

// start plays a rendered clip at the given volume.
func (s *Sounds) start(c []byte, gain float64) {
	if c == nil {
		return
	}
	p := s.ctx.NewPlayerFromBytes(c)
	p.SetVolume(gain)
	p.Play()
	s.vmu.Lock()
	s.voices = append(s.voices, p)
	s.vmu.Unlock()
}

// rendered reports whether a clip is ready to play.
func (s *Sounds) rendered(name string, v int) bool {
	s.cmu.Lock()
	e := s.cache[name+"#"+itoa(v)]
	s.cmu.Unlock()
	return e != nil && e.done.Load()
}

// punchFor is how far big impacts dip the music for an instant, so they
// land on top of it instead of fighting it.
var punchFor = map[string]float64{"boom": 0.35, "paddleboom": 0.3, "firestorm": 0.3, "thunder": 0.25,
	"shatter": 0.2, "bosshit": 0.15, "thump": 0.12, "crit": 0.1, "roar": 0.25}

// Punch dips the music by depth (0..1) for a beat-short moment.
func (s *Sounds) Punch(depth float64) {
	s.punch = math.Max(s.punch, math.Min(0.4, depth))
	s.updateVolumes()
}

// Duck lowers the music briefly under a big stinger.
func (s *Sounds) Duck(depth, secs float64) {
	s.duck, s.duckT, s.duckLen = depth, nowFn(), secs
	s.updateVolumes()
}

// ---------------------------------------------------------------- semantic helpers

// BrickBreak plays the world's instrument on the combo ladder, the
// material breaking, and weight for heavy bricks.
func (s *Sounds) BrickBreak(combo int, heavy bool) {
	st := min(14, combo)
	s.play("brk."+itoa(s.World)+"."+itoa(s.Key)+"."+itoa(st), 1)
	s.play("deb."+itoa(s.World), 0.8)
	if heavy {
		s.play("thump", 0.8)
	}
}

func (s *Sounds) BrickHit() { s.play("tuf."+itoa(s.World)+"."+itoa(s.Key), 0.9) }

// Coin climbs a scale while coins keep arriving in quick succession.
func (s *Sounds) Coin() {
	now := nowFn()
	if now.Sub(s.coinT) > 450*time.Millisecond {
		s.coinStep = 0
	} else {
		s.coinStep = min(11, s.coinStep+1)
	}
	s.coinT = now
	s.play("coin."+itoa(s.coinStep), 0.8)
}

// ComboMilestone plays a stinger whose size grows with the combo.
func (s *Sounds) ComboMilestone(combo int) {
	tier := 1
	switch {
	case combo >= 100:
		tier = 4
	case combo >= 50:
		tier = 3
	case combo >= 25:
		tier = 2
	}
	s.play("cmb."+itoa(tier)+"."+itoa(s.Key), 1)
	if tier >= 3 {
		s.Duck(0.5, 1.5)
	}
}

// Pick plays the reward sound for a rarity (0 common .. 5 mythic).
func (s *Sounds) Pick(rarity int) {
	s.play("pick."+itoa(rarity)+"."+itoa(s.Key), 1)
	if rarity >= 3 {
		s.Duck(0.55, 1.2+0.4*float64(rarity))
	}
}

// ---------------------------------------------------------------- music

// The soundtrack streams from the music engine (package mus): one song per
// act, the menu theme and the boss theme, arranged live from the intensity.

// trackSongs maps track numbers (0..2 acts, 3 menu, 4 boss) to songs.
var trackSongs = []string{"rush", "acid", "abyss", "groove", "warden"}

const trackBoss = 4

// SetTrack switches music (3 = menu, 0..2 = acts, 4 = boss), crossfading
// from the old song. Safe to call each frame.
func (s *Sounds) SetTrack(i int) {
	if s.curTrack == i {
		s.updateVolumes()
		return
	}
	s.curTrack = i
	s.Key = i
	s.warmKeyed()
	if s.old != nil {
		s.old.Close()
	}
	s.old, s.oldVol, s.oldT = s.player, s.curVol, nowFn()
	s.player, s.stream = nil, nil
	s.startStream()
	s.updateVolumes()
}

// startStream opens the current track's song (only while music is audible:
// a muted soundtrack costs nothing).
func (s *Sounds) startStream() {
	if s.player != nil || s.curTrack < 0 || s.Music <= 0 {
		return
	}
	st := mus.NewStream(trackSongs[s.curTrack%len(trackSongs)])
	if st == nil {
		return
	}
	st.SetIntensity(s.intensity)
	p, err := s.ctx.NewPlayer(st)
	if err != nil {
		return
	}
	p.SetBufferSize(150 * time.Millisecond)
	s.stream, s.player, s.startT = st, p, nowFn()
	s.updateVolumes()
	p.Play()
}

func (s *Sounds) SetIntensity(v float64) {
	s.intensity += (v - s.intensity) * 0.03
	if s.stream != nil {
		s.stream.SetIntensity(s.intensity)
	}
	if s.duck > 0 {
		if el := nowFn().Sub(s.duckT).Seconds(); el > s.duckLen {
			s.duck = math.Max(0, s.duck-0.02)
		}
	}
	s.updateVolumes()
}

func (s *Sounds) updateVolumes() {
	if s.Music <= 0 {
		for _, p := range []*audio.Player{s.player, s.old} {
			if p != nil {
				p.Close()
			}
		}
		s.player, s.stream, s.old = nil, nil, nil
		return
	}
	s.startStream()
	now := nowFn()
	if s.old != nil {
		u := now.Sub(s.oldT).Seconds() / 0.9
		if u >= 1 {
			s.old.Close()
			s.old = nil
		} else {
			s.old.SetVolume(s.oldVol * (1 - u))
		}
	}
	if s.player == nil {
		return
	}
	if s.punch > 0 {
		dt := math.Min(0.1, now.Sub(s.punchT).Seconds())
		s.punch = math.Max(0, s.punch-dt*2.2)
	}
	s.punchT = now
	in := math.Min(1, now.Sub(s.startT).Seconds()/0.6)
	s.curVol = s.Music * 0.5 * (1 - s.duck) * (1 - s.punch) * in // songs master near -12.5 dBFS RMS; this sits them where the old loops were, +1.5 dB
	s.player.SetVolume(s.curVol)
}
