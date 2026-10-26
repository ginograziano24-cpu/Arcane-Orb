"""Render server: leave running. Claude drops jobs/<id>.job files (one task
name per line); each job reloads the scripts, runs, and writes
out/<id>.log. Finished jobs are renamed to .done."""
import os, sys, time, importlib, io, traceback, contextlib
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
JOBS = os.path.join(ROOT, "jobs")
OUTD = os.path.join(ROOT, "out")
os.makedirs(JOBS, exist_ok=True)
os.makedirs(OUTD, exist_ok=True)


class Tee(io.TextIOBase):
    def __init__(self, *s):
        self.s = s

    def write(self, x):
        for s in self.s:
            try:
                s.write(x)
                s.flush()
            except Exception:
                pass
        return len(x)

    def flush(self):
        pass


def run_job(path):
    jid = os.path.splitext(os.path.basename(path))[0]
    tasks = [l.strip() for l in open(path, encoding="utf-8", errors="ignore").read().replace(",", " ").split() if l.strip()]
    os.replace(path, path + ".running")
    logp = os.path.join(OUTD, jid + ".log")
    with open(logp, "w", encoding="utf-8") as lf:
        tee = Tee(sys.__stdout__, lf)
        with contextlib.redirect_stdout(tee), contextlib.redirect_stderr(tee):
            print("[job]", jid, "tasks:", tasks, flush=True)
            t0 = time.time()
            try:
                import kit, backdrops
                importlib.reload(kit)
                importlib.reload(backdrops)
                import run as runmod
                importlib.reload(runmod)
                ok = 0
                for t in tasks:
                    fn = runmod.TASKS.get(t)
                    if fn is None:
                        print("[job] unknown task", t)
                        continue
                    ok += kit.run_task(t, fn)
                print("[job] finished %d/%d in %.0fs" % (ok, len(tasks), time.time() - t0), flush=True)
            except Exception:
                traceback.print_exc()
            print("[job] END", flush=True)
    os.replace(path + ".running", path + ".done")


print("=" * 60)
print(" Arcane Orb render server is running.")
print(" Leave this window open - Claude sends render jobs to it.")
print(" Close the window to stop.")
print("=" * 60, flush=True)
with open(os.path.join(OUTD, "server_alive.txt"), "w") as f:
    f.write(time.strftime("%Y-%m-%d %H:%M:%S"))
while True:
    try:
        jobs = sorted(p for p in os.listdir(JOBS) if p.endswith(".job"))
        for j in jobs:
            run_job(os.path.join(JOBS, j))
        with open(os.path.join(OUTD, "server_alive.txt"), "w") as f:
            f.write(time.strftime("%Y-%m-%d %H:%M:%S"))
    except Exception:
        traceback.print_exc()
    time.sleep(2)
