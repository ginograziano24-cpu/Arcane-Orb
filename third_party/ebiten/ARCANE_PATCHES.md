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

To move to a newer Ebitengine: copy its module here the same way (no
examples, tests or testdata) and re-apply the patches above.
