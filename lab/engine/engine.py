#!/usr/bin/env python3
"""A design you can run.

A design says what a system's runs do. The engine plays every run the
design allows and reports each one that leaves a question open. With
--against it plays a second design, written from the code, beside the
first and reports where the two differ.

    python3 engine.py DESIGN.json [--root DIR] [--against CODE.json] [--lines | --json]

Standard library only. DESIGN.md beside this file is the reference.
"""
import argparse
import json
import os
import sys
import textwrap

KEYS = {
    "design": {"system", "reports", "faults", "places", "facts", "parts", "runs", "expects", "rules"},
    "place": {"name", "kind", "keyed", "owner"},
    "fact": {"name", "says", "fault", "open", "about", "at", "only"},
    "action": {"do", "when", "reads", "changes", "handles", "says"},
    "seq": {"seq", "name", "when", "handles", "says"},
    "all": {"all", "name", "when", "handles", "says"},
    "choose": {"choose", "otherwise", "name", "when", "handles", "says"},
    "use": {"use", "when", "name", "says"},
    "branch": {"when", "run"},
    "read": {"place", "when"},
    "change": {"place", "how", "when", "later", "many", "makes", "because", "hold", "says"},
    "handle": {"when", "then", "no", "tells", "because", "hold", "says"},
    "accepted": {"accepted"},
    "acceptance": {"finding", "because"},
    "expect": {"name", "in", "when", "on", "never changes", "never creates", "ends", "runs", "never runs", "before", "place", "is", "because", "hold", "says"},
    "rule": {"rule", "about", "when", "hold", "says"},
    "hold": {"file", "test"},
}
KINDS = ("outside", "lasting", "run", "report")
HOWS = {"set": "set", "create": "created", "replace": "replaced", "delete": "deleted"}
THENS = ("refuse", "fail", "warn", "tell", "ask", "accept")
PART_THENS = ("warn", "tell", "accept")       # how a part answers a failure one of its children returns
STATES = {"untouched", "set", "created", "replaced", "deleted", "undone", "changed", "unchanged"}
ONS = ("any", "success", "failed", "declined")


class Refused(Exception):
    """A design the engine cannot play. The message names what to fix."""


class Need(Exception):
    """A play asked for a fact that has not been decided yet."""

    def __init__(self, name):
        self.name = name


class Failure:
    """What an action returns to the part that holds it when it refuses,
    fails, or a person says no. how is "failed" or "declined"."""

    def __init__(self, how, action, handle):
        self.how, self.action, self.handle = how, action, handle


# ---------- facts ----------

def names_in(c):
    if c is None:
        return set()
    if isinstance(c, str):
        return {c}
    if isinstance(c, dict) and len(c) == 1:
        (k, v), = c.items()
        if k == "not":
            return names_in(v)
        if k in ("all", "any") and isinstance(v, list):
            return set().union(*[names_in(x) for x in v]) if v else set()
    raise Refused("where a fact is asked for, give a name, {\"not\": ...}, {\"all\": [...]} or {\"any\": [...]}; got %s" % json.dumps(c))


def holds(c, env):
    if c is None:
        return True
    if isinstance(c, str):
        if c not in env:
            raise Need(c)
        return env[c]
    (k, v), = c.items()
    if k == "not":
        return not holds(v, env)
    return all(holds(x, env) for x in v) if k == "all" else any(holds(x, env) for x in v)


def assume(c, env, want=True):
    """The fewest undecided facts to decide so that c is want, given env;
    or None when env already rules it out."""
    if c is None:
        return {} if want else None
    if isinstance(c, str):
        if c in env:
            return {} if env[c] == want else None
        return {c: want}
    (k, v), = c.items()
    if k == "not":
        return assume(v, env, not want)
    if (k == "all") == want:               # every part must be want
        out = {}
        for x in v:
            got = assume(x, dict(env, **out), want)
            if got is None:
                return None
            out.update(got)
        return out
    for x in v:
        got = assume(x, env, want)
        if got is not None:
            return got
    return None


# ---------- reading a design ----------

def only(obj, kind, where):
    if not isinstance(obj, dict):
        raise Refused("%s: an object was expected" % where)
    extra = sorted(set(obj) - KEYS[kind])
    if extra:
        raise Refused("%s: the key %s is not one a %s has (%s)" % (where, ", ".join('"%s"' % k for k in extra), kind, ", ".join(sorted(KEYS[kind]))))


def already(place):
    return "%s already has the record" % place


def twice(place):
    return "%s is given a key twice" % place


def anothers(place):
    return "%s holds another's record" % place


def listing(x):
    """One name or a list of names, as a list."""
    return [] if x is None else [x] if isinstance(x, str) else list(x)


def in_run(run):
    return "in %s" % run


def yes_at(action):
    return "the person says yes at %s" % action


