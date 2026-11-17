import numpy as np, json
from PIL import Image, ImageDraw
from scipy import ndimage
src = np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGBA')).astype(np.float32) / 255
im = src[..., :3].copy()
H, W = im.shape[:2]
yy, xx = np.mgrid[0:H, 0:W].astype(np.float32)
curves = json.load(open('lidcurves.json'))
FACE_CX = 1768
mask = np.zeros((H, W), np.float32)
lash_col = np.array([0.08, 0.05, 0.06])
for (x0, x1, c) in curves:
    cols = np.arange(W)
    lid = np.polyval(c, np.clip(cols, x0, x1))
    inx = (cols >= x0 - 10) & (cols <= x1 + 10)
    outer_right = (x0 + x1) / 2 > FACE_CX
    # 1) lift the floating clumps off the lid: repaint the strip above the lid line as skin
    zone = (yy < lid[None, :] - 2) & (yy > lid[None, :] - 44) & inx[None, :]
    cur = np.clip(im, 0, 1)
    cmx = cur.max(2); csat = (cmx - cur.min(2)) / np.maximum(cmx, 1e-3)
    hairpx = ndimage.binary_opening((csat < 0.16) & (cmx > 0.4), iterations=2) & (yy < lid[None, :] - 10)
    zsoft = np.clip(ndimage.gaussian_filter(zone.astype(np.float32), 6) * 1.35, 0, 1)
    zsoft *= (yy < lid[None, :])               # never touch the eye or the lid line
    hk = np.clip(ndimage.gaussian_filter(hairpx.astype(np.float32), 1.5) * 1.5, 0, 1)
    wgt = zsoft * (1 - hk)
    skin_ = (cmx > 0.55) & (cur[..., 0] > cur[..., 2] * 1.05) & (csat > 0.12) & ~zone
    est = cur.copy(); got = np.zeros((H, W), bool)
    for sig in (8, 16, 30, 50):
        num = ndimage.gaussian_filter(cur * skin_[..., None], (sig, sig, 0))
        den = ndimage.gaussian_filter(skin_.astype(np.float32), sig)
        take = (~got) & (den > 0.06)
        est[take] = (num / np.maximum(den, 1e-4)[..., None])[take]; got |= take
    dist = np.clip(lid[None, :] - yy, 0, None)
    shade = (0.14 * np.exp(-dist / 14.0))[..., None]
    est = est * (1 - shade) + est * np.array([0.78, 0.6, 0.6]) * shade
    im = im * (1 - wgt[..., None]) + est * wgt[..., None]
    mask = np.maximum(mask, (wgt > 0.03).astype(np.float32))
    # 2) thick lashes rooted on the lid line, swept to the outer corner
    SS = 4
    bx0, by0 = x0 - 80, int(min(lid[x0:x1 + 1])) - 90
    bw_, bh_ = (x1 - x0) + 160, 140
    cv = Image.new("L", (bw_ * SS, bh_ * SS), 0); dr = ImageDraw.Draw(cv)
    P = lambda x, y: ((x - bx0) * SS, (y - by0) * SS)
    out = 1 if outer_right else -1
    X = np.arange(x0, x1 + 1)
    up, dn = [], []
    for x in X:
        u = (x - x0) / (x1 - x0); uo = u if outer_right else 1 - u
        y = np.polyval(c, x)
        th = 3.5 + 6.0 * uo ** 1.2
        up.append(P(x, y - th)); dn.append(P(x, y + 2.0))
    dr.polygon(up + dn[::-1], fill=255)
    ex = x1 if outer_right else x0; ey = np.polyval(c, ex)
    dr.polygon([P(ex, ey - 9), P(ex + out * 15, ey - 15), P(ex + out * 3, ey + 1)], fill=255)
    rng = np.random.default_rng(int(x0))
    tipcv = Image.new("L", (bw_ * SS, bh_ * SS), 0); tdr = ImageDraw.Draw(tipcv)
    def lash(u, L, bw, sweep, conv=None):
        uo = u if outer_right else 1 - u
        x = x0 + u * (x1 - x0)
        y = np.polyval(c, x) - (2.5 + 5.0 * uo)
        slope = np.polyval(np.polyder(c), x)
        nx, ny = slope, -1.0; ln = np.hypot(nx, ny); nx, ny = nx / ln, ny / ln
        dx, dy = nx + out * sweep, ny; dl = np.hypot(dx, dy); dx, dy = dx / dl, dy / dl
        tip = conv if conv is not None else (x + dx * L + out * L * 0.22, y + dy * L * 0.9 + L * 0.06)
        c1 = (x + dx * L * 0.45, y + dy * L * 0.62)
        left, right = [], []
        for q in range(13):
            t = q / 12
            bxp = (1 - t) ** 2 * x + 2 * (1 - t) * t * c1[0] + t * t * tip[0]
            byp = (1 - t) ** 2 * y + 2 * (1 - t) * t * c1[1] + t * t * tip[1]
            tx = 2 * (1 - t) * (c1[0] - x) + 2 * t * (tip[0] - c1[0]); ty = 2 * (1 - t) * (c1[1] - y) + 2 * t * (tip[1] - c1[1])
            tl = np.hypot(tx, ty) or 1
            px_, py_ = -ty / tl, tx / tl
            w_ = bw * (1 - t) ** 1.3 * 0.5 + 0.25
            left.append(P(bxp + px_ * w_, byp + py_ * w_)); right.append(P(bxp - px_ * w_, byp - py_ * w_))
        dr.polygon(left + right[::-1], fill=255)
        tdr.ellipse([P(tip[0] - 3, tip[1] - 3), P(tip[0] + 3, tip[1] + 3)], fill=255)
        return tip
    # clumps: 2-3 lashes whose tips meet, plus short fillers between them
    nclump = 7
    for j in range(nclump):
        u = 0.1 + 0.88 * (j / (nclump - 1)) ** 0.85
        uo = u if outer_right else 1 - u
        L = (22 + 24 * uo) * rng.uniform(0.9, 1.12)
        sweep = 0.15 + 0.85 * uo
        bw = (8.0 + 4.0 * uo)
        tip = lash(u, L, bw, sweep)
        for k in range(1 + (uo > 0.4)):
            du = rng.uniform(0.025, 0.045) * (1 if k == 0 else -1)
            uu = min(max(u + du, 0.03), 1.0)
            lash(uu, L * 0.92, bw * 0.75, sweep, conv=(tip[0] + rng.uniform(-2, 2), tip[1] + rng.uniform(1, 3)))
        if j < nclump - 1:
            uf = u + 0.5 * (0.88 / (nclump - 1))
            lash(min(uf, 1.0), L * 0.55, bw * 0.7, sweep * 0.9)
    cv = cv.resize((bw_, bh_), Image.LANCZOS)
    A = ndimage.gaussian_filter(np.array(cv).astype(np.float32) / 255, 0.75) * 0.95
    TIPS = ndimage.gaussian_filter(np.array(tipcv.resize((bw_, bh_), Image.LANCZOS)).astype(np.float32) / 255, 4)
    ys0, ys1 = max(0, by0), min(H, by0 + bh_); xs0, xs1 = max(0, bx0), min(W, bx0 + bw_)
    A = A[ys0 - by0:ys1 - by0, xs0 - bx0:xs1 - bx0]
    TIPS = np.clip(TIPS[ys0 - by0:ys1 - by0, xs0 - bx0:xs1 - bx0] * 2.5, 0, 1)
    colmap = lash_col * (1 - 0.6 * TIPS[..., None]) + np.array([0.22, 0.13, 0.12]) * 0.6 * TIPS[..., None]
    im[ys0:ys1, xs0:xs1] = im[ys0:ys1, xs0:xs1] * (1 - A[..., None]) + colmap * A[..., None]
    mask[ys0:ys1, xs0:xs1] = np.maximum(mask[ys0:ys1, xs0:xs1], ndimage.binary_dilation(A > 0.03, iterations=3).astype(np.float32))
mask = ndimage.gaussian_filter(mask, 1.5)
out_ = np.concatenate([np.clip(im, 0, 1), src[..., 3:4]], -1)
Image.fromarray((out_ * 255).astype(np.uint8)).save('eyes_fixed.png')
Image.fromarray((np.clip(mask, 0, 1) * 255).astype(np.uint8)).save('eyes_mask.png')
a = Image.open('/tmp/eyes_front_v1.png').convert('RGB').crop((1380, 560, 2180, 800))
bb = Image.open('eyes_fixed.png').convert('RGB').crop((1380, 560, 2180, 800))
cc = Image.new('RGB', (800, 490)); cc.paste(a, (0, 0)); cc.paste(bb, (0, 250)); cc.resize((1200, 735)).save('/tmp/eyes_ba.png')
