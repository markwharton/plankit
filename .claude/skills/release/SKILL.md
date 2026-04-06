---
name: release
description: Validate pre-flight checks and push the release to origin
pk_sha256: 0f2914bb66e3c7159b49a6329f543a818ba0b42f34a743169eea7970489c6419
---

Push a release created by pk changelog.

**Always use `pk release` to push — never run `git push` directly.** `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
