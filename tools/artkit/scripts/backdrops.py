"""Backdrop scenes for each world, rendered in Cycles.

The game draws a backdrop behind the 3D arena with the image's horizon at
72% of its height, so every camera here looks level with shift_y=0.12375
(which places the horizon exactly there for a 16:9 frame).
"""
import bpy, math, os, random
from mathutils import Vector
import kit
from kit import log

W, H = 3840, 2160
SHIFT = 0.12375
LENS = 20


def lathe(name, profile, sides=24, mat=None, loc=(0, 0, 0), smooth=True):
    """Revolve [(radius, z), ...] around Z."""
    verts, faces = [], []
    for (r, z) in profile:
        for s in range(sides):
            a = s / sides * 2 * math.pi
            verts.append((loc[0] + math.cos(a) * r, loc[1] + math.sin(a) * r, loc[2] + z))
    for i in range(len(profile) - 1):
        for s in range(sides):
            a = i * sides + s
            b = i * sides + (s + 1) % sides
            faces.append((a, b, b + sides, a + sides))
    return kit.mesh_obj(name, verts, faces, mat, smooth)


def box(loc, size, mat=None, bev=0.0, rot=(0, 0, 0)):
    o = kit.prim('cube', loc, (size[0] / 2, size[1] / 2, size[2] / 2), rot=rot, mat=mat)
    if bev > 0:
        kit.bevel(o, bev, 2)
    return o


def finish(name, samples=256):
    kit.render_to(os.path.join(kit.OUT, "bg", name + ".png"), 'PNG')


def base_scene(samples=256):
    kit.reset()
    kit.setup_render(W, H, samples)
    os.makedirs(os.path.join(kit.OUT, "bg"), exist_ok=True)
    kit.camera((0, 0, 3), (90, 0, 0), lens=LENS, shift_y=SHIFT, clip_end=6000)


# ====================================================================== NEON

def neon_material_grid(name, base, line, line_str, scale=0.25, width=0.04):
    """Dark glossy surface with glowing grid lines in object space."""
    m, nt, out = kit.new_mat(name)
    p = kit.principled(nt, base, rough=0.25, metal=0.3)
    tc = nt.nodes.new('ShaderNodeTexCoord')
    mp = nt.nodes.new('ShaderNodeMapping')
    nt.links.new(tc.outputs['Object'], mp.inputs['Vector'])
    kit.setin(mp, (scale, scale, scale), 'Scale')
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(mp.outputs[0], sep.inputs[0])
    lines = []
    for ax in ('X', 'Y'):
        fr = nt.nodes.new('ShaderNodeMath'); fr.operation = 'FRACT'
        nt.links.new(sep.outputs[ax], fr.inputs[0])
        d = nt.nodes.new('ShaderNodeMath'); d.operation = 'SUBTRACT'
        nt.links.new(fr.outputs[0], d.inputs[0]); d.inputs[1].default_value = 0.5
        a = nt.nodes.new('ShaderNodeMath'); a.operation = 'ABSOLUTE'
        nt.links.new(d.outputs[0], a.inputs[0])
        st = nt.nodes.new('ShaderNodeMath'); st.operation = 'GREATER_THAN'
        nt.links.new(a.outputs[0], st.inputs[0]); st.inputs[1].default_value = 0.5 - width
        lines.append(st)
    mx = nt.nodes.new('ShaderNodeMath'); mx.operation = 'MAXIMUM'
    nt.links.new(lines[0].outputs[0], mx.inputs[0]); nt.links.new(lines[1].outputs[0], mx.inputs[1])
    em = nt.nodes.new('ShaderNodeMath'); em.operation = 'MULTIPLY'
    nt.links.new(mx.outputs[0], em.inputs[0]); em.inputs[1].default_value = line_str
    kit.setin(p, (*line, 1), 'Emission Color', 'Emission')
    nt.links.new(em.outputs[0], kit.sock(p, 'Emission Strength'))
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    return m


