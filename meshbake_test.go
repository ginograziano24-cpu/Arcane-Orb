package main

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"testing"
)

// The baked path must emit exactly the quads drawShared does.
func TestBakedMatchesShared(t *testing.T) {
	b1, meshes := projectedModelBank(t)
	b2, _ := projectedModelBank(t)
	rotation := rotXYZ(.12, .45, .04)
	scale := V3{1.2, .8, 1.4}
	for _, tint := range []color.RGBA{{180, 220, 164, 255}, {}} {
		for _, m := range meshes {
			for _, pos := range [...]V3{{0, 0, 10}, {-35, 0, 10}, {10, 0, 50}, {0, 0, -40}} {
				b1.drawShared(m, pos, scale, rotation, tint, .08)
				g := bakeMesh(m, pos, scale, rotation, tint, .08)
				b2.drawBaked(g, b2.bakeColors(g))
			}
		}
	}
	if len(b1.quads) != len(b2.quads) || len(b1.quads) == 0 {
		t.Fatalf("quad counts differ: %d vs %d", len(b1.quads), len(b2.quads))
	}
	var w1, w2 bytes.Buffer
	binary.Write(&w1, binary.LittleEndian, b1.quads)
	binary.Write(&w2, binary.LittleEndian, b2.quads)
	if !bytes.Equal(w1.Bytes(), w2.Bytes()) {
		t.Fatal("baked quads differ from drawShared")
	}
}

// A placement is baked on its second frame and reused after that.
func TestBakeCacheFrames(t *testing.T) {
	b, meshes := projectedModelBank(t)
	m := meshes[0]
	pos := V3{1, 0, 12}
	if b.drawCached(m, pos, v3(1), rotXYZ(0, 0.3, 0), color.RGBA{}, 0) {
		t.Fatal("baked on first sight")
	}
	bakeTick()
	if !b.drawCached(m, pos, v3(1), rotXYZ(0, 0.3, 0), color.RGBA{}, 0) {
		t.Fatal("not baked on the second frame")
	}
}
