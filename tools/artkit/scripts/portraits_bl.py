"""Origin portraits rendered in Cycles (tasks portraits, portrait_<id>).

Each origin is a waist-up hero built from lofted cloth, bevelled plate and
emissive magic, lit like a character-select splash: a warm key, two strong
rims in the origin's colour, haze and drifting motes behind with a shallow
depth of field. Every task writes, under out/portraits/:

  <id>.png          the splash (1200 x 1600)
  <id>_mask.png     the figure alone on transparency (for the alpha cut)
  <id>.json         anchor points (eyes, orb, glint...) for live effects
  hud/<id>_<pose>.png  head-and-shoulders poses for the HUD (536 x 680,
                    transparent): rest, look, fierce, hurt

tools/portraits/pack.py turns these into the game's portrait files.
"""
import os, math, json, random
import bpy, bmesh
from mathutils import Vector, Matrix, Euler
import kit
from kit import log, setin

OUT = os.path.join(kit.OUT, "portraits")
PW, PH = 1200, 1600
HW, HH = 536, 680


# ------------------------------------------------------------------ context

class Ctx:
    def __init__(self, col):
        self.col = col                # origin colour (0..1)
        self.anchors = []             # (kind, Vector, rgb, size)
        self.head = []                # objects that turn with the head
        self.eye_nodes = []           # emission strength sockets of the eyes
        self.magic_nodes = []         # emission sockets that flare when fierce
        self.backdrop = []            # hidden for the mask and HUD passes
        self.rims = []                # rim lights (boosted when fierce)
        self.neck = Vector((0, 0, 1.6))
        self.eye_c = Vector((0, -0.1, 1.72))

    def anchor(self, kind, p, rgb, size=1.0):
        self.anchors.append((kind, Vector(p), rgb, size))


C = None   # the current Ctx


def V(*a):
    return Vector(a)


def lighten(c, k):
    return tuple(min(1.0, x + (1 - x) * k) for x in c)


def darken(c, k):
    return tuple(x * (1 - k) for x in c)


# ------------------------------------------------------------------ materials

def _coord(nt, kind='Object'):
    return nt.nodes.new('ShaderNodeTexCoord').outputs[kind]


def pbr(name, base, rough=0.5, metal=0.0, coat=0.0, sheen=0.0, emit=None, es=0.0, trans=0.0,
        var=0.0, vscale=4.0, bump=0.0, bscale=40.0, weave=0.0, scratch=0.0, fiber=0.0, sss=0.0,
        film=0.0, alpha=1.0):
    """A principled material with optional colour variation, noise bump,
    cloth weave, brushed scratches and fibre streaks."""
    m, nt, out = kit.new_mat(name)
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    setin(p, (*base, 1), 'Base Color')
    setin(p, rough, 'Roughness')
    setin(p, metal, 'Metallic')
    setin(p, coat, 'Coat Weight', 'Clearcoat')
    setin(p, 0.04, 'Coat Roughness', 'Clearcoat Roughness')
    setin(p, sheen, 'Sheen Weight', 'Sheen')
    setin(p, 0.4, 'Sheen Roughness')
    setin(p, trans, 'Transmission Weight', 'Transmission')
    setin(p, sss, 'Subsurface Weight', 'Subsurface')
    setin(p, film, 'Thin Film Thickness')
    if alpha < 1:
        setin(p, alpha, 'Alpha')
    if emit is not None:
        setin(p, (*emit, 1), 'Emission Color', 'Emission')
        s = setin(p, es, 'Emission Strength')
        if s is not None:
            C.magic_nodes.append((s, es))
    co = _coord(nt)
    ao = nt.nodes.new('ShaderNodeAmbientOcclusion')
    try:
        ao.samples = 12
    except Exception:
        pass
    setin(ao, 0.09, 'Distance')
    setin(ao, (*base, 1), 'Color')
    nt.links.new(ao.outputs['Color'], p.inputs['Base Color'])
    if var > 0:
        n = kit.noise_tex(nt, vscale, 6, 0.55, co)
        lo = darken(base, var)
        hi = lighten(base, var * 0.25)
        r = kit.ramp(nt, [(0.3, lo), (0.7, hi)])
        nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
        nt.links.new(r.outputs['Color'], ao.inputs['Color'])
    heights = []
    if bump > 0:
        n = kit.noise_tex(nt, bscale, 10, 0.6, co)
        heights.append((n.outputs['Fac'], bump))
    if weave > 0:
        for axis in ('X', 'Z'):
            w = nt.nodes.new('ShaderNodeTexWave')
            try:
                w.bands_direction = axis
            except Exception:
                pass
            setin(w, 260.0, 'Scale')
            setin(w, 2.0, 'Distortion')
            nt.links.new(co, w.inputs['Vector'])
            heights.append((w.outputs['Fac'], weave))
    if fiber > 0:
        mp = nt.nodes.new('ShaderNodeMapping')
        setin(mp, (40.0, 40.0, 2.5), 'Scale')
        nt.links.new(co, mp.inputs['Vector'])
        n = kit.noise_tex(nt, 6.0, 4, 0.5, mp.outputs['Vector'])
        heights.append((n.outputs['Fac'], fiber))
        r = kit.ramp(nt, [(0.35, darken(base, 0.45)), (0.65, lighten(base, 0.12))])
        nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
        nt.links.new(r.outputs['Color'], ao.inputs['Color'])
    if scratch > 0:
        mp = nt.nodes.new('ShaderNodeMapping')
        setin(mp, (1.0, 60.0, 1.0), 'Scale')
        nt.links.new(co, mp.inputs['Vector'])
        n = kit.noise_tex(nt, 8.0, 6, 0.6, mp.outputs['Vector'])
        r = kit.ramp(nt, [(0.35, (max(0, rough - scratch),) * 3), (0.65, (min(1, rough + scratch),) * 3)])
        nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
        nt.links.new(r.outputs['Color'], p.inputs['Roughness'])
    if heights:
        prev = None
        for h, s in heights:
            b = kit.bump(nt, h, s, 0.02)
            if prev is not None:
                nt.links.new(prev.outputs['Normal'], b.inputs['Normal'])
            prev = b
        nt.links.new(prev.outputs['Normal'], p.inputs['Normal'])
    return m


def glow(name, rgb, strength, eye=False):
    m, nt, out = kit.new_mat(name)
    e = nt.nodes.new('ShaderNodeEmission')
    setin(e, (*rgb, 1), 'Color')
    s = setin(e, strength, 'Strength')
    nt.links.new(e.outputs[0], out.inputs['Surface'])
    (C.eye_nodes if eye else C.magic_nodes).append((s, strength))
    return m


def hollow():
    """The dark inside of a hood or helm: no specular, so it stays a void."""
    m, nt, out = kit.new_mat("hollow")
    d = nt.nodes.new('ShaderNodeBsdfDiffuse')
    setin(d, (0.002, 0.002, 0.003, 1), 'Color')
    nt.links.new(d.outputs[0], out.inputs['Surface'])
    return m


def fire_volume(name, rgb, strength=18.0, scale=5.0):
    """Emissive flame volume: noise-driven emission, brightest at the core."""
    m, nt, out = kit.new_mat(name)
    v = nt.nodes.new('ShaderNodeVolumePrincipled')
    setin(v, 0.0, 'Density')
    setin(v, (*rgb, 1), 'Emission Color')
    co = _coord(nt, 'Generated')
    n = kit.noise_tex(nt, scale, 8, 0.6, co)
    # radial falloff from the bounding-box centre
    sub = nt.nodes.new('ShaderNodeVectorMath')
    sub.operation = 'SUBTRACT'
    nt.links.new(co, sub.inputs[0])
    sub.inputs[1].default_value = (0.5, 0.5, 0.42)
    ln = nt.nodes.new('ShaderNodeVectorMath')
    ln.operation = 'LENGTH'
    nt.links.new(sub.outputs[0], ln.inputs[0])
    fall = nt.nodes.new('ShaderNodeMapRange')
    setin(fall, 0.0, 'From Min')
    setin(fall, 0.5, 'From Max')
    setin(fall, 1.0, 'To Min')
    setin(fall, 0.0, 'To Max')
    nt.links.new(ln.outputs['Value'], fall.inputs['Value'])
    mul = nt.nodes.new('ShaderNodeMath')
    mul.operation = 'MULTIPLY'
    nt.links.new(n.outputs['Fac'], mul.inputs[0])
    nt.links.new(fall.outputs['Result'], mul.inputs[1])
    pw = nt.nodes.new('ShaderNodeMath')
    pw.operation = 'POWER'
    nt.links.new(mul.outputs[0], pw.inputs[0])
    pw.inputs[1].default_value = 2.2
    st = nt.nodes.new('ShaderNodeMath')
    st.operation = 'MULTIPLY'
    nt.links.new(pw.outputs[0], st.inputs[0])
    st.inputs[1].default_value = strength
    C.magic_nodes.append((st.inputs[1], strength))
    nt.links.new(st.outputs[0], kit.sock(v, 'Emission Strength'))
    nt.links.new(v.outputs[0], out.inputs['Volume'])
    return m


# ------------------------------------------------------------------ geometry

def _obj(name, bm, mat, smooth=True, head=False):
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    o = bpy.data.objects.new(name, me)
    bpy.context.scene.collection.objects.link(o)
    if mat is not None:
        if isinstance(mat, (list, tuple)):
            for mm in mat:
                o.data.materials.append(mm)
        else:
            o.data.materials.append(mat)
    for f in o.data.polygons:
        f.use_smooth = smooth
    if head:
        C.head.append(o)
    return o


def folds(o, strength=0.012, across=0.07, along=1.4):
    """Vertical cloth folds: noise stretched along Z, applied as geometry."""
    e = bpy.data.objects.new(o.name + "_fold", None)
    bpy.context.scene.collection.objects.link(e)
    e.scale = (across, across, along)
    d = o.modifiers.new("folds", 'DISPLACE')
    t = bpy.data.textures.new("folds", 'CLOUDS')
    t.noise_scale = 1.0
    try:
        t.noise_depth = 3
    except Exception:
        pass
    d.texture = t
    d.texture_coords = 'OBJECT'
    d.texture_coords_object = e
    d.strength = strength
    d.mid_level = 0.5
    return o


def subsurf(o, lv=2):
    m = o.modifiers.new("sub", 'SUBSURF')
    m.levels = lv
    m.render_levels = lv
    return o


