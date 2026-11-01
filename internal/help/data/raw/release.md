---
name: release
description: Tag the pending release from the Release-Tag trailer and push it, with pre-flight checks and rollback
---

# pk release

Completes what `pk changelog` staged: reads the `Release-Tag` trailer
on HEAD, runs pre-flight checks, creates the tag, and pushes
atomically to origin.

## Two flows

Merge flow keeps two branches: work lands on the working branch, and a
release fast-forwards the release branch to it, tags, pushes branch and
tag together, switches back, and pushes the working branch. Trunk flow
keeps one: a release tags and pushes the default branch, and refuses
to run anywhere else. `release.branch` set selects merge flow; empty
selects trunk flow.

Merge flow needs a second place where unreleased work can live and
run. For code, that place is the developer's own machine: check out the
working branch, build it, run it. A repository that is itself a live
system has no such place unless a second system is set up for
development. A Microsoft Fabric workspace connected to `main` is a live
system; development would mean a second workspace, with every connector
and credential set up again: a second bill, and a second copy of the
data to secure. Until that is worth doing, use trunk flow. The choice
is a cost, not a commitment: either flow becomes the other later, and
tags made under one stay valid under the other.

To move to merge flow, create the working branch from the release
branch, set `release.branch` and `guard.branches` on it, and ship; the
first release fast-forwards. To move to trunk flow, ship once more,
delete the working branch, and on the release branch remove those two
keys from your own terminal, since guard blocks a session committing
on a guarded branch.

## Pre-flight

Clean tree; branch on origin and not behind it; release branch
resolvable and not diverged from the working branch; the last version
tag in the working branch's history. A failure after
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

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
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
