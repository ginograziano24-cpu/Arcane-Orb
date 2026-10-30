"""Bakes the paddle design sheet: python3 tools/textures/paddles.py

One 1024 x 256 band per origin (plus a default), in pairs of rows:
row 2i is the engraved shading (greyscale, tinted with the paddle colour),
row 2i+1 the glow mask (lit with the world's trim colour, additive)."""
import os
import numpy as np
from texlib import *

W, H = 1024, 256
OUT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'floor', 'paddles.jpg')
yy, xx = np.mgrid[0:H, 0:W].astype(float)
U, V = xx / W, yy / H
X, Y = (U - 0.5) * 4.0, (V - 0.5) * 1.0       # design space: 4 wide, 1 tall
ORIGINS = ['default', 'guardian', 'berserker', 'arcane', 'engineer', 'lucky', 'pyro', 'storm']


def line(d, w):
    return smoothstep(w, w * 0.35, np.abs(d))


def frame():
    e = np.minimum(np.minimum(U, 1 - U) * 4, np.minimum(V, 1 - V)) # design units to the edge
    return line(e - 0.07, 0.018), e


def bake(eng, glow):
    """eng: engraving depth 0..1 (cut in), glow: 0..1."""
    h = -eng * 0.01
    base = np.full((H, W, 3), 0.85) * (0.92 + 0.12 * fbm(H, W, 3, 12, 4, 7, periodic=False))[..., None]
    shade_img = shade(base, h, W / 4.0, light=(-0.4, 0.8, -0.45), ao=1.5, spec=0.25, periodic=False, amb=0.65)
    g = np.clip(glow, 0, 1)
    g = np.clip(g + blur(g, 4, False) * 0.8, 0, 1)
    return shade_img, np.repeat(g[..., None], 3, -1)


def d_default():
    f, e = frame()
    chev = line(np.mod(np.abs(X) * 2 - Y * 1.5, 1) - 0.5, 0.06) * (np.abs(X) > 0.5)
    return f + chev * 0.6, line(np.abs(Y) - 0.0, 0.02) * (np.abs(X) < 1.6) * 0.8


def d_guardian():
    f, e = frame()
    r = np.hypot(X, Y * 1.0)
    ang = np.arctan2(Y, X)
    sun = line(r - 0.28, 0.025) + (r < 0.16) * 1.0 + line(np.mod(ang * 12 / (2 * np.pi), 1) - 0.5, 0.12) * (r > 0.32) * (r < 0.42)
    fil = line(np.sin(X * 6) * 0.18 - Y, 0.025) * (np.abs(X) > 0.55) * (np.abs(X) < 1.8)
    fil += line(-np.sin(X * 6) * 0.18 - Y, 0.025) * (np.abs(X) > 0.55) * (np.abs(X) < 1.8)
    return np.clip(f + fil + sun * 0.8, 0, 1), np.clip(sun * 0.9 + f * 0.4, 0, 1)


def d_berserker():
    f, e = frame()
    teeth = line(np.abs(np.mod(X * 3, 1) - 0.5) * 0.8 - (0.42 - np.abs(Y)), 0.03) * (np.abs(Y) > 0.12)
    slash = line(X * 0.5 + Y * 1.2, 0.04) * (np.abs(X) < 0.6)
    slash += line(X * 0.5 + Y * 1.2 - 0.25, 0.03) * (np.abs(X - 0.4) < 0.5)
    return np.clip(f + teeth * 0.8 + slash, 0, 1), np.clip(slash * 1.0 + teeth * 0.3, 0, 1)


def d_arcane():
    f, e = frame()
    runes = np.zeros_like(X)
    rng = np.random.default_rng(3)
    for k in range(13):
        cx = -1.75 + k * 0.29
        a, b, c = rng.random(3)
        dx = X - cx
        g1 = line(dx, 0.012) * (np.abs(Y) < 0.22)
        g2 = line(dx - (Y - 0.1) * (a - 0.5) * 2, 0.012) * (Y > -0.05 + b * 0.1) * (np.abs(dx) < 0.1) * (np.abs(Y) < 0.22)
        g3 = line(Y - (c - 0.5) * 0.3, 0.012) * (np.abs(dx) < 0.08)
        runes += g1 + g2 + g3
    runes = np.clip(runes, 0, 1) * (np.abs(X) > 0.0)
    band = line(np.abs(Y) - 0.32, 0.015)
    return np.clip(f + band + runes * 0.7, 0, 1), np.clip(runes * 1.0 + band * 0.5, 0, 1)


