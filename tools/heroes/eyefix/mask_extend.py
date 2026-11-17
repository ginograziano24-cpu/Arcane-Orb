# Widens the face paint mask (eyes_mask.png) over her right cheek and the nose
# bridge, with soft edges, so the rebuilt pieces and the skin around them are
# all coloured from the same clean front painting (no seams, no dark base
# texture showing between the brows). Run: python mask_extend.py in out
import sys
import numpy as np
from PIL import Image, ImageFilter

src, dst = sys.argv[1], sys.argv[2]
m = np.asarray(Image.open(src).convert("L")).astype(np.float32) / 255
H, W = m.shape
yy, xx = np.mgrid[0:H, 0:W].astype(np.float32)

def soft_box(x0, x1, y0, y1, f):
    a = np.clip((xx - x0) / f, 0, 1) * np.clip((x1 - xx) / f, 0, 1)
    b = np.clip((yy - y0) / f, 0, 1) * np.clip((y1 - yy) / f, 0, 1)
    return a * b

add = np.maximum(soft_box(1345, 1768, 560, 1015, 40), soft_box(1688, 1852, 556, 772, 18))
out = np.maximum(m, add)
img = Image.fromarray((np.clip(out, 0, 1) * 255).astype(np.uint8)).filter(ImageFilter.GaussianBlur(1.5))
img.save(dst)
print("mask covered", round(float(m.mean()), 4), "->", round(float(np.asarray(img).mean() / 255), 4))
