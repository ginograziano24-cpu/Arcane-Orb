package ebiten

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/legacyshader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderprecomp"
)

func TestArcaneDXCacheMatchesRuntime(t *testing.T) {
	source := []byte(`//kage:unit pixels

package main

func Fragment(position vec4, srcPos vec2, color vec4) vec4 {
    return imageSrc1At(srcPos) * vec4(0.2, 0.45, 0.7, 1.0)
}
`)
	program, _, err := legacyshader.CompileShader(source)
	if err != nil {
		t.Fatal(err)
	}
	id, vertex, pixel, err := CompileArcaneDXShader(source)
	if err != nil {
		t.Fatal(err)
	}
	if id != program.SourceID.String() {
		t.Fatalf("generated shader ID %s differs from runtime ID %s", id, program.SourceID)
	}
	if RegisterArcaneDXShader(append(bytes.Clone(source), '\n'), id, vertex, pixel) {
		t.Fatal("registered bytecode for altered source")
	}
	if _, _, registered := shaderprecomp.DXBCs(program.SourceID, shaderprecomp.DXBCPlatformWindows); !registered && !RegisterArcaneDXShader(source, id, vertex, pixel) {
		t.Fatal("could not register compiled shader")
	}
	gotVertex, gotPixel, ok := shaderprecomp.DXBCs(program.SourceID, shaderprecomp.DXBCPlatformWindows)
	if !ok || !bytes.Equal(vertex, gotVertex) || !bytes.Equal(pixel, gotPixel) {
		t.Fatal("the runtime ID did not retrieve the compiled shader bytes")
	}
}
