"""Bakes the brick texture sheet: python3 tools/textures/bricks.py

The sheet is 8 x 6 cells of 256 x 128 px. Bricks are drawn as boxes: the
top face samples a 'top' cell and the faces facing the camera sample a
'side' cell. World cells are near-white (the game tints them with the
brick's colour); special bricks carry their own markings.

  row 0  tops:  neon meadow caverns ruins clouds machine steel elite
  row 1  sides: same order
  row 2  special tops: armored explosive chain split regen moving shield teleport
  row 3  special tops: spawner multiplier crystal mystery
  row 4  special sides: armored explosive chain split regen moving shield teleport
  row 5  special sides: spawner multiplier crystal mystery
"""
import os
import numpy as np
from texlib import *

CW, CH = 256, 128
OUT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'floor', 'bricks.jpg')
yy, xx = np.mgrid[0:CH, 0:CW].astype(float)
U = xx / CW            # 0..1 across the face
V = yy / CH            # 0..1 down the face (top face: 0 = far edge)
PX = CW / 1.0          # px per world unit across a 1.0-wide brick
E = np.minimum(np.minimum(U, 1 - U) * 1.0, np.minimum(V, 1 - V) * 0.5)  # world units to the edge


def nz(cy, cx, o=4, seed=0):
    return fbm(CH, CW, cy, cx, o, seed, periodic=True)


def bevel(e, w=0.06, depth=0.04):
    t = np.clip(e / w, 0, 1)
    return np.sqrt(1 - (1 - t) ** 2) * depth


def lit(alb, h, emissive=None, spec=0.15, ao=0.8, light=(-0.4, 0.8, -0.45)):
    return shade(alb, h, PX, light=light, relief=1.0, ao=ao, spec=spec, spec_pow=24, periodic=False,
                 emissive=emissive, amb=0.6)


def grey(v):
    return np.repeat(np.clip(v, 0, 1)[..., None], 3, -1)


def side_shade(alb, emissive=None):
    """Sides: brighter rim along the top edge, darker toward the base."""
    g = 1.08 - 0.32 * V
    rim = smoothstep(0.08, 0.0, V) * 0.25
    col = alb * g[..., None] + rim[..., None]
    if emissive is not None:
        col = col + emissive
    return np.clip(col, 0, 1)


# ------------------------------------------------------------- world tops
def top_neon():
    frame = smoothstep(0.05, 0.035, E)
    inner = smoothstep(0.02, 0.0, np.abs(E - 0.085))
    glass = 0.72 + 0.18 * nz(4, 8, 3, 1) + 0.12 * smoothstep(0.25, 0.0, np.abs(U - V * 0.5 - 0.25))
    v = np.where(frame > 0.5, 0.18, glass) + inner * 0.3
    h = bevel(E, 0.05, 0.03)
    return lit(grey(v), h, spec=0.4)


def side_neon():
    frame = (np.minimum(U, 1 - U) < 0.04) | (V > 0.86) | (V < 0.08)
    v = np.where(frame, 0.2, 0.8 + 0.15 * nz(4, 8, 3, 2))
    return side_shade(grey(v))


def top_meadow():
    plank = np.floor(V * 3)
    pv = V * 3 - plank
    grain = nz(3, 40, 4, 10 + 0) * 0.5 + 0.5 * np.sin(U * 90 + nz(2, 6, 3, 11) * 12 + plank * 3)
    v = 0.78 + 0.12 * grain
    groove = smoothstep(0.06, 0.0, np.minimum(pv, 1 - pv))
    v = v * (1 - groove * 0.45)
    h = bevel(E, 0.05, 0.03) - groove * 0.01
    # iron corner brackets with nails
    br = (np.minimum(U, 1 - U) < 0.09)
    v = np.where(br, 0.32 + 0.08 * nz(8, 8, 2, 12), v)
    h = h + br * 0.006
    for cx in (0.045, 0.955):
        for cy in (0.18, 0.5, 0.82):
            d = np.hypot((U - cx) * CW, (V - cy) * CH)
            nail = smoothstep(5, 3, d)
            v = np.where(nail > 0.5, 0.75, v)
            h = h + nail * 0.004
    return lit(grey(v), h)


