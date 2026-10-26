ARCANE ORB  -  build 0.7
=====================================

Run ArcaneOrb.exe. No install needed. Starts fullscreen at your
monitor's native resolution (F11 toggles windowed). Renders at native
resolution (1440p / 4K) and runs at your monitor's refresh rate.

CONTROLS
  Mouse ............ move paddle        A/D or Arrows ... move paddle
  Click / Space .... launch orb, fire   Shift ........... faster keyboard move
  Esc / P .......... pause              F11 ............. fullscreen
  Right-click / E .. class ability (Pyromancer, Stormcaller)

NEW IN 0.7  -  title screen and origins
  - New title screen: a hovering arcane orb rendered live by a shader,
    swirling violet energy, crackling surface arcs, a glass rim with
    glints, a flickering corona, an orbiting rune ring and drifting motes
  - Menu moved to a left-hand column beside the orb
  - New "Choose your Origin" screen: origin list on the left, a large
    character portrait in the centre, attributes, passive, active
    ability and unlock/Begin on the right (arrow keys + Enter work too)
  - Character portraits for all 7 classes, rendered by the included
    tools/portraits renderer (go run ./tools/portraits)

NEW IN 0.6  -  story, endless mode and classes
  - Story Mode: the three acts are now the main story, with a prologue,
    chapter cards between acts and an epilogue after the Crystal Warden
    (click to reveal, Esc to skip)
  - Endless Mode: beat the story once to unlock it from the main menu.
    The worlds loop forever with generated levels; every loop bricks get
    25% tougher, orbs 6% faster and dropped orbs hurt more. Each loop
    is worth 2,500 bonus score. You can also keep going straight from
    the victory screen.
  - Endless Records: a local top-10 leaderboard (Codex > Endless Records)
  - Two new classes with an active ability on Right-click / E:
      * Pyromancer: every 4th hit ignites, fire burns 50% hotter.
        FIRESTORM sets the two lowest rows ablaze (22s cooldown)
      * Stormcaller: every paddle bounce arcs lightning, lightning +30%.
        THUNDERSTRIKE smites the toughest brick (or the boss core) and
        chains to 4 more (18s cooldown)
    Ability medallion with cooldown ring on the HUD; new sounds for both
  - Class select redesigned to fit all 7 paddles

NEW IN 0.5  -  progression, items, fusions and sound
  - 102 upgrades and relics (46 new), each with its own hand-built emblem
    art shown on cards, the HUD, your build list and the Codex
  - 18 FUSIONS: own two specific items and they forge a new one with its
    own effect and visuals (Plasma Storm, Cryo Charge, Volcano, Gold Rush,
    Cataclysm, Absolute Zero, Tesla Coil and more). Cards show which item
    completes a fusion, and partners appear more often once you own one.
  - Arcane Rank: every run earns XP; ranking up unlocks new items into
    the pool (35 items unlock between ranks 1 and 10)
  - New Sanctum with 15 permanent upgrades, several ranks each
  - Ascension 1-10: win a run to unlock harder difficulty tiers with
    bigger shard and XP rewards
  - The Codex: a collection of every item and fusion you have discovered
  - Rebuilt sound: brick hits play melodies that climb with your combo
    in the key of the music, a signature instrument per world, layered
    impacts, rarity fanfares, fusion and synergy stingers, low-health
    heartbeat, and music that ducks under big moments
  - Balance: brick and boss health scale with depth, more shards per run

NEW IN 0.4  -  art and interface pass
  - Painted panoramic backdrops for every world, two variants each
    (wireframe mountains and a neon city; cumulus skies over farmland;
    crystal caverns with light shafts; desert mesas and a step pyramid;
    a sunlit cloud sea with floating islands; a refinery at dusk)
  - Low-poly 3D set pieces replace the old box decorations: faceted trees
    and oaks, a working windmill, lathe-turned columns and palms, sky
    castles and an airship, crystal clusters, boilers, pipes and gears
  - Bloom and per-world colour grading (toggle in Settings)
  - Redesigned interface, one consistent system across the game:
      * new HUD: health bar with damage trail, animated gold counter,
        barrier pips, combo meter, ornate boss bar, power-up timers,
        relic badges, keycap launch prompt
      * drawn icons for health, gold, barrier, shards and more
      * per-world framing: holographic glass (Neon), walnut and brass
        (Meadow), faceted crystal (Caverns), carved stone and gold
        (Ruins), ivory and gilt (Clouds), riveted steel (Machine)
      * restyled buttons, cards, map, shop, events and settings switches
      * notifications for relics, upgrades and synergies
      * smooth fade transitions between screens
  - New typefaces: Lilita One (titles) and Fira Sans (interface)

NEW IN 0.3
  - Six worlds, each with its own art direction, lighting, brick
    materials, destruction effects, paddle trim and hero set piece.
    Every chamber takes you to a different world than the last.
  - 27 hand-drawn showcase levels (castles, a cathedral, a ziggurat,
    a pagoda, animals, faces, a heart, star, sword, rocket, crown,
    planet and more), mixed in with generated levels about 1 time in 4

NEW IN 0.2
  - Stacked 3D brick structures: pyramids, towers, terraces, ramparts,
    battlements, dunes and domes. Break the base and the stack drops down.
  - 7 arena shapes: hall, octagon, gable, dome, bastion, notched, funnel

WHAT'S IN THIS BUILD
  - Procedural level generator: 16 base formations, combinable, symmetric
    or asymmetric, with reachability checks. Every level shows its seed.
  - 15 brick types: basic/reinforced, steel, armoured, explosive, chain,
    split, regenerating, moving, shield, teleporting, spawner, multiplier,
    crystal, mystery, elite
  - Roguelite run: 3 acts with branching maps (combat, elite, shop,
    treasure, event, rest, boss), 28 upgrades + 30 relics across 6
    rarities, 12 synergies (own 3 items with the same tag)
  - Three act bosses: Prism Colossus, Crucible Golem, Crystal Warden
  - 5 paddle archetypes (2 unlockable), Sanctum meta-progression (shards),
    seeded runs, daily challenge, combo system, procedural music that
    intensifies with your combo
  - Settings: volumes, screen shake, flashes, particles, VSync, FPS
    counter, bloom and colour grading

COMING IN LATER PHASES
  Enemies, more bosses, more events, collection screen, achievements,
  Steamworks hooks.

Save file: %APPDATA%\ArcaneOrb\save.json

LICENSES
  Engine: Ebitengine (Apache-2.0).
  Fonts: Lilita One by Juan Montoreano and Fira Sans by Mozilla /
  Carrois, both under the SIL Open Font License 1.1 (license texts in
  the source archive under assets/fonts/).
  All other art, audio and music were made for this game: backdrops are
  painted by the included tools/paint_backdrops.py, models and sounds are
  generated by the game's own code.
