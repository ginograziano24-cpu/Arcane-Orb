# Arcane Orb patches to Ebitengine v2.8.8

This is Ebitengine v2.8.8 (github.com/hajimehoshi/ebiten/v2, Apache 2.0, see
LICENSE) without its examples and tests, used through a `replace` in the
game's go.mod. Changes, each marked "Arcane Orb patch" in the source:

1. `internal/graphicsdriver/directx`: DirectX 11 adjusts each draw's
   uniforms into a buffer kept on the shader (`adjustUniformsTo`) instead
   of a fresh slice per draw. Those allocations were ~80% of all bytes the
   game allocated (a benchmark run: 12.3 GB in 128 s), driving the GC.
2. `internal/restorable`: `makeStaleIfDependingOn(AtRegion)` return at once
   when restoration is off (every platform but Android). They ran over
   every image on every draw call, and without restoration no draw history
   is kept, so they could not change anything.
3. `internal/graphicscommand`: a new draw command reuses its pooled
   struct's destination-region slice instead of allocating one per
   command, and the per-command frame log call (a no-op outside debug
   builds, but its variadic argument allocated) is skipped unless
   `debug.IsDebug`.
4. `DrawTrianglesOptions.UniformMipmapLevel` (new): the caller promises
   every quad of the draw needs the same mipmap level, so the mipmap layer
   works it out from the first quad instead of every edge of every
   triangle (a new restorable hint, HintUniformMipmapLevel, carries it
   there and is dropped before the restorable layer). The game's planned
   runs are keyed by mip level, so they set it.
5. `Image.DrawTriangles32` (new, as in later Ebitengine versions): 32-bit
   indices passed on as they are, without the per-call conversion. With
   premultiplied colour scales, no colour matrix and whole (not sub-)
   images, where its per-vertex conversion would change nothing, it uses
   the caller's vertices in place (Vertex has the internal layout), so it
   may overwrite them.

To move to a newer Ebitengine: copy its module here the same way (no
examples, tests or testdata) and re-apply the patches above.
