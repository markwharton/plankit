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

A configured repository that has commits but no tag, or whose release
branch is its only branch, gets a note naming the command that
finishes the bootstrap. `--quiet` drops the notes.

## Flags

<!-- generated: flags -->
```
  --format <value>
        Output format: text or json (default text)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
