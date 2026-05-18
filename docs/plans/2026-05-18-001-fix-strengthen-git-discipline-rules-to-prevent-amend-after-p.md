# Fix: Strengthen git-discipline rules to prevent amend-after-push and blind recovery

## Context

Catastrophic incident in timekeeper: the model amended a commit that was already pushed, then blindly ran `git pull --rebase` when it encountered "local is behind origin" — creating duplicate commits in permanent pushed history. Two distinct failures:

1. Amended without verifying push state (existing rule says "locally you have full freedom" which gives blanket permission)
2. Hit unexpected state and reached for a destructive command instead of investigating

The existing rules implicitly assume the model knows whether a commit is pushed. It doesn't reliably track this. Needs explicit verification gates and a "stop, don't fix" rule for unexpected state.

## Changes

### File: `internal/setup/rules/git-discipline.md` (embedded source)

**1. Modify "Don't push" rule** — remove blanket "full freedom" language, scope to unpushed:

```
- **Don't push your work until you're happy with it.** Unpushed commits are yours to amend, reorder, and combine. Once pushed, history is shared and rewriting creates problems downstream.
```

**2. Add new rule after "Don't push"** — the verification gate:

```
- **Verify push state before any history rewrite.** Before `--amend`, soft reset, or any operation that rewrites a commit, run `git log --oneline @{push}..HEAD` (or compare against `origin/<branch>`) to confirm the target commit has not been pushed. If the command errors (no upstream) or the target commit is not in the output, it has been pushed — make a new commit instead. Never assume a recent commit is local; always check.
```

**3. Modify "Rewrite unpushed commits with soft reset"** — add push-state check as step 2:

```
- **Rewrite unpushed commits with soft reset.** To fold an edit into an earlier commit: `git log --oneline` (note hashes); confirm the target commit appears in `git log --oneline @{push}..HEAD` (unpushed); verify the target is the commit you intend to modify, not an unrelated commit that landed after it; `git reset --soft <target>~1`; `git restore --staged <files-for-later-commits>`; edit; `git add` + `git commit -C <target-hash>`; then re-stage and re-commit later files with their hashes. Reflog recovers mistakes within ~30 days.
```

**4. Add new rule after "Don't improvise git history rewrites"** — stop on unexpected state:

```
- **When git state is unexpected, stop and investigate.** If a command reports diverged branches, "local is behind remote", or any state you didn't anticipate, do not reflexively run `git pull`, `git pull --rebase`, `git merge`, or `git reset` to "fix" it. These can replay or duplicate commits irreversibly. Instead: run `git log --oneline --graph HEAD origin/<branch>` to understand the divergence; report what you see to the user; and wait for explicit instructions.
```

**5. Update frontmatter description:**

```
description: Verify before rewriting, commit with purpose, conventional commits, commit before risk
```

### File: `.claude/rules/git-discipline.md` (local copy)

Apply same content changes, then recompute `pk_sha256` from the body (excluding frontmatter).

### Final rule order

1. Don't push your work until you're happy with it. (modified)
2. Verify push state before any history rewrite. (NEW)
3. Commit and push are separate decisions. (unchanged)
4. Never force push. (unchanged)
5. Rewrite unpushed commits with soft reset. (modified)
6. Don't improvise git history rewrites. (unchanged)
7. When git state is unexpected, stop and investigate. (NEW)
8. Don't rewrite history between `pk changelog` and `pk release`. (unchanged)
9. Commit with purpose. (unchanged)
10. Configure automation... (unchanged)
11. Match message weight... (unchanged)
12. Never include BREAKING CHANGE (unchanged)
13. Commit before risk. (unchanged)

## Critical files

- `/Users/markwharton/Projects/markwharton/plankit/internal/setup/rules/git-discipline.md` — embedded source (edit this first)
- `/Users/markwharton/Projects/markwharton/plankit/.claude/rules/git-discipline.md` — local copy with `pk_sha256`

## Verification

1. `make build` — confirms embed compiles
2. `make test` — no test changes needed (rule content, not logic)
3. `make lint` — clean
4. Smoke: run `pk setup --dry-run` (or inspect the sha computation) to confirm the local copy's hash matches the embedded source body
5. Read both files and confirm they match (minus frontmatter differences)
