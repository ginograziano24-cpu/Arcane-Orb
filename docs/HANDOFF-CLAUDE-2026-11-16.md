# Handoff: Claude, Nov 16 2026

Git: branch `claude/sync-2026-11-16` (pushed). Source of truth for code is that branch plus the PC folder.

## Done today
- Camera pop fix (backcache refresh on move, LOD from un-shaken camera, less roll), Sunking's eyes in soft pinks, letters (fit paper, letterheads, per-world lines), audit scenes for hero faces/busts/letters.
- Heroes re-exported in HQ (export_hq.py: 60k tris, 4K bake, weights kept).
- **New Arcanist**: `heroes/glb/fantasy witch 3d model.glb` now drives her game model (`Source/assets/heroes/arcane.orbs/.jpg`) and HUD busts. Old files backed up in `Backup-before-Claude-2026-11-16/heroes`. Rebuild the game exe to see her in-game.
- **Story intro** (docs/opening.md): `intro/build_story.py`, rendered to `intro/frames` (2012 frames, ~84 s) with `intro/voice.wav` (Piper voices + original score) and `intro/lines.txt`. Old film kept as `frames-before-story`, `voice-before-story.wav`, `lines-before-story.txt`.
  - Re-render: `Render-Story-Test.cmd` (stills to intro/test) / `Render-Story.cmd` (full).
  - Voices: `Make-Intro-Voices.cmd` then the mix step; casting + ElevenLabs notes in `docs/voice-casting.md`.
  - Fumbleton: Tripo multiview model from his turnaround, rigged (`heroes/intro_chars/fumbleton_tripo`). His file faces +x and the rig's left/right are swapped, so his facing is set by hand and his arms keep the model pose.
  - Mirabel: `tripo_convert_21334b29…zip` from Downloads (rigged).
- Tripo balance: 5 credits.

## In progress / next
- Arcanist layered .blend (`heroes/arcane_layers/arcanist_layers.blend`): first pass by texture colour is ragged (hair counted as robe, gaps in skin); base form failed. Needs a better split.
- Bell Witch files live in `~/.codex/visualizations` (Codex's folder, not accessible to Claude); copy one into Downloads to use it.
- Still open from the plan: full re-audit of worlds (clipping, pop-in), Cloud Kingdom brightness, Caverns darkness, vegetation pass, item review, playtest build.
