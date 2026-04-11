# Plan: Trailer-driven release flow and `pk changelog --undo`

## Context

The current `pk changelog` creates a git tag as part of its commit step, then `pk release` pushes that tag alongside the merge. Two problems fall out of this:

1. **Tag timing.** The tag is the release trigger (`release.yml` fires on `push: tags: v*`), but it's created before the "go" decision is made. If anything goes wrong between `pk changelog` and `pk release` — rejected review, bad bump detection, changed mind — the local tag is an orphan with no tooling to clean it up. `pk changelog --push` makes this worse by pushing the tag before the reviewer sees anything.
2. **Source of pending version.** The git tag is canonical once it exists, but nothing carries the pending version cleanly from `pk changelog` to `pk release`. The existing code reads "is there a tag at HEAD" as a proxy, which couples the two commands through a side-effect rather than a contract.

Project is pre-1.0 and nothing has been released officially, so this plan drops all backwards-compatibility and breaking-change concerns.

## Design

The fix is a **handshake via a `Release-Tag:` commit trailer**, not a local tag. `pk changelog` writes the intent into the commit body using `git commit --trailer`; `pk release` reads the intent via `git log --format=%(trailers:...)` and turns it into the actual git tag at the right moment.

**Two distinct artifacts:**

| Thing | What it is | Created by | Consumed by |
|---|---|---|---|
| `Release-Tag:` trailer | text in commit message body | `pk changelog` via `git commit --trailer "Release-Tag: v0.6.2"` | `pk release` and `pk changelog --undo` via `git log --format=%(trailers:key=Release-Tag,valueonly)` |
| `v0.6.2` git tag | ref in `refs/tags/` | `pk release` via `git tag v0.6.2` | `release.yml` on tag push; humans running `git tag --list` |

The trailer is intent; the tag is fact. The trailer disappears automatically if the commit is rewound (because it lives inside the commit body). The tag is only created when `pk release` commits to publishing, seconds before the push.

**Why a trailer and not a magic commit subject:** git has first-class support for trailers (`git interpret-trailers`, `git commit --trailer`, `git log --format=%(trailers:...)`) with defined semantics — key/value format, whitespace trimming, deduplication. Using a trailer means speaking git's native language for commit metadata, not inventing a parser. It also means a developer can't accidentally trigger the release flow by typing a similar commit subject.

**Validation:** trailer value is passed to `version.ParseSemver`, then round-tripped (`parsed.String() == value`) to ensure exact match with no trailing characters. No regex anywhere, per project convention against regex for structured data.

## Out of scope

- **`pk release --pr`** — parked. The PR flow has its own open questions (push-branch-now-tag-later, post-merge tag step, guard interaction) to be revisited after independent testing.
- **`pk release` "refuse on release branch" logic** — untouched.
- **`pk guard`** — untouched.
- **Anti-patterns doc squash-merge section** — stays relevant because it still applies under `--pr`. Minor clarification only.

## Implementation

Four commits, each leaving the tree with passing tests and buildable binaries.

### Commit 1: Move tag creation via `Release-Tag` trailer

Atomic handshake — both sides must ship together or the flow is broken.

**`internal/changelog/changelog.go`** (approx line 300-312, the commit step):

- Replace the current `git commit -m "chore: release <tag>"` with:
  ```go
  cfg.GitExec("", "commit", "-m", fmt.Sprintf("chore: release %s", nextTag), "--trailer", fmt.Sprintf("Release-Tag: %s", nextTag))
  ```
- Delete the subsequent `git tag <tag>` call
- Remove the `Push` field from `Config` if still present (commit 3 cleans the flag plumbing)

**`internal/release/release.go`** (approx line 100-104):

- Replace `git tag --points-at HEAD` reading with:
  ```go
  cfg.GitExec("", "log", "-1", "--format=%(trailers:key=Release-Tag,valueonly)", "HEAD")
  ```
- Run the returned value through `version.ParseSemver` with a round-trip equality check:
  ```go
  parsed, ok := version.ParseSemver(value)
  if !ok || parsed.String() != value {
      return errNoReleaseTag
  }
  ```
- If missing or round-trip fails → error: `"no Release-Tag trailer on HEAD — run pk changelog first"`
- `findVersionTag` helper (line ~352) can be deleted or repurposed
- Create the tag (`git tag <version>`) after pre-flight checks pass and before any merge/push. The tag lives on the source branch at that moment; the fast-forward merge carries it naturally into the release branch.
- On failure after `git tag` but before `git push`, delete the local tag as cleanup (`git tag -d <version>`) — wire this into the existing deferred error recovery paths at release.go:192-198

**`internal/version/` verification:**

Before implementing, read `ParseSemver` to confirm behavior on `v0.6.2-rc1`, `v0.6.2 trailing`, and empty string. The round-trip check is load-bearing if the parser is lenient. If it panics on bad input, add a guard.

**`internal/changelog/changelog_test.go`:**

- Remove assertions that verify a tag was created
- Add assertions that the commit body (`git log -1 --format=%B`) contains `Release-Tag: v0.6.2` as a string
- Delete `TestRun_push`, `TestRun_pushFailure`, `TestRun_pushDryRun`

