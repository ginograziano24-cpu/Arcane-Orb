"""Set pieces behind the between-chamber screens (task scenes): a
campfire glade for rests, a merchant's tent for the shop, a rune shrine
for events and a treasure vault for relic rooms. Each is rendered
1920 x 1080 to out/scenes/<name>.jpg; the screens draw them full-bleed
and lay their UI on top, so the focus sits low and in the middle and the
edges fall into shadow."""
import os, math, random
import bpy, bmesh
from mathutils import Vector
import kit
from kit import setin

OUT = os.path.join(kit.OUT, "scenes")


def V(*a):
    return Vector(a)


# ------------------------------------------------------------------ helpers

def mat(name, base, rough=0.6, metal=0.0, emit=None, es=0.0, bump=0.0, bscale=20.0, var=0.0, coat=0.0, trans=0.0):
    m, nt, out = kit.new_mat(name)
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    setin(p, (*base, 1), 'Base Color')
    setin(p, rough, 'Roughness')
    setin(p, metal, 'Metallic')
    setin(p, coat, 'Coat Weight', 'Clearcoat')
    setin(p, trans, 'Transmission Weight', 'Transmission')
    if emit is not None:
        setin(p, (*emit, 1), 'Emission Color', 'Emission')
        setin(p, es, 'Emission Strength')
    co = nt.nodes.new('ShaderNodeTexCoord').outputs['Object']
    if var > 0:
        n = kit.noise_tex(nt, 3.0, 6, 0.55, co)
        r = kit.ramp(nt, [(0.3, tuple(c * (1 - var) for c in base)), (0.7, tuple(min(1, c * (1 + var * 0.4)) for c in base))])
        nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
        nt.links.new(r.outputs['Color'], p.inputs['Base Color'])
    if bump > 0:
        n = kit.noise_tex(nt, bscale, 10, 0.6, co)
        b = kit.bump(nt, n.outputs['Fac'], bump, 0.05)
        nt.links.new(b.outputs['Normal'], p.inputs['Normal'])
    return m


def glow(name, rgb, s):
    return kit.mat_emit(name, rgb, s)


def obj(name, bm, m, smooth=True):
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    if m is not None:
        o.data.materials.append(m)
    for f in o.data.polygons:
        f.use_smooth = smooth
    return o


def box(loc, size, m, rot=(0, 0, 0), bev=0.0):
    bm = bmesh.new()
    bmesh.ops.create_cube(bm, size=1.0)
    for v in bm.verts:
        v.co = Vector((v.co.x * size[0], v.co.y * size[1], v.co.z * size[2]))
    o = obj("box", bm, m, smooth=False)
    o.location = loc
    o.rotation_euler = [math.radians(a) for a in rot]
    if bev:
        b = o.modifiers.new("b", 'BEVEL')
        b.width = bev
        b.segments = 2
    return o


def cyl(a, b, r, m, n=16, r2=None):
    a, b = Vector(a), Vector(b)
    d = b - a
    bm = bmesh.new()
    bmesh.ops.create_cone(bm, cap_ends=True, segments=n, radius1=r, radius2=r if r2 is None else r2, depth=d.length)
    o = obj("cyl", bm, m)
    o.location = (a + b) / 2
    o.rotation_euler = d.to_track_quat('Z', 'Y').to_euler()
    return o


def ball(loc, r, m, scale=(1, 1, 1), sub=2, jitter=0.0, seed=0, smooth=True):
    bm = bmesh.new()
    bmesh.ops.create_icosphere(bm, subdivisions=sub, radius=r)
    rnd = random.Random(seed)
    for v in bm.verts:
        k = 1 + rnd.uniform(-jitter, jitter)
        v.co = Vector((v.co.x * scale[0] * k, v.co.y * scale[1] * k, v.co.z * scale[2] * k))
    o = obj("ball", bm, m, smooth=smooth)
    o.location = loc
    return o


