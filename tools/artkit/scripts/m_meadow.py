"""Sunset Meadow set pieces. Props face -Y (towards the game camera)."""
import math, random
from mathutils import Matrix
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)


@asset("windmill_tower", "meadow", ao=3.0)
def windmill_tower():
    stone = M("mstone", (156, 146, 132), var=0.15)
    plaster = M("plaster", (238, 226, 206), var=0.06)
    wood = M("mwood", (120, 88, 66), var=0.12)
    win = M("mwin", (255, 214, 140), emit=0.6)
    roof = M("mroof", (188, 82, 62), var=0.1)
    for k in range(3):
        cyl((0, 0, k * 0.45), 2.42 - k * 0.04, 0.42, stone, n=14, bev=0.05)
    lathe([(2.05, 1.35), (1.62, 9.6), (1.78, 9.75), (1.78, 10.25), (1.52, 10.35)], plaster, n=14,
          mats=[plaster, wood, wood, wood])
    # timber bands on the plaster
    for z in (4.0, 7.0):
        r = 2.05 - (z - 1.35) / (9.6 - 1.35) * 0.43
        cyl((0, 0, z), r + 0.04, 0.16, wood, n=14)
    lathe([(1.9, 10.3), (1.72, 11.2), (1.2, 12.1), (0.5, 12.8), (0.0, 13.1)], roof, n=14)
    cyl((0, 0, 13.0), 0.08, 0.7, wood, n=6)
    box((0, -2.06, 2.3), (1.1, 0.25, 1.9), wood, bev=0.04, seg=1)
    box((0, -2.12, 3.3), (1.3, 0.2, 0.16), wood)
    for z, y in ((5.2, -1.9), (8.0, -1.73)):
        box((0, y, z), (0.74, 0.2, 0.96), wood, bev=0.03, seg=1)
        box((0, y - 0.08, z), (0.52, 0.06, 0.72), win)
    torus((0, 0, 9.85), 1.95, 0.1, wood, seg=20, rseg=5)
    place(cyl((0, 0, 0), 0.55, 1.0, wood, n=10), loc=(0, -1.55, 12.6), rot=(90, 0, 0))


@asset("windmill_sails", "meadow", ao=0.6)
def windmill_sails():
    """Four lattice sails around the hub at the origin, turning in the XZ plane."""
    wood = M("swood", (128, 96, 72), var=0.1)
    cloth = M("cloth", (244, 238, 226), var=0.05)
    place(cyl((0, 0, 0), 0.5, 0.45, wood, n=10), loc=(0, -0.25, 0), rot=(90, 0, 0))
    for k in range(4):
        a = k * 90
        parts = [box((0, -0.1, 4.6), (0.24, 0.2, 9.0), wood, bev=0.02, seg=1),
                 box((0.85, -0.02, 5.4), (1.45, 0.04, 6.4), cloth)]
        for j in range(5):
            parts.append(box((0.85, -0.08, 2.4 + j * 1.5), (1.6, 0.07, 0.08), wood))
        for j in (0.15, 1.55):
            parts.append(box((j, -0.08, 5.4), (0.07, 0.07, 6.4), wood))
        R = Matrix.Rotation(math.radians(a), 3, 'Y')
        for o in parts:
            o.rotation_euler = (0, math.radians(a), 0)
            o.location = R @ o.location


def _oak(seed, tall):
    bark = M("bark", (112, 78, 52), var=0.12)
    leaf = M("leaf", (128, 186, 90), var=0.16)
    leaf2 = M("leaf2", (98, 156, 72), var=0.16)
    rng = random.Random(seed)
    h = 3.2 if tall else 2.6
    tube([(0, 0, 0), (0.05, 0, h * 0.5), (-0.08, 0.05, h)], 0.5, bark, n=8, r_end=0.26)
    for k in range(3):
        a = k * 2.1 + seed
        tube([(0, 0, h * 0.7), (math.cos(a) * 1.0, math.sin(a) * 0.8, h * 0.95 + 0.3)], 0.18, bark, n=6, r_end=0.08)
    for k in range(5):
        a = k * 1.26 + rng.random()
        d = 1.1 if k else 0.0
        foliage((math.cos(a) * d, math.sin(a) * d * 0.8, h + 1.0 + rng.uniform(-0.2, 0.7)), 1.45 - 0.25 * (k > 0),
                leaf if k % 2 == 0 else leaf2, seed=seed * 10 + k, sub=2)


