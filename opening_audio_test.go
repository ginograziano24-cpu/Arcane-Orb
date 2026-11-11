package main

import (
	"encoding/binary"
	"io"
	"testing"
)

func openingTestPCM(values ...int16) []byte {
	b := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(v))
	}
	return b
}
func TestOpeningMixerSyncGainAndSeek(t *testing.T) {
	m := newOpeningMixer([3][]byte{openingTestPCM(1000, -1000, 2000, -2000), openingTestPCM(200, -200, 400, -400), openingTestPCM(100, -100, 300, -300)})
	m.setGains(.5, 1, 0)
	b := make([]byte, 4)
	n, e := m.Read(b)
	if n != 4 || e != nil || int16(binary.LittleEndian.Uint16(b)) != 700 || int16(binary.LittleEndian.Uint16(b[2:])) != -700 {
		t.Fatalf("bad synchronized mix %v %d %v", b, n, e)
	}
	m.setGains(0, 0, 1)
	_, _ = m.Read(b)
	if int16(binary.LittleEndian.Uint16(b)) != 300 {
		t.Fatal("voice control did not preserve stem position")
	}
	if _, e = m.Read(b); e != io.EOF {
		t.Fatal("missing end of stream")
	}
	pos, e := m.Seek(1, io.SeekStart)
	if pos != 0 || e != nil {
		t.Fatal("seek not frame aligned")
	}
	_, _ = m.Read(b)
	if int16(binary.LittleEndian.Uint16(b)) != 100 {
		t.Fatal("seek failed to rewind all stems together")
	}
	pos, _ = m.Seek(-100, io.SeekStart)
	if pos != 0 {
		t.Fatal("negative seek")
	}
	pos, _ = m.Seek(100, io.SeekEnd)
	if pos != m.length {
		t.Fatal("seek past end")
	}
	if _, e = m.Seek(0, 7); e == nil {
		t.Fatal("invalid seek accepted")
	}
}
func TestOpeningMixerSaturatesAndPads(t *testing.T) {
	m := newOpeningMixer([3][]byte{openingTestPCM(30000, -30000, 100, -100), openingTestPCM(30000, -30000), nil})
	b := make([]byte, 8)
	_, _ = m.Read(b)
	if int16(binary.LittleEndian.Uint16(b)) != 32767 || int16(binary.LittleEndian.Uint16(b[2:])) != -32768 || int16(binary.LittleEndian.Uint16(b[4:])) != 100 {
		t.Fatal("clipping wraparound or short-stem padding")
	}
}
func TestOpeningStoryAssetsAndSubtitles(t *testing.T) {
	r := loadOpening()
	if r.err != nil {
		t.Fatal(r.err)
	}
	if len(r.images) != 19 || len(r.lines) != 20 {
		t.Fatalf("incomplete opening: %d images, %d subtitles", len(r.images), len(r.lines))
	}
	for i, s := range r.stems {
		seconds := float64(len(s)) / float64(sampleRate*4)
		if seconds < 81.4 || seconds > 81.6 {
			t.Fatalf("stem %d duration %.3f", i, seconds)
		}
	}
	last := 0.0
	for _, l := range r.lines {
		if l.Time < last || l.End <= l.Time || l.End > openingDuration || l.Text == "" {
			t.Fatal("invalid subtitle timing")
		}
		last = l.End
	}
	for _, check := range []struct {
		t    float64
		name string
	}{{0, "tower"}, {14, "scroll"}, {33, "arrival"}, {38, "conjure"}, {43, "send"}, {55.5, "hero"}, {69.5, "gather"}, {76, "title"}} {
		name, _ := openingShot(check.t)
		if name != check.name {
			t.Fatalf("wrong story beat at %.1f: %s", check.t, name)
		}
	}
}
