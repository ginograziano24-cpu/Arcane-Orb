"""Shared helpers for the Arcane Orb art kit (Blender 4.x / 5.x, Cycles).

Everything here is written defensively: Blender renames sockets and enum
values between versions, so setters try several names and log instead of
crashing.
"""
import bpy, bmesh, math, os, random, sys, time, traceback
from mathutils import Vector, Euler

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "out")
os.makedirs(OUT, exist_ok=True)


def log(*a):
    print("[kit]", *a, flush=True)


# ------------------------------------------------------------------ scene

def reset():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    sc = bpy.context.scene
    sc.unit_settings.system = 'METRIC'
    return sc


def setup_gpu():
    """Enable the best GPU backend Cycles can find; fall back to CPU."""
    try:
        prefs = bpy.context.preferences.addons['cycles'].preferences
    except Exception as e:
        log("cycles prefs unavailable:", e)
        return 'CPU'
    for kind in ('OPTIX', 'CUDA', 'HIP', 'ONEAPI', 'METAL'):
        try:
            prefs.compute_device_type = kind
        except Exception:
            continue
        try:
            prefs.refresh_devices()
        except Exception:
            try:
                prefs.get_devices()
            except Exception:
                pass
        devs = [d for d in prefs.devices if d.type == kind]
        if devs:
            for d in prefs.devices:
                d.use = (d.type == kind)
            log("GPU backend:", kind, "->", ", ".join(d.name for d in devs))
            return kind
    log("No GPU backend found, rendering on CPU")
    return 'CPU'


def setup_render(w, h, samples=128, transparent=False):
    sc = bpy.context.scene
    sc.render.engine = 'CYCLES'
    backend = setup_gpu()
    sc.cycles.device = 'GPU' if backend != 'CPU' else 'CPU'
    sc.cycles.samples = samples
    try:
        sc.cycles.use_adaptive_sampling = True
        sc.cycles.adaptive_threshold = 0.02
    except Exception:
        pass
    sc.cycles.use_denoising = True
    for dn in ('OPENIMAGEDENOISE', 'OPTIX'):
        try:
            sc.cycles.denoiser = dn
            break
        except Exception:
            pass
    sc.cycles.max_bounces = 6
    sc.render.resolution_x, sc.render.resolution_y = w, h
    sc.render.resolution_percentage = 100
    sc.render.film_transparent = transparent
    for vt in ('AgX', 'Filmic', 'Standard'):
        try:
            sc.view_settings.view_transform = vt
            break
        except Exception:
            pass
    try:
        sc.view_settings.look = 'AgX - Medium High Contrast' if sc.view_settings.view_transform == 'AgX' else 'None'
    except Exception:
        pass
    return backend


def render_to(path, fmt='PNG', quality=92):
    sc = bpy.context.scene
    sc.render.image_settings.file_format = fmt
    if fmt == 'JPEG':
        sc.render.image_settings.quality = quality
    else:
        sc.render.image_settings.color_mode = 'RGBA' if sc.render.film_transparent else 'RGB'
    sc.render.filepath = path
    t = time.time()
    bpy.ops.render.render(write_still=True)
    log("rendered", os.path.basename(path), "in %.1fs" % (time.time() - t))


def camera(loc, rot_deg, lens=35, shift_y=0.0, clip_end=4000):
    cam_data = bpy.data.cameras.new("Cam")
    cam_data.lens = lens
    cam_data.shift_y = shift_y
    cam_data.clip_end = clip_end
    cam = bpy.data.objects.new("Cam", cam_data)
    bpy.context.scene.collection.objects.link(cam)
    cam.location = loc
    cam.rotation_euler = Euler([math.radians(a) for a in rot_deg])
    bpy.context.scene.camera = cam
    return cam


def sun(rot_deg, strength=3.0, color=(1, 1, 1), angle_deg=2.0):
    d = bpy.data.lights.new("Sun", 'SUN')
    d.energy = strength
    d.color = color
    d.angle = math.radians(angle_deg)
    o = bpy.data.objects.new("Sun", d)
    bpy.context.scene.collection.objects.link(o)
    o.rotation_euler = Euler([math.radians(a) for a in rot_deg])
    return o


