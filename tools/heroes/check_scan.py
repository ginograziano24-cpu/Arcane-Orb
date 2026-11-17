# Reads heroes/arcane_layers2/check_*.png (from arcane_check.py) and reports broken spots:
#   back faces (pure magenta) and see-through holes (clear pixels enclosed by her).
# Writes check_marked_<name>.png with every problem circled in red.
import sys, glob, os
import numpy as np
from PIL import Image, ImageDraw

def dil(m, n):
    for _ in range(n):
        p = np.pad(m, 1); m = m | p[:-2, 1:-1] | p[2:, 1:-1] | p[1:-1, :-2] | p[1:-1, 2:]
    return m

def ero(m, n):
    for _ in range(n):
        p = np.pad(m, 1, constant_values=True); m = m & p[:-2, 1:-1] & p[2:, 1:-1] & p[1:-1, :-2] & p[1:-1, 2:]
    return m

def blobs(m):
    lab = -np.ones(m.shape, int); out = []
    ys, xs = np.nonzero(m)
    for y0, x0 in zip(ys, xs):
        if lab[y0, x0] >= 0:
            continue
        st = [(y0, x0)]; lab[y0, x0] = len(out); pts = []
        while st:
            y, x = st.pop(); pts.append((y, x))
            for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                yy, xx = y + dy, x + dx
                if 0 <= yy < m.shape[0] and 0 <= xx < m.shape[1] and m[yy, xx] and lab[yy, xx] < 0:
                    lab[yy, xx] = len(out); st.append((yy, xx))
        out.append(np.array(pts))
    return out

d = sys.argv[1]
tot = 0
for f in sorted(glob.glob(os.path.join(d, "check_*.png"))):
    nm = os.path.basename(f)
    if nm in ("check_front.png", "check_turn.png") or nm.startswith("check_marked_"):
        continue
    a = np.asarray(Image.open(f).convert("RGBA")).astype(int)
    al = a[:, :, 3] > 127
    mag = al & (a[:, :, 0] > 225) & (a[:, :, 2] > 225) & (a[:, :, 1] < 40)
    holes = ero(dil(al, 4), 4) & ~al            # small clear gaps closed in by her
    bad = mag | holes
    bl = [b for b in blobs(bad) if len(b) >= 3]
    tot += len(bl)
    im = Image.open(f).convert("RGBA"); bg = Image.new("RGBA", im.size, (60, 62, 70, 255)); bg.alpha_composite(im)
    dr = ImageDraw.Draw(bg)
    for b in bl:
        y, x = b[:, 0].mean(), b[:, 1].mean(); r = max(8, int(np.sqrt(len(b))) + 6)
        dr.ellipse((x - r, y - r, x + r, y + r), outline=(255, 30, 30, 255), width=3)
    bg.convert("RGB").save(os.path.join(d, "check_marked_" + nm[6:]))
    print("%-24s back faces %5d px  holes %5d px  spots %d" % (nm, int(mag.sum()), int(holes.sum()), len(bl)))
print("total spots", tot)
