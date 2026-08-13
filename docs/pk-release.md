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
4. **Pre-flight checks** — clean working tree, source branch exists on origin and is not behind remote. Release branch resolves locally or on origin, and has not diverged from origin.
5. **Switch to release branch** and merge from source (`git merge --ff-only`). Fails if not fast-forward.
6. **Run pre-release hook** if configured.
7. **Create the git tag** on HEAD (the fast-forwarded release branch points at the same commit as the source branch).
8. **Push** release branch + tag to origin atomically (`git push --atomic`), so a rejected branch ref can never leave the tag stranded on origin.
9. **Switch back** to source branch and push it to sync origin.

When `release.branch` is NOT configured (trunk flow):

1. Read `Release-Tag:` trailer from HEAD and validate it as semver.
2. Check the tag doesn't already exist locally.
3. Pre-flight checks — clean working tree, current branch exists on origin, not behind remote.
4. Run pre-release hook if configured.
5. Create the git tag on HEAD.
6. Push current branch + tag to origin atomically (`git push --atomic`).

On any failure after the tag is created but before the push completes, `pk release` deletes the local tag automatically. The next run then starts from a clean state. Because the push is atomic, a rejected push updates no refs on origin — there is no partial remote state to clean up.

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
      "preRelease": "go test -race ./...",
      "prePush": "sign-tag $TAG"
    }
  }
}
```

- **branch** — The release branch that `pk release` merges to and pushes from. The current branch is the implicit source — no hard-coded "dev" name. If omitted, `pk release` uses the trunk flow (validate current branch and push).
- **hooks.preRelease** — Shell command that runs after merge but before the tag is created. If it fails, the release is aborted and nothing is pushed. Rehearsed by `--dry-run`; the release tag does not exist yet when it runs.
- **hooks.prePush** — Shell command that runs after tagging, before the push, so the tag ref exists (for signing or artifact builds). If it fails, the local tag is removed and nothing is pushed. Does not run under `--dry-run`.

Both hooks receive `$VERSION` (no leading `v`) and `$TAG` (with it) as environment variables.

Neither runs before a commit, so a file written by either hook leaves the working tree dirty after the release. To have a file edit committed and covered by the tag, use the `changelog` hooks instead: see [pk changelog — hooks](pk-changelog.md#hooks) and the [hook timeline](pk-json.md#hook-timeline) comparing all four.

## Details

### Workflows

`pk release` supports two flows. Pick whichever matches the project:

- **Merge flow** — projects with a protected main branch and a development branch where work happens before being promoted. Use when you want a separation between "where work lands" and "what gets released."
- **Trunk flow** — single-branch projects (content sites, fast iteration). No develop branch, no merge step. Use when you commit directly on the branch you ship from.

| Flow | Config | Command | What happens |
|------|--------|---------|--------------|
| Merge | `release.branch` set | `pk release` | Tag, merge to release branch, push both |
| Trunk | no `release.branch` | `pk release` | Tag HEAD, push current branch + tag |

### Release-Tag trailer

`pk release` reads the pending version from the `Release-Tag:` trailer on HEAD, which `pk changelog` wrote when it generated the release commit. See [pk changelog — Release-Tag trailer](pk-changelog.md#release-tag-trailer) for the format and rationale.

The trailer value is validated as strict semver: it must parse via plankit's semver parser and round-trip back to the same string. Missing, malformed, or non-semver values are refused with a clear error.

### Merge behavior

The merge uses `git merge --ff-only`. If the release branch has diverged (e.g., someone committed directly to it from the terminal), the merge fails with:

```
Error: merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.
```

That check guards the local merge. A pre-flight check guards the push target. If `origin/main` carries a commit your source branch doesn't, `pk release` fails before creating the tag with `Error: origin/main has diverged from develop; the release push would be rejected`. That is common when the release branch was set up outside the create-new-project flow. Both cases reconcile the same way — see [Error recovery](#error-recovery).

### Error recovery

If any step fails after switching to the release branch (merge, hook, push), `pk release` automatically switches back to the source branch before exiting.

Divergence means the release branch carries a commit that is not on the source branch, so the fast-forward merge can't proceed. Recovery is to reconcile that commit back into the source branch (`git merge origin/main`), not to drop it. If it was already pushed, never force push. When you have an unpushed `pk changelog` release commit, the order matters. Undo it before the merge and regenerate it after, so the `Release-Tag` trailer stays at HEAD instead of being buried under the merge commit. See [not fast-forward](error-reference.md#not-fast-forward) for the exact ordered steps.

### Guard interaction

`pk release` runs git commands internally via `exec.Command`, not through Claude Code's Bash tool. This means `pk guard` (a PreToolUse hook that only intercepts Bash tool calls) does not block `pk release`. Guard blocks everything else on protected branches — `pk release` is the single command that legitimately touches the release branch.

If you are already on the release branch when you run `pk release`, it refuses: "switch to your working branch first". This prevents accidental pushes without a merge.

### Scope

Guard and `release.branch` are for the merge flow. Trunk-flow projects don't need guard or `release.branch` — they run `pk changelog` and `pk release` directly on their working branch. No configuration needed; an empty `.pk.json` (or no `.pk.json` at all) is fine.
