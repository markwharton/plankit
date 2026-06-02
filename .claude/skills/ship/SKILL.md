---
name: ship
description: "Ship a release: changelog, tag, merge, and push in one pass"
disable-model-invocation: true
allowed-tools: Bash(pk:*), Bash(git:*)
argument-hint: [auto]
pk_sha256: 5cbe65bec52e8d90af5b05dbe5c6a7f03aa5dad6f2c70039a26be9e28f66dfa0
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

- **The only argument is `auto`; `/ship` takes no version.** The version is computed by `pk changelog` and shown in its dry-run. Ignore any version-like argument (e.g. `ship v0.4.0`): run the flow normally and let the dry-run reveal the real version. If a typed version differs from the computed one, note it in one calm line (the version is derived from commit types; for a deliberately larger bump use `pk changelog --bump minor|major`) and proceed with the computed version. Don't treat the mismatch as an error or interrogate the user.
- **Use the Bash tool for all commands.** pk requires a POSIX shell. Do not use the PowerShell tool.
- **Exit plan mode first.** If you are in plan mode when this skill is invoked, exit plan mode immediately before doing anything else. This skill executes commands — it does not need a plan.
- Never skip a confirmation unless auto mode is active and the dry-run completed without errors.
- If the user declines at step 2, stop — do not proceed to step 3.
- If `pk changelog` succeeds but `pk release` fails, the user can simply re-run `/ship` — step 1 will detect the `Release-Tag` trailer and resume at step 3.
- If the user wants to back out after step 2 but before step 3, run `pk changelog --undo` — never `git reset`. The command refuses unless HEAD is the unpushed `pk changelog` commit and the tree is clean.
- Never run `git push` directly. `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.
