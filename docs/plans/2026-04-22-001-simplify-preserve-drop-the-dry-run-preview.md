# Simplify /preserve: drop the dry-run preview

## Context

The `/preserve` skill has one job: save an already-approved plan to `docs/plans/` and commit it with a `plan:` conventional commit message. Today the skill instructs Claude to run `pk preserve --dry-run`, show a preview, ask for confirmation, and only then run `pk preserve`. In practice that preview-confirm step invites Claude to editorialize. In the session that prompted this change, Claude added a false caveat about uncommitted CSS changes possibly getting "swept in" and offered two confusing options — even though `pk preserve` stages only the plan file by path at `internal/preserve/preserve.go:170` (`cfg.GitExec(projectDir, "add", relPath)`) and cannot touch other working-tree changes.

Since the plan was already approved via `ExitPlanMode`, a second approval on a deterministic, narrowly-scoped commit is redundant ceremony. The fix: remove the dry-run preview from the skill so `/preserve` runs `pk preserve` directly and reports the command's `systemMessage` back to the user. The `--dry-run` flag stays on the `pk` CLI for power users running the command from the terminal.

## Changes

### 1. Simplify the preserve skill (embedded source)

File: `internal/setup/skills/preserve/SKILL.md`

Replace the body (everything after the frontmatter) with:

```
Preserve the most recently approved plan to docs/plans/ and commit it.

Run:

pk preserve

This commits the plan locally with a `plan:` conventional commit. Do not push — the user decides when to push.

Report the result to the user.

With the plan preserved, proceed with its implementation.
```

No preview, no confirmation prompt. The command's `systemMessage` (e.g. `"Approved plan committed: docs/plans/…"`) is the user-visible report.

### 2. Update the local skill copy

File: `.claude/skills/preserve/SKILL.md`

Apply the same body change. Then recompute the body hash (per `CLAUDE.md` line 79):

```bash
sed -n '/^---$/,/^---$/!p' internal/setup/skills/preserve/SKILL.md | shasum -a 256
```

Replace the `pk_sha256:` value in the local copy with the new hash so `pk setup` treats the file as pristine rather than user-modified.

### 3. Update docs/pk-preserve.md

File: `docs/pk-preserve.md:27`

Current:
> **--dry-run** — Preview the plan title, destination file, and commit message without writing, committing, or pushing. Used by the `/preserve` skill for confirmation before proceeding.

New — drop the second sentence; `--dry-run` is now a terminal power-user flag:
> **--dry-run** — Preview the plan title, destination file, and commit message without writing, committing, or pushing.

### 4. Update CLAUDE.md

File: `CLAUDE.md:65`

Remove `preserve` from the "Skill-managed commands" list — after this change, `/preserve` no longer uses a preview/confirm cycle. It already appears (correctly) in the "Hook commands" list on line 64.

Current:
> - **Skill-managed commands** (changelog, preserve, release) — `/command` skills handle the preview/confirm cycle. `--dry-run` exists for the skill to preview before executing. Power users typing `pk command` in the terminal bypass the skill and execute directly.

New:
> - **Skill-managed commands** (changelog, release) — `/command` skills handle the preview/confirm cycle. `--dry-run` exists for the skill to preview before executing. Power users typing `pk command` in the terminal bypass the skill and execute directly.

### 5. Update docs/creating-skills.md

File: `docs/creating-skills.md:73`

Current:
> Before destructive or hard-to-reverse actions, run the dry-run version, show the user what will happen, and ask for confirmation. The `/changelog`, `/preserve`, and `/release` skills all use this pattern:

New — drop `/preserve` from the example list:
> Before destructive or hard-to-reverse actions, run the dry-run version, show the user what will happen, and ask for confirmation. The `/changelog` and `/release` skills use this pattern:

## Out of scope

- **`pk preserve --dry-run` flag** stays on the CLI. It still works from the terminal; the skill just no longer invokes it.
- **`pk preserve` staging logic** is already correct (single-file `git add` by path at `internal/preserve/preserve.go:170`). No code changes.
- **`pk_sha256` test** at `internal/setup/setup_test.go:77` only checks that the field is present, not a specific value — no test update needed.
- **Other skills** (`/changelog`, `/release`, `/ship`) keep their preview/confirm cycle; those commands are hard-to-reverse (CHANGELOG edits, tags, pushes) and warrant the extra gate.

## Verification

1. **Build**: `make build` — expect clean build, `dist/pk` produced.
2. **Tests**: `make test` — all tests pass (no preserve tests should break; the change is skill-side only).
3. **SHA regeneration check**: run `dist/pk setup --project-dir $(mktemp -d)` and verify the emitted `.claude/skills/preserve/SKILL.md` matches the new body and embeds the same `pk_sha256` as the local copy.
4. **Smoke test `/preserve`**:
   - Enter plan mode on any small task, call `ExitPlanMode`, approve.
   - Type `/preserve` and confirm Claude runs `pk preserve` directly (no `--dry-run`, no "1) run as-is / 2) check git status" prompt).
   - Confirm the commit lands on the current branch as `plan: <title> [skip ci]` and contains only the new file under `docs/plans/`.
5. **Smoke test direct CLI**: run `pk preserve --dry-run` in the terminal and confirm the preview output still works for power users.
6. **Negative check**: with an uncommitted edit in the working tree, run `/preserve` and confirm the uncommitted edit stays unstaged after the commit — proves the skill change doesn't affect what `pk preserve` stages.
