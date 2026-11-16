package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The test guards the renderer, not the art, so it keeps projecting the bank
// it was recorded with. Models added later (Tripo art pass) are skipped, and
// models replaced since then are projected from their original copies in
// testdata/projection-bank. New and replaced models are covered by
// TestAllModelGeometryAndTopologyRemainIdentical.
var projectionNewModels = map[string]bool{
	"menhir.orbm": true, "guardian_1.orbm": true, "guardian_2.orbm": true, "guardian_3.orbm": true,
	"crystal_formation.orbm": true,
}

var projectionReplacedModels = map[string]bool{
	"boiler.orbm": true, "stone_head.orbm": true, "sky_castle.orbm": true,
}

func projectedModelBank(t testing.TB) (*Batch, []*Mesh) {
	t.Helper()
	names, bank := modelTestBank(t)
	meshes := make([]*Mesh, 0, len(bank))
	for i, data := range bank {
		if projectionNewModels[names[i]] {
			continue
		}
		if projectionReplacedModels[names[i]] {
			orig, err := os.ReadFile(filepath.Join("testdata", "projection-bank", names[i]))
			if err != nil {
				t.Fatal(err)
			}
			data = orig
		}
		m, err := parseModel(data)
		if err != nil {
			t.Fatal(err)
		}
		m.topology()
		meshes = append(meshes, m)
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
	// re-recorded when near-plane clipping replaced the old clamp: the bank
	// placement behind the camera (z -40) no longer smears across the view
	const expected = "8fec0cf310212f0a8716be9802c4e91328f8d9329b46d9ba7aac3c122419ee0d"
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
