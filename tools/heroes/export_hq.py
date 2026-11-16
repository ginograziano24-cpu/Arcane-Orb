# Blender 5.2 headless: high-quality animated heroes for the game.
#   blender -b -P export_hq.py -- [ids...]
#
# The previous game heroes came from Tripo's 20k retopology with its own
# 2048px texture, where the face got only a few hundred pixels. This rebuilds
# every hero from the ORIGINAL full-detail Tripo model (heroes/glb/hero_<id>.glb,
# ~1.7M faces, 4K texture), so faces and proportions are exactly the source's:
#   1. the rigged Tripo model (heroes/rigged/<id>.glb, arcane_rigged.glb for
#      the Arcanist, with her staff made rigid) supplies the skeleton and weights
#   2. the original is aligned onto it, welded and decimated to TARGET faces
#   3. fresh UVs, with the head's islands enlarged so the face gets far more
#      texture, then a 4096px bake of the original's colour
#   4. skin weights transferred from the rigged model, nearest surface
#   5. written as Source/assets/heroes/<id>.orbs + <id>.jpg (same format as
#      export_anim.py, so the game loads it unchanged)
import bpy, bmesh, os, sys, math, struct, zlib, array, time
from mathutils import Vector, Matrix
from mathutils.bvhtree import BVHTree

HERE = os.path.dirname(os.path.abspath(__file__))          # heroes/
ROOT = os.path.dirname(HERE)
OUT = os.path.join(ROOT, "Source", "assets", "heroes")
RIG = os.path.join(HERE, "rigged")
GLB = os.path.join(HERE, "glb")
os.makedirs(OUT, exist_ok=True)
argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
IDS = argv or ["guardian", "berserker", "arcane", "engineer", "lucky", "pyro", "storm"]
TARGET = 60000        # faces after decimation
TEX = 4096
HEAD_Z = 0.80         # islands above this fraction of the height count as head
HEAD_K = 2.6          # how much bigger the head's UV islands get before packing
FRAMES, FPS = 72, 24.0
NEWER = {"arcane": ["fantasy witch 3d model.glb", "arcane_new.glb"]}
log = open(os.path.join(HERE, "hq-log.txt"), "a")


def say(*a):
    s = time.strftime("%H:%M:%S ") + " ".join(str(x) for x in a)
    print(s, flush=True)
    log.write(s + "\n")
    log.flush()


def clear():
    bpy.ops.wm.read_factory_settings(use_empty=True)


def import_glb(path):
    before = set(bpy.context.scene.objects)
    bpy.ops.import_scene.gltf(filepath=path)
    return [o for o in bpy.context.scene.objects if o not in before]


def bone(arm, part):
    for b in arm.pose.bones:
        if b.name.split(":")[-1] == part:
            return b
    return None


def world_verts(o):
    mw = o.matrix_world
    return [mw @ v.co for v in o.data.vertices]


def bbox(P):
    xs = [p.x for p in P]; ys = [p.y for p in P]; zs = [p.z for p in P]
    return (min(xs), max(xs), min(ys), max(ys), min(zs), max(zs))


def select_only(objs, active):
    bpy.ops.object.select_all(action="DESELECT")
    for o in objs:
        o.select_set(True)
    bpy.context.view_layer.objects.active = active


def remove_specks(me, min_verts=12):
    bm = bmesh.new(); bm.from_mesh(me); bm.verts.ensure_lookup_table()
    seen = set(); kill = []
    for v in bm.verts:
        if v.index in seen:
            continue
        stack = [v]; part = []; seen.add(v.index)
        while stack:
            a = stack.pop(); part.append(a)
            for e in a.link_edges:
                b = e.other_vert(a)
                if b.index not in seen:
                    seen.add(b.index); stack.append(b)
        if len(part) < min_verts:
            kill += part
    if kill:
        bmesh.ops.delete(bm, geom=kill, context="VERTS")
        bm.to_mesh(me)
    bm.free()
    return len(kill)


