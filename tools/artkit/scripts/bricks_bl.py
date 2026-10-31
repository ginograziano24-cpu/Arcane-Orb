"""Shiny brick tiles rendered in Cycles (task bricks_tiles).

Every brick material is modelled as a real bevelled block (2 x 1 x 1) and
rendered orthographically from above and from the front, twice: once for
the diffuse look (colour, shading, glow) and once for the specular layer
alone (white highlights on black). The game multiplies the first by the
brick colour and adds the second on top, so highlights stay white and
bright whatever the colour. Tiles land in out/bricks/<name>_<view>_<pass>.png
(512 x 256) and are packed into sheets by tools/textures/pack_bricks.py.
"""
import os, math
import bpy, bmesh
import kit
from kit import log, setin

OUT = os.path.join(kit.OUT, "bricks")
W, H = 512, 256


# ------------------------------------------------------------------ helpers

def _mat(name, base=(0.9, 0.9, 0.9), rough=0.25, metal=0.0, coat=0.6, coat_rough=0.05, trans=0.0, ior=1.45,
         emit=None, emit_str=0.0, film=0.0, pattern=None, rough_pattern=None):
    """A material description: built per pass by _build()."""
    return dict(name=name, base=base, rough=rough, metal=metal, coat=coat, coat_rough=coat_rough, trans=trans,
                ior=ior, emit=emit, emit_str=emit_str, film=film, pattern=pattern, rough_pattern=rough_pattern)


def _build(md, spec_pass):
    m, nt, out = kit.new_mat(md["name"] + ("_s" if spec_pass else "_d"))
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    nt.links.new(p.outputs[0], out.inputs[0])
    if spec_pass:
        setin(p, (0, 0, 0, 1), 'Base Color')
        setin(p, 0.0, 'Metallic')
        setin(p, 0.0, 'Transmission Weight', 'Transmission')
        setin(p, 1.0 if md["metal"] < 0.5 else 2.0, 'Specular IOR Level', 'Specular')
        setin(p, md["rough"] if md["metal"] < 0.5 else min(md["rough"], 0.3), 'Roughness')
        setin(p, md["coat"], 'Coat Weight', 'Clearcoat')
        setin(p, md["coat_rough"], 'Coat Roughness', 'Clearcoat Roughness')
        setin(p, md["film"], 'Thin Film Thickness')
        if md["rough_pattern"]:
            md["rough_pattern"](nt, p)
        return m
    setin(p, (*md["base"], 1), 'Base Color')
    setin(p, md["metal"], 'Metallic')
    setin(p, md["rough"], 'Roughness')
    setin(p, 0.0, 'Specular IOR Level', 'Specular')
    setin(p, 0.0, 'Coat Weight', 'Clearcoat')
    setin(p, md["trans"], 'Transmission Weight', 'Transmission')
    setin(p, md["ior"], 'IOR')
    if md["emit"] is not None:
        setin(p, (*md["emit"], 1), 'Emission Color', 'Emission')
        setin(p, md["emit_str"], 'Emission Strength')
    if md["pattern"]:
        md["pattern"](nt, p)
    return m


def _obj_from_bm(name, bm, mat_d, mat_s):
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    o.data.materials.append(mat_d)
    o["mat_s"] = mat_s.name
    for f in o.data.polygons:
        f.use_smooth = True
    return o


def block(name, md, size=(2.0, 1.0, 1.0), loc=(0, 0, 0), bev=0.09, seg=5, extra=None):
    """A bevelled box; returns the object (diffuse material; spec material
    is swapped in for the spec pass)."""
    bm = bmesh.new()
    bmesh.ops.create_cube(bm, size=1.0)
    for v in bm.verts:
        v.co.x *= size[0]
        v.co.y *= size[1]
        v.co.z *= size[2]
        v.co.x += loc[0]
        v.co.y += loc[1]
        v.co.z += loc[2]
    o = _obj_from_bm(name, bm, _build(md, False), _build(md, True))
    if bev > 0:
        b = o.modifiers.new("bev", 'BEVEL')
        b.width = bev
        b.segments = seg
        try:
            b.harden_normals = False
        except Exception:
            pass
    if extra:
        extra(o)
    return o


