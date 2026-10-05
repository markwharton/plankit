# The last version tag must be in the branch's history

## Context

`pk changelog` measures the next entry from the latest version tag to
HEAD, and never asks whether that tag is in HEAD's history. When a branch
has been rewritten after a release, by a rebuild and a force push, the
tag stays on the old commits and `git log v0.1.0..HEAD` lists everything
on the new branch the tag cannot reach: the work `v0.1.0` already
released appears again in the `v0.2.0` entry, and the entry's compare link
joins two unrelated histories. `pk release` tags on top of that, and
`pk status` shows the tag as if all were well. Nothing says anything.

The fact is one git question, whether the tag is an ancestor of HEAD, and
it belongs in the three commands that already read the tag. Changelog
refuses before writing, release refuses in its pre-flight before tagging,
and status notes it beside the tag line. The way out is the developer's,
since only they know whether the rewritten history has a commit that
matches the release: move the tag there, or delete a tag that was never
meant. pk names both and moves no tag.

This is a fix, one commit, a patch release.

## What exists to build on

- `internal/changelog/changelog.go:141-163`: the tag lookup,
  `latestSemverTag`, the no-tag path, and the range. The check slots in
  after `found`, before the range is built.
- `internal/release/release.go:99-182`: the pre-flight list. The check
  sits after the release-branch block ends at `:179` and before the
  `Release-Tag trailer` line at `:180`, so it runs before the rollback
  defer and before any mutation. Release has no tag lookup of its own and
  calls `git.LatestTag`; the pending tag is absent by the `:86` check, so
  that returns the previous release.
- Release already runs `merge-base` inline three times: `:138` compares
  `merge-base HEAD origin/<src>` with `rev-parse origin/<src>` for "not
  behind", which is the same question as "is origin/<src> an ancestor of
  HEAD"; `:166` `--is-ancestor origin/<rel> HEAD`; `:214` `--is-ancestor
  <rel> <src>` in dry run.
- `internal/git/git.go`: no ancestor helper; `CountCommits` at `:92` and
  `RootCommit` at `:193` are the neighbours.
- `internal/repo/repo.go:371-391`: the readiness notes after the tag line;
  the new note goes after the ahead-of-tag note at `:391`.
- `internal/ship/ship.go` adds no checks of its own; both halves run
  through `cli.RunIO`, so a refusal in either reaches ship unchanged.
- Fixtures: `changelog_test.go` `repo` and `release_test.go` `repo`
  (returns the bare origin too) both tag `v0.0.0` on the root and push
  `main` and `develop`; `release_test.go` `pending` commits and runs
  changelog. `repo_test.go` `scratch` has no origin. A branch that does
  not contain a tag is made in any of them with
  `git switch -c side v0.0.0`, a commit, `git tag v0.0.1`, and a switch
  back: `LatestTag` then returns `v0.0.1`, outside `develop`.

## Changes

### 1. One predicate, `internal/git/git.go`

```go
// IsAncestor reports whether ancestor is reachable from rev.
func IsAncestor(dir, ancestor, rev string) (bool, error)
```

`merge-base --is-ancestor ancestor rev`: exit 0 is true, exit 1 is false,
anything else is the error. Release's three inline `merge-base` calls move
onto it with identical behaviour: "not behind" becomes
`IsAncestor(origin/<src>, HEAD)`, the diverged check becomes
`IsAncestor(origin/<rel>, HEAD)`, and the dry-run merge check becomes
`IsAncestor(<rel>, <src>)`. The pass and refuse cases are unchanged; the
one difference is that today's two inline `--is-ancestor` calls read any
git failure as "diverged", and the helper reports a real git error as an
error instead of exit 1's false. The inline command text leaves.

### 2. One sentence, `internal/changelog`

```go
// TagOutsideHistory returns the message for a latest tag that the branch
// does not contain, or "" when the tag is in the branch's history or
// there is no tag.
func TagOutsideHistory(root, branch string) (string, error)
```

It reads `git.LatestTag` and `git.IsAncestor(tag, "HEAD")`. The message:

```
the last tag, v0.1.0, is not in develop's history: the branch was
rewritten after that release, and the next entry would list work v0.1.0
already released. Move the tag to the commit in the new history that
matches the release, or delete it if it was never meant, and update
origin's copy of it
```

The changelog package owns it because the consequence it states is the
changelog's. Three callers:

- **changelog**, after `latestSemverTag` finds a tag and before the range:
  a non-empty message is a state error, exit 2, in dry run and real run,
  before anything is written.
- **release**, in the pre-flight list before the trailer line: a
  non-empty message is the same state error; on success the list prints
  `Last tag v0.1.0 is in develop's history`. The check runs on the source
  branch's HEAD, before the merge.
