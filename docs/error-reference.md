# Error reference

Every message pk prints on stderr with an `Error:`, `Warning:` or `Note:` prefix, or with a hook's `pk <cmd>:` prefix, quoted as the source prints it, with its cause and the recovery. Hints (`To ...:` lines) are exact commands.

## Any command

### not a git repository

```
Error: not a git repository
```

**Cause:** `pk changelog`, `pk release`, `pk status` and `pk init` need a git working tree; none was found walking up from the directory.

**Fix:** `git init`, or run from inside a repository.

### working tree not clean

```
Error: working tree is not clean; commit or stash changes first
```

**Cause:** `pk changelog`, `pk release` and `pk init` commit or merge, and refuse to sweep uncommitted changes into it. `pk changelog --dry-run` does not check.

**Fix:** commit or stash.

### branch not on origin

```
Error: develop does not exist on origin
  To push it: git push -u origin develop
```

**Cause:** `pk changelog` and `pk release` publish from the current branch; a branch origin lacks cannot be released.

**Fix:** the hint.

### not the default branch

```
Error: you're on "feature" but the default branch on origin is "main"; trunk flow releases from the default branch
  To release this work from main: git switch main && git merge feature
  Then: pk changelog && pk release
```

**Cause:** in trunk flow (no `release.branch`), `pk changelog` and `pk release` tag and push the branch they run on, so only origin's default branch may release. `pk release` prints `Then: pk release`. The check is skipped when origin advertises no HEAD.

**Fix:** the hint. To release from another branch permanently, change the default branch on the host, or set `release.branch` for merge flow. A default branch listed in `guard.branches` cannot release in trunk flow; set `release.branch`.

### unknown command

```
Error: unknown command "foo"
```

**Fix:** `pk --help` lists the commands.

### malformed .pk.json

```
Error: failed to parse .pk.json: ...
Error: failed to read .pk.json: ...
```

**Cause:** the file is not valid JSON, or cannot be read. A hook prints the same with its `pk guard:` prefix.

**Fix:** correct the JSON; check the file's permissions.

## pk changelog

### no version tags found

```
Error: no version tags found
  To anchor at v0.0.0: pk setup --baseline [--at <ref>] --push
  or: git tag v0.0.0 && git push origin v0.0.0
```

**Cause:** the section is built from the commits since the latest semver tag, and there is none.

