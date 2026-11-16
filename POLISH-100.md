# Arcane Orb: the 100-item polish pass

Started 2026-11-12 after reviewing a screenshot of every screen (`Shots.cmd` writes them to `shots/`).
`[x]` done and checked in a screenshot or test, `[~]` in progress, `[ ]` to do. Work goes top to bottom inside each group; world art first.

## A. World art (the biggest gap)

1. [x] Screenshot tour: `ArcaneOrb.exe -shots` saves a picture of 31 screens, so every change can be judged by eye (`shots.go`).
2. [x] Neon Grove was a synthwave scene (striped sun, neon grid, city skyline, speakers, equalizer, hologram signs, palms). It is now **Glimmer Grove**, an enchanted wood at night. New sky paintings: moon, aurora, misty ridges, conifers, a wizard's tower.
3. [x] Glimmer Grove floor: four painted flagstone floors with moss in the joints and worn rune inlays (great circle, moon phases, vines, constellation) instead of the neon grid.
4. [x] Glimmer Grove ground: scrolling grid, neon pylons and conduits replaced by pooled moss glow, standing stones and a glow-worm trail.
5. [x] Glimmer Grove surroundings: rune stones that wake as the chamber clears, giant glowcap mushrooms that swell on bounces, lantern posts, young lumen trees, fireflies, and an aurora on the last wave.
6. [x] Glimmer Grove walls and corner posts: dressed stone with capstones and a moss-light edge instead of the neon rails.
7. [x] Machine Works sky: two foundry-citadel paintings (keep with furnace mouth, chimneys and smoke, a great gear, a gantry crane). The office-window skyline is gone.
8. [x] Machine Works floor: riveted iron plates with brass strips, tread plates, grates and rust, warmer toward the furnace end. The worn hazard stripe by the paddle stays.
9. [x] Crystal Caverns floor: dark slate slabs with glowing cyan and violet crystal veins in place of the neon grid.
10. [ ] Crystal Caverns sky: deeper cave, light shafts, bigger crystal formations.
11. [x] Sunset Meadow sky: two golden-hour paintings with lit cumulus heaps, sun rays, layered hills, tree clumps and a windmill.
12. [~] Grass and flowers (grown in clumps since cycle 17): two-sided blade tufts that sway, with flower heads, for the Meadow, Grove (glowing tips) and Ruins (desert scrub) (foliage.go). Detailed trees and bushes wait for the Tripo Studio models.
13. [~] Ancient Ruins sky: a repaint was tried and came out flatter than the current one, so the original stays for now. The Tripo stone head now sits half-buried in a sand drift.
14. [x] Cloud Kingdom: the flat blue band is fixed (skyedge.go fills below every sky painting with its own edge colour), and two new sky paintings with cumulus towers rising from the cloud sea and far castle isles.
15. [ ] Shop scene painting redone at the new standard.
16. [ ] Campfire scene painting redone.
17. [ ] Event scene painting redone.
18. [ ] Treasure room painting redone.
19. [x] Map, reward and act-intro screens showed the city skyline along the bottom; they now show the grove's forest line. (Acts II and III strips follow items 7 and 10.)
20. [x] Each Guardian now has its own giant Tripo-made figure looming behind its brick armour (guardianfig.go): breathes, leans in when hurt, sinks when defeated, with its own backlight colour.
21. [ ] Guardian intro: a splash using the opening's boss art.
22. [ ] Day-to-night colour grading re-checked per world after the repaints.
23. [ ] Classic backdrop pictures refreshed to match.

## B. Bricks, orb and paddle

24. [ ] Brick materials: Meadow's bricks read as flat plastic. Add bevel light and a top texture per world.
25. [ ] Damage states: clearer cracks at 2/3 and 1/3 health.
26. [ ] Special bricks: clearer icon on the top face.
27. [ ] Orb: trail and glow per origin.
28. [ ] Paddle: origin-specific trim that shows at play distance.
29. [ ] Contact shadows under bricks and along walls.
30. [ ] Break debris per material (stone chips, glass shards, metal sparks).
31. [ ] Brick emphasis: tune after you have judged it.
32. [ ] The word "PARRY" printed on the floor becomes a proper marker.
33. [ ] Camera: short fly-in per chamber and consistent framing (some layouts leave the bricks small at the far end).

## C. HUD and menus

34. [x] Empty scroll slots show a faint tome outline.
35. [ ] Top HUD panel: less clutter (chamber name, seed).
36. [ ] Keepsakes screen: large empty areas. Bigger nodes, tree art behind.
37. [ ] Trials panel: larger text and rows.
38. [ ] Settings: Effects tab has 12 rows. Split into Effects and Gameplay.
39. [ ] Reward cards: frame and rarity colour consistency.
40. [ ] Shop: tooltip overlaps cards; lock and sold badges.
41. [ ] Map: larger node icons, clearer paths.
42. [ ] Origin select: hero lighting and stat bars.
43. [ ] Codex: locked "?" tiles become silhouettes.
44. [ ] Run summary polish.
45. [ ] Pause screen: empty build panel state.
46. [ ] Tooltips: one style and delay everywhere.
47. [ ] Text size audit at 1280x800 (Steam Deck) using the screenshot tour.
48. [ ] Ultrawide and 4:3 layout audit.
49. [ ] Controller glyphs and focus ring on every screen.
50. [ ] Title screen: slow parallax and idle motion.
51. [ ] One consistent transition between screens.
52. [ ] First-boot splash while assets load.

## D. Sound and music

