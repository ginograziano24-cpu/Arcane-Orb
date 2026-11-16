# Blender 5.x: builds and renders Arcane Orb's story opening (docs/opening.md).
#   blender -b -P intro/build_story.py -- [test]
# Timing comes from intro/intro_cues.json (the voiced lines, made by
# Make-Intro-Voices.cmd + the mix step). "test" renders a handful of stills
# into intro/test/; otherwise frames go to intro/frames/f%04d.jpg
# (1280x720, 24 fps) which the game plays with intro/voice.wav.
#
# Beats (seconds follow the voice cues; ~84.6 s):
#   S1  tower at night
#   S2  Fumbleton's room: date night, the laundry
#   S3  the Instant Tidy scroll; the three guardians appear around the tower
#   S4  the guardians tidy the meadow: Prism outlines, Crucible compresses,
#       Warden stamps bricks; Fumbleton's "undo" makes more
#   S5  brick walls overtake the land
#   S6  Mirabel arrives, makes one orb and splits it into seven
#   S7  she sends the seven lights out
#   S8  an orb shows how it's done: crunch
#   S9  seven catches, seven cards
#   S10 the seven together
#   S11 title, then Fumbleton's last word at the dinner table
import bpy, os, sys, math, random, json, glob, zipfile
from mathutils import Vector, Quaternion, Matrix

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
GLB = os.path.join(ROOT, "heroes", "glb")
RIG = os.path.join(ROOT, "heroes", "rigged")
CHARS = os.path.join(ROOT, "heroes", "intro_chars")
SKY = os.path.join(HERE, "sky.jpg")
FPS = 24
ARGS = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
TEST = "test" in ARGS
LOG = open(os.path.join(HERE, "story-log.txt"), "w")


def say(*a):
    s = " ".join(str(x) for x in a)
    print(s)
    LOG.write(s + "\n")
    LOG.flush()


CUES = json.load(open(os.path.join(HERE, "intro_cues.json")))
C = {c["id"]: c for c in CUES["cues"]}
DUR = CUES["length"]


def T(cid):
    return C[cid]["start"]


def f(t):
    return int(round(t * FPS)) + 1


def smooth(x):
    x = max(0.0, min(1.0, x))
    return x * x * (3 - 2 * x)


def ramp(t, a, b):
    return smooth((t - a) / max(1e-6, b - a))


rng = random.Random(11)

# ------------------------------------------------------------------ timeline
S2 = 2.4                      # into the room
S3 = T("m01") - 0.2           # merchant
S3b = T("f04") - 0.15         # outside: the guardians around the tower
S4 = T("g01") - 0.1           # tidying the meadow
S4b = T("f05") - 0.1          # Fumbleton on the balcony
S5 = T("f06") - 0.15          # walls rise
S6 = T("r01") - 0.5           # Mirabel arrives
S6b = T("f07") - 0.05         # Fumbleton: "funny story"
S6c = T("r02") - 0.1          # back to Mirabel, one orb -> seven
S7 = T("r03") - 0.2           # send them out
S8 = T("r04") - 0.4           # the demonstration
HEROES = ["guardian", "berserker", "arcane", "engineer", "lucky", "pyro", "storm"]
NAMES = {"guardian": ("GUARDIAN", "Parry"), "berserker": ("BERSERKER", "Rampage"),
         "arcane": ("ARCANIST", "Attunement"), "engineer": ("ENGINEER", "Overclock"),
         "lucky": ("GAMBLER", "High Roller"), "pyro": ("PYROMANCER", "Stoke"),
         "storm": ("STORMCALLER", "Static Charge")}
HCUE = dict(zip(HEROES, ["h01", "h02", "h03", "h04", "h05", "h06", "h07"]))
CATCH_T = {h: T(HCUE[h]) - 0.12 for h in HEROES}
SHOT0 = {h: CATCH_T[h] - 1.0 for h in HEROES}
S9 = SHOT0["guardian"]
S10 = T("h08") - 0.35
S11 = T("r05") - 0.35         # title
S12 = T("f08") - 0.25         # dinner table
GLOW = {"guardian": (1.0, 0.85, 0.4), "berserker": (1.0, 0.25, 0.15), "arcane": (0.6, 0.45, 1.0),
        "engineer": (0.3, 0.8, 1.0), "lucky": (0.4, 1.0, 0.5), "pyro": (1.0, 0.5, 0.1), "storm": (0.5, 0.75, 1.0)}
say("timeline", dict(S2=S2, S3=S3, S3b=S3b, S4=S4, S4b=S4b, S5=S5, S6=S6, S6b=S6b, S6c=S6c, S7=S7, S8=S8, S9=S9, S10=S10, S11=S11, S12=S12, DUR=DUR))

# ------------------------------------------------------------------ scene
bpy.ops.wm.read_factory_settings(use_empty=True)
sc = bpy.context.scene
for eng in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE"):
    try:
        sc.render.engine = eng
        break
    except Exception:
        pass
sc.render.resolution_x, sc.render.resolution_y = 1280, 720
sc.render.fps = FPS
sc.frame_start, sc.frame_end = 1, f(DUR)
sc.render.image_settings.file_format = "JPEG"
sc.render.image_settings.quality = 90
for k, v in (("taa_render_samples", 32), ("use_bloom", True), ("use_gtao", True), ("use_shadows", True)):
    try:
        setattr(sc.eevee, k, v)
    except Exception:
        pass
try:
    sc.view_settings.view_transform = "AgX"
    sc.view_settings.look = "AgX - Punchy"
except Exception:
    pass


def mat(name, col, emit=0.0, rough=0.5, metal=0.0, alpha=None):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    b = m.node_tree.nodes["Principled BSDF"]
    b.inputs["Base Color"].default_value = (*col, 1)
    b.inputs["Roughness"].default_value = rough
    b.inputs["Metallic"].default_value = metal
    if emit > 0:
        b.inputs["Emission Color"].default_value = (*col, 1)
        b.inputs["Emission Strength"].default_value = emit
    if alpha is not None:
        b.inputs["Alpha"].default_value = alpha
        try:
            m.surface_render_method = "BLENDED"
        except Exception:
            pass
    return m


def emit_mat(name, col, strength):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    o = nt.nodes.new("ShaderNodeOutputMaterial")
    e = nt.nodes.new("ShaderNodeEmission")
    e.inputs["Color"].default_value = (*col, 1)
    e.inputs["Strength"].default_value = strength
    nt.links.new(e.outputs[0], o.inputs[0])
    return m