- **status**, after the ahead-of-tag note, under the same `--quiet` gate:
  the message as a note. The `state` struct gains no field; the notes are
  stderr.

Ship needs nothing: each half's refusal passes through it as today.

### 3. Pages

- `skills/changelog/SKILL.md`, the "Run it from the working branch"
  paragraph: the last tag must be in the branch's history; a branch
  rewritten after a release is refused, naming the tag, since the entry
  would list that release's work again.
- `skills/release/SKILL.md`, Pre-flight: add "last tag in the branch's
  history" to the list.
- `skills/status/SKILL.md`, the notes paragraph: a last tag the branch
  does not contain is noted.
- No release note: a patch.

### Tests

- `internal/git/git_test.go`: `IsAncestor` true, false, and an error for
  an unknown revision.
- `internal/changelog/changelog_test.go`: from the fixture, a side branch
  off `v0.0.0` with a commit tagged `v0.0.1`, back on `develop` with a
  feat: `pk changelog --dry-run` exits 2 naming `v0.0.1` and `develop`,
  and writes nothing; `pk changelog` the same. With the side tag deleted,
  the same repository generates as before.
- `internal/release/release_test.go`: after `pending`, the same side
  branch and tag; `pk release` refuses in pre-flight naming `v0.0.1`, no
  tag is created locally or on origin, the merge did not happen, and the
  working branch is checked out. The three moved `merge-base` calls keep
  their tests: behind-origin and diverged in `TestRefusals` and
  `TestDivergedReleaseBranchRefused`, and `TestDryRunTouchesNothing`.
- `internal/repo/repo_test.go`: `scratch`, `pk init`, a side branch tag
  as above; `pk status` on `develop` notes the tag; `--quiet` drops it;
  after `git tag -d` the note is gone.

## Boundaries, so the change adds no second copy

The implementation introduces no second copy of any fact and none of the
anti-patterns in the repository's record: no silent fallback, no network
call added, no injected filesystem in tests, no sibling helper where an
existing one can be extended. Before committing, grep the diff for a
repeated block and read the commit's `--stat` file list.

- **One predicate.** `git.IsAncestor`, with release's three inline
  `merge-base` calls moved onto it, so after this commit `merge-base`
  appears in one place.
- **One sentence.** `changelog.TagOutsideHistory` builds the message; the
  three commands print it and decide only whether to stop or to note.
- **One tag lookup per command.** Changelog uses the tag it already
  found; release and status use `git.LatestTag` as they do today.
- **Existing fixtures.** A side branch and tag inside `repo`, `pending`,
  and `scratch`; no new fixture.

## Commit

`fix: the last version tag must be in the branch's history`

## Verification

- `make test` before and after, to a log, gated on the exit status.
- Smoke, the rewritten history: scratch repository, `pk init`, two
  conventional commits, `git tag v0.1.0`; replay the two commits on a new
  branch from the root with `git cherry-pick`, add a third, point
  `develop` at it, give it an origin; `pk status` prints the note;
  `pk changelog --dry-run` exits 2 with the message and no `CHANGELOG.md`;
  `pk release` exits 2 in pre-flight with no tag made; move the tag to the
  replayed second commit with `git tag -f`; all three run clean and the
  dry run lists only the third commit.
- Must fail is the smoke's refusals; the clean run after the retag is the
  pass.
- `claude plugin validate . --strict` and `make docs` leave the tree as
  expected.
