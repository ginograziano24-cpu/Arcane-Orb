# Making Arcane Orb 1.5× more enjoyable

Brainstorm, Nov 17, 2026. Nothing here is built yet. Each idea says what it
is, why it should help, what it costs, and how we'd know it worked.

## What "1.5× more enjoyable" means

Enjoyment can't be measured directly, so pick proxies and move them by
half. The game already records most of these (`recap.go`, `chronicle.go`,
`balance_run.go`); the rest are small additions.

| Proxy | Why it tracks fun | Target |
|---|---|---|
| Runs per session | "One more run" is the roguelite's core promise | +50% |
| Share of runs where a synergy or fusion fires | Builds coming together is the payoff moment | 1.5× |
| Median chamber length (35–75 s band) | Too short is trivial, too long drags (POLISH_AUDIT item 7) | 90% of chambers in band |
| Quits mid-chamber | A quit is a "this isn't fun" vote | −33% |
| Playtester "best moment" answers that name a build, not a single item | Depth over luck | majority |

## The ideas, ranked by fun per hour of work

### Tier 1: cheap, high impact

1. **Build-around cards ("archetype seeds").** In the first reward of a
   run, guarantee one card that *starts* a synergy or fusion line (shown
   with its partner: "FUSES WITH …" already exists). Roguelites feel best
   when the player has a plan from chamber 2. *Cost:* a flag in `Offer`.
   *Measure:* synergy/fusion share.
2. **A paid reroll on every reward.** Banish is already on the reward
   screen, but rerolling a reward is only possible with free rerolls (the
   Gambler). Let everyone reroll once per reward for gold (e.g. 15, rising
   each time). It gives gold a second use between shops and turns a dead
   offer into a decision. *Cost:* the button exists (`game.go` "Reroll"),
   add the gold path. *Measure:* gold spent outside shops, skip rate.
3. **"Chamber modifiers" on the map.** Some nodes show a small twist before
   you enter (all bricks armoured but double gold; mirror arena; fog;
   two paddles). Omens (`omens.go`) and challenges already contain most of
   these. Visible choice on the map turns a path into a decision. *Cost:*
   reuse omens, add map icons. *Measure:* path variety per run.
4. **Juice the last brick.** The finale slow-mo exists; add a guaranteed
   *crit-explosion* on the last brick plus a short combo tally ("42 hits,
   ×3.1 combo, 120 gold"). The end of every chamber becomes a small reward
   beat. *Cost:* small, `startFinale`.
5. **Combo stakes.** Long combos already pay coins every 10 hits. Add a
   visible "streak" meter that fills toward a free power-up at 25/50/100,
   and *breaks with a sound* when you drop the orb. Loss aversion makes
   every save matter. *Cost:* HUD + one rule. *Measure:* average combo.

### Tier 2: medium work, big feel

6. **Paddle skill expression.** Right now angle comes from where the orb
   hits. Add a **flick**: moving the paddle fast at contact adds spin
   (curving the orb a little for a second). Skilled players get a
   ceiling; casual players never need it. The Stormcaller's sweep
   mechanic shows the input reads already work. *Cost:* physics tweak +
   tuning. *Measure:* playtester "control" rating.
7. **Elite affixes.** Elite chambers get one named affix (Shielded,
   Splitting, Regenerating, Hasted) shown on entry, like Diablo/Hades
   elites. Elites become memorable fights instead of tougher boards.
8. **Guardian phases with a tell.** Each Guardian gets a second phase at
   50% HP with one new attack, telegraphed for 1 s. Boss fights are where
   runs are remembered; a phase change doubles the story.
9. **Run-end "build card".** A shareable image of your build at the end of
   a run (origin, items, fusions, best combo, seed). Players share wins;
   shares sell the game. The photo mode and recap screens have the pieces.
10. **Weekly seed leaderboard with ghosts.** The weekly seed exists. Show
    the top three players' builds (item lists) for that seed. Seeing what
    worked for others is half the fun of a seeded run.

### Tier 3: bigger bets

11. **Co-op drop-in on Steam** (two paddles, shared build or split loot).
    `coop.go` exists; finishing it gives the game a second audience.
12. **Endless mode with mutators** every act (stacking rules, à la Hades
    heat). Already partly there (`loops.go`, `weekly.go` mutators).
13. **Daily "puzzle chamber":** one handcrafted board, one try, fewest
    hits wins. A two-minute daily habit that brings people back.

## What to try first

The five Tier 1 ideas together are about two sessions of work and hit
every proxy above. Suggested order: 4 (last-brick juice), 1 (build-around
first card), 5 (combo stakes), 2 (banish surfacing), 3 (map modifiers).
Run the balance simulator before and after each, and get a playtester's
"best moment" answer for each build.

## Taste calls for you

- How much skill ceiling (the flick) vs. how relaxing the game should stay.
- Whether combo breaks should *cost* something or only stop the bonus.
- Sharing: a build card image, a seed code, or both.
