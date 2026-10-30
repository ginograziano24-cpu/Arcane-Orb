"""Crystal Caverns set pieces, and the crystal shard every world uses."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)


@asset("crystal_shard", "caverns", ao=0.3)
def crystal_shard():
    """A six-sided shard 1 tall and about 1 across, tinted in code."""
    cr = M("shard", (240, 240, 250), tint=True, emit=0.3)
    n = 6
    rng = random.Random(2)
    verts = []
    for ring, (r, z) in enumerate(((0.42, 0.0), (0.5, 0.62), (0.0, 1.0))):
        for s in range(n):
            a = s / n * 2 * math.pi + 0.3
            j = 1 + rng.uniform(-0.08, 0.08)
            verts.append((math.cos(a) * r * j, math.sin(a) * r * j, z + (rng.uniform(-0.04, 0.04) if ring == 1 else 0)))
    faces = [tuple(range(n))[::-1]]
    for i in range(n):
        j = (i + 1) % n
        faces.append((i, j, j + n, i + n))
        faces.append((i + n, j + n, 2 * n + 0))
    verts = verts[:2 * n] + [(0, 0, 1.0)]
    mesh(verts, faces, cr)


@asset("cluster_base", "caverns", ao=1.0)
def cluster_base():
    st = M("cb_rock", (70, 74, 104), var=0.15)
    rock((0, 0, 0.0), 1.0, st, seed=31, scale=(1.0, 0.85, 0.4), sub=2, rough=0.2)
    for k in range(5):
        a = k * 1.25
        rock((math.cos(a) * 0.8, math.sin(a) * 0.65, 0.05), 0.32, st, seed=32 + k, sub=1)


@asset("mush_stem", "caverns", ao=0.4)
def mush_stem():
    stem = M("ms_stem", (210, 220, 236), emit=0.1)
    lathe([(0.24, 0), (0.17, 0.6), (0.14, 1.2), (0.19, 1.4)], stem, n=8)


@asset("mush_cap", "caverns", ao=0.4)
def mush_cap():
    """A glowing cap, tinted in code; its rim sits at z=0."""
    cap = M("mc_cap", (240, 240, 250), tint=True, emit=0.4)
    gill = M("mc_gill", (250, 250, 255), tint=True, emit=0.6)
    dots = M("mc_dot", (255, 255, 255), emit=0.8)
    lathe([(0.9, 0.0), (0.86, 0.22), (0.62, 0.46), (0.0, 0.56)], cap, n=12)
    lathe([(0.2, -0.04), (0.88, -0.02)], gill, n=12)
    for k in range(5):
        a = k * 1.3
        sphere((math.cos(a) * 0.5, math.sin(a) * 0.5, 0.42), 0.07, dots, sub=1)


@asset("stalagmite", "caverns", ao=1.0)
def stalagmite():
    st = M("sg_rock", (70, 76, 104), var=0.15)
    tip = M("sg_tip", (110, 120, 150))
    lathe([(1.0, 0), (0.75, 0.25), (0.55, 0.5), (0.32, 0.75), (0.0, 1.0)], st, n=9, mats=[st, st, st, tip])
    rock((0.5, 0.2, 0.05), 0.35, st, seed=8, sub=1, scale=(1, 1, 0.5))


@asset("boulder_a", "caverns", ao=0.8)
def boulder_a():
    st = M("bd_rock", (200, 200, 205), tint=True, var=0.15)
    rock((0, 0, 0.6), 1.0, st, seed=20, scale=(1, 0.95, 0.7), sub=2, rough=0.2)


@asset("boulder_b", "caverns", ao=0.8)
def boulder_b():
    st = M("bd_rock", (200, 200, 205), tint=True, var=0.15)
    rock((0, 0, 0.55), 1.0, st, seed=24, scale=(1.1, 0.8, 0.65), sub=2, rough=0.24)
    rock((0.8, 0.3, 0.25), 0.4, st, seed=25, sub=1)


@asset("cave_arch", "caverns", ao=2.0)
def cave_arch():
    """A long rock arch overhead, 18 deep along Y, centred on the origin."""
    st = M("ca_rock", (64, 70, 100), var=0.15)
    for k in range(6):
        rock((0, -7.5 + k * 3, 0), 2.0, st, seed=50 + k, scale=(1.5, 1.0, 0.8), sub=2, rough=0.2)
