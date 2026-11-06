---
name: protect
description: Why preserved plans are immutable and how pk protect enforces it
---

# pk protect

`pk protect` is a PreToolUse hook on Edit and Write. It denies any write
under the plans directory, `preserve.dir` in `.pk.json`, because a
preserved plan is a record of what was approved, and a record that can
be edited is not one.

The check resolves symlinks and recognizes the directory by its
identity on disk, so it cannot be reached through an alias or, where
the filesystem ignores case, through another spelling.

## Amending a plan

Plans are superseded, never edited. When the approach changes, write
and approve a new plan. The sequence in the plans directory is the
history of decisions, reversals included.

`pk protect` is a no-op in an unconfigured repository (no `.pk.json`).

## Flags

<!-- generated: flags -->
Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
