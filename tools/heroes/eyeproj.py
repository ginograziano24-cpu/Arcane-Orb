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


def _proj_data(ob, cam, src_from_world=None, say=print):
    """Per-vertex projection UVs into the face image and a 0..1 weight that is
    1 only where the surface faces the front camera, belongs to her face (not
    the cape behind) and is not hidden behind something (the hair lock)."""
    from mathutils import Vector
    from mathutils.bvhtree import BVHTree
    me = ob.data
    M = np.array(ob.matrix_world)
    n = len(me.vertices)
    co = np.empty(n * 3, dtype=np.float32); me.vertices.foreach_get("co", co); co = co.reshape(-1, 3)
    no = np.empty(n * 3, dtype=np.float32); me.vertices.foreach_get("normal", no); no = no.reshape(-1, 3)
    R = M[:3, :3]
    wco = co @ R.T + M[:3, 3]
    wno = no @ np.linalg.inv(R)          # normals: inverse-transpose
    if src_from_world is not None:
        S = np.array(src_from_world)
        wco = wco @ S[:3, :3].T + S[:3, 3]
        wno = wno @ np.linalg.inv(S[:3, :3])
        R = S[:3, :3] @ R
    wno /= np.maximum(np.linalg.norm(wno, axis=1, keepdims=True), 1e-9)
    os_ = cam["ortho_scale"]; rx, ry = cam["res"]; cx, cyy, cz = cam["loc"]; hgt = cam["hgt"]
    u = (wco[:, 0] - cx) / os_ + 0.5
    v = (wco[:, 2] - cz) / (os_ * ry / rx) + 0.5
    win = (u > 0.62) & (u < 0.86) & (v > 0.38) & (v < 0.5)
    face_y = float(np.percentile(wco[win, 1], 3)) if win.any() else float(cyy + 0.5 * hgt)
    depth_lim = face_y + 0.045 * hgt
    ok = (wco[:, 1] < depth_lim).astype(np.float32) * np.clip(-3.0 * wno[:, 1], 0, 1)
    # hidden-surface test: a ray from each candidate vertex toward the camera
    cand = np.where((ok > 0) & (u > 0) & (u < 1) & (v > 0) & (v < 1))[0]
    hidden = 0
    try:
        bvh = BVHTree.FromObject(ob, bpy.context.evaluated_depsgraph_get())
        d = np.linalg.inv(R) @ np.array([0.0, -1.0, 0.0])      # camera direction in the object's own space
        d /= np.linalg.norm(d)
        ext = float(np.ptp(co[:, 2])) or 1.0
        eps = 4e-4 * max(ext, float(np.ptp(co[:, 1])), float(np.ptp(co[:, 0])))
        dv = Vector(d.tolist())
        for i in cand:
            o = Vector(co[i].tolist()) + dv * eps
            if bvh.ray_cast(o, dv, 1e6)[0] is not None:
                ok[i] = 0.0; hidden += 1
    except Exception as e:
        say("eye projection: hidden-surface test skipped", e)
    say("eye projection: face front y", round(face_y, 4), "limit", round(depth_lim, 4), "candidates", len(cand), "hidden", hidden)
    return u, v, ok


def attach_overlay(ob, say=print, src_from_world=None, fixed="eyes_fixed.png", mask="eyes_mask.png", coat=None):
    """Lays the cleaned-up face image over the model's colour at the image's
    own resolution (her texture has too few pixels on the face to hold
    lashes). Projection UVs + weight are stored on the mesh; every material's
    Base Color is mixed with the image. Call right after importing the model,
    before it is moved (or pass src_from_world)."""
    cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
    me = ob.data
    u, v, ok = _proj_data(ob, cam, src_from_world, say)
    if "eyeok" in me.color_attributes:
        me.color_attributes.remove(me.color_attributes["eyeok"])
    at = me.color_attributes.new(name="eyeok", type="FLOAT_COLOR", domain="POINT")
    at.data.foreach_set("color", np.repeat(ok, 4).astype(np.float32))
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
        nt.links.new(mix.outputs[2], bc)
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
    u, v, ok = _proj_data(ob, cam, src_from_world, say)
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