def striped_sun(loc, radius, top, bot, stripes=7):
    """Retro sun: emissive disc with horizontal gaps across its lower half."""
    m, nt, out = kit.new_mat("SunDisc")
    tc = nt.nodes.new('ShaderNodeTexCoord')
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(tc.outputs['Generated'], sep.inputs[0])
    r = kit.ramp(nt, [(0.0, bot), (1.0, top)])
    nt.links.new(sep.outputs['Y'], r.inputs['Fac'])
    e = nt.nodes.new('ShaderNodeEmission')
    nt.links.new(r.outputs['Color'], e.inputs['Color'])
    kit.setin(e, 6.0, 'Strength')
    # stripes: gaps get thicker towards the bottom
    mul = nt.nodes.new('ShaderNodeMath'); mul.operation = 'MULTIPLY'
    nt.links.new(sep.outputs['Y'], mul.inputs[0]); mul.inputs[1].default_value = stripes * 2
    fr = nt.nodes.new('ShaderNodeMath'); fr.operation = 'FRACT'
    nt.links.new(mul.outputs[0], fr.inputs[0])
    thr = nt.nodes.new('ShaderNodeMath'); thr.operation = 'MULTIPLY_ADD'
    nt.links.new(sep.outputs['Y'], thr.inputs[0]); thr.inputs[1].default_value = -0.9; thr.inputs[2].default_value = 0.55
    gap = nt.nodes.new('ShaderNodeMath'); gap.operation = 'GREATER_THAN'
    nt.links.new(fr.outputs[0], gap.inputs[0]); nt.links.new(thr.outputs[0], gap.inputs[1])
    upper = nt.nodes.new('ShaderNodeMath'); upper.operation = 'LESS_THAN'
    nt.links.new(sep.outputs['Y'], upper.inputs[0]); upper.inputs[1].default_value = 0.55
    hole = nt.nodes.new('ShaderNodeMath'); hole.operation = 'MULTIPLY'
    nt.links.new(gap.outputs[0], hole.inputs[0]); nt.links.new(upper.outputs[0], hole.inputs[1])
    tr = nt.nodes.new('ShaderNodeBsdfTransparent')
    mix = nt.nodes.new('ShaderNodeMixShader')
    nt.links.new(hole.outputs[0], mix.inputs[0])
    nt.links.new(e.outputs[0], mix.inputs[1]); nt.links.new(tr.outputs[0], mix.inputs[2])
    nt.links.new(mix.outputs[0], out.inputs['Surface'])
    bpy.ops.mesh.primitive_circle_add(vertices=128, radius=radius, fill_type='NGON', location=loc, rotation=(math.radians(90), 0, 0))
    o = bpy.context.active_object
    o.data.materials.append(m)
    # glow halo behind the sun
    hm, hnt, hout = kit.new_mat("HaloSoft")
    tc2 = hnt.nodes.new('ShaderNodeTexCoord')
    grad = hnt.nodes.new('ShaderNodeTexGradient'); grad.gradient_type = 'SPHERICAL'
    mp = hnt.nodes.new('ShaderNodeMapping')
    kit.setin(mp, (-1, -1, 0), 'Location'); kit.setin(mp, (2, 2, 2), 'Scale')
    hnt.links.new(tc2.outputs['UV'], mp.inputs[0]); hnt.links.new(mp.outputs[0], grad.inputs[0])
    pw = hnt.nodes.new('ShaderNodeMath'); pw.operation = 'POWER'
    hnt.links.new(grad.outputs['Fac'], pw.inputs[0]); pw.inputs[1].default_value = 2.2
    em = hnt.nodes.new('ShaderNodeEmission'); kit.setin(em, (1.0, 0.3, 0.6, 1), 'Color')
    ms = hnt.nodes.new('ShaderNodeMath'); ms.operation = 'MULTIPLY'
    hnt.links.new(pw.outputs[0], ms.inputs[0]); ms.inputs[1].default_value = 2.5
    hnt.links.new(ms.outputs[0], em.inputs['Strength'])
    add = hnt.nodes.new('ShaderNodeAddShader')
    hnt.links.new(em.outputs[0], add.inputs[0])
    hnt.links.new(hnt.nodes.new('ShaderNodeBsdfTransparent').outputs[0], add.inputs[1])
    hnt.links.new(add.outputs[0], hout.inputs['Surface'])
    p = kit.prim('plane', (loc[0], loc[1] + 5, loc[2]), (radius * 2.4, radius * 2.4, 1), rot=(90, 0, 0), mat=hm)
    return o


