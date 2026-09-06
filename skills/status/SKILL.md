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
