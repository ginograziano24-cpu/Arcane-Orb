"""Rendered UI kit (task ui_kit): nine-slice panel frames per theme, a bar
frame, and HUD icons, all real 3D lit in Cycles and rendered top-down with
a transparent background to out/ui/.

Frames are 4 x 4 units with a 0.42 border and ornate corners within 0.8
units of each corner, rendered at 512 x 512 (128 px per unit): the game
slices them 104 px from each edge."""
import os, math
import bpy
import kit
from kit import log, setin
from models import M, box, cyl, sphere, lathe, prism, torus, tube, place

OUT = os.path.join(kit.OUT, "ui")


def _setup(w, h, ortho):
    kit.reset()
    kit.setup_render(w, h, samples=128)
    sc = bpy.context.scene
    sc.render.film_transparent = True
    try:
        sc.view_settings.view_transform = 'Standard'
        sc.view_settings.look = 'None'
    except Exception:
        pass
    cam = kit.camera((0, 0, 20), (0, 0, 0), lens=50)
    cam.data.type = 'ORTHO'
    cam.data.ortho_scale = ortho
    wd = bpy.data.worlds.new("w")
    sc.world = wd
    wd.use_nodes = True
    bg = wd.node_tree.nodes.get("Background")
    bg.inputs[0].default_value = (0.05, 0.05, 0.06, 1)
    bg.inputs[1].default_value = 1.0

    def area(loc, rot, size, energy, color=(1, 1, 1), sy=None):
        d = bpy.data.lights.new("a", 'AREA')
        d.energy = energy
        d.color = color
        d.shape = 'RECTANGLE'
        d.size = size
        d.size_y = sy if sy else size
        o = bpy.data.objects.new("a", d)
        sc.collection.objects.link(o)
        o.location = loc
        o.rotation_euler = [math.radians(a) for a in rot]
    area((-4, 4, 7), (-30, -30, 0), 5, 900, (1, 0.96, 0.9))
    area((5, -5, 5), (40, 40, 0), 6, 300, (0.8, 0.88, 1.0))
    area((0, 0, 9), (0, 0, 35), 0.4, 120, (1, 1, 1), sy=9)


def _mat(name, rgb, rough=0.3, metal=0.0, emit=None, emit_str=0.0, coat=0.0):
    m, nt, out = kit.new_mat(name)
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    nt.links.new(p.outputs[0], out.inputs[0])
    setin(p, (*rgb, 1), 'Base Color')
    setin(p, rough, 'Roughness')
    setin(p, metal, 'Metallic')
    setin(p, coat, 'Coat Weight', 'Clearcoat')
    if emit:
        setin(p, (*emit, 1), 'Emission Color', 'Emission')
        setin(p, emit_str, 'Emission Strength')
    return m


def _bar(mat, x0, y0, x1, y1, h, bev=0.06, z=0.0):
    bpy.ops.mesh.primitive_cube_add(size=1, location=((x0 + x1) / 2, (y0 + y1) / 2, z + h / 2))
    o = bpy.context.active_object
    o.scale = (abs(x1 - x0), abs(y1 - y0), h)
    bpy.ops.object.transform_apply(scale=True)
    o.data.materials.append(mat)
    b = o.modifiers.new("b", 'BEVEL')
    b.width = bev
    b.segments = 4
    bpy.ops.object.shade_smooth()
    return o


def _ring(mat, half, border, h, bev=0.06, z=0.0, inset=0.0):
    """A rectangular frame: four bars around a square of half-size `half`."""
    a, b = half - inset, half - inset - border
    _bar(mat, -a, b, a, a, h, bev, z)
    _bar(mat, -a, -a, a, -b, h, bev, z)
    _bar(mat, -a, -b, -b, b, h, bev, z)
    _bar(mat, b, -b, a, b, h, bev, z)


def _gem(mat, x, y, r, z, sub=1):
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=sub, radius=r, location=(x, y, z))
    o = bpy.context.active_object
    o.scale = (1, 1, 0.55)
    o.data.materials.append(mat)
    return o


def _corners(fn):
    for sx in (-1, 1):
        for sy in (-1, 1):
            fn(sx, sy)


