# Claude handoff, 2026-11-13 (update after every task)

Session goal from the user:
1. Make the dreamscape worlds extra worlds.
2. Fix clipping in every world, and the background pop when the camera moves.
3. Give the worlds high-quality set pieces, rough up the terrain, and give the trees bark.
4. Fix the Sunking's face.
5. Import new hero models.
6. Finish the 3D Blender intro, with unique character voices.

Keep working until usage runs out. A message is scheduled for 5:01 AM Chicago time to resume.

## Done (all committed to the PC's Source/; tests pass, build OK)

### Sunking's Road is its own world
- It has its own world id, `WDream`, added in `biome.go`.
- It appears in the Act I and Act III world pools. Meadow is back to normal.
- `dreamWorld(id)` is now simply `id == WDream`.
- The `-dreamcore` flag only forces every chamber onto that world.

### Clipping and camera fixes
- **Near-plane clipping** (`nearclip.go`): polygons behind the camera used to smear across the screen. This fixes the "background zooms in" bug. The `TestProjectedModelBankOutput` hash was re-recorded for this.
- **The sky follows the camera:** it pans with yaw, uses mirrored tiles, and its horizon is taken straight ahead of the camera (`world.go`, `skyPanX`).
- **Scenery veil** (`brickfocus.go`): now drawn only from the play camera. This removed the dark X bands.

### Terrain (`terrain.go`)
- Rolling hills, dunes and rock walls per world. The ground stays flat near the arena and is distressed with light and shade patches.
- `drawBakedGround` routes to the terrain whenever `activeTerrain` has a style.
- The ground extends to ±300 in every direction, including behind the camera.
- `hillRange` blobs are disabled.
- `groundUnder(x, z, r)` seats far props on the terrain (barn, hedgerows, pyramids).
- Fog was pushed further out in several worlds.

### Trees (`trees.go`)
- `leafyTree` builds a textured tree:
  - trunk wrapped in bark (`assets/floor/tex_bark.jpg`)
  - boughs
  - a crown of leaf lumps (`tex_leaf.jpg`) that sways
- `oakTree` uses it, and `oakTreeAt` takes an explicit y: the Cloud Kingdom island trees were floating.
- `shrub` is now textured.

### Set pieces
- **Windmill:** moved onto the hills on a stone footing, with the sails clear of the tower.
- **Meadow skies:** repainted (`assets/bg/meadow_0` and `meadow_1`) with painted tree clumps and softer clouds. The originals are in `Backup-before-Claude-2026-11-13`.
- **Machine Works:** the molten runnels are now opaque and drawn in segments, so they no longer show through props.
- **Dream hedges:** `dreamCylinder` is capped at both ends, so the rolls no longer look hollow.

### Sunking (`dreamsun.go`)
- Redone to match the reference: all crimson, wide white eyes with tiny pupils, a furrowed brow, a toothy grin, and thicker wavy rays.
- It sorts at depth 9e8, so hills can cover it but the flat ground can't.

### Audit tool
- `Audit-Shots.cmd` runs the game with `-audit`, which photographs every world from the play camera plus left, right, back and high angles into `shots-audit/`. The code is in `audit.go`.

## Hero models — DONE (01:25)
- 7 Tripo models generated (credits 1835 -> 1450). GLBs (63-67MB, ~1.7M faces, 4K tex) are in
  ArcaneOrb-Updated/heroes/glb/hero_<id>.glb (copied from Downloads GUID .tmp files by Render-Heroes.cmd).
- heroes/render_heroes.py (Blender 5.2, EEVEE): <id>_full.png 1200x1600 + bust poses rest/look/fierce/hurt
  536x680 into heroes/renders. Render-Heroes.cmd = copy+all; Render-Hero-Busts.cmd = busts only.
- Converted into Source/assets/portraits/<id>.jpg + _a.png + .json (new FX anchors) and hud/<id>_<pose>.png
  (converter: /tmp/heroconv/conv.py; bg (24,17,40)). Originals in Backup-before-Claude-2026-11-13/assets/portraits.
- Audit run started at 01:25 (adds audit-classsel shot) to verify class select + HUD + tests.

