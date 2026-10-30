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
PX = CW / 2.0          # px per world unit across a 2.0-wide brick
E = np.minimum(np.minimum(U, 1 - U) * 2.0, np.minimum(V, 1 - V) * 1.1)  # world units to the edge


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
# Clean, bright "candy" blocks: a bold bevel that catches light on the top
# and left edges, an inset panel, a gloss streak, and a faint world motif.

def candy_top(motif=None, panel=True, gloss=0.12):
    ex, ey = np.minimum(U, 1 - U) * 2.0, np.minimum(V, 1 - V) * 1.1
    e = np.minimum(ex, ey)
    bw = 0.13
    t = np.clip(e / bw, 0, 1)
    # which edge is nearest decides light (top/left) or shade (bottom/right)
    lightside = np.where(ex < ey, U < 0.5, V < 0.5)
    bev = np.where(lightside, 0.3, -0.38) * (1 - t) ** 1.5
    v = 0.93 + bev
    if panel:
        pe = e - 0.2
        groove = smoothstep(0.03, 0.0, np.abs(pe)) * 0.16
        inner = (pe > 0) * 0.03
        v = v - groove + inner
    if motif is not None:
        v = v - motif * 0.14
    streak = smoothstep(0.18, 0.0, np.abs((U * 2.0 - V * 1.1) - 0.45)) * (e > bw) * gloss
    v = v + streak + 0.02 * (nz(6, 12, 3, 5) - 0.5)
    return np.clip(grey(v), 0, 1)


def candy_side(lines=None):
    v = 1.0 - 0.3 * V
    v = v + smoothstep(0.1, 0.0, V) * 0.22 - smoothstep(0.9, 1.0, V) * 0.25
    ex = np.minimum(U, 1 - U) * 2.0
    v = v - smoothstep(0.06, 0.0, ex) * 0.2
    if lines is not None:
        v = v - lines * 0.15
    return np.clip(grey(v), 0, 1)


def _panel_coords():
    return (U - 0.5) * 2.0, (V - 0.5) * 1.1


def top_neon():
    ex, ey = np.minimum(U, 1 - U) * 2.0, np.minimum(V, 1 - V) * 1.1
    e = np.minimum(ex, ey)
    rim = smoothstep(0.1, 0.06, e)
    glass = 0.86 + 0.1 * smoothstep(0.3, 0.0, np.abs(U * 2 - V * 1.1 - 0.5))
    v = np.where(rim > 0.5, 0.35, glass) + smoothstep(0.02, 0.0, np.abs(e - 0.14)) * 0.15
    return np.clip(grey(v), 0, 1)


def side_neon():
    ex = np.minimum(U, 1 - U) * 2.0
    rim = (ex < 0.08) | (V > 0.85) | (V < 0.1)
    return grey(np.where(rim, 0.35, 0.92 - 0.12 * V))


def top_meadow():
    px, py = _panel_coords()
    # a little flower stamp in the panel
    ang = np.arctan2(py, px)
    r = np.hypot(px, py)
    petals = smoothstep(0.02, 0.0, np.abs(r - (0.16 + 0.06 * np.cos(ang * 5)))) + smoothstep(0.05, 0.03, r)
    return candy_top(np.clip(petals, 0, 1))


def side_meadow():
    return candy_side()


def top_caverns():
    du, dv = np.abs(U - 0.5) * 2, np.abs(V - 0.5) * 2
    table = (du < 0.6) & (dv < 0.5)
    fac = np.where(du * 0.9 > dv, np.where(U < 0.5, 0.18, -0.12), np.where(V < 0.5, 0.12, -0.2))
    v = np.where(table, 0.97, 0.82 + fac)
    edge = smoothstep(0.02, 0.0, np.abs(np.maximum(du / 0.6, dv / 0.5) - 1)) * 0.2
    return np.clip(grey(v + edge), 0, 1)


def side_caverns():
    v = 0.85 + 0.12 * np.sign(np.sin(U * 6.283 * 2)) * 0.5 - 0.25 * V + smoothstep(0.1, 0, V) * 0.2
    return np.clip(grey(v), 0, 1)


def top_ruins():
    px, py = _panel_coords()
    d = np.hypot(px, py * 1.4)
    glyph = smoothstep(0.025, 0.01, np.abs(d - 0.2)) + smoothstep(0.025, 0.01, np.abs(px)) * (d < 0.2)
    return candy_top(np.clip(glyph, 0, 1), gloss=0.06)


