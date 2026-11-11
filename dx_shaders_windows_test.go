package main

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var generateDXShaders = flag.Bool("generate-dx-shaders", false, "regenerate bundled DirectX shader bytecode")

func TestGenerateDXShaders(t *testing.T) {
	if !*generateDXShaders {
		t.Skip("explicit asset generation only")
	}
	shaders := make(map[string]dxShaderBinary)
	for name, source := range dxShaderSources() {
		hash, vertex, pixel, err := ebiten.CompileArcaneDXShader([]byte(source))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		shaders[name] = dxShaderBinary{Hash: hash, Vertex: vertex, Pixel: pixel}
	}
	data, err := json.Marshal(shaders)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("assets/dxshaders.json", data, 0644); err != nil {
		t.Fatal(err)
	}
}
