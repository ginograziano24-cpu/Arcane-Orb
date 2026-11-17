# Arcane Orb on Roblox: the Luau port

Started Nov 17, 2026. This is Phase 1 of `docs/ROBLOX_PLAN.md` (the core
loop prototype), done early so it's ready when you want it.

## What's in `roblox/`

A [Rojo](https://rojo.space) project. `rojo serve` in `roblox/`, then
connect from the Rojo plugin in Studio, and the files appear in the place:

| In the repo | In Studio | What it is |
|---|---|---|
| `src/shared/Data/*.luau` | ReplicatedStorage.ArcaneOrb.Data | **Generated** from the Go code: Stats, all 145 items (Mods), Fusions, Synergies, Origins, Keepsakes, Worlds |
| `src/shared/LevelBank/*.luau` | ReplicatedStorage.ArcaneOrb.LevelBank | **Generated** by the game's own level generator: 288 brick layouts (3 acts × 8 depths × 12), plus the 7 arena shapes |
| `src/shared/Sim/Const.luau` | …Sim.Const | Arena measurements, brick kinds, power-ups |
| `src/shared/Sim/Level.luau` | …Sim.Level | One chamber: orb physics, bricks, damage, procs, loot, lives |
| `src/shared/Sim/Run.luau` | …Sim.Run | A run: origin, build, rewards, gold, lives, acts |
| `src/client/Main.client.luau` | StarterPlayerScripts | The game loop, input (mouse, touch, keys, gamepad), camera |
| `src/client/Renderer.luau` | StarterPlayerScripts | Draws a chamber with pooled parts, plus effects |
| `src/client/UI.luau` | StarterPlayerScripts | Origin select, HUD, reward cards, banners, end card |
| `src/server/Main.server.luau` | ServerScriptService | Meta save (DataStore), leaderstats, run reports |

Without Rojo: make the same folders by hand and paste each file into a
ModuleScript (or LocalScript / Script for the `.client` / `.server` files)
with the same name.

## The Go code stays the source of truth

- **Data tables** come from `tools/luauport` (Go). It parses the game's Go
  source and translates the tables, including every item's `Apply`
  closure, statement by statement. After a balance change:
  `cd tools/luauport && go run . -src ../.. -out ../../roblox/src/shared/Data`.
  Anything it can't translate stops the run with the file and line, so a
  table is never silently wrong.
- **Brick layouts** come from `tools/levelbank`, which calls
  `levelgen.Generate` with the same pacing as `play.go`. It needs Go 1.26,
  so it runs on the PC: `Roblox-LevelBank.cmd` writes `roblox-levels\`.
- **Gameplay code** (`Sim/Level.luau`, `Sim/Run.luau`) is a hand port of
  `play.go` and `run.go` with the same numbers. When those Go functions
  change, port the change by hand; the function names match.

## Tested without Roblox

There's no Luau runtime in the cloud workspace, so two small tools in
`tools/luauport` stand in:

- `luaucheck.py`: a syntax checker for the Luau the port uses, which also
  flags globals that are never defined (usually typos).
- `luaurun.py`: a tiny Luau interpreter with stubs for `Random`, `Color3`,
  `script` and `require`. `tests/sim_test.luau` plays whole 3-act runs
  with an autopilot and a build picked from real reward offers, and fails
  on any runtime error (nil fields, wrong names, arithmetic on nil).

```
python3 tools/luauport/luaucheck.py roblox/src
python3 tools/luauport/luaurun.py roblox/src tools/luauport/tests/sim_test.luau
```

Neither is Roblox: the first time in Studio, check the Output window and
the Script Analysis window (strict mode is on in every file).

## What's ported and what isn't

**Ported, same numbers as Steam:**
- Orb physics: sub-stepped movement, the angle fix, shaped arena walls,
  brick contact with splash damage and ploughing, paddle angles, the
  last-brick aim assist, the barrier.
- Damage: crits, combo damage, every damage multiplier in `ballHit`,
  frozen and burning bonuses, armour, shields, steel, execute.
- Procs: burn, freeze, lightning chains (with recast), explosions with
  fuses, void, echo, Crescendo, Hawkstrike, Eagle Eye, Molten Core,
  Shatter, Steam Engine, Plasma Storm, black hole, Hydra.
- Brick kinds: all 15 (explosive, chain, split, regen, moving, shield,
  teleport, spawner, multiplier, crystal, mystery, elite…), stacks that
  drop when the brick under them breaks.
- Loot: coins with physics, power-up drops (all 11), magnet, lasers.
- Runs: origins, items, synergies, fusions, rarity weights and the
  early-run rarity gate, interest, chamber rewards, lives per act.

**Not yet (Phase 2 and 3 in the plan):**
- Guardians (boss fights): the 8th chamber of each act is an elite
  chamber for now.
- Origin passives and active abilities (Parry, Rampage, Attunement…). The
  origins' stat changes are in; their special mechanics aren't.
- Familiars, scrolls, curses from events, shops, events, the branching map.
- Hero models, music and sound (the renderer raises `sound` events; they
  need uploaded audio IDs).
- Co-op, duels, lobby, badges, cosmetics.

## Next steps in Studio

1. Install Rojo (Studio plugin + the `rojo` CLI), `rojo serve` in `roblox/`.
2. Game Settings → Security → turn on Studio Access to API Services (for
   the DataStore).
3. Press Play. Pick an origin, play a chamber, check the Output window.
4. Tune the camera in `Main.client.luau` (`setCamera`) for phones.
