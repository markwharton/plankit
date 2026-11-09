---
name: preserve
description: Keep each approved plan as a record in the plans directory, on approval or on request
---

# pk preserve

`pk preserve` copies an approved Claude Code plan, byte for byte, into
the plans directory, `preserve.dir` in `.pk.json`, under a dated,
sequenced, slugged filename and commits it with a `plan:` message.
The commit holds the plan file alone; anything else staged stays
staged. The subject carries `[skip ci]` when the plan is the only
unpushed work on the branch; when unpushed commits precede it, or the
branch has no upstream, the marker is left off and the output says so.

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
the existing file. A plan is preserved once it is committed: a file a
failed commit left behind is committed by the next run, and a plans
directory that git ignores is refused. A plan shorter than `minPlanSize` bytes is not preserved (`grep minPlanSize internal/preserve/preserve.go`).
With nothing pending, or a plan that short, a typed run says so and
the hook declines in silence, because its stdout is the response
envelope. A preserve that fails, a commit git refuses for example, is
reported: a typed run exits non-zero, and the hook's message is shown
without blocking the session. The hook reports the same way when the
approved plan's path is outside the session's plans directory, and
names the path.

## Preserving a plan revised after approval

The approval hook reads the plan's path from `tool_response.filePath`
on stdin and records it in `.git/pk-pending-plan`. A typed
`pk preserve` consumes the pointer. A plan revised after it was
preserved has no pointer: preserve it again with `--plan <path>`, and
it becomes the next sequence number for the day. The same rules
apply: the path is a Claude Code plan under the session's plans directory
(`~/.claude/plans/`, or `plans/` under `CLAUDE_CONFIG_DIR` when that is set:
a second account keeps its own, and a same-named file in the other
account's directory is another session's plan), and
identical bytes report the existing file. While the pointer names a
different plan, `--plan` refuses and names both files; preserve the
pending plan first.

## Discarding a pending plan

The hook records the pointer in the repository the session runs in.
When the approved plan belongs to another repository and was preserved
there with `--plan`, the pointer here still names it, and a later
`pk preserve` would commit that plan into the wrong repository.
`pk preserve --discard` removes the pointer without preserving anything
and names the plan it pointed at; with nothing pending it says so.
`--dry-run` shows what would be discarded. A pointer whose plan file no
longer exists is discarded the same way and reported as such.
`pk status` shows the pending plan, so a stale pointer is easy to see.
Discarding is the developer's call: a session never discards a plan on
its own.

## Flags

<!-- generated: flags -->
```
  --discard
        Remove the pending plan pointer without preserving the plan it names
  --dry-run
        Preview without writing or committing
  --plan <value>
        Plan file to preserve, for a plan revised after approval (a path under ~/.claude/plans, or plans/ under CLAUDE_CONFIG_DIR when that is set)
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
  "dir": "<dir>",
  "mode": "auto" | "manual" | "off"
}
```

- `preserve.dir`: a directory, relative to the repository root; default `docs/plans`. Where preserved plans are written and kept immutable; forward slashes, inside the repository.
- `preserve.mode`: `auto`, `manual`, or `off`; default `manual`. An approved plan is committed at once, recorded for `/plankit:preserve` to commit, or ignored.

An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem.
<!-- /generated: settings -->
