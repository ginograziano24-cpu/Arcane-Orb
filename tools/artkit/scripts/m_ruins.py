"""Ancient Ruins set pieces. Props face -Y (towards the game camera)."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)

SAND = (214, 182, 132)


def _stone(name="sand", rgb=SAND, var=0.1):
    return M(name, rgb, var=var)


@asset("stone_head", "ruins", ao=4.0)
def stone_head():
    """The colossal head; its base sits at z=0 (the game puts it at y=-0.6)."""
    st = _stone()
    dark = M("hd_dark", (110, 90, 66))
    eye = M("hd_eye", (90, 220, 210), emit=1.2)
    moss = M("hd_moss", (110, 150, 80), var=0.2)
    # stacked courses of big blocks, slightly uneven
    rng = random.Random(3)
    for k in range(6):
        z = k * 1.62
        w = 10 - (0.3 if k % 2 else 0)
        box((rng.uniform(-0.1, 0.1), 0, z + 0.8), (w, 6, 1.58), st, bev=0.12, seg=1)
    box((0, -0.2, 10.95), (11.2, 6.4, 3.9), _stone("sand2", (196, 166, 120)), bev=0.15, seg=1)
    for k in range(-2, 3):
        hgt = 2.5 - abs(k) * 0.6
        box((k * 2.0, -0.2, 12.9 + hgt / 2), (1.2, 5.2, hgt), st, bev=0.1, seg=1)
    box((0, -3.25, 7.9), (8.4, 0.6, 0.8), _stone("brow", (184, 154, 110)), bev=0.08, seg=1)
    for sx in (-1, 1):
        box((sx * 2.35, -3.05, 6.6), (2.3, 0.3, 1.2), dark)
        sphere((sx * 2.35, -3.2, 6.6), 0.32, eye, sub=1)
    box((0, -3.6, 5.6), (1.6, 1.2, 3.6), st, bev=0.1, seg=1)
    box((0, -3.2, 2.5), (4.8, 0.4, 1.0), dark)
    for k in range(5):
        foliage((-4.6 + k * 2.3, -3.1, 13.3 + (k % 2) * 0.2), 0.7, moss, seed=60 + k, scale=(1.2, 0.8, 0.5), sub=1)


def _column(h, broken, seed):
    st = _stone("col", (204, 174, 128))
    prof = [(1.25, 0), (1.25, 0.4), (1.05, 0.5), (1.1, 0.7), (0.85, 0.85), (0.78, h - 0.9)]
    if not broken:
        prof += [(1.0, h - 0.6), (1.3, h - 0.3), (1.3, h)]
    else:
        prof += [(0.0, h - 0.7)]
    lathe(prof, st, n=14)
    # fluting
    for k in range(10):
        a = k / 10 * 2 * math.pi
        box((math.cos(a) * 0.8, math.sin(a) * 0.8, (h - 0.9 + 0.85) / 2 if not broken else (h - 0.5) / 2 + 0.4),
            (0.07, 0.07, (h - 1.8) if not broken else h - 1.6), _stone("colshade", (178, 150, 108)))
    if not broken:
        box((0, 0, h + 0.2), (3.0, 3.0, 0.4), _stone("cap", (214, 184, 136)), bev=0.06, seg=1)
    else:
        rng = random.Random(seed)
        for k in range(2):
            rock((rng.uniform(1.2, 1.8) * (1 if k else -1), rng.uniform(-1.5, 1.5), 0.25), 0.55 - k * 0.15, st, seed=seed + k, scale=(1, 0.9, 0.6), sub=1)


@asset("column", "ruins", ao=2.0)
def column():
    _column(8.0, False, 0)


@asset("column_broken", "ruins", ao=2.0)
def column_broken():
    _column(4.0, True, 7)


@asset("palm_trunk", "ruins", ao=1.0)
def palm_trunk():
    """A straight trunk 7.8 tall (the game leans it); fronds are separate."""
    bark = M("palmbark", (150, 116, 80), var=0.12)
    bark2 = M("palmbark2", (128, 98, 66), var=0.12)
    for k in range(10):
        r = 0.44 - k * 0.022
        cyl((0, 0, k * 0.78), r * 1.12, 0.5, bark if k % 2 else bark2, n=8, r2=r)
        cyl((0, 0, k * 0.78 + 0.5), r, 0.28, bark2, n=8)


@asset("palm_fronds", "ruins", ao=0.6)
def palm_fronds():
    """Drooping fronds around the crown at the origin."""
    leaf = M("frond", (86, 150, 70), var=0.15)
    leaf2 = M("frond2", (70, 128, 58), var=0.15)
    sphere((0, 0, 0), 0.4, M("coco", (110, 80, 50)), sub=1)
    for k in range(8):
        a = k / 8 * 2 * math.pi
        pts = []
        for j in range(6):
            t = j / 5
            pts.append((math.cos(a) * t * 4.0, math.sin(a) * t * 4.0, 0.4 * math.sin(t * 2.6) - t * t * 2.0))
        # a frond: a flat strip that narrows to the tip, with leaflets
        verts, faces = [], []
        for j, p in enumerate(pts):
            w = 0.55 * (1 - j / 6)
            nx, ny = -math.sin(a) * w, math.cos(a) * w
            verts += [(p[0] + nx, p[1] + ny, p[2] - 0.05 * j), (p[0] - nx, p[1] - ny, p[2] - 0.05 * j)]
        for j in range(len(pts) - 1):
            i = j * 2
            faces.append((i, i + 1, i + 3, i + 2))
        mesh(verts, faces, leaf if k % 2 else leaf2)


@asset("temple_gate", "ruins", ao=3.0)
def temple_gate():
    """Two pillars and a lintel; the doorway is 6.6 wide. Base at z=0."""
    st = _stone("gate", (204, 174, 128))
    trim = M("gatetrim", (220, 180, 80))
    teal = M("gateteal", (70, 170, 165), emit=0.4)
    dark = M("gatedark", (30, 22, 18))
    for sx in (-1, 1):
        for k in range(5):
            box((sx * 4.4, 0, k * 1.92 + 0.96), (2.3 - (k % 2) * 0.1, 2.5, 1.88), st, bev=0.1, seg=1)
        box((sx * 4.4, -1.27, 5.0), (1.2, 0.06, 4.0), teal)
    box((0, 0, 10.4), (11.8, 2.9, 1.7), st, bev=0.12, seg=1)
    box((0, -1.47, 10.4), (9.0, 0.08, 0.3), trim)
    box((0, -1.47, 11.0), (2.0, 0.08, 0.6), teal)
    box((0, 0.2, 4.8), (6.6, 0.4, 9.6), dark)


@asset("temple_door", "ruins", ao=0.8)
def temple_door():
    """The stone slab that slides up out of the doorway (6.6 x 9.6)."""
    st = _stone("door", (176, 148, 108), var=0.08)
    teal = M("doorteal", (80, 180, 170), emit=0.5)
    box((0, 0, 4.8), (6.6, 0.25, 9.6), st, bev=0.05, seg=1)
    box((0, -0.14, 4.8), (0.1, 0.04, 9.0), _stone("doorseam", (120, 100, 74)))
    for z in (3.0, 6.0):
        torus((0, -0.15, z), 0.9, 0.08, teal, seg=16, rseg=4, rot=(90, 0, 0))


@asset("brazier", "ruins", ao=0.8)
def brazier():
    st = _stone("brz_st", (130, 104, 76))
    bronze = M("bronze", (210, 160, 70), metal=0.8)
    coal = M("coal", (60, 30, 20))
    lathe([(0.62, 0), (0.52, 0.22), (0.2, 0.42), (0.16, 2.0)], st, n=10)
    lathe([(0.26, 2.0), (0.82, 2.25), (0.92, 2.7), (0.78, 2.75), (0.7, 2.5), (0.0, 2.45)], bronze, n=12)
    cyl((0, 0, 2.45), 0.7, 0.12, coal, n=12)
    for k in range(3):
        a = k / 3 * 2 * math.pi
        tube([(math.cos(a) * 0.5, math.sin(a) * 0.5, 0.1), (math.cos(a) * 0.3, math.sin(a) * 0.3, 1.2), (math.cos(a) * 0.4, math.sin(a) * 0.4, 2.1)],
             0.05, bronze, n=4)


@asset("urn", "ruins", ao=0.5)
def urn():
    clay = M("clay", (190, 110, 70), var=0.06)
    teal = M("urnteal", (60, 150, 150))
    lathe([(0.3, 0), (0.55, 0.3), (0.6, 0.62), (0.6, 0.78), (0.4, 1.1), (0.25, 1.25), (0.35, 1.4), (0.0, 1.38)], clay, n=12,
          mats=[clay, clay, teal, clay, clay, clay, clay])
    for sx in (-1, 1):
        torus((sx * 0.42, 0, 1.0), 0.14, 0.04, clay, seg=8, rseg=4, rot=(90, 0, 0))


@asset("glyph_slab", "ruins", ao=1.0)
def glyph_slab():
    """A standing slab 6 wide, its carved face towards -Y."""
    st = _stone("slab", (204, 174, 128))
    glyph = M("glyph", (70, 140, 140), emit=0.35)
    box((0, 0, 3.3), (6, 0.8, 6.6), st, bev=0.1, seg=1)
    rng = random.Random(9)
    for k in range(12):
        x = -2.2 + (k % 3) * 2.2
        z = 1.2 + (k // 3) * 1.35
        kind = rng.randint(0, 2)
        if kind == 0:
            torus((x, -0.42, z + 0.4), 0.3, 0.06, glyph, seg=10, rseg=3, rot=(90, 0, 0))
        elif kind == 1:
            box((x, -0.42, z + 0.4), (0.1, 0.05, 0.8), glyph)
            box((x, -0.42, z + 0.7), (0.5, 0.05, 0.1), glyph)
        else:
            box((x, -0.42, z + 0.4), (0.45, 0.05, 0.45), glyph, rot=(0, 45, 0))


@asset("altar", "ruins", ao=1.0)
def altar():
    st = _stone("altar", (196, 166, 120))
    gold = M("altargold", (230, 180, 60), emit=0.2)
    box((0, 0, 0.35), (4.6, 3.6, 0.7), st, bev=0.08, seg=1)
    box((0, 0, 1.05), (4.0, 3.0, 0.7), st, bev=0.08, seg=1)
    box((0, -1.52, 1.05), (3.2, 0.06, 0.16), gold)


@asset("relic", "ruins", ao=0.2)
def relic():
    """The floating golden relic: a faceted double pyramid around the origin."""
    gold = M("relicgold", (255, 214, 110), emit=0.6)
    cyl((0, 0, 0), 0.95, 1.0, gold, n=6, r2=0.0)
    place(cyl((0, 0, 0), 0.95, 1.0, gold, n=6, r2=0.0), loc=(0, 0, -0.5), rot=(180, 0, 0))
