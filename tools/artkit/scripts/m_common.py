"""Assets every world uses: bricks, paddle, capsules, rocks, arena walls and
posts, and the critters. See models.py for the conventions."""
import math, random
from models import (asset, M, box, cyl, sphere, lathe, mesh, bevel, place, tube, prism,
                    rock, foliage, torus, move, rot)

# ---------------------------------------------------------------- bricks
# Built at the nominal brick size: 1.88 wide, 0.80 tall, 0.99 deep, on z=0.
BW, BH, BD = 1.88, 0.80, 0.99


@asset("brick_meadow", "bricks", ao=0.45)
def brick_meadow():
    """Three painted planks held by two dark iron bands."""
    paint = M("paint", (236, 236, 236), tint=True, var=0.07)
    iron = M("iron", (58, 54, 50))
    h = BH / 3
    for i in range(3):
        box((0, 0, h * i + h / 2), (BW, BD, h * 0.9), paint, bev=0.04, seg=1)
    for x in (-BW / 2 + 0.24, BW / 2 - 0.24):
        box((x, 0, BH / 2), (0.11, BD + 0.03, BH + 0.03), iron)


@asset("brick_ruins", "bricks", ao=0.45)
def brick_ruins():
    """A sandstone block with a sunken glyph panel on its face."""
    stone = M("stone", (240, 236, 228), tint=True, var=0.1)
    deep = M("deep", (176, 170, 160), tint=True)
    box((0, 0, BH / 2), (BW, BD, BH), stone, bev=0.07, seg=1)
    box((0, -BD / 2 + 0.005, BH / 2), (BW * 0.66, 0.03, BH * 0.5), deep)
    for x, w in ((-0.34, 0.07), (0, 0.2), (0.34, 0.07)):
        box((x, -BD / 2 - 0.01, BH / 2), (w, 0.03, BH * 0.3), stone)


@asset("brick_clouds", "bricks", ao=0.4)
def brick_clouds():
    """A marble tile with a raised, rounded crown and a gilded band."""
    marble = M("marble", (248, 248, 252), tint=True, var=0.05)
    gold = M("gold", (255, 214, 120), emit=0.2)
    box((0, 0, BH * 0.42), (BW, BD, BH * 0.84), marble, bev=0.1, seg=2)
    box((0, 0, BH * 0.9), (BW * 0.84, BD * 0.76, BH * 0.2), marble, bev=0.06, seg=1)
    box((0, -BD / 2 - 0.005, BH * 0.2), (BW * 0.9, 0.03, 0.06), gold)


@asset("brick_neon", "bricks", ao=0.3)
def brick_neon():
    """A glowing glass core set in a dark frame that shows on every side."""
    core = M("core", (236, 236, 246), tint=True, emit=0.3)
    frame = M("frame", (34, 28, 52))
    t = 0.09
    # frame: four corner posts and top/bottom rims, so the core shows through
    for x in (-BW / 2 + t / 2, BW / 2 - t / 2):
        for y in (-BD / 2 + t / 2, BD / 2 - t / 2):
            box((x, y, BH / 2), (t, t, BH), frame)
    for z in (t / 2, BH - t / 2):
        box((0, -BD / 2 + t / 2, z), (BW, t, t), frame)
        box((0, BD / 2 - t / 2, z), (BW, t, t), frame)
        box((-BW / 2 + t / 2, 0, z), (t, BD, t), frame)
        box((BW / 2 - t / 2, 0, z), (t, BD, t), frame)
    box((0, 0, BH / 2), (BW - 0.04, BD - 0.04, BH - 0.06), core, bev=0.03, seg=1)


