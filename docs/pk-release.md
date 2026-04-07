# pk release

Merge to the release branch, validate pre-flight checks, and push to origin.

## Usage

```bash
pk release                        # merge, validate, and push
pk release --dry-run              # validate without merging or pushing
```

## How it works

When `release.branch` is configured in `.pk.json`:

1. **Note current branch** — this is the source branch (no hard-coded name).
2. **Pre-flight checks** — clean working tree, source branch not behind remote.
3. **Find version tag** at HEAD (optional — no tag is OK for CI/CD workflows without changelog).
4. **Switch to release branch** and merge from source (`git merge --ff-only`). Fails if not fast-forward.
5. **Run pre-release hook** if configured.
6. **Push** release branch + tag (if tag exists) to origin.
7. **Switch back** to source branch and push it to sync origin.

When `release.branch` is NOT configured (legacy flow):

1. Find version tag at HEAD (required).
2. Pre-flight checks — clean working tree, not behind remote.
3. Run pre-release hook if configured.
4. Push current branch + tag to origin.

## Configuration

Add a `release` key to `.pk.json`:

```json
{
  "release": {
    "branch": "main"
  }
}
```

This tells `pk release` which branch to merge to and push from. The current branch is the implicit source — no hard-coded "dev" name.

If `release.branch` is not set, `pk release` uses the legacy flow (validate current branch and push).

## Guard interaction

`pk release` runs git commands internally via `exec.Command`, not through Claude Code's Bash tool. This means `pk guard` (a PreToolUse hook that only intercepts Bash tool calls) does not block `pk release`. Guard blocks everything else on protected branches — `pk release` is the single command that legitimately touches the release branch.

If you are already on the release branch when you run `pk release`, it refuses with an error: "switch to your development branch first." This prevents accidental pushes without a merge.

## Merge behavior

The merge uses `git merge --ff-only`. If the release branch has diverged (e.g., someone committed directly to it from the terminal), the merge fails with:

```
Error: merge failed — main has diverged from dev (not fast-forward).
Resolve on main manually, then try again.
```

## Error recovery

If any step fails after switching to the release branch (merge, hook, push), `pk release` automatically switches back to the source branch before exiting.

## Pre-release hook

The `changelog.hooks.preRelease` field in `.pk.json` runs a shell command before pushing:

```json
{
  "changelog": {
    "hooks": {
      "preRelease": "go test -race ./..."
    }
  }
}
```

If the hook fails, the release is aborted and nothing is pushed.

## Flags

- **--dry-run** — Run all checks without merging or pushing. In the merge flow, verifies that a fast-forward merge is possible.

## Scope

Guard and `release.branch` are for multi-branch workflows (e.g., dev/main). Single-branch developers working directly on `main` don't need guard or `release.branch` — they run `pk changelog` and `pk release` with the legacy flow. No configuration needed.
