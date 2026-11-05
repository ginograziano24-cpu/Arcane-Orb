import sys, os
import numpy as np
from PIL import Image
a, b = sys.argv[1], sys.argv[2]
out = sys.argv[3] if len(sys.argv) > 3 else None
for f in sorted(os.listdir(a)):
    if not f.endswith('.png') or not os.path.exists(os.path.join(b, f)): continue
    x = np.asarray(Image.open(os.path.join(a, f)).convert('RGB')).astype(int)
    y = np.asarray(Image.open(os.path.join(b, f)).convert('RGB')).astype(int)
    d = np.abs(x - y).max(axis=2)
    print('%-12s max %3d  mean %.3f  px>8 %6d  px>32 %6d' % (f[:-4], d.max(), d.mean(), (d > 8).sum(), (d > 32).sum()))
    if out:
        os.makedirs(out, exist_ok=True)
        Image.fromarray(np.clip(d * 4, 0, 255).astype('uint8')).save(os.path.join(out, f))