@asset("brick_caverns", "bricks", ao=0.4)
def brick_caverns():
    """A faceted crystal lozenge with a split, chipped crown."""
    cr = M("crystal", (240, 240, 250), tint=True, emit=0.15)
    hw, hd = BW / 2, BD / 2
    t = 0.7
    ring0 = [(-hw, -hd, 0), (hw, -hd, 0), (hw, hd, 0), (-hw, hd, 0)]
    ring1 = [(-hw, -hd, BH * 0.6), (hw, -hd, BH * 0.6), (hw, hd, BH * 0.6), (-hw, hd, BH * 0.6)]
    ring2 = [(-hw * t, -hd * t, BH), (hw * t, -hd * t, BH), (hw * t, hd * t, BH), (-hw * t, hd * t, BH)]
    v = ring0 + ring1 + ring2 + [(0, -hd * 0.95, BH * 0.8), (0, hd * 0.95, BH * 0.8)]
    f = [(0, 3, 2, 1)]
    for i in range(4):
        a, b = i, (i + 1) % 4
        f.append((a, b, b + 4, a + 4))
    # upper band, with a ridge on the long faces
    f += [(4, 12, 8), (12, 5, 9, 8), (4, 5, 12), (6, 13, 10), (13, 7, 11, 10), (6, 7, 13), (5, 6, 10, 9), (7, 4, 8, 11)]
    f.append((8, 9, 10, 11))
    mesh(v, f, cr)


@asset("brick_machine", "bricks", ao=0.45)
def brick_machine():
    """A riveted steel plate with a raised centre panel."""
    plate = M("plate", (226, 226, 232), tint=True, var=0.05)
    rivet = M("rivet", (86, 88, 96))
    box((0, 0, BH / 2), (BW, BD, BH), plate, bev=0.05, seg=1)
    box((0, -BD / 2, BH / 2), (BW * 0.64, 0.06, BH * 0.48), plate)
    for x in (-BW / 2 + 0.14, BW / 2 - 0.14):
        for z in (0.15, BH - 0.15):
            box((x, -BD / 2 - 0.01, z), (0.07, 0.03, 0.07), rivet)
        for y in (-BD / 2 + 0.15, BD / 2 - 0.15):
            box((x, y, BH + 0.01), (0.07, 0.07, 0.03), rivet)


# ---------------------------------------------------------------- paddle

@asset("paddle_body", "common", ao=0.4)
def paddle_body():
    """The paddle's middle, 1 unit long (the game stretches it), 0.7 tall."""
    body = M("pbody", (236, 236, 236), tint=True, var=0.03)
    groove = M("pgroove", (36, 30, 46))
    box((0, 0, 0.35), (1.0, 0.7, 0.7), body, bev=0.08, seg=1)
    box((0, -0.36, 0.42), (1.0, 0.03, 0.08), groove)
    box((0, -0.36, 0.2), (1.0, 0.03, 0.05), groove)


@asset("paddle_cap", "common", ao=0.4)
def paddle_cap():
    """An end cap 0.6 wide with a glowing gem facing the arena."""
    cap = M("pcap", (236, 236, 236), tint=True)
    gem = M("pgem", (255, 240, 210), emit=0.7)
    box((0, 0, 0.41), (0.6, 0.8, 0.82), cap, bev=0.12, seg=2)
    sphere((0, -0.4, 0.42), 0.13, gem, sub=1, scale=(1, 0.55, 1))


@asset("capsule", "common", ao=0.3)
def capsule():
    """The power-up capsule: a pill 1.2 long lying along x, centred."""
    shell = M("cshell", (236, 236, 236), tint=True, emit=0.25)
    band = M("cband", (250, 246, 236), emit=0.4)
    o = cyl((0, 0, 0), 0.25, 0.8, shell, n=12, rot=(0, 90, 0))
    place(o, loc=(0, 0, 0))
    for sx in (-1, 1):
        sphere((sx * 0.4, 0, 0), 0.25, shell, sub=2)
    place(cyl((0, 0, 0), 0.265, 0.12, band, n=12, rot=(0, 90, 0)), loc=(0, 0, 0))


@asset("rock_fall", "common", ao=0.6)
def rock_fall():
    """The boulder that drops toward the paddle (about 1.2 across)."""
    stone = M("rstone", (128, 116, 106), var=0.15)
    rock((0, 0, 0.55), 0.62, stone, seed=4, scale=(1, 0.9, 0.85))


# ---------------------------------------------------------------- walls
# One wall section 2 units long (x -1..1), 1.4 tall, reaching from its arena
# face at y=0 out to y=1.4. The game rotates and stretches it to each piece
# of the arena outline.

