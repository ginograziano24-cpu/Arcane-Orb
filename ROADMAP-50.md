# Arcane Orb: the 50 most pressing improvements

Written 2026-11-11. Gameplay, feel, balance, UI, audio and Steam
readiness only; the opening cinematic and the 3D models are out of scope
here. Ordered by how much each item matters to a player right now; work
top-down. Mark items `[x]` when done, with the date and a word on how.

Legend: **PC** = needs the PC (build, play or benchmark) to finish or verify.

## A. Must work, every run

1. [x] **Compile and test the Nov 11 changes on the PC.** `go vet ./...`,
   `go test ./...`, build, play one full run. (PC) (2026-11-11: vet, test and build all passed on the PC this morning (test-results.txt). The full-run playtest is still yours)
2. [x] **Anti-stall: no orb loops forever.** An orb bouncing between
   walls and unbreakable bricks without touching the paddle or hurting a
   brick should be nudged out of the loop, and no orb should crawl
   sideways at a near-flat angle. (2026-11-11: `antistall.go`: after 9 s without a paddle touch
   or a brick hurt, each wall bounce turns the orb ~9° towards the paddle;
   `antistall_test.go`.)
3. [ ] **Chamber-entry hitch.** 49.6 ms inside DX11 Present on entering a
   chamber (11-10 profile). Trace with ETW, then fix or hide it behind
   the transition. (PC)
4. [ ] **Memory over a full campaign.** Resident and GPU memory after
   three acts and several returns to the menu. (PC)
5. [ ] **Low and Medium presets on weaker hardware**, including Steam
   Deck class: 60 fps in boss and multiball scenes. (PC)
6. [ ] **Save and resume** mid-chamber, mid-cache-choice and mid-boss:
   nothing lost, nothing duplicated. (2026-11-11 code audit: a quit
   mid-chamber, mid-cache or mid-boss resumes from the snapshot taken as
   the chamber was entered, so nothing can be duplicated; items and gold
   picked up mid-chamber are rolled back with it, by design. Fixed: a
   resumed run forgot the origin's mastery, which hid origin items and the
   livery; `runsave_test.go`. Left: one quit-and-continue of each by hand. PC)

## B. Feel of the core loop

7. [ ] **Hit-stop and shake scale with impact size**, so small hits stay
   crisp and big ones land, and multiball never turns into constant shake.
   (Audit 2026-11-11: every shake uses max(), never adds, and decays at
   2/s, so it can't build up; what's left is tuning by feel. PC)
