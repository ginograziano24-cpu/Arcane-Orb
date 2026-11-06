# Arcane Orb: handoff prompt for another AI assistant (the game itself)

Paste this whole file as your first message. Then share the repository (GitHub
`ginograziano24-cpu/Arcane-Orb`, branch `main`) or the source zip. A separate brief for the opening
intro is in `HANDOFF_CHATGPT.md`.

---

## Your role

You're continuing work on **Arcane Orb**, a roguelite brick-breaker for Steam (Windows first), made by
a solo developer (the user). The game is feature-complete at "v1.0 RC". The work now is performance,
polish, and the content roadmap below. Work in small, verifiable steps; show screenshots or numbers;
commit after each milestone; keep `HANDOFF.md` and `ROADMAP.md` current.

## The game

- Seven hero origins (run.go): guardian (Dame Oriel), berserker (Brakka), arcane (Professor Vell),
  engineer (Rivet), lucky (Lucky Sol), pyro (Cinder), storm (Zephyr).
- A run is three acts of four worlds each, items carried between acts (biome.go `acts`); bosses: Prism
  Colossus, Crucible Golem, Crystal Warden. Also endless, daily, classic, challenge chambers
  (challenges.go) and online duels (duel*.go).
- 7 languages (en zh ja ru es pt de). Every new player-facing string needs all 7
  (`tools/i18n/src/*.jsonl`, rebuild with `go run tools/i18n/extract.go > tools/i18n/strings.txt &&
  python3 tools/i18n/build.py`; CJK glyphs must be in the font subsets, `tools/i18n/subset.py`).
- Music: 5 synthesised songs in package `mus` (menu, acts 1–3, boss); `tools/musrender` renders them.

## Tech stack

- **Go 1.24.7** with a **patched Ebitengine v2.8.8** in `third_party/ebiten` (go.mod `replace`).
- Build (Windows): `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ArcaneOrb.exe .`
- Vet both targets: `GOOS=windows go vet . && GOOS=js GOARCH=wasm go vet -tags capture .`
- Playtest zip: `ArcaneOrb/` with ArcaneOrb.exe, PLAYTEST.txt, README.txt, steam_api64.dll, and
  `steam_appid.txt` containing `480`.
- Test harness (`tools/harness/`): a wasm "capture" build run in headless Chromium for screenshots,
  draw-order hashes, campaign balance sims and planner timings (see the table in `HANDOFF.md`).

## Performance work already done

### The renderer (how it works now)

The world is drawn as CPU-projected quads, sorted by depth and drawn with the painter's algorithm.

- **Side jobs on goroutines.** Scenery, dressing, arena floor, walls, posts, barrier and bricks are drawn
  in parallel into side batches (world.go). Big jobs are split into **pieces**: each piece runs the
  whole job but only emits its own cost window, so output is identical to the unsplit job
  (`beginPiece` in render3d.go).
- **Presorting.** Each side batch sorts its own opaque and glow lists on its own goroutine, starting from
  last frame's order (cheaper than a full radix sort).
- **Flush.** Merges the presorted lists (merge passes parallel on 4+ threads), then a **run planner**
  groups quads by texture and mip level without breaking overlap order, gathers vertices in parallel and
  submits them with as few draw calls as possible.
- **Culling** drops quads wholly off-screen when they're emitted. Code that edits the quad it just added
  must use `B.lastQuad()` / `B.lastGlow()`, because pieces and culling can skip a quad.
- **Anti-aliasing** is our own 2x supersampled target (`aaBegin`/`aaEnd`), not Ebitengine's.
- **Post** (post.go): bloom (bright pass, two separable blur passes run twice), colour grade, vignette,
  and an FSR1-style 16-tap clamped Catmull-Rom upscaler when rendering below native resolution.
- **Dynamic resolution** (settings.go `renderScale`): drops the 3D resolution in 5% steps (down to 60%)
  when frames run long, and climbs back when there's headroom. Quality presets Low/Medium/High set render
  scale, bloom, particles and AA.