@asset("wall_meadow", "walls", ao=1.0)
def wall_meadow():
    stone = M("wst", (150, 140, 124), var=0.12)
    wood = M("wwd", (150, 106, 70), var=0.1)
    cap = M("wcap", (120, 84, 56))
    for k in range(2):
        for i in range(3):
            off = 0.33 * (k % 2)
            x = -1 + (i + 0.5) * 0.667 + off
            if x > 1:
                x -= 2
            box((x, 0.7, 0.2 + k * 0.38), (0.64, 1.4, 0.36), stone, bev=0.05, seg=1)
    for i in range(4):
        box((-0.75 + i * 0.5, 0.25, 1.05), (0.46, 0.16, 0.42), wood, bev=0.02, seg=1)
    box((0, 0.7, 1.32), (2.0, 1.4, 0.16), cap, bev=0.03, seg=1)


@asset("wall_ruins", "walls", ao=1.0)
def wall_ruins():
    stone = M("rst", (206, 176, 128), var=0.1)
    band = M("rband", (172, 144, 104))
    teal = M("rteal", (80, 170, 165), emit=0.15)
    box((0, 0.7, 0.3), (2.0, 1.4, 0.6), stone, bev=0.05, seg=1)
    box((-0.5, 0.7, 0.85), (0.98, 1.4, 0.5), stone, bev=0.05, seg=1)
    box((0.5, 0.7, 0.85), (0.98, 1.4, 0.5), stone, bev=0.05, seg=1)
    box((0, 0.7, 1.25), (2.0, 1.4, 0.3), band, bev=0.04, seg=1)
    for x in (-0.5, 0.5):
        box((x, -0.01, 0.85), (0.4, 0.04, 0.2), teal)


@asset("wall_clouds", "walls", ao=0.8)
def wall_clouds():
    marble = M("cm", (240, 238, 248), var=0.04)
    gold = M("cg", (240, 200, 110), emit=0.15)
    box((0, 0.7, 0.18), (2.0, 1.4, 0.36), marble, bev=0.05, seg=1)
    box((0, 0.7, 1.28), (2.0, 1.4, 0.24), marble, bev=0.06, seg=1)
    for i in range(4):
        x = -0.75 + i * 0.5
        lathe([(0.12, 0.36), (0.16, 0.45), (0.1, 0.7), (0.16, 0.95), (0.12, 1.16)], marble, n=8, c=(x, 0.35, 0))
    box((0, -0.01, 1.2), (2.0, 0.03, 0.05), gold)


@asset("wall_neon", "walls", ao=0.6)
def wall_neon():
    panel = M("np", (54, 42, 86), var=0.05)
    dark = M("nd", (26, 20, 44))
    glow = M("ng", (255, 90, 220), emit=1.0)
    box((0, 0.7, 0.7), (2.0, 1.4, 1.4), dark, bev=0.04, seg=1)
    box((0, -0.02, 0.55), (1.8, 0.06, 0.8), panel, bev=0.02, seg=1)
    box((0, -0.06, 0.55), (1.6, 0.03, 0.05), glow)
    for x in (-0.95, 0.95):
        box((x, -0.03, 0.7), (0.06, 0.06, 1.3), glow)


@asset("wall_caverns", "walls", ao=1.0)
def wall_caverns():
    rockm = M("cr", (70, 78, 108), var=0.18)
    cry = M("cc", (140, 200, 255), emit=0.6)
    for i in range(3):
        rock((-0.7 + i * 0.7, 0.7, 0.55), 0.62, rockm, seed=10 + i, scale=(0.75, 1.1, 1.05), sub=2, rough=0.18)
    for x, a in ((-0.4, -15), (0.5, 20)):
        o = cyl((0, 0, 0), 0.08, 0.5, cry, n=5, r2=0.0)
        place(o, loc=(x, 0.2, 1.05), rot=(a, 0, 0))