def point_light(loc, power, color=(1, 1, 1), radius=0.5):
    d = bpy.data.lights.new("Pt", 'POINT')
    d.energy = power
    d.color = color
    d.shadow_soft_size = radius
    o = bpy.data.objects.new("Pt", d)
    bpy.context.scene.collection.objects.link(o)
    o.location = loc
    return o


# ------------------------------------------------------------------ nodes

def sock(node, *names):
    for n in names:
        if n in node.inputs:
            return node.inputs[n]
    return None


def setin(node, value, *names):
    s = sock(node, *names)
    if s is not None:
        try:
            s.default_value = value
        except Exception as e:
            log("could not set", names, e)
    return s


def new_mat(name):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    for n in list(nt.nodes):
        nt.nodes.remove(n)
    out = nt.nodes.new('ShaderNodeOutputMaterial')
    return m, nt, out


def principled(nt, base=(0.8, 0.8, 0.8), rough=0.5, metal=0.0, emit=None, emit_str=0.0, spec=0.5):
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    setin(p, (*base, 1), 'Base Color')
    setin(p, rough, 'Roughness')
    setin(p, metal, 'Metallic')
    setin(p, spec, 'Specular IOR Level', 'Specular')
    if emit is not None:
        setin(p, (*emit, 1), 'Emission Color', 'Emission')
        setin(p, emit_str, 'Emission Strength')
    return p


def ramp(nt, stops):
    """stops: [(pos, (r,g,b)), ...]"""
    r = nt.nodes.new('ShaderNodeValToRGB')
    el = r.color_ramp.elements
    while len(el) > 2:
        el.remove(el[-1])
    el[0].position, el[0].color = stops[0][0], (*stops[0][1], 1)
    el[1].position, el[1].color = stops[-1][0], (*stops[-1][1], 1)
    for pos, c in stops[1:-1]:
        e = el.new(pos)
        e.color = (*c, 1)
    return r


def noise_tex(nt, scale=5, detail=8, rough=0.55, coord=None):
    n = nt.nodes.new('ShaderNodeTexNoise')
    setin(n, scale, 'Scale')
    setin(n, detail, 'Detail')
    setin(n, rough, 'Roughness')
    if coord is not None:
        nt.links.new(coord, n.inputs['Vector'])
    return n


def voronoi_tex(nt, scale=5, coord=None, feature='F1'):
    v = nt.nodes.new('ShaderNodeTexVoronoi')
    setin(v, scale, 'Scale')
    try:
        v.feature = feature
    except Exception:
        pass
    if coord is not None:
        nt.links.new(coord, v.inputs['Vector'])
    return v


def bump(nt, height_out, strength=0.3, distance=0.1):
    b = nt.nodes.new('ShaderNodeBump')
    setin(b, strength, 'Strength')
    setin(b, distance, 'Distance')
    nt.links.new(height_out, b.inputs['Height'])
    return b


def mat_color(name, base, rough=0.6, metal=0.0, emit=None, emit_str=0.0, noise=0.0, noise_scale=8.0, bump_str=0.0):
    """A principled material with optional colour variation and bump."""
    m, nt, out = new_mat(name)
    p = principled(nt, base, rough, metal, emit, emit_str)
    if noise > 0 or bump_str > 0:
        tc = nt.nodes.new('ShaderNodeTexCoord')
        n = noise_tex(nt, noise_scale, 10, 0.6, tc.outputs['Object'])
        if noise > 0:
            lo = tuple(max(0, c * (1 - noise)) for c in base)
            hi = tuple(min(1, c * (1 + noise * 0.6)) for c in base)
            r = ramp(nt, [(0.3, lo), (0.7, hi)])
            nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
            nt.links.new(r.outputs['Color'], p.inputs['Base Color'])
        if bump_str > 0:
            b = bump(nt, n.outputs['Fac'], bump_str, 0.2)
            nt.links.new(b.outputs['Normal'], p.inputs['Normal'])
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    return m


