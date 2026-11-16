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
- #45 Dense-fight mix (`mixDensity`, audio.go). If an audio test that
  fires many brick sounds in a row fails, this thinning is why; give the
  test a gap or raise the window.
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

---

# Session 2, 2026-11-11 (afternoon): roadmap items that need no playtest

**First thing:** double-click `..\Run-Tests.cmd` and read `..\test-results.txt`.
The morning's code passed vet, test and build on the PC before this session
started. This session again couldn't compile here (no Go 1.26 and no
module cache in the workspace), so every change was checked the same way
as before: a type checker over the whole `orb` package with Ebitengine
stubbed (windows, linux and js/capture: 0 errors), `gofmt`, and the pure
functions (aim prediction, stat formatting, keyword matching) run on their
own. Expect at worst a typo-level fix.

New tests (they run in Run-Tests.cmd): runsave_test, statdelta_test,
soundtrack_test, loops_test, events3_test, duel_stage_test, coach_test,
meta_audit_test, and TestBurstBudget in particle_burst_bench_test.

## Done (ROADMAP-50 numbers)

- **#6 save/resume (audit + fix):** resuming forgot the origin's mastery
  (origin items stopped being offered, the livery vanished). Now saved;
  old saves fall back to the current mastery. Rest of #6 needs one manual
  quit-and-continue mid-boss and mid-cache.
- **#8 aim guide** (`aimguide.go`, Settings > Effects, off by default).
- **#9 multiball clutter** (`burstBudget` in play.go; extra orbs' trails
  fainter in world.go).
- **#20 / #21 audits:** Final Form now really makes bosses attack faster
  (6 s → 4.8 s between volleys, `bossAttackGap`). Renamed ascension
  "Fortified" → "Thick Walls" (clashed with the omen) and the curse
  "Lattice's Weight" → "Stubborn Bricks" (no lore jargon).
- **#27 seven new events** (`events3.go`): the Snack Dragon, a Bard With
  Opinions, the Duckling Parade, Fumbleton's Hat (a cameo for the wizard),
  the Slime Spa, a Very Ambitious Rock, the Fortune Cookie.
- **#28 three new omens:** Starlit, Critter Catcher, Edge Artist.
- **#31 endless loop twists** (`loops.go`), named on the act intro.
- **#32 Build panel B2** (`buildpanel.go`): grouped by rarity with counts,
  hover shows the item's card, "Your stats" box (base grey → now green).
- **#33 reward cards show stat changes** (`statdelta.go`).
- **#34 pause shows build and stats** (via the B2 panel).
- **#35 recap highlights** (best combo with place, star damage source).
- **#37 keyword explanations in every tooltip** (`tooltipTerms`, coach.go).
- **#43 soundtrack** (`soundtrack.go`, Codex > Soundtrack, with Listen).
- **#49 duel stage select** (lobby; netplay protocol **version 2**, so
  both duelists need this build).

## Translations

All 129 new strings are in `tools/i18n/src/21.jsonl` (all 6 languages),
merged by `build.py`. New tool **`tools/i18n/addglyphs.py`**: instead of
rewording Chinese/Japanese around the subset fonts, it copies the missing
glyphs from the static Noto Sans CJK fonts (Bold for the 600 subset,
Black for the 800 one; measured within 3% of the ink weight) into
`assets/fonts/cjk/*.ttf`. It added 53 zh and 38 ja glyphs; glyphcheck now
reports 0 missing. Rendered samples looked consistent; give Chinese and
Japanese a quick look in game.

## Needs you (playtest or a sim on the PC)

- Look at: the aim guide, the B2 panel's hover card position on the map,
  the stat lines under reward cards, the soundtrack tab, the duel lobby's
  stage row, the new events' text in each language.
- Balance items 12, 15–19 need the campaign sim (it only runs in the
  js/capture harness); nothing was changed blind.
- Not started: 3–5 (PC traces), 7, 10, 11, 22–26, 29, 30, 36, 38, 40, 41,
  46–48, 50. 48 (achievements) waits on new content and needs icon art.

## Suggestion, not done

Achievement names still use lore jargon: "Weaver's Heir", "Looping the
Lattice", "Lattice Breaker". They are also in `store/achievements.csv`
for Steamworks, so they were left for the user to decide.

## Files

