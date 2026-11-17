# Blender 5.x: projects the cleaned-up face (heroes/arcane_layers2/eyes_fixed.png,
# edited in 2D from eyes_front.png) back onto the Arcanist's colour texture,
# only where eyes_mask.png says something changed and only on surfaces facing
# the camera. Writes heroes/glb/arcane_eyes.glb (export_hq and the bust
# renderer prefer it) and heroes/arcane_layers2/base_texture_fixed.png.
import bpy, os, json, math
import numpy as np

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
L2 = os.path.join(HERE, "arcane_layers2")
SRC = os.path.join(ROOT, "heroes", "glb", "fantasy witch 3d model.glb")
OUT = os.path.join(ROOT, "heroes", "glb", "arcane_eyes.glb")
log = open(os.path.join(L2, "eyes_bake_log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s)
    log.write(s + "\n")
    log.flush()


cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
sc.render.engine = "CYCLES"
try:
    sc.cycles.device = "GPU"
except Exception:
    pass
sc.cycles.samples = 1
bpy.ops.import_scene.gltf(filepath=SRC)
meshes = [o for o in sc.objects if o.type == "MESH"]
ob = max(meshes, key=lambda o: len(o.data.polygons))
me = ob.data
mat = me.materials[0]
nt = mat.node_tree
bsdf = next(n for n in nt.nodes if n.type == "BSDF_PRINCIPLED")
texn = next(l.from_node for l in bsdf.inputs["Base Color"].links)
img = texn.image
W, H = img.size
say("texture", img.name, W, H, "faces", len(me.polygons))

# projection UVs from the orthographic front camera (world x -> u, world z -> v)
mw = ob.matrix_world
co = np.empty(len(me.vertices) * 3, dtype=np.float32)
me.vertices.foreach_get("co", co)
co = co.reshape(-1, 3)
M = np.array(mw)
wco = co @ M[:3, :3].T + M[:3, 3]
os_ = cam["ortho_scale"]
rx, ry = cam["res"]
aspect = ry / rx
cx, _, cz = cam["loc"]
u = (wco[:, 0] - cx) / os_ + 0.5
v = (wco[:, 2] - cz) / (os_ * aspect) + 0.5
base_uv = me.uv_layers.active
puv = me.uv_layers.new(name="proj")
li = np.empty(len(me.loops), dtype=np.int32)
me.loops.foreach_get("vertex_index", li)
puv.data.foreach_set("uv", np.stack([u[li], v[li]], 1).ravel().astype(np.float32))
me.uv_layers.active = base_uv
base_uv.active_render = True

# shader: original texture, replaced by the fixed image where the mask says so (front faces only)
fixed = bpy.data.images.load(os.path.join(L2, "eyes_fixed.png"))
maskimg = bpy.data.images.load(os.path.join(L2, "eyes_mask.png"))
maskimg.colorspace_settings.name = "Non-Color"
uvn = nt.nodes.new("ShaderNodeUVMap")
uvn.uv_map = "proj"
tf = nt.nodes.new("ShaderNodeTexImage")
tf.image = fixed
tf.extension = "CLIP"
tm = nt.nodes.new("ShaderNodeTexImage")
tm.image = maskimg
tm.extension = "CLIP"
nt.links.new(uvn.outputs[0], tf.inputs[0])
nt.links.new(uvn.outputs[0], tm.inputs[0])
geo = nt.nodes.new("ShaderNodeNewGeometry")
sep = nt.nodes.new("ShaderNodeSeparateXYZ")
nt.links.new(geo.outputs["True Normal"], sep.inputs[0])
facing = nt.nodes.new("ShaderNodeMath")       # -normal.y : facing the camera
facing.operation = "MULTIPLY"
facing.inputs[1].default_value = -3.0
nt.links.new(sep.outputs["Y"], facing.inputs[0])
fclamp = nt.nodes.new("ShaderNodeClamp")
nt.links.new(facing.outputs[0], fclamp.inputs[0])
mul = nt.nodes.new("ShaderNodeMath")
mul.operation = "MULTIPLY"
nt.links.new(tm.outputs["Color"], mul.inputs[0])
nt.links.new(fclamp.outputs[0], mul.inputs[1])
mix = nt.nodes.new("ShaderNodeMix")
mix.data_type = "RGBA"
nt.links.new(mul.outputs[0], mix.inputs["Factor"])
nt.links.new(texn.outputs["Color"], mix.inputs[6])
nt.links.new(tf.outputs["Color"], mix.inputs[7])
em = nt.nodes.new("ShaderNodeEmission")
nt.links.new(mix.outputs[2], em.inputs["Color"])
outn = next(n for n in nt.nodes if n.type == "OUTPUT_MATERIAL")
nt.links.new(em.outputs[0], outn.inputs["Surface"])
# bake target
tgt = bpy.data.images.new("arcane_fixed", W, H)
tn = nt.nodes.new("ShaderNodeTexImage")
tn.image = tgt
for n in nt.nodes:
    n.select = False
tn.select = True
nt.nodes.active = tn
bpy.ops.object.select_all(action="DESELECT")
ob.select_set(True)
bpy.context.view_layer.objects.active = ob
sc.render.bake.margin = 8
sc.render.bake.use_clear = True
bpy.ops.object.bake(type="EMIT", margin=8)
say("baked")
tgt.filepath_raw = os.path.join(L2, "base_texture_fixed.png")
tgt.file_format = "PNG"
tgt.save()
# put the fixed texture into the original material and export
nt.links.new(bsdf.outputs[0], outn.inputs["Surface"])
texn.image = tgt
for n in (uvn, tf, tm, geo, sep, facing, fclamp, mul, mix, em, tn):
    nt.nodes.remove(n)
me.uv_layers.remove(me.uv_layers["proj"])
bpy.ops.object.select_all(action="DESELECT")
for o in sc.objects:
    o.select_set(True)
bpy.ops.export_scene.gltf(filepath=OUT, export_format="GLB", use_selection=True, export_image_format="JPEG", export_jpeg_quality=95)
say("wrote", OUT)
open(os.path.join(L2, "eyes_done.txt"), "w").write("ok")
