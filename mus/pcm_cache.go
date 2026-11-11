package mus

import (
	"encoding/binary"
	"errors"
	"io"
)

const cacheHeaderSize = 32
const cacheMagic = "AOPCM001"

// PCMCache is immutable pre-rendered stereo16 audio. The backing bank must
// remain unchanged. It contains an introduction, a loop and a continuation
// used to crossfade into the loop without restarting on a discontinuity.
type PCMCache struct {
	pcm                         []byte
	loopStart, loopEnd, overlap int
}

func ParsePCMCache(bank []byte) (*PCMCache, error) {
	if len(bank) < cacheHeaderSize || string(bank[:8]) != cacheMagic {
		return nil, errors.New("invalid music cache header")
	}
	u := func(at int) uint64 { return uint64(binary.LittleEndian.Uint32(bank[at:])) }
	start, end, overlap, frames := u(12), u(16), u(20), u(24)
	if u(8) != SampleRate || u(28) != 4 || start >= end || overlap < 2 || overlap > end-start || end+overlap != frames || frames*4 != uint64(len(bank)-cacheHeaderSize) {
		return nil, errors.New("invalid music cache format or loop bounds")
	}
	return &PCMCache{pcm: bank[cacheHeaderSize:], loopStart: int(start), loopEnd: int(end), overlap: int(overlap)}, nil
}

// NewStream has its own playhead; multiple players share the immutable audio.
func (c *PCMCache) NewStream() *PCMStream { return &PCMStream{cache: c, blend: -1, pendingAt: 4} }

type PCMStream struct {
	cache     *PCMCache
	frame     int
	blend     int
	pending   [4]byte
	pendingAt int
}

func (p *PCMStream) nextFrame() {
	c := p.cache
	if p.frame == c.loopEnd {
		p.frame = c.loopStart
		p.blend = 0
	}
	copy(p.pending[:], c.pcm[p.frame*4:p.frame*4+4])
	if p.blend >= 0 {
		// The continuation is the same musical beat after the recorded loop.
		// Its first sample follows the previous frame exactly. Fade toward the
		// cached loop over100ms, retaining the loop's full musical duration.
		for ch := 0; ch < 2; ch++ {
			at := ch * 2
			a := int64(int16(binary.LittleEndian.Uint16(c.pcm[(c.loopEnd+p.blend)*4+at:])))
			b := int64(int16(binary.LittleEndian.Uint16(p.pending[at:])))
			den := int64(max(1, c.overlap-1))
			num := int64(p.blend)
			v := (a*(den-num) + b*num) / den
			binary.LittleEndian.PutUint16(p.pending[at:], uint16(int16(v)))
		}
		p.blend++
		if p.blend == c.overlap {
			p.blend = -1
		}
	}
	p.frame++
	p.pendingAt = 0
}

func (p *PCMStream) Read(dst []byte) (int, error) {
	written := 0
	for written < len(dst) {
		if p.pendingAt < 4 {
			n := copy(dst[written:], p.pending[p.pendingAt:])
			p.pendingAt += n
			written += n
			continue
		}
		// Bulk-copy regular PCM; only the short loop join needs sample math.
		if p.frame < p.cache.loopEnd && p.blend < 0 && len(dst)-written >= 4 {
			n := min((len(dst)-written)/4, p.cache.loopEnd-p.frame) * 4
			copy(dst[written:written+n], p.cache.pcm[p.frame*4:p.frame*4+n])
			p.frame += n / 4
			written += n
			continue
		}
		p.nextFrame()
	}
	return written, nil
}

// WriteMenuCache renders at build time, never on the game's update/audio thread.
// The first pass is the original synth PCM, including its opening and effects.
func WriteMenuCache(w io.Writer) error {
	def := songs["groove"]
	sp := newSongPlayer(def)
	loopBars := 0
	for _, name := range grooveOrder {
		loopBars += def.secs[name].bars
	}
	start := int(sp.barLen) * def.secs[def.first].bars
	end := start + int(sp.barLen)*loopBars
	overlap := SampleRate / 10
	frames := end + overlap
	header := make([]byte, cacheHeaderSize)
	copy(header, cacheMagic)
	for i, v := range []uint32{SampleRate, uint32(start), uint32(end), uint32(overlap), uint32(frames), 4} {
		binary.LittleEndian.PutUint32(header[8+i*4:], v)
	}
	if err := writeCacheBytes(w, header); err != nil {
		return err
	}
	s := NewStream("groove")
	s.SetIntensity(.25)
	buf := make([]byte, musBlock*4)
	for at := 0; at < frames; {
		n := min(musBlock, frames-at) * 4
		if _, err := s.Read(buf[:n]); err != nil {
			return err
		}
		if err := writeCacheBytes(w, buf[:n]); err != nil {
			return err
		}
		at += n / 4
	}
	return nil
}

func writeCacheBytes(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}