class Design:
    def __init__(self, doc, root="."):
        only(doc, "design", "the design")
        for k in ("system", "places", "parts", "runs"):
            if k not in doc:
                raise Refused("the design has no \"%s\"" % k)
        self.doc, self.root = doc, root
        self.budget = doc.get("faults", 1)
        if not isinstance(self.budget, int) or self.budget < 0:
            raise Refused("\"faults\" is how many things may go wrong in one play: a whole number")
        self.places, self.facts, self.actions, self.inside = {}, {}, {}, {}
        for p in doc["places"]:
            only(p, "place", "a place")
            if p.get("kind") not in KINDS:
                raise Refused("place \"%s\": its kind is one of %s" % (p.get("name"), ", ".join(KINDS)))
            if p["name"] in self.places:
                raise Refused("two places are named \"%s\"" % p["name"])
            self.places[p["name"]] = p
        for f in doc.get("facts", []):
            only(f, "fact", "a fact")
            if "name" not in f:
                raise Refused("a fact has no name")
            if f["name"] in self.facts:
                raise Refused("two facts are named \"%s\"" % f["name"])
            if f.get("open") and f.get("fault"):
                raise Refused("the fact \"%s\" is open and a fault: an open fact is a decision nobody has made yet, not something that goes wrong" % f["name"])
            if "about" in f:
                self.check_place(f["about"], "the fact \"%s\"" % f["name"])
            self.facts[f["name"]] = f
        for p in self.places.values():          # what can go wrong with any keyed place
            if p.get("keyed"):
                for name, says in [(already(p["name"]), "a record is already at the key"), (twice(p["name"]), "one key is given twice in the run"),
                                   (anothers(p["name"]), "the record at the key belongs to something else")]:
                    self.facts.setdefault(name, {"name": name, "fault": True, "about": p["name"], "says": says, "built in": True})
        for r in doc["runs"]:                   # which run it is, is a fact too
            self.facts.setdefault(in_run(r), {"name": in_run(r), "says": "the run is %s" % r, "built in": True, "run": r})
        if self.reports() is not None:
            self.check_place(self.reports(), "\"reports\"")
        self.parts = doc["parts"]
        for name, part in self.parts.items():
            self.check_part(part, "part \"%s\"" % name)
        for f in list(self.facts.values()):
            self.check_fact(f.get("only"), "the fact \"%s\"" % f["name"])
        for name in self.parts:                 # no part uses itself, directly or through another
            seen, todo = set(), list(self.uses(self.parts[name]))
            while todo:
                u = todo.pop()
                if u == name:
                    raise Refused("part \"%s\" uses a part which uses it in turn" % name)
                if u not in seen:
                    seen.add(u)
                    todo += self.uses(self.parts[u])
        for r in doc["runs"]:
            if r not in self.parts:
                raise Refused("the run \"%s\" is not a part" % r)
        for name in self.parts:
            self.inside[name] = set(self.actions_of(name))
        self.at = {}                            # action -> the faults that go wrong at it
        for f in self.facts.values():
            for a in listing(f.get("at")):
                if a not in self.actions:
                    raise Refused("the fact \"%s\" goes wrong at \"%s\", which is not an action" % (f["name"], a))
                self.at.setdefault(a, []).append(f["name"])
        for p in self.places.values():
            if "owner" in p and p["owner"] not in self.inside:
                raise Refused("place \"%s\": its owner \"%s\" is not a part" % (p["name"], p["owner"]))
        for e in doc.get("expects", []):
            self.check_expect(e)
        for r in doc.get("rules", []):
            only(r, "rule", "a rule")
            self.check_fact(r.get("when"), "the rule \"%s\"" % r.get("rule"))

    def reports(self):
        return self.doc.get("reports")

    def told(self, h):
        """The places where an answer tells the person: its own, or the design's."""
        return listing(h["tells"]) if "tells" in h else listing(self.reports())

    def check_place(self, name, where):
        if name not in self.places:
            raise Refused("%s names the place \"%s\", which the design does not list" % (where, name))

    def check_fact(self, c, where):
        for n in names_in(c):
            if n not in self.facts:
                raise Refused("%s names the fact \"%s\", which the design does not list" % (where, n))

    def check_expect(self, e):
        only(e, "expect", "an expectation")
        if "name" not in e:
            raise Refused("an expectation has no name")
        where = "the expectation \"%s\"" % e["name"]
        self.check_fact(e.get("when"), where)
        if e.get("in") is not None and e["in"] not in self.doc["runs"]:
            raise Refused("%s: \"in\" names \"%s\", which is not a run" % (where, e["in"]))
        if e.get("on", "any") not in ONS:
            raise Refused("%s: \"on\" is one of %s" % (where, ", ".join(ONS)))
        for p in e.get("never changes", []) + e.get("never creates", []) + list(e.get("ends", {})) + ([e["place"]] if "place" in e else []):
            self.check_place(p, where)
        for s in list(e.get("ends", {}).values()) + ([e["is"]] if "is" in e else []):
            if s not in STATES:
                raise Refused("%s: a state is one of %s" % (where, ", ".join(sorted(STATES))))
        for a in e.get("runs", []) + e.get("never runs", []) + ([e["before"]] if "before" in e else []):
            if a not in self.actions:
                raise Refused("%s names \"%s\", which is not an action" % (where, a))
        if not any(k in e for k in ("never changes", "never creates", "ends", "runs", "never runs", "before")):
            raise Refused("%s says nothing to check: give \"never changes\", \"never creates\", \"ends\", \"runs\", \"never runs\" or \"before\"" % where)

    def check_part(self, part, where):
        if isinstance(part, str):
            part = {"use": part}
        if "use" in part:
            only(part, "use", where)
            if part["use"] not in self.parts:
                raise Refused("%s uses \"%s\", which is not a part" % (where, part["use"]))
            self.check_fact(part.get("when"), where)
        elif "do" in part:
            only(part, "action", where)
            name = part["do"]
            if name in self.actions and self.actions[name] is not part:
                raise Refused("two actions are named \"%s\"" % name)
            self.actions[name] = part
            where = "the action \"%s\"" % name
            self.check_fact(part.get("when"), where)
            for r in part.get("reads", []):
                if isinstance(r, dict):
                    only(r, "read", where)
                    self.check_fact(r.get("when"), where)
                self.check_place(r["place"] if isinstance(r, dict) else r, where)
            for c in part.get("changes", []):
                if isinstance(c, dict):
                    only(c, "change", where)
                    self.check_fact(c.get("when"), where)
                    if c.get("how", "set") not in HOWS:
                        raise Refused("%s: a change is one of %s" % (where, ", ".join(HOWS)))
                    if c.get("later") not in (None, "undone", "stands"):
                        raise Refused("%s: when a failure leaves its part a change is \"undone\" or \"stands\"" % where)
                    for fact, value in c.get("makes", {}).items():
                        if not isinstance(value, bool):
                            raise Refused("%s: a change makes a fact true or false" % where)
                        if fact not in self.facts:
                            raise Refused("%s: a change makes the fact \"%s\", which the design does not list" % (where, fact))
                    if c.get("how", "set") in ("create", "replace") and not self.places.get(c["place"], {}).get("keyed"):
                        raise Refused("%s: only a keyed place has records to %s; \"%s\" is not keyed" % (where, c["how"], c["place"]))
                self.check_place(c["place"] if isinstance(c, dict) else c, where)
            if not part.get("reads") and not part.get("changes"):
                raise Refused("%s reads nothing and changes nothing" % where)
            for h in part.get("handles", []):
                only(h, "handle", where + ", an answer")
                if h.get("then") not in THENS:
                    raise Refused("%s: an answer's \"then\" is one of %s" % (where, ", ".join(THENS)))
                if h.get("no", "stop") not in ("stop", "skip"):
                    raise Refused("%s: when the person says no the run will \"stop\" or \"skip\" the action" % where)
                self.check_fact(h.get("when"), where)
                for t in listing(h.get("tells")):
                    self.check_place(t, where)
                if h["then"] == "ask":
                    self.facts.setdefault(yes_at(name), {"name": yes_at(name), "says": "the person asked at %s says yes" % name, "built in": True})
        elif "seq" in part or "all" in part:
            kind = "seq" if "seq" in part else "all"
            only(part, kind, where)
            self.check_fact(part.get("when"), where)
            self.check_part_answers(part, where)
            for i, child in enumerate(part[kind]):
                self.check_part(child, "%s, part %d" % (where, i + 1))
        elif "choose" in part:
            only(part, "choose", where)
            self.check_fact(part.get("when"), where)
            self.check_part_answers(part, where)
            for b in part["choose"]:
                only(b, "branch", where)
                self.check_fact(b.get("when"), where)
                self.check_part(b["run"], where)
            if part.get("otherwise") is not None:
                self.check_part(part["otherwise"], where)
        else:
            raise Refused("%s is not an action (\"do\"), a sequence (\"seq\"), parts in no order (\"all\"), a choice (\"choose\") or a use of a part (\"use\")" % where)

    def check_part_answers(self, part, where):
        for h in part.get("handles", []):
            only(h, "handle", where + ", an answer")
            if h.get("then") not in PART_THENS:
                raise Refused("%s: a part answers a failure its children return with %s" % (where, ", ".join(PART_THENS[:-1]) + " or " + PART_THENS[-1]))
            self.check_fact(h.get("when"), where)
            for t in listing(h.get("tells")):
                self.check_place(t, where)

    def called(self, part):
        """What a finding calls a part: its name, or the actions in it."""
        return part.get("name") or part.get("do") or "the part that does " + ", ".join(self.actions_of(part))

    def kids(self, part):
        return part.get("seq", []) + part.get("all", []) + [b["run"] for b in part.get("choose", [])] + ([part["otherwise"]] if part.get("otherwise") is not None else [])

    def uses(self, part):
        if isinstance(part, str):
            return [part]
        if "use" in part:
            return [part["use"]]
        return [u for k in self.kids(part) for u in self.uses(k)]

    def actions_of(self, part, seen=None):
        seen = set() if seen is None else seen
        if isinstance(part, str):
            part = {"use": part}
        if "use" in part:
            if part["use"] in seen:
                return []
            seen.add(part["use"])
            return self.actions_of(self.parts[part["use"]], seen)
        if "do" in part:
            return [part["do"]]
        return [a for k in self.kids(part) for a in self.actions_of(k, seen)]

    def is_fault(self, name):
        return bool(self.facts[name].get("fault"))

    def possible(self, env):
        """False when more has gone wrong than a play allows, or a fact is
        true that can be true only when something that is false holds."""
        if sum(1 for n, v in env.items() if v and self.is_fault(n)) > self.budget:
            return False
        for name, value in env.items():
            if value and not holds(self.facts[name].get("only"), env):
                return False
        return True

    def fits(self, env):
        """env with whatever each true fact's "only" needs decided as well;
        or None when that cannot be, or more would be wrong than a play allows."""
        env, more = dict(env), True
        while more:
            more = False
            for name, value in list(env.items()):
                if value:
                    need = assume(self.facts[name].get("only"), env)
                    if need is None:
                        return None
                    if need:
                        env.update(need)
                        more = True
        return env if sum(1 for n, v in env.items() if v and self.is_fault(n)) <= self.budget else None

    def about(self, place, faults=True):
        """The facts about a place that say what can be wrong with the value it holds."""
        return [f["name"] for f in self.facts.values() if f.get("about") == place and not f.get("built in") and bool(f.get("fault")) == faults]


