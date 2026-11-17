# Blender 5.x: lists what is in the Human Base Meshes bundle and renders the
# female bodies, so the fit to the Arcanist can be planned.
import bpy, os, math
from mathutils import Vector
HERE = os.path.dirname(os.path.abspath(__file__))
BM = os.path.join(HERE, "basemesh")
log = open(os.path.join(BM, "inspect.txt"), "w")
bpy.ops.wm.open_mainfile(filepath=os.path.join(BM, "human_base_meshes_bundle.blend"))
sc = bpy.context.scene
for o in bpy.data.objects:
    bb = [o.matrix_world @ Vector(c) for c in o.bound_box] if o.type == "MESH" else []
    dims = tuple(round(v, 3) for v in o.dimensions)
    mnz = round(min(p.z for p in bb), 3) if bb else None
    log.write("%s | %s | verts %s | dims %s | minz %s | loc %s | mods %s | asset %s | cols %s\n" % (
        o.name, o.type, len(o.data.vertices) if o.type == "MESH" else "", dims, mnz,
        tuple(round(v, 2) for v in o.matrix_world.translation), [m.type for m in o.modifiers],
        bool(o.asset_data), [c.name for c in o.users_collection]))
log.write("collections: %s\n" % [c.name for c in bpy.data.collections])
log.flush()
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
w = bpy.data.worlds.new("w"); sc.world = w; w.use_nodes = True
w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.6, 0.6, 0.65, 1)
d = bpy.data.lights.new("s", "SUN"); d.energy = 3
lo = bpy.data.objects.new("s", d); lo.rotation_euler = (0.9, 0.1, 0.4); sc.collection.objects.link(lo)
cd = bpy.data.cameras.new("c"); cam = bpy.data.objects.new("c", cd); sc.collection.objects.link(cam); sc.camera = cam
sc.render.resolution_x, sc.render.resolution_y = 700, 900
targets = [o for o in bpy.data.objects if o.type == "MESH" and "female" in o.name.lower() and "body" in o.name.lower()]
for t in targets:
    for o in bpy.data.objects:
        if o.type == "MESH":
            o.hide_render = o is not t
    for c in bpy.data.collections:
        c.hide_render = False
    bb = [t.matrix_world @ Vector(c) for c in t.bound_box]
    mn = Vector((min(p.x for p in bb), min(p.y for p in bb), min(p.z for p in bb)))
    mx = Vector((max(p.x for p in bb), max(p.y for p in bb), max(p.z for p in bb)))
    ctr = (mn + mx) / 2; h = max(mx.z - mn.z, mx.x - mn.x)
    for k, ang in enumerate((-90, 0)):
        a = math.radians(ang)
        cam.location = ctr + Vector((math.cos(a), math.sin(a), 0.05)) * h * 2.3
        cam.rotation_euler = (ctr - cam.location).to_track_quat("-Z", "Y").to_euler()
        sc.render.filepath = os.path.join(BM, "inspect_%s_%d.png" % (t.name.replace(" ", "_"), k))
        bpy.ops.render.render(write_still=True)
    log.write("rendered %s\n" % t.name)
log.write("done\n")
log.close()
