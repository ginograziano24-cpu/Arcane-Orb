package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image/color"
	"runtime"
	"testing"
)

func projectedModelBank(t testing.TB) (*Batch, []*Mesh) {
	t.Helper()
	_, bank := modelTestBank(t)
	meshes := make([]*Mesh, len(bank))
	for i, data := range bank {
		m, err := parseModel(data)
		if err != nil {
			t.Fatal(err)
		}
		m.topology()
		meshes[i] = m
	}
	cam := &Camera{Eye: V3{0, 10, -20}, Target: V3{0, 1, 8}, FOV: 1.05, Roll: .018}
	cam.Setup(1280, 800)
	b := newBatch(cam)
	b.fogColor = [3]float32{.22, .16, .28}
	return b, meshes
}

func projectModelBank(b *Batch, meshes []*Mesh) {
	rotation := rotXYZ(.12, .45, .04)
	scale := V3{1.2, .8, 1.4}
	tint := color.RGBA{180, 220, 164, 255}
	for _, m := range meshes {
		for _, pos := range [...]V3{{0, 0, 10}, {-35, 0, 10}, {10, 0, 50}, {0, 0, -40}} {
			b.drawShared(m, pos, scale, rotation, tint, .08)
		}
	}
}

func TestProjectedModelBankOutput(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("reference uses amd64 camera trigonometry")
	}
	b, meshes := projectedModelBank(t)
	projectModelBank(b, meshes)
	digest := sha256.New()
	if err := binary.Write(digest, binary.LittleEndian, b.quads); err != nil {
		t.Fatal(err)
	}
	const expected = "c0dedb1a68bc1abe9d742801fd65373b3d88a50846c93a1798232f95d40f7ad4"
	if hash := fmt.Sprintf("%x", digest.Sum(nil)); hash != expected {
		t.Fatalf("projected positions, shading or ordering changed: %s (%d quads)", hash, len(b.quads))
	}
}

func BenchmarkProjectedModelBank(b *testing.B) {
	batch, meshes := projectedModelBank(b)
	projectModelBank(batch, meshes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch.quads = batch.quads[:0]
		projectModelBank(batch, meshes)
	}
}