# ---------- playing one run ----------

def listed(items, env):
    """The reads or changes that apply in this play, each as an object."""
    out = []
    for it in items:
        it = it if isinstance(it, dict) else {"place": it}
        if holds(it.get("when"), env):
            out.append(it)
    return out


class Now:
    """The facts as they stand at one moment of a play: what was true
    when the run started, and over that what the run has made true or
    false since."""

    def __init__(self, start):
        self.start, self.made = start, {}

    def __contains__(self, name):
        return name in self.made or name in self.start

    def __getitem__(self, name):
        return self.made[name] if name in self.made else self.start[name]

    def get(self, name, default=None):
        return self[name] if name in self else default


class Play:
    """One run under one choice of facts."""

    def __init__(self, design, run, env):
        self.d, self.run, self.env, self.now = design, run, env, Now(env)
        self.state = {p: "untouched" for p in design.places}
        self.rests = {p: set() for p in design.places}      # the unanswered faults a place's value rests on
        self.by = {}                                       # place -> the action that last changed it
        self.effects, self.steps, self.before = [], [], {}
        self.saw, self.did = {}, {}                        # action -> what it read and what it changed, with the states
        self.answered, self.open, self.unwritten, self.warned, self.left = set(), [], [], [], []
        self.asked = set()                                 # the faults this play has already put as a question
        self.stop = self.exec(run)                         # the failure that reached the run, or None

    def exec(self, part):
        """Plays a part. Returns the failure it passes to its parent, or None.
        A failure that leaves a part decides what happens later to every change made inside it."""
        mark = len(self.effects)
        failure = self.play(part)
        if failure:
            for e in reversed(self.effects[mark:]):
                if e.get("decided"):
                    continue
                e["decided"] = True
                if e["later"] == "undone":
                    self.state[e["place"]] = e["was"] if e["was"] != "untouched" else "undone"
                    self.rests[e["place"]] = e["rested"]
                elif e["later"] is None and self.d.places[e["place"]]["kind"] == "lasting":
                    self.left.append(dict(e, at=failure.action))
        return failure

    def play(self, part):
        d, env = self.d, self.now
        if isinstance(part, str):
            part = {"use": part}
        if not holds(part.get("when"), env):
            for a in d.actions_of(part):
                self.steps.append((a, "skipped"))
            return None
        if "use" in part:
            return self.exec(d.parts[part["use"]])
        if "seq" in part:                                  # each child depends on the ones before: a failure ends the sequence
            for child in part["seq"]:
                failure = self.exec(child)
                if failure:
                    return self.answer(part, failure)
            return None
        if "all" in part:                                  # the children are independent: the others still run
            failures = [f for f in [self.exec(child) for child in part["all"]] if f]
            return self.answer(part, failures[0]) if failures else None
        if "choose" in part:
            for b in part["choose"]:
                if holds(b.get("when"), env):
                    return self.answer(part, self.exec(b["run"]))
            return self.answer(part, self.exec(part["otherwise"])) if part.get("otherwise") is not None else None
        return self.act(part)

    def answer(self, part, failure):
        """A part's own answer to a failure one of its children returned. Answered, the failure stops here."""
        if not failure:
            return None
        for h in part.get("handles", []):
            if holds(h.get("when"), self.now):
                self.answered |= {n for n in names_in(h.get("when")) if self.now.get(n) and self.d.is_fault(n)}
                if h["then"] != "accept":
                    for t in self.d.told(h):
                        self.state[t] = "set"
                    self.warned.append(self.d.called(part))
                return None
        return failure

    def act(self, a):
        d, env, name = self.d, self.now, a["do"]
        self.before[name] = dict(self.state)
        reads = [r["place"] for r in listed(a.get("reads", []), env)]
        changes = listed(a.get("changes", []), env)
        self.saw[name] = [(p, self.state[p], bool(self.rests[p])) for p in reads]
        wrong = []                                    # what is wrong here, in this play
        for p in reads:
            if d.places[p]["kind"] == "run" and self.state[p] in ("untouched", "undone"):
                self.unwritten.append((name, p))
            wrong += [f for f in d.about(p) if holds(f, env)]
        for c in changes:
            how, p = c.get("how", "set"), c["place"]
            for f in ([already(p)] + ([twice(p)] if c.get("many") else []) if how == "create" else [anothers(p)] if how == "replace" else []):
                if holds(f, env) and d.is_fault(f):       # a design can list one of these itself, without "fault", where it is normal
                    wrong.append(f)
        here = [f for f in d.at.get(name, []) if holds(f, env)]   # what goes wrong at this step
        fired = None
        for h in a.get("handles", []):
            if holds(h.get("when"), env):
                fired = h
                break
        if fired:
            self.answered |= {n for n in names_in(fired.get("when")) if env.get(n) and d.is_fault(n)}
            then = fired["then"]
            if then != "accept":
                for t in d.told(fired):
                    self.state[t] = "set"
            if then == "refuse":
                self.steps.append((name, "refused"))
                return Failure("failed", name, fired)
            if then == "warn":
                self.warned.append(name)
                self.steps.append((name, "warned"))
                return None
            if then == "ask" and not holds(yes_at(name), env):
                self.steps.append((name, "declined"))
                if fired.get("no", "stop") == "skip":
                    return None
                return Failure("declined", name, fired)
        for f in here:                                # a fault at this step is asked here, whatever the step changes
            if f not in self.answered:
                self.ask(name, f)
        # what is read carries its faults into what is made from it. One becomes a question
        # when something lasting, or a report, is changed while nothing has answered it.
        rests = (set(wrong) | set(here)).union(*[self.rests[p] for p in reads]) - self.answered
        if any(d.places[c["place"]]["kind"] in ("lasting", "report") for c in changes):
            for f in [f for f in wrong if f in rests] + sorted(rests - set(wrong)):
                self.ask(name, f)
        for c in changes:
            p = c["place"]
            self.effects.append({"place": p, "action": name, "later": c.get("later"), "was": self.state[p], "rested": self.rests[p]})
            self.state[p] = HOWS[c.get("how", "set")]
            self.rests[p] = set(rests)
            self.by[p] = name
            for f in d.about(p):                      # a value the run has just written is not the bad one it started with
                self.now.made[f] = False
            self.now.made.update(c.get("makes", {}))
        self.did[name] = [(c["place"], self.state[c["place"]], bool(rests)) for c in changes]
        if fired and fired["then"] == "fail":         # it did its work, and then returns the failure
            self.steps.append((name, "failed"))
            return Failure("failed", name, fired)
        self.steps.append((name, "ran"))
        return None

    def ask(self, action, fault):
        if fault not in self.asked:
            self.asked.add(fault)
            self.open.append((action, fault))

    def outcome(self):
        return self.stop.how if self.stop else "completed with a warning" if self.warned else "completed"

    def given(self):
        return [c for c in sorted(self.env) if self.env[c] and not self.d.facts[c].get("run")]

    def ran(self, action):
        return (action, "ran") in self.steps

    def label(self, more=None):
        given = sorted(set(self.given()) | {c for c, v in (more or {}).items() if v and not self.d.facts[c].get("run")})
        return {"run": self.run, "given": given, "outcome": self.outcome(), "at": self.stop.action if self.stop else None}


