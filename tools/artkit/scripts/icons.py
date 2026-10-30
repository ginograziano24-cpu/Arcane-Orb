"""Rendered UI icons: the map's room icons (task icons_map).

Each icon is a small 3D scene rendered front-on with a transparent
background to out/icons/<name>.png (512 x 512)."""
import os, math
import bpy
from mathutils import Vector
import kit
from models import M, box, cyl, sphere, lathe, prism, torus, tube, place

OUT = os.path.join(kit.OUT, "icons")


def _render(name, size=2.4):
    os.makedirs(OUT, exist_ok=True)
    sc = bpy.context.scene
    kit.setup_render(512, 512, samples=96)
    sc.render.film_transparent = True
    cam = kit.camera((0, -10, 0.9), (85, 0, 0), lens=50)
    cam.data.type = 'ORTHO'
    cam.data.ortho_scale = size
    kit.sun((40, -25, -30), strength=4.0, color=(1, 0.95, 0.88))
    kit.point_light((2.5, 2.0, 2.5), 900, color=(0.7, 0.8, 1.0), radius=1.0)
    kit.point_light((-2.5, 2.0, 1.5), 700, color=(1.0, 0.75, 0.6), radius=1.0)
    kit.point_light((0, -4, 0), 250, color=(1, 1, 1), radius=2.0)
    w = bpy.data.worlds.new("w")
    sc.world = w
    w.use_nodes = True
    bg = w.node_tree.nodes.get("Background")
    bg.inputs[0].default_value = (0.25, 0.25, 0.3, 1)
    bg.inputs[1].default_value = 0.6
    kit.render_to(os.path.join(OUT, name + ".png"))


def _blade(x, rot, steel, gold, leather):
    """A sword lying in the XZ plane (faces -Y), centred, rotated by rot degrees."""
    objs = []
    objs.append(prism([(-0.07, -0.55), (0.07, -0.55), (0.07, 0.75), (0.0, 0.95), (-0.07, 0.75)], 0.05, steel, z0=-0.025))
    objs.append(prism([(-0.012, -0.5), (0.012, -0.5), (0.012, 0.8), (-0.012, 0.8)], 0.06, M("fuller", (150, 160, 180), metal=1, rough=0.2), z0=-0.03))
    objs.append(box((0, 0, 0), (0.42, 0.09, 0.07), gold, bev=0.02, seg=1))
    objs.append(box((0, 0, 0), (0.07, 0.07, 0.3), leather))
    objs.append(sphere((0, 0, 0), 0.06, gold, sub=2))
    # prisms are built in XY; stand them up into XZ
    for o in objs[:2]:
        o.rotation_euler = (math.radians(90), 0, 0)
    objs[2].location = (0, 0, -0.55)
    objs[3].location = (0, 0, -0.75)
    objs[4].location = (0, 0, -0.93)
    for o in objs:
        p = o.location.copy()
        r = math.radians(rot)
        o.location = (x + p.x * math.cos(r) - p.z * math.sin(r), p.y, p.x * math.sin(r) + p.z * math.cos(r))
        o.rotation_euler = (o.rotation_euler.x, -r, 0)


def icon_combat():
    steel = M("steel", (200, 208, 220), metal=1.0, rough=0.25)
    gold = M("gold", (255, 200, 90), metal=1.0, rough=0.3)
    leather = M("leather", (110, 60, 40), rough=0.7)
    _blade(0, 40, steel, gold, leather)
    _blade(0, -40, steel, gold, leather)
    _render("node_combat")


