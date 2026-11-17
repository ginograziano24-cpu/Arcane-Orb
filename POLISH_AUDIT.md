# Arcane Orb — Polish Audit (Phase 1)

Written 2026-11-17 for docs/POLISH_PASS.md. This phase made no code changes.

**How it was done.** I read the game loop (`game.go`, `play.go`, `feel.go`, `quips.go`, `hud.go`, `save.go`, `runsave.go`, `settings.go`) and ran targeted scans across the ~53k lines of game code. `third_party/` and `tools/` were left out.

**What it can't cover yet.** Runtime numbers (fps, 1% lows, allocations per frame) need the PC, because Go 1.26 can't be installed in the cloud workspace. Those are measured at the start of Phase 2 with the existing bench (`-bench`, the F3 overlay, pprof).

**How it's ranked.** By player impact, highest first. Each item says where it lives and what to do.

---

## 1. Character: quips repeat (highest impact on "feels alive")

- `quips.go` holds **one line per moment per hero** (`heroQuips`, 7 moments × 7 origins). There are 6 extra lines for long combos.
  - Every life lost, item pickup or boss kill plays the *same* sentence for that hero.
  - Across a run, players will hear "Okay. Think. Think." a dozen times.
- The cooldowns are good and already in place: 8 s between quips, 45 s for combo lines, 3.6 s on screen.
- The lines themselves have charm ("Drinks are on Oriel.", "This was not in the syllabus.").
- **Fix (Phase 4):**
  - Write 4–6 lines per moment per hero.
  - Pick randomly without repeating until the pool is used up.
  - Add a per-moment cooldown so the same moment can't fire twice in a row.
  - Add a couple of new moments: near miss (orb saved at the last frame), first-chamber greeting and a rare idle line.
  - The keyword lists in `docs/quips.md` are the starting point.

## 2. Game feel: impact values are scattered, no single tuning file

- Hit-stop exists but is hard-coded in four places: `play.go:1425` 0.045 s, `:2739` 0.09, `:2744` 0.035, and `lives.go:132` 0.12.
- Paddle squash, crackle, camera kick (`camKick`) and orb impact (`orbImpact`) each have their own magic numbers in `feel.go` and `play.go`.
- The brief asks for **one tunable config**. That's also what makes playtest tuning fast.
- **Fix (Phase 3):**
  - Add a `feelcfg.go` struct with every value: hit-stop per event, shake amplitude and decay, camera punch, trail length by speed, spark counts, squash amount, combo pitch step.
  - Optionally load overrides from `feel.json` next to the exe in debug builds, so values can change without a rebuild.

## 3. Game feel: missing juice

- **No controller rumble anywhere.** Ebitengine has `ebiten.VibrateGamepad`, and controllers are otherwise well supported (148 gamepad references). A short, light buzz on paddle hits, boss hits and life lost is cheap and very noticeable on Steam Deck and pads. It needs a settings toggle next to screen shake.
- The combo already heats up visually: white → gold → fire, with embers (`feel.go`). **Check that the audio climbs too.** The brief asks for a rising pitch per combo step; confirm this in `sfx_systems.go` during Phase 3.
- Debris per material is still open: POLISH-100 item 30 (stone chips, glass, metal sparks). Heavy (armoured or heated) bricks should get their own crunch and a slightly longer hit-stop.
- Ball trail by speed exists (`orb_motion_trail.go`). Per-origin colour is still open (POLISH-100 item 27).

## 4. Performance: per-frame garbage in the hottest loop

- `play.go collideBricks()` allocates the `splash` and `hits` slices **on every physics substep, for every ball** (`var splash []*Brick` and `append`).
  - Substeps scale with speed (`steps = ceil(s*dt / (0.45 r))`).
  - In a late-game multiball chamber that's hundreds of small allocations per frame. That's GC pressure, and the 1% lows suffer.
  - **Fix:** level-owned scratch slices reused with `[:0]`.
