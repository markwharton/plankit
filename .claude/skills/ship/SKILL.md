---
name: ship
description: Run pk changelog then pk release in one pass, with preview + confirm at each step
disable-model-invocation: true
allowed-tools: Bash(pk:*), Bash(git:*)
pk_sha256: e80fd3fdadb48141d5a45a8b540f038050d737cb17b36c0494b3acb69f26b73d
---

Combined changelog + release workflow. `pk changelog` and `pk release` are always run in sequence when shipping a version; this skill chains them while preserving the preview+confirm gate for each step so nothing lands unreviewed.

Run this on a development branch, not a guarded branch (e.g., `main`).

## Flow

1. **Detect state.** Check whether HEAD already carries a `Release-Tag` trailer (i.e., `pk changelog` ran previously but `pk release` did not). Run:

   git log -1 --pretty='%(trailers:key=Release-Tag,valueonly)'

   - Empty output → start at step 2 (changelog then release).
   - Non-empty output → skip step 2, jump to step 3 (release only). Tell the user: "HEAD already has a Release-Tag trailer — skipping changelog, going straight to release."

2. **Changelog preview + commit.**

   pk changelog --dry-run

   Show the preview to the user and ask for confirmation before proceeding. If confirmed, run:

   pk changelog

   The resulting commit carries a `Release-Tag` trailer; no git tag is created yet.

3. **Release preview + publish (tag, merge, push).**

   pk release --dry-run

   Show the preview to the user and ask for confirmation before proceeding. If confirmed, run:

   pk release

Report the final result to the user.

## Rules

- Never skip a confirmation. Each `pk` command gets its own `--dry-run` preview and explicit user approval before the real run.
- If the user declines at step 2, stop — do not proceed to step 3.
- If `pk changelog` succeeds but `pk release` fails, the user can simply re-run `/ship` — step 1 will detect the `Release-Tag` trailer and resume at step 3.
- If the user wants to back out after step 2 but before step 3, run `pk changelog --undo` — never `git reset`. The command refuses unless HEAD is the unpushed `pk changelog` commit and the tree is clean.
- Never run `git push` directly. `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.