def frame_arcane():
    gold = _mat("gold", (1.0, 0.76, 0.32), 0.18, 1.0, coat=0.3)
    obs = _mat("obs", (0.05, 0.03, 0.08), 0.12, coat=1.0)
    gem = _mat("vio", (0.6, 0.35, 1.0), 0.02, emit=(0.6, 0.35, 1.0), emit_str=1.5, coat=1.0)
    _ring(obs, 2.0, 0.4, 0.16, 0.05)
    _ring(gold, 2.0, 0.07, 0.22, 0.03, inset=0.0)
    _ring(gold, 2.0, 0.04, 0.2, 0.02, inset=0.3)

    def c(sx, sy):
        x, y = sx * 1.7, sy * 1.7
        _bar(gold, x - 0.24, y - 0.24, x + 0.24, y + 0.24, 0.26, 0.08).rotation_euler = (0, 0, math.radians(45))
        _gem(gem, x, y, 0.17, 0.3)
    _corners(c)


def frame_holo():
    chrome = _mat("chr", (0.12, 0.1, 0.18), 0.12, 0.8, coat=1.0)
    pink = _mat("pink", (1.0, 0.3, 0.85), 0.2, emit=(1.0, 0.25, 0.8), emit_str=1.1)
    cyan = _mat("cyan", (0.4, 0.9, 1.0), 0.2, emit=(0.3, 0.85, 1.0), emit_str=1.1)
    _ring(chrome, 2.0, 0.38, 0.14, 0.04)
    _ring(pink, 2.0, 0.04, 0.17, 0.01, inset=0.12)

    def c(sx, sy):
        x, y = sx * 1.7, sy * 1.7
        _bar(chrome, x - sx * 0.3 - 0.25, y - 0.08, x - sx * 0.3 + 0.25, y + 0.08, 0.2, 0.03)
        _bar(chrome, x - 0.08, y - sy * 0.3 - 0.25, x + 0.08, y - sy * 0.3 + 0.25, 0.2, 0.03)
        _bar(cyan, x - 0.12, y - 0.12, x + 0.12, y + 0.12, 0.24, 0.03)
    _corners(c)


def frame_brass():
    brass = _mat("brass", (0.95, 0.7, 0.35), 0.22, 1.0, coat=0.4)
    wood = _mat("wood", (0.3, 0.17, 0.08), 0.5)
    leaf = _mat("leaf", (0.4, 0.75, 0.25), 0.35, coat=0.5)
    _ring(wood, 2.0, 0.38, 0.12, 0.04)
    _ring(brass, 2.0, 0.08, 0.18, 0.03)
    _ring(brass, 2.0, 0.04, 0.16, 0.02, inset=0.28)

    def c(sx, sy):
        x, y = sx * 1.7, sy * 1.7
        o = cyl((x, y, 0.1), 0.24, 0.14, brass, n=24)
        for k in range(5):
            a = math.radians(k * 72 + 18)
            _gem(leaf, x + math.cos(a) * 0.22, y + math.sin(a) * 0.22, 0.11, 0.24)
        _gem(brass, x, y, 0.1, 0.28, sub=2)
    _corners(c)


def frame_crystal():
    rock = _mat("rock", (0.08, 0.1, 0.18), 0.4, coat=0.5)
    edge = _mat("edge", (0.55, 0.85, 1.0), 0.05, emit=(0.35, 0.75, 1.0), emit_str=0.8, coat=1.0)
    shard = _mat("shard", (0.7, 0.5, 1.0), 0.02, emit=(0.65, 0.4, 1.0), emit_str=0.9, coat=1.0)
    _ring(rock, 2.0, 0.38, 0.14, 0.06)
    _ring(edge, 2.0, 0.035, 0.17, 0.01, inset=0.06)

    def c(sx, sy):
        x, y = sx * 1.68, sy * 1.68
        for k, (dx, dy, h) in enumerate(((0, 0, 0.55), (-0.18, 0.1, 0.35), (0.14, -0.16, 0.4))):
            verts = [(x + dx + 0.09, y + dy, 0.1), (x + dx, y + dy + 0.09, 0.1), (x + dx - 0.09, y + dy, 0.1), (x + dx, y + dy - 0.09, 0.1), (x + dx, y + dy, 0.1 + h)]
            faces = [(0, 1, 4), (1, 2, 4), (2, 3, 4), (3, 0, 4), (3, 2, 1, 0)]
            me = bpy.data.meshes.new("s")
            me.from_pydata(verts, [], faces)
            o = bpy.data.objects.new("s", me)
            bpy.context.scene.collection.objects.link(o)
            o.data.materials.append(shard if k == 0 else edge)
    _corners(c)


