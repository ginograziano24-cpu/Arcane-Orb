"""Packs the Blender origin portraits into the game's portrait files:
python3 tools/textures/pack_portraits.py <out/portraits dir> [id ...]

  assets/portraits/<id>.jpg       colour, with a soft bloom added
  assets/portraits/<id>_a.png     alpha: a soft-edged card, kept solid where
                                  the figure runs off the bottom edge
  assets/portraits/<id>.json      live-effect anchors
  assets/portraits/hud/<id>_<pose>.png  HUD poses (transparent)
"""
import json, os, shutil, sys
import numpy as np
from PIL import Image, ImageFilter

ROOT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'portraits')
IDS = ["arcane", "berserker", "guardian", "engineer", "lucky", "pyro", "storm"]
POSES = ["rest", "look", "fierce", "hurt"]


def bloom(img, strength=0.55):
    a = np.asarray(img.convert("RGB")).astype(np.float32) / 255
    lum = a @ np.array([0.2126, 0.7152, 0.0722], np.float32)
    bright = a * np.clip((lum - 0.6) / 0.4, 0, 1)[..., None]
    bi = Image.fromarray((bright * 255).astype(np.uint8))
    out = a.copy()
    for r, k in ((6, 0.5), (22, 0.35), (70, 0.25)):
        b = np.asarray(bi.filter(ImageFilter.GaussianBlur(r))).astype(np.float32) / 255
        out += b * k * strength
    out = np.clip(out, 0, 1)
    return Image.fromarray((out * 255 + 0.5).astype(np.uint8))


def smooth(e0, e1, x):
    t = np.clip((x - e0) / (e1 - e0), 0, 1)
    return t * t * (3 - 2 * t)


def card_alpha(w, h, mask):
    ys, xs = np.mgrid[0:h, 0:w].astype(np.float32)
    u, v = xs / w, ys / h
    edge = smooth(0.02, 0.16, u) * smooth(0.02, 0.16, 1 - u) * smooth(0.0, 0.12, v) * smooth(0.0, 0.2, 1 - v)
    m = np.asarray(mask.split()[-1].filter(ImageFilter.GaussianBlur(3))).astype(np.float32) / 255
    keep = m * smooth(0.45, 0.8, v)
    return np.maximum(edge, keep)


def main(src, ids):
    os.makedirs(os.path.join(ROOT, "hud"), exist_ok=True)
    for oid in ids:
        p = os.path.join(src, oid + ".png")
        if not os.path.exists(p):
            print("missing", oid)
            continue
        img = bloom(Image.open(p))
        img.save(os.path.join(ROOT, oid + ".jpg"), quality=90, subsampling=0)
        a = card_alpha(img.width, img.height, Image.open(os.path.join(src, oid + "_mask.png")).convert("RGBA"))
        Image.fromarray((a * 255).astype(np.uint8), "L").save(os.path.join(ROOT, oid + "_a.png"), optimize=True)
        shutil.copy(os.path.join(src, oid + ".json"), os.path.join(ROOT, oid + ".json"))
        for pose in POSES:
            hp = os.path.join(src, "hud", "%s_%s.png" % (oid, pose))
            if os.path.exists(hp):
                Image.open(hp).convert("RGBA").save(os.path.join(ROOT, "hud", "%s_%s.png" % (oid, pose)), optimize=True)
        print("packed", oid)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2:] or IDS)
