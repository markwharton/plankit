# control view: a map of a definition, its faults, and the tests it lacks

## Context

`control check` answers three questions about a definition file, and its
findings are lines of text. Run on `definition/pk.json` alone, with no
description files, it found four design findings and nine rules with no
hold before any code was read. The owner could not see that from 763
lines of JSON. Drawing the definition by hand showed what helps: the
control tree with each step as outputs = step(inputs), a lens on one
place (its owner, writers, readers), a sweep that breaks each place in
turn and plays the run, and a list of tests to write for the rules with
no hold. Doing it also showed that the findings JSON loses its
structure (the subject had to be parsed back out of a string), and that
the sweep needs rules the format does not state.

The outcome: `control view <definition file> [<description file> ...]`
writes one self-contained HTML page with the map, the sweep and the
tests to write. The checker computes everything; the page only draws.
Stepping through a run is not in this plan. Each played run is kept as
a full trace, so a stepper on the map is a later reader of the same
data, not a new computation.

The work is on the `checker` branch, in the control module. No release
is part of this plan.

## The play rules (decided)

- A `join` runs its children in order; when one refuses, the rest do
  not run (fields.md).
- An `include` runs every child, because its children are independent:
  one child's refusal does not stop the others. Each refusal climbs from
  its own child, the trace records all of them, and the verdict reports
  the first in list order.
- An `or` is played once for each choice of branches (a path), because
  its conditions are words and cannot be evaluated. Each path gets its
  own trace, and each fault a verdict for each path, shown with the
  branches taken.
- A place from outside the system starts given, a lasting place starts
  present, and any other place starts empty. A broken place holds its
  `invalid` value whenever it has a value.
- A leaf that reads an invalid value and declares `rejects` refuses. The
  refusal follows `to: "parent"` up the tree until a `to` names a place,
  or stops at a parent with no `rejects`.
- A leaf that reads an invalid value and declares no `rejects` runs, and
  every place it writes now holds an invalid value, `message` included.
  The sweep then shows where the value is caught, if anywhere. No place
  is exempt by name.
- Checks before a write and the conditions of an `or` are shown and not
  evaluated.

Verdicts, in the order they are decided from a trace:

1. **refused**: the refusal reaches a place. It is held when the
   refusing leaf's `rejects` names a hold that `missing` finds.
2. **lost**: the refusal stops at a parent with no `rejects`.
3. **output**: no leaf in the run reads the broken place.
4. **passed**: no refusal, and some step read the invalid value.
5. **unread**: no refusal, and no step read the place while it held the
   invalid value.

A refusal that lands in the broken place itself is still refused. The
place's invalid value is a separate fault.

## Commits

1. **`docs(control): three findings about the checker`**:
   `docs/control-findings.md`, under "The checker", from this session on
   24 November 2026:
   - A run on a definition alone fails on one `follows` finding for each
     step, because no description file was given.
   - The format cannot state a failure that warns and does not refuse
     (`switch back`, `push working branch` in pk.json).
   - A `rejects` cannot name the places it refuses, so nothing can check
     that a refusal covers an invalid value that reaches it.

   Not logged, because this plan settles them: findings carry their
   element and place (commit 2), and the sweep's rules (above,
   documented in commit 5).

2. **`feat(control): a finding names its element and place`**
   - checks.go: `type Subject struct{ Text, Element, Place, Check, Unsaid string }`. Each `s*` constructor returns one.
   - `collector.add` takes a `Subject` and keys on `Text`. `Finding` gains `Element` and `Place`. Acceptance matching and the sort still use the text.
   - Every call site changes together: `sound`, `held`, `sourced`, `follows`, `compareRefusal`, `undescribed`, and the stale finding in `checkFiles` (no target).
   - main.go: the JSON line gains `element` and `place` (omitempty).
   - checks.md: the subject table gains a column for what the subject names.
   - Tests:
     - `TestTheSubjectsAreThePages` is extended to the new column.
     - New `TestAFindingNamesItsElementAndPlace`.

3. **`refactor(control): one walk of the rules`**
   - New rules.go: `rules(m) []rule`, in `held`'s current order (places that refuse, checks before writes, rejects, declines, unsaid). Each `rule` carries its `Subject`, evidence and the object it is on.
   - `held` becomes a loop over `rules(m)`.
   - `(m *model) ownedBy(element string, p *Place) bool` is extracted from one-owner, for the lens.

4. **`refactor(control): checkFiles returns the model`**: it returns the definition's model and the description models beside the findings, for `runView`. `runCheck` is unchanged in behaviour.

5. **`feat(control): play each run with one place broken`**
   - play.go:
     - Types: `Break{Place, Refuses string}` (a place broken, or a refusal forced at a leaf), `Pick{Or, Child}`, `Seen{Place, State}`, `StepTrace{Element, Outcome string; Reads, Writes []Seen}`, `Climb{From string; Through []string; To, LostAt string}` and `Trace{Run string; Break Break; Path []Pick; Steps []StepTrace; Refusals []Climb}`.
     - `play(m, run, brk, path) Trace` is the only interpreter.
     - `verdict(m, t) string` reads only the trace and the run's readers.
   - sweep.go: `sweep(m, root) []RunView`. Each `RunView` holds one baseline trace per path, a fault for each touched place on each path, and each fault's verdict, refusing leaf, held flag and trace index.
   - States and verdicts are Go slices.
   - New control/view.md, the reference for the sweep: the start states, the play rules, the verdicts and an example.
   - Tests:
     - `TestEachPlayRuleFailsOnItsBreak`: one baseline mutation per rule in view.md's table.
     - `TestEveryPlayRuleHasABreak`
     - `TestTheBaselineSweepIsFixed`
     - `TestTheTrialsSweepAsFixed`: the trials, including save's nested or.
     - `TestVerdictsComeFromTraces`
     - `TestTheSweepPageNamesEveryStateAndVerdict`

