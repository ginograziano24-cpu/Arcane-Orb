package mus

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
)

func smallCacheBank() []byte {
	b := make([]byte, cacheHeaderSize+10*4)
	copy(b, cacheMagic)
	for i, v := range []uint32{SampleRate, 2, 8, 2, 10, 4} {
		binary.LittleEndian.PutUint32(b[8+i*4:], v)
	}
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint16(b[cacheHeaderSize+i*4:], uint16(i*100))
		binary.LittleEndian.PutUint16(b[cacheHeaderSize+i*4+2:], uint16(int16(-i*100)))
	}
	return b
}

func TestCachedMusicLoopAndFragmentedReads(t *testing.T) {
	c, err := ParsePCMCache(smallCacheBank())
	if err != nil {
		t.Fatal(err)
	}
	whole := make([]byte, 200)
	p := c.NewStream()
	if n, e := p.Read(whole); n != len(whole) || e != nil {
		t.Fatal(n, e)
	}
	if !bytes.Equal(whole[:32], c.pcm[:32]) {
		t.Fatal("first pass changed")
	}
	// At the wrap, continuation frame8 is followed by loop-head frame3.
	if binary.LittleEndian.Uint16(whole[32:]) != 800 || binary.LittleEndian.Uint16(whole[36:]) != 300 || binary.LittleEndian.Uint16(whole[40:]) != 400 {
		t.Fatal("wrong musical loop/crossfade")
	}
	fragmented := make([]byte, len(whole))
	q := c.NewStream()
	for at := 0; at < len(fragmented); {
		n := min((at%9)+1, len(fragmented)-at)
		if _, e := io.ReadFull(q, fragmented[at:at+n]); e != nil {
			t.Fatal(e)
		}
		at += n
	}
	if !bytes.Equal(whole, fragmented) {
		t.Fatal("PCM depends on audio read buffer boundaries")
	}
	if n, e := q.Read(nil); n != 0 || e != nil {
		t.Fatal("empty read")
	}
	r := c.NewStream()
	first := make([]byte, 4)
	r.Read(first)
	if !bytes.Equal(first, whole[:4]) {
		t.Fatal("shared playheads")
	}
}

func TestCachedMusicRejectsMalformedBanks(t *testing.T) {
	for _, b := range [][]byte{nil, smallCacheBank()[:31], smallCacheBank()[:60], append(smallCacheBank(), 0)} {
		if _, e := ParsePCMCache(b); e == nil {
			t.Fatal("bad length accepted")
		}
	}
	for _, at := range []int{8, 12, 16, 20, 24, 28} {
		b := smallCacheBank()
		binary.LittleEndian.PutUint32(b[at:], ^uint32(0))
		if _, e := ParsePCMCache(b); e == nil {
			t.Fatalf("invalid field%d", at)
		}
	}
}

func TestMenuBankPreservesSynthFirstPass(t *testing.T) {
	b, err := os.ReadFile("../assets/music/groove.aopcm")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParsePCMCache(b)
	if err != nil {
		t.Fatal(err)
	}
	live := NewStream("groove")
	live.SetIntensity(.25)
	cached := c.NewStream()
	a, z := make([]byte, musBlock*4), make([]byte, musBlock*4)
	// Check the opening, effects tails and the entire original arrangement.
	for at := 0; at < c.loopEnd*4; {
		n := min(len(a), c.loopEnd*4-at)
		// Render complete synth blocks, as the bank generator does. The old
		// live synth schedules events per block, including its final partial
		// block; shortening that call would change its scheduling near the end.
		live.Read(a)
		cached.Read(z)
		if !bytes.Equal(a[:n], z[:n]) {
			t.Fatalf("changed synth audio at byte%d", at)
		}
		at += n
	}
}

func BenchmarkLiveMenuSynthesis(b *testing.B) {
	s := NewStream("groove")
	s.SetIntensity(.25)
	buf := make([]byte, 4096)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Read(buf)
	}
}
func BenchmarkCachedMenuPlayback(b *testing.B) {
	bank, e := os.ReadFile("../assets/music/groove.aopcm")
	if e != nil {
		b.Fatal(e)
	}
	c, e := ParsePCMCache(bank)
	if e != nil {
		b.Fatal(e)
	}
	s := c.NewStream()
	buf := make([]byte, 4096)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Read(buf)
	}
}
