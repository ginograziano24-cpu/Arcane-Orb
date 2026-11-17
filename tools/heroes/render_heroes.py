# Blender (5.x) script: renders the Tripo hero models into Arcane Orb's
# portrait art. Run by Render-Heroes.cmd:
#   blender -b -P heroes/render_heroes.py
# For every heroes/glb/<id>.glb it writes, into heroes/renders/:
#   <id>_full.png        1200x1600, transparent, the whole figure (class select)
#   <id>_rest.png ...    536x680 bust crops for the HUD: rest, look, fierce, hurt
import bpy, os, math, sys
from mathutils import Vector

HERE = os.path.dirname(os.path.abspath(__file__))
GLB = os.path.join(HERE, "glb")
OUT = os.path.join(HERE, "renders")
os.makedirs(OUT, exist_ok=True)

def reset():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    sc = bpy.context.scene
    try:
        sc.render.engine = "BLENDER_EEVEE_NEXT"
    except Exception:
        sc.render.engine = "BLENDER_EEVEE"
    sc.render.film_transparent = True
    sc.render.image_settings.file_format = "PNG"
    sc.render.image_settings.color_mode = "RGBA"
    sc.view_settings.view_transform = "AgX" if "AgX" in [i.identifier for i in sc.view_settings.bl_rna.properties["view_transform"].enum_items] else "Filmic"
    try:
        sc.view_settings.look = "AgX - Medium High Contrast"
    except Exception:
        pass
    w = bpy.data.worlds.new("w")
    sc.world = w
    w.use_nodes = True
    bg = w.node_tree.nodes.get("Background")
    bg.inputs[0].default_value = (0.05, 0.035, 0.09, 1)
    bg.inputs[1].default_value = 0.6
    try:
        sc.eevee.taa_render_samples = 64
    except Exception:
        pass
    return sc