## Intro + HUD (01:35)
- The 01:25 audit was verified: class select shows the new heroes, and tests pass.
- **HUD busts:** re-rendered with tighter framing, converted, and committed to `assets/portraits/hud`. All 7 heroes look correct.
- **`introfilm.go`:** new and committed. It plays `intro/frames/f####.jpg` at 24 fps, with `intro/voice.wav` and captions from `intro/lines.txt` (`start|end|who|text`). `opening.go` uses it whenever `intro/frames` exists. Escape skips it.
- **`intro/build_intro.py`:** fixed and committed:
  - Heroes were never visible. They now use `span()` with step keys.
  - The orb was huge; it is now radius 0.17.
  - The equirect sky caused a blank floor. The sky is now a big emissive backdrop plane.
  - The test render was re-launched at 01:35.

## Intro voices + full render (05:01 resume)
- **Hero voices** (SAPI David, Zira and Mark, each pitch-shifted and given its own effects, so every hero sounds different):
  - `voices/lines.tsv` holds the line text and voice settings.
  - `Make-Voices.cmd` runs `voices/make-lines.ps1`, which writes `voices/lines/*.wav`.
  - `voices/mix.py` (run here) turns those into `intro/voice.wav` and `intro/lines.txt`. Both are committed.
  - Arcanist opens, Guardian calls the fall, each hero has a catch line, and Berserker and Gambler banter at the end.
- **Intro scene fixes:** rolling hills grid, noise-textured ground, flowers sitting on the hills, a taller backdrop (the castle is visible), a small blue orb, and a closer finale camera.
- **Full render:** `Render-Intro.cmd` was launched at about 06:00 local and takes about 2 hours. When `intro/done.txt` appears, check a few frames in `intro/frames`.
- **Gotcha:** a commit right after a Bash edit once pushed the old file. After committing, always check the size with `device_list_dir`.

## Live 3D heroes in class select (06:20, the user asked for this)
- The user said the portraits look like generated art, not models. They want the real 3D heroes and want it to look good.
- **`heroes/export_live.py`** runs headless in Blender via `Export-Live-Heroes.cmd`:
  - Decimates each GLB to 32k triangles.
  - Re-UVs it with smart project.
  - Cycles-bakes the full model's diffuse colour into a 2048px texture.
  - Writes `Source/assets/heroes/<id>.orbt` and `<id>.jpg`.
  - The first version reused Tripo's UVs, which gave crack artifacts. Baking fixes that.
- **`heromodel.go`:**
  - Loads `.orbt` and fixes the winding, because the y/z swap mirrors it.
  - `drawHero3D` renders the hero to an offscreen target at 2× supersampling with smooth per-vertex shading and a class-tinted rim light.
  - `classsel.go` `drawP` uses it whenever a model exists. Yaw sways ±0.38 rad, plus mouse parallax.
- **Engine fix in `render3d.go` `drawRuns`:** return early on an empty list. `planList` left a stale `runIdx`, which crashed `fillVerts` for a batch with no glow quads.
- **Debugging aids:**
  - `crash.go`: in bench mode, a crash also writes `crash-last.log` next to the exe.
  - `Audit-Shots.cmd` captures the game's stderr to `audit-stderr.txt`.
- **Preview tool:** `/tmp/heroprev.py <stem> out.jpg` software-renders an `.orbt`. Run it from the folder holding the files.
- **Done (06:50):** the GLB seams are welded before decimation. Without the weld, 39% of edges were open, which showed up as cracks; now it's 2.8%.
  - The bake goes through emission, so metal no longer comes out dark.
  - All 7 heroes are baked and verified in the game build (`shots-audit/audit-classsel.jpg`), and tests pass.

## Stage pass (09:15, the user's top priority: redesign stages, camera pop-in, clipping)
- The user approved Tripo credits for rigging all 7 heroes, to happen AFTER the stage work.
- **Camera clearance (`camclear.go`).** Fixes photo-mode orbits and other non-play cameras where props fill the lens:
  - Each prop function in `scenery.go` opens a scope with `B.objBegin(x,y,z,r)` / `objEnd`.
  - Lone `Batch.Draw` / `DrawTexTint` calls off the board are cleared automatically using their mesh bounds (`autoClear`, `onBoard`).
  - Any prop near the lens or inside the eye-to-board cone shrinks and sinks away smoothly.
  - The play view is unaffected (verified in the audit).
