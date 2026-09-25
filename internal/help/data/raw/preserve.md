---
name: preserve
description: Keep each approved plan as a record in docs/plans, on approval or on request
---

# pk preserve

`pk preserve` copies an approved Claude Code plan, byte for byte, into
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

Identical plan bytes are never preserved twice; a duplicate reports
the existing file. A plan shorter than `minPlanSize` bytes is ignored (`grep minPlanSize internal/preserve/preserve.go`).
A typed run with nothing pending says so; the hook declines in
silence, because its stdout is the response envelope.

## Flags

<!-- generated: flags -->
```
  --dry-run
        Preview without writing or committing
  --plan <value>
        Plan file to preserve, for a plan revised after approval (a path under ~/.claude/plans)
  --push
        Push to origin after committing
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->

## Settings

<!-- generated: settings -->
The `preserve` section of `.pk.json`:

```
"preserve": {
  "mode": "auto" | "manual" | "off"
}
```

- `preserve.mode`: `auto`, `manual`, or `off`; default `manual`. An approved plan is committed at once, recorded for `/plankit:preserve` to commit, or ignored.

An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem.
<!-- /generated: settings -->
