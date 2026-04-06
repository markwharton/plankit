# pk release

Validate pre-flight checks and push the release tag to origin.

## Usage

```bash
pk release                        # validate and push
pk release --dry-run              # validate without pushing
pk release --branch develop       # release from a non-main branch
```

## How it works

`pk release` is the second step after `pk changelog`. Run them in sequence — any commits between `pk changelog` and `pk release` will move HEAD past the version tag, causing the release to fail.

1. **Find version tag at HEAD** — looks for a `v*` tag created by `pk changelog`. Exits with an error if none exists.
2. **Validate semver format** — ensures the tag matches `vX.Y.Z`.
3. **Pre-flight checks:**
   - Working tree is clean (no uncommitted changes)
   - On the expected branch (default: `main`)
   - Local branch is not behind the remote (fetches and compares)
4. **Run pre-release hook** — executes `hooks.preRelease` from `.changelog.json` if configured.
5. **Push** — pushes the branch and tag to origin.

## Pre-release hook

The `hooks.preRelease` field in `.changelog.json` runs a shell command before pushing. Use it for project-specific validation like tests or builds:

```json
{
  "hooks": {
    "preRelease": "go test -race ./..."
  }
}
```

Examples for different project types:

| Project type | `hooks.preRelease` |
|---|---|
| Go | `"go test -race ./..."` |
| Node/npm | `"npm test && npm run build"` |
| Python | `"pytest && mypy ."` |
| No tests | (omit — pk release just validates and pushes) |

If the hook fails, the release is aborted and nothing is pushed.

## Flags

- **--dry-run** — run all checks and hooks without pushing. Useful to verify everything passes before committing to the push.
- **--branch** — expected branch name (default: `"main"`). The release is aborted if the current branch doesn't match.
