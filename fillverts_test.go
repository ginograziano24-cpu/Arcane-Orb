package main

import (
	"math/rand"
	"testing"
)

// fillVerts' single pass must give exactly what copying and then
// premultiplying in a second pass gave.
func TestFillVertsOnePass(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, n := range []int{3, 5000} { // small: one goroutine; big: split
		qs := make([]quad, n)
		ptrs := make([]*quad, n)
		for i := range qs {
			for k := range 4 {
				v := &qs[i].v[k]
				v.DstX, v.DstY = rng.Float32()*2000, rng.Float32()*1200
				v.SrcX, v.SrcY = rng.Float32(), rng.Float32()
				v.ColorR, v.ColorG, v.ColorB, v.ColorA = rng.Float32(), rng.Float32(), rng.Float32(), rng.Float32()
			}
			ptrs[i] = &qs[i]
		}
		ids := rng.Perm(n)
		for _, scale := range []float32{1, 0.75} {
			b := &Batch{}
			for _, i := range ids {
				b.runIdx = append(b.runIdx, int32(i))
			}
			b.fillVerts(ptrs, scale)
			for j, oi := range ids {
				for k := range 4 {
					want := qs[oi].v[k]
					want.ColorR *= want.ColorA
					want.ColorG *= want.ColorA
					want.ColorB *= want.ColorA
					if scale != 1 {
						want.DstX *= scale
						want.DstY *= scale
					}
					if b.verts[j*4+k] != want {
						t.Fatalf("n %d scale %v: vertex %d/%d differs", n, scale, j, k)
					}
				}
			}
		}
	}
}