8. [x] **Paddle angle readability:** a faint aim guide on the paddle's
   edge zones while the orb approaches (option, off by default). (2026-11-11: `aimguide.go`, Settings > Effects > Aim guide, off by default: a dotted ray from where the incoming orb will land, at the angle it will leave, using the paddle's real angle rule; only for the first orb to arrive, only in the last stretch, only if it will land on the paddle)
9. [x] **Multiball clutter:** cap simultaneous particle bursts per frame
   and dim secondary orbs' trails so the main orb stays readable. (2026-11-11: `burstBudget` in play.go: per frame the first 6 spark/debris bursts are full, the next 6 a third, the rest a sixth; extra orbs in multiball leave fainter, thinner trails (world.go))
10. [ ] **Armour readability in later chambers** (open item 11 from
    VISUAL-PERFORMANCE-20).
11. [ ] **Boss telegraphs** for every attack and the recovery window
    (open item 7). (PC to verify live)
12. [ ] **Chamber pacing per act:** wave count, advance speed and heat so
    act 1 chambers run about 90 s and act 3 about 150 s.
13. [x] **Last-brick hunt:** the paddle-bounce lean already exists; also
    mark the last few bricks. (2026-11-11: `lastbricks.go`: a bobbing
    chevron over each remaining required brick once the assist kicks in.)
14. [x] **Hero quips and combo commentary** (2026-11-11, quips.go).

## C. Balance

15. [ ] **Engineer runs ~20% faster than the others:** trim overclock.
16. [ ] **Act 3 timeouts in carried runs** (8 in the last sim): soften
    heat scaling in act 3.
17. [ ] **Origin spread ≤ 15%** in the campaign sim.
18. [ ] **No auto-pick items:** pick-rate audit from the sim.
19. [ ] **Gold economy:** income vs shop prices per act; nobody should be
    unable to afford anything in act 3, nor buy everything.
20. [x] **Curse severity audit:** every curse a real choice, none a run-ender. (2026-11-11: all eight are mild stat penalties, none a run-ender. Renamed "Lattice's Weight" to "Stubborn Bricks" (players won't know "Lattice"))
21. [x] **Ascension tiers:** check each tier adds one clear rule. (2026-11-11: each tier adds one rule. Fixed: Final Form promised bosses that attack faster but only raised health; now volleys come every 4.8 s instead of 6 (`bossAttackGap`, meta.go). Renamed tier 4 "Fortified" to "Thick Walls" (the omen has that name). Test: meta_audit_test.go)

## D. Content

22. [ ] **Prism Colossus gets its own kit** (rotating shield, refracting
    prisms, mirror cores).
23. [ ] **Crucible Golem gets its own kit** (molten rows, vents that pull
    the orb, overheat stun).
24. [ ] **Crystal Warden gets its own kit** (frozen lanes, reflecting
    crystals, ball-steal phase).
25. [ ] **10–15 new items that change how you play** (ball behaviour,
    paddle tricks), with sims.
26. [ ] **Cross-origin synergies** for the new items.
27. [x] **More events**, in the same funny voice as the troll and the goat. (2026-11-11: seven new events in `events3.go`: the Snack Dragon, a Bard With Opinions, the Duckling Parade, Fumbleton's Hat, the Slime Spa, a Very Ambitious Rock, the Fortune Cookie)
28. [x] **More omens and chamber challenges.** (2026-11-11: Starlit (+15% crit), Critter Catcher (pop 3 critters, 45 gold), Edge Artist (12 paddle-edge hits, a scroll), in omens.go)
29. [ ] **Elite brick variety:** two new elite behaviours per world.
30. [ ] **Cameo challenges** at chamber select (locked heroes as a teaser).
31. [x] **Endless loop modifiers:** each loop adds a named twist. (2026-11-11: `loops.go`: each loop adds one named twist that stacks and cycles: Powder Kegs, Iron Bark, Twin Moons, Critter Season, Crystal Bloom; the act intro names the new one)

## E. Interface

32. [x] **Chamber select Build panel (B2):** items grouped by rarity,
    hover card, "Your stats" with base vs current values. (2026-11-11: `buildpanel.go`: groups (fusions, Mythic down to Common, curses) with counts, hover shows the item's card beside the panel, and a "Your stats" box: base in grey, now in green)
33. [x] **Reward cards show the stat change** the card would make. (2026-11-11: `statdelta.go`: under each reward card, the two biggest number changes ("Orb damage 1.5 → 2") worked out on a copy of the build, plus any synergy it would switch on)
34. [x] **Pause menu shows the build and stats.** (2026-11-11: the pause screen's Build panel now carries the "Your stats" box (#32))
35. [x] **Run recap highlights the best moment** (biggest combo, the
    item that did the most damage). (2026-11-11: a gold band on the summary: best combo and where (world or Guardian, act), and the star damage source with its share)
36. [ ] **Controller pass on every screen**, including focus rings and
    lock text. (PC)
37. [x] **Keyword tooltips everywhere** a keyword appears. (2026-11-11: every tooltip now explains up to three keywords it uses, like cards do; Burn no longer matches laser items that "fire")
38. [ ] **4K and ultrawide layout check** for every screen. (PC)
39. [x] **Quip voice and bubble setting** (2026-11-11, Settings > Audio).
40. [ ] **First-run onboarding:** fewer, better-timed tips.
41. [ ] **Colour-vision pass** for every status (open item 17).

## F. Audio

42. [x] **Sculpted sounds for every pickup and moment** (2026-11-11,
    sfx_sculpt.go).
43. [x] **Soundtrack selection** with named tracks (Settings or Codex).
    The six chamber songs already carry titles (`mus.Title`). (2026-11-11: Codex > Soundtrack: all eleven songs by name and place, Listen/Stop; an act's songs unlock once a run reaches that act)
43b. [x] **Music changes from chamber to chamber, like Magic Ball.**
    (2026-11-11: six new sibling songs, two per act, same key and palette
    as the act song; `mus/song_chambers.go`, `chamberTrack` in audio.go.)
44. [x] **A separate Voice volume slider** for quips. (2026-11-11:
    Settings > Audio > Hero voices.)
45. [x] **Dense-fight mix:** limit same-sound stacking per frame and duck
    small hits under big ones. (2026-11-11: `mixDensity` in audio.go: in a
    60 ms window the first 3 small hits play, the next 3 at half volume,
    the rest drop; small hits sit 30% lower for 250 ms after a big moment.)
46. [ ] **Per-world brick material sounds** (the on-hold #128 rework).
47. [ ] **Boss phase music** sections.

## G. Steam readiness

48. [ ] **Achievements and stats** for the new content (quips not needed;
    bosses, cameos, endless loops).
49. [x] **Duel stage select** (roadmap step 6). (2026-11-11: the host picks the stage in the duel lobby: a new world every round (as before) or one world for all rounds. Protocol version 2, so both players need this build)
50. [ ] **Steam Deck verified checklist:** controls, text size, suspend
    and resume, performance. (PC)

## H. Polish list (added 2026-11-11 by the user, 30 more)

Numbered on from 50. Same rules: `[x]` with a date and a word on how.

### Game feel
51. [ ] **Orb speed lines** when the orb passes its usual top speed.
52. [ ] **Brick crack stages:** one or two visible cracks as bricks lose health.
53. [ ] **Combo meter flourish:** pulse, colour step every 25 hits, shatter when it drops.
54. [ ] **Paddle squash and stretch** on catches, more on edge hits.
55. [ ] **Last-orb heartbeat:** vignette and low pulse on the last life with one orb.
56. [x] **Perfect-clear bonus:** a FLAWLESS stamp and stinger on the reward screen. (2026-11-11: `drawFlawless` in polish.go, sound `flawless`)

### Rewards and items
57. [x] **Card rarity reveal:** Epic and up flip in from face-down. (2026-11-11: with #92)
58. [x] **Fusion hints:** fusions one item away, as faint outlines in the Build panel. (2026-11-11: "One item away" group, `nearFusions`)
59. [x] **Item levels:** a second copy shows as "Heavy Core II" (cards and Build panel). (2026-11-11)
60. [ ] **Banish button:** once or twice a run, remove a card from the pool.
61. [x] **Shop sale:** sometimes one item 40% off with a bouncing tag. (2026-11-11: 45% of shops, `drawSaleTag`)
62. [ ] **Seed sharing:** copy-seed button and a "challenge a friend" text.

### Heroes and story
63. [ ] **Hero idle animations** on origin select.
64. [ ] **Hero banter** after a Guardian falls, in the run summary.
65. [x] **Letters between acts** from Archmage Fumbleton and from Mirabel. Mirabel's
    carry a lipstick kiss; either writer's letter has their picture, taped on as a photo
    or drawn faintly behind the writing. (2026-11-11: `letters.go`, 10 letters, a P.S.
    about your run, sounds `letter`, `scribble`, `kiss`. The pictures are the opening's
    art, `assets/opening/mirabel-*.png` and `fumble-*.png`; swap in Blender renders later)
66. [ ] **Hero-specific events:** one per hero.

### Map and progression
67. [ ] **Animated map paths:** the next path draws itself in, reachable rooms bob.
68. [ ] **Act-end stats card** ("Act I in numbers").
69. [ ] **Mastery cosmetics preview.**
70. [ ] **Daily streak** counter with small rewards at 3 and 7 days.

### Bosses
71. [ ] **Boss health phases** with slow motion and a roar at each threshold.
    (2026-11-11, part: rage marks at ⅔ and ⅓ on the bar, a flare and "ENRAGED!" when
    crossed. The slow motion waits for the boss kits, #22–24)
72. [ ] **Boss defeat replay** of the final hit in slow motion.
73. [ ] **Boss taunts** in a speech bubble.

### Interface
74. [ ] **Damage numbers option:** off, small or big.
75. [x] **Quick restart** from the summary with the same origin and challenge, plus
    "Same seed". (2026-11-11: "Play again" already kept both; "Same seed" added)
76. [ ] **Settings search.**
77. [ ] **Photo mode** on pause.

### Audio
78. [ ] **Music stingers on the beat.**
79. [ ] **Low-pass on pause.**

### Steam
80. [ ] **Steam Cloud saves and trading cards.**

## I. Visuals, performance and content (added 2026-11-11, 33 picked by the user)

Numbered on from 80. The user turned down: new brick types, mini-boss
chambers, treasure goblin chambers, puzzle chambers, a curse-only
challenge run, more letters (hero replies), and a New Game+ fourth-Guardian
twist. Don't re-propose those.

### Visuals: the board
81. [x] **Brick shatter by material:** glass splinters (Crystal Caverns), wood splits (Sunset Meadow), gears (Machine Works). (2026-11-11: already in game: `debris` in play.go breaks each world's bricks in its own material (crystal shards, wood splinters, stone and dust, metal and sparks, cloud puffs))
82. [x] **Orb-lit bricks:** bricks near the orb catch its colour. (2026-11-11: `orbLight` (visualfx.go), bricks within 4 units take up to 22% of the orb's colour)
83. [x] **Impact decals:** scorch, frost or spark marks on the floor that fade. (2026-11-11: `addMark`: blasts, crits; fading scorch with a coloured heart, at most 28)
84. [x] **Wave transitions:** the next wave drops in with a bounce. (2026-11-11: already in game: new bricks fade and drop into place (`Appear`, `FallY`))
85. [x] **Element trails:** fire, frost, storm and void each shape the orb's trail. (2026-11-11: `updateTrails`: embers (fire), glitter (frost), crackle (storm), smoke (void); less on extra orbs)
86. [x] **Paddle charge glow** as Overclock or Parry builds up. (2026-11-11: `drawPaddleCharge`: a pool of light under the paddle grows with Overclock, Rage, Parry or Attunement and pulses when ready)
87. [ ] **Living backgrounds:** one small animated story moment per world.
88. [x] **Weather per chamber** as a rare variant (rain, petals, snow). (2026-11-11: `pickWeather`: about 1 chamber in 6 (never a Guardian's) gets rain, petals or snow suited to its world)
89. [x] **Victory confetti** in the act's colours after a Guardian. (2026-11-11: `confetti` when a Guardian falls, in the world's colours with gold flecks)
90. [x] **Day-to-night shift** across each act's chambers. (2026-11-11: morning light → golden afternoon → dusk/night at the Guardian, a mild grade over each world's; Settings > Access can turn it off; `style.go`)

### Visuals: interface
91. [ ] **Animated menu background** with the seven heroes.
92. [x] **Card backs per rarity**, seen during the deal-in. (2026-11-11: reward cards rise face down on a back in their rarity's colour and turn over; Epic and up flash as they turn (also covers #57))
93. [x] **Damage recap chart:** damage per chamber on the summary. (2026-11-11: `drawChamberChart` on the summary, the best chamber in gold)
94. [x] **Animated map icons** (shop sign swings, campfire flickers). (2026-11-11: reachable rooms bob, the shop sways, the campfire flickers, treasure glints, the Guardian's room breathes)
95. [ ] **More HUD portrait reactions** (low life, jackpot). (2026-11-11, part: the portrait cheers a Gambler jackpot; new expressions need art)
96. [x] **Visual style setting:** classic, high contrast, calm. (2026-11-11: Settings > Access; grades the world only, calm also halves shake; works on Low through a colour matrix)

### Performance
97. [x] **GPU timing overlay** on the debug key. (2026-11-11: F3 shows GPU+present time, CPU/GPU-bound and GPU image memory; the frame graph splits game thread vs GPU)
98. [x] **Texture atlas** for UI icons and emblems. (2026-11-11: Ebitengine already packs the 192 px emblems into its shared atlas; added baking every item and fusion emblem on the main menu so no screen stops to draw one)
99. [x] **Cache static card text** as images. (2026-11-11: line wrapping was already cached; text widths are now cached too. Glyphs already draw from the engine's glyph atlas, and the reward screen's whole draw is 0.13 ms, so images would add memory for no gain)
100. [x] **Particle pooling.** (2026-11-11: already done: particles are removed in place (swap-delete), no per-frame allocation)
101. [x] **Cheaper shadows on Low.** (2026-11-11: Low skips the per-brick shadow decals; orbs, drops and paddle keep theirs)
102. [x] **Frame pacing at 144/240 Hz**, and a 30 fps battery cap for Steam Deck. (2026-11-11: 30, 165 and 240 fps caps; a shorter spin-wait above 200 Hz; the 30 fps cap works with VSync on)
103. [x] **Load-time trace** to the main menu; trim the slowest step. (2026-11-11: `-boottrace` and every bench report list each startup step. PC: 0.54 s to the first menu frame; the game's own setup is 29 ms and ~505 ms is Windows/DirectX opening the window, so there is nothing worth trimming on the game's side)
104. [x] **Memory after 10 runs** in a row. (2026-11-11: `-soak` plays 10 autopilot runs and reports the heap after each to bench_soak_TAG.txt)
105. [x] **Benchmark mode in the menu:** 60 s scripted scene, average and worst frame times. (2026-11-11: Settings > Display > Run benchmark: six chambers with your settings, a results card; earns nothing and saves nothing)

### Content
106. [x] **Secret chambers:** a hidden door on the map after a flawless clear. (2026-11-11: 40% after a flawless fight, one per act; a gold-crystal vault beside the next row with a Rare+ relic; `secret.go`)
107. [ ] **Two new origins:** e.g. a Bard (combo plays music), a Necromancer (broken bricks return as allies).
108. [x] **Weekly mutator run** with fixed rules for everyone. (2026-11-11: on the Challenges screen; ISO-week seed and two of eight mutators, best act per week saved, first win 100 shards; `weekly.go`)
109. [x] **Bestiary** in the Codex: critters, elites, bosses, funny descriptions. (2026-11-11: Codex > Bestiary: five critters, four special bricks, the three Guardians (shown once a run reaches their act))
110. [ ] **Hero friendship scenes:** comic panels for winning with each pair.
111. [ ] **Ending variations** by the hero who won the final fight.
112. [ ] **Trophy room:** achievements as objects on shelves.
113. [x] **Collectible orb skins** from challenges (cookie, frog, glitter). (2026-11-11: picker on the Challenges screen; cookie = seal 1 challenge, frog = seal 3, glitter = win a weekly run)

## J. Twenty more (2026-11-11 evening; 13-17 not chosen for now)

114. [x] **Paddle squash and stretch** on a hard return, with a very slight arcane crackle where the orb struck (not dust). (`feel.go`)
115. [x] **Brick crack stages** at 66% and 33% health (not on steel).
116. [x] **Slow-motion final hit:** already in the game (startFinale: slow motion, camera lean, shockwave).
117. [x] **Combo heat:** the number goes white, gold, orange, then fire, with embers past 10.
118. [x] **Guardian death sequence:** its armour shatters from the core outward over 1.8 s.
119. [x] **Seasonal map dressing** from the PC's date: snow, petals, fireflies, leaves (fewer on Low).
120. [x] **Run history graph** in Codex > Run History: last 20 runs, act bars and a damage line.
121. [x] **Build ideas:** after taking a card, a toast names up to 3 items that fuse or share its school.
122. [x] **Pause screen "In effect" panel:** the chamber's omen, the challenge or the weekly mutators (the build panel was already there).
123. [x] **Gamepad:** rebinding already existed; added a left-handed preset (right stick steers, d-pad buttons) in Controls.
124. [x] **Damage numbers coloured by source** (orb, fire, lightning, blasts...), Settings > Effects; at most 12 start per frame.
125. [x] **Quick restart:** hold R on the defeat screen too (it worked in chambers already); weekly runs restart the week's run.
126. [ ] Shrine events (not chosen yet)
127. [ ] Elite affixes (not chosen yet)
128. [ ] Item synergy sets (not chosen yet)
129. [ ] Daily login streak (not chosen yet)
130. [ ] Endless milestone cosmetics (not chosen yet)
131. [x] **Ready the next chamber while choosing:** on the map and reward screen, the reachable chambers' worlds' textures and contact sounds are warmed. (`prewarm.go`)
132. [x] **Automatic quality on first launch:** a 24-frame off-screen test at High picks Low/Medium/High; never on a save with runs or on the Deck. (`autoquality.go`)
133. [x] **Crash-safe mid-chamber autosave** every 15 s: broken bricks, wear, Guardian health, barrier and clock; Continue rebuilds the chamber and lays it back on (first formation only). (`chambersave.go`)

## K. Release readiness (2026-11-11 night)

134. [x] **Steam Cloud saves:** nothing to code; Auto-Cloud settings for save.json and run.json are in STEAM.md section 4 (set them in Steamworks).
135. [x] **Controller button names match the pad:** Xbox letters, PlayStation (Cross/Circle/Square/Triangle, L1/R1, Options), Nintendo (positional B/A/Y/X, L/R/ZL/ZR) and Steam Deck (L1/R1, View/Menu), following the last pad used. (`padglyphs.go`)
136. [~] **Steam Deck Verified pass:** text starts one size up at 1280x800 (and other small screens) unless chosen; on-screen keyboard for seeds and auto-pause on suspend already existed. Needs a Deck to confirm.
137. [x] **Achievement audit:** a test checks every achievement can be unlocked by some code path, the Steam CSV lists exactly the game's achievements with both icons, and benchmark runs earn nothing. (`achaudit_test.go`)
138. [x] **Crash reporter:** already existed (crash.log beside the save, a dialog, the run and settings); now also logs the graphics path, screen size, Deck and uptime.
139. [x] **Store capture / photo mode:** Pause > Photo mode freezes the chamber, hides the HUD, orbits and zooms the camera, and saves full-resolution PNGs to Pictures/Arcane Orb. (`photomode.go`)
140. [x] **Fumbleton's Chronicle:** a Codex tab with a foreword and pages for the six worlds and three Guardians, opening as you reach them. (`chronicle.go`)
141. [x] **Auto-launch** (one-switch play), Settings > Controls. A dyslexia-friendly font needs a font file that isn't in the project yet (OpenDyslexic is free; add it to assets/fonts and it can be wired in).
142. [x] **Audio:** the music dips when the paddle breaks, and a quiet heartbeat plays when you drop to your last life. Big moments already ducked the music.
143. [x] **Folder tidy:** benchmark files, old builds, old scripts, backups and the plain opening WAVs moved to Archive-2026-11-11 (nothing deleted).
144. [x] **Smaller build:** the opening's three stems are zlib-packed (narration as mono): 21.6 MB -> 8.6 MB; the 28 HUD portraits losslessly recompressed: 10.5 MB -> 8.0 MB. About 15 MB off the exe.

## L. Run variety and pacing (2026-11-12)

145. [x] **Power pacing:** in act 1, ordinary offers (cards and shops) rarely show Epic items in the first 3 chambers, and Legendary/Mythic not before chamber 4/8; from act 2 nothing is held back. Promised rarities (elites, Guardians, treasure) are never held back. (`run.go` rarityGate)
146. [x] **Keepsakes:** 10 charms unlocked by play (coin, ward, feather, whetstone, ember, token, clover, ring, egg, knot), one carried per run, each about as strong as one Common card. Not in daily, weekly or challenge runs. (`keepsakes.go`)
147. [x] **Trials (custom difficulty):** 8 rules worth 1-2 points each, +6% shards per point, stack with Ascension; best won trial total saved. (`trials.go`)
148. [x] **Banish and lock:** 3 banishes per run (+2 with the Banisher's Ring) on reward screens; lock up to 2 shop items through a reroll. (`banish.go`)
149. [x] **Familiars:** 7 new items: Wisp Lantern, Ember Sprite, Frost Moth, Warden Owl, Familiar Bond, Spirit Banner, Spirit Lure; the support ones only appear once you have a familiar. About 5-8% extra damage each. (`familiars.go`)
150. [~] **Couch co-op:** player two on a controller plays a second paddle (both 80% width, bricks +15% health). Needs two people to playtest. (`coop.go`)

## M. Visual pass (2026-11-12)

All of it is a few sprites, rings or particles; nothing new is drawn per brick (`visfx9.go`).

151. [x] Familiars shed drifting motes (Full particles only).
152. [x] Familiars arrive with a burst of their colour at the start of a chamber.
153. [x] Familiar shots leave a spark where they land.
154. [x] Warden Owl catch: feathers and a soft ring.
155. [x] Guardian's Knot save: a gold ring sweeps the arena (shake only if Screen shake is on).
156. [x] Barrier bounce ripple where the orb hits the barrier.
157. [x] Critical hits flash a quick ring (thinned in busy frames).
158. [x] Falling coins sparkle (Full particles only).
159. [x] Last life: the screen edges pulse red with the heartbeat, stronger while an orb drops (half strength with Flash effects off).
160. [x] HUD keepsake charm under the lives (fades once the knot is spent), with a tooltip.
161. [x] HUD trial embers: one flickering ember per trial point, with the trial names on hover.
162. [x] Banished cards burn away in purple flame.
163. [x] Legendary and Mythic reward cards stand in a soft column of light.
164. [x] Locked shop items wear a gold chain badge.
165. [x] The Trials button on the origin screen glows warmer as more points are on.

## N. Performance pass (2026-11-12)

166. [x] **Window icon fix:** in fullscreen, Ebitengine kept the icon request pending and re-converted all four icon images pixel by pixel every frame (92% of all allocations, ~18% of CPU). The icon is now set only once the game is windowed. Benchmark (two runs each, 2560x1440 fullscreen): +32% to +163% fps in every scene (worlds +42-83%, classic +32%, menus about double), 1% lows up 25-100%, allocations per frame from ~17,500 to 100-300. (`main.go` syncWindowIcon)

## O. Performance pass 2 (2026-11-12)

167. [x] **Model drawing (drawShared):** each triangle's quad is filled in place instead of built and copied, lighting is inlined with the same float steps and rounding (colours bit-for-bit unchanged; the projected-model hash test guards it), and screen positions are stored once as float32. drawShared was the largest game-side cost (16% of CPU) after the icon fix. Result: within noise (-1.5% to +2.1% per scene; classic backdrop +2%), because frames are now mostly waiting on the GPU driver (cgocall ~43% of CPU). Further gains need fewer draw calls or quads, not faster CPU loops.

## P. Visual pass 2 (2026-11-12)

168. [x] The Ember Sprite flickers.
169. [x] The Frost Moth flutters.
170. [x] Arcane Wisps leave two fading afterimages.
171. [x] The Warden Owl brightens over its last 3 seconds of rest and pulses when ready.
172. [x] Player two's paddle streaks when it moves fast.
173. [x] The chosen keepsake has five lights circling it in the picker.
174. [x] Locked keepsakes show a small chain.
175. [x] Trials that are on get a flickering ember line.
176. [x] The Trials button sends up embers at 4+ points.
177. [x] The Co-op button glows in player two's colour when on.
178. [x] In banish mode the hovered card gets a flickering purple outline.
179. [x] Shop items catch a sweep of light when stocked or rerolled.
180. [x] The shop's gold counter glows when your gold changes.
181. [x] The HUD keepsake flares when it acts (Spare Ward barrier, Guardian's Knot save).
182. [x] On the last life the remaining heart sends out a ring with each beat. (`visfx10.go`) Measured cost of section P: none outside noise.

## Q. Performance pass 3 (2026-11-12)

183. [~] **Re-profile, no code change.** Correction to section O: the ~43% "cgocall" time is mostly the audio thread idling in WaitForSingleObject (it runs even with audio off in the benchmark), not the GPU. The frame's real cost is the main thread: Batch.Flush (planning ~4%, Ebitengine's DrawTriangles ~5%) and the parallel scenery tasks (walls, bricks, props via drawShared). Walls are re-projected every frame because the camera follows the paddle, so caching them would only help when the paddle is still. The run planner is already tuned; nothing safe and visual-neutral was left to change this round. Next idea worth a try: cache a wall section's quads keyed on the exact camera, and measure how often it hits during real play.

## R. Visual pass 3 (2026-11-12)

184. [x] Rerolling the shop throws a burst of gold sparks from the button.
185. [x] Buying a life sends green sparks up from the button.
186. [x] Buying a scroll bursts in the scroll's colour.
187. [x] Skipping a reward tosses gold sparks upward.
188. [x] The Banish button shows one purple pip per banish left.
189. [x] The trial total on the Trials panel heats from amber to red as it climbs.
190. [x] Epic reward cards get a faint glow (Legendary and Mythic keep the light column).
191. [x] The card dealt in after a banish glows in purple.
192. [x] Familiars' glow breathes faster with Familiar Bond.
193. [x] Spirit Banner: a soft ring of light under the paddle, wider per kind of familiar.
194. [x] The ready Warden Owl's eyes glint every few seconds.
195. [x] Player two's launch flashes their paddle.
196. [x] Side by side, the two co-op paddles share a soft bridge of light.
197. [x] Trial embers on the HUD flicker faster in Guardian chambers.
198. [x] The shop's lock badge sways gently. (`visfx11.go`) Measured cost of section R: none outside noise.

## S. Performance pass 4 (2026-11-12)

199. [x] **Baked models:** a model drawn at the same place, size, turn and tint two frames running has its world-space corners, face normals and lit colours worked out once and kept; each frame only projects the corners and fills the quads. Moving things are never baked; unused placements are dropped after about 2 seconds; colours rebake if the light changes. Bit-for-bit the same picture (TestBakedMatchesShared compares every quad). Result (two runs each): classic backdrop +17% fps (its 3D work 1.3 ms -> 0.9 ms), worlds 3-6 +1-2%, world2's 3D work 0.54 -> 0.46 ms with fps unchanged, everything else within noise. (`meshbake.go`)

## T. Visual pass 4 (2026-11-12)

200. [x] Keepsake cards glow under the cursor in the picker.
201. [x] Choosing a keepsake gives a gold burst.
202. [x] The chosen keepsake's emblem sits beside the origin screen's Keepsake button.
203. [x] Switching a trial on throws up embers.
204. [x] "Clear all" on the Trials panel puffs the embers away.
205. [x] Turning co-op on bursts teal.
206. [x] Entering banish mode bursts purple from the button.
207. [x] Locking a shop item gives a small gold burst.
208. [x] On your last life the shop's "Regain a life" button pulses green.
209. [x] HUD familiar dots: one per familiar in its colour (the owl's dims while it rests).
210. [x] Familiars flash as they fire.
211. [x] Ember Sprite shots leave embers rising off the brick.
212. [x] Frost Moth shots ring the frozen brick with frost.
213. [x] The Warden Owl waking sends a soft ring round the paddle.
214. [x] Coins player two catches sparkle teal. (`visfx12.go`) Measured cost: none outside noise.

## U. Keepsake upgrade tree (2026-11-12, asked for)

215. [x] **Keepsakes menu** (main menu > Keepsakes, or the origin screen's Keepsake button): a tree of the 10 keepsakes bought with shards. Each has 2-6 small ranks (rank n costs base × n shards); deeper keepsakes need 2 ranks of the one above. Equip one; it's carried into every run that allows personal rules. Full ranks equal the old keepsakes, so the cap on power is unchanged; the whole tree costs 2,452 shards. Old saves get rank 1 of every keepsake they had unlocked. (`keepsaketree.go`, `keepsakes.go`)
    - Row 1: Lucky Coin (+6 gold/act per rank, 5), Feather Charm (+1% paddle, 6), Whetstone (+0.04 damage, 5)
    - Row 2: Trader's Token (−2% prices, 4; after Coin), Spare Ward (barrier in act 1 / acts 1-2 / every act, 3; after Feather), Banisher's Ring (+1 banish, 2; after Whetstone)
    - Row 3: Clover Pin (+3% luck, 5; after Token), Ember Locket (+1% ignite, 4; after Ring)
    - Row 4: Wisp Egg (wisp at 25/38/50%, 3; after Clover), Guardian's Knot (keep 20/35/50% gold, 3; after Ward)

## V. Performance pass 5 (2026-11-12)

216. [ ] **One-pass vertex fill (tried, reverted):** the flush used to copy every quad's corners into the vertex buffer and then walk the whole buffer again to premultiply colours and scale; it now does both while each quad is in cache. Same arithmetic, but it measured 4-7% slower in every chamber (Session13 vs 14, flush 0.62 -> 0.67 ms in world1): the bulk copy beats a per-vertex copy. Reverted in cycle 6; TestFillVertsOnePass still guards the two-pass result.

## W. Visual pass 5 (2026-11-12): the Keepsakes tree and more

217. [x] A faint field of twinkling stars behind the tree.
218. [x] Buying a rank swells the node with light.
219. [x] The gem for the rank just bought flashes.
220. [x] When a branch opens, a light runs down its link.
221. [x] Small lights drift down the links that are open.
222. [x] Open keepsakes you haven't bought yet breathe softly.
223. [x] A shard glint marks keepsakes whose next rank you can afford.
224. [x] Fully upgraded keepsakes have a slow gold sheen.
225. [x] Hovering a locked keepsake outlines the one it needs.
226. [x] The shard counter glows when shards are spent.
227. [x] The equipped keepsake has lights circling it in the tree.
228. [x] The detail panel's emblem sits in a soft glow (gold when maxed).
229. [x] Pressing Equip rings the emblem in gold.
230. [x] As a chamber starts, the HUD keepsake pulses once so you remember what you carry. (`visfx13.go`)

## X. Performance pass 6 (2026-11-12)

231. [x] Reverted the one-pass vertex fill (see 216). Result (Session14 vs 15, two runs each): chambers back up 3-7% (world1 +6%, classic +5%, Guardian +7%), level with Session13; menus unchanged. The 15 visuals in Y cost nothing measurable.

## Y. Visual pass 6 (2026-11-12)

232. [x] Relentless Guardians trial: the Guardian's core sits in a slow red glow.
233. [x] A Spare Ward barrier shines gold for the chamber's first seconds.
234. [x] Familiar Bond: faint threads of light join the familiars.
235. [x] The Wisp Egg's young wisp is smaller and paler until a real one joins.
236. [x] Familiars drift up and fade once a chamber is cleared.
237. [x] The co-op paddles spark when they bump.
238. [x] A teal ring marks player two's paddle as each chamber starts.
239. [x] The Guardian's Knot washes the screen gold (softer with Flash effects off).
240. [x] Lucky Coin: coins tumble past the HUD as an act's first chamber starts.
241. [x] Shop items stamp SOLD when bought.
242. [x] HUD trial embers flare as a Guardian chamber begins.
243. [x] The HUD owl dot shows its rest as a closing ring.
244. [x] A shard gem on the origin screen's Keepsake button when an upgrade is affordable.
245. [x] The spent banish pip pops away.
246. [x] Skip dims while banish mode is on. (`visfx14.go`)

## Z. Performance pass 7 (2026-11-12)

247. [~] **Looked, nothing changed.** Brick bodies (drawTexBrick) already skip unreachable lights and run on side goroutines; the run planner is tuned; wall models are baked. No exact, visual-neutral change worth its risk was left this round. Next real gain needs fewer quads (a design call, not a free speed-up).

## AA. Visual pass 7 (2026-11-12): the run's charms between chambers

248. [x] Map run panel: the keepsake emblem sits in the corner (hover for its rank and effect).
249. [x] Map run panel: trial embers beside the seed line.
250. [x] Map run panel: the familiars circle the hero portrait.
251. [x] Map run panel: a teal second glow under the hero in co-op.
252. [x] Pause screen: keepsake, trial embers and banishes left under the seed line.
253. [x] Act intro: the carried keepsake rises into view under the act's name.
254. [x] Act intro: with trials on, embers rise past the title.
255. [x] Main menu: a shard gem beside Keepsakes when an upgrade is affordable.
256. [x] Shop: with the Trader's Token, a gold sheen crosses the heading.
257. [x] Rewards: with the Clover Pin, green motes drift over the cards as they're dealt.
258. [x] Rewards: with the Banisher's Ring, its emblem sits by the Banish button.
259. [x] Chambers: with the Feather Charm, the paddle's ends gleam as each chamber starts.
260. [x] Keepsakes tree: a crown of light over the title once every rank is bought.
261. [x] Trials panel: the backdrop warms with the points on.
262. [x] Origin screen: a teal glow under the hero when co-op is on. (`visfx15.go`) Result (Session15 vs 16): every scene within run-to-run noise (-2% to +5%, one slow world1 run); tests pass.

## AB. Performance pass 8 (2026-11-12)

263. [~] No change this round (see Z): the cycle went straight to visuals, as asked, to keep the cycles quick.

## AC. Visual pass 8 (2026-11-12): run end, events, campfire, toasts

264. [x] Run end: the keepsake and trial embers sit under the title.
265. [x] Run end: in a win with trials on, the embers rise in triumph.
266. [x] Run end: in a win, the familiars circle the hero's portrait.
267. [x] Run end: a co-op run shows player two's teal glow by the portrait.
268. [x] Run end: in defeat, grey ash drifts down over the tally.
269. [x] Events: a gold glow behind a good outcome, a red pulse behind a bad one.
270. [x] Events: the sigil quickens while a choice is under the cursor.
271. [x] Campfire: fireflies drift over the scene.
272. [x] Campfire: hovering Rest stokes the fire's glow.
273. [x] Toasts: a light sweeps across each toast as it arrives.
274. [x] Toasts: the colour bar glows.
275. [x] Act intro: the familiars circle the act's sigil.
276. [x] Map run panel: banishes left as purple pips.
277. [x] Pause screen: the familiar dots beside the charms.
278. [x] Keepsakes tree: an arrow pulses between Now and Next rank. (`visfx16.go`) Result (Session16 vs 17): within noise (-1% to +2%); tests pass.

## AD. Performance pass 9 (2026-11-12)

279. [~] Skipped to keep the cycles quick (see Z): no exact, visual-neutral change left that measures.

## AE. Visual pass 9 (2026-11-12): pickups, elites, scrolls and the other menus

280. [x] Catching a power-up rings the paddle in its colour; a good one sends up a column of light.
281. [x] A bad power-up (Shrink, Haste) leaves dark puffs.
282. [x] A glowing tick on the paddle line shows where each falling power-up will arrive.
283. [x] A toppled elite leaves a violet shock ring and a burst.
284. [x] Reading a scroll unfurls rings of its colour round the paddle.
285. [x] Map: a light circles each chamber you can reach.
286. [x] Settings: the tab underline slides to the chosen tab.
287. [x] Settings: the panel flashes faintly when a switch changes.
288. [x] Codex: the tab underline slides to the chosen tab.
289. [x] Codex: a soft light sweeps down the page as a tab opens.
290. [x] Play a Seed: stars drift round the heading.
291. [x] Play a Seed: the box glows more with each character.
292. [x] Play a Seed: each new character pops with light.
293. [x] Challenges: motes in each challenge's colour drift up its card.
294. [x] Challenges: a sealed challenge's medallion has a slow gold sheen. (`visfx17.go`) Result (Session17 vs 18): within noise (-0.5% to +5%); tests pass.

## AF/AG. Brick emphasis (2026-11-12, asked for)

295. [x] **Bright brick edges:** a thin bright rim on each brick's top edges facing the camera (front edge, plus the side edges the camera sees), in a light version of the brick's own colour. Guardian armour is left as it is.
296. [x] **Scenery steps back:** a soft, cool, darker veil over everything outside the arena walls (sky, hills, props), feathered over about 70 px at the walls, so the board is the most vivid thing on screen. Skipped in photo mode.
297. [x] **Settings > Effects > Brick emphasis:** Subtle (default), Strong, Off. Untested by eye: judge it in game and say stronger, weaker or different. (`brickfocus.go`)
298. [x] Result (Session18 vs 19): menus within noise; dense chambers 2-4% slower in the steadier run (multiball -3.8%, world4 -2.9%), more in the noisier run. Trimmed in cycle 11: Subtle now draws only the front rim (one quad per brick); the side rims are Strong only.

## AH. Effects intensity (2026-11-12)

299. [x] **Settings > Effects > Effects intensity:** High (default, today's look), Medium, Low. One helper in `fxlevel.go`: `fxAmount()` (1 / 0.5 / 0), `ambientOff()` (Medium and Low), `flourishOff()` (Low).
300. [x] Medium turns off the ambient decoration from visfx9-17: drifting stars and motes, embers, fireflies, ash, shimmers, sheens, link flows, auras, the partner trail, coin glints.
301. [x] Low also turns off the one-shot flourishes: hover glows, pops, flashes, the sold stamp, coin rain, toast sweeps, celebratory rings and sparks (owl, knot, scroll, elite), screen sparks, the banish flame.
302. [x] **Gameplay cues are never cut:** pickup rings and bad-pickup smoke, crit ring, barrier ripple, last-life edge and heart ring, drop markers, ward barrier, familiar shots/dots/orbit, bond and partner links, banish pips, shop lock badges, rarity pillars, affordability and parent hints in the keepsake tree, run charms, settings/codex tab underline.

## AI. Sounds for the new systems (2026-11-12)

303. [x] `sfx_systems.go`, quiet (played at 50-70% of SFX volume): keepsake rank chime rising per rank, equip clasp / unequip, banish (falling breath under the shatter), shop lock / unlock latch, owl save "hoo-hoo", co-op launch plucks, knot (rope pulled taut + bell), trial on (ember catch) / off (tick). Covered by TestEveryRecipeRenders.

## AJ. Achievements for the new systems (2026-11-12)

304. [x] Heirloom (any keepsake at max rank), The Full Reliquary (every rank, with progress), Trial by Fire (win at Trial 5+), Two Wardens (win in co-op), Took What Came (win at Trial 3+ without banishing; Trial 3 so it isn't free on a first win), Menagerie (three kinds of familiar at once).
305. [x] Steam CSV rows and 256px icons (plus grey locked versions) in `store/achievements/`, drawn to match the existing icon frame; in-game glyphs gem, crown, flame, orb, ring, moon. Translated in all 6 languages.
306. [x] Tests: `cycle11_test.go` (fx dial, unlock rules). Its TestMain sets noPersist so no test can write the player's real save file (some earlier tests called writeSave).
307. [x] Result (Session19 vs 20): tests pass; chambers +1% to +4% (the Subtle rim now draws one quad per brick), menus within noise.

## AK. The 100-item polish pass (2026-11-12, asked for)

The list and its status live in `Source/POLISH-100.md`. Method: `Shots.cmd` (or `ArcaneOrb.exe -shots`) saves a picture of 31 screens to `shots/`; every visual change is judged from those pictures before it is marked done.

308. [x] First batch: the synthwave "Neon Grove" is now Glimmer Grove (new sky paintings, four painted floors, new scenery, dressing, stone walls and posts; lore name updated). The city skyline along the bottom of the map, reward and act-intro screens went with it. Painting scripts are kept in `Source/tools/paint/` (Python: numpy, scipy, Pillow).
309. [x] `menu.go` deleted (an old synthwave title scene nothing called). The Low quality preset now also sets Effects intensity Low.
310. [x] Cycle 13-16 (2026-11-12): Crystal Caverns and Machine Works floors and foundry sky; Sunset Meadow sky; Tripo models (standing stone, three Guardian figures, boiler, stone head, sky castle, crystal formation); sky-foot fill (no seam under any painting); grass and flowers; empty scroll slots. Status in POLISH-100.md.
\n