@asset("wall_machine", "walls", ao=0.8)
def wall_machine():
    steel = M("ms", (92, 96, 106), var=0.06)
    dark = M("md", (56, 58, 66))
    hazard = M("mh", (232, 168, 40))
    rivet = M("mr", (150, 152, 160))
    box((0, 0.7, 0.7), (2.0, 1.4, 1.4), dark)
    box((0, 0.05, 0.7), (2.0, 0.1, 1.2), steel, bev=0.02, seg=1)
    for k in range(5):
        x = -0.8 + k * 0.4
        o = box((x, -0.01, 0.2), (0.18, 0.04, 0.3), hazard, rot=(0, 30, 0))
    box((0, 0.7, 1.35), (2.0, 1.4, 0.1), steel)
    for x in (-0.9, -0.3, 0.3, 0.9):
        box((x, -0.02, 1.1), (0.06, 0.04, 0.06), rivet)


# ---------------------------------------------------------------- posts

@asset("post_meadow", "walls", ao=1.2)
def post_meadow():
    wood = M("pw", (140, 100, 64), var=0.12)
    iron = M("pi", (60, 50, 40))
    lamp = M("pl", (255, 220, 140), emit=1.2)
    cyl((0, 0, 0), 0.52, 2.9, wood, n=8, bev=0.03)
    cyl((0, 0, 2.9), 0.62, 0.15, wood, n=8)
    box((0, 0, 3.4), (0.6, 0.6, 0.6), iron)
    box((0, 0, 3.4), (0.46, 0.66, 0.46), lamp)
    box((0, 0, 3.4), (0.66, 0.46, 0.46), lamp)
    cyl((0, 0, 3.7), 0.42, 0.3, iron, n=4, r2=0.05)


@asset("post_ruins", "walls", ao=1.2)
def post_ruins():
    stone = M("prs", (196, 168, 124), var=0.1)
    gold = M("prg", (224, 184, 84))
    lathe([(0.95, 0), (0.95, 0.3), (0.72, 0.42), (0.66, 2.3), (0.9, 2.5), (0.92, 2.85), (0.4, 2.95), (0.0, 3.0)], stone, n=12,
          mats=[stone, stone, stone, gold, gold, gold, gold])
    for k in range(8):
        a = k / 8 * 2 * math.pi
        box((math.cos(a) * 0.68, math.sin(a) * 0.68, 1.35), (0.08, 0.08, 1.7), stone)


@asset("post_clouds", "walls", ao=1.0)
def post_clouds():
    marble = M("pcm", (238, 236, 248))
    roof = M("pcr", (120, 150, 236))
    gold = M("pcg", (250, 210, 120), emit=0.3)
    lathe([(0.82, 0), (0.82, 2.4), (1.0, 2.5), (1.0, 2.8), (0.95, 2.85)], marble, n=12)
    lathe([(1.12, 2.85), (0.7, 3.5), (0.25, 4.3), (0.0, 4.6)], roof, n=12)
    sphere((0, 0, 4.65), 0.12, gold, sub=1)
    for k in range(4):
        a = k / 4 * 2 * math.pi + math.pi / 4
        box((math.cos(a) * 0.82, math.sin(a) * 0.82, 1.6), (0.12, 0.12, 0.5), marble)


@asset("post_neon", "walls", ao=0.8)
def post_neon():
    body = M("pnb", (70, 56, 110))
    dark = M("pnd", (34, 26, 56))
    glow = M("png", (255, 90, 220), emit=1.2)
    cyan = M("pnc", (90, 240, 255), emit=1.2)
    cyl((0, 0, 0), 0.86, 0.3, dark, n=8)
    cyl((0, 0, 0.3), 0.7, 1.4, body, n=8)
    cyl((0, 0, 1.7), 0.78, 0.12, glow, n=8)
    cyl((0, 0, 1.82), 0.6, 0.4, dark, n=8)
    cyl((0, 0, 2.22), 0.32, 0.12, cyan, n=8)
    for k in range(4):
        a = k / 4 * 2 * math.pi
        box((math.cos(a) * 0.7, math.sin(a) * 0.7, 1.0), (0.06, 0.06, 1.2), cyan)


