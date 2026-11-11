package main

import (
	"math"
	"reflect"
	"testing"
)

// Uniform inputs must keep exactly the existing shader values across world
// transitions and changing grades, and remain independent between renderers.
func TestPostUniformsPreserveValuesAcrossWorlds(t *testing.T) {
	p, q := &Post{}, &Post{}
	u := p.shaderInputs()
	v := q.shaderInputs()
	for _, g := range worldGrades {
		u.setGrade(g)
		for _, tc := range []struct {
			m    map[string]any
			k    string
			want []float32
		}{
			{u.bright, "Threshold", []float32{g.Threshold}}, {u.horizontal, "Dir", []float32{1.5, 0}}, {u.vertical, "Dir", []float32{0, 1.5}},
			{u.grade, "Tint", g.Tint[:]}, {u.grade, "Lift", g.Lift[:]}, {u.grade, "Saturation", []float32{g.Saturation}},
			{u.grade, "Contrast", []float32{g.Contrast}}, {u.grade, "Vignette", []float32{g.Vig}}, {u.upscale, "Sharp", []float32{.85}},
		} {
			if !reflect.DeepEqual(tc.m[tc.k], tc.want) {
				t.Fatalf("stale/changed %s: %v != %v", tc.k, tc.m[tc.k], tc.want)
			}
		}
	}
	if v.scalars != [4]float32{} || u == v {
		t.Fatal("renderer uniform state shared")
	}
	if got := testing.AllocsPerRun(100, func() { u.setGrade(Grade{Threshold: .5}); p.shaderInputs() }); got != 0 {
		t.Fatalf("steady-state uniform preparation allocates: %f", got)
	}
}

var uniformBitsSink uint32

// This is the same immediate conversion contract as Shader.AppendUniforms:
// maps are not retained, and scalar or vector floats become float32 bits.
//
//go:noinline
func benchmarkConsumeUniforms(m map[string]any) {
	for _, value := range m {
		switch v := value.(type) {
		case float32:
			uniformBitsSink ^= math.Float32bits(v)
		case []float32:
			for _, f := range v {
				uniformBitsSink ^= math.Float32bits(f)
			}
		}
	}
}

func BenchmarkPostUniformPreparationLegacy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g := worldGrades[WMeadow]
		g.Threshold += float32(i%10) * .001
		benchmarkConsumeUniforms(map[string]any{"Threshold": g.Threshold})
		for j := 0; j < 2; j++ {
			benchmarkConsumeUniforms(map[string]any{"Dir": []float32{1.5, 0}})
			benchmarkConsumeUniforms(map[string]any{"Dir": []float32{0, 1.5}})
		}
		benchmarkConsumeUniforms(map[string]any{"Tint": g.Tint[:], "Lift": g.Lift[:], "Saturation": g.Saturation, "Contrast": g.Contrast, "Vignette": g.Vig})
		benchmarkConsumeUniforms(map[string]any{"Sharp": float32(.85)})
	}
}

func BenchmarkPostUniformPreparationReused(b *testing.B) {
	p := &Post{}
	u := p.shaderInputs()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := worldGrades[WMeadow]
		g.Threshold += float32(i%10) * .001
		u.setGrade(g)
		benchmarkConsumeUniforms(u.bright)
		for j := 0; j < 2; j++ {
			benchmarkConsumeUniforms(u.horizontal)
			benchmarkConsumeUniforms(u.vertical)
		}
		benchmarkConsumeUniforms(u.grade)
		benchmarkConsumeUniforms(u.upscale)
	}
}
