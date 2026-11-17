# Blender 5.x: projects the cleaned-up face (heroes/arcane_layers2/eyes_fixed.png)
# onto the Arcanist's source colour texture (see eyeproj.py) and writes
# heroes/glb/arcane_eyes.glb, which export_hq.py and the bust renderer prefer.
# export_hq.py projects the same fix again onto the game model's own texture,
# where her face has more pixels.
import bpy, os, sys
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import eyeproj
ROOT = os.path.dirname(HERE)
SRC = os.path.join(ROOT, "heroes", "glb", "fantasy witch 3d model.glb")
OUT = os.path.join(ROOT, "heroes", "glb", "arcane_eyes.glb")
log = open(os.path.join(eyeproj.L2, "eyes_bake_log.txt"), "w")
def say(*a):
    s = " ".join(str(x) for x in a); print(s); log.write(s + "\n"); log.flush()
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
bpy.ops.import_scene.gltf(filepath=SRC)
ob = max((o for o in sc.objects if o.type == "MESH"), key=lambda o: len(o.data.polygons))
mat = ob.data.materials[0]
bsdf = next(n for n in mat.node_tree.nodes if n.type == "BSDF_PRINCIPLED")
texn = next(l.from_node for l in bsdf.inputs["Base Color"].links)
say("texture", texn.image.name, texn.image.size[0], "faces", len(ob.data.polygons))
# her right eye: rebuild the face the hair lock hides, then keep or trim the lock
MODE = os.environ.get("LOCK_MODE", "").strip().lower()
_mf = os.path.join(eyeproj.L2, "lock_mode.txt")
if not MODE and os.path.exists(_mf):
    MODE = open(_mf).read().strip().lower()
MODE = MODE or "keep"
say("lock mode", MODE)
try:
    bpy.context.view_layer.update()
    eyeproj.flatten_lids(ob, say)
    try:                                   # shading must follow the reshaped lids, not the old stored normals
        bpy.ops.object.select_all(action="DESELECT")
        ob.select_set(True); bpy.context.view_layer.objects.active = ob
        bpy.ops.mesh.customdata_custom_splitnormals_clear()
        bpy.ops.object.shade_smooth()
    except Exception as e:
        say("normals reset skipped", e)
    FRINGE = os.environ.get("FRINGE", "").strip().lower()
    if FRINGE in ("swept", "over", "middle"):
        eyeproj.remove_fringe(ob, say)
        bpy.context.view_layer.update()
        add_forehead = eyeproj.rebuild_forehead(ob, say)        # fits and clears; the surface goes on below
        bpy.context.view_layer.update()
        eyeproj.clear_right_eye(ob, say)
        bpy.context.view_layer.update()
        eyeproj.build_face_patch(ob, say, win=[(1330, 1716, 584, 900), (1330, 1412, 318, 600), (1330, 1710, 500, 606)], smooth=False, overlap=True)
        bpy.context.view_layer.update()
        if add_forehead:
            add_forehead()
        eyeproj.remove_fringe(ob, say, ymax=590)        # leftovers that only now have face behind them
        bpy.context.view_layer.update()
        eyeproj.add_backing(ob, say)
        bpy.context.view_layer.update()
        eyeproj.remove_fringe(ob, say, ymax=340)        # old pieces under the hat brim, now in front of the backing
        bpy.context.view_layer.update()
        eyeproj.add_fringe(ob, FRINGE, say)
    else:
        eyeproj.build_face_patch(ob, say)
        if MODE == "trim":
            eyeproj.trim_lock(ob, say)
    say("faces now", len(ob.data.polygons))
except Exception as e:
    import traceback
    say("face rebuild failed", e, traceback.format_exc())
fixed = eyeproj.project_bake(ob, texn.image, ob.data.uv_layers.active.name, say)
fixed.filepath_raw = os.path.join(eyeproj.L2, "base_texture_fixed.png")
fixed.file_format = "PNG"
fixed.save()
texn.image = fixed
bpy.ops.object.select_all(action="DESELECT")
for o in sc.objects:
    o.select_set(True)
bpy.ops.export_scene.gltf(filepath=OUT, export_format="GLB", use_selection=True, export_image_format="JPEG", export_jpeg_quality=95)
say("wrote", OUT)
# close-up check render of her face with the fixed texture (flat colour)
import json, math
cam = json.load(open(os.path.join(eyeproj.L2, "eyes_cam.json")))
mat_ = ob.data.materials[0]
nt_ = mat_.node_tree
outn = next(n for n in nt_.nodes if n.type == "OUTPUT_MATERIAL")
em = nt_.nodes.new("ShaderNodeEmission")
try:
    eyeproj.attach_overlay(ob, say)
except Exception as e:
    say("overlay for check render failed", e)
_bc = bsdf.inputs["Base Color"]
nt_.links.new(_bc.links[0].from_socket if _bc.links else texn.outputs["Color"], em.inputs["Color"])
nt_.links.new(em.outputs[0], outn.inputs["Surface"])
cd = bpy.data.cameras.new("chk"); cd.type = "ORTHO"; cd.ortho_scale = cam["ortho_scale"]
co_ = bpy.data.objects.new("chk", cd); sc.collection.objects.link(co_); sc.camera = co_
co_.location = cam["loc"]; co_.rotation_euler = (math.pi / 2, 0, 0)
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
try:
    sc.view_settings.view_transform = "Standard"
except Exception:
    pass
sc.render.resolution_x, sc.render.resolution_y = cam["res"]
for ang, tag in ((0.0, "front"), (0.5, "turn")):
    co_.location = cam["loc"]
    if ang:
        r_ = 0.5 * cam["hgt"]
        cx, cy, cz = cam["loc"]
        co_.location = (cx + math.sin(ang) * r_, cy + r_ - math.cos(ang) * r_, cz)
        co_.rotation_euler = (math.pi / 2, 0, ang)
    sc.render.filepath = os.path.join(eyeproj.L2, "check_%s.png" % tag)
    bpy.ops.render.render(write_still=True)
open(os.path.join(eyeproj.L2, "eyes_done.txt"), "w").write("ok")
