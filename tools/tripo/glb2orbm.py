"""Convert a (textured) GLB into Arcane Orb's .orbm model format.

.orbm: b"ORBM" + zlib( version u32, count u32, per triangle: 9 f32 positions,
R G B A, emission byte, flags byte (bit0 = tintable) ).
Each triangle gets one flat colour sampled from the texture (corners + centre),
which matches the game's faceted look. The model is centred on x/z, rests on
y=0 and is scaled to --height game units. Optional --tint-hue marks faces near
a hue as tintable; --emit marks bright faces as emissive.
"""
import sys, json, struct, zlib, io, argparse
import numpy as np
from PIL import Image

CT = {5120: np.int8, 5121: np.uint8, 5122: np.int16, 5123: np.uint16, 5125: np.uint32, 5126: np.float32}
NC = {"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT4": 16}

def read_glb(path):
    b = open(path, "rb").read()
    assert b[:4] == b"glTF", "not a GLB"
    off = 12; js = None; bin_ = b""
    while off < len(b):
        ln, typ = struct.unpack_from("<II", b, off); off += 8
        chunk = b[off:off + ln]; off += ln
        if typ == 0x4E4F534A: js = json.loads(chunk)
        elif typ == 0x004E4942: bin_ = chunk
    return js, bin_

def accessor(js, bin_, i):
    a = js["accessors"][i]; bv = js["bufferViews"][a["bufferView"]]
    dt = CT[a["componentType"]]; n = NC[a["type"]]
    start = bv.get("byteOffset", 0) + a.get("byteOffset", 0)
    stride = bv.get("byteStride", 0); isz = np.dtype(dt).itemsize * n
    if stride and stride != isz:
        raw = np.frombuffer(bin_, np.uint8, count=stride * a["count"], offset=start).reshape(a["count"], stride)[:, :isz]
        arr = np.frombuffer(raw.tobytes(), dt).reshape(a["count"], n)
    else:
        arr = np.frombuffer(bin_, dt, count=a["count"] * n, offset=start).reshape(a["count"], n)
    arr = arr.astype(np.float64)
    if a.get("normalized"):
        arr /= float(np.iinfo(dt).max)
    return arr

def image_for(js, bin_, mat_i):
    if mat_i is None: return None, (1, 1, 1, 1)
    m = js["materials"][mat_i]; pbr = m.get("pbrMetallicRoughness", {})
    fac = pbr.get("baseColorFactor", [1, 1, 1, 1])
    t = pbr.get("baseColorTexture")
    if t is None: return None, fac
    src = js["textures"][t["index"]]["source"]; im = js["images"][src]
    bv = js["bufferViews"][im["bufferView"]]
    data = bin_[bv.get("byteOffset", 0):bv.get("byteOffset", 0) + bv["byteLength"]]
    return np.asarray(Image.open(io.BytesIO(data)).convert("RGB"), np.float64) / 255, fac

def qmat(q):
    x, y, z, w = q
    return np.array([[1-2*(y*y+z*z), 2*(x*y-z*w), 2*(x*z+y*w)],
                     [2*(x*y+z*w), 1-2*(x*x+z*z), 2*(y*z-x*w)],
                     [2*(x*z-y*w), 2*(y*z+x*w), 1-2*(x*x+y*y)]])

def node_mat(n):
    if "matrix" in n: return np.array(n["matrix"]).reshape(4, 4).T
    M = np.eye(4)
    S = np.diag(n.get("scale", [1, 1, 1])); R = qmat(n.get("rotation", [0, 0, 0, 1]))
    M[:3, :3] = R @ S; M[:3, 3] = n.get("translation", [0, 0, 0])
    return M

