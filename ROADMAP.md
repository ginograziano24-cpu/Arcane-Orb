# Arcane Orb: roadmap

Twelve ideas for later updates, and a plan to work through them.

## The ideas

| # | Idea | Area |
|---|---|---|
| 1 | More items, kept balanced | Content |
| 2 | Unique bosses that do more than throw projectiles | Content |
| 3 | More immersive lore for each origin | Story |
| 4 | General world lore: immersive but simple to understand | Story |
| 5 | Short origin quips: on selection, and when high-value things happen in a run | Story |
| 6 | A brief intro/showcase per origin, ideally a quick in-game demo of its abilities | Story / UI |
| 7 | Cameo challenges at chamber select: try a different character for a chamber | Modes |
| 8 | Chamber select redesign: simpler but charming (it looks stale now) | UI |
| 9 | A stage select for multiplayer | Modes / UI |
| 10 | A soundtrack selection, with the new music named and added | Audio / UI |
| 11 | Collectible origin art, unlocked by completing challenge chambers | Collectibles |
| 12 | Art scenes of the origins hanging out, dueling and so on (the collectible set) | Collectibles |

## How to work through them

- **One session per step.** Each step below is sized to fit one fresh session (see HANDOFF.md) and ends in a playtest build.
- **Story first.** The lore and the voices (steps 1–2) come first because the quips, showcases, cameos and art all draw on them. Writing them first means nothing gets rewritten later.
- **Cheap to expensive.** Steps run from cheap and low-risk to expensive. The bosses come last: they are the biggest piece of work and benefit from everything before them.
- **Translations cost the most.** Every new player-facing line needs all 7 languages, so each step lists roughly how much new text it adds.
- **Decisions marked ❓ are for you.** Each step can start as soon as its questions are answered.

### Step 1: Lore bible and origin lore (ideas 3, 4)

- **What I'll do:**
  - Write `docs/lore.md`, a one-page world primer (the Weavers, the Orb, the realms, the Wardens) in plain, short sentences.
  - Write half a page per origin: who they are, why they fight, how they relate to the others.
  - Reconcile both with what already exists: the story cards (story.go), the origin descriptions (run.go) and the act and boss names (biome.go).
  - In game, add a "Lore" page per origin, opened from class select and the codex. The prologue cards get a light pass so they match.
- **New text:** about 1,500 words (×7 languages).
- **Checks:** screenshots of the new pages in all languages, and a missing-translation sweep (`__trMiss`).
- ❓ **Your call:** tone. I'd go "warm fairy tale with a wink" to fit the synthwave look; the alternative is darker and more epic. Also: any origin facts you already have in mind.

### Step 2: Origin quips (idea 5)

- **What I'll do:**
  - Write 6–10 lines per origin, each a few words long:
    - on selection
    - at run start
    - on a big combo
    - on a boss kill
    - on a legendary item pickup
    - on the last life
    - on a win
  - Show them as a speech bubble next to the HUD portrait, which already has poses (hudportrait.go).
  - Add a cooldown so quips stay rare and never cover the play area. A setting turns them off.
- **New text:** about 60 lines (×7).
- **Checks:** a sim run with a quip log, to confirm the frequency is about 1 every 1–2 minutes; screenshots of the bubbles.
- ❓ **Your call:** text only (my recommendation), or also short synthesised "voice" blips like old JRPGs?

### Step 3: Chamber select redesign (idea 8)

- **What I'll do:**
  - Mock up 2–3 directions as images first, so you can pick one before I build it. For example:
    - a hand-drawn map with path lines and stamps
    - a constellation of glowing nodes
    - a board game with the origin as a token
  - Rebuild `drawMap` (game.go) in the chosen style, with fewer labels, clearer icons, gentle animation and a short tooltip per node.
  - Keep keyboard, controller and mouse navigation as they are (nav.go).
- **New text:** small.
- **Checks:** screenshots at 3 resolutions; controller-navigation walk-through via capture.
- ❓ **Your call:** pick a mock-up.

### Step 4: Cameo challenges (idea 7)

- **What I'll do:** add an occasional special node on chamber select, at most one per act, where you play that one chamber as a different origin.
  - Your items stay, but the passive and ability are the cameo's.
  - Finishing it pays a reward (gold, plus a chance at a quip or art unlock later).
  - The challenge system (challenges.go) and origin data (run.go) already hold what's needed.
  - The swap must cleanly restore your origin afterwards, including the saved-run format (runsave.go).
- **Checks:** campaign sim with cameos forced on, to make sure they are neither a free win nor a trap; save/resume mid-cameo.
- ❓ **Your call:** should cameos include locked origins as a "try before you unlock" teaser? I'd say yes.

### Step 5: Soundtrack selection (idea 10)

- **What I'll do:**
  - Name the five tracks. Working titles from the code: "Arcane Groove" (menu), "Orb Rush" (act 1), "Acid Cascade" (act 2), "Abyss Engine" (act 3) and "Warden" (boss).
  - Add a Music screen (from Settings or the codex) that plays any track and shows its name.
  - Add a per-run option: "act music" (default), "shuffle", or "pick one".
  - Optionally add 1–2 new tracks if you want more variety, since the synth engine (package `mus`) makes that cheap.
- **New text:** small.
- **Checks:** render each track with `tools/musrender`; screenshots.
- ❓ **Your call:** confirm or change the names; say whether you want new tracks.

### Step 6: Multiplayer stage select (idea 9)

- **What I'll do:**
  - In the duel lobby (duel_ui.go), the host picks a world or "random". The choice is sent to the guest and shown in the lobby.
  - Reuse the step 3 visual style for the cards.
  - The duel network messages (duel_net_*.go) need one new field. It must stay compatible: an older guest should fall back to random.
