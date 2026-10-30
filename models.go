package main

import (
	"bytes"
	"compress/zlib"
	"embed"
	"encoding/binary"
	"image/color"
	"io"
	"math"
)

// Modelled assets from the Blender art pass (tools/artkit/scripts/models.py).
// Each .orbm file is a zlib-compressed list of coloured triangles in game
// units (x right, y up, z away from the camera), with the ambient occlusion
// baked into the face colours. Faces whose material is marked as tintable
// take the colour a mesh is drawn with; the rest keep their own.
//
// Code asks for a model by name and falls back to its procedural mesh when
// the model isn't there, so assets can be swapped in one at a time.

//go:embed assets/models
var modelFS embed.FS

var modelCache = map[string]*Mesh{}

// modelsOff lets the procedural meshes be compared against the art pass.
var modelsOff = false

// model returns the named asset, or nil if it isn't in the build.
func model(name string) *Mesh {
	if modelsOff {
		return nil
	}
	if m, ok := modelCache[name]; ok {
		return m
	}
	m := loadModel(name)
	modelCache[name] = m
	return m
}

func loadModel(name string) *Mesh {
	data, err := modelFS.ReadFile("assets/models/" + name + ".orbm")
	if err != nil {
		return nil
	}
	m, err := parseModel(data)
	if err != nil {
		return nil
	}
	return m
}

// parseModel reads the .orbm format: "ORBM", version, triangle count, then
// per triangle 9 float32 positions, RGBA, emission and flags bytes.
func parseModel(data []byte) (*Mesh, error) {
	if len(data) < 8 || string(data[:4]) != "ORBM" {
		return nil, io.ErrUnexpectedEOF
	}
	zr, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	if len(raw) < 8 {
		return nil, io.ErrUnexpectedEOF
	}
	n := int(binary.LittleEndian.Uint32(raw[4:8]))
	const rec = 9*4 + 4 + 2
	if len(raw) < 8+n*rec {
		return nil, io.ErrUnexpectedEOF
	}
	m := &Mesh{Tris: make([]Tri, n)}
	off := 8
	f := func() float64 {
		v := math.Float32frombits(binary.LittleEndian.Uint32(raw[off:]))
		off += 4
		return float64(v)
	}
	for i := 0; i < n; i++ {
		var t Tri
		for k := 0; k < 3; k++ {
			t.P[k] = V3{f(), f(), f()}
		}
		t.Col = color.RGBA{raw[off], raw[off+1], raw[off+2], 255}
		t.Emi = float32(raw[off+4]) / 255 * 2
		t.Fixed = raw[off+5]&1 == 0
		off += rec - 36
		m.Tris[i] = t
	}
	return m, nil
}

// drawModel draws the named asset if it exists and reports whether it did.
func drawModel(B *Batch, name string, pos, scale V3, rot Mat3, tint color.RGBA, emissive float32) bool {
	m := model(name)
	if m == nil {
		return false
	}
	B.Draw(m, pos, scale, rot, tint, emissive)
	return true
}
