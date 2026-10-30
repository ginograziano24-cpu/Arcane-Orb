"""Bakes the arena floor textures: python3 tools/textures/floors.py [world ...]

Each floor covers x -14..14 and z -3..33 world units (2048 x 2633 px).
The 2-unit tile grid lines up with the game's floor quads."""
import sys, os
import numpy as np
from texlib import *

W, Hh = 2048, 2633
S = W / 28.0  # px per world unit
OUT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'floor')

yy, xx = np.mgrid[0:Hh, 0:W].astype(np.float32)
WX = xx / S - 14.0          # world x
WZ = 33.0 - yy / S          # world z
TU = (WX + 14.0) / 2.0      # tile coords
TV = (WZ + 3.0) / 2.0
TI = np.floor(TU).astype(int)
TJ = np.floor(TV).astype(int)
FX = TU - TI                # 0..1 inside the tile
FY = TV - TJ
EDGE = np.minimum(np.minimum(FX, 1 - FX), np.minimum(FY, 1 - FY)) * 2.0  # world units to the tile edge
CHECK = (TI + TJ) % 2
TILE_ID = TI * 97 + TJ * 31


def tile_rand(seed, k=1):
    """A per-tile random value (or k of them) in 0..1."""
    r = RNG(seed).random((64, 64, k))
    v = r[TI % 64, TJ % 64]
    return v[..., 0] if k == 1 else v


def noise(cy, cx, oct=5, seed=0, gain=0.5):
    return fbm(Hh, W, cy, cx, oct, seed, gain, periodic=False)


def bevel_height(edge, grout, bev, depth=0.05):
    """Tile height (world units): 0 in the grout, rounding up to the top."""
    t = np.clip((edge - grout) / bev, 0, 1)
    h = np.where(edge < grout, 0.0, np.sqrt(1 - (1 - t) ** 2))
    return h * depth


def save(name, rgb):
    os.makedirs(OUT, exist_ok=True)
    img = to_img(rgb)
    if name.startswith('neon'):
        img = img.convert('L')
    img.save(os.path.join(OUT, name + '.jpg'), quality=86, subsampling=0)
    print('wrote', name, img.size)


