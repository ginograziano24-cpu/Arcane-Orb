package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
)

// One interleaved PCM stream keeps all three stems on exactly the same sample.
// The audio thread and UI only share this small mixer, protected by its mutex.
type openingMixer struct {
	mu          sync.Mutex
	stems       [3][]byte
	pos, length int64
	gain        [3]float64
}

func newOpeningMixer(stems [3][]byte) *openingMixer {
	m := &openingMixer{stems: stems, gain: [3]float64{1, 1, 1}}
	for _, s := range stems {
		if int64(len(s)) > m.length {
			m.length = int64(len(s))
		}
	}
	m.length -= m.length % 4
	return m
}
func (m *openingMixer) setGains(music, effects, voice float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, v := range []float64{music, effects, voice} {
		m.gain[i] = math.Max(0, math.Min(1, v))
	}
}
func (m *openingMixer) gains() [3]float64 { m.mu.Lock(); defer m.mu.Unlock(); return m.gain }
func (m *openingMixer) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pos >= m.length {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), m.length-m.pos))
	n -= n % 4
	if n == 0 {
		return 0, io.ErrShortBuffer
	}
	for i := 0; i < n; i += 2 {
		sum := 0.0
		for j, stem := range m.stems {
			at := int(m.pos) + i
			if at+1 < len(stem) {
				sum += float64(int16(binary.LittleEndian.Uint16(stem[at:]))) * m.gain[j]
			}
		}
		sum = math.Max(-32768, math.Min(32767, math.Round(sum)))
		binary.LittleEndian.PutUint16(p[i:], uint16(int16(sum)))
	}
	m.pos += int64(n)
	return n, nil
}
func (m *openingMixer) Seek(offset int64, whence int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += m.pos
	case io.SeekEnd:
		offset += m.length
	default:
		return m.pos, fmt.Errorf("invalid seek origin %d", whence)
	}
	offset = max(0, min(m.length, offset))
	offset -= offset % 4
	m.pos = offset
	return offset, nil
}
