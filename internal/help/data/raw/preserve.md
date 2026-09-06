---
name: preserve
description: Preserve the approved plan into docs/plans - modes, the pending pointer, and /plankit:preserve
---

# pk preserve

Preserve copies an approved Claude Code plan, byte for byte, into
`docs/plans/` under a dated, sequenced, slugged filename and commits
it with a `plan:` message.

## Modes

The hook fires when a plan is approved (ExitPlanMode) and follows
`preserve.mode` in `.pk.json`:

- `auto`: preserve and commit at once
- `manual` (default): record the approved plan in a pending pointer
  and say so; nothing is committed until you ask
- `off`: do nothing

## Completing a manual preserve

Type `/plankit:preserve`, or run `pk preserve` in the repository. The
explicit invocation consumes the pending pointer and commits, whatever
the mode. `--push` also pushes the commit. `--dry-run` previews the
filename and commit message.

Limit: identical plan bytes are never preserved twice; a duplicate
reports the existing file. Limit: a plan shorter than `minPlanSize`
bytes is ignored (`grep minPlanSize internal/preserve/preserve.go`).
