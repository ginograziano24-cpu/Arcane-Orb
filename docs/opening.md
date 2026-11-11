# Arcane Orb: current opening direction

The active November 6 development previews are the workspace outputs `Intro-Dual-Pass/Animated-Story.html` and `Intro-Dual-Pass/Blender-Story.html`. They are separate, self-contained 81.5-second previews, not native-game playback. The original supplied cinematic files are historical references; they no longer define the current story or Mirabel design.

Mirabel uses the approved purple gown, dark hair and locked proportions. Fumbleton remains the small, comic wizard. Both retain their story roles but are absent from the seven-hero gathering, hero cards and title lineup. The three guardians retain their actual game construction and silhouettes. There is no Academy in this story.

| Time | Current beat |
|---|---|
| 0–2.5 | The tower at night; warm opening. |
| 2.5–11 | Fumbleton anticipates date night and notices his untidy room. |
| 11–17.5 | The merchant's Instant Tidy scroll summons the three guardians. The score turns panicked at 14 seconds. |
| 17.5–26.5 | The guardians continue tidying outdoors. Lead transformation: Prism outlines scenery, Crucible compresses it, Warden stamps and orders the bricks. Fumbleton's attempted correction makes the problem worse. |
| 26.5–33 | Brick rows overtake the landscape; the escalation stays lighthearted. |
| 33–38 | Mirabel arrives, sees the mess and reacts to Fumbleton. |
| 38–43 | Mirabel visibly creates one orb and splits it into seven. |
| 43–47.5 | She sends the seven orbs to the heroes. The high-energy call to action starts at 43.6 seconds. The page transition follows. |
| 47.5–55.5 | The orb demonstrates the game's brick-breaking premise. |
| 55.5–69.5 | Seven different catches introduce the seven heroes with their own cards and signatures. |
| 69.5–76 | Staggered arrivals resolve into one shared location and a spacious ensemble reveal. |
| 76–81.5 | Title payoff and final invitation. |

The score is original, with warm, panic and heroic sections; no Magic Ball recordings or note patterns are used. Three audio stems share a playback clock; the music ducks under twenty lines of local guide narration. Independent music, effects and narration sliders are provided in the previews. Guide speech is not final voice casting.

Visual option boards are delivered at workspace `outputs/Intro-Storyboards/Storyboards.html`: three monster-transformation approaches and thirteen call-to-action panels. Panels clearly distinguish current blocking references from planned gesture animation. The illustrated and Blender previews already share the lead transformation language, seven distinct identities and hand-following orbs.

## Verification and remaining work

Native Skia rendered 21 representative illustrated frames. Twelve GLBs pass loader and normalized-weight checks. A real Three scene-logic test with a renderer stub checks 830 timestamps, seven receiving-hand catches and exclusion of Mirabel and Fumbleton from the lineup. These are structural and native-render checks, not browser playback or GPU performance measurements.

Remaining work includes final character acting, shoulder and hand deformation review during real playback, cloth and hair follow-through, narration casting, audio playback review and native-game integration. The preview's frame-rate meter and adaptive resolution do not establish sustained 60 fps. Browser playback has not been verified; permitted local browser access was unavailable. Do not replace this limitation with CPU frame timing claims.
