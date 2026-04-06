---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
---

Preserve the most recently approved plan to docs/plans/ and commit it.

First, preview with a dry run:

pk preserve --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk preserve

This commits the plan locally. Do not push — the user decides when to push.

Report the result to the user.