def icon_elite():
    bone = M("bone", (236, 226, 200), rough=0.5)
    dark = M("socket", (20, 10, 20))
    glow = M("eyeglow", (255, 80, 60), emit=4.0)
    gold = M("gold", (255, 200, 90), metal=1.0, rough=0.3)
    gem = M("gem", (255, 60, 90), emit=1.5, rough=0.1)
    sphere((0, 0, 0.1), 0.62, bone, sub=3, scale=(1, 0.9, 1))
    box((0, -0.05, -0.42), (0.6, 0.6, 0.3), bone, bev=0.08, seg=2)
    for x in (-0.22, 0.22):
        sphere((x, -0.48, 0.05), 0.16, dark, sub=2)
        sphere((x, -0.6, 0.05), 0.06, glow, sub=1)
    o = prism([(-0.06, 0.0), (0.06, 0.0), (0.0, 0.14)], 0.1, dark, z0=-0.05)
    o.rotation_euler = (math.radians(90), 0, 0)
    o.location = (0, -0.5, -0.15)
    for x in (-0.2, -0.07, 0.07, 0.2):
        box((x, -0.36, -0.42), (0.09, 0.05, 0.12), bone)
    for sx in (-1, 1):
        tube([(sx * 0.5, 0, 0.35), (sx * 0.85, 0, 0.55), (sx * 0.95, 0, 0.95)], 0.12, bone, n=8, r_end=0.02)
    lathe([(0.0, 0.62), (0.5, 0.62), (0.5, 0.78), (0.0, 0.78)], gold, n=24)
    for k in range(5):
        a = math.radians(-60 + k * 30)
        x, y = math.sin(a) * 0.5, -math.cos(a) * 0.5
        o = prism([(-0.08, 0), (0.08, 0), (0, 0.28)], 0.05, gold, z0=-0.025)
        o.rotation_euler = (math.radians(90), 0, -a)
        o.location = (x, y, 0.78)
    sphere((0, -0.52, 0.72), 0.07, gem, sub=2)
    _render("node_elite", size=2.6)


def icon_shop():
    sack = M("sack", (150, 100, 60), rough=0.85)
    rope = M("rope", (220, 180, 110), rough=0.7)
    gold = M("coin", (255, 205, 80), metal=1.0, rough=0.3)
    lathe([(0.0, -0.75), (0.45, -0.7), (0.62, -0.4), (0.6, -0.05), (0.38, 0.2), (0.16, 0.32), (0.22, 0.45), (0.3, 0.6), (0.0, 0.58)], sack, n=20)
    torus((0, 0, 0.3), 0.18, 0.05, rope, seg=20, rseg=6)
    for i, (x, z) in enumerate([(-0.55, -0.72), (-0.35, -0.75), (0.5, -0.7), (0.62, -0.62)]):
        o = cyl((0, 0, 0), 0.17, 0.05, gold, n=20)
        o.location = (x, -0.35, z)
        o.rotation_euler = (math.radians(60 + i * 10), math.radians(i * 20), 0)
    cyl((0, -0.1, -0.05), 0.2, 0.04, gold, n=20).rotation_euler = (math.radians(90), 0, 0)
    _render("node_shop")


def icon_treasure():
    wood = M("wood", (130, 76, 40), rough=0.7)
    gold = M("gold", (255, 200, 90), metal=1.0, rough=0.3)
    loot = M("loot", (255, 215, 90), emit=1.5, metal=1.0, rough=0.2)
    gem = M("gem", (90, 220, 255), emit=2.0, rough=0.05)
    box((0, 0, -0.35), (1.3, 0.8, 0.6), wood, bev=0.04, seg=1)
    for x in (-0.55, 0.55):
        box((x, 0, -0.35), (0.1, 0.82, 0.62), gold)
    box((0, -0.41, -0.25), (0.18, 0.04, 0.22), gold)
    o = cyl((0, 0, 0), 0.4, 1.3, wood, n=16)
    o.rotation_euler = (0, math.radians(90), 0)
    o.location = (0, 0.32, 0.25)
    o.rotation_euler = (math.radians(-55), math.radians(90), 0)
    sphere((0, -0.05, -0.02), 0.5, loot, sub=2, scale=(1.1, 0.6, 0.3))
    sphere((0.2, -0.3, 0.12), 0.1, gem, sub=1)
    sphere((-0.25, -0.25, 0.1), 0.08, M("ruby", (255, 60, 90), emit=1.8), sub=1)
    _render("node_treasure")


def icon_event():
    glass = M("orbglass", (170, 140, 255), emit=1.2, rough=0.05)
    swirl = M("swirl", (255, 255, 255), emit=4.0)
    brass = M("brass", (200, 160, 80), metal=1.0, rough=0.3)
    sphere((0, 0, 0.2), 0.6, glass, sub=3)
    pts = []
    for k in range(30):
        a = k / 30 * 4 * math.pi
        r = 0.08 + k / 30 * 0.38
        pts.append((math.cos(a) * r, -0.62, 0.2 + math.sin(a) * r))
    tube(pts, 0.03, swirl, n=6)
    lathe([(0.0, -0.75), (0.55, -0.75), (0.45, -0.6), (0.32, -0.45), (0.38, -0.36), (0.0, -0.36)], brass, n=24)
    _render("node_event")


