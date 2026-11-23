# Control: hold every one of Hamilton's sources

## Context

`control axioms` prints each of Hamilton's sources and what holds it.
Three rows are not fully held: Axiom 5 is held for outside input only,
Or is held for named conditions but not for the cases they leave, and
"reject values exist in all types" is held by nothing. They are one
idea seen three ways: every value can be invalid, and every step must
say what happens to an invalid value and to the case no condition
covers. Designed together they take one form; closed one plan at a time
by later sessions, they would take three.

The findings waiting in `docs/control-findings.md` are settled here
too. One of them, asking before a replace without looking at what is
there, is a fault in a real import step, and it becomes the checker's
first trial on real code. The checker has only been tried on a model of
its author's making.

The goal is that every row of the table is held, with no "partly" and
no "not held", and that the checker finds the real fault on its own.
Each decision names its hold.

## Decisions

| Decision | Held by |
|---|---|
| Every place names its reject value, in a new field `invalid` | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (reject-values breaks) |
| Every leaf that reads declares its refusal, not only one that reads outside input | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (rejects-declared breaks) |
| A refusal goes through a place the element writes, or to `"parent"`; an element whose child refuses to its parent declares its own refusal; the root's refusal goes to a place written within the tree | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" and "TestMalformedDefinitionsAreRefusedByName" |
| Every `or` names a child in a new field `otherwise`, which runs when no condition holds | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (or-conditions breaks) |
| `ifExists` gains `"ask"`: the create asks a person first | `control/check_test.go`, "TestMalformedDefinitionsAreRefusedByName" |
| A create that refuses or asks on an existing record reads its target, since a step cannot decide about what it has not looked at; reading its own target for that is not a breach of reads-for-reference | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (create-never-replaces and reads-for-reference breaks) |
| The table below is the owner's reference, replacing the earlier plan's | `control/sources_test.go`, "TestAxiomsPrintsTheOwnersTable" |
| The upload trial keeps its nine findings: the new declarations are added where the old model said nothing, and left out where the known faults are | `control/trial_test.go`, "TestTrialFindsTheKnownFaultsAndPassesTheFix" |
| A second trial models a real import step: the known fault and one the review did not find before the fix, none after | `control/trial_test.go`, "TestTheImportTrialFindsItsFault" |
| A plan that drops a hold an earlier plan named says so, with a line of the form ``Retires `<file>_test.go`, "<Test>" from `docs/plans/<plan>.md` `` | `control/format_test.go`, "TestEveryNamedHoldExists" and "TestRetiredHoldsAreRead" |
| The three findings become cases or a rule, and the findings file is left with none waiting | `control/standard_test.go`, "TestEveryRuleHasACheckAndEveryCaseAPrinciple" |

## Reference: Hamilton's sources

| Source | Hamilton | Held by |
|---|---|---|
| Axiom 1 | A parent invokes its immediate children, and only those | well-formedness: one root, one parent each |
| Axiom 2 | A parent is responsible for its own output, and only its own | one-owner |
| Axiom 3 | A parent decides each child's output access | one-owner, replace-checks-owner |
| Axiom 4 | A parent decides each child's input access, "for reference only" | reads-for-reference |
| Axiom 5 | A parent rejects what is outside its own input | rejects-declared |
| Axiom 6 | A parent orders its immediate children | join-ordered |
| Theorem | Single reference, single assignment (from axioms 3 and 4) | one-owner with reads-for-reference |
| Type map | No two objects in one place | create-never-replaces, key-from-content |
| Type map | Reject values exist in all types | reject-values with rejects-declared |
| Join | Each child depends on the ones before | join-ordered |
| Include | The children are independent | include-independent |
| Or | One child is chosen | or-conditions |

This replaces the table in
`docs/plans/2026-11-22-001-control-contain-drift-in-the-standard-and-the-checker.md`;
the test reads the newest plan with this heading.

## The format

- **Place `invalid`**: what an invalid value of the place is, in words.
  A new check, `reject-values`, reports a place with none. Principle:
  Reference.
- **`rejects` on every reading leaf.** `rejects-declared` reports a leaf
  that reads and declares no refusal. `to` is a place the element
  writes, or a descendant writes for a composite, or `"parent"`.
  Composites may now declare `rejects`. An element with a child that
  refuses to `"parent"` must declare its own; the root may not use
  `"parent"`. Holds text: "every element that reads declares its
  refusal, with a reason, and every refusal reaches a place".
