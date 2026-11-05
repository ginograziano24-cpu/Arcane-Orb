# Handoff: where Arcane Orb stands

Read this first in a new session, then `ROADMAP.md` for what's next.

## The project

- Go 1.24.7, Ebitengine v2.8.8 (patched local copy, see below). A
  roguelite brick-breaker for Steam (Windows first).
- Repo: GitHub `ginograziano24-cpu/Arcane-Orb`, branch `main`. Commit and
  push after each milestone. Commit messages end with the session's
  `Co-Authored-By` / `Claude-Session` lines.
- Seven origins (run.go: guardian, berserker, arcane, engineer, lucky,
  pyro, storm). A run is three acts of four worlds each, with items carried
  between acts (biome.go `acts`). There are also endless, daily, classic,
  challenge chambers (challenges.go) and online duels (duel*.go).
- Story cards: story.go. Hero Lore pages: lore.go (the story itself is in docs/lore.md). Translations: 7 languages (en zh ja ru es pt de).

## Building

```sh
# Windows exe (what the user plays)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ArcaneOrb.exe .

# vet both targets (the Linux build needs X11 headers this container lacks)
GOOS=windows go vet . && GOOS=js GOARCH=wasm go vet -tags capture .
```

- **Playtest zip.** Make an `ArcaneOrb/` folder with:
  - ArcaneOrb.exe
  - PLAYTEST.txt and README.txt (both in the repo root)
  - steam_api64.dll (Steamworks SDK redistributable; a copy is in the user's Downloads\ArcaneOrbBench)
  - steam_appid.txt containing `480`

  Then zip the folder.
- **Benchmark kit.** The device bridge takes files up to about 26 MB:
  - Split the exe with `split -b 26000000 -d -a 2 ArcaneOrb.exe ArcaneOrb.part`.
  - Add a `run_bench.bat` with CRLF line endings. It rejoins the parts with `copy /b ArcaneOrb.part00+ArcaneOrb.part01 ArcaneOrb.exe`, then runs `ArcaneOrb.exe -bench -benchtag=NAME [switches]`.
  - Put it all in `C:\Users\Gino\Downloads\ArcaneOrbBench`.
  - The bench writes `bench_report_NAME.txt` plus CPU and alloc pprofs next to the exe.
  - Bench A/B switches: `-nopieces -noparmerge -ebitenaa -glowaside -nopresort -noruns -notextcache`.
- **Running it on the PC.** This needs the linked computer:
  - Open File Explorer (click-only access, so no typing) → Downloads → ArcaneOrbBench → double-click run_bench.bat.
  - It takes about 2 min per configuration.
  - Release the computer-use lock afterwards.
  - Never run the bench while a game (e.g. Deadlock) is open, or when the user said they're away.

## Constraints

- Magic Ball (Alawar) is the inspiration for the game and its music. Use its audio for analysis only; never copy its audio.
- Package registries and other hosts that return 403 are blocked: do not work around them.
- The user's File Explorer is granted at "click" tier: do not type into it, and do not work around that.

## Checking changes (`tools/harness/`)

The harness runs the wasm capture build in headless Chromium (Playwright is preinstalled).

1. `tools/harness/setup.sh 8` builds `tools/harness/web8/game.wasm` and serves the folder on :8767.
2. Run the checks from `tools/harness`.

| Check | Command |
|---|---|
| Screenshots | `WEB=8 node golden.js OUTDIR scene,scene,...` |
| Screenshots with run planning active (draws the last N frames) | add `DRAWS=5` |
| Compare two screenshot sets (max / mean / pixel counts per scene) | `python3 imgdiff.py DIR_A DIR_B [DIFFDIR]` |
| Draw-order hash of a scene; must match between builds that should draw the same | `WEB=8 DRAWS=6 node hashw.js SCENE 90` |
| Campaign sim (balance) | `WEB=8 node campw.js out.json 'C{"seeds":6,"maxT":400,"carry":true}'`, then `python3 camp3.py out.json` |
| Planner and side-job timings (wasm, single-threaded: relative only) | `WEB=8 JOBS=1 node planb.js world2,world5` |

- Scenes: `menu classsel map world1..world6 multiball classic2 classic5 boss shop reward`.
- The machine has 2 cores: don't run sims and timings at once.
- The debug hook `__trMiss` (capture build) records untranslated strings.

## Renderer, in one paragraph

Each frame, the world is drawn as CPU-projected quads, sorted by depth and drawn with the painter's algorithm.

- **Side jobs.** Scenery, dressing, the arena floor, walls, posts, the barrier and bricks are drawn on goroutines into side batches (world.go). The big jobs are split into "pieces": each piece runs the whole job but emits only its cost window (`beginPiece` in render3d.go; output is identical to the unsplit job). Each batch presorts itself on its own goroutine.
- **Flush.** The flush merges the presorted lists (merge passes run in parallel on 4+ threads) and plans draw runs. The planner groups quads by texture and mip level without breaking overlap order. The flush then gathers vertices (in parallel) and submits them.
- **Culling.** Quads wholly off-screen are dropped when they are emitted.
- **Anti-aliasing.** AA is our own 2× supersampled target (`aaBegin`/`aaEnd`), not Ebitengine's.
- **Code that edits the quad it just added** must use `B.lastQuad()` / `B.lastGlow()`, because pieces and culling can skip a quad.

## Ebitengine fork (`third_party/ebiten`)

- go.mod `replace`s Ebitengine with this copy. `third_party/ebiten/ARCANE_PATCHES.md` lists the 6 patches:
  - DX11 uniform buffer reuse (about 80% of all allocated bytes)
  - restorable scan skip
  - command slice reuse
  - `DrawTrianglesOptions.UniformMipmapLevel`
  - `DrawTriangles32` with a zero-copy path
  - fused atlas loops
- To upgrade Ebitengine, re-apply the patches.
- The DX11 patch has only been exercised on Windows. The next PC benchmark is its first real run.

## Loading and audio

- **prefetch.go.** Floors, backdrops, the brick sheet, portraits, HUD poses and scenes are built once, by a background worker ahead of need or by the loader itself if it gets there first. The worker covers the current act and the next. All models are parsed at boot.
- **audio.go.** Sound effects render once per clip, guarded by a once, on 4 warming workers.
  - Each new world or track queues its brick sounds.
  - A clip that isn't ready starts late, from a goroutine, rather than stalling the frame.
  - Warming is off in the balance harnesses (`sfxWarm`).
- **Music.** Package `mus` holds 5 synthesised songs: groove (menu), rush / acid / abyss (acts 1–3) and warden (boss). `tools/musrender` renders them to files.

## Translations

- `tools/i18n/src/*.jsonl` has one row per string, with columns en zh ja ru es pt de.
- Rebuild with `go run tools/i18n/extract.go > tools/i18n/strings.txt && python3 tools/i18n/build.py`.
- CJK glyphs must exist in the `assets/fonts/cjk/*.ttf` subsets (`tools/i18n/subset.py`).
- Composite text is translated piece by piece (`trJoin`), and tags use `tag:` keys.
- **Every new player-facing string needs all 7 languages.**

## Waiting on the user

- **Opening cinematic: ON HOLD.** It is a work in progress and not part of any playtest (the game never plays it). The user didn't like the Mirabel redesign and will write a new brief. Status and files: `docs/cinematic/wip/README.md`. Don't put it in playtest zips or Steam builds until the user says so.

- **Roadmap step 1 (lore) is written but not yet compiled.** Go module downloads were blocked in the session that wrote it (proxy.golang.org 403), so story.go, classsel.go, progress_ui.go, game.go and the new lore.go have only passed `gofmt -e`. First thing next session: vet both targets, then screenshot `classsel`, the Lore page and the story cards in all 7 languages, and fix anything that turns up.

- **Performance plan.** `docs/PERF_NEXT.md`: Go 1.26 (Green Tea GC), PGO from the bench pprofs, Ebitengine 2.10 (Proton fix, shader precompile), frame cap and idle throttling, dual-Kawase bloom, cached distant layers. Start with section A.

- **PC benchmark of the current build.** Compare against perf4 (2560×1440, DX11). The bench files are on the PC but are an older build; copy the current exe (split as above) there first.

  | Scene | fps (1% low) |
  |---|---|
  | world2 | 287 (211) |
  | world3 | 452 |
  | world4 | 378 |
  | world5 | about 360 |
  | world6 | 400 |
  | multiball | 383 |
  | classic2 | 278 |
  | world1 | 550 |
  | boss | 660 |

  Run three configurations: plain, `-nopieces` and `-noparmerge`.
- **Music feedback.** The SFX per-material rework (#128) is on hold until then.
- **A Linux build** needs the user's yes, and may be blocked by missing X11 packages.
- **Smooth Motion check** on the PC.

## Balance notes

- Act 3 carried runs: origin spread 11.4%, 8 timeouts in the last full sim.
- Engineer runs about 20% fast: keep an eye on it.
- Gambler +25% luck, 8 gold / +45%. Storm chain +0.25. Berserker crit removed.
- Power controller: power.go (learnt per-item feed-forward).
