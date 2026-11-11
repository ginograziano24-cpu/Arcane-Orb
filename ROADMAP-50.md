# Arcane Orb: the 50 most pressing improvements

Written 2026-11-11. Gameplay, feel, balance, UI, audio and Steam
readiness only; the opening cinematic and the 3D models are out of scope
here. Ordered by how much each item matters to a player right now; work
top-down. Mark items `[x]` when done, with the date and a word on how.

Legend: **PC** = needs the PC (build, play or benchmark) to finish or verify.

## A. Must work, every run

1. [ ] **Compile and test the Nov 11 changes on the PC.** `go vet ./...`,
   `go test ./...`, build, play one full run. (PC)
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
   nothing lost, nothing duplicated.

## B. Feel of the core loop

7. [ ] **Hit-stop and shake scale with impact size**, so small hits stay
   crisp and big ones land, and multiball never turns into constant shake.
8. [ ] **Paddle angle readability:** a faint aim guide on the paddle's
   edge zones while the orb approaches (option, off by default).
9. [ ] **Multiball clutter:** cap simultaneous particle bursts per frame
   and dim secondary orbs' trails so the main orb stays readable.
10. [ ] **Armour readability in later chambers** (open item 11 from
    VISUAL-PERFORMANCE-20).
11. [ ] **Boss telegraphs** for every attack and the recovery window
    (open item 7). (PC to verify live)
12. [ ] **Chamber pacing per act:** wave count, advance speed and heat so
    act 1 chambers run about 90 s and act 3 about 150 s.
13. [ ] **Last-brick hunt:** the paddle-bounce lean already exists; also
    make the last 3 bricks glow and pulse on the minimap edge.
14. [x] **Hero quips and combo commentary** (2026-11-11, quips.go).

## C. Balance

15. [ ] **Engineer runs ~20% faster than the others:** trim overclock.
16. [ ] **Act 3 timeouts in carried runs** (8 in the last sim): soften
    heat scaling in act 3.
17. [ ] **Origin spread ≤ 15%** in the campaign sim.
18. [ ] **No auto-pick items:** pick-rate audit from the sim.
19. [ ] **Gold economy:** income vs shop prices per act; nobody should be
    unable to afford anything in act 3, nor buy everything.
20. [ ] **Curse severity audit:** every curse a real choice, none a run-ender.
21. [ ] **Ascension tiers:** check each tier adds one clear rule.

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
27. [ ] **More events**, in the same funny voice as the troll and the goat.
28. [ ] **More omens and chamber challenges.**
29. [ ] **Elite brick variety:** two new elite behaviours per world.
30. [ ] **Cameo challenges** at chamber select (locked heroes as a teaser).
31. [ ] **Endless loop modifiers:** each loop adds a named twist.

## E. Interface

32. [ ] **Chamber select Build panel (B2):** items grouped by rarity,
    hover card, "Your stats" with base vs current values.
33. [ ] **Reward cards show the stat change** the card would make.
34. [ ] **Pause menu shows the build and stats.**
35. [ ] **Run recap highlights the best moment** (biggest combo, the
    item that did the most damage).
36. [ ] **Controller pass on every screen**, including focus rings and
    lock text. (PC)
37. [ ] **Keyword tooltips everywhere** a keyword appears.
38. [ ] **4K and ultrawide layout check** for every screen. (PC)
39. [x] **Quip voice and bubble setting** (2026-11-11, Settings > Audio).
40. [ ] **First-run onboarding:** fewer, better-timed tips.
41. [ ] **Colour-vision pass** for every status (open item 17).

## F. Audio

42. [x] **Sculpted sounds for every pickup and moment** (2026-11-11,
    sfx_sculpt.go).
43. [ ] **Soundtrack selection** with named tracks (Settings or Codex).
44. [x] **A separate Voice volume slider** for quips. (2026-11-11:
    Settings > Audio > Hero voices.)
45. [ ] **Dense-fight mix:** limit same-sound stacking per frame and duck
    small hits under big ones.
46. [ ] **Per-world brick material sounds** (the on-hold #128 rework).
47. [ ] **Boss phase music** sections.

## G. Steam readiness

48. [ ] **Achievements and stats** for the new content (quips not needed;
    bosses, cameos, endless loops).
49. [ ] **Duel stage select** (roadmap step 6).
50. [ ] **Steam Deck verified checklist:** controls, text size, suspend
    and resume, performance. (PC)
