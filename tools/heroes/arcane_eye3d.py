# Blender 5.x: real 3D eye + lashes for the Arcanist (first pass: her right eye only,
# the one on the viewer's left). Nothing is painted: the eye is a sphere set
# behind an opening cut along her lid lines, the liner and lashes are geometry.
# The only texture change is plain skin where the old painted lashes were.
#   -> heroes/arcane_layers2/eye3d_front.png, eye3d_34.png, eye3d_close.png, arcane_eye3d.blend
import bpy, bmesh, os, sys, json, math
import numpy as np
from mathutils import Vector
from mathutils.bvhtree import BVHTree
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import eyeproj
ROOT = os.path.dirname(HERE)
L2 = eyeproj.L2
SRC = os.path.join(ROOT, "heroes", "glb", "fantasy witch 3d model.glb")
log = open(os.path.join(L2, "eye3d_log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a); print(s); log.write(s + "\n"); log.flush()


cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
geo = json.load(open(os.path.join(L2, "eye3d_geom.json")))
CX, CY, CZ = cam["loc"]; OS = cam["ortho_scale"]; RX, RY = cam["res"]
PX = OS / RX                                   # world units per image pixel


def w_of(px, py):
    return CX + (px / RX - 0.5) * OS, CZ + (0.5 - py / RY) * OS * RY / RX


bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
bpy.ops.import_scene.gltf(filepath=SRC)
new = list(sc.objects)
meshes = [o for o in new if o.type == "MESH"]
bpy.ops.object.select_all(action="DESELECT")
for m in meshes:
    m.select_set(True)
bpy.context.view_layer.objects.active = meshes[0]
if len(meshes) > 1:
    bpy.ops.object.join()
ob = bpy.context.view_layer.objects.active
keep = ob.matrix_world.copy(); ob.parent = None; ob.matrix_world = keep
bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
for o in new:
    if o != ob and o.name in bpy.data.objects:
        bpy.data.objects.remove(o)
me = ob.data
# 1) plain skin where the painted lashes were
mat = me.materials[0]
bsdf = next(n for n in mat.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
texn = next(l.from_node for l in bsdf.inputs["Base Color"].links)
texn.image = eyeproj.project_bake(ob, texn.image, me.uv_layers.active.name, say, fixed="skinfill_fixed.png", mask="skinfill_mask.png")

# 2) geometry of the eye in world space
F = len(me.polygons)
cen = np.empty(F * 3, dtype=np.float32); me.polygons.foreach_get("center", cen); cen = cen.reshape(-1, 3)
upper = np.array(geo["upper"]); lower = np.array(geo["lower"])     # image px polylines, same x samples
ix, iy, iR = geo["iris"]
ex, ez = w_of(ix, iy)
# local faces near the eye, for ray casts
near = (np.abs(cen[:, 0] - ex) < 0.03) & (np.abs(cen[:, 2] - ez) < 0.025) & (cen[:, 1] < CY + 0.75 * cam["hgt"])
idx = np.where(near)[0]
vmap = {}; verts = []; polys = []
for i in idx:
    p = me.polygons[int(i)]
    loc = []
    for v in p.vertices:
        if v not in vmap:
            vmap[v] = len(verts); verts.append(me.vertices[v].co.copy())
        loc.append(vmap[v])
    polys.append(loc)
bvh = BVHTree.FromPolygons(verts, polys)
hit = bvh.ray_cast(Vector((ex, CY, ez)), Vector((0, 1, 0)))
# the eye's own surface: look past any hair strand in front by sampling a few points on the iris
ys = []
for dx, dz in ((0, 0), (0.3, 0), (-0.3, 0), (0, 0.3), (0, -0.3), (0.5, -0.3), (-0.5, -0.3)):
    h = bvh.ray_cast(Vector((ex + dx * iR * PX, CY, ez + dz * iR * PX)), Vector((0, 1, 0)))
    if h[0] is not None:
        ys.append(h[0].y)
yc = float(np.median(ys))
say("eye centre", round(ex, 4), round(ez, 4), "surface y", round(yc, 4), "samples", [round(v, 4) for v in ys])


def surf_y(x, z):
    h = bvh.ray_cast(Vector((x, yc - 0.004, z)), Vector((0, 1, 0)))
    if h[0] is None or h[3] > 0.014:
        return yc
    return h[0].y


# 3) the eye itself: a lens-shaped surface that follows her lid lines exactly and
#    sits just in front of the old painted eye (no cutting into the fused mesh)
xs_px = upper[:, 0]
NXg, NYg = 200, 70
gv, gf, gc = [], [], []
for a in range(NXg + 1):
    fx = a / NXg
    i = fx * (len(upper) - 1); k = int(min(len(upper) - 2, math.floor(i))); t = i - k
    px = upper[k, 0] * (1 - t) + upper[k + 1, 0] * t
    pu = upper[k, 1] * (1 - t) + upper[k + 1, 1] * t
    pl = lower[k, 1] * (1 - t) + lower[k + 1, 1] * t
    for bq in range(NYg + 1):
        fy = bq / NYg
        py = pu + (pl - pu) * fy
        x, z = w_of(px, py)
        edge = min(fx, 1 - fx) * 2 * min(fy, 1 - fy) * 2
        bulge = 0.0007 * math.sqrt(max(0.0, min(1.0, edge * 1.5)))
        gv.append((x, surf_y(x, z) - 0.00035 - bulge, z))
        d = math.hypot(px - ix, py - iy) / iR
        if d < 1.0:
            kk = min(1.0, max(0.0, (py - (iy - iR)) / (2 * iR))) ** 0.8
            c = Vector((0.03, 0.14, 0.45)).lerp(Vector((0.32, 0.85, 1.0)), kk)
            c = c.lerp(Vector((0.02, 0.07, 0.26)), min(1.0, max(0.0, (d - 0.82) / 0.18)) * 0.9)
            c = c.lerp(Vector((0.02, 0.07, 0.24)), min(1.0, max(0.0, (0.25 - d) / 0.06)) * 0.7)
            if d > 0.3:
                c = c * (1 + 0.05 * math.sin(math.atan2(py - iy, px - ix) * 30))
        else:
            c = Vector((0.95, 0.95, 0.97))
        sh = 1 - 0.32 * max(0.0, 1 - (py - pu) / 18.0)
        c = c * sh
        for ox, oy, rr, al in ((-0.3, -0.3, 0.17, 0.95), (0.28, 0.3, 0.07, 0.6)):
            dd = math.hypot(px - (ix + ox * iR), py - (iy + oy * iR)) / (rr * iR)
            if dd < 1.0 and d < 1.0:
                c = c.lerp(Vector((1, 1, 1)), al * min(1.0, (1 - dd) * 3))
        gc.append(c)
for a in range(NXg):
    for bq in range(NYg):
        v0 = a * (NYg + 1) + bq
        gf.append((v0, v0 + NYg + 1, v0 + NYg + 2, v0 + 1))
em_ = bpy.data.meshes.new("Eye_R"); em_.from_pydata(gv, [], gf); em_.update()
eye = bpy.data.objects.new("Eye_R", em_); sc.collection.objects.link(eye)
att = em_.color_attributes.new(name="eye", type="FLOAT_COLOR", domain="POINT")
flat = []
for c in gc:
    flat += [c.x, c.y, c.z, 1.0]
att.data.foreach_set("color", flat)
for p in em_.polygons:
    p.use_smooth = True
em = bpy.data.materials.new("Eye"); em.use_nodes = True
nt = em.node_tree; b = nt.nodes["Principled BSDF"]
vc = nt.nodes.new("ShaderNodeVertexColor"); vc.layer_name = "eye"
nt.links.new(vc.outputs["Color"], b.inputs["Base Color"])
nt.links.new(vc.outputs["Color"], b.inputs["Emission Color"])
b.inputs["Emission Strength"].default_value = 0.5
b.inputs["Roughness"].default_value = 0.25
em_.materials.append(em)
say("eye patch verts", len(gv))

# 5) liner and lashes as geometry
cu = bpy.data.curves.new("Lashes", "CURVE"); cu.dimensions = "3D"; cu.bevel_depth = 1.0; cu.bevel_resolution = 3; cu.use_fill_caps = True
lash = bpy.data.objects.new("Lashes", cu); sc.collection.objects.link(lash)
lm = bpy.data.materials.new("Lash"); lm.use_nodes = True
lb = lm.node_tree.nodes["Principled BSDF"]
lb.inputs["Base Color"].default_value = (0.02, 0.012, 0.015, 1); lb.inputs["Roughness"].default_value = 0.5
cu.materials.append(lm)
outer_sign = -1.0 if geo["outer"] == "left" else 1.0       # image x direction of the outer corner
n_up = len(upper)


def lid_pt(u):
    """u 0..1 from inner to outer corner along the upper lid -> world point on the skin"""
    f = (1 - u) if outer_sign < 0 else u
    i = f * (n_up - 1)
    a = int(math.floor(i)); bq = min(n_up - 1, a + 1); t = i - a
    px = upper[a, 0] * (1 - t) + upper[bq, 0] * t; py = upper[a, 1] * (1 - t) + upper[bq, 1] * t
    x, z = w_of(px, py)
    return Vector((x, surf_y(x, z), z))


def spline(points, radii):
    sp = cu.splines.new("POLY"); sp.points.add(len(points) - 1)
    for p, pt, r in zip(sp.points, points, radii):
        p.co = (pt.x, pt.y, pt.z, 1); p.radius = r
    sp.use_smooth = True


# liner: follows the lid, thicker toward the outer corner, ends in a small wing
pts, rad = [], []
for k in range(41):
    u = k / 40
    p = lid_pt(u) + Vector((0, -0.0011, 0.0003 + 0.0005 * u))
    pts.append(p); rad.append((1.6 + 3.4 * u ** 1.2) * PX)
wing = pts[-1] + Vector((outer_sign * 13 * PX, 0.0008, 9 * PX))
pts.append(wing); rad.append(0.3 * PX)
spline(pts, rad)
import random
rng = random.Random(5)


def one_lash(u, L, w, sweep, tip=None):
    base = lid_pt(u) + Vector((0, -0.0012, 0.0008 + 0.0005 * u))
    d = Vector((outer_sign * sweep, -0.55, 1.0)).normalized()
    tip = tip or (base + d * L + Vector((outer_sign * L * 0.25, -L * 0.15, -L * 0.1)))
    c1 = base + d * L * 0.5 + Vector((0, -L * 0.18, L * 0.12))
    P, Rr = [], []
    for q in range(9):
        t = q / 8
        P.append(base * (1 - t) ** 2 + c1 * 2 * (1 - t) * t + tip * t * t)
        Rr.append(max(0.15 * PX, w * (1 - t) ** 1.2))
    spline(P, Rr)
    return tip


NC = 7
for j in range(NC):
    u = 0.1 + 0.88 * (j / (NC - 1)) ** 0.85
    L = (20 + 24 * u) * PX * rng.uniform(0.9, 1.12)
    sw = 0.15 + 0.85 * u
    w = (3.6 + 2.2 * u) * PX
    tip = one_lash(u, L, w, sw)
    for k in range(1 + (u > 0.4)):
        du = rng.uniform(0.025, 0.045) * (1 if k == 0 else -1)
        one_lash(min(max(u + du, 0.03), 1.0), L * 0.92, w * 0.75, sw, tip + Vector((rng.uniform(-2, 2) * PX, 0, rng.uniform(-3, -1) * PX)))
    if j < NC - 1:
        one_lash(min(u + 0.5 * 0.88 / (NC - 1), 1.0), L * 0.55, w * 0.7, sw * 0.9)

# 6) renders
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
try:
    sc.eevee.taa_render_samples = 64
    sc.view_settings.view_transform = "Standard"
except Exception:
    pass
w_ = bpy.data.worlds.new("w"); sc.world = w_; w_.use_nodes = True
w_.node_tree.nodes["Background"].inputs["Color"].default_value = (0.5, 0.5, 0.55, 1)
w_.node_tree.nodes["Background"].inputs["Strength"].default_value = 0.6
h = cam["hgt"]
face = Vector((CX, yc, CZ))


def light(name, loc, energy, col, size):
    d = bpy.data.lights.new(name, "AREA"); d.energy = energy; d.color = col; d.size = size
    o = bpy.data.objects.new(name, d); o.location = loc
    o.rotation_euler = (face - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    sc.collection.objects.link(o)


light("key", face + Vector((-0.25, -0.6, 0.3)) * h, 5 * h * h, (1.0, 0.95, 0.9), 0.5 * h)
light("fill", face + Vector((0.4, -0.5, 0.0)) * h, 1.5 * h * h, (0.8, 0.85, 1.0), 0.6 * h)
cd = bpy.data.cameras.new("c"); co = bpy.data.objects.new("c", cd); sc.collection.objects.link(co); sc.camera = co
cd.clip_start = 0.001
eye_w = Vector((ex, yc, ez))
for name, tgt, ang, dist, lens, res in (("front", face, 0.0, 0.42, 85, (1400, 1000)), ("34", face, -0.5, 0.42, 85, (1400, 1000)),
                                        ("close", eye_w, 0.0, 0.11, 85, (1400, 900)), ("close34", eye_w, -0.6, 0.11, 85, (1400, 900))):
    co.location = tgt + Vector((math.sin(ang), -math.cos(ang), 0.02)) * h * dist
    co.rotation_euler = (tgt - co.location).to_track_quat("-Z", "Y").to_euler()
    cd.lens = lens
    sc.render.resolution_x, sc.render.resolution_y = res
    sc.render.filepath = os.path.join(L2, "eye3d_%s.png" % name)
    bpy.ops.render.render(write_still=True)
    say("rendered", name)
bpy.ops.wm.save_as_mainfile(filepath=os.path.join(L2, "arcane_eye3d.blend"))
say("done")