def side_ruins():
    return candy_side(smoothstep(0.02, 0.0, np.abs(V - 0.5)))


def top_clouds():
    px, py = _panel_coords()
    tufts = np.zeros_like(U)
    for cx in (-0.35, 0.0, 0.35):
        tufts = np.maximum(tufts, smoothstep(0.05, 0.03, np.hypot(px - cx, py)))
    return candy_top(tufts, gloss=0.16)


def side_clouds():
    return candy_side(smoothstep(0.02, 0.0, np.abs(V - 0.3)) * 0.6)


def top_machine():
    px, py = _panel_coords()
    slots = (np.abs(py) < 0.12) & (np.abs(np.mod(px * 4, 1) - 0.5) < 0.18) & (np.abs(px) < 0.55)
    rivets = np.zeros_like(U)
    for cx in (-0.85, 0.85):
        for cy in (-0.4, 0.4):
            rivets = np.maximum(rivets, smoothstep(0.045, 0.03, np.hypot(px - cx, py - cy)))
    return np.clip(candy_top(slots.astype(float) * 1.5, gloss=0.1) + rivets[..., None] * 0.1, 0, 1)


def side_machine():
    hz = (V > 0.6) & (V < 0.8) & (np.mod((U * 8 + V * 2), 1) < 0.5)
    return candy_side(hz.astype(float) * 1.6)


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
def emblem_top(base_col, emblem, glow_col, glow=0.35, h_em=0.012):
    """A vivid candy block with a bold emblem: dark outline, bright fill."""
    shade = candy_top(panel=False, gloss=0.14)[..., 0]
    m = emblem.astype(bool)
    outline = ndimage.binary_dilation(m, iterations=5) & ~m
    shadow = np.roll(np.roll(ndimage.binary_dilation(m, iterations=5), 4, 0), 4, 1) & ~m & ~outline
    alb = base_col * shade[..., None]
    alb = lerp(alb, alb * 0.55, shadow.astype(float))
    alb = np.where(outline[..., None], base_col * 0.25, alb)
    fill = lerp(glow_col, np.array([1.0, 1.0, 1.0]), 0.35)
    fill_sh = fill * (0.92 + 0.12 * (1 - V))[..., None]
    alb = np.where(m[..., None], fill_sh, alb)
    alb = blur(alb, 0.6, False) if False else alb
    return np.clip(alb + (m * glow)[..., None] * glow_col * 0.25, 0, 1)


def emblem_side(base_col, glow_col):
    shade = candy_side()[..., 0]
    band = smoothstep(0.05, 0.0, np.abs(V - 0.5))
    alb = base_col * shade[..., None]
    alb = lerp(alb, lerp(glow_col, np.array([1.0, 1, 1]), 0.3), band * 0.9)
    return np.clip(alb, 0, 1)


DU, DV = (U - 0.5) * 2, (V - 0.5) * 2 * 0.5   # emblem space: 1 unit tall = 1 unit wide
R = np.hypot(DU, DV * 2)


