# Blender 5.x: the new Arcanist ("fantasy witch 3d model.glb").
#   1. compares her with the current Arcanist source (faces, texture size,
#      matched renders) -> heroes/arcane_layers/compare_*.png + report.txt
#   2. splits her into toggleable layers by the colours of her texture:
#      Body (skin + hair), Robe, Armor (gold, metal, gems, staff), Hat
#   3. adds a smooth clay Base Form inside the clothing so the figure stays
#      solid with the robe and armor hidden (for proportion work in Solid view)
#   -> heroes/arcane_layers/arcanist_layers.blend (+ preview renders)
# Toggle a layer with the eye icon on its collection in the Outliner.
import bpy, bmesh, os, glob, math, time
import numpy as np
from mathutils import Vector, Matrix
from mathutils.bvhtree import BVHTree

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
OUT = os.path.join(HERE, "arcane_layers")
os.makedirs(OUT, exist_ok=True)
REP = open(os.path.join(OUT, "report.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s)
    REP.write(s + "\n")
    REP.flush()


NEW = None
for p in (os.path.join(ROOT, "heroes", "glb", "fantasy witch 3d model.glb"), os.path.expanduser("~/Downloads/fantasy witch 3d model.glb")):
    if os.path.exists(p):
        NEW = p
        break
OLD = os.path.join(ROOT, "heroes", "glb", "hero_arcane.glb")
say("new", NEW)
say("old", OLD)


def reset():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    sc = bpy.context.scene
    for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
        try:
            sc.render.engine = eng
            break
        except Exception:
            pass
    sc.render.resolution_x, sc.render.resolution_y = 900, 1200
    w = bpy.data.worlds.new("w")
    sc.world = w
    w.use_nodes = True
    w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.55, 0.55, 0.6, 1)
    w.node_tree.nodes["Background"].inputs["Strength"].default_value = 1.0
    d = bpy.data.lights.new("key", "SUN")
    d.energy = 2.5
    o = bpy.data.objects.new("key", d)
    o.rotation_euler = (0.9, 0.15, 0.5)
    sc.collection.objects.link(o)
    try:
        sc.view_settings.view_transform = "Standard"
    except Exception:
        pass
    return sc


def load(path):
    before = set(bpy.data.objects)
    bpy.ops.import_scene.gltf(filepath=path)
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
    return ob


def bounds(ob):
    P = np.empty(len(ob.data.vertices) * 3, dtype=np.float32)
    ob.data.vertices.foreach_get("co", P)
    P = P.reshape(-1, 3)
    return P.min(0), P.max(0)


def tex_of(ob):
    for m in ob.data.materials:
        if m and m.use_nodes:
            for n in m.node_tree.nodes:
                if n.type == "TEX_IMAGE" and n.image:
                    return n.image
    return None


def front_dir(ob):
    """Tripo files face -y or +x; pick the side with the face (most skin)"""
    return Vector((0, -1, 0))


def shoot(ob, name, face=False):
    sc = bpy.context.scene
    mn, mx = bounds(ob)
    h = mx[2] - mn[2]
    ctr = Vector(((mn[0] + mx[0]) / 2, (mn[1] + mx[1]) / 2, (mn[2] + mx[2]) / 2))
    cd = bpy.data.cameras.new("c")
    cam = bpy.data.objects.new("c", cd)
    sc.collection.objects.link(cam)
    sc.camera = cam
    for k, ang in enumerate((-90, 0, 90, 180)):
        a = math.radians(ang)
        if face:
            tgt = Vector((ctr.x, ctr.y, mn[2] + 0.86 * h))
            cam.location = tgt + Vector((math.cos(a), math.sin(a), 0.02)) * h * 0.55
            cd.lens = 85
        else:
            tgt = ctr
            cam.location = tgt + Vector((math.cos(a), math.sin(a), 0.1)) * h * 2.2
            cd.lens = 50
        cam.rotation_euler = (tgt - cam.location).to_track_quat("-Z", "Y").to_euler()
        sc.render.filepath = os.path.join(OUT, "%s_%s_%d.png" % (name, "face" if face else "full", k))
        bpy.ops.render.render(write_still=True)
    bpy.data.objects.remove(cam)


# ------------------------------------------------------------------ 1. compare
for tag, path in (("old", OLD), ("new", NEW)):
    if not path or not os.path.exists(path):
        say("missing", tag)
        continue
    reset()
    ob = load(path)
    img = tex_of(ob)
    mn, mx = bounds(ob)
    say("%s: faces %d verts %d texture %s size %s" % (tag, len(ob.data.polygons), len(ob.data.vertices),
        img.name if img else None, tuple(img.size) if img else None))
    shoot(ob, "compare_" + tag)
    shoot(ob, "compare_" + tag, face=True)

# ------------------------------------------------------------------ 2. layers
t0 = time.time()
sc = reset()
sc.render.resolution_x, sc.render.resolution_y = 900, 1200
ob = load(NEW)
ob.name = "Arcanist_full"
nf = len(ob.data.polygons)
dm = ob.modifiers.new("dec", "DECIMATE")
dm.ratio = min(1.0, 400000 / nf)
bpy.ops.object.modifier_apply(modifier="dec")
tri = ob.modifiers.new("tri", "TRIANGULATE")
bpy.ops.object.modifier_apply(modifier="tri")
me = ob.data
F = len(me.polygons)
say("layers: working mesh", F, "tris (from", nf, ")")
img = tex_of(ob)
W, H = img.size
px = np.empty(W * H * 4, dtype=np.float32)
img.pixels.foreach_get(px)
px = px.reshape(H, W, 4)[:, :, :3]
uv = np.empty(len(me.loops) * 2, dtype=np.float32)
me.uv_layers.active.data.foreach_get("uv", uv)
uv = uv.reshape(-1, 2)
xi = np.clip((uv[:, 0] % 1.0) * (W - 1), 0, W - 1).astype(np.int32)
yi = np.clip((uv[:, 1] % 1.0) * (H - 1), 0, H - 1).astype(np.int32)
col = px[yi, xi].reshape(F, 3, 3).mean(1)          # per triangle colour
mx_ = col.max(1)
mn_ = col.min(1)
v = mx_
s = np.where(mx_ > 1e-4, (mx_ - mn_) / np.maximum(mx_, 1e-4), 0)
r, g, b = col[:, 0], col[:, 1], col[:, 2]
hue = np.zeros(F)
d = np.maximum(mx_ - mn_, 1e-6)
hue = np.where(mx_ == r, ((g - b) / d) % 6, np.where(mx_ == g, (b - r) / d + 2, (r - g) / d + 4)) * 60
cen = np.empty(F * 3, dtype=np.float32)
me.polygons.foreach_get("center", cen)
cen = cen.reshape(-1, 3)
z0, z1 = cen[:, 2].min(), cen[:, 2].max()
zh = (cen[:, 2] - z0) / (z1 - z0)
BODY, ROBE, ARMOR, HAT = 0, 1, 2, 3
lab = np.full(F, ROBE)
skin = (hue < 40) & (s > 0.12) & (s < 0.62) & (v > 0.42) & (r > b)
gold = (hue > 30) & (hue < 62) & (s >= 0.4) & (v > 0.35)
gem = (s > 0.55) & (v > 0.5) & (hue > 170) & (hue < 260)
metal = (s < 0.12) & (v > 0.62)
hair = (v < 0.22) & (s < 0.35) & (zh > 0.72)
lab[skin | hair] = BODY
lab[gold | gem | metal] = ARMOR
# the hat: everything robe-coloured above the forehead
lab[(zh > 0.9) & (lab != BODY)] = HAT
# smooth the labels over neighbouring triangles
eidx = np.empty(len(me.loops), dtype=np.int32)
me.loops.foreach_get("edge_index", eidx)
fol = np.repeat(np.arange(F), 3)
order = np.argsort(eidx, kind="stable")
es, fs = eidx[order], fol[order]
same = es[1:] == es[:-1]
f1, f2 = fs[:-1][same], fs[1:][same]
for it in range(6):
    cnt = np.zeros((F, 4))
    np.add.at(cnt, (f1, lab[f2]), 1)
    np.add.at(cnt, (f2, lab[f1]), 1)
    cnt[np.arange(F), lab] += 1.6
    lab = cnt.argmax(1)
for k, nm in enumerate(("Body", "Robe", "Armor", "Hat")):
    say("  label", nm, int((lab == k).sum()))
# one material per layer (same texture), then split by material
base = me.materials[0]
me.materials.clear()
for nm in ("Body", "Robe", "Armor", "Hat"):
    m = base.copy()
    m.name = "Arcanist_" + nm
    me.materials.append(m)
me.polygons.foreach_set("material_index", lab.astype(np.int32))
me.update()
bpy.ops.object.select_all(action="DESELECT")
ob.select_set(True)
bpy.context.view_layer.objects.active = ob
bpy.ops.mesh.separate(type="MATERIAL")
parts = {}
for o in bpy.context.selected_objects:
    nm = o.data.materials[o.data.polygons[0].material_index].name.replace("Arcanist_", "") if len(o.data.polygons) else "Empty"
    o.name = "Arcanist_" + nm
    parts[nm] = o
cols = {}
for nm in ("Body", "Robe", "Armor", "Hat", "Base Form"):
    c = bpy.data.collections.new(nm)
    sc.collection.children.link(c)
    cols[nm] = c
for nm, o in parts.items():
    for c in list(o.users_collection):
        c.objects.unlink(o)
    cols.get(nm, sc.collection).objects.link(o)

# ------------------------------------------------------------------ 3. base form
# watertight copy of body+robe+armor, thin cloth removed, pulled inside and smoothed
keep = [parts[n] for n in ("Body", "Robe", "Armor") if n in parts]
cp = []
for o in keep:
    c = o.copy()
    c.data = o.data.copy()
    sc.collection.objects.link(c)
    cp.append(c)
bpy.ops.object.select_all(action="DESELECT")
for c in cp:
    c.select_set(True)
bpy.context.view_layer.objects.active = cp[0]
bpy.ops.object.join()
bf = bpy.context.view_layer.objects.active
bf.name = "Arcanist_BaseForm"
hgt = float(z1 - z0)
rm = bf.modifiers.new("rm", "REMESH")
rm.mode = "VOXEL"
rm.voxel_size = 0.006 * hgt
bpy.ops.object.modifier_apply(modifier="rm")
# drop thin sheets (capes, sleeves, skirts): measure thickness inward
bm = bmesh.new()
bm.from_mesh(bf.data)
bm.verts.ensure_lookup_table()
bm.normal_update()
tree = BVHTree.FromBMesh(bm)
thin = []
for vt in bm.verts:
    o_ = vt.co - vt.normal * (0.002 * hgt)
    hit = tree.ray_cast(o_, -vt.normal, 0.2 * hgt)
    if hit[0] is None or hit[3] < 0.03 * hgt:
        thin.append(vt)
bmesh.ops.delete(bm, geom=thin, context="VERTS")
# keep the biggest connected piece only (the body)
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
bmesh.ops.holes_fill(bm, edges=[e for e in bm.edges if e.is_boundary], sides=0)
bm.to_mesh(bf.data)
bm.free()
for m_ in ("sm", "sh"):
    pass
sm = bf.modifiers.new("sm", "SMOOTH")
sm.factor, sm.iterations = 0.8, 12
bpy.ops.object.modifier_apply(modifier="sm")
sh = bf.modifiers.new("sh", "DISPLACE")
sh.strength = -0.006 * hgt
bpy.ops.object.modifier_apply(modifier="sh")
bf.data.materials.clear()
clay = bpy.data.materials.new("BaseForm_Clay")
clay.diffuse_color = (0.72, 0.68, 0.66, 1)
clay.use_nodes = True
clay.node_tree.nodes["Principled BSDF"].inputs["Base Color"].default_value = (0.72, 0.68, 0.66, 1)
clay.node_tree.nodes["Principled BSDF"].inputs["Roughness"].default_value = 0.7
bf.data.materials.append(clay)
bpy.ops.object.shade_smooth()
for c in list(bf.users_collection):
    c.objects.unlink(bf)
cols["Base Form"].objects.link(bf)
say("base form verts", len(bf.data.vertices), "in", int(time.time() - t0), "s")

# ------------------------------------------------------------------ previews and save
def show(names):
    for nm, c in cols.items():
        c.hide_render = nm not in names
        c.hide_viewport = nm not in names
        lc = bpy.context.view_layer.layer_collection.children.get(c.name)
        if lc:
            lc.hide_viewport = nm not in names


allp = list(parts.values())
for tag, names in (("all", ("Body", "Robe", "Armor", "Hat")), ("no_robe", ("Body", "Armor", "Hat", "Base Form")),
                   ("no_armor", ("Body", "Robe", "Hat", "Base Form")), ("body_only", ("Body", "Base Form")),
                   ("layers_robe", ("Robe",)), ("layers_armor", ("Armor",))):
    show(names)
    shoot(allp[0] if tag != "layers_armor" else parts.get("Armor", allp[0]), "layer_" + tag)
show(("Body", "Robe", "Armor", "Hat"))
for nm in ("Base Form",):
    cols[nm].hide_viewport = True
    lc = bpy.context.view_layer.layer_collection.children.get(nm)
    if lc:
        lc.hide_viewport = True
bpy.ops.wm.save_as_mainfile(filepath=os.path.join(OUT, "arcanist_layers.blend"))
say("saved arcanist_layers.blend")
open(os.path.join(OUT, "done.txt"), "w").write("ok")
