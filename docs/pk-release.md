# pk release

Read the `Release-Tag` trailer from HEAD, create the git tag, merge to the release branch, validate, and push.

## Usage

```bash
pk release                        # tag, merge, validate, and push
pk release --dry-run              # validate without tagging, merging, or pushing
```

## How it works

When `release.branch` is configured in `.pk.json`:

1. **Note current branch** — this is the source branch (no hard-coded name).
2. **Read `Release-Tag:` trailer from HEAD** (written by `pk changelog`) and validate it as semver. Refuses if the trailer is missing or invalid.
3. **Check the tag doesn't already exist locally** — refuses if it does.
4. **Pre-flight checks** — clean working tree, source branch exists on origin, source branch not behind remote.
5. **Switch to release branch** and merge from source (`git merge --ff-only`). Fails if not fast-forward.
6. **Run pre-release hook** if configured.
7. **Create the git tag** on HEAD (the fast-forwarded release branch points at the same commit as the source branch).
8. **Push** release branch + tag to origin.
9. **Switch back** to source branch and push it to sync origin.

When `release.branch` is NOT configured (legacy flow):

1. Read `Release-Tag:` trailer from HEAD and validate it as semver.
2. Check the tag doesn't already exist locally.
3. Pre-flight checks — clean working tree, current branch exists on origin, not behind remote.
4. Run pre-release hook if configured.
5. Create the git tag on HEAD.
6. Push current branch + tag to origin.

On any failure after the tag is created but before the push completes, `pk release` deletes the local tag automatically so the next run starts from a clean state.

## Flags

- **--dry-run** — Run all checks without tagging, merging, or pushing. In the merge flow, verifies that a fast-forward merge is possible.

## Requirements

- **git 2.32 or newer** for `git log --format=%(trailers:...)` and `git commit --trailer`.

## Configuration

Add a `release` key to `.pk.json`:

```json
{
  "release": {
    "branch": "main",
    "hooks": {
      "preRelease": "go test -race ./..."
    }
  }
}
```

- **branch** — The release branch that `pk release` merges to and pushes from. The current branch is the implicit source — no hard-coded "dev" name. If omitted, `pk release` uses the legacy flow (validate current branch and push).
- **hooks.preRelease** — Shell command that runs before pushing. If the hook fails, the release is aborted and nothing is pushed.

## Details

### Workflows

| Flow | Config | Command | What happens |
|------|--------|---------|--------------|
| Legacy | no `release.branch` | `pk release` | Tag HEAD, push current branch + tag |
| Merge | `release.branch` set | `pk release` | Tag, merge to release branch, push both |

### Release-Tag trailer

`pk release` reads the pending version from the `Release-Tag:` trailer on HEAD, which `pk changelog` wrote when it generated the release commit. See [pk changelog — Release-Tag trailer](pk-changelog.md#release-tag-trailer) for the format and rationale.

The trailer value is validated as strict semver: it must parse via plankit's semver parser and round-trip back to the same string. Missing, malformed, or non-semver values are refused with a clear error.

### Merge behavior

The merge uses `git merge --ff-only`. If the release branch has diverged (e.g., someone committed directly to it from the terminal), the merge fails with:

```
Error: merge failed — main has diverged from dev (not fast-forward).
Resolve on main manually, then try again.
```

### Error recovery

If any step fails after switching to the release branch (merge, hook, push), `pk release` automatically switches back to the source branch before exiting.

### Guard interaction

`pk release` runs git commands internally via `exec.Command`, not through Claude Code's Bash tool. This means `pk guard` (a PreToolUse hook that only intercepts Bash tool calls) does not block `pk release`. Guard blocks everything else on protected branches — `pk release` is the single command that legitimately touches the release branch.

If you are already on the release branch when you run `pk release`, it refuses with an error: "switch to your development branch first." This prevents accidental pushes without a merge.

### Scope

Guard and `release.branch` are for multi-branch workflows (e.g., dev/main). Single-branch developers working directly on `main` don't need guard or `release.branch` — they run `pk changelog` and `pk release` with the legacy flow. No configuration needed.
