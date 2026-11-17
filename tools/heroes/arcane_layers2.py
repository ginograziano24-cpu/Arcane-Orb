# Blender 5.x: Arcanist layers, second pass, plus a flat front render of her
# face for the eye clean-up.
#   heroes/arcane_layers2/arcanist_layers.blend   Body / Robe / Armor / Hat / Base Form
#   heroes/arcane_layers2/eyes_front.png          texture colours, front, orthographic
#   heroes/arcane_layers2/eyes_cam.json            camera for projecting the fix back
# Changes from the first pass: silver hair counts as Body (it was going to the
# robe), purple cloth is told apart by hue, and the Base Form is built from the
# skin plus only the clothing that hugs the body (corset, sleeves on the arm,
# boots), not the cape or the loose skirt panels.
import bpy, bmesh, os, math, time, json
import numpy as np
from mathutils import Vector
from mathutils.bvhtree import BVHTree

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
OUT = os.path.join(HERE, "arcane_layers2")
os.makedirs(OUT, exist_ok=True)
REP = open(os.path.join(OUT, "report.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s)
    REP.write(s + "\n")
    REP.flush()


SRC = os.path.join(ROOT, "heroes", "glb", "fantasy witch 3d model.glb")
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
try:
    sc.view_settings.view_transform = "Standard"
except Exception:
    pass
w = bpy.data.worlds.new("w")
sc.world = w
w.use_nodes = True
w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.55, 0.55, 0.6, 1)

before = set(bpy.data.objects)
bpy.ops.import_scene.gltf(filepath=SRC)
new = [o for o in bpy.data.objects if o not in before]
meshes = [o for o in new if o.type == "MESH"]
bpy.ops.object.select_all(action="DESELECT")
for m in meshes:
    m.select_set(True)
bpy.context.view_layer.objects.active = meshes[0]
if len(meshes) > 1:
    bpy.ops.object.join()
ob = bpy.context.view_layer.objects.active
keep = ob.matrix_world.copy()
ob.parent = None
ob.matrix_world = keep
bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
for o in new:
    if o != ob and o.name in bpy.data.objects:
        bpy.data.objects.remove(o)
me = ob.data
P = np.empty(len(me.vertices) * 3, dtype=np.float32)
me.vertices.foreach_get("co", P)
P = P.reshape(-1, 3)
mn, mx = P.min(0), P.max(0)
z0, z1 = float(mn[2]), float(mx[2])
hgt = z1 - z0
img = None
for m in me.materials:  # the colour texture (the one feeding Base Color), not the normal map
    for n in m.node_tree.nodes:
        if n.type == "TEX_IMAGE" and n.image:
            feeds_color = any(l.to_socket.name == "Base Color" for l in n.outputs["Color"].links)
            if feeds_color or (img is None and "normal" not in n.image.name.lower()):
                img = n.image

# ------------------------------------------------------------------ flat front face render (full detail mesh)
mat = me.materials[0]
nt = mat.node_tree
outn = next(n for n in nt.nodes if n.type == "OUTPUT_MATERIAL")
texn = next(n for n in nt.nodes if n.type == "TEX_IMAGE" and n.image == img)
emi = nt.nodes.new("ShaderNodeEmission")
nt.links.new(texn.outputs["Color"], emi.inputs["Color"])
old_link = outn.inputs["Surface"].links[0].from_socket if outn.inputs["Surface"].links else None
nt.links.new(emi.outputs[0], outn.inputs["Surface"])
# find the face: highest dense patch of skin seen from the front, around 85% height
cd = bpy.data.cameras.new("eyecam")
cd.type = "ORTHO"
cam = bpy.data.objects.new("eyecam", cd)
sc.collection.objects.link(cam)
sc.camera = cam
fz = z0 + 0.866 * hgt
head = P[(P[:, 2] > z0 + 0.8 * hgt) & (P[:, 2] < z0 + 0.92 * hgt)]
cx = float(np.median(head[:, 0]))
front_y = float(head[:, 1].min())
cam.location = (cx, front_y - 0.5 * hgt, fz)
cam.rotation_euler = (math.pi / 2, 0, 0)
cd.ortho_scale = 0.18 * hgt
cd.clip_end = hgt * 3
sc.render.resolution_x, sc.render.resolution_y = 2400, 1200
sc.render.film_transparent = False
sc.render.filepath = os.path.join(OUT, "eyes_front.png")
bpy.ops.render.render(write_still=True)
json.dump({"loc": list(cam.location), "ortho_scale": cd.ortho_scale, "res": [2400, 1200], "hgt": hgt, "z0": z0},
          open(os.path.join(OUT, "eyes_cam.json"), "w"))
say("eye render at z", round(fz, 4), "cx", round(cx, 4))
# a wider one for the hair around her face
cd.ortho_scale = 0.42 * hgt
cam.location = (cx, front_y - 0.5 * hgt, z0 + 0.84 * hgt)
sc.render.resolution_x, sc.render.resolution_y = 1600, 1600
sc.render.filepath = os.path.join(OUT, "hair_front.png")
bpy.ops.render.render(write_still=True)
# her texture as it is, for the clean-up passes
img.filepath_raw = os.path.join(OUT, "base_texture.png")
img.file_format = "PNG"
img.save()
say("texture", img.size[0], img.size[1])
if old_link:
    nt.links.new(old_link, outn.inputs["Surface"])
nt.nodes.remove(emi)
bpy.data.objects.remove(cam)

# ------------------------------------------------------------------ layers v2
t0 = time.time()
dm = ob.modifiers.new("dec", "DECIMATE")
dm.ratio = min(1.0, 400000 / len(me.polygons))
bpy.ops.object.modifier_apply(modifier="dec")
tri = ob.modifiers.new("tri", "TRIANGULATE")
bpy.ops.object.modifier_apply(modifier="tri")
me = ob.data
F = len(me.polygons)
W, H = img.size
px = np.empty(W * H * 4, dtype=np.float32)
img.pixels.foreach_get(px)
px = px.reshape(H, W, 4)[:, :, :3]
uv = np.empty(len(me.loops) * 2, dtype=np.float32)
me.uv_layers.active.data.foreach_get("uv", uv)
uv = uv.reshape(-1, 2)
xi = np.clip((uv[:, 0] % 1.0) * (W - 1), 0, W - 1).astype(np.int32)
yi = np.clip((uv[:, 1] % 1.0) * (H - 1), 0, H - 1).astype(np.int32)
col = px[yi, xi].reshape(F, 3, 3).mean(1)
r, g, b = col[:, 0], col[:, 1], col[:, 2]
mx_, mn_ = col.max(1), col.min(1)
v = mx_
s = np.where(mx_ > 1e-4, (mx_ - mn_) / np.maximum(mx_, 1e-4), 0)
d = np.maximum(mx_ - mn_, 1e-6)
hue = np.where(mx_ == r, ((g - b) / d) % 6, np.where(mx_ == g, (b - r) / d + 2, (r - g) / d + 4)) * 60
cen = np.empty(F * 3, dtype=np.float32)
me.polygons.foreach_get("center", cen)
cen = cen.reshape(-1, 3)
zh = (cen[:, 2] - z0) / hgt
BODY, ROBE, ARMOR, HAT = 0, 1, 2, 3
lab = np.full(F, ROBE)
skin = (hue < 42) & (s > 0.1) & (s < 0.65) & (v > 0.38) & (r > b * 1.05)
hair = (s < 0.22) & (zh > 0.5) & (zh < 0.9) & (v > 0.12)
purple = (hue > 235) & (hue < 320) & (s > 0.18)
gold = (hue > 28) & (hue < 65) & (s >= 0.38) & (v > 0.3)
gem = (s > 0.5) & (v > 0.45) & (hue > 165) & (hue < 235)
metal = (s < 0.1) & (v > 0.7) & (zh < 0.5)
lab[skin | (hair & ~purple)] = BODY
lab[gold | gem | metal] = ARMOR
lab[purple & ~(gold | gem)] = ROBE
lab[(zh > 0.885) & ~skin & ~hair] = HAT
eidx = np.empty(len(me.loops), dtype=np.int32)
me.loops.foreach_get("edge_index", eidx)
fol = np.repeat(np.arange(F), 3)
order = np.argsort(eidx, kind="stable")
es, fs = eidx[order], fol[order]
same = es[1:] == es[:-1]
f1, f2 = fs[:-1][same], fs[1:][same]
for it in range(8):
    cnt = np.zeros((F, 4))
    np.add.at(cnt, (f1, lab[f2]), 1)
    np.add.at(cnt, (f2, lab[f1]), 1)
    cnt[np.arange(F), lab] += 1.6
    lab = cnt.argmax(1)
# hair triangles in texture space (for the hair clean-up mask)
hair_f = np.where((lab == BODY) & (s < 0.25))[0]
np.save(os.path.join(OUT, "hair_uv.npy"), uv.reshape(F, 3, 2)[hair_f].astype(np.float32))
skin_f = np.where((lab == BODY) & skin)[0]
np.save(os.path.join(OUT, "skin_uv.npy"), uv.reshape(F, 3, 2)[skin_f].astype(np.float32))
names = ("Body", "Robe", "Armor", "Hat")
for k, nm in enumerate(names):
    say("  label", nm, int((lab == k).sum()))
base = me.materials[0]
me.materials.clear()
for nm in names:
    m = base.copy()
    m.name = "Arcanist_" + nm
    me.materials.append(m)
me.polygons.foreach_set("material_index", lab.astype(np.int32))
me.update()
# keep the labels on the full mesh for the base form step
full_lab = lab.copy()
full_cen = cen.copy()
bpy.ops.object.select_all(action="DESELECT")
ob.select_set(True)
bpy.context.view_layer.objects.active = ob
bpy.ops.mesh.separate(type="MATERIAL")
parts = {}
for o in bpy.context.selected_objects:
    if not len(o.data.polygons):
        continue
    nm = o.data.materials[o.data.polygons[0].material_index].name.replace("Arcanist_", "")
    o.name = "Arcanist_" + nm
    parts[nm] = o
cols = {}
for nm in names + ("Base Form",):
    c = bpy.data.collections.new(nm)
    sc.collection.children.link(c)
    cols[nm] = c
for nm, o in parts.items():
    for c in list(o.users_collection):
        c.objects.unlink(o)
    cols[nm].objects.link(o)

# ------------------------------------------------------------------ base form v2
body = parts["Body"]
bvh_body = BVHTree.FromPolygons([vv.co for vv in body.data.vertices], [tuple(p.vertices) for p in body.data.polygons])
tight = []
for nm in ("Robe", "Armor"):
    o = parts.get(nm)
    if not o:
        continue
    bm = bmesh.new()
    bm.from_mesh(o.data)
    bm.faces.ensure_lookup_table()
    far = []
    for f_ in bm.faces:
        c = f_.calc_center_median()
        hit = bvh_body.find_nearest(c, 0.06 * hgt)
        if hit[0] is None:
            far.append(f_)
    bmesh.ops.delete(bm, geom=far, context="FACES")
    m2 = bpy.data.meshes.new("tight_" + nm)
    bm.to_mesh(m2)
    bm.free()
    t_ = bpy.data.objects.new("tight_" + nm, m2)
    sc.collection.objects.link(t_)
    tight.append(t_)
bc = body.copy()
bc.data = body.data.copy()
sc.collection.objects.link(bc)
bpy.ops.object.select_all(action="DESELECT")
for o in tight + [bc]:
    o.select_set(True)
bpy.context.view_layer.objects.active = bc
bpy.ops.object.join()
bf = bpy.context.view_layer.objects.active
bf.name = "Arcanist_BaseForm"
sol = bf.modifiers.new("sol", "SOLIDIFY")
sol.thickness = 0.012 * hgt
sol.offset = -1
bpy.ops.object.modifier_apply(modifier="sol")
rm = bf.modifiers.new("rm", "REMESH")
rm.mode = "VOXEL"
rm.voxel_size = 0.007 * hgt
bpy.ops.object.modifier_apply(modifier="rm")
# biggest connected piece only
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
sm.factor, sm.iterations = 0.7, 10
bpy.ops.object.modifier_apply(modifier="sm")
sh = bf.modifiers.new("sh", "DISPLACE")
sh.strength = -0.004 * hgt
bpy.ops.object.modifier_apply(modifier="sh")
bf.data.materials.clear()
clay = bpy.data.materials.new("BaseForm_Clay")
clay.diffuse_color = (0.72, 0.68, 0.66, 1)
clay.use_nodes = True
pb = clay.node_tree.nodes["Principled BSDF"]
pb.inputs["Base Color"].default_value = (0.72, 0.68, 0.66, 1)
pb.inputs["Roughness"].default_value = 0.7
bf.data.materials.append(clay)
bpy.ops.object.shade_smooth()
for c in list(bf.users_collection):
    c.objects.unlink(bf)
cols["Base Form"].objects.link(bf)
say("base form verts", len(bf.data.vertices), "in", int(time.time() - t0), "s")

# ------------------------------------------------------------------ previews and save
d_ = bpy.data.lights.new("key", "SUN")
d_.energy = 2.5
lo = bpy.data.objects.new("key", d_)
lo.rotation_euler = (0.9, 0.15, 0.5)
sc.collection.objects.link(lo)
cd = bpy.data.cameras.new("c")
cam = bpy.data.objects.new("c", cd)
sc.collection.objects.link(cam)
sc.camera = cam
sc.render.resolution_x, sc.render.resolution_y = 900, 1200
ctr = Vector(((mn[0] + mx[0]) / 2, (mn[1] + mx[1]) / 2, (z0 + z1) / 2))


def show(nms):
    for nm, c in cols.items():
        c.hide_render = nm not in nms
        lc = bpy.context.view_layer.layer_collection.children.get(c.name)
        if lc:
            lc.hide_viewport = nm not in nms


for tag, nms in (("all", ("Body", "Robe", "Armor", "Hat")), ("no_robe", ("Body", "Armor", "Hat", "Base Form")),
                 ("no_armor", ("Body", "Robe", "Hat", "Base Form")), ("no_clothes", ("Body", "Base Form")),
                 ("base_only", ("Base Form",)), ("robe_only", ("Robe",)), ("armor_only", ("Armor",)), ("hat_only", ("Hat",))):
    show(nms)
    for k, ang in enumerate((-90, 0)):
        a = math.radians(ang)
        cam.location = ctr + Vector((math.cos(a), math.sin(a), 0.1)) * hgt * 2.2
        cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
        sc.render.filepath = os.path.join(OUT, "layer_%s_%d.png" % (tag, k))
        bpy.ops.render.render(write_still=True)
show(("Body", "Robe", "Armor", "Hat"))
lc = bpy.context.view_layer.layer_collection.children.get("Base Form")
if lc:
    lc.hide_viewport = True
bpy.data.objects.remove(cam)
bpy.data.objects.remove(lo)
bpy.ops.wm.save_as_mainfile(filepath=os.path.join(OUT, "arcanist_layers.blend"))
say("saved")
open(os.path.join(OUT, "done.txt"), "w").write("ok")