def stars(n, dist, seed, zmin=0.06, size=1.5):
    rnd = random.Random(seed)
    m = kit.mat_emit("Star", (1, 0.95, 1), 6)
    verts, faces = [], []
    for i in range(n):
        az = rnd.uniform(-1.2, 1.2)
        el = zmin + (1 - zmin) * rnd.random() ** 1.6
        el = el * 1.1
        d = Vector((math.sin(az) * math.cos(el), math.cos(az) * math.cos(el), math.sin(el))) * dist
        s = size * (0.4 + rnd.random() ** 3 * 1.6)
        b = len(verts)
        verts += [(d.x - s, d.y, d.z - s), (d.x + s, d.y, d.z - s), (d.x + s, d.y, d.z + s), (d.x - s, d.y, d.z + s)]
        faces.append((b, b + 1, b + 2, b + 3))
    kit.mesh_obj("Stars", verts, faces, m)


def wire_mountains(seed, y0, depth, height, color):
    rnd = random.Random(seed)

    def h(x, y):
        ax = abs(x)
        valley = min(1, max(0, (ax - 60) / 160))  # keep the sun's valley open
        ridge = (math.sin(x * 0.021 + seed) * 0.5 + 0.5) * (math.sin(x * 0.0071 + y * 0.01) * 0.4 + 0.6)
        n = (math.sin(x * 0.09 + y * 0.05) + math.sin(x * 0.033 - y * 0.08)) * 0.15
        return height * valley * max(0, ridge + n) * min(1, (y - y0) / 40)

    m = neon_material_grid("MtnGrid", (0.02, 0.0, 0.05), color, 9.0, scale=0.11, width=0.035)
    kit.grid_terrain("Mountains", 1400, depth, 280, 60, h, m, origin=(0, y0, -1), smooth=False)


def neon(variant):
    base_scene(256)
    kit.world_gradient([(0.5, (0.55, 0.06, 0.32)), (0.56, (0.28, 0.03, 0.30)), (0.72, (0.07, 0.01, 0.16)), (1.0, (0.01, 0.0, 0.04))], 1.0)
    kit.world_fog(0.0012, (0.9, 0.3, 0.7), 0.5, height=260)
    stars(900, 2500, 3)
    # glossy grid floor reaching to the horizon
    floor = neon_material_grid("Floor", (0.01, 0.0, 0.03), (1.0, 0.15, 0.7), 7.0, scale=0.12, width=0.03)
    kit.prim('plane', (0, 900, 0), (1500, 900, 1), mat=floor)
    striped_sun((0, 1700, 150), 260, (1.0, 0.85, 0.3), (1.0, 0.15, 0.55))
    if variant == 0:
        wire_mountains(4, 500, 700, 160, (0.2, 0.85, 1.0))
        wire_mountains(9, 260, 300, 70, (1.0, 0.2, 0.8))
    else:
        city(5)
        wire_mountains(9, 900, 500, 140, (0.2, 0.85, 1.0))
    finish("neon_%d" % variant)


