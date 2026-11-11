package main

import (
	_ "embed"
	"encoding/json"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/dxshaders.json
var dxShaderData []byte

type dxShaderBinary struct {
	Hash          string
	Vertex, Pixel []byte
}

func dxShaderSources() map[string]string {
	return map[string]string{
		"access":       daltonSrc,
		"orb":          orbSrc,
		"post-bright":  brightSrc,
		"post-blur":    blurSrc,
		"post-grade":   gradeSrc,
		"post-upscale": upscaleSrc,
		"sanctum":      sanctumSrc,
		"sdf":          sdfSrc,
		"sky":          skyfxSrc,
	}
}

func init() {
	var shaders map[string]dxShaderBinary
	if json.Unmarshal(dxShaderData, &shaders) != nil {
		return
	}
	for name, source := range dxShaderSources() {
		if shader, ok := shaders[name]; ok {
			ebiten.RegisterArcaneDXShader([]byte(source), shader.Hash, shader.Vertex, shader.Pixel)
		}
	}
}
