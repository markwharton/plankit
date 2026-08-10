# Error Reference

Common errors from pk commands, what causes them, and how to recover.

## pk changelog

### no version tags found

```
Error: no version tags found
  To anchor at v0.0.0: pk setup --baseline [--at <ref>] --push
  or: git tag v0.0.0 && git push origin v0.0.0
```

**Cause:** `pk changelog` scans commits since the most recent semver tag. Without a tag, there is no starting point.

**Fix:** Run `pk setup --baseline` to create a `v0.0.0` anchor tag. Add `--push` to publish it. Use `--at $(git rev-list --max-parents=0 HEAD)` to include all prior commits in the first changelog entry.

### no version tags found locally

```
Error: no version tags found locally
  Origin has tags; fetch them: git fetch --tags
```

**Cause:** The remote has tags but they are not present locally. Common in shallow-clone cloud sandboxes.

**Fix:** Run `git fetch --tags`. The `install-pk.sh` bootstrap script does this automatically in cloud sandboxes.

### protected branch

```
Error: you're on "main" which is a protected branch; switch to your development branch first
  To start one: git switch -c develop && git push -u origin develop
```

**Cause:** `pk changelog` refuses to create release commits on branches listed in `guard.branches`.

**Fix:** Switch to your development branch (`git switch develop`). The create hint appears only when no other local branch exists — the main-only adoption case. See [Moving between setups](adoption.md#moving-between-setups).

### branch not on origin

```
Error: develop does not exist on origin
  To push it: git push -u origin develop
```

**Cause:** `pk changelog` checks that the current branch exists on the remote before committing. Without this, `pk changelog` succeeds but `pk release` fails, leaving a Release-Tag commit that requires a manual push to continue.

**Fix:** Push the branch: `git push -u origin develop`.

### working tree not clean

```
Error: working tree is not clean; commit or stash changes first
```

**Cause:** `pk changelog` and `pk release` require a clean working tree before proceeding.

**Fix:** Commit or stash your changes first.

### HEAD already pushed (--undo)

```
Error: HEAD is already on the remote; cannot undo a pushed commit
```

**Cause:** `pk changelog --undo` only rewinds unpushed commits to avoid rewriting shared history.

**Fix:** If the changelog commit has been pushed, create a new commit to correct it. Do not force push.

### changelog already pending

```
Error: changelog for v0.19.9 is already pending (HEAD has Release-Tag: v0.19.9)
  To complete the release: pk release
  To undo and start over:  pk changelog --undo
```

**Cause:** `pk changelog` was already run and committed a Release-Tag trailer on HEAD. Running it again without `pk release` or `pk changelog --undo` in between would create duplicate changelog sections.

**Fix:** Run `pk release` to complete the release, or `pk changelog --undo` to unwind the pending release and start over.

## pk release

### no Release-Tag trailer

```
Error: no Release-Tag trailer on HEAD; run pk changelog first
```

**Cause:** `pk release` reads the version from a git trailer on HEAD that `pk changelog` writes.

**Fix:** Run `pk changelog` first, then `pk release`. Or use `/ship` which chains them.

### on the release branch

```
Error: you're on the release branch "main"; switch to your working branch first
  To start one: git switch -c develop && git push -u origin develop
  Then: pk changelog && pk release
```

**Cause:** `pk release` merges from the source branch into the release branch. Running it directly on the release branch would skip the merge.

**Fix:** Switch to your working branch (`git switch develop`). The create hints appear only when no other local branch exists — the main-only adoption case. See [Moving between setups](adoption.md#moving-between-setups).

### invalid release branch

```
Error: invalid release.branch "--output=/tmp/x" in .pk.json; branch names cannot start with -
```

**Cause:** `.pk.json` names a `release.branch` beginning with `-`. Git branch names can never start with `-`, and passing one to git would be read as an option rather than a branch.

**Fix:** Correct `release.branch` in `.pk.json` to a real branch name.

### release branch missing

```
Error: release branch main does not exist locally or on origin
  To create it: git branch main && git push -u origin main
```

**Cause:** `.pk.json` names a `release.branch` that resolves neither as a local branch nor as a remote-tracking ref. Without this pre-flight check the failure would surface as a raw `git switch` error mid-release.

**Fix:** Create the branch and publish it, or correct `release.branch` in `.pk.json`.

### tag already exists

```
Error: tag v0.8.1 already exists locally; nothing to release
```

**Cause:** The tag from the `Release-Tag` trailer already exists. The release was already completed or partially completed.

**Fix:** If the release was already pushed, there is nothing to do. If the tag is leftover from a failed attempt, delete it (`git tag -d v0.8.1`) and retry.

### branch not on origin

```
Error: develop does not exist on origin
  To push it: git push -u origin develop
```

**Cause:** `pk release` verifies the source branch exists on the remote before proceeding.

**Fix:** Push the branch: `git push -u origin develop`.

### behind remote

```
Error: local develop is behind origin/develop; pull first
```

**Cause:** Someone pushed commits to the branch since your last pull.

**Fix:** Pull the latest changes: `git pull origin develop`.

### not fast-forward

```
Error: merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.
```

**Cause:** The release branch has commits that are not on the source branch. `pk release` only does fast-forward merges to avoid merge conflicts.

**Fix:** Reconcile the histories, then retry. If you have an unpushed `pk changelog` release commit at HEAD, undo it *before* the merge. That keeps the `Release-Tag` trailer at the released tip instead of burying it under the merge commit. `pk release` reads the trailer from HEAD only:

1. `pk changelog --undo` — drop the unpushed release commit (skip if you have not run `pk changelog` yet).
2. `git merge origin/main` — merge the release branch into your source branch to reconcile the divergent commit. That commit is already pushed and can't be dropped (never force push), so merging is how you keep it.
3. `pk changelog` — regenerate the release commit so the `Release-Tag` trailer sits at HEAD, above the merge commit.
4. `pk release` — fast-forwards cleanly now, pushing the release branch, source branch, and tag.

### release branch diverged from origin

```
Error: origin/main has diverged from develop; the release push would be rejected
```

**Cause:** `origin/main` carries a commit that your source branch does not, so the release push (branch + tag) would be rejected as non-fast-forward. Unlike [not fast-forward](#not-fast-forward), which is caught by the local merge, this is caught by a pre-flight check against `origin` before any tag is created. It commonly appears when the release branch was set up outside the create-new-project flow (e.g. `origin/main` was auto-created with an initial commit).

**Fix:** Reconcile `origin/main` into your source branch, then retry — the same ordered steps as [not fast-forward](#not-fast-forward). In short: `pk changelog --undo` (if you have an unpushed release commit), `git merge origin/main`, `pk changelog`, `pk release`.

### push failed

```
Error: git push failed: ...
```

**Cause:** The push was rejected by the remote (permissions, branch protection rules, or network issues).

**Fix:** `pk release` automatically cleans up the local tag on push failure. The push is atomic (`git push --atomic`), so a rejected push updates no refs on origin — there is no stray remote tag to remove. Fix the underlying issue (permissions, network) and run `pk release` again.

### pre-release hook failed

```
Error: pre-release hook failed: ...
```

**Cause:** The `release.hooks.preRelease` command exited non-zero. It runs after the merge but before the tag is created — commonly a final test or build gate that failed.

**Fix:** Nothing was tagged or pushed. Fix what the hook reported and run `pk release` again. The hook receives `$VERSION` (no leading `v`) and `$TAG` (with it). Because it runs before tagging, `pk release --dry-run` rehearses it, so you can reproduce the failure without releasing.

### pre-push hook failed

```
Error: pre-push hook failed: ...
```

**Cause:** The `release.hooks.prePush` command exited non-zero. It runs after the tag is created, before the push — commonly a signing or artifact-build step that needs the tag ref.

**Fix:** The release is aborted and nothing is published. `pk release` removes the local tag (and rolls back the merge in merge flow), so origin is untouched. Fix the hook and run `pk release` again. The hook receives `$VERSION` and `$TAG`, and the tag ref exists on disk while it runs. Note `--dry-run` does **not** exercise prePush (it returns before tagging); rehearse tag-dependent steps against a scratch tag instead.

## pk setup

### invalid mode

```
Error: invalid --preserve mode "xyz" (must be auto, manual, or off)
Error: invalid --guard mode "xyz" (must be block, ask, or off)
Error: invalid --push-guard mode "xyz" (must be block, ask, or off)
```

**Cause:** The `--preserve`, `--guard`, or `--push-guard` flag received an unrecognized value. (`pk guard --push-guard` emits the same error when run directly.)

**Fix:** Use `auto`, `manual`, or `off` for `--preserve`; `block`, `ask`, or `off` for `--guard` and `--push-guard`.

### flag dependencies

```
Error: --at requires --baseline
Error: --push requires --baseline
```

**Cause:** `--at` and `--push` only apply to the baseline tag workflow.

**Fix:** Add `--baseline` to the command.

### invalid --at ref

```
Error: invalid --at ref "--force"; refs cannot start with -
```

**Cause:** `--at` received a value beginning with `-`. Git refs can never start with `-`, and passing one to git would be read as an option rather than a ref.

**Fix:** Pass a real ref (branch, tag, or commit SHA) to `--at`.

### not a git repository

```
Warning: this is not a git repository. Proceeding because --allow-non-git was set.
Some commands (changelog, release) will not work until git is initialized.
```

**Cause:** `pk setup` was run outside a git repository with `--allow-non-git`.

**Fix:** Run `git init` when ready. Rules and `pk protect` work without git; other commands do not.

## pk init

Every `pk init` refusal happens in pre-flight, before anything is written, so the repository is unchanged when one of these appears.

### not a git repository

```
Error: this is not a git repository. Run git init first
```

**Cause:** `pk init` shapes an existing repository; it does not create one.

**Fix:** Run `git init`, make a first commit, then re-run. To create the GitHub repo too, use `gh repo create --clone` first.

### no commits

```
Error: this repository has no commits. Make one first, so v0.0.0 has something to anchor to
```

**Cause:** `pk init` tags `v0.0.0` on the current commit, and there isn't one.

**Fix:** Make the project's first commit, then re-run. `git commit --allow-empty -m "chore: init"` works when there is nothing to commit yet.

### working tree not clean

```
Error: working tree is not clean; commit or stash changes first
```

**Cause:** `pk init` commits the files it writes, so it needs to start from a clean tree to avoid sweeping unrelated changes into `chore: pk setup`.

**Fix:** Commit or stash your changes first.

### on the wrong branch

```
Error: you are on "develop" but the release branch is "main"; switch to it first
```

**Cause:** `pk init` anchors `v0.0.0` and the topology on the release branch, so it has to be the one checked out.

**Fix:** `git switch main`, then re-run. Or drop `--release` to use the branch you are on.

### detached HEAD

```
Error: HEAD is detached; check out your release branch, or name it with --release
```

**Cause:** With no branch checked out there is no default release branch to infer.

**Fix:** Check out the release branch, or pass `--release <name>`.

### source branch equals release branch

```
Error: source branch "main" is the release branch; they must differ
```

**Cause:** The working branch and the release branch must be distinct: `pk release` fast-forward merges one into the other.

**Fix:** Pass a different `--source`, or drop the flag to use the `develop` default.

### push without an origin

```
Error: --push needs an origin remote, and this repository has none
```

**Cause:** `--push` publishes the release branch, the tag, and the working branch, and there is nowhere to publish them.

**Fix:** Add the remote (`git remote add origin <url>`), or drop `--push` and run `pk init` local-only.

## pk rules

### flag dependencies

```
Error: --strict requires --lint
```

**Cause:** `--strict` only adds house-style checks to the `--lint` safety scan; it does nothing on its own.

**Fix:** Run `pk rules --lint --strict`.

### lint findings

```
Found 2 issue(s):
  .claude/rules/example.md: 12:3 hidden/format U+200B [safety]
  .claude/rules/example.md: line 40: em dash (U+2014) [style]
```

**Cause:** `pk rules --lint` found hidden/Trojan-source characters (always), or, under `--strict`, house-style violations. The command exits non-zero so scripts and CI can gate on it.

**Fix:** Remove the flagged characters. `[safety]` findings are genuine risks; `[style]` findings reflect plankit's house style and only appear with `--strict`.

## pk guard

### out of memory at startup

```
runtime: out of memory
...
runtime.mallocgc(0x100, ...)
    .../src/runtime/malloc.go:1150
...
runtime.schedinit()
    .../src/runtime/proc.go:878
runtime.rt0_go()
```

**Cause:** The machine ran out of memory (on Windows, commit charge: RAM plus pagefile) at the moment Claude Code spawned the hook binary. Every frame in the trace is Go runtime source (`runtime.schedinit`, `internal/cpu.doinit`). The process died during Go runtime bootstrap, before any pk code ran. Any Go binary would crash identically under the same pressure. `pk guard` runs on every Bash call, so it is usually the process that hits the limit first. This is memory pressure on the machine, not a pk leak.

**Fix:** Nothing is left unguarded. Go fatal errors exit with status 2, which Claude Code treats as a blocking error for PreToolUse hooks. The command was blocked, and retrying it succeeds once memory frees. If it recurs, find what is consuming memory. On Windows, Event Viewer > System log > Event ID 2004 names the largest consumers. Also check the pagefile is not disabled or capped small.

## pk pin

### invalid semver

```
Error: "abc" is not valid semver
```

**Cause:** The version argument does not parse as valid semantic versioning.

**Fix:** Use a valid semver string (e.g., `1.0.0`, `0.8.1-beta.1`).

## pk preserve

### no plan found (dry-run)

```
pk preserve --dry-run: no plan found (tool_response did not contain a .claude/plans/*.md path)
pk preserve --dry-run: no plan found (plan file not found: /path/to/.claude/plans/my-plan.md)
pk preserve --dry-run: no plan found (stdin had no valid hook payload and no pending-plan pointer was found)
```

**Cause:** `--dry-run` found no plan to preview. The diagnostic in parentheses explains why:

- The `tool_response` didn't contain a path matching `.claude/plans/*.md`.
- The matched path doesn't exist on disk.
- Stdin had no valid JSON payload, and no pending-plan pointer was available.

**Fix:** Ensure the `tool_response` contains an absolute path with `.claude/plans/` in it (e.g., `/Users/you/.claude/plans/my-plan.md`). Paths using `~` or outside `.claude/plans/` are not recognized.

### failed to read plan

```
pk preserve: failed to read plan: open /path/to/plan.md: no such file or directory
```

**Cause:** The plan path was extracted from `tool_response` and passed the existence check, but reading the file failed (permissions, race condition).

**Fix:** Verify the plan file exists and is readable.

### not a git repository

```
pk preserve: not a git repository: /path/to/project
```

**Cause:** The resolved project directory is not inside a git working tree. `pk preserve` needs git to commit the preserved plan.

**Fix:** Run `git init` in the project directory, or set `CLAUDE_PROJECT_DIR` to a directory inside a git repository.

## pk version

### pinned version mismatch (binary behind)

```
Note: .claude/install-pk.sh pins v0.19.2 but you're running 0.19.1
  To update: go install github.com/markwharton/plankit/cmd/pk@latest
```

**Cause:** A newer version was released and the bootstrap script was updated, but the local binary hasn't been reinstalled yet.

**Fix:** Run `go install github.com/markwharton/plankit/cmd/pk@latest` to update the binary.

### pinned version mismatch (script behind)

```
Note: .claude/install-pk.sh pins v0.18.0 but you're running 0.19.0
  To refresh it: pk setup
```

**Cause:** The local binary is newer than the version pinned in the bootstrap script. Cloud sandboxes will install the pinned version, not the version running locally.

**Fix:** Run `pk setup` to update the pin to the current version.

## .pk.json

### malformed JSON

```
Error: failed to parse .pk.json: ...
```

**Cause:** `.pk.json` contains invalid JSON syntax.

**Fix:** Check for missing commas, unmatched brackets, or trailing commas (not allowed in JSON).

### read error

```
Error: failed to read .pk.json: ...
```

**Cause:** The file exists but could not be read (permissions, disk error).

**Fix:** Check file permissions: `ls -la .pk.json`.