def city(seed):
    rnd = random.Random(seed)
    body = kit.mat_color("Tower", (0.02, 0.01, 0.05), rough=0.35, metal=0.4)
    # windows: emission from a brick texture
    wm, nt, out = kit.new_mat("Windows")
    p = kit.principled(nt, (0.02, 0.01, 0.05), 0.3, 0.4)
    tc = nt.nodes.new('ShaderNodeTexCoord')
    br = nt.nodes.new('ShaderNodeTexBrick')
    kit.setin(br, 1.4, 'Scale'); kit.setin(br, 0.12, 'Mortar Size')
    kit.setin(br, (1, 1, 1, 1), 'Color1'); kit.setin(br, (0, 0, 0, 1), 'Color2'); kit.setin(br, (0, 0, 0, 1), 'Mortar')
    nt.links.new(tc.outputs['Object'], br.inputs['Vector'])
    nz = kit.noise_tex(nt, 3.0, 2, 0.5, tc.outputs['Object'])
    th = nt.nodes.new('ShaderNodeMath'); th.operation = 'GREATER_THAN'
    nt.links.new(nz.outputs['Fac'], th.inputs[0]); th.inputs[1].default_value = 0.55
    mul = nt.nodes.new('ShaderNodeMath'); mul.operation = 'MULTIPLY'
    nt.links.new(br.outputs['Fac'], mul.inputs[0]); nt.links.new(th.outputs[0], mul.inputs[1])
    s = nt.nodes.new('ShaderNodeMath'); s.operation = 'MULTIPLY'
    nt.links.new(mul.outputs[0], s.inputs[0]); s.inputs[1].default_value = 8
    kit.setin(p, (1.0, 0.75, 0.45, 1), 'Emission Color', 'Emission')
    nt.links.new(s.outputs[0], kit.sock(p, 'Emission Strength'))
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    rim_c = [(1.0, 0.2, 0.75), (0.2, 0.85, 1.0), (0.7, 0.3, 1.0)]
    for i in range(70):
        x = rnd.uniform(-700, 700)
        if abs(x) < 120 and rnd.random() < 0.7:
            continue
        y = rnd.uniform(520, 820)
        w, d, h = rnd.uniform(18, 50), rnd.uniform(18, 40), rnd.uniform(40, 220) * (1 + abs(x) / 900)
        b = box((x, y, h / 2), (w, d, h), wm)
        rim = kit.mat_emit("Rim%d" % i, rim_c[i % 3], 12)
        box((x, y - d / 2 - 0.5, h + 0.6), (w + 1, 1.2, 1.2), rim)
        if rnd.random() < 0.3:
            box((x, y, h + 12), (1.2, 1.2, 24), body)
            kit.prim('ico', (x, y, h + 25), (1.6, 1.6, 1.6), mat=kit.mat_emit("Beacon%d" % i, (1, 0.2, 0.3), 30), subdivisions=2)


TASKS = {
    "neon": lambda: (neon(0), neon(1)),
}


# ====================================================================== the other worlds
# Same framing as neon: camera at eye height looking along +Y, horizon at 72%.

def _bg_scene(samples=160):
    kit.reset()
    kit.setup_render(2560, 1440, samples)
    os.makedirs(os.path.join(kit.OUT, "bg"), exist_ok=True)
    kit.camera((0, 0, 3), (90, 0, 0), lens=LENS, shift_y=SHIFT, clip_end=8000)


def _puff_cloud(c, size, mat, seed):
    rnd = random.Random(seed)
    for k in range(7):
        off = (rnd.uniform(-1, 1) * size * 1.2, rnd.uniform(-0.4, 0.4) * size, rnd.uniform(-0.15, 0.35) * size)
        r = size * rnd.uniform(0.45, 0.85)
        o = kit.prim('ico', (c[0] + off[0], c[1] + off[1], c[2] + off[2]), (r, r * 0.8, r * 0.7), mat=mat, subdivisions=3)
        for p in o.data.polygons:
            p.use_smooth = True


def _tree(x, y, z, s, bark, leaf, rnd):
    kit.prim('cyl', (x, y, z + 1.6 * s), (0.3 * s, 0.3 * s, 1.6 * s), mat=bark, vertices=8)
    for k in range(3):
        o = kit.prim('ico', (x + rnd.uniform(-0.8, 0.8) * s, y + rnd.uniform(-0.6, 0.6) * s, z + (3.8 + k * 0.6) * s),
                     (1.8 * s, 1.8 * s, 1.5 * s), mat=leaf, subdivisions=2)
        kit.displace(o, 0.35 * s, 0.8)


