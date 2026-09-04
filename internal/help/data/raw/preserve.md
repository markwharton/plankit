---
name: preserve
description: Preserve the approved plan into docs/plans - modes, the pending pointer, and /plankit:preserve
---

# pk preserve

Preserve captures an approved Claude Code plan, byte for byte, into
`docs/plans/` under a dated, sequenced, slugged filename, and commits
it with a `plan:` message. The plan lands exactly as approved; nothing
rewrites it.

## Modes

The automatic hook fires when a plan is approved (ExitPlanMode) and
honors `preserve.mode` in `.pk.json`:

- `auto`: preserve and commit immediately
- `manual` (default): record the approved plan in a pending pointer
  and say so; nothing is committed until you ask
- `off`: do nothing

## Completing a manual preserve

Type `/plankit:preserve`, or run `pk preserve` in the repository. The
explicit invocation consumes the pending pointer and always commits,
whatever the mode; running it is the consent. `--push` also pushes the
commit; `--dry-run` previews the filename and commit message.

Identical plan bytes are never preserved twice: a duplicate reports
the existing file instead. Short drafts under 50 bytes are ignored.
