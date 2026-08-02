---
description: Release flow invariants, version stamping, commit rewrite procedure, and development standards
kind: craft
---

# Plankit Craft

## Releases

- **The git tag is the single source of truth for version.** `pk changelog` computes the next version from the latest tag and the conventional commits since it, and writes it into configured files (`versionFiles` for a root JSON version field, `pk pin` for source constants, a hook script for anything else); `pk release` creates the tag. The version is never read out of a file: never read it from package.json or a source constant in code you write; a task that surfaces the release version goes through the changelog config.
- **All work happens on the source branch; the release branch advances only via `pk release`.** A commit made directly on the release branch breaks the fast-forward merge and the next release fails with "`<release>` has diverged from `<source>` (not fast-forward)". If that commit was pushed it can't be dropped (never force push); recovery means reconciling it back into the source branch.
- **Don't rewrite history between `pk changelog` and `pk release`.** Changelog captures commit SHAs, release publishes them; rewriting mid-flow produces stale references.
- **Release is not an ordinary push.** `pk release` fast-forward merges into the release branch, tags, and pushes as one atomic action, because the tag must travel with the merge that anchors it. Never push by hand to publish a release.
- **Commit `pk setup` updates on their own.** Keeps pk-upgrade churn distinguishable from project changes. Suggested message: `chore: update pk-managed files for v<VERSION>`.
- **Conventional Commits, honestly weighted.** `pk changelog` reads commit types, so follow the convention, and each commit is one logical change. Never include BREAKING CHANGE unless the change actually breaks: a decorative one forces a major version bump. Match message weight to change weight; substantive features get a body, small changes get one line.

## Rewriting Commits

- **Rewrite unpushed commits with soft reset; verify push state first.** Confirm the target commit appears in `git log --oneline @{push}..HEAD` (if the command errors or the target is absent, it has been pushed: make a new commit instead). Then: `git reset --soft <target>~1`; `git restore --staged <files-for-later-commits>`; edit; `git add` + `git commit -C <target-hash>`; re-stage and re-commit later files with their hashes. Don't improvise alternatives to this procedure.

## Development Standards

- **Preserve the structure you were given.** Let the data model drive the code. Never flatten structured data into flat lists then reconstruct with heuristics; the context is already lost.
- **Fail fast, no silent fallbacks.** When something is missing or wrong, fail with a clear message, never a made-up default.
- **Grep before done.** Update every related location together: when fixing a pattern or renaming, grep the repo for all instances. One fix is not done until every occurrence is fixed.
- **Work isn't done until automated checks and a smoke test pass.** Build, tests, and lint; then a manual end-to-end check with specific commands and observable outcomes, including at least one negative case, whenever the change alters observable behavior. Skip smoke for pure internal refactors.
- **Diagnostic scripts over rebuild cycles.** If you are about to do your second full rebuild while debugging, stop and write a minimal script that tests the specific issue.
- **A failed text search means "not found by this method", never "not present".** When absence drives a root cause or code change, confirm by parsing the structure (walk the JSON/XML/AST), not the serialized surface.
