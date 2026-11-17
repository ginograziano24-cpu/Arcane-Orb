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
    skinf = np.ones(len(bodyo.data.polygons), dtype=bool)
    try:
        img = None
        for m in bodyo.data.materials:
            if m and m.node_tree:
                bs = next((n for n in m.node_tree.nodes if n.type == "BSDF_PRINCIPLED"), None)
                if bs and bs.inputs["Base Color"].links:
                    nd = bs.inputs["Base Color"].links[0].from_node
                    if nd.type == "TEX_IMAGE" and nd.image:
                        img = nd.image; break
        me = bodyo.data
        W_, H_ = img.size
        px = np.empty(W_ * H_ * 4, dtype=np.float32); img.pixels.foreach_get(px)
        px = px.reshape(H_, W_, 4)[:, :, :3]
        uv = np.empty(len(me.loops) * 2, dtype=np.float32); me.uv_layers.active.data.foreach_get("uv", uv); uv = uv.reshape(-1, 2)
        ls = np.empty(len(me.polygons), dtype=np.int32); me.polygons.foreach_get("loop_start", ls)
        xi = np.clip((uv[ls, 0] % 1.0) * (W_ - 1), 0, W_ - 1).astype(np.int32)
        yi = np.clip((uv[ls, 1] % 1.0) * (H_ - 1), 0, H_ - 1).astype(np.int32)
        col = px[yi, xi]
        r, g, b = col[:, 0], col[:, 1], col[:, 2]
        mx_, mn_ = col.max(1), col.min(1)
        sat = np.where(mx_ > 1e-4, (mx_ - mn_) / np.maximum(mx_, 1e-4), 0)
        d = np.maximum(mx_ - mn_, 1e-6)
        hue = np.where(mx_ == r, ((g - b) / d) % 6, np.where(mx_ == g, (b - r) / d + 2, (r - g) / d + 4)) * 60
        skinf = (hue < 42) & (sat > 0.1) & (sat < 0.65) & (mx_ > 0.38) & (r > b * 1.05)
        del px
        say("skin faces", int(skinf.sum()), "of", len(skinf))
    except Exception as e:
        say("skin colour test skipped", e)
    BW = wverts(bodyo)
    tri3 = np.array([p[:3] for p in polys(bodyo)], dtype=np.int64)
    SP = BW[tri3].mean(1)[skinf]                       # centres of her skin faces
    # ---- landmarks measured on her own body (the game rig is a generic
    # skeleton stretched over the hat and robe, so its joints are not used)
    import json
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    ppx = cam["ortho_scale"] / cam["res"][0]
    fz = lambda py: cam["loc"][2] + (cam["res"][1] / 2 - py) * ppx
    fx = cam["loc"][0] + (1767 - cam["res"][0] / 2) * ppx
    z_eye, z_chin = fz(694), fz(1236)
    Hhead = (z_eye - z_chin) / 0.45
    m_ = (np.abs(SP[:, 0] - fx) < 0.03 * HH) & (np.abs(SP[:, 2] - z_eye) < 0.012 * HH)
    yfront = float(np.percentile(SP[m_, 1], 3)) if m_.sum() > 10 else -0.06
    depth = 0.886 * Hhead
    head_y = yfront + 0.5 * depth
    say("head: eye z", round(z_eye / HH, 3), "chin z", round(z_chin / HH, 3), "height", round(Hhead / HH, 3), "front y", round(yfront / HH, 3))
    TJ = {}
    Hd = np.array([fx, head_y, z_chin + 0.22 * Hhead]); top = np.array([fx, head_y, z_chin + Hhead])
    Nk = np.array([fx, head_y + 0.12 * depth, Hd[2] - 0.40 * Hhead])
    TJ["head"] = (Hd, top); TJ["neck"] = (Nk, Hd)
    # legs: centre lines from slices of her visible skin
    leg = (SP[:, 2] > z0 + 0.1 * HH) & (SP[:, 2] < z0 + 0.62 * HH) & (np.abs(SP[:, 0] - fx) < 0.24 * HH)
    LPts = SP[leg]
    cxb = 0.5 * (np.percentile(LPts[:, 0], 1) + np.percentile(LPts[:, 0], 99))
    rows = {+1: [], -1: []}; gaps = []
    for zf in np.arange(0.12, 0.62, 0.02):
        zz = z0 + zf * HH
        sl = LPts[np.abs(LPts[:, 2] - zz) < 0.01 * HH]
        info = {}
        for sgn in (+1, -1):
            q = sl[(sl[:, 0] - cxb) * sgn > 0]
            if len(q) < 25:
                continue
            x0_, x1_ = np.percentile(q[:, 0], 2), np.percentile(q[:, 0], 98)
            yf_ = np.percentile(q[:, 1], 2)
            wd_ = x1_ - x0_
            info[sgn] = (x0_, x1_)
            rows[sgn].append((zz, 0.5 * (x0_ + x1_), yf_ + 0.5 * wd_, wd_))
        if +1 in info and -1 in info:
            gaps.append((zz, info[+1][0] - info[-1][1]))
        say("  slice z", round(zf, 2), {k: (round((v[0] - cxb) / HH, 3), round((v[1] - cxb) / HH, 3)) for k, v in info.items()})
    gz = [g_ for g_ in gaps if z0 + 0.3 * HH < g_[0] < z0 + 0.6 * HH]
    z_crotch = min(gz, key=lambda g_: g_[1])[0] if gz else z0 + 0.47 * HH
    say("centre x", round(cxb, 4), "crotch z", round((z_crotch - z0) / HH, 3))
    z_hip = z_crotch + 0.055 * HH; z_ank = z0 + 0.075 * HH; z_knee = z_ank + 0.5 * (z_hip - z_ank)

    def leg_line(sgn):
        R = np.array(rows[sgn])
        wmax = R[:, 3].max()
        R = R[(R[:, 3] > 0.6 * wmax) & (R[:, 0] < z_crotch + 0.005 * HH)]
        cxp = np.polyfit(R[:, 0], R[:, 1], 1); cyp = np.polyfit(R[:, 0], R[:, 2], 1)

        def at(z):
            zc = np.clip(z, R[:, 0].min(), R[:, 0].max())
            x = np.interp(zc, R[:, 0], R[:, 1]); y = np.interp(zc, R[:, 0], R[:, 2])
            if z < R[:, 0].min():                         # below the boot tops: carry the line on
                x = x + 0.5 * (np.polyval(cxp, z) - np.polyval(cxp, zc)); y = y + 0.5 * (np.polyval(cyp, z) - np.polyval(cyp, zc))
            return np.array([x, y, z])
        return at, float(np.median(R[:, 3]))
    for sd, sgn in ((".L", +1), (".R", -1)):
        at, wth = leg_line(sgn)
        hip = at(z_crotch - 0.02 * HH); hip[2] = z_hip
        hip[0] = cxb + sgn * max(abs(hip[0] - cxb) * 0.85, 0.045 * HH)
        knee = at(z_knee); ank = at(z_ank)
        toe = ank + np.array([0.0, -0.085 * HH, 0.0]); toe[2] = z0 + 0.012 * HH
        TJ["legU" + sd] = (hip, knee); TJ["legL" + sd] = (knee, ank); TJ["foot" + sd] = (ank, toe)
    # shoulders, then the arms from where her hands are
    cxt = 0.5 * (cxb + fx)
    Nk[0] = cxt
    for sd, sgn in ((".L", +1), (".R", -1)):
        TJ["armU" + sd] = [np.array([cxt + sgn * 0.078 * HH, Nk[1], Nk[2] - 0.018 * HH]), None]
    torso(TJ, TJ["legU.L"][0], TJ["legU.R"][0], TJ["armU.L"][0], TJ["armU.R"][0], Nk)
    TJ["neck"] = (Nk, Hd)
    ln = lambda d, keys: sum(np.linalg.norm(d[k][1] - d[k][0]) for k in keys)
    chain = ("hips", "spine", "chest", "legU.L", "legL.L", "legU.R", "legL.R")
    G = ln(TJ, chain) / ln(BJ, chain)
    say("overall scale", round(G, 4))
    for sd, sgn in ((".L", +1), (".R", -1)):
        sh = TJ["armU" + sd][0]
        L1 = np.linalg.norm(BJ["armU" + sd][1] - BJ["armU" + sd][0]) * G
        L2 = np.linalg.norm(BJ["armL" + sd][1] - BJ["armL" + sd][0]) * G
        Lh = np.linalg.norm(BJ["hand" + sd][1] - BJ["hand" + sd][0]) * G
        hm = (SP[:, 2] > z0 + 0.4 * HH) & ((SP[:, 0] - cxb) * sgn > 0.17 * HH)
        if hm.sum() > 30:
            hc = np.median(SP[hm], axis=0)
        else:
            hc = sh + np.array([sgn * 0.2, 0.0, -0.25]) * HH
        say("hand", sd, "skin faces", int(hm.sum()), "at", np.round((hc - np.array([cxb, 0, z0])) / HH, 3))
        u_ = (hc - sh) / max(np.linalg.norm(hc - sh), 1e-9)
        wr = hc - u_ * 0.4 * Lh
        for _ in range(2):
            dv_ = wr - sh; d_ = np.linalg.norm(dv_)
            if d_ >= 0.98 * (L1 + L2):
                el = sh + dv_ * (L1 / (L1 + L2))
            else:
                a_ = (L1 * L1 - L2 * L2 + d_ * d_) / (2 * d_)
                hgt_ = math.sqrt(max(L1 * L1 - a_ * a_, 0.0))
                bend = np.array([0.0, 0.35, -1.0]); bend = bend - dv_ * (bend @ dv_) / (d_ * d_); bend /= max(np.linalg.norm(bend), 1e-9)
                el = sh + dv_ * (a_ / d_) + bend * hgt_
            u_ = (hc - el) / max(np.linalg.norm(hc - el), 1e-9)
            wr = hc - u_ * 0.4 * Lh
        TJ["armU" + sd] = (sh, el); TJ["armL" + sd] = (el, wr); TJ["hand" + sd] = (wr, wr + u_ * Lh)
    for s in SEG:
        say("her joint", s, np.round((TJ[s][0] - np.array([cxb, 0, z0])) / HH, 3), np.round((TJ[s][1] - np.array([cxb, 0, z0])) / HH, 3))
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

    # ------------------------------------------------------------ E. pull onto her
    tmp = bpy.data.meshes.new("Arcanist_BaseForm")
    tmp.from_pydata([tuple(p) for p in Pn], [], BP)
    tmp.update()

    def normals(P):
        tmp.vertices.foreach_set("co", P.astype(np.float32).ravel())
        tmp.update()
        n = np.empty(len(P) * 3, dtype=np.float32)
        tmp.vertices.foreach_get("normal", n)
        return n.reshape(-1, 3).astype(np.float64)
    #            rin    rout   cloth-out (0 = ignore clothing)
    PAR = {"hips": (0.04, 0.08, 0.045), "spine": (0.04, 0.07, 0.045), "chest": (0.04, 0.10, 0.05), "neck": (0.02, 0.03, 0.0),
           "head": (0.03, 0.04, 0.0), "armU": (0.015, 0.035, 0.0), "armL": (0.015, 0.03, 0.0), "hand": None,
           "legU": (0.035, 0.08, 0.03), "legL": (0.025, 0.045, 0.03), "foot": (0.02, 0.03, 0.03)}
    segname = [SEG[i].split(".")[0] for i in lab]
    THICK = 0.004 * HH

    def first(tree, o, n, maxd, okidx=None):
        org = Vector(o); nv = Vector(n); done = 0.0
        for _ in range(5):
            hit, hn, idx, dist = tree.ray_cast(org, nv, maxd - done)
            if hit is None:
                return None
            done += dist
            if hn.dot(nv) > 0.2 and (okidx is None or okidx[idx]):
                return done
            org = hit + nv * 1e-5; done += 1e-5
        return None
    P = Pn.copy()
    for it in range(3):
        N = normals(P)
        sk = np.zeros(len(P)); w = np.zeros(len(P)); cnt = [0, 0]
        for i in range(len(P)):
            par = PAR[segname[i]]
            if par is None:
                continue
            rin, rout, rc = (x * HH for x in par)
            o = P[i] - N[i] * rin
            t = first(btree, o, N[i], rin + rout, skinf)
            if t is not None:
                sk[i] = t - rin; w[i] = 4.0; cnt[0] += 1
                continue
            if rc > 0:
                if N[i][1] > 0.3:
                    rc = min(rc, 0.02 * HH)          # her back: the cape hangs away from the body
                t = first(ctree, o, N[i], rin + rc)
                if t is not None:
                    sk[i] = t - rin - THICK; w[i] = 0.8; cnt[1] += 1
        s = np.zeros(len(P))
        for _ in range(300):
            s = (w * sk + navg(s)) / (w + 1.0)
        P = P + N * s[:, None]
        say("pass", it + 1, "skin hits", cnt[0], "cloth hits", cnt[1], "of", len(P), "| mean move", round(float(np.abs(s).mean()) / HH, 4))
    # keep the torso smooth and featureless
    tw = Wt[:, [SI["hips"], SI["spine"], SI["chest"]]].sum(1)[:, None]
    for _ in range(4):
        P = P + 0.5 * tw * (navg(P) - P)
    for _ in range(2):
        P = P + 0.3 * (navg(P) - P)
    tmp.vertices.foreach_set("co", P.astype(np.float32).ravel())
    tmp.update()
    for p_ in tmp.polygons:
        p_.use_smooth = True
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
    tmp.materials.append(clay)
    sub = bf.modifiers.new("Smooth", "SUBSURF"); sub.levels = 1; sub.render_levels = 2
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
    for tag, show in sets:
        for c in sc.collection.children:
            c.hide_render = c.name not in show
        for k, ang in enumerate((-90, -50, 0, 90)):
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
