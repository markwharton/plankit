---
name: changelog
description: Update CHANGELOG.md from git history, commit, and tag version
---

Generate a changelog release using pk.

Run this on a development branch, not on a guarded branch (e.g., main).

First, preview with a dry run:

pk changelog --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk changelog

Report the result to the user. Follow with `/release` to merge and push.