def meadow_bg(variant):
    _bg_scene()
    if variant == 0:  # golden hour
        kit.world_gradient([(0.48, (0.95, 0.55, 0.32)), (0.52, (1.0, 0.72, 0.45)), (0.62, (0.62, 0.62, 0.78)), (0.8, (0.3, 0.45, 0.8)), (1.0, (0.14, 0.24, 0.55))],
                           1.0, sun_dir=(0.35, 1, 0.06), sun_color=(1, 0.75, 0.45), sun_str=40)
        kit.sun((86, 0, 160), strength=3.0, color=(1, 0.75, 0.5))
        fogc = (1.0, 0.78, 0.6)
    else:  # late morning
        kit.world_gradient([(0.48, (0.75, 0.82, 0.9)), (0.53, (0.7, 0.82, 0.95)), (0.75, (0.4, 0.62, 0.92)), (1.0, (0.2, 0.42, 0.85))],
                           1.0, sun_dir=(-0.4, 1, 0.5), sun_color=(1, 0.95, 0.85), sun_str=30)
        kit.sun((40, 0, -150), strength=4.0, color=(1, 0.96, 0.9))
        fogc = (0.8, 0.88, 0.98)
    kit.world_fog(0.0018, fogc, 0.4, height=160)
    grass = kit.mat_color("Grass", (0.22, 0.48, 0.13), rough=0.95, noise=0.45, noise_scale=0.08, bump_str=0.2)
    far = kit.mat_color("FarHill", (0.28, 0.42, 0.22), rough=0.95, noise=0.3, noise_scale=0.02)

    def h(x, y):
        return 6 * math.sin(x * 0.012 + variant) * math.cos(y * 0.01) + 3 * math.sin(x * 0.031 + y * 0.017) + 0.03 * max(0, y - 120) ** 1.15

    kit.grid_terrain("Hills", 2400, 1400, 220, 120, h, grass, origin=(0, 30, -2))
    kit.grid_terrain("Ridge", 4000, 800, 160, 30, lambda x, y: 60 + 50 * abs(math.sin(x * 0.0021 + 1.3)) + 25 * math.sin(x * 0.0071), far, origin=(0, 1700, 0))
    bark = kit.mat_color("Bark", (0.25, 0.15, 0.08), rough=0.9)
    leaf = kit.mat_color("Leaf", (0.13, 0.36, 0.09), rough=0.85, noise=0.4, noise_scale=1.2, bump_str=0.4)
    rnd = random.Random(10 + variant)
    for i in range(70):
        x, y = rnd.uniform(-700, 700), rnd.uniform(120, 1100)
        if abs(x) < 40 and y < 300:
            continue
        _tree(x, y, h(x, y) - 2, rnd.uniform(1.6, 3.2) * (1 + y / 700), bark, leaf, rnd)
    cloud = kit.mat_color("Cloud", (0.98, 0.95, 0.94), rough=1.0)
    for i in range(12):
        _puff_cloud((rnd.uniform(-900, 900), rnd.uniform(1200, 2600), rnd.uniform(220, 520)), rnd.uniform(40, 90), cloud, 100 + i)
    # a far windmill on a ridge
    st = kit.mat_color("MillWall", (0.9, 0.86, 0.78), rough=0.8)
    x0 = -220 if variant == 0 else 260
    kit.prim('cone', (x0, 900, h(x0, 900) + 18), (1, 1, 18), mat=st, vertices=10, radius1=7, radius2=5, depth=2)
    kit.prim('cone', (x0, 900, h(x0, 900) + 40), (1, 1, 5), mat=kit.mat_color("MillRoof", (0.6, 0.22, 0.16)), vertices=10, radius1=6, radius2=0, depth=2)
    for k in range(4):
        b = box((x0, 893, h(x0, 900) + 36), (2.2, 0.6, 30), kit.mat_color("Sail", (0.85, 0.8, 0.72)), rot=(0, 45 + k * 90, 0))
    finish("meadow_%d" % variant)


def ruins_bg(variant):
    _bg_scene()
    kit.world_gradient([(0.48, (0.95, 0.82, 0.62)), (0.52, (0.98, 0.86, 0.66)), (0.66, (0.62, 0.78, 0.92)), (1.0, (0.22, 0.48, 0.85))],
                       1.0, sun_dir=(0.5 if variant else -0.3, 1, 0.45), sun_color=(1, 0.95, 0.8), sun_str=45)
    kit.sun((50, 0, 150 if variant else -160), strength=5.0, color=(1, 0.92, 0.78))
    kit.world_fog(0.0022, (1.0, 0.86, 0.66), 0.5, height=120)
    sand = kit.mat_color("Sand", (0.78, 0.6, 0.36), rough=0.95, noise=0.25, noise_scale=0.05, bump_str=0.15)

    def h(x, y):
        return 7 * abs(math.sin(x * 0.009 + y * 0.004 + variant)) + 4 * math.sin(x * 0.027 - y * 0.013) + 0.02 * max(0, y - 150) ** 1.2

    kit.grid_terrain("Dunes", 3000, 1600, 240, 120, h, sand, origin=(0, 30, -3))
    stone = kit.mat_color("Pyr", (0.82, 0.66, 0.42), rough=0.9, noise=0.2, noise_scale=0.3)
    rnd = random.Random(20 + variant)
    for (x, y, s) in ((-320, 1100, 140), (180, 1400, 190), (520, 1000, 110), (-700, 1500, 160)):
        x = x * (1 if variant == 0 else -1)
        for k in range(8):  # stepped pyramid
            w = s * (1 - k / 8)
            box((x, y, h(x, y) + k * s / 12 + s / 24), (w * 2, w * 2, s / 12), stone)
    box((-60 if variant == 0 else 90, 700, h(0, 700) + 45), (8, 8, 90), stone)
    kit.prim('cone', (-60 if variant == 0 else 90, 700, h(0, 700) + 95), (1, 1, 5), mat=stone, vertices=4, radius1=5.6, radius2=0, depth=2)
    cloud = kit.mat_color("HazeCloud", (1.0, 0.95, 0.9), rough=1.0)
    for i in range(5):
        _puff_cloud((rnd.uniform(-1000, 1000), rnd.uniform(2000, 3200), rnd.uniform(400, 700)), rnd.uniform(60, 110), cloud, 200 + i)
    finish("ruins_%d" % variant)