def sphere_at(name, md, loc, r, seg=24):
    bpy.ops.mesh.primitive_uv_sphere_add(segments=seg, ring_count=seg // 2, radius=r, location=loc)
    o = bpy.context.active_object
    o.name = name
    o.data.materials.append(_build(md, False))
    o["mat_s"] = _build(md, True).name
    bpy.ops.object.shade_smooth()
    return o


def mesh_at(name, md, verts, faces, smooth=False):
    me = bpy.data.meshes.new(name)
    me.from_pydata(verts, [], faces)
    me.update()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    o.data.materials.append(_build(md, False))
    o["mat_s"] = _build(md, True).name
    for f in o.data.polygons:
        f.use_smooth = smooth
    return o


def _coords(nt):
    tc = nt.nodes.new('ShaderNodeTexCoord')
    return tc.outputs['Object']


def emit_pattern(color, strength, kind, scale=4.0, width=0.04, base_mix=None):
    """Returns a pattern() that drives emission (and optionally the base
    colour) from a procedural mask."""
    def fn(nt, p):
        co = _coords(nt)
        if kind == "cracks":
            v = nt.nodes.new('ShaderNodeTexVoronoi')
            try:
                v.feature = 'DISTANCE_TO_EDGE'
            except Exception:
                pass
            setin(v, scale, 'Scale')
            nt.links.new(co, v.inputs['Vector'])
            src = v.outputs['Distance']
            r = kit.ramp(nt, [(0.0, (1, 1, 1)), (width, (0, 0, 0)), (1.0, (0, 0, 0))])
        elif kind == "veins":
            n = nt.nodes.new('ShaderNodeTexNoise')
            setin(n, scale, 'Scale')
            setin(n, 6, 'Detail')
            nt.links.new(co, n.inputs['Vector'])
            src = n.outputs['Fac']
            r = kit.ramp(nt, [(0.0, (0, 0, 0)), (0.5 - width, (0, 0, 0)), (0.5, (1, 1, 1)), (0.5 + width, (0, 0, 0)), (1.0, (0, 0, 0))])
        elif kind == "swirl":
            mg = nt.nodes.new('ShaderNodeTexMagic')
            setin(mg, scale, 'Scale')
            setin(mg, 2.5, 'Distortion')
            nt.links.new(co, mg.inputs['Vector'])
            src = mg.outputs['Fac']
            r = kit.ramp(nt, [(0.0, (0, 0, 0)), (0.55, (0.2, 0.2, 0.2)), (0.75, (1, 1, 1)), (1.0, (1, 1, 1))])
        elif kind == "grid":
            v = nt.nodes.new('ShaderNodeTexVoronoi')
            try:
                v.feature = 'DISTANCE_TO_EDGE'
            except Exception:
                pass
            setin(v, scale, 'Scale')
            setin(v, 0.0, 'Randomness')
            nt.links.new(co, v.inputs['Vector'])
            src = v.outputs['Distance']
            r = kit.ramp(nt, [(0.0, (1, 1, 1)), (width, (0, 0, 0)), (1.0, (0, 0, 0))])
        elif kind == "seam":
            sep = nt.nodes.new('ShaderNodeSeparateXYZ')
            nt.links.new(co, sep.inputs[0])
            n = nt.nodes.new('ShaderNodeTexNoise')
            setin(n, 6.0, 'Scale')
            nt.links.new(co, n.inputs['Vector'])
            add = nt.nodes.new('ShaderNodeMath')
            add.operation = 'ADD'
            mul = nt.nodes.new('ShaderNodeMath')
            mul.operation = 'MULTIPLY'
            mul.inputs[1].default_value = 0.25
            nt.links.new(n.outputs['Fac'], mul.inputs[0])
            nt.links.new(sep.outputs['X'], add.inputs[0])
            sub = nt.nodes.new('ShaderNodeMath')
            sub.operation = 'SUBTRACT'
            nt.links.new(mul.outputs[0], sub.inputs[0])
            sub.inputs[1].default_value = 0.125
            nt.links.new(sub.outputs[0], add.inputs[1])
            ab = nt.nodes.new('ShaderNodeMath')
            ab.operation = 'ABSOLUTE'
            nt.links.new(add.outputs[0], ab.inputs[0])
            src = ab.outputs[0]
            r = kit.ramp(nt, [(0.0, (1, 1, 1)), (width, (0, 0, 0)), (1.0, (0, 0, 0))])
        else:  # dots / flakes
            v = nt.nodes.new('ShaderNodeTexVoronoi')
            setin(v, scale, 'Scale')
            nt.links.new(co, v.inputs['Vector'])
            src = v.outputs['Distance']
            r = kit.ramp(nt, [(0.0, (1, 1, 1)), (width, (0, 0, 0)), (1.0, (0, 0, 0))])
        nt.links.new(src, r.inputs['Fac'])
        mix = nt.nodes.new('ShaderNodeMix')
        try:
            mix.data_type = 'RGBA'
        except Exception:
            pass
        ins = [s for s in mix.inputs if s.type == 'RGBA']
        ins[0].default_value = (0, 0, 0, 1)
        ins[1].default_value = (*color, 1)
        nt.links.new(r.outputs['Color'], mix.inputs[0])
        out = [s for s in mix.outputs if s.type == 'RGBA'][0]
        nt.links.new(out, kit.sock(p, 'Emission Color', 'Emission'))
        setin(p, strength, 'Emission Strength')
        if base_mix is not None:
            m2 = nt.nodes.new('ShaderNodeMix')
            try:
                m2.data_type = 'RGBA'
            except Exception:
                pass
            i2 = [s for s in m2.inputs if s.type == 'RGBA']
            i2[0].default_value = (*base_mix[0], 1)
            i2[1].default_value = (*base_mix[1], 1)
            nt.links.new(r.outputs['Color'], m2.inputs[0])
            nt.links.new([s for s in m2.outputs if s.type == 'RGBA'][0], kit.sock(p, 'Base Color'))
    return fn


def flakes(scale=60.0):
    """Roughness pattern for the spec pass: a metal-flake sparkle."""
    def fn(nt, p):
        co = _coords(nt)
        v = nt.nodes.new('ShaderNodeTexVoronoi')
        setin(v, scale, 'Scale')
        nt.links.new(co, v.inputs['Vector'])
        r = kit.ramp(nt, [(0.0, (0.02, 0.02, 0.02)), (0.15, (0.35, 0.35, 0.35)), (1.0, (0.35, 0.35, 0.35))])
        nt.links.new(v.outputs['Distance'], r.inputs['Fac'])
        nt.links.new(r.outputs['Color'], kit.sock(p, 'Roughness'))
    return fn


# ------------------------------------------------------------------ the bricks
# Each builder makes the scene objects for one brick (centred, 2 x 1 x 1).

WHITE = (0.92, 0.92, 0.92)


def b_neon():
    glass = _mat("neon_glass", base=(0.95, 0.95, 1.0), rough=0.04, trans=0.85, ior=1.5, coat=1.0)
    frame = _mat("neon_frame", base=(0.12, 0.1, 0.16), rough=0.3, coat=0.8)
    core = _mat("neon_core", base=WHITE, rough=0.4, coat=0.0, emit=(1, 1, 1), emit_str=0.9)
    block("frame", frame, bev=0.06)
    block("glass", glass, size=(1.84, 0.88, 1.02), bev=0.05)
    block("core", core, size=(1.3, 0.5, 0.6), bev=0.08)


def b_meadow():
    # a clean candy block: deep enamel under a glass clear-coat, no inlay
    enamel = _mat("meadow_enamel", base=WHITE, rough=0.18, coat=1.0, coat_rough=0.02)
    o = block("body", enamel, bev=0.16, seg=8)


def b_caverns():
    # a cut gem: chamfered facets, flat table on top
    gem = _mat("cave_gem", base=WHITE, rough=0.04, trans=0.35, ior=1.7, coat=1.0, coat_rough=0.0)
    block("gem", gem, bev=0.16, seg=1)
    block("table", gem, size=(1.5, 0.56, 0.04), loc=(0, 0, 0.5), bev=0.08, seg=1)


def b_ruins():
    # polished veined marble
    stone = _mat("ruins_stone", base=(0.93, 0.9, 0.84), rough=0.22, coat=0.8, coat_rough=0.04,
                 pattern=emit_pattern((0, 0, 0), 0.0, "veins", scale=2.2, width=0.012,
                                      base_mix=((0.95, 0.92, 0.86), (0.62, 0.56, 0.5))))
    block("body", stone, bev=0.1, seg=5)


def b_clouds():
    # a plump pearl marshmallow on a thin gold belt
    pearl = _mat("cloud_pearl", base=WHITE, rough=0.16, coat=1.0, coat_rough=0.03, film=380.0)
    block("body", pearl, bev=0.24, seg=10)
    gold = _mat("cloud_gold", base=(1.0, 0.78, 0.38), rough=0.15, metal=1.0, coat=0.5)
    block("band", gold, size=(2.02, 1.02, 0.07), loc=(0, 0, -0.2), bev=0.03, seg=3)


def b_machine():
    # a machined steel plate with a raised centre and corner rivets
    steel = _mat("mach_steel", base=(0.85, 0.85, 0.88), rough=0.24, metal=1.0, coat=0.4)
    block("body", steel, bev=0.07, seg=3)
    block("boss", steel, size=(1.6, 0.6, 0.06), loc=(0, 0, 0.5), bev=0.04, seg=4)
    for x in (-0.88, 0.88):
        for y in (-0.38, 0.38):
            sphere_at("rivet", steel, (x, y, 0.5), 0.045)


def b_steel():
    steel = _mat("steel_plate", base=(0.66, 0.68, 0.72), rough=0.32, metal=1.0, coat=0.4)
    block("body", steel, bev=0.05, seg=3)
    for i in range(9):
        for j in range(4):
            x = -0.8 + i * 0.2
            y = -0.3 + j * 0.2
            a = 45 if (i + j) % 2 == 0 else -45
            o = block("lz%d_%d" % (i, j), steel, size=(0.14, 0.035, 0.03), loc=(x, y, 0.505), bev=0.012, seg=2)
            o.rotation_euler = (0, 0, math.radians(a))


def b_elite():
    # deep lacquer with a gold belt and gilded edges
    lac = _mat("elite_lacquer", base=(0.18, 0.04, 0.24), rough=0.1, coat=1.0, coat_rough=0.02, film=500.0)
    gold = _mat("elite_gold", base=(1.0, 0.76, 0.3), rough=0.12, metal=1.0, coat=0.6)
    block("body", lac, bev=0.12, seg=6)
    block("trim", gold, size=(2.03, 1.03, 0.1), loc=(0, 0, 0.3), bev=0.03, seg=3)
    block("edge", gold, size=(2.0, 1.0, 0.04), loc=(0, 0, 0.47), bev=0.02, seg=2)
    block("top", lac, size=(1.84, 0.84, 0.08), loc=(0, 0, 0.47), bev=0.04, seg=4)


def b_armored():
    gun = _mat("arm_gun", base=(0.42, 0.45, 0.52), rough=0.25, metal=1.0, coat=0.5)
    block("body", gun, bev=0.06, seg=3)
    block("plateA", gun, size=(0.95, 0.94, 0.06), loc=(-0.5, 0, 0.5), bev=0.03, seg=2)
    block("plateB", gun, size=(0.95, 0.94, 0.06), loc=(0.5, 0, 0.5), bev=0.03, seg=2)
    for x in (-0.9, -0.1, 0.1, 0.9):
        for y in (-0.38, 0.38):
            sphere_at("rv", gun, (x, y, 0.54), 0.045)


def b_explosive():
    red = _mat("exp_red", base=(0.85, 0.08, 0.04), rough=0.2, coat=1.0, coat_rough=0.03,
               pattern=emit_pattern((1.0, 0.45, 0.05), 6.0, "cracks", scale=1.6, width=0.03,
                                    base_mix=((0.8, 0.06, 0.03), (0.25, 0.03, 0.0))))
    block("body", red, bev=0.1, seg=5)
    band = _mat("exp_band", base=(0.12, 0.11, 0.1), rough=0.45, metal=0.6, coat=0.2)
    block("band", band, size=(2.02, 1.02, 0.12), loc=(0, 0, -0.12), bev=0.03, seg=3)


def b_chain():
    navy = _mat("chain_navy", base=(0.05, 0.12, 0.45), rough=0.12, coat=1.0,
                pattern=emit_pattern((0.45, 0.9, 1.0), 6.0, "veins", scale=2.5, width=0.028))
    block("body", navy, bev=0.1, seg=5)


def b_split():
    stone = _mat("split_stone", base=(0.95, 0.55, 0.12), rough=0.25, coat=0.9,
                 pattern=emit_pattern((1.0, 0.9, 0.5), 6.0, "seam", width=0.035,
                                      base_mix=((0.95, 0.55, 0.12), (0.3, 0.12, 0.02))))
    block("body", stone, bev=0.1, seg=5)


def b_regen():
    jade = _mat("regen_jade", base=(0.12, 0.7, 0.3), rough=0.14, trans=0.15, coat=1.0,
                pattern=emit_pattern((0.6, 1.0, 0.5), 4.5, "veins", scale=2.6, width=0.04))
    block("body", jade, bev=0.12, seg=6)


def b_moving():
    chrome = _mat("move_chrome", base=(0.55, 0.38, 0.95), rough=0.12, metal=1.0, coat=0.7)
    block("body", chrome, bev=0.1, seg=5)
    dark = _mat("move_dark", base=(0.08, 0.05, 0.15), rough=0.4)
    for k in range(3):
        block("str%d" % k, dark, size=(2.01, 0.06, 1.01), loc=(0, -0.28 + k * 0.28, 0), bev=0.0)


def b_shield():
    blue = _mat("shield_blue", base=(0.12, 0.35, 0.95), rough=0.1, coat=1.0,
                pattern=emit_pattern((0.6, 0.9, 1.0), 4.0, "grid", scale=7.0, width=0.06))
    block("body", blue, bev=0.1, seg=5)
    dome = _mat("shield_dome", base=(0.6, 0.85, 1.0), rough=0.02, trans=0.9, coat=1.0, emit=(0.4, 0.7, 1.0), emit_str=0.3)
    o = block("dome", dome, size=(1.9, 0.9, 0.5), loc=(0, 0, 0.42), bev=0.0)
    s = o.modifiers.new("ss", 'SUBSURF')
    s.levels = 3
    s.render_levels = 3


def b_teleport():
    purple = _mat("tele_purple", base=(0.35, 0.05, 0.55), rough=0.06, trans=0.3, coat=1.0,
                  pattern=emit_pattern((1.0, 0.5, 1.0), 4.0, "swirl", scale=2.2))
    block("body", purple, bev=0.12, seg=6)


def b_spawner():
    amber = _mat("spawn_amber", base=(0.95, 0.6, 0.08), rough=0.08, trans=0.45, coat=1.0,
                 pattern=emit_pattern((1.0, 0.92, 0.45), 4.0, "dots", scale=9.0, width=0.1))
    block("body", amber, bev=0.14, seg=6)


def b_multiplier():
    gold = _mat("mult_gold", base=(1.0, 0.72, 0.18), rough=0.15, metal=1.0, coat=0.8, rough_pattern=flakes(70))
    block("body", gold, bev=0.1, seg=5)


def b_crystal():
    shell = _mat("crys_shell", base=(0.6, 0.95, 1.0), rough=0.02, trans=0.9, ior=1.6, coat=1.0)
    core = _mat("crys_core", base=(0.3, 0.9, 1.0), rough=0.1, emit=(0.25, 0.85, 1.0), emit_str=2.5)
    block("shell", shell, bev=0.1, seg=4)
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=1, radius=0.38, location=(0, 0, 0.05))
    o = bpy.context.active_object
    o.scale = (1.4, 0.8, 1.0)
    o.data.materials.append(_build(core, False))
    o["mat_s"] = _build(core, True).name