def light(name, kind, loc, energy, col, size=3.0, aim=(0, 0, 0), dist=None):
    d = bpy.data.lights.new(name, kind)
    d.energy = energy
    d.color = col
    if hasattr(d, "size"):
        d.size = size
    if hasattr(d, "shadow_soft_size") and kind == "POINT":
        d.shadow_soft_size = size
    if dist:
        try:
            d.use_custom_distance = True
            d.cutoff_distance = dist
        except Exception:
            pass
    o = bpy.data.objects.new(name, d)
    o.location = loc
    o.rotation_euler = (Vector(aim) - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    sc.collection.objects.link(o)
    return o


def objs_of(o):
    return [o] + list(o.children_recursive)


def vis(o, t0, t1=None):
    """o (and its children) render only between t0 and t1 seconds."""
    for ob in objs_of(o):
        ob.hide_render = True
        ob.keyframe_insert("hide_render", frame=1)
        ob.hide_render = False
        ob.keyframe_insert("hide_render", frame=max(1, f(t0)))
        if t1 is not None:
            ob.hide_render = True
            ob.keyframe_insert("hide_render", frame=f(t1))


def key_loc(o, t, loc):
    o.location = loc
    o.keyframe_insert("location", frame=f(t))


def key_scale(o, t, s):
    o.scale = (s, s, s) if isinstance(s, (int, float)) else s
    o.keyframe_insert("scale", frame=f(t))


def sphere(r, loc, m, seg=24):
    bpy.ops.mesh.primitive_uv_sphere_add(radius=r, segments=seg, ring_count=seg // 2, location=loc)
    o = bpy.context.object
    o.data.materials.append(m)
    bpy.ops.object.shade_smooth()
    return o


def cube(loc, scale, m, bevel=0.0):
    bpy.ops.mesh.primitive_cube_add(size=1, location=loc)
    o = bpy.context.object
    o.scale = scale
    if bevel:
        b = o.modifiers.new("b", "BEVEL")
        b.width = bevel
        b.segments = 2
    o.data.materials.append(m)
    return o


def text(body, size, m, extrude=0.02):
    cu = bpy.data.curves.new("txt", "FONT")
    cu.body = body
    cu.size = size
    cu.extrude = extrude
    cu.align_x = "CENTER"
    cu.align_y = "CENTER"
    o = bpy.data.objects.new("txt", cu)
    sc.collection.objects.link(o)
    o.data.materials.append(m)
    return o


# world: night violet with stars
w = bpy.data.worlds.new("w")
sc.world = w
w.use_nodes = True
nt = w.node_tree
bg = nt.nodes["Background"]
bg.inputs["Strength"].default_value = 1.0
tc = nt.nodes.new("ShaderNodeTexCoord")
nz = nt.nodes.new("ShaderNodeTexNoise")
nz.inputs["Scale"].default_value = 260
nz.inputs["Detail"].default_value = 0
cr = nt.nodes.new("ShaderNodeValToRGB")
cr.color_ramp.elements[0].position = 0.72
cr.color_ramp.elements[0].color = (0.03, 0.015, 0.07, 1)
cr.color_ramp.elements[1].position = 0.76
cr.color_ramp.elements[1].color = (0.9, 0.85, 1.0, 1)
nt.links.new(tc.outputs["Generated"], nz.inputs["Vector"])
nt.links.new(nz.outputs["Fac"], cr.inputs["Fac"])
nt.links.new(cr.outputs["Color"], bg.inputs["Color"])

if os.path.exists(SKY):
    bpy.ops.mesh.primitive_plane_add(size=1, location=(0, 190, 95), rotation=(math.pi / 2, 0, 0))
    bd = bpy.context.object
    bd.scale = (620, 360, 1)
    m = bpy.data.materials.new("sky")
    m.use_nodes = True
    n2 = m.node_tree
    n2.nodes.clear()
    out = n2.nodes.new("ShaderNodeOutputMaterial")
    em = n2.nodes.new("ShaderNodeEmission")
    tx = n2.nodes.new("ShaderNodeTexImage")
    tx.image = bpy.data.images.load(SKY)
    n2.links.new(tx.outputs["Color"], em.inputs["Color"])
    em.inputs["Strength"].default_value = 0.9
    n2.links.new(em.outputs["Emission"], out.inputs["Surface"])
    bd.data.materials.append(m)

light("moon", "SUN", (-8, -6, 10), 3.2, (0.75, 0.68, 1.0))
light("rimA", "AREA", (6, 8, 6), 2600, (1.0, 0.45, 0.9), 8, aim=(0, 0, 1))
light("rimB", "AREA", (-7, 7, 5), 2200, (0.3, 0.7, 1.0), 8, aim=(0, 0, 1))
moon_m = emit_mat("moonm", (1.0, 0.95, 0.85), 6)
sphere(9, (-60, 170, 85), moon_m, 32)

# ------------------------------------------------------------------ ground
bpy.ops.mesh.primitive_grid_add(x_subdivisions=200, y_subdivisions=200, size=420, location=(0, 60, 0))
ground = bpy.context.object


def hill(x, y):  # rolling hills, flat where the heroes stand
    d = math.hypot(x, y * 0.7)
    k = max(0.0, min(1.0, (d - 9) / 25))
    h = 1.6 * math.sin(x * 0.11 + 1.3) * math.cos(y * 0.07) + 0.9 * math.sin(x * 0.27 + y * 0.19) + 0.35 * math.sin(x * 0.9) * math.sin(y * 0.8)
    return k * k * (h + 1.2) * (1.0 + min(d, 60) * 0.02)


for v in ground.data.vertices:
    v.co.z = hill(v.co.x, v.co.y + 60)
bpy.ops.object.shade_smooth()
gm = bpy.data.materials.new("ground")
gm.use_nodes = True
gn = gm.node_tree.nodes
gl = gm.node_tree.links
gb = gn["Principled BSDF"]
gb.inputs["Roughness"].default_value = 1.0
nz = gn.new("ShaderNodeTexNoise")
nz.inputs["Scale"].default_value = 0.6
nz.inputs["Detail"].default_value = 8
rp = gn.new("ShaderNodeValToRGB")
rp.color_ramp.elements[0].color = (0.006, 0.026, 0.020, 1)
rp.color_ramp.elements[1].color = (0.035, 0.120, 0.050, 1)
gl.new(nz.outputs["Fac"], rp.inputs["Fac"])
gl.new(rp.outputs["Color"], gb.inputs["Base Color"])
bump = gn.new("ShaderNodeBump")
bump.inputs["Strength"].default_value = 0.35
nz2 = gn.new("ShaderNodeTexNoise")
nz2.inputs["Scale"].default_value = 40
gl.new(nz2.outputs["Fac"], bump.inputs["Height"])
gl.new(bump.outputs["Normal"], gb.inputs["Normal"])
ground.data.materials.append(gm)
flower = mat("flower", (1.0, 0.3, 0.8), emit=6)
flower2 = mat("flower2", (1.0, 0.85, 0.3), emit=6)
bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=1, radius=0.09)
fl_proto = bpy.context.object
fl_proto.data.materials.append(flower)
for i in range(600):
    x, y = rng.uniform(-45, 45), rng.uniform(-12, 90)
    if abs(x) < 2.2 and y < 12:
        continue
    o = fl_proto.copy()
    o.data = fl_proto.data if i % 2 else fl_proto.data.copy()
    if i % 2 == 0:
        o.data.materials[0] = flower2
    o.location = (x, y, hill(x, y) + 0.09)
    sc.collection.objects.link(o)
fl_proto.location = (0, -500, -50)

# ------------------------------------------------------------------ tower
TW = Vector((-17.0, 42.0, 0))
TW.z = hill(TW.x, TW.y) - 0.3
stone = mat("stone", (0.32, 0.30, 0.42), rough=0.9)
roofm = mat("roof", (0.28, 0.18, 0.62), rough=0.6)
warm = emit_mat("warmwin", (1.0, 0.72, 0.35), 9)
gold = mat("gold", (1.0, 0.78, 0.25), emit=2.0, metal=0.8, rough=0.3)
bpy.ops.mesh.primitive_cylinder_add(vertices=24, radius=2.2, depth=13, location=TW + Vector((0, 0, 6.5)))
tower = bpy.context.object
tower.data.materials.append(stone)
bpy.ops.mesh.primitive_cylinder_add(vertices=24, radius=3.0, depth=0.35, location=TW + Vector((0, 0, 12.2)))
bpy.context.object.data.materials.append(stone)                       # balcony
bpy.ops.mesh.primitive_cone_add(vertices=24, radius1=2.9, depth=6.5, location=TW + Vector((0, 0, 16.5)))
roof = bpy.context.object
roof.rotation_euler = (0.12, 0, 0)
roof.data.materials.append(roofm)
st = text("★", 1.4, gold)
st.location = TW + Vector((0, -2.35, 16.0))
st.rotation_euler = (math.pi / 2, 0, 0)
for k in range(9):
    a = k / 9 * 2 * math.pi + 0.3
    z = 3 + (k % 3) * 3.3
    win = cube(TW + Vector((math.cos(a) * 2.18, math.sin(a) * 2.18, z)), (0.45, 0.45, 0.8), warm)
    win.rotation_euler = (0, 0, a)
light("towerglow", "POINT", TW + Vector((0, -3.5, 9)), 1600, (1.0, 0.7, 0.4), 2, dist=40)
BALC = TW + Vector((0.0, -2.35, 12.38))

# ------------------------------------------------------------------ room (a set far away)
R0 = Vector((300, 0, 0))
wood = mat("wood", (0.30, 0.17, 0.09), rough=0.7)
wall = mat("wall", (0.36, 0.26, 0.42), rough=0.95)
cube(R0 + Vector((0, 0, -0.05)), (10, 9, 0.1), wood)
cube(R0 + Vector((0, 3.2, 2.5)), (10, 0.2, 5), wall)
cube(R0 + Vector((-4.0, 0, 2.5)), (0.2, 9, 5), wall)
cube(R0 + Vector((4.0, 0, 2.5)), (0.2, 9, 5), wall)
cube(R0 + Vector((0, 0, 5.0)), (10, 9, 0.2), wall)
nightwin = emit_mat("nightwin", (0.15, 0.2, 0.55), 2.5)
cube(R0 + Vector((1.6, 3.08, 2.6)), (1.5, 0.05, 1.8), nightwin)
for dx, dz, sx, sz in ((0, 0.92, 1.7, 0.12), (0, -0.92, 1.7, 0.12), (0.8, 0, 0.12, 1.9), (-0.8, 0, 0.12, 1.9), (0, 0, 0.06, 1.8), (0, 0, 1.5, 0.06)):
    cube(R0 + Vector((1.6 + dx, 3.0, 2.6 + dz)), (sx, 0.1, sz), wood)
sphere(0.25, R0 + Vector((1.95, 3.02, 3.0)), moon_m, 16)
# table, candles, roses
cube(R0 + Vector((-1.4, -0.3, 0.75)), (1.6, 0.9, 0.08), wood)
for lx in (-2.1, -0.7):
    for ly in (-0.65, 0.05):
        cube(R0 + Vector((lx, ly, 0.37)), (0.08, 0.08, 0.74), wood)
cloth = mat("cloth", (0.75, 0.1, 0.2), rough=0.8)
cube(R0 + Vector((-1.4, -0.3, 0.8)), (1.65, 0.6, 0.01), cloth)
wax = mat("wax", (0.95, 0.92, 0.85), rough=0.6)
flame = emit_mat("flame", (1.0, 0.65, 0.2), 30)
for cx in (-1.9, -0.9):
    bpy.ops.mesh.primitive_cylinder_add(vertices=12, radius=0.05, depth=0.32, location=R0 + Vector((cx, -0.3, 0.97)))
    bpy.context.object.data.materials.append(wax)
    fl = sphere(0.035, R0 + Vector((cx, -0.3, 1.17)), flame, 8)
    fl.scale = (1, 1, 1.8)
    light("candle", "POINT", R0 + Vector((cx, -0.3, 1.25)), 40, (1.0, 0.62, 0.3), 0.05, dist=12)
rose = mat("rose", (0.9, 0.05, 0.15), rough=0.5)
stem = mat("stem", (0.1, 0.4, 0.1))
bpy.ops.mesh.primitive_cylinder_add(vertices=12, radius=0.09, depth=0.25, location=R0 + Vector((-1.4, -0.3, 0.92)))
bpy.context.object.data.materials.append(mat("vase", (0.5, 0.7, 0.95), rough=0.15))
for k in range(3):
    sphere(0.07, R0 + Vector((-1.4 + (k - 1) * 0.07, -0.3, 1.18 + (k % 2) * 0.05)), rose, 12)
light("roomkey", "AREA", R0 + Vector((-1.5, -3.5, 3.5)), 260, (1.0, 0.78, 0.6), 3, aim=R0 + Vector((0, 0, 1)), dist=20)
light("roomfill", "POINT", R0 + Vector((2.5, 1.0, 3.0)), 120, (0.6, 0.5, 1.0), 1, dist=15)
# the mess: laundry heaps, socks, books, scrolls
cols = [(0.8, 0.3, 0.3), (0.3, 0.5, 0.85), (0.9, 0.8, 0.3), (0.5, 0.8, 0.5), (0.85, 0.5, 0.85), (0.95, 0.95, 0.9)]
cms = [mat("laundry%d" % i, c, rough=0.95) for i, c in enumerate(cols)]
mess = []
for k in range(26):
    x, y = rng.uniform(-3.6, 3.6), rng.uniform(-2.6, 2.9)
    if abs(x + 1.4) < 1.0 and abs(y + 0.3) < 0.6:
        continue
    if abs(x) < 0.6 and abs(y - 0.5) < 0.6:
        continue
    kind = k % 3
    if kind == 0:
        bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=2, radius=rng.uniform(0.25, 0.5), location=R0 + Vector((x, y, 0.1)))
        o = bpy.context.object
        o.scale = (1.2, 1.0, 0.45)
        d = o.modifiers.new("d", "DISPLACE")
        d.strength = 0.12
        d.texture = bpy.data.textures.new("lump", "CLOUDS")
        bpy.ops.object.shade_smooth()
    elif kind == 1:
        o = cube(R0 + Vector((x, y, 0.06)), (0.32, 0.24, 0.1 + rng.random() * 0.25), cms[rng.randrange(6)], 0.015)
        o.rotation_euler.z = rng.uniform(-1, 1)
    else:
        bpy.ops.mesh.primitive_torus_add(major_radius=0.12, minor_radius=0.035, location=R0 + Vector((x, y, 0.04)))
        o = bpy.context.object
        o.rotation_euler = (math.pi / 2, 0, rng.uniform(0, 3))
        o.scale = (1, 2.2, 1)
    if not o.data.materials:
        o.data.materials.append(cms[rng.randrange(6)])
    mess.append(o)