# ------------------------------------------------------------------ meadow
def meadow():
    """Sun-baked terracotta tiles with mossy grout, chips, cracks and leaves."""
    grout_w, bev = 0.05, 0.12
    chip = noise(36, 28, 4, 11)
    edge = EDGE + (chip - 0.5) * 0.06 * smoothstep(0.25, 0.0, EDGE)
    h = bevel_height(edge, grout_w, bev, 0.06)
    surf = noise(144, 112, 5, 12)
    h += (surf - 0.5) * 0.012 * (h > 0)
    pits = noise(400, 320, 2, 13)
    h -= smoothstep(0.82, 0.88, pits) * 0.003
    tilt = tile_rand(24, 2) - 0.5
    h += ((FX - 0.5) * tilt[..., 0] + (FY - 0.5) * tilt[..., 1]) * 0.008 * (h > 0)
    # hairline cracks on a few tiles
    cr = ridged(Hh, W, 14, 11, 5, 14, periodic=False)
    crack_tile = tile_rand(15) < 0.16
    crack = smoothstep(0.988, 0.997, cr) * crack_tile * (edge > grout_w)
    h -= crack * 0.03
    tr = tile_rand(16, 3)
    warm, pale = hex2('c4704a'), hex2('d8a274')
    base = np.where(CHECK[..., None] == 1, warm, pale)
    base = base * (0.9 + 0.16 * tr[..., 0:1]) + (tr[..., 1:2] - 0.5) * np.array([0.05, 0.02, -0.02])
    stain = noise(24, 18, 5, 17)
    alb = base * (0.86 + 0.24 * stain[..., None])
    alb = lerp(alb, alb * 0.8, smoothstep(0.35, 0.05, edge) * 0.5)        # dirt toward the edges
    alb = lerp(alb, hex2('e9cfa4'), smoothstep(0.6, 0.85, noise(60, 48, 4, 18)) * 0.25)  # sun bleach
    # grout: earthy with moss and grass tufts
    moss_n = noise(30, 24, 5, 19)
    grout_col = lerp(hex2('6d5a44'), hex2('4f7a2e'), smoothstep(0.45, 0.65, moss_n))
    in_grout = smoothstep(grout_w + 0.015, grout_w - 0.01, edge)
    alb = lerp(alb, grout_col, in_grout)
    # moss creeping onto tile edges
    creep = smoothstep(0.55, 0.75, moss_n) * smoothstep(0.22, 0.04, edge)
    alb = lerp(alb, hex2('5d8a33') * (0.8 + 0.4 * noise(300, 240, 2, 20))[..., None], creep * 0.85)
    h += creep * 0.004 * noise(300, 240, 2, 21)
    # tiny grass blades in the grout
    blades = smoothstep(0.62, 0.75, noise(600, 480, 2, 22)) * in_grout * smoothstep(0.5, 0.6, moss_n)
    alb = lerp(alb, hex2('7fb04a'), blades * 0.8)
    alb = lerp(alb, alb * 0.45, crack[..., None] * 0.9)
    grit = noise(900, 700, 1, 25)
    alb = alb * (0.94 + 0.12 * grit[..., None])
    # fallen leaves
    r = RNG(23)
    leaves = np.zeros((Hh, W))
    leaf_col = np.zeros((Hh, W, 3))
    for _ in range(70):
        cx, cy = r.random() * W, r.random() * Hh
        a = r.random() * np.pi
        L, Wd = (0.16 + r.random() * 0.12) * S, (0.07 + r.random() * 0.04) * S
        x0, x1 = int(max(0, cx - L)), int(min(W, cx + L))
        y0, y1 = int(max(0, cy - L)), int(min(Hh, cy + L))
        if x1 <= x0 or y1 <= y0:
            continue
        sx, sy = xx[y0:y1, x0:x1] - cx, yy[y0:y1, x0:x1] - cy
        u = sx * np.cos(a) + sy * np.sin(a)
        v = -sx * np.sin(a) + sy * np.cos(a)
        m = smoothstep(1.0, 0.85, (u / L) ** 2 + (v / (Wd * (1 - (u / L) ** 2 + 0.05))) ** 2)
        col = [hex2('d99a2b'), hex2('b8532a'), hex2('8a9a3a'), hex2('c8b04a')][r.integers(4)]
        leaves[y0:y1, x0:x1] = np.maximum(leaves[y0:y1, x0:x1], m)
        leaf_col[y0:y1, x0:x1] = np.where(m[..., None] > 0.01, col, leaf_col[y0:y1, x0:x1])
    alb = lerp(alb, leaf_col, leaves * 0.95)
    h += leaves * 0.01
    rgb = shade(alb, h, S, light=(-0.7, 0.55, 0.2), relief=1.0, ao=1.2, spec=0.05, periodic=False)
    save('meadow', rgb)


# ------------------------------------------------------------------ ruins
def glyph_mask(fx, fy, kind):
    """Engraved symbols inside a tile (fx, fy 0..1); returns groove depth 0..1."""
    dx, dy = fx - 0.5, fy - 0.5
    d = np.hypot(dx, dy)
    line = lambda dist, w: smoothstep(w, w * 0.4, dist)
    if kind == 0:   # sun disk with rays
        g = line(np.abs(d - 0.16), 0.022) + line(d, 0.05)
        ang = np.arctan2(dy, dx)
        rays = (np.abs(np.sin(ang * 6)) < 0.18) & (d > 0.22) & (d < 0.33)
        return np.clip(g + rays * 1.0, 0, 1)
    if kind == 1:   # eye
        lens = np.abs(dy) + (dx ** 2) * 1.6
        g = line(np.abs(lens - 0.16), 0.02) + line(d, 0.055)
        return np.clip(g, 0, 1)
    if kind == 2:   # spiral
        ang = np.arctan2(dy, dx)
        r = (ang + np.pi) / (2 * np.pi) * 0.07
        sp = np.abs(np.mod(d - r, 0.07) - 0.035)
        return line(sp, 0.016) * (d < 0.3)
    # stepped pyramid / ziggurat
    step = np.floor((fy - 0.2) / 0.12)
    half = 0.3 - step * 0.06
    inside = (fy > 0.2) & (fy < 0.8) & (np.abs(dx) < half)
    edge = np.minimum(half - np.abs(dx), np.minimum(fy - 0.2 - step * 0.12, 0.12))
    return line(np.where(inside, edge, 1), 0.02) * inside


