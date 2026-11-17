# Shared by arcane_eyes_bake.py and export_hq.py: projects the cleaned-up face
# image (arcane_layers2/eyes_fixed.png + eyes_mask.png, drawn from the front
# orthographic camera in eyes_cam.json) onto an object's colour texture.
# Only surfaces facing that camera AND close to her face (not the cape or hair
# behind her head) take the fix. Returns the baked image (same size as base).
import bpy, os, json
import numpy as np

L2 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "arcane_layers2")


def available():
    return all(os.path.exists(os.path.join(L2, f)) for f in ("eyes_fixed.png", "eyes_mask.png", "eyes_cam.json"))


# the hair lock that hangs over her right eye, in face-image pixels
LOCK_WIN = (1330, 1700, 520, 790)        # x0, x1, y0, y1
FACE_MID_X = 1767                        # image column of the face's centre line
EYE_DROP = 8                             # her right eye sits this many pixels lower than her left
LOCK_GAP = 0.0035                        # hair counts as "in front" when this far (x height) off the face
PATCH_WIN = (1336, 1700, 430, 800)       # where the face is rebuilt (x0, x1, y0, y1)
PATCH_X0 = 1386                          # the rebuilt face piece reaches this far toward her temple


class _Geo:
    """Vertex data in the face camera's space plus ray helpers."""

    def __init__(self, ob, cam, src_from_world=None):
        from mathutils import Vector
        from mathutils.bvhtree import BVHTree
        self.Vector = Vector
        me = ob.data
        M = np.array(ob.matrix_world)
        n = len(me.vertices)
        co = np.empty(n * 3, dtype=np.float32); me.vertices.foreach_get("co", co); self.co = co.reshape(-1, 3)
        no = np.empty(n * 3, dtype=np.float32); me.vertices.foreach_get("normal", no); no = no.reshape(-1, 3)
        A = M[:3, :3].astype(np.float64); t = M[:3, 3].astype(np.float64)
        if src_from_world is not None:
            S = np.array(src_from_world, dtype=np.float64)
            t = S[:3, :3] @ t + S[:3, 3]
            A = S[:3, :3] @ A
        self.A, self.t, self.Ai = A, t, np.linalg.inv(A)
        self.wco = self.co @ A.T + t
        wno = no @ self.Ai
        self.wno = wno / np.maximum(np.linalg.norm(wno, axis=1, keepdims=True), 1e-9)
        os_ = cam["ortho_scale"]; rx, ry = cam["res"]; cx, cyy, cz = cam["loc"]; self.hgt = cam["hgt"]
        self.rx, self.ry = rx, ry
        self.px = os_ / rx
        self.u = (self.wco[:, 0] - cx) / os_ + 0.5
        self.v = (self.wco[:, 2] - cz) / (os_ * ry / rx) + 0.5
        self.pxu = self.u * rx; self.pxv = (1.0 - self.v) * ry
        win = (self.u > 0.62) & (self.u < 0.86) & (self.v > 0.38) & (self.v < 0.5)
        self.face_y = float(np.percentile(self.wco[win, 1], 3)) if win.any() else float(cyy + 0.5 * self.hgt)
        self.depth_lim = self.face_y + 0.045 * self.hgt
        self.xmid = cx + (FACE_MID_X - rx / 2) * self.px
        self.drop = EYE_DROP * self.px
        self.y_far = float(self.wco[:, 1].min()) - 0.2 * self.hgt
        self.bvh = BVHTree.FromObject(ob, bpy.context.evaluated_depsgraph_get())
        d = self.Ai @ np.array([0.0, -1.0, 0.0]); d /= np.linalg.norm(d)
        self.dv = Vector(d.tolist()); self.dback = Vector((-d).tolist())
        ext = max(float(np.ptp(self.co[:, 0])), float(np.ptp(self.co[:, 1])), float(np.ptp(self.co[:, 2]))) or 1.0
        self.eps = 4e-4 * ext
        # a surface this close in front does not count as hiding a vertex (overlapping rebuilt pieces)
        self.near = 0.003 * self.hgt / max(float(np.linalg.norm(A[:, 1])), 1e-12)

    def to_local(self, p):
        return self.Ai @ (np.asarray(p, dtype=np.float64) - self.t)

    def hits(self, x, z, n=1, facing=False):
        """Depths (camera space) of the first n surfaces met going into the face at (x, z).
        With facing=True returns (depth, faces_the_camera) pairs: the back side
        of a hair strand is a surface too, but it is not her face."""
        out = []
        o = self.Vector(self.to_local((x, self.y_far, z)).tolist())
        for _ in range(n):
            h, nrm, _i, _d = self.bvh.ray_cast(o, self.dback, 1e6)
            if h is None:
                break
            yv = float((self.A @ np.array(h) + self.t)[1])
            out.append((yv, nrm.dot(self.dv) > 0.1) if facing else yv)
            o = h + self.dback * self.eps
        return out

    def hidden(self, i):
        o = self.Vector(self.co[i].tolist()) + self.dv * max(self.eps, self.near)
        return self.bvh.ray_cast(o, self.dv, 1e6)[0] is not None

    def lockwin(self):
        return (self.pxu > LOCK_WIN[0]) & (self.pxu < LOCK_WIN[1]) & (self.pxv > LOCK_WIN[2]) & (self.pxv < LOCK_WIN[3]) & (self.wco[:, 1] < self.depth_lim)

    def mirror_fit(self, say=print):
        """How the covered (right) side of her face sits relative to the mirrored
        left side: depth_right = depth_left_mirrored + a + b*(x - xmid) + c*z."""
        idx = np.where(self.lockwin() & (self.wno[:, 1] < -0.3))[0]
        if len(idx) > 1500:
            idx = idx[:: max(1, len(idx) // 1500)]
        X = []; D = []
        for i in idx:
            x, y, z = self.wco[i]
            h = self.hits(2 * self.xmid - x, z + self.drop, 1)
            if h:
                X.append((1.0, x - self.xmid, z)); D.append(y - h[0])
        fit = np.zeros(3)
        if len(D) >= 60:
            X = np.array(X); D = np.array(D)
            keep = np.abs(D) < 0.0025 * self.hgt
            for tol in (0.0018, 0.0012):
                if keep.sum() < 40:
                    break
                fit = np.linalg.lstsq(X[keep], D[keep], rcond=None)[0]
                keep = np.abs(D - X @ fit) < tol * self.hgt
            say("mirror fit", [round(float(f), 5) for f in fit], "from", int(keep.sum()), "of", len(D), "samples")
        self.fit = fit
        return fit

    def face_depth(self, x, z):
        """Expected depth of her face at a right-side point, from the mirrored left side."""
        h = self.hits(2 * self.xmid - x, z + self.drop, 1)
        if not h:
            return None
        return h[0] + self.fit[0] + self.fit[1] * (x - self.xmid) + self.fit[2] * z


def _classify(g, say=print):
    """Paint weight and "in front" (hair lock) flag for every vertex of a _Geo."""
    n = len(g.co)
    infront = np.zeros(n, dtype=bool)
    ok = (g.wco[:, 1] < g.depth_lim).astype(np.float32) * np.clip(-3.0 * g.wno[:, 1], 0, 1)
    lockwin = g.lockwin()
    hidden = 0
    try:
        g.mirror_fit(say)
        idx = np.where(lockwin)[0]
        for i in idx:
            yf = g.face_depth(g.wco[i, 0], g.wco[i, 2])
            if yf is None:
                continue
            if g.wco[i, 1] < yf - LOCK_GAP * g.hgt:
                infront[i] = True
            elif g.wno[i, 1] > -0.05 and g.wco[i, 1] < yf - 0.0004 * g.hgt:
                infront[i] = True          # underside of the hair lock lying close against her face
        ok[infront] = 0.0
        cand = np.where((ok > 0) & (g.u > 0) & (g.u < 1) & (g.v > 0) & (g.v < 1) & ~lockwin)[0]
        for i in cand:
            if g.hidden(i):
                ok[i] = 0.0; hidden += 1
        say("eye projection: lock window verts", len(idx), "in front", int(infront.sum()), "| other candidates", len(cand), "hidden", hidden)
    except Exception as e:
        import traceback
        say("eye projection: depth tests skipped", e, traceback.format_exc())
    say("eye projection: face front y", round(g.face_y, 4), "limit", round(g.depth_lim, 4))
    return ok, infront


def _proj_data(ob, cam, src_from_world=None, say=print):
    """Per-vertex projection UVs into the face image, a 0..1 paint weight and
    an "in front" flag. The face is painted (also where the hair lock hides
    it around her right eye); the lock itself is flagged and never painted."""
    g = _Geo(ob, cam, src_from_world)
    ok, infront = _classify(g, say)
    hairw = infront.astype(np.float32) * (g.wco[:, 1] < g.depth_lim) * np.clip(-3.0 * g.wno[:, 1], 0, 1)
    return g.u, g.v, ok, infront, hairw.astype(np.float32)


EYES_PX = ((1530, 698, -1), (2004, 690, 1))     # her right / left eye centres in the face image, outer-corner direction
LOCK_EDGE_X = [1380, 1400, 1420, 1445, 1470, 1492, 1530, 1580, 1630, 1680, 1720]
LOCK_EDGE_Y = [745, 722, 700, 678, 658, 640, 628, 614, 600, 585, 574]


def flatten_lids(ob, say=print, k=0.7, iters=3):
    """Her eyelids are sculpted as thick shelves that stick out of her face.
    Pulls whatever stands proud of the locally smoothed surface around each
    eye back toward it (the eyeball and sockets, which sit behind it, stay)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    moved_tot = 0.0
    for it in range(iters):
        g = _Geo(ob, cam, None)
        y = g.wco[:, 1].copy()
        front = (g.wco[:, 1] < g.depth_lim) & (g.wno[:, 1] < -0.05)
        dy = np.zeros(len(y))
        for (ex, ey, f) in EYES_PX:
            e = ((g.pxu - (ex + f * 12)) / 205.0) ** 2 + ((g.pxv - (ey - 22)) / 135.0) ** 2
            sel = front & (e < 1.0)
            if f < 0:                                           # leave the hair lock over her right eye alone
                sel &= g.pxv > np.interp(g.pxu, LOCK_EDGE_X, LOCK_EDGE_Y) + 4
            idx = np.where(sel)[0]
            if len(idx) < 50:
                continue
            # depth of the surface, splatted to a grid and blurred
            cell = 4.0
            x0 = ex - 230; y0 = ey - 170; gw = int(460 / cell) + 1; gh = int(330 / cell) + 1
            gx = np.clip(((g.pxu[idx] - x0) / cell).astype(int), 0, gw - 1)
            gy = np.clip(((g.pxv[idx] - y0) / cell).astype(int), 0, gh - 1)
            num = np.zeros((gh, gw)); den = np.zeros((gh, gw))
            np.add.at(num, (gy, gx), y[idx]); np.add.at(den, (gy, gx), 1.0)

            def box(a, r):
                c = np.cumsum(np.pad(a, ((r + 1, r), (0, 0)), mode="edge"), axis=0); a = c[2 * r + 1:] - c[:-2 * r - 1]
                c = np.cumsum(np.pad(a, ((0, 0), (r + 1, r)), mode="edge"), axis=1); return c[:, 2 * r + 1:] - c[:, :-2 * r - 1]
            for _ in range(3):
                num = box(num, 4); den = box(den, 4)
            ys = num / np.maximum(den, 1e-9)
            proud = np.clip(ys[gy, gx] - y[idx], 0, None)          # how far each vertex stands in front of it
            w = np.clip((1.0 - e[idx]) / 0.35, 0, 1)
            w = w * w * (3 - 2 * w)
            dy[idx] = k * proud * w
        if not dy.any():
            break
        moved_tot += float(dy.max())
        newsrc = g.wco.copy(); newsrc[:, 1] += dy
        co = (newsrc - g.t) @ g.Ai.T
        me.vertices.foreach_set("co", co.astype(np.float32).ravel())
        me.update()
        bpy.context.view_layer.update()
        say("lids: pass", it + 1, "moved", int((dy > 0).sum()), "verts, most", round(float(dy.max()) / g.hgt * 100, 3), "% of her height")
    return moved_tot


def build_face_patch(ob, say=print, win=None, smooth=True, overlap=False):
    """Her model is one fused shell: where the hair lock crosses her right eye
    there is no face underneath. This copies the matching piece of the left
    side of her face, mirrors it over and adds it wherever the right side has
    no face surface of its own. win: one window or a list of windows, in
    right-side face-image pixels (x0, x1, y0, y1)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    g.mirror_fit(say)
    mid_px = FACE_MID_X
    wins = win or PATCH_WIN
    if not isinstance(wins[0], (tuple, list)):
        wins = [wins]
    inwin = np.zeros(len(g.co), dtype=bool)
    for W_ in wins:
        lx0, lx1 = 2 * mid_px - W_[1], 2 * mid_px - W_[0]
        ly0, ly1 = W_[2] - EYE_DROP, W_[3] - EYE_DROP
        inwin |= (g.pxu > lx0) & (g.pxu < lx1) & (g.pxv > ly0) & (g.pxv < ly1)
    inwin &= g.wco[:, 1] < g.depth_lim
    npoly = len(me.polygons)
    tot = np.empty(npoly, dtype=np.int32); me.polygons.foreach_get("loop_total", tot)
    st = np.empty(npoly, dtype=np.int32); me.polygons.foreach_get("loop_start", st)
    li = np.empty(len(me.loops), dtype=np.int32); me.loops.foreach_get("vertex_index", li)
    if not (tot == 3).all():
        say("patch: mesh is not all triangles, skipped"); return 0
    loops = st[:, None] + np.arange(3)[None, :]
    tri = li[loops]
    fsel = np.where(inwin[tri].all(1) & (g.wno[tri, 1].mean(1) < 0.3))[0]
    used = np.unique(tri[fsel])
    vis = {}; newpos = {}; exists = {}; delta = {}
    for i in used:
        vis[i] = not g.hidden(i)
        xl, yl, zl = g.wco[i]
        x = 2 * g.xmid - xl; z = zl - g.drop
        y = yl + g.fit[0] + g.fit[1] * (x - g.xmid) + g.fit[2] * z
        hs = [hh for hh, fc in g.hits(x, z, 6, facing=True) if fc]
        near = min(hs, key=lambda hh: abs(hh - y)) if hs else None
        exists[i] = near is not None and abs(near - y) < 0.0015 * g.hgt
        delta[i] = (near - y) if exists[i] else 0.0
        newpos[i] = [x, y, z]
    # carry the small depth mismatch at the join smoothly across the piece
    nbr = {int(i): set() for i in used}
    for f in fsel:
        a_, b_, c_ = (int(q) for q in tri[f])
        nbr[a_] |= {b_, c_}; nbr[b_] |= {a_, c_}; nbr[c_] |= {a_, b_}
    free = [int(i) for i in used if not exists[i]]
    if any(exists.values()):
        for _ in range(80):
            for i in free:
                if nbr[i]:
                    delta[i] = sum(delta[j] for j in nbr[i]) / len(nbr[i])
    for i in used:
        newpos[i][1] += delta[i] + (0.0004 if overlap else 0.00015) * g.hgt
    keepf = [f for f in fsel if any(vis[i] for i in tri[f]) and (overlap or not all(exists[i] for i in tri[f]))]
    if not keepf:
        say("patch: nothing to add"); return 0
    uvl = me.uv_layers.active
    uv = np.empty(len(me.loops) * 2, dtype=np.float32); uvl.data.foreach_get("uv", uv); uv = uv.reshape(-1, 2)
    vmap = {}; verts = []; faces = []; fuv = []
    for f in keepf:
        ids = []
        for i in tri[f]:
            if i not in vmap:
                vmap[i] = len(verts); verts.append(tuple(g.to_local(newpos[i]).tolist()))
            ids.append(vmap[i])
        faces.append((ids[0], ids[2], ids[1]))                     # mirrored, so the winding flips
        l0, l1, l2 = loops[f]
        fuv += [uv[l0], uv[l2], uv[l1]]
    if smooth:
        # smooth the rebuilt skin (the mirrored copy carries small wrinkles);
        # vertices that meet her own face stay put
        try:
            V = np.array(verts, dtype=np.float64)
            inv = {v: k for k, v in vmap.items()}
            freev = np.array([not exists[inv[n]] for n in range(len(verts))])
            nb = [set() for _ in range(len(verts))]
            for a_, b_, c_ in faces:
                nb[a_] |= {b_, c_}; nb[b_] |= {a_, c_}; nb[c_] |= {a_, b_}
            nbl = [list(x) for x in nb]
            for _ in range(10):
                avg = np.array([V[l].mean(0) if l else V[i] for i, l in enumerate(nbl)])
                V[freev] = V[freev] * 0.35 + avg[freev] * 0.65
            verts = [tuple(v) for v in V]
        except Exception as e:
            say("patch: smoothing skipped", e)
    pm = bpy.data.meshes.new("face_patch")
    pm.from_pydata(verts, [], faces)
    pm.update()
    pl = pm.uv_layers.new(name=uvl.name)
    pl.data.foreach_set("uv", np.array(fuv, dtype=np.float32).ravel())
    try:
        pm.shade_smooth()
    except Exception:
        pass
    for m in me.materials:
        pm.materials.append(m)
    try:
        _ra = pm.attributes.new("rebuilt", "FLOAT", "POINT")
        _ra.data.foreach_set("value", np.ones(len(pm.vertices), dtype=np.float32))
    except Exception:
        pass
    po = bpy.data.objects.new("face_patch", pm)
    bpy.context.scene.collection.objects.link(po)
    po.matrix_world = ob.matrix_world.copy()
    bpy.ops.object.select_all(action="DESELECT")
    po.select_set(True); ob.select_set(True); bpy.context.view_layer.objects.active = ob
    bpy.ops.object.join()
    bpy.context.view_layer.update()
    say("patch: added", len(faces), "faces of rebuilt face (from", len(fsel), "candidates)")
    return len(faces)


EYE_WINS = ((1344, 1690, 598, 872), (1344, 1400, 330, 598), (1344, 1706, 514, 600))     # her right eye and temple, rebuilt as a mirror of the left


def clear_right_eye(ob, say=print):
    """Empties her right eye and temple area completely (face and every
    leftover of the old hair lock), ready for a mirrored copy of the left."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    inw = np.zeros(len(g.co), dtype=bool)
    for (x0, x1, y0, y1) in EYE_WINS:
        inw |= (g.pxu > x0) & (g.pxu < x1) & (g.pxv > y0) & (g.pxv < y1)
    inw &= g.wco[:, 1] < g.depth_lim
    tri = _tris(me)
    if tri is None:
        say("eye: mesh not all triangles, skipped"); return 0
    sel = inw[tri].sum(1) >= 2
    k = int(sel.sum())
    if k:
        _edit_faces(ob, sel, lambda: bpy.ops.mesh.delete(type="FACE"))
    say("eye: cleared", k, "faces around her right eye and temple")
    return k


def _tris(me):
    npoly = len(me.polygons)
    tot = np.empty(npoly, dtype=np.int32); me.polygons.foreach_get("loop_total", tot)
    st = np.empty(npoly, dtype=np.int32); me.polygons.foreach_get("loop_start", st)
    li = np.empty(len(me.loops), dtype=np.int32); me.loops.foreach_get("vertex_index", li)
    if not (tot == 3).all():
        return None
    return li[st[:, None] + np.arange(3)[None, :]]


def _edit_faces(ob, sel, op):
    me = ob.data
    me.vertices.foreach_set("select", np.zeros(len(me.vertices), dtype=bool))
    me.edges.foreach_set("select", np.zeros(len(me.edges), dtype=bool))
    me.polygons.foreach_set("select", sel)
    bpy.ops.object.select_all(action="DESELECT")
    ob.select_set(True); bpy.context.view_layer.objects.active = ob
    bpy.ops.object.mode_set(mode="EDIT")
    bpy.ops.mesh.select_mode(type="FACE")
    op()
    bpy.ops.object.mode_set(mode="OBJECT")
    bpy.context.view_layer.update()


def trim_lock(ob, say=print, mask="eyes_mask.png", level=0.3):
    """Cuts away the part of the hair lock that crosses her right eye, along
    the edge of the cleaned-up eye area, and only where there is face behind
    it (her own or the rebuilt piece), so no hole can open."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    img = bpy.data.images.load(os.path.join(L2, mask), check_existing=True)
    w, h = img.size
    pix = np.empty(w * h * 4, dtype=np.float32); img.pixels.foreach_get(pix)
    m = pix.reshape(h, w, 4)[:, :, 0].astype(np.float64)
    R = 10                                                   # soften the cut line (box blur)
    cs = np.cumsum(np.pad(m, ((R + 1, R), (0, 0)), mode="edge"), axis=0); m = (cs[2 * R + 1:] - cs[:-2 * R - 1]) / (2 * R + 1)
    cs = np.cumsum(np.pad(m, ((0, 0), (R + 1, R)), mode="edge"), axis=1); m = (cs[:, 2 * R + 1:] - cs[:, :-2 * R - 1]) / (2 * R + 1)

    def maskv(g):
        mv = m[np.clip((g.v * h).astype(int), 0, h - 1), np.clip((g.u * w).astype(int), 0, w - 1)]
        mv[(g.u < 0) | (g.u > 1) | (g.v < 0) | (g.v > 1)] = 0
        return mv

    # 1) finer triangles along the cut line, so the cut edge is smooth
    for cuts, lo, hi in ((3, 0.12, 3.3), (2, 0.4, 2.2)):
        try:
            g = _Geo(ob, cam, None)
            ok, infront = _classify(g, lambda *a: None)
            tri = _tris(me)
            mv = maskv(g)
            band = infront[tri].any(1) & (mv[tri] > level * lo).any(1) & (mv[tri] < min(1.0, level * hi)).any(1)
            if band.any():
                def sub():
                    bpy.ops.mesh.subdivide(number_cuts=cuts)
                    bpy.ops.mesh.select_all(action="SELECT")
                    bpy.ops.mesh.quads_convert_to_tris()
                _edit_faces(ob, band, sub)
                say("trim: refined", int(band.sum()), "faces along the cut line; faces now", len(me.polygons))
        except Exception as e:
            import traceback
            say("trim: refine skipped", e, traceback.format_exc())
    # 2) the cut
    g = _Geo(ob, cam, None)
    ok, infront = _classify(g, say)
    tri = _tris(me)
    if tri is None:
        say("trim: mesh is not all triangles, skipped"); return 0
    mv = maskv(g)
    cand = (infront[tri].sum(1) >= 2) & (mv[tri].mean(1) > level) & (g.pxu[tri] > PATCH_X0 + 14).all(1)
    backed = {}
    def is_backed(i):
        if i not in backed:
            y = g.wco[i, 1]
            hs = g.hits(g.wco[i, 0], g.wco[i, 2], 6, facing=True)
            backed[i] = any(fc and (hh > y + 0.4 * LOCK_GAP * g.hgt) and (hh < y + 0.03 * g.hgt) for hh, fc in hs)
        return backed[i]
    sel = np.zeros(len(tri), dtype=bool)
    for f in np.where(cand)[0]:
        sel[f] = all(is_backed(int(i)) for i in tri[f])
    k = int(sel.sum())
    if not k:
        say("trim: nothing to remove"); return 0
    _edit_faces(ob, sel, lambda: bpy.ops.mesh.delete(type="FACE"))
    say("trim: removed", k, "hair faces over her right eye (", int(cand.sum()) - k, "kept because nothing is behind them)")
    # relax the cut edge of the hair so it is not saw-toothed
    try:
        bpy.ops.object.select_all(action="DESELECT")
        ob.select_set(True); bpy.context.view_layer.objects.active = ob
        bpy.ops.object.mode_set(mode="EDIT")
        bpy.ops.mesh.select_mode(type="VERT")
        bpy.ops.mesh.select_all(action="DESELECT")
        bpy.ops.mesh.select_non_manifold(extend=False, use_wire=False, use_boundary=True, use_multi_face=False, use_non_contiguous=False, use_verts=False)
        bpy.ops.object.mode_set(mode="OBJECT")
        g = _Geo(ob, cam, None)
        ok, infront = _classify(g, lambda *a: None)
        vs = np.zeros(len(me.vertices), dtype=bool); me.vertices.foreach_get("select", vs)
        vs &= infront & g.lockwin()
        me.vertices.foreach_set("select", vs)
        me.edges.foreach_set("select", np.zeros(len(me.edges), dtype=bool))
        me.polygons.foreach_set("select", np.zeros(len(me.polygons), dtype=bool))
        bpy.ops.object.mode_set(mode="EDIT")
        bpy.ops.mesh.select_more()
        bpy.ops.mesh.vertices_smooth(factor=0.5, repeat=4)
        bpy.ops.object.mode_set(mode="OBJECT")
        bpy.context.view_layer.update()
        say("trim: relaxed the cut edge around", int(vs.sum()), "edge verts")
    except Exception as e:
        import traceback
        try:
            bpy.ops.object.mode_set(mode="OBJECT")
        except Exception:
            pass
        say("trim: edge relax skipped", e, traceback.format_exc())
    return k


def attach_overlay(ob, say=print, src_from_world=None, fixed="eyes_fixed.png", mask="eyes_mask.png", coat=None):
    """Lays the cleaned-up face image over the model's colour at the image's
    own resolution (her texture has too few pixels on the face to hold
    lashes). Projection UVs + weight are stored on the mesh; every material's
    Base Color is mixed with the image. Call right after importing the model,
    before it is moved (or pass src_from_world)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    u, v, ok, _inf, hairw = _proj_data(ob, cam, src_from_world, say)
    if "eyeok" in me.color_attributes:
        me.color_attributes.remove(me.color_attributes["eyeok"])
    at = me.color_attributes.new(name="eyeok", type="FLOAT_COLOR", domain="POINT")
    at.data.foreach_set("color", np.repeat(ok, 4).astype(np.float32))
    if "eyehair" in me.color_attributes:
        me.color_attributes.remove(me.color_attributes["eyehair"])
    at2 = me.color_attributes.new(name="eyehair", type="FLOAT_COLOR", domain="POINT")
    at2.data.foreach_set("color", np.repeat(hairw, 4).astype(np.float32))
    hair_files = all(os.path.exists(os.path.join(L2, f)) for f in ("eyes_hair.png", "eyes_hairmask.png"))
    keep_active = me.uv_layers.active.name if me.uv_layers.active else None
    keep_render = next((l.name for l in me.uv_layers if l.active_render), keep_active)
    if "proj" in me.uv_layers:
        me.uv_layers.remove(me.uv_layers["proj"])
    puv = me.uv_layers.new(name="proj")
    li = np.empty(len(me.loops), dtype=np.int32); me.loops.foreach_get("vertex_index", li)
    puv.data.foreach_set("uv", np.stack([u[li], v[li]], 1).ravel().astype(np.float32))
    if keep_active:
        me.uv_layers.active = me.uv_layers[keep_active]
        me.uv_layers[keep_render].active_render = True
    imf = bpy.data.images.load(os.path.join(L2, fixed), check_existing=True)
    imm = bpy.data.images.load(os.path.join(L2, mask), check_existing=True)
    imm.colorspace_settings.name = "Non-Color"
    done = 0
    for slot in ob.material_slots:
        m = slot.material
        if not m or not m.node_tree or m.get("eye_overlay") or m.name.startswith("fringe_hair"):
            continue
        nt = m.node_tree
        bsdf = next((n for n in nt.nodes if n.type == "BSDF_PRINCIPLED"), None)
        if bsdf is None:
            continue
        bc = bsdf.inputs["Base Color"]
        uvp = nt.nodes.new("ShaderNodeUVMap"); uvp.uv_map = "proj"
        tf = nt.nodes.new("ShaderNodeTexImage"); tf.image = imf; tf.extension = "CLIP"; tf.interpolation = "Cubic"
        tm = nt.nodes.new("ShaderNodeTexImage"); tm.image = imm; tm.extension = "CLIP"
        nt.links.new(uvp.outputs[0], tf.inputs[0]); nt.links.new(uvp.outputs[0], tm.inputs[0])
        dep = nt.nodes.new("ShaderNodeAttribute"); dep.attribute_name = "eyeok"
        mu = nt.nodes.new("ShaderNodeMath"); mu.operation = "MULTIPLY"; mu.use_clamp = True
        nt.links.new(tm.outputs["Color"], mu.inputs[0]); nt.links.new(dep.outputs["Fac"], mu.inputs[1])
        mix = nt.nodes.new("ShaderNodeMix"); mix.data_type = "RGBA"
        nt.links.new(mu.outputs[0], mix.inputs["Factor"])
        if bc.links:
            nt.links.new(bc.links[0].from_socket, mix.inputs[6])
        else:
            mix.inputs[6].default_value = bc.default_value
        nt.links.new(tf.outputs["Color"], mix.inputs[7])
        last = mix.outputs[2]
        if hair_files:
            # the hair lock in front of her right eye takes the clean hair colours, never the eye
            th = nt.nodes.new("ShaderNodeTexImage"); th.image = bpy.data.images.load(os.path.join(L2, "eyes_hair.png"), check_existing=True); th.extension = "CLIP"; th.interpolation = "Cubic"
            thm = nt.nodes.new("ShaderNodeTexImage"); thm.image = bpy.data.images.load(os.path.join(L2, "eyes_hairmask.png"), check_existing=True); thm.extension = "CLIP"
            thm.image.colorspace_settings.name = "Non-Color"
            nt.links.new(uvp.outputs[0], th.inputs[0]); nt.links.new(uvp.outputs[0], thm.inputs[0])
            ha = nt.nodes.new("ShaderNodeAttribute"); ha.attribute_name = "eyehair"
            hm = nt.nodes.new("ShaderNodeMath"); hm.operation = "MULTIPLY"; hm.use_clamp = True
            nt.links.new(thm.outputs["Color"], hm.inputs[0]); nt.links.new(ha.outputs["Fac"], hm.inputs[1])
            mixh = nt.nodes.new("ShaderNodeMix"); mixh.data_type = "RGBA"
            nt.links.new(hm.outputs[0], mixh.inputs["Factor"])
            nt.links.new(last, mixh.inputs[6]); nt.links.new(th.outputs["Color"], mixh.inputs[7])
            last = mixh.outputs[2]
        nt.links.new(last, bc)
        # the painted lashes and the cleaned lids stay matte: no shine, clear
        # coat or old lash bumps from the normal map on top of them
        try:
            sep = nt.nodes.new("ShaderNodeSeparateColor"); sep.mode = "HSV"
            nt.links.new(tf.outputs["Color"], sep.inputs[0])
            mr = nt.nodes.new("ShaderNodeMapRange"); mr.clamp = True
            mr.inputs[1].default_value = 0.40; mr.inputs[2].default_value = 0.12
            mr.inputs[3].default_value = 0.0; mr.inputs[4].default_value = 1.0
            nt.links.new(sep.outputs[2], mr.inputs[0])
            dk = nt.nodes.new("ShaderNodeMath"); dk.operation = "MULTIPLY"
            nt.links.new(mr.outputs[0], dk.inputs[0]); nt.links.new(mu.outputs[0], dk.inputs[1])
            ro = bsdf.inputs["Roughness"]
            base_r = nt.nodes.new("ShaderNodeMath"); base_r.operation = "MAXIMUM"; base_r.inputs[1].default_value = 0.0
            if ro.links:
                nt.links.new(ro.links[0].from_socket, base_r.inputs[0])
            else:
                base_r.inputs[0].default_value = ro.default_value
            lift = nt.nodes.new("ShaderNodeMapRange")          # mask 0..1 -> at least 0.5 rough
            lift.inputs[3].default_value = 0.0; lift.inputs[4].default_value = 0.5
            nt.links.new(mu.outputs[0], lift.inputs[0])
            r1 = nt.nodes.new("ShaderNodeMath"); r1.operation = "MAXIMUM"
            nt.links.new(base_r.outputs[0], r1.inputs[0]); nt.links.new(lift.outputs[0], r1.inputs[1])
            r2 = nt.nodes.new("ShaderNodeMix"); r2.data_type = "FLOAT"
            nt.links.new(dk.outputs[0], r2.inputs[0]); nt.links.new(r1.outputs[0], r2.inputs[2]); r2.inputs[3].default_value = 0.9
            nt.links.new(r2.outputs[0], ro)
            inv = nt.nodes.new("ShaderNodeMath"); inv.operation = "SUBTRACT"; inv.inputs[0].default_value = 1.0
            nt.links.new(mu.outputs[0], inv.inputs[1])
            if "Coat Weight" in bsdf.inputs:
                cw = bsdf.inputs["Coat Weight"]
                cm = nt.nodes.new("ShaderNodeMath"); cm.operation = "MULTIPLY"
                cm.inputs[1].default_value = coat if coat is not None else cw.default_value
                nt.links.new(inv.outputs[0], cm.inputs[0]); nt.links.new(cm.outputs[0], cw)
            nl = bsdf.inputs["Normal"]
            if nl.links and nl.links[0].from_node.type == "NORMAL_MAP":
                nm = nl.links[0].from_node
                st = nt.nodes.new("ShaderNodeMath"); st.operation = "MULTIPLY"; st.inputs[1].default_value = nm.inputs["Strength"].default_value
                nt.links.new(inv.outputs[0], st.inputs[0]); nt.links.new(st.outputs[0], nm.inputs["Strength"])
        except Exception as e:
            say("eye overlay: matte pass skipped", e)
        m["eye_overlay"] = 1
        done += 1
    say("eye overlay attached to", done, "material(s)")
    return done


def project_bake(ob, base_img, base_uv_name, say=print, src_from_world=None, fixed="eyes_fixed.png", mask="eyes_mask.png"):
    """src_from_world: 4x4 (numpy) taking this object's world space back to the
    source GLB's space the camera was set up in (export_hq moves the model)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    sc = bpy.context.scene
    old_engine = sc.render.engine
    sc.render.engine = "CYCLES"
    try:
        sc.cycles.device = "GPU"
    except Exception:
        pass
    sc.cycles.samples = 1
    me = ob.data
    u, v, ok, _inf, _hw = _proj_data(ob, cam, src_from_world, say)
    at = me.color_attributes.new(name="eyeok", type="FLOAT_COLOR", domain="POINT")
    at.data.foreach_set("color", np.repeat(ok, 4).astype(np.float32))
    puv = me.uv_layers.new(name="proj")
    li = np.empty(len(me.loops), dtype=np.int32)
    me.loops.foreach_get("vertex_index", li)
    puv.data.foreach_set("uv", np.stack([u[li], v[li]], 1).ravel().astype(np.float32))
    me.uv_layers.active = me.uv_layers[base_uv_name]
    me.uv_layers[base_uv_name].active_render = True
    # temporary bake material on every slot
    m = bpy.data.materials.new("eyeproj_tmp")
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    out = nt.nodes.new("ShaderNodeOutputMaterial")
    uvb = nt.nodes.new("ShaderNodeUVMap"); uvb.uv_map = base_uv_name
    tb = nt.nodes.new("ShaderNodeTexImage"); tb.image = base_img
    uvp = nt.nodes.new("ShaderNodeUVMap"); uvp.uv_map = "proj"
    tf = nt.nodes.new("ShaderNodeTexImage"); tf.image = bpy.data.images.load(os.path.join(L2, fixed)); tf.extension = "CLIP"
    tm = nt.nodes.new("ShaderNodeTexImage"); tm.image = bpy.data.images.load(os.path.join(L2, mask)); tm.extension = "CLIP"
    tm.image.colorspace_settings.name = "Non-Color"
    nt.links.new(uvb.outputs[0], tb.inputs[0]); nt.links.new(uvp.outputs[0], tf.inputs[0]); nt.links.new(uvp.outputs[0], tm.inputs[0])
    geo = nt.nodes.new("ShaderNodeNewGeometry")
    sn = nt.nodes.new("ShaderNodeSeparateXYZ"); nt.links.new(geo.outputs["True Normal"], sn.inputs[0])
    fac = nt.nodes.new("ShaderNodeMath"); fac.operation = "MULTIPLY"; fac.inputs[1].default_value = -3.0
    nt.links.new(sn.outputs["Y"], fac.inputs[0])
    fcl = nt.nodes.new("ShaderNodeClamp"); nt.links.new(fac.outputs[0], fcl.inputs[0])
    dep = nt.nodes.new("ShaderNodeAttribute"); dep.attribute_name = "eyeok"
    m1 = nt.nodes.new("ShaderNodeMath"); m1.operation = "MULTIPLY"
    nt.links.new(tm.outputs["Color"], m1.inputs[0]); nt.links.new(fcl.outputs[0], m1.inputs[1])
    m2 = nt.nodes.new("ShaderNodeMath"); m2.operation = "MULTIPLY"
    nt.links.new(m1.outputs[0], m2.inputs[0]); nt.links.new(dep.outputs["Fac"], m2.inputs[1])
    mix = nt.nodes.new("ShaderNodeMix"); mix.data_type = "RGBA"
    rb = nt.nodes.new("ShaderNodeAttribute"); rb.attribute_name = "rebuilt"
    rbi = nt.nodes.new("ShaderNodeMath"); rbi.operation = "SUBTRACT"; rbi.inputs[0].default_value = 1.0; rbi.use_clamp = True
    nt.links.new(rb.outputs["Fac"], rbi.inputs[1])
    m3 = nt.nodes.new("ShaderNodeMath"); m3.operation = "MULTIPLY"
    nt.links.new(m2.outputs[0], m3.inputs[0]); nt.links.new(rbi.outputs[0], m3.inputs[1])
    nt.links.new(m3.outputs[0], mix.inputs["Factor"]); nt.links.new(tb.outputs["Color"], mix.inputs[6]); nt.links.new(tf.outputs["Color"], mix.inputs[7])
    em = nt.nodes.new("ShaderNodeEmission"); nt.links.new(mix.outputs[2], em.inputs["Color"])
    nt.links.new(em.outputs[0], out.inputs["Surface"])
    W, H = base_img.size
    tgt = bpy.data.images.new(base_img.name + "_eyes", W, H)
    tn = nt.nodes.new("ShaderNodeTexImage"); tn.image = tgt
    for n in nt.nodes:
        n.select = False
    tn.select = True
    nt.nodes.active = tn
    saved = [s.material for s in ob.material_slots]
    for s in ob.material_slots:
        s.material = m
    if not ob.material_slots:
        me.materials.append(m)
    bpy.ops.object.select_all(action="DESELECT")
    ob.select_set(True)
    bpy.context.view_layer.objects.active = ob
    bpy.ops.object.bake(type="EMIT", margin=8, use_clear=True)
    for s, mm in zip(ob.material_slots, saved):
        s.material = mm
    me.uv_layers.remove(me.uv_layers["proj"])
    me.color_attributes.remove(me.color_attributes["eyeok"])
    sc.render.engine = old_engine
    say("eye projection baked into", tgt.name)
    return tgt


def satin(materials, say=print):
    """Her materials import very glossy, which reads as wet plastic on hair and
    skin. Raise the roughness to a satin floor, drop the clear coat, soften the
    specular."""
    n = 0
    for m in materials:
        if not m or not m.node_tree:
            continue
        nt = m.node_tree
        for b in [x for x in nt.nodes if x.type == "BSDF_PRINCIPLED"]:
            ro = b.inputs["Roughness"]
            if ro.links:
                mx = nt.nodes.new("ShaderNodeMath"); mx.operation = "MAXIMUM"; mx.inputs[1].default_value = 0.62
                nt.links.new(ro.links[0].from_socket, mx.inputs[0]); nt.links.new(mx.outputs[0], ro)
            else:
                ro.default_value = max(ro.default_value, 0.62)
            if "Coat Weight" in b.inputs and not b.inputs["Coat Weight"].is_linked:
                b.inputs["Coat Weight"].default_value = 0.0
            for nm in ("Specular IOR Level", "Specular"):
                if nm in b.inputs and not b.inputs[nm].is_linked:
                    b.inputs[nm].default_value = min(b.inputs[nm].default_value, 0.35)
                    break
            n += 1
    say("satin finish on", n, "material(s)")


# ---------------------------------------------------------------------------
# New fringe: the sculpted front slab is removed and replaced by tapered
# strands, side-swept ("swept") or hanging over her right eye ("over").
FR_WIN = (1300, 1810, 196, 830)          # where the old fringe slab is removed


def _px_world(cam, px, py):
    rx, ry = cam["res"]; s_ = cam["ortho_scale"] / rx
    return cam["loc"][0] + (px - rx / 2) * s_, cam["loc"][2] + (ry / 2 - py) * s_


def remove_fringe(ob, say=print, ymax=None):
    """Deletes the old fringe slab in front of her right forehead and eye
    (only where face is behind it, so no hole opens); hat and ornaments stay."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    fr = bpy.data.images.load(os.path.join(L2, "eyes_front.png"), check_existing=True)
    w, h = fr.size
    pix = np.empty(w * h * 4, dtype=np.float32); fr.pixels.foreach_get(pix)
    img = pix.reshape(h, w, 4)[::-1, :, :3]
    px_ = np.clip(g.pxu.astype(int), 0, w - 1); py_ = np.clip(g.pxv.astype(int), 0, h - 1)
    col = img[py_, px_]
    r, gg, b = col[:, 0], col[:, 1], col[:, 2]
    mx = col.max(1); st = (mx - col.min(1)) / np.maximum(mx, 1e-3)
    other = ((b > r * 1.02) & (b > gg * 1.12) & (st > 0.28)) | ((r > b * 1.4) & (gg > b * 1.25) & (st > 0.31))
    inwin = (g.pxu > FR_WIN[0]) & (g.pxu < FR_WIN[1]) & (g.pxv > FR_WIN[2]) & (g.pxv < FR_WIN[3]) & (g.wco[:, 1] < g.depth_lim) & ~other
    if ymax is not None:
        inwin &= g.pxv < ymax
    try:
        _rb = np.zeros(len(me.vertices), dtype=np.float32)
        if "rebuilt" in me.attributes:
            me.attributes["rebuilt"].data.foreach_get("value", _rb)
        inwin &= _rb < 0.5
    except Exception:
        pass
    slab = np.zeros(len(g.co), dtype=bool)
    for i in np.where(inwin)[0]:
        y = g.wco[i, 1]
        hs = g.hits(g.wco[i, 0], g.wco[i, 2], 8, facing=True)
        slab[i] = any(fc and (hh > y + 0.0012 * g.hgt) and (hh < y + 0.045 * g.hgt) for hh, fc in hs)
    tri = _tris(me)
    if tri is None:
        say("fringe: mesh not all triangles, skipped"); return 0
    sel = slab[tri].sum(1) >= 2
    k = int(sel.sum())
    if k:
        _edit_faces(ob, sel, lambda: bpy.ops.mesh.delete(type="FACE"))
    say("fringe: removed", k, "faces of the old fringe slab")
    return k


def _crom(ctrl, per=10):
    P = np.vstack([ctrl[0] * 2 - ctrl[1], ctrl, ctrl[-1] * 2 - ctrl[-2]])
    pts = []
    for k in range(1, len(P) - 2):
        p0, p1, p2, p3 = P[k - 1], P[k], P[k + 1], P[k + 2]
        for t in np.linspace(0, 1, per, endpoint=False):
            pts.append(0.5 * ((2 * p1) + (-p0 + p2) * t + (2 * p0 - 5 * p1 + 4 * p2 - p3) * t * t + (-p0 + 3 * p1 - 3 * p2 + p3) * t ** 3))
    pts.append(ctrl[-1])
    return np.array(pts)


def _resample(pts, n):
    d = np.r_[0, np.cumsum(np.linalg.norm(np.diff(pts, axis=0), axis=1))]
    t = np.linspace(0, d[-1], n)
    return np.stack([np.interp(t, d, pts[:, 0]), np.interp(t, d, pts[:, 1])], 1)


def _bundle(E0, E1, n, rng, layer, wd, clump=5, s0=0.0, s1=1.0, N=44, tip=(0.88, 1.0)):
    """n strands filling the band between edge curves E0 (s=0) and E1 (s=1).
    Tips gather into clumps; lengths vary a little."""
    raw = []
    for i in range(n):
        s = s0 + (s1 - s0) * (i + rng.uniform(0.2, 0.8)) / n
        ctrl = E0 * (1 - s) + E1 * s
        ctrl = ctrl + rng.uniform(-1, 1, ctrl.shape) * np.array([2.2, 2.2])
        raw.append((s, _resample(_crom(ctrl), N)))
    out = []
    groups = {}
    for i, (s, p) in enumerate(raw):
        groups.setdefault(int(i // clump), []).append(p)
    t = np.linspace(0, 1, N)[:, None]
    for i, (s, p) in enumerate(raw):
        cm = np.mean(groups[int(i // clump)], axis=0)
        w = np.clip((t - 0.5) / 0.5, 0, 1) ** 1.6 * 0.8
        p = p + (cm - p) * w
        keep = int(N * rng.uniform(*tip))
        p = p[:max(keep, 8)]
        out.append({"pts": p, "wd": wd * rng.uniform(0.82, 1.18), "layer": layer})
    return out


PART_X = 1767


def _mir(E):
    E = np.array(E, float).copy(); E[:, 0] = 2 * PART_X - E[:, 0]; E[:, 1] -= 8
    return E


FH_X0, FH_X1 = 1398, 2136          # forehead surface, temple to temple
FH_Y0, FH_Y1 = 256, 522


def _fh_region(px, py):
    """Forehead region in face-image pixels (both sides; her left side is
    tested through its mirror image, which sits 8 px higher)."""
    px = np.asarray(px, float); py = np.asarray(py, float)
    left = px > PART_X
    mx = np.where(left, 2 * PART_X - px, px)
    my = np.where(left, py + EYE_DROP, py)
    r = (mx > FH_X0) & (mx <= PART_X + 0.5) & (my > FH_Y0) & (my < FH_Y1)
    r |= (mx > 1714) & (mx <= PART_X + 0.5) & (my >= FH_Y1 - 4) & (my < 622)       # nose bridge between the brows
    r &= ~((mx < 1494) & (my < 324))                     # corner under the hat brim
    return r


def rebuild_forehead(ob, say=print):
    """Replaces her whole forehead, temple to temple, with one smooth surface.
    Its shape is fitted to her clean left forehead and used for both sides, so
    there is no seam down the middle and nothing left of the old fringe on
    her right. Colour comes from the face overlay.
    Returns a function that adds the surface (call it after the eye rebuild)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    g.mirror_fit(say)
    hgt = g.hgt

    def depth(px, py):
        wx, wz = _px_world(cam, px, py)
        for hh, fc in g.hits(wx, wz, 3, facing=True):
            if fc:
                return hh if hh < g.depth_lim else None
        return None

    def basis(px, py):
        u = (np.abs(np.asarray(px, float) - PART_X) / 370.0) ** 2
        v = (np.asarray(py, float) - 430.0) / 200.0
        return np.stack([u ** i * v ** j for i in range(4) for j in range(5 - i)], -1)
    P = []; D = []
    for px in range(PART_X, 2146, 9):                    # her left forehead and the strip under it
        for py in range(250, 548 if px > 1840 else 640, 9):
            h = depth(px, py)
            if h is not None:
                P.append((px, py)); D.append(h)
    P = np.array(P, float); D = np.array(D, float)
    if len(D) < 80:
        say("forehead: too few samples", len(D)); return None
    Bm = basis(P[:, 0], P[:, 1])
    keep = np.ones(len(D), bool)
    for _ in range(5):
        c = np.linalg.lstsq(Bm[keep], D[keep], rcond=None)[0]
        r = D - Bm @ c
        mad = np.median(np.abs(r[keep])) + 1e-9
        keep = np.abs(r) < 3.0 * mad + 0.0004 * hgt
    say("forehead: fit from", int(keep.sum()), "of", len(D), "samples, residual", round(float(np.median(np.abs(r[keep]))) / hgt * 1000, 3), "permille")

    def right_shift(px, wx, wz):
        """Small left-to-right difference of her face (from the mirror fit), faded in away from the centre."""
        w = np.clip((PART_X - px) / 150.0, 0, 1)
        return w * (g.fit[0] + g.fit[1] * (wx - g.xmid) + g.fit[2] * wz)
    step = 6.0
    xs = np.arange(FH_X0 - 22, FH_X1 + 24, step); ys = np.arange(FH_Y0 - 26, 650, step)        # down past the nose bridge
    GX, GY = np.meshgrid(xs, ys)
    core = _fh_region(GX, GY)
    inside = core.copy()
    for _ in range(3):                                   # overlap the face around the cleared area (about 18 px)
        p = np.pad(inside, 1, constant_values=False)
        inside = inside | p[:-2, 1:-1] | p[2:, 1:-1] | p[1:-1, :-2] | p[1:-1, 2:]
    inside &= ~((np.where(GX > PART_X, 2 * PART_X - GX, GX) < 1494) & (np.where(GX > PART_X, GY + EYE_DROP, GY) < 318))
    DY = basis(GX, np.where(GX > PART_X, GY, GY - EYE_DROP)) @ c
    nyy, nxx = GX.shape
    WX = np.zeros_like(GX); WZ = np.zeros_like(GX)
    for j in range(nyy):
        for i in range(nxx):
            WX[j, i], WZ[j, i] = _px_world(cam, GX[j, i], GY[j, i])
    DY = DY + right_shift(GX, WX, WZ) - 0.0004 * hgt
    # near its edges the surface settles exactly onto the face around it
    dist = np.where(inside, 99, 0)
    for _ in range(7):
        p = np.pad(dist, 1, constant_values=0)
        nb = np.minimum.reduce([p[:-2, 1:-1], p[2:, 1:-1], p[1:-1, :-2], p[1:-1, 2:]]) + 1
        dist = np.where(inside, np.minimum(dist, nb), 0)
    snapped = 0
    for j in range(nyy):
        for i in range(nxx):
            if inside[j, i] and dist[j, i] <= 6:
                px, py = GX[j, i], GY[j, i]
                if px > PART_X:
                    h = depth(px, py)
                else:                                     # her right side takes the mirrored left
                    h = depth(2 * PART_X - px, py - EYE_DROP)
                    if h is not None:
                        h += right_shift(px, WX[j, i], WZ[j, i])
                if h is not None and abs(h - DY[j, i]) < 0.004 * hgt:
                    w = (1 - (dist[j, i] - 1) / 6.0) ** 2
                    DY[j, i] = DY[j, i] * (1 - w) + (h - 0.0004 * hgt) * w
                    snapped += 1
    # one forehead texel for the base texture (the overlay paints the real colour)
    uvl = me.uv_layers.active
    uvname = uvl.name if uvl else "UVMap"
    wx, wz = _px_world(cam, 1950, 400)
    hit = g.bvh.ray_cast(g.Vector(g.to_local((wx, g.y_far, wz)).tolist()), g.dback, 1e6)
    uv0 = (0.5, 0.5)
    if hit[0] is not None and uvl is not None:
        ls_ = me.polygons[hit[2]].loop_start
        uv0 = tuple(np.mean([np.array(uvl.data[ls_ + q].uv) for q in range(3)], axis=0))
    # ---- clear the old forehead
    tri = _tris(me)
    if tri is None:
        say("forehead: mesh not all triangles, skipped"); return None
    reg = _fh_region(g.pxu, g.pxv)
    fitted = basis(g.pxu, np.where(g.pxu > PART_X, g.pxv, g.pxv - EYE_DROP)) @ c
    near = np.abs(g.wco[:, 1] - fitted) < 0.0028 * hgt
    rightv = reg & ((g.pxu <= PART_X) | ((g.pxu < 1856) & (g.pxv < 522))) & (g.wco[:, 1] < g.depth_lim + 0.01 * hgt)   # right (and beside the part): skin and every leftover in front
    leftv = reg & (g.pxu > PART_X) & near & (g.wno[:, 1] < -0.1)                    # left: the skin only, hair stays
    sel = (rightv[tri].sum(1) >= 2) | leftv[tri].all(1)
    k = int(sel.sum())
    if k:
        _edit_faces(ob, sel, lambda: bpy.ops.mesh.delete(type="FACE"))
    say("forehead: cleared", k, "faces; edge nodes settled", snapped)
    to_local = g.to_local; A_ = g.A.copy()

    def add():
        idx = -np.ones(GX.shape, int); V = []
        for j in range(nyy):
            for i in range(nxx):
                if inside[j, i]:
                    idx[j, i] = len(V); V.append(tuple(to_local((WX[j, i], DY[j, i], WZ[j, i])).tolist()))
        F = []
        for j in range(nyy - 1):
            for i in range(nxx - 1):
                a_, b_, c_, d_ = idx[j, i], idx[j, i + 1], idx[j + 1, i + 1], idx[j + 1, i]
                if min(a_, b_, c_, d_) >= 0:
                    F.append((a_, d_, c_)); F.append((a_, c_, b_))
        pm = bpy.data.meshes.new("forehead_fill")
        pm.from_pydata(V, [], F)
        pm.update()
        nz = np.empty(len(pm.polygons) * 3, dtype=np.float32); pm.polygons.foreach_get("normal", nz)
        if np.mean((nz.reshape(-1, 3) @ A_.T)[:, 1]) > 0:
            pm.flip_normals() if hasattr(pm, "flip_normals") else [p.flip() for p in pm.polygons]
            pm.update()
        ul = pm.uv_layers.new(name=uvname)
        ul.data.foreach_set("uv", np.tile(np.array(uv0, np.float32), len(pm.loops)))
        pm.shade_smooth()
        for m in ob.data.materials:
            pm.materials.append(m)
        try:
            _ra = pm.attributes.new("rebuilt", "FLOAT", "POINT")
            _ra.data.foreach_set("value", np.ones(len(pm.vertices), dtype=np.float32))
        except Exception:
            pass
        po = bpy.data.objects.new("forehead_fill", pm)
        bpy.context.scene.collection.objects.link(po)
        po.matrix_world = ob.matrix_world.copy()
        bpy.ops.object.select_all(action="DESELECT")
        po.select_set(True); ob.select_set(True); bpy.context.view_layer.objects.active = ob
        bpy.ops.object.join()
        bpy.context.view_layer.update()
        say("forehead: added", len(F), "faces, temple to temple")
        return len(F)
    return add


def add_backing(ob, say=print):
    """Closes the see-through gaps beside her right temple and eye corner:
    a hair-coloured sheet just behind that area, shaped like the same area on
    her left side (mirrored)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    g.mirror_fit(lambda *a: None)
    step = 8.0
    xs = np.arange(1282, 1430, step); ys = np.arange(200, 830, step)
    GX, GY = np.meshgrid(xs, ys)
    D = np.full(GX.shape, np.nan)
    WX = np.zeros_like(GX); WZ = np.zeros_like(GX)
    for j in range(GX.shape[0]):
        for i in range(GX.shape[1]):
            WX[j, i], WZ[j, i] = _px_world(cam, GX[j, i], GY[j, i])
            lx, lz = _px_world(cam, 2 * PART_X - GX[j, i], GY[j, i] - EYE_DROP)
            for hh, fc in g.hits(lx, lz, 3, facing=True):
                if fc and hh < g.depth_lim + 0.02 * g.hgt:
                    D[j, i] = hh + g.fit[0] + g.fit[1] * (WX[j, i] - g.xmid) + g.fit[2] * WZ[j, i]
                    break
    if np.isnan(D).all():
        say("backing: no samples"); return 0
    for _ in range(40):                                   # fill the blanks from their neighbours
        bad = np.isnan(D)
        if not bad.any():
            break
        p = np.pad(D, 1, constant_values=np.nan)
        nb = np.stack([p[:-2, 1:-1], p[2:, 1:-1], p[1:-1, :-2], p[1:-1, 2:]])
        cnt = (~np.isnan(nb)).sum(0)
        avg = np.nansum(nb, 0) / np.maximum(cnt, 1)
        D = np.where(bad & (cnt > 0), avg, D)
    for _ in range(4):
        p = np.pad(D, 1, mode="edge")
        D = (p[:-2, 1:-1] + p[2:, 1:-1] + p[1:-1, :-2] + p[1:-1, 2:] + 4 * D) / 8.0
    D = D + 0.0035 * g.hgt
    V = [tuple(g.to_local((WX[j, i], D[j, i], WZ[j, i])).tolist()) for j in range(GX.shape[0]) for i in range(GX.shape[1])]
    nx = GX.shape[1]
    F = []
    for j in range(GX.shape[0] - 1):
        for i in range(nx - 1):
            a_ = j * nx + i
            F.append((a_, a_ + nx, a_ + nx + 1)); F.append((a_, a_ + nx + 1, a_ + 1))
    pm = bpy.data.meshes.new("backing")
    pm.from_pydata(V, [], F)
    pm.update()
    nz = np.empty(len(pm.polygons) * 3, dtype=np.float32); pm.polygons.foreach_get("normal", nz)
    if np.mean((nz.reshape(-1, 3) @ g.A.T)[:, 1]) > 0:
        pm.flip_normals()
        pm.update()
    uvl = me.uv_layers.active
    uv0 = (0.5, 0.5)
    wx, wz = _px_world(cam, 2250, 450)                    # dark hair on her left side
    hit = g.bvh.ray_cast(g.Vector(g.to_local((wx, g.y_far, wz)).tolist()), g.dback, 1e6)
    if hit[0] is not None and uvl is not None:
        ls_ = me.polygons[hit[2]].loop_start
        uv0 = tuple(np.mean([np.array(uvl.data[ls_ + q].uv) for q in range(3)], axis=0))
    ul = pm.uv_layers.new(name=uvl.name if uvl else "UVMap")
    ul.data.foreach_set("uv", np.tile(np.array(uv0, np.float32), len(pm.loops)))
    pm.shade_smooth()
    for m in me.materials:
        pm.materials.append(m)
    try:
        _ra = pm.attributes.new("rebuilt", "FLOAT", "POINT")
        _ra.data.foreach_set("value", np.ones(len(pm.vertices), dtype=np.float32))
    except Exception:
        pass
    po = bpy.data.objects.new("backing", pm)
    bpy.context.scene.collection.objects.link(po)
    po.matrix_world = ob.matrix_world.copy()
    bpy.ops.object.select_all(action="DESELECT")
    po.select_set(True); ob.select_set(True); bpy.context.view_layer.objects.active = ob
    bpy.ops.object.join()
    bpy.context.view_layer.update()
    say("backing: added", len(F), "faces behind her right temple")
    return len(F)


def _fringe_paths(mode):
    """Strand centre lines in face-image pixels: list of dicts {pts, wd, layer}."""
    rng = np.random.RandomState({"swept": 5, "over": 9, "middle": 13}[mode])
    A = lambda L: np.array(L, float)
    out = []
    if mode == "swept":
        E0 = A([(1882, 236), (1836, 330), (1748, 430), (1642, 484), (1528, 502), (1446, 534), (1400, 604), (1380, 700), (1372, 810)])
        E1 = A([(1690, 224), (1596, 240), (1512, 266), (1446, 304), (1400, 372), (1372, 470), (1356, 590), (1346, 700), (1342, 810)])
        out += _bundle(E0, E1, 22, rng, 0, 54, clump=4, s0=0.04, s1=0.96)
        out += _bundle(E0, E1, 36, rng, 1, 40, clump=5)
    elif mode == "over":
        E0 = A([(1882, 236), (1840, 340), (1794, 450), (1758, 560), (1734, 660), (1722, 772)])
        E1 = A([(1690, 224), (1576, 252), (1476, 316), (1412, 436), (1388, 586), (1378, 722)])
        out += _bundle(E0, E1, 20, rng, 0, 54, clump=4, s0=0.04, s1=0.96)
        out += _bundle(E0, E1, 32, rng, 1, 40, clump=6, tip=(0.66, 1.0))
    else:  # middle part, curtain bangs framing both sides; roots start up in her hair
        E0 = A([(1764, 168), (1762, 214), (1748, 256), (1706, 294), (1630, 330), (1540, 372), (1460, 440), (1404, 540), (1378, 650), (1368, 810)])
        E1 = A([(1730, 150), (1700, 186), (1650, 216), (1590, 240), (1504, 276), (1430, 340), (1386, 440), (1360, 560), (1348, 680), (1344, 810)])
        for side in (0, 1):
            e0, e1 = (E0, E1) if side == 0 else (_mir(E0), _mir(E1))
            out += _bundle(e0, e1, 12, rng, 0, 46, clump=4, s0=0.03, s1=0.97, N=52)
            out += _bundle(e0, e1, 22, rng, 1, 34, clump=5, N=52)
    return out


def add_fringe(ob, mode, say=print):
    """Builds the new fringe as tapered strands lying on her forehead, with
    their own hair texture (heroes/arcane_layers2/fringe_tex.png)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    g = _Geo(ob, cam, None)
    px = g.px; hgt = g.hgt
    V = []; F = []; UV = []
    K = 8
    paths = _fringe_paths(mode)
    paths.sort(key=lambda d: d["layer"])
    for si, d in enumerate(paths):
        pts, wd, layer = d["pts"], d["wd"], d["layer"]
        n = len(pts)
        P = []
        for (x, y) in pts:
            wx, wz = _px_world(cam, x, y)
            hs = g.hits(wx, wz, 4, facing=True)
            fy = next((hh for hh, fc in hs if fc), None)
            P.append([wx, fy if fy is not None else np.nan, wz])
        P = np.array(P)
        bad = np.isnan(P[:, 1])
        if bad.all():
            continue
        idx = np.arange(n)
        P[bad, 1] = np.interp(idx[bad], idx[~bad], P[~bad, 1])
        yy = P[:, 1].copy()
        for _ in range(16):
            yy[1:-1] = 0.25 * yy[:-2] + 0.5 * yy[1:-1] + 0.25 * yy[2:]
        P[:, 1] = np.minimum(yy, P[:, 1])
        t = idx / (n - 1)
        # root tucked into her hair, a little volume, tips resting close to the skin
        rise = np.clip(t / 0.2, 0, 1) ** 0.8
        lift = (0.0030 if layer else 0.0019) * rise * (1 - t) ** 1.4 + (0.0011 if layer else 0.0008)
        lift = lift * hgt - (1 - rise) * 0.0018 * hgt
        P[:, 1] -= lift
        width = (wd * px) * np.clip((1 - t ** 1.7) ** 0.75, 0.05, 1) * (0.12 + 0.88 * rise)
        thick = width * 0.2 + 0.0002 * hgt
        T = np.gradient(P, axis=0); T /= np.maximum(np.linalg.norm(T, axis=1, keepdims=True), 1e-12)
        N0 = np.array([0.0, -1.0, 0.0])
        base = len(V)
        u0 = (0.0 if layer else 0.5) + (si * 0.137) % 0.3
        for i in range(n):
            Nn = N0 - T[i] * T[i].dot(N0); Nn /= max(np.linalg.norm(Nn), 1e-12)
            Bn = np.cross(T[i], Nn)
            for k in range(K):
                a = 2 * np.pi * k / K
                q = P[i] + Bn * np.cos(a) * width[i] * 0.5 + Nn * np.sin(a) * thick[i] * 0.5
                V.append(tuple(g.to_local(q).tolist()))
                UV.append((u0 + 0.18 * k / K, t[i]))
        for i in range(n - 1):
            for k in range(K):
                a_ = base + i * K + k; b_ = base + i * K + (k + 1) % K
                c_ = base + (i + 1) * K + (k + 1) % K; d_ = base + (i + 1) * K + k
                F.append((a_, b_, c_)); F.append((a_, c_, d_))
    me = bpy.data.meshes.new("fringe")
    me.from_pydata(V, [], F)
    me.update()
    ul = me.uv_layers.new(name="UVMap")
    loops = np.empty(len(me.loops), dtype=np.int32); me.loops.foreach_get("vertex_index", loops)
    UVa = np.array(UV, dtype=np.float32)
    ul.data.foreach_set("uv", UVa[loops].ravel())
    me.shade_smooth()
    m = bpy.data.materials.new("fringe_hair"); m.use_nodes = True
    bs = next(x for x in m.node_tree.nodes if x.type == "BSDF_PRINCIPLED")
    tx = m.node_tree.nodes.new("ShaderNodeTexImage")
    tx.image = bpy.data.images.load(os.path.join(L2, "fringe_tex.png"), check_existing=True)
    m.node_tree.links.new(tx.outputs["Color"], bs.inputs["Base Color"])
    bs.inputs["Roughness"].default_value = 0.66
    if "Coat Weight" in bs.inputs:
        bs.inputs["Coat Weight"].default_value = 0.0
    for nm in ("Specular IOR Level", "Specular"):
        if nm in bs.inputs:
            bs.inputs[nm].default_value = 0.3
            break
    me.materials.append(m)
    fo = bpy.data.objects.new("fringe", me)
    bpy.context.scene.collection.objects.link(fo)
    fo.matrix_world = ob.matrix_world.copy()
    if ob.parent:
        fo.parent = ob.parent; fo.matrix_world = ob.matrix_world.copy()
    say("fringe:", mode, "built", len(F), "faces")
    return fo
