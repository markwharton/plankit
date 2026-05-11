# Error Reference

Common errors from pk commands, what causes them, and how to recover.

## pk changelog

### no version tags found

```
Error: no version tags found
  To anchor at v0.0.0: pk setup --baseline [--at <ref>] --push
  Or tag a specific version manually (e.g., git tag v0.0.0 && git push origin v0.0.0)
```

**Cause:** `pk changelog` scans commits since the most recent semver tag. Without a tag, there is no starting point.

**Fix:** Run `pk setup --baseline` to create a `v0.0.0` anchor tag. Add `--push` to publish it. Use `--at $(git rev-list --max-parents=0 HEAD)` to include all prior commits in the first changelog entry.

### no version tags found locally

```
Error: no version tags found locally
  Origin has tags — fetch them: git fetch --tags
```

**Cause:** The remote has tags but they are not present locally. Common in shallow-clone cloud sandboxes.

**Fix:** Run `git fetch --tags`. The `install-pk.sh` bootstrap script does this automatically in cloud sandboxes.

### protected branch

```
Error: you're on "main" which is a protected branch — switch to your development branch first
```

**Cause:** `pk changelog` refuses to create release commits on branches listed in `guard.branches`.

**Fix:** Switch to your development branch (`git switch develop`).

### branch not on origin

```
Error: develop does not exist on origin — push it first:
  git push -u origin develop
```

**Cause:** `pk changelog` checks that the current branch exists on the remote before committing. Without this, `pk changelog` succeeds but `pk release` fails, leaving a Release-Tag commit that requires a manual push to continue.

**Fix:** Push the branch: `git push -u origin develop`.

### working tree not clean

```
Error: working tree is not clean — commit or stash changes first
```

**Cause:** `pk changelog` and `pk release` require a clean working tree before proceeding.

**Fix:** Commit or stash your changes first.

### HEAD already pushed (--undo)

```
Error: HEAD is already on the remote — cannot undo a pushed commit
```

**Cause:** `pk changelog --undo` only rewinds unpushed commits to avoid rewriting shared history.

**Fix:** If the changelog commit has been pushed, create a new commit to correct it. Do not force push.

## pk release

### no Release-Tag trailer

```
Error: no Release-Tag trailer on HEAD — run 'pk changelog' first
```

**Cause:** `pk release` reads the version from a git trailer on HEAD that `pk changelog` writes.

**Fix:** Run `pk changelog` first, then `pk release`. Or use `/ship` which chains them.

### on the release branch

```
Error: you're on the release branch "main" — switch to your working branch first
```

**Cause:** `pk release` merges from the source branch into the release branch. Running it directly on the release branch would skip the merge.

**Fix:** Switch to your working branch (`git switch develop`).

### tag already exists

```
Error: tag v0.8.1 already exists locally — nothing to release
```

**Cause:** The tag from the `Release-Tag` trailer already exists. The release was already completed or partially completed.

**Fix:** If the release was already pushed, there is nothing to do. If the tag is leftover from a failed attempt, delete it (`git tag -d v0.8.1`) and retry.

### branch not on origin

```
Error: develop does not exist on origin — push it first:
  git push -u origin develop
```

**Cause:** `pk release` verifies the source branch exists on the remote before proceeding.

**Fix:** Push the branch: `git push -u origin develop`.

### behind remote

```
Error: local develop is behind origin/develop — pull first
```

**Cause:** Someone pushed commits to the branch since your last pull.

**Fix:** Pull the latest changes: `git pull origin develop`.

### not fast-forward

```
Error: merge failed — main has diverged from develop (not fast-forward).
Resolve on main manually, then try again.
```

**Cause:** The release branch has commits that are not on the source branch. `pk release` only does fast-forward merges to avoid merge conflicts.

**Fix:** Merge main into your source branch first to reconcile the histories, then retry.

### push failed

```
Error: git push failed: ...
```

**Cause:** The push was rejected by the remote (permissions, branch protection rules, or network issues).

**Fix:** `pk release` automatically cleans up the local tag on push failure. Fix the underlying issue (permissions, network) and run `pk release` again.

## pk setup

### invalid mode

```
Error: invalid --preserve mode "xyz" (must be auto or manual)
Error: invalid --guard mode "xyz" (must be block or ask)
```

**Cause:** The `--preserve` or `--guard` flag received an unrecognized value.

**Fix:** Use `auto` or `manual` for `--preserve`, `block` or `ask` for `--guard`.

### flag dependencies

```
Error: --at requires --baseline
Error: --push requires --baseline
```

**Cause:** `--at` and `--push` only apply to the baseline tag workflow.

**Fix:** Add `--baseline` to the command.

### not a git repository

```
Warning: this is not a git repository. Proceeding because --allow-non-git was set.
Some commands (changelog, release) will not work until git is initialized.
```

**Cause:** `pk setup` was run outside a git repository with `--allow-non-git`.

**Fix:** Run `git init` when ready. Rules and `pk protect` work without git; other commands do not.

## pk pin

### invalid semver

```
Error: "abc" is not valid semver
```

**Cause:** The version argument does not parse as valid semantic versioning.

**Fix:** Use a valid semver string (e.g., `1.0.0`, `0.8.1-beta.1`).

## pk version

### pinned version mismatch (binary behind)

```
Note: .claude/install-pk.sh pins v0.19.2 but you're running 0.19.1 — run 'go install github.com/markwharton/plankit/cmd/pk@latest' to update
```

**Cause:** A newer version was released and the bootstrap script was updated, but the local binary hasn't been reinstalled yet.

**Fix:** Run `go install github.com/markwharton/plankit/cmd/pk@latest` to update the binary.

### pinned version mismatch (script behind)

```
Note: .claude/install-pk.sh pins v0.18.0 but you're running 0.19.0 — re-run 'pk setup' to update
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
