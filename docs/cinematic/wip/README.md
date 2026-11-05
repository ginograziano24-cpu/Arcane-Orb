# Opening cinematic: work in progress (on hold)

The opening cinematic is **on hold** and is **not part of any playtest**. The
game itself never plays it: it exists only as browser previews in this folder
and in `../opening_preview.html`.

- `../opening_preview.html`: preview 5, the last version before the Mirabel
  redesign (68 s, sketch-to-arcane, comic pass).
- `opening_redesign_attempt.html`: an unfinished attempt. It has a painted
  Mirabel (her own violet and magenta mage outfit), sprites baked during
  loading, render-ahead double buffering for the sketch half, and every cache
  pre-built behind the loading screen. The user didn't like the redesign: her
  look didn't land, and the painted style clashes with the pencil scenes.
  Keep the performance parts (render-ahead, prewarm, frame-timing hooks in
  `window.__perf`); they were never measured.
- `opening_src.html`: the source of that attempt (build with the scratchpad
  `build.py`, which embeds the hero busts). `mirabel_painted.js` is the
  standalone painter for the redesign.

Before picking it up again: the user will write a new brief for Mirabel and
the overall art direction.
