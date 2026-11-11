# Claude handoff, 2026-11-11

Read this first, then **`ROADMAP-50.md`** (the new 50-item list the user asked for; work it top-down) and `ROADMAP.md`. `HANDOFF.md` still describes the
renderer, the harness and the build in general.

## First thing next session: compile and test

This session could **not compile the game**. The workspace had no internet,
so the Go module cache (Ebitengine's dependencies, golang.org/x/image) was
missing, and its Go was 1.24 while go.mod asks for 1.26. Instead, every
change was checked with:

- a type checker that loads the whole `orb` package with the external
  packages stubbed out (all of the game's own code is checked, including
  calls between files; only calls *into* Ebitengine are unchecked),
  for both `GOOS=windows` and `GOOS=js -tags capture`: 0 errors;
- `gofmt`;
- the sound recipes compiled and run on their own (they don't need
  Ebitengine): every recipe renders, and the new test passes;
- `go test` for `levelgen`, `loc` and `netplay` (pass).

So on the PC, from `Source`:

```
go vet ./...
go test ./...
go build -o ArcaneOrb.exe .
```

If anything fails it will be small (a typo-level fix). Then play a run and
listen.

## What changed this session

### 1. Every pickup and moment has its own sound (`sfx_sculpt.go`)

The old power-up sound was one shared rising glass triad with a small extra
on top, and a few clips ("warp", "abilready", "coinbig", "heal") were reused
for a dozen unrelated moments. Now:

| Moment | New sound |
|---|---|
| Multi-Orb | one orb wobbles, three bubbles pop out and rattle away |
| Expand | rubber band stretch ("ooo-aah") into a spring boing |
| Slow | bell chord sags like a tape stopping, clock ticks spread out |
| Fire Orb | match strike, flame "fwoomp", crackle |
| Laser | capacitor charge, two arming blips, latch clack |
| Barrier | stone grinds up, crystal grows, locks with a "chunk" |
| Glue | fat bubbles, resonant squelch, unsticking pop |
| Magnet | field hum, filings skitter in faster, a ding |
| Shrink (bad) | balloon letting go, sad boing; no reward bells |
| Haste (bad) | nervous accelerating ticks, small alarm |
| Heart caught | heartbeat blooming into a warm chord |
| Heart falls / Rune cache falls / cache opened | own cues |
| Next wave | two-note horn call and drum |
| Arcanist attune, Engineer overclock, Stormcaller static full | own cues |
| Gambler jackpot, Prismatic Nova, Event Horizon | own cues |
| Golden critter, critter popped, thief caught, bomber arrives | own cues |
| Brick teleports, elite core wakes, paddle reforms, curse lifted, duel start | own cues |
| Reward cards | each rarity has its own instrument (wood, kalimba, glass, choir, brass, dark gong) |
| Omen challenge met / failed | wax-seal stamp and two bells / a soft "aww" (failure used to be silent) |
| Draught of Warding, rest "two barriers" | the Barrier sound; rest "regain a life" uses the heart sound |

Recipes in `sfx_sculpt.go` override same-named ones in `sfx.go`.
`sfx_sculpt_test.go` renders every recipe and fails if the power-up sounds
drift back into sounding alike (spectral similarity check; the old set
fails it, the new set passes).

**Listen first:** `../Sound-Preview-2026-11-11.zip` has all of them as WAVs.

### 2. Hero quips (roadmap step 2, `quips.go`)

- The lines from `docs/quips.md` appear in a comic speech bubble beside the
  HUD portrait, each with a short synthesised voice babble in that hero's
  register (Oriel warm, Brakka gruff, Vell reedy, Rivet clipped, Sol
  sliding, Cinder bouncy, Zephyr breathy).
- Moments: pick (a toast on the origin screen), run start, long combos,
  Guardian defeated, legendary card (said when play resumes), last life,
  and the win (on the final Guardian).
- **Combo commentary (the user asked for this specifically):** each hero
  has six combo lines in their own voice: three that take turns at a
  50-hit combo (45 s cooldown), two at 100 and one at 150. The table is in
  `docs/quips.md`; the code is `heroComboQuips` / `comboQuip` in quips.go.
- At least 8 s between any two lines; the last-life and win lines always
  get through.
- Settings > Audio > "Hero quips" turns them off (`NoQuips`, default on).
- Never in duels, sims, the trailer or benchmarks.
- **Needs a look in game:** the bubble sits at (186, 100), below the
  gold/barrier plate. Check that it doesn't cover anything at 16:9 and
  ultrawide.

### 3. Text with more charm (all 7 languages)

- The coach tips, the map legend, the chamber-select side panel, empty
  states (leaderboards, duel rooms, Codex, damage recap), toasts and a few
  floating combat words were rewritten in the warm, corny storybook voice.
  The full list is the top of `tools/i18n/src/20.jsonl`.
- Found and fixed: the chamber-select panel (added on 11-10) had 15
  untranslated strings. They are rewritten and translated now.
- New `tools/i18n/glyphcheck.py`: lists Chinese/Japanese characters the
  tables use that the subset CJK fonts lack. The full Noto fonts weren't
  available, so every new CJK line was worded to use only glyphs the fonts
  already have (glyphcheck: 0 missing). If you re-run `subset.py` with the
  full fonts later, that restriction goes away.
- Still untranslated on purpose: the opening cinematic's strings (it is on
  hold) and debug overlays.