6. **`feat(control): tests from the rules with no hold`**
   - gaps.go: `testEntries(m, runs, root)` ranges over `rules(m)` where `missing` reports a gap. For each rule:
     - **Input faults:** the faults whose refusal leaves from or climbs through the rule's leaf. When there are none, the leaf that refuses each of its inputs first.
     - **Asserts:** the leaf the run fails at, the refusal's path, and the join steps that do not run. These come from `play` with `Break{Refuses: leaf}`, so no second walker is needed.
     - **Break and baseline**, in the Control standard's form.
     - **Test file:** the hold file most used under the run, ties to the first in walk order. When the run has no hold, no file is suggested and the entry says so.
     - **Hold JSON** to add, with `"test": ""` left for the writer. If it is committed blank, `held` still reports it.
   - Unsaid entries show their words, and their four lines are marked unwritten: they cannot be derived.
   - view.md gains the section on test entries.
   - Tests:
     - `TestTheTestsFromTheGapsAreFixed`: the baseline with chosen holds removed.
     - `TestAnUnsaidEntryIsMarkedUnwritten`
     - `TestHeldAndTheTestsWalkTheSameRules`: the set of `held` subjects equals the set of entry subjects.

7. **`feat(control): control view draws the map, the sweep and the tests`**
   - view.go: `control view [--root dir] <definition> [<description> ...]`, added to `usage` and `run`.
     - It calls `checkFiles`. A refused file exits 2 with the reasons on stderr.
     - Otherwise it writes the HTML to stdout and a one-line summary to stderr, and exits 0 whatever the findings.
   - The view model:
     - Nodes carry the outputs = step(inputs) label, built in Go, plus the places each owns, its refusals with held flags, and its findings by index.
     - Places carry the owner, kind, invalid value, writers, readers, and writers outside the owner (from `ownedBy`).
     - The model also holds the findings, runs and tests.
   - Findings pin by `Element` and `Place`.
   - When no description file is given, the `follows` findings are shown unchanged and one line says question 3 was not asked. The view and `check` then report the same findings line for line.
   - view.html: `//go:embed`, with inline CSS and JS and system font stacks, in the look of the drawn map.
     - The data goes in once, as JSON in `<script type="application/json">`, encoded by `encoding/json`, which escapes `<`, `>` and `&`.
     - The script parses it and draws. It has no rules of its own.
   - README.md: the pages table, "Running it", and each new hold in the `Held by` form, so that `TestEveryNamedHoldExists` covers it.
   - design.md: what play does not model.
   - Tests:
     - `TestTheViewShowsTheFindingsCheckPrints`
     - `TestTheViewIsOneFileWithNoNetwork`
     - `TestTheViewEscapesItsData`: a place named `</script><b>` round-trips.
     - `TestTheViewExitsZeroWithFindings`
     - `view` cases added to `TestACommandGivenWronglyExitsOne` and `TestARefusedFileStopsTheRun`.

## What stops a second copy

| Seam | The one place | Held by |
|---|---|---|
| Interpreting the format | `play`; `verdict` reads traces only; the page draws | `TestVerdictsComeFromTraces`; view.html is given computed data, not the definition |
| The rules | `rules(m)`, used by `held` and `testEntries` | `TestHeldAndTheTestsWalkTheSameRules` |
| Subjects and their targets | the `s*` constructors | `TestTheSubjectsAreThePages` |
| Ownership | `ownedBy`, used by one-owner and the lens | one-owner's existing break tests |
| Whether a hold exists | `missing` | `held`'s tests |
| The findings | `checkFiles`, shared by `check` and `view` | `TestTheViewShowsTheFindingsCheckPrints` |
| Names of states and verdicts | Go slices | `TestTheSweepPageNamesEveryStateAndVerdict` |
| The data in the page | one JSON encode into the template | `TestTheViewEscapesItsData` |

## Verification

- `make test` before the first commit and after each one.
- Smoke test, from `control/`:
  - `go run . view --root .. ../definition/pk.json > <scratch>/view.html`
    exits 0. Its stderr summary names 15 findings. Open the page and see
    the tree, the lens and the sweep.
  - Cross-check: the sweep for pk.json gives 22 faults: 17 refused with
    a hold, 3 refused with no hold (`next version`, `version files`,
    `release commit`), and 2 output (`message`, once for each run). The
    drawn map computed these independently. A difference is reported,
    not tuned away.
  - The test list gives 9 entries: 6 derived, and 3 unsaid entries
    marked unwritten. Three refusals have no input fault that reaches
    them: `preCommit hook`, `preRelease hook` and `commit`.
  - `go run . view` with no arguments exits 1. A malformed definition
    exits 2 with its reasons on stderr and nothing on stdout.
  - `go run . check --format json --root .. ../definition/pk.json`
    shows `element` and `place` on each finding.
