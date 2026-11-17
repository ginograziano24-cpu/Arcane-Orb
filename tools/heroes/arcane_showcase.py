# Blender 5.x: showcase renders of the current Arcanist (heroes/glb/arcane_eyes.glb)
# -> heroes/arcane_layers2/show_*.png
import bpy, os, math
from mathutils import Vector
HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
SRC = os.path.join(ROOT, "heroes", "glb", "arcane_eyes.glb")
OUT = os.path.join(HERE, "arcane_layers2")
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
try:
    sc.eevee.taa_render_samples = 64
except Exception:
    pass
try:
    sc.view_settings.view_transform = "AgX"
    sc.view_settings.look = "AgX - Base Contrast"
except Exception:
    pass
w = bpy.data.worlds.new("w"); sc.world = w; w.use_nodes = True
w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.06, 0.05, 0.1, 1)
w.node_tree.nodes["Background"].inputs["Strength"].default_value = 0.6
bpy.ops.import_scene.gltf(filepath=SRC)
obs = [o for o in sc.objects if o.type == "MESH"]
try:
    import sys
    sys.path.insert(0, HERE)
    import eyeproj
    if eyeproj.available():
        bpy.context.view_layer.update()
        eyeproj.attach_overlay(max(obs, key=lambda o: len(o.data.polygons)))
        eyeproj.satin(bpy.data.materials)
except Exception as e:
    print("eye overlay failed", e)
mn = Vector((1e9,) * 3); mx = Vector((-1e9,) * 3)
for o in obs:
    for c in o.bound_box:
        p = o.matrix_world @ Vector(c); mn = Vector(map(min, mn, p)); mx = Vector(map(max, mx, p))
h = mx.z - mn.z
ctr = (mn + mx) / 2


def light(name, kind, loc, energy, col, size):
    d = bpy.data.lights.new(name, kind); d.energy = energy; d.color = col
    if hasattr(d, "size"): d.size = size
    o = bpy.data.objects.new(name, d); o.location = loc
    o.rotation_euler = (ctr - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    sc.collection.objects.link(o)


light("key", "AREA", ctr + Vector((-1.2, -2.2, 1.4)) * h, 520 * h * h, (1.0, 0.9, 0.82), 1.2 * h)
light("fill", "AREA", ctr + Vector((1.6, -1.6, 0.4)) * h, 210 * h * h, (0.7, 0.75, 1.0), 1.5 * h)
light("rim", "AREA", ctr + Vector((0.6, 2.0, 1.2)) * h, 380 * h * h, (0.8, 0.6, 1.0), 1.0 * h)
cd = bpy.data.cameras.new("c"); cam = bpy.data.objects.new("c", cd); sc.collection.objects.link(cam); sc.camera = cam
import json
_c = json.load(open(os.path.join(OUT, "eyes_cam.json")))
_px = _c["ortho_scale"] / _c["res"][0]
EYES = Vector((_c["loc"][0] + (1767 - _c["res"][0] / 2) * _px, -0.05, _c["loc"][2] + (_c["res"][1] / 2 - 690) * _px))
shots = [
    ("full_front", -90, 0.5, 2.3, 50, (900, 1200)),
    ("full_34", -55, 0.5, 2.3, 50, (900, 1200)),
    ("face", -90, 0.855, 0.42, 85, (1200, 1000)),
    ("face_34", -62, 0.855, 0.42, 85, (1200, 1000)),
    ("eyes", -90, None, 0.26, 85, (1600, 800)),
    ("eyes_34", -66, None, 0.26, 85, (1600, 800)),
    ("eyes_r34", -114, None, 0.26, 85, (1600, 800)),
    ("side_left", -24, None, 0.30, 85, (1200, 900)),
    ("side_right", -156, None, 0.30, 85, (1200, 900)),
]
for name, ang, zf, dist, lens, res in shots:
    a = math.radians(ang)
    tgt = Vector((ctr.x, ctr.y, mn.z + zf * h)) if zf is not None else EYES
    cam.location = tgt + Vector((math.cos(a), math.sin(a), 0.03)) * h * dist
    cam.rotation_euler = (tgt - cam.location).to_track_quat("-Z", "Y").to_euler()
    cd.lens = lens
    sc.render.resolution_x, sc.render.resolution_y = res
    sc.render.filepath = os.path.join(OUT, "show_%s%s.png" % (name, os.environ.get("SHOW_TAG", "")))
    bpy.ops.render.render(write_still=True)
open(os.path.join(OUT, "show_done.txt"), "w").write("ok")
