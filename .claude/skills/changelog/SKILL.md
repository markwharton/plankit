---
name: changelog
description: Update CHANGELOG.md from git history and commit (tag is created by pk release)
disable-model-invocation: true
allowed-tools: Bash(pk:*)
pk_sha256: 026d0186aee13ab0d1b46de39fa3398e41d5b272a1e7db851a7c9defdf3ff55a
---

Generate a changelog release using pk.

Run this on a development branch, not on a guarded branch (e.g., main).

First, preview with a dry run:

pk changelog --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk changelog

The commit carries a Release-Tag trailer; no git tag is created yet.

To back out before `/release`, run `pk changelog --undo` — never `git reset`. The command refuses if HEAD has been pushed or the tree isn't clean.

Report the result to the user. Follow with `/release` to tag, merge, and push.
