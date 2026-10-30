"""Entry point: blender -b --factory-startup -P scripts/run.py -- <task>"""
import os, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import bpy
import kit
from kit import log


def smoke():
    import math, random
    log("Blender", bpy.app.version_string, "| python", sys.version.split()[0])
    kit.reset()
    backend = kit.setup_render(960, 540, samples=64)
    log("render backend:", backend)
    kit.world_gradient([(0.45, (0.9, 0.5, 0.35)), (0.52, (1.0, 0.75, 0.5)), (0.7, (0.35, 0.5, 0.85)), (1.0, (0.1, 0.18, 0.45))],
                       strength=1.0, sun_dir=(0.3, 1, 0.12), sun_str=30)
    kit.world_fog(0.004, (0.95, 0.8, 0.7), height=40)
    ground = kit.mat_color("grass", (0.2, 0.45, 0.12), rough=0.9, noise=0.4, noise_scale=0.6, bump_str=0.3)
    rnd = random.Random(1)

    def h(x, y):
        return 3 * math.sin(x * 0.05) * math.cos(y * 0.04) + 1.5 * math.sin(x * 0.13 + y * 0.07) + 0.02 * max(0, y - 40) ** 1.3

    kit.grid_terrain("Ground", 400, 300, 120, 90, h, ground)
    leaf = kit.mat_color("leaf", (0.12, 0.35, 0.08), rough=0.8, noise=0.5, noise_scale=3, bump_str=0.5)
    bark = kit.mat_color("bark", (0.25, 0.15, 0.08), rough=0.9)
    for i in range(14):
        x, y = rnd.uniform(-60, 60), rnd.uniform(20, 120)
        z = h(x, y)
        kit.prim('cyl', (x, y, z + 2), (0.35, 0.35, 2), mat=bark, vertices=8)
        c = kit.prim('ico', (x, y, z + 5), (2.2, 2.2, 1.8), mat=leaf, subdivisions=3)
        kit.displace(c, 0.6, 0.6)
    kit.sun((60, 0, 160), strength=3.5, color=(1.0, 0.85, 0.7))
    kit.camera((0, -10, 6), (90, 0, 0), lens=30, shift_y=0.12375)
    kit.render_to(os.path.join(kit.OUT, "smoke.png"))


TASKS = {
    "smoke": smoke,
}
import importlib
import backdrops
TASKS.update(backdrops.TASKS)
import models
importlib.reload(models)
TASKS.update(models.TASKS)
import icons
importlib.reload(icons)
TASKS.update(icons.TASKS)


def main():
    argv = sys.argv[sys.argv.index("--") + 1:] if "--" in sys.argv else []
    task = " ".join(argv).lower().strip() or "smoke"
    t0 = time.time()
    if task == "all":
        names = [n for n in TASKS if n != "smoke"]
    else:
        names = [n for n in task.replace("+", " ").replace(",", " ").split() if n in TASKS]
    if not names:
        log("unknown task", task, "- available:", ", ".join(TASKS))
        return
    ok = 0
    for n in names:
        ok += kit.run_task(n, TASKS[n])
    log("finished %d/%d tasks in %.0fs" % (ok, len(names), time.time() - t0))


if not os.environ.get("ARTKIT_SERVER"):
    main()
