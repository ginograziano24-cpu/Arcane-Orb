#!/usr/bin/env python3
"""Paints Arcane Orb's world backdrops (sky + distant landscape) offline.

Output: assets/bg/<world>_<variant>.jpg  (2048x1152, horizon at 72% height)
        assets/bg/<world>_mid.png         (mid-ground silhouette layer, RGBA)
Everything is generated procedurally from noise; no external art is used.
"""
import os, sys, math
import numpy as np
from PIL import Image
from scipy.ndimage import gaussian_filter, zoom, map_coordinates

W, H = 2048, 1152
HY = int(H * 0.72)
OUT = os.path.join(os.path.dirname(__file__), "..", "assets", "bg")
os.makedirs(OUT, exist_ok=True)

# ------------------------------------------------------------------ helpers

def rgb(h):
    h = h.lstrip('#')
    return np.array([int(h[i:i + 2], 16) / 255 for i in (0, 2, 4)])

def noise2(w, h, cell, rng):
    gw, gh = max(2, int(w / cell) + 3), max(2, int(h / cell) + 3)
    g = rng.random((gh, gw))
    z = zoom(g, (h / (gh - 3) * 1.0, w / (gw - 3) * 1.0), order=3)
    return np.clip(z[:h, :w], 0, 1)

def fbm(w, h, cell, oct, rng, gain=0.5):
    a, s, tot = 1.0, cell, 0
    out = np.zeros((h, w))
    for _ in range(oct):
        out += a * noise2(w, h, s, rng)
        tot += a
        a *= gain
        s /= 2
    return out / tot

def noise1(w, cell, rng):
    n = max(2, int(w / cell) + 4)
    g = rng.random(n)
    x = np.linspace(0, n - 3, w)
    i = np.floor(x).astype(int)
    t = x - i
    # cubic interpolation
    p0, p1, p2, p3 = g[i - 1 + 1], g[i + 1], g[i + 2], g[np.minimum(i + 3, n - 1)]
    p0 = g[np.maximum(i, 0)]
    return p1 + 0.5 * t * (p2 - p0 + t * (2 * p0 - 5 * p1 + 4 * p2 - p3 + t * (3 * (p1 - p2) + p3 - p0)))

def fbm1(w, cell, oct, rng, gain=0.5, ridged=False):
    out, a, s, tot = np.zeros(w), 1.0, cell, 0
    for _ in range(oct):
        n = noise1(w, s, rng)
        if ridged:
            n = 1 - np.abs(n * 2 - 1)
        out += a * n
        tot += a
        a *= gain
        s /= 2
    return out / tot

def vgrad(top, bot, curve=1.0, y0=0, y1=None):
    y1 = y1 if y1 is not None else HY
    t = np.clip((np.arange(H) - y0) / max(1, (y1 - y0)), 0, 1) ** curve
    col = top[None, :] * (1 - t[:, None]) + bot[None, :] * t[:, None]
    return np.repeat(col[:, None, :], W, axis=1)

def radial(cx, cy, rx, ry=None):
    ry = ry or rx
    yy, xx = np.mgrid[0:H, 0:W]
    return np.sqrt(((xx - cx) / rx) ** 2 + ((yy - cy) / ry) ** 2)

def lerp(a, b, t):
    t = np.asarray(t)
    if t.ndim == 2:
        t = t[..., None]
    return a * (1 - t) + b * t

def ridge_mask(heights):
    """heights: array W of pixel heights above the horizon -> HxW mask."""
    yy = np.arange(H)[:, None]
    top = HY - heights[None, :]
    return (yy >= top).astype(float), top

def soft(mask, s=1.2):
    return gaussian_filter(mask, s)

def paste(img, mask, col):
    return lerp(img, col if np.ndim(col) == 3 else np.broadcast_to(col, img.shape), mask)