- **Caverns.** The floating arch slabs with wire curtains are replaced by `caveGrotto` (in `dressing.go`): stacked-boulder pillars down to the floor, stalactites, and glow-worm bead strands.
- **Sunking's Road:**
  - Terrain is flat out past the lake (`terrain.go` WDream start 64). The hills used to cut the lake into strips.
  - `ground_dream_meadow.jpg` repainted as a brighter emerald with mottling and flowers (`/tmp/paint/dreamground.py`).
  - Topiaries now come in 3 shapes and 3 tints.
  - The lake has a winding shore with pebbles and glowing reeds, and runs into the fog (z −40..150, x to HalfW+150).
- **Commit gotchas:**
  - A commit can push a stale copy when the file was copied to outputs just before. Always check sizes with `device_list_dir` after a commit, and re-commit if they differ.
  - JPEG commits get re-encoded, so for images compare pixel content, not file size.

## Rigging (11:00–11:45)
- **Tripo:** retopologized (20k triangles, 35 credits each) and auto-rigged (Humanoid/Mixamo, 20 credits each) storm, lucky, engineer, guardian, pyro and berserker. Credits went 1450 → 1085.
  - The Arcanist was blocked by Tripo's "Not Ready" pre-check; "Try Rigging Anyway" does nothing.
- **Exports:** downloaded as random names into Downloads. `Rig-Heroes.cmd` runs `heroes/rigged/collect.ps1`, which copies the last 75 minutes of downloads to `heroes/rigged/in/`, then runs `sort_and_rig.py` in Blender.
  - `sort_and_rig.py` identifies each file by face count (rigged files import with +80 faces) and writes `heroes/rigged/<id>.glb`.
- **Arcanist rig:** the Pyromancer skeleton, mirrored bone by bone in edit mode, fitted to her height, with L/R names swapped in two passes.
  - Her mesh is welded first, then automatic weights. Output: `arcane_rigged.glb`.
  - Known flaws: the staff bends because it isn't a separate loose part, and a small speck stays behind.
- **Pose check:** `heroes/rigged/check/<id>.png` shows rest pose and test pose side by side; contact sheet at `rig-check.jpg`. The six Tripo rigs deform cleanly.
- **Done since then (afternoon):**
  - The Arcanist's staff is rigid now. `heroes/rigged/export_anim.py` welds the mesh, finds the staff axis on the −x side and gives its vertices (radius 0.012h, plus the head) 100% weight on RightHand. It also deletes the speck. Running it writes `<id>_fixed.glb` for the intro.
  - **Skeletal idle in the game:** `Export-Anim-Heroes.cmd` writes `Source/assets/heroes/<id>.orbs` (skinned format, 72-frame breathing idle, 24 fps). `heromodel.go` loads `.orbs` first, falls back to `.orbt`, and does CPU skinning in `animate(heroClock)`. `classsel.go` sets `heroClock = g.stateT`. To preview without the game: `/tmp/orbsprev.py <frame> <ids>`.
  - **Keepsakes moved into Sanctum:** the main-menu "Keepsakes" item is gone, and its badge now sits on Sanctum. Sanctum and the keepsake tree share a tab bar (`sanctumTabs` in `progress_ui.go`). `keepsaketree.go` draws the tabs when `keepFrom == StMenu`.
  - **Intro (`intro/build_intro.py` v5):** uses `heroes/rigged/<id>_fixed.glb` and keys the idle on every rig. Before each catch the left arm rises (`CATCH_T = 12+4k+1.2`). The flying Orb is keyed to the hand's world position at that frame, and a held orb is parented to the LeftHand bone until the shot ends. In the finale all arms are held raised. The flying Orb shrinks to the held-orb size as it lands, and the catch shots are framed tighter. **Fully rendered**: `intro/frames`, 1200 frames, finished 15:59. Verified in build: tests pass and class select shows the skinned idle hero.

