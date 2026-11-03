# pk preserve adds [skip ci] only when the plan is the only unpushed work

## Context

`pk preserve` always puts `[skip ci]` in the plan commit's subject
(`internal/preserve/preserve.go:312`). GitHub reads that marker from
the newest commit of a push and skips the workflow for the whole push.
That is right when the plan is preserved before the work, because the
push holds documentation alone. With `preserve.mode: manual` the plan
is often preserved after the work, on top of unpushed implementation
commits; the next plain `git push`, or `pk preserve --push`, then runs
no CI for any of them and nothing says so. Seen twice in one session
on pk 1.8.1.

Outcome: the marker goes on only when HEAD is already on its upstream,
so the plan commit is the only thing the next push carries. Otherwise
the subject has no marker and the output says why. No new setting. A
plan-only push still skips CI as it does today.

## Design

**Push state is one primitive.** The branch's upstream (`@{u}`) is the
reference, not `@{push}`: probing git 2.54 showed `@{push}` fails with
no upstream and also under the default `push.default=simple` whenever
the upstream's name differs, while `@{u}` resolves whenever a plain
push would. `pk changelog --undo` already uses `@{u}` inline
(`internal/changelog/changelog.go:331`); the same check moves into one
helper that both commands call.

**No upstream means unknown, and unknown means no marker.** A branch
with no upstream (fresh branch, no remote, or first published by
`pk preserve --push`, which does not set one) cannot tell what is
unpushed, so the marker stays off and the output names the reason.
Leaving CI on is the safe side, so `--push` keeps its current push
command: setting an upstream there would be a separate change, and
this fix does not need it.

## Changes

### 1. `internal/git/git.go`: add `Ahead`

```go
// Ahead reports how many commits HEAD holds that its upstream does
// not. The bool is false when the branch has no upstream, so the
// count is unknown. err is a failed count, never a missing upstream.
func Ahead(dir string) (int, bool, error)
```

- `rev-parse --abbrev-ref @{u}` failing means no upstream: `(0, false, nil)`.
- Otherwise `CountCommits(dir, "@{u}..HEAD")` (existing helper at
  `internal/git/git.go:128`).
- Bare `(int, bool, error)` as `DefaultBranch` does; the bool is
  documented in prose, matching the package.

### 2. `internal/changelog/changelog.go`: `undo` calls `Ahead`

Replace the inline `rev-parse HEAD@{upstream}` plus `log @{u}..HEAD`
block (lines 330–339) with one call: refuse when the upstream is known
and the count is 0, with the same message. `TestUndo` at
`changelog_test.go:437` already covers the refusal and the no-upstream
case. Behaviour is unchanged; this is the second copy going away.

### 3. `internal/preserve/preserve.go`: compute the subject once

In `preserve()`, before the dry-run branch, call `git.Ahead(root)`.
A non-nil error is reported with `msg.Hookf` and the run stops, like
the `git.Clean` failure beside it (fail fast, no guessed default).
Then:

- marker on when the upstream is known and the count is 0; subject is
  `plan: <title> [skip ci]`
- marker off otherwise; subject is `plan: <title>`, and a `reason`
  string is set: `unpushed commits precede it` when the upstream is
  known, `the branch has no upstream` when it is not

The one subject feeds both the `--dry-run` preview line
(`preserve.go:285`) and `git.CommitPaths` (`preserve.go:312`), so the
preview can never disagree with the commit.

Output when the marker is off, in every path that commits:

- response message: `Approved plan committed: <path> ([skip ci] left off: <reason>)`,
  and the same clause on the `committed and pushed` and `push failed`
  messages
- dry-run: a line `  CI:     [skip ci] left off: <reason>` after the
  `Commit:` line

When the marker is on, output is exactly what it is today. No count
appears in the text: the number is in `git log @{u}..HEAD`, and prose
with a number needs plural handling the package has no helper for.

### 4. Tests, `internal/preserve/preserve_test.go`

- New fixture `withUpstream(t, dir)`: bare origin in `t.TempDir()`,
  `remote add origin`, `push -q -u origin main`. Replaces the inline
  three-step loop in `TestPushCarriesOnlyThePlanCommit` (line 167),
  which becomes a second caller.
- `TestAutoModePreservesAndCommits` (line 102) gains `withUpstream`, so
  HEAD is on the remote and its `[skip ci]` assertion at line 121
  stays as the plan-only case.
- New `TestMarkerLeftOffWhenUnpushedCommitsPrecedeThePlan`:
  `withUpstream`, commit one more file without pushing, preserve;
  subject is `plan: Ship The Widget`, response contains
  `[skip ci] left off: unpushed commits precede it`; a `--dry-run`
  first shows the same subject and the `CI:` line.
- New `TestMarkerLeftOffWithoutAnUpstream`: plain `scratch` (no
  remote); subject has no marker, response names `no upstream`.
- `TestCRLFPlanKeepsCleanTitle` (line 416): the repo has no remote, so
  the expected subject becomes `plan: Windows Plan`; the test's point,
  no `\r` in the subject, is unchanged.
- `TestPushCarriesOnlyThePlanCommit` now pushes with `-u`, so its
  pushed commit carries the marker; its assertions do not look at the
  subject, so nothing else changes.

### 5. `skills/preserve/SKILL.md`

One sentence after "commits it with a `plan:` message" in the opening
paragraph, stating what you get: the subject carries `[skip ci]` when
the plan is the only unpushed work on the branch; when unpushed
commits precede it, or the branch has no upstream, the marker is left
off and the output says so. Then `make docs` to recompile
`internal/help/data` (committed output).

No other page names the marker (grep of skills/, README, docs/ outside
the immutable plans). The command `Summary` and skill `description`
state purpose and stay as they are. No release note: a `fix` commit is
a patch.

## DRY boundaries

- Push-state check: `git.Ahead`, two callers (preserve, changelog
  undo); the inline copy in undo is removed, and a grep for
  `@{u}` / `@{upstream}` outside `internal/git` must come back empty.
- Subject string: built once in `preserve()`, read by the preview and
  the commit.
- Reason text: one `reason` value feeds the dry-run line and the three
  response messages.
- Origin fixture: `withUpstream` in the test file, two callers.

## Verification

1. `make test` before the change (passed at session start: 18 packages ok) and after.
2. `make build && make docs`; `git status` shows only the intended files.
3. Smoke test in a scratch repo under the session scratchpad, run with
   `./pk` (not the installed pk), after `pk init`:
   - HEAD pushed to a bare origin with upstream set, preserve a plan:
     `git log -1 --format=%s` ends in `[skip ci]`; output has no reason clause.
   - Commit one file without pushing, preserve another plan (`--plan`):
     subject has no marker; output ends `([skip ci] left off: unpushed commits precede it)`.
   - `--dry-run` on the same state prints the bare subject and the `CI:` line.
   - A repo with no remote: subject has no marker; output names `no upstream`.
   - Must fail: `pk changelog --undo` on a release commit that is on the
     remote still exits 2 with `already on the remote`.
4. `claude plugin validate . --strict` passes (skill text changed).
5. Commit on `develop` as `fix(preserve): add [skip ci] only when the plan is the only unpushed work`.
