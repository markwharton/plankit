#!/usr/bin/env python3
"""Convert a design written before places had a shape.

    python3 convert.py DESIGN.json [DESIGN.json ...]

Each file is rewritten in place. The rule:

- A keyed place becomes "is": "many". Any other place becomes "is": "one".
- A change with no "how", or with "set", becomes "create" on a place that
  is keyed, lasts for one run, or is a report. On any other place it
  becomes "replace".
- "delete" becomes "destroy".
- The states "set" and "deleted" in an expectation become the state the
  rule above gives that place, and "destroyed".
- The three facts a keyed place brought are named for members, not records.

A design that already gives a place an "is" is left as it is. The rule
cannot tell an add from an overwrite on a place that was not keyed, so a
converted design asks, of each lasting place it replaces, what happens
to what was there.

Standard library only.
"""
import json
import sys

OLD_BROUGHT = ["%s already has the record", "%s is given a key twice", "%s holds another's record"]
NEW_BROUGHT = ["%s already has the member", "%s is given a key twice", "%s holds another's member"]
NEW_STATE = {"create": "created", "replace": "replaced", "destroy": "destroyed"}


def how_for(place, how):
    """The operation a change in the old form is, on this place."""
    if how in ("create", "replace"):
        return how
    if how == "delete":
        return "destroy"
    if place.get("keyed") or place["kind"] in ("run", "report"):
        return "create"
    return "replace"


def convert(doc):
    """The design in the new form. doc is not changed."""
    if any("is" in p for p in doc.get("places", [])):
        return doc
    doc = json.loads(json.dumps(doc))
    old = {p["name"]: dict(p) for p in doc["places"]}
    names = {o % n: w % n for n in old for o, w in zip(OLD_BROUGHT, NEW_BROUGHT)}

    def rename(x):
        """Every use of a brought fact's old name, wherever a fact is named."""
        if isinstance(x, str):
            return names.get(x, x)
        if isinstance(x, list):
            return [rename(v) for v in x]
        if isinstance(x, dict):
            return {names.get(k, k): rename(v) for k, v in x.items()}
        return x

    def part(p):
        if isinstance(p, dict):
            if "do" in p and "changes" in p:
                p["changes"] = [dict(c if isinstance(c, dict) else {"place": c},
                                     how=how_for(old[c["place"] if isinstance(c, dict) else c], c.get("how", "set") if isinstance(c, dict) else "set"))
                                for c in p["changes"]]
            for k in ("seq", "all"):
                for child in p.get(k, []):
                    part(child)
            for b in p.get("choose", []):
                part(b["run"])
            part(p.get("otherwise"))

    for p in doc["places"]:
        p["is"] = "many" if p.pop("keyed", False) else "one"
    for p in doc["parts"].values():
        part(p)

    def state(place, s):
        return NEW_STATE[how_for(old[place], "set")] if s == "set" else "destroyed" if s == "deleted" else s

    for e in doc.get("expects", []):
        if "ends" in e:
            e["ends"] = {place: state(place, s) for place, s in e["ends"].items()}
        if "is" in e:
            e["is"] = state(e["place"], e["is"])
    return rename(doc)


def main(argv=None):
    for path in (argv if argv is not None else sys.argv[1:]):
        with open(path, encoding="utf-8") as f:
            doc = json.load(f)
        new = convert(doc)
        if new is doc:
            print("%s: already has shapes, left as it is" % path)
            continue
        with open(path, "w", encoding="utf-8") as f:
            json.dump(new, f, indent=2, ensure_ascii=False)
            f.write("\n")
        print("%s: converted" % path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