def fix_arcane(me_obj, arm):
    """Staff rigid on the hand that holds it (same rule as export_anim.py)."""
    me = me_obj.data
    mw = me_obj.matrix_world
    zs = [(mw @ v.co).z for v in me.vertices]; h0 = max(zs) - min(zs)
    bm = bmesh.new(); bm.from_mesh(me)
    bmesh.ops.remove_doubles(bm, verts=bm.verts[:], dist=0.0004 * h0 / max(1e-9, mw.to_scale().z))
    bm.to_mesh(me); bm.free()
    P = [mw @ v.co for v in me.vertices]
    zs = [p.z for p in P]; z0, z1 = min(zs), max(zs); h = z1 - z0
    band = [p for p in P if z0 + 0.15 * h < p.z < z0 + 0.32 * h]
    xmin, xmax = min(p.x for p in band), max(p.x for p in band)
    pts = [p for p in band if abs(p.x - xmin) < 0.035 * h]
    ax = sum(p.x for p in pts) / len(pts); ay = sum(p.y for p in pts) / len(pts)
    hands = [b for b in arm.data.bones if b.name.endswith("Hand")]
    hand = min(hands, key=lambda b: ((arm.matrix_world @ b.head_local).xy - Vector((ax, ay))).length)
    hz = (arm.matrix_world @ hand.head_local).z
    picked = []
    for i, p in enumerate(P):
        d = math.hypot(p.x - ax, p.y - ay)
        if (d < 0.012 * h and p.z < z0 + 0.62 * h) or (d < 0.09 * h and p.z > z0 + 0.62 * h and abs(p.x - ax) < 0.09 * h and p.z > hz + 0.05 * h):
            picked.append(i)
    vg = me_obj.vertex_groups
    g = vg.get(hand.name) or vg.new(name=hand.name)
    for i in picked:
        for other in vg:
            if other != g:
                other.remove([i])
        g.add([i], 1.0, "REPLACE")
    say("arcane: staff verts", len(picked), "->", hand.name, "specks", remove_specks(me))


def idle_pose(arm, k):
    t = k / FRAMES * 2 * math.pi

    def rot(part, x=0.0, y=0.0, z=0.0):
        b = bone(arm, part)
        if b is None:
            return
        b.rotation_mode = "XYZ"; b.rotation_euler = (math.radians(x), math.radians(y), math.radians(z))
    br = math.sin(t)
    sw = math.sin(t + 0.6)
    rot("Spine", 1.2 * br, 0, 0)
    rot("Spine1", 1.6 * br, 1.0 * math.sin(t + 1.3), 0)
    rot("Spine2", 1.4 * br, 0, 0)
    rot("Neck", -1.2 * br, 0, 0)
    rot("Head", -0.8 * br + 1.2 * math.sin(t + 2.0), 2.0 * math.sin(t + 0.4), 0)
    rot("LeftShoulder", 0, 0, 1.2 * br); rot("RightShoulder", 0, 0, -1.2 * br)
    rot("LeftArm", 2.0 * sw, 0, 1.5 * br); rot("RightArm", 2.0 * sw, 0, -1.5 * br)
    rot("LeftForeArm", 2.5 * sw, 0, 0); rot("RightForeArm", 2.5 * sw, 0, 0)
    b = bone(arm, "Hips")
    if b is not None:
        b.location = (0, 0, 0)


