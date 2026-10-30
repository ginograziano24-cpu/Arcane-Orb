"""Game models for Arcane Orb, built in Blender and exported as .orbm.

Every asset is modelled in game units with Blender's Z up and its front
facing -Y (towards the game camera). The exporter applies modifiers,
triangulates, bakes ambient occlusion into each face's colour with ray casts,
maps Blender (x, y, z) to game (x, z, y) and writes out/models/<name>.orbm
plus a preview render in out/models/preview/<name>.png.

Materials carry their game colour as custom properties: orb_rgb (sRGB
0-255), orb_tint (1 = takes the colour the game draws the mesh with),
orb_emit (glow, 0-2) and orb_var (random per-face brightness variation).
"""
import bpy, bmesh, math, os, random, struct, time, zlib
from mathutils import Vector, Matrix
from mathutils.bvhtree import BVHTree
import kit
from kit import log

OUT = os.path.join(kit.OUT, "models")
PREV = os.path.join(OUT, "preview")
ASSETS = {}


def asset(name, group, ao=1.0, rays=28, preview=True):
    def deco(fn):
        ASSETS[name] = (fn, dict(group=group, ao=ao, rays=rays, preview=preview))
        return fn
    return deco


# ------------------------------------------------------------------ materials

def _lin(c):
    c = c / 255.0
    return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4


def M(name, rgb, tint=False, emit=0.0, var=0.0, rough=0.6, metal=0.0):
    """A game material; the Blender node setup is only for previews."""
    if name in bpy.data.materials:
        return bpy.data.materials[name]
    m, nt, out = kit.new_mat(name)
    base = tuple(_lin(c) for c in rgb)
    p = kit.principled(nt, base, rough=rough, metal=metal,
                       emit=base if emit > 0 else None, emit_str=emit * 2.0)
    nt.links.new(p.outputs[0], out.inputs[0])
    m.diffuse_color = (*base, 1)
    m["orb_rgb"] = list(rgb)
    m["orb_tint"] = 1 if tint else 0
    m["orb_emit"] = float(emit)
    m["orb_var"] = float(var)
    return m


# ------------------------------------------------------------------ geometry

def _sel(o):
    for x in bpy.context.selected_objects:
        x.select_set(False)
    o.select_set(True)
    bpy.context.view_layer.objects.active = o


def box(c, size, mat, bev=0.0, seg=2, rot=(0, 0, 0)):
    o = kit.prim('cube', c, (size[0] / 2, size[1] / 2, size[2] / 2), rot=rot, mat=mat)
    _sel(o)
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    if bev > 0:
        m = o.modifiers.new("Bevel", 'BEVEL')
        m.width = bev
        m.segments = seg
        try:
            m.limit_method = 'ANGLE'
        except Exception:
            pass
    return o


def cyl(c, r, h, mat, n=12, bev=0.0, rot=(0, 0, 0), r2=None):
    """Cylinder (or cone frustum with r2) standing on c."""
    if r2 is None:
        o = kit.prim('cyl', (c[0], c[1], c[2] + h / 2), (r, r, h / 2), rot=rot, mat=mat, vertices=n)
    else:
        o = kit.prim('cone', (c[0], c[1], c[2] + h / 2), (1, 1, h / 2), rot=rot, mat=mat, vertices=n, radius1=r, radius2=r2, depth=2)
    _sel(o)
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    if bev > 0:
        m = o.modifiers.new("Bevel", 'BEVEL')
        m.width = bev
        m.segments = 2
        try:
            m.limit_method = 'ANGLE'
        except Exception:
            pass
    return o


def sphere(c, r, mat, sub=2, scale=(1, 1, 1), jitter=0.0, seed=0, smooth=False):
    o = kit.prim('ico', c, (r * scale[0], r * scale[1], r * scale[2]), mat=mat, subdivisions=sub)
    _sel(o)
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    if jitter > 0:
        kit.jitter_mesh(o, jitter * r, seed)
    return o


