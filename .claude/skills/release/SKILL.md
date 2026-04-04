---
name: release
description: Validate pre-flight checks and push the release to origin
---

Push a release created by pk changelog.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