@asset("post_caverns", "walls", ao=1.0)
def post_caverns():
    rockm = M("pcr2", (80, 86, 116), var=0.15)
    cry = M("pcc", (150, 210, 255), tint=True, emit=0.5)
    rock((0, 0, 0.6), 0.9, rockm, seed=3, scale=(1, 1, 1.2))
    for a, h, tilt in ((0, 2.2, 0), (120, 1.4, 25), (240, 1.6, -22)):
        o = cyl((0, 0, 0), 0.22, h, cry, n=6, r2=0.0)
        place(o, loc=(math.cos(math.radians(a)) * 0.25, math.sin(math.radians(a)) * 0.25, 1.1), rot=(tilt, tilt * 0.5, a))


@asset("post_machine", "walls", ao=1.0)
def post_machine():
    steel = M("pms", (70, 72, 80))
    dark = M("pmd", (54, 56, 64))
    amber = M("pma", (255, 150, 40), emit=1.0)
    hazard = M("pmh", (220, 150, 40))
    lathe([(0.92, 0), (0.92, 0.3), (0.7, 0.36), (0.7, 1.8), (0.92, 1.86), (0.92, 2.1), (0.42, 2.2)], steel, n=10,
          mats=[dark, dark, steel, steel, hazard, dark])
    cyl((0, 0, 2.2), 0.3, 0.32, amber, n=10)
    cyl((0, 0, 2.52), 0.34, 0.06, dark, n=10)
    for k in range(6):
        a = k / 6 * 2 * math.pi
        sphere((math.cos(a) * 0.72, math.sin(a) * 0.72, 1.0), 0.06, dark, sub=1)


# ---------------------------------------------------------------- critters
# Critters face +Y (their walking direction), sit on z=0 and are built at
# their in-game size. Tinted materials take the world's critter colour.

def _eye(x, y, z, r=0.08):
    black = M("eye", (20, 16, 24))
    shine = M("shine", (255, 255, 255), emit=0.5)
    sphere((x, y, z), r, black, sub=1)
    sphere((x + r * 0.3, y + r * 0.65, z + r * 0.35), r * 0.38, shine, sub=1)


@asset("critter_meadow", "critters", ao=0.5)
def critter_meadow():
    """A fluffy rabbit."""
    fur = M("fur", (240, 240, 240), tint=True, var=0.05)
    pink = M("pink", (240, 170, 180))
    white = M("tail", (255, 255, 255))
    sphere((0, 0, 0.78), 0.75, fur, sub=2, scale=(0.95, 1.15, 0.82))
    sphere((0, 0.68, 1.4), 0.5, fur, sub=2, scale=(0.95, 1.0, 0.92))
    for sx in (-1, 1):
        e = sphere((sx * 0.2, 0.62, 2.0), 0.17, fur, sub=1, scale=(0.6, 0.45, 2.2))
        place(e, rot=(-12, sx * 10, 0))
        e2 = sphere((sx * 0.2, 0.55, 2.0), 0.1, pink, sub=1, scale=(0.5, 0.35, 1.9))
        place(e2, rot=(-12, sx * 10, 0))
        _eye(sx * 0.27, 1.08, 1.5, 0.09)
        sphere((sx * 0.38, 0.35, 0.12), 0.24, fur, sub=1, scale=(0.9, 1.6, 0.5))
    sphere((0, 1.16, 1.33), 0.06, pink, sub=1)
    sphere((0, -0.85, 0.95), 0.24, white, sub=1)


@asset("critter_ruins", "critters", ao=0.4)
def critter_ruins():
    """A scarab with a gilded shell seam (legs are animated in code)."""
    shell = M("shell", (240, 240, 240), tint=True, emit=0.1, var=0.04)
    dark = M("sdark", (40, 50, 50))
    gold = M("sgold", (255, 210, 90), emit=0.4)
    sphere((0, -0.05, 0.55), 0.82, shell, sub=2, scale=(0.95, 1.2, 0.58))
    box((0, -0.05, 0.98), (0.04, 1.6, 0.06), gold)
    sphere((0, 0.95, 0.45), 0.4, dark, sub=2, scale=(1.0, 0.75, 0.62))
    for sx in (-1, 1):
        o = cyl((0, 0, 0), 0.05, 0.5, dark, n=5, r2=0.01)
        place(o, loc=(sx * 0.18, 1.18, 0.55), rot=(-70, sx * 20, 0))
        _eye(sx * 0.2, 1.2, 0.6, 0.07)


