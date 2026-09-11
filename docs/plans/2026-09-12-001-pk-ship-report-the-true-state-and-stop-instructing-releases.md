# pk ship: report the true state, and stop instructing releases

## Context

Two problems surfaced while using the plugin in another repository.

**1. pk ship and pk release misread a finished release as a pending
one.** Both treat "HEAD carries a Release-Tag trailer" as "a release is
pending". A release commit keeps its trailer after it ships, so a second
pk ship on an already released HEAD prints:

```
Release-Tag v0.1.0 already pending; skipping changelog
Error: tag v0.1.0 already exists locally; nothing to release
Hint: the release commit remains pending; rerun pk ship to retry, or pk changelog --undo to unwind
```

Every line is wrong: the release is finished, nothing is pending, and
pk changelog --undo then refuses because the commit is pushed.
Reproduced locally. The same defect gives pk ship a second wrong
sequence when there is nothing to release at all: changelog reports no
commits, then release fails with "no Release-Tag trailer on HEAD; run
pk changelog first" followed by the same "remains pending" hint.

**2. The ship page instructs an agent to release.** skills/ship/SKILL.md
carries a standing procedure, "Unattended releases", telling Claude Code
to run pk ship after a dry run. It never says what starts a release.
Being shown the page, and later a preview, was enough to trigger two
releases in one session that the developer had not asked for. A
published tag cannot be withdrawn without a force push.

The outcome: pk's messages describe the state the repository is actually
in, and the plugin no longer tells an agent to start a release on its
own reading.

## Part 1: define pending once

A release is pending when HEAD carries a Release-Tag trailer **and** that
tag does not exist yet.

In `internal/changelog/trailer.go`, beside `ReadReleaseTagTrailer`, which
stays a pure read:

```go
// TagState says where a Release-Tag trailer's tag stands.
type TagState int

const (
	TagAbsent    TagState = iota // no such tag: the release is pending
	TagOnHead                    // the tag is on HEAD: the release has been made
	TagElsewhere                 // the tag is on another commit: the trailer conflicts with it
)

func ReleaseTagState(dir, tag string) (TagState, error)
```

Computed from `git tag --list <tag>`, the call `release.go:84` already
makes, plus `git rev-parse <tag>^{commit}` against HEAD.

`internal/release/release.go`, replacing the check at line 84:

- `TagOnHead`: "HEAD is already tagged v1.2.3; nothing to release",
  exit 2 as now.
- `TagElsewhere`: "tag v1.2.3 already exists on another commit", with
  the hint "to restage under the next version: pk changelog --undo,
  then pk changelog". changelog anchors on the highest local `v*` tag
  (`changelog.go:133`), so a rerun computes a version above the
  conflicting tag.

`internal/ship/ship.go`:

- Skip the changelog half only when a release is genuinely pending
  (trailer present and `TagAbsent`).
- After the changelog half, re-read. Run the release half only if a
  release is now pending. Otherwise return, because changelog has
  already said why, for example "No new conventional commits found."
- The "remains pending" hint then prints only when a release commit
  really is waiting.

Both sequences in the Context then exit 0 after changelog's own
narration. pk release on its own still exits 2, now saying HEAD is
already tagged.

## Part 2: remove the standing instruction

In `skills/ship/SKILL.md`, delete `## Unattended releases` (lines 28 to
34). Its one piece of real content, the caution about a major bump,
moves into a section that states the trigger first:

```markdown
## Releasing

Releasing is the developer's call. Run `pk ship` when the developer
asks for a release, not on your own reading of a preview. When the
inferred bump is a major, show the commits carrying `!` or
`BREAKING CHANGE` and the section, and wait for the developer's go.
```

And a note for the developer reading `pk help ship`:

```markdown
## Permissions

`pk ship` and `pk release` publish: they tag and push. Where a session
allowlists pk commands, allowlist the ones that only read or stage
(`pk status`, `pk changelog`, `pk ship --dry-run`) and leave `pk ship`
and `pk release` off, so a release still needs the developer's
approval.
```

## Part 3: the prose that says the same thing

- `docs/architecture.md:97` ("Its only state is the trailer") and the
  package comment in `internal/ship/ship.go`: the state is the trailer
  and whether its tag exists. The phrase "can run unattended" in that
  comment stays; it describes a command that never prompts, which is
  still true.
- `make docs` recompiles skills into `internal/help/data`, which is
  committed.

## Tests

`internal/ship/ship_test.go`, using the existing `repo`, `work` and
`runShip` helpers:

- ship on an already released HEAD: run ship, then run it again. The
  second run exits 0, says "No new conventional commits found.", and
  says neither "already pending" nor "remains pending".
- ship with nothing to release (baseline tag, no trailer): exits 0,
  with no "no Release-Tag trailer on HEAD" and no "remains pending".
- `TestShipResumesFromPendingTrailer` must stay green: a trailer whose
  tag does not exist is still pending.

`internal/release/release_test.go`, `TestRefusals`: the "already exists
locally" case at line 153 becomes "HEAD is already tagged"; add a case
with the tag on an earlier commit, expecting "another commit".

## Verification

- `make test` before and after each change, plus `make build` and
  `make docs`.
- Smoke test with `./pk` in a throwaway repo, its origin made with
  `git clone --bare`. No push happens: every path here ends before
  release's push.
  - pk ship on the baseline: exit 0, "No new conventional commits
    found."
  - commit, pk ship, then pk ship again: exit 0, the same line, and
    neither "already pending" nor "remains pending".
  - Must fail: pk release on the released HEAD exits 2 with "HEAD is
    already tagged v0.1.0".
- `pk help ship` renders without the Unattended section and with the
  two new ones.
- `grep -rn -i "unattended"` finds only the ship.go package comment.

## Not in this change

- `guard.release`, a setting that would ask before pk ship and
  pk release. A hook sees the command, not the conversation, so it
  cannot tell a requested release from an unrequested one and taxes
  every release equally. Revisit only if this recurs.
- Guard resolving the repository from a command's `git -C` or `cd`
  target rather than from the session's directory.
- The commit-type discipline that put a tooling change in the
  changelog's Added section.