def ruins():
    """Sandstone flagstones with carved borders, glyph tiles, turquoise inlay and sand."""
    grout_w, bev = 0.035, 0.08
    chip = noise(40, 32, 4, 31)
    edge = EDGE + (chip - 0.5) * 0.08 * smoothstep(0.3, 0.0, EDGE)
    h = bevel_height(edge, grout_w, bev, 0.05)
    # strata and pitting
    strata = noise(260, 30, 4, 32)
    h += (strata - 0.5) * 0.006 * (h > 0)
    h += (noise(180, 140, 5, 33) - 0.5) * 0.01 * (h > 0)
    # carved inner border
    inner = np.abs(EDGE - 0.26)
    groove = smoothstep(0.035, 0.015, inner)
    h -= groove * 0.012
    # engraved glyph tiles
    tr = tile_rand(34, 3)
    gl = np.zeros_like(h)
    has = tr[..., 0] < 0.16
    for k in range(4):
        sel = has & (np.floor(tr[..., 1] * 4) == k)
        if sel.any():
            gl = np.where(sel, glyph_mask(FX, FY, k), gl)
    h -= gl * 0.014
    # turquoise faience diamonds where four tiles meet
    cdx = np.minimum(FX, 1 - FX) * 2
    cdy = np.minimum(FY, 1 - FY) * 2
    diamond = smoothstep(0.2, 0.17, cdx + cdy)
    cracks = smoothstep(0.986, 0.996, ridged(Hh, W, 12, 10, 5, 35, periodic=False)) * (tr[..., 2] < 0.25)
    h -= cracks * 0.02
    base = lerp(hex2('d9b27c'), hex2('c99d66'), CHECK.astype(float))
    alb = base * (0.9 + 0.14 * tile_rand(36)[..., None])
    alb = alb * (0.85 + 0.25 * noise(20, 16, 5, 37)[..., None])
    alb = lerp(alb, hex2('b98952'), smoothstep(0.55, 0.75, strata) * 0.4)
    alb = alb * (0.93 + 0.12 * noise(800, 640, 1, 38)[..., None])
    # sand settles in every low spot
    low = smoothstep(0.012, -0.002, h - blur(h, S * 0.05, False))
    sand = hex2('ead1a0') * (0.92 + 0.1 * noise(500, 400, 2, 39)[..., None])
    alb = lerp(alb, sand, np.clip(low * 0.8 + smoothstep(grout_w + 0.01, grout_w - 0.01, edge), 0, 1) * 0.85)
    drift = smoothstep(0.6, 0.8, noise(16, 12, 4, 40))
    alb = lerp(alb, sand, drift * 0.35)
    alb = lerp(alb, alb * 0.55, gl[..., None] * 0.7)
    alb = lerp(alb, alb * 0.5, cracks[..., None])
    # inlay on top
    turq = hex2('2fb7a8') * (0.85 + 0.25 * noise(300, 240, 3, 41)[..., None])
    alb = lerp(alb, turq, diamond)
    h = np.where(diamond > 0.5, 0.045, h)
    gold_ring = smoothstep(0.03, 0.01, np.abs(cdx + cdy - 0.2)) * (cdx + cdy < 0.25)
    alb = lerp(alb, hex2('e2b34a'), gold_ring)
    rgb = shade(alb, h, S, light=(0.55, 0.75, -0.35), ao=1.1, spec=0.0, periodic=False)
    rgb += (diamond * 0.08)[..., None] * np.array([0.6, 1, 1])
    save('ruins', np.clip(rgb, 0, 1))


