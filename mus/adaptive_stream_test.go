package mus

import (
	"bytes"
	"testing"
)

func TestAdaptiveStreamsAreIndependent(t *testing.T) {
	const blocks = 180
	reference := NewStream("rush")
	reference.SetIntensity(0.8)
	baseline := make([]byte, blocks*musBlock*4)
	for offset := 0; offset < len(baseline); offset += musBlock * 4 {
		reference.Read(baseline[offset : offset+musBlock*4])
	}

	a, b := NewStream("rush"), NewStream("warden")
	a.SetIntensity(0.8)
	b.SetIntensity(0.9)
	got, other := make([]byte, musBlock*4), make([]byte, musBlock*4)
	for i := range blocks {
		b.Read(other)
		a.Read(got)
		if !bytes.Equal(got, baseline[i*len(got):(i+1)*len(got)]) {
			t.Fatalf("another stream changed PCM at block %d", i)
		}
	}
}

func BenchmarkAdaptiveStreams(b *testing.B) {
	for _, name := range []string{"rush", "acid", "abyss", "warden"} {
		b.Run(name, func(b *testing.B) {
			stream := NewStream(name)
			stream.SetIntensity(0.85)
			data := make([]byte, 4096)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				stream.Read(data)
			}
		})
	}
}
