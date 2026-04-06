---
name: release
description: Validate pre-flight checks and push the release to origin
---

Push a release created by pk changelog.

**Always use `pk release` to push — never run `git push` directly.** `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
