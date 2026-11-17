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
nt_.links.new(texn.outputs["Color"], em.inputs["Color"])
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
