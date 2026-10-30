"""Small numpy toolkit for baking detailed, lit floor and ground textures.

Everything is procedural: height fields are built from tiles, noise and
voronoi cells, then shaded with the world's key light, a cavity/AO term
and a little specular, so the game can draw them with flat lighting.
"""
import numpy as np
from scipy import ndimage

RNG = np.random.default_rng


def _fade(t):
    return t * t * t * (t * (t * 6 - 15) + 10)


def value_noise(h, w, cells_y, cells_x, seed=0, periodic=True):
    """Smooth value noise with cells_x by cells_y lattice cells over the image.
    Periodic (seamless) by construction when periodic=True."""
    r = RNG(seed)
    cy, cx = int(cells_y), int(cells_x)
    lat = r.random((cy + 1, cx + 1))
    if periodic:
        lat[-1, :] = lat[0, :]
        lat[:, -1] = lat[:, 0]
    ys = np.linspace(0, cy, h, endpoint=False)
    xs = np.linspace(0, cx, w, endpoint=False)
    yi = ys.astype(int)
    xi = xs.astype(int)
    fy = _fade(ys - yi)[:, None]
    fx = _fade(xs - xi)[None, :]
    a = lat[yi][:, xi]
    b = lat[yi][:, xi + 1]
    c = lat[yi + 1][:, xi]
    d = lat[yi + 1][:, xi + 1]
    return (a * (1 - fx) + b * fx) * (1 - fy) + (c * (1 - fx) + d * fx) * fy


def fbm(h, w, base_y, base_x, octaves=5, seed=0, gain=0.5, periodic=True):
    out = np.zeros((h, w))
    amp, tot = 1.0, 0.0
    for o in range(octaves):
        f = 2 ** o
        out += amp * value_noise(h, w, base_y * f, base_x * f, seed + 17 * o, periodic)
        tot += amp
        amp *= gain
    return out / tot


def ridged(h, w, base_y, base_x, octaves=4, seed=0, periodic=True):
    n = fbm(h, w, base_y, base_x, octaves, seed, periodic=periodic)
    return 1 - np.abs(n * 2 - 1)


def voronoi(h, w, n, seed=0, jitter=1.0, periodic=True):
    """Returns (F1 distance, F2-F1 edge distance, cell id), distances in pixels."""
    r = RNG(seed)
    pts = r.random((n, 2)) * [h, w]
    if periodic:
        tiles = []
        for dy in (-h, 0, h):
            for dx in (-w, 0, w):
                tiles.append(pts + [dy, dx])
        allp = np.concatenate(tiles)
        ids = np.tile(np.arange(n), 9)
    else:
        allp, ids = pts, np.arange(n)
    from scipy.spatial import cKDTree
    tree = cKDTree(allp)
    yy, xx = np.mgrid[0:h, 0:w]
    q = np.stack([yy.ravel(), xx.ravel()], 1).astype(float)
    d, i = tree.query(q, k=2)
    f1 = d[:, 0].reshape(h, w)
    f2 = d[:, 1].reshape(h, w)
    cid = ids[i[:, 0]].reshape(h, w)
    return f1, f2 - f1, cid


def blur(a, s, periodic=True):
    return ndimage.gaussian_filter(a, s, mode='wrap' if periodic else 'nearest')


def smoothstep(e0, e1, x):
    t = np.clip((x - e0) / (e1 - e0), 0, 1)
    return t * t * (3 - 2 * t)


def shade(albedo, height, px_per_unit, light=(-0.35, 0.85, -0.4), relief=1.0, ao=0.6, spec=0.0,
          spec_pow=30, periodic=True, emissive=None, amb=0.55):
    """Lights an albedo with a height field (in world units).

    Image rows run toward -z (the top of the texture is the far end), columns
    toward +x. light is the world-space direction toward the light."""
    hy, hx = np.gradient(height * relief, edge_order=1)
    # world-space slopes (height units per world unit)
    dx = hx * px_per_unit
    dz = -hy * px_per_unit
    n = np.stack([-dx, np.ones_like(dx), -dz], -1)
    n /= np.linalg.norm(n, axis=-1, keepdims=True)
    L = np.array(light, float)
    L /= np.linalg.norm(L)
    diff = np.clip(n @ L, 0, 1)
    flat = max(L[1], 1e-3)
    lit = amb + (1 - amb) * diff / flat  # flat ground keeps ~unit brightness
    # cavity: below the local mean is darker
    s = px_per_unit * 0.08
    cav = height - blur(height, s, periodic)
    cav2 = height - blur(height, s * 4, periodic)
    occl = np.clip(1 + (cav * 6 + cav2 * 3) * ao, 0.35, 1.15)
    col = albedo * (lit * occl)[..., None]
    if np.any(spec):
        V = np.array([0, 0.75, -0.66])
        Hh = L + V
        Hh /= np.linalg.norm(Hh)
        sp = np.clip(n @ Hh, 0, 1) ** spec_pow * spec
        col = col + sp[..., None]
    if emissive is not None:
        col = col + emissive
    return np.clip(col, 0, 1)


def to_img(a):
    from PIL import Image
    return Image.fromarray((np.clip(a, 0, 1) * 255 + 0.5).astype(np.uint8))


def hex2(c):
    c = c.lstrip('#')
    return np.array([int(c[i:i + 2], 16) / 255 for i in (0, 2, 4)])


def lerp(a, b, t):
    t = np.asarray(t)
    if t.ndim == 2:
        t = t[..., None]
    return a + (b - a) * t