def light(name, kind, loc, energy, color, size=2.0):
    d = bpy.data.lights.new(name, kind)
    d.energy = energy
    d.color = color
    if hasattr(d, "size"):
        d.size = size
    o = bpy.data.objects.new(name, d)
    o.location = loc
    bpy.context.scene.collection.objects.link(o)
    tgt = Vector((0, 0, 1.0))
    o.rotation_euler = (tgt - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    return o

def import_hero(path):
    bpy.ops.import_scene.gltf(filepath=path)
    objs = [o for o in bpy.context.scene.objects if o.type == "MESH"]
    if os.path.basename(path) in ("arcane_eyes.glb", NEWER_ARCANE):
        # her lashes and eyes at full sharpness, laid over the texture
        try:
            sys.path.insert(0, HERE)
            import eyeproj
            if eyeproj.available():
                bpy.context.view_layer.update()
                eyeproj.attach_overlay(max(objs, key=lambda o: len(o.data.polygons)), coat=0.0)
                eyeproj.satin(bpy.data.materials)
        except Exception as e:
            print("eye overlay failed", e)
    # bounds over all meshes
    mn = Vector((1e9, 1e9, 1e9)); mx = Vector((-1e9, -1e9, -1e9))
    for o in objs:
        for c in o.bound_box:
            p = o.matrix_world @ Vector(c)
            mn = Vector(map(min, mn, p)); mx = Vector(map(max, mx, p))
    h = mx.z - mn.z
    s = 2.0 / h  # 2 units tall
    root = bpy.data.objects.new("root", None)
    bpy.context.scene.collection.objects.link(root)
    for o in bpy.context.scene.objects:
        if o.parent is None and o is not root and o.type in ("MESH", "EMPTY") :
            o.parent = root
    root.scale = (s, s, s)
    root.location = (-(mn.x + mx.x) / 2 * s, -(mn.y + mx.y) / 2 * s, -mn.z * s)
    bpy.context.view_layer.update()
    # glossier, slightly embossed look: boost specular, keep texture colours
    for m in bpy.data.materials:
        if not m.use_nodes:
            continue
        for n in m.node_tree.nodes:
            if n.type == "BSDF_PRINCIPLED":
                if "Roughness" in n.inputs and not n.inputs["Roughness"].is_linked:
                    n.inputs["Roughness"].default_value = max(n.inputs["Roughness"].default_value, 0.62)
                if "Coat Weight" in n.inputs and not n.inputs["Coat Weight"].is_linked:
                    n.inputs["Coat Weight"].default_value = 0.0
    return root

def camera(loc, look, lens):
    d = bpy.data.cameras.new("cam")
    d.lens = lens
    o = bpy.data.objects.new("cam", d)
    o.location = loc
    o.rotation_euler = (Vector(look) - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    bpy.context.scene.collection.objects.link(o)
    bpy.context.scene.camera = o
    return o

def render(path, w, h):
    sc = bpy.context.scene
    sc.render.resolution_x, sc.render.resolution_y = w, h
    sc.render.filepath = path
    bpy.ops.render.render(write_still=True)

def rig_lights(kind):
    warm = (1.0, 0.86, 0.72); rim = (0.62, 0.45, 1.0)
    k = {"rest": 1.0, "look": 1.0, "fierce": 1.25, "hurt": 0.8}[kind]
    light("key", "AREA", (-2.2, -3.2, 3.0), 520 * k, warm, 2.5)
    light("fill", "AREA", (2.8, -2.4, 1.6), 160 * k, (0.75, 0.8, 1.0), 3.0)
    light("rim", "AREA", (1.6, 2.8, 3.2), 650 * k, rim if kind != "hurt" else (1.0, 0.3, 0.3), 2.0)
    light("rim2", "AREA", (-2.0, 2.4, 2.2), 380 * k, (0.4, 0.8, 1.0) if kind != "fierce" else (1.0, 0.55, 0.25), 2.0)

# per hero: face x offset in model units (measured from the first busts:
# about 0.0015 units per pixel at the bust camera) and a step-back factor
BUST = {
    "arcane": (0.046, 1.08),
    "berserker": (0.073, 1.1),
    "engineer": (0.0, 1.0),
    "guardian": (-0.03, 1.0),
    "lucky": (0.01, 1.0),
    "pyro": (0.018, 1.0),
    "storm": (-0.06, 1.0),
}

NEWER_ARCANE = "fantasy witch 3d model.glb"
if os.path.exists(os.path.join(GLB, NEWER_ARCANE)):
    BUST["arcane"] = (0.0, 1.08)   # her face is centred on the new model


def main():
    files = sorted(f for f in os.listdir(GLB) if f.startswith("hero_") and f.endswith(".glb"))
    # a newer model the user picked wins (same rule as export_hq.py)
    for cand in ("arcane_eyes.glb", NEWER_ARCANE):
        if os.path.exists(os.path.join(GLB, cand)):
            files = [cand if f == "hero_arcane.glb" else f for f in files]
            break
    args = [a for a in sys.argv[sys.argv.index("--") + 1:]] if "--" in sys.argv else []
    busts_only = "busts" in args
    only = [a for a in args if a != "busts"]
    for fn in files:
        hid = "arcane" if fn in (NEWER_ARCANE, "arcane_eyes.glb") else fn[:-4].replace("hero_", "")
        if only and hid not in only:
            continue
        for pose in (["rest", "look", "fierce", "hurt"] if busts_only else ["full", "rest", "look", "fierce", "hurt"]):
            reset()
            root = import_hero(os.path.join(GLB, fn))
            rig_lights("rest" if pose == "full" else pose)
            if pose == "full":
                camera((0, -3.7, 1.02), (0, 0, 1.0), 50)
                render(os.path.join(OUT, hid + "_full.png"), 1200, 1600)
                continue
            turn = {"rest": 0.0, "look": 0.35, "fierce": -0.12, "hurt": 0.15}[pose]
            root.rotation_euler = (0, 0, turn)
            if pose == "hurt":
                root.rotation_euler = (0.06, 0, turn)
            near = 0.98 if pose == "fierce" else 1.12
            # centre the bust on the face, not on the whole figure's bounds
            # (staffs, shields, drones and hats pull those sideways), and
            # step back for wide hats so nothing is cropped
            fx, back = BUST.get(hid, (0.0, 1.0))
            near *= back
            fy = -0.12
            cx = fx * math.cos(turn) - fy * math.sin(turn)
            camera((cx, -near, 1.86), (cx, 0, 1.79), 50)
            render(os.path.join(OUT, hid + "_" + pose + ".png"), 536, 680)
    with open(os.path.join(OUT, "done.txt"), "w") as f:
        f.write("ok\n")

main()
