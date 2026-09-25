# pk preserve --plan: preserve a plan revised after approval

## Context

The ExitPlanMode hook records the approved plan's path in `.git/pk-pending-plan`; a typed `pk preserve` consumes the pointer and commits the plan. When the plan file in `~/.claude/plans/` is revised after that, there is no pointer, and a second `pk preserve` says "No pending plan to preserve." Today the only way to preserve the revision is to reconstruct the hook's stdin payload by hand and pipe it in to re-arm the pointer. The help does not say how the pointer is set or where it lives, so even finding that detour takes reading `hooks/hooks.json` and the binary's strings.

Outcome: `pk preserve --plan <path>` preserves the named plan under the existing rules, and the preserve page explains the pointer and the revised-plan flow. The hook path and the pointer flow are unchanged. Suite status at session start: all packages pass.

## Change 1: the flag (`feat(preserve)`)

**`internal/preserve/preserve.go`**

- Add `{Name: "plan", Type: cli.StringFlag, Usage: "Plan file to preserve, for a plan revised after approval (a path under ~/.claude/plans)"}` to `Cmd.Flags`, and order the three flags alphabetically (`dry-run`, `plan`, `push`) as the changelog and init pages do.
- In `run`, read `ctx.String("plan")` first. When it is set, do not read stdin at all: the run is explicit (`hookInvocation = false`, no payload cwd). The resolved directory then comes from `--project-dir`, `PK_PROJECT_DIR`, `CLAUDE_PROJECT_DIR`, or the process directory, as a typed run resolves today.
- Resolve the typed value through the existing `matchPlanPath(value, os.UserHomeDir)`, which normalizes backslashes and expands `~/`. An empty match returns `cli.Usagef` naming the value and the rule (a `.md` under `~/.claude/plans/`), exit 1.
- The named file must exist: a failed `os.Stat` returns `cli.Statef` naming the path, exit 2. A plan below `minPlanSize` returns `cli.Statef` with the size and the minimum, exit 2 (the hook stays silent for this, as today; a person who named the file is told why).
- Not a git repository returns `cli.Statef` naming the directory, exit 2.
- Pointer rule: if `readPointer(root)` returns a path different from the matched one, return `cli.Statef` naming both files and the fix ("run pk preserve to commit the pending plan first"), exit 2, and change nothing. A pointer naming the same file proceeds; the existing `removePointer` calls after commit or duplicate consume it.
- From there the existing path runs unchanged: `mode = "auto"`, title, slug, `scanDestDir` for the duplicate check and the next sequence number, the `--dry-run` preview, write, `git add`, the identical-bytes check, the `plan: <title> [skip ci]` commit, `--push`. Output shape for a typed run stays what the typed run prints today.

**`internal/preserve/preserve_test.go`**

- Split `runPreserve` into a code-returning helper (`runPreserveIO` returning out, errw, code) with `runPreserve` as the fatal-on-nonzero wrapper, so refusal tests can assert the exit code.
- Add, each on a scratch repo from `scratch`:
  - `TestPlanFlagPreservesARevisedPlanWithoutAPointer`: manual hook run, typed run commits `001`; rewrite the plan file with an added section; `--plan <path>` commits `2026-09-05-002-...`, pointer absent, commit count up by one.
  - `TestPlanFlagIdenticalBytesReportTheExistingFile`: auto hook run, then `--plan` on the same file: no commit, one entry, output says already preserved.
  - `TestPlanFlagRefusesAPathOutsideClaudePlans`: `--plan` on a file under a plain `t.TempDir()`: exit 1, stderr names the path, nothing committed.
  - `TestPlanFlagRefusesWhenThePointerNamesAnotherPlan`: manual hook run sets the pointer to plan A; write plan B beside it; `--plan B` exits 2, stderr names both files, pointer still present, no commit.
  - `TestPlanFlagDryRunPreviewsOnly`: `--plan --dry-run` prints the `001` filename on stderr, no commit, no `docs/plans/`.

**Generated output**: `make docs` regenerates the Flags block in `skills/preserve/SKILL.md` and `internal/help/data`; both are committed with this change. No hand edit to the block.

## Change 2: the help (`docs(preserve)`)

**`skills/preserve/SKILL.md`**: add a section after "Completing a manual preserve":

> ## Preserving a plan revised after approval
>
> The approval hook reads the plan's path from `tool_response.filePath` on stdin and records it in `.git/pk-pending-plan`. A typed `pk preserve` consumes the pointer. A plan revised after it was preserved has no pointer: preserve it again with `--plan <path>`, and it becomes the next sequence number for the day. The same rules apply: the path is a Claude Code plan under `~/.claude/plans/`, and identical bytes report the existing file. While the pointer names a different plan, `--plan` refuses and names both files; preserve the pending plan first.

Prose, no placeholders beyond `<path>` in the flag form the page already uses for flags; `pk preserve` form throughout. `make docs` recompiles `internal/help/data`, committed with it.

**Keeping the hand-named fact honest**: the pointer filename is a Go constant, and the page now names it. Add `TestPageNamesThePointerFile` in `preserve_test.go` that reads `../../skills/preserve/SKILL.md` and fails unless it contains `.git/` + `pointerFilename`, in the style of `TestChangelogSkillListsDefaultTypeTable`.

`pk brief`, the SessionStart text, `Summary`, and `description` stay as they are: both one-liners still state purpose. `docs/architecture.md` needs no change.

## Commits

Two commits on `develop`, Conventional Commits, no breaking marker, no Co-Authored-By trailer, no push:

1. `feat(preserve): --plan names the plan to preserve, for a plan revised after approval`
2. `docs(preserve): how the pointer is set and where it lives`

`make test` before and after each, reading the fail count.

## Verification

- `make test` clean at each step (baseline already green).
- `make build`, then in a scratch repository under the scratchpad (git init, `.pk.json` via `./pk init`):
  - `./pk preserve --plan ~/.claude/plans/wise-crafting-dijkstra.md --dry-run` prints the `001` preview.
  - `./pk preserve --plan ~/.claude/plans/wise-crafting-dijkstra.md` commits `docs/plans/<date>-001-...`; a second run reports the existing file.
  - Pipe the hook payload for another plan to set a pointer, then `--plan` on the first file: exit 2, message names both.
  - Must fail: `./pk preserve --plan /tmp/not-a-plan.md` exits 1 and names the rule.
- `./pk help preserve` shows the new section and the `--plan` line.
- `gofmt -l` empty; `make docs` leaves the tree clean after commit.