def frame_stone():
    sand = _mat("sand", (0.8, 0.66, 0.45), 0.6)
    gold = _mat("gold", (1.0, 0.75, 0.3), 0.2, 1.0)
    turq = _mat("turq", (0.15, 0.75, 0.7), 0.15, coat=1.0)
    _ring(sand, 2.0, 0.4, 0.14, 0.05)
    _ring(gold, 2.0, 0.035, 0.17, 0.01, inset=0.08)
    _ring(gold, 2.0, 0.03, 0.16, 0.01, inset=0.26)

    def c(sx, sy):
        x, y = sx * 1.7, sy * 1.7
        _bar(sand, x - 0.3, y - 0.3, x + 0.3, y + 0.3, 0.2, 0.04)
        _bar(gold, x - 0.2, y - 0.2, x + 0.2, y + 0.2, 0.23, 0.02).rotation_euler = (0, 0, math.radians(45))
        _gem(turq, x, y, 0.13, 0.27)
    _corners(c)


def frame_cloud():
    marble = _mat("marble", (0.94, 0.94, 0.97), 0.15, coat=1.0)
    gold = _mat("gold", (1.0, 0.8, 0.4), 0.15, 1.0)
    blue = _mat("blue", (0.35, 0.6, 1.0), 0.02, coat=1.0, emit=(0.3, 0.55, 1.0), emit_str=0.6)
    _ring(marble, 2.0, 0.38, 0.14, 0.08)
    _ring(gold, 2.0, 0.04, 0.17, 0.015, inset=0.04)
    _ring(gold, 2.0, 0.03, 0.16, 0.01, inset=0.24)

    def c(sx, sy):
        x, y = sx * 1.7, sy * 1.7
        for k in range(6):
            a = math.radians(k * 60)
            _gem(marble, x + math.cos(a) * 0.2, y + math.sin(a) * 0.2, 0.13, 0.18, sub=2)
        _gem(gold, x, y, 0.15, 0.26, sub=2)
        _gem(blue, x, y, 0.1, 0.32)
    _corners(c)


def frame_metal():
    steel = _mat("steel", (0.55, 0.57, 0.62), 0.3, 1.0, coat=0.3)
    dark = _mat("dark", (0.08, 0.08, 0.09), 0.5)
    amber = _mat("amber", (1.0, 0.6, 0.15), 0.2, emit=(1.0, 0.55, 0.1), emit_str=3.0)
    haz = _mat("haz", (0.95, 0.7, 0.1), 0.35)
    _ring(steel, 2.0, 0.38, 0.14, 0.03)
    _ring(dark, 2.0, 0.03, 0.155, 0.005, inset=0.28)

    def c(sx, sy):
        x, y = sx * 1.72, sy * 1.72
        _bar(haz, x - 0.28, y - 0.28, x + 0.28, y + 0.28, 0.18, 0.03)
        for dx in (-0.2, 0.2):
            for dy in (-0.2, 0.2):
                _gem(steel, x + dx, y + dy, 0.035, 0.2, sub=2)
        _bar(dark, x - 0.16, y - 0.16, x + 0.16, y + 0.16, 0.2, 0.02)
        _gem(amber, x, y, 0.1, 0.24)
    _corners(c)


FRAMES = [("arcane", frame_arcane), ("holo", frame_holo), ("brass", frame_brass), ("crystal", frame_crystal),
          ("stone", frame_stone), ("cloud", frame_cloud), ("metal", frame_metal)]


def bar_frame():
    """A capsule-shaped bar well frame, 8 x 1 units, rendered 1024 x 128."""
    _setup(1024, 128, 8.0)
    gold = _mat("gold", (1.0, 0.78, 0.38), 0.18, 1.0, coat=0.3)
    dark = _mat("dark", (0.03, 0.02, 0.05), 0.2, coat=1.0)
    for (y0, y1) in ((0.3, 0.42), (-0.42, -0.3)):
        _bar(gold, -3.6, y0, 3.6, y1, 0.16, 0.05)
    for sx in (-1, 1):
        o = cyl((sx * 3.6, 0, 0), 0.42, 0.16, gold, n=32)
        cyl((sx * 3.6, 0, 0.01), 0.3, 0.17, dark, n=32)
        _gem(_mat("ruby", (1, 0.2, 0.35), 0.02, emit=(1, 0.2, 0.35), emit_str=1.2, coat=1.0), sx * 3.6, 0, 0.16, 0.22)
    kit.render_to(os.path.join(OUT, "bar.png"))


