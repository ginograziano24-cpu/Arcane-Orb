# Codex handoff: Arcane Orb (2026-11-12)

Start here. For the long-standing build, bench and harness notes, read `HANDOFF.md` next, then `ROADMAP.md`.

## What the project is

Arcane Orb is a roguelite brick-breaker headed for Steam (Windows first).

- **Stack:** Go 1.26 (`module orb`) and Ebitengine. The patched copy lives in `third_party/ebiten`, via a `replace` in go.mod.
- **Location on the PC:** `C:\Users\Gino\Downloads\RoguelikeBreaker\ArcaneOrb-Updated\`. The Go source, which is the repo root, is in `Source\`.
- **Heroes:** seven origins, defined in `run.go`. Their ids are guardian, berserker, arcane (Arcanist), engineer, lucky (Gambler), pyro (Pyromancer) and storm (Stormcaller).
- **Runs:** a run is 3 acts × 4 worlds (`biome.go`). The other modes are endless, daily, classic, challenge chambers and online duels.
- **3D models:** `.orbm` files in `assets/models/`. Each is a zlib-compressed list of coloured triangles, loaded by `models.go`. Tripo GLBs are converted into this format (see "Hero models" below).
- **Backup:** originals from before this session are in `Backup-before-Claude-2026-11-12-session9`.
- **Last playtest build:** `ArcaneOrb-Playtest-Nov12.exe`. This is the normal look, saved before the restyle began.

## Building and checking

```sh
# Windows exe
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ArcaneOrb.exe .
GOOS=windows go vet .
gofmt -l .     # play.go already has gofmt diffs; leave them alone
```

These helper scripts sit in the PC folder:

- **Run-Tests.cmd:** builds `Source\ArcaneOrb-test.exe`.
- **Dreamcore-Shots.cmd:**
  1. Kills any running `ArcaneOrb-Dreamcore.exe`, then runs the tests and the build.
  2. Copies the build to `ArcaneOrb-Dreamcore.exe` and runs it with `-shots -dreamcore`.
  3. Force-kills it after about 3 minutes.
  4. Renames `shots\` to `shots-dream\`. The Sunking's Road screenshot is `world2.jpg`.
  5. Writes a log to `dreamcore-test.txt`.
- **Dreamcore-Bisect.cmd:** runs 3 timed runs with `ARCANE_DC_NOSUN=1` and `ARCANE_DC_NOBLOOM=1`, for performance bisection.
- **Play-Dreamcore.cmd:** launches `ArcaneOrb-Dreamcore.exe -dreamcore`.

## Current direction: the "dreamcore" restyle

The user wants the game restyled in the retro-90s-CGI, dark-fantasy "dreamcore" look from TikTok. Their four reference images show:

- a lit castle under a galaxy, with a grinning sun
- a leopard-print Jester King
- a goblin on a bridge
- a wizard riding a starry toad

**What the look needs:** vibrant, whimsical and saturated, with lots of stars and galaxies. Textures should be embossed and low-poly. Bright surfaces should glow, like bloom shaders.

**What the user rejected:** anything that looks smooth or "AI".

**The rule:** catching the aesthetic matters most.

The restyle sits behind a flag: `-dreamcore` on the command line or `ARCANE_DREAMCORE=1`. Without the flag the game is unchanged.

### First dreamcore world: Sunking's Road (replaces Sunset Meadow, `WMeadow`)

Files added:

- **`dreamcore.go`**
  - Sets the flag and `dreamWorld(id)`.
  - `applyDreamTheme` sets the name, subtitle ("the sun is watching you"), fog, lighting and textures.
  - Texture helpers: `DrawTex` / `DrawTexTint`, with box or cylinder UVs and emissive.
  - Primitive meshes.
  - The sky halo and glints: `drawDreamSky`, which also sets `dreamSunAt`.
  - The post-process `dreamFilter`: Kage sharpen, saturation and contrast, then a quarter-res bright pass, a separable blur, and an additive bloom.
- **`dreamscene.go`**
  - `drawDreamMeadow`, which draws:
    - the cobble road
    - a lake that reflects the sky
    - candy-cane cones
    - glowing topiary balls
    - lollipop trees
    - flowers, fairy lights and sparkles
  - `dreamHedge`: the arena walls. Each is a brick curb, a glowing gold trim, a hedge body and a rolled flower top. Each wall wobbles when the ball hits it.
  - `dreamWallHit`: petal sparks and a ring, called from the wall-bounce block in `play.go`.
  - `dreamPost`: the candy-cane posts.
  - `dreamGround`: the ground only spans z −16..88. This fixed a green band under the castle.
- **`dreamsun.go`:** `drawSunking`, the low-poly 3D sun drawn into the main batch. It has:
  - 14 two-tone curling flame rays
  - a red face
  - half-lidded eyes that look sideways or follow the ball, and blink
  - soft brows and a smirking toothy grin

  The user wants a "chill", smug expression, and keeps critiquing it, so expect more iteration.

Hooks in existing files:

- **`world.go`:** the dreamcore camera, the sky switch, the sun draw, and `dreamFilter` after the flush. It also routes wall and brick drawing to the dream versions.
- **`scenery.go`:** the WMeadow case and `drawPost`.
- **`dressing.go`:** `envState.hitX/hitZ/hitT`, and skips `dressMeadow` in a dream world.
- **`play.go`:** calls `dreamWallHit`.

Assets:

- **Sky:** `assets/bg/dream_meadow_0.jpg` and `_1.jpg`, the galaxy and castle sky at 2048×1152. `_1` is the mirrored copy.
- **Floors and textures:** `assets/floor/`:
  - `dream_meadow.jpg`
  - `ground_dream_meadow.jpg`
  - `dream_flowers.jpg`
  - `dream_candy.jpg`
  - `dream_brick.jpg`
  - `dream_roof.jpg`
  - `dream_gem.jpg`
- **Unused:** `assets/dream/dream_sun.png`.
- **Concept art:** in `Concepts-Dark-Fantasy\`.

**Performance:** about 780 fps in the shot run, against 447 for the old Meadow.

**Engine gotchas, learned the hard way:**

- **Batches:** never add a second `Batch` with its own `Flush` in a frame. It made each frame take seconds and froze the PC. Draw moving meshes with `B.drawShared` inside the main batch. `drawCached` is keyed by placement and only suits static meshes.
- **Textures:** `Batch.Quad` and `Batch.QuadUV` take registered textures. `bakedTexture(name)` loads `assets/floor/name.jpg`.

### Latest state

The last build, from 21:12 on 2026-11-12, showed:

- no horizon band
- the new sun

### Open polish items

- Make the hedges thicker and taller.
- Make the trim warmer gold, since it currently reads bluish.
- Make the hedge ends meet the posts cleanly.
- Keep refining the sun's expression.

### Worlds still to build, in the same pipeline

- Toad Hollow
- Goblin Gorge
- Jester's Dungeon
- Moon Palace
- Alchemist's Orrery

## Hero redesigns (in progress)

The user asked for "updated designs and Tripo models for the heroes, make them all very attractive".

Concept images were made in **Tripo Studio**, under Generate Image with the GPT Image 2.5 model. Image generation there is free, up to 100 a month, and 88 have been used. Each concept is a full-body front view in an A-pose, at 3:4, on a plain grey background, which suits image-to-3D. The images are still in the Tripo image history:

| Hero | Design |
|---|---|
| Guardian | Platinum-braided paladin, silver and gold plate, sun shield, violet starry cape |
| Berserker | Crimson-haired warrior, fur and red leather harness, glowing twin-blade axe. Regenerated so the background is plain |
| Arcanist | Midnight-blue-haired sorceress, constellation robe, star hat, orb staff |
| Engineer | Copper-haired inventor, goggles, brass gauntlet, clockwork drone |
| Gambler | Dark-haired card sharp, emerald tailcoat, top hat, glowing cards, coin |
| Pyromancer | Flame-haired battle mage, crimson and gold coat, fire in hand |
| Stormcaller | Silver-haired storm mage, grey-blue coat, lightning staff, storm clouds |

The user then asked for much curvier female heroes: bust and hips/glutes at about 2.25×. New Guardian, Arcanist and Pyromancer versions with an exaggerated stylized hourglass figure have been generated and are the newest three images in the history. The user has not yet confirmed them.

**Next steps:**

1. Get the user's sign-off on the final 7 images.
2. Run image-to-3D in Tripo Generate Model on H3.1, at **55 credits per model**. The user has 2055 credits, so all 7 come to about 385 credits. Rigging and animation are extra. **Do not spend credits without the user's explicit OK.**
3. Export the GLBs and convert them to `.orbm`.
4. Update the class-select and HUD portraits: `classsel.go`, `portraits/`, and `capture_native_heroreview.go` for the review shots.

## Rules the user has set

- **Credentials:** never type passwords, API keys or credentials, and never sign in for the user. The user does all sign-ins.
- **Credits:** never spend Tripo credits without listing the models and costs first and getting a yes.
- **Downloads:** downloading files to the PC needs the user's permission.
- **File Explorer** on the PC is click-only. Don't type into it, and don't work around that.
- **Magic Ball (Alawar)** is the inspiration, but never copy its audio.
- **Process:** small iterative polish passes with frequent screenshots. The user reviews visuals closely and wants quick turnaround between passes.
