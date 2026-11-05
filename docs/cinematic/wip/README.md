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

## Style tests and the final Mirabel (2026-11-05)

- `style_tests.html`: the intro with buttons for four art styles (pencil sketch,
  ink & watercolour, comic, flat cartoon). The magic half is the same in all four.
- **Mirabel's final design is the user's illustration**, `mirabel_art.png`: long wavy
  dark hair, purple witch hat with a star, strapless red gown with heart motifs,
  sheer black high-neck top and sleeves, red bell sleeves, a leg slit, spiral
  staff. The page draws it, cut out from its background, in the daydream, the
  end scene and the title cameo. A redrawn vector version is behind the
  "Mirabel: redrawn" toggle, but the user said it doesn't look like the
  reference, so use the illustration.

## Mirabel decided (2026-11-05)

- Look: **ink & watercolour**, used only on the drawn segments (the pencil half and the
  paper in the burn shot). The magic half is unchanged.
- Mirabel: the **redrawn** model, body **C · Curvy** with hips 15% wider and **no leg slit**
  (`mirabel_drawn_C.png`). It is the default on the style test page; the illustration
  is still available behind the "Mirabel" toggle as the reference.
