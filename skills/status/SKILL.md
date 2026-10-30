---
name: status
description: Report plankit configuration and repository state
---

# pk status

Reports the resolved policy from `.pk.json`, the current branch and
tree state, the preserved plan count, the pending plan, and the latest
tag.

The tag line reads the clone. When the clone has no tags, as a fresh
clone of only the working branch does in a cloud sandbox, the line
says what origin has, `none locally; origin has v1.2.0`, read from
origin without fetching; `none yet` means neither has a version tag,
so nothing has been released. `pk changelog` fetches the tags itself
when it runs.

The release line names the branch releases merge into, or, when
`.pk.json` sets none, the trunk flow: `trunk flow: pk ship tags main,
origin's default branch`. The branch comes from the clone's record of
origin's default, or from origin when the clone holds none; only a
repository with no origin leaves it unnamed. An empty release branch
is that flow, not a gap in the policy. JSON carries it as
`releaseFlow`, `merge` or `trunk`.

The guard line ends with the protected branches, or `no protected
branches` when `guard.branches` is empty: then no branch is guarded
against mutation, while the push guard still applies on every branch.

## Usage

```bash
pk status
pk status --format json
```

Exit code `0` means configured; `2` means not configured or not a git
repository. `--format json` emits one object with the same fields as
the text report.

An unconfigured repository is a state, not an error: plankit is off
wherever `.pk.json` is absent.

A configured repository whose release branch is its only branch gets
a note naming the command that starts a working branch. One whose
release branch does not exist gets the command that creates it at the
root commit, carrying no unreleased work. In merge flow, a release
branch that has moved past its last tag is noted with the number of
commits not in a release. A last tag the checked-out branch does not
contain, the mark of a history rewritten after a release, is noted
with the way out. A pending plan, recorded by the preserve hook and
not yet committed, is shown with its path and noted with its two ways
out, `/plankit:preserve` and `pk preserve --discard`; a pointer whose
plan file is gone is marked missing. One whose `.claude/settings.json`
lacks a plugin entry gets a note with the JSON to merge in. `--quiet`
drops the notes.

## Flags

<!-- generated: flags -->
```
  --format <value>
        Output format: text or json (default text)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