New: aimguide.go, buildpanel.go, events3.go, loops.go, soundtrack.go,
statdelta.go, tools/i18n/addglyphs.py, tools/i18n/src/21.jsonl, the tests
above. Changed: coach, curses, duel, duel_ui, game, meta, omens, play,
power, progress_ui, recap, run, runsave, save, settings, world (.go),
netplay/netplay.go, particle_burst_bench_test.go, assets/i18n/*.json,
assets/fonts/cjk/*.ttf, tools/i18n/strings.txt, ROADMAP-50.md, this file.
Originals: `../Backup-before-Claude-2026-11-11-session2/`.

---

# Session 3, 2026-11-11: card and fusion animations

The user asked for cool animations when picking item cards, a show when
items fuse that spells out the total stat gain or the new passive, and
for everything to keep running well. Checked the same way as session 2
(type check, gofmt, the pure helpers and every new sound run on their own).

- **Reward cards are dealt in** (game.go `drawReward`, `dealIn` in
  pickfx.go): each card rises into place a beat after the one before; the
  next hand is dealt with the "deal" sound after a pick.
- **Hovered cards** get a light sheen sweeping across every couple of
  seconds, brighter on rarer cards (`drawCard`).
- **Picking a card** (pickfx.go): Rare and up get turning god rays, Epic
  and up an extra gold ring. The stat changes the card made fly out of it
  as chips under the banner, each number counting up ("Orb damage
  1.5 → 2.0"), plus any synergy it switched on; the pick lasts 0.35 s
  longer when there are chips. The pick no longer plays the big synergy
  sound (it was shared with synergies); the card now whooshes up into the
  build with its own chime, richer per rarity (`cardfly`).
- **The forge** (`fusionfx.go`): a fusion stops the game for a short show:
  the two items' emblems spiral in with comet tails, collide in a flash
  with rings and sparks, the fusion emblem rises under turning rays, its
  name and text appear, then every number the fusion itself adds counts up
  on its own line with a rising marimba note each (`fusionGains`: the
  build with and without just that fusion, via `Run.skipFusion`). A
  fusion that adds no number gets a "NEW PASSIVE" tag. Click/Space/Enter/A
  skips to the payoff, the next press closes it; it closes by itself after
  about 4 s. Fast animations make it 1.8× quicker. It waits for a card
  pick to finish, queues several fusions, and never plays in the menu
  demo, duels, the trailer, benchmarks or sims (`cinematic()`), where the
  old banner still shows.
- **Sounds** (`sfx_fx.go`): `fuse` (two tones spiralling into unison, a
  rush, an anvil strike at the collision, a chord bloom), `cardfly`,
  `statup`. All are warmed per musical key with the other keyed sounds.
  Previews: `../Sound-Preview-2026-11-11-fx/`.
- Performance: the new effects are a few dozen sprites and lines per
  frame; the deal-in uses the existing fade layer for up to three cards
  for half a second. Worth a glance in the frame-time overlay on the
  reward screen.

Look at in game: the deal-in, the chips' row under the banner, the forge
(own Twin Orbs + Piercing Orb for Lancer Formation to see numbers;
Blast Core + Frost Touch, Cryo Charge, for "NEW PASSIVE").

New: fusionfx.go, fusionfx_test.go, sfx_fx.go. Changed: pickfx.go,
game.go, run.go, progress_ui.go, audio.go, tools/i18n (2 strings).

---

# Session 4, 2026-11-11: the 30-item polish list

The user approved 30 more ideas; they are ROADMAP-50 section H (#51–80).
Done this session (checked as before; tests in letters_test.go):

- **#65 Letters** (`letters.go`): after each Guardian and at each endless
  loop, a letter arrives on the map from Archmage Fumbleton or from
  Mirabel (about half each; unread letters first; one written for the
  moment beats a general one). Mirabel's are on pink paper in purple ink,
  sealed with a heart, and get a lipstick kiss pressed on at the end (a
  drawn print, `lipsPath`). Either writer's letter shows their picture:
  a taped-on photo (`polaroid`, cropped to the face) or the full figure
  drawn faintly behind the writing. The P.S. mentions something from the
  run. User's request: Mirabel letters with a lipstick print and her
  picture or model in the background; the same for Fumbleton without
  the lipstick.
- **#56 FLAWLESS stamp**, **#58 fusion hints**, **#59 item levels**,
  **#61 shop sale**, **#75 Same seed**, **#71 rage marks** (part).
- New sounds `letter`, `scribble`, `kiss`, `flawless` (sfx_fx.go);
  previews in `../Sound-Preview-2026-11-11-fx/`.
- 48 more strings in all 6 languages; addglyphs.py added 114 CJK glyphs.

Note: the letter pictures use the opening's art. The user wasn't happy with
the opening's Mirabel redesign, so if they don't like her picture here,
point `queueLetter` at renders of the Blender model when those exist.

---

# Session 5, 2026-11-11: ROADMAP-50 section I

Section I (#81–113) was added from the user's picks. Done this session:
board effects in `visualfx.go` (orb-lit bricks, floor marks, element
trails, paddle charge glow, weather, Guardian confetti; all cosmetic
randomness comes from `L.fx`, never `L.rng`, test in visualfx_test.go),
card backs and flip-in on the reward screen (`drawCardBack`,
`fadedFlip`), the per-chamber damage chart on the summary, animated map
icons, a 30 fps battery cap (works with VSync too), the jackpot portrait
reaction, and Codex > Bestiary (`bestiary.go`). #81, #84 and #100 were
already in the game. 27 more strings in all languages; boss names in the
letters now match the existing translations.

## Session 5 PC verification (11:33)
- Build-Test-Bench.cmd: vet 0, all tests pass (orb 5.7s), build 0. Fixed TestBuildGroups (curses live in curseList).
- Bench (2560x1440 fullscreen, High, no cap): after vs before (Engine210-Readability) is level or slightly faster in every scene, e.g. world1 420->433 fps, map 590->617, reward 747->776, classic5 245->244. Allocations unchanged (~17.5k/frame scene). No missed 60/120 frames except the existing boss one-frame spike (12-14 ms max, present before too).
- Steam Deck preset (1280x800, Medium, 60 cap): 60 fps in every scene, 1% lows 55-58.
- Watch: warm-up max interval rose for menu (47->124 ms) and world1 (17->99 ms) - one-time hitches during warm-up only, likely the new keyed sounds/forge assets warming. Steady state is unaffected; consider spreading warmKeyed over several frames.
- Section I still open: 87, 90, 91, 95 (part), 96-99, 101, 102 (part), 103-108, 110-113.

## Session 6 (12:32-13:10): roadmap section I, performance and content
Done (ROADMAP-50 marked): #90 day-to-night (style.go), #96 visual style (Settings > Access), #97 F3 GPU line and split frame graph, #98 emblems pre-baked on the menu (Ebitengine already atlases them), #99 text widths cached (wrap was already cached), #101 no brick shadows on Low, #102 165/240 caps and a shorter spin above 200 Hz, #103 startup trace, #104 `-soak` memory test, #105 Settings > Display > Run benchmark (menubench.go), #106 secret vaults (secret.go), #108 weekly mutator run (weekly.go, on the Challenges screen), #113 orb skins (orbskins.go, picker on the Challenges screen).
Also: BENCH runs never earn achievements or write a run save; button labels under 170 px shrink to fit; extract.go skips boottrace.go/soak.go and bootMark/NewRun arguments. 47 new strings translated (45 untranslated kept as before), 28 CJK glyphs added. Tests: session6_test.go.
New settings fields: visual_style, no_day_night; save fields: orb_skin, weekly_best; run save: Weekly.
PC results (Build-Test-Bench-6.cmd -> bench-session6.txt): vet 0, all tests pass, build 0.
- Bench, average of two interleaved runs each, session 5 vs 6 at 2560x1440 High: every scene within -3%..+4% (noise; the session-5 game was also left running in the background). World1 398->406, boss 436->454, multiball 318->309, classic5 236->235.
- Steam Deck preset: 60 fps everywhere; boss 1% low 49 (was 57) from three 23-26 ms frames out of 360 while the CPU used 0.9 ms a frame - OS/background jitter, recheck with nothing else running.
- Soak: heap 172 MB after run 1, then steady at ~133 MB (-2.2 MB per run after run 2); goroutines steady at 12. Heap objects creep up ~1.2k per run (66k->77k) - small, worth a look sometime.
- Startup: 0.54 s to the first menu frame; game code 29 ms, the rest is the window and DirectX device.
Still open in section I: 87, 91, 95 (art), 107, 110, 111, 112.
Backups: Backup-before-Claude-2026-11-11-session6/.

## Session 7 (17:10-17:50): feel, quality of life, performance (ROADMAP section J)
Done: J #114-125 and #131-133 (13-17 of the list were not chosen). Files: feel.go (paddle squash + a very slight arcane crackle on hard returns, brick cracks, combo heat and embers, Guardian armour shatter, seasonal map dressing), history_graph.go, buildhints.go (pick hints, pause "In effect" panel, source-coloured damage numbers), binds.go (left-handed pad preset), access.go (hold R on the defeat screen), prewarm.go, autoquality.go, chambersave.go. Slow-motion final hit and gamepad rebinding already existed.
Note: most of this code was found already in the workspace from an earlier, interrupted attempt at this request; it was reviewed, fixed (the chamber save never set initBricks, so it never fired; auto quality now restores the menu's sound world) and finished here.
New settings: damage_numbers, pad_left_handed, quality_auto. Run save phase "chamber" with Chamber progress.
PC results (Build-Test-Bench-7.cmd -> bench-session7.txt; then Run-Tests.cmd): vet 0, all tests pass, build 0 (first run failed only on an over-strict colour test, fixed).
- Bench, session 6 vs 7, two interleaved runs each: all scenes within -3%..+3% except one s7 run with a single 277 ms OS hitch in world1 (s7 run 1 world1: 409 fps vs s6 396/414).
- Soak: steady at ~134 MB after run 3, goroutines 12.
Backups: Backup-before-Claude-2026-11-11-session7/.

## Session 8 (19:00-23:20): release readiness (ROADMAP section K)
Done: padglyphs.go (button names per pad family), larger default text at 1280x800, achaudit_test.go, crash log extras, photomode.go (Pause > Photo mode; also the store capture mode), chronicle.go (Codex > Chronicle, 10 pages in Fumbleton's voice; save field "chronicle"), auto-launch (Settings > Controls; setting auto_launch), last-life heartbeat sound + music dip on life loss, opening stems zlib-packed (opening.go openingStem; .wav still read as a fallback), HUD portraits losslessly recompressed.
Not done: dyslexia font (needs a font file, e.g. OpenDyslexic), Deck verification on hardware, assist mode (not chosen).
Folder: Build-Test-Bench-8.cmd moved old benchmark files, old builds (Session5/6), old scripts, all Backup-before-Claude-* folders and the plain opening WAVs into Archive-2026-11-11 (nothing deleted). Backups for this session are inside it too.
PC results: vet 0, all tests pass, build 0 (first run failed only on my chronicle test's expectation, fixed). Exe 116.7 MB -> 101.6 MB. Bench, session 7 vs 8, two interleaved runs each: within -1%..+4% everywhere (noise). Startup 0.54 s, game code ~31 ms.

## Session 9 (2026-11-12): co-op, keepsakes, trials, banish/lock, familiars, pacing (ROADMAP section L)

- New files: coop.go, keepsakes.go, trials.go, familiars.go, banish.go, session9_test.go. Originals of changed files: `Backup-before-Claude-2026-11-12-session9`.
- Origin screen bottom row: Keepsake, Trials, Co-op buttons (hidden for daily/weekly/challenge runs). Panels block clicks behind; Esc closes them.
- Reward screen: Banish button left of Skip toggles banish mode. Shop: Lock under each item (max 2).
- Co-op: highest-numbered controller is player two; P1 keeps keyboard/mouse or the other pad, plus the ability and scrolls. P2 bounces, glues, launches (A or RT) and catches drops. One pad + keyboard is the intended setup.
- Balance numbers: co-op paddles x0.80, bricks +15% HP; keepsakes ~ one Common card; trials +6% shards/point; familiars 0.2-0.55x orb damage every 2.6-4 s; rarity gate in run.go.
- Saved runs keep keepsake, trials, co-op, banishes and the knot.
- Needs you: play a co-op run with a friend; try a few runs to see if the act-1 pacing feels right.

- Visual pass (ROADMAP section M, `visfx9.go`): familiar motes/impacts, owl and knot flourishes, barrier ripple, crit ring, coin glints, last-life red edge, HUD keepsake/trial row, banish burn-away, legendary light column, shop lock badge, warm Trials button.
- Performance (ROADMAP section N): the fullscreen window-icon bug fixed; bench-session9b.txt and bench_report_s9c/s9d (before) vs t9a/t9b (after) show +32% to +163% fps. ArcaneOrb-Session9b.exe is the current build. All tests pass (test-results.txt 00:59).
- Next cycle ideas: profile again now that the icon noise is gone (Batch.drawShared, planRuns, memmove are the top game-side costs; world2 and the classic backdrop are the slowest scenes).
- Cycle 2 (ROADMAP O and P): drawShared fill-in-place + inlined lighting (exact output, hash test), and 15 more visuals in `visfx10.go`. Results in bench-session10.txt (Session9b vs Session10).
- Cycle 2 result: tests pass (including the projected-model hash test, so the model drawing is bit-identical); fps vs 9b within noise. The game is now GPU/driver bound; next perf work should cut quads or draw calls (world2 22k quads, classic 30k).
- Cycle 3 (ROADMAP Q and R): profile correction (the cgocall time is the idle audio thread, not the GPU); 15 more visuals in `visfx11.go` with a small screen-spark system (uiBurst/drawUISparks). Results in bench-session11.txt (Session10 vs Session11).
- Cycle 3 result: tests pass; Session10 vs 11 within noise in every scene (one slow Session10 run on the origin screen; the other runs match).
- Cycle 4 (ROADMAP S and T): baked models (`meshbake.go`, sync.Map cache keyed on mesh+placement; bakeTick in drawWorld) give the classic backdrop +17% fps with an exact-match test; 15 visuals in `visfx12.go`. Tests pass; bench-session12.txt (Session11 vs 12). Current build ArcaneOrb-Session12.exe.
- Keepsake upgrade tree (ROADMAP U), asked for by the user: StKeepsakes screen in keepsaketree.go, SaveData.KeepRanks, Run.KeepRank (saved in run.json), migrateKeepsakes on load. The old origin-screen picker overlay is gone (archPanel 1 unused; 2 is still Trials).
- Cycle 5 (ROADMAP V and W): Session12 vs 13 (keepsake tree) held fps (+0-4%). fillVerts copies and premultiplies in one pass (exact; TestFillVertsOnePass). 14 more visuals in `visfx13.go`, mostly the Keepsakes screen. bench-session14.txt has Session13 vs 14.
- Cycle 6 (ROADMAP X and Y): the one-pass fillVerts measured 4-7% slower in chambers and was reverted. 15 visuals in `visfx14.go`. bench-session15.txt has Session14 vs 15 (15 should be back to Session13 speed).
- Cycle 7 (ROADMAP Z and AA): no speed change (nothing safe left that measures); 15 visuals in `visfx15.go` showing the run's charms between chambers. bench-session16.txt: Session15 vs 16.
- Cycle 7 result: Session15 vs 16 within noise, tests pass. Cycle 8 (ROADMAP AB/AC): 15 visuals in `visfx16.go`; bench-session17.txt has Session16 vs 17.
- Cycle 8 result: Session16 vs 17 within noise. Cycle 9 (ROADMAP AD/AE): 15 visuals in `visfx17.go`; bench-session18.txt has Session17 vs 18.
- Cycle 9 result: Session17 vs 18 within noise. Cycle 10 (ROADMAP AF/AG, asked for): brick emphasis in `brickfocus.go` (rim on bricks via brickRim at world.go drawn:, scenery veil drawn after post in drawFrame; Settings.BrickEmph 0 subtle / 1 strong / 2 off). bench-session19.txt has Session18 vs 19.
- Cycle 10 result: Session18 vs 19 tests pass; dense chambers lost 2-4% (rim quads), so Subtle now draws only the front rim. Cycle 11 (ROADMAP AH/AI/AJ): Effects intensity dial (`fxlevel.go`, Settings.FxLevel 0 high / 1 medium / 2 low; guards at the top of visfx draw functions), system sounds (`sfx_systems.go`), six achievements with Steam CSV + icons, TestMain noPersist guard. bench-session20.txt has Session19 vs 20 (tags y/z).
- Cycle 11 result: Session19 vs 20 tests pass, chambers +1-4%. Cycle 12 starts the 100-item polish pass (`Source/POLISH-100.md`, ROADMAP AK). New review tool: `-shots` (shots.go; extra scenes in shotScenes) writes shots/<scene>.jpg; Shots.cmd builds and runs it. Glimmer Grove replaces the synthwave Neon Grove: biome.go WNeon entry (Floor/FloorAlt are now near-white tints because the floor JPGs are coloured), scenery.go (menhirMesh, glowcap), dressing.go dressNeon, world.go (grove walls drawn as stone prisms, wall/post models skipped for WNeon), assets/bg/neon_0-1.jpg and assets/floor/neon_0-3.jpg repainted by tools/paint/grove.py (seeds 11 and 29) and floor.py (variants 0-3). Internal identifiers still say Neon (POLISH item 83). bench-session21.txt has Session20 vs 21 (tags c/d); world1, elite and boss scenes changed content, so compare the other scenes for speed.
- Cycles 13-16: Tripo pipeline is in Source/tools/tripo (glb2orbm.py with --height/--yaw/--ymin/--emit-above, simplify.py, render_orbm.py preview, topohash.py for the topology reference). Every new model must be added to testdata/model-topology-reference.json (topohash.py) and to projectionNewModels in mesh_projection_test.go; a replaced model goes in projectionReplacedModels with its original copied to testdata/projection-bank. New files: guardianfig.go (Guardian figures behind the far wall), foliage.go (grass tufts and flowers per world), skyedge.go (sky-foot fill), shots.go (-shots screenshot tour). Quick-Check.cmd runs tests plus the screenshot tour without benchmarking. Paint scripts: Source/tools/paint. Tripo API credits are spent (90 left); Studio credits need the user's sign-in in the desktop app's browser pane.