53. [ ] Loudness pass: a tool that prints peak and average level for every sound, then even them out.
54. [ ] Glimmer Grove brick sounds: chime and wood instead of synth.
55. [ ] Glimmer Grove music: check the track for synthwave arpeggios and restyle.
56. [ ] Menu sounds: hover, click and back consistent on every screen.
57. [ ] Guardian phase-change stingers.
58. [ ] Last-life heartbeat ducks the music.
59. [ ] Audio fades on pause and when the window loses focus.
60. [ ] Stereo panning by where things happen on the board.
61. [ ] Ambient bed per world (night insects, drips, machinery, wind).
62. [ ] Silent moments audit: reward pick, shop buy, map travel.
63. [ ] Opening: narration against music levels.

## E. Feel

64. [ ] Hit-stop and shake audit, including the Fast animations and Screen shake settings.
65. [ ] Paddle hit: squash and spark.
66. [ ] Combo counter presentation.
67. [ ] Wave-clear and chamber-clear celebration.
68. [ ] Last life and defeat sequence.
69. [ ] Guardian defeat sequence.
70. [ ] Power-up capsules readable at a glance.

## F. Performance

71. [ ] Fewer draw calls: put floor and brick textures on shared sheets.
72. [ ] Bake still scenery to one picture per chamber when the camera is still (biggest remaining cost).
73. [ ] Particle pooling to stop per-frame allocations.
74. [x] Grove floors are 45% smaller on disk (0.6 MB vs 1.1 MB each).
75. [ ] Half-size floor textures on the Low preset.
76. [x] The Low preset also sets Effects intensity Low. (Brick emphasis stays your choice.)
77. [ ] Steam Deck preset benchmark and defaults.
78. [ ] Frame pacing check with VSync at 60, 120 and 144 Hz.
79. [ ] Start-up time measured and trimmed.
80. [ ] Exe size (100 MB): compress the opening's large PNGs.
81. [ ] 30-minute soak test for memory growth.

## G. Code quality

82. [~] Dead code: the old synthwave title scene (`menu.go`) is deleted. Still to go: the unused sky generator styles in `biome.go`.
83. [ ] Rename internal `Neon` identifiers to Grove.
84. [ ] Split `game.go` (2,488 lines) and `play.go` (2,885 lines) by subject.
85. [ ] Static analysis clean: unused functions and parameters.
86. [ ] Picture tests: a hash of key scenes so accidental visual changes fail a test.
87. [ ] Save migration tests from each older save version.
88. [ ] Corrupted-save recovery test.
89. [ ] Stale comments fixed.
90. [ ] The 44 untranslated strings finished.
91. [ ] Hard-coded screen positions moved to layout helpers on the screens touched above.
92. [x] Tests can no longer write the player's real save (`TestMain` guard).

## Tripo models (asked for 2026-11-12)

- [x] Tripo CLI installed and signed in on the PC; `tripo doctor` passes.
- [x] Pipeline: `tools/tripo/glb2orbm.py` (GLB to the game's model format, one colour per face sampled from the texture), `simplify.py` (down to game weight), `render_orbm.py` (preview), `topohash.py` (adds the new model to the topology test reference).
- [x] First model: the Glimmer Grove standing stone (3,379 triangles, simplified to 837). Cost 110 credits; 860 left.
- [ ] Painted texture detail (like the runes) does not survive one-colour-per-face. Shapes do.
- [x] API batch (770 credits, 90 left): three Guardian figures (guardian_1-3), boiler, stone head, sky castle, crystal formation. All converted, placed and checked in screenshots.
- [x] Tests: new and replaced models are registered in testdata/model-topology-reference.json; the renderer hash test keeps projecting the original bank (originals of replaced models in testdata/projection-bank).
- [ ] Studio models (user generated 8 in Tripo Studio: oak, glowing tree, glowing mushrooms, flowering bush, grass and flower clump, desert palm, temple gateway, windmill). Waiting on the user to sign in to Tripo in the Claude desktop app's browser pane, or to share the GLBs.

## H. Features, only where they fill a gap

93. [ ] Elite variety.
94. [ ] Shrine events.
95. [ ] Synergy sets shown in the build panel.
96. [ ] Seed sharing: copy-code button.
97. [ ] Reduce-motion option tied to Effects intensity.
98. [ ] Photo mode frames and filters.
99. [ ] Steam: rich presence for the new systems; cloud save check.
100. [ ] Store screenshots and trailer re-captured after the art pass (the trailer still shows the neon world).

## Session 27-28 notes
- Item 10 (Crystal Caverns sky + floor): DONE. Painted backdrop and Voronoi-cracked glow floor replace the neon grid; verified in world3 and boss3 shots.
- Grass: one triangle per blade, baked per chamber. Bench vs no-grass Session24: Grove -2.6%, Meadow -2.3%, boss -1.8%, Ruins -3.5%, so Ruins scrub is trimmed 45 to 33 tufts.
- Cycle 20-21: Sunset Meadow, Ancient Ruins and Cloud Kingdom floors repainted. The old checkerboard grids became bevelled sandstone flagstones with mossy joints, sand-drifted sandstone blocks with carved turquoise roundels, and white marble slabs with gold inlay and small lapis diamonds. Generators are in tools/paint (meadow_floor.py, ruins_floor.py, clouds_floor.py) and the originals are backed up.
- Meadow: dry-stone field walls at x ±29 and flower beds by the fence (skipped on Low).
- Machine Works: molten-iron runnels in stone channels under the copper pipes, with drifting slag, flickering heat glow and light pools on the floor.
