# pk preserve commits only the plan file

## Context

`preserve()` in `internal/preserve/preserve.go:260-273` stages the plan
with `git add <relPath>`, checks `git diff --cached --quiet` over the
whole index, then runs a bare `git commit -m ...`. Whatever else was
staged goes into the `plan: ... [skip ci]` commit. Seen in another
repository: a staged deletion from earlier work landed in a plan commit
and left the branch broken, with `[skip ci]` keeping CI from catching it.

The root cause: preserve carries its own add/commit sequence where the
repository already has the right primitive. `git.CommitPaths`
(`internal/git/git.go:214`) stages the named paths and commits with
`--only`, leaving the rest of the index as it was; `pk init`
(`internal/repo/repo.go:179`) already uses it. The hook path, the
explicit run, and `--plan` all reach the same `preserve()` function, and
`--push` pushes the commit that function made, so one change covers all
of them.

Expected after the fix: a plan commit holds the plan file and nothing
else; other staged changes stay staged; unstaged changes are untouched.

## Change

`internal/preserve/preserve.go`, in `preserve()` after the file is written:

1. Replace the whole-index "unchanged" check with a check scoped to the
   plan path, `git.Clean(root, relPath)` (`internal/git/git.go:83`, runs
   `git status --porcelain -- <path>`). Clean means the identical bytes
   are already committed at that path: remove the pointer, report "Plan
   unchanged, no commit needed." as today. A clean-check error is
   reported with `msg.Hookf` the way the add error is today.
2. Replace the `git add` + `git commit` pair with
   `git.CommitPaths(root, fmt.Sprintf("%s: %s [skip ci]", config.PlanType, title), relPath)`,
   reporting a failure with `msg.Hookf(..., "git commit failed: %v", err)`.
3. Push and the response messages stay as they are.

No other change to messages (the "N staged changes left out" note was
considered and declined).

Grep check: the other bare commit in non-test code is
`internal/changelog/changelog.go:309`, which runs after
`git.CheckCleanTree` (line 93) refuses a dirty tree, so it cannot sweep
in unrelated staged work. It stays as is.

## Docs

`skills/preserve/SKILL.md` opening paragraph: say the commit holds only
the plan file, and anything else staged stays staged. One clause; the
skill page owns this concept, so `docs/architecture.md` and README are
not touched. Then `make docs` to recompile `internal/help/data`.

## Tests (`internal/preserve/preserve_test.go`)

Reuse `scratch`, `runPreserve`, `commitCount`, `plansDir`.

- `TestOtherChangesStayOutOfThePlanCommit`: in an auto-mode scratch repo,
  commit one more tracked file `b.txt`; then stage a new file `c.txt`,
  stage the deletion of `b.txt` (`git rm`), and edit `a.txt` without
  staging it. Record `git status --porcelain` before; run the hook. Assert: one new
  commit; `git show --name-status --format= HEAD` lists only the plan
  path; `git status --porcelain` after equals before (staged addition
  still `A `, staged deletion still `D `, unstaged modification still
  ` M` with unchanged bytes). Covers brief items 1-3.
- `TestPushCarriesOnlyThePlanCommit`: scratch repo plus a bare origin in
  `t.TempDir()` (`git init --bare`, `remote add origin`, push main).
  Stage an unrelated file, run with `--push`. Assert origin's `main`
  equals local HEAD, its tip commit's `--name-only` lists only the plan,
  and the unrelated file is still staged locally. Covers item 4.
- Item 5 (clean index as today) is already covered by
  `TestAutoModePreservesAndCommits`, `TestConfiguredDirIsUsed` (asserts
  the commit's file list), and `TestDuplicateContentIsNotRecommitted`.

## Verification

1. `make test` before the change (report status), and after.
2. Confirm the new staged-work test fails against the current code
   before the fix is applied (write test first, see it fail, then fix).
3. `make build`, then smoke test in a scratch repo with `./pk`:
   `git init`, commit a `.pk.json` with `preserve.mode: auto`, stage a deletion and an unrelated new file,
   leave one unstaged edit, write a plan under a temp `~/.claude/plans`
   path, and run `./pk preserve --plan <path>`. Show
   `git show --stat HEAD` (plan only) and `git status --short`
   (unchanged). The failing case: rerun `--plan` with the same file and
   see "already preserved", no new commit.
4. `make fmt`; `claude plugin validate . --strict`.

Commit on `develop` as `fix: pk preserve commits only the plan file`.
