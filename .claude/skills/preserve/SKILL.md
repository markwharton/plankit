---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
pk_sha256: f532434f2bace0307a64197c1b0cee9b5a890c5e5858e83b14ceb547bd71183d
---

Preserve the most recently approved plan to docs/plans/ and commit it.

First, preview with a dry run:

pk preserve --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk preserve

This commits the plan locally. Do not push — the user decides when to push.

Report the result to the user.
