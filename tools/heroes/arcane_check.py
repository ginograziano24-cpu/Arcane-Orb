# Blender 5.x: full-body check of the current Arcanist (heroes/glb/arcane_eyes.glb).
# Any surface seen from behind (a hole, a gap, a flipped piece) renders hot magenta,
# and the background seen straight through her is magenta too, so broken spots stand out.
# -> heroes/arcane_layers2/check_*.png and check_stats.txt
import bpy, os, math, json, sys
import bmesh
from mathutils import Vector
HERE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(HERE, "glb", "arcane_eyes.glb")
OUT = os.path.join(HERE, "arcane_layers2")
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng; break
    except Exception:
        pass
try:
    sc.eevee.taa_render_samples = 32
except Exception:
    pass
# transparent film: anything seen straight through her comes out clear (alpha 0) and is
# found afterwards; soft neutral light so colours stay true
sc.render.film_transparent = True
try:
    sc.view_settings.view_transform = "Standard"
except Exception:
    pass
w = bpy.data.worlds.new("w"); sc.world = w; w.use_nodes = True
w.node_tree.nodes["Background"].inputs["Color"].default_value = (0.35, 0.35, 0.37, 1)
w.node_tree.nodes["Background"].inputs["Strength"].default_value = 0.5
bpy.ops.import_scene.gltf(filepath=SRC)
obs = [o for o in sc.objects if o.type == "MESH"]
try:
    sys.path.insert(0, HERE)
    import eyeproj
    if eyeproj.available():
        bpy.context.view_layer.update()
        eyeproj.attach_overlay(max(obs, key=lambda o: len(o.data.polygons)))
        eyeproj.satin(bpy.data.materials)
except Exception as e:
    print("eye overlay failed", e)
# back faces glow magenta (except thin sheets that are meant to be seen from both sides: lashes, fringe strands)
SHEETS = ("lash", "fringe", "brow")
for m in bpy.data.materials:
    if not m.use_nodes or any(s in m.name.lower() for s in SHEETS):
        continue
    t = m.node_tree
    mo = next((n for n in t.nodes if n.type == "OUTPUT_MATERIAL" and n.is_active_output), None) or \
         next((n for n in t.nodes if n.type == "OUTPUT_MATERIAL"), None)
    if mo is None or not mo.inputs["Surface"].links:
        continue
    src = mo.inputs["Surface"].links[0].from_socket
    geo = t.nodes.new("ShaderNodeNewGeometry")
    em = t.nodes.new("ShaderNodeEmission"); em.inputs["Color"].default_value = (1, 0, 1, 1); em.inputs["Strength"].default_value = 1.0
    mx_ = t.nodes.new("ShaderNodeMixShader")
    t.links.new(geo.outputs["Backfacing"], mx_.inputs[0]); t.links.new(src, mx_.inputs[1]); t.links.new(em.outputs[0], mx_.inputs[2])
    t.links.new(mx_.outputs[0], mo.inputs["Surface"])
    m.use_backface_culling = False
    try:
        m.surface_render_method = "DITHERED"
    except Exception:
        pass
# mesh stats
lines = []
for o in obs:
    bm = bmesh.new(); bm.from_mesh(o.data)
    bound = sum(1 for e in bm.edges if e.is_boundary)
    nonman = sum(1 for e in bm.edges if not e.is_manifold and not e.is_boundary)
    tiny = sum(1 for f in bm.faces if f.calc_area() < 1e-12)
    loose = sum(1 for v in bm.verts if not v.link_edges)
    lines.append("%-28s faces %7d  open edges %6d  non-manifold %5d  zero-area %4d  loose verts %4d" % (o.name[:28], len(bm.faces), bound, nonman, tiny, loose))
    bm.free()
open(os.path.join(OUT, "check_stats.txt"), "w").write("\n".join(lines) + "\n")
mn = Vector((1e9,) * 3); mx = Vector((-1e9,) * 3)
for o in obs:
    for c in o.bound_box:
        p = o.matrix_world @ Vector(c); mn = Vector(map(min, mn, p)); mx = Vector(map(max, mx, p))
h = mx.z - mn.z; ctr = (mn + mx) / 2


def light(name, loc, energy, col, size):
    d = bpy.data.lights.new(name, "AREA"); d.energy = energy; d.color = col; d.size = size
    o = bpy.data.objects.new(name, d); o.location = loc
    o.rotation_euler = (ctr - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    sc.collection.objects.link(o)


light("key", ctr + Vector((-1.2, -2.2, 1.4)) * h, 520 * h * h, (1.0, 0.92, 0.85), 1.2 * h)
light("fill", ctr + Vector((1.6, -1.6, 0.4)) * h, 240 * h * h, (0.8, 0.82, 1.0), 1.5 * h)
light("back", ctr + Vector((0.4, 2.2, 1.2)) * h, 380 * h * h, (0.9, 0.85, 1.0), 1.4 * h)
light("under", ctr + Vector((0, -1.0, -1.6)) * h, 120 * h * h, (0.9, 0.9, 1.0), 1.5 * h)
cd = bpy.data.cameras.new("c"); cam = bpy.data.objects.new("c", cd); sc.collection.objects.link(cam); sc.camera = cam
_c = json.load(open(os.path.join(OUT, "eyes_cam.json")))
_px = _c["ortho_scale"] / _c["res"][0]
EYES = Vector((_c["loc"][0] + (1767 - _c["res"][0] / 2) * _px, -0.05, _c["loc"][2] + (_c["res"][1] / 2 - 690) * _px))
shots = []
for ang in range(-90, 270, 45):          # eight views around her, full body
    shots.append(("orbit%03d" % ((ang + 90) % 360), ang, 0.5, 1.75, 50, 0.04, (800, 1100)))
shots += [
    ("high_front", -90, 0.5, 1.75, 50, 0.9, (800, 1100)),
    ("low_front", -90, 0.45, 1.75, 50, -0.55, (800, 1100)),
    ("low_back", 90, 0.45, 1.75, 50, -0.55, (800, 1100)),
]
for ang in (-90, -45, 0, 45, 90, 135, 180, 225):   # head all the way round
    shots.append(("head%03d" % ((ang + 90) % 360), ang, None, 0.5, 85, 0.06, (900, 900)))
shots += [
    ("head_top", -90, None, 0.5, 85, 0.75, (900, 900)),
    ("head_under", -90, None, 0.45, 85, -0.5, (900, 900)),
    ("torso_front", -90, 0.66, 0.9, 70, 0.05, (900, 900)),
    ("torso_back", 90, 0.66, 0.9, 70, 0.05, (900, 900)),
    ("legs_front", -90, 0.25, 1.0, 70, 0.05, (900, 900)),
    ("legs_back", 90, 0.25, 1.0, 70, 0.05, (900, 900)),
]
for name, ang, zf, dist, lens, up, res in shots:
    a = math.radians(ang)
    tgt = Vector((ctr.x, ctr.y, mn.z + zf * h)) if zf is not None else Vector((ctr.x, ctr.y, EYES.z + 0.01 * h))
    dirv = Vector((math.cos(a), math.sin(a), up)).normalized()
    cam.location = tgt + dirv * h * dist
    cam.rotation_euler = (tgt - cam.location).to_track_quat("-Z", "Y").to_euler()
    cd.lens = lens
    sc.render.resolution_x, sc.render.resolution_y = res
    sc.render.filepath = os.path.join(OUT, "check_%s.png" % name)
    bpy.ops.render.render(write_still=True)
open(os.path.join(OUT, "check_done.txt"), "w").write("ok")