def explore(design, run, start=None):
    """Every play of a run. A fact is decided only when a play first asks
    for it, so two plays never differ in a fact that made no difference."""
    todo, out = [dict({in_run(r): r == run for r in design.doc["runs"]}, **(start or {}))], []
    while todo:
        env = todo.pop()
        try:
            if not design.possible(env):
                continue
            out.append(Play(design, run, env))
        except Need as n:
            if n.name not in design.facts:
                raise Refused("a play asked for the fact \"%s\", which the design does not list" % n.name)
            todo += [dict(env, **{n.name: True}), dict(env, **{n.name: False})]
    return out


def explore_all(design):
    return [p for run in design.doc["runs"] for p in explore(design, run)]


# ---------- what the plays show ----------

TITLES = {
    "open": "Not decided yet",
    "expect": "Broken expectation",
    "unanswered": "Something wrong that nothing answers",
    "left-behind": "Left behind after a stop",
    "reads-unwritten": "Read before it is written",
    "order": "Order matters where none is stated",
    "owner": "Changed by a part that does not own it",
    "silent": "A failure nobody is told about",
    "unexplained": "Waved through without a reason",
    "stale": "An acceptance that matches nothing",
    "unused": "A fact nothing asks about",
    "differs": "The code does something else",
}


def held(design, hold):
    if not hold:
        return False
    try:
        with open(os.path.join(design.root, hold["file"]), encoding="utf-8") as f:
            return hold["test"] in f.read()
    except OSError:
        return False