def bevel(o, w=0.01, seg=3):
    b = o.modifiers.new("bev", 'BEVEL')
    b.width = w
    b.segments = seg
    try:
        b.limit_method = 'ANGLE'
    except Exception:
        pass
    return o


def solidify(o, t=0.012, off=-1.0, inner=None):
    s = o.modifiers.new("solid", 'SOLIDIFY')
    s.thickness = t
    s.offset = off
    if inner is not None:
        o.data.materials.append(inner)
        s.material_offset = 1
    return o


def loft(name, rings, mat, n=48, cap_top=False, cap_bot=False, fold=None, sub=1, head=False):
    """Stacks horizontal ellipses: rings = [(z, rx, ry, cx, cy), ...] bottom
    up. fold = (amp, k, twist, fade_top) adds vertical cloth folds whose
    depth fades toward the top."""
    bm = bmesh.new()
    rows = []
    z0, z1 = rings[0][0], rings[-1][0]
    for (z, rx, ry, *c) in rings:
        cx, cy = (c + [0, 0])[:2]
        row = []
        for i in range(n):
            a = 2 * math.pi * i / n
            k = 1.0
            if fold:
                amp, kk, tw, ft = fold
                w = 1 - (z - z0) / max(1e-6, z1 - z0) * ft
                k += amp * w * (math.sin(kk * a + tw * z) * 0.7 + 0.3 * math.sin(kk * 2.3 * a - tw * 1.7 * z))
            row.append(bm.verts.new((cx + math.cos(a) * rx * k, cy + math.sin(a) * ry * k, z)))
        rows.append(row)
    for r0, r1 in zip(rows, rows[1:]):
        for i in range(n):
            j = (i + 1) % n
            bm.faces.new((r0[i], r0[j], r1[j], r1[i]))
    if cap_bot:
        bm.faces.new(list(reversed(rows[0])))
    if cap_top:
        bm.faces.new(rows[-1])
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    o = _obj(name, bm, mat, head=head)
    if sub:
        subsurf(o, sub)
    return o


def tube(name, pts, radii, mat, n=20, caps=True, sub=1, head=False, flat=1.0):
    """A tube along a polyline with per-point radii (parallel-transport
    frames); flat < 1 squashes the section."""
    pts = [Vector(p) for p in pts]
    if isinstance(radii, (int, float)):
        radii = [radii] * len(pts)
    bm = bmesh.new()
    rows = []
    t0 = (pts[1] - pts[0]).normalized()
    up = Vector((0, 0, 1)) if abs(t0.z) < 0.9 else Vector((1, 0, 0))
    nrm = t0.cross(up).normalized()
    prev_t = t0
    for i, p in enumerate(pts):
        if i == 0:
            t = t0
        elif i == len(pts) - 1:
            t = (pts[i] - pts[i - 1]).normalized()
        else:
            t = (pts[i + 1] - pts[i - 1]).normalized()
        # transport the normal
        ax = prev_t.cross(t)
        if ax.length > 1e-6:
            ang = prev_t.angle(t)
            nrm = Matrix.Rotation(ang, 3, ax.normalized()) @ nrm
        prev_t = t
        bn = t.cross(nrm).normalized()
        row = []
        for k in range(n):
            a = 2 * math.pi * k / n
            off = nrm * math.cos(a) * radii[i] + bn * math.sin(a) * radii[i] * flat
            row.append(bm.verts.new(p + off))
        rows.append(row)
    for r0, r1 in zip(rows, rows[1:]):
        for k in range(n):
            j = (k + 1) % n
            bm.faces.new((r0[k], r0[j], r1[j], r1[k]))
    if caps:
        bm.faces.new(list(reversed(rows[0])))
        bm.faces.new(rows[-1])
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    o = _obj(name, bm, mat, head=head)
    if sub:
        subsurf(o, sub)
    return o