def ground(m, size=60, z=0.0, disp=0.0):
    bm = bmesh.new()
    bmesh.ops.create_grid(bm, x_segments=80, y_segments=80, size=size / 2)
    rnd = random.Random(1)
    if disp:
        for v in bm.verts:
            d = math.hypot(v.co.x, v.co.y)
            v.co.z = disp * (math.sin(v.co.x * 0.3) * math.cos(v.co.y * 0.25) + rnd.uniform(-0.3, 0.3)) * min(1, d / 8)
    o = obj("ground", bm, m)
    o.location = (0, 0, z)
    return o


def pt(loc, power, color, radius=0.2):
    d = bpy.data.lights.new("pt", 'POINT')
    d.energy = power
    d.color = color
    d.shadow_soft_size = radius
    o = bpy.data.objects.new("pt", d)
    bpy.context.scene.collection.objects.link(o)
    o.location = loc
    return o


def area(loc, target, size, power, color, sy=None):
    d = bpy.data.lights.new("ar", 'AREA')
    d.energy = power
    d.color = color
    d.size = size
    if sy:
        d.shape = 'RECTANGLE'
        d.size_y = sy
    o = bpy.data.objects.new("ar", d)
    bpy.context.scene.collection.objects.link(o)
    o.location = loc
    o.rotation_euler = (Vector(target) - Vector(loc)).to_track_quat('-Z', 'Y').to_euler()
    return o


def world(top, horizon, strength=1.0, stars=0.0):
    w = bpy.data.worlds.new("w")
    bpy.context.scene.world = w
    w.use_nodes = True
    nt = w.node_tree
    bg = nt.nodes.get("Background")
    tc = nt.nodes.new('ShaderNodeTexCoord')
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(tc.outputs['Generated'], sep.inputs[0])
    r = kit.ramp(nt, [(0.0, horizon), (0.35, top), (1.0, tuple(c * 0.5 for c in top))])
    nt.links.new(sep.outputs['Z'], r.inputs['Fac'])
    col = r.outputs['Color']
    if stars > 0:
        v = kit.voronoi_tex(nt, 380.0, tc.outputs['Generated'])
        st = kit.ramp(nt, [(0.0, (1, 1, 1)), (0.035, (0, 0, 0)), (1.0, (0, 0, 0))])
        nt.links.new(v.outputs['Distance'], st.inputs['Fac'])
        mx = nt.nodes.new('ShaderNodeMix')
        mx.data_type = 'RGBA'
        mx.blend_type = 'ADD'
        mx.inputs[0].default_value = stars
        nt.links.new(col, mx.inputs[6])
        nt.links.new(st.outputs['Color'], mx.inputs[7])
        col = mx.outputs[2]
    nt.links.new(col, bg.inputs['Color'])
    bg.inputs['Strength'].default_value = strength


def haze(color, density, size=(40, 40, 12), loc=(0, 6, 4)):
    m, nt, out = kit.new_mat("haze")
    v = nt.nodes.new('ShaderNodeVolumePrincipled')
    setin(v, (*color, 1), 'Color')
    setin(v, density, 'Density')
    setin(v, 0.3, 'Anisotropy')
    nt.links.new(v.outputs[0], out.inputs['Volume'])
    box(loc, size, m)


def camera(loc, target, lens=35, fstop=None, focus=None):
    cam = kit.camera(tuple(loc), (0, 0, 0), lens=lens)
    cam.rotation_euler = (Vector(target) - Vector(loc)).to_track_quat('-Z', 'Y').to_euler()
    if fstop:
        cam.data.dof.use_dof = True
        cam.data.dof.focus_distance = focus or (Vector(target) - Vector(loc)).length
        cam.data.dof.aperture_fstop = fstop
    return cam


def render(name, samples=256):
    sc = bpy.context.scene
    try:
        sc.view_settings.look = 'AgX - Punchy'
        sc.cycles.volume_bounces = 1
    except Exception:
        pass
    os.makedirs(OUT, exist_ok=True)
    kit.render_to(os.path.join(OUT, name + ".jpg"), fmt='JPEG', quality=90)