def side_meadow():
    plank = np.floor(V * 2)
    grain = 0.5 + 0.5 * np.sin(U * 70 + nz(2, 6, 3, 13) * 10 + plank * 2)
    v = 0.76 + 0.1 * grain
    v = np.where(np.minimum(U, 1 - U) < 0.09, 0.34, v)
    v = v * (1 - smoothstep(0.08, 0.0, np.abs(V - 0.5)) * 0.4)
    return side_shade(grey(v))


def top_caverns():
    # a cut gem: a table in the middle and facets falling to the edges
    du, dv = np.abs(U - 0.5) * 2, np.abs(V - 0.5) * 2
    table = (du < 0.55) & (dv < 0.45)
    fac = np.where(du * 0.9 > dv, np.sign(U - 0.5) * 1.0, np.sign(V - 0.5) * 0.5)
    v = np.where(table, 0.95, 0.72 + 0.18 * fac)
    edge = smoothstep(0.02, 0.0, np.abs(np.maximum(du / 0.55, dv / 0.45) - 1)) * 0.25
    spark = smoothstep(0.92, 0.97, nz(10, 20, 2, 20)) * 0.6
    v = np.clip(v + edge + spark, 0, 1.2)
    return np.clip(grey(v) * 1.05, 0, 1)


def side_caverns():
    v = 0.7 + 0.2 * np.abs(np.sin(U * 9 + V * 2)) + smoothstep(0.1, 0, V) * 0.2
    return side_shade(grey(v))


def top_ruins():
    stone = 0.8 + 0.12 * nz(6, 12, 5, 30) + 0.05 * nz(30, 60, 2, 31)
    h = bevel(E, 0.07, 0.04) + (nz(6, 12, 4, 32) - 0.5) * 0.006
    groove = smoothstep(0.012, 0.004, np.abs(E - 0.085))
    h = h - groove * 0.008
    d = np.hypot((U - 0.5) * 2, (V - 0.5))
    glyph = smoothstep(0.03, 0.012, np.abs(d - 0.22)) + smoothstep(0.03, 0.012, np.abs(U - 0.5) * 2) * (d < 0.22)
    h = h - np.clip(glyph, 0, 1) * 0.008
    v = stone * (1 - 0.35 * groove - 0.35 * np.clip(glyph, 0, 1))
    return lit(grey(v), h, spec=0.0)


def side_ruins():
    v = 0.78 + 0.12 * nz(4, 10, 5, 33)
    v = v * (1 - 0.3 * smoothstep(0.02, 0.0, np.abs(V - 0.55)))
    return side_shade(grey(v))


def top_clouds():
    du, dv = (U - 0.5) * 2, (V - 0.5) * 2
    pillow = 1 - np.clip(np.maximum(np.abs(du), np.abs(dv)) ** 4, 0, 1)
    h = pillow * 0.05
    pipe = smoothstep(0.02, 0.008, np.abs(E - 0.03))
    tuft = np.zeros_like(U)
    for cx in (0.3, 0.5, 0.7):
        tuft = np.maximum(tuft, smoothstep(5, 2.5, np.hypot((U - cx) * CW, (V - 0.5) * CH)))
    stitch = smoothstep(0.012, 0.0, np.abs(E - 0.08)) * (np.mod(U * 40 + V * 20, 1) < 0.6)
    v = 0.9 + 0.06 * nz(4, 8, 3, 40) + pipe * 0.1 - tuft * 0.25 - stitch * 0.15
    h = h - tuft * 0.01 + pipe * 0.004
    return lit(grey(v), h, spec=0.25)


def side_clouds():
    v = 0.92 - 0.1 * np.abs(V - 0.45) + 0.1 * smoothstep(0.03, 0.0, np.abs(V - 0.2))
    return side_shade(grey(v))