def godrays(img, cx, cy, col, strength, rng, n=28):
    yy, xx = np.mgrid[0:H, 0:W]
    ang = np.arctan2(yy - cy, xx - cx)
    d = np.sqrt((xx - cx) ** 2 + (yy - cy) ** 2)
    rays = np.zeros_like(ang)
    for k in range(n):
        a0 = rng.uniform(-math.pi, math.pi)
        wdt = rng.uniform(0.01, 0.05)
        rays += np.exp(-((np.angle(np.exp(1j * (ang - a0)))) / wdt) ** 2) * rng.uniform(0.3, 1)
    rays = gaussian_filter(rays, 3) * np.exp(-d / (W * 0.45))
    return img + col[None, None, :] * (rays[..., None] * strength)

def stars(img, rng, count, ymax, bright=1.0):
    for _ in range(count):
        x, y = rng.integers(0, W), rng.integers(0, int(ymax))
        b = rng.uniform(0.3, 1) * bright
        r = 1 if rng.random() < 0.9 else 2
        img[max(0, y - r):y + r, max(0, x - r):x + r] = np.maximum(img[max(0, y - r):y + r, max(0, x - r):x + r], b)
    return img

def grain(img, rng, amt=0.012):
    return img + (rng.random(img.shape[:2])[..., None] - 0.5) * amt

def save(img, name, q=90):
    img = np.clip(img, 0, 1)
    Image.fromarray((img * 255).astype(np.uint8)).save(os.path.join(OUT, name), quality=q, optimize=True)

def save_rgba(rgb_, a, name):
    out = np.dstack([np.clip(rgb_, 0, 1), np.clip(a, 0, 1)])
    Image.fromarray((out * 255).astype(np.uint8), "RGBA").save(os.path.join(OUT, name), optimize=True)

def clouds(img, rng, ymin, ymax, cover, lit, shade, scale=380, alpha=0.9, stretch=2.6):
    n = fbm(W, H, scale, 6, rng)
    yy = np.arange(H)[:, None]
    band = np.clip(1 - np.abs((yy - (ymin + ymax) / 2) / ((ymax - ymin) / 2)), 0, 1) ** 0.6
    d = np.clip((n * band - (1 - cover)) * 4, 0, 1)
    # lighting from the gradient of the density field (light from below/sun side)
    gy = np.gradient(gaussian_filter(d, 6), axis=0)
    l = np.clip(0.5 - gy * 18, 0, 1)
    col = lerp(np.broadcast_to(shade, img.shape), np.broadcast_to(lit, img.shape), l)
    return lerp(img, col, d * alpha)

# ------------------------------------------------------------------ worlds