def tree(x, y, h, m_bark, m_leaf, seed):
    rnd = random.Random(seed)
    cyl((x, y, 0), (x, y, h * 0.45), 0.18 * h / 6, m_bark, n=8, r2=0.1 * h / 6)
    for k in range(4):
        z = h * (0.32 + k * 0.17)
        r = h * (0.32 - k * 0.06)
        bm = bmesh.new()
        bmesh.ops.create_cone(bm, cap_ends=True, segments=10, radius1=r, radius2=0.0, depth=h * 0.34)
        for v in bm.verts:
            v.co.x += rnd.uniform(-0.05, 0.05) * h
            v.co.y += rnd.uniform(-0.05, 0.05) * h
        o = obj("leaf", bm, m_leaf, smooth=False)
        o.location = (x, y, z)


# ------------------------------------------------------------------ rest: campfire glade

def scene_rest():
    kit.reset()
    kit.setup_render(1920, 1080, samples=256)
    world((0.02, 0.025, 0.07), (0.07, 0.05, 0.12), 1.0, stars=0.9)
    grass = mat("grass", (0.05, 0.09, 0.04), rough=0.9, var=0.4, bump=0.4, bscale=6)
    ground(grass, 80, 0, disp=0.35)
    bark = mat("bark", (0.07, 0.045, 0.03), rough=0.9, bump=0.3)
    leaf = mat("pine", (0.02, 0.06, 0.04), rough=0.8, var=0.3)
    rnd = random.Random(4)
    for i in range(46):
        a = rnd.uniform(0, math.pi)
        d = rnd.uniform(9, 26)
        x, y = math.cos(a) * d * 1.3, 4 + math.sin(a) * d
        if abs(x) < 4 and y < 12:
            continue
        tree(x, y, rnd.uniform(5, 10), bark, leaf, i)
    # the fire: logs, a ring of stones, coals and flame shapes
    log_m = mat("log", (0.13, 0.08, 0.05), rough=0.85, bump=0.4, bscale=14)
    stone = mat("stone", (0.18, 0.18, 0.2), rough=0.8, bump=0.5, bscale=8, var=0.3)
    for k in range(10):
        a = k / 10 * 2 * math.pi
        ball((math.cos(a) * 0.95, math.sin(a) * 0.95, 0.1), 0.22, stone, scale=(1, 0.9, 0.7), jitter=0.2, seed=k, smooth=False)
    for a in (20, 80, 140):
        r = math.radians(a)
        cyl((-math.cos(r) * 0.75, -math.sin(r) * 0.75, 0.12), (math.cos(r) * 0.55, math.sin(r) * 0.55, 0.55), 0.11, log_m, n=10)
    coal = glow("coal", (1.0, 0.3, 0.05), 6.0)
    for k in range(14):
        ball((rnd.uniform(-0.35, 0.35), rnd.uniform(-0.35, 0.35), 0.08), 0.07, coal, jitter=0.3, seed=k + 50, smooth=False)
    f1, f2 = glow("f1", (1.0, 0.35, 0.05), 14.0), glow("f2", (1.0, 0.75, 0.3), 22.0)
    for k in range(7):
        a = k / 7 * 2 * math.pi
        rr = 0.18
        h = rnd.uniform(0.7, 1.2)
        cyl((math.cos(a) * rr, math.sin(a) * rr, 0.15), (math.cos(a) * rr * 0.3, math.sin(a) * rr * 0.3, 0.15 + h), 0.16, f1 if k % 2 else f2, n=8, r2=0.0)
    pt((0, 0, 0.8), 900, (1.0, 0.5, 0.18), 0.4)
    pt((0, -0.4, 0.4), 250, (1.0, 0.6, 0.25), 0.3)
    # a tent and a bedroll to the side, a pack and a sword in the ground
    canvas = mat("canvas", (0.45, 0.36, 0.24), rough=0.85, bump=0.2, bscale=30)
    bm = bmesh.new()
    vs = [bm.verts.new(p) for p in [(-1.4, -1.2, 0), (1.4, -1.2, 0), (0, -1.2, 2.2), (-1.4, 1.6, 0), (1.4, 1.6, 0), (0, 1.6, 2.2)]]
    for f in [(0, 1, 2), (3, 5, 4), (0, 2, 5, 3), (1, 4, 5, 2)]:
        bm.faces.new([vs[i] for i in f])
    t = obj("tent", bm, canvas, smooth=False)
    t.location = (4.2, 3.5, 0)
    t.rotation_euler = (0, 0, math.radians(-25))
    box((-2.6, 0.6, 0.12), (1.9, 0.75, 0.22), mat("bedroll", (0.35, 0.08, 0.08), rough=0.9, bump=0.2), rot=(0, 0, 18), bev=0.08)
    steel = mat("steel", (0.6, 0.62, 0.66), rough=0.25, metal=1.0)
    cyl((2.2, -0.8, 0), (2.25, -0.75, 1.1), 0.03, steel, n=8)
    box((2.25, -0.75, 1.15), (0.36, 0.06, 0.05), mat("gold", (1.0, 0.72, 0.3), rough=0.3, metal=1.0))
    # moon and drifting haze
    ball((-14, 38, 16), 2.4, glow("moon", (0.75, 0.8, 1.0), 6.0))
    area((-8, 20, 18), (0, 0, 0), 6, 1500, (0.55, 0.65, 1.0))
    haze((0.6, 0.6, 0.8), 0.012)
    camera((0, -9.5, 3.0), (0, 4, 1.4), lens=30, fstop=4.0, focus=9.6)
    render("rest")


