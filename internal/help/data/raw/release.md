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
release branch to the working branch, tag, push branch and tag
together, switch back, push the working branch. Without it, the trunk
flow: tag and push the default branch, and refuse to run anywhere
else.

## Pre-flight

Clean tree; branch on origin and not behind it; release branch
resolvable and not diverged from the working branch. A failure after
tagging rolls back: the local tag is deleted, the merge is reset, and
the working branch is checked out.

## Hooks

`preRelease` runs before the tag exists and is rehearsed by
`--dry-run`. A `preRelease` hook that commits produces a commit the
tag then covers. `prePush` runs with the tag ref in existence, for
signing or artifact builds keyed on the tag. Both receive `$VERSION`
(no v) and `$TAG` (with it).

## Usage

```bash
pk release --dry-run
pk release
```