def clouds_bg(variant):
    _bg_scene()
    kit.world_gradient([(0.45, (0.85, 0.9, 1.0)), (0.52, (0.8, 0.88, 1.0)), (0.7, (0.45, 0.66, 0.98)), (1.0, (0.2, 0.42, 0.92))],
                       1.1, sun_dir=(0.6 if variant else -0.5, 1, 0.35), sun_color=(1, 0.97, 0.9), sun_str=60)
    kit.sun((55, 0, 140 if variant else -140), strength=5.0)
    kit.world_fog(0.0012, (0.92, 0.95, 1.0), 0.6, height=90, base=-40)
    cloud = kit.mat_color("SeaCloud", (0.98, 0.98, 1.0), rough=1.0)
    rnd = random.Random(30 + variant)
    # a sea of cloud below the horizon line
    for i in range(140):
        x, y = rnd.uniform(-1400, 1400), rnd.uniform(80, 2200)
        _puff_cloud((x, y, -30 - rnd.uniform(0, 12) + y * 0.01), rnd.uniform(18, 45) * (1 + y / 1500), cloud, 300 + i)
    rockm = kit.mat_color("IslandRock", (0.55, 0.48, 0.44), rough=0.9, noise=0.2)
    grass = kit.mat_color("IslandGrass", (0.42, 0.7, 0.35), rough=0.9)
    wall = kit.mat_color("CastleWall", (0.95, 0.93, 0.98), rough=0.7)
    roofs = [kit.mat_color("R%d" % i, c) for i, c in enumerate(((0.35, 0.45, 0.9), (0.9, 0.45, 0.6), (1.0, 0.75, 0.3)))]
    for (x, y, z, s) in ((-420, 1300, 90, 1.0), (380, 1700, 150, 1.3), (900, 1100, 60, 0.7)):
        x = x * (1 if variant == 0 else -1)
        kit.prim('cone', (x, y, z - 20 * s), (1, 1, 20 * s), mat=rockm, vertices=9, radius1=36 * s, radius2=0, depth=2, rot=(180, 0, 0))
        kit.prim('cyl', (x, y, z + 1 * s), (36 * s, 36 * s, 2 * s), mat=grass, vertices=9)
        for k in range(4):
            tx, ty = x + (k - 1.5) * 12 * s, y + (k % 2) * 8 * s
            hh = (24 + (k % 2) * 14) * s
            kit.prim('cyl', (tx, ty, z + hh / 2), (4 * s, 4 * s, hh / 2), mat=wall, vertices=10)
            kit.prim('cone', (tx, ty, z + hh + 6 * s), (1, 1, 6 * s), mat=roofs[k % 3], vertices=10, radius1=5.5 * s, radius2=0, depth=2)
    finish("clouds_%d" % variant)