def collect(js, bin_):
    tris, cols = [], []
    def walk(ni, parent):
        n = js["nodes"][ni]; M = parent @ node_mat(n)
        if "mesh" in n:
            for p in js["meshes"][n["mesh"]]["primitives"]:
                if p.get("mode", 4) != 4: continue
                pos = accessor(js, bin_, p["attributes"]["POSITION"])
                pos = (np.c_[pos, np.ones(len(pos))] @ M.T)[:, :3]
                idx = accessor(js, bin_, p["indices"]).astype(int).ravel() if "indices" in p else np.arange(len(pos))
                tri = pos[idx].reshape(-1, 3, 3)
                img, fac = image_for(js, bin_, p.get("material"))
                if img is not None and "TEXCOORD_0" in p["attributes"]:
                    uv = accessor(js, bin_, p["attributes"]["TEXCOORD_0"])[idx].reshape(-1, 3, 2)
                    h, w = img.shape[:2]
                    samples = [uv[:, 0], uv[:, 1], uv[:, 2], uv.mean(1), uv.mean(1)]  # centre counts double
                    acc = np.zeros((len(tri), 3))
                    for s in samples:
                        x = np.clip((np.mod(s[:, 0], 1)) * (w - 1), 0, w - 1).astype(int)
                        y = np.clip((np.mod(s[:, 1], 1)) * (h - 1), 0, h - 1).astype(int)
                        acc += img[y, x]
                    c = acc / len(samples) * np.array(fac[:3])
                elif "COLOR_0" in p["attributes"]:
                    vc = accessor(js, bin_, p["attributes"]["COLOR_0"])[idx][:, :3].reshape(-1, 3, 3)
                    c = vc.mean(1) * np.array(fac[:3])
                else:
                    c = np.tile(np.array(fac[:3]), (len(tri), 1))
                tris.append(tri); cols.append(c)
        for ch in n.get("children", []): walk(ch, M)
    sc = js["scenes"][js.get("scene", 0)]
    for ni in sc["nodes"]: walk(ni, np.eye(4))
    return np.concatenate(tris), np.concatenate(cols)

def srgb_boost(c, sat, gain):
    g = c.mean(1, keepdims=True)
    return np.clip((g + (c - g) * sat) * gain, 0, 1)

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("glb"); ap.add_argument("out")
    ap.add_argument("--height", type=float, default=6.0, help="model height in game units")
    ap.add_argument("--yaw", type=float, default=0.0, help="turn around the up axis, degrees")
    ap.add_argument("--sat", type=float, default=1.15); ap.add_argument("--gain", type=float, default=1.0)
    ap.add_argument("--emit-above", type=float, default=2.0, help="faces brighter than this (0-1) glow")
    ap.add_argument("--emit", type=float, default=0.6)
    ap.add_argument("--tintable", action="store_true", help="every face takes the draw tint")
    ap.add_argument("--ymin", type=float, default=0.0, help="lowest point sits at this height")
    a = ap.parse_args()
    js, bin_ = read_glb(a.glb)
    tri, col = collect(js, bin_)
    # glTF: +y up, +z toward the viewer; the game: +z away from the camera
    tri = tri * np.array([1, 1, -1])
    tri = tri[:, ::-1]                       # keep winding after the mirror
    if a.yaw:
        r = np.radians(a.yaw); R = np.array([[np.cos(r), 0, np.sin(r)], [0, 1, 0], [-np.sin(r), 0, np.cos(r)]])
        tri = tri @ R.T
    lo, hi = tri.reshape(-1, 3).min(0), tri.reshape(-1, 3).max(0)
    s = a.height / max(hi[1] - lo[1], 1e-9)
    ctr = np.array([(lo[0] + hi[0]) / 2, lo[1], (lo[2] + hi[2]) / 2])
    tri = (tri - ctr) * s + np.array([0, a.ymin, 0])
    col = srgb_boost(col, a.sat, a.gain)
    lum = col @ np.array([0.299, 0.587, 0.114])
    out = io.BytesIO()
    out.write(struct.pack("<II", 1, len(tri)))
    for t, c, l in zip(tri.astype(np.float32), col, lum):
        out.write(t.tobytes())
        e = int(min(255, a.emit * 127.5)) if l > a.emit_above else 0
        out.write(bytes([int(c[0] * 255 + .5), int(c[1] * 255 + .5), int(c[2] * 255 + .5), 255, e, 1 if a.tintable else 0]))
    open(a.out, "wb").write(b"ORBM" + zlib.compress(out.getvalue(), 9))
    ext = tri.reshape(-1, 3); print(f"{a.out}: {len(tri)} triangles, size {np.ptp(ext,0).round(2)}, {len(open(a.out,'rb').read())} bytes")

if __name__ == "__main__":
    main()
