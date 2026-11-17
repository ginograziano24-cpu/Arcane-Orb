# Blender 5.x: fits a clean mannequin body under the Arcanist's clothes.
#   blender -b -P arcane_basebody.py
# The earlier "Base Form" was a blob made from her own surface, so it looked
# like her armour. This one starts from Blender's CC0 stylized female base
# mesh (heroes/basemesh/human_base_meshes_bundle.blend):
#   1. the base body is cut into segments with the bundle's own mannequin parts
#   2. each segment is moved, turned and stretched onto her skeleton
#      (heroes/rigged/arcane_rigged.glb), blended smoothly at the joints
#   3. the surface is pulled onto her visible skin, and onto clothing that
#      hugs the body (corset, boots) minus a cloth thickness; hidden parts are
#      filled in smoothly from what is known around them
# Result: heroes/arcane_layers2/arcanist_layers.blend gets a new "Base Form"
# (smooth clay mannequin, no anatomical detail) next to Body / Robe / Armor / Hat,
# plus check renders base_*.png.
import bpy, os, math, shutil, time
import numpy as np
from mathutils import Vector, Matrix
from mathutils.bvhtree import BVHTree

HERE = os.path.dirname(os.path.abspath(__file__))
L2 = os.path.join(HERE, "arcane_layers2")
BUNDLE = os.path.join(HERE, "basemesh", "human_base_meshes_bundle.blend")
RIGGED = os.path.join(HERE, "rigged", "arcane_rigged.glb")
LAYERS = os.path.join(L2, "arcanist_layers.blend")
LOG = open(os.path.join(L2, "basebody_log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s); LOG.write(s + "\n"); LOG.flush()


def wverts(o):
    me = o.data
    co = np.empty(len(me.vertices) * 3, dtype=np.float64)
    me.vertices.foreach_get("co", co)
    M = np.array(o.matrix_world)
    return co.reshape(-1, 3) @ M[:3, :3].T + M[:3, 3]


def polys(o):
    return [tuple(p.vertices) for p in o.data.polygons]


def ends(P):
    c = P.mean(0)
    _u, _s, vt = np.linalg.svd(P - c, full_matrices=False)
    t = (P - c) @ vt[0]
    return P[t <= np.percentile(t, 12)].mean(0), P[t >= np.percentile(t, 88)].mean(0)


try:
    # ------------------------------------------------------------ A. base body
    bpy.ops.wm.open_mainfile(filepath=BUNDLE)
    body = bpy.data.objects["GEO-body_female_stylized"]
    BV = wverts(body)
    BP = polys(body)
    bcx = 0.5 * (BV[:, 0].min() + BV[:, 0].max())
    parts = [o for o in bpy.data.objects if o.type == "MESH" and "female_primitive_stylized" in o.name]
    pel = next(o for o in parts if "pelvis" in o.name)
    pv = wverts(pel)
    dx = bcx - 0.5 * (pv[:, 0].min() + pv[:, 0].max())
    say("base body verts", len(BV), "faces", len(BP), "parts", len(parts), "offset", round(dx, 3))

    def seg_of(name):
        side = ".L" if name.endswith(".L") else ".R" if name.endswith(".R") else ""
        n = name.lower()
        for key, seg, sided in (("arm_lower", "armL", 1), ("arm_upper", "armU", 1), ("leg_lower", "legL", 1), ("leg_upper", "legU", 1),
                                ("finger", "hand", 1), ("thumb", "hand", 1), ("hand", "hand", 1), ("toe", "foot", 1), ("foot", "foot", 1),
                                ("belly", "spine", 0), ("breast", "chest", 0), ("chest", "chest", 0), ("shoulder", "chest", 0),
                                ("pelvis", "hips", 0), ("neck", "neck", 0),
                                ("ear", "head", 0), ("eyelid", "head", 0), ("eye", "head", 0), ("nose", "head", 0), ("head", "head", 0)):
            if key in n:
                return seg + (side if sided else "")
        return None
    SEG = ["hips", "spine", "chest", "neck", "head"] + [s + sd for sd in (".L", ".R") for s in ("armU", "armL", "hand", "legU", "legL", "foot")]
    SI = {s: i for i, s in enumerate(SEG)}
    PV = {}                                   # segment -> world verts of its parts (in the body's frame)
    best = np.full(len(BV), 1e9); lab = np.zeros(len(BV), dtype=np.int32)
    for o in parts:
        sg = seg_of(o.name)
        if sg is None:
            continue
        P = wverts(o); P[:, 0] += dx
        key = o.name.lower()
        PV.setdefault(sg, []).append(P)
        PV.setdefault("raw:" + ("pelvis" if "pelvis" in key else "belly" if "belly" in key else "chestpart" if "chest" in key else "neckpart" if "neck" in key else "headpart" if key.startswith("geo-head") else key), []).append(P)
        tree = BVHTree.FromPolygons([tuple(p) for p in P], polys(o))
        for i, p in enumerate(BV):
            d = tree.find_nearest(Vector(p))[3]
            if d is not None and d < best[i]:
                best[i] = d; lab[i] = SI[sg]
    PV = {k: np.vstack(v) for k, v in PV.items()}
    say("labels", {s: int((lab == i).sum()) for i, s in enumerate(SEG)})
    # joints of the base body
    BJ = {}                                   # segment -> (head, tail)
    cy = lambda P: P[:, 1].mean()
    pe, be, ch, ne, he = PV["raw:pelvis"], PV["raw:belly"], PV["raw:chestpart"], PV["raw:neckpart"], PV["raw:headpart"]
    zA = pe[:, 2].min() + 0.45 * np.ptp(pe[:, 2]); zB = be[:, 2].mean(); zC = ch[:, 2].mean()
    zD = ne[:, 2].min() + 0.2 * np.ptp(ne[:, 2]); zE = he[:, 2].min() + 0.22 * np.ptp(he[:, 2]); zF = he[:, 2].max()
    pt = lambda P, z: np.array([bcx, cy(P), z])
    BJ["neck"] = (pt(ne, zD), pt(he, zE)); BJ["head"] = (pt(he, zE), pt(he, zF))
    for sd in (".L", ".R"):
        a, b = ends(PV["legU" + sd]); hip, k1 = (a, b) if a[2] > b[2] else (b, a)
        a, b = ends(PV["legL" + sd]); k2, an = (a, b) if a[2] > b[2] else (b, a)
        knee = 0.5 * (k1 + k2)
        ft = PV["foot" + sd]
        toe = ft[ft[:, 1] <= np.percentile(ft[:, 1], 6)].mean(0)
        BJ["legU" + sd] = (hip, knee); BJ["legL" + sd] = (knee, an); BJ["foot" + sd] = (an, toe)
        near = lambda a, b: (a, b) if abs(a[0] - bcx) < abs(b[0] - bcx) else (b, a)
        sh, e1 = near(*ends(PV["armU" + sd])); e2, w1 = near(*ends(PV["armL" + sd])); w2, tip = near(*ends(PV["hand" + sd]))
        el = 0.5 * (e1 + e2); wr = 0.5 * (w1 + w2)
        BJ["armU" + sd] = (sh, el); BJ["armL" + sd] = (el, wr); BJ["hand" + sd] = (wr, tip)
    def torso(Jd, hipL, hipR, shL, shR, neck):
        A = 0.5 * (hipL + hipR); S = 0.5 * (shL + shR)
        a1 = A + 0.33 * (S - A); a2 = A + 0.68 * (S - A)
        Jd["hips"] = (A, a1); Jd["spine"] = (a1, a2); Jd["chest"] = (a2, neck)
    torso(BJ, BJ["legU.L"][0], BJ["legU.R"][0], BJ["armU.L"][0], BJ["armU.R"][0], BJ["neck"][0])
    for s in SEG:
        say("base joint", s, np.round(BJ[s][0], 3), np.round(BJ[s][1], 3))
    say("base .L side x", round(BJ["legU.L"][0][0] - bcx, 3))
    E = np.array([tuple(e.vertices) for e in body.data.edges], dtype=np.int64)
    # smooth segment weights
    Wt = np.zeros((len(BV), len(SEG))); Wt[np.arange(len(BV)), lab] = 1.0
    deg = np.zeros(len(BV)); np.add.at(deg, E[:, 0], 1); np.add.at(deg, E[:, 1], 1)

    def navg(X):
        out = np.zeros_like(X)
        np.add.at(out, E[:, 0], X[E[:, 1]]); np.add.at(out, E[:, 1], X[E[:, 0]])
        return out / np.maximum(deg, 1)[(...,) + (None,) * (X.ndim - 1)]
    for _ in range(8):
        Wt = 0.5 * Wt + 0.5 * navg(Wt)
    Wt /= Wt.sum(1, keepdims=True)

    # ------------------------------------------------------------ B. her model
    bk = os.path.join(L2, "arcanist_layers_before_base.blend")
    if not os.path.exists(bk):
        shutil.copyfile(LAYERS, bk)
    bpy.ops.wm.open_mainfile(filepath=LAYERS)
    sc = bpy.context.scene
    lay = {n: bpy.data.objects.get("Arcanist_" + n) for n in ("Body", "Robe", "Armor", "Hat")}
    say("layers", {k: (len(v.data.polygons) if v else None) for k, v in lay.items()})
    allP = np.vstack([wverts(o) for o in lay.values() if o])
    z0, z1 = allP[:, 2].min(), allP[:, 2].max(); HH = z1 - z0
    say("her height", round(HH, 4), "z0", round(z0, 4))
    # ---- her surfaces (needed for the landmarks below)
    def tree_of(objs):
        V = []; F = []; off = 0
        for o in objs:
            P = wverts(o); V.append(P)
            F += [tuple(i + off for i in p) for p in polys(o)]
            off += len(P)
        V = np.vstack(V)
        return BVHTree.FromPolygons([tuple(p) for p in V], F)
    bodyo = lay["Body"]
    btree = tree_of([bodyo])
    ctree = tree_of([o for k, o in lay.items() if o and k in ("Robe", "Armor")])
    # ---- which of her Body faces are skin: colour learned from her own cheek
    import json
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    ppx = cam["ortho_scale"] / cam["res"][0]
    fz = lambda py: cam["loc"][2] + (cam["res"][1] / 2 - py) * ppx
    fx = cam["loc"][0] + (1767 - cam["res"][0] / 2) * ppx
    z_eye, z_chin = fz(694), fz(1236)
    Hhead = (z_eye - z_chin) / 0.45
    ST = (z_chin + Hhead) - z0                           # her stature, crown to soles (the hat is not counted)
    say("stature", round(ST, 4), "of total height", round(HH, 4))
    BW = wverts(bodyo)
    Bpol = polys(bodyo)
    tri3 = np.array([p[:3] for p in Bpol], dtype=np.int64)
    FC = BW[tri3].mean(1)                               # face centres
    me = bodyo.data
    col = None
    try:
        img = None
        for m in me.materials:
            if m and m.node_tree:
                bs = next((n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED"), None)
                if bs and bs.inputs["Base Color"].links:
                    nd = bs.inputs["Base Color"].links[0].from_node
                    if nd.type == "TEX_IMAGE" and nd.image:
                        img = nd.image; break
        W_, H_ = img.size
        px = np.empty(W_ * H_ * 4, dtype=np.float32); img.pixels.foreach_get(px)
        px = px.reshape(H_, W_, 4)[:, :, :3]
        uv = np.empty(len(me.loops) * 2, dtype=np.float32); me.uv_layers.active.data.foreach_get("uv", uv); uv = uv.reshape(-1, 2)
        ls = np.empty(len(me.polygons), dtype=np.int32); me.polygons.foreach_get("loop_start", ls)
        lt = np.empty(len(me.polygons), dtype=np.int32); me.polygons.foreach_get("loop_total", lt)
        uc = np.zeros((len(ls), 2))
        for q in range(3):                              # centre of each face in the texture
            uc += uv[ls + np.minimum(q, lt - 1)]
        uc /= 3.0
        xi = np.clip((uc[:, 0] % 1.0) * (W_ - 1), 0, W_ - 1).astype(np.int32)
        yi = np.clip((uc[:, 1] % 1.0) * (H_ - 1), 0, H_ - 1).astype(np.int32)
        col = px[yi, xi].astype(np.float64)
        del px
    except Exception as e:
        say("skin colour read failed", e)
    skinf = np.zeros(len(Bpol), dtype=bool)
    if col is not None:
        ssum = np.maximum(col.sum(1), 1e-6)
        feat = np.stack([col[:, 0] / ssum, col[:, 1] / ssum], 1)
        val = col.max(1)
        cx_ = fx + 0.045 * ST
        cz_ = z_eye - 0.03 * ST
        box = (np.abs(FC[:, 0] - cx_) < 0.014 * ST) & (np.abs(FC[:, 2] - cz_) < 0.012 * ST)
        if box.sum() > 20:
            ymin = np.percentile(FC[box, 1], 5)
            box &= FC[:, 1] < ymin + 0.012 * ST
        cheek = feat[box]
        if len(cheek) < 20:
            say("cheek sample too small", len(cheek)); cheek = feat[(val > 0.4)]
        mu = np.median(cheek, 0); C = np.cov((cheek - mu).T) + np.eye(2) * 1e-5
        Ci = np.linalg.inv(C)
        dm = np.sqrt(np.einsum("ij,jk,ik->i", feat - mu, Ci, feat - mu))
        skinf = (dm < 3.2) & (val > 0.22)
        say("skin model from", len(cheek), "cheek faces: chroma", np.round(mu, 3), "spread", np.round(np.sqrt(np.diag(C)), 4), "-> skin faces", int(skinf.sum()), "of", len(skinf))
    # ---- weld, erode once, connected pieces of skin
    _, vid = np.unique(np.round(BW / (1e-4 * HH)).astype(np.int64), axis=0, return_inverse=True)
    vid = vid.ravel()
    T_all = vid[tri3]
    nonskin_v = np.zeros(vid.max() + 1, dtype=bool); nonskin_v[T_all[~skinf].ravel()] = True
    skin_e = skinf & ~nonskin_v[T_all].any(1)
    T = T_all[skin_e]
    labv = np.arange(vid.max() + 1)
    for _it in range(400):
        m_ = labv[T].min(1); old = labv.copy()
        np.minimum.at(labv, T.ravel(), np.repeat(m_, 3))
        labv = labv[labv]
        if (labv == old).all():
            break
    compf = np.full(len(Bpol), -1); compf[skin_e] = labv[T[:, 0]]
    ids, cnt = np.unique(compf[skin_e], return_counts=True)
    order = np.argsort(-cnt)
    keepc = set(); comps = []
    for k in order[:40]:
        cid = ids[k]; P_ = FC[compf == cid]
        if cnt[k] < max(60, 0.002 * skin_e.sum()):
            continue
        c_ = P_.mean(0); ev = np.sort(np.linalg.eigvalsh(np.cov((P_ - c_).T)))[::-1]
        elong = math.sqrt(ev[0] / max(ev[1], 1e-12))
        ok = elong < 8.0
        comps.append((cid, int(cnt[k]), c_, elong, ok))
        if ok:
            keepc.add(cid)
    for cid, n_, c_, el_, ok in comps[:16]:
        say("  skin piece", n_, "faces at", np.round((c_ - np.array([fx, 0, z0])) / ST, 3), "elongation", round(el_, 1), "kept" if ok else "dropped")
    kept = np.isin(compf, list(keepc)) & skin_e
    SP = FC[kept]
    SPc = compf[kept]
    say("skin faces used for landmarks", len(SP))
    # ---- head (from the face camera)
    m_ = (np.abs(SP[:, 0] - fx) < 0.03 * ST) & (np.abs(SP[:, 2] - z_eye) < 0.012 * ST)
    yfront = float(np.percentile(SP[m_, 1], 3)) if m_.sum() > 10 else float(np.percentile(FC[:, 1], 1))
    depth = 0.886 * Hhead
    head_y = yfront + 0.5 * depth
    say("head: eye z", round((z_eye - z0) / ST, 3), "chin z", round((z_chin - z0) / ST, 3), "height", round(Hhead / ST, 3))
    TJ = {}
    Hd = np.array([fx, head_y, z_chin + 0.22 * Hhead]); top = np.array([fx, head_y, z_chin + Hhead])
    Nk = np.array([fx, head_y + 0.12 * depth, Hd[2] - 0.40 * Hhead])
    TJ["head"] = (Hd, top); TJ["neck"] = (Nk, Hd)
    # ---- inner surface of her (skin + robe + armour), with the layer of every face
    iv = [BW]; ifc = [[p for p, sk in zip(Bpol, skinf) if sk]]; ilay = [0] * len(ifc[0]); off = len(BW)
    for li_, k_ in ((1, "Robe"), (2, "Armor")):
        o = lay.get(k_)
        if o:
            V_ = wverts(o); P_ = polys(o)
            iv.append(V_); ifc.append([tuple(i + off for i in p) for p in P_]); ilay += [li_] * len(P_); off += len(V_)
    IV = np.vstack(iv); IF = [f for fl in ifc for f in fl]; ILAY = np.array(ilay)
    itree = BVHTree.FromPolygons([tuple(p) for p in IV], IF)
    # all of her (for inside / outside tests): every Body face too
    av = [BW]; af = list(Bpol); off = len(BW)
    for k_ in ("Robe", "Armor"):
        o = lay.get(k_)
        if o:
            V_ = wverts(o); av.append(V_); af += [tuple(i + off for i in p) for p in polys(o)]; off += len(V_)
    atree = BVHTree.FromPolygons([tuple(p) for p in np.vstack(av)], af)

    def inside(p):
        """Signed distance to her nearest surface: negative inside."""
        h = atree.find_nearest(Vector(p))
        if h[0] is None:
            return 1.0
        d = np.asarray(p) - np.array(h[0])
        return float(np.linalg.norm(d)) * (1.0 if float(d @ np.array(h[1])) > 0 else -1.0)

    def circle(P2):
        """Kasa least-squares circle with two trimmed re-fits: centre, radius."""
        Q = P2.copy()
        for _ in range(3):
            A_ = np.c_[2 * Q, np.ones(len(Q))]; b_ = (Q ** 2).sum(1)
            sol = np.linalg.lstsq(A_, b_, rcond=None)[0]
            c_ = sol[:2]; r_ = math.sqrt(max(sol[2] + c_ @ c_, 1e-12))
            res = np.abs(np.linalg.norm(P2 - c_, axis=1) - r_)
            Q = P2[res <= np.percentile(res, 80)] if len(P2) > 12 else P2
        return c_, r_
    # ---- legs: two clusters per slice, a circle through each
    legpts = SP[(SP[:, 2] > z0 + 0.1 * ST) & (SP[:, 2] < z0 + 0.6 * ST) & (np.abs(SP[:, 0] - fx) < 0.2 * ST)]
    cx0 = float(np.median(legpts[:, 0])) if len(legpts) else fx
    rows = {+1: [], -1: []}; joined = []
    for zf in np.arange(0.14, 0.58, 0.012):
        zz = z0 + zf * ST
        sl = legpts[np.abs(legpts[:, 2] - zz) < 0.006 * ST]
        if len(sl) < 30:
            continue
        xs_ = np.sort(sl[:, 0]); gaps_ = np.diff(xs_); mid = (xs_[:-1] + xs_[1:]) / 2
        cen = np.abs(mid - cx0) < 0.08 * ST
        if not cen.any():
            continue
        k = int(np.argmax(np.where(cen, gaps_, -1)))
        if gaps_[k] < 0.004 * ST:
            joined.append(zz); continue
        cut = mid[k]
        for sgn in (+1, -1):
            q = sl[(sl[:, 0] - cut) * sgn > 0]
            if len(q) < 15:
                continue
            c_, r_ = circle(q[:, :2])
            if 0.02 * ST < r_ < 0.12 * ST and abs(c_[0] - cut) < 0.16 * ST:
                rows[sgn].append((zz, c_[0], c_[1], r_))
    for sgn in (+1, -1):
        say("  leg", "L" if sgn > 0 else "R", "circles", [(round((r[0] - z0) / ST, 3), round((r[1] - fx) / ST, 3), round(r[3] / ST, 3)) for r in rows[sgn]])
    jz = [z for z in joined if z > z0 + 0.4 * ST]
    z_crotch = min(jz) if jz else z0 + 0.48 * ST
    z_crotch = float(np.clip(z_crotch, z0 + 0.45 * ST, z0 + 0.52 * ST))
    say("crotch z", round((z_crotch - z0) / ST, 3), "centre x", round((cx0 - fx) / ST, 3))
    z_hip = z_crotch + 0.05 * ST; z_ank = z0 + 0.07 * ST; z_knee = z_ank + 0.5 * (z_hip - z_ank)

    def side_pts(zlo, zhi, sgn):
        out = []
        for k_ in ("Robe", "Armor"):
            o = lay.get(k_)
            if o:
                V_ = wverts(o)
                m2 = (V_[:, 2] > zlo) & (V_[:, 2] < zhi) & ((V_[:, 0] - cx0) * sgn > 0.03 * ST) & ((V_[:, 0] - cx0) * sgn < 0.2 * ST)
                out.append(V_[m2])
        return np.vstack(out) if out else np.zeros((0, 3))
    for sd, sgn in ((".L", +1), (".R", -1)):
        R_ = np.array([r for r in rows[sgn] if r[0] < z_crotch]) if rows[sgn] else np.zeros((0, 4))
        if len(R_) >= 3:
            cxp = np.polyfit(R_[:, 0], R_[:, 1], 1); cyp = np.polyfit(R_[:, 0], R_[:, 2], 1)
        else:
            cxp = np.array([0.0, cx0 + sgn * 0.06 * ST]); cyp = np.array([0.0, head_y])
        at = lambda z: np.array([np.polyval(cxp, z), np.polyval(cyp, z), z])
        hip = at(z_hip); hip[0] = cx0 + sgn * max(abs(hip[0] - cx0) * 0.9, 0.04 * ST)
        knee = at(z_knee)
        # the ankle: centre of the boot just above the heel, near the line carried down
        bp = side_pts(z_ank - 0.02 * ST, z_ank + 0.03 * ST, sgn)
        guess = at(z_ank)
        if len(bp) > 40:
            near_ = bp[np.linalg.norm(bp[:, :2] - guess[:2], axis=1) < 0.07 * ST]
            if len(near_) > 20:
                c_, r_ = circle(near_[:, :2])
                if np.linalg.norm(c_ - guess[:2]) < 0.06 * ST:
                    guess[:2] = c_
        ank = guess
        toe = ank + np.array([0.0, -0.085 * ST, 0.0]); toe[2] = z0 + 0.012 * ST
        TJ["legU" + sd] = (hip, knee); TJ["legL" + sd] = (knee, ank); TJ["foot" + sd] = (ank, toe)
    cxb = cx0
    # ---- shoulders from her width just below the neck
    cxt = 0.5 * (cx0 + fx)
    Nk[0] = cxt
    zs = Nk[2] - 0.035 * ST
    hw = {}
    for sgn in (+1, -1):
        o_ = np.array([cxt, head_y + 0.05 * depth, zs])
        h_ = itree.ray_cast(Vector(o_), Vector((sgn, 0, 0)), 0.4 * ST)
        hw[sgn] = h_[3] if h_[0] is not None and h_[3] > 0.05 * ST else 0.1 * ST
    for sd, sgn in ((".L", +1), (".R", -1)):
        TJ["armU" + sd] = [np.array([cxt + sgn * 0.8 * hw[sgn], Nk[1], zs]), None]
    say("shoulder half widths", round(hw[1] / ST, 3), round(hw[-1] / ST, 3))
    torso(TJ, TJ["legU.L"][0], TJ["legU.R"][0], TJ["armU.L"][0], TJ["armU.R"][0], Nk)
    TJ["neck"] = (Nk, Hd)
    ln = lambda d, keys: sum(np.linalg.norm(d[k][1] - d[k][0]) for k in keys)
    chain = ("hips", "spine", "chest", "legU.L", "legL.L", "legU.R", "legL.R")
    G = ln(TJ, chain) / ln(BJ, chain)
    say("overall scale", round(G, 4))
    # ---- hands: a compact piece of skin within reach of each shoulder; elbow chosen to lie inside her
    head_c = 0.5 * (Hd + top)
    for sd, sgn in ((".L", +1), (".R", -1)):
        sh = TJ["armU" + sd][0]
        LA = np.linalg.norm(BJ["armU" + sd][1] - BJ["armU" + sd][0]) * G
        LB = np.linalg.norm(BJ["armL" + sd][1] - BJ["armL" + sd][0]) * G
        Lh = np.linalg.norm(BJ["hand" + sd][1] - BJ["hand" + sd][0]) * G
        best_ = None
        for cid, n_, c_, el_, ok in comps:
            if not ok or (c_[0] - cx0) * sgn < 0.13 * ST or c_[2] < z0 + 0.5 * ST:     # hands are out beside her, not on her body
                continue
            if np.linalg.norm(c_ - head_c) < 0.12 * ST or np.linalg.norm(c_ - sh) > LA + LB + Lh:
                continue
            if best_ is None or n_ > best_[1]:
                best_ = (cid, n_, c_)
        hc = best_[2] if best_ else sh + np.array([sgn * 0.2, -0.05, -0.25]) * ST
        say("hand", sd, "piece", best_[1] if best_ else None, "at", np.round((hc - np.array([fx, 0, z0])) / ST, 3))
        u_ = (hc - sh) / max(np.linalg.norm(hc - sh), 1e-9)
        wr = hc - u_ * 0.4 * Lh
        dv_ = wr - sh; d_ = np.linalg.norm(dv_)
        if d_ >= 0.98 * (LA + LB):
            el = sh + dv_ * (LA / (LA + LB)); sc_best = None
        else:
            a_ = (LA * LA - LB * LB + d_ * d_) / (2 * d_)
            hgt_ = math.sqrt(max(LA * LA - a_ * a_, 0.0))
            ax_ = dv_ / d_
            e1_ = np.cross(ax_, [0, 0, 1.0]); e1_ /= max(np.linalg.norm(e1_), 1e-9); e2_ = np.cross(ax_, e1_)
            sc_best = None; el = None
            for th in np.linspace(0, 2 * math.pi, 24, endpoint=False):
                cand = sh + ax_ * a_ + hgt_ * (math.cos(th) * e1_ + math.sin(th) * e2_)
                pts = [sh + (cand - sh) * t for t in (0.25, 0.5, 0.75, 1.0)] + [cand + (wr - cand) * t for t in (0.25, 0.5, 0.75)]
                scv = sum(inside(p) for p in pts)
                if sc_best is None or scv < sc_best:
                    sc_best, el = scv, cand
        u_ = (hc - el) / max(np.linalg.norm(hc - el), 1e-9)
        wr = hc - u_ * 0.4 * Lh
        TJ["armU" + sd] = (sh, el); TJ["armL" + sd] = (el, wr); TJ["hand" + sd] = (wr, wr + u_ * Lh)
    for s in SEG:
        mid = 0.5 * (TJ[s][0] + TJ[s][1])
        say("her joint", s, np.round((TJ[s][0] - np.array([fx, 0, z0])) / ST, 3), np.round((TJ[s][1] - np.array([fx, 0, z0])) / ST, 3),
            "middle inside her" if inside(mid) < 0 else "MIDDLE OUTSIDE HER (%.3f)" % (inside(mid) / ST))
    # ------------------------------------------------------------ C. pose by segment
    Pn = np.zeros_like(BV)
    for s in SEG:
        h0, t0 = BJ[s]; h1, t1 = TJ[s]
        d0 = t0 - h0; d1 = t1 - h1
        L0 = np.linalg.norm(d0); L1 = np.linalg.norm(d1)
        R = np.array(Vector(d0).rotation_difference(Vector(d1)).to_matrix())
        sL = L1 / L0
        sR = sL if s.split(".")[0] in ("hand", "foot") else G
        if s == "head":
            sL = sR = 0.96 * sL
        a = d0 / L0
        S = sR * np.eye(3) + (sL - sR) * np.outer(a, a)
        M = R @ S
        Pn += Wt[:, SI[s], None] * (h1 + (BV - h0) @ M.T)
        say("segment", s, "length x", round(sL, 3), "girth x", round(sR, 3), "turn deg", round(math.degrees(math.acos(np.clip(np.dot(d0, d1) / (L0 * L1), -1, 1))), 1))

    # ------------------------------------------------------------ E. her shape, without the clothing
    # Station fit: along every bone, at 12 cross-sections, rays go out from the
    # bone to her innermost surface (skin, or clothing lying on it, minus its
    # thickness) and to the posed mannequin. Their ratio becomes a smooth
    # profile per cross-section: one number for limbs, a low Fourier series
    # (up to the 2nd harmonic) for the torso, so the result stays smooth and
    # featureless by construction. Loose cloth only ever adds radius, so where
    # rays hit the robe a low percentile is used. Cross-sections nobody can see
    # (under the skirt) are interpolated and logged as estimates. The fit is
    # repeated with a girth correction until the base matches her.
    THK = np.array([0.0, 0.003 * HH, 0.009 * HH])           # skin, robe, armour thickness taken off
    NA = 64; NS = 12
    TH = np.linspace(0, 2 * math.pi, NA, endpoint=False)
    US = np.linspace(0.1, 0.9, NS)
    FIT = [s for s in SEG if s.split(".")[0] not in ("hand", "head")]
    TORSO = ("hips", "spine", "chest")

    def frame(s):
        h, t = TJ[s]; d = t - h; L_ = np.linalg.norm(d); d = d / L_
        e1 = np.cross(d, [0, 1.0, 0])
        if np.linalg.norm(e1) < 0.3:
            e1 = np.cross(d, [0, 0, 1.0])
        e1 /= np.linalg.norm(e1); e2 = np.cross(d, e1)
        return h, d, L_, e1, e2

    def rays(tree, c, e1, e2, cap, lay_of=None):
        r = np.full(NA, np.nan); ly = np.zeros(NA, int)
        for k, a_ in enumerate(TH):
            v = math.cos(a_) * e1 + math.sin(a_) * e2
            hit = tree.ray_cast(Vector(c), Vector(v), cap if np.isscalar(cap) else (cap[k] if np.isfinite(cap[k]) else 0.6 * HH))
            if hit[0] is not None:
                r[k] = hit[3]
                if lay_of is not None:
                    ly[k] = lay_of[hit[2]]
        return r, ly

    def fill_periodic(r):
        ok = np.isfinite(r)
        if ok.sum() < 3:
            return None
        idx = np.arange(NA)
        return np.interp(idx, idx[ok], r[ok], period=NA)

    def perim(c, e1, e2, r):
        pts = c + r[:, None] * (np.cos(TH)[:, None] * e1 + np.sin(TH)[:, None] * e2)
        return float(np.linalg.norm(np.roll(pts, -1, 0) - pts, axis=1).sum())

    def mtree(P):
        return BVHTree.FromPolygons([tuple(p) for p in P], BP)
    mt0 = mtree(Pn)
    # her side, measured once
    HER = {}
    for s in FIT:
        h, d, L_, e1, e2 = frame(s)
        st = []
        for u in US:
            c = h + d * L_ * u
            rm, _ = rays(mt0, c, e1, e2, 0.6 * HH)
            cap = 1.6 * rm
            rh, ly = rays(itree, c, e1, e2, cap, ILAY)
            rh = rh - THK[ly]
            ok = np.isfinite(rh) & (rh > 0.004 * HH)
            if s not in TORSO and ok.sum() > 8:
                skin_ok = ok & (ly == 0)
                ref = np.median(rh[skin_ok]) if skin_ok.sum() >= 8 else np.percentile(rh[ok], 40)
                ok &= rh < 1.3 * ref
            st.append(dict(c=c, rm=rm, rh=np.where(ok, rh, np.nan), ly=ly, n=int(ok.sum()), robe=float((ly[ok] > 0).mean()) if ok.any() else 1.0))
        HER[s] = st

    def harm(n):
        return np.stack([np.ones(NA)] + [f(k * TH) for k in range(1, n + 1) for f in (np.cos, np.sin)], 1)
    B0, B2 = harm(0), harm(2)

    def station_coef(s, stt):
        ok = np.isfinite(stt["rh"]) & np.isfinite(stt["rm"]) & (stt["rm"] > 1e-6)
        if ok.sum() < 16:
            return None
        q = stt["rh"][ok] / stt["rm"][ok]
        if s in TORSO and ok.sum() >= 32:
            # robust low-order fit; loose cloth handled by trimming the high side
            keep = np.ones(ok.sum(), bool)
            for _ in range(3):
                c_ = np.linalg.lstsq(B2[ok][keep], q[keep], rcond=None)[0]
                res = q - B2[ok] @ c_
                keep = res < np.percentile(res, 75) + 0.02
            return c_
        a0 = np.percentile(q, 30) if stt["robe"] > 0.3 else np.median(q)
        return np.r_[a0, np.zeros(4)]
    COEF = {}; EST = {}
    for s in FIT:
        cs = [station_coef(s, stt) for stt in HER[s]]
        okk = [c_ is not None for c_ in cs]
        EST[s] = [not k for k in okk]
        if not any(okk):
            cs = [np.r_[1.0, np.zeros(4)]] * NS
        else:
            iu = np.array([i for i, k in enumerate(okk) if k])
            M_ = np.array([cs[i] for i in iu])
            cs = [np.array([np.interp(i, iu, M_[:, j]) for j in range(5)]) for i in range(NS)]
        C_ = np.array(cs)
        for _ in range(2):                                      # smooth along the bone
            C_[1:-1] = 0.25 * C_[:-2] + 0.5 * C_[1:-1] + 0.25 * C_[2:]
        C_[:, 0] = np.clip(C_[:, 0], 0.55, 2.4)
        COEF[s] = C_
    say("fit: stations measured / estimated", {s: "%d/%d" % (NS - sum(EST[s]), sum(EST[s])) for s in FIT})
    # vertex -> (segment, u, theta) once
    FR = {s: frame(s) for s in SEG}
    UTH = {}
    for s in FIT:
        h, d, L_, e1, e2 = FR[s]
        rel = Pn - h
        u = np.clip((rel @ d) / L_, 0.0, 1.0)
        rad = rel - (rel @ d)[:, None] * d
        th = np.arctan2(rad @ e2, rad @ e1) % (2 * math.pi)
        UTH[s] = (u, th)
    Vbar = np.zeros_like(Pn)
    for s in SEG:
        h, d, L_, e1, e2 = FR[s]
        u = np.clip(((Pn - h) @ d) / L_, 0.0, 1.0)
        Vbar += Wt[:, SI[s], None] * (Pn - (h + (u * L_)[:, None] * d))
    keepw = Wt[:, [SI[x] for x in SEG if x.split(".")[0] in ("hand", "head")]].sum(1)

    def qfield():
        num = keepw * 1.0; den = keepw.copy()
        for s in FIT:
            w_ = Wt[:, SI[s]]
            sel = w_ > 1e-3
            u, th = UTH[s]
            pos = np.clip((u[sel] - US[0]) / (US[1] - US[0]), 0, NS - 1)
            i0 = np.floor(pos).astype(int); i1 = np.minimum(i0 + 1, NS - 1); f_ = pos - i0
            C_ = COEF[s]
            cc = C_[i0] * (1 - f_)[:, None] + C_[i1] * f_[:, None]
            t_ = th[sel]
            q = cc[:, 0] + cc[:, 1] * np.cos(t_) + cc[:, 2] * np.sin(t_) + cc[:, 3] * np.cos(2 * t_) + cc[:, 4] * np.sin(2 * t_)
            num[sel] += w_[sel] * q; den[sel] += w_[sel]
        return num / np.maximum(den, 1e-9)

    def taubin(P, it, mask=None):
        for _ in range(it):
            for lam in (0.5, -0.53):
                dlt = navg(P) - P
                P = P + lam * (dlt if mask is None else mask[:, None] * dlt)
        return P
    tw = Wt[:, [SI["hips"], SI["spine"], SI["chest"]]].sum(1)
    NR = 5
    for rnd in range(NR):
        q = qfield()
        for _ in range(6):
            q = 0.5 * q + 0.5 * navg(q)
        P = Pn + (q - 1.0)[:, None] * Vbar
        P = taubin(P, 6)
        P = taubin(P, 10, tw)                                   # the torso stays smooth and featureless
        if rnd == NR - 1:
            break
        bt = mtree(P)
        corr = []
        for s in FIT:                                           # girth correction per station
            h, d, L_, e1, e2 = FR[s]
            for i, stt in enumerate(HER[s]):
                if EST[s][i]:
                    continue
                rh = fill_periodic(stt["rh"])
                rb, _ = rays(bt, stt["c"], e1, e2, 0.6 * HH)
                rb = fill_periodic(rb)
                if rh is None or rb is None:
                    continue
                k = float(np.clip(perim(stt["c"], e1, e2, rh) / perim(stt["c"], e1, e2, rb), 0.8, 1.3))
                COEF[s][i] *= k; corr.append(k)
        say("fit round", rnd + 1, "girth corrections: median %.3f, range %.3f..%.3f" % (np.median(corr), min(corr), max(corr)) if corr else "none")
    tmp = bpy.data.meshes.new("Arcanist_BaseForm"); tmp.from_pydata([tuple(p) for p in P], [], BP); tmp.update()
    for p_ in tmp.polygons:
        p_.use_smooth = True
    # ---- measurements: her vs the base, at named cross-sections
    bt = mtree(P)
    say("measurements (girth, % of her stature): her innermost surface vs base form")
    for nm_, s, u in (("neck", "neck", 0.5), ("chest", "chest", 0.35), ("under chest", "chest", 0.12), ("waist", "spine", 0.5),
                      ("hips", "hips", 0.3), ("thigh L", "legU.L", 0.25), ("thigh R", "legU.R", 0.25), ("knee L", "legU.L", 0.9),
                      ("calf L", "legL.L", 0.35), ("calf R", "legL.R", 0.35), ("upper arm L", "armU.L", 0.5), ("upper arm R", "armU.R", 0.5),
                      ("forearm L", "armL.L", 0.4), ("forearm R", "armL.R", 0.4)):
        h, d, L_, e1, e2 = FR[s]
        i = int(np.argmin(np.abs(US - u))); stt = HER[s][i] if s in HER else None
        c = h + d * L_ * US[i]
        rb = fill_periodic(rays(bt, c, e1, e2, 0.6 * HH)[0])
        rh = fill_periodic(stt["rh"]) if stt else None
        gb = perim(c, e1, e2, rb) if rb is not None else None
        gh = perim(c, e1, e2, rh) if rh is not None else None
        tag = "estimate (hidden)" if (s in EST and EST[s][i]) else "%d rays" % (stt["n"] if stt else 0)
        say("  %-12s her %6s  base %6s  %s  [%s]" % (nm_, "-" if gh is None else "%.1f" % (gh / ST * 100), "-" if gb is None else "%.1f" % (gb / ST * 100),
            "" if (gh is None or gb is None) else "diff %+.1f%%" % ((gb - gh) / gh * 100), tag))
    # ------------------------------------------------------------ F. into the layered file
    old = bpy.data.objects.get("Arcanist_BaseForm")
    if old:
        bpy.data.objects.remove(old, do_unlink=True)
    bf = bpy.data.objects.new("Arcanist_BaseForm", tmp)
    col = bpy.data.collections.get("Base Form")
    if col is None:
        col = bpy.data.collections.new("Base Form"); sc.collection.children.link(col)
    col.objects.link(bf)
    clay = bpy.data.materials.new("Arcanist_Clay"); clay.use_nodes = True
    bs = next(n for n in clay.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
    bs.inputs["Base Color"].default_value = (0.62, 0.6, 0.58, 1); bs.inputs["Roughness"].default_value = 0.7
    tmp.materials.clear(); tmp.materials.append(clay)
    # ------------------------------------------------------------ G. check renders
    for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
        try:
            sc.render.engine = eng
            break
        except Exception:
            pass
    cd = bpy.data.cameras.new("basecam"); cd.type = "ORTHO"; cd.ortho_scale = HH * 1.08
    cam = bpy.data.objects.new("basecam", cd); sc.collection.objects.link(cam); sc.camera = cam
    sun = bpy.data.lights.new("basesun", "SUN"); sun.energy = 3.0
    so = bpy.data.objects.new("basesun", sun); sc.collection.objects.link(so)
    so.rotation_euler = (math.radians(55), 0, math.radians(-30))
    sc.render.resolution_x, sc.render.resolution_y = 700, 1000
    ctr = Vector((0.5 * (allP[:, 0].min() + allP[:, 0].max()), 0.5 * (allP[:, 1].min() + allP[:, 1].max()), z0 + 0.5 * HH))
    sets = (("base", ("Base Form",)), ("base_body", ("Base Form", "Body")), ("all", ("Body", "Robe", "Armor", "Hat")))
    say("renders: base_<set>_<view>.png, views 0 front, 1 three-quarter, 2 side, 3 back")
    for tag, show in sets:
        for c in sc.collection.children:
            c.hide_render = c.name not in show
        for k, ang in enumerate((-90, -50, 0, 90)):  # front, 3/4, side, back
            a = math.radians(ang)
            cam.location = ctr + Vector((math.cos(a), math.sin(a), 0)) * HH * 3
            cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
            so.rotation_euler = (math.radians(55), 0, a + math.radians(60))
            sc.render.filepath = os.path.join(L2, "base_%s_%d.png" % (tag, k))
            bpy.ops.render.render(write_still=True)
    try:
        jc = bpy.data.collections.new("Joints"); sc.collection.children.link(jc)
        red = bpy.data.materials.new("jred"); red.use_nodes = True
        nb_ = next(n for n in red.node_tree.nodes if n.type == "BSDF_PRINCIPLED"); nb_.inputs["Base Color"].default_value = (1, 0.05, 0.05, 1)
        V_ = []; F_ = []
        for s in SEG:
            for q in TJ[s]:
                b0 = len(V_); r_ = 0.008 * HH
                for dx_, dy_, dz_ in ((1, 0, 0), (-1, 0, 0), (0, 1, 0), (0, -1, 0), (0, 0, 1), (0, 0, -1)):
                    V_.append((q[0] + dx_ * r_, q[1] + dy_ * r_ - 0.25 * HH, q[2] + dz_ * r_))
                F_ += [(b0 + a_, b0 + b_, b0 + c_) for a_, b_, c_ in ((0, 2, 4), (2, 1, 4), (1, 3, 4), (3, 0, 4), (2, 0, 5), (1, 2, 5), (3, 1, 5), (0, 3, 5))]
            h_, t_ = TJ[s]; b0 = len(V_); r_ = 0.003 * HH
            for q in (h_, t_):
                for dx_, dz_ in ((1, 0), (0, 1), (-1, 0), (0, -1)):
                    V_.append((q[0] + dx_ * r_, q[1] - 0.25 * HH, q[2] + dz_ * r_))
            F_ += [(b0 + i_, b0 + (i_ + 1) % 4, b0 + 4 + (i_ + 1) % 4, b0 + 4 + i_) for i_ in range(4)]
        jm = bpy.data.meshes.new("joints"); jm.from_pydata(V_, [], F_); jm.update(); jm.materials.append(red)
        jo = bpy.data.objects.new("joints", jm); jc.objects.link(jo)
        for c in sc.collection.children:
            c.hide_render = c.name not in ("Joints", "Body")
        cam.location = ctr + Vector((0, -1, 0)) * HH * 3
        cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
        sc.render.filepath = os.path.join(L2, "base_joints.png")
        bpy.ops.render.render(write_still=True)
        bpy.data.objects.remove(jo, do_unlink=True); bpy.data.collections.remove(jc)
    except Exception as e:
        say("joint picture skipped", e)
    for c in sc.collection.children:
        c.hide_render = False
    bpy.data.objects.remove(cam, do_unlink=True); bpy.data.objects.remove(so, do_unlink=True)
    bpy.ops.wm.save_as_mainfile(filepath=LAYERS)
    say("saved", LAYERS)
except Exception as e:
    import traceback
    say("FAILED", e, traceback.format_exc())
open(os.path.join(L2, "basebody_done.txt"), "w").write("ok")