- **`otherwise` on an `or`**: the child that runs when no condition
  holds. It is a child like the branches, with one parent. `or-conditions`
  also reports an `or` with none. Holds text: "each branch of an or names
  its condition, no two name the same one, and an otherwise branch takes
  every other case".
- **`ifExists: "ask"`**, beside `"refuse"`. `create-never-replaces`
  reports a create with either one that does not read its target: "a
  create that refuses or asks on an existing record reads its target
  first, so it knows which records exist". `reads-for-reference` lets
  that read through.

## Trials

**Upload pipeline** (`examples/upload-*.json`). Each place gains
`invalid`. `file`, `record` and `report` refuse to `"parent"`, and
`upload` refuses to `status`, which `report` writes: a failed step marks
the upload failed, as the real pipeline did. `parse` and `manual-file`
are left without a refusal, which is the known fault. Predicted before
running: the same nine findings as now, and none after the fix.

**Import step** (`examples/import-*.json`, new). A page puts the files
of a disk image a person chose on a volume of saved files. Places: the
image (outside), the volume (keyed by file name; the store puts a
record in place of any of the same name), and the page's message.
`import` reads the image, refuses an unreadable one through the
message, and creates each file on the volume with its key, the file's
name, from the image's directory; `ifExists: "ask"`. The image readers
(`machines/c64/filing.ts`, `machines/apple2e/filing.ts`) map the
directory's entries straight to files and refuse no name given twice,
so the model does not declare `repeated: "refuse"`.

Before: `import` reads only the image. Predicted, before running:
exactly two findings, `create-never-replaces  import → volume` (it asks
without reading the volume, so the question cannot name what it
replaces: the known fault) and `key-from-content  import → volume` (an
image whose directory names a file twice puts the second where the
first was: a fault no review named). After, `import` reads the volume
and refuses a name given twice: none. The trial models the import step
only.

## The standard

- **Chokepoints case: an import that asked without looking.** The step
  asked the person before putting files on a volume and said a file of
  the same name would be replaced, without reading which files the
  volume held, so the person could not tell what the answer would
  replace. A question is a door only when it names what it lets
  through.
- **Proof case: a proof run that replayed.** A mutation run's scratch
  copies kept their files' modification times, and the test runner
  reused a cached pass for changed documents, so five breaks read as
  passes until the run was repeated with the cache off. The runner is
  part of the proof's instrument.
- **Rule, under Reference**, "A decision names its hold, and a hold
  names its decision", gains: "A plan that drops a hold an earlier plan
  named says so, naming the hold and the plan it came from."
- `docs/control-findings.md` says no findings are waiting.

## Commits

The work continues on branch `control`, unpushed.

1. Preserve this plan. From here until commit 4, TestAxiomsPrintsTheOwnersTable
   fails: the reference has moved and the code has not. That is the hold
   doing its work.
2. `feat(control)`: reject values and refusals at every level: `invalid`,
   `rejects` on every reading leaf, `"parent"`, the new check; baseline,
   breaks and malformed cases first, each seen failing.
3. `feat(control)`: `otherwise`, with its breaks first.
4. `feat(control)`: `ifExists: "ask"` and the read of the target; the
   sources table in `sources.go` matches the reference.
5. `test(control)`: both trials, the predictions above in the test.
6. `docs(control)`: the README's tables and the DESIGN limits.
7. `test(control)` and `docs(control)`: retired holds, the two cases,
   the rule's sentence, the findings file.

Each commit carries no Co-Authored-By trailer.

## One copy of each thing

- **The reference table**: only in this plan; `sources.go` is the code.
- **Who reads a place's target**: one helper in `check.go`, used by
  create-never-replaces and reads-for-reference.
- **Where a refusal lands**: one function that follows `"parent"` up the
  tree, used by rejects-declared and well-formedness.
- **Named and retired holds**: the existing `namedHolds` in
  `control/format_test.go` reads both.

## Verification

- `make test`, and the control suite with `-count=1`.
- The mutation run, uncached, in a copy shaped like the repository:
  each check disabled, the parser mutants, the standard and plan
  mutants, and the new checks' breaks, each failing a named test.
- Smoke: `control check examples/import-before.json` prints the two
  findings and exits 2; `import-after.json` exits 0; `control axioms`
  shows every row held.

## Not in this plan

- Ownership per record, or by a composite: the import trial models the
  import step only, so it does not need it.
- Removing a record: the format has no delete.
- The fix in the repository the import step comes from: its own plan,
  with both findings as its input.
- Conformance, facts, publishing and release.
