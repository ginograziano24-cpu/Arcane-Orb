> Status: queued. Starts once the hero models (Arcanist layers/eyes, Tripo parts) are final. Added by Gino on Nov 16.

# ARCANE ORB — POLISH PASS

You are the polish lead on "Arcane Orb," a roguelite brick-breaker built in Go with Ebitengine, heading for a Steam release. It's inspired by Alawar's Magic Ball: a warm fairy tale with lighthearted, meme-friendly humour, where a band of funny heroes uses Orb magic to defeat the bosses. Archmage Fumbleton and Mirabel the witch are part of that world.

Your goal: take the game from "works" to "people can't put it down." The two things that matter most are GAME FEEL and CHARACTER. Every hit should feel great, and every hero should feel alive.

Work in phases. Finish and commit each phase before starting the next. Don't skip ahead.

## PHASE 1 — AUDIT (change nothing yet)
Read the whole codebase and trace the full game loop. Write POLISH_AUDIT.md, ranked by player impact, covering:

**Game feel**
- Does every action have satisfying feedback: paddle hit, brick break, item pickup, combo, life lost, chamber clear, boss hit?
- Missing juice: hit-stop, screen shake, camera punch, ball trails, squash and stretch, particles, flashes, impact sounds
- Paddle control: responsiveness, input latency, acceleration and feel. Does aiming the ball feel skillful?
- Ball physics: predictable angles, no boring horizontal loops, no stuck balls, and a good sense of speed

**Character detail**
- Does each hero have a distinct personality in how they look, move, sound and speak?
- Idle animations, reactions (big combo, near miss, losing a life, picking up a great item, beating a boss)
- Quips: are they funny, varied and human, or generic and repetitive? Check trigger frequency so they never get annoying
- Does each hero's ability feel like *them* (unique visuals, sounds and timing)?
- Do Fumbleton's and Mirabel's letters between acts have charm and personality?

**Bugs and stability**
- Ball tunneling, softlocks, item interactions that break, save/load issues, edge cases across all 3 worlds

**Performance (Ebitengine/Go specific)**
- Draw call batching: find anything that breaks batching (interleaved source textures, blend mode or filter changes, shader switches). Pack sprites into atlases using SubImage
- No ebiten.NewImage, ReadPixels or At() inside Update/Draw. Pre-allocate and reuse
- Particles and trails: draw them in one DrawTriangles call with reused vertex/index slices
- GC stutter: find heap allocations in Update/Draw (slices, closures, fmt.Sprintf, UI string building). Use pools and reuse slices with [:0]
- Text: cache text/v2 faces and don't rebuild strings every frame
- Kage shaders: no full-screen effects running when they aren't needed
- Physics: fixed TPS in Update, with substeps or swept collision so fast balls never tunnel
- Audio: decode once at load and reuse audio.Player instances
- Target a locked 60fps on low-end hardware, not just a high-end GPU

**Clarity and UX**
- Can a new player always tell what's happening, what an item does, and why they lost a life?
- Card picks, fusions, menus, transitions, readability
- Keyboard and controller support, and settings that persist (volume, resolution, fullscreen, screen shake toggle)

**Pacing and balance**
- Chamber length, difficulty curve across the 3 worlds, item power spikes, dead time

**Audio**
- Generic, repetitive or missing sounds. Every sound should be unique and sculpted, never a stock power-up noise
- Mix levels, and smooth music transitions between chambers

**Steam readiness**
- Layout() handles scaling and high-DPI, the fullscreen/windowed toggle works, pause works everywhere, no debug leftovers, and go-steamworks is integrated if achievements are planned

## PHASE 2 — STABILITY AND PERFORMANCE
Fix bugs and performance issues from the top of the list down. Add a debug overlay (ActualFPS, ActualTPS, allocations per frame) and profile with pprof during a heavy late-game chamber. Report before-and-after numbers.

## PHASE 3 — GAME FEEL (the most important phase)
Make every player action satisfying:
- Hit-stop on big impacts, scaled screen shake, camera punch on boss hits
- Ball trails that react to speed, sparks on paddle contact, paddle squash and stretch
- Brick breaks with particles, debris and a crunchy, unique sound. Armoured or heated bricks feel heavier
- Combos that build visibly and audibly (rising pitch, growing effects, hero reactions at milestones)
- Item pickups and card picks with anticipation, payoff and a little flourish
- Fusions show their total stat buff or new passive with a satisfying reveal
- Put all feel values in one tunable config file so I can adjust them while playtesting
Keep every effect cheap enough to hold 60fps.

## PHASE 4 — CHARACTER DETAIL
Bring every hero to life:
- Distinct idle, reaction and victory animations for each hero
- Quips for selection, high combos, near misses, lost lives, great item pickups and boss kills. They should sound like real, funny people, not generic game barks. Add variety and cooldowns so they never repeat too often
- Synth voice blips that match each hero's personality
- Abilities with visuals, sounds and timing unique to each hero
- Polish the letters between acts so Fumbleton and Mirabel each have a clear voice

## PHASE 5 — CLARITY AND UX
Tighten the UI, feedback and the first 2 minutes of play. A new player should understand the game without reading anything.

## PHASE 6 — VERIFY
Re-run the full audit. Run `go vet` and a race-detector build (`go build -race`). Report what's fixed, what's left, and anything that needs my playtest feedback.

## RULES
- Don't change the core design without asking: 3 worlds per run, items carry over between acts, lives instead of health
- Never break existing saves
- All text should be human, warm and charismatic. Corny humour is fine, generic is not
- Keep the lore simple and avoid jargon
- Explain any change that affects balance
- If something needs my feel or taste judgment, flag it instead of guessing
- Commit after each phase with a clear summary

Things that feel bad to me right now: [ADD YOUR PLAYTEST NOTES HERE]