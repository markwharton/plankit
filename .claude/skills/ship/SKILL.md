---
name: ship
description: "Ship a release: changelog, tag, merge, and push in one pass"
disable-model-invocation: true
allowed-tools: Bash(pk:*), Bash(git:*)
argument-hint: [auto]
pk_sha256: 4e48f21a40946403b4fc2b7a512c1103b6180fb920a934cb983240e402796def
---

The release workflow. `pk changelog` and `pk release` are always run in sequence when shipping a version; this skill chains them while preserving the preview+confirm gate for each step so nothing lands unreviewed.

Run this on the branch where you've been working. For develop→main projects, that's `develop`; for trunk-based projects, that's the main branch. `pk release` refuses to release directly from a configured release branch.

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

## Auto mode

When invoked as `/ship auto`, proceed through each step without pausing for confirmation as long as the `--dry-run` preview shows no errors. If either dry-run produces an error or unexpected output, stop and ask before continuing.

Auto mode changes steps 2 and 3: run the dry-run, check for errors, and if clean, execute immediately rather than showing the preview and waiting for approval.

## Rules

- Never skip a confirmation unless auto mode is active and the dry-run completed without errors.
- If the user declines at step 2, stop — do not proceed to step 3.
- If `pk changelog` succeeds but `pk release` fails, the user can simply re-run `/ship` — step 1 will detect the `Release-Tag` trailer and resume at step 3.
- If the user wants to back out after step 2 but before step 3, run `pk changelog --undo` — never `git reset`. The command refuses unless HEAD is the unpushed `pk changelog` commit and the tree is clean.
- Never run `git push` directly. `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.