def caverns_bg(variant):
    _bg_scene()
    kit.world_gradient([(0.0, (0.0, 0.0, 0.01)), (0.5, (0.02, 0.04, 0.08)), (1.0, (0.0, 0.01, 0.03))], 1.0)
    kit.world_fog(0.006, (0.3, 0.6, 0.9), 0.3, height=260)
    rockm = kit.mat_color("CaveRock", (0.07, 0.08, 0.12), rough=0.9, noise=0.3, noise_scale=0.05, bump_str=0.3)

    def ceil(x, y):
        return 140 + 50 * math.sin(x * 0.011 + y * 0.007 + variant) - 30 * math.sin(x * 0.03) - 0.02 * max(0, y - 200)

    kit.grid_terrain("Ceiling", 3000, 1800, 200, 120, ceil, rockm, origin=(0, 40, 0))
    kit.grid_terrain("Floor", 3000, 1800, 160, 90, lambda x, y: -4 + 6 * math.sin(x * 0.02) * math.cos(y * 0.013), rockm, origin=(0, 40, -2))
    rnd = random.Random(40 + variant)
    for i in range(140):  # stalactites
        x, y = rnd.uniform(-1200, 1200), rnd.uniform(80, 1600)
        L = rnd.uniform(20, 80)
        kit.prim('cone', (x, y, ceil(x, y) - L / 2), (1, 1, L / 2), mat=rockm, vertices=6, radius1=0, radius2=rnd.uniform(4, 12), depth=2)
    cols = [(0.3, 0.8, 1.0), (0.6, 0.4, 1.0), (0.3, 1.0, 0.8), (1.0, 0.45, 0.85)]
    for i in range(70):  # glowing crystal clusters on the floor
        x, y = rnd.uniform(-1100, 1100), rnd.uniform(150, 1500)
        c = cols[i % 4]
        m = kit.mat_color("Cr%d" % (i % 4), c, rough=0.15, emit=c, emit_str=8)
        for k in range(rnd.randint(3, 6)):
            hgt = rnd.uniform(15, 60)
            kit.prim('cone', (x + rnd.uniform(-12, 12), y + rnd.uniform(-12, 12), hgt / 2), (1, 1, hgt / 2), mat=m, vertices=6,
                     radius1=rnd.uniform(3, 7), radius2=0, depth=2, rot=(rnd.uniform(-15, 15), rnd.uniform(-15, 15), 0))
        kit.point_light((x, y, 30), 60000, c, radius=10)
    finish("caverns_%d" % variant)


def machine_bg(variant):
    _bg_scene()
    kit.world_gradient([(0.48, (0.9, 0.42, 0.18)), (0.53, (0.75, 0.32, 0.16)), (0.66, (0.32, 0.16, 0.12)), (1.0, (0.06, 0.04, 0.06))],
                       1.0, sun_dir=(0.15 if variant else -0.25, 1, 0.03), sun_color=(1, 0.5, 0.2), sun_str=25)
    kit.world_fog(0.004, (0.5, 0.32, 0.22), 0.5, height=200)
    kit.prim('plane', (0, 1500, 0), (3000, 1600, 1), mat=kit.mat_color("Yard", (0.05, 0.045, 0.045), rough=0.6))
    city(7 + variant)
    rnd = random.Random(50 + variant)
    brick = kit.mat_color("Stack", (0.25, 0.12, 0.08), rough=0.85, noise=0.3, noise_scale=0.4)
    glow = kit.mat_emit("StackGlow", (1, 0.45, 0.15), 15)
    smoke = kit.mat_color("Smoke", (0.18, 0.15, 0.15), rough=1.0)
    for i in range(14):
        x, y = rnd.uniform(-900, 900), rnd.uniform(600, 1500)
        hgt = rnd.uniform(120, 320)
        kit.prim('cone', (x, y, hgt / 2), (1, 1, hgt / 2), mat=brick, vertices=12, radius1=14, radius2=9, depth=2)
        kit.prim('cyl', (x, y, hgt + 2), (9.5, 9.5, 2), mat=glow, vertices=12)
        for k in range(4):
            _puff_cloud((x + k * 25, y, hgt + 30 + k * 35), 18 + k * 9, smoke, 500 + i * 10 + k)
    finish("machine_%d" % variant)


TASKS.update({
    "meadow": lambda: (meadow_bg(0), meadow_bg(1)),
    "ruins": lambda: (ruins_bg(0), ruins_bg(1)),
    "clouds": lambda: (clouds_bg(0), clouds_bg(1)),
    "caverns": lambda: (caverns_bg(0), caverns_bg(1)),
    "machine": lambda: (machine_bg(0), machine_bg(1)),
})