def is_state(got, want):
    if want == "changed":
        return got in ("set", "created", "replaced", "deleted")
    if want == "unchanged":
        return got in ("untouched", "undone")
    return got == want


def reach(design, part):
    """The places the actions of a part read, and the ones they change."""
    r, c = set(), set()
    for a in design.actions_of(part):
        act = design.actions[a]
        r |= {x["place"] if isinstance(x, dict) else x for x in act.get("reads", [])}
        c |= {x["place"] if isinstance(x, dict) else x for x in act.get("changes", [])}
    return r, c


def walk(design, visit):
    def go(part):
        if isinstance(part, dict):
            visit(part)
            for k in design.kids(part):
                go(k)
    for part in design.parts.values():
        go(part)


def findings(design, plays):
    """Each finding is one question for the owner: what it is about, what
    the plays showed, how often, one play that shows it, and how the
    design can settle it."""
    out = {}

    def add(kind, subject, play, more=None, **fields):
        f = out.setdefault((kind, subject), dict({"kind": kind, "subject": subject, "plays": 0, "example": play.label(more) if play else None}, **fields))
        f["plays"] += 1 if play else 0
        return f

    for p in plays:
        for a, fact in p.open:
            f = add("unanswered", fact, p, actions=[], rest=[], shows=[],
                    settle='Give one of those actions, or an action that runs before them, an answer for it in "handles": refuse, fail, warn, tell, ask or accept.')
            if a not in f["actions"]:
                f["actions"].append(a)
            for place, facts in p.rests.items():
                if fact in facts and p.state[place] not in ("untouched", "undone") and place not in f["rest"]:
                    f["rest"].append(place)
        for e in p.left:
            f = add("left-behind", e["place"], p, writers=[], at=[], shows=[],
                    settle='Say what happens to that change when a failure leaves the part it is in: give it "later": "undone" or "later": "stands".')
            if e["action"] not in f["writers"]:
                f["writers"].append(e["action"])
            if e["at"] not in f["at"]:
                f["at"].append(e["at"])
        for a, place in p.unwritten:
            add("reads-unwritten", "%s reads %s" % (a, place), p, shows=["No action has changed %s by then, in this run." % place],
                settle="Make its writer run in those plays, or make %s run only when its writer does." % a)
        for e in design.doc.get("expects", []):
            if e.get("in") not in (None, p.run):
                continue
            more = assume(e.get("when"), p.env)        # a flag the play never asked for made no difference to it
            if more is None or any(v and design.is_fault(n) for n, v in more.items()):
                continue                               # and a fault no action met did not happen in it
            more = design.fits(dict(p.env, **more))
            if more is None:
                continue
            on = e.get("on", "any")
            if on != "any" and not ((on == "success" and not p.stop) or (p.stop and on == p.stop.how)):
                continue
            broke = []
            for place in e.get("never changes", []):
                if is_state(p.state[place], "changed"):
                    broke.append("%s changes %s." % (p.by[place], place) if place in p.by else "A failure is reported to %s." % place)
            for place in e.get("never creates", []):
                if p.state[place] == "created":
                    broke.append("%s creates a record in %s." % (p.by[place], place))
            for place, want in e.get("ends", {}).items():
                if not is_state(p.state[place], want):
                    broke.append("%s ends %s, and the design says %s." % (place, p.state[place], want))
            for a in e.get("runs", []):
                if not p.ran(a):
                    broke.append("%s does not run." % a)
            for a in e.get("never runs", []):
                if p.ran(a):
                    broke.append("%s runs." % a)
            if "before" in e and e["before"] in p.before and not is_state(p.before[e["before"]][e["place"]], e["is"]):
                broke.append("%s is %s when %s starts, and the design says %s." % (e["place"], p.before[e["before"]][e["place"]], e["before"], e["is"]))
            if broke:
                f = add("expect", e["name"], p, more, shows=[], settle="Change the design so that this cannot happen, or change the expectation.")
                f["shows"] += [b for b in broke if b not in f["shows"]]
    for f in out.values():
        if f["kind"] == "unanswered":
            says = design.facts[f["subject"]].get("says", f["subject"])
            f["shows"] = ["%s." % (says[0].upper() + says[1:]),
                          "Nothing answers it at: %s." % ", ".join(f["actions"]) if design.facts[f["subject"]].get("at") else
                          "%s the first to change something lasting with it unanswered: %s." % ("This action is" if len(f["actions"]) == 1 else "These actions are", ", ".join(f["actions"]))]
            if f["rest"]:
                f["shows"].append("What the run changes then rests on it: %s." % ", ".join(f["rest"]))
        if f["kind"] == "left-behind":
            f["shows"] = ["%s changes %s." % (" and ".join(f["writers"]), f["subject"]),
                          "The change is still there when a failure leaves the part it is in. The failure comes from: %s." % ", ".join(f["at"]),
                          "Nothing says it is undone, and nothing says it stands."]

    def order(part):                               # parts in no stated order must not touch each other's places
        if "all" in part:
            reaches = [(k if isinstance(k, str) else k.get("name") or k.get("do") or "the part that does " + ", ".join(design.actions_of(k)), reach(design, k)) for k in part["all"]]
            for i, (a, (ar, ac)) in enumerate(reaches):
                for b, (br, bc) in reaches[i + 1:]:
                    clash = sorted((ac & (br | bc)) | (bc & ar))
                    if clash:
                        add("order", "%s and %s" % (a, b), None, shows=["%s and %s are given in no order, and both touch %s, which one of them changes." % (a, b, ", ".join(clash))],
                            settle='Put them in a "seq" in the order they must run, or stop one of them touching what the other changes.')
    walk(design, order)

    def part_answers(part):                        # a part that waves a failure through says why, as an action does
        if "do" not in part:
            for h in part.get("handles", []):
                if h["then"] == "accept" and not h.get("because"):
                    add("unexplained", "%s: accept, %s" % (design.called(part), whenever(h)), None,
                        shows=["%s goes on after a failure, and the design does not say why that is fine." % design.called(part)], settle='Give the answer a "because".')
    walk(design, part_answers)
    for name, a in design.actions.items():
        for c in a.get("changes", []):
            place = design.places[c["place"] if isinstance(c, dict) else c]
            if place["kind"] == "outside":
                add("owner", "%s changes %s" % (name, place["name"]), None, shows=["%s comes from outside the system, and %s changes it." % (place["name"], name)],
                    settle="If the system changes it, it is not outside: make it lasting and say who owns it.")
            elif "owner" in place and name not in design.inside[place["owner"]]:
                add("owner", "%s changes %s" % (name, place["name"]), None, shows=["%s belongs to %s, and %s is not part of it." % (place["name"], place["owner"], name)],
                    settle="Move the change into the owner, or give the place an owner that holds both.")
        for h in a.get("handles", []):             # the two ways a design waves something through each say why
            if h["then"] == "accept" and not h.get("because"):
                add("unexplained", "%s: accept, %s" % (name, whenever(h)), None,
                    shows=["%s goes on with something wrong, and the design does not say why that is fine." % name], settle='Give the answer a "because".')
        for c in a.get("changes", []):
            if isinstance(c, dict) and c.get("later") == "stands" and not c.get("because"):
                add("unexplained", "%s: its change to %s stands" % (name, c["place"]), None,
                    shows=["The change stays when a failure leaves its part, and the design does not say why that is fine."], settle='Give the change a "because".')
        quiet = [h for h in a.get("handles", []) if h["then"] != "accept" and not listing(h.get("tells"))]
        if quiet and not design.reports():
            f = add("silent", "the design", None, shows=[], quiet=[],
                    settle='Name the place where a person learns of a failure once, as "reports" at the top of the design.')
            f["quiet"].append(name)
    for fact in design.doc.get("facts", []):       # a decision nobody has made: played both ways, and asked
        name = fact["name"]
        if fact.get("open"):
            ran = {v: {a for p in plays if p.env.get(name) is v for a, s in p.steps if s == "ran"} for v in (True, False)}
            for p in plays:
                if name in p.env:
                    add("open", name, p, shows=[], settle="Decide it. Then say what is so wherever the design uses this fact, and take the fact out.")
            if ("open", name) in out:
                f, says = out[("open", name)], fact.get("says", name)
                f["shows"] = ["%s." % (says[0].upper() + says[1:]), "The design plays it both ways, and %d plays depend on it." % f["plays"]]
                if ran[True] - ran[False]:
                    f["shows"].append("When it holds, these run and otherwise do not: %s." % ", ".join(sorted(ran[True] - ran[False])))
                if ran[False] - ran[True]:
                    f["shows"].append("When it does not hold, these run and otherwise do not: %s." % ", ".join(sorted(ran[False] - ran[True])))
                f["example"] = None
    asked = set().union(*[set(p.env) for p in plays]) if plays else set()
    for x in design.doc.get("expects", []) + design.doc.get("rules", []):
        asked |= names_in(x.get("when"))
    for fact in design.doc.get("facts", []):
        if fact["name"] not in asked:
            add("unused", fact["name"], None, shows=["No play asks about it, and no expectation or rule names it."],
                settle='Use it where it matters: in a "when", in an answer, or as the "when" of the rule in words it changes. Or take it out.')
    for f in out.values():
        if f["kind"] == "silent":
            f["shows"] = ["The design names no place where a person learns of a failure, and these actions refuse, warn, tell or ask without naming one: %s." % ", ".join(f["quiet"])]
    order_ = list(TITLES)
    return sorted(out.values(), key=lambda f: (order_.index(f["kind"]), f["subject"]))


