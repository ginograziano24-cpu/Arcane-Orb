# Handoff: Claude, Nov 17 2026 (overnight session)

Git: pushed to GitHub. The PC folder (`Downloads/RoguelikeBreaker/ArcaneOrb-Updated`) has every game change; each passed `go vet`, `go test` and `go build` there (Go 1.26.8, `Go-Check.cmd`).

## Arcanist model
- **Face** (`tools/heroes/eyeproj.py`, `arcane_eyes_bake.py`): the bridge spike, cheek crack (seam-split vertices now welded before offsets), forehead gaps and stray strands to the hat brim are fixed. A faint line remains at the patch's lower edge on the right cheek.
- **Base body** (`tools/heroes/arcane_basebody.py`, run with `Arcanist-Base-Body.cmd`): a smooth clay form fitted to her measurements (RMS girth error 7.8% over 90 cross-sections), with an outward skin snap so it follows her visible skin, symmetric shoulders, the shoulder-top held to mannequin size, and the crotch lifted to a smooth join. Still to improve: the shoulders read a little broad and blocky, and there's a small dent on the upper back.

## Game polish (`POLISH_AUDIT.md` has the progress table)
- Near-miss quip; Settings split into Effects and Gameplay tabs (translated); pooled `near()` buffers; dead code removed.
- **Class select fix:** 377 → 672 fps, 394 KB → 5 KB garbage per frame. The cause was the hero render target resizing every frame.
- World 7 (dream scenery) is 146 fps vs 190 on Nov 13. It's a content cost; fixing it means a look change, so it's left as a taste call.
- **Memory soak:**
  - Bush flora variants were an unbounded cache (blossom tints vary per chamber) and are now quantised.
  - The soak now ages the frame-based caches (`soakAgeCaches`). Heap growth is +1.9 MB/run and levelling off (it was +15).
  - `Soak.cmd` writes heap profiles for diffing.
- **PC launchers** (copies in `tools/pc/`):
  - `Shots-Bench.cmd`: screenshots plus the benchmark.
  - `Bench-Clean.cmd`
  - `Soak.cmd`
  - `Build-Platforms.cmd`
  - `Roblox-LevelBank.cmd`

## Roblox port (`docs/ROBLOX_PORT.md`)
- `roblox/` is a Rojo project with the Phase 1 core loop.
- Data tables are generated from Go: run `tools/luauport` (Go), which translates the item `Apply` closures too.
- 288 layouts come from the real level generator (`tools/levelbank`, run on the PC).
- Sim/Level and Sim/Run are a hand port of play.go and run.go.
- Renderer, UI and server meta save are included.
- Tested headless with `tools/luauport/luaucheck.py` (syntax) and `luaurun.py` (a small Luau interpreter: whole runs plus an every-item test). It has **not been opened in Roblox Studio yet**.
- Chambers take longer than on Steam under the autopilot. Origin passives aren't ported yet, and that is likely part of the gap.

## Platforms (`docs/PLATFORMS.md`)
- Linux, macOS and web all cross-compile with no Cgo.
- The Linux build ran under Xvfb here: menu, origin select, chambers and a Guardian all drew correctly.
- With no audio device the game quits on start (an upstream Ebitengine behaviour). The proposed fix is `docs/patches/ebiten-silent-audio.patch`. The PC's `third_party` files are read-only, so it isn't applied: that's the user's call.

## Brainstorm
- `docs/ENJOYMENT_1_5X.md`: proxies for "1.5× more fun", then ideas in three tiers. Tier 1 first: last-brick juice, a build-around first card, combo stakes, a paid reward reroll, map modifiers.

## Next
1. Open `roblox/` in Studio via Rojo, play a chamber and fix whatever the Output window shows.
2. Port the origin passives (`passives.go`) to the Roblox sim.
3. Face: the lower-cheek patch edge. Body: soften the shoulders.
4. The purego Steam bridge for native Linux, then a Steam Deck check.
5. Tier 1 enjoyment items, measured with the balance simulator.