- `L.near()` allocates a new slice per call (5 call sites, all blast and lightning paths). **Fix:** pass a scratch slice in.
- `hud.go` builds about 10 strings per frame with `fmt.Sprintf`: combo, frenzy timer, ability timer, class counters and the FPS readout. **Fix:** cache each string keyed by its integer value (there's already a `textcache.go`) and rebuild only when the value changes.
- Fire trail: `L.Parts = append(...)` every frame per ball. Fine if `Parts` is reused, but **confirm in pprof** that its capacity stabilises.
- **Already good:**
  - Substepped swept collision, so fast orbs can't tunnel.
  - `dt` clamped to 1/30.
  - Frame pacing, auto-quality and dynamic resolution.
  - Audio caches and shader prewarm.
  - `ReadPixels` and `At()` only appear in load-time, screenshot, photo-mode and capture code, never in the game loop.
- The deeper performance plan in `docs/PERF_NEXT.md` (Go 1.26, PGO, Ebitengine 2.10, frame cap, idle throttle) still stands. Phase 2 does A1–A3 and B1–B2 from it.

## 5. Stability and bugs to verify in Phase 2

- **Variable timestep.** `ebiten.SetTPS(ebiten.SyncWithFPS)` ties Update to the render rate.
  - Physics is substepped, so it's safe from tunnelling.
  - But anything using `x += k*dt` lerps (for example `PaddleW += (tw-PaddleW)*min(1,10*dt)`) behaves slightly differently at 30, 60 and 240 Hz.
  - **Check:** run the bench at 60 and at 240 and compare a recorded chamber, or switch gameplay to a fixed 120 Hz accumulator (render stays free).
- **Stuck orbs.** `fixAngle` keeps the vertical component at ≥30% of speed, which stops horizontal loops. `antistall.go` exists and has a test. Keep the soak test (`soak.go`) in the Phase 6 run.
- **Saves.** `runsave.go` is versioned (v2, rejects unknown versions). `save.go` (meta-progress) has no version field. **Add one before release** so future changes can migrate instead of resetting progress.
- **Multi-ball loss.** `lostReal` is computed and then discarded (`_ = lostReal`). Harmless, but it's leftover code worth cleaning.

## 6. Clarity and UX

- **First two minutes.**
  - `coach.go` and its test exist.
  - Check that a new player learns aiming (the hit position on the paddle sets the angle) without reading anything.
  - The aim guide (`aimguide.go`) is a good candidate for the first chamber only.
- **Settings.**
  - Screen shake has a toggle (`settings.go:200`). Rumble needs one (item 3).
  - The Effects tab has 12 rows (POLISH-100 item 38). Split it into Effects and Gameplay.
- **Accessibility present:**
  - Colour-blind references in 4 files.
  - Subtitles for voice.
  - Auto-launch.
  - Keyboard and controller binds (`binds.go`).
- **Text size** at 1280×800 (Steam Deck): POLISH-100 item 47, still open.

## 7. Pacing and balance (needs playtest, flagged rather than guessed)

- Chamber budgets come from `chamberBudget` and `pacedLevel` (`play.go:2776+`). `balance_run.go` exists for simulated runs.
- **Recommendation:** run the balance simulator across all 3 worlds and record chamber clear times. Flag any chamber whose median runs above about 90 s or below about 25 s. This is a taste call, so the numbers come to you before anything changes.

## 8. Audio

- The sound design is sculpted, not stock (`sfx_sculpt.go`, Freeverb, pitch-swept subs). Good.
- **To check in Phase 3:** that brick-break sounds vary per material and pitch-randomise slightly (±3%) so long chains don't machine-gun.
- **Music transitions:** `soundtrack.go` has envelope tests. Confirm a crossfade between chamber and shop or map.

## 9. Steam readiness

- `Layout()` multiplies by `DeviceScaleFactor` (high-DPI aware). Fullscreen and windowed toggles exist.
- Steam achievements are wired through `steam_api64.dll` (`steam_windows.go` SetAchievement / GetAchievement). Leaderboards are in `steam_lb_windows.go`.
- **Debug leftovers:** `fmt.Fprintln(os.Stderr)` only appears in bench flag parsing. The FPS readout on the HUD is behind a setting. Nothing blocking.
- **Linux / Steam Deck native build:** blocked on Ebitengine 2.10 (no Cgo needed). Covered by PERF_NEXT A3/A4.

---

## Order of work

| Phase | Items |
|---|---|
| 2: stability and performance | 4, 5, then the PERF_NEXT A1–A3 and B1–B2 toolchain wins. Bench before and after on the PC. |
| 3: game feel | 2 (config first), then 3 |
| 4: character | 1, plus hero idle and reaction animation hooks |
| 5: clarity and UX | 6 |
| 6: verify | Re-run this audit, `go vet`, `-race` build, soak test, bench |

**Taste calls for Gino:** combo pitch curve, rumble strength, chamber length targets (item 7), and the new quip lines (you'll get them to read before they ship).