# ------------------------------------------------------------------ shop: the merchant's tent

def scene_shop():
    kit.reset()
    kit.setup_render(1920, 1080, samples=256)
    world((0.03, 0.02, 0.02), (0.05, 0.03, 0.02), 0.5)
    rug = mat("rug", (0.35, 0.06, 0.06), rough=0.9, bump=0.2, bscale=40, var=0.3)
    ground(mat("floor", (0.12, 0.08, 0.05), rough=0.8, bump=0.3, bscale=6), 40, 0)
    box((0, 1.5, 0.01), (6, 4, 0.02), rug)
    cloth = mat("tentcloth", (0.42, 0.12, 0.1), rough=0.85, bump=0.15, bscale=20, var=0.25)
    stripe = mat("tentstripe", (0.75, 0.58, 0.3), rough=0.8)
    # the tent's back wall and roof, draped
    for i in range(14):
        a = math.radians(-70 + i * 10)
        x, y = math.sin(a) * 7, 6 + math.cos(a) * 3
        box((x, y, 3.0), (1.4, 0.1, 6.5), cloth if i % 2 else stripe, rot=(0, 0, math.degrees(-a)))
    wood = mat("wood", (0.22, 0.13, 0.07), rough=0.7, bump=0.3, bscale=18, var=0.2)
    brass = mat("brass", (0.95, 0.66, 0.3), rough=0.25, metal=1.0)
    # shelves of potions, left and right
    rnd = random.Random(7)
    cols = [(0.2, 0.9, 0.45), (0.95, 0.25, 0.75), (0.25, 0.55, 1.0), (1.0, 0.7, 0.15), (1.0, 0.25, 0.25), (0.6, 0.3, 1.0)]
    for side in (-1, 1):
        sx = side * 4.2
        for k in range(3):
            z = 0.8 + k * 1.1
            box((sx, 4.2, z), (2.6, 0.7, 0.08), wood)
            for j in range(6):
                c = cols[(j + k + (side > 0) * 2) % len(cols)]
                x = sx - 1.1 + j * 0.44
                h = rnd.uniform(0.3, 0.55)
                pm = mat("potion%d%d%d" % (side, k, j), c, rough=0.05, trans=0.6, emit=c, es=2.5)
                ball((x, 4.2, z + 0.04 + h * 0.4), h * 0.4, pm, scale=(1, 1, 1.15), sub=2)
                cyl((x, 4.2, z + h * 0.7), (x, 4.2, z + h * 0.95), h * 0.1, pm, n=8)
                cyl((x, 4.2, z + h * 0.95), (x, 4.2, z + h * 1.05), h * 0.12, wood, n=8)
        cyl((sx - 1.3, 4.2, 0), (sx - 1.3, 4.2, 3.4), 0.07, wood, n=8)
        cyl((sx + 1.3, 4.2, 0), (sx + 1.3, 4.2, 3.4), 0.07, wood, n=8)
    # counter with goods: a chest, coins, a crystal ball, scrolls
    box((0, 0.9, 0.55), (5.0, 1.2, 1.1), wood, bev=0.04)
    box((0, 0.9, 1.13), (5.2, 1.3, 0.06), brass)
    box((-1.6, 0.9, 1.45), (0.9, 0.6, 0.55), wood, bev=0.03)
    gold = mat("coins", (1.0, 0.75, 0.3), rough=0.25, metal=1.0)
    for k in range(40):
        a = rnd.uniform(0, 2 * math.pi)
        r = rnd.uniform(0, 0.5)
        cyl((0.6 + math.cos(a) * r, 0.8 + math.sin(a) * r * 0.6, 1.16 + rnd.uniform(0, 0.12)), (0.6 + math.cos(a) * r, 0.8 + math.sin(a) * r * 0.6, 1.19 + rnd.uniform(0, 0.12)), 0.09, gold, n=14)
    orb = ball((1.8, 0.9, 1.55), 0.3, mat("crys", (0.6, 0.5, 1.0), rough=0.0, trans=0.9, emit=(0.5, 0.35, 1.0), es=4.0), sub=3)
    cyl((1.8, 0.9, 1.16), (1.8, 0.9, 1.3), 0.22, brass, n=16, r2=0.15)
    pt((1.8, 0.9, 1.55), 30, (0.6, 0.45, 1.0))
    # hanging lanterns
    lamp = glow("lamp", (1.0, 0.65, 0.3), 12.0)
    for x in (-2.8, 0, 2.8):
        cyl((x, 2.2, 3.4), (x, 2.2, 5.5), 0.015, brass, n=6)
        ball((x, 2.2, 3.2), 0.24, lamp, scale=(1, 1, 1.3), sub=2)
        pt((x, 2.0, 3.2), 260, (1.0, 0.6, 0.28), 0.25)
    # the merchant behind the counter: cloak, hood, glowing eyes
    cloak = mat("cloak", (0.08, 0.06, 0.12), rough=0.9, bump=0.2)
    m = ball((0, 2.2, 1.6), 0.9, cloak, scale=(1, 0.8, 1.7), sub=3)
    ball((0, 2.1, 3.15), 0.5, cloak, scale=(1, 1, 1.15), sub=3)
    ball((0, 1.75, 3.1), 0.34, mat("void", (0.003, 0.003, 0.004), rough=1.0), scale=(0.95, 0.6, 1.0), sub=2)
    eye = glow("eye", (1.0, 0.8, 0.3), 30.0)
    for dx in (-0.12, 0.12):
        ball((dx, 1.5, 3.12), 0.035, eye, scale=(1.4, 1, 0.8), sub=1)
    haze((0.9, 0.7, 0.5), 0.02, size=(20, 14, 8), loc=(0, 3, 3))
    camera((0, -7.5, 2.4), (0, 2, 1.9), lens=32, fstop=5.6, focus=8.2)
    render("shop")


