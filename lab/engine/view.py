"""The engine's plays, laid out the way the lab's page draws a design.

The page was built for the checker's model: a tree of elements, places,
and for each step one trace for each place broken. This turns a design
and its plays into that shape, so the same five views draw it. A choice
of flags stands where the page has a path, and the one fault of a play
stands where it has a broken place.
"""
import engine

SHAPE = {"seq": "join", "all": "include", "choose": "or"}
STATUS = {"ran": "done", "refused": "refused", "failed": "refused", "warned": "warned", "skipped": "not-run", "declined": "declined"}


def view_model(d, plays, findings, tests, shown):
    system = d.doc["system"]
    nodes, used, names = [], {system} | set(d.doc["runs"]), {}     # names[run][action] -> the node's name; a run's own name is taken

    def unique(base, run):
        name, i = base, 2
        if name in used:
            name = "%s (in %s)" % (base, run)
        while name in used:
            name, i = "%s (in %s, %d)" % (base, run, i), i + 1
        used.add(name)
        return name

    def build(part, parent, run, hint, top=False, gates=frozenset()):
        while isinstance(part, str) or "use" in part:
            if isinstance(part, dict):
                gates = gates | engine.names_in(part.get("when"))
            hint = part if isinstance(part, str) else part["use"]
            part = d.parts[hint]
        gates = gates | engine.names_in(part.get("when"))
        node = {"name": None, "shape": "leaf", "parent": parent, "children": [], "branches": [], "otherwise": "", "reads": [], "writes": [],
                "owns": [], "refusals": [], "findings": []}
        if "do" in part:
            node["name"] = run if top else unique(part["do"], run)
            names[run][part["do"]] = node["name"]
            reads, writes = d.touches(part)                # for an action that uses a run below, the run's, with the places it was given
            node["reads"] = list(reads)
            node["writes"] = [{"place": place, "mode": how + (", by " + part["uses"] if "uses" in part else ""), "key": "", "records": "", "before": []} for place, how in writes]
            node["refusals"] = [{"kind": h["then"], "reason": ("when " + engine.words(h["when"]) if h.get("when") is not None else "always") + (", because " + h["because"] if h.get("because") else ""), "to": "" if h["then"] == "accept" else (d.told(h) or [""])[0],
                                 "held": engine.held(d, h.get("hold")), "test": (h.get("hold") or {}).get("test", "")} for h in part.get("handles", [])]
            node["label"] = ", ".join(w["place"] for w in node["writes"]) + " = " + part["do"] + "(" + ", ".join(node["reads"]) + ")"
            # who answers what: the facts each answer names, the faults the action meets, and the facts that decide whether it runs
            node["answers"] = [{"facts": sorted(engine.names_in(h.get("when"))), "then": h["then"]} for h in part.get("handles", [])]
            meets = [f for p in node["reads"] for f in d.about(p)]
            for c in part.get("changes", []):
                meets += d.brings(c) + (d.about(c["from"]) if "from" in c else [])
            if "uses" in part:
                meets += [f for f in d.comes_up(part) if d.is_fault(f)]
            node["meets"] = sorted(set(meets) | {f for a in node["answers"] for f in a["facts"] if d.is_fault(f)})
            own = set(gates) | engine.names_in(part.get("rehearsal"))
            for it in part.get("reads", []) + part.get("changes", []):
                if isinstance(it, dict):
                    own |= engine.names_in(it.get("when"))
            node["asks"] = sorted(f for f in own if not d.facts[f].get("run"))
            nodes.append(node)
            return node["name"]
        kind = next(k for k in SHAPE if k in part)
        node["shape"] = SHAPE[kind]
        node["name"] = run if top else unique(part.get("name") or hint or {"seq": "in order", "all": "in no order", "choose": "one of"}[kind], run)
        nodes.append(node)
        if kind == "choose":
            for b in part["choose"]:
                child = build(b["run"], node["name"], run, None, gates=gates | engine.names_in(b.get("when")))
                node["children"].append(child)
                node["branches"].append({"when": engine.words(b.get("when")), "child": child})
            if part.get("otherwise") is not None:
                node["otherwise"] = build(part["otherwise"], node["name"], run, None, gates=gates | set().union(*[engine.names_in(b.get("when")) for b in part["choose"]]))
                node["children"].append(node["otherwise"])
        else:
            node["children"] = [build(k, node["name"], run, None, gates=gates) for k in part[kind]]
        return node["name"]

    root = {"name": system, "shape": "steps", "children": [], "branches": [], "otherwise": "", "reads": [], "writes": [], "owns": [], "refusals": [], "findings": []}
    nodes.append(root)
    for run in d.doc["runs"]:
        names[run] = {}
        root["children"].append(build(run, system, run, run, top=True))

    kinds = {"outside": "outside", "lasting": "lasting", "run": "in a run", "report": "in a run"}
    leaves = [n for n in nodes if n["shape"] == "leaf"]
    places = [{"name": p["name"], "owner": p.get("owner", ""), "kind": kinds[p["kind"]], "keyed": p.get("is") == "many", "onExisting": "",
               "invalid": "; ".join(d.facts[f].get("says", f) for f in d.about(p["name"])),
               "writers": [n["name"] for n in leaves if any(w["place"] == p["name"] for w in n["writes"])],
               "readers": [n["name"] for n in leaves if p["name"] in n["reads"]], "writesFromOutside": [], "findings": []} for p in d.places.values()]

    def old(place, state, rests):
        """A place's state in the words the page colours."""
        if rests:
            return "invalid"
        if state in ("untouched", "undone"):
            return {"outside": "given", "lasting": "present"}.get(d.places[place]["kind"], "empty")
        return "written"

    runs = []
    for run in d.doc["runs"]:
        mine = [p for p in plays if p.run == run]
        touched = [p["name"] for p in d.places.values() if any(p["name"] in n["reads"] or any(w["place"] == p["name"] for w in n["writes"])
                                                           for n in leaves if n["name"] in names[run].values())]
        rv = {"run": run, "paths": [], "traces": [], "baselines": [], "faults": []}
        flags_of = lambda p: tuple(f for f in p.given() if not d.is_fault(f))
        faults_of = lambda p: [f for f in p.given() if d.is_fault(f)]
        order = []
        for p in sorted(mine, key=lambda p: (len(faults_of(p)), flags_of(p))):
            if flags_of(p) not in order:
                order.append(flags_of(p))
        for i, flags in enumerate(order):
            rv["paths"].append([{"or": "given", "child": " and ".join(flags) or "nothing special"}])
            group = [p for p in mine if flags_of(p) == flags]
            group.sort(key=lambda p: len(faults_of(p)))
            for j, p in enumerate(group):
                bad = set(faults_of(p))
                wrong_at = {f: d.facts[f].get("about") for f in bad}
                went_on = {a for a, f in p.open}
                start = [{"place": x, "state": "invalid" if x in wrong_at.values() else old(x, "untouched", False)} for x in touched]
                steps = []
                for a, status in p.steps:
                    if a not in names[run]:
                        continue
                    seen = [{"place": x, "state": "invalid" if (x in wrong_at.values() and s == "untouched") else old(x, s, r)} for x, s, r in p.saw.get(a, [])]
                    made = [{"place": x, "state": old(x, s, r)} for x, s, r in p.did.get(a, [])] if status in ("ran", "failed") else []
                    steps.append({"element": names[run][a], "outcome": "passed-on" if a in went_on and status == "ran" else STATUS[status], "reads": seen, "writes": made})
                told = (d.told(p.stop.handle) or [""])[0] if p.stop else ""
                refusals = [dict({"kind": "rejects", "from": names[run][p.stop.action], "through": []}, **({"to": told} if told else {"lostAt": names[run][p.stop.action]}))] if p.stop else []
                end = [{"place": x, "state": "refusal" if p.stop and x == told else old(x, p.state[x], bool(p.rests[x]))} for x in touched]
                rv["traces"].append({"run": run, "break": {}, "path": rv["paths"][i], "start": start, "steps": steps, "end": end, "refusals": refusals, "given": p.given()})
                if j == 0:
                    rv["baselines"].append(len(rv["traces"]) - 1)
                if not bad:
                    continue
                if p.stop and p.stop.how == "failed":
                    verdict, what = "refused", "%s stops the run, and %s." % (p.stop.action, "the person is told in " + told if told else "nobody is told")
                elif p.stop:
                    verdict, what = "declined", "The person says no at %s, and the run stops." % p.stop.action
                elif went_on:
                    rest = [x for x, fs in p.rests.items() if fs and p.state[x] not in ("untouched", "undone")]
                    verdict, what = "passed", "%s on with it, and nothing answers it.%s" % (", ".join(sorted(went_on)) + (" goes" if len(went_on) == 1 else " go"),
                                                                                         " What the run changes then rests on it: %s." % ", ".join(rest) if rest else "")
                elif p.warned:
                    verdict, what = "warned", "%s warns, does nothing, and the run goes on." % ", ".join(p.warned)
                else:
                    verdict, what = "answered", "An action answers it, and the run goes on."
                rv["faults"].append({"place": " and ".join(sorted(bad)), "path": i, "verdict": verdict, "refusedAt": p.stop.action if p.stop else "",
                                     "held": bool(p.stop and engine.held(d, p.stop.handle.get("hold"))), "trace": len(rv["traces"]) - 1, "what": what,
                                     "says": "; ".join(d.facts[f].get("says", f) for f in sorted(bad))})
        runs.append(rv)

    met = {f for n in leaves for f in n.get("meets", []) + n.get("asks", []) + [x for a in n.get("answers", []) for x in a["facts"]]}
    facts = [{"name": f["name"], "fault": bool(f.get("fault")), "open": bool(f.get("open")), "about": f.get("about", ""), "says": f.get("says", "")}
             for f in d.facts.values() if not f.get("run") and (not f.get("built in") or f["name"] in met)]
    return {"system": system, "definition": shown, "descriptions": [], "nodes": nodes, "places": places, "runs": runs, "facts": facts,
            "labels": {"path": "Given", "break": "Wrong", "check": "the engine"},
            "findings": [{"check": f["kind"], "subject": f["subject"], "message": " ".join(f["shows"]) + " " + engine.seen_in(f) + " To settle it: " + f["settle"], "accepted": False} for f in findings],
            "tests": [{"kind": t["what"], "subject": t["subject"], "words": "", "inputs": [], "firstRefusers": [], "input": "the facts it names hold",
                       "asserts": "what the statement says happens", "break": "take the statement's behaviour out of the code. The test fails.",
                       "baseline": "the code as it is: the test passes.", "unwritten": False, "run": ""} for t in tests if not t["held"]]}


def check(doc, root, shown, base=None):
    """What the lab's server returns for a design in this form. base is the directory of the design's file."""
    try:
        d = engine.Design(doc, root, base)
        plays = engine.explore_all(d)
    except engine.Refused as e:
        return {"ok": False, "refused": ["%s: %s" % (shown, e)]}
    fs, ts = engine.findings(d, plays), engine.tests(d)
    lines = ["%s  %s: %s" % (f["kind"], f["subject"], " ".join(f["shows"])) for f in fs]
    summary = "%s: %d plays, %d questions; %d statements a test should hold, %d with a test found" % (shown, len(plays), len(fs), len(ts), sum(1 for t in ts if t["held"]))
    return {"ok": True, "model": view_model(d, plays, fs, ts, shown), "viewSummary": summary, "check": {"exit": 2 if fs else 0, "lines": lines, "summary": summary}}