- **UI** is drawn with an SDF shape renderer (sdf.go): every rounded box, outline, glow, capsule and arc
  is one quad with an anti-aliased shader, and consecutive shapes batch into one draw call. Text is cached
  (textcache.go).

### Ebitengine patches (`third_party/ebiten/ARCANE_PATCHES.md`)

1. **DX11 uniform buffer reuse**: uniforms adjusted into a buffer kept on the shader instead of a fresh
   slice per draw. These allocations were ~80% of all bytes the game allocated (12.3 GB in a 128 s run),
   driving the GC.
2. **Restorable scan skip**: `makeStaleIfDependingOn` returns at once when restoration is off (every
   platform but Android); it was scanning every image on every draw call.
3. **Command slice reuse** and skipping the per-command debug log call (its variadic argument allocated).
4. **`DrawTrianglesOptions.UniformMipmapLevel`**: the planner's runs share one mip level, so the mipmap
   layer reads it from the first quad instead of every triangle edge.
5. **`DrawTriangles32`** with 32-bit indices and a zero-copy vertex path.
6. **Fused atlas loops** (offset and texel-scale in one pass).

To upgrade Ebitengine, copy the new module in the same way and re-apply these.

### Loading and audio

- **prefetch.go**: floors, backdrops, the brick sheet, portraits, HUD poses and scenes are built once by a
  background worker ahead of need (current act and the next); all models are parsed at boot.
- **audio.go**: sound effects render once per clip on 4 warming workers; each new world or track queues
  its brick sounds; a clip that isn't ready starts late from a goroutine instead of stalling the frame.

### Measuring

- **F3 overlay** (perf.go): section timers (update, world, flush, post, HUD, draw), draw counters
  (quads, 3D draws, SDF, sprites, text, rects) and allocation rates; cheap enough for release builds.
- **Benchmark tour**: `ArcaneOrb.exe -bench -benchtag=NAME` plays every scene type and writes
  `bench_report_NAME.txt` plus CPU and alloc pprofs. A/B switches:
  `-nopieces -noparmerge -ebitenaa -glowaside -nopresort -noruns -notextcache`.

### Latest PC numbers (2026-11-01, v1.0 RC, 2560x1440 fullscreen, vsync off, High, 32-thread CPU)

| Scene | DX11 fps (1% low) | Game-thread ms (world + flush) | Quads |
|---|---|---|---|
| menu | 615 (156) | 1.6 | 0 |
| world1 | 205 (148) | 4.9 (1.8 + 2.1) | 12k |
| world2 | 119 (101) | 8.4 (3.4 + 3.9) | 22k |
| world3 | 262 (196) | 3.8 | 9k |
| world4 | 152 (127) | 6.6 | 17k |
| world5 | 94 (74) | 10.6 (4.4 + 5.1) | 28k |
| world6 | 119 (98) | 8.4 | 22k |
| multiball | 149 (123) | 6.7 | 17k |
| boss | 380 (241) | 2.6 | 5k |
| shop / reward | 487 / 1253 | 2.0 / 0.8 | |

- **The game is CPU-bound on the game thread**: almost all frame time is `world` (building the scene)
  plus `flush` (sorting and submitting), and it scales with quad count. World 5 is the worst case.
- Allocations are low (~2.8k objects / ~390 KB per frame in worlds).
- **DirectX 12 is broken for us**: the same build on DX12 ran at 8–9 fps (110+ ms waiting on the driver).
  Keep DX11 (`EBITENGINE_DIRECTX=version=11` is the default the game uses); our uniform patch only covers DX11.
- An older benchmark ("perf4") reported higher fps on a different build; re-run the bench after any
  renderer change and compare like with like (same build flags, same settings).

## Performance plan (next) — full detail and sources in `docs/PERF_NEXT.md`

