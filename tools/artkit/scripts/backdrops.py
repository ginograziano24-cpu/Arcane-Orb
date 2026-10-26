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
