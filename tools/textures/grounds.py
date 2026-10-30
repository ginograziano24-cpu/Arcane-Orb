"""Bakes the seamless ground tiles around the arena (each covers 8 x 8 world units)."""
import sys, os
import numpy as np
from texlib import *

N = 1024
U = 8.0
S = N / U
OUT = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'floor')


def nz(c, o=5, seed=0, gain=0.5):
    return fbm(N, N, c, c, o, seed, gain, periodic=True)


def save(name, rgb):
    os.makedirs(OUT, exist_ok=True)
    to_img(rgb).save(os.path.join(OUT, 'ground_' + name + '.jpg'), quality=88, subsampling=0)
    print('wrote ground', name)


def meadow():
    base = nz(4, 5, 1)
    blades = nz(256, 2, 2)
    clumps = nz(16, 4, 3)
    h = base * 0.05 + blades * 0.02 + clumps * 0.03
    alb = lerp(hex2('4f7d2f'), hex2('7aa548'), smoothstep(0.3, 0.7, clumps))
    alb = lerp(alb, hex2('a6b45a'), smoothstep(0.62, 0.8, base) * 0.6)    # dry patches
    alb = lerp(alb, hex2('35602a'), smoothstep(0.55, 0.3, base) * 0.5)
    alb = alb * (0.8 + 0.4 * blades[..., None])
    flowers = smoothstep(0.86, 0.9, nz(128, 2, 4)) * smoothstep(0.5, 0.65, clumps)
    fcol = lerp(hex2('f4e46a'), hex2('ffffff'), smoothstep(0.4, 0.6, nz(32, 2, 5)))
    alb = lerp(alb, fcol, flowers)
    clover = smoothstep(0.7, 0.8, nz(48, 3, 6))
    alb = lerp(alb, hex2('3f8a35'), clover * 0.4)
    save('meadow', shade(alb, h, S, light=(-0.7, 0.55, 0.2), ao=1.0))


def ruins():
    dune = nz(3, 5, 11)
    ripple = np.sin((np.mgrid[0:N, 0:N][1] / N * 2 * np.pi * 28) + nz(6, 4, 12) * 14)
    h = dune * 0.15 + ripple * 0.006 + nz(200, 2, 13) * 0.004
    alb = lerp(hex2('d8b47c'), hex2('ebcf9b'), smoothstep(0.35, 0.7, dune))
    alb = alb * (0.9 + 0.15 * nz(300, 1, 14)[..., None])
    peb = smoothstep(0.87, 0.9, nz(100, 2, 15))
    alb = lerp(alb, hex2('9b7a52'), peb * 0.7)
    h += peb * 0.01
    save('ruins', shade(alb, h, S, light=(0.55, 0.75, -0.35), ao=0.8))


def caverns():
    f1, f2, cid = voronoi(N, N, 160, 21, periodic=True)
    f2 = f2 / S
    cr = RNG(22).random(cid.max() + 1)
    h = (cr[cid] * 0.04 + nz(20, 5, 23) * 0.06) * smoothstep(0.0, 0.06, f2)
    alb = lerp(hex2('1e2338'), hex2('2e3654'), cr[cid][..., None] * 0.6 + nz(10, 4, 24)[..., None] * 0.4)
    vein = smoothstep(0.03, 0.0, f2) * smoothstep(0.5, 0.7, nz(8, 3, 25))
    glow = blur(vein, 3) * 1.4
    save('caverns', shade(alb, h, S, light=(0.3, 0.8, -0.5), ao=1.2, emissive=glow[..., None] * hex2('42d6ff') * 0.4))


def clouds():
    puffs = nz(6, 6, 31, 0.55)
    h = puffs * 0.4
    alb = lerp(hex2('c8d4ee'), hex2('ffffff'), smoothstep(0.35, 0.7, puffs))
    save('clouds', shade(alb, h, S, light=(-0.2, 0.9, -0.3), ao=0.6, amb=0.7))


def machine():
    plate = 2.0
    yy, xx = np.mgrid[0:N, 0:N] / S
    ex = np.minimum(np.mod(xx, plate), plate - np.mod(xx, plate))
    ez = np.minimum(np.mod(yy, plate * 2), plate * 2 - np.mod(yy, plate * 2))
    seam = smoothstep(0.03, 0.0, np.minimum(ex, ez))
    h = nz(30, 4, 41) * 0.006 - seam * 0.01
    alb = hex2('3a3d44') * (0.8 + 0.35 * nz(8, 5, 42)[..., None])
    alb = lerp(alb, hex2('121214'), smoothstep(0.72, 0.85, nz(10, 4, 43)) * 0.35)   # oil
    alb = lerp(alb, hex2('101012'), seam)
    rust = smoothstep(0.65, 0.8, nz(16, 5, 44))
    alb = lerp(alb, hex2('6a3b22'), rust * 0.45)
    save('machine', shade(alb, h, S, light=(0.4, 0.7, -0.6), ao=1.0, spec=0.1))


WORLDS = {'meadow': meadow, 'ruins': ruins, 'caverns': caverns, 'clouds': clouds, 'machine': machine}
if __name__ == '__main__':
    for n in (sys.argv[1:] or list(WORLDS)):
        WORLDS[n]()