def align(hi, lo):
    """Move/scale hi onto lo (same model, possibly different scale/offset),
    trying the four turns about the vertical; keeps the closest fit."""
    PL = world_verts(lo)
    bl = bbox(PL)
    hl = bl[5] - bl[4]
    cl = Vector(((bl[0] + bl[1]) / 2, (bl[2] + bl[3]) / 2, bl[4]))
    me = lo.data
    tree = BVHTree.FromPolygons(PL, [tuple(p.vertices) for p in me.polygons])
    base = hi.matrix_world.copy()
    best = None
    for turn in range(4):
        hi.matrix_world = Matrix.Rotation(turn * math.pi / 2, 4, "Z") @ base
        bpy.context.view_layer.update()
        P = world_verts(hi)
        b = bbox(P)
        s = hl / (b[5] - b[4])
        c = Vector(((b[0] + b[1]) / 2, (b[2] + b[3]) / 2, b[4]))
        M = Matrix.Translation(cl) @ Matrix.Scale(s, 4) @ Matrix.Translation(-c)
        step = max(1, len(P) // 3000)
        err = 0.0; n = 0
        for p in P[::step]:
            q = M @ p
            hit = tree.find_nearest(q)
            if hit[0] is not None:
                err += hit[3]; n += 1
        err /= max(1, n)
        say("  align turn", turn * 90, "mean gap", round(err / hl, 4))
        if best is None or err < best[0]:
            best = (err, M @ Matrix.Rotation(turn * math.pi / 2, 4, "Z") @ base)
    hi.matrix_world = best[1]
    select_only([hi], hi)
    bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
    return best[0] / hl


def head_islands_up(me, z0, h):
    """Enlarge the UV islands of the head so the face gets more texture."""
    bm = bmesh.new(); bm.from_mesh(me)
    uv = bm.loops.layers.uv.active
    bm.faces.ensure_lookup_table()
    parent = list(range(len(bm.faces)))

    def find(a):
        while parent[a] != a:
            parent[a] = parent[parent[a]]; a = parent[a]
        return a

    def uvat(f, v):
        for l in f.loops:
            if l.vert == v:
                return l[uv].uv
        return None
    for e in bm.edges:
        if len(e.link_faces) != 2:
            continue
        f1, f2 = e.link_faces
        a, b = e.verts
        ua1, ua2, ub1, ub2 = uvat(f1, a), uvat(f2, a), uvat(f1, b), uvat(f2, b)
        if (ua1 - ua2).length < 1e-6 and (ub1 - ub2).length < 1e-6:
            ra, rb = find(f1.index), find(f2.index)
            if ra != rb:
                parent[ra] = rb
    isl = {}
    for f in bm.faces:
        isl.setdefault(find(f.index), []).append(f)
    big = 0
    for faces in isl.values():
        zc = sum(f.calc_center_median().z for f in faces) / len(faces)
        if zc < z0 + HEAD_Z * h:
            continue
        loops = [l for f in faces for l in f.loops]
        c = sum((l[uv].uv for l in loops), Vector((0, 0))) / len(loops)
        for l in loops:
            l[uv].uv = c + (l[uv].uv - c) * HEAD_K
        big += 1
    bm.to_mesh(me); bm.free()
    return big, len(isl)


def emission_route(o):
    for slot in o.material_slots:
        m = slot.material
        if not m or not m.node_tree:
            continue
        nt = m.node_tree
        out = next((n for n in nt.nodes if n.type == "OUTPUT_MATERIAL"), None)
        bsdf = next((n for n in nt.nodes if n.type == "BSDF_PRINCIPLED"), None)
        if out is None or bsdf is None:
            continue
        em = nt.nodes.new("ShaderNodeEmission")
        bc = bsdf.inputs["Base Color"]
        if bc.links:
            nt.links.new(bc.links[0].from_socket, em.inputs["Color"])
        else:
            em.inputs["Color"].default_value = bc.default_value
        nt.links.new(em.outputs["Emission"], out.inputs["Surface"])


def write_orbs(hid, me_obj, arm):
    me = me_obj.data
    mw = me_obj.matrix_world
    P = [mw @ v.co for v in me.vertices]
    xs = [p.x for p in P]; ys = [p.y for p in P]; zs = [p.z for p in P]
    cx, cy, z0 = (min(xs) + max(xs)) / 2, (min(ys) + max(ys)) / 2, min(zs)
    h = max(zs) - z0
    Rm = Matrix(((1, 0, 0, 0), (0, 0, 1, 0), (0, 1, 0, 0), (0, 0, 0, 1)))
    G = Matrix.Diagonal((1 / h, 1 / h, 1 / h, 1)) @ Matrix.Translation(Vector((-cx, -z0, -cy))) @ Rm
    Gi = G.inverted()
    bones = [b.name for b in arm.data.bones]
    bix = {n: i for i, n in enumerate(bones)}
    gname = {g.index: g.name for g in me_obj.vertex_groups}
    VW = []
    unweighted = 0
    for v in me.vertices:
        ws = sorted(((g.weight, bix[gname[g.group]]) for g in v.groups if gname.get(g.group) in bix and g.weight > 0.001), reverse=True)[:4]
        if not ws:
            unweighted += 1
            hips = bix.get(next((n for n in bones if n.endswith("Hips")), bones[0]), 0)
            ws = [(1.0, hips)]
        tot = sum(w for w, _ in ws) or 1.0
        ws = [(w / tot, j) for w, j in ws] + [(0.0, 0)] * (4 - len(ws))
        VW.append(ws)
    me.calc_loop_triangles()
    uvl = me.uv_layers.active
    cn = getattr(me, "corner_normals", None)
    key = {}; pos = array.array("f"); nrm = array.array("b"); uvs = array.array("f")
    jt = array.array("B"); wt = array.array("B"); idx = array.array("I")
    m3 = mw.to_3x3()
    for tri in me.loop_triangles:
        ids = []
        for li in tri.loops:
            vi = me.loops[li].vertex_index
            u = uvl.data[li].uv
            k = (vi, round(u.x, 5), round(u.y, 5))
            if k not in key:
                key[k] = len(key)
                p = G @ P[vi]
                pos.extend((p.x, p.y, p.z))
                n = cn[li].vector if cn is not None else me.loops[li].normal
                n = m3 @ n; n = Vector((n.x, n.z, n.y)).normalized()
                nrm.extend((int(max(-127, min(127, n.x * 127))), int(max(-127, min(127, n.y * 127))), int(max(-127, min(127, n.z * 127)))))
                uvs.extend((u.x, 1.0 - u.y))
                for w, j in VW[vi]:
                    jt.append(j); wt.append(int(round(w * 255)))
            ids.append(key[k])
        idx.extend((ids[0], ids[2], ids[1]))
    skin = array.array("f")
    amw, amwi = arm.matrix_world, arm.matrix_world.inverted()
    for k in range(FRAMES):
        idle_pose(arm, k)
        bpy.context.view_layer.update()
        for n in bones:
            pb = arm.pose.bones[n]
            D = amw @ pb.matrix @ arm.data.bones[n].matrix_local.inverted() @ amwi
            M = G @ D @ Gi
            for r in range(3):
                skin.extend((M[r][0], M[r][1], M[r][2], M[r][3]))
    nv, nt = len(key), len(idx) // 3
    body = struct.pack("<IIIIIf", 1, nv, nt, len(bones), FRAMES, FPS) + pos.tobytes() + nrm.tobytes() + uvs.tobytes() + jt.tobytes() + wt.tobytes() + idx.tobytes() + skin.tobytes()
    open(os.path.join(OUT, hid + ".orbs"), "wb").write(b"ORBS" + zlib.compress(body, 9))
    say("  wrote", hid + ".orbs", "verts", nv, "tris", nt, "bones", len(bones), "unweighted", unweighted)


def export(hid):
    rig = os.path.join(RIG, "arcane_rigged.glb" if hid == "arcane" else hid + ".glb")
    src = os.path.join(GLB, "hero_%s.glb" % hid)
    # a newer source model dropped into heroes/glb wins (the user's own pick)
    for alt in NEWER.get(hid, []):
        if os.path.exists(os.path.join(GLB, alt)):
            src = os.path.join(GLB, alt)
            say(hid, "using newer model", alt)
            break
    if not (os.path.exists(rig) and os.path.exists(src)):
        say("missing", rig, src); return
    clear()
    sc = bpy.context.scene
    new = import_glb(rig)
    arm = next(o for o in new if o.type == "ARMATURE")
    lo = max((o for o in new if o.type == "MESH"), key=lambda o: len(o.data.vertices))
    if hid == "arcane":
        fix_arcane(lo, arm)
    # the original, joined and aligned onto the rig's mesh
    hiobjs = [o for o in import_glb(src) if o.type == "MESH"]
    select_only(hiobjs, hiobjs[0])
    if len(hiobjs) > 1:
        bpy.ops.object.join()
    hi = bpy.context.view_layer.objects.active
    keep = hi.matrix_world.copy()
    hi.parent = None
    hi.matrix_world = keep
    bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
    gap = align(hi, lo)
    say(hid, "original", len(hi.data.polygons), "faces; fit gap", round(gap, 4))
    emission_route(hi)
    # the game mesh
    mid = hi.copy(); mid.data = hi.data.copy(); mid.name = "hq"
    sc.collection.objects.link(mid)
    select_only([mid], mid)
    P = world_verts(mid); b = bbox(P); z0, h = b[4], b[5] - b[4]
    bpy.ops.object.mode_set(mode="EDIT")
    bpy.ops.mesh.select_all(action="SELECT")
    bpy.ops.mesh.remove_doubles(threshold=0.0005 * h)
    bpy.ops.object.mode_set(mode="OBJECT")
    nf = len(mid.data.polygons)
    if nf > TARGET:
        mod = mid.modifiers.new("dec", "DECIMATE")
        mod.decimate_type = "COLLAPSE"; mod.ratio = TARGET / nf
        bpy.ops.object.modifier_apply(modifier=mod.name)
    tri = mid.modifiers.new("tri", "TRIANGULATE")
    bpy.ops.object.modifier_apply(modifier=tri.name)
    say("  decimated to", len(mid.data.polygons), "tris; specks", remove_specks(mid.data))
    while mid.data.uv_layers:
        mid.data.uv_layers.remove(mid.data.uv_layers[0])
    mid.data.uv_layers.new(name="bake")
    for g in list(mid.vertex_groups):
        mid.vertex_groups.remove(g)
    bpy.ops.object.mode_set(mode="EDIT")
    bpy.ops.mesh.select_all(action="SELECT")
    bpy.ops.uv.smart_project(angle_limit=1.15, island_margin=0.003)
    bpy.ops.object.mode_set(mode="OBJECT")
    big, total = head_islands_up(mid.data, z0, h)
    bpy.ops.object.mode_set(mode="EDIT")
    bpy.ops.mesh.select_all(action="SELECT")
    bpy.ops.uv.pack_islands(rotate=True, margin=0.003)
    bpy.ops.object.mode_set(mode="OBJECT")
    say("  uv islands", total, "head islands enlarged", big)
    bpy.ops.object.shade_smooth()
    # bake the original's colour onto it
    img = bpy.data.images.new("bake_" + hid, TEX, TEX)
    mat = bpy.data.materials.new("hqmat"); mat.use_nodes = True
    tn = mat.node_tree.nodes.new("ShaderNodeTexImage"); tn.image = img
    mat.node_tree.nodes.active = tn
    mid.data.materials.clear(); mid.data.materials.append(mat)
    sc.render.engine = "CYCLES"
    sc.cycles.device = "CPU"
    sc.cycles.samples = 8
    sc.render.bake.use_selected_to_active = True
    sc.render.bake.cage_extrusion = 0.008 * h
    sc.render.bake.max_ray_distance = 0.04 * h
    sc.render.bake.margin = 16
    lo.hide_render = True
    select_only([hi, mid], mid)
    t0 = time.time()
    bpy.ops.object.bake(type="EMIT", use_clear=True, margin=16)
    say("  baked", TEX, "px in", int(time.time() - t0), "s")
    sc.render.image_settings.file_format = "JPEG"
    sc.render.image_settings.quality = 92
    img.save_render(os.path.join(OUT, hid + ".jpg"))
    # weights from the rigged model
    lo.hide_render = False
    dt = mid.modifiers.new("dt", "DATA_TRANSFER")
    dt.object = lo
    dt.use_vert_data = True
    dt.data_types_verts = {"VGROUP_WEIGHTS"}
    dt.vert_mapping = "POLYINTERP_NEAREST"
    dt.layers_vgroup_select_src = "ALL"
    dt.layers_vgroup_select_dst = "NAME"
    select_only([mid], mid)
    bpy.ops.object.datalayout_transfer(modifier=dt.name)
    bpy.ops.object.modifier_apply(modifier=dt.name)
    say("  weights: groups", len(mid.vertex_groups))
    write_orbs(hid, mid, arm)


for hid in IDS:
    try:
        export(hid)
    except Exception as e:
        import traceback
        say("FAILED", hid, e, traceback.format_exc())
open(os.path.join(HERE, "hq-done.txt"), "w").write("ok")
