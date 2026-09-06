---
name: protect
description: Why docs/plans is immutable and how pk protect enforces it
---

# pk protect

Protect is a PreToolUse hook on Edit and Write. It denies any write
under `docs/plans/`. Reason: a preserved plan is a record of what was
approved, and a record that can be edited is not one.

The check resolves symlinks and compares case-insensitively on
Windows. Limit: the directory cannot be reached through an alias.

## Amending a plan

Plans are superseded, never edited. When the approach changes, write
and approve a new plan. The sequence in `docs/plans/` is the history
of decisions, reversals included.

Protect is a no-op in an unconfigured repository (no `.pk.json`).