# ------------------------------------------------------------------ caverns
def caverns():
    """Dark slate slabs split by fractures that glow with crystal light."""
    grout_w, bev = 0.06, 0.1
    edge = EDGE + (noise(30, 24, 4, 51) - 0.5) * 0.1 * smoothstep(0.35, 0, EDGE)
    h = bevel_height(edge, grout_w, bev, 0.07)
    f1, f2, cid = voronoi((Hh + 1) // 2, W // 2, 900, 52, periodic=False)
    f2 = ndimage.zoom(f2, 2, order=1)[:Hh, :W] * 2 / S
    cid = ndimage.zoom(cid, 2, order=0)[:Hh, :W]
    frac = smoothstep(0.03, 0.008, f2) * (EDGE > grout_w + 0.05)
    cr = RNG(53).random(cid.max() + 1)
    h += (cr[cid] - 0.5) * 0.012 * (h > 0)
    h += (noise(120, 96, 6, 54) - 0.5) * 0.02 * (h > 0)
    lay = ridged(Hh, W, 70, 56, 4, 55, periodic=False)
    h += lay * 0.006 * (h > 0)
    h -= frac * 0.025
    base = lerp(hex2('5c6c94'), hex2('4a5880'), CHECK.astype(float))
    alb = base * (0.8 + 0.4 * cr[cid][..., None] * 0.5 + 0.3 * noise(26, 20, 5, 56)[..., None])
    wet = smoothstep(0.55, 0.75, noise(14, 11, 4, 57))
    alb = lerp(alb, alb * 0.65, wet * 0.6)
    alb = alb * (0.9 + 0.2 * noise(700, 560, 1, 58)[..., None])
    # crystal light in fractures and grout
    hue = noise(8, 6, 3, 59)
    glow_col = lerp(hex2('39e6ff'), hex2('b46bff'), smoothstep(0.35, 0.65, hue))
    in_grout = smoothstep(grout_w + 0.02, grout_w - 0.01, edge)
    vein = np.clip(frac * smoothstep(0.35, 0.6, noise(24, 20, 3, 60)) + in_grout * 0.55, 0, 1)
    glow = blur(vein, S * 0.05, False) * 1.6 + vein
    alb = lerp(alb, hex2('141a2a'), in_grout * 0.7)
    # pebbles
    peb = smoothstep(0.83, 0.86, noise(500, 400, 2, 61)) * (EDGE > 0.15)
    h += peb * 0.01
    alb = lerp(alb, hex2('5b6684'), peb * 0.6)
    rgb = shade(alb, h, S, light=(0.3, 0.8, -0.5), ao=1.3, spec=0.18 * (0.3 + wet), spec_pow=40,
                periodic=False, emissive=glow[..., None] * glow_col * 0.7)
    save('caverns', rgb)


# ------------------------------------------------------------------ clouds
def clouds():
    """Polished marble with veins, gold inlay strips and rosettes."""
    grout_w, bev = 0.04, 0.05
    h = bevel_height(EDGE, grout_w, bev, 0.03)
    warp = noise(10, 8, 4, 71)
    v1 = np.abs(np.sin((WX * 0.9 + WZ * 0.6) * 1.3 + warp * 9 + noise(40, 32, 4, 72) * 3))
    vein = smoothstep(0.05, 0.0, v1) * 0.9 + smoothstep(0.18, 0.02, v1) * 0.25
    v2 = np.abs(np.sin((WX * -0.5 + WZ) * 2.1 + noise(30, 24, 5, 73) * 7))
    vein2 = smoothstep(0.03, 0.0, v2) * 0.6
    white, blue = hex2('f2f1f4'), hex2('d6e1f4')
    base = np.where(CHECK[..., None] == 1, blue, white)
    alb = base * (0.95 + 0.08 * noise(30, 24, 5, 74)[..., None])
    vc = np.where(CHECK[..., None] == 1, hex2('8c9cc0'), hex2('a7a6b4'))
    alb = lerp(alb, vc, np.clip(vein + vein2, 0, 1) * 0.6)
    # gold strips in the joints
    gold = smoothstep(grout_w + 0.012, grout_w - 0.004, EDGE)
    gold_col = hex2('d8a83c') * (0.85 + 0.3 * noise(400, 300, 3, 75)[..., None])
    alb = lerp(alb, gold_col, gold)
    h = np.where(gold > 0.5, 0.034 - 0.01 * smoothstep(0, grout_w, EDGE), h)
    # rosettes where four tiles meet
    cdx = np.minimum(FX, 1 - FX) * 2
    cdy = np.minimum(FY, 1 - FY) * 2
    d = np.hypot(cdx, cdy)
    ang = np.arctan2(cdy, cdx)
    petal = 0.17 + 0.04 * np.cos(ang * 8)
    ros = smoothstep(petal + 0.01, petal - 0.01, d)
    alb = lerp(alb, gold_col, ros)
    alb = lerp(alb, hex2('6fa8ff'), smoothstep(0.06, 0.045, d))
    h = np.where(ros > 0.5, 0.04 - d * 0.05, h)
    # central medallion ring on alternating tiles
    md = np.hypot(FX - 0.5, FY - 0.5) * 2
    ring = smoothstep(0.02, 0.008, np.abs(md - 0.62)) * (CHECK == 0) * (tile_rand(76) < 0.5)
    alb = lerp(alb, hex2('c9b27a'), ring * 0.8)
    rgb = shade(alb, h, S, light=(-0.2, 0.9, -0.3), ao=0.8, spec=0.35, spec_pow=60, periodic=False, amb=0.62)
    save('clouds', rgb)


# ------------------------------------------------------------------ machine
def machine():
    """Riveted steel deck with tread plate, seams, grime and a hazard band."""
    grout_w, bev = 0.035, 0.07
    h = bevel_height(EDGE, grout_w, bev, 0.05)
    # diamond tread: raised lozenges in alternating directions
    s = 0.3
    gx, gz = WX / s, WZ / s
    ci, cj = np.floor(gx), np.floor(gz)
    lx, lz = gx - ci - 0.5, gz - cj - 0.5
    alt = ((ci + cj) % 2) * 2 - 1
    u = (lx + alt * lz) / 1.414
    v = (lx - alt * lz) / 1.414
    lozenge = smoothstep(0.08, 0.03, np.abs(v)) * smoothstep(0.36, 0.3, np.abs(u))
    tread_tile = (CHECK == 0)
    h += lozenge * 0.014 * tread_tile * (EDGE > 0.12)
    # brushed plates on the other tiles
    brushed = noise(2000, 14, 2, 81)
    h += (brushed - 0.5) * 0.002 * (~tread_tile)
    # rivets along the plate edges
    rp = 0.25
    rx = np.mod(TU * 2.0 / rp, 1) - 0.5
    rz = np.mod(TV * 2.0 / rp, 1) - 0.5
    near_edge = (EDGE > 0.07) & (EDGE < 0.15)
    rv = np.where(np.minimum(FX, 1 - FX) * 2 < np.minimum(FY, 1 - FY) * 2, np.hypot((np.minimum(FX, 1 - FX) * 2 - 0.11) / 0.04, rz / 0.16),
                  np.hypot((np.minimum(FY, 1 - FY) * 2 - 0.11) / 0.04, rx / 0.16))
    rivet = smoothstep(1.0, 0.6, rv) * near_edge
    h += rivet * 0.016
    # corner bolts
    cd = np.hypot(np.minimum(FX, 1 - FX) * 2 - 0.16, np.minimum(FY, 1 - FY) * 2 - 0.16)
    bolt = smoothstep(0.06, 0.045, cd)
    hexs = smoothstep(0.03, 0.02, cd)
    h += bolt * 0.012 - hexs * 0.006
    steel_a, steel_b = hex2('a3a9b4'), hex2('8a909c')
    alb = np.where(tread_tile[..., None], steel_a, steel_b) * (0.92 + 0.12 * tile_rand(82)[..., None])
    alb = alb * (0.85 + 0.25 * noise(18, 14, 5, 83)[..., None])
    grime = smoothstep(0.25, 0.02, EDGE) * 0.6 + smoothstep(0.55, 0.8, noise(12, 10, 5, 84)) * 0.5
    alb = lerp(alb, hex2('2a2622'), np.clip(grime, 0, 1) * 0.4)
    oil = smoothstep(0.84, 0.9, noise(20, 16, 4, 85)) * 0.6
    alb = lerp(alb, hex2('1c1a20'), oil * 0.6)
    sc = noise(1400, 18, 1, 86)
    scratch = smoothstep(0.9, 0.97, sc) * smoothstep(0.55, 0.75, noise(30, 24, 3, 88))
    alb = lerp(alb, hex2('aab0ba'), scratch * 0.35)
    alb = lerp(alb, hex2('15161a'), smoothstep(grout_w + 0.01, grout_w - 0.005, EDGE))
    alb = lerp(alb, hex2('d8dce4'), (rivet > 0.5) * 0.5)
    # amber guide lights set into every other seam row
    seam_row = (np.abs(FY - 0.0) < 0.012) & (TJ % 2 == 0) & (np.mod(TU * 4, 1) < 0.5)
    lamp = blur(seam_row.astype(float), 3, False)
    alb = lerp(alb, hex2('ffb547'), np.clip(seam_row + lamp * 2, 0, 1))
    # hazard band in front of the paddle (z < 1)
    band = (WZ < 1.0) & (WZ > -3)
    stripe = np.mod((WX + WZ) / 0.9, 1) < 0.5
    worn = smoothstep(0.35, 0.6, noise(120, 96, 4, 87))
    haz = np.where(stripe[..., None], hex2('e8b021'), hex2('1d1d20'))
    alb = np.where(band[..., None], lerp(haz, alb, (1 - worn)[..., None] * 0.35 + 0.0), alb)
    edge_line = (np.abs(WZ - 1.0) < 0.05)
    alb = np.where(edge_line[..., None], hex2('f2c94c'), alb)
    rgb = shade(alb, h, S, light=(0.4, 0.7, -0.6), ao=1.0, spec=0.3, spec_pow=25, periodic=False, emissive=lamp[..., None] * hex2('ff9a2e') * 0.6)
    save('machine', rgb)


# ------------------------------------------------------------------ neon
def neon(pattern):
    """Greyscale glass floor, tinted and pulsed in game. pattern 0 hex,
    1 dots, 2 diamond lattice, 3 rings."""
    e = EDGE
    frame = smoothstep(0.035, 0.012, e) + smoothstep(0.12, 0.0, e) * 0.25
    dx, dy = (FX - 0.5) * 2, (FY - 0.5) * 2
    if pattern == 0:
        px, py = WX * 1.3, WZ * 1.3
        rx, ry = 1.0, 1.7320508
        ax, ay = np.mod(px, rx) - rx / 2, np.mod(py, ry) - ry / 2
        bx, by = np.mod(px - rx / 2, rx) - rx / 2, np.mod(py - ry / 2, ry) - ry / 2
        use_a = ax * ax + ay * ay < bx * bx + by * by
        gx, gy = np.abs(np.where(use_a, ax, bx)), np.abs(np.where(use_a, ay, by))
        hd = np.maximum(gx * 0.5 + gy * 0.8660254, gx)
        inner = smoothstep(0.035, 0.0, np.abs(0.5 - hd)) * 0.55
    elif pattern == 1:
        gx = np.mod(FX * 6, 1) - 0.5
        gy = np.mod(FY * 6, 1) - 0.5
        inner = smoothstep(0.14, 0.08, np.hypot(gx, gy)) * 0.5
    elif pattern == 2:
        s = np.abs(dx) + np.abs(dy)
        inner = smoothstep(0.03, 0.0, np.abs(s - 1.0)) * 0.7 + smoothstep(0.02, 0.0, np.abs(s - 0.5)) * 0.35
    else:
        d = np.hypot(dx, dy)
        inner = smoothstep(0.03, 0.0, np.abs(d - 0.62)) * 0.7 + smoothstep(0.02, 0.0, np.abs(d - 0.35)) * 0.3
    # circuit traces
    tr = noise(6, 5, 1, 90 + pattern)
    traces = (smoothstep(0.012, 0.0, np.abs(np.mod(WX * 4 + 0.5, 1) - 0.5) / 4) * (np.mod(WZ * 1.5 + tr * 3, 1) < 0.35) +
              smoothstep(0.012, 0.0, np.abs(np.mod(WZ * 4 + 0.5, 1) - 0.5) / 4) * (np.mod(WX * 1.5 + tr * 3, 1) < 0.35))
    traces = np.clip(traces, 0, 1) * 0.18 * (e > 0.15)
    glass = 0.07 + 0.05 * noise(10, 8, 4, 95) + 0.04 * smoothstep(0.3, 0.0, np.abs(FX - FY + 0.2))
    v = np.clip(glass + frame + inner * (e > 0.1) + traces, 0, 1)
    v = v + blur(v, S * 0.04, False) * 0.35
    save('neon_%d' % pattern, np.repeat(np.clip(v, 0, 1)[..., None], 3, -1))


WORLDS = {'meadow': meadow, 'ruins': ruins, 'caverns': caverns, 'clouds': clouds, 'machine': machine,
          'neon_0': lambda: neon(0), 'neon_1': lambda: neon(1), 'neon_2': lambda: neon(2), 'neon_3': lambda: neon(3)}

if __name__ == '__main__':
    for n in (sys.argv[1:] or list(WORLDS)):
        WORLDS[n]()