def icon_rest():
    log_m = M("log", (110, 70, 40), rough=0.8)
    stone = M("stone", (120, 120, 130), rough=0.8)
    f1 = M("flame1", (255, 110, 20), emit=2.2)
    f2 = M("flame2", (255, 200, 70), emit=3.0)
    for a in (25, -25, 90):
        o = cyl((0, 0, 0), 0.1, 1.2, log_m, n=10)
        o.location = (0, 0, -0.55)
        o.rotation_euler = (math.radians(90 if a == 90 else 0), math.radians(90 if a != 90 else 0), math.radians(a if a != 90 else 0))
    for k in range(8):
        a = k / 8 * 2 * math.pi
        sphere((math.cos(a) * 0.7, math.sin(a) * 0.3, -0.68), 0.13, stone, sub=1, jitter=0.2, seed=k)
    lathe([(0.0, -0.55), (0.32, -0.45), (0.36, -0.2), (0.22, 0.15), (0.08, 0.45), (0.0, 0.62)], f1, n=12)
    for x, h in ((-0.25, 0.25), (0.27, 0.32)):
        o = lathe([(0.0, -0.5), (0.12, -0.4), (0.1, -0.15), (0.0, h - 0.1)], f1, n=8)
        o.location = (x, 0, 0)
    lathe([(0.0, -0.5), (0.18, -0.42), (0.18, -0.25), (0.1, 0.0), (0.0, 0.22)], f2, n=10)
    _render("node_rest")


def icon_boss():
    face = M("mask", (150, 30, 40), rough=0.4, metal=0.3)
    dark = M("maskdark", (40, 8, 14))
    horn = M("horn", (230, 215, 180), rough=0.5)
    glow = M("bosseye", (255, 210, 80), emit=6.0)
    teeth = M("teeth", (245, 240, 220))
    sphere((0, 0.1, 0), 0.7, face, sub=3, scale=(0.95, 0.6, 1.05))
    for x in (-0.25, 0.25):
        o = prism([(-0.16, -0.05), (0.16, 0.06), (0.1, -0.1), (-0.12, -0.12)], 0.1, dark, z0=-0.05)
        o.rotation_euler = (math.radians(90), 0, math.radians(180 if x > 0 else 0))
        o.location = (x, -0.35, 0.18)
        sphere((x, -0.42, 0.14), 0.07, glow, sub=1)
    box((0, -0.36, -0.38), (0.6, 0.1, 0.18), dark)
    for k in range(6):
        x = -0.25 + k * 0.1
        o = prism([(-0.035, 0), (0.035, 0), (0, -0.1)], 0.04, teeth, z0=-0.02)
        o.rotation_euler = (math.radians(90), 0, 0)
        o.location = (x, -0.42, -0.3)
    for sx in (-1, 1):
        tube([(sx * 0.45, 0, 0.45), (sx * 0.85, 0, 0.75), (sx * 0.8, 0, 1.15), (sx * 0.6, 0, 1.3)], 0.13, horn, n=10, r_end=0.02)
    _render("node_boss", size=2.9)


ICONS = {"node_combat": icon_combat, "node_elite": icon_elite, "node_shop": icon_shop, "node_treasure": icon_treasure,
         "node_event": icon_event, "node_rest": icon_rest, "node_boss": icon_boss}


def build_icons():
    for n, fn in ICONS.items():
        try:
            kit.reset()
            fn()
            kit.log("icon", n)
        except Exception:
            import traceback
            kit.log("icon FAILED", n)
            traceback.print_exc()


TASKS = {"icons_map": build_icons}


# ------------------------------------------------------------------ shop stall
def _sx(px):  # screen px (0..1920) -> scene x
    return (px - 960) / 100.0


def _sz(py):  # screen px (0..1080) -> scene z
    return (540 - py) / 100.0