shelf = cube(R0 + Vector((-3.5, 1.5, 1.5)), (0.6, 2.4, 3.0), wood)
for k in range(30):
    bk = cube(R0 + Vector((-3.25, 0.5 + (k % 10) * 0.2, 0.5 + (k // 10) * 1.0)), (0.3, 0.15, 0.6 + rng.random() * 0.2), cms[k % 6])
    bk.rotation_euler.x = rng.uniform(-0.15, 0.15)

# the Instant Tidy scroll
parch = mat("parch", (0.95, 0.86, 0.62), rough=0.8)
scroll = cube(R0 + Vector((1.6, 3.6, 2.6)), (0.6, 0.02, 0.8), parch)
scroll_t = text("INSTANT\nTIDY", 0.13, mat("ink", (0.5, 0.1, 0.6), emit=2), 0.0)
scroll_t.parent = scroll
scroll_t.location = (0, -0.8, 0.05)
scroll_t.rotation_euler = (math.pi / 2, 0, 0)
scroll_t.scale = (1 / 0.6, 1 / 0.8, 1 / 0.02)
SC_IN = T("m01") + 0.3
SC_AT = R0 + Vector((0.55, -0.35, 1.2))
key_loc(scroll, S3 - 0.01, R0 + Vector((1.6, 3.6, 2.6)))
key_loc(scroll, SC_IN, R0 + Vector((1.6, 3.6, 2.6)))
key_loc(scroll, SC_IN + 1.6, SC_AT)
key_loc(scroll, T("f03") + 1.6, SC_AT + Vector((0, 0, 0.05)))
scroll.rotation_euler = (0, 0, 0.6)
scroll.keyframe_insert("rotation_euler", frame=f(SC_IN))
scroll.rotation_euler = (0, 0, 0)
scroll.keyframe_insert("rotation_euler", frame=f(SC_IN + 1.6))
key_scale(scroll, SC_IN, (0.6, 0.02, 0.1))
key_scale(scroll, SC_IN + 1.4, (0.6, 0.02, 0.8))
key_scale(scroll, T("f03") + 1.7, (0.6, 0.02, 0.8))
key_scale(scroll, T("f03") + 2.1, (0.01, 0.01, 0.01))
flash = light("scrollflash", "POINT", SC_AT + Vector((0, -0.3, 0.2)), 0, (0.8, 0.5, 1.0), 0.3, dist=30)
for t, e in ((T("f03") + 1.6, 0), (T("f03") + 1.9, 25000), (T("f03") + 2.6, 0)):
    flash.data.energy = e
    flash.data.keyframe_insert("energy", frame=f(t))

# ------------------------------------------------------------------ models
def bounds(objs):
    bpy.context.view_layer.update()
    dg = bpy.context.evaluated_depsgraph_get()
    mn = Vector((1e9,) * 3)
    mx = Vector((-1e9,) * 3)
    for o in objs:
        if o.type != "MESH":
            continue
        ev = o.evaluated_get(dg)
        me = ev.to_mesh()
        for v in me.vertices:
            p = ev.matrix_world @ v.co
            mn = Vector(map(min, mn, p))
            mx = Vector(map(max, mx, p))
        ev.to_mesh_clear()
    return mn, mx


def import_model(path, height, name):
    before = set(bpy.data.objects)
    ext = os.path.splitext(path)[1].lower()
    if ext == ".fbx":
        bpy.ops.import_scene.fbx(filepath=path)
    else:
        bpy.ops.import_scene.gltf(filepath=path)
    new = [o for o in bpy.data.objects if o not in before]
    arm = next((o for o in new if o.type == "ARMATURE"), None)
    mn, mx = bounds(new)
    s = height / max(1e-6, (mx.z - mn.z))
    root = bpy.data.objects.new(name, None)
    sc.collection.objects.link(root)
    for o in new:
        if o.parent is None:
            o.parent = root
    root.scale = (s, s, s)
    root["off"] = (-(mn.x + mx.x) / 2 * s, -(mn.y + mx.y) / 2 * s, -mn.z * s)
    for o in new:  # tripo markers and helper spheres
        if o.type == "MESH" and ("Icosphere" in o.name):
            o.hide_render = True
    say("model", name, os.path.basename(path), "objects", len(new), "armature", arm.name if arm else None,
        "bones", len(arm.data.bones) if arm else 0, "h", round(mx.z - mn.z, 3))
    return root, arm


def place(root, t, loc, rot_z=0.0):
    off = Vector(root["off"])
    c, s = math.cos(rot_z), math.sin(rot_z)
    root.location = Vector(loc) + Vector((off.x * c - off.y * s, off.x * s + off.y * c, off.z))
    root.rotation_euler = (0, 0, rot_z)
    root.keyframe_insert("location", frame=f(t))
    root.keyframe_insert("rotation_euler", frame=f(t))


FAR = (0, -600, -80)
EPS = 1.0 / FPS


def hero_file(hid):
    for p in (os.path.join(RIG, hid + "_fixed.glb"), os.path.join(RIG, hid + "_rigged.glb"), os.path.join(GLB, "hero_" + hid + ".glb")):
        if os.path.exists(p):
            return p
    return None


def find_file(patterns):
    for pat in patterns:
        hits = sorted(glob.glob(pat, recursive=True), key=os.path.getmtime, reverse=True)
        if hits:
            return hits[0]
    return None


cast = {}   # name -> dict(root, arm, ...)
for hid in HEROES:
    p = hero_file(hid)
    if p:
        r, a = import_model(p, 2.0, "hero_" + hid)
        cast[hid] = {"root": r, "arm": a}

# Mirabel: Gino's rigged Tripo model (downloads zip), extracted next to the intro
mir_dir = os.path.join(CHARS, "mirabel")
mir = find_file([os.path.join(mir_dir, "*.fbx"), os.path.join(mir_dir, "*.glb")])
if not mir:
    z = find_file([os.path.join(CHARS, "mirabel*.zip"), os.path.expanduser("~/Downloads/tripo_convert_21334b29*.zip")])
    if z:
        os.makedirs(mir_dir, exist_ok=True)
        zipfile.ZipFile(z).extractall(mir_dir)
        mir = find_file([os.path.join(mir_dir, "**", "*.fbx")])
if mir:
    r, a = import_model(mir, 1.95, "mirabel")
    cast["mirabel"] = {"root": r, "arm": a}
else:
    say("WARNING no Mirabel model")

# Fumbleton: the Tripo model made from his turnaround (rigged if the rig step ran)
fb = find_file([os.path.join(CHARS, "fumbleton_tripo", "rigged", "**", "*.glb"),
                os.path.join(CHARS, "fumbleton_tripo", "rigged", "**", "*.fbx"),
                os.path.join(CHARS, "fumbleton_tripo", "**", "model.glb"),
                os.path.join(CHARS, "fumbleton_tripo", "**", "*.glb")])
if fb:
    r, a = import_model(fb, 1.35, "fumbleton")
    cast["fumbleton"] = {"root": r, "arm": a}
else:
    say("WARNING no Fumbleton model; using a stand-in")
    root = bpy.data.objects.new("fumbleton", None)
    sc.collection.objects.link(root)
    robe = mat("robe", (0.45, 0.5, 0.78))
    bpy.ops.mesh.primitive_cone_add(vertices=24, radius1=0.42, radius2=0.25, depth=0.75, location=(0, 0, 0.38))
    bpy.context.object.data.materials.append(robe)
    bpy.context.object.parent = root
    sphere(0.17, (0, -0.05, 0.86), mat("skin", (0.92, 0.7, 0.55)), 16).parent = root
    bpy.ops.mesh.primitive_cone_add(vertices=24, radius1=0.4, depth=0.75, location=(0, 0, 1.25))
    bpy.context.object.data.materials.append(robe)
    bpy.context.object.parent = root
    root["off"] = (0, 0, 0)
    cast["fumbleton"] = {"root": root, "arm": None}

GUARD = {}
for g in ("prism", "crucible", "warden"):
    p = find_file([os.path.join(ROOT, "tripo-out", "guardian_" + g, "**", "model.glb")])
    if p:
        r, a = import_model(p, 9.0, "guardian_" + g)
        GUARD[g] = r
for c in cast.values():
    place(c["root"], 0, FAR)
for r in GUARD.values():
    place(r, 0, FAR)

# ------------------------------------------------------------------ posing
# Arms are aimed by direction in the character's own space (x = their left,
# -y = toward the camera they face, z = up), so the same pose works on the
# Mixamo-named hero rigs and on the Tripo rigs of Mirabel and Fumbleton.
import re


def side_of(n):
    n = n.lower()
    if "left" in n or re.search(r"(^|[_.:\s])l([_.\s]|$)", n) or n.endswith("_l") or n.endswith(".l"):
        return "L"
    if "right" in n or re.search(r"(^|[_.:\s])r([_.\s]|$)", n) or n.endswith("_r") or n.endswith(".r"):
        return "R"
    return ""


FINGER = ("thumb", "index", "middle", "ring", "pinky", "finger")


def find_bone(arm, part):
    if arm is None:
        return None
    pbs = arm.pose.bones
    for b in pbs:
        if b.name.split(":")[-1] == part:
            return b
    want = "L" if part.startswith("Left") else ("R" if part.startswith("Right") else "")
    core = part.replace("Left", "").replace("Right", "").lower()
    for b in pbs:
        n = b.name.lower().split(":")[-1]
        if want and side_of(n) != want:
            continue
        if any(x in n for x in FINGER) or n.endswith("end") or "nub" in n or "top" in n:
            continue
        if core == "head" and "head" in n:
            return b
        if core == "arm" and (("upperarm" in n) or (n.endswith("arm") and "fore" not in n and "lower" not in n) or "upper_arm" in n):
            return b
        if core == "forearm" and ("forearm" in n or "lowerarm" in n or "lower_arm" in n or "elbow" in n):
            return b
        if core == "hand" and "hand" in n:
            return b
    return None


class Rig:
    def __init__(self, name, c):
        self.name = name
        self.root = c["root"]
        self.arm = c["arm"]
        self.ok = False
        if self.arm is None:
            return
        self.b = {p: find_bone(self.arm, p) for p in ("LeftArm", "LeftForeArm", "RightArm", "RightForeArm", "Head", "LeftHand", "RightHand")}
        say("rig", name, {k: (v.name if v else None) for k, v in self.b.items()})
        if not all(self.b[p] for p in ("LeftArm", "LeftForeArm", "RightArm", "RightForeArm")):
            return
        self.arm.animation_data_clear()
        bpy.context.view_layer.update()
        rel = self.root.matrix_world.inverted() @ self.arm.matrix_world
        self.A = rel.to_3x3().normalized().inverted()     # character dir -> armature space
        for b in self.arm.pose.bones:
            b.rotation_mode = "QUATERNION"
        self.ok = True

    def d(self, v):
        return (self.A @ Vector(v)).normalized()

    def rest_dir(self, pb):
        return (pb.bone.tail_local - pb.bone.head_local).normalized()

    def aim_pair(self, ua, fa, d_ua, d_fa):
        R_ua = ua.bone.matrix_local.to_quaternion()
        q_ua = self.rest_dir(ua).rotation_difference(self.d(d_ua))
        b_ua = R_ua.inverted() @ q_ua @ R_ua
        R_fa = fa.bone.matrix_local.to_quaternion()
        cur = q_ua @ self.rest_dir(fa)
        q_fw = cur.rotation_difference(self.d(d_fa))
        b_fa = (q_ua @ R_fa).inverted() @ q_fw @ q_ua @ R_fa
        return b_ua, b_fa

    def head_rot(self, pitch, yaw):
        h = self.b.get("Head")
        if h is None:
            return None
        R = h.bone.matrix_local.to_quaternion()
        q = Quaternion(self.d((1, 0, 0)), math.radians(pitch)) @ Quaternion(self.d((0, 0, 1)), math.radians(yaw))
        return R.inverted() @ q @ R


POSES = {   # (left upper, left fore, right upper, right fore) in character space
    "rest":    ((0.32, 0.04, -1), (0.18, -0.25, -1), (-0.32, 0.04, -1), (-0.18, -0.25, -1)),
    "worry":   ((0.55, -0.55, 0.15), (0.05, -0.35, 1), (-0.55, -0.55, 0.15), (-0.05, -0.35, 1)),
    "read":    ((0.3, -0.75, -0.6), (0.1, -1, 0.15), (-0.3, -0.75, -0.6), (-0.1, -1, 0.15)),
    "cheer":   ((0.5, -0.2, 1), (0.2, -0.1, 1), (-0.5, -0.2, 1), (-0.2, -0.1, 1)),
    "flail":   ((0.9, -0.3, 0.6), (0.4, -0.2, 1), (-0.9, -0.3, 0.6), (-0.4, -0.2, 1)),
    "shrug":   ((0.7, -0.3, -0.4), (0.3, -1, 0.3), (-0.7, -0.3, -0.4), (-0.3, -1, 0.3)),
    "offer":   ((0.32, 0.04, -1), (0.18, -0.25, -1), (-0.45, -0.7, -0.35), (-0.1, -1, 0.3)),
    "split":   ((0.75, -0.45, 0.6), (0.3, -0.3, 1), (-0.75, -0.45, 0.6), (-0.3, -0.3, 1)),
    "send":    ((0.32, 0.04, -1), (0.18, -0.25, -1), (-0.25, -1, 0.45), (-0.1, -1, 0.5)),
    "catch":   ((0.35, -0.55, 0.75), (0.15, -0.35, 1), (-0.32, 0.04, -1), (-0.18, -0.25, -1)),
    "hold":    ((0.4, -0.45, 0.45), (0.1, -0.6, 0.8), (-0.32, 0.04, -1), (-0.18, -0.25, -1)),
    "hands":   ((0.5, -0.8, -0.1), (-0.3, -0.8, 0.4), (-0.5, -0.8, -0.1), (0.3, -0.8, 0.4)),
}
STATES = {k: [(0.0, "rest")] for k in list(cast.keys())}


def state(who, t, name):
    if who in STATES:
        STATES[who].append((t, name))


def pose_at(who, t):
    seq = sorted(STATES[who])
    cur, prev, t0 = "rest", "rest", 0.0
    for tt, nm in seq:
        if tt <= t:
            prev, cur, t0 = cur, nm, tt
    w = ramp(t, t0, t0 + 0.35)
    out = []
    for i in range(4):
        a, b = Vector(POSES[prev][i]).normalized(), Vector(POSES[cur][i]).normalized()
        v = a.lerp(b, w)
        out.append(v)
    if cur == "flail":  # wave the arms
        s = math.sin(t * 14)
        out[0] = (out[0] + Vector((0, 0, 0.5 * s))).normalized()
        out[2] = (out[2] + Vector((0, 0, -0.5 * s))).normalized()
    return out


def talk(who, t):
    """head nod/turn from the speaker's loudness, plus a slow idle sway"""
    pitch = 1.5 * math.sin(t * 2.1 + hash(who) % 7)
    yaw = 3.0 * math.sin(t * 0.9 + hash(who) % 5)
    for c in CUES["cues"]:
        spk = c["who"].lower()
        spk = {"arcanist": "arcane", "gambler": "lucky", "pyromancer": "pyro", "stormcaller": "storm"}.get(spk, spk)
        if spk != who:
            continue
        if c["start"] <= t < c["start"] + c["dur"]:
            i = int((t - c["start"]) * 12)
            e = c["env"][min(i, len(c["env"]) - 1)]
            pitch += 9 * e * math.sin(t * 11)
            yaw += 4 * e * math.sin(t * 5.3)
    return pitch, yaw


def bake_rigs():
    rigs = {k: Rig(k, v) for k, v in cast.items()}
    for k, r in rigs.items():
        cast[k]["rig"] = r
        root = r.root
        fr = 1
        while fr <= f(DUR) + 1:
            t = (fr - 1) / FPS
            p, y = talk(k, t)
            if r.ok:
                lu, lf, ru, rf = pose_at(k, t)
                br = math.sin(t * 2.1 + len(k))
                lu = (lu + Vector((0, 0.04 * br, 0))).normalized()
                ru = (ru + Vector((0, -0.04 * br, 0))).normalized()
                a, b = r.aim_pair(r.b["LeftArm"], r.b["LeftForeArm"], lu, lf)
                c, d = r.aim_pair(r.b["RightArm"], r.b["RightForeArm"], ru, rf)
                for pb, q in ((r.b["LeftArm"], a), (r.b["LeftForeArm"], b), (r.b["RightArm"], c), (r.b["RightForeArm"], d)):
                    pb.rotation_quaternion = q
                    pb.keyframe_insert("rotation_quaternion", frame=fr)
                hq = r.head_rot(p, y)
                if hq is not None:
                    r.b["Head"].rotation_quaternion = hq
                    r.b["Head"].keyframe_insert("rotation_quaternion", frame=fr)
            else:
                # no skeleton: a little bob and lean for life
                for ch in root.children:
                    b0 = ch.get("rot0")
                    if b0 is None:
                        b0 = tuple(ch.rotation_euler)
                        ch["rot0"] = b0
                    ch.rotation_euler = (b0[0] + math.radians(p * 0.4), b0[1], b0[2] + math.radians(y * 0.5))
                    ch.keyframe_insert("rotation_euler", frame=fr)
            fr += 2
    return rigs


# ------------------------------------------------------------------ props
orb_m = mat("orb", (0.22, 0.5, 1.0), emit=2.2, rough=0.1)
gem = [mat("g%d" % i, c, emit=1.2, rough=0.25) for i, c in enumerate(
    [(1, 0.2, 0.6), (0.3, 0.8, 1), (1, 0.8, 0.2), (0.6, 0.3, 1), (0.3, 1, 0.5), (1, 0.45, 0.15)])]


def orb_obj(r=0.11, energy=260, col=(0.7, 0.85, 1.0), m=None):
    o = sphere(r, FAR, m or orb_m, 32)
    l = bpy.data.lights.new("orbL", "POINT")
    l.energy = energy
    l.color = col
    l.shadow_soft_size = 0.3
    lo = bpy.data.objects.new("orbL", l)
    lo.parent = o
    sc.collection.objects.link(lo)
    return o


def burst(center, t, n=26, col=(1.0, 0.6, 1.0), radius=1.6, life=0.9):
    m = emit_mat("spark", col, 12)
    for i in range(n):
        d = Vector((rng.uniform(-1, 1), rng.uniform(-1, 1), rng.uniform(-0.3, 1))).normalized()
        s = sphere(0.035, center, m, 8)
        key_loc(s, t, center)
        key_loc(s, t + life, Vector(center) + d * radius * rng.uniform(0.6, 1.2))
        key_scale(s, t, 1)
        key_scale(s, t + life, 0.01)
        vis(s, t, t + life)


# trees in the meadow (S4 turns them to bricks)
bark = mat("bark", (0.25, 0.15, 0.1), rough=0.9)
leafm = [mat("leaf%d" % i, c, rough=0.8) for i, c in enumerate([(0.1, 0.45, 0.25), (0.25, 0.55, 0.2), (0.15, 0.35, 0.35)])]
outline = emit_mat("prismline", (0.5, 0.95, 1.0), 14)
trees = []
for i in range(34):
    x, y = rng.uniform(-24, 24), rng.uniform(14, 40)
    if abs(x) < 4 and y < 20:
        continue
    z = hill(x, y)
    root = bpy.data.objects.new("tree", None)
    sc.collection.objects.link(root)
    root.location = (x, y, z)
    bpy.ops.mesh.primitive_cylinder_add(vertices=8, radius=0.18, depth=1.6, location=(0, 0, 0.8))
    tr = bpy.context.object
    tr.data.materials.append(bark)
    tr.parent = root
    for k in range(3):
        bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=2, radius=1.0 - k * 0.2, location=(rng.uniform(-0.3, 0.3), rng.uniform(-0.3, 0.3), 1.9 + k * 0.7))
        c = bpy.context.object
        c.data.materials.append(leafm[(i + k) % 3])
        c.parent = root
        bpy.ops.object.shade_smooth()
    ol = bpy.data.objects.new("outline", None)
    sc.collection.objects.link(ol)
    ol.parent = root
    for ch in list(root.children):
        if ch.type == "MESH":
            o2 = ch.copy()
            o2.data = ch.data.copy()
            o2.data.materials.clear()
            o2.data.materials.append(outline)
            wf = o2.modifiers.new("w", "WIREFRAME")
            wf.thickness = 0.04
            o2.parent = ol
            o2.scale = ch.scale * 1.03
            sc.collection.objects.link(o2)
    trees.append((root, ol, Vector((x, y, z))))

# brick walls (S5) and the brick blocks the trees become (S4)
bricks = []
for row in range(5):
    for col in range(-11, 12):
        for wl in range(3):
            y = 18 + wl * 14
            b = cube((col * 1.95 + (row % 2) * 0.97, y, -3), (0.95, 0.5, 0.45), gem[(row + col + wl) % 6], 0.06)
            bricks.append((b, row, col, wl))


def brick_rise(b, t0, z1):
    b.location.z = z1 - 3
    b.keyframe_insert("location", frame=f(t0))
    b.location.z = z1 + 0.3
    b.keyframe_insert("location", frame=f(t0 + 0.35))
    b.location.z = z1
    b.keyframe_insert("location", frame=f(t0 + 0.55))


# S8 targets: bricks on the front wall near the middle
targets = [bb for bb in bricks if bb[3] == 0 and abs(bb[2]) <= 2 and bb[1] <= 3]
rng.shuffle(targets)
targets = targets[:9]
for b, row, col, wl in bricks:
    t0 = S5 + 0.3 + wl * 0.9 + abs(col) * 0.05 + row * 0.12
    z1 = 0.45 + row * 0.92 + hill(b.location.x, b.location.y) * 0.0
    brick_rise(b, t0, z1)
    b.hide_render = True
    b.keyframe_insert("hide_render", frame=1)
    b.hide_render = False
    b.keyframe_insert("hide_render", frame=f(t0))
    if wl == 0 and abs(col) <= 1 and row <= 3:
        b.hide_render = True  # opened up behind the heroes from the catches on
        b.keyframe_insert("hide_render", frame=f(S9))

# S4: Prism outlines, Crucible compresses, Warden stamps a brick block in its place
order = sorted(trees, key=lambda tr: tr[2].x)
for i, (root, ol, p) in enumerate(order):
    t_o = S4 + 0.4 + i * (S4b - S4 - 0.8) / len(order)
    if i >= len(order) * 0.6:   # Fumbleton's "undo" speeds it all up
        t_o = S4b + 0.6 + (i - len(order) * 0.6) * 0.07
    vis(ol, t_o, t_o + 0.5)
    for ob in objs_of(root):
        if ob is ol or ob.parent is ol:
            continue
        ob.hide_render = False
        ob.keyframe_insert("hide_render", frame=1)
        ob.hide_render = True
        ob.keyframe_insert("hide_render", frame=f(t_o + 0.85))
    key_scale(root, t_o + 0.4, 1)
    key_scale(root, t_o + 0.8, (0.7, 0.7, 0.25))
    for k in range(2 if i % 3 else 3):
        blk = cube(p + Vector((0, 0, 4)), (1.6, 1.0, 0.8), gem[(i + k) % 6], 0.08)
        z = p.z + 0.4 + k * 0.8
        key_loc(blk, t_o + 0.85 + k * 0.12, p + Vector((0, 0, 5)))
        key_loc(blk, t_o + 1.05 + k * 0.12, Vector((p.x, p.y, z - 0.08)))
        key_loc(blk, t_o + 1.15 + k * 0.12, Vector((p.x, p.y, z)))
        vis(blk, t_o + 0.85 + k * 0.12)
# extra bricks rain when Fumbleton tries to undo it
for k in range(40):
    x, y = rng.uniform(-22, 22), rng.uniform(12, 38)
    p = Vector((x, y, hill(x, y) + 0.4))
    t0 = S4b + 0.8 + k * 0.06
    blk = cube(p + Vector((0, 0, 8)), (1.6, 1.0, 0.8), gem[k % 6], 0.08)
    key_loc(blk, t0, p + Vector((0, 0, 8)))
    key_loc(blk, t0 + 0.3, p)
    vis(blk, t0)

# ------------------------------------------------------------------ staging
fum = cast["fumbleton"]["root"]
ROOM_F = R0 + Vector((0.2, 0.4, 0))
place(fum, 0, FAR)
place(fum, S2 - EPS, FAR)
place(fum, S2, ROOM_F, 0.25)
place(fum, S3, ROOM_F, 0.15)
place(fum, S3b - EPS, ROOM_F, -0.1)
place(fum, S3b, FAR)
place(fum, S4b - EPS, FAR)
place(fum, S4b, BALC, 0.35)
place(fum, S5 + 4.0, BALC, 0.0)
place(fum, S6b, BALC, -0.4)
place(fum, S6c, BALC, -0.4)
place(fum, S12 - EPS, FAR)
place(fum, S12, R0 + Vector((-1.4, 0.55, 0)), 0.0)
state("fumbleton", S2, "rest")
state("fumbleton", T("f02") + 0.4, "worry")
state("fumbleton", T("m01") + 1.8, "read")
state("fumbleton", T("f03") + 0.8, "cheer")
state("fumbleton", T("f04"), "worry")
state("fumbleton", S4b, "flail")
state("fumbleton", S5 + 0.4, "worry")
state("fumbleton", S6b, "shrug")
state("fumbleton", S12, "hands")

# guardians: appear round the tower (S3b), then stride through the meadow (S4)
GPOS = {"prism": Vector((-27, 34, 0)), "crucible": Vector((-9, 31, 0)), "warden": Vector((-17, 26, 0))}
for g, r in GUARD.items():
    gs0 = r.scale.x
    p = GPOS[g]
    p.z = hill(p.x, p.y)
    place(r, S3b - EPS, FAR)
    place(r, S3b, p, 0.0)
    q = Vector((p.x * 0.4 + 6, p.y - 8, 0))
    q.z = hill(q.x, q.y)
    place(r, S4, p, 0.2)
    place(r, S5, q, 0.0)
    place(r, S6 - EPS, q, 0.0)
    place(r, S6, FAR)
    key_scale(r, S3b - EPS, gs0 * 0.05)
    key_scale(r, S3b + 0.5, gs0)
gflash = light("gflash", "POINT", TW + Vector((0, -8, 6)), 0, (0.9, 0.6, 1.0), 2, dist=60)
for t, e in ((S3b, 0), (S3b + 0.15, 60000), (S3b + 0.9, 0)):
    gflash.data.energy = e
    gflash.data.keyframe_insert("energy", frame=f(t))

MIR = Vector((7.0, 8.0, 0))
MIR.z = hill(MIR.x, MIR.y)
if "mirabel" in cast:
    mr = cast["mirabel"]["root"]
    s0 = mr.scale.x
    mr["s0"] = s0
    place(mr, S6 - EPS, FAR)
    place(mr, S6, MIR, -0.35)
    place(mr, S8 - EPS, MIR, -0.25)
    place(mr, S8, FAR)
    key_scale(mr, S6, s0 * 0.02)
    key_scale(mr, S6 + 0.6, s0 * 1.04)
    key_scale(mr, S6 + 0.8, s0)
    state("mirabel", S6 + 0.6, "rest")
    state("mirabel", T("r02") + 0.6, "offer")
    state("mirabel", T("r02") + 2.6, "split")
    state("mirabel", S7 + 0.2, "send")
    burst(MIR + Vector((0, 0, 1.0)), S6, 40, (1.0, 0.55, 1.0), 2.2, 1.1)

# heroes: catches (S9) at the origin, then the V (S10, S11)
for k, h in enumerate(HEROES):
    if h not in cast:
        continue
    r = cast[h]["root"]
    t0 = SHOT0[h]
    t1 = SHOT0[HEROES[k + 1]] if k + 1 < len(HEROES) else S10
    place(r, t0 - EPS, FAR)
    place(r, t0, (0, 0, 0), 0.0)
    place(r, t1 - EPS, (0, 0, 0), 0.22)
    place(r, t1, FAR)
    state(h, CATCH_T[h] - 0.5, "catch")
    state(h, CATCH_T[h] + 0.6, "hold")
    state(h, t1, "rest")
V = [(0, 0.0), (-1.6, 1.4), (1.6, 1.4), (-3.2, 2.8), (3.2, 2.8), (-4.8, 4.2), (4.8, 4.2)]
for k, h in enumerate(HEROES):
    if h not in cast:
        continue
    x, y = V[k]
    place(cast[h]["root"], S10, (x, y, 0), -x * 0.06)
    place(cast[h]["root"], S12 - EPS, (x, y, 0), -x * 0.06)
    place(cast[h]["root"], S12, FAR)
    state(h, S10 + 0.2, "hold")

rigs = bake_rigs()

# ------------------------------------------------------------------ orbs
# Mirabel's one orb -> seven, then sent out
seven = []
if "mirabel" in cast:
    sc.frame_set(f(T("r02") + 1.4))
    rr = rigs["mirabel"]
    hb = rr.b.get("RightHand") if rr.ok else None
    hand = (rr.arm.matrix_world @ hb.matrix).translation if hb else MIR + Vector((-0.3, -0.5, 1.2))
else:
    hand = MIR + Vector((-0.3, -0.5, 1.2))
one = orb_obj(0.13, 400)
t_form = T("r02") + 1.2
key_loc(one, t_form, hand + Vector((0, 0, 0.15)))
key_scale(one, t_form, 0.01)
key_scale(one, t_form + 0.6, 1.0)
key_loc(one, T("r02") + 2.6, hand + Vector((0, 0, 0.15)))
above = MIR + Vector((0, -0.4, 2.6))
key_loc(one, T("r02") + 3.2, above)
vis(one, t_form, T("r02") + 3.4)
burst(above, T("r02") + 3.35, 30, (0.6, 0.8, 1.0), 1.2, 0.7)
for i in range(7):
    o = orb_obj(0.09, 160, GLOW[HEROES[i]], mat("o%d" % i, GLOW[HEROES[i]], emit=3.0, rough=0.1))
    a = i / 7 * 2 * math.pi
    ring = above + Vector((math.cos(a) * 0.9, math.sin(a) * 0.4, math.sin(a) * 0.25))
    key_loc(o, T("r02") + 3.35, above)
    key_loc(o, T("r02") + 3.9, ring)
    key_loc(o, S7 + 0.5, ring + Vector((0, 0, 0.3)))
    away = Vector((math.cos(a) * 60, 60 + math.sin(a) * 40, 30 + i * 4))
    mid = ring + (away - ring) * 0.3 + Vector((0, 0, 6))
    key_loc(o, S7 + 1.0, mid)
    key_loc(o, S7 + 2.4, away)
    vis(o, T("r02") + 3.35, S8)
    seven.append(o)

# S8: one orb breaks bricks on the front wall
demo = orb_obj(0.14, 500)
frag_m = gem
FRONT = Vector((0, 11.5, 1.6))
key_loc(demo, S8, Vector((-6, 30, 18)))
tt = S8 + 0.7
dt = (S9 - 0.2 - tt) / max(1, len(targets))
dt = min(0.75, dt)
for k, (b, row, col, wl) in enumerate(targets):
    hit = b.location.copy()
    hit.z = 0.45 + row * 0.92
    hit.y -= 0.55
    key_loc(demo, tt, hit)
    b.hide_render = True
    b.keyframe_insert("hide_render", frame=f(tt))
    for j in range(6):
        fr_ = cube(hit, (0.3, 0.2, 0.18), gem[(row + col) % 6], 0.02)
        d = Vector((rng.uniform(-1, 1), rng.uniform(-1.6, -0.3), rng.uniform(0.2, 1.4)))
        for s_ in range(6):
            u = s_ / 5 * 0.9
            pos = hit + d * 3.2 * u + Vector((0, 0, -4.9 * u * u))
            key_loc(fr_, tt + u, pos)
        fr_.rotation_euler = (0, 0, 0)
        fr_.keyframe_insert("rotation_euler", frame=f(tt))
        fr_.rotation_euler = (rng.uniform(-6, 6), rng.uniform(-6, 6), 0)
        fr_.keyframe_insert("rotation_euler", frame=f(tt + 0.9))
        vis(fr_, tt, tt + 0.95)
    fl = light("hitflash", "POINT", hit, 0, (0.7, 0.85, 1.0), 0.4, dist=25)
    for t_, e in ((tt - 0.01, 0), (tt, 9000), (tt + 0.35, 0)):
        fl.data.energy = e
        fl.data.keyframe_insert("energy", frame=f(t_))
    bounce = FRONT + Vector((rng.uniform(-3, 3), 0, rng.uniform(-0.6, 1.4)))
    key_loc(demo, tt + dt * 0.5, bounce)
    tt += dt
vis(demo, S8, S9)

# S9: the flying orb meets each raised hand at its catch, then is held
orb = orb_obj(0.11, 260)
vis(orb, S9, S10)
for k, h in enumerate(HEROES):
    if h not in cast:
        continue
    r = rigs[h]
    tc = CATCH_T[h]
    sc.frame_set(f(tc))
    hb = r.b.get("LeftHand") if r.ok else None
    hand = (r.arm.matrix_world @ hb.matrix).translation if hb else Vector((0.4, -0.4, 1.6))
    key_loc(orb, SHOT0[h] + 0.02, Vector((3.5, 9.0, 6.0)))
    key_loc(orb, tc - 0.4, hand + Vector((0.9, 1.2, 0.9)))
    key_loc(orb, tc - 0.02, hand)
    if hb is not None:
        held = orb_obj(0.07, 140)
        held.parent = r.arm
        held.parent_type = "BONE"
        held.parent_bone = hb.name
        sx = r.root.scale.x
        held.scale = (1 / sx, 1 / sx, 1 / sx)
        held.location = (0, 0.02 / sx, 0.05 / sx)
        vis(held, tc, S12)
        burst(hand, tc, 18, GLOW[h], 0.8, 0.5)
key_loc(orb, S10, Vector((0, 2.5, 4.4)))
key_loc(orb, S12, Vector((0, 2.5, 4.0)))
vis(orb, S10, S12)

# ------------------------------------------------------------------ cameras
cam_d = bpy.data.cameras.new("cam")
cam = bpy.data.objects.new("cam", cam_d)
sc.collection.objects.link(cam)
sc.camera = cam
cam_d.lens = 32
cam_d.clip_end = 1200
tgt = bpy.data.objects.new("tgt", None)
sc.collection.objects.link(tgt)
tcn = cam.constraints.new("TRACK_TO")
tcn.target = tgt
tcn.track_axis = "TRACK_NEGATIVE_Z"
tcn.up_axis = "UP_Y"


def shot(t, eye, look, lens=None):
    key_loc(cam, t, eye)
    key_loc(tgt, t, look)
    if lens:
        cam_d.lens = lens
        cam_d.keyframe_insert("lens", frame=f(t))


def cut(t0, t1, keys):
    """keys: list of (u 0..1, eye, look, lens)"""
    for u, eye, look, lens in keys:
        tt = t0 + u * (t1 - t0 - 1.0 / FPS)
        shot(tt, eye, look, lens)


Fh = Vector((0, 0, 0.95))
fum_room = ROOM_F
cut(0, S2, [(0, TW + Vector((10, -26, 4)), TW + Vector((0, 0, 11)), 30), (1, TW + Vector((6, -16, 9)), TW + Vector((0, 0, 12)), 34)])
cut(S2, S3, [(0, fum_room + Vector((0.8, -4.6, 1.5)), fum_room + Fh, 30), (0.55, fum_room + Vector((0.5, -3.6, 1.35)), fum_room + Fh, 34),
             (0.6, fum_room + Vector((-2.5, -2.5, 1.4)), fum_room + Vector((0.6, 1.0, 0.5)), 26), (1, fum_room + Vector((-1.8, -3.2, 1.5)), fum_room + Vector((0.3, 1.0, 0.6)), 24)])
cut(S3, S3b, [(0, fum_room + Vector((1.2, -3.4, 1.4)), fum_room + Vector((0.7, 1.5, 1.3)), 28), (0.4, fum_room + Vector((0.6, -2.6, 1.3)), fum_room + Vector((0.3, -0.2, 1.1)), 35),
              (1, fum_room + Vector((0.2, -2.0, 1.25)), fum_room + Fh + Vector((0, 0, 0.1)), 40)])
cut(S3b, S4, [(0, TW + Vector((4, -34, 3)), TW + Vector((-2, -6, 7)), 24), (1, TW + Vector((6, -40, 5)), TW + Vector((-2, -6, 7)), 22)])
cut(S4, S4b, [(0, Vector((-4, -6, 4)), Vector((0, 26, 3)), 26), (1, Vector((6, -2, 7)), Vector((-2, 26, 2)), 26)])
cut(S4b, S5, [(0, BALC + Vector((1.4, -3.4, 0.9)), BALC + Vector((0, 0, 0.8)), 30), (0.5, BALC + Vector((2.2, -5.0, 1.6)), BALC + Vector((3, 6, 0)), 24),
              (1, BALC + Vector((3.0, -7.0, 2.5)), BALC + Vector((10, 14, -4)), 22)])
cut(S5, S6, [(0, Vector((0, -12, 1.6)), Vector((0, 30, 2.5)), 26), (1, Vector((0, -16, 9)), Vector((0, 30, 1.0)), 24)])
mlook = MIR + Vector((0, 0, 1.35))
cut(S6, S6b, [(0, MIR + Vector((2.5, -6.0, 1.2)), mlook, 30), (1, MIR + Vector((1.2, -4.0, 1.5)), mlook, 38)])
cut(S6b, S6c, [(0, BALC + Vector((0.6, -2.6, 0.9)), BALC + Vector((0, 0, 0.75)), 40), (1, BALC + Vector((0.4, -2.3, 0.95)), BALC + Vector((0, 0, 0.75)), 42)])
cut(S6c, S7, [(0, MIR + Vector((-1.4, -3.8, 1.4)), mlook + Vector((0, 0, 0.2)), 34), (0.55, MIR + Vector((-0.8, -3.2, 1.6)), mlook + Vector((0, 0, 0.5)), 36),
              (1, MIR + Vector((-0.4, -4.4, 1.0)), above, 30)])
cut(S7, S8, [(0, MIR + Vector((0.5, -4.0, 0.6)), above, 22), (1, MIR + Vector((0.2, -3.0, 0.4)), MIR + Vector((5, 40, 26)), 20)])
cut(S8, S9, [(0, Vector((2.5, 5.0, 1.4)), Vector((0, 18, 2.2)), 26), (1, Vector((-1.5, 7.5, 1.6)), Vector((0, 18, 2.0)), 30)])
for h in HEROES:
    t0 = SHOT0[h]
    k = HEROES.index(h)
    t1 = SHOT0[HEROES[k + 1]] if k + 1 < len(HEROES) else S10
    sgn = 1 if k % 2 == 0 else -1
    cut(t0, t1, [(0, Vector((1.9 * sgn, -4.6, 1.35)), Vector((0, 0, 1.15)), 40), (0.35, Vector((1.0 * sgn, -3.3, 1.45)), Vector((0, 0, 1.3)), 45),
                 (1, Vector((0.45 * sgn, -2.5, 1.55)), Vector((0, 0, 1.4)), 50)])
cut(S10, S11, [(0, Vector((0, -4.5, 1.4)), Vector((0, 1.5, 1.5)), 28), (1, Vector((0, -9.5, 2.6)), Vector((0, 2.5, 1.6)), 30)])
cut(S11, S12, [(0, Vector((0, -9.5, 2.6)), Vector((0, 2.5, 3.2)), 30), (1, Vector((0, -10.5, 3.0)), Vector((0, 2.5, 3.4)), 30)])
cut(S12, DUR + 0.5, [(0, R0 + Vector((-1.4, -2.8, 1.25)), R0 + Vector((-1.4, 0.4, 0.85)), 34), (1, R0 + Vector((-1.4, -2.2, 1.2)), R0 + Vector((-1.4, 0.4, 0.85)), 38)])

# ------------------------------------------------------------------ cards, title, fades
def cam_child(o, x, y, d):
    o.parent = cam
    o.location = (x, y, -d)
    o.rotation_euler = (0, 0, 0)


card_bg = mat("cardbg", (0.05, 0.02, 0.1), alpha=0.75)
for h in HEROES:
    if h not in cast:
        continue
    k = HEROES.index(h)
    t1 = SHOT0[HEROES[k + 1]] if k + 1 < len(HEROES) else S10
    nm, sub = NAMES[h]
    holder = bpy.data.objects.new("card_" + h, None)
    sc.collection.objects.link(holder)
    cam_child(holder, -0.33, -0.15, 1.0)
    t_ = text(nm, 0.055, emit_mat("cn" + h, GLOW[h], 4), 0.0)
    t_.parent = holder
    t_.data.align_x = "LEFT"
    t_2 = text(sub.upper(), 0.024, emit_mat("cs" + h, (0.95, 0.92, 1.0), 2), 0.0)
    t_2.parent = holder
    t_2.data.align_x = "LEFT"
    t_2.location = (0.003, -0.045, 0)
    bpy.ops.mesh.primitive_plane_add(size=1, location=(0.15, -0.018, -0.002))
    pl = bpy.context.object
    pl.scale = (0.38, 0.13, 1)
    pl.data.materials.append(card_bg)
    pl.parent = holder
    vis(holder, CATCH_T[h] + 0.15, t1 - 0.05)
    key_loc(holder, CATCH_T[h] + 0.15, (-0.45, -0.15, -1.0))
    key_loc(holder, CATCH_T[h] + 0.45, (-0.33, -0.15, -1.0))

title = text("ARCANE ORB", 0.16, mat("titlem", (0.75, 0.55, 1.0), emit=5, metal=0.6, rough=0.25), 0.02)
cam_child(title, 0, 0.06, 1.2)
vis(title, S11, S12)
key_scale(title, S11, 0.6)
key_scale(title, S11 + 0.5, 1.0)
key_scale(title, S12, 1.06)
tag = text("break the walls  ·  save the land", 0.03, emit_mat("tagm", (1, 0.9, 0.7), 3), 0.0)
cam_child(tag, 0, -0.06, 1.2)
vis(tag, S11 + 0.6, S12)

fade_m = bpy.data.materials.new("fade")
fade_m.use_nodes = True
nt = fade_m.node_tree
nt.nodes.clear()
o_ = nt.nodes.new("ShaderNodeOutputMaterial")
mx_ = nt.nodes.new("ShaderNodeMixShader")
tr_ = nt.nodes.new("ShaderNodeBsdfTransparent")
em_ = nt.nodes.new("ShaderNodeEmission")
em_.inputs["Color"].default_value = (0, 0, 0, 1)
nt.links.new(tr_.outputs[0], mx_.inputs[1])
nt.links.new(em_.outputs[0], mx_.inputs[2])
nt.links.new(mx_.outputs[0], o_.inputs[0])
try:
    fade_m.surface_render_method = "BLENDED"
except Exception:
    pass
bpy.ops.mesh.primitive_plane_add(size=1)
fp = bpy.context.object
fp.scale = (3, 2, 1)
fp.data.materials.append(fade_m)
cam_child(fp, 0, 0, 0.5)
fac = mx_.inputs["Fac"]
for t_, v in ((0, 1), (0.8, 0), (S12 - 0.15, 0), (S12 - 0.05, 0.85), (S12 + 0.1, 0), (DUR - 0.6, 0), (DUR, 1)):
    fac.default_value = v
    fac.keyframe_insert("default_value", frame=f(t_))

# hide/show switches stay switches; motion between keys eases
def all_curves(idb):
    ad = getattr(idb, "animation_data", None)
    act = ad.action if ad else None
    if act is None:
        return []
    if hasattr(act, "fcurves") and len(getattr(act, "fcurves", [])):
        return list(act.fcurves)
    out = []
    for layer in getattr(act, "layers", []):
        for strip in layer.strips:
            for cb in strip.channelbags:
                out += list(cb.fcurves)
    return out


for ob in bpy.data.objects:
    for fc in all_curves(ob):
        mode = "CONSTANT" if fc.data_path in ("hide_render",) else None
        if ob.type == "EMPTY" and ob.name.startswith(("hero_", "mirabel", "fumbleton", "guardian_")) and fc.data_path in ("location", "rotation_euler"):
            mode = "CONSTANT" if mode else "LINEAR"
        for kp in fc.keyframe_points:
            if mode:
                kp.interpolation = mode
            else:
                kp.handle_left_type = kp.handle_right_type = "AUTO_CLAMPED"
sc.frame_set(1)
say("built; frames", f(DUR))

# ------------------------------------------------------------------ render
os.makedirs(os.path.join(HERE, "frames"), exist_ok=True)
if TEST:
    os.makedirs(os.path.join(HERE, "test"), exist_ok=True)
    stills = [1.2, S2 + 2.0, T("f02") + 1.5, T("m01") + 2.2, T("f03") + 1.0, S3b + 1.0, S4 + 2.5, S4b + 1.0, S5 + 2.5,
              S6 + 1.6, S6b + 0.4, T("r02") + 2.0, T("r02") + 3.8, S7 + 1.3, S8 + 2.5,
              CATCH_T["guardian"] + 0.6, CATCH_T["arcane"] + 0.6, CATCH_T["storm"] + 0.6, S10 + 2.5, S11 + 0.8, S12 + 1.2]
    for t in stills:
        sc.frame_set(f(t))
        sc.render.filepath = os.path.join(HERE, "test", "t%05.1f.jpg" % t)
        bpy.ops.render.render(write_still=True)
        say("still", round(t, 2))
else:
    sc.render.filepath = os.path.join(HERE, "frames", "f")
    bpy.ops.render.render(animation=True)
open(os.path.join(HERE, "done.txt"), "w").write("ok\n")
say("done")
