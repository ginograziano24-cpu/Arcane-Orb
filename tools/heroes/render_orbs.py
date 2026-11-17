"""Render the game's skinned hero model (.orbs, rest pose) with its texture,
from chosen camera positions, using a small numpy z-buffer rasterizer."""
import struct, sys, zlib, math
import numpy as np
from PIL import Image


def load_orbs(path, tex_path):
    data = open(path, "rb").read()
    assert data[:4] == b"ORBS"
    raw = zlib.decompress(data[4:])
    nv, nt, nb, nf = struct.unpack_from("<4I", raw, 4)
    o = 24
    pos = np.frombuffer(raw, "<f4", nv * 3, o).reshape(nv, 3).astype(np.float64); o += nv * 12
    nrm = np.frombuffer(raw, "i1", nv * 3, o).reshape(nv, 3).astype(np.float64); o += nv * 3
    uv = np.frombuffer(raw, "<f4", nv * 2, o).reshape(nv, 2).astype(np.float64); o += nv * 8
    o += nv * 4 + nv * 4
    idx = np.frombuffer(raw, "<u4", nt * 3, o).reshape(nt, 3).astype(np.int64)
    nrm /= np.maximum(np.linalg.norm(nrm, axis=1, keepdims=True), 1e-9)
    tex = np.asarray(Image.open(tex_path).convert("RGB")).astype(np.float32) / 255
    return pos, nrm, uv, idx, tex


def look_at(eye, tgt, up=(0, 1, 0)):
    f = np.array(tgt, float) - eye; f /= np.linalg.norm(f)
    r = np.cross(up, f); r /= np.linalg.norm(r)
    u = np.cross(f, r)
    return np.stack([r, u, f])


def render(pos, nrm, uv, idx, tex, eye, tgt, fov_deg, W, H, bg=(0.78, 0.79, 0.82)):
    eye = np.array(eye, float)
    R = look_at(eye, tgt)
    pc = (pos - eye) @ R.T                      # camera space: x right, y up, z forward
    f = 0.5 * H / math.tan(math.radians(fov_deg) / 2)
    z = pc[:, 2]
    sx = W / 2 + f * pc[:, 0] / np.maximum(z, 1e-6)
    sy = H / 2 - f * pc[:, 1] / np.maximum(z, 1e-6)
    img = np.empty((H, W, 3), np.float32); img[:] = bg
    zb = np.full((H, W), np.inf, np.float32)
    light = np.array([-0.4, 0.6, -0.7]); light /= np.linalg.norm(light)
    fill = np.array([0.5, -0.6, -0.4]); fill /= np.linalg.norm(fill)
    sh = 0.42 + 0.5 * np.clip(nrm @ light, 0, 1) + 0.22 * np.clip(nrm @ fill, 0, 1)
    th, tw = tex.shape[:2]
    for t in range(len(idx)):
        a, b, c = idx[t]
        if z[a] <= 0.01 or z[b] <= 0.01 or z[c] <= 0.01:
            continue
        xs = np.array([sx[a], sx[b], sx[c]]); ys = np.array([sy[a], sy[b], sy[c]])
        x0, x1 = int(max(0, math.floor(xs.min()))), int(min(W - 1, math.ceil(xs.max())))
        y0, y1 = int(max(0, math.floor(ys.min()))), int(min(H - 1, math.ceil(ys.max())))
        if x0 > x1 or y0 > y1:
            continue
        den = (ys[1] - ys[2]) * (xs[0] - xs[2]) + (xs[2] - xs[1]) * (ys[0] - ys[2])
        if abs(den) < 1e-12:
            continue
        gx, gy = np.meshgrid(np.arange(x0, x1 + 1) + 0.5, np.arange(y0, y1 + 1) + 0.5)
        w0 = ((ys[1] - ys[2]) * (gx - xs[2]) + (xs[2] - xs[1]) * (gy - ys[2])) / den
        w1 = ((ys[2] - ys[0]) * (gx - xs[2]) + (xs[0] - xs[2]) * (gy - ys[2])) / den
        w2 = 1 - w0 - w1
        m = (w0 >= 0) & (w1 >= 0) & (w2 >= 0)
        if not m.any():
            continue
        iz = w0 / z[a] + w1 / z[b] + w2 / z[c]
        zz = 1 / iz
        sub = zb[y0:y1 + 1, x0:x1 + 1]
        m &= zz < sub
        if not m.any():
            continue
        # perspective-correct uv
        u = (w0 * uv[a, 0] / z[a] + w1 * uv[b, 0] / z[b] + w2 * uv[c, 0] / z[c]) * zz
        v = (w0 * uv[a, 1] / z[a] + w1 * uv[b, 1] / z[b] + w2 * uv[c, 1] / z[c]) * zz
        s_ = w0 * sh[a] + w1 * sh[b] + w2 * sh[c]
        tx = np.clip((u % 1.0) * (tw - 1), 0, tw - 1).astype(int)
        ty = np.clip((v % 1.0) * (th - 1), 0, th - 1).astype(int)
        col = tex[ty, tx] * np.clip(s_, 0, 1.3)[..., None]
        sub[m] = zz[m]
        blk = img[y0:y1 + 1, x0:x1 + 1]
        blk[m] = np.clip(col[m], 0, 1)
    return (img * 255).astype(np.uint8)


if __name__ == "__main__":
    pos, nrm, uv, idx, tex = load_orbs(sys.argv[1], sys.argv[2])
    out = sys.argv[3]
    lo, hi = pos.min(0), pos.max(0)
    print("verts", len(pos), "tris", len(idx), "bounds", np.round(lo, 3), np.round(hi, 3))
    cx, cz = 0.5 * (lo[0] + hi[0]), 0.5 * (lo[2] + hi[2]); Hh = hi[1] - lo[1]; y0 = lo[1]
    views = []
    # from the floor: camera at ankle height in front, looking up at her
    for name, eye, tgt, fov in (
        ("floor_front", (cx, y0 + 0.03 * Hh, cz - 0.75 * Hh), (cx, y0 + 0.62 * Hh, cz), 62),
        ("floor_threequarter", (cx + 0.55 * Hh, y0 + 0.03 * Hh, cz - 0.55 * Hh), (cx, y0 + 0.62 * Hh, cz), 62),
        ("floor_back", (cx, y0 + 0.03 * Hh, cz + 0.75 * Hh), (cx, y0 + 0.62 * Hh, cz), 62),
    ):
        im = render(pos, nrm, uv, idx, tex, eye, tgt, fov, 700, 1000)
        Image.fromarray(im).save(out + "_" + name + ".png"); views.append(im)
        print("rendered", name)
    Image.fromarray(np.concatenate(views, axis=1)).save(out + "_floor_views.png")
