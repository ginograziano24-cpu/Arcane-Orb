# Handoff for another AI assistant (ChatGPT or similar)

Paste this whole file as your first message to the new assistant, then attach or link the files it
asks for. It is written so any assistant can pick up the work where Claude left it.

---

## Your role

You are continuing work on **Arcane Orb**, a roguelite brick-breaker game headed for Steam (Windows
first), made by a solo developer (the user). Work in small, verifiable steps, show the user pictures
of visual changes so they can grade them, and keep the files listed below up to date as you go.

## The project in one paragraph

Go 1.24 game on a patched Ebitengine 2.8.8 (copy in `third_party/ebiten`, patches listed in
`third_party/ebiten/ARCANE_PATCHES.md`). Repo: GitHub `ginograziano24-cpu/Arcane-Orb`, branch `main`.
Seven hero origins, three acts of four worlds, items carried between acts, endless/daily/classic modes,
challenge chambers, online duels. 7 languages (en zh ja ru es pt de): every new player-facing string
needs all 7 (`tools/i18n/src/*.jsonl`).

**Read first:** `HANDOFF.md` (build, benchmark, renderer, constraints), then `ROADMAP.md`.

## Building the game

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ArcaneOrb.exe .
GOOS=windows go vet . && GOOS=js GOARCH=wasm go vet -tags capture .
```

Playtest zip: `ArcaneOrb/` folder with ArcaneOrb.exe, PLAYTEST.txt, README.txt, steam_api64.dll and a
`steam_appid.txt` containing `480`.

**Not yet compiled:** the step-1 lore code (story.go, classsel.go, progress_ui.go, game.go, lore.go)
was written in a session that couldn't download Go modules. First job on a machine with internet:
build and vet both targets and fix anything that turns up, then screenshot the class select, Lore page
and story cards in all 7 languages.

## The opening intro (the current focus)

A ~64 s animated intro, prototyped as a single HTML/canvas page. **It is a work in progress and not in
playtests** (the game doesn't play it yet). Plan: polish it here, then rebuild it in Blender for final
lighting, then play it in game (skippable, first launch, rewatchable from the Codex).

- Current build: `docs/cinematic/opening_preview.html` (same as `docs/cinematic/wip/style_tests.html`).
  Open it in Chrome. Style buttons switch the drawn half between pencil / ink & watercolour / comic /
  flat; **flat cartoon is the chosen look**. The source (before hero images are embedded) is the same file.
- Storyboard and sound cue sheet: `docs/opening.md`. Ideas and the Blender plan: `docs/cinematic/NEXT.md`.
  Status log: `docs/cinematic/wip/README.md`. Character reference sheets: `docs/cinematic/refs/`.
- Story: Archmage Fumbleton buys an "Instant Tidy" scroll from a travelling merchant (NO REFUNDS) to clean
  his tower before his girlfriend Mirabel visits. The three bosses it summons tidy the whole world into
  bricks. He throws the Orb, the page burns away into the arcane half: Orb smashes bricks, seven hero
  splashes, portal march, title. Lighthearted, LORT-style jokes.
- Scenes (start times): exterior 0, the date 2.5 (freeze-frame name card, Mirabel daydream), scroll 11,
  outside 17.5 (Sorcerer's Apprentice escalation), bricks 26.5, turn 33, orb 37.5, heroes 45, march 52.5,
  title 59. The "Mirabel knocks at the bricked-up door" ending was cut and saved as an epilogue idea
  (function `sEnd` is still in the source).
- Music: a composed WebAudio score (waltz theme, daydream variation, march, arcane variation) scheduled on
  the audio clock. Sound effects are synthesized stand-ins for a real mix.
- Performance: drawn half renders one drawing ahead in idle time; every cache is prebuilt behind the
  loading screen; magic-half sprites (magic circle, sky rays, sunburst, logo letters, hero titles) are
  cached; the magic half lowers its own resolution if frames run long. Measure with `window.__perf`.

### Locked character designs (do not change without the user asking)

**Mirabel** (adult sorceress, Fumbleton's girlfriend): body model "Final · C★ pose with C+++ proportions"
(`MVARS` index 9): bust as wide as the hips, narrow waist, hand on hip, staff in the other hand.
Purple fitted gown matching her purple witch hat (star on the cone, dark band), sheer black high-neck
top and fitted sleeves to the wrist, heart motifs, no leg slit, spiral wooden staff. Dark violet-black
hair pulled back into a low wavy ponytail (only visible from behind), two front locks ("Soft taper").
Face: dark brown eyes with a winged lash line, light makeup, one filled brow per side, red lip.
Keep her clothed and non-explicit; the user is happy with these proportions as the ceiling.

**Fumbleton**: pear-shaped blue robe with moons and stars, gold hem, belt, floppy hat with a bent tip,
gold star and dark band, huge white beard with curls, moustache, round nose, rosy cheeks, curled slippers.

**Bosses**: Prism Colossus, Crucible Golem, Crystal Warden (simple sketch shapes in the intro; match
the in-game bosses).

## Open tasks (pick up here)

1. Intro: lock the front-lock style; build a 3/4 view and close-ups (hands, staff, face) for Blender;
   boss turnaround sheets.
2. Intro: check playback on a real GPU (the magic half was only measured in headless software rendering).
3. Game: compile and verify the step-1 lore code (see above), then the performance plan in
   `docs/PERF_NEXT.md` (Go 1.26, PGO, Ebitengine 2.10, frame cap, idle throttling, cheaper bloom).
4. Then the rest of `ROADMAP.md`.

## Rules the user has set

- Don't work around blocked downloads or permissions; say what's blocked.
- Magic Ball (Alawar) inspired the game and music: analysis only, never copy its audio.
- Commit and push after each milestone; keep `HANDOFF.md`, `ROADMAP.md` and the intro docs current.
- Show progress pictures for visual work so the user can grade them.