### 4. QOL

- Alt-tabbing away mid-chamber now pauses the game (not in duels,
  benchmarks or scripted captures).
- Settings > Audio > "Mute while the game is in the background" (off by
  default). Music keeps streaming silently, so the song doesn't restart.
- The quip bubble's layout is worked out once per line, not every frame.

### 5. Performance: sound warm-up about 30% cheaper

The 11-10 profile flagged SFX warm-up as CPU work when a world starts.
In `sfx.go`:

- The reverb in `finish` now runs each comb and allpass filter over the
  whole clip in turn instead of interleaving them per sample. The sums
  happen in the same order, so the output is **bit-for-bit identical**
  (checked: all 153 clip variants hash the same).
- The modal resonator (`modes`, behind every bell, marimba, glass and
  metal hit) uses a rotating phasor instead of a `sin` and an `exp` per
  sample. Not bit-identical, but within 54 dB SNR of the old output
  (float rounding plus peak normalisation): inaudible.
- `BenchmarkWorldSoundWarm` (sfx_sculpt_test.go): about 490 ms to 350 ms
  per op here. Re-measure on the PC.
- The sky shader is now created and run once into an 8×8 scratch image
  while the main menu is up (`warmShaders`, skyfx.go), instead of on
  entering the first chamber, where the 11-10 profile showed 7.3 ms of
  shader creation inside the chamber-entry hitch. Check the first
  chamber entry in a trace.

### 6. From ROADMAP-50 so far

- #2 Anti-stall (`antistall.go`, test in `antistall_test.go`).
- #44 Hero voices volume slider (Settings > Audio).
- **Music per chamber (the user asked for this, Magic Ball style).** Each
  act now has a family of three songs that take turns from chamber to
  chamber: the act song for the 1st, 4th, 7th chamber, then two new
  siblings. Siblings keep the act's key (the musical brick SFX still fit),
  stay within a few BPM and use the same instruments, with their own chord
  loop, groove, bass and lead:
  - Act 1 (A minor): Orb Rush, **Lantern Steps** (136), **Kite Weather** (142)
  - Act 2 (G minor): Acid Cascade, **Copper Gears** (144), **Steam Choir** (148)
  - Act 3 (B minor): Abyss Engine, **Crystal Hollow** (150), **Starfall Engine** (154)
  Code: `mus/song_chambers.go` (one shared arrangement, six configs),
  `trackSongs` / `trackKey` / `chamberTrack` in audio.go, the track pick in
  game.go. Previews: `../New-Chamber-Music-2026-11-11.zip`. The mus tests
  pass here; levels sit within about 1.3 dB of the act songs.

## Where the code lives now

- GitHub: branch `claude/sync-2026-11-11` of ginograziano24-cpu/Arcane-Orb
  holds the PC copy (Nov 6-10 work, which had never been pushed) plus this
  session. Not merged into `main` yet; ask the user.
- The PC folder `Downloads\RoguelikeBreaker\ArcaneOrb-Updated\Source` has
  the same files.
- `third_party/ebiten` is upstream Ebitengine v2.10.6 plus the Arcane
  patches (image.go, run.go, internal/mipmap, internal/ui and the
  arcane_shader_windows files); everything else matches upstream.

## Files

New: `sfx_sculpt.go`, `sfx_sculpt_test.go`, `quips.go`,
`tools/i18n/glyphcheck.py`, `tools/i18n/src/20.jsonl`, this file.

Changed: ability, achievements, audio, chamber_projection, classsel,
coach, critters, critters2, curses, duel, duel_ui, effects, game, help, hud,
hudportrait, leaderboard, lives, omens, passives, play, progress_ui, recap,
save, scrolls, settings, sfx, waves (.go); docs/quips.md;
assets/i18n/*.json; tools/i18n/strings.txt.

Originals of every changed file: `../Backup-before-Claude-2026-11-11/`.

## Open questions for the user

- The main menu has a **"Watch opening"** button (added by an earlier
  session). Memory says the opening is on hold and should stay out of
  playtests. Keep it or hide it?
- The quip lines are still draft 1. Change any you like in
  `docs/quips.md`.

## Next steps (in order)

1. Compile, test, play; listen to every new sound in context.
2. Check the quip bubble placement and frequency in a real run.
3. Continue the roadmap: step 3 chamber select (B2 approved), then cameos.
4. Performance items still open from `../VISUAL-PERFORMANCE-20.md`
   (ETW presentation trace, Present wait, weaker hardware, full-run memory).
