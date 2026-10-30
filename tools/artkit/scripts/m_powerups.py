"""Power-up drops: each power has its own little object, about 1.2 across,
centred on the origin and facing -Y (the game camera). Not tinted: every
power has fixed colours so it reads at a glance."""
import math
from models import (asset, M, box, cyl, sphere, lathe, mesh, place, tube, prism, torus)


def _plate(col, emit=0.5):
    """A thin glowing ring the object floats in front of, shared by all."""
    torus((0, 0.12, 0), 0.5, 0.045, M("pw_ring_" + col[0], col[1], emit=emit), seg=24, rseg=4, rot=(90, 0, 0))


@asset("pw_multi", "powerups", ao=0.4)
def pw_multi():
    gold = M("pm_gold", (255, 206, 80), emit=0.3, metal=0.8, rough=0.3)
    orb = M("pm_orb", (255, 246, 210), emit=0.9)
    for k in range(3):
        a = math.radians(90 + k * 120)
        sphere((math.cos(a) * 0.28, 0, math.sin(a) * 0.28), 0.17, orb, sub=2)
    torus((0, 0, 0), 0.28, 0.035, gold, seg=24, rseg=4, rot=(90, 0, 0))
    _plate(("multi", (255, 210, 60)))


def _arrow(x0, x1, y, mat, head=0.16):
    """A flat arrow from x0 toward x1 (thick enough to read in 3D)."""
    d = 1 if x1 > x0 else -1
    shaft_end = x1 - d * head
    box(((x0 + shaft_end) / 2, y, 0), (abs(shaft_end - x0), 0.1, 0.09), mat)
    pts = [(shaft_end, -0.15), (x1, 0.0), (shaft_end, 0.15)]
    if d < 0:
        pts = pts[::-1]
    o = prism([(p[0], p[1]) for p in pts], 0.1, mat, z0=-0.05)
    o.rotation_euler = (math.radians(90), 0, 0)
    o.location = (0, y + 0.05, 0)


@asset("pw_expand", "powerups", ao=0.4)
def pw_expand():
    blue = M("pe_blue", (70, 150, 255), emit=0.35, rough=0.35)
    white = M("pe_white", (236, 244, 255), emit=0.6)
    box((0, 0, -0.18), (0.5, 0.22, 0.16), blue, bev=0.05, seg=2)
    for d in (-1, 1):
        _arrow(d * 0.12, d * 0.58, -0.02, white)
    _plate(("expand", (70, 150, 255)))


@asset("pw_shrink", "powerups", ao=0.4)
def pw_shrink():
    red = M("ps_red", (230, 60, 60), emit=0.3, rough=0.35)
    white = M("ps_white", (255, 236, 236), emit=0.5)
    box((0, 0, -0.18), (0.3, 0.22, 0.16), red, bev=0.05, seg=2)
    for d in (-1, 1):
        _arrow(d * 0.6, d * 0.2, -0.02, white)
    _plate(("shrink", (230, 60, 60)))


@asset("pw_slow", "powerups", ao=0.5)
def pw_slow():
    glass = M("pl_glass", (150, 245, 230), emit=0.35, rough=0.15)
    sand = M("pl_sand", (255, 214, 120), emit=0.4)
    wood = M("pl_wood", (120, 80, 50))
    for z in (-0.36, 0.36):
        cyl((0, 0, z - 0.04), 0.3, 0.08, wood, n=12)
    for z in (-0.33, 0.33):
        pass
    lathe([(0.0, -0.32), (0.24, -0.3), (0.26, -0.16), (0.06, -0.02), (0.06, 0.02), (0.26, 0.16), (0.24, 0.3), (0.0, 0.32)], glass, n=14)
    lathe([(0.0, -0.31), (0.2, -0.29), (0.16, -0.2), (0.0, -0.17)], sand, n=12)
    lathe([(0.0, 0.06), (0.1, 0.12), (0.16, 0.24), (0.0, 0.24)], sand, n=12)
    for a in (45, 135, 225, 315):
        x, y = math.cos(math.radians(a)) * 0.27, math.sin(math.radians(a)) * 0.27
        cyl((x, y, -0.32), 0.025, 0.64, wood, n=6)
    _plate(("slow", (80, 220, 200)))


@asset("pw_haste", "powerups", ao=0.4)
def pw_haste():
    bolt = M("ph_bolt", (255, 90, 190), emit=0.7)
    rim = M("ph_rim", (120, 20, 70))
    pts = [(-0.12, 0.5), (0.2, 0.5), (0.04, 0.1), (0.26, 0.1), (-0.14, -0.5), (0.0, -0.06), (-0.22, -0.06)]
    o = prism(pts, 0.14, bolt, z0=-0.07)
    o.rotation_euler = (math.radians(90), 0, 0)
    o2 = prism([(x * 1.12, y * 1.08) for x, y in pts], 0.08, rim, z0=-0.04)
    o2.rotation_euler = (math.radians(90), 0, 0)
    o2.location = (0, 0.06, 0)
    _plate(("haste", (200, 40, 120)))