@asset("critter_clouds", "critters", ao=0.4)
def critter_clouds():
    """A puff of cloud (its little wings are animated in code)."""
    puff = M("puff", (250, 250, 255), tint=True, emit=0.1)
    sphere((0, 0, 1.35), 0.75, puff, sub=2, scale=(1.0, 0.95, 0.82))
    for sx, z, r in ((-1, 1.22, 0.48), (1, 1.26, 0.5)):
        sphere((sx * 0.55, -0.05, z), r, puff, sub=2)
    sphere((0, -0.3, 1.75), 0.42, puff, sub=2)
    for sx in (-1, 1):
        _eye(sx * 0.22, 0.68, 1.45, 0.1)
    pink = M("blush", (255, 180, 200))
    for sx in (-1, 1):
        sphere((sx * 0.42, 0.62, 1.25), 0.08, pink, sub=1, scale=(1, 0.4, 0.7))


@asset("critter_neon", "critters", ao=0.4)
def critter_neon():
    """A hover drone with a glowing visor (rotor rings are animated in code)."""
    hull = M("hull", (90, 70, 150))
    visor = M("visor", (240, 240, 250), tint=True, emit=1.0)
    dark = M("ddark", (30, 24, 50))
    box((0, 0, 1.62), (1.36, 1.15, 0.5), hull, bev=0.12, seg=2)
    box((0, 0.58, 1.66), (0.86, 0.08, 0.18), visor)
    for sx in (-1, 1):
        for sy in (-1, 1):
            o = cyl((0, 0, 0), 0.06, 0.18, dark, n=6)
            place(o, loc=(sx * 0.8, sy * 0.68, 1.85))
            box((sx * 0.42, sy * 0.36, 1.85), (0.75, 0.08, 0.05), dark, rot=(0, 0, sy * sx * 40))
    o = cyl((0, 0, 0), 0.03, 0.4, dark, n=4)
    place(o, loc=(0, -0.2, 1.87))
    sphere((0, -0.2, 2.3), 0.07, visor, sub=1)


@asset("critter_caverns", "critters", ao=0.4)
def critter_caverns():
    """A crystal crab (legs are animated in code)."""
    shell = M("crab", (240, 240, 240), tint=True, emit=0.2, var=0.05)
    spike = M("cspike", (250, 250, 255), tint=True, emit=0.4)
    sphere((0, 0, 0.55), 0.9, shell, sub=2, scale=(1.0, 0.75, 0.48))
    for sx in (-1, 1):
        sphere((sx * 1.0, 0.6, 0.68), 0.3, shell, sub=2, scale=(1.0, 1.25, 0.75))
        o = box((sx * 0.75, 0.42, 0.62), (0.4, 0.12, 0.12), shell, rot=(0, 0, sx * -30))
        st = cyl((0, 0, 0), 0.035, 0.32, shell, n=5)
        place(st, loc=(sx * 0.26, 0.42, 0.78))
        _eye(sx * 0.26, 0.48, 1.12, 0.09)
    for k in range(4):
        o = cyl((0, 0, 0), 0.1, 0.45, spike, n=5, r2=0.0)
        place(o, loc=(-0.45 + k * 0.3, -0.1 + (k % 2) * 0.15, 0.88), rot=(-10 + k * 6, (k - 1.5) * 12, 0))


@asset("critter_machine", "critters", ao=0.4)
def critter_machine():
    """A clockwork beetle with copper plating (its key turns in code)."""
    brass = M("brass", (240, 240, 240), tint=True, var=0.05)
    dark = M("bdark", (70, 60, 50))
    red = M("bred", (255, 90, 60), emit=1.0)
    sphere((0, -0.05, 0.68), 0.78, brass, sub=2, scale=(0.95, 1.2, 0.62))
    box((0, -0.05, 1.12), (0.05, 1.6, 0.06), dark)
    for i in range(3):
        box((0, -0.5 + i * 0.45, 1.12), (1.1, 0.05, 0.05), dark)
    sphere((0, 0.9, 0.62), 0.36, dark, sub=2, scale=(1.0, 0.8, 0.75))
    for sx in (-1, 1):
        sphere((sx * 0.2, 1.17, 0.7), 0.09, red, sub=1)


