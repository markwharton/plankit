---
name: release
description: Tag the pending release from the Release-Tag trailer and push it, with pre-flight checks and rollback
---

# pk release

Completes what `pk changelog` staged: reads the `Release-Tag` trailer
on HEAD, runs pre-flight checks, creates the tag, and pushes
atomically to origin.

## Two flows

With `release.branch` configured, the merge flow: fast-forward the
release branch to your working branch, tag, push branch and tag
together, switch back, push the working branch. Without it, trunk
flow: tag and push the default branch directly, and refuse to run
anywhere else.

## Pre-flight

Clean tree, branch exists on origin and is not behind it, release
branch resolvable and not diverged from your work. Failures after
tagging roll everything back: the local tag is deleted, the merge is
reset, and you are returned to your working branch, so an aborted
release leaves the repository as it found it.

## Hooks

`preRelease` runs before the tag exists and is rehearsed by
`--dry-run`; a hook that commits produces a commit the tag then
covers. `prePush` runs with the tag ref in existence, for signing or
artifact builds keyed on the tag. Both receive `$VERSION` (no v) and
`$TAG` (with it).

## Usage

```bash
pk release --dry-run
pk release
```
