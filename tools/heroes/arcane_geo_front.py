# Blender 5.x: surface-direction picture of the Arcanist's face from the same
# front camera as eyes_front.png (heroes/arcane_layers2/geo_front.png), so the
# lash line can be fitted to the sculpted eyelids of each eye.
import bpy, os, json, math
HERE = os.path.dirname(os.path.abspath(__file__))
L2 = os.path.join(HERE, "arcane_layers2")
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
bpy.ops.import_scene.gltf(filepath=os.path.join(HERE, "glb", "fantasy witch 3d model.glb"))
cam = json.load(open(os.path.join(L2, "eyes_cam.json")))
m = bpy.data.materials.new("geo"); m.use_nodes = True
nt = m.node_tree; nt.nodes.clear()
out = nt.nodes.new("ShaderNodeOutputMaterial")
geo = nt.nodes.new("ShaderNodeNewGeometry")
ma = nt.nodes.new("ShaderNodeVectorMath"); ma.operation = "MULTIPLY_ADD"
ma.inputs[1].default_value = (0.5, 0.5, 0.5); ma.inputs[2].default_value = (0.5, 0.5, 0.5)
nt.links.new(geo.outputs["Normal"], ma.inputs[0])
em = nt.nodes.new("ShaderNodeEmission"); nt.links.new(ma.outputs[0], em.inputs["Color"])
nt.links.new(em.outputs[0], out.inputs["Surface"])
for o in sc.objects:
    if o.type == "MESH":
        for p in o.data.polygons:
            p.use_smooth = True
        o.data.materials.clear(); o.data.materials.append(m)
cd = bpy.data.cameras.new("c"); cd.type = "ORTHO"; cd.ortho_scale = cam["ortho_scale"]
co = bpy.data.objects.new("c", cd); sc.collection.objects.link(co); sc.camera = co
co.location = cam["loc"]; co.rotation_euler = (math.pi / 2, 0, 0)
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
sc.view_settings.view_transform = "Raw"
sc.render.resolution_x, sc.render.resolution_y = cam["res"]
sc.render.filepath = os.path.join(L2, "geo_front.png")
bpy.ops.render.render(write_still=True)
open(os.path.join(L2, "geo_done.txt"), "w").write("ok")
