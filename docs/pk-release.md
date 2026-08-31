# pk release

Read the `Release-Tag` trailer from HEAD, merge to the release branch when there is one, tag, and push branch and tag together.

## Usage

```bash
pk release                        # merge, tag, push
pk release --dry-run              # run every check and the preRelease hook; tag and push nothing
```

## How it works

With `release.branch` set (merge flow):

1. The current branch is the source branch.
2. Reads the `Release-Tag:` trailer from HEAD and validates it as strict semver. Refuses without it: `no Release-Tag trailer on HEAD; run pk changelog first`.
3. Refuses when the tag already exists locally.
4. Pre-flight: clean tree; source branch on origin and not behind it; release branch present locally or on origin, and `origin/<release>` an ancestor of HEAD.
5. Switches to the release branch and runs `git merge --ff-only` from the source branch. Not a fast-forward: `merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.`
6. Runs `preRelease` when configured. A failure stops the run; nothing is tagged.
7. Creates the tag on HEAD.
8. Runs `prePush` when configured. A failure deletes the local tag; nothing is pushed.
9. Pushes the release branch and the tag with `git push --atomic`: a rejected push updates no ref on origin.
10. Switches back to the source branch and pushes it.

Without `release.branch` (trunk flow): steps 2 and 3; pre-flight with the current branch, which must be origin's default branch, on origin and not behind it; then steps 6 to 9 on the current branch.

After any failure past the tag, the local tag is deleted; after any failure past the switch, the source branch is checked out again.

## Flags

- **--dry-run**: every check, including that the merge is a fast-forward, and the `preRelease` hook. No merge, tag or push. `prePush` does not run: its tag does not exist.

## Configuration

`release.branch` and `release.hooks`: see [.pk.json](pk-json.md#release).

## Limits

- Needs git 2.32 or newer, for `git log --format=%(trailers:...)` and `git commit --trailer`.
- Refuses when run on the release branch: `you're on the release branch "main"; switch to your working branch first`.
- `pk guard` never sees the push: `pk release` runs git as its own child process, not through the Bash tool. It is the one command that legitimately advances the release branch.
- A commit on the release branch that the source branch lacks is reconciled by merging it back (`git merge origin/main` on the source branch), never by force-push; with an unpushed release commit at HEAD, `pk changelog --undo` first and `pk changelog` after the merge. See [not fast-forward](error-reference.md#not-fast-forward).