@asset("oak_a", "meadow", ao=2.0)
def oak_a():
    _oak(1, True)


@asset("oak_b", "meadow", ao=2.0)
def oak_b():
    _oak(2, False)


@asset("shrub", "meadow", ao=1.0)
def shrub():
    leaf = M("shrubleaf", (112, 168, 82), var=0.18)
    foliage((0, 0, 0.55), 0.95, leaf, seed=5, scale=(1.35, 1.3, 0.9))


@asset("blooms", "meadow", ao=0.4)
def blooms():
    """A ring of little flowers to sit on a shrub; tinted per shrub."""
    petal = M("petal", (250, 250, 250), tint=True, emit=0.1)
    heart = M("heart", (255, 220, 90))
    for k in range(7):
        a = k * 0.9
        x, y, z = math.cos(a) * 0.95, math.sin(a) * 0.95, 0.6 + (k % 3) * 0.22
        star = []
        for p in range(10):
            b = p / 10 * 2 * math.pi
            r = 0.17 if p % 2 == 0 else 0.07
            star.append((x + math.cos(b) * r, y + math.sin(b) * r))
        prism(star, 0.04, petal, z0=z)
        box((x, y, z + 0.05), (0.07, 0.07, 0.03), heart)


@asset("fence_post", "meadow", ao=0.8)
def fence_post():
    wood = M("fwood", (160, 116, 76), var=0.12)
    cyl((0, 0, 0), 0.16, 1.45, wood, n=6)
    cyl((0, 0, 1.45), 0.19, 0.08, wood, n=6, r2=0.04)


@asset("fence_rails", "meadow", ao=0.6)
def fence_rails():
    """Two split rails 3 long running along +Y from the post at the origin."""
    wood = M("rwood", (172, 126, 82), var=0.12)
    for z in (0.32, 0.82):
        box((0, 1.5, z), (0.1, 3.0, 0.15), wood, bev=0.02, seg=1)


@asset("hay_bale", "meadow", ao=0.8)
def hay_bale():
    """A round bale lying on its side, axis along X, resting on z=0."""
    hay = M("hay", (226, 186, 104), var=0.12)
    band = M("hband", (190, 150, 80))
    place(cyl((0, 0, 0), 1.05, 1.4, hay, n=16, bev=0.12), loc=(0, 0, 1.05), rot=(0, 90, 0))
    for x in (-0.4, 0.4):
        place(cyl((0, 0, 0), 1.07, 0.06, band, n=16), loc=(x, 0, 1.05), rot=(0, 90, 0))


@asset("barn", "meadow", ao=3.0)
def barn():
    red = M("barnred", (176, 62, 50), var=0.08)
    white = M("barnwhite", (240, 230, 210))
    roof = M("barnroof", (96, 70, 60), var=0.08)
    box((0, 0, 2.8), (10, 8, 5.6), red)
    # gambrel roof as two pitched slabs on each side
    prof = [(-5.4, 0), (-4.2, 1.6), (0, 3.0), (4.2, 1.6), (5.4, 0)]
    verts, faces = [], []
    for y in (-4.3, 4.3):
        for x, z in prof:
            verts.append((x, y, 5.6 + z))
    n = len(prof)
    for i in range(n - 1):
        faces.append((i, i + 1, i + 1 + n, i + n))
    faces.append(tuple(range(n)))
    faces.append(tuple(range(n, 2 * n))[::-1])
    mesh(verts, faces, roof)
    box((0, -4.02, 1.6), (2.8, 0.08, 3.2), white)
    for d in (-1, 1):
        box((0, -4.06, 1.6), (2.6, 0.06, 0.16), white, rot=(0, d * 50, 0))
    box((0, -4.02, 4.7), (1.2, 0.08, 1.0), white)