def lathe(profile, mat, n=16, c=(0, 0, 0), mats=None):
    """Revolve [(radius, z), ...] around Z; mats gives a material per band."""
    verts, faces, fm = [], [], []
    for (r, z) in profile:
        for s in range(n):
            a = s / n * 2 * math.pi
            verts.append((c[0] + math.cos(a) * r, c[1] + math.sin(a) * r, c[2] + z))
    for i in range(len(profile) - 1):
        for s in range(n):
            a = i * n + s
            b = i * n + (s + 1) % n
            faces.append((a, b, b + n, a + n))
            fm.append(i)
    o = kit.mesh_obj("lathe", verts, faces)
    if mats:
        for mm in mats:
            o.data.materials.append(mm)
        for p, i in zip(o.data.polygons, fm):
            p.material_index = min(i, len(mats) - 1)
    else:
        o.data.materials.append(mat)
    _fix_normals(o)
    return o


def mesh(verts, faces, mat):
    o = kit.mesh_obj("mesh", verts, faces, mat)
    _fix_normals(o)
    return o


def _fix_normals(o):
    bm = bmesh.new()
    bm.from_mesh(o.data)
    bmesh.ops.remove_doubles(bm, verts=bm.verts, dist=1e-5)
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    bm.to_mesh(o.data)
    bm.free()


def bevel(o, w, seg=2):
    m = o.modifiers.new("Bevel", 'BEVEL')
    m.width = w
    m.segments = seg
    try:
        m.limit_method = 'ANGLE'
    except Exception:
        pass
    return o


def solidify(o, t):
    m = o.modifiers.new("Solid", 'SOLIDIFY')
    m.thickness = t
    return o


def decimate(o, ratio):
    m = o.modifiers.new("Dec", 'DECIMATE')
    m.ratio = ratio
    return o


def rot(o, rx=0, ry=0, rz=0):
    o.rotation_euler = (math.radians(rx), math.radians(ry), math.radians(rz))
    return o


def move(o, x=0, y=0, z=0):
    o.location = (o.location[0] + x, o.location[1] + y, o.location[2] + z)
    return o


def torus(c, R, r, mat, seg=16, rseg=6, rot=(0, 0, 0)):
    o = kit.prim('torus', c, (1, 1, 1), rot=rot, mat=mat, major_radius=R, minor_radius=r,
                 major_segments=seg, minor_segments=rseg)
    return o


def place(o, loc=None, rot=None):
    """Set an object's final location / rotation (degrees)."""
    if rot is not None:
        o.rotation_euler = tuple(math.radians(a) for a in rot)
    if loc is not None:
        o.location = loc
    return o


def tube(points, r, mat, n=8, r_end=None, closed_ends=True):
    """A tube along a polyline of (x, y, z) points; radius tapers to r_end."""
    pts = [Vector(p) for p in points]
    verts, faces = [], []
    m = len(pts)
    for i, p in enumerate(pts):
        d = (pts[min(i + 1, m - 1)] - pts[max(i - 1, 0)]).normalized()
        up = Vector((0, 0, 1)) if abs(d.z) < 0.95 else Vector((1, 0, 0))
        u = d.cross(up).normalized()
        v = d.cross(u).normalized()
        rr = r if r_end is None else r + (r_end - r) * i / max(1, m - 1)
        for k in range(n):
            a = k / n * 2 * math.pi
            verts.append(tuple(p + (u * math.cos(a) + v * math.sin(a)) * rr))
    for i in range(m - 1):
        for k in range(n):
            a = i * n + k
            b = i * n + (k + 1) % n
            faces.append((a, b, b + n, a + n))
    if closed_ends:
        faces.append(tuple(range(n))[::-1])
        faces.append(tuple(range((m - 1) * n, m * n)))
    return mesh(verts, faces, mat)