# ------------------------------------------------------------------ event: the rune shrine

def scene_event():
    kit.reset()
    kit.setup_render(1920, 1080, samples=256)
    world((0.02, 0.02, 0.06), (0.04, 0.05, 0.1), 0.8, stars=0.6)
    stone = mat("stone", (0.25, 0.25, 0.28), rough=0.85, bump=0.6, bscale=5, var=0.35)
    moss = mat("moss", (0.06, 0.1, 0.05), rough=0.9, var=0.4, bump=0.3, bscale=8)
    ground(moss, 80, 0, disp=0.3)
    rune = glow("rune", (0.45, 0.75, 1.0), 10.0)
    # a ring of standing stones with glowing runes
    rnd = random.Random(3)
    for k in range(9):
        a = math.radians(200 + k * (140 / 8))
        x, y = math.cos(a) * 7.5, 6 + math.sin(a) * -4
        h = rnd.uniform(2.6, 4.2)
        o = box((x, y, h / 2), (0.9, 0.6, h), stone, rot=(rnd.uniform(-4, 4), rnd.uniform(-4, 4), math.degrees(-a)), bev=0.08)
        box((x - math.sin(a) * 0.0, y - 0.31, h * 0.62), (0.12, 0.02, h * 0.35), rune, rot=(0, 0, math.degrees(-a)))
    # the arch and its altar
    for sx in (-1.6, 1.6):
        box((sx, 6, 2.0), (0.7, 0.7, 4.0), stone, bev=0.06)
    box((0, 6, 4.25), (4.2, 0.8, 0.6), stone, bev=0.06)
    box((0, 5.6, 4.25), (1.4, 0.05, 0.25), rune)
    box((0, 4.0, 0.5), (1.8, 1.0, 1.0), stone, bev=0.05)
    box((0, 3.48, 0.55), (1.2, 0.02, 0.4), rune)
    # the floating crystal and its light
    cm = mat("crystal", (0.55, 0.8, 1.0), rough=0.02, trans=0.85, emit=(0.4, 0.7, 1.0), es=6.0)
    bm = bmesh.new()
    bmesh.ops.create_icosphere(bm, subdivisions=1, radius=0.45)
    c = obj("crystal", bm, cm, smooth=False)
    c.location = (0, 4.0, 2.2)
    c.scale = (0.7, 0.7, 1.4)
    pt((0, 4.0, 2.2), 500, (0.45, 0.7, 1.0), 0.3)
    # inside the arch, a portal shimmer
    pm = glow("portal", (0.25, 0.4, 1.0), 0.7)
    box((0, 6.05, 2.0), (2.5, 0.02, 3.9), pm)
    # candles on the altar steps
    wax = mat("wax", (0.9, 0.85, 0.75), rough=0.5)
    flame = glow("candle", (1.0, 0.7, 0.3), 25.0)
    for x in (-1.2, -0.7, 0.7, 1.2):
        cyl((x, 3.3, 0), (x, 3.3, 0.35 + abs(x) * 0.2), 0.06, wax, n=10)
        ball((x, 3.3, 0.45 + abs(x) * 0.2), 0.05, flame, scale=(1, 1, 1.8), sub=1)
        pt((x, 3.2, 0.5 + abs(x) * 0.2), 12, (1.0, 0.6, 0.25))
    area((6, -2, 10), (0, 4, 1), 5, 600, (0.6, 0.65, 1.0))
    haze((0.5, 0.6, 0.9), 0.02)
    camera((0, -6.5, 2.2), (0, 4, 2.0), lens=30, fstop=5.0, focus=10.2)
    render("event")


