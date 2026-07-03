# Ship the versioning invariant as always-on guidance

## Context

An agent working in a pk-managed repo has no ambient statement of pk's versioning model: the always-on context pk installs is CLAUDE.md and the shipped rules, and none of them state it. A version-surfacing task therefore tempts the convenient anti-pattern (read the current version from a file and write it into committed output) during ordinary feature work, long before `/ship` runs. The fix is a general statement of the invariant in the shipped rules: present every session, self-contained as a rule must be, and free of any framework- or incident-specific detail.

The model, verified in code (`internal/changelog/changelog.go`): **versions are computed from git history and written into files; they are never read from files.** `pk changelog` finds the latest semver tag as the baseline (`changelog.go:160-166`), computes the next version from the conventional commits since it, writes that computed version into configured files, and records it as a `Release-Tag` trailer; `pk release` creates the tag afterwards. `updateVersionFile` (`changelog.go:636`) locates the root JSON `"version"` key and splices in the computed value, discarding the existing one; `pk pin` scans for a marker line but the value always comes from `$VERSION`. Committed generated artifacts are regenerated and staged by a changelog `preCommit` hook, so they are built with the release version and land atomically in the changelog commit (and are reverted by `--undo`).

Placement (confirmed with Mark): split across the craft/conduct axis — the invariant in `git-discipline.md` (craft), a task-redirect in `plankit-tooling.md` (conduct). CLAUDE.md template stays terse; `/ship` is too late in the flow.

## Changes

### 1. `git-discipline.md` — craft invariant (embedded + local copy)

Files: `internal/setup/rules/git-discipline.md` and `.claude/rules/plankit/git-discipline.md` (identical body edits).

Add one bullet adjacent to "Don't rewrite history between `pk changelog` and `pk release`" (same release-flow cluster; bullets are ordered by importance). Single-line bullet, no hard wraps. Draft:

> **The git tag is the single source of truth for version.** `pk changelog` computes the next version from the latest tag and the conventional commits since it, writes that version into configured files (versionFiles, `pk pin`, hook scripts), and `pk release` creates the tag; the version is never read out of a file. Never read the version out of package.json or a source constant in code you write. Files that must carry the version are wired into the changelog config; generated files that are committed are regenerated and staged by a changelog hook so they carry the release version.

Frontmatter `description:` unchanged.

### 2. `plankit-tooling.md` — conduct redirect (embedded + local copy)

Files: `internal/setup/rules/plankit-tooling.md` and `.claude/rules/plankit/plankit-tooling.md`.

New `## Versioning` section (after `## Hook Behavior`, before `## Session Bootstrap`) with one bullet. Draft:

> **A task that surfaces the release version goes through release-time stamping, never a read of the current version.** Wire the file into `.pk.json`'s changelog config: `versionFiles` for a root JSON version field, `pk pin` in a hook for source constants, a hook script for anything else. Never read the version out of package.json or a source constant in code you write; versions are computed from the git tag history and written into files, never read from them.

### 3. `pk_sha256` recompute for both local copies

Per CLAUDE.md "Updating pk-managed files": recompute with `sed -n '/^---$/,/^---$/!p' <embedded-source> | shasum -a 256` and replace each local copy's `pk_sha256:` line. Avoids running `pk setup`.

### 4. `docs/versioning.md` — two general additions

- **Source of truth section** (after the existing paragraph at line 7), house style (bold lead-in; docs/ uses no callout boxes):

  > **The version flows one way.** Never read it back out of a downstream target: reading the version from package.json or a source constant at build or generate time and writing the result into committed files bakes in whatever the last release left behind, and those files lag every release. Files that must carry the version are wired into the changelog config and stamped at release time.

- **Hook scripts — custom version sync section** (end of section, ~line 128), the committed-generated-artifacts case stated generally:

  > Generated files that are committed follow the same model: regenerate them in a `preCommit` hook and stage the result, so they are built with the release version and land in the changelog commit; `pk changelog --undo` reverts them with everything else.

### 5. `docs/pk-rules.md` — refresh drifted example numbers

The Example output block (lines 39–49) lists real sizes/token counts for the shipped rules; the `git-discipline.md` and `plankit-tooling.md` rows and the always-on total drift with these edits. Refresh from `dist/pk rules` output (only the changed rows and totals; the stylized `scoped.md` line stays).

### Explicitly not changed

- **CLAUDE.md template** (`internal/setup/template/CLAUDE.md`) — stays a terse critical-rules list.
- **`/ship` skill** — release time is too late; the redirect must be ambient.
- **`docs/adoption.md`, `docs/pk-changelog.md`** — propagation taxonomy unchanged (no new mechanism or config key); `versioning.md` is the narrative home, already linked from both.
- **README footprint line** — regenerates via `go run ./evals/footprint` in the `preCommit` changelog hook at next release.

## Verification

Automated:
- `make test` before and after (includes `embed_safety_test.go`, which scans the edited embedded rules for hidden characters) and `make lint`. No test fixtures pin real rule content or hashes (`managed_test.go` uses synthetic content only).

Smoke:
- `make build`, then in a scratch dir: `git init`, run `dist/pk setup`; confirm installed `.claude/rules/plankit/git-discipline.md` and `plankit-tooling.md` contain the new text with a `pk_sha256` line.
- In the plankit repo: `dist/pk rules` reports both edited local copies as `[managed]` pristine (proves the hash recompute matched the embedded body byte-for-byte). **Negative case:** run `dist/pk rules` after editing the local copies but before recomputing the hashes — they must report as modified; after recompute, pristine.

Commits (on `develop`, no push — developer's call):
1. `feat(setup): ship version-source rule (tag is the single source of truth)` — both embedded rules, both local copies, hash recomputes, pk-rules.md example refresh.
2. `docs: state the one-way version flow in versioning.md`
