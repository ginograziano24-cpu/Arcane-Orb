package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

type Sounds struct {
	ctx   *audio.Context
	SFX   float64
	Music float64

	cmu     sync.Mutex
	cache   map[string][]byte // rendered clips, key "name#variant"
	lastVar map[string]int
	last    map[string]time.Time
	voices  []*audio.Player
	rng     *rand.Rand

	Key, World int // musical key (act) and current world, for musical SFX
	coinStep   int
	coinT      time.Time
	duck       float64
	duckT      time.Time
	duckLen    float64

	mu        sync.Mutex
	tracks    map[int][2][]byte
	base, hot *audio.Player
	curTrack  int
	intensity float64
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
	s := &Sounds{ctx: audio.NewContext(sampleRate), cache: map[string][]byte{}, lastVar: map[string]int{}, last: map[string]time.Time{},
		SFX: 0.8, Music: 0.6, tracks: map[int][2][]byte{}, curTrack: -1, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	go func() {
		for _, n := range prewarmNames() {
			s.clip(n, 0)
		}
	}()
	go s.buildMusic()
	return s
}

// clip returns variant v of a named clip, rendering and caching it on first use.
func (s *Sounds) clip(name string, v int) []byte {
	key := name + "#" + itoa(v)
	s.cmu.Lock()
	c, ok := s.cache[key]
	s.cmu.Unlock()
	if ok {
		return c
	}
	base, params := parseName(name)
	rc, ok := recipes[base]
	if !ok {
		return nil
	}
	seed := hashStr(key)
	c = rc.build(params, v, rand.New(rand.NewSource(seed)))
	s.cmu.Lock()
	s.cache[key] = c
	s.cmu.Unlock()
	return c
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
	now := time.Now()
	if t, ok := s.last[name]; ok && now.Sub(t) < 35*time.Millisecond {
		return
	}
	s.last[name] = now
	// voice management: drop finished players, cap the total
	live := s.voices[:0]
	for _, p := range s.voices {
		if p.IsPlaying() {
			live = append(live, p)
		}
	}
	s.voices = live
	important := base != "brk" && base != "deb" && base != "wall" && base != "coin" && base != "tuf" && base != "thump"
	if len(s.voices) >= 26 && !important {
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
	c := s.clip(name, v)
	if c == nil {
		return
	}
	p := s.ctx.NewPlayerFromBytes(c)
	p.SetVolume(s.SFX * vol)
	p.Play()
	s.voices = append(s.voices, p)
}

// Duck lowers the music briefly under a big stinger.
func (s *Sounds) Duck(depth, secs float64) {
	s.duck, s.duckT, s.duckLen = depth, time.Now(), secs
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
	now := time.Now()
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

var biomeChords = [][][3]int{
	{{0, 3, 7}, {-4, 0, 3}, {-9, -5, -2}, {-2, 2, 5}}, // i VI III VII
	{{0, 3, 7}, {-2, 2, 5}, {-4, 0, 3}, {-5, -1, 2}},  // i VII VI V
	{{0, 3, 7}, {-7, -4, 0}, {-4, 0, 3}, {-2, 2, 5}},  // i iv VI VII
	{{0, 4, 7}, {-5, -1, 2}, {-3, 0, 4}, {-7, -3, 0}}, // menu: I V vi IV
}

func note(root float64, semi int) float64 { return root * math.Pow(2, float64(semi)/12) }

func (s *Sounds) renderTrack(idx int, root float64) [2][]byte {
	const bpm = 104.0
	beat := 60 / bpm
	bars := 8
	n := int(float64(bars*4) * beat * sampleRate)
	base := make([]float64, n)
	hot := make([]float64, n)
	rng := rand.New(rand.NewSource(int64(idx) + 11))
	chords := biomeChords[idx%len(biomeChords)]
	bell := []int{0, 2, 1, 2, 0, 1, 2, 1}
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		bt := t / beat
		bar := int(bt / 4)
		ch := chords[(bar/2)%4]
		inBar := math.Mod(bt, 4)
		// pad
		v := 0.0
		for k, semi := range ch {
			f := note(root, semi)
			det := 1 + 0.003*float64(k-1)
			v += 0.06 * (sine(f*det, t) + 0.4*sine(f*2*det, t) + 0.15*tri(f*0.5, t))
		}
		swell := 0.75 + 0.25*math.Sin(2*math.Pi*bt/8)
		v *= swell
		// bass on eighths
		e := math.Mod(bt, 0.5)
		bf := note(root/2, ch[0])
		v += 0.16 * (tri(bf, t) + 0.3*sine(bf*2, t)) * math.Exp(-e*5)
		// soft bells every beat
		bi := int(bt) % len(bell)
		bb := math.Mod(bt, 1)
		v += 0.05 * sine(note(root*2, ch[bell[bi]]), t) * math.Exp(-bb*4)
		base[i] = v
		// intensity layer: 16th arp + drums
		s16 := math.Mod(bt, 0.25)
		step := int(bt*4) % 8
		af := note(root*2, ch[step%3]+12*(step/3%2))
		h := 0.07 * (sq(af, t)*0.35 + sine(af, t)) * math.Exp(-s16*14)
		kb := math.Mod(inBar, 1)
		h += 0.35 * sine(55*math.Exp(-kb*18)+45, t) * math.Exp(-kb*9)
		hb := math.Mod(bt+0.5, 1)
		h += 0.05 * (rng.Float64()*2 - 1) * math.Exp(-hb*40)
		if math.Abs(inBar-1) < 0.5 || math.Abs(inBar-3) < 0.5 {
			sb := math.Mod(inBar, 2) - 1
			if sb >= 0 {
				h += 0.12 * (rng.Float64()*2 - 1) * math.Exp(-sb*14)
			}
		}
		hot[i] = h
	}
	return [2][]byte{pcm(base), pcm(hot)}
}

func (s *Sounds) buildMusic() {
	roots := []float64{110, 98, 123.5, 130.8}
	for _, i := range []int{3, 0, 1, 2} {
		tr := s.renderTrack(i, roots[i])
		s.mu.Lock()
		s.tracks[i] = tr
		s.mu.Unlock()
	}
}

// SetTrack switches music (3 = menu, 0..2 = acts). Safe to call each frame.
func (s *Sounds) SetTrack(i int) {
	s.mu.Lock()
	tr, ok := s.tracks[i]
	s.mu.Unlock()
	if !ok || s.curTrack == i {
		s.updateVolumes()
		return
	}
	if s.base != nil {
		s.base.Close()
		s.hot.Close()
	}
	s.curTrack = i
	s.Key = i
	l1 := audio.NewInfiniteLoop(bytes.NewReader(tr[0]), int64(len(tr[0])))
	l2 := audio.NewInfiniteLoop(bytes.NewReader(tr[1]), int64(len(tr[1])))
	s.base, _ = s.ctx.NewPlayer(l1)
	s.hot, _ = s.ctx.NewPlayer(l2)
	s.updateVolumes()
	s.base.Play()
	s.hot.Play()
}

func (s *Sounds) SetIntensity(v float64) {
	s.intensity += (v - s.intensity) * 0.03
	if s.duck > 0 {
		if el := time.Since(s.duckT).Seconds(); el > s.duckLen {
			s.duck = math.Max(0, s.duck-0.02)
		}
	}
	s.updateVolumes()
}

func (s *Sounds) updateVolumes() {
	if s.base == nil {
		return
	}
	d := 1 - s.duck
	s.base.SetVolume(s.Music * 0.8 * d)
	s.hot.SetVolume(s.Music * (0.15 + 0.85*s.intensity) * d)
}
