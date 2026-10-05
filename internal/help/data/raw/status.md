---
name: status
description: Report plankit configuration and repository state
---

# pk status

Reports the resolved policy from `.pk.json`, the current branch and
tree state, the preserved plan count, and the latest tag.

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
with the way out. One whose `.claude/settings.json` lacks a
plugin entry gets a note with the JSON to merge in. `--quiet` drops
the notes.

## Flags

<!-- generated: flags -->
```
  --format <value>
        Output format: text or json (default text)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