DID = {"ran": "goes ahead", "refused": "refuses", "failed": "does its work and then fails", "warned": "warns and goes on", "skipped": "skips it", "declined": "stops at a person's no", "missing": "has no such action"}


def read_accepted(path):
    """The differences from the design that the owner has accepted, each with its reason. No file means none."""
    if not os.path.isfile(path):
        return []
    with open(path, encoding="utf-8") as f:
        doc = json.load(f)
    only(doc, "accepted", path)
    for a in doc.get("accepted", []):
        only(a, "acceptance", path)
        if not a.get("finding") or not a.get("because"):
            raise Refused("%s: an acceptance names its \"finding\" and says why, in \"because\"" % path)
    return doc.get("accepted", [])


def compare(design, code, accepted=None):
    """Play the code's design beside the owner's, play for play. Each
    finding is one point where the two part ways: the first action that
    does one thing in the design and another in the code, or a place
    that ends differently though every action did the same. It names
    the facts that every such play shares, since they are the cause."""
    out = {}

    def add(key, subject, first, run, given, places=()):
        f = out.setdefault(key, {"kind": "differs", "subject": subject, "plays": 0, "first": first, "runs": [], "common": None, "places": [],
                                 "example": {"run": run, "given": given, "outcome": None, "at": None},
                                 "settle": "Decide which is right. Change the code, or change the design, or accept the difference with a reason."})
        f["plays"] += 1
        f["common"] = set(given) if f["common"] is None else f["common"] & set(given)
        if run not in f["runs"]:
            f["runs"].append(run)
        f["places"] += [p for p in places if p not in f["places"]]

    for run in design.doc["runs"]:
        if run not in code.doc["runs"]:
            out[run] = {"kind": "differs", "subject": run, "plays": 0, "example": None, "runs": [], "common": set(), "places": [],
                        "first": "The design has the run %s, and the code's design has none of that name." % run, "settle": "Describe that run of the code."}
            continue
        shared = [p for p in design.places if p in code.places]
        for p in explore(design, run):
            for c in explore(code, run, {k: v for k, v in p.env.items() if k in code.facts}):
                given = sorted(set(p.given()) | set(c.given()))
                places = [x for x in shared if p.state[x] != c.state[x]]
                did = dict(c.steps)
                parted = [(a, s, did.get(a, "missing")) for a, s in p.steps if did.get(a, "missing") != s]
                if parted:
                    a, ds, cs = parted[0]
                    add(("parts", a, ds, cs), "%s: the design %s, the code %s" % (a, DID[ds], DID[cs]),
                        "At %s the design %s, and the code %s." % (a, DID[ds], DID[cs]), run, given, places)
                else:
                    for x in places:
                        add(("place", x, p.state[x], c.state[x]), "%s ends %s in the design and %s in the code" % (x, p.state[x], c.state[x]),
                            "Every action does the same in both, and %s ends %s in the design and %s in the code." % (x, p.state[x], c.state[x]), run, given)
    for f in out.values():
        f["shows"] = [f.pop("first")]
        common = sorted(f.pop("common") or [])
        if common:
            f["shows"].append("Every play where this happens has: %s." % ", ".join(common))
        if f["runs"]:
            f["shows"].append("It happens in: %s." % ", ".join(f["runs"]))
        if f["places"]:
            f["shows"].append("Because of it these end differently: %s." % ", ".join(f["places"]))
    found, used = sorted(out.values(), key=lambda f: f["subject"]), set()
    for f in found:
        for a in accepted or []:
            if a["finding"] == f["subject"]:
                f["accepted"], f["because"] = True, a["because"]
                used.add(a["finding"])
    for a in accepted or []:                       # an acceptance that outlives its difference is a question again
        if a["finding"] not in used:
            found.append({"kind": "stale", "subject": a["finding"], "plays": 0, "example": None, "settle": "Take the acceptance out.",
                          "shows": ["It was accepted, because: %s. The code no longer differs from the design in that way." % a["because"]]})
    return found