**A. Toolchain (cheap, measurable):**
1. Go 1.26: Green Tea GC on by default (10–40% less GC overhead), more stack allocation.
2. PGO: merge the bench's CPU pprofs into `default.pgo` next to main.go (Go reports 2–14% CPU gains).
3. Ebitengine 2.10 (needs Go 1.25+): Proton/DXVK crash fix (Steam Deck), VSync-off fix, no memory growth
   while minimised, shader precompilation; patch 5 is upstream now, re-check patch 2.
4. A native Linux build becomes possible (2.10 dropped Cgo on desktop) — ask the user first.

**B. Pacing, 1% lows, heat:** frame-rate cap setting (Off/60/120/144/monitor), idle throttling in menus and
when unfocused, GC tuning (`SetMemoryLimit`, `SetGCPercent`), shader warm-up at boot, 0.1% lows and
frame-time deviation in the bench, a Steam Deck bench preset (1280x800, Medium, 60 cap).

**C. Since the game thread is the bottleneck, prioritise:**
- **Retained static quads** (C5): walls, posts and the floor rarely change within a world; cache their
  projected vertices per camera pose and only re-emit on a camera or brick change.
- **Cached distant layers** (C4): render far scenery and the backdrop into a texture, redraw only when the
  camera moves past a threshold; small shakes become a 2D offset. Removes their quads from sort and flush.
- Particle and glow budgets per preset (multiball and bursts hurt the 1% low).
- Then GPU-side items: dual-Kawase bloom, fewer full-screen passes, single-pass AA for Medium.

Every change: run the benchmark tour before and after (plain, `-nopieces`, `-noparmerge`), check the
draw-order hash in the harness for changes that should draw identically, and compare screenshots.

## Roadmap (`ROADMAP.md`)

One session per step, each ending in a playtest build:

1. **Lore** — done in code, **not yet compiled** (written when Go modules couldn't be downloaded):
   story.go, classsel.go, progress_ui.go, game.go, lore.go, `tools/i18n/src/19.jsonl`. First job: build,
   vet, fix, then screenshot class select, the Lore page and the story cards in all 7 languages.
   Story: three bosses bury the world in bricks; seven heroes use the Orb. Text in `docs/lore.md`.
2. **Origin quips** — text plus short synthesised voice blips per hero; lines drafted in `docs/quips.md`
   (awaiting the user's read-through). Speech bubble by the HUD portrait, rare, with a setting to turn off.
3. **Chamber select redesign** — approved direction "B2, arcane constellation" (rotating rune circle,
   summoning circle round the boss, rune rings on pickable rooms). Keep the run panel (top left) and the
   Build panel (right); Build panel lists items by rarity, hover shows the card, and a stats section shows
   base stats with current stats beside them in another colour.
4. **Cameo challenges** — play one chamber as a different origin; locked heroes included as a teaser.
5. **Soundtrack selection** — name the 5 tracks, a Music screen, per-run music option (names still open).
6. **Duel stage select** — host picks a world or random; one new compatible network field.
7. **Origin showcases** — 10–15 s self-playing demo per origin on class select (built on trailer autoplay).
8. **Collectible gallery** — Blender scenes of the hero models, unlocked by challenge chambers.
9. **More items** — 10–15 that change how you play; balance with the campaign sim (origin spread ≤ 15%).
10. **Unique bosses** — each boss its own kit and 2–3 phases (ideas listed in ROADMAP.md; user to pick).

Also in progress: the **opening intro** (see `HANDOFF_CHATGPT.md`), and **Steam key art** rendered at
every size in `marketing/steam/` (`tools/keyart`).

## Balance notes

Act 3 carried runs: origin spread 11.4%, 8 timeouts in the last full sim. Engineer runs ~20% fast (watch
it). Gambler +25% luck, 8 gold / +45%. Storm chain +0.25. Berserker crit removed. Power controller:
power.go (learnt per-item feed-forward); each new item needs an entry.

## Rules the user has set

- Don't work around blocked downloads or permissions; say what's blocked.
- Magic Ball (Alawar) inspired the game and music: analysis only, never copy its audio.
- Never run the PC benchmark while another game is open or while the user is away.
- Commit and push after each milestone; keep the docs current; show pictures and numbers.
