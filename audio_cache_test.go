package main

import (
	"bytes"
	"sync"
	"testing"
)

var soundPCMSink []byte

func BenchmarkWarmedSoundLookup(b *testing.B) {
	s := &Sounds{cache: map[string]*clipEntry{}}
	s.clip("brk.1.0.3", 1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if c, ready := s.cachedClip("brk.1.0.3", 1); ready {
			soundPCMSink = c
		}
	}
}

func TestSoundCacheReadinessAndVariants(t *testing.T) {
	s := &Sounds{cache: map[string]*clipEntry{}}
	if _, ready := s.cachedClip("brk.1.0.3", 1); ready {
		t.Fatal("missing clip ready")
	}
	e := &clipEntry{data: []byte{1, 2, 3}}
	s.cache["pending#0"] = e
	if _, ready := s.cachedClip("pending", 0); ready {
		t.Fatal("unpublished data visible")
	}
	e.done.Store(true)
	if c, ready := s.cachedClip("pending", 0); !ready || !bytes.Equal(c, e.data) {
		t.Fatal("published PCM missing")
	}
	want := s.clip("brk.1.0.3", 1)
	if len(want) == 0 {
		t.Fatal("fixture recipe missing")
	}
	if c, ready := s.cachedClip("brk.1.0.3", 1); !ready || &c[0] != &want[0] {
		t.Fatal("warmed lookup copies or changes samples")
	}
	if _, ready := s.cachedClip("brk.1.0.3", 0); ready {
		t.Fatal("variant readiness leaked")
	}
}

func TestConcurrentSoundWarmingPublishesOneBuffer(t *testing.T) {
	s := &Sounds{cache: map[string]*clipEntry{}}
	var wg sync.WaitGroup
	got := make([][]byte, 16)
	for i := range got {
		wg.Add(1)
		go func(i int) { defer wg.Done(); got[i] = s.clip("brk.1.0.3", 1) }(i)
	}
	wg.Wait()
	for _, c := range got {
		if len(c) == 0 || &c[0] != &got[0][0] {
			t.Fatal("concurrent warming did not reuse one PCM buffer")
		}
	}
	if c, ready := s.cachedClip("brk.1.0.3", 1); !ready || !bytes.Equal(c, got[0]) {
		t.Fatal("warm data not published")
	}
}
