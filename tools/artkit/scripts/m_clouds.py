"""Cloud Kingdom set pieces. Props face -Y (towards the game camera)."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)


def _float_rock(r, top_rgb, under_rgb, seed, name):
    """A floating island: grassy cap on top (z=0), rocky cone hanging below."""
    top = M(name + "_top", top_rgb, var=0.12)
    under = M(name + "_under", under_rgb, var=0.15)
    rng = random.Random(seed)
    prof = [(0.0, -1.6), (0.35, -1.25), (0.62, -0.85), (0.86, -0.42), (1.0, -0.1), (1.0, 0.0)]
    verts, faces = [], []
    n = 12
    for i, (rr, z) in enumerate(prof):
        for s in range(n):
            a = s / n * 2 * math.pi
            j = 1 + (rng.uniform(-0.12, 0.12) if 0 < i < len(prof) - 1 else 0)
            verts.append((math.cos(a) * rr * r * j, math.sin(a) * rr * r * 0.8 * j, z * r))
    for i in range(len(prof) - 1):
        for s in range(n):
            a = i * n + s
            b = i * n + (s + 1) % n
            faces.append((a, b, b + n, a + n))
    mesh(verts, faces, under)
    cyl((0, 0, -0.05 * r), r * 1.02, 0.12 * r, top, n=n)
    sphere((0, 0, 0.0), r * 0.98, top, sub=2, scale=(1, 0.8, 0.12))


@asset("float_rock_green", "clouds", ao=1.5)
def float_rock_green():
    _float_rock(1.0, (130, 200, 110), (150, 130, 120), 3, "fg")


@asset("sky_castle", "clouds", ao=4.0)
def sky_castle():
    """A castle on its island; the island's top is at z=0 (scaled by s in game)."""
    _float_rock(7.0, (130, 200, 110), (150, 130, 120), 11, "sc")
    wall = M("sc_wall", (245, 240, 250), var=0.04)
    shade = M("sc_shade", (220, 216, 232))
    win = M("sc_win", (255, 220, 140), emit=0.8)
    roofs = [M("sc_r0", (120, 140, 230)), M("sc_r1", (240, 130, 170)), M("sc_r2", (255, 200, 90))]
    pole = M("sc_pole", (120, 100, 80))
    box((0, 0, 1.7), (6.4, 1.2, 3.4), wall, bev=0.05, seg=1)
    for k in range(7):
        box((-2.7 + k * 0.9, -0.6, 3.6), (0.5, 0.25, 0.4), wall)
    box((0, -0.62, 1.1), (1.0, 0.06, 1.8), M("sc_door", (140, 100, 80)))
    for i, (tx, ty, hs) in enumerate(((-3.2, 0, 1.1), (3.2, 0, 1.1), (0, 1.6, 1.6), (-1.9, -2.6, 0.9), (2.3, -2.4, 0.95))):
        h = 5.0 * hs
        lathe([(1.0, 0), (1.0, h), (1.3, h + 0.1), (1.3, h + 0.7), (1.05, h + 0.75)], wall, n=12,
              c=(tx, ty, 0), mats=[wall, shade, wall, shade])
        lathe([(1.45, 0), (1.1, 0.8), (0.5, 2.0), (0.0, 2.9)], roofs[i % 3], n=12, c=(tx, ty, h + 0.75))
        for z in (h * 0.45, h * 0.8):
            box((tx, ty - 1.0, z), (0.35, 0.1, 0.6), win)
        cyl((tx, ty, h + 3.6), 0.04, 1.2, pole, n=4)


@asset("airship", "clouds", ao=1.0)
def airship():
    """Balloon over a wooden gondola; the gondola's base sits at z=0."""
    stripe = [M("as_a", (240, 150, 170)), M("as_b", (250, 236, 220))]
    wood = M("as_wood", (150, 100, 70), var=0.1)
    rope = M("as_rope", (90, 70, 60))
    n = 12
    verts, faces = [], []
    rings = 10
    for k in range(rings + 1):
        a = k / rings * math.pi
        x = -math.cos(a) * 4.6
        rr = math.sin(a) * 2.2
        for s in range(n):
            b = s / n * 2 * math.pi
            verts.append((x, math.cos(b) * rr, 3.6 + math.sin(b) * rr))
    fm = []
    for k in range(rings):
        for s in range(n):
            a = k * n + s
            b = k * n + (s + 1) % n
            faces.append((a, b, b + n, a + n))
            fm.append(s % 2)
    o = mesh(verts, faces, stripe[0])
    o.data.materials.append(stripe[1])
    for p, m in zip(o.data.polygons, fm):
        p.material_index = m
    box((0, 0, 0.5), (4.8, 1.6, 1.0), wood, bev=0.08, seg=1)
    box((0, 0, 1.05), (5.0, 1.8, 0.12), wood)
    for dx in (-1.8, 1.8):
        for dy in (-0.7, 0.7):
            tube([(dx, dy, 1.0), (dx * 0.8, dy * 1.5, 2.2)], 0.03, rope, n=4)
    place(cyl((0, 0, 0), 0.12, 0.5, rope, n=6), loc=(-2.6, 0, 0.5), rot=(0, 90, 0))


@asset("banner_pole", "clouds", ao=0.5)
def banner_pole():
    pole = M("bp_pole", (230, 220, 240))
    gold = M("bp_gold", (255, 210, 100), emit=0.3)
    cyl((0, 0, 0), 0.08, 6.0, pole, n=6)
    cyl((0, 0, 0), 0.2, 0.3, pole, n=8)
    sphere((0, 0, 6.1), 0.2, gold, sub=1)


@asset("buoy", "clouds", ao=0.4)
def buoy():
    """A tethered balloon buoy, centred on the origin; tinted per buoy."""
    skin = M("buoy_skin", (240, 240, 240), tint=True, var=0.03)
    band = M("buoy_band", (250, 250, 255))
    sphere((0, 0, 0), 0.9, skin, sub=2, scale=(1, 1, 1.2))
    torus((0, 0, -0.2), 0.88, 0.08, band, seg=16, rseg=4)
    cyl((0, 0, -1.25), 0.12, 0.2, band, n=6)


def _cloud(seed):
    puff = M("cloudpuff", (252, 252, 255))
    under = M("cloudunder", (196, 200, 232))
    rng = random.Random(seed)
    n = 4 + rng.randint(0, 2)
    for k in range(n):
        a = k / n * 2 * math.pi
        rad = 1.0 if k == 0 else 0.55 + rng.random() * 0.35
        off = 0 if k == 0 else 0.7 + rng.random() * 0.3
        sphere((math.cos(a) * off, math.sin(a) * off * 0.8, (rad - 1) * 0.4), rad, puff, sub=2, jitter=0.06, seed=seed + k)
    sphere((0, 0, -0.35), 1.15, under, sub=1, scale=(1.3, 1.1, 0.3))


@asset("cloud_a", "clouds", ao=1.0)
def cloud_a():
    _cloud(1)


@asset("cloud_b", "clouds", ao=1.0)
def cloud_b():
    _cloud(7)


@asset("cloud_c", "clouds", ao=1.0)
def cloud_c():
    _cloud(13)


@asset("cloud_d", "clouds", ao=1.0)
def cloud_d():
    _cloud(21)
