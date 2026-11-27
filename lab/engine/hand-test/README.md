# The hand test of 2026-11-26

A record. Before the engine played the design language's present form,
the form was written on paper and tested by hand. This directory keeps
what that test produced. It is not kept current, and nothing here is
part of the language or of `make test`.

The plan it followed is
`docs/plans/2026-11-27-001-the-design-language-s-next-form-on-paper-and-tested-by-hand.md`.

## What is here

| File | What it is |
|---|---|
| `REPORT.md` | The test's report, as written: a verdict on each of the five statements, the trace of pk by hand, what the language still lacked, and the two cold reads. Its numbers are from before the engine. |
| `check.py` | The script that gave the report its numbers. It reads a design and the ones it stands on, checks that every name resolves, and lists what comes up to each action that uses a run below. It plays nothing. |
| `cold/round1/release.json`, `cold/round2/release.json` | What the two cold readers wrote for `pk release`, each from the page as it stood in that round. They are evidence, and the engine may refuse them. |

## What is not here

The report names three files of the test that were not kept, because
each has a successor that is the agreed one:

| In the test | Now |
|---|---|
| `LANGUAGE.md`, the page on paper | `lab/engine/DESIGN.md` |
| `git.json` | `lab/engine/git.json` |
| `pk.json`, pk on git | `lab/engine/pk.json` |

The two outside definitions the report counts were never in this
repository.

## Using `check.py` again

It is for a design that no engine plays yet, which is the state a
change to the language starts in.

```
python3 check.py ../pk.json
```

The engine now reports what comes up, so for a design it can play, use
the engine.