@asset("pw_fire", "powerups", ao=0.4)
def pw_fire():
    outer = M("pf_outer", (255, 110, 30), emit=0.9)
    inner = M("pf_inner", (255, 220, 90), emit=1.2)
    core = M("pf_core", (255, 255, 230), emit=1.5)
    lathe([(0.0, -0.4), (0.24, -0.34), (0.3, -0.16), (0.24, 0.06), (0.12, 0.28), (0.0, 0.5)], outer, n=10)
    for a, s in ((-0.5, 0.6), (0.5, 0.55)):
        o = lathe([(0.0, -0.1), (0.1, -0.05), (0.08, 0.12), (0.0, 0.26)], outer, n=8)
        o.location = (a * 0.4, 0, -0.18)
        o.rotation_euler = (0, -a * 0.8, 0)
    o = lathe([(0.0, -0.36), (0.16, -0.3), (0.18, -0.12), (0.08, 0.12), (0.0, 0.26)], inner, n=10)
    o.location = (0, -0.06, 0)
    sphere((0, -0.14, -0.2), 0.08, core, sub=1)
    _plate(("fire", (255, 120, 30)))


@asset("pw_laser", "powerups", ao=0.5)
def pw_laser():
    steel = M("pz_steel", (90, 96, 110), metal=0.8, rough=0.35)
    red = M("pz_red", (255, 70, 70), emit=1.0)
    box((0, 0, -0.16), (0.7, 0.3, 0.18), steel, bev=0.04, seg=2)
    for x in (-0.18, 0.18):
        o = cyl((0, 0, 0), 0.07, 0.5, steel, n=10)
        o.location = (x, 0, -0.1)
        sphere((x, 0, 0.42), 0.075, red, sub=1)
        torus((x, 0, 0.2), 0.08, 0.02, red, seg=10, rseg=3)
    _plate(("laser", (255, 70, 70)))


@asset("pw_barrier", "powerups", ao=0.5)
def pw_barrier():
    stone = M("pb_stone", (150, 120, 255), emit=0.35, rough=0.4)
    glow = M("pb_glow", (220, 210, 255), emit=0.9)
    box((0, 0, -0.08), (0.9, 0.2, 0.34), stone, bev=0.03, seg=1)
    for x in (-0.36, -0.12, 0.12, 0.36):
        box((x, 0, 0.16), (0.16, 0.2, 0.16), stone, bev=0.02, seg=1)
    box((0, -0.105, -0.08), (0.8, 0.02, 0.05), glow)
    _plate(("barrier", (150, 120, 255)))


@asset("pw_glue", "powerups", ao=0.4)
def pw_glue():
    goo = M("pg_goo", (120, 230, 70), emit=0.35, rough=0.15)
    shine = M("pg_shine", (230, 255, 210), emit=0.8)
    lathe([(0.0, -0.36), (0.26, -0.3), (0.32, -0.1), (0.24, 0.12), (0.1, 0.32), (0.0, 0.46)], goo, n=14)
    sphere((-0.1, -0.24, 0.0), 0.06, shine, sub=1)
    for x, s in ((-0.4, 0.09), (0.38, 0.07)):
        sphere((x, 0, -0.32), s, goo, sub=1)
    _plate(("glue", (120, 220, 70)))


@asset("pw_magnet", "powerups", ao=0.5)
def pw_magnet():
    red = M("pq_red", (230, 50, 50), emit=0.3, rough=0.35)
    steel = M("pq_steel", (220, 226, 236), metal=0.8, rough=0.25, emit=0.2)
    pts = []
    for k in range(13):
        a = math.pi * k / 12
        pts.append((math.cos(a) * 0.3, 0, -0.02 + math.sin(a) * 0.3))
    tube(pts, 0.1, red, n=10)
    for x in (-0.3, 0.3):
        o = cyl((0, 0, 0), 0.1, 0.24, red, n=10)
        o.location = (x, 0, -0.26)
        o = cyl((0, 0, 0), 0.105, 0.12, steel, n=10)
        o.location = (x, 0, -0.38)
    _plate(("magnet", (255, 200, 80)))


@asset("pw_life", "powerups", ao=0.4)
def pw_life():
    pink = M("pv_pink", (255, 90, 140), emit=0.6, rough=0.3)
    hi = M("pv_hi", (255, 220, 236), emit=0.9)
    pts = []
    for k in range(40):
        t = k / 40 * 2 * math.pi
        x = 16 * math.sin(t) ** 3
        y = 13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t)
        pts.append((x / 36, y / 36))
    o = prism(pts[::-1], 0.22, pink, z0=-0.11)
    o.rotation_euler = (math.radians(90), 0, 0)
    sphere((-0.15, -0.12, 0.12), 0.05, hi, sub=1)
    _plate(("life", (255, 110, 160)))


@asset("pw_cache", "powerups", ao=0.6)
def pw_cache():
    wood = M("pc_wood", (140, 86, 50), var=0.06)
    gold = M("pc_gold", (255, 206, 80), emit=0.35, metal=0.8, rough=0.3)
    rune = M("pc_rune", (140, 240, 255), emit=1.2)
    box((0, 0, -0.14), (0.72, 0.46, 0.4), wood, bev=0.03, seg=1)
    o = cyl((0, 0, 0), 0.23, 0.72, wood, n=12, rot=(0, 90, 0))
    o.location = (0, 0, 0.06)
    for x in (-0.3, 0.3):
        box((x, 0, -0.04), (0.06, 0.48, 0.62), gold)
    box((0, -0.24, -0.02), (0.14, 0.04, 0.16), gold)
    box((0, -0.265, -0.02), (0.06, 0.02, 0.07), rune)
    _plate(("cache", (255, 214, 90)))
