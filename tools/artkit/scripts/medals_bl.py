"""Map medallions (task medals_map): each room type as a struck medal, a
gold bezel around a glossy enamel field in the room's colour with its
emblem raised in front, lit like a jewellery shot. Writes
out/icons/medal_<type>.png (512 x 512, transparent)."""
import os, math
import bpy, bmesh
from mathutils import Vector, Matrix
import kit
from kit import setin
import icons as IC

OUT = os.path.join(kit.OUT, "icons")

# enamel colours follow the map legend (hud.go nodeColors), deepened
ENAMEL = {
    "combat": ((0.03, 0.05, 0.12), (0.16, 0.24, 0.45)),
    "elite": ((0.05, 0.0, 0.12), (0.32, 0.06, 0.6)),
    "shop": ((0.1, 0.03, 0.0), (0.6, 0.32, 0.03)),
    "treasure": ((0.0, 0.04, 0.09), (0.03, 0.36, 0.55)),
    "event": ((0.0, 0.05, 0.02), (0.05, 0.42, 0.18)),
    "rest": ((0.1, 0.0, 0.02), (0.6, 0.12, 0.18)),
    "boss": ((0.08, 0.0, 0.0), (0.62, 0.05, 0.02)),
}
GEMS = {"elite": (0.75, 0.3, 1.0), "boss": (1.0, 0.25, 0.1)}
SCALE = {"combat": 0.6, "elite": 0.56, "shop": 0.62, "treasure": 0.6, "event": 0.66, "rest": 0.64, "boss": 0.52}


def _gold(name="mgold"):
    m, nt, out = kit.new_mat(name)
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    setin(p, (1.0, 0.72, 0.3, 1), 'Base Color')
    setin(p, 1.0, 'Metallic')
    setin(p, 0.18, 'Roughness')
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    return m


def _enamel(name, dark, light):
    """Glossy enamel: brighter in the middle, with fine guilloché rings."""
    m, nt, out = kit.new_mat(name)
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    co = nt.nodes.new('ShaderNodeTexCoord').outputs['Object']
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(co, sep.inputs[0])
    comb = nt.nodes.new('ShaderNodeCombineXYZ')
    nt.links.new(sep.outputs['X'], comb.inputs[0])
    nt.links.new(sep.outputs['Z'], comb.inputs[1])
    ln = nt.nodes.new('ShaderNodeVectorMath')
    ln.operation = 'LENGTH'
    nt.links.new(comb.outputs[0], ln.inputs[0])
    r = kit.ramp(nt, [(0.0, light), (0.55, tuple((a + b) / 2 for a, b in zip(light, dark))), (0.92, dark)])
    nt.links.new(ln.outputs['Value'], r.inputs['Fac'])
    nt.links.new(r.outputs['Color'], p.inputs['Base Color'])
    setin(p, 0.25, 'Roughness')
    setin(p, 1.0, 'Coat Weight', 'Clearcoat')
    setin(p, 0.03, 'Coat Roughness', 'Clearcoat Roughness')
    w = nt.nodes.new('ShaderNodeTexWave')
    try:
        w.wave_type = 'RINGS'
    except Exception:
        pass
    setin(w, 14.0, 'Scale')
    setin(w, 1.5, 'Distortion')
    nt.links.new(comb.outputs[0], w.inputs['Vector'])
    b = kit.bump(nt, w.outputs['Fac'], 0.06, 0.01)
    nt.links.new(b.outputs['Normal'], p.inputs['Normal'])
    # a faint inner glow so the field reads even on dark maps
    setin(p, (*light, 1), 'Emission Color')
    setin(p, 0.05, 'Emission Strength')
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    return m


def _disc(name, r, y0, y1, mat, n=96, bev=0.02):
    bm = bmesh.new()
    bmesh.ops.create_cone(bm, cap_ends=True, segments=n, radius1=r, radius2=r, depth=y1 - y0)
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    o.data.materials.append(mat)
    o.rotation_euler = (math.radians(90), 0, 0)
    o.location = (0, (y0 + y1) / 2, 0)
    for f in o.data.polygons:
        f.use_smooth = True
    if bev > 0:
        m = o.modifiers.new("b", 'BEVEL')
        m.width = bev
        m.segments = 3
        m.limit_method = 'ANGLE'
    return o


def _ring(name, R, r, y, mat, n=96, sq=1.0):
    bm = bmesh.new()
    rs = 16
    for i in range(n):
        a = 2 * math.pi * i / n
        for j in range(rs):
            b = 2 * math.pi * j / rs
            rr = R + r * math.cos(b)
            bm.verts.new((rr * math.cos(a), y + r * math.sin(b) * sq, rr * math.sin(a)))
    bm.verts.ensure_lookup_table()
    for i in range(n):
        for j in range(rs):
            i2, j2 = (i + 1) % n, (j + 1) % rs
            bm.faces.new((bm.verts[i * rs + j], bm.verts[i2 * rs + j], bm.verts[i2 * rs + j2], bm.verts[i * rs + j2]))
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    o.data.materials.append(mat)
    for f in o.data.polygons:
        f.use_smooth = True
    return o


