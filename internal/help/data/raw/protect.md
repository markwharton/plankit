---
name: protect
description: Why docs/plans is immutable and how pk protect enforces it
---

# pk protect

Protect is a PreToolUse hook on Edit and Write. Any attempt to modify
a file under `docs/plans/` is denied: preserved plans are immutable
historical records of what was approved, and their value is exactly
that they cannot be quietly rewritten afterward.

The check resolves symlinks and compares case-insensitively on
Windows, so the directory cannot be reached through an alias.

## Amending a plan

Plans are superseded, never edited. If the approach changes, write and
approve a new plan; the sequence in `docs/plans/` is the history of
decisions, including the reversals.

Protect is a no-op in an unconfigured repository (no `.pk.json`).
