# Hamilton's sources

The owner's reference for `control axioms`: each of Hamilton's sources,
read from her side, and what holds it. `control axioms` is compared with
this table row for row, by `control/sources_test.go`,
"TestAxiomsPrintsTheOwnersTable". A change to this file is the owner's
decision; once `guard.references` lists it, guard asks before a commit
that changes it.

## Hamilton's sources

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
