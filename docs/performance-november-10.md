# Arcane Orb â€” first performance review build

A separate local candidate. Your original game, saves, Downloads and private phone preview are unchanged.

Use Play-Performance-Review.cmd to play without saving. Compare-Live-Menu.cmd uses the same candidate with the old live menu synthesis for listening comparison. These launchers have not been run on your desktop. The four-second menu-loop-join-review.wav crosses the cached loop boundary halfway through; human listening is still required.

Implemented: native menu music rendered before packaging, single-lookup warmed SFX cache, timely finished-player cleanup, reusable post-processing uniform inputs, standard integer formatting, pooled model decode scratch and smaller initial topology allocations. No item values, difficulty, content, hero geometry or shader formulas changed. Browser/capture builds retain live music to avoid adding18.5MB to their download.

Component benchmarks on this Ryzen7950X3D (three trials each): live menu read about0.423ms per4096bytes versus cached about0.000040ms. This excludes audio-device mixing/resampling and is NOT a whole-game speedup. Raw cache bank is18.5MB. Small integer formatting goes from3 allocations to0; large score formatting14 to1. Warmed SFX lookup goes from1 allocation to0. Post uniform setup benchmark goes from15 allocations to0; shader output values remain the same. Model decode bank temporary allocation is about13.88MB before versus3.42MB after; shared-corner construction about3.97MB versus2.88MB. See exact medians and timing scopes in PERFORMANCE-STATUS.json and raw reports.

Measurement tools now support p95/p99/p99.9/max frame intervals,0.1percent lows, budget misses, duration controls, explicit scene subsets, optional audio, and separate transition/warmup observations. Recorder storage is reserved before timing and the following interval excluded to avoid measuring its own allocation as a stutter.

All Go package tests pass, including local multiplayer loopback. Native and WASM vet/build checks pass. All102 authored model geometry/material/topology hashes match the pre-change baseline; original assets and dependency files remain byte-identical. Full-game GPU/FPS measurements and listening/playback review are pending PC availability. New High/Medium/Low effects, frame caps and adaptive game-music caching have not been accepted or implemented.

For a short native A/B when the PC is idle, use both executables with:
-bench -benchwin=1280x800 -benchscenes=menu,multiball,boss -benchduration=10s -benchwarmup=2.5s -benchaudio -benchtag=UNIQUE
This first pair takes roughly two minutes. Use distinct tags and repeat three comparable runs before making firm performance claims. Keep other games closed. Do not run unattended while PC availability is unknown. Each benchmark exits automatically and never writes save/run progress. Old uninstrumented ArcaneOrb-Baseline.exe is preserved for provenance; use ArcaneOrb-Instrumented-Baseline.exe for identical measurement controls.

Source changes are in ../ArcaneOrb-Performance-Branch. A text patch is included; the generated binary music asset and its manifest must accompany it. Regenerate with go generate . after changing synth code; stale asset tests enforce this. No deployment or remote push has been performed.