def _lights():
    sc = bpy.context.scene
    w = bpy.data.worlds.new("w")
    sc.world = w
    w.use_nodes = True
    bg = w.node_tree.nodes.get("Background")
    bg.inputs[0].default_value = (0.02, 0.02, 0.03, 1)
    bg.inputs[1].default_value = 1.0

    def area(name, loc, size, energy, color=(1, 1, 1), sy=None):
        d = bpy.data.lights.new(name, 'AREA')
        d.energy = energy
        d.color = color
        d.size = size
        if sy:
            d.shape = 'RECTANGLE'
            d.size_y = sy
        else:
            d.shape = 'DISK'
        o = bpy.data.objects.new(name, d)
        sc.collection.objects.link(o)
        o.location = loc
        o.rotation_euler = (Vector((0, 0, 0)) - Vector(loc)).to_track_quat('-Z', 'Y').to_euler()
    # a big soft key up-left (the glossy sweep across the enamel), a cool
    # fill, and a warm rim from behind for the bezel's edge
    area("key", (-3.5, -6.0, 4.0), 1.6, 900, (1.0, 0.96, 0.9))
    area("strip", (3.5, -5.0, -1.5), 5.0, 280, (0.75, 0.85, 1.0), sy=0.5)
    area("top", (0.0, -3.0, 6.0), 2.0, 300, (1, 1, 1))
    area("rim", (0.0, 4.0, 3.0), 3.0, 400, (1.0, 0.8, 0.6))


def medal(kind):
    kit.reset()
    kit.setup_render(512, 512, samples=160, transparent=True)
    sc = bpy.context.scene
    try:
        sc.view_settings.look = 'AgX - Punchy'
    except Exception:
        pass
    # the emblem: reuse the room icon models, shrunk and set proud of the field
    real = IC._render
    IC._render = lambda *a, **k: None
    try:
        IC.ICONS["node_" + kind]()
    finally:
        IC._render = real
    objs = [o for o in sc.objects if o.type == 'MESH']
    s = SCALE[kind]
    # centre the emblem on its bounding box
    bpy.context.view_layer.update()
    lo, hi = Vector((1e9,) * 3), Vector((-1e9,) * 3)
    for o in objs:
        for c in o.bound_box:
            w = o.matrix_world @ Vector(c)
            lo = Vector(map(min, lo, w))
            hi = Vector(map(max, hi, w))
    mid = (lo + hi) / 2
    big = max(hi.x - lo.x, hi.z - lo.z)
    if big > 2.3:
        s *= 2.3 / big
    T = Matrix.Translation(Vector((0, -0.32, 0))) @ Matrix.Scale(s, 4) @ Matrix.Translation(-Vector((mid.x, 0, mid.z)))
    for o in objs:
        if o.parent is None:
            o.matrix_world = T @ o.matrix_world
        # round off the organic shapes (skulls, sacks, flames, orbs)
        if any(k in o.name for k in ("Ico", "lathe", "tube", "Sphere")) and len(o.data.polygons) < 4000:
            for f in o.data.polygons:
                f.use_smooth = True
            m = o.modifiers.new("sub", 'SUBSURF')
            m.levels = 2
            m.render_levels = 2
    # the medal
    dark, light = ENAMEL[kind]
    gold = _gold()
    _disc("field", 0.86, -0.06, 0.06, _enamel("enamel_" + kind, dark, light), bev=0.0)
    _disc("back", 1.0, 0.04, 0.16, gold, bev=0.03)
    _ring("bezel", 0.93, 0.075, -0.06, gold, sq=1.3)
    _ring("inner", 0.84, 0.02, -0.07, gold)
    if kind in GEMS:
        gm, nt, out = kit.new_mat("gem_" + kind)
        p = nt.nodes.new('ShaderNodeBsdfPrincipled')
        setin(p, (*GEMS[kind], 1), 'Base Color')
        setin(p, 0.02, 'Roughness')
        setin(p, 0.4, 'Transmission Weight', 'Transmission')
        setin(p, (*GEMS[kind], 1), 'Emission Color')
        setin(p, 1.5, 'Emission Strength')
        nt.links.new(p.outputs[0], out.inputs['Surface'])
        for k in range(8):
            a = math.pi / 2 + k * math.pi / 4
            bm = bmesh.new()
            bmesh.ops.create_icosphere(bm, subdivisions=1, radius=0.075)
            me = bpy.data.meshes.new("gem")
            bm.to_mesh(me)
            bm.free()
            o = bpy.data.objects.new("gem", me)
            sc.collection.objects.link(o)
            o.data.materials.append(gm)
            o.location = (math.cos(a) * 0.93, -0.16, math.sin(a) * 0.93)
            o.scale = (1, 0.6, 1)
    _lights()
    cam = kit.camera((0, -10, 0), (90, 0, 0), lens=50)
    cam.data.type = 'ORTHO'
    cam.data.ortho_scale = 2.16
    os.makedirs(OUT, exist_ok=True)
    kit.render_to(os.path.join(OUT, "medal_%s.png" % kind))


KINDS = ["combat", "elite", "shop", "treasure", "event", "rest", "boss"]


def medals_map():
    for k in KINDS:
        medal(k)
    kit.log("medals done")


TASKS = {"medals_map": medals_map}
for _k in KINDS:
    TASKS["medal_" + _k] = (lambda k: (lambda: medal(k)))(_k)