def mat_emit(name, color, strength):
    m, nt, out = new_mat(name)
    e = nt.nodes.new('ShaderNodeEmission')
    setin(e, (*color, 1), 'Color')
    setin(e, strength, 'Strength')
    nt.links.new(e.outputs[0], out.inputs['Surface'])
    return m


def world_gradient(stops, strength=1.0, sun_dir=None, sun_color=(1, 0.8, 0.6), sun_size=0.03, sun_str=20.0, ground=None):
    """Sky from a vertical colour ramp over the view direction's elevation.
    stops are positions in 0..1 where 0.5 is the horizon, 1 straight up."""
    w = bpy.data.worlds.new("World")
    bpy.context.scene.world = w
    w.use_nodes = True
    nt = w.node_tree
    for n in list(nt.nodes):
        nt.nodes.remove(n)
    out = nt.nodes.new('ShaderNodeOutputWorld')
    tc = nt.nodes.new('ShaderNodeTexCoord')
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(tc.outputs['Generated'], sep.inputs[0])
    # generated coords of the world are the view direction
    mr = nt.nodes.new('ShaderNodeMapRange')
    setin(mr, -1.0, 'From Min')
    setin(mr, 1.0, 'From Max')
    nt.links.new(sep.outputs['Z'], mr.inputs['Value'])
    r = ramp(nt, stops)
    nt.links.new(mr.outputs['Result'], r.inputs['Fac'])
    bg = nt.nodes.new('ShaderNodeBackground')
    setin(bg, strength, 'Strength')
    col = r.outputs['Color']
    if sun_dir is not None:
        # sun disc + glow added on top
        d = Vector(sun_dir).normalized()
        dot = nt.nodes.new('ShaderNodeVectorMath')
        dot.operation = 'DOT_PRODUCT'
        nt.links.new(tc.outputs['Generated'], dot.inputs[0])
        dot.inputs[1].default_value = d
        disc = nt.nodes.new('ShaderNodeMapRange')
        setin(disc, math.cos(sun_size), 'From Min')
        setin(disc, math.cos(sun_size * 0.8), 'From Max')
        nt.links.new(dot.outputs['Value'], disc.inputs['Value'])
        glow = nt.nodes.new('ShaderNodeMath')
        glow.operation = 'POWER'
        clamp = nt.nodes.new('ShaderNodeMath')
        clamp.operation = 'MAXIMUM'
        nt.links.new(dot.outputs['Value'], clamp.inputs[0])
        clamp.inputs[1].default_value = 0.0
        nt.links.new(clamp.outputs[0], glow.inputs[0])
        glow.inputs[1].default_value = 24.0
        add = nt.nodes.new('ShaderNodeMath')
        add.operation = 'MULTIPLY_ADD'
        nt.links.new(disc.outputs['Result'], add.inputs[0])
        add.inputs[1].default_value = sun_str
        nt.links.new(glow.outputs[0], add.inputs[2])
        sc_ = nt.nodes.new('ShaderNodeMix')
        sc_.data_type = 'RGBA'
        sc_.blend_type = 'ADD'
        nt.links.new(add.outputs[0], sc_.inputs[0])
        nt.links.new(col, sc_.inputs[6])
        sc_.inputs[7].default_value = (*sun_color, 1)
        col = sc_.outputs[2]
    nt.links.new(col, bg.inputs['Color'])
    nt.links.new(bg.outputs[0], out.inputs['Surface'])
    return w