@asset("scarecrow", "meadow", ao=0.8)
def scarecrow():
    wood = M("sc_wood", (130, 96, 62), var=0.1)
    shirt = M("sc_shirt", (176, 72, 62), var=0.08)
    straw = M("sc_straw", (236, 210, 150), var=0.1)
    hat = M("sc_hat", (204, 164, 84), var=0.08)
    cyl((0, 0, 0), 0.09, 3.3, wood, n=6)
    box((0, 0, 2.4), (2.8, 0.14, 0.14), wood)
    box((0, 0, 2.1), (1.1, 0.5, 1.2), shirt, bev=0.08, seg=1)
    for sx in (-1, 1):
        box((sx * 0.85, 0, 2.38), (0.7, 0.36, 0.34), shirt, bev=0.05, seg=1)
        for k in range(3):
            o = cyl((0, 0, 0), 0.04, 0.35, straw, n=4, r2=0.0)
            place(o, loc=(sx * 1.38, 0, 2.38), rot=(0, sx * (80 + k * 15), k * 30))
    sphere((0, 0, 3.1), 0.42, straw, sub=2)
    lathe([(0.9, 3.38), (0.9, 3.43), (0.46, 3.48), (0.38, 3.92), (0.0, 3.98)], hat, n=12)
    dark = M("sc_dark", (40, 30, 26))
    for sx in (-1, 1):
        box((sx * 0.14, -0.4, 3.15), (0.1, 0.04, 0.1), dark)
    box((0, -0.41, 2.96), (0.24, 0.03, 0.04), dark)


@asset("sunflower_head", "meadow", ao=0.4)
def sunflower_head():
    """The flower head, facing -Y, centred on the origin (stem drawn in code)."""
    seed = M("sf_seed", (96, 60, 30), var=0.1)
    petal = M("sf_petal", (250, 200, 50), var=0.1)
    place(cyl((0, 0, 0), 0.28, 0.1, seed, n=12), loc=(0, -0.05, 0), rot=(90, 0, 0))
    for k in range(14):
        a = k / 14 * 2 * math.pi
        o = sphere((math.cos(a) * 0.45, 0, math.sin(a) * 0.45), 0.2, petal, sub=1, scale=(1, 0.25, 0.45))
        place(o, rot=(0, -math.degrees(a), 0))


@asset("pumpkin", "meadow", ao=0.5)
def pumpkin():
    orange = M("pk", (250, 140, 40), var=0.08)
    stem = M("pkstem", (90, 110, 50))
    for k in range(8):
        a = k / 8 * 2 * math.pi
        sphere((math.cos(a) * 0.22, math.sin(a) * 0.22, 0.5), 0.4, orange, sub=1, scale=(0.75, 0.75, 1.0))
    tube([(0, 0, 0.82), (0.05, 0, 1.0), (0.15, 0, 1.08)], 0.06, stem, n=5, r_end=0.04)


@asset("lantern", "meadow", ao=0.4)
def lantern():
    """A hanging lantern frame (its glow is drawn in code), centred at z=0."""
    iron = M("ln_iron", (60, 50, 40))
    glass = M("ln_glass", (255, 230, 170), emit=0.2)
    box((0, 0, 0), (0.3, 0.3, 0.42), glass)
    for sx in (-1, 1):
        for sy in (-1, 1):
            box((sx * 0.16, sy * 0.16, 0), (0.04, 0.04, 0.5), iron)
    box((0, 0, -0.24), (0.38, 0.38, 0.06), iron)
    cyl((0, 0, 0.22), 0.24, 0.16, iron, n=4, r2=0.04)


@asset("hedgerow", "meadow", ao=1.5)
def hedgerow():
    """A 6-long hedge section along X."""
    leaf = M("hedge", (76, 118, 64), var=0.15)
    for k in range(5):
        foliage((-2.4 + k * 1.2, 0, 0.8), 0.95, leaf, seed=40 + k, scale=(0.85, 0.8, 0.95), sub=1)


@asset("crow", "meadow", ao=0.3)
def crow():
    """A perched crow facing +Y."""
    black = M("crow", (30, 28, 36))
    beak = M("crowbeak", (230, 170, 60))
    sphere((0, 0, 0.22), 0.16, black, sub=1, scale=(0.75, 1.5, 0.8))
    sphere((0, 0.24, 0.36), 0.11, black, sub=1)
    o = cyl((0, 0, 0), 0.035, 0.12, beak, n=5, r2=0.0)
    place(o, loc=(0, 0.36, 0.35), rot=(-90, 0, 0))
    prism([(-0.08, -0.2), (0.08, -0.2), (0.1, -0.42), (-0.1, -0.42)], 0.03, black, z0=0.16)
