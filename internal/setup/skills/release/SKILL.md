---
name: release
description: Tag and push a release created by pk changelog
disable-model-invocation: true
allowed-tools: Bash(pk:*)
---

Tag and push a release created by pk changelog. Reads the Release-Tag trailer
from the HEAD commit, creates the git tag, and pushes. When `release.branch`
is configured in `.pk.json`, this command also merges to the release branch
and switches back.

**Always use `pk release` to push — never run `git push` directly.** `pk release` re-runs all pre-flight checks before pushing; bypassing it skips safety validation.

First, preview with a dry run:

pk release --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk release

Report the result to the user.