@asset("critter_thief", "critters", ao=0.4)
def critter_thief():
    """A masked pilferer (its swag bag is drawn in code)."""
    fur = M("tfur", (240, 240, 240), tint=True, var=0.05)
    mask = M("tmask", (22, 20, 26))
    belly = M("tbelly", (236, 226, 210))
    sphere((0, 0, 0.82), 0.62, fur, sub=2, scale=(0.95, 1.0, 1.05))
    sphere((0, 0.36, 0.7), 0.42, belly, sub=1, scale=(0.9, 0.5, 1.0))
    sphere((0, 0.22, 1.6), 0.47, fur, sub=2)
    box((0, 0.6, 1.66), (0.75, 0.16, 0.18), mask, bev=0.04, seg=1)
    for sx in (-1, 1):
        _eye(sx * 0.17, 0.68, 1.68, 0.08)
        o = cyl((0, 0, 0), 0.14, 0.3, fur, n=5, r2=0.02)
        place(o, loc=(sx * 0.3, 0.12, 1.95), rot=(0, sx * 18, 0))
        sphere((sx * 0.3, 0.15, 0.1), 0.18, fur, sub=1, scale=(0.9, 1.5, 0.5))
    sphere((0, 0.68, 1.5), 0.07, mask, sub=1)
    tube([(0, -0.5, 0.6), (0, -0.95, 0.75), (0, -1.15, 1.1)], 0.16, fur, n=6, r_end=0.08)


@asset("critter_mason", "critters", ao=0.4)
def critter_mason():
    """A little stone golem in a hard hat (its brick is drawn in code)."""
    stone = M("gstone", (240, 240, 240), tint=True, var=0.12)
    hat = M("ghat", (255, 200, 60))
    rock((0, 0, 0.9), 0.75, stone, seed=21, scale=(1.0, 0.85, 1.05), sub=2, rough=0.12)
    rock((0, 0.1, 1.75), 0.42, stone, seed=22, sub=2, rough=0.1)
    lathe([(0.52, 2.0), (0.52, 2.05), (0.4, 2.08), (0.36, 2.3), (0.0, 2.38)], hat, n=10)
    for sx in (-1, 1):
        _eye(sx * 0.15, 0.48, 1.8, 0.07)
        rock((sx * 0.75, 0.05, 1.0), 0.26, stone, seed=23 + sx, sub=1, rough=0.1)
        rock((sx * 0.32, 0.0, 0.18), 0.25, stone, seed=25 + sx, sub=1, rough=0.1)


@asset("bird_body", "critters", ao=0.3)
def bird_body():
    """The bomber's body, facing +Y, centred on the origin; wings are separate."""
    body = M("bbody", (240, 240, 240), tint=True)
    beak = M("bbeak", (255, 190, 60), emit=0.3)
    sphere((0, 0, 0), 0.4, body, sub=2, scale=(0.7, 1.35, 0.7))
    sphere((0, 0.55, 0.15), 0.26, body, sub=2)
    o = cyl((0, 0, 0), 0.09, 0.3, beak, n=6, r2=0.0)
    place(o, loc=(0, 0.75, 0.13), rot=(-90, 0, 0))
    for sx in (-1, 1):
        _eye(sx * 0.14, 0.72, 0.24, 0.05)
    tail = prism([(-0.25, -0.5), (0.25, -0.5), (0.3, -0.95), (-0.3, -0.95)], 0.06, body, z0=0.0)


@asset("bird_wing", "critters", ao=0.2)
def bird_wing():
    """One wing reaching out along +X from the shoulder at the origin."""
    wing = M("bwing", (225, 225, 225), tint=True)
    prism([(0, -0.25), (0.6, -0.32), (1.25, -0.2), (1.45, 0.05), (0.8, 0.2), (0, 0.25)], 0.05, wing, z0=-0.025)
