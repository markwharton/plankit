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

## Flags

<!-- generated: flags -->
```
  --dry-run
        Validate without merging, tagging, or pushing
```
<!-- /generated: flags -->

## Settings

<!-- generated: settings -->
The `release` section of `.pk.json`:

```
"release": {
  "branch": "<branch>",
  "hooks": {
    "prePush": "<command>",
    "preRelease": "<command>"
  }
}
```

- `release.branch`: a branch name; default the branch `pk init` ran on. The branch releases merge into; empty selects the trunk flow, which tags the default branch.
- `release.hooks.prePush`: a shell command; default none. Runs after the tag exists and before the push, with `$VERSION` and `$TAG`.
- `release.hooks.preRelease`: a shell command; default none. Runs before the tag is created, with `$VERSION` and `$TAG`; `--dry-run` rehearses it.

An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem.
<!-- /generated: settings -->
