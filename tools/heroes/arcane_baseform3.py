# Blender 5.x: Base Form, third pass, in heroes/arcane_layers2/arcanist_layers.blend.
# Built from her own skin plus only the clothing that hugs her body (corset,
# bodice, boots), found slice by slice: a clothing face is kept only if it sits
# inside the outline her skin makes at that height (or just above/below), so the
# cape, wide sleeves, skirt panels, hat and staff stay out. The kept surfaces are
# thickened inward, voxel-remeshed into one solid and smoothed.
import bpy, bmesh, os, math, time
import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
L2 = os.path.join(HERE, "arcane_layers2")
BLEND = os.path.join(L2, "arcanist_layers.blend")
log = open(os.path.join(L2, "baseform3_log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s)
    log.write(s + "\n")
    log.flush()


bpy.ops.wm.open_mainfile(filepath=BLEND)
sc = bpy.context.scene
obs = {o.name: o for o in bpy.data.objects}
body, robe, armor = obs["Arcanist_Body"], obs.get("Arcanist_Robe"), obs.get("Arcanist_Armor")
old = obs.get("Arcanist_BaseForm")


def world_pts(o):
    co = np.empty(len(o.data.vertices) * 3, dtype=np.float32)
    o.data.vertices.foreach_get("co", co)
    co = co.reshape(-1, 3)
    M = np.array(o.matrix_world)
    return co @ M[:3, :3].T + M[:3, 3]


def face_centers(o):
    c = np.empty(len(o.data.polygons) * 3, dtype=np.float32)
    o.data.polygons.foreach_get("center", c)
    c = c.reshape(-1, 3)
    M = np.array(o.matrix_world)
    return c @ M[:3, :3].T + M[:3, 3]


def skin_faces(o):
    me = o.data
    img = None
    for n in o.data.materials[0].node_tree.nodes:
        if n.type == "TEX_IMAGE" and n.image and any(l.to_socket.name == "Base Color" for l in n.outputs["Color"].links):
            img = n.image
    W, H = img.size
    px = np.empty(W * H * 4, dtype=np.float32)
    img.pixels.foreach_get(px)
    px = px.reshape(H, W, 4)[:, :, :3]
    F = len(me.polygons)
    uv = np.empty(len(me.loops) * 2, dtype=np.float32)
    me.uv_layers.active.data.foreach_get("uv", uv)
    uv = uv.reshape(-1, 2)
    xi = np.clip((uv[:, 0] % 1.0) * (W - 1), 0, W - 1).astype(np.int32)
    yi = np.clip((uv[:, 1] % 1.0) * (H - 1), 0, H - 1).astype(np.int32)
    col = px[yi, xi].reshape(F, 3, 3).mean(1)
    r, g, b = col[:, 0], col[:, 1], col[:, 2]
    mx, mn = col.max(1), col.min(1)
    s_ = np.where(mx > 1e-4, (mx - mn) / np.maximum(mx, 1e-4), 0)
    return (r > b * 1.05) & (s_ > 0.1) & (mx > 0.35)


SKIN = skin_faces(body)
say("skin faces", int(SKIN.sum()), "of", len(SKIN))
bmb = bmesh.new()
bmb.from_mesh(body.data)
bmb.faces.ensure_lookup_table()
bmesh.ops.delete(bmb, geom=[f for f, k in zip(bmb.faces, SKIN) if not k], context="FACES")
skin_me = bpy.data.meshes.new("skin_only")
bmb.to_mesh(skin_me)
bmb.free()
skin_ob = bpy.data.objects.new("skin_only", skin_me)
skin_ob.matrix_world = body.matrix_world
sc.collection.objects.link(skin_ob)
bp = world_pts(body)
allp = np.concatenate([bp] + [world_pts(o) for o in (robe, armor) if o])
z0, z1 = allp[:, 2].min(), allp[:, 2].max()
h = z1 - z0
x0, y0 = allp[:, 0].min(), allp[:, 1].min()
CELL = 0.012 * h
NZ = 60
nx = int((allp[:, 0].max() - x0) / CELL) + 3
ny = int((allp[:, 1].max() - y0) / CELL) + 3
# skin occupancy per height slice (skin only below the hat band; hair excluded by keeping skin colour faces)
bc = face_centers(body)[SKIN]
# body layer = skin + hair; hair hangs wide, so use only faces whose texture-free test is "not hair":
# hair sits outside the torso at shoulder height; keep everything, the slice masks get dilated little
occ = np.zeros((NZ, nx, ny), bool)
iz = np.clip(((bc[:, 2] - z0) / h * NZ).astype(int), 0, NZ - 1)
ix = ((bc[:, 0] - x0) / CELL).astype(int)
iy = ((bc[:, 1] - y0) / CELL).astype(int)
occ[iz, ix, iy] = True


def dilate2(m, k):
    out = m.copy()
    for _ in range(k):
        o2 = out.copy()
        o2[1:, :] |= out[:-1, :]; o2[:-1, :] |= out[1:, :]
        o2[:, 1:] |= out[:, :-1]; o2[:, :-1] |= out[:, 1:]
        out = o2
    return out


def fill2(m):
    """fill enclosed holes in a 2D mask (flood from the border)"""
    H_, W_ = m.shape
    outside = np.zeros_like(m)
    stack = [(i, j) for i in range(H_) for j in (0, W_ - 1)] + [(i, j) for j in range(W_) for i in (0, H_ - 1)]
    while stack:
        i, j = stack.pop()
        if i < 0 or j < 0 or i >= H_ or j >= W_ or outside[i, j] or m[i, j]:
            continue
        outside[i, j] = True
        stack += [(i + 1, j), (i - 1, j), (i, j + 1), (i, j - 1)]
    return ~outside


masks = np.zeros_like(occ)
for k in range(NZ):
    near = occ[max(0, k - 3):k + 4].any(0)       # skin from neighbouring heights covers the corset band
    m = dilate2(near, 2)
    masks[k] = fill2(m)
keep_objs = []
for o in (robe, armor):
    if not o:
        continue
    fc = face_centers(o)
    kz = np.clip(((fc[:, 2] - z0) / h * NZ).astype(int), 0, NZ - 1)
    kx = np.clip(((fc[:, 0] - x0) / CELL).astype(int), 0, nx - 1)
    ky = np.clip(((fc[:, 1] - y0) / CELL).astype(int), 0, ny - 1)
    inside = masks[kz, kx, ky] & ((fc[:, 2] - z0) / h < 0.86)
    say(o.name, "tight faces", int(inside.sum()), "of", len(fc))
    bm = bmesh.new()
    bm.from_mesh(o.data)
    bm.faces.ensure_lookup_table()
    bmesh.ops.delete(bm, geom=[f for f, k in zip(bm.faces, inside) if not k], context="FACES")
    m2 = bpy.data.meshes.new("tight")
    bm.to_mesh(m2)
    bm.free()
    t = bpy.data.objects.new("tight", m2)
    t.matrix_world = o.matrix_world
    sc.collection.objects.link(t)
    keep_objs.append(t)
# skin copy without the long hair: drop body faces outside the slice masks' core (hair falls wide of the skin outline only partly; keep all)
b2 = skin_ob
bpy.ops.object.select_all(action="DESELECT")
for o in keep_objs + [b2]:
    o.select_set(True)
bpy.context.view_layer.objects.active = b2
bpy.ops.object.join()
bf = bpy.context.view_layer.objects.active
sol = bf.modifiers.new("sol", "SOLIDIFY")
sol.thickness = 0.03 * h
sol.offset = -1
bpy.ops.object.modifier_apply(modifier="sol")
rm = bf.modifiers.new("rm", "REMESH")
rm.mode = "VOXEL"
rm.voxel_size = 0.006 * h
bpy.ops.object.modifier_apply(modifier="rm")
# biggest piece only
bm = bmesh.new()
bm.from_mesh(bf.data)
bm.verts.ensure_lookup_table()
seen, best = set(), []
for vt in bm.verts:
    if vt in seen:
        continue
    comp, stack = [], [vt]
    seen.add(vt)
    while stack:
        x = stack.pop()
        comp.append(x)
        for e in x.link_edges:
            y = e.other_vert(x)
            if y not in seen:
                seen.add(y)
                stack.append(y)
    if len(comp) > len(best):
        best = comp
bs = set(best)
bmesh.ops.delete(bm, geom=[x for x in bm.verts if x not in bs], context="VERTS")
bm.to_mesh(bf.data)
bm.free()
sm = bf.modifiers.new("sm", "SMOOTH")
sm.factor, sm.iterations = 0.6, 8
bpy.ops.object.modifier_apply(modifier="sm")
sh = bf.modifiers.new("sh", "DISPLACE")
sh.strength = -0.003 * h
bpy.ops.object.modifier_apply(modifier="sh")
bf.data.materials.clear()
bf.data.materials.append(bpy.data.materials.get("BaseForm_Clay") or bpy.data.materials.new("BaseForm_Clay"))
bpy.ops.object.shade_smooth()
col = bpy.data.collections["Base Form"]
for c in list(bf.users_collection):
    c.objects.unlink(bf)
col.objects.link(bf)
if old:
    bpy.data.objects.remove(old)
bf.name = "Arcanist_BaseForm"
say("base form verts", len(bf.data.vertices))
# previews
d_ = bpy.data.lights.new("key", "SUN"); d_.energy = 2.5
lo = bpy.data.objects.new("key", d_); lo.rotation_euler = (0.9, 0.15, 0.5); sc.collection.objects.link(lo)
cd = bpy.data.cameras.new("c"); cam = bpy.data.objects.new("c", cd); sc.collection.objects.link(cam); sc.camera = cam
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
sc.render.resolution_x, sc.render.resolution_y = 900, 1200
from mathutils import Vector
ctr = Vector(((allp[:, 0].min() + allp[:, 0].max()) / 2, (allp[:, 1].min() + allp[:, 1].max()) / 2, (z0 + z1) / 2))
layer = bpy.context.view_layer.layer_collection.children


def show(nms):
    for c in bpy.data.collections:
        if c.name in ("Body", "Robe", "Armor", "Hat", "Base Form"):
            c.hide_render = c.name not in nms
            if layer.get(c.name):
                layer[c.name].hide_viewport = c.name not in nms


for tag, nms in (("base_only", ("Base Form",)), ("no_clothes", ("Body", "Base Form")), ("no_robe", ("Body", "Armor", "Hat", "Base Form"))):
    show(nms)
    for k, ang in enumerate((-90, 0)):
        a = math.radians(ang)
        cam.location = ctr + Vector((math.cos(a), math.sin(a), 0.1)) * h * 2.2
        cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
        sc.render.filepath = os.path.join(L2, "bf3_%s_%d.png" % (tag, k))
        bpy.ops.render.render(write_still=True)
show(("Body", "Robe", "Armor", "Hat"))
bpy.data.objects.remove(cam); bpy.data.objects.remove(lo)
bpy.ops.wm.save_as_mainfile(filepath=BLEND)
say("saved")
open(os.path.join(L2, "bf3_done.txt"), "w").write("ok")