def neon(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#12052c'), rgb('#ff4f9a'), 1.7)
    neb = fbm(W, H, 500, 6, rng)
    img = lerp(img, rgb('#7a2bd6'), np.clip((neb - 0.5) * 1.6, 0, 1) * (np.arange(H)[:, None] < HY) * 0.45)
    img = stars(img, rng, 900, HY * 0.65)
    # sun with stripes and glow
    sx, sr = W * rng.uniform(0.38, 0.62), 230
    sy = HY - sr * 0.45
    r = radial(sx, sy, sr)
    glow = np.exp(-np.maximum(r - 1, 0) * 1.6)
    img = img + rgb('#ff5fb4')[None, None, :] * (glow[..., None] * 0.45)
    yy = np.mgrid[0:H, 0:W][0]
    t = np.clip((yy - (sy - sr)) / (2 * sr), 0, 1)
    sun = lerp(np.broadcast_to(rgb('#ffe36e'), img.shape), np.broadcast_to(rgb('#ff3aa0'), img.shape), t)
    stripe = np.ones_like(t)
    lower = t > 0.42
    stripe[lower] = (np.mod((t[lower] - 0.42) * 26, 1) > (t[lower] - 0.42) * 1.9).astype(float)
    m = soft((r < 1).astype(float) * stripe * (yy < HY), 1.0)
    img = lerp(img, sun, m)
    # wireframe mountains: two ranges with neon grid lines on their faces
    for li, (base, amp, col, line, alpha) in enumerate([(70, 210, '#2a0f4a', '#3ef0ff', 0.55), (30, 150, '#160629', '#ff5fd0', 0.9)]):
        h = base + amp * fbm1(W, 420 - li * 120, 5, rng, ridged=True) ** 1.6
        h *= np.clip(np.abs(np.arange(W) - sx) / 420, 0.25, 1) if li == 1 else 1
        mask, top = ridge_mask(h)
        body = np.broadcast_to(rgb(col), img.shape).copy()
        # grid: horizontal contour lines + vertical lines
        rel = (HY - yy) / np.maximum(h[None, :], 1)
        grid = (np.abs(np.mod(rel * 7, 1) - 0.5) > 0.47) | (np.abs(np.mod(np.arange(W)[None, :] / 34.0, 1) - 0.5) > 0.47)
        body = lerp(body, rgb(line), grid * 0.35 * mask)
        rim = soft(((yy >= top) & (yy < top + 3)).astype(float), 1)
        img = paste(img, soft(mask, 0.8) * alpha, body)
        img = img + rgb(line)[None, None, :] * rim[..., None] * 0.9
    if variant == 1:  # neon city on the horizon
        x = 0
        while x < W:
            bw, bh = rng.integers(26, 80), rng.uniform(40, 240)
            if abs(x - sx) < sr * 0.8:
                bh *= 0.3
            img[int(HY - bh):HY, x:x + bw] = rgb('#0b0418')
            for wy in range(int(HY - bh + 8), HY - 6, 12):
                for wx in range(x + 5, x + bw - 5, 9):
                    if rng.random() < 0.3:
                        img[wy:wy + 4, wx:wx + 4] = rgb('#ffb2f0') if rng.random() < 0.5 else rgb('#7ff6ff')
            img[int(HY - bh):int(HY - bh) + 2, x:x + bw] = rgb('#ff4fd8')
            x += bw + rng.integers(2, 10)
    # ground below horizon
    g = vgrad(rgb('#ff4f9a'), rgb('#0e0420'), 0.5, HY, HY + 70)
    img[HY:] = g[HY:]
    return grain(img, rng)

def cumulus(img, rng, n, ymin, ymax, lit, shade, rim):
    yy, xx = np.mgrid[0:H, 0:W]
    for _ in range(n):
        cx, cy = rng.uniform(0, W), rng.uniform(ymin, ymax)
        cw, ch = rng.uniform(180, 420), rng.uniform(60, 130)
        dens = np.zeros((H, W))
        for k in range(14):
            bx = cx + rng.normal(0, cw * 0.35)
            by = cy - abs(rng.normal(0, ch * 0.35))
            br = rng.uniform(0.35, 0.7) * ch
            dens = np.maximum(dens, np.clip(1 - radial(bx, by, br * 1.25, br), 0, 1))
        flat = yy < cy + ch * 0.15  # flat cloud base
        dens = dens * flat
        tex = fbm(W, H, 40, 4, rng)
        dens = np.clip(dens * 2.2 * (0.75 + 0.5 * tex) - 0.25, 0, 1)
        dens = gaussian_filter(dens, 1.2)
        gy = gaussian_filter(np.gradient(gaussian_filter(dens, 8), axis=0), 2)
        light = np.clip(0.55 + gy * 26 + (cy - yy) / (ch * 3), 0, 1)
        col = lerp(np.broadcast_to(shade, img.shape), np.broadcast_to(lit, img.shape), light)
        edge = np.clip(dens * (1 - dens) * 4, 0, 1)
        col = col + rim[None, None, :] * edge[..., None] * 0.25
        img = lerp(img, col, np.clip(dens * 1.15, 0, 0.97))
    return img

def meadow(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#4268b0'), rgb('#ffbb84'), 1.35)
    sx, sy = W * rng.uniform(0.3, 0.7), HY - rng.uniform(70, 130)
    r = radial(sx, sy, 700, 480)
    img = img + rgb('#ffd29c')[None, None, :] * (np.exp(-r * 2.0)[..., None] * 0.6)
    img = lerp(img, rgb('#fff6e2'), soft((radial(sx, sy, 64) < 1).astype(float), 2.5))
    img = godrays(img, sx, sy, rgb('#ffe2b8'), 0.06, rng, n=14)
    img = cumulus(img, rng, 7, 120, HY * 0.62, rgb('#fff6ec'), rgb('#d78a96'), rgb('#ffd8a8'))
    yy, xx = np.mgrid[0:H, 0:W]
    # far blue mountains
    h = 120 + 130 * fbm1(W, 500, 5, rng, ridged=True) ** 1.5
    mask, top = ridge_mask(h)
    body = lerp(np.broadcast_to(rgb('#9aa4cc'), img.shape), np.broadcast_to(rgb('#c3b4c8'), img.shape), np.clip((yy - top) / 120, 0, 1))
    img = paste(img, soft(mask, 0.8), lerp(body, rgb('#ffcfa4'), 0.35))
    layers = [(84, 46, '#86a77c', '#b9c98e', 0.32), (52, 40, '#6a9a56', '#a2c272', 0.2), (22, 30, '#4f8442', '#86b05e', 0.08)]
    for li, (base, amp, col, top_col, haze) in enumerate(layers):
        h = base + amp * (0.6 * fbm1(W, 900, 3, rng) + 0.4 * np.sin(np.arange(W) / (260 - li * 50) + rng.uniform(0, 6)) * 0.5 + 0.2)
        mask, top = ridge_mask(h)
        tex = fbm(W, H, 50, 4, rng)
        depth = np.clip((yy - top) / 70, 0, 1)
        body = lerp(np.broadcast_to(rgb(top_col), img.shape), np.broadcast_to(rgb(col), img.shape), depth)
        body = body * (0.92 + 0.16 * tex[..., None])
        # sunlit side of each swell
        slope = np.gradient(h)
        lit = np.clip(0.5 + np.sign(np.arange(W) - sx) * -slope * 0.6, 0.3, 1)[None, :]
        body = body * (0.88 + 0.22 * lit)[..., None]
        body = lerp(body, rgb('#ffcfa4'), haze)
        img = paste(img, soft(mask, 0.8), body)
        # sparse trees and hedgerows sitting on the crest
        for _ in range(8 + li * 6):
            x = rng.integers(0, W)
            ty = top[0, x] + 2
            sc = (0.6 + li * 0.5) * rng.uniform(0.8, 1.3)
            tcol = lerp(rgb('#3f6a34') * (0.9 + 0.1 * li), rgb('#ffcfa4'), haze * 0.9)
            img[int(ty - 10 * sc):int(ty + 2), int(x - 1.5 * sc):int(x + 1.5 * sc) + 1] = lerp(rgb('#5a4030'), rgb('#ffcfa4'), haze)
            for k in range(4):
                bx, by = x + rng.normal(0, 4 * sc), ty - 14 * sc - rng.uniform(0, 8 * sc)
                br = rng.uniform(6, 10) * sc
                m = soft((radial(bx, by, br, br * 0.9) < 1).astype(float), 0.7)
                shade_side = np.clip((xx - bx) * np.sign(sx - bx) / br, -1, 1)
                img = paste(img, m, tcol[None, None, :] * (1 + 0.18 * shade_side)[..., None])
    if variant == 1:  # a distant farmhouse and silo on the middle hill
        fx = int(W * rng.uniform(0.15, 0.85))
        top_y = int(HY - (52 + 40 * 0.6))
        wall = lerp(rgb('#b8735a'), rgb('#ffcfa4'), 0.25)
        img[top_y - 34:top_y, fx:fx + 60] = wall
        for k in range(26):
            img[top_y - 34 - k, fx - 4 + k:fx + 64 - k] = lerp(rgb('#6b3a33'), rgb('#ffcfa4'), 0.25)
        img[top_y - 78:top_y, fx + 74:fx + 92] = lerp(rgb('#e3d8c8'), rgb('#ffcfa4'), 0.25)
        img = paste(img, soft((radial(fx + 83, top_y - 78, 9, 6) < 1).astype(float) * (yy < top_y - 76), 0.6), lerp(rgb('#9a8f88'), rgb('#ffcfa4'), 0.25))
    img[HY:] = vgrad(rgb('#6a9a52'), rgb('#3f6634'), 1, HY, H)[HY:]
    return grain(img, rng)

def caverns(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#05070f'), rgb('#18284a'), 1.2)
    yy, xx = np.mgrid[0:H, 0:W]
    # back cave wall texture
    rock = fbm(W, H, 260, 6, rng)
    img = img * (0.6 + 0.7 * rock[..., None])
    # glowing crystal veins and distant formations
    for _ in range(26):
        cx, ch = rng.uniform(0, W), rng.uniform(80, 360)
        cw = rng.uniform(16, 50)
        col = rgb(rng.choice(['#58d0ff', '#a07bff', '#7cffe0', '#ff8ce0', '#c8e6ff']))
        base = HY + rng.uniform(-10, 20)
        tri = (np.abs(xx - cx) < cw * np.clip((base - yy) / ch, 0, 1) * 0) | False
        t = np.clip((base - yy) / ch, 0, 1)
        half = cw * (1 - t)
        m = ((np.abs(xx - cx) < half) & (yy < base) & (yy > base - ch)).astype(float)
        shade = np.clip(0.55 + 0.45 * (xx - cx) / np.maximum(half, 1), 0, 1)
        c = lerp(col * 0.35, col, shade)
        glow = np.exp(-radial(cx, base - ch * 0.4, cw * 5, ch * 0.9) * 1.5)
        img = img + col[None, None, :] * glow[..., None] * 0.25
        img = paste(img, soft(m, 0.8) * 0.9, c)
    # light shafts from cracks in the ceiling
    for _ in range(3):
        x0 = rng.uniform(0.15, 0.85) * W
        slope = rng.uniform(-0.25, 0.25)
        d = np.abs(xx - (x0 + slope * yy))
        shaft = np.exp(-(d / rng.uniform(40, 90)) ** 2) * np.clip(1 - yy / HY, 0, 1) ** 0.5
        img = img + rgb('#9fdcff')[None, None, :] * shaft[..., None] * 0.12
    # stalactites with rim light
    for _ in range(34):
        x, l, w = rng.uniform(0, W), rng.uniform(80, 460), rng.uniform(26, 80)
        t = np.clip(yy / l, 0, 1)
        half = w * (1 - t) ** 1.2
        m = ((np.abs(xx - x) < half) & (yy < l)).astype(float)
        side = np.clip((xx - x) / np.maximum(half, 1), -1, 1)
        rock = lerp(rgb('#0c1120'), rgb('#1f2c46'), np.clip(-side * 0.5 + 0.5, 0, 1) * (1 - yy / max(l, 1)) * 0.9)
        img = paste(img, soft(m, 0.7), rock)
        rim = np.clip(1 - np.abs(side + 0.92) * 12, 0, 1) * (m > 0)
        img = img + rgb('#5a8cd0')[None, None, :] * rim[..., None] * 0.4
    # mist
    mist = fbm(W, H, 300, 5, rng) * np.exp(-((yy - HY) / 120.0) ** 2)
    img = lerp(img, rgb('#2c5a8c'), np.clip(mist * 0.8, 0, 0.6))
    for _ in range(500):
        x, y = rng.integers(0, W), rng.integers(0, HY)
        img[y:y + 2, x:x + 2] += rgb(rng.choice(['#7ff6ff', '#c9a6ff'])) * rng.uniform(0.2, 0.6)
    img[HY:] = vgrad(rgb('#1a2a48'), rgb('#0a0e1c'), 1, HY, H)[HY:]
    return grain(img, rng, 0.02)

def ruins(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#3d8fc6'), rgb('#f4e2bd'), 1.3)
    sx, sy = W * rng.uniform(0.2, 0.8), rng.uniform(120, 260)
    r = radial(sx, sy, 700, 600)
    img = img + rgb('#fff7e0')[None, None, :] * (np.exp(-r * 2.5)[..., None] * 0.55)
    img = lerp(img, rgb('#ffffff'), soft((radial(sx, sy, 48) < 1).astype(float), 2))
    img = clouds(img, rng, 60, HY * 0.5, 0.35, rgb('#ffffff'), rgb('#cbd8e8'), scale=520, alpha=0.55)
    yy, xx = np.mgrid[0:H, 0:W]
    # stratified mesas at several distances
    for li in range(3):
        haze = 0.62 - li * 0.2
        for _ in range(3 + li):
            x0, w0 = rng.uniform(0, W), rng.uniform(160, 420)
            h0 = rng.uniform(70, 190) * (1 - li * 0.15)
            prof = np.clip(h0 - np.maximum(np.abs(np.arange(W) - x0) - w0 / 2, 0) * rng.uniform(1.4, 2.6), 0, None)
            prof += fbm1(W, 40, 3, rng) * 8 * (prof > 0)
            mask, top = ridge_mask(prof)
            strata = 0.85 + 0.15 * np.sin((yy - top) * 0.25 + fbm(W, H, 80, 3, rng) * 6)
            lit = np.clip(0.75 + 0.35 * np.sign(xx - x0) * (1 if sx > x0 else -1), 0.5, 1.1)
            body = rgb('#c27a4a')[None, None, :] * (strata * lit)[..., None]
            body = lerp(body, rgb('#f4e2bd'), haze)
            img = paste(img, soft(mask, 0.8), body)
    # a step pyramid and an obelisk on the far plain
    px = W * rng.uniform(0.15, 0.85)
    for s in range(9):
        y1 = HY - 4 - s * 22
        half = 260 - s * 27
        lit = 1.05 if s % 2 == 0 else 0.95
        img[int(y1 - 22):int(y1), int(px - half):int(px + half)] = lerp(rgb('#d2a46e') * lit, rgb('#f4e2bd'), 0.4)
        img[int(y1 - 22):int(y1), int(px):int(px + half)] *= 0.85
    ox = W * rng.uniform(0.1, 0.9)
    for k in range(260):
        half = 14 - k * 0.04
        img[HY - 10 - k, int(ox - half):int(ox + half)] = lerp(rgb('#b98a5c'), rgb('#f4e2bd'), 0.35)
    # dust haze near the horizon
    dust = fbm(W, H, 240, 4, rng) * np.exp(-((yy - HY) / 90.0) ** 2)
    img = lerp(img, rgb('#f1d6a8'), np.clip(dust * 0.9, 0, 0.7))
    img[HY:] = vgrad(rgb('#e8cc98'), rgb('#c79c66'), 1, HY, H)[HY:]
    return grain(img, rng)

def cloudsky(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#4e93e8'), rgb('#e4efff'), 1.2)
    sx, sy = W * rng.uniform(0.25, 0.75), HY * rng.uniform(0.25, 0.5)
    r = radial(sx, sy, 800, 700)
    img = img + rgb('#fff8e0')[None, None, :] * (np.exp(-r * 2.2)[..., None] * 0.6)
    img = lerp(img, rgb('#ffffff'), soft((radial(sx, sy, 60) < 1).astype(float), 3))
    img = godrays(img, sx, sy, rgb('#fffbe8'), 0.07, rng, n=12)
    img = clouds(img, rng, 40, HY * 0.55, 0.38, rgb('#ffffff'), rgb('#b7c7ea'), scale=600, alpha=0.75)
    yy, xx = np.mgrid[0:H, 0:W]
    # distant floating islands with castles and waterfalls
    for _ in range(3 + variant):
        ix, iy, iw = rng.uniform(0.08, 0.92) * W, HY * rng.uniform(0.35, 0.8), rng.uniform(80, 170)
        col = lerp(rgb('#8a7f96'), rgb('#e4efff'), 0.45)
        t = np.clip((yy - iy) / (iw * 0.9), 0, 1)
        half = iw * (1 - t) ** 0.8
        m = ((np.abs(xx - ix) < half) & (yy >= iy) & (yy < iy + iw * 0.9)).astype(float)
        img = paste(img, soft(m, 1), col * (0.85 + 0.15 * fbm(W, H, 40, 2, rng))[..., None])
        img[int(iy - 6):int(iy), int(ix - iw):int(ix + iw)] = lerp(rgb('#7cc46a'), rgb('#e4efff'), 0.4)
        for k in range(3):
            tx = ix + (k - 1) * iw * 0.45
            th = rng.uniform(40, 90)
            img[int(iy - 6 - th):int(iy - 6), int(tx - 10):int(tx + 10)] = lerp(rgb('#ffffff'), rgb('#e4efff'), 0.3)
            for j in range(30):
                hw = 13 - j * 0.43
                img[int(iy - 6 - th - j), int(tx - hw):int(tx + hw)] = lerp(rgb(['#7f8ff0', '#f08fb8', '#ffd27a'][k]), rgb('#e4efff'), 0.35)
        wf = np.exp(-((xx - (ix + iw * 0.3)) / 6.0) ** 2) * ((yy > iy) & (yy < iy + iw * 2.2))
        img = lerp(img, rgb('#ffffff'), wf * 0.6)
    # the cloud sea, lit from above
    # billowing cloud sea: puffy tops lit by the sun, cool shadows beneath
    puff = np.zeros((H, W))
    for _ in range(140):
        bx = rng.uniform(-100, W + 100)
        by = HY - 70 + rng.uniform(-30, 90) + (bx - W / 2) ** 2 * 0
        br = rng.uniform(40, 120)
        puff = np.maximum(puff, np.clip(1 - radial(bx, by, br * 1.7, br), 0, 1))
    puff = np.maximum(puff, np.clip((yy - (HY - 20)) / 60.0, 0, 1))
    tex = fbm(W, H, 60, 5, rng)
    dens = np.clip(puff * 2.4 * (0.8 + 0.4 * tex) - 0.2, 0, 1)
    gy = gaussian_filter(np.gradient(gaussian_filter(dens, 10), axis=0), 3)
    light = np.clip(0.6 - gy * 30, 0.35, 1.0)
    sea = lerp(np.broadcast_to(rgb('#a9a6d8'), img.shape), np.broadcast_to(rgb('#ffffff'), img.shape), light)
    sea = lerp(sea, rgb('#ffe9d2'), np.exp(-radial(sx, HY, 600, 140))[..., None] * 0.35)
    img = lerp(img, sea, dens)
    return grain(img, rng, 0.008)

def machine(seed, variant):
    rng = np.random.default_rng(seed)
    img = vgrad(rgb('#141217'), rgb('#b4562a'), 1.6)
    yy, xx = np.mgrid[0:H, 0:W]
    glow = np.exp(-radial(W / 2, HY, W * 0.7, 260) * 1.2)
    img = img + rgb('#ff7a2e')[None, None, :] * glow[..., None] * 0.45
    smog = fbm(W, H, 420, 6, rng)
    img = lerp(img, rgb('#3d2c2a'), np.clip((smog - 0.42) * 1.5, 0, 0.75) * (yy < HY))
    dark = rgb('#17151a')
    mid = rgb('#2a2328')
    # far refinery layer (hazy)
    x = 0
    while x < W:
        bw, bh = rng.integers(50, 160), rng.uniform(60, 200)
        img[int(HY - bh):HY, x:x + bw] = lerp(mid, rgb('#b4562a'), 0.35)
        x += bw + rng.integers(0, 40)
    # cooling towers
    for _ in range(2):
        cx, ch, cw = rng.uniform(0.1, 0.9) * W, rng.uniform(180, 300), rng.uniform(70, 110)
        t = np.clip((HY - yy) / ch, 0, 1)
        half = cw * (0.62 + 0.38 * (2 * t - 1) ** 2)
        m = ((np.abs(xx - cx) < half) & (yy < HY) & (yy > HY - ch)).astype(float)
        img = paste(img, soft(m, 1), lerp(rgb('#3a3236'), rgb('#b4562a'), 0.25) * (0.9 + 0.2 * np.clip((xx - cx) / cw, -1, 1))[..., None])
        plume = fbm(W, H, 90, 5, rng) * np.exp(-radial(cx + 60, HY - ch - 120, 160, 140) * 1.3)
        img = lerp(img, rgb('#8c7a76'), np.clip(plume * 1.4, 0, 0.8))
    # near skyline: chimneys, gantries, cranes, lit windows
    x = 0
    while x < W:
        bw, bh = rng.integers(40, 130), rng.uniform(40, 150)
        img[int(HY - bh):HY, x:x + bw] = dark
        for wy in range(int(HY - bh + 10), HY - 6, 14):
            for wx in range(x + 6, x + bw - 6, 11):
                if rng.random() < 0.28:
                    img[wy:wy + 4, wx:wx + 5] = rgb('#ffae4a') * rng.uniform(0.7, 1.1)
        if rng.random() < 0.45:
            cx = x + bw // 2
            ch = bh + rng.uniform(60, 180)
            img[int(HY - ch):int(HY - bh), cx - 7:cx + 7] = dark
            img[int(HY - ch):int(HY - ch) + 4, cx - 9:cx + 9] = rgb('#ff8c3a')
            plume = fbm(W, H, 50, 4, rng) * np.exp(-radial(cx + 40, HY - ch - 70, 70, 90) * 1.5)
            img = lerp(img, rgb('#6f6260'), np.clip(plume * 1.5, 0, 0.7))
        x += bw + rng.integers(0, 14)
    # crane
    cx = rng.uniform(0.2, 0.8) * W
    img[int(HY - 330):HY, int(cx):int(cx + 10)] = dark
    img[int(HY - 330):int(HY - 320), int(cx - 200):int(cx + 120)] = dark
    for k in range(0, 300, 18):
        img[int(HY - 330 + 10):int(HY - 330 + 10 + 2 * (k % 36 == 0) + 1), int(cx - 200 + k):int(cx - 196 + k)] = dark
    img[int(HY - 334):int(HY - 330), int(cx - 2):int(cx + 12)] = rgb('#ff4030')
    for _ in range(140):  # floating embers
        x, y = rng.integers(0, W), rng.integers(0, HY)
        img[y:y + 2, x:x + 2] += rgb('#ff9a4a') * rng.uniform(0.2, 0.7)
    img[HY:] = vgrad(rgb('#2a2228'), rgb('#141216'), 1, HY, H)[HY:]
    return grain(img, rng, 0.02)

WORLDS = {
    "neon": neon, "meadow": meadow, "caverns": caverns,
    "ruins": ruins, "clouds": cloudsky, "machine": machine,
}

if __name__ == "__main__":
    only = sys.argv[1:] or list(WORLDS)
    for name in only:
        for v in range(2):
            img = WORLDS[name](1000 + v * 77 + sum(map(ord, name)) * 13, v)
            save(img, f"{name}_{v}.jpg", q=90)
            print("painted", name, v, flush=True)