**`internal/release/release_test.go`:**

- Replace "tag at HEAD" fixtures with "`Release-Tag:` trailer at HEAD" — the mock `GitExec` returns the trailer value for the `log --format=%(trailers:...)` call
- Add `TestRun_missingTrailer` (empty value → exit 1 with clear message)
- Add `TestRun_invalidTrailerValue` (non-semver value → exit 1)
- Add `TestRun_tagCleanupOnPushFailure` (push fails → local tag is deleted)

**Commit message:** `refactor: move tag creation from pk changelog to pk release via Release-Tag trailer`

### Commit 2: Add `pk changelog --undo`

**`internal/changelog/changelog.go`:**

Add `Undo(cfg Config) int`:

1. Read `Release-Tag:` trailer from HEAD. Missing → error: `"HEAD is not a pk changelog commit"`.
2. Validate value with `ParseSemver` round-trip. Invalid → same error.
3. Check working tree clean via `git status --porcelain`. Dirty → error: `"working tree is not clean — commit or stash changes first"`.
4. Check HEAD is unpushed:
   - Try `git rev-parse --abbrev-ref HEAD@{upstream}` to get upstream ref
   - If no upstream configured → allow (branch has never been pushed, definitely safe)
   - If upstream exists → check `git log @{u}..HEAD --oneline` is non-empty (HEAD is ahead of upstream)
   - If HEAD is at or behind upstream → error: `"HEAD is already on the remote — cannot undo a pushed commit"`
5. `git reset --hard HEAD~1` — discards the commit, CHANGELOG.md changes, version file changes atomically
6. Report: `"Undid release commit <version>; CHANGELOG.md and version files restored"`

**`cmd/pk/main.go:runChangelog`:**

- Add `--undo` flag
- When set, dispatch to `changelog.Undo(cfg)` and skip the normal flow
- Update usage string

**`internal/changelog/changelog_test.go`:**

Add:
- `TestUndo_happyPath` — clean tree, HEAD has trailer, unpushed → reset succeeds
- `TestUndo_dirtyTree` — uncommitted changes → refuses
- `TestUndo_noTrailer` — HEAD has no `Release-Tag:` trailer → refuses
- `TestUndo_alreadyPushed` — HEAD is at/behind upstream → refuses
- `TestUndo_noUpstream` — branch has no upstream → allowed
- `TestUndo_invalidTrailerValue` — trailer present but value fails semver round-trip → refuses

**Commit message:** `feat(changelog): add --undo to unwind an unpushed release commit`

### Commit 3: Remove `pk changelog --push`

**`cmd/pk/main.go:runChangelog`:**

- Delete `--push` flag registration and its plumbing
- Update usage string

**`internal/changelog/changelog.go`:**

- Delete `Push` field from `Config` if commit 1 didn't already remove it
- Delete any remaining `if cfg.Push { ... }` block

**Commit message:** `refactor(changelog): drop --push flag`

### Commit 4: Documentation

All docs, skills, help strings updated. No code changes.

**`docs/pk-changelog.md`:**

- Usage block: remove `--push` line, add `--undo` line
- "How it works": step 10 changes from "Tags the new version" to "Adds `Release-Tag:` trailer to the commit message body via `git commit --trailer`". Delete step 11 (old `--push` behavior).
- Flags: remove `--push` entry, add `--undo` entry describing trailer-gated / HEAD-only / unpushed-only guarantees
- New "Details: Release-Tag trailer" subsection — format, why it exists, who reads it
- Remove any claim that `pk changelog` creates a git tag
- Add "Requirements" note: git 2.32+ (for `git commit --trailer`)

**`docs/pk-release.md`:**

- How it works: at the top of both multi-branch and legacy flows, insert step "Read `Release-Tag:` trailer from HEAD"
- The "Find version tag at HEAD" step → "Read `Release-Tag:` trailer from HEAD and create the local tag"
- New "Details: Release-Tag trailer" subsection cross-referencing pk-changelog.md
- Workflows table unchanged; `--pr` row unchanged
- Add "Requirements" note: git 2.32+

**`docs/resources.md`:**

- Remove the "After a GitHub-managed merge (or on a single-branch repo), cut a release in one command: `pk changelog --push`" block entirely — the misleading third flow disappears
- Direct merge example: update comment on `pk changelog` line from "generate changelog, commit, and tag version" to "generate changelog and commit (tag is created by pk release)"
- PR flow example: keep unchanged (parked)

**`README.md`** (lines 53-54):

- `pk changelog` table row: "Generate changelog, commit, and tag" → "Generate CHANGELOG.md and commit (tag created by `pk release`)"
- `pk release` table row: "Merge to release branch, validate, and push" → "Create tag, merge to release branch, validate, push"

**`CLAUDE.md` (project):**

- "Quick Commands" section: `pk changelog` description drops "and tag"; `pk release` description mentions tagging

**`CONTRIBUTING.md`** (lines 38-41):

- Update comment on `pk changelog` line to remove "and tag version"
- Update `pk release` comment to mention tag creation