def shop_stall():
    """A market-stall frame drawn over the shop screen: striped awning with
    lanterns, two posts, a counter, shelves of potions and a hooded merchant
    in the lower right. Rendered 1920x1080 with a transparent background."""
    kit.reset()
    wood = M("st_wood", (120, 74, 44), rough=0.75, var=0.08)
    wood_d = M("st_wood_d", (80, 48, 30), rough=0.8)
    red = M("st_red", (170, 40, 50), rough=0.7)
    cream = M("st_cream", (236, 222, 190), rough=0.7)
    brass = M("st_brass", (210, 170, 80), metal=1.0, rough=0.3)
    lamp = M("st_lamp", (255, 190, 90), emit=6.0)
    cloak = M("st_cloak", (50, 40, 80), rough=0.9)
    hood_in = M("st_hood", (6, 4, 10), rough=1)
    eye = M("st_eye", (255, 220, 120), emit=8.0)
    # awning: alternating stripes with a scalloped hem
    y0, y1 = _sz(0), _sz(120)
    n = 24
    x0, x1 = _sx(200), _sx(1720)
    sw = (x1 - x0) / n
    for i in range(n):
        m = red if i % 2 == 0 else cream
        cx = x0 + sw * (i + 0.5)
        box((cx, 0.6, (y0 + y1) / 2 + 0.1), (sw, 0.05, y0 - y1 + 0.2), m)
        o = cyl((0, 0, 0), sw / 2, 0.05, m, n=12)
        o.rotation_euler = (math.radians(90), 0, 0)
        o.location = (cx, 0.6, y1)
    box(((x0 + x1) / 2, 0.55, y0 - 0.05), (x1 - x0 + 0.4, 0.2, 0.25), wood_d)
    # posts
    for px in (205, 1715):
        box((_sx(px), 0.5, 0), (0.28, 0.3, 10.8), wood, bev=0.03, seg=1)
        box((_sx(px), 0.4, _sz(1000)), (0.4, 0.4, 0.2), brass)
    # counter along the bottom
    box(((x0 + x1) / 2, 0.3, _sz(1055)), (x1 - x0 + 0.3, 0.6, 0.5), wood, bev=0.03, seg=1)
    box(((x0 + x1) / 2, 0.0, _sz(1030)), (x1 - x0 + 0.4, 0.7, 0.08), wood_d)
    # lanterns hanging from the awning
    for px in (330, 960, 1590):
        x = _sx(px)
        box((x, 0.4, _sz(150)), (0.02, 0.02, 0.5), brass)
        lathe([(0.0, -0.25), (0.18, -0.2), (0.2, 0.05), (0.12, 0.2), (0.0, 0.22)], lamp, n=10, c=(x, 0.3, _sz(220)))
        cyl((x, 0.3, _sz(200)), 0.12, 0.06, brass, n=10)
        kit.point_light((x, -0.5, _sz(220)), 60, color=(1.0, 0.75, 0.4), radius=0.2)
    # potion shelf at lower left, behind the counter edge
    for k in range(5):
        x = _sx(250 + k * 34)
        h = 0.35 + 0.12 * (k % 3)
        col = [(90, 220, 140), (220, 90, 200), (90, 160, 255), (255, 180, 60), (255, 90, 90)][k]
        lathe([(0.0, 0), (0.12, 0.02), (0.13, h * 0.6), (0.05, h * 0.8), (0.05, h), (0.0, h)],
              M("potion%d" % k, col, emit=1.5, rough=0.1), n=10, c=(x, 0.2, _sz(1030)))
    # the merchant, hooded, behind the counter at the lower right
    mx = _sx(1610)
    lathe([(0.0, 0.0), (0.75, 0.0), (0.7, 0.8), (0.55, 1.6), (0.42, 2.0), (0.0, 2.1)], cloak, n=20, c=(mx, 0.9, _sz(1080)))
    sphere((mx, 0.8, _sz(1080) + 2.35), 0.45, cloak, sub=3, scale=(1, 1, 1.15))
    sphere((mx, 0.55, _sz(1080) + 2.3), 0.32, hood_in, sub=2, scale=(1, 0.6, 1.1))
    for dx in (-0.11, 0.11):
        sphere((mx + dx, 0.3, _sz(1080) + 2.35), 0.04, eye, sub=1)
    # a few coins on the counter
    for k, px in enumerate((1180, 1215, 1250)):
        o = cyl((0, 0, 0), 0.12, 0.03, brass, n=14)
        o.location = (_sx(px), 0.0, _sz(1003) + k * 0.035)
    sc = bpy.context.scene
    kit.setup_render(1920, 1080, samples=96)
    sc.render.film_transparent = True
    cam = kit.camera((0, -20, 0), (90, 0, 0), lens=50)
    cam.data.type = 'ORTHO'
    cam.data.ortho_scale = 19.2
    kit.sun((50, 20, -20), strength=2.2, color=(1, 0.9, 0.8))
    kit.point_light((0, -6, 3), 900, color=(1, 0.85, 0.7), radius=3)
    w = bpy.data.worlds.new("w")
    sc.world = w
    w.use_nodes = True
    bg = w.node_tree.nodes.get("Background")
    bg.inputs[0].default_value = (0.2, 0.18, 0.25, 1)
    bg.inputs[1].default_value = 0.4
    os.makedirs(OUT, exist_ok=True)
    kit.render_to(os.path.join(OUT, "shop_stall.png"))


TASKS["shop_stall"] = shop_stall
