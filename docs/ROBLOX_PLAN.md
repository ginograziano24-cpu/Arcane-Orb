# Arcane Orb on Roblox: notes and game plan

Written Nov 17, 2026. Status: planning only. Nothing here starts before the
Steam release is out the door.

## Why bother

Popular Steam games get cloned on Roblox within weeks, usually by teams who
copy the name, the look and the core loop and then own the search results.
If Arcane Orb lands at all, the safest move is to be the first and best
"Arcane Orb" on Roblox ourselves:

- we own the name and the look there before a copy does
- Roblox players who find it there can be pointed at the Steam version (and
  the other way round)
- it is a second income stream from the same design, art and lore

You already know the platform from Egg Tycoon, so the Studio side is
familiar ground.

## What carries over and what doesn't

| Part | Carries over? | Notes |
|---|---|---|
| Design: brick-breaker roguelite, 3 worlds per run, lives, items carry over between acts | Yes | The whole point. Keep the rules identical. |
| Origins (Guardian, Berserker, Arcanist, Engineer, Gambler, Pyromancer, Stormcaller), passives, abilities | Yes | Same numbers as a starting point, then tune for shorter sessions. |
| Items, scrolls, fusions, curses, keepsakes | Yes | Port the data tables first; they are plain data in the Go code. |
| Worlds (Glimmer Grove, Sunset Meadow, Crystal Caverns, Ancient Ruins, Cloud Kingdom, Machine Works) and bosses | Yes, rebuilt | Same palettes and names; built from Roblox parts/meshes. |
| Lore, letters, quips, Fumbleton and Mirabel | Yes | Text ports straight across. Keep it short; Roblox players skim. |
| Hero 3D models | Yes, reduced | Roblox caps a single imported mesh at about 20k triangles and textures should stay at 1024 or below. Our game heroes are 60k with a 4K bake, so each needs a reduced export (the export_hq.py pipeline can do that with TARGET lowered and TEX set to 1024). |
| The Go code itself | No | Everything is rewritten in Luau. The Go code is the spec. |
| Synth music and sculpted SFX | Yes, as files | Render each track and sound to audio files and upload them as Roblox audio. |
| Duels (1v1) | Yes, and better | Roblox servers make multiplayer easy; this is where Roblox can beat the Steam version. |
| Steam achievements | Replace | Roblox badges. |

## Making it a Roblox game, not just a port

Roblox players expect different things from Steam players:

- **Short sessions.** A full 3-world run is too long for many Roblox
  sessions. Add a "Quick Run" (one world, about 8 to 10 minutes) next to the
  full run.
- **Play together.** Co-op on one board (two paddles), the 1v1 duel, and a
  lobby where you see other players' heroes and trophies.
- **Show-off progression.** Hero skins, orb trails and paddle styles that
  other players see in the lobby. Cosmetic only.
- **Mobile first.** Most Roblox players are on phones: touch paddle control
  (drag anywhere on the lower half), big buttons, lower poly budgets.
- **Daily reasons to come back.** The weekly seed (weekly.go) and challenge
  chambers map neatly onto daily and weekly quests.

## Money (keep it fair)

- Cosmetics only for real money: skins, trails, paddle styles, victory
  emotes (Developer Products and Game Passes).
- No pay-to-win: no buying items, lives or stat boosts. The roguelite only
  works if every run is fair.
- Premium Payouts reward play time from Roblox Premium members, which suits
  a replayable game.

## Protecting the name

- Publish an official Arcane Orb experience before the Steam launch hype
  peaks, even a small one, so the name is taken by us.
- Register as a rights holder in Roblox's **Rights Manager** (Creator Hub).
  It is the official route for reporting copied games, models, audio and
  images, and it needs ID verification and proof that you own the IP. Keep
  dated files (the Blender sources, the Git history, the Steam page) as that
  proof.
- Use the same title, logo and key art on Steam and Roblox so players can
  tell the official one at a glance, and link each from the other.
- Expect the Rights Manager to be slow and imperfect (creators complain about
  it on the DevForum); being first and best matters more than takedowns.

## Game plan

### Phase 0: after Steam launch (planning, about 1 week)
- Decide Quick Run length, co-op rules and the cosmetic list.
- Pick the first 3 origins and 1 world for the Roblox launch (smaller scope).
- Set up the Roblox group, the experience page and Rights Manager
  registration.

### Phase 1: core loop prototype (2 to 3 weeks)
- Paddle, orb physics and bricks in Luau, server-authoritative with
  client-side prediction so it feels instant.
- Port the item and origin data tables (a script can convert the Go tables
  to Luau modules).
- One world (Glimmer Grove), one boss, touch and keyboard controls.
- Target: a fun 5-minute run on a phone.

### Phase 2: content and art (3 to 4 weeks)
- Reduced hero meshes (under the triangle cap, 1024 textures), rigged with
  Roblox's R15 or a custom rig, idle and reaction animations.
- 3 origins, 1 to 2 worlds, the item pool, fusions.
- Audio: render the 5 music tracks and the SFX set, upload them.
- Letters and quips (short versions).

### Phase 3: Roblox-native features (2 to 3 weeks)
- Lobby with heroes on display, co-op mode, duels on Roblox servers.
- Badges, daily and weekly quests, the weekly seed.
- Cosmetics shop (skins, trails, paddle styles).

### Phase 4: soft launch and grow
- Private beta with friends, then public.
- Watch session length and retention in Roblox analytics; tune Quick Run
  length and difficulty from real data.
- Add the remaining origins and worlds as updates, roughly one every few
  weeks, timed with Steam updates so both versions grow together.

## Tools we already have

- The Roblox Studio MCP connection on your PC, so Claude can build and test
  inside Studio with you.
- Blender pipeline (export_hq.py) for reduced meshes and bakes.
- `tools/musrender` for rendering the music, and the SFX sculpting code for
  sound files.
- Tripo for any new props.

## Open questions for you

- Roblox launch before, alongside, or after the Steam release?
- Which 3 origins and which world go first?
- Solo dev on the Roblox side, or bring in a Luau scripter?
