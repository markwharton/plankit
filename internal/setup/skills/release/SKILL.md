---
name: release
description: Merge to release branch, validate, and push to origin
---

Push a release created by pk changelog. When `release.branch` is configured
in `.pk.json`, this command merges to the release branch, pushes, and switches back.

**Always use `pk release` to push — never run `git push` directly.** `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
