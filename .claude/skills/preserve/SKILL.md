---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
disable-model-invocation: true
allowed-tools: Bash(pk:*)
pk_sha256: b201b6dcd420d994ec65cd0b53b0be78c37dd4d9be7fd1548de8fbb23c3a0f45
---

Preserve the most recently approved plan to docs/plans/ and commit it.

First, preview with a dry run:

pk preserve --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk preserve

This commits the plan locally. Do not push — the user decides when to push.

Report the result to the user.

With the plan preserved, proceed with its implementation.