def whenever(h):
    """An answer's condition in words: the facts it names, or "always" for one with no "when"."""
    names = sorted(names_in(h.get("when")))
    return "when " + " and ".join(names) if names else "always"


def tests(design):
    """Everything the design states that a test should hold, and whether one is named and found."""
    out = []
    for name, a in design.actions.items():
        for h in a.get("handles", []):
            out.append({"what": "answer", "subject": "%s: %s, %s" % (name, h["then"], whenever(h)), "held": held(design, h.get("hold"))})
        for c in a.get("changes", []):
            if isinstance(c, dict) and c.get("later"):
                out.append({"what": "recovery", "subject": "%s: its change to %s %s when a failure leaves its part" % (name, c["place"], "is undone" if c["later"] == "undone" else "stands"), "held": held(design, c.get("hold"))})
    for e in design.doc.get("expects", []):
        out.append({"what": "expectation", "subject": e["name"], "held": held(design, e.get("hold"))})
    for r in design.doc.get("rules", []):
        out.append({"what": "rule in words", "subject": r["rule"], "held": held(design, r.get("hold"))})
    return out


def said(design):
    """Every sentence a statement of the design stands for ("says"), and whether the engine plays that statement or it stays words."""
    out = []

    def go(x, how):
        if isinstance(x, dict):
            if "says" in x:
                out.extend((s, how) for s in ([x["says"]] if isinstance(x["says"], str) else x["says"]))
            for v in x.values():
                go(v, how)
        elif isinstance(x, list):
            for v in x:
                go(v, how)

    go(design.doc.get("parts"), "played")
    go(design.doc.get("expects"), "played")
    go(design.doc.get("rules"), "words")
    return out