## Scenery detail pass (afternoon, 16:02 request)
The user wants about 10 passes over every level. Rule: **never use simple shapes for bushes or trees.** Grass needs real blades, trees need bark and roots, and backgrounds should be scenic and detailed, with ambient SFX. Progress so far:
- **`flora.go` (new):** a procedural flora library.
  - Wood is built from tubes along bent paths, with bark ridges, moss low on the up side, and roots.
  - Foliage is individual folded two-tone leaves, needles, fronds, or five-petal flowers. `floraGet` caches per variant. `floraFar` (more than 58 units away) switches to a lite variant.
  - Species: `growTree` (oak; dome crown of leaf clusters fed by twigs), `growPine`, `growPalm`, `growLumen` (tintable glowing hanging leaves), `growBush`, `growTopiary` (0 ball / 1 cone 3.2 tall / 2 hedge tile), `growStem`, `growHedgerow`, `growSunflowerStem`, `growPumpkinVine`, `growFern`, `growReed`, `growHangingRoots`, `growFlowerBed`, `mossTuft`.
  - Stone: `growRock` (chiselled noise boulder, tintable, optional moss) and `growStalagmite`.
  - `drawFlora(B, set, pos, scale, ry, tint, t, sway)` draws the wood with the bark texture (or plain colours when `plain` is set) and sways the foliage.
- **Replaced in the worlds:**
  - `leafyTree`/`oakTree`, `shrub`/`shrubPal`, `palm`, `lumenTree`, `boulder`.
  - Cavern stalagmites, grotto pillar rocks, dream pebbles, dream lake reeds.
  - Dream topiaries, the lollipop trees (now cherry-blossom trees) and the dream wall hedges (`dreamHedge` draws leaf tiles).
  - Meadow: hedgerows, sunflower stems, pumpkin vines, flower beds, moss on the dry-stone walls, moss on the stone head.
  - New: a meadow copse (oaks plus `pineTree`), six more Glimmer Grove lumen trees, night-palette shrubs, glowing cave ferns and moss, and hanging roots under the cloud islands.
- **Grass (`foliage.go`):** curved three-segment tapered blades with a dark-to-light gradient, 7 to 10 per tuft, sitting on the terrain. Dreamscape has grass now too.
- **Fixed:** the dream bricks were black because `darken(col, 0.08)` keeps 8% of the colour; it is now `darken(col, 0.9)`.
- **`ambience.go` (new):** a 24-second seamless stereo bed per world, synthesised on first entry. `Sounds.SetAmbience(world, on, dt)` is called from `game.go` every frame; it is on during `StLevel`/`StCache` and its volume follows the SFX slider.
- **Engine note:** `Icosphere` winds its faces inward (the culling sees the inside). `growRock` flips its faces outward; other icosphere props were left alone.
- **Offline preview harness:** `/tmp/florasim` (copies `flora.go`, `go run .`, then `python3 render.py x.bin`).
- **Still to do:** passes 5 to 10 on Clouds, Machine, Ruins and Dream polish; a performance check with the bench (oak about 7.3k triangles, pine about 4.8k); and a further look at the Glimmer Grove menhir/lantern posts.

### Later the same evening (16:30 to 17:15)
- **User requests:**
  - **Sun:** "the sun is creepy, make him relaxed". Then the user sent a reference (a 90s CGI sun with heavy half-closed lids) and asked for "a stoner, match his geometry and texture". Then: fix the eyelid clipping, make him glow like bloom, and drop the bloodshot look (just reddish eyes).
  - **Bricks:** "remove the weird symbol on the dream bricks".
  - **Colour:** "more vibrant hues and bloom on light sources" in Dreamscape.
  - **Meadow:** "Sunset Meadow is a cluttered mess".
  - **Plan:** finish the passes, then 5 optimisation passes, then 5 bug-fix/QA passes.
- **Sun (`dreamsun.go`):**
  - `sunFaceMesh` (`sunFace4`) is a subdivision-3 ball, noise-pitted, mottled red/orange with gold flecks, wound outward.
  - `sunFlame` rays are flattened four-sided spikes banded gold/orange/red.
  - `sunBall` (`sunBall2`) is wound outward.
  - Face: heavy lids (depth 0.14) wholly in front of flat eye whites (0.025), iris at 0.07, pupil at 0.088. Plus a nose, full red smirking lips, puffy cheeks, bags, slow drifting irises, a lazy lid sag, and lilac smoke puffs.
  - Additive bloom billboards around him, and a stronger orange halo in `dreamcore.go`.