def ell(name, loc, r, mat, rot=(0, 0, 0), seg=48, head=False):
    bm = bmesh.new()
    bmesh.ops.create_uvsphere(bm, u_segments=seg, v_segments=seg // 2, radius=1.0)
    for v in bm.verts:
        v.co = Vector((v.co.x * r[0], v.co.y * r[1], v.co.z * r[2]))
    o = _obj(name, bm, mat, head=head)
    o.location = loc
    o.rotation_euler = [math.radians(a) for a in rot]
    return o


def rbox(name, loc, size, mat, bev=0.01, seg=3, rot=(0, 0, 0), head=False):
    bm = bmesh.new()
    bmesh.ops.create_cube(bm, size=1.0)
    for v in bm.verts:
        v.co = Vector((v.co.x * size[0], v.co.y * size[1], v.co.z * size[2]))
    o = _obj(name, bm, mat, smooth=True, head=head)
    o.location = loc
    o.rotation_euler = [math.radians(a) for a in rot]
    if bev > 0:
        bevel(o, bev, seg)
        try:
            o.modifiers[-1].limit_method = 'NONE'
        except Exception:
            pass
    return o


def cyl(name, a, b, r, mat, n=32, bev=0.004, head=False, r2=None):
    """A cylinder (or cone when r2 is given) from point a to point b."""
    a, b = Vector(a), Vector(b)
    d = b - a
    bm = bmesh.new()
    bmesh.ops.create_cone(bm, cap_ends=True, segments=n, radius1=r, radius2=r if r2 is None else r2, depth=d.length)
    o = _obj(name, bm, mat, smooth=True, head=head)
    o.location = (a + b) / 2
    o.rotation_euler = d.to_track_quat('Z', 'Y').to_euler()
    if bev > 0:
        bevel(o, bev, 2)
    return o


def torus(name, loc, R, r, mat, rot=(0, 0, 0), scale=(1, 1, 1), head=False, n=64):
    """A ring: the centre line is an ellipse of radii R*scale[0], R*scale[1];
    the section keeps radius r (its height scaled by scale[2])."""
    bm = bmesh.new()
    rx, ry = R * scale[0], R * scale[1]
    for i in range(n):
        a = 2 * math.pi * i / n
        ca, sa = math.cos(a), math.sin(a)
        for j in range(16):
            b = 2 * math.pi * j / 16
            bm.verts.new((rx * ca + r * math.cos(b) * ca, ry * sa + r * math.cos(b) * sa, r * math.sin(b) * scale[2]))
    bm.verts.ensure_lookup_table()
    for i in range(n):
        for j in range(16):
            i2, j2 = (i + 1) % n, (j + 1) % 16
            bm.faces.new((bm.verts[i * 16 + j], bm.verts[i2 * 16 + j], bm.verts[i2 * 16 + j2], bm.verts[i * 16 + j2]))
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    o = _obj(name, bm, mat, head=head)
    o.location = loc
    o.rotation_euler = [math.radians(a) for a in rot]
    return o


def plate(name, outline, depth, mat, loc=(0, 0, 0), rot=(90, 0, 0), bev=0.006, head=False):
    """An extruded flat shape (outline in its local XY plane)."""
    bm = bmesh.new()
    vs = [bm.verts.new((x, y, 0)) for x, y in outline]
    f = bm.faces.new(vs)
    r = bmesh.ops.extrude_face_region(bm, geom=[f])
    for v in [e for e in r['geom'] if isinstance(e, bmesh.types.BMVert)]:
        v.co.z += depth
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    o = _obj(name, bm, mat, smooth=False, head=head)
    o.location = loc
    o.rotation_euler = [math.radians(a) for a in rot]
    if bev > 0:
        bevel(o, bev, 2)
    return o


def plate3d(name, outline, origin, uvec, vvec, depth, mat, bev=0.004):
    """An extruded shape laid out in the plane of uvec, vvec at origin."""
    origin, uvec, vvec = Vector(origin), Vector(uvec), Vector(vvec)
    nrm = uvec.cross(vvec).normalized()
    bm = bmesh.new()
    vs = [bm.verts.new(origin + uvec * u + vvec * v - nrm * depth / 2) for u, v in outline]
    f = bm.faces.new(vs)
    r = bmesh.ops.extrude_face_region(bm, geom=[f])
    for v in [e for e in r['geom'] if isinstance(e, bmesh.types.BMVert)]:
        v.co += nrm * depth
    bmesh.ops.recalc_face_normals(bm, faces=bm.faces)
    o = _obj(name, bm, mat, smooth=False)
    if bev > 0:
        bevel(o, bev, 2)
    return o


def boolean(o, cutter, op='DIFFERENCE'):
    m = o.modifiers.new("bool", 'BOOLEAN')
    m.operation = op
    m.object = cutter
    cutter.hide_render = True
    cutter.display_type = 'WIRE'
    return o


def bolt_pts(a, b, seed, jag=0.08, steps=10):
    rnd = random.Random(seed)
    a, b = Vector(a), Vector(b)
    out = []
    for i in range(steps + 1):
        t = i / steps
        p = a.lerp(b, t)
        if 0 < i < steps:
            p += Vector((rnd.uniform(-1, 1), rnd.uniform(-1, 1) * 0.4, rnd.uniform(-1, 1))) * jag * math.sin(t * math.pi)
        out.append(p)
    return out


# ------------------------------------------------------------------ figure parts

SH_L, SH_R = V(-0.24, 0.0, 1.44), V(0.24, 0.0, 1.44)   # character's right is screen left


def torso(mat, wide=1.0, chest=1.0, fold=None, low=0.55, neck=True):
    w = wide
    rings = [(low, 0.29 * w, 0.22), (0.85, 0.26 * w, 0.19), (1.05, 0.235 * w, 0.165 * chest),
             (1.22, 0.25 * w, 0.18 * chest), (1.36, 0.265 * w, 0.175 * chest), (1.45, 0.24 * w, 0.15),
             (1.51, 0.16 * w, 0.12), (1.555, 0.075, 0.075)]
    if neck:
        rings.append((1.64, 0.062, 0.062))
    o = loft("torso", rings, mat, fold=fold, cap_top=True, sub=2)
    if fold:
        folds(o, 0.014)
    return o


def shoulder_caps(mat, r=0.1):
    for s, p in (("L", SH_L), ("R", SH_R)):
        ell("shoulder" + s, p + V(0, 0, -0.01), (r, r * 0.95, r * 0.85), mat)


def arm(side, elbow, wrist, mat, r=(0.075, 0.062, 0.05), bell=0.0, cuff=None):
    sh = SH_L if side == "L" else SH_R
    e, w = Vector(elbow), Vector(wrist)
    pts = [sh, sh.lerp(e, 0.5), e, e.lerp(w, 0.5), w]
    rad = [r[0], (r[0] + r[1]) / 2, r[1], (r[1] + r[2]) / 2 + bell * 0.4, r[2] + bell]
    tube("arm" + side, pts, rad, mat, n=24, caps=False)
    if cuff is not None:
        d = (w - e).normalized()
        o = tube("cuff" + side, [w - d * 0.035, w + d * 0.005], [r[2] + bell + 0.01] * 2, cuff, n=24, caps=False, sub=0)
        solidify(o, 0.008, 1.0)
    return w


def hand(name, at, mat, toward=(0, 0, 1), size=1.0, open_=False):
    """A gloved hand: palm, four fingers and a thumb, pointing `toward`."""
    at, d = Vector(at), Vector(toward).normalized()
    q = d.to_track_quat('Z', 'Y')
    ell(name + "palm", at + d * 0.045 * size, (0.042 * size, 0.026 * size, 0.05 * size), mat, rot=[math.degrees(a) for a in q.to_euler()])
    side = (q @ Vector((1, 0, 0)))
    fwd = (q @ Vector((0, -1, 0)))
    for i in range(4):
        x = (i - 1.5) * 0.019 * size
        base = at + d * 0.09 * size + side * x
        if open_:
            continue
        else:
            k1 = base + d * 0.03 * size - fwd * 0.012 * size
            pts = [base, k1, k1 - fwd * 0.03 * size - d * 0.015 * size]
        tube(name + "f%d" % i, pts, [0.0105 * size, 0.0095 * size, 0.0085 * size], mat, n=10)
    if open_:
        ell(name + "mitt", at + d * 0.115 * size + fwd * 0.008 * size, (0.04 * size, 0.017 * size, 0.045 * size), mat,
            rot=[math.degrees(a) for a in q.to_euler()])
    tb = at + d * 0.04 * size - side * 0.04 * size
    tube(name + "thumb", [tb, tb + d * 0.035 * size - side * 0.02 * size - fwd * 0.015 * size,
                          tb + d * 0.06 * size - side * 0.012 * size - fwd * 0.03 * size], [0.013 * size, 0.011 * size, 0.009 * size], mat, n=10)


def head_ball(mat=None):
    ell("head", C.eye_c + V(0, 0.1, -0.01), (0.1, 0.1, 0.125), mat or hollow(), head=True)


def eyes(rgb, strength=24.0, sep=0.034, size=0.011, at=None, shape=(1.5, 1.0, 0.7)):
    at = Vector(at) if at is not None else C.eye_c
    m = glow("eyes", rgb, strength, eye=True)
    for s in (-1, 1):
        p = at + V(s * sep, 0, 0)
        ell("eye", p, (size * shape[0], size * shape[1], size * shape[2]), m, head=True, seg=16)
        C.anchor("eye", p, lighten(rgb, 0.2), 0.5)
    # the eyes light the inside of the hood
    pt = bpy.data.lights.new("eyelight", 'POINT')
    pt.energy = 1.2
    pt.color = rgb
    pt.shadow_soft_size = 0.02
    o = bpy.data.objects.new("eyelight", pt)
    bpy.context.scene.collection.objects.link(o)
    o.location = at + V(0, -0.03, 0)
    C.head.append(o)


def hood(cloth, at=None, r=(0.165, 0.18, 0.2), peak=0.09, drape=1.25, opening=(0.088, 0.2, 0.118), trim=None):
    """A deep cowl: a shell around the head peaked at the back, flared onto
    the shoulders, with the face cut out."""
    at = Vector(at) if at is not None else C.eye_c + V(0, 0.05, 0.01)
    bm = bmesh.new()
    bmesh.ops.create_uvsphere(bm, u_segments=64, v_segments=32, radius=1.0)
    for v in bm.verts:
        x, y, z = v.co
        k = 1.0
        if z < -0.2:
            k = 1 + (drape - 1) * min(1, (-0.2 - z) / 0.8) ** 1.4   # flare onto the shoulders
        x, y = x * k, y * (1 + (k - 1) * 0.6)
        if z > 0 and y > 0:
            z += peak / r[2] * (z * y) ** 1.2 * 2.2                  # the peak falls back
            y += peak / r[1] * (z * y) * 0.8
        v.co = Vector((x * r[0], y * r[1], z * r[2]))
    o = _obj("hood", bm, cloth, head=True)
    o.location = at
    solidify(o, 0.022, 1.0, inner=hollow())
    cut = ell("hoodcut", at + V(0, -r[1] - 0.03, -0.025), opening, trim, head=True)
    boolean(o, cut)
    if trim is not None:
        try:
            o.modifiers["bool"].material_mode = 'TRANSFER'
        except Exception:
            pass
    subsurf(o, 1)
    d = o.modifiers.new("folds", 'DISPLACE')
    t = bpy.data.textures.new("hoodfold", 'CLOUDS')
    t.noise_scale = 0.12
    d.texture = t
    d.strength = 0.014
    d.mid_level = 0.5
    head_ball()
    return o


def capelet(mat, inner=None, low=1.28, amp=0.07):
    """A short shoulder cape that ties the hood into the body."""
    o = loft("capelet", [(low, 0.37, 0.28), (1.4, 0.355, 0.26), (1.48, 0.33, 0.24), (1.53, 0.27, 0.2),
                         (1.58, 0.17, 0.15), (1.63, 0.11, 0.11)], mat, n=64, fold=(amp, 11, 6, 1.0), sub=0)
    solidify(o, 0.012, 1.0, inner=inner)
    subsurf(o, 2)
    folds(o, 0.012, 0.05, 0.6)
    return o


def staff(base, top, wood, metal, r=0.022):
    base, top = Vector(base), Vector(top)
    d = (top - base)
    pts = [base.lerp(top, t) + V(0.01 * math.sin(t * 9), 0, 0) for t in (0, 0.25, 0.5, 0.75, 1.0)]
    tube("staff", pts, [r * 0.9, r, r * 0.95, r, r * 1.05], wood, n=16)
    for t in (0.18, 0.62, 0.9):
        p = base.lerp(top, t)
        cyl("staffring", p - d.normalized() * 0.018, p + d.normalized() * 0.018, r * 1.35, metal)


def cape(mat, z0=1.48, z1=0.3, r0=0.27, r1=0.5, a0=15, a1=165, cy=0.04, nfold=7, amp=0.05, inner=None):
    """A cloak hanging behind the shoulders: an open arc lofted downward."""
    bm = bmesh.new()
    rows = []
    nz, na = 18, 48
    for j in range(nz + 1):
        t = j / nz
        z = z0 + (z1 - z0) * t
        r = r0 + (r1 - r0) * t ** 0.8
        row = []
        for i in range(na + 1):
            a = math.radians(a0 + (a1 - a0) * i / na)
            k = 1 + amp * t * math.sin(nfold * a * 2 + 2.0 * t)
            row.append(bm.verts.new((math.cos(a) * r * k, cy + math.sin(a) * r * k * 0.8, z)))
        rows.append(row)
    for r0_, r1_ in zip(rows, rows[1:]):
        for i in range(na):
            bm.faces.new((r0_[i], r0_[i + 1], r1_[i + 1], r1_[i]))
    o = _obj("cape", bm, mat)
    solidify(o, 0.015, 0.0, inner=inner)
    subsurf(o, 2)
    folds(o, 0.02)
    return o


def ribbon(name, x, w, z0, z1, mat, depth=0.01):
    """A strip hugging the torso front (stoles, tabards): follows the torso
    loft's front surface."""
    prof = [(0.55, 0.22), (0.85, 0.19), (1.05, 0.165), (1.22, 0.18), (1.36, 0.175), (1.45, 0.15), (1.51, 0.12)]

    def front(z):
        for (za, ya), (zb, yb) in zip(prof, prof[1:]):
            if za <= z <= zb:
                return ya + (yb - ya) * (z - za) / (zb - za)
        return prof[0][1] if z < prof[0][0] else prof[-1][1]
    bm = bmesh.new()
    rows = []
    n = 24
    for j in range(n + 1):
        z = z0 + (z1 - z0) * j / n
        y = -front(z) - 0.012
        row = [bm.verts.new((x - w, y + abs(x - w) * 0.25, z)), bm.verts.new((x + w, y + abs(x + w) * 0.25, z))]
        rows.append(row)
    for a, b in zip(rows, rows[1:]):
        bm.faces.new((a[0], a[1], b[1], b[0]))
    o = _obj(name, bm, mat)
    solidify(o, depth, 0.0)
    return o


def belt(z, mat, buckle=None, rx=0.25, ry=0.18, h=0.03):
    o = loft("belt", [(z - h / 2, rx + 0.006, ry + 0.006), (z + h / 2, rx + 0.006, ry + 0.006)], mat, n=64, sub=0)
    solidify(o, 0.008, 1.0)
    bevel(o, 0.003, 2)
    if buckle is not None:
        rbox("buckle", V(0, -ry - 0.012, z), (0.06, 0.012, 0.05), buckle, bev=0.005)


# ------------------------------------------------------------------ scene

def lights(col, key=150.0, rim=520.0, warm=(1.0, 0.92, 0.82)):
    sc = bpy.context.scene

    def area(name, loc, target, size, energy, color, sy=None):
        d = bpy.data.lights.new(name, 'AREA')
        d.energy = energy
        d.color = color
        d.size = size
        if sy is not None:
            d.shape = 'RECTANGLE'
            d.size_y = sy
        o = bpy.data.objects.new(name, d)
        sc.collection.objects.link(o)
        o.location = loc
        o.rotation_euler = (Vector(target) - Vector(loc)).to_track_quat('-Z', 'Y').to_euler()
        return o
    area("key", (-2.3, -1.9, 2.7), (0, 0, 1.55), 0.7, key, warm)
    area("fill", (2.2, -2.2, 1.1), (0, 0, 1.4), 2.5, key * 0.07, (0.7, 0.78, 1.0))
    C.rims.append(area("rimL", (-1.5, 1.5, 2.0), (0, 0, 1.6), 0.4, rim, lighten(col, 0.25), sy=2.4))
    C.rims.append(area("rimR", (1.6, 1.4, 1.9), (0, 0, 1.55), 0.4, rim * 1.25, col, sy=2.4))
    area("top", (0.3, 0.8, 3.4), (0, 0, 1.7), 0.6, key * 0.3, lighten(col, 0.6))


def backdrop(col, seed, haze=0.005, motes=260, mote_col=None, floor=None):
    """Nebula wall, haze volume and drifting motes (all hidden for the mask
    and HUD passes)."""
    rnd = random.Random(seed)
    w = bpy.data.worlds.new("bg")
    bpy.context.scene.world = w
    w.use_nodes = True
    bg = w.node_tree.nodes.get("Background")
    if bg:
        setin(bg, (*darken(col, 0.975), 1), 'Color')
        setin(bg, 1.0, 'Strength')
    # the wall: a dark gradient with a soft nebula in the origin's colour
    m, nt, out = kit.new_mat("wall")
    e = nt.nodes.new('ShaderNodeEmission')
    co = _coord(nt)
    n = kit.noise_tex(nt, 0.45, 12, 0.62, co)
    r = kit.ramp(nt, [(0.35, darken(col, 0.985)), (0.55, darken(col, 0.92)), (0.75, darken(lighten(col, 0.15), 0.7)), (0.95, lighten(col, 0.2))])
    nt.links.new(n.outputs['Fac'], r.inputs['Fac'])
    # a halo of the origin colour behind the head
    ln = nt.nodes.new('ShaderNodeVectorMath')
    ln.operation = 'DISTANCE'
    nt.links.new(co, ln.inputs[0])
    ln.inputs[1].default_value = (0.02, 0.1, 0.0)
    hr = nt.nodes.new('ShaderNodeMapRange')
    setin(hr, 0.0, 'From Min')
    setin(hr, 0.55, 'From Max')
    setin(hr, 1.0, 'To Min')
    setin(hr, 0.0, 'To Max')
    nt.links.new(ln.outputs['Value'], hr.inputs['Value'])
    hp = nt.nodes.new('ShaderNodeMath')
    hp.operation = 'POWER'
    nt.links.new(hr.outputs['Result'], hp.inputs[0])
    hp.inputs[1].default_value = 2.0
    mix = nt.nodes.new('ShaderNodeMix')
    mix.data_type = 'RGBA'
    mix.blend_type = 'ADD'
    nt.links.new(hp.outputs[0], mix.inputs[0])
    nt.links.new(r.outputs['Color'], mix.inputs[6])
    mix.inputs[7].default_value = (*darken(col, 0.45), 1)
    nt.links.new(mix.outputs[2], e.inputs['Color'])
    setin(e, 1.0, 'Strength')
    nt.links.new(e.outputs[0], out.inputs['Surface'])
    bm = bmesh.new()
    bmesh.ops.create_grid(bm, x_segments=1, y_segments=1, size=1.0)
    wall = _obj("wall", bm, m)
    wall.location = (0, 3.5, 1.5)
    wall.rotation_euler = (math.radians(90), 0, 0)
    wall.scale = (5, 4, 1)
    C.backdrop.append(wall)
    # haze
    hm, hnt, hout = kit.new_mat("haze")
    v = hnt.nodes.new('ShaderNodeVolumePrincipled')
    setin(v, (*lighten(col, 0.5), 1), 'Color')
    setin(v, haze, 'Density')
    setin(v, 0.35, 'Anisotropy')
    hnt.links.new(v.outputs[0], hout.inputs['Volume'])
    hz = rbox("haze", (0, 1.0, 1.5), (5.0, 5.0, 4.0), hm, bev=0)
    C.backdrop.append(hz)
    # motes
    mc = mote_col or lighten(col, 0.45)
    mm = kit.mat_emit("mote", mc, 5.0)
    for i in range(motes // 2):
        p = V(rnd.uniform(-1.6, 1.6), rnd.uniform(-1.2, 2.6), rnd.uniform(0.5, 2.9))
        if abs(p.x) < 0.35 and -0.3 < p.y < 0.4:
            continue
        s = rnd.uniform(0.0012, 0.0035) * (1 + max(0, p.y) * 0.5)
        bm = bmesh.new()
        bmesh.ops.create_icosphere(bm, subdivisions=1, radius=s)
        o = _obj("mote", bm, mm)
        o.location = p
        C.backdrop.append(o)


def look_at(o, target):
    o.rotation_euler = (Vector(target) - o.location).to_track_quat('-Z', 'Y').to_euler()


def splash_camera(yaw=14.0, dist=2.6, target=(0, 0, 1.5), lens=58, fstop=2.2):
    a = math.radians(yaw)
    loc = V(math.sin(a) * dist, -math.cos(a) * dist, target[2] + 0.06)
    cam = kit.camera(tuple(loc), (0, 0, 0), lens=lens)
    look_at(cam, target)
    cam.data.sensor_fit = 'VERTICAL'
    cam.data.sensor_height = 34.0
    cam.data.dof.use_dof = True
    cam.data.dof.focus_distance = (C.eye_c - loc).length
    cam.data.dof.aperture_fstop = fstop
    return cam


def hud_camera(yaw=12.0, dist=1.9, lens=72):
    t = C.eye_c + V(0, 0, -0.07)
    a = math.radians(yaw)
    loc = V(math.sin(a) * dist, -math.cos(a) * dist, t.z + 0.03)
    cam = kit.camera(tuple(loc), (0, 0, 0), lens=lens)
    look_at(cam, t)
    cam.data.sensor_fit = 'VERTICAL'
    cam.data.sensor_height = 24.0
    return cam


def _project(cam, p):
    from bpy_extras.object_utils import world_to_camera_view
    v = world_to_camera_view(bpy.context.scene, cam, p)
    return v.x, 1 - v.y


def _head_rig():
    e = bpy.data.objects.new("headrig", None)
    bpy.context.scene.collection.objects.link(e)
    e.location = C.neck
    bpy.context.view_layer.update()
    for o in C.head:
        mw = o.matrix_world.copy()
        o.parent = e
        o.matrix_parent_inverse = e.matrix_world.inverted()
        o.matrix_world = mw
    return e


def _scale_glow(eye_k, magic_k, rim_k):
    for s, v in C.eye_nodes:
        s.default_value = v * eye_k
    for s, v in C.magic_nodes:
        s.default_value = v * magic_k
    for o in C.rims:
        o.data.energy = o["e0"] * rim_k


def render_origin(oid, build, col, seed=1):
    global C
    kit.reset()
    C = Ctx(col)
    kit.setup_render(PW, PH, samples=512)
    sc = bpy.context.scene
    try:
        sc.view_settings.look = 'AgX - Punchy'
    except Exception:
        pass
    try:
        sc.cycles.volume_bounces = 1
        sc.cycles.transparent_max_bounces = 16
    except Exception:
        pass
    lights(col)
    build()
    backdrop(col, seed)
    for o in C.rims:
        o["e0"] = o.data.energy
    rig = _head_rig()
    os.makedirs(os.path.join(OUT, "hud"), exist_ok=True)

    # the splash
    cam = splash_camera()
    sc.camera = cam
    bpy.context.view_layer.update()
    kit.render_to(os.path.join(OUT, oid + ".png"))
    anchors = []
    for kind, p, rgb, size in C.anchors:
        x, y = _project(cam, p)
        anchors.append({"kind": kind, "x": x, "y": y, "r": rgb[0], "g": rgb[1], "b": rgb[2], "size": size})
    with open(os.path.join(OUT, oid + ".json"), "w") as f:
        json.dump(anchors, f, indent=1)

    # the figure alone, for the alpha cut
    for o in C.backdrop:
        o.hide_render = True
    sc.render.film_transparent = True
    sc.cycles.samples = 48
    kit.render_to(os.path.join(OUT, oid + "_mask.png"))

    # HUD poses
    sc.render.resolution_x, sc.render.resolution_y = HW, HH
    sc.cycles.samples = 256
    hc = hud_camera()
    sc.camera = hc
    poses = {
        "rest": ((0, 0, 0), 1.0, 1.0, 1.0),
        "look": ((0, 0, 24), 1.0, 1.0, 1.0),
        "fierce": ((9, 0, 4), 2.6, 1.8, 1.6),
        "hurt": ((-12, 10, -14), 0.35, 0.6, 0.7),
    }
    for name, (rot, ek, mk, rk) in poses.items():
        rig.rotation_euler = [math.radians(a) for a in rot]
        _scale_glow(ek, mk, rk)
        kit.render_to(os.path.join(OUT, "hud", "%s_%s.png" % (oid, name)))
    rig.rotation_euler = (0, 0, 0)
    _scale_glow(1, 1, 1)


# ------------------------------------------------------------------ the origins

GOLD = (1.0, 0.74, 0.32)


def m_gold(name="gold"):
    return pbr(name, GOLD, rough=0.22, metal=1.0, scratch=0.08, bump=0.02, bscale=80)


def m_leather(base=(0.16, 0.085, 0.045), name="leather"):
    return pbr(name, base, rough=0.55, var=0.35, vscale=6, bump=0.12, bscale=60, coat=0.15)


def m_steel(base=(0.62, 0.64, 0.68), name="steel", rough=0.28):
    return pbr(name, base, rough=rough, metal=1.0, scratch=0.12, var=0.15, vscale=3)


def m_cloth(base, name="cloth", sheen=0.6):
    return pbr(name, base, rough=0.72, sheen=sheen * 0.3, var=0.3, vscale=3, weave=0.05, bump=0.05, bscale=20)


def b_arcane():
    col = (0.58, 0.42, 1.0)
    robe = m_cloth((0.13, 0.05, 0.3), "robe")
    gold = m_gold()
    glove = m_leather((0.07, 0.04, 0.09), "glove")
    torso(robe, fold=(0.04, 9, 3, 0.9), neck=False)
    capelet(robe, inner=m_cloth((0.05, 0.02, 0.1), "robein"))
    hood(robe, trim=gold)
    eyes((0.65, 0.85, 1.0), 30)
    # gold-embroidered stole and belt
    st = pbr("stole", (0.55, 0.32, 0.08), rough=0.35, metal=0.7, emit=(1.0, 0.7, 0.3), es=0.15, bump=0.1, bscale=90)
    for x in (-0.075, 0.075):
        ribbon("stole", x, 0.022, 0.55, 1.3, st)
    belt(1.0, gold, gold, rx=0.245, ry=0.17)
    # staff in the right hand (screen left), crescent and crystal on top
    w = arm("L", (-0.36, -0.02, 1.13), (-0.42, -0.1, 1.24), robe, bell=0.05, cuff=gold)
    staff((-0.43, -0.12, 0.45), (-0.43, -0.12, 2.25), pbr("wood", (0.12, 0.06, 0.03), rough=0.5, fiber=0.3, coat=0.3), gold)
    hand("handL", w, glove, toward=(0.0, -0.3, 1.0))
    cres = torus("crescent", V(-0.43, -0.12, 2.32), 0.1, 0.014, gold, rot=(90, 0, 0))
    cut = rbox("crescut", V(-0.43, -0.12, 2.43), (0.12, 0.2, 0.08), None, bev=0)
    boolean(cres, cut)
    cm = pbr("crystal", (0.75, 0.6, 1.0), rough=0.02, trans=0.85, emit=(0.6, 0.4, 1.0), es=6.0)
    bm = bmesh.new()
    bmesh.ops.create_icosphere(bm, subdivisions=1, radius=0.05)
    o = _obj("crystal", bm, cm, smooth=False)
    o.location = (-0.43, -0.12, 2.33)
    o.scale = (0.8, 0.8, 1.5)
    C.anchor("core", (-0.43, -0.12, 2.33), (0.7, 0.5, 1.0), 0.6)
    # orb hovering over the left palm (screen right)
    w = arm("R", (0.37, -0.06, 1.12), (0.38, -0.24, 1.22), robe, bell=0.05, cuff=gold)
    hand("handR", w, glove, toward=(0.15, -0.6, 0.6), open_=True)
    orb = V(0.4, -0.33, 1.34)
    ell("orb", orb, (0.055,) * 3, glow("orb", (0.8, 0.65, 1.0), 14.0))
    shell = pbr("orbshell", (0.7, 0.55, 1.0), rough=0.0, trans=1.0, emit=(0.45, 0.25, 1.0), es=0.6)
    ell("orbshell", orb, (0.1,) * 3, shell)
    rm = glow("ring", (0.7, 0.5, 1.0), 5.0)
    torus("ring1", orb, 0.15, 0.004, rm, rot=(70, 10, 0))
    torus("ring2", orb, 0.18, 0.003, rm, rot=(60, -35, 30))
    pt = bpy.data.lights.new("orblight", 'POINT')
    pt.energy = 25
    pt.color = (0.7, 0.5, 1.0)
    pt.shadow_soft_size = 0.08
    lo = bpy.data.objects.new("orblight", pt)
    bpy.context.scene.collection.objects.link(lo)
    lo.location = orb
    C.anchor("orb", orb, (0.65, 0.45, 1.0), 0.9)


def b_berserker():
    col = (1.0, 0.27, 0.18)
    jerkin = m_leather((0.2, 0.09, 0.04), "jerkin")
    iron = m_steel((0.35, 0.34, 0.33), "iron", rough=0.38)
    fur = pbr("fur", (0.32, 0.25, 0.18), rough=0.9, fiber=0.5, bump=0.4, bscale=30, var=0.4)
    bone = pbr("bone", (0.86, 0.8, 0.66), rough=0.4, var=0.25, vscale=8, bump=0.1, bscale=40, coat=0.2)
    hair = pbr("beard", (0.42, 0.13, 0.04), rough=0.55, fiber=0.6, sheen=0.4, coat=0.2)
    torso(jerkin, wide=1.12, chest=1.1)
    # studded straps across the chest
    for s in (-1, 1):
        tube("strap", [V(s * 0.2, -0.15, 1.45), V(s * 0.02, -0.21, 1.22), V(-s * 0.16, -0.19, 0.98)], 0.022, m_leather((0.09, 0.05, 0.03), "strap"), flat=0.3, n=12)
    for i in range(7):
        t = i / 6
        ell("stud", V(-0.2 + 0.36 * t, -0.215 + 0.03 * math.sin(t * 3.1), 1.43 - 0.45 * t), (0.012,) * 3, iron)
    # fur mantle over the shoulders
    m = loft("mantle", [(1.33, 0.36, 0.25), (1.42, 0.38, 0.26), (1.5, 0.3, 0.22), (1.56, 0.16, 0.15)], fur,
             fold=(0.12, 13, 9, 0.0), n=64, sub=2)
    disp = m.modifiers.new("d", 'DISPLACE')
    t = bpy.data.textures.new("furn", 'CLOUDS')
    t.noise_scale = 0.05
    disp.texture = t
    disp.strength = 0.03
    # iron pauldron with spikes on the left shoulder (screen right)
    pa = ell("pauldron", SH_R + V(0.03, 0, 0.04), (0.14, 0.13, 0.09), iron)
    for k in range(3):
        a = math.radians(-40 + 40 * k)
        b = SH_R + V(0.03 + 0.09 * math.sin(a), -0.02, 0.1 + 0.03 * math.cos(a))
        cyl("spike", b, b + V(0.07 * math.sin(a), 0, 0.09), 0.02, iron, r2=0.002)
    # helm, horns, beard
    head_ball()
    hc = C.eye_c + V(0, 0.06, 0.06)
    helm = ell("helm", hc, (0.135, 0.145, 0.13), iron, head=True)
    cut = rbox("helmcut", hc + V(0, 0, -0.13), (0.4, 0.4, 0.22), None, bev=0, head=True)
    boolean(helm, cut)
    solidify(helm, 0.01, 1.0)
    torus("helmrim", hc + V(0, 0, -0.005), 1.0, 0.012, iron, scale=(0.14, 0.15, 1.0), head=True)
    rbox("nasal", C.eye_c + V(0, -0.115, 0.0), (0.016, 0.012, 0.09), iron, bev=0.004, head=True)
    for s in (-1, 1):
        base = hc + V(s * 0.12, 0, 0.03)
        pts = [base, base + V(s * 0.1, 0.0, 0.03), base + V(s * 0.18, -0.02, 0.12), base + V(s * 0.2, -0.05, 0.25), base + V(s * 0.17, -0.07, 0.33)]
        tube("horn", pts, [0.042, 0.036, 0.026, 0.014, 0.003], bone, n=20, head=True)
    ell("beardmass", C.eye_c + V(0, -0.07, -0.13), (0.1, 0.07, 0.11), hair, head=True)
    for i in range(5):
        x = (i - 2) * 0.034
        top = C.eye_c + V(x, -0.11 - 0.01 * (2 - abs(i - 2)), -0.15)
        pts = [top + V(0, -0.005 * k, -0.07 * k) + V(0.006 * math.sin(k * 2.2 + i), 0, 0) for k in range(5)]
        tube("braid", pts, [0.022, 0.02, 0.018, 0.015, 0.011], hair, n=14, head=True)
        cyl("bead", pts[3] + V(0, 0, 0.012), pts[3] - V(0, 0, 0.012), 0.017, m_gold("ring"), head=True)
    eyes((1.0, 0.35, 0.08), 28, at=C.eye_c + V(0, -0.025, 0))
    # arms in leather with iron bracers
    w = arm("L", (-0.38, -0.08, 1.12), (-0.25, -0.24, 1.3), jerkin, r=(0.085, 0.07, 0.058))
    cyl("bracerL", V(-0.36, -0.12, 1.16), V(-0.27, -0.22, 1.27), 0.064, iron)
    hand("handL", w, m_leather((0.1, 0.06, 0.03), "glove"), toward=(0.3, -0.5, 0.8))
    w = arm("R", (0.39, -0.02, 1.08), (0.36, -0.2, 1.0), jerkin, r=(0.085, 0.07, 0.058))
    hand("handR", w, m_leather((0.1, 0.06, 0.03), "glove2"), toward=(0, -0.4, -1))
    # the great axe over the right shoulder (screen left)
    haft_a, haft_b = V(-0.1, -0.27, 0.82), V(-0.44, 0.04, 1.98)
    tube("haft", [haft_a, haft_a.lerp(haft_b, 0.5), haft_b], 0.022, pbr("haftwood", (0.16, 0.08, 0.04), rough=0.55, fiber=0.35), n=14)
    d = (haft_b - haft_a).normalized()
    for t in (0.3, 0.35, 0.4):
        p = haft_a.lerp(haft_b, t)
        cyl("wrap", p - d * 0.012, p + d * 0.012, 0.026, m_leather((0.08, 0.04, 0.02), "wrap"))
    hd = haft_a.lerp(haft_b, 0.86)
    out = d.cross(V(0, -1, 0)).normalized()          # broad side toward the camera
    axe_m = m_steel((0.6, 0.61, 0.64), "axe", rough=0.2)
    blade = [(0.02, -0.06), (0.1, -0.1), (0.2, -0.19), (0.25, -0.08), (0.27, 0.0), (0.25, 0.08), (0.2, 0.19), (0.1, 0.1), (0.02, 0.06)]
    for sgn in (1, -1):
        plate3d("blade", [(sgn * u, v) for u, v in blade], hd, out, d, 0.016, axe_m)
    cyl("axeeye", hd - d * 0.07, hd + d * 0.07, 0.034, axe_m)
    rune = glow("rune", (1.0, 0.25, 0.05), 8.0)
    for sgn in (1, -1):
        for k in range(3):
            p = hd + out * sgn * (0.09 + 0.045 * k) + V(0, -0.012, 0)
            ell("rune", p, (0.011, 0.004, 0.011), rune, seg=12)
    C.anchor("glint", hd - out * 0.26 + V(0, -0.02, 0), (1, 0.9, 0.8), 0.8)
    C.anchor("ember", V(0.05, -0.3, 0.95), (1, 0.35, 0.1), 0.5)


def b_guardian():
    col = (1.0, 0.8, 0.45)
    steel = m_steel((0.72, 0.74, 0.78), "plate", rough=0.2)
    gold = m_gold()
    blue = m_cloth((0.03, 0.07, 0.3), "tabard", sheen=0.5)
    torso(steel, wide=1.1, chest=1.15)
    rbox("ridge", V(0, -0.2, 1.24), (0.02, 0.03, 0.34), steel, bev=0.01, rot=(-8, 0, 0))
    torus("gorget", V(0, 0, 1.53), 1.0, 0.016, gold, scale=(0.15, 0.13, 1.0))
    for k in range(3):
        z = 0.95 - k * 0.09
        loft("fauld%d" % k, [(z - 0.07, 0.29 + k * 0.02, 0.22 + k * 0.015), (z + 0.02, 0.255 + k * 0.02, 0.19 + k * 0.012)], steel, cap_top=False, sub=1)
        torus("faulrim", V(0, 0, z - 0.07), 1.0, 0.008, gold, scale=(0.29 + k * 0.02, 0.22 + k * 0.015, 1.0))
    ribbon("tabard", 0, 0.1, 0.4, 1.12, blue, depth=0.008)
    # layered pauldrons
    for s, p in ((-1, SH_L), (1, SH_R)):
        for k in range(3):
            c = p + V(s * (0.03 + 0.02 * k), 0, 0.05 - 0.05 * k)
            o = ell("paul", c, (0.15 - 0.01 * k, 0.15 - 0.01 * k, 0.1), steel)
            cut = rbox("paulcut", c + V(0, 0, -0.12), (0.4, 0.4, 0.2), None, bev=0)
            boolean(o, cut)
            solidify(o, 0.01, 1.0)
            torus("paulrim", c + V(0, 0, -0.02), 1.0, 0.007, gold, scale=(0.15 - 0.01 * k, 0.15 - 0.01 * k, 1.0))
    # great helm
    hc = C.eye_c + V(0, 0.04, 0.0)
    helm = loft("helm", [(hc.z - 0.18, 0.12, 0.13), (hc.z - 0.05, 0.125, 0.135), (hc.z + 0.08, 0.12, 0.13), (hc.z + 0.14, 0.09, 0.1), (hc.z + 0.17, 0.03, 0.03)],
                steel, cap_top=True, head=True, sub=2)
    helm.location = (0, hc.y, 0)
    slit = rbox("slit", V(0, hc.y - 0.12, C.eye_c.z), (0.16, 0.08, 0.016), None, bev=0, head=True)
    boolean(helm, slit)
    solidify(helm, 0.008, 1.0, inner=hollow())
    head_ball()
    rbox("crossv", V(0, hc.y - 0.137, C.eye_c.z - 0.06), (0.018, 0.012, 0.11), gold, bev=0.004, head=True)
    rbox("brow", V(0, hc.y - 0.131, C.eye_c.z + 0.03), (0.17, 0.012, 0.014), gold, bev=0.004, head=True)
    crim = pbr("plume", (0.32, 0.01, 0.02), rough=0.45, sheen=1.0, coat=0.3, fiber=0.5)
    rnd = random.Random(11)
    for i in range(46):
        a = rnd.uniform(-0.035, 0.035)
        b = V(a, hc.y + rnd.uniform(-0.05, 0.03), hc.z + 0.165)
        L = rnd.uniform(0.85, 1.15)
        pts = [b, b + V(a * 0.3, 0.05 * L, 0.09 * L), b + V(a * 0.8, 0.17 * L, 0.11 * L), b + V(a * 1.5, 0.28 * L, 0.02 * L), b + V(a * 1.9, 0.33 * L, -0.12 * L)]
        tube("plume", pts, [0.009, 0.012, 0.011, 0.008, 0.002], crim, n=8, head=True)
    eyes((1.0, 0.85, 0.5), 12, sep=0.04, size=0.009, at=V(0, hc.y - 0.1, C.eye_c.z))
    cape(blue, inner=m_cloth((0.25, 0.03, 0.05), "capein"))
    # sword hilt in the right hand (screen left)
    w = arm("L", (-0.4, -0.04, 1.08), (-0.34, -0.18, 1.0), steel, r=(0.08, 0.068, 0.058))
    hand("handL", w, steel, toward=(0, -0.3, -1))
    hilt = V(-0.33, -0.25, 0.95)
    cyl("grip", hilt + V(0, 0, -0.07), hilt + V(0, 0, 0.1), 0.017, m_leather((0.1, 0.05, 0.02), "grip"))
    rbox("guard", hilt + V(0, 0, -0.08), (0.2, 0.025, 0.025), gold, bev=0.008)
    ell("pommel", hilt + V(0, 0, 0.12), (0.03,) * 3, gold)
    ell("pomgem", hilt + V(0, -0.025, 0.12), (0.014,) * 3, glow("gem", (0.3, 0.6, 1.0), 6.0))
    C.anchor("glint", hilt + V(0, -0.03, 0.12), (0.8, 0.9, 1.0), 0.6)
    # kite shield on the left arm (screen right)
    w = arm("R", (0.4, -0.08, 1.12), (0.28, -0.3, 1.18), steel, r=(0.08, 0.068, 0.058))
    kite = [(0.0, 0.34), (0.2, 0.26), (0.22, 0.05), (0.15, -0.2), (0.0, -0.4), (-0.15, -0.2), (-0.22, 0.05), (-0.2, 0.26)]
    sc_ = V(0.3, -0.42, 1.12)
    sh = plate("shield", kite, 0.035, pbr("enamel", (0.03, 0.08, 0.38), rough=0.25, coat=1.0, var=0.2), loc=sc_ + V(0, 0.02, 0), rot=(90, 0, -18))
    rim = [Vector(sc_) + Matrix.Rotation(math.radians(-18), 3, 'Z') @ V(x, -0.006, y) for x, y in kite + [kite[0]]]
    tube("shieldrim", rim, 0.016, gold, n=10, caps=False, sub=0)
    sun = []
    for i in range(32):
        a = 2 * math.pi * i / 32
        rr = 0.11 if i % 2 == 0 else 0.05
        sun.append((math.cos(a) * rr, math.sin(a) * rr))
    plate("sun", sun, 0.012, pbr("sunm", GOLD, rough=0.2, metal=1.0, emit=(1.0, 0.75, 0.3), es=3.0), loc=sc_ + V(0, -0.025, 0.05), rot=(90, 0, -18))
    ell("suncore", sc_ + V(0.005, -0.04, 0.05), (0.035, 0.015, 0.035), glow("suncore", (1.0, 0.9, 0.6), 30.0), rot=(0, 0, -18))
    C.anchor("core", sc_ + V(0, -0.04, 0.05), (1, 0.85, 0.5), 0.8)
    C.anchor("glint", sc_ + V(-0.15, -0.04, 0.3), (1, 0.95, 0.8), 0.7)


def b_engineer():
    col = (1.0, 0.6, 0.22)
    coat = m_leather((0.22, 0.12, 0.06), "coat")
    canvas = m_cloth((0.36, 0.3, 0.2), "canvas", sheen=0.3)
    brass = pbr("brass", (0.9, 0.62, 0.3), rough=0.25, metal=1.0, scratch=0.1, var=0.25)
    steel = m_steel((0.45, 0.47, 0.5), "steel")
    rubber = pbr("rubber", (0.03, 0.03, 0.03), rough=0.7, bump=0.15, bscale=50)
    torso(coat, wide=1.08, fold=(0.025, 7, 4, 0.8), neck=False)
    loft("collar", [(1.46, 0.22, 0.17), (1.56, 0.19, 0.16), (1.62, 0.17, 0.15)], coat, n=48)
    hood(canvas, r=(0.16, 0.175, 0.19), peak=0.05, drape=1.3, opening=(0.11, 0.2, 0.14), trim=m_leather((0.12, 0.07, 0.03), "hoodtrim"))
    # goggles and respirator
    gz = C.eye_c + V(0, -0.105, 0.005)
    lens_m = pbr("lens", (1.0, 0.6, 0.2), rough=0.05, trans=0.5, emit=(1.0, 0.55, 0.15), es=6.0)
    for s in (-1, 1):
        p = gz + V(s * 0.045, 0, 0)
        torus("gogrim", p, 0.03, 0.009, brass, rot=(90, 0, 0), head=True)
        cyl("goglens", p + V(0, 0.005, 0), p + V(0, -0.006, 0), 0.028, lens_m, head=True)
        C.anchor("eye", p, (1.0, 0.6, 0.2), 0.6)
    rbox("gogbridge", gz, (0.03, 0.012, 0.012), brass, bev=0.003, head=True)
    tube("gogstrap", [gz + V(-0.07, 0.02, 0), gz + V(-0.12, 0.1, 0.0), gz + V(-0.13, 0.2, 0.0)], 0.012, rubber, flat=0.3, n=10, head=True)
    tube("gogstrap2", [gz + V(0.07, 0.02, 0), gz + V(0.12, 0.1, 0.0), gz + V(0.13, 0.2, 0.0)], 0.012, rubber, flat=0.3, n=10, head=True)
    mask = C.eye_c + V(0, -0.1, -0.08)
    ell("mask", mask, (0.07, 0.05, 0.055), rubber, head=True)
    for s in (-1, 1):
        a = mask + V(s * 0.05, -0.02, -0.02)
        cyl("filter", a, a + V(s * 0.04, -0.03, -0.02), 0.026, steel, head=True)
        torus("filtrim", a + V(s * 0.04, -0.03, -0.02), 0.026, 0.004, brass, rot=(60, 0, s * 50), head=True)
    pt = bpy.data.lights.new("goglight", 'POINT')
    pt.energy = 2.0
    pt.color = (1.0, 0.55, 0.2)
    lo = bpy.data.objects.new("goglight", pt)
    bpy.context.scene.collection.objects.link(lo)
    lo.location = gz + V(0, -0.04, 0)
    C.head.append(lo)
    # apron straps, tool belt, pouches
    for s in (-1, 1):
        ribbon("apron", s * 0.09, 0.022, 0.6, 1.5, canvas, depth=0.006)
    belt(0.98, m_leather((0.1, 0.05, 0.02), "belt"), brass, rx=0.26, ry=0.19)
    for x in (-0.18, -0.08, 0.14):
        rbox("pouch", V(x, -0.2, 0.93), (0.07, 0.045, 0.08), m_leather((0.14, 0.08, 0.04), "pouch"), bev=0.012)
    # the mechanical left arm (screen right) with a glowing core, fist raised
    sh = SH_R
    el, wr = V(0.42, -0.1, 1.18), V(0.36, -0.28, 1.4)
    ell("mechshoulder", sh + V(0.03, 0, 0.02), (0.13, 0.12, 0.11), brass)
    cyl("upper", sh + V(0.02, 0, -0.02), el, 0.06, steel)
    ell("elbowball", el, (0.065,) * 3, brass)
    cyl("fore", el, wr, 0.07, steel, r2=0.06)
    for t in (0.25, 0.75):
        p = el.lerp(wr, t)
        d = (wr - el).normalized()
        cyl("forering", p - d * 0.012, p + d * 0.012, 0.078, brass)
    core = el.lerp(wr, 0.5) + V(0, -0.07, 0)
    ell("core", core, (0.03, 0.02, 0.03), glow("core", (1.0, 0.55, 0.12), 25.0))
    torus("corering", core, 0.034, 0.008, brass, rot=(90, 0, 0))
    C.anchor("core", core, (1.0, 0.55, 0.15), 0.7)
    for s in (-1, 1):
        cyl("piston", sh + V(0.06, s * 0.05, -0.06), el + V(0.03, s * 0.05, 0.02), 0.012, pbr("chrome", (0.9, 0.9, 0.92), rough=0.08, metal=1.0))
    gear_c = el + V(0.07, 0.0, 0)
    torus("gear", gear_c, 0.05, 0.012, brass, rot=(0, 90, 0))
    for k in range(10):
        a = 2 * math.pi * k / 10
        rbox("tooth", gear_c + V(0, math.cos(a) * 0.065, math.sin(a) * 0.065), (0.016, 0.014, 0.014), brass, bev=0.002, rot=(math.degrees(a), 0, 0))
    hand("fist", wr, steel, toward=(-0.2, -0.5, 1), size=1.25)
    # wrench in the right hand (screen left) resting on the shoulder
    w = arm("L", (-0.38, -0.12, 1.1), (-0.27, -0.26, 1.24), coat, r=(0.08, 0.068, 0.056))
    hand("handL", w, m_leather((0.08, 0.05, 0.03), "glove"), toward=(0.3, -0.5, 0.8))
    wa, wb = V(-0.2, -0.33, 1.2), V(-0.42, -0.05, 1.82)
    cyl("wrench", wa, wb, 0.02, steel)
    wh = wb + (wb - wa).normalized() * 0.04
    o = torus("wrenchhead", wh, 0.055, 0.022, steel, rot=(90, 0, -38))
    cut = rbox("wrenchcut", wh + (wb - wa).normalized() * 0.06, (0.05, 0.2, 0.08), None, bev=0, rot=(0, -38, 0))
    boolean(o, cut)
    # pack with pipes over the shoulder
    rbox("pack", V(0, 0.22, 1.25), (0.32, 0.14, 0.38), steel, bev=0.03)
    for s in (-1, 1):
        tube("pipe", [V(s * 0.12, 0.27, 1.4), V(s * 0.14, 0.27, 1.62), V(s * 0.16, 0.2, 1.72)], 0.022, brass, n=14)
    C.anchor("steam", V(0.16, 0.2, 1.75), (0.9, 0.85, 0.8), 0.8)


def b_lucky():
    col = (0.28, 0.85, 0.58)
    coat = m_cloth((0.02, 0.16, 0.09), "velvet", sheen=1.0)
    vest = pbr("vest", (0.32, 0.05, 0.06), rough=0.45, sheen=0.6, weave=0.04, var=0.2)
    gold = m_gold()
    silk = pbr("silk", (0.9, 0.88, 0.82), rough=0.3, sheen=0.8)
    torso(vest, wide=1.0)
    # long coat: an open shell around the torso, lapels wide
    loft("coat", [(0.5, 0.31, 0.24), (0.9, 0.28, 0.21), (1.2, 0.27, 0.2), (1.42, 0.26, 0.17), (1.5, 0.18, 0.14), (1.58, 0.12, 0.11)],
         coat, fold=(0.03, 7, 3, 0.9), cap_top=False)
    cut = rbox("coatcut", V(0, -0.3, 1.0), (0.2, 0.2, 0.95), None, bev=0)
    boolean(bpy.data.objects["coat"], cut)
    solidify(bpy.data.objects["coat"], 0.012, 1.0, inner=pbr("lining", (0.4, 0.3, 0.08), rough=0.3, sheen=0.5))
    for s in (-1, 1):
        plate("lapel", [(0, 0), (s * 0.09, 0.0), (s * 0.12, -0.25), (s * 0.04, -0.3)], 0.01, coat, loc=V(s * 0.04, -0.205, 1.5), rot=(90, 0, 0))
    for z in (1.18, 1.08, 0.98):
        ell("button", V(-0.06, -0.205, z), (0.012, 0.008, 0.012), gold)
        ell("button", V(0.06, -0.205, z), (0.012, 0.008, 0.012), gold)
    ell("cravat", V(0, -0.16, 1.47), (0.035, 0.022, 0.05), silk)
    ell("pin", V(0, -0.19, 1.49), (0.012,) * 3, glow("emerald", (0.2, 1.0, 0.5), 8.0))
    shoulder_caps(coat, 0.1)
    # face in the brim's shadow
    head_ball()
    hb = C.eye_c + V(0, 0.03, 0.09)
    loft("brim", [(hb.z - 0.004, 0.27, 0.25), (hb.z + 0.006, 0.26, 0.24), (hb.z + 0.012, 0.15, 0.15)], coat, cap_bot=True, head=True, sub=2).location = (0, hb.y, 0)
    loft("crown", [(hb.z, 0.13, 0.12), (hb.z + 0.12, 0.125, 0.115), (hb.z + 0.15, 0.11, 0.1)], coat, cap_top=True, head=True, sub=2).location = (0, hb.y, 0)
    torus("band", V(0, hb.y, hb.z + 0.03), 1.0, 0.016, pbr("hatband", (0.05, 0.03, 0.02), rough=0.4), scale=(0.132, 0.122, 1.2), head=True)
    rbox("bandbuckle", V(0.0, hb.y - 0.125, hb.z + 0.03), (0.03, 0.006, 0.026), gold, bev=0.003, head=True)
    C.anchor("glint", V(0.0, hb.y - 0.13, hb.z + 0.03), (1, 0.95, 0.7), 0.6)
    tube("feather", [V(0.1, hb.y + 0.02, hb.z + 0.03), V(0.16, hb.y + 0.06, hb.z + 0.12), V(0.2, hb.y + 0.1, hb.z + 0.2)], [0.012, 0.018, 0.004],
         pbr("feather", (0.1, 0.5, 0.3), rough=0.5, sheen=1.0, fiber=0.6, film=300), flat=0.25, n=12, head=True)
    eyes((0.3, 1.0, 0.6), 26, sep=0.032)
    # card fan in the right hand (screen left)
    w = arm("L", (-0.38, -0.08, 1.14), (-0.33, -0.27, 1.3), coat, r=(0.08, 0.066, 0.055), cuff=silk)
    hand("handL", w, pbr("glove", (0.9, 0.9, 0.86), rough=0.5), toward=(0.2, -0.4, 0.9))
    face_m = pbr("cardface", (0.95, 0.93, 0.88), rough=0.35, coat=0.6)
    pip_r = glow("pipr", (1.0, 0.15, 0.2), 4.0)
    pip_k = pbr("pipk", (0.02, 0.02, 0.02), rough=0.3)
    fan_c = V(-0.33, -0.33, 1.36)
    for i in range(5):
        a = -36 + 18 * i
        rot = (90 - 8, a, 18)
        R = Euler([math.radians(x) for x in rot]).to_matrix()
        off = R @ V(0, 0.075, 0)
        rbox("card", fan_c + off + V(0, -0.003 * i, 0), (0.065, 0.095, 0.0025), face_m, bev=0.004, rot=rot)
        ell("pip", fan_c + off * 1.25 + V(0, -0.003 * i - 0.004, 0), (0.012, 0.012, 0.003), pip_r if i % 2 == 0 else pip_k, rot=rot)
    # coin flipping over the left palm (screen right)
    w = arm("R", (0.38, -0.06, 1.1), (0.36, -0.27, 1.2), coat, r=(0.08, 0.066, 0.055), cuff=silk)
    hand("handR", w, pbr("glove2", (0.9, 0.9, 0.86), rough=0.5), toward=(0.1, -0.6, 0.6), open_=True)
    coin_m = pbr("coin", (1.0, 0.76, 0.3), rough=0.15, metal=1.0, emit=(1.0, 0.7, 0.3), es=0.6, bump=0.03, bscale=120)
    cp = V(0.38, -0.36, 1.42)
    cyl("coin", cp + V(-0.004, 0, -0.004), cp + V(0.004, 0, 0.004), 0.04, coin_m, n=48)
    C.anchor("coin", cp, (1.0, 0.8, 0.35), 0.8)
    rnd = random.Random(3)
    for k in range(5):
        p = cp + V(rnd.uniform(-0.3, 0.25), rnd.uniform(-0.1, 0.3), rnd.uniform(-0.25, 0.35))
        a = V(rnd.uniform(-1, 1), rnd.uniform(-1, 1), rnd.uniform(-1, 1)).normalized() * 0.003
        cyl("coin2", p - a, p + a, 0.028, coin_m, n=32)


def b_pyro():
    col = (1.0, 0.42, 0.1)
    robe = m_cloth((0.035, 0.028, 0.026), "char", sheen=0.5)
    sash = m_cloth((0.45, 0.04, 0.02), "sash", sheen=0.6)
    metal = m_steel((0.3, 0.27, 0.25), "mask", rough=0.3)
    hem = pbr("hem", (0.1, 0.05, 0.03), rough=0.6, emit=(1.0, 0.35, 0.05), es=0.0)
    torso(robe, fold=(0.05, 8, 4, 0.9), neck=False)
    capelet(robe, inner=hollow(), low=1.3)
    # ember cracks glowing up from the hem
    m, nt, out = kit.new_mat("ember_robe")
    p = nt.nodes.new('ShaderNodeBsdfPrincipled')
    setin(p, (0.06, 0.04, 0.035, 1), 'Base Color')
    setin(p, 0.8, 'Roughness')
    co = _coord(nt)
    v = kit.voronoi_tex(nt, 9.0, co)
    try:
        v.feature = 'DISTANCE_TO_EDGE'
    except Exception:
        pass
    r = kit.ramp(nt, [(0.0, (1, 1, 1)), (0.03, (0, 0, 0)), (1.0, (0, 0, 0))])
    nt.links.new(v.outputs['Distance'], r.inputs['Fac'])
    sep = nt.nodes.new('ShaderNodeSeparateXYZ')
    nt.links.new(co, sep.inputs[0])
    fade = nt.nodes.new('ShaderNodeMapRange')
    setin(fade, 0.55, 'From Min')
    setin(fade, 0.95, 'From Max')
    setin(fade, 1.0, 'To Min')
    setin(fade, 0.0, 'To Max')
    nt.links.new(sep.outputs['Z'], fade.inputs['Value'])
    mul = nt.nodes.new('ShaderNodeMath')
    mul.operation = 'MULTIPLY'
    nt.links.new(r.outputs['Color'], mul.inputs[0])
    nt.links.new(fade.outputs['Result'], mul.inputs[1])
    nt.links.new(mul.outputs[0], kit.sock(p, 'Emission Strength'))
    setin(p, (1.0, 0.35, 0.05, 1), 'Emission Color')
    nt.links.new(p.outputs[0], out.inputs['Surface'])
    loft("robehem", [(0.5, 0.31, 0.24), (0.7, 0.29, 0.22), (0.95, 0.262, 0.192)], m, fold=(0.05, 8, 4, 0.9))
    belt(1.02, sash, metal, rx=0.25, ry=0.175, h=0.05)
    tube("sashtail", [V(0.12, -0.18, 1.0), V(0.16, -0.21, 0.85), V(0.15, -0.22, 0.6)], 0.035, sash, flat=0.2, n=12)
    hood(robe, r=(0.165, 0.18, 0.2), peak=0.12, trim=sash)
    ell("halfmask", C.eye_c + V(0, -0.095, -0.08), (0.085, 0.05, 0.06), metal, head=True)
    for k in range(3):
        rbox("vent", C.eye_c + V(-0.03 + 0.03 * k, -0.142, -0.09), (0.008, 0.004, 0.03), glow("vent", (1.0, 0.4, 0.05), 6.0), bev=0.002, head=True)
    eyes((1.0, 0.55, 0.1), 34)
    # fire held aloft in the left hand (screen right)
    w = arm("R", (0.38, -0.02, 1.2), (0.38, -0.18, 1.45), robe, bell=0.05)
    hand("handR", w, m_leather((0.08, 0.04, 0.02), "glove"), toward=(0, -0.3, 1), open_=True)
    fc = V(0.39, -0.24, 1.66)
    ell("firecore", fc, (0.035, 0.035, 0.045), glow("firecore", (1.0, 0.8, 0.45), 12.0))
    fv = ell("flame", fc + V(0, 0, 0.08), (0.16, 0.16, 0.26), fire_volume("flame", (1.0, 0.38, 0.05), 140.0))
    tongue_o = glow("tongue", (1.0, 0.42, 0.06), 9.0)
    tongue_y = glow("tongue2", (1.0, 0.75, 0.25), 14.0)
    rnd2 = random.Random(21)
    for k in range(9):
        a = 2 * math.pi * k / 9 + rnd2.uniform(-0.3, 0.3)
        rr = rnd2.uniform(0.02, 0.06)
        b0 = fc + V(math.cos(a) * rr, math.sin(a) * rr * 0.6, -0.01)
        h = rnd2.uniform(0.14, 0.28)
        pts = [b0, b0 + V(rnd2.uniform(-0.03, 0.03), 0, h * 0.35), b0 + V(rnd2.uniform(-0.05, 0.05), 0, h * 0.7), b0 + V(rnd2.uniform(-0.04, 0.04), 0, h)]
        tube("tongue", pts, [0.03, 0.026, 0.014, 0.002], tongue_y if k % 3 == 0 else tongue_o, n=10)
    pt = bpy.data.lights.new("firelight", 'POINT')
    pt.energy = 140
    pt.color = (1.0, 0.45, 0.12)
    pt.shadow_soft_size = 0.12
    lo = bpy.data.objects.new("firelight", pt)
    bpy.context.scene.collection.objects.link(lo)
    lo.location = fc
    C.anchor("fire", fc + V(0, 0, 0.05), (1.0, 0.5, 0.1), 1.0)
    # trailing embers from the right hand (screen left)
    w = arm("L", (-0.38, -0.06, 1.1), (-0.36, -0.24, 1.0), robe, bell=0.05)
    hand("handL", w, m_leather((0.08, 0.04, 0.02), "glove2"), toward=(0, -0.3, -1))
    em = glow("ember", (1.0, 0.45, 0.08), 20.0)
    rnd = random.Random(5)
    for k in range(40):
        t = rnd.random()
        p = V(-0.36, -0.28, 1.0).lerp(fc, 0.0) + V(rnd.uniform(-0.12, 0.12), rnd.uniform(-0.1, 0.05), t * 0.6)
        ell("ember", p, (0.004,) * 3, em, seg=8)
    C.anchor("ember", V(-0.36, -0.3, 1.05), (1.0, 0.45, 0.1), 0.6)


def b_storm():
    col = (0.37, 0.75, 1.0)
    cloak = m_cloth((0.06, 0.1, 0.18), "cloak", sheen=0.6)
    silver = pbr("silver", (0.85, 0.88, 0.92), rough=0.18, metal=1.0, scratch=0.08)
    torso(cloak, fold=(0.05, 9, 5, 0.85), neck=False)
    capelet(cloak, inner=m_cloth((0.03, 0.05, 0.1), "capin"), amp=0.1)
    cape(cloak, r1=0.6, nfold=9, amp=0.09, a0=-10, a1=190, inner=m_cloth((0.03, 0.05, 0.1), "cloakin"))
    hood(cloak, r=(0.17, 0.185, 0.21), peak=0.1, drape=1.35, trim=silver)
    torus("clasp", V(0, -0.17, 1.47), 0.03, 0.008, silver, rot=(90, 0, 0))
    ell("claspgem", V(0, -0.175, 1.47), (0.018, 0.01, 0.018), glow("claspgem", (0.5, 0.85, 1.0), 10.0))
    for x in (-0.06, 0.06):
        ribbon("trim", x, 0.012, 0.55, 1.46, silver, depth=0.004)
    eyes((0.75, 0.92, 1.0), 34)
    # staff with a forked head crackling with lightning (screen left)
    w = arm("L", (-0.37, -0.03, 1.12), (-0.42, -0.12, 1.26), cloak, bell=0.05, cuff=silver)
    staff((-0.43, -0.14, 0.45), (-0.43, -0.14, 2.15), pbr("darkwood", (0.05, 0.05, 0.07), rough=0.4, fiber=0.3, coat=0.4), silver)
    hand("handL", w, m_leather((0.05, 0.06, 0.08), "glove"), toward=(0, -0.3, 1))
    top = V(-0.43, -0.14, 2.15)
    for s in (-1, 1):
        tube("fork", [top, top + V(s * 0.05, 0, 0.06), top + V(s * 0.06, 0, 0.16), top + V(s * 0.03, 0, 0.22)], [0.016, 0.013, 0.009, 0.003], silver, n=12)
    orb = top + V(0, 0, 0.12)
    ell("stormorb", orb, (0.035,) * 3, glow("stormorb", (0.8, 0.95, 1.0), 60.0))
    boltm = glow("bolt", (0.65, 0.88, 1.0), 40.0)
    rnd = random.Random(9)
    for k in range(6):
        end = orb + V(rnd.uniform(-0.5, 0.4), rnd.uniform(-0.2, 0.4), rnd.uniform(0.15, 0.9))
        pts = bolt_pts(orb, end, k, jag=0.09, steps=12)
        tube("bolt", pts, [0.011 - 0.007 * i / 12 for i in range(13)], boltm, n=6, sub=0)
        # a branch
        mid = pts[5]
        b_end = mid + V(rnd.uniform(-0.2, 0.2), rnd.uniform(-0.1, 0.1), rnd.uniform(0.05, 0.25))
        tube("branch", bolt_pts(mid, b_end, k + 50, jag=0.04, steps=6), 0.005, boltm, n=6, sub=0)
    pt = bpy.data.lights.new("boltlight", 'POINT')
    pt.energy = 120
    pt.color = (0.6, 0.85, 1.0)
    lo = bpy.data.objects.new("boltlight", pt)
    bpy.context.scene.collection.objects.link(lo)
    lo.location = orb
    C.anchor("bolt", orb, (0.7, 0.9, 1.0), 1.0)
    # the left hand (screen right) crackles with sparks
    w = arm("R", (0.38, -0.05, 1.1), (0.36, -0.25, 1.22), cloak, bell=0.05, cuff=silver)
    hand("handR", w, m_leather((0.05, 0.06, 0.08), "glove2"), toward=(0.1, -0.6, 0.6), open_=True)
    hp = w + V(0.02, -0.08, 0.08)
    for k in range(5):
        end = hp + V(rnd.uniform(-0.1, 0.1), rnd.uniform(-0.1, 0.05), rnd.uniform(0.02, 0.15))
        tube("spark", bolt_pts(hp, end, k + 20, jag=0.03, steps=6), 0.002, boltm, n=6, sub=0)
    C.anchor("spark", hp, (0.7, 0.9, 1.0), 0.6)
    # floating stones
    stone = pbr("stone", (0.2, 0.22, 0.26), rough=0.8, bump=0.4, bscale=12, var=0.3)
    for k in range(5):
        p = V(rnd.choice((-1, 1)) * rnd.uniform(0.45, 0.6), rnd.uniform(-0.1, 0.3), rnd.uniform(1.2, 2.1))
        bm = bmesh.new()
        bmesh.ops.create_icosphere(bm, subdivisions=2, radius=rnd.uniform(0.025, 0.05))
        for v in bm.verts:
            v.co *= rnd.uniform(0.75, 1.2)
        o = _obj("rock", bm, stone, smooth=False)
        o.location = p


ORIGINS = [
    ("arcane", b_arcane, (0.58, 0.42, 1.0)),
    ("berserker", b_berserker, (1.0, 0.27, 0.18)),
    ("guardian", b_guardian, (1.0, 0.8, 0.45)),
    ("engineer", b_engineer, (1.0, 0.6, 0.22)),
    ("lucky", b_lucky, (0.28, 0.85, 0.58)),
    ("pyro", b_pyro, (1.0, 0.42, 0.1)),
    ("storm", b_storm, (0.37, 0.75, 1.0)),
]


def portraits():
    for i, (oid, fn, col) in enumerate(ORIGINS):
        render_origin(oid, fn, col, seed=i + 1)
    log("portraits done")


TASKS = {"portraits": portraits}
for _i, (_id, _fn, _col) in enumerate(ORIGINS):
    TASKS["portrait_" + _id] = (lambda a, b, c, s: (lambda: render_origin(a, b, c, s)))(_id, _fn, _col, _i + 1)