def top_machine():
    h = bevel(E, 0.04, 0.03)
    v = 0.74 + 0.06 * np.sin(U * 300 + nz(2, 8, 2, 50) * 3) + 0.05 * nz(8, 16, 3, 51)
    # vent slots in the middle
    slot = (np.abs(V - 0.5) < 0.2) & (np.abs(np.mod(U * 7, 1) - 0.5) < 0.18) & (U > 0.22) & (U < 0.78)
    v = np.where(slot, 0.2, v)
    h = h - slot * 0.008
    for cx in (0.06, 0.94):
        for cy in (0.14, 0.86):
            d = np.hypot((U - cx) * CW, (V - cy) * CH)
            r = smoothstep(6, 4, d)
            v = np.where(r > 0.5, 0.95, v)
            h = h + r * 0.006
    return lit(grey(v), h, spec=0.35)


def side_machine():
    v = 0.72 + 0.05 * np.sin(V * 60)
    v = np.where(np.minimum(U, 1 - U) < 0.05, 0.5, v)
    hz = (V > 0.62) & (V < 0.82) & (np.mod((U * 8 + V * 2), 1) < 0.5)
    v = np.where(hz, 0.35, v)
    return side_shade(grey(v))


def top_steel():
    h = bevel(E, 0.04, 0.03)
    s = 0.12
    gx, gy = U / s, V * 0.5 / s
    ci, cj = np.floor(gx), np.floor(gy)
    lx, ly = gx - ci - 0.5, gy - cj - 0.5
    alt = ((ci + cj) % 2) * 2 - 1
    loz = smoothstep(0.1, 0.04, np.abs(lx - alt * ly) / 1.4) * smoothstep(0.38, 0.3, np.abs(lx + alt * ly) / 1.4)
    h = h + loz * 0.006
    alb = np.stack([0.62 + 0 * U] * 3, -1) * np.array([0.95, 0.98, 1.05])
    return lit(alb * (0.9 + 0.15 * nz(6, 12, 3, 60))[..., None], h, spec=0.5)


def side_steel():
    alb = np.stack([0.58 + 0.05 * np.sin(V * 50)] * 3, -1) * np.array([0.95, 0.98, 1.05])
    return side_shade(alb)


def top_elite():
    gold = np.array([1.0, 0.8, 0.35])
    h = bevel(E, 0.06, 0.04)
    du, dv = (U - 0.5) * 2, (V - 0.5) * 2
    # a crown emblem
    crown = (np.abs(du) < 0.42) & (dv > -0.05) & (dv < 0.45)
    pts = np.abs(np.mod(du * 2.5 + 0.5, 1) - 0.5) * 2
    spikes = (np.abs(du) < 0.42) & (dv < -0.05) & (dv > -0.5 + pts * 0.45)
    em = (crown | spikes).astype(float)
    em = blur(em, 1.0, False)
    h = h + em * 0.012
    alb = lerp(np.array([0.35, 0.12, 0.45]), gold, smoothstep(0.3, 0.6, em))
    trim = smoothstep(0.03, 0.015, np.abs(E - 0.05))
    alb = lerp(alb, gold, trim)
    return lit(alb, h, spec=0.5, emissive=(em * 0.15)[..., None] * gold)


def side_elite():
    gold = np.array([1.0, 0.8, 0.35])
    alb = lerp(np.array([0.32, 0.1, 0.42]), gold, smoothstep(0.03, 0.0, np.abs(V - 0.3)) + smoothstep(0.03, 0.0, np.abs(V - 0.75)))
    return side_shade(alb)