- **Dream bricks:** drawn as `bevelBox(0.16)` (`flora.go`); the symbol came from the `dream_gem` texture.
- **Dream light sources:** 14 floating paper lanterns with layered bloom, glowing stars on the candy spires, and pink glow inside the blossom canopies.
- **Meadow declutter:**
  - No coloured bloom dots in the grass; dry-stone walls removed.
  - One flower bed per fence line.
  - Garden side (`-side`): sunflowers, one scarecrow, a three-pumpkin patch. Pasture side: one hay bale, shrubs at the tree feet.
  - The copse is grouped into two groves on the far hills; hedgerows are 3 per side.
- **Performance:**
  - `drawFlora` is static (no sway, no textured bark) so it stays in the bake cache. It also does whole-plant screen/speck culling.
  - `floraFar` threshold is 42. Lite variants are much lighter; leaf budgets cut.
  - `meshbake.go`: each `bakeGeo` has a bounding sphere, and `sphereOffscreen` skips whole placements.
  - `heromodel.go`: skinning at 30 Hz into reused buffers, 4 goroutines; the rendered hero frame is cached by `heroShotKey`.
- **Benchmark:**
  - `Bench-Flora.cmd` builds, then runs `ArcaneOrb-Flora.exe -bench -benchtag=fl`. Compare with `bench_report_dd.txt`.
  - With budgets only: world2 134 to 280 fps, world7 77 to 146 (the culling build is not measured yet).
  - Do not double-click near the bottom of the list: new files shift the rows. (`ArcaneOrb-Flora.exe` got launched by mistake once.)

- **Optimisation passes (5):**
  1. Flora budgets cut.
  2. Static `drawFlora` with whole-plant culling.
  3. `meshbake` bounding-sphere culling.
  4. Hero render cache, with skinning at 30 Hz in parallel.
  5. Screen-size detail levels: `floraPx`/`floraLOD`, a tiny tier built with `stripBacks` (single-sided leaves), via `treeVariantLOD`.
- **QA passes (5):**
  1. Full audit; all 7 worlds render and tests pass.
  2. Back/high angles: no floating props.
  3. Class select: the cached hero renders correctly.
  4. Code review: ambience threading, LOD keys, cache keys.
  5. Final build, test and benchmark.
  - Also fixed: sun eyelid clipping (layer depths), the sun's iris hidden at equal depth, the bloodshot look (whites are soft off-white), and the dream bricks' black colour bug.
  - The Sanctum and Keepsakes tabs were checked live in the game.

## FPS pass: all 9 ideas (evening, 18:22 request "Do them all")
- **Background layer cache** (`backcache.go`): far scenery is drawn into an image ~30 times a second and reused between refreshes, shifted with the camera. It switches itself off for 2 s whenever it doesn't pay (`backWorth`), and is always off for classic mode, bosses, the intro, the finale and photo mode. The key includes size, FOV and `aaScale`.
- **Impostors** (`impostor.go`): plants under ~70 px become 8-view billboards baked on the main goroutine (`bakeImpostors`).
- **Leaf cards**: far oaks and blossom trees draw their crowns as crossed textured cards (`bakeLeafCard`, `cardsFromLeaves`).
- **Fewer batches / merged statics**: `mergedStatic` merges meadow hedgerows and far dream topiaries per band. Candy spires use vertex colours, with no texture.
- **Quality tiers**: `floraDetail` 0.35/0.6/1 by Quality.
- **Dynamic resolution**: `dynamicAA` steps the supersample 1→2 against the display Hz when DynRes is on. The bench pins it to 2, so bench numbers don't show this gain.
- **Thinner dream topiaries**: 27 forms, far ones merged.
- **Skip hidden props**: `sphereOffscreen`, `hiddenByTerrain`, and the `skipBack` early-out in `drawBaked`.
- **Bench (fps, High quality, AA pinned to 2):**
  - world 7 (Dreamscape): 154 → 202
  - world 2 (Meadow): 289 → 297
  - class select: 247 → ~1250