def world_fog(density=0.002, color=(0.8, 0.8, 0.9), anisotropy=0.3, size=6000.0, height=120.0, base=-5.0):
    """Aerial perspective as a low haze layer: wide but only `height` tall,
    so horizontal (horizon) rays get hazy while sun and sky pass through.
    (A world volume would be infinitely deep and block sun and sky.)"""
    m, nt, out = new_mat("Fog")
    v = nt.nodes.new('ShaderNodeVolumePrincipled')
    setin(v, (*color, 1), 'Color')
    setin(v, density, 'Density')
    setin(v, anisotropy, 'Anisotropy')
    nt.links.new(v.outputs[0], out.inputs['Volume'])
    bpy.ops.mesh.primitive_cube_add(location=(0, size / 2 - 50, base + height / 2))
    o = bpy.context.active_object
    o.name = "FogVolume"
    o.scale = (size / 2, size / 2, height / 2)
    o.data.materials.append(m)
    try:
        o.visible_shadow = True
    except Exception:
        pass
    return o


# ------------------------------------------------------------------ geometry

def link(obj):
    bpy.context.scene.collection.objects.link(obj)
    return obj


def mesh_obj(name, verts, faces, mat=None, smooth=False):
    me = bpy.data.meshes.new(name)
    me.from_pydata(verts, [], faces)
    me.update()
    o = link(bpy.data.objects.new(name, me))
    if mat:
        o.data.materials.append(mat)
    if smooth:
        for p in me.polygons:
            p.use_smooth = True
    return o


def grid_terrain(name, size_x, size_y, nx, ny, height_fn, mat=None, origin=(0, 0, 0), smooth=True):
    verts, faces = [], []
    for j in range(ny + 1):
        for i in range(nx + 1):
            x = origin[0] + (i / nx - 0.5) * size_x
            y = origin[1] + (j / ny) * size_y
            verts.append((x, y, origin[2] + height_fn(x, y)))
    for j in range(ny):
        for i in range(nx):
            a = j * (nx + 1) + i
            faces.append((a, a + 1, a + nx + 2, a + nx + 1))
    return mesh_obj(name, verts, faces, mat, smooth)


def prim(kind, loc=(0, 0, 0), scale=(1, 1, 1), rot=(0, 0, 0), mat=None, name=None, smooth=False, **kw):
    ops = {
        'cube': bpy.ops.mesh.primitive_cube_add,
        'ico': bpy.ops.mesh.primitive_ico_sphere_add,
        'uv': bpy.ops.mesh.primitive_uv_sphere_add,
        'cyl': bpy.ops.mesh.primitive_cylinder_add,
        'cone': bpy.ops.mesh.primitive_cone_add,
        'torus': bpy.ops.mesh.primitive_torus_add,
        'plane': bpy.ops.mesh.primitive_plane_add,
    }
    ops[kind](location=loc, rotation=[math.radians(a) for a in rot], **kw)
    o = bpy.context.active_object
    o.scale = scale
    if name:
        o.name = name
    if mat:
        o.data.materials.append(mat)
    if smooth:
        for p in o.data.polygons:
            p.use_smooth = True
    return o


def bevel(o, width=0.05, segments=2):
    m = o.modifiers.new("Bevel", 'BEVEL')
    m.width = width
    m.segments = segments
    try:
        m.limit_method = 'ANGLE'
    except Exception:
        pass
    return m


def subsurf(o, levels=2):
    m = o.modifiers.new("Sub", 'SUBSURF')
    m.levels = levels
    m.render_levels = levels
    return m


def displace(o, strength=0.3, scale=1.0, kind='CLOUDS'):
    t = bpy.data.textures.new("disp", kind)
    try:
        t.noise_scale = scale
    except Exception:
        pass
    m = o.modifiers.new("Disp", 'DISPLACE')
    m.texture = t
    m.strength = strength
    return m


def jitter_mesh(o, amount, seed=0):
    rng = random.Random(seed)
    for v in o.data.vertices:
        v.co += Vector((rng.uniform(-1, 1), rng.uniform(-1, 1), rng.uniform(-1, 1))) * amount


def run_task(name, fn):
    log("==== task:", name)
    t = time.time()
    try:
        fn()
        log("==== ok:", name, "(%.0fs)" % (time.time() - t))
        return True
    except Exception:
        log("==== FAILED:", name)
        traceback.print_exc()
        sys.stdout.flush()
        return False