def b_mystery():
    pearl = _mat("myst_pearl", base=(0.75, 0.55, 0.95), rough=0.1, coat=1.0, film=650.0,
                 pattern=emit_pattern((0.95, 0.75, 1.0), 3.5, "swirl", scale=3.0))
    block("body", pearl, bev=0.12, seg=6)


WORLD_BRICKS = [("neon", b_neon), ("meadow", b_meadow), ("caverns", b_caverns), ("ruins", b_ruins),
                ("clouds", b_clouds), ("machine", b_machine), ("steel", b_steel), ("elite", b_elite)]
SPECIAL_BRICKS = [("armored", b_armored), ("explosive", b_explosive), ("chain", b_chain), ("split", b_split),
                  ("regen", b_regen), ("moving", b_moving), ("shield", b_shield), ("teleport", b_teleport),
                  ("spawner", b_spawner), ("multiplier", b_multiplier), ("crystal", b_crystal), ("mystery", b_mystery)]


# ------------------------------------------------------------------ render

def _studio():
    sc = bpy.context.scene
    w = bpy.data.worlds.new("studio")
    sc.world = w
    w.use_nodes = True
    nt = w.node_tree
    bg = nt.nodes.get("Background")
    # a soft studio dome: bright above, dim below, so reflections read
    tc = nt.nodes.new('ShaderNodeTexCoord')
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(tc.outputs['Generated'], sep.inputs[0])
    r = kit.ramp(nt, [(0.0, (0.01, 0.01, 0.012)), (0.5, (0.03, 0.03, 0.04)), (0.8, (0.08, 0.085, 0.1)), (1.0, (0.12, 0.12, 0.14))])
    nt.links.new(sep.outputs['Z'], r.inputs['Fac'])
    nt.links.new(r.outputs['Color'], bg.inputs[0])
    bg.inputs[1].default_value = 1.0

    def area(name, loc, rot, size, energy, color=(1, 1, 1), shape='RECTANGLE', sy=None):
        d = bpy.data.lights.new(name, 'AREA')
        d.energy = energy
        d.color = color
        try:
            d.shape = shape
            d.size = size
            if sy is not None:
                d.size_y = sy
        except Exception:
            d.size = size
        o = bpy.data.objects.new(name, d)
        sc.collection.objects.link(o)
        o.location = loc
        o.rotation_euler = [math.radians(a) for a in rot]
    # key softbox up and to the front-left, a long strip for the front edge,
    # a warm rim from behind, and a thin diagonal strip straight overhead
    # that the flat top reflects as a glossy streak
    area("key", (-3.0, -3.5, 6.0), (35, 0, -40), 3.0, 260, (1.0, 0.97, 0.92), sy=1.5)
    area("fill", (3.0, -4.0, 3.0), (60, 0, 35), 4.0, 90, (0.85, 0.9, 1.0), sy=4.0)
    area("strip", (0.0, -6.0, 2.5), (68, 0, 0), 8.0, 140, (0.95, 0.97, 1.0), sy=0.3)
    area("rim", (3.5, 4.0, 3.5), (-45, 0, 140), 3.0, 160, (1.0, 0.85, 0.7), sy=3.0)
    area("streak", (-0.35, 0.05, 5.0), (0, 0, 28), 0.22, 60, (1, 1, 1), sy=4.0)
    area("streak2", (0.55, -0.05, 5.0), (0, 0, 28), 0.08, 30, (1, 1, 1), sy=4.0)


