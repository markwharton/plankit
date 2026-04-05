---
name: release
description: Validate pre-flight checks and push the release to origin
pk_sha256: 7aa04f8cd061aa78a1ecf8a0a1bc372e2c998ea5cd6c3e41653d76b0ae1875e4
---

Push a release created by pk changelog.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
