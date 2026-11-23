# Control: say what real code does

## Context

The first definition written by a session other than the checker's
author modelled a game's Save and Save As from their code, field by
field, with each field's file and line. The checker found a real fault:
Save writes to the name stored inside the game's zip, not the name the
game was opened under, so a renamed game saved over another one
without asking. But the ten findings came down to five causes, and four
of them were things the format cannot say. They are recorded under
"The checker" in `docs/control-findings.md`:

- two routes chosen by an `or` that write one place, read as two
  writers;
- one save writes one file, read as a possible repeat;
- places owned by other parts of the same program, written as outside
  input;
- a declined ask (Cancel, or no to "Replace?"), folded into the refusal;
- each field's source, kept in a separate file.

The checker also reported one cause three times: a refusal handed up a
chain of parents to one that declares none was a finding at each step.

This plan gives the format what real code needs and settles those
findings. Its trial is that same definition: once the format can say
these things, the findings left must each be a fault in the code or an
absence the rules require. Each decision names its hold.

## Decisions

| Decision | Held by |
|---|---|
| A place's `owner` may be a composite; every writer of the place is the owner or below it, and where two leaves write one place, the element nearest to both is an `or`, so only one of them runs | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (one-owner breaks) |
| A keyed write may say `records: "one"`: it makes one record per input, so its key cannot repeat within one, and key-from-content does not apply | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (key-from-content breaks) and "TestMalformedDefinitionsAreRefusedByName" |
| A place may say `elsewhere`: the part of the same system that owns it, in words. It is not outside input and has no owner here; a write to it is a one-owner finding, and its readers still declare a refusal, since Axiom 5 holds at every receiver and "no bad value reaches it" is review, not a hold | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" and "TestMalformedDefinitionsAreRefusedByName" |
| A step that asks declares `declines`: what happens when the answer is no, with the same shape as `rejects`; a decline reaches a place by the same rules as a refusal. A new check, `declines-declared`, reports an ask with none and a decline that reaches no place | `control/check_test.go`, "TestEveryCheckFailsOnItsBreakAlone" (declines-declared breaks) |
| A refusal or a decline that reaches no place is one finding, at the element that hands it to a parent declaring none, not one at every step of the chain | `control/check_test.go`, "TestOneUnlandedChainIsOneFinding" |
| Places, elements and writes may carry `source`: the file and function the field comes from. The checker does not read it | not held: a source is evidence for a reader, and nothing checks that it is right |
| The table below is the owner's reference, replacing the earlier plan's | `control/sources_test.go`, "TestAxiomsPrintsTheOwnersTable" |
| The save trial, modelled from the real code with the new fields, finds exactly the six causes below before the fix and none after | `control/trial_test.go`, "TestTheSaveTrialFindsOnlyFaults" |
| The upload and import trials keep their findings | `control/trial_test.go`, "TestTrialFindsTheKnownFaultsAndPassesTheFix" and "TestTheImportTrialFindsItsFault" |
| The checker's findings are settled, and the findings file is left with none waiting | `control/standard_test.go`, "TestEveryRuleHasACheckAndEveryCaseAPrinciple" |

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
| Or | One child is chosen | or-conditions, declines-declared |

An ask is a choice, and the person's no is the case its question
leaves; `declines-declared` holds that case as `or-conditions` holds an
`or`'s. This replaces the table in
`docs/plans/2026-11-22-002-control-hold-every-one-of-hamilton-s-sources.md`;
the test reads the newest plan with this heading.

## The save trial

`examples/save-before.json` and `save-after.json`, new: the
definition the other session wrote, with its names kept general (a
game's Save and Save As) and the new fields where the code calls for
them:

- the disk is owned by `store`, the `or` between Save As and Save in
  place, so the two routes are one owner;
- both disk writes say `records: "one"`;
- the game being played and the window's own title are `elsewhere`
  (the game's engine, the page); the typed name stays outside input;
- `save as` declines when Cancel or no is chosen, handed up through
  `store` and `saving` to `save`, which declares nothing, since the
  code drops `save`'s false return.

Predicted before the first run, the findings before the fix are
exactly these six, each a fault or an absence the rules require:

1. `replace-checks-owner  save in place → disk`: the Save fault.
2. `rejects-declared  mark saved`, 3. `rejects-declared  retitle`,
   4. `rejects-declared  save in place`: readers with no refusal.
5. `rejects-declared  saving`: a refusal handed to `save`, which
   declares none.
6. `declines-declared  saving`: a decline handed to `save`, which
   declares none.

After the fix (Save compares the file at its name with the game, each
reader refuses, and `save` refuses and declines through the message):
none.

## The format, field by field

- **`owner`** may name a composite. one-owner, for each place: every
  writer is the owner or a descendant of it; for every pair of leaves
  writing it, their nearest common ancestor is an `or`. Holds text: "each
  place is written by its owner or what lies below it, two writers of a
  place are alternatives of an or, and an outside or elsewhere place is
  written by no one".
- **`records`** on a keyed write: `"one"`, or left out for many.
  Well-formed: only on a keyed write.
- **`elsewhere`** on a place: words naming the owning part. Well-formed:
  not with `owner` or `outside`. one-owner reports a write to it;
  rejects-declared treats a read of it like any other read.
- **`declines`** on an element: `reason` and `to`, as `rejects`.
  Well-formed: as a refusal; the root may not decline to a parent.
  declines-declared: a leaf with a write `ifExists: "ask"` declares
  `declines`, and every decline reaches a place.
- **`source`** on a place, an element and a write: free text.

## Commits

The work continues on branch `control`, unpushed. The suite is red from
the plan commit until the reference matches (commit 4) and until the
tests this plan names exist (commit 5).

1. Preserve this plan.
2. `feat(control)`: a composite owner, with its breaks first.
3. `feat(control)`: `records`, `elsewhere` and `source`, with their
   breaks first.
4. `feat(control)`: `declines`, the new check, one finding per
   unlanded chain, and the sources table; the import trial declares its
   decline.
5. `test(control)`: the save trial, its prediction in the test.
6. `docs(control)`: the README's tables follow each field as it is
   added; DESIGN's limits; the findings settled.

Each commit carries no Co-Authored-By trailer.

## One copy of each thing

- **Where a refusal or a decline lands**: the existing `landing` in
  `control/def.go`, taking which of the two it follows, and reporting
  the one element where a chain breaks.
- **The nearest common ancestor**: one helper beside `landing`, using
  the parent map wellFormed builds.
- **The reference table**: only in this plan.

## Verification

- `make test`, and the control suite with `-count=1`.
- The mutation run, uncached, in a copy shaped like the repository: every
  earlier mutant, and each new mechanism disabled in turn (the `or` rule
  for two writers, `records`, `elsewhere` writes, declines, one finding
  per chain), each failing a named test.
- Smoke, with an installed binary: `control check examples/save-before.json`
  prints the six findings and exits 2; `save-after.json` exits 0.

## Not in this plan

- Ownership per record.
- Writers of the same places outside the modelled step: a definition
  describes one step.
- Two tabs on one disk: the checker has no concurrency.
- Removing a record: the format has no delete.
- The fix of the Save fault and the two small faults in the game's own
  repository: its own plan, with these findings as input.
- Conformance, facts, publishing and release.
