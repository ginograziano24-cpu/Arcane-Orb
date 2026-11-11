package directx

import (
	"bytes"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
)

// CompileFXCBytecode uses the same profiles and flags as runtime shader creation.
func CompileFXCBytecode(program *shaderir.Program) (vertex, pixel []byte, err error) {
	vs, ps, _, _ := hlsl.Compile(program)
	compile := func(source, entry, profile string) ([]byte, error) {
		blob, err := _D3DCompile([]byte(source), "shader", nil, nil, entry, profile, uint32(_D3DCOMPILE_OPTIMIZATION_LEVEL3), 0)
		if err != nil {
			return nil, err
		}
		defer blob.Release()
		return bytes.Clone(unsafe.Slice((*byte)(blob.GetBufferPointer()), blob.GetBufferSize())), nil
	}
	vertex, err = compile(vs, VertexShaderEntryPoint, VertexShaderProfile)
	if err != nil {
		return nil, nil, err
	}
	pixel, err = compile(ps, PixelShaderEntryPoint, PixelShaderProfile)
	return vertex, pixel, err
}
