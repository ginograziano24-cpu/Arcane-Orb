"""Packs the Blender brick tiles into the game's two brick sheets:
python3 tools/textures/pack_bricks.py <tiles dir>

  assets/floor/bricks.jpg       diffuse (tinted in game for world bricks)
  assets/floor/bricks_spec.jpg  specular highlights (added on top)

Layout, 8 x 6 cells of 512 x 256: rows 0/1 tops/fronts of the world bricks
(neon meadow caverns ruins clouds machine steel elite), rows 2-3 tops of
the special bricks, rows 4-5 their fronts."""
import os, sys
from PIL import Image

CW, CH = 512, 256
WORLD = ["neon", "meadow", "caverns", "ruins", "clouds", "machine", "steel", "elite"]
SPECIAL = ["armored", "explosive", "chain", "split", "regen", "moving", "shield", "teleport",
           "spawner", "multiplier", "crystal", "mystery"]
ROOT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'floor')


def main(src):
    for pas, out in (("d", "bricks.jpg"), ("s", "bricks_spec.jpg")):
        sheet = Image.new("RGB", (CW * 8, CH * 6))
        missing = []

        def put(name, view, i, j):
            p = os.path.join(src, "%s_%s_%s.png" % (name, view, pas))
            if not os.path.exists(p):
                missing.append(os.path.basename(p))
                return
            im = Image.open(p).convert("RGB").resize((CW, CH), Image.LANCZOS)
            sheet.paste(im, (i * CW, j * CH))
        for i, n in enumerate(WORLD):
            put(n, "top", i, 0)
            put(n, "front", i, 1)
        for k, n in enumerate(SPECIAL):
            put(n, "top", k % 8, 2 + k // 8)
            put(n, "front", k % 8, 4 + k // 8)
        sheet.save(os.path.join(ROOT, out), quality=92, subsampling=0)
        print("wrote", out, "missing:", len(missing), missing[:6])


if __name__ == "__main__":
    main(sys.argv[1])