def _set_pass(spec):
    for o in bpy.context.scene.objects:
        if o.type != 'MESH' or "mat_s" not in o.keys():
            continue
        name = o.data.materials[0].name
        if spec:
            o["mat_d"] = name
            o.data.materials[0] = bpy.data.materials[o["mat_s"]]
        elif "mat_d" in o.keys():
            o.data.materials[0] = bpy.data.materials[o["mat_d"]]


def _render_views(name):
    sc = bpy.context.scene
    cam_top = kit.camera((0, 0, 10), (0, 0, 0), lens=50)
    cam_top.data.type = 'ORTHO'
    cam_top.data.ortho_scale = 2.0
    cam_front = kit.camera((0, -10, 0), (90, 0, 0), lens=50)
    cam_front.data.type = 'ORTHO'
    cam_front.data.ortho_scale = 2.0
    for view, cam in (("top", cam_top), ("front", cam_front)):
        sc.camera = cam
        for spec in (False, True):
            _set_pass(spec)
            kit.render_to(os.path.join(OUT, "%s_%s_%s.png" % (name, view, "s" if spec else "d")))
        _set_pass(False)


def render_brick(name, fn):
    kit.reset()
    kit.setup_render(W, H, samples=160)
    sc = bpy.context.scene
    sc.render.film_transparent = False
    try:
        sc.view_settings.view_transform = 'Standard'
        sc.view_settings.look = 'None'
    except Exception:
        pass
    sc.cycles.max_bounces = 10
    try:
        sc.cycles.transmission_bounces = 10
        sc.cycles.glossy_bounces = 6
    except Exception:
        pass
    _studio()
    fn()
    _render_views(name)


def bricks_tiles():
    os.makedirs(OUT, exist_ok=True)
    for name, fn in WORLD_BRICKS + SPECIAL_BRICKS:
        render_brick(name, fn)
    log("brick tiles done")


TASKS = {"bricks_tiles": bricks_tiles}
for _n, _f in WORLD_BRICKS + SPECIAL_BRICKS:
    TASKS["brick_" + _n] = (lambda n, f: (lambda: (os.makedirs(OUT, exist_ok=True), render_brick(n, f))))(_n, _f)