def prism(outline, h, mat, z0=0.0):
    """Extrude a 2D outline [(x, y), ...] (counter-clockwise) upward by h."""
    n = len(outline)
    verts = [(x, y, z0) for x, y in outline] + [(x, y, z0 + h) for x, y in outline]
    faces = [tuple(range(n))[::-1], tuple(range(n, 2 * n))]
    for i in range(n):
        j = (i + 1) % n
        faces.append((i, j, j + n, i + n))
    return mesh(verts, faces, mat)


def rock(c, r, mat, seed=0, scale=(1, 1, 1), sub=2, rough=0.22):
    o = sphere(c, r, mat, sub=sub, scale=scale, jitter=rough, seed=seed)
    return o


def foliage(c, r, mat, seed=0, scale=(1, 1, 1), sub=2):
    """A leafy clump: a lumpy sphere with a few smaller bulges."""
    rng = random.Random(seed)
    sphere(c, r, mat, sub=sub, scale=scale, jitter=0.18, seed=seed)
    for k in range(3):
        a = rng.uniform(0, 2 * math.pi)
        d = r * 0.6
        sphere((c[0] + math.cos(a) * d * scale[0], c[1] + math.sin(a) * d * scale[1], c[2] + rng.uniform(-0.1, 0.35) * r),
               r * rng.uniform(0.45, 0.65), mat, sub=1, jitter=0.15, seed=seed + k + 1)


# ------------------------------------------------------------------ export

def _hemisphere(n, seed=7):
    rng = random.Random(seed)
    out = []
    for i in range(n):
        # stratified cosine-weighted directions
        u = (i + rng.random()) / n
        v = rng.random()
        r = math.sqrt(u)
        a = 2 * math.pi * v
        out.append(Vector((r * math.cos(a), r * math.sin(a), math.sqrt(max(0.0, 1 - u)))))
    return out


def export(name, opts):
    os.makedirs(OUT, exist_ok=True)
    dg = bpy.context.evaluated_depsgraph_get()
    tris = []
    for o in list(bpy.context.scene.objects):
        if o.type != 'MESH' or o.hide_render:
            continue
        oe = o.evaluated_get(dg)
        me = oe.to_mesh()
        me.calc_loop_triangles()
        mw = oe.matrix_world
        slots = [s.material for s in oe.material_slots]
        for lt in me.loop_triangles:
            vs = [mw @ me.vertices[i].co for i in lt.vertices]
            mat = slots[lt.material_index] if lt.material_index < len(slots) else None
            tris.append((vs, mat))
        oe.to_mesh_clear()
    if not tris:
        log("export", name, ": no geometry")
        return 0
    verts, polys = [], []
    for i, (vs, _) in enumerate(tris):
        verts.extend(vs)
        polys.append((3 * i, 3 * i + 1, 3 * i + 2))
    bvh = BVHTree.FromPolygons(verts, polys, epsilon=0.0)
    dirs = _hemisphere(opts['rays'])
    rng = random.Random(hash(name) & 0xffff)
    body = bytearray()
    n = 0
    for vs, mat in tris:
        nrm = (vs[1] - vs[0]).cross(vs[2] - vs[0])
        if nrm.length < 1e-12:
            continue
        nrm.normalize()
        c = (vs[0] + vs[1] + vs[2]) / 3
        t = (vs[1] - vs[0]).normalized()
        b = nrm.cross(t)
        hit = 0
        for d in dirs:
            wd = t * d.x + b * d.y + nrm * d.z
            loc, _, _, _ = bvh.ray_cast(c + nrm * 1e-3, wd, opts['ao'])
            if loc is not None:
                hit += 1
        ao = 1 - hit / len(dirs)
        shade = 0.42 + 0.58 * (ao ** 0.8)
        rgb = list(mat["orb_rgb"]) if mat is not None and "orb_rgb" in mat else [200, 200, 200]
        tint = int(mat["orb_tint"]) if mat is not None and "orb_tint" in mat else 0
        emit = float(mat["orb_emit"]) if mat is not None and "orb_emit" in mat else 0.0
        var = float(mat["orb_var"]) if mat is not None and "orb_var" in mat else 0.0
        if emit > 0:
            shade = max(shade, 0.85)
        k = shade * (1 + var * (rng.random() - 0.5))
        col = [max(0, min(255, int(round(ch * k)))) for ch in rgb]
        # Blender (x, y, z) -> game (x, z, y); the mirror flips winding
        for v in (vs[0], vs[2], vs[1]):
            body += struct.pack("<fff", v.x, v.z, v.y)
        body += bytes(col[:3]) + bytes([255, min(255, int(emit / 2 * 255)), 1 if tint else 0])
        n += 1
    data = b"ORBM" + zlib.compress(struct.pack("<II", 1, n) + bytes(body), 9)
    with open(os.path.join(OUT, name + ".orbm"), "wb") as f:
        f.write(data)
    log("export", name, ":", n, "tris,", len(data), "bytes")
    return n


