# Blender: renders Fumbleton (Tripo rigged model) from four sides in his rest
# pose, to settle which way he faces. Output: heroes/intro_chars/fumb_check_*.png
import bpy, os, glob, math
from mathutils import Vector
HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
CH = os.path.join(ROOT, "heroes", "intro_chars")
p = sorted(glob.glob(os.path.join(CH, "fumbleton_tripo", "rigged", "**", "*.glb"), recursive=True))[0]
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
sc.render.resolution_x = sc.render.resolution_y = 640
bpy.ops.import_scene.gltf(filepath=p)
log = open(os.path.join(CH, "fumb_check.txt"), "w")
for o in sc.objects:
    log.write("%s %s %s rot=%s loc=%s\n" % (o.name, o.type, len(o.data.polygons) if o.type == "MESH" else "", tuple(round(v, 2) for v in o.rotation_euler), tuple(round(v, 2) for v in o.location)))
    if o.type == "MESH" and len(o.data.polygons) < 200:
        o.hide_render = True
    if o.type == "ARMATURE":
        for b in ("mixamorig:LeftArm", "mixamorig:RightArm", "mixamorig:Head", "mixamorig:Hips"):
            bb = o.data.bones.get(b)
            if bb:
                log.write("  %s head=%s\n" % (b, tuple(round(v, 3) for v in (o.matrix_world @ bb.head_local))))
log.close()
w = bpy.data.worlds.new("w"); sc.world = w; w.use_nodes = True
w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.6, 0.6, 0.65, 1)
w.node_tree.nodes["Background"].inputs["Strength"].default_value = 1.2
d = bpy.data.lights.new("sun", "SUN"); d.energy = 3
lo = bpy.data.objects.new("sun", d); lo.rotation_euler = (0.8, 0.2, 0.3); sc.collection.objects.link(lo)
cam_d = bpy.data.cameras.new("c"); cam = bpy.data.objects.new("c", cam_d); sc.collection.objects.link(cam); sc.camera = cam
bpy.context.view_layer.update()
mn = Vector((1e9,) * 3); mx = Vector((-1e9,) * 3)
for o in sc.objects:
    if o.type == "MESH" and not o.hide_render:
        for c in o.bound_box:
            q = o.matrix_world @ Vector(c); mn = Vector(map(min, mn, q)); mx = Vector(map(max, mx, q))
ctr = (mn + mx) / 2; h = mx.z - mn.z
for name, ang in (("from_minusY", -90), ("from_plusX", 0), ("from_plusY", 90), ("from_minusX", 180)):
    a = math.radians(ang)
    cam.location = ctr + Vector((math.cos(a), math.sin(a), 0.15)) * h * 2.2
    cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
    sc.render.filepath = os.path.join(CH, "fumb_check_%s.png" % name)
    bpy.ops.render.render(write_still=True)
