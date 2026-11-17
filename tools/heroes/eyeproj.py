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


def project_bake(ob, base_img, base_uv_name, say=print, src_from_world=None):
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
    M = np.array(ob.matrix_world)
    co = np.empty(len(me.vertices) * 3, dtype=np.float32)
    me.vertices.foreach_get("co", co)
    wco = co.reshape(-1, 3) @ M[:3, :3].T + M[:3, 3]
    if src_from_world is not None:
        S = np.array(src_from_world)
        wco = wco @ S[:3, :3].T + S[:3, 3]
    os_ = cam["ortho_scale"]
    rx, ry = cam["res"]
    cx, cyy, cz = cam["loc"]
    hgt = cam["hgt"]
    u = (wco[:, 0] - cx) / os_ + 0.5
    v = (wco[:, 2] - cz) / (os_ * ry / rx) + 0.5
    # the face's front: nearest skin to the camera inside the eye window
    win = (u > 0.62) & (u < 0.86) & (v > 0.38) & (v < 0.5)     # eyes and nose bridge
    face_y = float(np.percentile(wco[win, 1], 3)) if win.any() else float(cyy + 0.5 * hgt)
    depth_lim = face_y + 0.045 * hgt
    say("eye projection: face front y", round(face_y, 4), "limit", round(depth_lim, 4))
    ok = (wco[:, 1] < depth_lim).astype(np.float32)
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
    tf = nt.nodes.new("ShaderNodeTexImage"); tf.image = bpy.data.images.load(os.path.join(L2, "eyes_fixed.png")); tf.extension = "CLIP"
    tm = nt.nodes.new("ShaderNodeTexImage"); tm.image = bpy.data.images.load(os.path.join(L2, "eyes_mask.png")); tm.extension = "CLIP"
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