- **Checks:** two-client duel in the browser harness; a lobby screenshot.

### Step 7: Origin showcases (idea 6)

- **What I'll do:**
  - Add a 10–15 second scripted demo per origin, played on class select ("Watch"). It's a small chamber where the game plays itself: the ability fires, the passive triggers, and captions name each.
  - Build it on the existing autoplay used by the trailer (trailer.go) and the bots in the balance sims.
  - Show it the first time you pick an origin, and let you skip it.
- **New text:** about 3 captions per origin (×7).
- **Checks:** capture each showcase to video for you to review.

### Step 8: Collectible art (ideas 11, 12)

- **What I'll do:**
  - Add a Gallery (codex tab) of about 12–20 illustrations: each origin solo, plus group scenes such as hanging out at the campfire, a duel, or a victory pose.
  - Each completed challenge chamber unlocks one illustration (challenges.go already tracks completion, and there are 10 challenges). Locked slots show a silhouette and how to unlock them.
  - Track unlocks in save.go. Steam achievements can mirror them.
- **The art itself is the open question.**
  - What I can make: Blender-rendered scenes of the origins' 3D portrait models (the tools/portraits and tools/artkit pipeline). These give a consistent, stylised look.
  - For true "graphic designer" illustration, you'd commission it or supply it, and I'd build the gallery around your files.
- ❓ **Your call:** Blender scenes from me, commissioned art, or both (mine as placeholders until the art arrives)?

### Step 9: More items (idea 1)

- **What I'll do:**
  - Design 10–15 new items that fill gaps in the current pool (mods.go and items2.go):
    - more items that change *how* you play (ball behaviour, paddle tricks), not just bigger numbers
    - a few cross-origin synergies
  - Write each item's description in the existing style, with all 7 languages.
  - Balance with the campaign sim: run before and after, and require origin spread ≤ 15% and no item that's an auto-pick. The power controller (power.go) needs a feed-forward entry per new item.
- **Checks:** the description and text audit; sims; screenshots of the shop and reward screens.
- ❓ **Your call:** any item ideas you already have. Also: should some unlock through progression rather than start in the pool?

### Step 10: Unique bosses (idea 2)

- **Today:** all three act bosses (Prism Colossus, Crucible Golem, Crystal Warden) share one behaviour. It's a brick body with a moving core that rebuilds, plus falling-rock volleys (play.go `updateBoss`).
- **What I'll do:**
  - Give each boss its own kit and 2–3 phases. Some ideas to pick from:
    - **Prism Colossus:** splits the ball's path with refracting prisms; its shield rotates and must be hit from the open side; phase 2 spawns mirror copies of the core.
    - **Crucible Golem:** pours molten rows that advance and harden into armour; vents pull the ball (a gravity well); an overheat phase where hitting the vents stuns it.
    - **Crystal Warden:** freezes lanes and makes the paddle slippery; grows crystal bricks that reflect at sharp angles; a final phase where it steals the ball and you must break its grip.
  - Each boss gets its own intro line (step 2's voice) and its own music section.
- **Size:** 2–3 sessions, one per boss.
- **Checks:** campaign sims per boss, for clear rate and time-to-kill in each act and difficulty; capture videos for you to playtest; a PC bench of the boss scene.
- ❓ **Your call:** pick or adjust the mechanics per boss before I start.

## Decisions so far (2026-11-05)

- **Step 1 (lore):** done in code, not yet compiled (see HANDOFF.md). Simple story: three bosses bury the world in bricks, seven heroes use the Orb to beat them. Text in docs/lore.md; hero names Oriel, Brakka, Vell, Rivet, Sol, Cinder, Zephyr.
- **Step 2 (quips):** text **plus** short synthesised voice blips per hero. Lines drafted in docs/quips.md (awaiting the user's read-through).
- **Step 3 (chamber select):** direction **B, constellation**, reworked to be more arcane (rotating rune circle, summoning circle round the boss, rune rings on the rooms you can pick, flowing magic on the next path). **Keep the existing run panel (top left) and the Build panel's place and look (right)**. The user approved B2, plus Build panel changes: items grouped by rarity (fusions, then legendary down to common, with counts); hovering or tapping an item shows its card (name, rarity, description, fusion link) beside the panel; a "Your stats" box at the bottom lists each stat with its base value in dim grey and the current value (after items) to the right in green, so progress is visible. Mock-up: board "B2" on the "Chamber Select Directions" design canvas.
- **Step 4 (cameos):** locked heroes **are** included, as a try-before-you-unlock teaser.
- **Step 8 (gallery):** **Blender scenes only** (tools/portraits and tools/artkit), no commissioned art.
- **Opening cinematic (new):** a ~49 s skippable intro replacing the prologue card: pencil-sketch half, then the Orb burns through the page into an arcane half. Script and storyboard in docs/opening.md; preview in the "Arcane Orb Opening" artifact. Story tone shifts to sillier and more magical (the old wizard's botched Tidy Up spell, LORT-like humour); the story cards, bios and Lore pages from step 1 get redone in that voice once the user approves the opening.
- Still open: step 5 track names, step 9 item ideas, step 10 boss mechanics.

## Order at a glance

1 Lore → 2 Quips → 3 Chamber select → 4 Cameos → 5 Soundtrack → 6 Duel stage select → 7 Showcases → 8 Gallery → 9 Items → 10 Bosses

Steps 5 and 6 are independent and can move earlier. Step 10 can start any time after step 2 if bosses matter more to you.