def d_engineer():
    f, e = frame()
    haz = (np.mod((X + Y) * 3, 1) < 0.5) * (np.abs(X) > 1.45)
    gear = np.zeros_like(X)
    for cx in (-0.55, 0.55):
        r = np.hypot(X - cx, Y)
        a = np.arctan2(Y, X - cx)
        teeth = 0.27 + 0.05 * (np.mod(a * 10 / (2 * np.pi), 1) < 0.5)
        gear += line(r - teeth, 0.02) + line(r - 0.1, 0.02)
    pipe = line(np.abs(Y) - 0.0, 0.03) * (np.abs(X) > 0.85) * (np.abs(X) < 1.4)
    return np.clip(f + haz * 0.8 + gear * 0.9 + pipe, 0, 1), np.clip(gear * 0.6 + pipe * 1.0, 0, 1)


def d_lucky():
    f, e = frame()
    suits = np.zeros_like(X)
    for i, cx in enumerate((-1.35, -0.45, 0.45, 1.35)):
        dx, dy = X - cx, Y
        if i == 0:   # heart
            s = (np.hypot(dx + 0.07, dy + 0.06) < 0.09) | (np.hypot(dx - 0.07, dy + 0.06) < 0.09) | ((np.abs(dx) < 0.16 - (dy + 0.02) * 0.9) & (dy > -0.02) & (dy < 0.18))
        elif i == 1:  # diamond
            s = np.abs(dx) / 0.13 + np.abs(dy) / 0.2 < 1
        elif i == 2:  # club
            s = (np.hypot(dx, dy + 0.09) < 0.07) | (np.hypot(dx - 0.08, dy + 0.0) < 0.07) | (np.hypot(dx + 0.08, dy) < 0.07) | ((np.abs(dx) < 0.02) & (dy > 0) & (dy < 0.18))
        else:        # spade
            s = (np.hypot(dx + 0.06, dy + 0.02) < 0.08) | (np.hypot(dx - 0.06, dy + 0.02) < 0.08) | ((np.abs(dx) < 0.14 - (0.0 - dy) * -0.7) & (dy < 0.02) & (dy > -0.18)) | ((np.abs(dx) < 0.02) & (dy > 0) & (dy < 0.18))
        suits += s
    return np.clip(f + suits * 0.9, 0, 1), np.clip(suits * 0.8, 0, 1)


def d_pyro():
    f, e = frame()
    Y = -globals()['Y']  # flames point up the paddle
    fl = np.zeros_like(X)
    for k in range(9):
        cx = -1.6 + k * 0.4
        dx = X - cx
        hgt = 0.35 + 0.1 * (k % 3)
        tongue = (np.abs(dx) < (0.12 * (1 - (0.4 - Y) / hgt * 0.0)) * np.clip((Y + 0.4) / hgt, 0, 1) ** 0.5 * (1 - np.clip((Y + 0.4) / (hgt * 1.0), 0, 1)) * 3)
        fl += tongue
    fl = np.clip(fl, 0, 1)
    return np.clip(f + fl * 0.7, 0, 1), np.clip(fl, 0, 1)


def d_storm():
    f, e = frame()
    zz = line(Y - 0.25 * (np.abs(np.mod(X * 1.6, 1) - 0.5) * 4 - 1), 0.035) * (np.abs(X) < 1.8)
    zz2 = line(Y + 0.2 - 0.12 * (np.abs(np.mod(X * 3 + 0.3, 1) - 0.5) * 4 - 1), 0.02) * (np.abs(X) < 1.6)
    return np.clip(f + zz + zz2 * 0.6, 0, 1), np.clip(zz * 1.0 + zz2 * 0.6, 0, 1)


def main():
    designs = [d_default, d_guardian, d_berserker, d_arcane, d_engineer, d_lucky, d_pyro, d_storm]
    sheet = np.zeros((H * 16, W, 3))
    for i, d in enumerate(designs):
        eng, glow = d()
        s, g = bake(eng, glow)
        sheet[(2 * i) * H:(2 * i + 1) * H] = s
        sheet[(2 * i + 1) * H:(2 * i + 2) * H] = g
    to_img(sheet).save(OUT, quality=90, subsampling=0)
    print('wrote', OUT, sheet.shape)


if __name__ == '__main__':
    main()