- **New flag**: `-world=N` plays every chamber in one world (0 Glimmer, 1 Meadow, 2 Caverns, 3 Ruins, 4 Clouds, 5 Machine, 6 Sunking's Road).
- **QA, 23:17**: the audit crashed in `bakeLeafCard` because the card mesh had no corner table. Fixed: `drawShared` now builds the table itself if it's missing (mesh.go), and the card mesh builds it first (impostor.go).
- **Final bench, 23:30, build including the fix:** world7 190 fps (1% low 105), world2 291, world1 646, world3 474, world4 397, world5 370, world6 606. Within noise of the 202/297 run. No frame over 33 ms anywhere. The playtest exe was built at 23:25, and the audit shots of all 7 worlds look clean.
- **Playtest**: `Build-Playtest-Nov13.cmd` builds `ArcaneOrb-Playtest-Nov13.exe`. Launchers: `Play-Playtest-Nov13.cmd`, `Play-Meadow-Nov13.cmd` (-world=1) and `Play-Dreamscape-Nov13.cmd` (-world=6).

## Playtest feedback fixes (Nov 14, 00:15 request)
- **3D heroes in class select** (`herodepth.go`): real per-pixel depth in three passes.
  - Coarse 8-bit depth with a min blend, then fine depth inside the winning step, then a colour pass that discards hidden fragments.
  - No back-face culling, because some scans have flipped patches.
  - Falls back to the old painter's path if the shaders fail.
  - Ebiten fork gotcha: in pixel-unit Kage, image N≥1 is sampled with `imageSrcNUnsafeAtFromSrc0Pos(pos)`, where pos is in image 0's pixel space. `imageSrc1UnsafeAt` is the deprecated alias of the same function, so don't add `imageSrc1Origin()`.
  - The audit dumps `dbg-depth*/dbg-color-*.png`.
  - Remaining: close-up faces look soft because of the live mesh and texture resolution.
- **HUD busts:** `render_heroes.py` has a `BUST` table with a per-hero face x offset and step-back. Arcanist and Berserker were off-centre, and the Arcanist's hat was cropped. Re-rendered and copied into `Source/assets/portraits/hud`. Scripts: `Fix-Heroes-Audit.cmd`, `Render-Busts-Fix.cmd`.
- **Dreamscape** (`dreamorb.go`):
  - Post and spire orbs are now crystal orreries: a gem, a chalice and two turning armillary rings.
  - Floating lanterns are ribbed paper lanterns with gold caps and a tassel; they used to blow out to white balls.
- **Audit:** new scenes `audit-hero-<id>` and `audit-hero-<id>-face` (`heroAuditZoom` frames the head). `Hero-Check.cmd` saves the shots to `shots-hero/`.
- **Open:** the user saw "slight clipping on the right" in a room called Radiant Orrery, but room names are random, so the world is unknown. A thin dark diagonal line crosses the right-hand hedge and post in the `audit-w7-play` shot, so suspect that.
- **Playtest:** `Play-Playtest-Nov14.cmd` and `Play-Dreamscape-Nov14.cmd` (copies of `Source\ArcaneOrb-test.exe`).

## Next
1. **Intro:**
   - DONE: rendered with the rigged heroes (row y=328, about 4 minutes). Possible polish: victory poses in the finale, and per-hero catch styles.
   - `introDir()` looks for `intro/` next to the exe, one folder above it, and in the working directory, so the files already in `ArcaneOrb-Updated/intro` work as they are.
   - Then write `intro/lines.txt` and `voice.wav`.
2. Run another audit pass. Known rough spots:
   - the dream lake reads as strips
   - drip lines under the cave ledges look like wires
   - glow decals draw over props from some angles
3. **Intro:** Blender 5.2 is installed at `C:/Program Files/Blender Foundation/Blender 5.2/blender.exe`. Build the opening with a `.cmd` that runs `blender -b -P script.py`. The game has no video decoder, so play it back as a frame sequence loaded from disk.
4. **Voices:** only the David, Zira and Mark SAPI voices are installed (see `List-Voices.cmd` and `voices/voices.txt`). Ask the user before downloading a neural TTS such as Piper or Kokoro. The user has not answered yet.

## Notes
- **Running things on the PC:**
  - File Explorer is click-tier only.
  - `computer_open_application("File Explorer")` often opens an extra Home window. Close it from its X, or click the ArcaneOrb window's title bar to bring that window forward.
  - List rows start at y=216 and are 16 px apart (x=672).
- **Committing files:** copy the file first and check it, then commit. Never copy and commit in the same parallel block.