def icons():
    def one(name, build, size=2.4):
        _setup(256, 256, size)
        build()
        kit.render_to(os.path.join(OUT, "ic_%s.png" % name))

    def coin():
        g = _mat("g", (1.0, 0.77, 0.25), 0.15, 1.0, coat=0.5)
        cyl((0, 0, 0), 0.95, 0.22, g, n=48)
        torus((0, 0, 0.22), 0.8, 0.06, g, seg=48, rseg=8)
        # a stamped star, faintly lit so it never reads as a dark hole
        st = _mat("s", (1.0, 0.82, 0.38), 0.25, 0.85, emit=(1.0, 0.7, 0.25), emit_str=0.35)
        pts = []
        for k in range(10):
            a = math.pi / 2 + k * math.pi / 5
            r = 0.52 if k % 2 == 0 else 0.22
            pts.append((math.cos(a) * r, math.sin(a) * r))
        o = prism(pts[::-1], 0.3, st)
        m = o.modifiers.new("b", 'BEVEL')
        m.width = 0.04
        m.segments = 3

    def heart():
        r = _mat("r", (0.95, 0.12, 0.25), 0.12, coat=1.0, emit=(0.9, 0.05, 0.2), emit_str=0.2)
        pts = []
        for k in range(48):
            t = k / 48 * 2 * math.pi
            x = 16 * math.sin(t) ** 3
            y = 13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t)
            pts.append((x / 17, y / 17 + 0.1))
        o = prism(pts[::-1], 0.3, r)
        bev = o.modifiers.new("b", 'BEVEL')
        bev.width = 0.12
        bev.segments = 6
        for f in o.data.polygons:
            f.use_smooth = True

    def shield():
        b = _mat("b", (0.3, 0.45, 0.95), 0.15, coat=1.0)
        g = _mat("g", (1.0, 0.8, 0.4), 0.15, 1.0)
        pts = [(-0.85, 0.85), (0.85, 0.85), (0.8, -0.1), (0, -1.0), (-0.8, -0.1)]
        o = prism(pts[::-1], 0.25, g)
        o2 = prism([(x * 0.8, y * 0.8 + 0.02) for x, y in pts][::-1], 0.32, b)
        for ob in (o, o2):
            m = ob.modifiers.new("b", 'BEVEL')
            m.width = 0.05
            m.segments = 4

    def gem():
        p = _mat("p", (0.7, 0.4, 1.0), 0.02, coat=1.0, emit=(0.6, 0.3, 1.0), emit_str=0.6)
        bpy.ops.mesh.primitive_cone_add(vertices=8, radius1=0.9, radius2=0.5, depth=0.35, location=(0, 0, 0.35))
        o = bpy.context.active_object
        o.data.materials.append(p)
        bpy.ops.mesh.primitive_cone_add(vertices=8, radius1=0.9, radius2=0.0, depth=0.7, location=(0, 0, -0.17))
        o = bpy.context.active_object
        o.rotation_euler = (math.pi, 0, 0)
        o.data.materials.append(p)

    for n, f in (("coin", coin), ("heart", heart), ("shield", shield), ("gem", gem)):
        one(n, f)


def ui_frames():
    os.makedirs(OUT, exist_ok=True)
    for name, fn in FRAMES:
        _setup(512, 512, 4.0)
        fn()
        kit.render_to(os.path.join(OUT, "frame_%s.png" % name))


def ui_kit():
    os.makedirs(OUT, exist_ok=True)
    for name, fn in FRAMES:
        _setup(512, 512, 4.0)
        fn()
        kit.render_to(os.path.join(OUT, "frame_%s.png" % name))
    bar_frame()
    icons()
    log("ui kit done")


TASKS = {"ui_kit": ui_kit, "ui_frames": ui_frames, "ui_icons": icons}
