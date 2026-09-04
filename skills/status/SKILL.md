---
name: status
description: Report plankit configuration and repository state
---

# pk status

Reports what plankit sees in this repository: the resolved policy from
`.pk.json`, the current branch and tree state, the preserved plan
count, and the latest tag.

## Usage

```bash
pk status
pk status --format json
```

Exit codes carry the answer for scripts: `0` means configured, `2`
means not configured or not a git repository. `--format json` emits
one object with the same fields as the text report.

An unconfigured repository is not an error, it is a state: plankit is
off wherever `.pk.json` is absent, and every hook exits immediately
there.
