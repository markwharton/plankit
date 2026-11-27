"""Apply a candidate edit to copies of pk.json and git.json, play them, and print the findings that changed."""
import json, subprocess, sys, os, copy, tempfile
HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.normpath(os.path.join(HERE, "..", "..", ".."))
ENG = REPO + "/lab/engine"
OUT = tempfile.mkdtemp(prefix="evidence-")
base = {n: json.load(open(f"{ENG}/{n}.json")) for n in ("pk", "git")}

def play(name, pk, git):
    d = f"{OUT}/{name}"; os.makedirs(d, exist_ok=True)
    json.dump(pk, open(f"{d}/pk.json", "w"), indent=2); json.dump(git, open(f"{d}/git.json", "w"), indent=2)
    r = subprocess.run([sys.executable, f"{ENG}/engine.py", f"{d}/pk.json", "--root", REPO, "--lines"], capture_output=True, text=True)
    lines = [l for l in r.stdout.splitlines() if l.strip()]
    return {l.split(":")[0]: l for l in lines}, r

def part(pk, name): return pk["parts"][name]
def find(seq_owner, key, val):
    for x in seq_owner:
        if isinstance(x, dict) and x.get(key) == val: return x
def fact(design, name): return find(design["facts"], "name", name)

B, _ = play("baseline", base["pk"], base["git"])
EDITS = {}
def edit(f): EDITS[f.__name__] = f; return f
exec(open(f"{HERE}/edits.py").read())
for n in (sys.argv[1:] or EDITS):
    pk, git = copy.deepcopy(base["pk"]), copy.deepcopy(base["git"])
    EDITS[n](pk, git)
    F, r = play(n, pk, git)
    if r.returncode not in (0, 2) or (not F and r.stderr): print(f"== {n}: engine refused: {r.stderr.strip()[-400:]}"); continue
    gone = [k for k in B if k not in F]; new = [k for k in F if k not in B]
    print(f"== {n}: {len(F)} findings (baseline {len(B)}); gone: {gone or '-'}")
    for k in new: print("   NEW " + F[k][:600])