**`docs/anti-patterns.md`** ("Squash Merge and Release Tags", line 68):

- Update wording to clarify the scenario applies when using `pk release --pr` or similar PR-based flows, not the direct merge flow. The direct merge flow under the new design creates the tag on the commit that's about to be pushed, so it's immune.

**`.claude/skills/changelog/SKILL.md` + `internal/setup/skills/changelog/SKILL.md`:**

- Frontmatter description: drop "and tag version"
- Body: "Follow with `/release` to merge and push." → "Follow with `/release` to tag, merge, and push."
- Recompute body hash per CLAUDE.md:
  ```bash
  sed -n '/^---$/,/^---$/!p' internal/setup/skills/changelog/SKILL.md | shasum -a 256
  ```
- Update `pk_sha256` in `.claude/skills/changelog/SKILL.md`

**`.claude/skills/release/SKILL.md` + `internal/setup/skills/release/SKILL.md`:**

- Body: "Push a release created by pk changelog." → "Tag and push a release created by pk changelog."
- Recompute body hash, update `pk_sha256`

**Commit message:** `docs: update command docs, skills, README for trailer-driven release flow`

## Critical files to modify

| File | Change |
|---|---|
| `internal/changelog/changelog.go` | stop tagging, write trailer, add `Undo` |
| `internal/changelog/changelog_test.go` | trailer assertions, `Undo` tests, drop `--push` tests |
| `internal/release/release.go` | read trailer, create tag, cleanup on failure |
| `internal/release/release_test.go` | trailer fixtures, cleanup test |
| `cmd/pk/main.go` | `--undo` flag, drop `--push` flag, usage strings |
| `internal/version/` (read-only verify) | confirm `ParseSemver` strictness |
| `docs/pk-changelog.md` | flags, how-it-works, trailer details |
| `docs/pk-release.md` | how-it-works, trailer details |
| `docs/resources.md` | drop third flow, update comment |
| `README.md` | table rows 53-54 |
| `CLAUDE.md` | Quick Commands section |
| `CONTRIBUTING.md` | lines 38-41 comments |
| `docs/anti-patterns.md` | scope the squash section to `--pr` |
| `.claude/skills/changelog/SKILL.md` + embedded copy | description, body, sha |
| `.claude/skills/release/SKILL.md` + embedded copy | body, sha |

## Existing functions/utilities to reuse

- **`version.ParseSemver`** in `internal/version/` — canonical semver parsing. Used for round-trip validation of the trailer value. Already used in `changelog.go:201` and elsewhere.
- **`pkgit.Exec`** in `internal/git/` — the shared `GitExec` implementation (already wired into both `changelog.Config` and `release.Config`).
- **`git commit --trailer`** — native git support for writing trailers, available in git 2.32+ (June 2021).
- **`git log --format=%(trailers:key=...,valueonly)`** — native git support for reading trailer values, already used by nothing in plankit but fully documented.
- **Deferred switch-back pattern** in `release.go:192-198` — the existing error recovery for switching back to source branch on merge failure. Extend this to also delete the local tag on push failure.

## Git version notes

- `git commit --trailer` requires **git 2.32+** (June 2021)
- `git log --format=%(trailers:key=...,valueonly)` requires **git 2.29+** (October 2020)
- `git switch` (already used in `release.go:185`) requires **git 2.23+** (August 2019)

All comfortably old. Document minimum as **git 2.32+** in `pk-changelog.md` and `pk-release.md` under "Requirements".

## Verification

1. `make build` — compiles
2. `make test` — all tests pass, including new trailer and `Undo` tests
3. `make lint` — `go vet` clean
4. **Manual smoke: `pk changelog`**
   - On a throwaway dev branch: `pk changelog --dry-run` → preview looks right
   - `pk changelog` → commit exists
   - `git log -1 --format=%B` → shows `Release-Tag: v0.X.Y` in the body
   - `git tag --list v0.X.Y` → returns nothing (no tag yet)
5. **Manual smoke: `pk release --dry-run`**
   - Reports the version read from the trailer
   - Reports it would tag and push
6. **Manual smoke: `pk release`** (on the throwaway setup)
   - Tag is created, branch + tag pushed to origin
   - `release.yml` fires on GitHub
7. **Manual smoke: `pk changelog --undo`**
   - Fresh `pk changelog` → `pk changelog --undo` → HEAD moves back, files restored, tree clean
   - `pk changelog` → manual `git push` → `pk changelog --undo` → refuses with clear error
   - `pk changelog --undo` with dirty tree → refuses
   - `pk changelog --undo` on an unrelated commit → refuses
   - `pk changelog --undo` on a branch with no upstream → succeeds

## Commit plan summary

1. `refactor: move tag creation from pk changelog to pk release via Release-Tag trailer`
2. `feat(changelog): add --undo to unwind an unpushed release commit`
3. `refactor(changelog): drop --push flag`
4. `docs: update command docs, skills, README for trailer-driven release flow`

Each commit leaves the tree buildable with passing tests. Commit 1 is atomic because the trailer handshake must exist on both sides at once.
