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
- Test model **C+** (10% curvier) is behind the "Body" button on the style test page; C stays the default until the user picks. Arms were cleaned up (the sleeves wrap the forearms) and the staff is thicker.
- The wide sleeves are the same sheer black material as the high neck (no red sleeves).
- Pose: fitted sheer sleeves that end at the wrists, right hand resting on her hip (the arm is fitted to each body's hip width), staff gripped in the left hand and clear to her feet.
- Polish: the staff ends in a carved spiral curl (as in the reference); the hat is layered brim, cone with a band, then the brim's front lip, so head lines no longer show through. New test model **C++** = C+ with a 15% fuller bust; the "Body" button cycles C, C+, C++.
- The neckline cups, under-bust curves, sheen and the sheer top now scale with the bust width, so larger bodies (C++) fill their outline.
- No ink lines run under the bust: only short outer accents and soft shading. **C++** is now C+ with a 25% fuller bust.
- Test models added: **C+++** (bust even with the hips: the agreed ceiling, never wider) and **C★** (the C++ bust sitting higher, stronger satin sheen, posed: bigger hip cock, slight lean, head tilt). The Body button cycles C, C+, C++, C+++, C★.

## Mirabel final (2026-11-05)

**Final** = the C★ pose (bigger hip cock, slight lean, head tilt), stronger satin sheen and
higher-set bust, with the C+++ proportions (bust as wide as the hips). It is the default on the
style test page (`mirabel_final.png`); the Body button still cycles the earlier test models.
- Large busts (wider than 44 units) get a rounded outline (upper curve, apex, lower curve, tuck into the underbust) and soft round shading on each side.