def preview(name):
    """A quick Cycles render of the asset, framed from the game's view angle."""
    os.makedirs(PREV, exist_ok=True)
    objs = [o for o in bpy.context.scene.objects if o.type == 'MESH']
    if not objs:
        return
    lo = Vector((1e9, 1e9, 1e9))
    hi = Vector((-1e9, -1e9, -1e9))
    dg = bpy.context.evaluated_depsgraph_get()
    for o in objs:
        for c in o.evaluated_get(dg).bound_box:
            w = o.matrix_world @ Vector(c)
            lo = Vector((min(lo.x, w.x), min(lo.y, w.y), min(lo.z, w.z)))
            hi = Vector((max(hi.x, w.x), max(hi.y, w.y), max(hi.z, w.z)))
    ctr = (lo + hi) / 2
    size = max((hi - lo).length, 0.5)
    sc = bpy.context.scene
    kit.setup_render(512, 512, samples=32)
    sc.render.film_transparent = True
    # the game looks down at roughly 25 degrees from in front
    d = Vector((0.35, -1.0, 0.55)).normalized() * size * 1.6
    cam = kit.camera(tuple(ctr + d), (0, 0, 0), lens=50)
    direction = ctr - cam.location
    cam.rotation_euler = direction.to_track_quat('-Z', 'Y').to_euler()
    kit.sun((50, 10, 30), strength=3.5, color=(1, 0.95, 0.88))
    try:
        w = bpy.data.worlds.new("w")
        sc.world = w
        w.use_nodes = True
        bg = w.node_tree.nodes.get("Background")
        bg.inputs[0].default_value = (0.35, 0.38, 0.45, 1)
        bg.inputs[1].default_value = 0.8
    except Exception:
        pass
    kit.render_to(os.path.join(PREV, name + ".png"))


def build(name):
    fn, opts = ASSETS[name]
    kit.reset()
    fn()
    n = export(name, opts)
    if opts['preview']:
        try:
            preview(name)
        except Exception as e:
            log("preview failed", name, e)
    return n


def build_group(group):
    t = time.time()
    names = [n for n, (_, o) in ASSETS.items() if group in ("all", o['group'])]
    for n in names:
        try:
            build(n)
        except Exception:
            import traceback
            log("FAILED", n)
            traceback.print_exc()
    log("group", group, ":", len(names), "assets in %.0fs" % (time.time() - t))


# ------------------------------------------------------------------ assets

import importlib
ASSET_MODULES = ("m_common", "m_meadow", "m_ruins", "m_clouds", "m_neon", "m_caverns", "m_machine")
for _m in ASSET_MODULES:
    try:
        importlib.reload(importlib.import_module(_m))
    except ModuleNotFoundError:
        pass

GROUPS = sorted(set(o['group'] for _, o in ASSETS.values()))
TASKS = {}
for _g in GROUPS + ["all"]:
    TASKS["models_" + _g] = (lambda g: (lambda: build_group(g)))(_g)
for _n in list(ASSETS):
    TASKS[_n] = (lambda n: (lambda: build(n)))(_n)