def report(design, code=None, accepted=None):
    plays = explore_all(design)
    fs = findings(design, plays) + (compare(design, code, accepted) if code else [])
    runs = {}
    for p in plays:
        r = runs.setdefault(p.run, {"plays": 0})
        r["plays"] += 1
        r[p.outcome()] = r.get(p.outcome(), 0) + 1
    return {"system": design.doc["system"], "runs": runs, "plays": len(plays), "findings": fs, "tests": tests(design),
            "said": [{"says": s, "how": h} for s, h in said(design)],
            "all plays": [{"run": p.run, "given": p.given(), "outcome": p.outcome(), "at": p.stop.action if p.stop else None,
                           "steps": p.steps, "ends": {k: v for k, v in p.state.items() if v != "untouched"}} for p in plays]}


def seen_in(f):
    """One sentence: how often a finding arises, and a play that shows it."""
    eg = f.get("example")
    if not eg:
        return ""
    given = "with " + " and ".join(eg["given"]) if eg["given"] else "with nothing special given"
    how = {"failed": "fails", "declined": "is declined"}
    return "Seen in %d %s. For example, %s%s." % (f["plays"], "play" if f["plays"] == 1 else "plays", given, ", where the run %s at %s" % (how[eg["outcome"]], eg["at"]) if eg.get("at") else "")


def main(argv=None):
    ap = argparse.ArgumentParser(description="play every run a design allows and report what it finds")
    ap.add_argument("design")
    ap.add_argument("--root", default=".", help="the directory the files of holds start from")
    ap.add_argument("--against", help="a second design, written from the code, to play beside the first")
    ap.add_argument("--json", action="store_true", help="print everything, with every play, as JSON")
    ap.add_argument("--lines", action="store_true", help="print one line for each finding")
    args = ap.parse_args(argv)
    path = args.design
    try:
        designs = []
        for path in [args.design] + ([args.against] if args.against else []):
            with open(path, encoding="utf-8") as f:
                designs.append(Design(json.load(f), args.root))
        accepted = read_accepted(args.against[:-len(".json")] + ".accepted.json") if args.against and args.against.endswith(".json") else []
        r = report(*designs, **({"accepted": accepted} if args.against else {}))
        asked = [f for f in r["findings"] if not f.get("accepted")]
    except (OSError, ValueError, Refused) as e:
        print("%s: %s" % (path, e), file=sys.stderr)
        return 2
    if args.json:
        json.dump(r, sys.stdout, indent=1)
        print()
        return 2 if asked else 0
    if args.lines:
        for f in asked:
            print("%s  %s: %s %s" % (f["kind"], f["subject"], " ".join(f["shows"]), seen_in(f)))
        for f in r["findings"]:
            if f.get("accepted"):
                print("accepted  %s  %s: %s" % (f["kind"], f["subject"], f["because"]))
        return 2 if asked else 0
    wrap = lambda s, ind: textwrap.fill(s, 78, initial_indent=ind, subsequent_indent=ind, break_on_hyphens=False)
    print("%s: %d plays of %d runs\n" % (r["system"], r["plays"], len(r["runs"])))
    width = max(len(n) for n in r["runs"])
    for run, c in r["runs"].items():
        parts = ["%d %s" % (c[k], k) for k in ("completed", "completed with a warning", "failed", "declined") if c.get(k)]
        print("  %-*s  %4d plays   %s" % (width, run, c["plays"], ", ".join(parts)))
    n = len(asked)
    print("\n%s for the owner" % ("No questions" if not n else "1 question" if n == 1 else "%d questions" % n))
    for i, f in enumerate(asked, 1):
        print("\n%2d  %s" % (i, TITLES[f["kind"]].upper()))
        print(wrap(f["subject"], "    "))
        print()
        for line in f["shows"]:
            print(wrap(line, "    "))
        if seen_in(f):
            print(wrap(seen_in(f), "    "))
        print(wrap("To settle it: " + f["settle"][0].lower() + f["settle"][1:], "    "))
    for f in r["findings"]:
        if f.get("accepted"):
            print("\n" + wrap("Accepted: %s. Because: %s" % (f["subject"], f["because"]), "    "))
    unheld = [t for t in r["tests"] if not t["held"]]
    print("\nTests: %d statements should each have a test, and %d do." % (len(r["tests"]), len(r["tests"]) - len(unheld)))
    if unheld:
        print("The %d without one:" % len(unheld))
        for t in unheld:
            print(wrap("%s: %s" % (t["what"], t["subject"]), "  - ").replace("\n  - ", "\n    "))
    return 2 if asked else 0


if __name__ == "__main__":
    sys.exit(main())
