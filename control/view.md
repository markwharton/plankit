# The view

`control view` reads a definition file and writes one page: the design
drawn as its tree, a sweep that breaks each place in turn and plays the
design, and a test to write for each rule that names no hold. Read
`format.md`, `fields.md` and `checks.md` first.

This page says how a run is played, what the sweep reports, what the
tests from the gaps are, and what the page holds.

## Playing a run

Each step of the system, a child of the top element, is a run. To play
a run is to walk its elements the way the format says they combine,
with one thing changed, the break, and to record what each leaf reads
and writes and how it ends. The record is a trace. The format does not
say what a design does when a value is wrong. The rules below do, and
the checker applies them in one place.

A place holds one of these states:

| State | The place holds |
|---|---|
| `given` | a value from outside the system |
| `present` | a value an earlier run left, because the place is lasting |
| `empty` | no value yet in this run |
| `written` | a value a leaf of this run wrote |
| `invalid` | its invalid value |
| `refusal` | a refusal reported to it |

The rules:

| Rule | What happens |
|---|---|
| `start` | A place from outside the system starts `given`, a lasting place starts `present`, and any other place starts `empty`. |
| `broken` | The broken place holds its invalid value whenever it has a value: from the start when it starts `given` or `present`, and otherwise from when a leaf writes it. |
| `join` | A join runs its children in order. When a refusal reaches the join, the children after the one that refused do not run. |
| `include` | An include runs every child, because its children are independent. Each refusal climbs from its own child. |
| `or` | The conditions of an or are words, so each child of an or is played on a path of its own. A run has one path for each choice of branches. |
| `refuse` | A leaf that reads an invalid value and declares `rejects` refuses, and writes nothing. |
| `climb` | A refusal whose `to` is `"parent"` reaches the element above, which passes it on or reports it by its own `rejects`. An element a refusal reaches stops. |
| `report` | The element whose `to` names a place reports the refusal there, and the place holds `refusal`. The element above it is not told, and goes on. |
| `lost` | A refusal that reaches an element with no `rejects` stops there and is reported nowhere. The element above it is not told, and goes on. |
| `pass-on` | A leaf that reads an invalid value and declares no `rejects` runs, and every place it writes holds an invalid value. |

Each leaf of a trace ends in one of these outcomes:

| Outcome | The leaf |
|---|---|
| `done` | read no invalid value, and wrote its places |
| `refused` | refused, and wrote nothing |
| `passed-on` | read an invalid value, refused nothing, and wrote its places with invalid values |
| `not-run` | did not run, because a refusal stopped a join above it |

Each rule fails on its break. Held by `control/play_test.go`,
"TestEachPlayRuleFailsOnItsBreak" and "TestEveryPlayRuleHasABreak".

## The sweep

The sweep plays each run once on each path with nothing broken, its
baseline, and once more for each place the run reads or writes, with
that place broken. Each of those is a fault, and it gets the first of
these verdicts that applies:

| Verdict | When |
|---|---|
| `refused` | A refusal reaches a place. The fault is held when the leaf that refused names a hold for its `rejects` that the checker finds, the way `held` looks for one. |
| `lost` | A refusal stops at an element with no `rejects`. |
| `output` | No leaf in the run reads the broken place, so no refusal can meet its invalid value. Only a test of the leaves that write it can. |
| `passed` | Nothing refused, and a leaf read the invalid value. |
| `unread` | Nothing refused, and no leaf read the place while it held its invalid value. |

When more than one refusal happens, as in an include, the first one in
the trace decides the verdict. A refusal that lands in the broken place
itself is `refused`. Held by `control/play_test.go`,
"TestVerdictsComeFromTraces".

What a verdict does not say:

- `refused` says that the invalid value met a refusal. Whether the
  refusal's reason covers that value is in words, and the sweep cannot
  judge it.
- A place that is not lasting starts `empty` in every run. When one step
  of the system writes it and another reads it, it holds no value in
  the run that reads it, and breaking it there is `unread`. Whether it
  is lasting is the design's statement.
- Checks before a write are not played.

## An example

The definition file in `fields.md`, "An example", has one step,
`import`, with no or, so one path. Its sweep:

| Broken place | Verdict | Refused at |
|---|---|---|
| `image` | `refused` | `import` |
| `volume` | `unread` | |
| `message` | `output` | |

`import` reads `image` and refuses it, and its refusal names a hold.
`volume` is not lasting, so it is empty when `import` reads it, and
`import` writes it last. Nothing reads `message`. Held by
`control/play_test.go`, "TestThePagesExampleSweepsAsThePageShows".

The sweeps of the baseline and of the trials are fixed by tests. Held by
`control/play_test.go`, "TestTheBaselineSweepIsFixed" and
"TestTheTrialsSweepAsFixed".

## Tests from the gaps

Each rule that names no hold, as `held` reports it, becomes a test to
write. An entry has the four lines a hold is proven by, in the form the
Control standard gives: what the test gives the system, what it
asserts, the break in the code that must make it fail, and the
baseline, the code as it is, on which it must pass.

| Rule | What the test gives | What it asserts |
|---|---|---|
| A `rejects` | Each place of the sweep whose fault the element refuses, first or as a refusal passed up to it. When no fault reaches a leaf, because earlier leaves refuse each of its inputs first, the entry names those leaves, and the test brings about the refusal's reason. | From a play of the fault, or with the leaf made to refuse: the leaf the run fails at, where the refusal goes, and the leaves that do not run. |
| A `declines` | The person says no. | From a play with the leaf declined: where the person's no goes, and the leaves that do not run. |
| A check before a write | The case the check asks about, as `fields.md` gives it. | From a play with the leaf made to refuse, or declined when the check asks. |
| A store that refuses | A create at a key that already has a record. | The create fails, and the old record stays. |
| An entry in `unsaid` | Unwritten | Unwritten |

An entry in `unsaid` is words, so its four lines, and the file its test
goes in, are left for the session that writes the test. Every other
entry suggests the file the holds of the rule's run name most, and the
`hold` to add to the definition file, with the test's name left empty
for the writer. A hold committed with no
name is still reported by `held`. Held by `control/gaps_test.go`,
"TestTheTestsFromTheGapsAreFixed", "TestAnUnsaidEntryIsMarkedUnwritten"
and "TestHeldAndTheTestsWalkTheSameRules".

## The page

The page is one HTML file that loads nothing. The checker computes
everything on it, and the page's script only draws: the control tree,
each leaf as outputs = leaf(inputs), the places with each one's owner,
writers and readers, the sweep with the trace of each fault, the tests
to write, and the findings `check` prints for the same files, in the
same order. With no description file, the page says that whether the
code follows the design was not asked. Held by `control/view_test.go`,
"TestTheViewShowsTheFindingsCheckPrints",
"TestTheViewIsOneFileWithNoNetwork", "TestTheViewEscapesItsData" and
"TestTheViewDataHasNoNullLists".
