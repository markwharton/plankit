"""Reads a definition and the ones it stands on. It checks that every
name resolves, and lists what comes up to each action that uses a run
below: answered there, answered by another action, or not answered.
It plays nothing. Usage: python3 check.py <definition.json>"""
import json, sys, os

def names_in(c):
    if c is None: return []
    if isinstance(c, str): return [c]
    (k, v), = c.items()
    return names_in(v) if k == "not" else [n for x in v for n in names_in(x)]

BROUGHT = ["%s already has the member", "%s is given a key twice", "%s holds another's member",
           "%s holds what is kept nowhere", "%s is held elsewhere"]

def actions(d):
    out = []
    def go(p):
        if isinstance(p, str) or p is None: return
        if "do" in p: out.append(p)
        for k in ("seq", "all"):
            for c in p.get(k, []): go(c)
        for b in p.get("choose", []): go(b["run"])
        if "choose" in p: go(p.get("otherwise"))
    for p in d["parts"].values(): go(p)
    return out

def under(d, part, seen=None):
    """The actions of one part of a definition, following uses by name."""
    seen = set() if seen is None else seen
    if isinstance(part, str):
        if part in seen: return []
        seen.add(part); part = d["parts"][part]
    if part is None: return []
    if "use" in part: return under(d, part["use"], seen)
    if "do" in part: return [part]
    kids = part.get("seq", []) + part.get("all", []) + [b["run"] for b in part.get("choose", [])] + ([part["otherwise"]] if "choose" in part else [])
    return [a for k in kids for a in under(d, k, seen)]

top = json.load(open(sys.argv[1]))
base = os.path.dirname(os.path.abspath(sys.argv[1]))
below = [json.load(open(os.path.join(base, p))) for p in top.get("on", [])]
places = {p["name"]: p for d in below + [top] for p in d["places"]}
facts = {f["name"]: f for d in below + [top] for f in d["facts"]}
brought = {b % p for p in places for b in BROUGHT}
runs_below = {r: d for d in below for r in d["runs"]}
known = set(facts) | brought | {"in " + r for r in top["runs"]}

bad = []
for a in actions(top) + [a for d in below for a in actions(d)]:
    for h in a.get("handles", []):
        bad += [(a["do"], n) for n in names_in(h.get("when")) if n not in known]
    for r in a.get("reads", []):
        n = r if isinstance(r, str) else r["place"]
        if n not in places: bad.append((a["do"], n))
    for c in a.get("changes", []):
        for n in (c["place"], c.get("from")):
            if n and n not in places: bad.append((a["do"], n))
    for g, mine in a.get("with", {}).items():
        if g not in places or not places[g].get("given") or mine not in places: bad.append((a["do"], g + " -> " + mine))
    if "uses" in a and a["uses"] not in runs_below: bad.append((a["do"], "uses " + a["uses"]))
print("names that do not resolve:", bad or "none")

rows, used, seen_rows = [], set(), set()

def come_up(a, before):
    """What comes up to one action that uses a run below, and who answers it."""
    used.add(a["uses"])
    bind = a.get("with", {})
    for h in [h for low in under(runs_below[a["uses"]], a["uses"]) for h in low.get("handles", [])]:
        kind = "failure" if h["then"] in ("refuse", "fail") else "went on (%s)" % h["then"]
        for n in names_in(h.get("when")):
            if facts.get(n, {}).get("open"): continue
            for g, m in bind.items():                      # only a fact a place brings is renamed
                if n in brought and n.startswith(g + " "): n = m + n[len(g):]
            here = any(n in names_in(x.get("when")) for x in a.get("handles", []))
            row = (a["do"], n, kind, "here" if here else "earlier, by " + ", ".join(sorted(before[n])) if n in before else "NOT ANSWERED")
            if row not in seen_rows:
                seen_rows.add(row); rows.append(row)

def walk(part, before):
    """Walks a part in order. before: fact -> the earlier actions that answered it.
    Returns what this part answers. Children of an all do not answer for each other."""
    if isinstance(part, str): part = top["parts"][part]
    if part is None: return {}
    if "use" in part: return walk(part["use"], before)
    if "do" in part:
        if "uses" in part: come_up(part, before)
        return {n: {part["do"]} for h in part.get("handles", []) for n in names_in(h.get("when"))}
    out = {}
    def add(d):
        for k, v in d.items(): out.setdefault(k, set()).update(v)
    if "seq" in part:
        now = {k: set(v) for k, v in before.items()}
        for c in part["seq"]:
            got = walk(c, now); add(got)
            for k, v in got.items(): now.setdefault(k, set()).update(v)
    for c in part.get("all", []): add(walk(c, before))
    if "choose" in part:
        for c in [b["run"] for b in part["choose"]] + [part.get("otherwise")]: add(walk(c, before))
    return out

for r in top["runs"]: walk(r, {})
for r in rows: print("  %-32s %-58s %-18s %s" % r)
n = lambda f: sum(1 for r in rows if f(r))
print("comes up: %d (failures %d, went on %d). answered here %d, by an earlier action %d, not answered %d" % (
    len(rows), n(lambda r: r[2] == "failure"), n(lambda r: r[2] != "failure"),
    n(lambda r: r[3] == "here"), n(lambda r: r[3].startswith("earlier")), n(lambda r: r[3] == "NOT ANSWERED")))
print("runs below not used:", sorted(set(runs_below) - used) or "none")
print("facts listed here: %d (%d faults, %d open). facts listed below: %d" % (
    len(top["facts"]), sum(1 for f in top["facts"] if f.get("fault")), sum(1 for f in top["facts"] if f.get("open")),
    sum(len(d["facts"]) for d in below)))
