# Opening cinematic: ideas for the next session

The current preview (`opening_preview.html`) is the animatic: story, timing and
staging. This list is what to improve next, and how the Blender rebuild should
go. Pick from it; nothing here is decided.

## 1. Style references to borrow from

| Reference | What to take |
|---|---|
| **Fantasia, "The Sorcerer's Apprentice"** | The core trope we're already telling: a cleaning spell that won't stop. Lean into it: the bosses multiply or speed up, Fumbleton tries to stop them and makes it worse. |
| **Spider-Man: Into the Spider-Verse** | Mixed media: animate "on twos" (we do), halftone dots, comic SFX lettering, frame-freeze impacts, chromatic offset on big hits. |
| **Borderlands / Scott Pilgrim** | Freeze-frame character intros: the shot stops, a scribbled name card with a joke caption pops in. Perfect for the hero splashes and for Fumbleton's first appearance. |
| **Hades / Persona 5** | Slick, readable splash cards: bold diagonal layouts, one strong colour per character, fast in-out timing. Our hero splashes are close; push the typography. |
| **Smash Bros. / Overwatch** | "Joins the battle!" reveal energy for the portal march: each hero's arrival gets a pose and a sound stinger. |
| **Arcane (Fortiche) / Klaus** | 2D-looking characters with real volumetric light and painterly FX: target look for the arcane half in Blender. |
| **Cuphead** | Hand-drawn wobble and squash-and-stretch with snappy timing for the sketch half. |
| **Gravity Falls / Adventure Time** | Comedy timing: hold on a reaction a beat longer than feels normal ("…wow."). |
| **Jessica Rabbit (Roger Rabbit) / Rouge (Sonic)** | Mirabel's glamour silhouette and entrance: a slow-motion "imagine spot" with sparkles, soft focus and a music sting. |
| **Infomercials** | Optional twist for the scroll: "Is YOUR tower a mess? Try INSTANT TIDY! But wait, there's more!" as a flashback panel. |

Tropes that fit: the imagine spot (Mirabel), the infomercial, "Yep, that's me" freeze frame, the hero assembly montage, the monkey's paw (the scroll works too well), and the post-credits button (Mirabel arrives; the tower is a brick).

## Done in preview 5

Exterior opening shot, Fumbleton's freeze-frame name card, Mirabel's imagine
spot, the receipt (NO REFUNDS), the Sorcerer's Apprentice escalation (he
chases them, BAMF, they grow and multiply), boss name captions, hero
signature actions, the end button, and a Spider-Verse comic pass (halftone,
SFX lettering, caption boxes, colour split). Still open below: the 45-second
cut, model sheets, a narrator and score, localising the SFX words, and the
Blender rebuild.

**Ideas for the next pass**
- The escalation could go one step further: a boss splits in two like the
  brooms in Fantasia, and the copies carry the brick carpet faster.
- The imagine spot wants a real sax sting and a soft-focus bloom in Blender.
- The comic SFX words are English; swap them per language or keep them as
  drawn sound shapes.
- The end button's "…" could hold a beat longer (Gravity Falls timing), then
  cut to black on the wah-wah.

## 2. Artistic changes to consider

**Story and staging**
- Open on an exterior shot of the tower at night (one line of narration) before cutting inside.
- Freeze-frame intro for Fumbleton with a scribbled name card: "ARCHMAGE FUMBLETON · has a date tonight".
- Make Mirabel's daydream a proper imagine spot: the room fades, she glows, sparkles, a sax or harp sting, then snap back to the mess.
- The scroll: either a quick infomercial flashback, or a crumpled receipt that says "Travelling Merchant · NO REFUNDS".
- Sorcerer's Apprentice escalation: Fumbleton chases the bosses with the scroll, reads it again, and they get bigger.
- End button: the doorbell rings, Mirabel stands at a door that's now a brick wall.
- Consider a 45-second cut alongside the full 68-second one; test which keeps attention.

**Characters**
- Fumbleton: a real model sheet (turnaround, 6–8 expressions); give him mouth shapes if we add a voice.
- Mirabel: a finished character sheet from an artist, using the preview's silhouette as the brief. Keep her non-explicit so the store page and age rating stay safe; she could also become an in-game collectible or cameo.
- Bosses: match their sketch designs to the real in-game bosses (Prism Colossus, Crucible Golem, Crystal Warden) so players recognise them later.
- Heroes: give each splash a signature action (Sol flips a coin, Cinder sparks a flame, Zephyr crackles, Rivet's goggles flash, Brakka cracks her knuckles, Oriel raises her shield, Vell's orb spins).

**Sound and words**
- A real narrator (warm, slightly deadpan) instead of captions only; or gibberish character voices like Animal Crossing.
- A composed score with sync points: music box → silence at the burn → the arcane theme drops on the Orb.
- Sound effects with personality (a rubbery squeak, a POOF with a cough, brick "thunks" in rhythm).
- Localise: captions in all 7 languages; keep on-screen SFX words (TIDY!, POOF!) either language-neutral or swapped per language.

## 3. Optimising the film

**For the browser preview**
- Pre-build each pencil stroke variant as a single Path2D (tapered outline) instead of one stroke call per segment: the biggest remaining cost in the sketch half.
- Cache shapes into the layer cache as soon as they finish drawing, not only when the whole room is done.
- Pause rendering when the tab is hidden; cap device pixel ratio on phones.

**For the game**
- Play the final film as a pre-rendered video rather than drawing it live. Ebitengine has no built-in video player, but its video example decodes MPEG-1 in pure Go; check the current options before choosing a format.
- Encode at 1080p60 for size, with a 1440p version if it stays small enough for the Steam download.
- Skippable (hold), plays once on first launch, rewatchable from the Codex; captions drawn by the game (so they translate) over the video.
- Ends on a frame that matches the main menu, so the menu fades in seamlessly.

## 4. The Blender rebuild

- **Timing first:** export the preview's shot list and cue times (scene starts, hits, hero slams) into Blender timeline markers, so every cut lands where the animatic says.
- **Sketch half:** Grease Pencil. Its noise modifier with stepped time gives the "boiling" line natively; paper texture on a plane; camera moves on the same timings. Or keep this half 2D and render only the arcane half in 3D.
- **The burn:** a noise-threshold shader eating the paper plane, an emissive edge, ember and paper-flake particles, a camera shake.
- **Arcane half:** emissive rune circles as geometry, a volumetric light column, bloom and glare in the compositor, real reflections on a glossy floor for the march.
- **Heroes:** use the full 3D models from `tools/portraits` posed full-body (the HUD busts are cut at the chest); one key light plus a rim light in each hero's colour.
- **Output:** 60 fps image sequence with render passes, so grain, bloom and colour grade can be tuned in the compositor without re-rendering.
- **Reuse:** the same scene renders the Steam key art properly lit (the 3840 Library Hero especially benefits) and the first shots of a Steam trailer.
