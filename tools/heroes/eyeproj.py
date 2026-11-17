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

    def to_local(self, p):
        return self.Ai @ (np.asarray(p, dtype=np.float64) - self.t)

    def hits(self, x, z, n=1):
        """Depths (camera space) of the first n surfaces met going into the face at (x, z)."""
        out = []
        o = self.Vector(self.to_local((x, self.y_far, z)).tolist())
        for _ in range(n):
            h = self.bvh.ray_cast(o, self.dback, 1e6)[0]
            if h is None:
                break
            out.append(float((self.A @ np.array(h) + self.t)[1]))
            o = h + self.dback * self.eps
        return out

    def hidden(self, i):
        o = self.Vector(self.co[i].tolist()) + self.dv * self.eps
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
            if yf is not None and g.wco[i, 1] < yf - LOCK_GAP * g.hgt:
                infront[i] = True
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


def build_face_patch(ob, say=print):
    """Her model is one fused shell: where the hair lock crosses her right eye
    there is no face underneath. This copies the matching piece of the left
    side of her face, mirrors it over and adds it wherever the right side has
    no face surface of its own."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    g = _Geo(ob, cam, None)
    g.mirror_fit(say)
    mid_px = FACE_MID_X
    lx0, lx1 = 2 * mid_px - LOCK_WIN[1], 2 * mid_px - PATCH_X0
    ly0, ly1 = LOCK_WIN[2] - EYE_DROP, LOCK_WIN[3] - EYE_DROP
    inwin = (g.pxu > lx0) & (g.pxu < lx1) & (g.pxv > ly0) & (g.pxv < ly1) & (g.wco[:, 1] < g.depth_lim)
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
        hs = g.hits(x, z, 6)
        near = min(hs, key=lambda hh: abs(hh - y)) if hs else None
        exists[i] = near is not None and abs(near - y) < 0.003 * g.hgt
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
        newpos[i][1] += delta[i] + 0.00015 * g.hgt
    keepf = [f for f in fsel if any(vis[i] for i in tri[f]) and not all(exists[i] for i in tri[f])]
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
    po = bpy.data.objects.new("face_patch", pm)
    bpy.context.scene.collection.objects.link(po)
    po.matrix_world = ob.matrix_world.copy()
    bpy.ops.object.select_all(action="DESELECT")
    po.select_set(True); ob.select_set(True); bpy.context.view_layer.objects.active = ob
    bpy.ops.object.join()
    bpy.context.view_layer.update()
    say("patch: added", len(faces), "faces of rebuilt face under the hair lock (from", len(fsel), "candidates)")
    return len(faces)


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
    for cuts, lo, hi in ((3, 0.3, 2.4), (2, 0.6, 1.6)):
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
            hs = g.hits(g.wco[i, 0], g.wco[i, 2], 6)
            backed[i] = any((hh > y + 0.4 * LOCK_GAP * g.hgt) and (hh < y + 0.03 * g.hgt) for hh in hs)
        return backed[i]
    sel = np.zeros(len(tri), dtype=bool)
    for f in np.where(cand)[0]:
        sel[f] = all(is_backed(int(i)) for i in tri[f])
    k = int(sel.sum())
    if not k:
        say("trim: nothing to remove"); return 0
    _edit_faces(ob, sel, lambda: bpy.ops.mesh.delete(type="FACE"))
    say("trim: removed", k, "hair faces over her right eye (", int(cand.sum()) - k, "kept because nothing is behind them)")
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
        if not m or not m.node_tree or m.get("eye_overlay"):
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
    nt.links.new(m2.outputs[0], mix.inputs["Factor"]); nt.links.new(tb.outputs["Color"], mix.inputs[6]); nt.links.new(tf.outputs["Color"], mix.inputs[7])
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