# ------------------------------------------------------------- specials
def emblem_top(base_col, emblem, glow_col, glow=0.5, h_em=0.012):
    h = bevel(E, 0.06, 0.035)
    em = blur(emblem.astype(float), 1.0, False)
    h = h + em * h_em
    alb = base_col * (0.85 + 0.2 * nz(6, 12, 3, 70))[..., None]
    alb = lerp(alb, glow_col, em)
    trim = smoothstep(0.025, 0.01, np.abs(E - 0.045))
    alb = lerp(alb, alb * 0.6 + glow_col * 0.4, trim)
    return lit(alb, h, spec=0.3, emissive=(em * glow)[..., None] * glow_col)


def emblem_side(base_col, glow_col):
    band = smoothstep(0.06, 0.0, np.abs(V - 0.5))
    alb = lerp(base_col, glow_col, band * 0.9)
    return side_shade(alb, emissive=(band * 0.3)[..., None] * glow_col)


DU, DV = (U - 0.5) * 2, (V - 0.5) * 2 * 0.5   # emblem space: 1 unit tall = 1 unit wide
R = np.hypot(DU, DV * 2)


def emblems():
    out = {}
    # armored: crossed plates and bolts
    out['armored'] = (np.array([0.45, 0.48, 0.55]), ((np.abs(DU) < 0.06) | (np.abs(DV) < 0.03)) | (smoothstep(0.08, 0.05, np.hypot(np.abs(DU) - 0.7, np.abs(DV) - 0.15)) > 0.5), np.array([0.85, 0.9, 1.0]))
    # explosive: a bomb with a fuse
    bomb = np.hypot(DU * 1.0, DV * 2) < 0.28
    fuse = (np.abs(DU - 0.25 - (DV + 0.2) * 0.8) < 0.03) & (DV < -0.1) & (DV > -0.22)
    spark = np.hypot(DU - 0.36, DV * 2 + 0.3) < 0.07
    out['explosive'] = (np.array([0.6, 0.12, 0.08]), bomb | fuse | spark, np.array([1.0, 0.75, 0.2]))
    # chain: lightning zigzag
    zz = np.abs(DV * 2 - 0.4 * np.abs(np.mod(DU * 2.2 + 0.25, 1) - 0.5) * 4 + 0.4) < 0.12
    out['chain'] = (np.array([0.12, 0.2, 0.45]), zz & (np.abs(DU) < 0.8), np.array([0.6, 0.9, 1.0]))
    # split: a crack down the middle with arrows apart
    crack = np.abs(DU - 0.05 * np.sin(DV * 40)) < 0.04
    arrows = ((np.abs(np.abs(DU) - 0.5) + np.abs(DV * 2) * 0.6) < 0.12)
    out['split'] = (np.array([0.5, 0.35, 0.2]), crack | arrows, np.array([1.0, 0.85, 0.5]))
    # regen: a leaf
    leaf = (np.hypot(DU * 1.2, DV * 2 - 0.0) < 0.42) & (np.hypot(DU * 1.2 - 0.18, DV * 2 + 0.12) > 0.36)
    out['regen'] = (np.array([0.15, 0.4, 0.18]), leaf | ((np.abs(DU + DV * 1.4) < 0.025) & (R < 0.4)), np.array([0.6, 1.0, 0.5]))
    # moving: double arrows left-right
    mv = ((np.abs(DV) < 0.03) & (np.abs(DU) < 0.6)) | ((np.abs(np.abs(DU) - 0.55) * 1.0 + np.abs(DV) * 2 < 0.15) & (np.abs(DU) > 0.45))
    out['moving'] = (np.array([0.35, 0.3, 0.55]), mv, np.array([0.85, 0.8, 1.0]))
    # shield: a heater shield outline
    sh_y = DV * 2
    sh = (np.abs(DU) < 0.32 - np.clip(sh_y, 0, 1) * 0.5) & (sh_y > -0.35) & (sh_y < 0.45)
    sh_in = (np.abs(DU) < 0.24 - np.clip(sh_y, 0, 1) * 0.5) & (sh_y > -0.27) & (sh_y < 0.36)
    out['shield'] = (np.array([0.2, 0.3, 0.6]), sh & ~sh_in | ((np.abs(DU) < 0.03) & sh), np.array([0.7, 0.85, 1.0]))
    # teleport: a swirl
    ang = np.arctan2(DV * 2, DU)
    sw = (np.abs(np.mod(R * 9 - ang / np.pi, 2) - 1) < 0.35) & (R < 0.45)
    out['teleport'] = (np.array([0.35, 0.12, 0.5]), sw, np.array([0.9, 0.5, 1.0]))
    # spawner: three seeds
    sp = np.zeros_like(U, bool)
    for cx, cy in ((-0.25, 0.08), (0.25, 0.08), (0.0, -0.12)):
        sp |= np.hypot((DU - cx), (DV - cy) * 2) < 0.14
    out['spawner'] = (np.array([0.4, 0.3, 0.15]), sp, np.array([0.8, 1.0, 0.4]))
    # multiplier: x2
    x2 = ((np.abs(np.abs(DU + 0.22) - np.abs(DV * 2)) < 0.05) & (np.abs(DV * 2) < 0.3) & (np.abs(DU + 0.22) < 0.3))
    two_y = DV * 2
    two = (((np.abs(np.hypot(DU - 0.3, two_y + 0.1) - 0.14) < 0.04) & (two_y < -0.02)) |
           ((np.abs(two_y - 0.26) < 0.04) & (np.abs(DU - 0.3) < 0.16)) |
           ((np.abs((DU - 0.3) + (two_y - 0.26) * 0.8 + 0.0) < 0.05) & (two_y > -0.05) & (two_y < 0.26)))
    out['multiplier'] = (np.array([0.5, 0.35, 0.05]), x2 | two, np.array([1.0, 0.85, 0.3]))
    # crystal: a gem
    gy = DV * 2
    crown = (gy > -0.28) & (gy < -0.06) & (np.abs(DU) < 0.2 + (gy + 0.28) * 0.7)
    pav = (gy >= -0.06) & (gy < 0.34) & (np.abs(DU) < (0.34 - gy) * 0.9)
    facet = (np.abs(np.abs(DU) - 0.12 - (gy + 0.06) * -0.2) < 0.02) & (gy > -0.06)
    gem = (crown | pav) & ~facet
    out['crystal'] = (np.array([0.1, 0.35, 0.45]), gem, np.array([0.6, 1.0, 1.0]))
    # mystery: a question mark
    qy = DV * 2
    q = ((np.abs(np.hypot(DU, qy + 0.12) - 0.16) < 0.045) & ((qy < -0.12) | (DU > 0))) | ((np.abs(DU) < 0.045) & (qy > 0.0) & (qy < 0.16)) | (np.hypot(DU, qy - 0.3) < 0.06)
    out['mystery'] = (np.array([0.45, 0.2, 0.5]), q, np.array([1.0, 0.85, 1.0]))
    return out


def main():
    sheet = np.zeros((CH * 6, CW * 8, 3))

    def put(i, j, img):
        sheet[j * CH:(j + 1) * CH, i * CW:(i + 1) * CW] = img

    tops = [top_neon, top_meadow, top_caverns, top_ruins, top_clouds, top_machine, top_steel, top_elite]
    sides = [side_neon, side_meadow, side_caverns, side_ruins, side_clouds, side_machine, side_steel, side_elite]
    for i, (t, s) in enumerate(zip(tops, sides)):
        put(i, 0, t())
        put(i, 1, s())
    em = emblems()
    order = ['armored', 'explosive', 'chain', 'split', 'regen', 'moving', 'shield', 'teleport', 'spawner', 'multiplier', 'crystal', 'mystery']
    for k, name in enumerate(order):
        base, mask, glow = em[name]
        put(k % 8, 2 + k // 8, emblem_top(base, mask, glow))
        put(k % 8, 4 + k // 8, emblem_side(base, glow))
    to_img(sheet).save(OUT, quality=92, subsampling=0)
    print('wrote', OUT, sheet.shape)


if __name__ == '__main__':
    main()