def emblems():
    out = {}
    # armored: crossed plates and bolts
    out['armored'] = (np.array([0.55, 0.6, 0.72]), ((np.abs(DU) < 0.06) | (np.abs(DV) < 0.03)) | (smoothstep(0.08, 0.05, np.hypot(np.abs(DU) - 0.7, np.abs(DV) - 0.15)) > 0.5), np.array([0.85, 0.9, 1.0]))
    # explosive: a bomb with a fuse
    bomb = np.hypot(DU * 1.0, DV * 2) < 0.28
    fuse = (np.abs(DU - 0.25 - (DV + 0.2) * 0.8) < 0.03) & (DV < -0.1) & (DV > -0.22)
    spark = np.hypot(DU - 0.36, DV * 2 + 0.3) < 0.07
    out['explosive'] = (np.array([0.92, 0.16, 0.1]), bomb | fuse | spark, np.array([1.0, 0.75, 0.2]))
    # chain: lightning zigzag
    zz = np.abs(DV * 2 - 0.4 * np.abs(np.mod(DU * 2.2 + 0.25, 1) - 0.5) * 4 + 0.4) < 0.12
    out['chain'] = (np.array([0.15, 0.36, 0.95]), zz & (np.abs(DU) < 0.8), np.array([0.6, 0.9, 1.0]))
    # split: a crack down the middle with arrows apart
    crack = np.abs(DU - 0.05 * np.sin(DV * 40)) < 0.04
    arrows = ((np.abs(np.abs(DU) - 0.5) + np.abs(DV * 2) * 0.6) < 0.12)
    out['split'] = (np.array([0.9, 0.52, 0.12]), crack | arrows, np.array([1.0, 0.85, 0.5]))
    # regen: a leaf
    leaf = (np.hypot(DU * 1.2, DV * 2 - 0.0) < 0.42) & (np.hypot(DU * 1.2 - 0.18, DV * 2 + 0.12) > 0.36)
    out['regen'] = (np.array([0.15, 0.76, 0.25]), leaf | ((np.abs(DU + DV * 1.4) < 0.025) & (R < 0.4)), np.array([0.6, 1.0, 0.5]))
    # moving: double arrows left-right
    mv = ((np.abs(DV) < 0.03) & (np.abs(DU) < 0.6)) | ((np.abs(np.abs(DU) - 0.55) * 1.0 + np.abs(DV) * 2 < 0.15) & (np.abs(DU) > 0.45))
    out['moving'] = (np.array([0.56, 0.36, 0.92]), mv, np.array([0.85, 0.8, 1.0]))
    # shield: a heater shield outline
    sh_y = DV * 2
    sh = (np.abs(DU) < 0.32 - np.clip(sh_y, 0, 1) * 0.5) & (sh_y > -0.35) & (sh_y < 0.45)
    sh_in = (np.abs(DU) < 0.24 - np.clip(sh_y, 0, 1) * 0.5) & (sh_y > -0.27) & (sh_y < 0.36)
    out['shield'] = (np.array([0.2, 0.46, 0.96]), sh & ~sh_in | ((np.abs(DU) < 0.03) & sh), np.array([0.7, 0.85, 1.0]))
    # teleport: a swirl
    ang = np.arctan2(DV * 2, DU)
    sw = (np.abs(np.mod(R * 9 - ang / np.pi, 2) - 1) < 0.35) & (R < 0.45)
    out['teleport'] = (np.array([0.72, 0.15, 0.86]), sw, np.array([0.9, 0.5, 1.0]))
    # spawner: three seeds
    sp = np.zeros_like(U, bool)
    for cx, cy in ((-0.25, 0.08), (0.25, 0.08), (0.0, -0.12)):
        sp |= np.hypot((DU - cx), (DV - cy) * 2) < 0.14
    out['spawner'] = (np.array([0.55, 0.76, 0.1]), sp, np.array([0.8, 1.0, 0.4]))
    # multiplier: x2
    x2 = ((np.abs(np.abs(DU + 0.22) - np.abs(DV * 2)) < 0.05) & (np.abs(DV * 2) < 0.3) & (np.abs(DU + 0.22) < 0.3))
    two_y = DV * 2
    two = (((np.abs(np.hypot(DU - 0.3, two_y + 0.1) - 0.14) < 0.04) & (two_y < -0.02)) |
           ((np.abs(two_y - 0.26) < 0.04) & (np.abs(DU - 0.3) < 0.16)) |
           ((np.abs((DU - 0.3) + (two_y - 0.26) * 0.8 + 0.0) < 0.05) & (two_y > -0.05) & (two_y < 0.26)))
    out['multiplier'] = (np.array([1.0, 0.7, 0.05]), x2 | two, np.array([1.0, 0.85, 0.3]))
    # crystal: a gem
    gy = DV * 2
    crown = (gy > -0.28) & (gy < -0.06) & (np.abs(DU) < 0.2 + (gy + 0.28) * 0.7)
    pav = (gy >= -0.06) & (gy < 0.34) & (np.abs(DU) < (0.34 - gy) * 0.9)
    facet = (np.abs(np.abs(DU) - 0.12 - (gy + 0.06) * -0.2) < 0.02) & (gy > -0.06)
    gem = (crown | pav) & ~facet
    out['crystal'] = (np.array([0.08, 0.74, 0.9]), gem, np.array([0.6, 1.0, 1.0]))
    # mystery: a question mark
    qy = DV * 2
    q = ((np.abs(np.hypot(DU, qy + 0.12) - 0.16) < 0.045) & ((qy < -0.12) | (DU > 0))) | ((np.abs(DU) < 0.045) & (qy > 0.0) & (qy < 0.16)) | (np.hypot(DU, qy - 0.3) < 0.06)
    out['mystery'] = (np.array([0.62, 0.25, 0.78]), q, np.array([1.0, 0.85, 1.0]))
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
