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
    skin_wide = skinf.copy()
    if col is not None:                                         # her skin for measuring: the learned colour OR the broad warm-skin rule (glossy thighs)
        _mx = col.max(1); _mn = col.min(1); _sat = np.where(_mx > 1e-4, (_mx - _mn) / np.maximum(_mx, 1e-4), 0)
        _d = np.maximum(_mx - _mn, 1e-6)
        _hue = np.where(_mx == col[:, 0], ((col[:, 1] - col[:, 2]) / _d) % 6, np.where(_mx == col[:, 1], (col[:, 2] - col[:, 0]) / _d + 2, (col[:, 0] - col[:, 1]) / _d + 4)) * 60
        skin_wide |= (_hue < 36) & (_sat > 0.08) & (_sat < 0.6) & (_mx > 0.3) & (col[:, 0] > col[:, 2] * 1.05)
    say("skin faces for measuring", int(skin_wide.sum()))
    iv = [BW]; ifc = [[p for p, sk in zip(Bpol, skin_wide) if sk]]; ilay = [0] * len(ifc[0]); off = len(BW)
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
            # only the front of the thigh is skin, so the circle's depth is shaky:
            # put the centre half the thigh's width behind its front instead
            wdt = np.percentile(q[:, 0], 97) - np.percentile(q[:, 0], 3)
            cy_ = np.percentile(q[:, 1], 3) + 0.5 * wdt
            if 0.02 * ST < r_ < 0.12 * ST and abs(c_[0] - cut) < 0.16 * ST:
                rows[sgn].append((zz, 0.5 * (np.percentile(q[:, 0], 97) + np.percentile(q[:, 0], 3)), cy_, 0.5 * wdt))
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
    # her two sides read differently through the sleeves (a puffed sleeve
    # widens one side): both shoulders take the narrower, so the base is
    # symmetric and doesn't inherit the sleeve
    hwm = min(hw[1], hw[-1])
    for sd, sgn in ((".L", +1), (".R", -1)):
        TJ["armU" + sd] = [np.array([cxt + sgn * 0.8 * hwm, Nk[1], zs]), None]
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
            if not ok or (c_[0] - cx0) * sgn < 0.15 * ST or c_[2] < z0 + 0.5 * ST or c_[1] > head_y + 0.03 * ST:     # hands are out beside her, not on her body
                continue
            if np.linalg.norm(c_ - head_c) < 0.12 * ST or np.linalg.norm(c_ - sh) > LA + LB + Lh:
                continue
            # a real hand hangs off the arm: the line from the shoulder runs inside her sleeve
            ins = sum(1 for t in (0.2, 0.35, 0.5, 0.65, 0.8) if inside(sh + (c_ - sh) * t) < 0.004 * ST)
            if best_ is None or (ins, n_) > (best_[3], best_[1]):
                best_ = (cid, n_, c_, ins)
        hc = best_[2] if best_ else sh + np.array([sgn * 0.2, -0.05, -0.25]) * ST
        say("hand", sd, "piece", best_[1] if best_ else None, "inside", best_[3] if best_ else None, "at", np.round((hc - np.array([fx, 0, z0])) / ST, 3))
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
    NA = 64; NS = 14
    # the torso is fitted as one piece, hip joints to neck, so the hips, belly
    # and chest don't fight each other where their weights overlap
    TJ["torso"] = (TJ["hips"][0], TJ["chest"][1])
    TH = np.linspace(0, 2 * math.pi, NA, endpoint=False)
    US = np.linspace(0.1, 0.9, NS)
    FIT = ["torso"] + [s for s in SEG if s.split(".")[0] not in ("hand", "head", "hips", "spine", "chest")]
    TORSO = ("torso",)

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
            lim = cap if np.isscalar(cap) else (cap[k] if np.isfinite(cap[k]) else 0.6 * HH)
            o = Vector(c); vv = Vector(v); gone = 0.0
            for _ in range(5):
                hit = tree.ray_cast(o, vv, lim - gone)
                if hit[0] is None:
                    break
                if lay_of is not None and hit[1].dot(vv) < 0:
                    # an inner shell facing the bone (her model is a thick shell
                    # in places): not her outer surface, carry on through it
                    gone += hit[3] + 1e-5 * HH; o = hit[0] + vv * (1e-5 * HH)
                    continue
                r[k] = gone + hit[3]
                if lay_of is not None:
                    ly[k] = lay_of[hit[2]]
                break
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

    # each cross-section is measured against its own body part only, so an arm
    # crossing in front of the waist (or a leg beside a leg) isn't counted
    GROUP = {}
    for s_ in SEG:
        b_ = s_.split(".")[0]; sd_ = s_[-2:] if "." in s_ else ""
        GROUP[s_] = "torso" if b_ in ("hips", "spine", "chest", "neck") else "head" if b_ == "head" else ("arm" if b_ in ("armU", "armL", "hand") else "leg") + sd_
    GROUP["torso"] = "torso"
    fdom = np.array([int(np.argmax(Wt[list(f)].mean(0))) for f in BP])
    FG = {}
    for gname in set(GROUP.values()):
        FG[gname] = [f for f, d_ in zip(BP, fdom) if GROUP[SEG[d_]] == gname]

    def mtree(P):
        return {gname: BVHTree.FromPolygons([tuple(p) for p in P], fl) for gname, fl in FG.items() if fl}
    mt0 = mtree(Pn)
    # her side, measured once per cross-section (re-measured if recentred)
    def measure_her(s, c, e1, e2, u):
        rm, _ = rays(mt0[GROUP[s]], c, e1, e2, 0.6 * HH)
        cap = (2.2 if s in TORSO else 2.6) * rm           # the slim mannequin is much thinner than her; loose cloth is filtered below
        rh, ly = rays(itree, c, e1, e2, cap, ILAY)
        rh = rh - THK[ly]
        ok = np.isfinite(rh) & (rh > 0.004 * HH)
        if s in TORSO and ok.sum() > 16:
            ok &= ~((ly == 1) & (rh > 1.3 * np.median(rh[ok])))  # far robe hits (the cape hanging off her back) are not her
        if s not in TORSO and ok.sum() > 8:
            skin_ok = ok & (ly == 0)
            ref = np.median(rh[skin_ok]) if skin_ok.sum() >= 8 else np.percentile(rh[ok], 40)
            ok &= rh < 1.3 * ref
        skip = (s == "torso" and (u < 0.33 or u > 0.76)) or (s.startswith("legU") and u < 0.18) or (s.startswith("armU") and u < 0.2)
        return dict(c=c, rm=rm, rh=np.where(ok, rh, np.nan), ly=ly, n=int(ok.sum()), nb=int(np.isfinite(rm).sum()),
                    robe=float((ly[ok] > 0).mean()) if ok.any() else 1.0, skip=skip)
    HER = {}
    for s in FIT:
        h, d, L_, e1, e2 = frame(s)
        st = []
        for u in US:
            c0 = h + d * L_ * u
            m_ = measure_her(s, c0, e1, e2, u)
            # recentre on her own cross-section (the bone may sit off-centre in her limb)
            for _ in range(2):
                okr = np.isfinite(m_["rh"])
                if okr.sum() < 16:
                    break
                pts = m_["c"] + m_["rh"][:, None] * (np.cos(TH)[:, None] * e1 + np.sin(TH)[:, None] * e2)
                cn = pts[okr].mean(0)
                cn = cn - d * ((cn - c0) @ d)                    # stay in the cross-section's plane
                r0 = float(np.nanmedian(m_["rh"]))
                if np.linalg.norm(cn - c0) > 0.5 * r0 or inside(cn) > 0:
                    break
                m_ = measure_her(s, cn, e1, e2, u)
            st.append(m_)
        HER[s] = st

    def harm(n):
        return np.stack([np.ones(NA)] + [f(k * TH) for k in range(1, n + 1) for f in (np.cos, np.sin)], 1)
    B2 = harm(2)
    # ---- targets: her girth at each trustworthy cross-section, cleaned
    TG = {}; EST = {}
    for s in FIT:
        h, d, L_, e1, e2 = frame(s)
        g_ = np.full(NS, np.nan)
        for i, stt in enumerate(HER[s]):
            if stt["skip"] or stt["n"] < 28 or stt["nb"] < (28 if s.startswith("leg") else 48):     # the leg pieces are open at the hip, so fewer base rays land there
                continue
            rh = fill_periodic(stt["rh"])
            if rh is not None:
                g_[i] = perim(stt["c"], e1, e2, rh)
        if s.startswith("legU") or s.startswith("armU") or s == "torso":
            say("   %s stations (u: her rays/base rays/girth%%)" % s, " ".join("%.2f:%d/%d/%s" % (US[i], st_["n"], st_["nb"], "-" if not np.isfinite(g_[i]) else "%.0f" % (g_[i] / ST * 100)) for i, st_ in enumerate(HER[s])))
        tol = 0.35 if s in TORSO else 0.25
        okk = np.isfinite(g_)
        for i in np.where(okk)[0]:                              # drop cross-sections far off their neighbours
            nbv = [g_[j] for j in range(max(0, i - 2), min(NS, i + 3)) if j != i and okk[j]]
            if len(nbv) >= 2 and abs(g_[i] / np.median(nbv) - 1) > tol:
                g_[i] = np.nan
        okk = np.isfinite(g_)
        lg = np.log(np.where(okk, g_, 1.0)); sm = lg.copy()
        for i in np.where(okk)[0]:                              # smooth the target the way the model is smooth
            w_ = [0.5] + [0.25 for j in (i - 1, i + 1) if 0 <= j < NS and okk[j]]
            v_ = [lg[i]] + [lg[j] for j in (i - 1, i + 1) if 0 <= j < NS and okk[j]]
            sm[i] = np.dot(w_, v_) / sum(w_)
        TG[s] = np.where(okk, np.exp(sm), np.nan)
    for a_, b_ in (("armU.L", "armU.R"), ("armL.L", "armL.R")):  # an arm hidden in its sleeve borrows the other arm's profile
        for x_, y_ in ((a_, b_), (b_, a_)):
            if np.isfinite(TG[x_]).sum() < 3 and np.isfinite(TG[y_]).sum() >= 3:
                TG[x_] = TG[y_].copy()
                say("fit:", x_, "borrows the profile of", y_)
    for s in FIT:
        EST[s] = list(~np.isfinite(TG[s]))
    say("fit: trusted cross-sections", {s: int(np.isfinite(TG[s]).sum()) for s in FIT})
    # ---- fixed shape terms for the torso (front/back, side-to-side), from her rays
    HARM = {s: np.zeros((NS, 4)) for s in FIT}
    A0 = {}
    for s in FIT:
        a0 = np.full(NS, np.nan)
        for i, stt in enumerate(HER[s]):
            ok = np.isfinite(stt["rh"]) & np.isfinite(stt["rm"]) & (stt["rm"] > 1e-6)
            if ok.sum() < 16 or stt["skip"]:
                continue
            q = stt["rh"][ok] / stt["rm"][ok]
            if s in TORSO and ok.sum() >= 32:
                keep = np.ones(ok.sum(), bool)
                for _ in range(3):
                    c_ = np.linalg.lstsq(B2[ok][keep], q[keep], rcond=None)[0]
                    res = q - B2[ok] @ c_
                    keep = res < np.percentile(res, 75) + 0.02
                a0[i] = c_[0]; HARM[s][i] = 0.6 * c_[1:] / max(c_[0], 0.3)
            else:
                a0[i] = np.percentile(q, 30) if stt["robe"] > 0.3 else np.median(q)
        ok = np.isfinite(a0)
        A0[s] = np.interp(np.arange(NS), np.where(ok)[0], a0[ok]) if ok.any() else np.ones(NS)
        for _ in range(3):
            HARM[s][1:-1] = 0.25 * HARM[s][:-2] + 0.5 * HARM[s][1:-1] + 0.25 * HARM[s][2:]
    # ---- the parameters: a few knots of log-size per bone, smooth by construction
    NK = {s: (6 if s in TORSO else 3) for s in FIT}
    KPOS = {s: np.linspace(0, NS - 1, NK[s]) for s in FIT}
    X = {s: np.log(np.clip(np.interp(KPOS[s], np.arange(NS), A0[s]), 0.5, 2.2)) for s in FIT}
    # the pelvis below the hip line is shaped by the thighs, not by a torso slice
    # (a slice there cuts between the legs and turned into a skirt)
    legx = float(np.mean([np.median(X[x]) for x in ("legU.L", "legU.R")]))
    LOWK = [k for k in range(NK["torso"]) if US[int(round(KPOS["torso"][k]))] < 0.33]
    for k in LOWK:
        X["torso"][k] = min(X["torso"][k], legx)
    HIGHK = [k for k in range(NK["torso"]) if US[int(round(KPOS["torso"][k]))] > 0.72]
    for k in HIGHK:
        X["torso"][k] = min(X["torso"][k], 0.0)
    for sd_ in (".L", ".R"):                                    # upper arms hidden in the sleeves: start from the forearm's scale
        if np.isfinite(TG["armU" + sd_]).sum() < 3:
            X["armU" + sd_][:] = min(float(np.median(X["armL" + sd_])), math.log(1.2))
    if "neck" in X and np.isfinite(TG["neck"]).sum() < 3:
        X["neck"][:] = 0.0                                      # hidden neck: the mannequin's own
    COEF = {}

    def coef_from_X(XX):
        for s in FIT:
            a0 = np.exp(np.interp(np.arange(NS), KPOS[s], XX[s]))
            COEF[s] = np.c_[a0, HARM[s] * a0[:, None]]
    # vertex -> (segment, u, theta) once
    FR = {s: frame(s) for s in list(SEG) + ["torso"]}
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
            w_ = Wt[:, SI[s]] if s in SI else Wt[:, [SI["hips"], SI["spine"], SI["chest"]]].sum(1)
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

    def forward(XX):
        coef_from_X(XX)
        q = qfield()
        for _ in range(5):
            q = 0.5 * q + 0.5 * navg(q)
        P = Pn + (q - 1.0)[:, None] * Vbar
        P = taubin(P, 4)
        return taubin(P, 4, tw)                                 # the torso stays smooth and featureless
    KEYS = [(s, i) for s in FIT for i in range(NS) if np.isfinite(TG[s][i])]
    GT = np.array([TG[s][i] for s, i in KEYS])

    def girths(P):
        bt = mtree(P); out = []
        for s, i in KEYS:
            h, d, L_, e1, e2 = FR[s]
            rb = fill_periodic(rays(bt[GROUP[s]], HER[s][i]["c"], e1, e2, 0.6 * HH)[0])
            out.append(perim(HER[s][i]["c"], e1, e2, rb) if rb is not None else np.nan)
        return np.array(out)
    # ---- solve all knots together (least squares with a measured Jacobian)
    BODYGROUP = {"torso": "T", "neck": "T"}
    for s in FIT:
        if s not in BODYGROUP:
            BODYGROUP[s] = ("A" if s.startswith("arm") else "L") + s[-2:]
    PAR = [(s, k) for s in FIT for k in range(NK[s])]
    PI = {p: j for j, p in enumerate(PAR)}
    KG = {}
    for s, k in PAR:
        KG.setdefault(BODYGROUP[s], []).append((s, k))
    nmax = max(len(v) for v in KG.values())
    KEYG = np.array([BODYGROUP[s] for s, i in KEYS])

    def jacobian(XX, g0):
        J = np.zeros((len(KEYS), len(PAR)))
        for j in range(nmax):
            X2 = {s: v.copy() for s, v in XX.items()}
            moved = []
            for gname, plist in KG.items():
                if j < len(plist):
                    s, k = plist[j]; X2[s][k] += 0.1; moved.append((gname, s, k))
            g1 = girths(forward(X2))
            for gname, s, k in moved:
                rows = KEYG == gname
                J[rows, PI[(s, k)]] = (np.log(g1[rows]) - np.log(g0[rows])) / 0.1
        return np.nan_to_num(J)

    def D2rows():
        R_ = []
        for s in FIT:
            for k in range(1, NK[s] - 1):
                r_ = np.zeros(len(PAR)); r_[PI[(s, k - 1)]] = 1; r_[PI[(s, k)]] = -2; r_[PI[(s, k + 1)]] = 1
                R_.append(r_)
        return np.array(R_) if R_ else np.zeros((0, len(PAR)))
    D2 = D2rows()
    P = forward(X); g = girths(P)
    for outer in range(2):
        J = jacobian(X, g)
        live = np.linalg.norm(J, axis=0) >= 0.05
        say("fit: Jacobian built; %d of %d knots move a measured girth" % (int(live.sum()), len(PAR)))
        for it in range(4):
            okr = np.isfinite(g)
            r = np.where(okr, np.log(GT / np.where(okr, g, 1.0)), 0.0)
            rms = float(np.sqrt(np.mean(r[okr] ** 2)))
            say("fit: outer %d step %d  RMS girth error %.1f%%" % (outer + 1, it + 1, rms * 100))
            if rms < 0.03:
                break
            PR = []; pb = []                                    # low pelvis knots held near the thighs' scale
            legx = float(np.mean([np.median(X[x]) for x in ("legU.L", "legU.R")]))
            for k in LOWK:
                row = np.zeros(len(PAR)); row[PI[("torso", k)]] = 1.0; PR.append(row[live]); pb.append(legx - X["torso"][k])
            for k in HIGHK:                                     # shoulder-top knots held at the mannequin's size
                if PI[("torso", k)] < len(live) and live[PI[("torso", k)]]:
                    row = np.zeros(len(PAR)); row[PI[("torso", k)]] = 1.0; PR.append(row[live]); pb.append(0.0 - X["torso"][k])
            PR = np.array(PR).reshape(-1, int(live.sum())); pb = np.array(pb)
            A_ = np.vstack([J[:, live] * okr[:, None], math.sqrt(0.3) * D2[:, live], math.sqrt(0.05) * np.eye(int(live.sum())), 1.0 * PR])
            b_ = np.r_[r, np.zeros(len(D2)), np.zeros(int(live.sum())), pb]
            dx = np.linalg.lstsq(A_, b_, rcond=None)[0]
            full = np.zeros(len(PAR)); full[live] = dx
            for (s, k), j in PI.items():
                X[s][k] = float(np.clip(X[s][k] + full[j], math.log(0.5), math.log(2.2)))
                if s == "torso" and k in HIGHK:
                    X[s][k] = min(X[s][k], 0.05)
            P = forward(X); g = girths(P)
    # her arms are all sleeve: what was measured there is cloth, so the base
    # keeps slim arms (no thicker than the mannequin's own) instead
    for sd_ in (".L", ".R"):
        for s_ in ("armU", "armL"):
            if s_ + sd_ in X:
                X[s_ + sd_] = np.minimum(X[s_ + sd_], 0.0)
    P = forward(X); g = girths(P)
    okr = np.isfinite(g)
    r = np.log(GT[okr] / g[okr])
    say("fit: final RMS girth error %.1f%% over %d cross-sections" % (float(np.sqrt(np.mean(r ** 2))) * 100, int(okr.sum())))
    rms_knots = float(np.sqrt(np.mean(r ** 2)))
    # ---- second way: fit every vertex straight onto her surface where it can
    # be seen (now that the bones sit inside her), fill the hidden parts from
    # their neighbours, smooth. Keep whichever matches her girths better.
    r0 = np.linalg.norm(Vbar, axis=1)
    qv = np.full(len(Pn), np.nan); lyv = np.zeros(len(Pn), int)
    for i_ in range(len(Pn)):
        if keepw[i_] > 0.5 or r0[i_] < 0.003 * HH:
            continue
        dv = Vbar[i_] / r0[i_]
        o_ = Vector(Pn[i_] - Vbar[i_]); vv_ = Vector(dv); gone_ = 0.0; lim_ = 3.0 * r0[i_] + 0.03 * HH
        for _ in range(5):
            hit = itree.ray_cast(o_, vv_, lim_ - gone_)
            if hit[0] is None:
                break
            if hit[1].dot(vv_) < 0:
                gone_ += hit[3] + 1e-5 * HH; o_ = hit[0] + vv_ * (1e-5 * HH); continue
            lyv[i_] = ILAY[hit[2]]
            qv[i_] = max(gone_ + hit[3] - THK[lyv[i_]], 0.25 * r0[i_]) / r0[i_]
            break
    val = np.isfinite(qv) & (qv > 0.4) & (qv < 3.2)
    for _ in range(3):                                          # drop rays far off their neighbourhood (cloth, props)
        lq = np.log(np.where(val, qv, 1.0))
        num = np.where(val, lq, 0.0); den = val.astype(float)
        for _ in range(12):
            num = navg(num); den = navg(den)
        loc = num / np.maximum(den, 1e-9)
        val &= np.abs(lq - loc) < math.log(1.3)
    say("vertex fit: %d of %d vertices see her surface" % (int(val.sum()), int((keepw <= 0.5).sum())))
    qf = np.where(val, qv, np.nan)
    seedv = np.nanmedian(qf) if np.isfinite(qf).any() else 1.0
    qf = np.where(val, qv, seedv); qf = np.where(keepw > 0.5, 1.0, qf)
    anc = val | (keepw > 0.5); qa = qf.copy()
    for _ in range(120):
        qf = navg(qf); qf[anc] = qa[anc]
    for _ in range(6):
        qf = 0.5 * qf + 0.5 * navg(qf)
    qf = keepw * 1.0 + (1 - keepw) * qf
    Pv = Pn + (qf - 1.0)[:, None] * Vbar
    Pv = taubin(Pv, 6)
    Pv = taubin(Pv, 8, tw)
    gv = girths(Pv)
    okv = np.isfinite(gv)
    rv_ = np.log(GT[okv] / gv[okv])
    rms_vert = float(np.sqrt(np.mean(rv_ ** 2)))
    say("vertex fit: RMS girth error %.1f%% (knot fit %.1f%%)" % (rms_vert * 100, rms_knots * 100))
    if rms_vert < rms_knots:
        P, g = Pv, gv
        say("using the vertex fit")
    else:
        say("using the knot fit")
    # ---- final pass: SHAPE FIT. The knot fit gets the proportions roughly
    # right but not her shape (no waist, no curve of the hips and rear).
    # Here the base surface is pulled straight onto her own surface, inward
    # AND outward: onto her bare skin, and onto clothing that lies on her body
    # (bodice, the robe where it hugs her hips and rear, boots), minus the
    # cloth's thickness. Loose cloth (sleeves, the cape's flare, hems), hair,
    # the hat and ornaments are not targets: they are filtered out by colour,
    # by reach, and by disagreeing with their neighbours. Arms are left alone
    # (her arms are all sleeve). Repeated a few times, smoothing in between.
    def vnormals(P_):
        TRI_ = np.array([(f[0], f[i], f[i + 1]) for f in BP for i in range(1, len(f) - 1)], dtype=np.int64)
        fn_ = np.cross(P_[TRI_[:, 1]] - P_[TRI_[:, 0]], P_[TRI_[:, 2]] - P_[TRI_[:, 0]])
        vn = np.zeros_like(P_)
        for j in range(3):
            np.add.at(vn, TRI_[:, j], fn_)
        vn /= np.maximum(np.linalg.norm(vn, axis=1, keepdims=True), 1e-12)
        if np.mean(np.einsum("ij,ij->i", vn, P_ - P_.mean(0))) < 0:
            vn = -vn
        return vn

    def obj_face_colours(o):
        try:
            me_ = o.data
            img_ = None
            for m in me_.materials:
                if m and m.node_tree:
                    bs = next((n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED"), None)
                    if bs and bs.inputs["Base Color"].links:
                        nd = bs.inputs["Base Color"].links[0].from_node
                        if nd.type == "TEX_IMAGE" and nd.image:
                            img_ = nd.image; break
            if img_ is None:
                return None
            W_, H_ = img_.size
            px_ = np.empty(W_ * H_ * 4, dtype=np.float32); img_.pixels.foreach_get(px_)
            px_ = px_.reshape(H_, W_, 4)[:, :, :3]
            uv_ = np.empty(len(me_.loops) * 2, dtype=np.float32); me_.uv_layers.active.data.foreach_get("uv", uv_); uv_ = uv_.reshape(-1, 2)
            ls_ = np.empty(len(me_.polygons), dtype=np.int32); me_.polygons.foreach_get("loop_start", ls_)
            lt_ = np.empty(len(me_.polygons), dtype=np.int32); me_.polygons.foreach_get("loop_total", lt_)
            uc_ = np.zeros((len(ls_), 2))
            for q in range(3):
                uc_ += uv_[ls_ + np.minimum(q, lt_ - 1)]
            uc_ /= 3.0
            xi_ = np.clip((uc_[:, 0] % 1.0) * (W_ - 1), 0, W_ - 1).astype(np.int32)
            yi_ = np.clip((uc_[:, 1] % 1.0) * (H_ - 1), 0, H_ - 1).astype(np.int32)
            c_ = px_[yi_, xi_].astype(np.float64)
            del px_
            return c_
        except Exception as e:
            say("robe colour read failed", e)
            return None

    # target surface: her skin (wide rule) + robe faces that aren't hair
    tv = [BW]; tf = [p for p, sk in zip(Bpol, skin_wide) if sk]; tthk = [0.0] * len(tf); off_ = len(BW)
    ro = lay.get("Robe")
    if ro:
        RV = wverts(ro); RP = polys(ro)
        rc = obj_face_colours(ro)
        keep_r = np.ones(len(RP), bool)
        if rc is not None:
            mx_ = rc.max(1); mn_ = rc.min(1); sat_ = np.where(mx_ > 1e-4, (mx_ - mn_) / np.maximum(mx_, 1e-4), 0)
            keep_r = ~((mx_ < 0.22) & (sat_ < 0.45))            # hair: dark and grey
            say("shape fit: robe faces kept", int(keep_r.sum()), "of", len(RP), "(dark grey ones are hair)")
        tv.append(RV); tf += [tuple(i + off_ for i, _ in zip(p, p)) for p, k in zip(RP, keep_r) if k]
        tthk += [0.003 * HH] * int(keep_r.sum())
    TV = np.vstack(tv); TTHK = np.array(tthk)
    ttree = BVHTree.FromPolygons([tuple(p) for p in TV], tf)
    armw = Wt[:, [SI[x] for x in SEG if x.split(".")[0] in ("armU", "armL")]].sum(1)
    P0 = P.copy()
    zfrac = (P[:, 2] - z0) / ST
    region = (zfrac > 0.03) & (zfrac < 0.82) & (keepw < 0.5) & (armw < 0.5)
    wreg = np.clip(np.minimum((zfrac - 0.03) / 0.03, (0.82 - zfrac) / 0.04), 0, 1) * (1 - keepw) * np.clip(1 - 2 * armw, 0, 1)
    R_IN, R_OUT, R_TOT = 0.08 * ST, 0.08 * ST, 0.16 * ST
    for it_ in range(6):
        vn = vnormals(P)
        d = np.full(len(P), np.nan)
        for i_ in np.where(region)[0]:
            n_ = Vector(vn[i_]); o_ = Vector(P[i_] - vn[i_] * R_IN)
            gone = 0.0; best = None
            for _ in range(6):
                hit = ttree.ray_cast(o_, n_, R_IN + R_OUT - gone)
                if hit[0] is None:
                    break
                t = gone + hit[3]
                # an outward-facing surface (we cross it going outwards)
                if hit[1].dot(n_) > 0.3:
                    dd = t - R_IN - TTHK[hit[2]]
                    if best is None or abs(dd) < abs(best):
                        best = dd
                gone = t + 1e-5 * HH; o_ = hit[0] + n_ * (1e-5 * HH)
            if best is not None:
                d[i_] = best
        val = np.isfinite(d)
        for _ in range(3):                                      # drop pulls far off their neighbourhood (loose cloth, chains)
            num = np.where(val, d, 0.0); den = val.astype(float)
            for _ in range(10):
                num = navg(num); den = navg(den)
            loc = num / np.maximum(den, 1e-9)
            val &= np.abs(d - loc) < 0.025 * ST
        # never wander too far from the knot fit in total
        tot = np.einsum("ij,ij->i", (P + np.nan_to_num(d)[:, None] * vn) - P0, vn)
        val &= np.abs(tot) < R_TOT
        df = np.where(val, d, 0.0); pin = val.copy()
        for _ in range(40):
            df = navg(df); df[pin] = np.where(val, d, 0.0)[pin]
        for _ in range(4):
            df = 0.5 * df + 0.5 * navg(df)
        step = 0.8 if it_ < 5 else 1.0
        P = P + (df * wreg * step)[:, None] * vn
        P = taubin(P, 2)
        say("shape fit pass %d: %d of %d vertices found her surface, mean |pull| %.3f ST" % (
            it_ + 1, int(val.sum()), int(region.sum()), float(np.mean(np.abs(d[val]))) / ST if val.any() else 0))
    # smooth out creases cloth edges and chains leave (hip, crotch), without
    # losing the shape: a few shape-preserving passes over the fitted region
    P = taubin(P, 4, wreg)
    P = taubin(P, 2)
    # the crotch: between the hip joints, below them, the base should be a plain smooth
    # join of the two thighs, never a bump. Shrink-smooth that small region.
    hipL, hipR = TJ["legU.L"][0], TJ["legU.R"][0]
    cxh = 0.5 * (hipL[0] + hipR[0]); zh = 0.5 * (hipL[2] + hipR[2]); hwh = 0.5 * abs(hipL[0] - hipR[0])
    cm = np.clip(1 - np.abs(P[:, 0] - cxh) / (0.9 * hwh), 0, 1) * np.clip(1 - np.abs(P[:, 2] - (zh - 0.05 * ST)) / (0.08 * ST), 0, 1)
    cm = np.clip(cm * 2.0, 0, 1) * (1 - keepw)
    # lift the bottom of the pelvis between the legs to where the inner thighs meet,
    # so it reads as a smooth join, not a hanging lobe
    zc = zh - 0.055 * ST
    tw = Wt[:, [SI["hips"], SI["spine"]]].sum(1)
    mid = np.clip(1 - np.abs(P[:, 0] - cxh) / (0.6 * hwh), 0, 1) ** 0.7 * np.clip(tw * 1.5, 0, 1) * (1 - keepw)
    low = np.clip(zc - P[:, 2], 0, None)
    P[:, 2] += mid * low
    for _ in range(25):
        P = P + 0.5 * cm[:, None] * (navg(P) - P)
    say("crotch smoothing over %d vertices" % int((cm > 0.05).sum()))
    # shoulders: nothing wider than her own shoulder line (her sleeves puff out, she doesn't)
    shL, shR = TJ["armU.L"][0], TJ["armU.R"][0]
    zsh = 0.5 * (shL[2] + shR[2]); cxs = 0.5 * (shL[0] + shR[0])
    rsh = 0.060 * ST                                             # deltoid radius around the shoulder joint
    for sh_ in (shL, shR):
        d_ = P - sh_
        dist = np.linalg.norm(d_, axis=1)
        sgn_ = 1.0 if sh_[0] > cxs else -1.0
        side_ = np.clip((sgn_ * d_[:, 0] + 0.6 * rsh) / (1.2 * rsh), 0, 1) * np.clip((d_[:, 2] + 1.6 * rsh) / (1.6 * rsh), 0, 1)
        near_ = (dist < 2.2 * rsh) & (keepw < 0.5) * (side_ > 0)
        over = np.clip(dist - rsh, 0, None)
        wgt = np.clip(1 - (dist - rsh) / (1.2 * rsh), 0, 1) * near_ * side_
        P = P - (wgt * over * 0.6)[:, None] * (d_ / np.maximum(dist, 1e-9)[:, None])
    P = taubin(P, 3)
    tmp = bpy.data.meshes.new("Arcanist_BaseForm"); tmp.from_pydata([tuple(p) for p in P], [], BP); tmp.update()
    for p_ in tmp.polygons:
        p_.use_smooth = True
    # ---- measurements: the same cross-sections the fit used
    say("measurements (girth, % of her stature), her (smoothed target) vs base form:")
    for s in FIT:
        rows = [(i, TG[s][i], g[KEYS.index((s, i))]) for i in range(NS) if (s, i) in KEYS]
        if rows:
            say("  %-7s " % s + "  ".join("u%.2f %.1f/%.1f(%+.0f%%)" % (US[i], a / ST * 100, b / ST * 100, (b / a - 1) * 100) for i, a, b in rows if np.isfinite(b)))
        else:
            say("  %-7s all hidden: estimated from its neighbours" % s)
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
