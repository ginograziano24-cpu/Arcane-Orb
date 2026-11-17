# Blender 5.x:  blender -b -P arcane_skinmap.py
# Shows what counts as her skin and what counts as clothing, on her ORIGINAL
# model (heroes/arcane_layers2/arcanist_layers.blend, layers Body/Robe/Armor/Hat).
#   skinmap_nude_<view>.png  her original model, textured, with every piece of
#                            clothing below her belly taken away
#   skinmap_map_<view>.png   her Body layer: skin in peach, clothing in red
# views: front, back, side, under (from the floor, looking up)
# Writes skinmap_log.txt and skinmap_faces.npz (the verdict per Body face).
import bpy, bmesh, os, math, json
import numpy as np
from mathutils import Vector

HERE = os.path.dirname(os.path.abspath(__file__))
L2 = os.path.join(HERE, "arcane_layers2")
LAYERS = os.path.join(L2, "arcanist_layers.blend")
LOG = open(os.path.join(L2, "skinmap_log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s); LOG.write(s + "\n"); LOG.flush()


def wverts(o):
    me = o.data
    co = np.empty(len(me.vertices) * 3); me.vertices.foreach_get("co", co); co = co.reshape(-1, 3)
    M = np.array(o.matrix_world)
    return co @ M[:3, :3].T + M[:3, 3]


def polys(o):
    return [tuple(p.vertices) for p in o.data.polygons]


def face_samples(o):
    """Texture colour of each face at its centre and near its three corners (n,4,3)."""
    me = o.data; img = None
    for m in me.materials:
        if m and m.node_tree:
            bs = next((n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED"), None)
            if bs and bs.inputs["Base Color"].links:
                nd = bs.inputs["Base Color"].links[0].from_node
                if nd.type == "TEX_IMAGE" and nd.image:
                    img = nd.image; break
    if img is None:
        return None
    W_, H_ = img.size
    px = np.empty(W_ * H_ * 4, dtype=np.float32); img.pixels.foreach_get(px); px = px.reshape(H_, W_, 4)[:, :, :3]
    uv = np.empty(len(me.loops) * 2, dtype=np.float32); me.uv_layers.active.data.foreach_get("uv", uv); uv = uv.reshape(-1, 2)
    ls = np.empty(len(me.polygons), dtype=np.int32); me.polygons.foreach_get("loop_start", ls)
    lt = np.empty(len(me.polygons), dtype=np.int32); me.polygons.foreach_get("loop_total", lt)
    c = [uv[ls + np.minimum(q, lt - 1)] for q in range(3)]
    ctr = (c[0] + c[1] + c[2]) / 3
    pts = [ctr] + [0.6 * ci + 0.4 * ctr for ci in c]
    out = []
    for p in pts:
        xi = np.clip((p[:, 0] % 1.0) * (W_ - 1), 0, W_ - 1).astype(np.int32)
        yi = np.clip((p[:, 1] % 1.0) * (H_ - 1), 0, H_ - 1).astype(np.int32)
        out.append(px[yi, xi])
    return np.stack(out, 1).astype(np.float64)


def main():
    bpy.ops.wm.open_mainfile(filepath=LAYERS)
    sc = bpy.context.scene
    lay = {n: bpy.data.objects.get("Arcanist_" + n) for n in ("Body", "Robe", "Armor", "Hat")}
    allP = np.vstack([wverts(o) for o in lay.values() if o])
    z0, z1 = allP[:, 2].min(), allP[:, 2].max(); HH = z1 - z0
    cam_ = json.load(open(os.path.join(L2, "eyes_cam.json")))
    ppx = cam_["ortho_scale"] / cam_["res"][0]
    fz = lambda py: cam_["loc"][2] + (cam_["res"][1] / 2 - py) * ppx
    fx = cam_["loc"][0] + (1767 - cam_["res"][0] / 2) * ppx
    z_eye, z_chin = fz(694), fz(1236)
    Hhead = (z_eye - z_chin) / 0.45
    ST = (z_chin + Hhead) - z0
    z_belly = z0 + 0.70 * ST                                    # her navel line
    say("stature %.4f, belly line at %.4f (0.70 of stature)" % (ST, z_belly))

    body = lay["Body"]
    BW = wverts(body); BP = polys(body)
    FC = np.array([BW[list(f)].mean(0) for f in BP])
    smp = face_samples(body)
    # skin colour learned from her own cheek
    cx_, cz_ = fx + 0.045 * ST, z_eye - 0.03 * ST
    box = (np.abs(FC[:, 0] - cx_) < 0.014 * ST) & (np.abs(FC[:, 2] - cz_) < 0.012 * ST)
    if box.sum() > 20:
        box &= FC[:, 1] < np.percentile(FC[box, 1], 5) + 0.012 * ST

    def chroma(c):
        s_ = np.maximum(c.sum(-1), 1e-6)
        return np.stack([c[..., 0] / s_, c[..., 1] / s_], -1)
    ch = chroma(smp)                                             # (n,4,2)
    cheek = ch[box].reshape(-1, 2)
    mu = np.median(cheek, 0); C = np.cov((cheek - mu).T) + np.eye(2) * 1e-5; Ci = np.linalg.inv(C)
    d = ch - mu
    dm = np.sqrt(np.einsum("nsi,ij,nsj->ns", d, Ci, d))
    val = smp.max(-1)
    # a sample is skin if its colour is her skin's (bright highlights and deep
    # shade judged by colour, not brightness); a face by the majority of its 4
    vote = ((dm < 3.5) & (val > 0.12)).mean(1)
    say("skin model from %d cheek samples, chroma %s" % (len(cheek), np.round(mu, 3)))
    # smooth the verdict over neighbouring faces (speckles of highlight or
    # shadow don't flip a face; real seams between skin and cloth stay sharp)
    nv = len(BW)
    _, vid = np.unique(np.round(BW / (1e-5 * HH)).astype(np.int64), axis=0, return_inverse=True)
    vid = vid.ravel()
    Fv = [np.array([vid[v] for v in f]) for f in BP]
    score = vote.copy()
    for _ in range(3):
        vs = np.zeros(vid.max() + 1); vc = np.zeros(vid.max() + 1)
        for k, f in enumerate(Fv):
            vs[f] += score[k]; vc[f] += 1
        vavg = vs / np.maximum(vc, 1)
        score = 0.5 * vote + 0.5 * np.array([vavg[f].mean() for f in Fv])
    skin = score > 0.5
    say("Body layer: %d faces, %d skin, %d clothing" % (len(BP), int(skin.sum()), int((~skin).sum())))
    for zz in np.arange(0.1, 1.0, 0.1):
        band = (FC[:, 2] > z0 + (zz - 0.05) * ST) & (FC[:, 2] < z0 + (zz + 0.05) * ST)
        if band.any():
            say("  height %.1f of stature: %d faces, %.0f%% skin" % (zz, int(band.sum()), 100 * skin[band].mean()))
    np.savez(os.path.join(L2, "skinmap_faces.npz"), skin=skin, score=score)
    # her body is the large continuous piece(s) of the Body layer; hair, the
    # staff, hat bits and trims are separate small pieces
    par = np.arange(vid.max() + 1)
    def fr(a_):
        r_ = a_
        while par[r_] != r_:
            r_ = par[r_]
        while par[a_] != r_:
            par[a_], a_ = r_, par[a_]
        return r_
    for f in Fv:
        r0 = fr(f[0])
        for v in f[1:]:
            r1 = fr(v)
            if r1 != r0:
                par[r1] = r0
    comp = np.array([fr(f[0]) for f in Fv])
    from collections import Counter
    csz = Counter(comp.tolist())
    cxs = {}
    for c_, x_ in zip(comp.tolist(), FC[:, 0]):
        cxs.setdefault(c_, []).append(x_)
    bodyc = {c_ for c_, n_ in csz.items() if n_ >= 2000 and abs(float(np.mean(cxs[c_])) - fx) < 0.2 * ST}
    bodyF = np.array([c_ in bodyc for c_ in comp.tolist()])
    say("body pieces: %s faces; below the belly %d body faces kept whatever their paint" % (sorted([csz[c_] for c_ in bodyc], reverse=True), int((bodyF & (FC[:, 2] <= z_belly)).sum())))

    col = bpy.data.collections.new("SkinMap"); sc.collection.children.link(col)

    def copy_keep(o, keep, name, mats=None, mat_index=None):
        me_ = o.data.copy(); ob_ = bpy.data.objects.new(name, me_); ob_.matrix_world = o.matrix_world.copy()
        if mats is not None:
            me_.materials.clear()
            for m in mats:
                me_.materials.append(m)
            mi = np.asarray(mat_index, dtype=np.int32)
            me_.polygons.foreach_set("material_index", mi)
        bm_ = bmesh.new(); bm_.from_mesh(me_); bm_.faces.ensure_lookup_table()
        bmesh.ops.delete(bm_, geom=[f for f in bm_.faces if not keep[f.index]], context="FACES")
        bm_.to_mesh(me_); bm_.free(); col.objects.link(ob_)
        return ob_

    # set A: her original, textured, no clothing below the belly
    nude = []
    nude.append(copy_keep(body, bodyF | (FC[:, 2] > z_belly), "SM_Body"))
    for n in ("Robe", "Armor", "Hat"):
        o = lay.get(n)
        if o is None:
            continue
        V_ = wverts(o); P_ = polys(o)
        C_ = np.array([V_[list(f)].mean(0) for f in P_])
        nude.append(copy_keep(o, C_[:, 2] > z_belly, "SM_" + n))
    # set B: the map
    def flat(name, rgb):
        m = bpy.data.materials.new(name); m.use_nodes = True
        b = next(n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
        b.inputs["Base Color"].default_value = (*rgb, 1); b.inputs["Roughness"].default_value = 0.5
        return m
    m_skin, m_cloth = flat("SM_skin", (0.86, 0.6, 0.48)), flat("SM_cloth", (0.85, 0.05, 0.05))
    mapo = copy_keep(body, np.ones(len(BP), bool), "SM_Map", [m_skin, m_cloth], np.where(skin, 0, 1))

    for c in sc.collection.children:
        c.hide_render = c.name != "SkinMap"
    for o in col.objects:
        o.hide_render = False
    for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
        try:
            sc.render.engine = eng; break
        except Exception:
            pass
    cd = bpy.data.cameras.new("smcam"); cam = bpy.data.objects.new("smcam", cd); sc.collection.objects.link(cam); sc.camera = cam
    sun = bpy.data.lights.new("smsun", "SUN"); sun.energy = 3.0
    so = bpy.data.objects.new("smsun", sun); sc.collection.objects.link(so)
    fill = bpy.data.lights.new("smfill", "SUN"); fill.energy = 1.2
    fo = bpy.data.objects.new("smfill", fill); sc.collection.objects.link(fo)
    sc.render.resolution_x, sc.render.resolution_y = 800, 1100
    ctr = Vector((0.5 * (allP[:, 0].min() + allP[:, 0].max()), 0.5 * (allP[:, 1].min() + allP[:, 1].max()), z0 + 0.5 * HH))
    views = (("front", -90), ("back", 90), ("side", 0))
    for tag, objs in (("nude2", nude),):
        for o in col.objects:
            o.hide_render = o not in objs
        for vname, ang in views:
            a = math.radians(ang)
            cd.type = "ORTHO"; cd.ortho_scale = HH * 1.08
            cam.location = ctr + Vector((math.cos(a), math.sin(a), 0)) * HH * 3
            cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
            so.rotation_euler = (math.radians(55), 0, a + math.radians(60))
            fo.rotation_euler = (math.radians(70), 0, a - math.radians(120))
            sc.render.filepath = os.path.join(L2, "skinmap_%s_%s.png" % (tag, vname))
            bpy.ops.render.render(write_still=True)
        # under her, between the knees, looking up
        cd.type = "PERSP"; cd.angle = math.radians(80)
        cam.location = Vector((ctr.x, ctr.y + 0.02 * HH, z0 + 0.12 * HH))
        cam.rotation_euler = (math.pi, 0.0, 0.0)
        so.rotation_euler = (math.radians(180), 0, 0)
        fo.rotation_euler = (math.radians(150), 0, math.radians(90))
        sc.render.filepath = os.path.join(L2, "skinmap_%s_under.png" % tag)
        bpy.ops.render.render(write_still=True)
    say("renders: skinmap_nude2_*.png (her original, no clothing below the belly, her whole body piece kept)")
    open(os.path.join(L2, "skinmap_done.txt"), "w").write("ok")


try:
    main()
except Exception:
    import traceback
    say("FAILED", traceback.format_exc())
    open(os.path.join(L2, "skinmap_done.txt"), "w").write("failed")
