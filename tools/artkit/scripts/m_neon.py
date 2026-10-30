"""Neon Grove set pieces. Props face -Y (towards the game camera)."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)


@asset("lumen_tree", "neon", ao=2.0)
def lumen_tree():
    """A glowing violet tree at s=1; base at z=0 (the game sets it at y=-0.6)."""
    bark = M("lt_bark", (52, 30, 84), var=0.12)
    leaf = M("lt_leaf", (120, 70, 200), emit=0.25, var=0.15)
    vein = M("lt_vein", (255, 120, 230), emit=1.2)
    tube([(0, 0, 0), (0.1, 0, 2.0), (-0.1, 0.1, 4.0), (0, 0, 5.6)], 0.85, bark, n=8, r_end=0.4)
    for k in range(5):
        a = k * 1.3
        tube([(0, 0, 4.2), (math.cos(a) * 1.4, math.sin(a) * 1.0, 5.8 + (k % 3) * 0.5)], 0.22, bark, n=5, r_end=0.1)
        foliage((math.cos(a) * 2.2, math.sin(a) * 1.5, 6.2 + (k % 3) * 1.1), 1.7, leaf, seed=70 + k, scale=(1.2, 1.2, 0.85), sub=2)
    for k in range(6):
        a = k / 6 * 2 * math.pi
        tube([(math.cos(a) * 0.86, math.sin(a) * 0.86, 0.2), (math.cos(a) * 0.55, math.sin(a) * 0.55, 2.5)], 0.04, vein, n=4)


@asset("pylon", "neon", ao=1.0)
def pylon():
    dark = M("py_dark", (40, 28, 70))
    mid = M("py_mid", (60, 40, 100))
    pink = M("py_pink", (255, 90, 220), emit=1.2)
    cyan = M("py_cyan", (90, 240, 255), emit=1.2)
    lathe([(1.2, 0), (1.0, 0.5), (0.5, 1.0), (0.35, 6.0), (0.7, 6.3), (0.7, 6.6), (0.2, 7.4)], mid, n=10,
          mats=[dark, mid, mid, pink, mid, cyan])
    for k in range(3):
        torus((0, 0, 2.0 + k * 1.4), 0.5, 0.05, cyan, seg=12, rseg=4)


@asset("float_rock_neon", "neon", ao=1.0)
def float_rock_neon():
    top = M("fn_top", (58, 40, 92), var=0.12)
    under = M("fn_under", (24, 14, 44), var=0.12)
    rng = random.Random(5)
    rock((0, 0, -0.6), 1.0, under, seed=5, scale=(1.0, 0.85, 1.1), sub=2, rough=0.2)
    cyl((0, 0, -0.05), 1.0, 0.12, top, n=10)
    glow = M("fn_glow", (255, 90, 220), emit=1.0)
    torus((0, 0, -0.1), 1.0, 0.04, glow, seg=14, rseg=3)


@asset("speaker", "neon", ao=0.6)
def speaker():
    """A speaker stack 2.8 wide, 5 tall, cones facing -X (the game turns it)."""
    cab = M("sp_cab", (24, 18, 40))
    rim = M("sp_rim", (80, 60, 120))
    cone = M("sp_cone", (40, 30, 60))
    glow = M("sp_glow", (255, 90, 220), emit=0.9)
    box((0, 0, 2.5), (2.4, 2.8, 5.0), cab, bev=0.06, seg=1)
    for z, r in ((1.4, 0.95), (3.6, 0.65)):
        place(lathe([(0.0, 0.0), (0.3, 0.05), (r * 0.85, 0.2), (r, 0.25), (r, 0.32)], cone, n=14,
                    mats=[cone, cone, rim, glow]), loc=(-1.22, 0, z), rot=(0, -90, 0))
    box((-1.21, 0, 4.7), (0.04, 2.4, 0.08), glow)


@asset("holo_frame", "neon", ao=0.6)
def holo_frame():
    """The billboard's pole and frame; the screen's bars are drawn in code."""
    metal = M("hf_metal", (60, 50, 90))
    cyan = M("hf_cyan", (30, 200, 255), emit=1.0)
    cyl((0, 0, 0), 0.15, 7.0, metal, n=6)
    for z in (6.5, 11.0):
        box((0, 0, z), (0.15, 8.2, 0.15), cyan)
    for y in (-4.0, 4.0):
        box((0, y, 8.75), (0.15, 0.15, 4.6), cyan)


@asset("neon_palm", "neon", ao=0.6)
def neon_palm():
    trunk = M("np_trunk", (40, 30, 70))
    ring = M("np_ring", (255, 90, 220), emit=1.0)
    fr = [M("np_f0", (90, 240, 255), emit=1.0), M("np_f1", (255, 90, 220), emit=1.0), M("np_f2", (255, 220, 90), emit=1.0)]
    cyl((0, 0, 0), 0.2, 6.0, trunk, n=6)
    for k in range(4):
        torus((0, 0, 1.2 + k * 1.3), 0.22, 0.04, ring, seg=8, rseg=3)
    for k in range(6):
        a = k / 6 * 2 * math.pi
        pts = [(0, 0, 6.0), (math.cos(a) * 1.6, math.sin(a) * 1.6, 6.3), (math.cos(a) * 3.0, math.sin(a) * 3.0, 5.4)]
        tube(pts, 0.1, fr[k % 3], n=4, r_end=0.03)
