# Arcane Orb: opening cinematic (preview 5)

> **On hold, work in progress.** Not part of playtests; waiting for a new brief. See `docs/cinematic/wip/README.md`.

About 68 seconds, before the first run. Skippable (hold), plays once on first
launch, rewatchable from the Codex. The approved motion and timing
blueprint is `docs/cinematic/opening_preview.html` (open it in a browser);
the final version is to be rebuilt in Blender for proper lighting and polish.

**Look.** 0:00–0:37 is a sketchbook cartoon, with Spider-Verse comic touches (halftone dots, Bangers-style SFX lettering, yellow caption boxes, a colour split on big hits): wobbly pencil with pressure,
overshooting corners, coloured-pencil hatching, a pencil drawing each line,
animated "on twos" (12 drawings a second). The Orb burns through the page and
the rest is arcane: nebula, rune circles, bloom, at a full 60 fps.
Fumbleton's cartoon design: pear-shaped robe, floppy hat with a bent tip and
a star, huge cloud beard, round nose, bushy brows, tiny boots.

| Time | Shot | Narration |
|---|---|---|
| 0:00 | **Prologue.** The tower on its hill at night, drawn in pencil: a lit window, a moon, twinkling stars, a sign (ARCANE ACADEMY); the camera pushes into the window | Once upon a time, in a tower on a hill, lived a wizard with a problem. |
| 0:02 | **I. The Date.** The messy tower interior; the DATE NIGHT calendar. **Freeze frame** (Borderlands): the shot stops, halftone wash, a name card "ARCHMAGE FUMBLETON · has a date tonight". **Imagine spot**: the room melts into pink, Mirabel appears large and glowing (same design: magenta gown, gloves, red hair, hat, wink) with sparkles and hearts while Fumbleton swoons; a record SCRATCH! snaps back to the mess ("(nervous)", "(the mess)") | This is Archmage Fumbleton. His girlfriend Mirabel is coming over tonight, and his tower is a disaster. |
| 0:11 | **II. The Scroll.** "Ta-da!" INSTANT TIDY!, with a crumpled receipt: "Travelling Merchant · 1x Instant Tidy … 99g · batteries not incl. · NO REFUNDS" stamp. POOF (halftone, colour split); the spotless room with three bosses crammed inside; "…wow."; TIDY! MORE! | So he bought an Instant Tidy scroll from a travelling merchant. No refunds. |
| 0:17 | **III. Outside** (Sorcerer's Apprentice). SLAM!; the bosses stomp out with comic caption boxes naming them (THE PRISM COLOSSUS, THE CRUCIBLE GOLEM, THE CRYSTAL WARDEN), a brick carpet unrolling behind them. Fumbleton runs after them waving the scroll ("STOP!!", "Tidy… LESS?!"), BAMF!: they grow bigger and multiply; "TIDY MORE!!!"; "uh oh." | It worked! Three enormous bosses tidied his tower in seconds. Then they kept going. He tried to stop them. He made it worse. |
| 0:26 | **IV. Bricks.** Hills, river, trees and a smiling sun, then the wall climbs row by row; the sun goes O_O; "even the SKY?!" | They tidied everything outside into bricks. The hills, the rivers, even the sky. |
| 0:33 | Fumbleton pops up with the glowing Orb ("Ha-HA!"), hurls it; it falls at the camera, the page burns away | Desperate, Fumbleton grabbed the one thing that can break any spell. |
| 0:37 | The Orb smashes rune-stone bricks on the beat; each hit gets a comic SMASH! / KRAK! / BOOM! burst and a colour split | The Orb: a ball of pure magic that smashes whatever it hits. Mostly bricks. |
| 0:45 | One splash per hero, with halftone in the colour band and a signature flourish plus SFX: Oriel's shield flash (CLANG!), Brakka's knuckle crack (CRACK!), Vell's orbiting orbs (WHIRR…), Rivet's goggle glint (GLINT!), Sol's coin flip (FLIP!), Cinder's flame (FWOOSH!), Zephyr's lightning (KZZT!) | He begged for help, and seven heroes volunteered. |
| 0:52 | **Portal march.** The summoning circle and light column; seven heroes ignite from the centre outward; the Orb rises | Most of them even read the instructions. |
| 0:59 | Title: the Orb sunburst, logo, tagline; Fumbleton ("sorry!") and Mirabel ("…") cameo | Now it's your turn. Smash the walls, beat the bosses, and bring back the sky. |
| 1:04 | **End button.** Back to pencil, at night: Mirabel walks up to the tower, but the door is now a brick wall. KNOCK KNOCK; from inside: "JUST A MINUTE!!"; her "…"; a sad wah-wah | Meanwhile, Mirabel arrived right on time. |

**Sound cue sheet** (the preview synthesizes stand-ins): music-box theme and
pencil scratching through the sketch half; an owl over the prologue; a sting on the freeze frame; a harp glissando for the imagine spot and a record scratch out of it; page turns; a "ta-da"; the scroll glowing; POOF; sparkling-clean chimes; the door SLAM; giant footsteps; "STOP!"; a BAMF as the bosses grow; brick thuds; "Ha-HA"; falling
whistle; impact boom and crackle; from 0:37 the arcane music on a 120 bpm
pulse, shatters on the hits, a chord stinger per hero, bells as they ignite,
a choir swell and shimmer on the logo; a knock and a sad trombone "wah-wah" on the end button.

**Steam art** (separate): the team key art is in `marketing/steam/`.

## Preview 5 (story beats and comic style)

- New beats from `docs/cinematic/NEXT.md`: the exterior prologue, Fumbleton's
  freeze-frame name card, Mirabel's imagine spot, the merchant's receipt, the
  Sorcerer's Apprentice escalation, boss name captions, hero signature
  flourishes, and the end button with the bricked-up door.
- Spider-Verse kit in the preview: halftone patterns, pop-in comic lettering
  with starbursts, yellow caption boxes, a whole-frame colour split on big
  hits (record scratch, POOF, BAMF, Orb hits, hero entrances). The sketch half
  still animates on twos.
- Performance: finished strokes are single filled paths, hatching is batched,
  the prologue tower and the outside background are cached once drawn, the
  film pauses when the tab is hidden, and the pixel ratio is capped at 2.

## Polish passes (preview 4)

- Performance: finished drawings are cached per boil frame, brick sprites,
  pre-baked hero glows, no live blur on the rune circles, pre-built grain and
  vignette, canvas at the screen's real resolution; the film keeps real time
  even when a frame is slow.
- Life: Fumbleton blinks and breathes; hops for joy (squash and stretch) when
  the tower comes up spotless; the daydream bubble pops in with a bounce;
  dust puffs under the marching bosses.
