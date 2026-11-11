package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
	"github.com/hajimehoshi/ebiten/v2/internal/legacyshader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderprecomp"
)

// CompileArcaneDXShader is used by the Windows asset generator.
func CompileArcaneDXShader(source []byte) (id string, vertex, pixel []byte, err error) {
	program, _, err := legacyshader.CompileShader(source)
	if err != nil {
		return "", nil, nil, err
	}
	vertex, pixel, err = directx.CompileFXCBytecode(program)
	return program.SourceID.String(), vertex, pixel, err
}

// RegisterArcaneDXShader skips bytecode for an older shader or engine.
func RegisterArcaneDXShader(source []byte, expectedID string, vertex, pixel []byte) bool {
	id, err := legacyshader.CalcSourceID(source)
	if err != nil || id.String() != expectedID || len(vertex) == 0 || len(pixel) == 0 {
		return false
	}
	shaderprecomp.RegisterDXBCsForWindows(id, vertex, pixel)
	return true
}