**Fix:** the hint; `--at` cases in [pk setup](pk-setup.md#baseline).

### no version tags found locally

```
Error: no version tags found locally
  Origin has tags; fetch them: git fetch --tags
```

**Cause:** a clone without tags, as in a sandbox that cloned one branch. `.claude/install-pk.sh` runs the fetch at session start.

**Fix:** the hint.

### protected branch

```
Error: you're on "main" which is a protected branch; switch to your development branch first
  To start one: git switch -c develop && git push -u origin develop
```

**Cause:** the branch is in `guard.branches`. The `To start one` hint appears only when no other local branch exists.

**Fix:** switch to the working branch; to create one, [Changing flow](changing-flow.md).

### HEAD already pushed

```
Error: HEAD is already on the remote; cannot undo a pushed commit
```

**Cause:** `--undo` rewrites history, and refuses once the commit is on origin.

**Fix:** correct the changelog in a new commit. Never force-push.

### changelog already pending

```
Error: changelog for v0.19.9 is already pending (HEAD has Release-Tag: v0.19.9)
  To complete the release: pk release
  To undo and start over:  pk changelog --undo
```

**Cause:** HEAD is already a release commit.

**Fix:** either hint.

### invalid --bump

```
Error: invalid --bump value "big" (must be major, minor, or patch)
```

### unsupported versionFile type

```
Error: unsupported versionFile type "toml" for Cargo.toml (only "json" is supported)
```

**Cause:** `changelog.versionFiles` entries are JSON only.

**Fix:** pin the file from `preCommit` with [pk pin](pk-pin.md).

### hook failed

```
Error: postVersion hook failed: exit status 1
Error: preCommit hook failed: exit status 1
```

**Cause:** the hook command exited non-zero; its own output precedes the line.

**Fix:** nothing was committed. Fix what the hook reported and run `pk changelog` again. Version files the run already rewrote are left modified; `git checkout -- <file>` restores them, or the re-run rewrites them.

### exclude did not match

```
Warning: --exclude abc1234 did not match any commit
```

**Cause:** the value is not the short hash as printed in the section's parentheses. The run continues.

## pk release

### no Release-Tag trailer

```
Error: no Release-Tag trailer on HEAD; run pk changelog first
```

**Fix:** `pk changelog`, or `/ship`, which runs both.

### invalid trailer

```
Error: Release-Tag trailer value is not valid semver: "v1.2"
```

**Cause:** the trailer value does not parse as strict semver and round-trip to the same string.

**Fix:** `pk changelog --undo`, then `pk changelog`.

### on the release branch

```
Error: you're on the release branch "main"; switch to your working branch first
  To start one: git switch -c develop && git push -u origin develop
  Then: pk changelog && pk release
```

**Cause:** `pk release` merges from the working branch into the release branch; on the release branch there is nothing to merge. The `To start one` hint appears only when no other local branch exists.

**Fix:** switch to the working branch; to create one, [Changing flow](changing-flow.md).

### invalid release branch

```
Error: invalid release.branch "--output=/tmp/x" in .pk.json; branch names cannot start with -
```

**Fix:** correct `release.branch`.

### release branch missing

```
Error: release branch main does not exist locally or on origin
  To create it: git branch main && git push -u origin main
```

**Fix:** the hint, or correct `release.branch`.

### tag already exists

```
Error: tag v0.8.1 already exists locally; nothing to release
```

**Cause:** the release already ran, or a previous attempt left the tag.

**Fix:** if origin has the tag, nothing; otherwise `git tag -d v0.8.1` and run again.

### behind remote

```
Error: local develop is behind origin/develop; pull first
```

**Fix:** `git pull origin develop`.

### fetch failed

```
Warning: failed to fetch main from origin: ... (continuing with local state)
```

**Cause:** the pre-flight fetch of the release branch failed; the run continues with the last-fetched state, and the push is what will fail if origin moved.

### not fast-forward

```
Error: merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.
```

`--dry-run` reports the same condition before merging:

```
Error: merge would not be fast-forward; main has diverged from develop. Resolve on main manually, then try again.
```

**Cause:** the release branch has a commit the working branch lacks. `pk release` merges `--ff-only`.

**Fix:** in this order, because `pk release` reads the trailer from HEAD only:

1. `pk changelog --undo`, if an unpushed release commit is at HEAD.
2. `git merge origin/main` on the working branch. The stray commit is kept; never force-push.
3. `pk changelog`.
4. `pk release`.

### release branch diverged from origin

```
Error: origin/main has diverged from develop; the release push would be rejected
  To reconcile, on develop: git merge origin/main
```

**Cause:** `origin/main` has a commit the working branch lacks; caught by pre-flight, before the local merge. A release branch created on the host with its own initial commit produces it.

**Fix:** the steps under [not fast-forward](#not-fast-forward).

### push failed

```
Error: git push failed: ...
```

**Cause:** origin rejected the push: permissions, a ruleset, the network. The push is `--atomic`, so origin has neither the branch nor the tag; the local tag is deleted.

**Fix:** resolve what git reported; run `pk release` again.

### pre-release hook failed

```
Error: pre-release hook failed: exit status 1
```

**Cause:** `release.hooks.preRelease` exited non-zero, after the merge and before the tag; its output precedes the line.

**Fix:** nothing was tagged or pushed. Fix what the hook reported; `pk release --dry-run` reruns the hook without releasing.

### pre-push hook failed

```
Error: pre-push hook failed: exit status 1
```

**Cause:** `release.hooks.prePush` exited non-zero, after the tag and before the push.

**Fix:** the local tag is deleted and, in merge flow, the merge is rolled back; origin is untouched. Fix the hook; run `pk release` again. `--dry-run` does not run `prePush`.

## pk setup

### invalid mode

```
Error: invalid --preserve mode "xyz" (must be auto, manual, or off)
Error: invalid --guard mode "xyz" (must be block, ask, or off)
Error: invalid --push-guard mode "xyz" (must be block, ask, or off)
```

`pk guard --push-guard` prints the last one too.

### flag dependencies

```
Error: --at requires --baseline
Error: --push requires --baseline
Error: --baseline requires a git repository
```

### invalid --at ref

```
Error: invalid --at ref "--force"; refs cannot start with -
```

### not a git repository

```
Error: this is not a git repository. pk requires git for most commands.

Run `git init` first, or pass --allow-non-git to proceed anyway
```

With `--allow-non-git`:

```
Warning: this is not a git repository. Proceeding because --allow-non-git was set. Some commands (changelog, release) will not work until git is initialized.
```

### pk not on PATH

```
Warning: pk is not in your PATH. Hooks will silently skip until it is installed.
```

**Cause:** a hook line runs `pk`; a shell that cannot find it exits 127, which Claude Code treats as non-blocking.

**Fix:** put `pk` on PATH (see the README).

## pk init

Every refusal is pre-flight: nothing has been written.

### not a git repository

```
Error: this is not a git repository. Run git init first
```

### no commits

```
Error: this repository has no commits. Make one first, so v0.0.0 has something to anchor to
```

**Fix:** `git commit --allow-empty -m "chore: init"` when there is nothing to commit yet.

### on the wrong branch

```
Error: you are on "develop" but the release branch is "main"; switch to it first
```

**Cause:** the first run tags and commits on the release branch. A shaped repository runs from any branch.

**Fix:** `git switch main`, or drop `--release`.

### working branch exists but the repository is not shaped

```
Error: branch "develop" already exists but the repository is not plankit-shaped (no version tag); pk init shapes a fresh repository, so shape this one by hand
```

The parenthesis is one of `no version tag`, `no release.branch in .pk.json`, or `release.branch in .pk.json is "main", not "trunk"`.

**Cause:** an existing working branch marks a shaped repository; with the tag or topology missing, this is an established project, and shaping it would commit on the release branch where the working branch cannot reach.

**Fix:** [Adoption](adoption.md).

### detached HEAD

```
Error: HEAD is detached; check out your release branch, or name it with --release
```

### source branch equals release branch

```
Error: source branch "main" is the release branch; they must differ
```

**Fix:** another `--source`, or drop the flag for `develop`.

### push without an origin

```
Error: --push needs an origin remote, and this repository has none
```

**Fix:** `git remote add origin <url>`, or drop `--push`.

## pk rules

### flag dependencies

```
Error: --strict requires --lint
```

### lint findings

```
Found 2 issue(s):
  .claude/rules/example.md: 12:3 hidden/format U+200B [safety]
  .claude/rules/example.md: line 40: em dash (U+2014) [style]
```

**Cause:** `[safety]`: a control or Unicode format character, bare CR or invalid UTF-8. `[style]`: a house-style finding, `--strict` only. Exit 1.

**Fix:** remove the character; see [The em-dash check](design.md#the-em-dash-check) for the style rule.

## pk guard, pk protect, pk preserve

### failed to read input

```
pk guard: failed to read input: ...
pk protect: failed to read input: ...
```

**Cause:** stdin was not a hook payload. The command exits 0 and allows.

### write error

```
pk guard: write error: ...
pk protect: write error: ...
```

**Cause:** the decision could not be written to stdout. Exit 0.

### out of memory at startup

```
runtime: out of memory
...
runtime.schedinit()
    .../src/runtime/proc.go:878
```

**Cause:** the machine (on Windows, RAM plus pagefile) was out of memory when Claude Code spawned the hook; every frame is Go runtime, before any pk code ran. `pk guard` runs on every Bash call, so it is the process that meets the limit first.

**Fix:** the exit is 2, which Claude Code treats as a block, so the command did not run; retry once memory frees. On Windows, Event Viewer, System log, Event ID 2004 names the consumers; check the pagefile is not disabled.

### pk preserve: no plan found

```
pk preserve --dry-run: no plan found (tool_response did not contain a .claude/plans/*.md path)
pk preserve --dry-run: no plan found (plan file not found: /path/to/.claude/plans/my-plan.md)
pk preserve --dry-run: no plan found (stdin had no valid hook payload and no pending-plan pointer was found)
```

**Cause:** in order: the payload's path is not under `.claude/plans/`; the file is missing; there was neither a payload nor `.git/pk-pending-plan`. A leading `~/` is expanded.

### pk preserve: write failures

```
pk preserve: could not determine project directory
pk preserve: not a git repository: /path/to/project
pk preserve: failed to read plan: open /path/to/plan.md: no such file or directory
pk preserve: failed to create directory: ...
pk preserve: failed to write plan: ...
pk preserve: git add failed: ...
pk preserve: git commit failed: ...
pk preserve: failed to write pending-plan pointer: ...
```

**Cause:** each names the step. The project directory is `CLAUDE_PROJECT_DIR`, then the payload's `cwd`; it must be inside a git working tree.

**Fix:** resolve the named step; `/preserve` runs the preservation again.

## pk pin

### invalid semver

```
Error: "abc" is not valid semver
```

### no matching pin

```
Warning: SKILL.md has no pin for "version"
Warning: install-pk.sh has no VERSION pin
```

**Cause:** no line matches the rules in [pk pin](pk-pin.md). Exit 0, so a hook proceeds; the version was not written. A missing file exits 0 without the warning.

## pk teardown

### malformed hooks

```
Warning: skipping hooks.PreToolUse in .claude/settings.json; malformed JSON: ...
```

**Cause:** the hook category could not be parsed; it is left as it is and the rest of the teardown proceeds.

## pk version

### pinned version mismatch

```
Note: .claude/install-pk.sh pins v0.19.2 but you're running 0.19.1
  To update: go install github.com/markwharton/plankit/cmd/pk@latest
```

```
Note: .claude/install-pk.sh pins v0.18.0 but you're running 0.19.0
  To refresh it: pk setup
```

**Cause:** the first: the pin is newer than the binary. The second: the binary is newer than the pin, and a sandbox would install the pinned version.

**Fix:** the hint.
