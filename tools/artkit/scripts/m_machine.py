"""Machine Works set pieces. Props face -Y (towards the game camera)."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus)


@asset("gear", "machine", ao=0.6)
def gear():
    """A cog of radius 1 in the XZ plane (faces -Y); tinted in code."""
    metal = M("gr_metal", (240, 240, 240), tint=True, var=0.04)
    hub = M("gr_hub", (210, 210, 214), tint=True)
    teeth = 20
    outline = []
    for i in range(teeth):
        a0 = i / teeth * 2 * math.pi
        for da, r in ((0.0, 0.86), (0.18, 0.86), (0.26, 1.06), (0.56, 1.06), (0.64, 0.86)):
            a = a0 + da * 2 * math.pi / teeth
            outline.append((math.cos(a) * r, math.sin(a) * r))
    o = prism(outline, 0.5, metal, z0=-0.25)
    o.rotation_euler = (math.radians(90), 0, 0)
    # spokes and a hub on the front face
    rim = torus((0, 0, 0), 0.74, 0.08, metal, seg=24, rseg=4, rot=(90, 0, 0))
    for k in range(5):
        a = k / 5 * 2 * math.pi
        place(box((0, 0, 0), (1.3, 0.3, 0.14), metal), loc=(math.cos(a) * 0.4, -0.05, math.sin(a) * 0.4), rot=(0, -math.degrees(a), 0))
    place(cyl((0, 0, 0), 0.24, 0.7, hub, n=12), loc=(0, 0.35, 0), rot=(90, 0, 0))


@asset("boiler", "machine", ao=1.5)
def boiler():
    iron = M("bo_iron", (110, 112, 120), var=0.06)
    dark = M("bo_dark", (80, 82, 90))
    copper = M("bo_copper", (132, 98, 74), var=0.08)
    fire = M("bo_fire", (255, 160, 60), emit=1.4)
    lathe([(2.4, 0), (2.4, 0.4), (2.2, 0.5), (2.2, 2.4), (2.35, 2.5), (2.35, 2.7), (2.2, 2.8), (2.2, 5.6), (2.35, 5.7),
           (2.35, 5.9), (2.1, 6.0), (1.6, 7.0), (0.8, 7.5), (0.6, 7.6), (0.6, 10.0), (0.75, 10.1), (0.0, 10.2)], iron, n=14,
          mats=[dark, dark, copper, iron, iron, dark, copper, iron, iron, dark, copper, copper, dark, dark, dark, dark])
    for z in (1.4, 4.2):
        for k in range(14):
            a = k / 14 * 2 * math.pi
            sphere((math.cos(a) * 2.22, math.sin(a) * 2.22, z), 0.06, dark, sub=1)
    box((0, -2.2, 3.0), (1.3, 0.3, 1.0), dark, bev=0.05, seg=1)
    box((0, -2.33, 3.0), (0.9, 0.06, 0.6), fire)
    tube([(1.5, -1.5, 5.0), (3.0, -2.0, 5.0), (3.0, -2.0, 0.0)], 0.22, copper, n=8)


@asset("cabinet", "machine", ao=0.8)
def cabinet():
    """A control console; its readout faces -Y (the game turns it)."""
    steel = M("cb_steel", (88, 92, 102), var=0.05)
    dark = M("cb_dark", (58, 60, 68))
    hazard = M("cb_haz", (232, 168, 40))
    screen = M("cb_screen", (20, 26, 30))
    glow = M("cb_glow", (40, 210, 170), emit=1.2)
    gauge = M("cb_gauge", (230, 220, 200), emit=0.3)
    box((0, 0, 0.17), (1.9, 3.9, 0.35), dark)
    box((0, 0, 1.82), (1.44, 3.4, 2.95), steel, bev=0.05, seg=1)
    box((0, 0, 0.7), (1.46, 3.42, 0.3), hazard)
    box((0, 0, 3.42), (1.76, 3.76, 0.25), steel, bev=0.03, seg=1)
    box((0, -0.73, 2.2), (1.04, 0.06, 1.3), screen)
    box((0, -0.77, 2.2), (0.88, 0.03, 1.1), glow)
    for x in (-0.35, 0.35):
        place(cyl((0, 0, 0), 0.17, 0.06, gauge, n=10), loc=(x, -0.75, 1.15), rot=(90, 0, 0))
    for k in range(6):
        box((-1.3 + k * 0.52, -0.73, 3.0), (0.2, 0.05, 0.12), M("cb_btn" + str(k % 3), [(255, 80, 60), (90, 240, 120), (255, 210, 90)][k % 3], emit=0.8))


@asset("smokestack", "machine", ao=2.0)
def smokestack():
    brick = M("ss_brick", (150, 90, 60), var=0.12)
    base = M("ss_base", (70, 66, 70))
    rim = M("ss_rim", (40, 40, 44))
    lathe([(3.2, 0), (2.6, 2.0), (2.2, 20.0), (2.6, 21.0), (2.6, 22.0), (2.1, 22.2)], brick, n=12,
          mats=[base, brick, rim, rim, rim])
    for z in (6.0, 12.0, 18.0):
        r = 2.6 - (z - 2) / 18 * 0.4
        torus((0, 0, z), r + 0.02, 0.12, rim, seg=16, rseg=4)


@asset("vent", "machine", ao=0.6)
def vent():
    frame = M("vt_frame", (50, 52, 58))
    slat = M("vt_slat", (25, 25, 30))
    box((0, 0, 0.04), (1.2, 1.2, 0.08), frame)
    for k in (-0.3, 0, 0.3):
        box((0, k, 0.09), (1.0, 0.08, 0.02), slat)


@asset("gauge", "machine", ao=0.6)
def gauge():
    """A pressure gauge on a post; the dial (radius 1.2) faces -Y at z=4.8."""
    post = M("gg_post", (80, 82, 90))
    brass = M("gg_brass", (200, 150, 60))
    face = M("gg_face", (236, 228, 210), emit=0.2)
    red = M("gg_red", (200, 40, 40))
    cyl((0, 0, 0), 0.15, 4.0, post, n=6)
    place(cyl((0, 0, 0), 1.2, 0.2, brass, n=20), loc=(0, 0.1, 4.8), rot=(90, 0, 0))
    place(cyl((0, 0, 0), 1.08, 0.04, face, n=20), loc=(0, -0.02, 4.8), rot=(90, 0, 0))
    for k in range(9):
        a = math.radians(225 - k * 33.75)
        box((math.cos(a) * 0.92, -0.05, 4.8 + math.sin(a) * 0.92), (0.05, 0.02, 0.16), red if k > 6 else post, rot=(0, -math.degrees(a) + 90, 0))


@asset("pipe", "machine", ao=0.8)
def pipe():
    """A 48-long pipe along +Z with flanged joints (tinted in code)."""
    cu = M("pp_cu", (176, 110, 66), tint=True, var=0.04)
    prof = [(0.0, 0.0)]
    for k in range(0, 49, 6):
        prof += [(0.42, k), (0.55, k + 0.1), (0.55, k + 0.45), (0.42, k + 0.55)]
    prof.append((0.0, 48.0))
    lathe(prof, cu, n=10)