# ------------------------------------------------------------------ treasure: the vault

def scene_treasure():
    kit.reset()
    kit.setup_render(1920, 1080, samples=256)
    world((0.02, 0.015, 0.01), (0.03, 0.02, 0.01), 0.4)
    stone = mat("vstone", (0.22, 0.19, 0.16), rough=0.85, bump=0.5, bscale=4, var=0.3)
    ground(stone, 60, 0)
    gold = mat("vgold", (1.0, 0.72, 0.28), rough=0.22, metal=1.0)
    heap = mat("vheap", (1.0, 0.7, 0.26), rough=0.3, metal=1.0, bump=0.9, bscale=28)
    wood = mat("vwood", (0.2, 0.11, 0.06), rough=0.7, bump=0.3, bscale=18)
    rnd = random.Random(9)
    # back wall with pillars
    box((0, 9, 4), (30, 1, 8), stone)
    for x in (-6, -3, 3, 6):
        cyl((x, 7.5, 0), (x, 7.5, 7), 0.5, stone, n=16)
        cyl((x, 7.5, 6.9), (x, 7.5, 7.3), 0.7, stone, n=16)
    # heaps of gold: many coins piled into mounds
    for (hx, hy, hr, hh) in ((0, 4.5, 3.0, 1.4), (-4.5, 5.5, 2.2, 1.0), (4.8, 5.2, 2.4, 1.1)):
        ball((hx, hy, 0), hr, heap, scale=(1, 0.8, hh / hr), sub=4, jitter=0.03, seed=int(hx * 10))
        for k in range(220):
            a = rnd.uniform(0, 2 * math.pi)
            r = hr * math.sqrt(rnd.random()) * 0.95
            z = hh * math.sqrt(max(0, 1 - (r / hr) ** 2)) + 0.02
            p = V(hx + math.cos(a) * r, hy + math.sin(a) * r * 0.8, z)
            n = V(rnd.uniform(-0.3, 0.3), rnd.uniform(-0.3, 0.3), 1).normalized() * 0.02
            cyl(p - n, p + n, 0.14, gold, n=14)
    # chests, open, spilling light
    for (cx, cy, rot) in ((-2.4, 2.4, 18), (2.6, 2.2, -14)):
        box((cx, cy, 0.4), (1.4, 0.9, 0.8), wood, rot=(0, 0, rot), bev=0.04)
        box((cx, cy, 0.4), (1.46, 0.96, 0.12), gold, rot=(0, 0, rot))
        lid = box((cx, cy + 0.45, 1.05), (1.4, 0.15, 0.85), wood, rot=(-65, 0, rot), bev=0.03)
        ball((cx, cy, 0.8), 0.55, glow("loot%d" % cx, (1.0, 0.75, 0.3), 4.0), scale=(1.1, 0.7, 0.3), sub=2)
        pt((cx, cy, 1.2), 120, (1.0, 0.7, 0.3), 0.3)
    # gems scattered on the mounds
    for k in range(18):
        c = [(0.2, 0.8, 1.0), (1.0, 0.2, 0.35), (0.3, 1.0, 0.5), (0.7, 0.3, 1.0)][k % 4]
        gm = mat("gem%d" % k, c, rough=0.0, trans=0.7, emit=c, es=3.0)
        a = rnd.uniform(0, 2 * math.pi)
        r = rnd.uniform(0.3, 2.6)
        ball((math.cos(a) * r, 4.5 + math.sin(a) * r * 0.8, 0.9 * math.sqrt(max(0.05, 1 - (r / 3) ** 2)) + 0.1), 0.1, gm, sub=1, smooth=False)
    # light pouring down from a high window
    area((0, 6, 12), (0, 4, 0), 2.5, 2500, (1.0, 0.85, 0.6))
    area((-8, -4, 5), (0, 4, 1), 6, 300, (0.6, 0.7, 1.0))
    haze((1.0, 0.85, 0.6), 0.008, size=(30, 20, 14), loc=(0, 5, 6))
    camera((0, -6.5, 2.6), (0, 4.5, 1.2), lens=30, fstop=5.6, focus=11)
    render("treasure")


SCENES = {"rest": scene_rest, "shop": scene_shop, "event": scene_event, "treasure": scene_treasure}


def scenes():
    for n, f in SCENES.items():
        f()
    kit.log("scenes done")


TASKS = {"scenes": scenes}
for _n, _f in SCENES.items():
    TASKS["scene_" + _n] = _f
