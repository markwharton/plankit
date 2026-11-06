# Review fixes, phase 1: wrong releases, lost plans, a committed credential

## Context

A code review of the repository at v1.8.3 found defects, duplication,
and housekeeping across pk. All of it is to be fixed, in phases, so
each phase is small enough to review and each fix is its own commit.

Phase 1 is the five defects that can cut a wrong release, lose an
approved plan, or commit a secret. Each was reproduced with a build of
HEAD in a scratch repository:

1. A `v*` tag that is not a semver tag (`vnext` on a side branch) makes
   `pk changelog` refuse with "the last tag, vnext, is not in develop's
   history", and `pk status` reports it as the latest tag.
2. A refused `pk changelog` leaves the tree dirty: a second version
   file that fails to stamp leaves the first one stamped, and a hook
   that edits a tracked file and then fails leaves the edit, under a
   hint that says "the tree is clean".
3. `pk preserve` whose commit fails leaves the plan file written and
   staged; the next run says "Plan already preserved" and never commits
   it. With the plans directory ignored by git it says "Plan unchanged,
   no commit needed." and commits nothing.
4. A lifecycle hook command that uses `${NAME}` for a variable pk does
   not own is rewritten before the shell sees it: `${FOO}_x` becomes
   `$FOO_x`, and `${NOPE:-fallback}` becomes `$NOPE:-fallback`.
5. An origin URL that carries a credential
   (`https://user:token@host/owner/repo.git`) is written, credential
   included, into the compare link in `CHANGELOG.md`, which is then
   committed and pushed. An `ssh://` origin produces a dead link.

The outcome: five local commits on `develop`, the suite green before
and after each, and the five reproductions rerun against the new build.
Nothing is pushed and no release is cut.

## Fix 1: one definition of the latest release tag

`docs/design.md` already states the model: "The last version is the
highest semver tag." The code has two choosers that disagree
(`git.LatestTag` takes the first `v*` name by git's sort;
`latestSemverTag` takes the first that parses), and
`TagOutsideHistory` re-derives the tag with the first while its caller
measured with the second.

- `internal/version/semver.go`: add `Latest(names []string) (string,
  Semver, bool)`, the highest name by `Semver.Compare` among names that
  are release tags. A release tag is a name that parses and renders
  back to itself, the same rule `ReadReleaseTagTrailer` applies to the
  trailer (`internal/changelog/trailer.go`), so `1.2.3` without the `v`
  and `vnext` are both passed over.
- `internal/git/git.go`: replace `LatestTag` with `Tags(dir)
  ([]string, error)`, which only lists. `RemoteTags` already only
  lists. The git package cannot choose, because it cannot import
  `version` (version imports cli, cli imports git).
- `internal/changelog/changelog.go`: add `LatestTag(root) (string,
  version.Semver, error)`, `version.Latest` over `git.Tags`; the tag is
  "" before the first release. `commitsSinceLastTag` calls it in place
  of its own tag listing and `latestSemverTag`, which is deleted.
  `TagOutsideHistory(root, branch, tag)` takes the tag its caller
  measured and no longer looks one up.
- Callers: `internal/release/release.go` (one `changelog.LatestTag`
  call feeds both the history check and the "Last tag" line),
  `internal/repo/repo.go` (init's first-release note, status's tag line
  and history note), and `originLatestTag`, which becomes
  `version.Latest(git.RemoteTags(root))`.

Behaviour that changes: a non-semver `v*` tag is ignored everywhere,
and the order is semver precedence, so `v1.0.0` outranks `v1.0.0-rc1`
where git's sort put the pre-release first. A failed tag listing is now
an error where `git.LatestTag` returned "".

Tests: `version.Latest` ordering (numeric, pre-release, non-semver and
unprefixed names passed over); the reproduction in
`internal/changelog/changelog_test.go` (a `vnext` tag off the branch,
release still measured from the semver tag); status reports the semver
tag. The ordering assertion in `internal/git/git_test.go` moves to the
version test, and the `git.LatestTag` calls in
`internal/repo/repo_test.go` move to the new function.

Commit: `fix(changelog): the latest release tag is the highest semver tag for every reader`

## Fix 2: a refused changelog puts the whole tree back

The tree is verified clean before anything is written, so everything
present after a failure was made by the run: pk's stamps, the
changelog, and whatever a hook edited or created. The refusal puts all
of it back, because `pk ship` promises that a failed ship leaves the
repository as it found it, and a hook's output from a release that did
not happen belongs to nothing.

- `internal/git/git.go`: add `RestoreTree(dir) error`: tracked files
  reset to HEAD, then untracked files removed, ignored files left
  alone. It is the same definition of clean that `CheckCleanTree`
  reads.
- `internal/changelog/changelog.go`: the section of `run` from the
  first version-file stamp to the release commit moves into one
  function, and `run` restores the tree on any error it returns. That
  covers the paths that today return without restoring: a version file
  that fails, a failed write of `CHANGELOG.md`, a failed `git add` or
  `git commit`. The per-file `written` list and its `checkout` loop are
  deleted.
- The hint says "the tree is clean" only after `RestoreTree` succeeds;
  when the restore itself fails, the error names that instead.

Tests, extending `TestFailingHookAborts`: a hook that edits a tracked
file and creates a new one, then fails, leaves `git status` empty; two
version files with the second missing leave the first unstamped.

Commit: `fix(changelog): a refused release puts the whole tree back, hook changes included`

## Fix 3: a preserved plan is a committed plan

"Already preserved" is decided today by finding identical bytes in the
plans directory. Preserved means committed, and git already answers
that.

- `internal/git/git.go`: add `Ignored(dir, path) (bool, error)`,
  reading git's exit status the way `IsAncestor` does.
- `internal/preserve/preserve.go`, in `preserve`:
  - The target is the duplicate file when one exists, else the next
    sequence-numbered name. A target git ignores is refused before
    anything is written, naming `preserve.dir`; the pointer stays.
  - A duplicate that is committed reports "already preserved", as now.
    A duplicate that is not committed is the leftover of a failed run:
    it is committed as it stands, with no second copy written.
  - The "Plan unchanged, no commit needed." branch is deleted: a new
    sequence number is never already committed, and the ignored
    directory that reached it is now refused.

The refusal is reported the way this function reports its other
failures. Whether a typed `pk preserve` should exit non-zero on a
failure is a separate decision, in phase 2.

Tests in `internal/preserve/preserve_test.go`: a commit that fails
(a failing pre-commit hook in the scratch repository), then a rerun
that commits the same file once; an ignored plans directory is refused
and nothing is written.

Commit: `fix(preserve): a plan whose commit failed is committed on the next run`

## Fix 4: hook commands keep every variable pk does not own

`RunScript` pre-expands `$VERSION` and `$TAG` in the command string so
one spelling works under `cmd` as well as `sh`. It does so with
`os.Expand`, which hands back every other `${...}` without its braces.

- `internal/hookio/hookio.go`: replace the `os.Expand` call with one
  function that substitutes `$NAME` and `${NAME}` only when NAME is a
  key of the env map, and copies every other byte as written. The
  variables are still set in the environment, so `${VERSION:-x}` is
  left for the shell and resolves there.

Tests in `internal/hookio/hookio_test.go`, a table over the expansion:
`$VERSION`, `${VERSION}`, `v${VERSION}`, `$VERSIONS`, `${FOO}_x`,
`${NOPE:-fallback}`, `$(cmd)`, `$$`, a trailing `$`. The existing
changelog and release tests cover the hooks end to end.

Commit: `fix(changelog): a hook command keeps the ${NAME} forms that are not pk's`

## Fix 5: the compare link never carries a credential

- `internal/git/git.go`, `ParseRepoURL`: a URL with a scheme is parsed
  as a URL; its user and password are dropped; `ssh://` becomes
  `https://` on the same host without the port. The scp form
  (`user@host:path`) converts for any user, not only `git`. A local
  path is returned as it is today, which the changelog tests rely on.

Tests in `internal/git/git_test.go`, a table: credentialed https,
`ssh://` with and without a port, scp form, plain https, a local path.

Commit: `security(changelog): a credential in origin's URL never reaches CHANGELOG.md`

## Where each fact lives after this phase

- The chooser of the latest release tag is `version.Latest` and nothing
  else. `git.Tags` and `git.RemoteTags` list; `changelog.LatestTag` is
  the one composition for local tags; `TagOutsideHistory` is handed the
  tag. Check: a search for `git.LatestTag` and `latestSemverTag` shows
  no hits. Two copies remain in `tools/docgen` (`latestTag` in site.go,
  `previousTag` in notes.go) and go in phase 4.
- Putting the tree back is `git.RestoreTree`, and the changelog's
  writing section has one way out through it.
- "Committed" is `git.Clean` on the path, already there; "ignored" is
  the new `git.Ignored`.
- Hook-command expansion is one function with one caller, `RunScript`;
  the four hook call sites do not change.
- Remote URL to web URL is `ParseRepoURL`. docgen's `repoWebURL` copy
  goes in phase 4.

## Verification

- `make test` and the docgen tests before the first change and after
  each of the five, with the output shown.
- Each new test is run and seen to fail before its fix, then pass.
- `gofmt -l cmd internal tools` prints nothing; `make docs` leaves no
  diff.
- Smoke test against a fresh build, in scratch repositories whose bare
  origin is made by cloning, so nothing is pushed:
  1. `vnext` off the branch: `pk changelog --dry-run` generates the
     next version and `pk status` names the semver tag.
  2. The case that must fail: a semver tag that really is outside the
     branch's history is still refused, exit 2.
  3. A failing hook that edited a file, and a missing second version
     file: `git status --porcelain` is empty after each refusal.
  4. A failing commit under `pk preserve --plan`, then a rerun: one
     plan file, one `plan:` commit.
  5. A hook that echoes `${FOO}_x` and `${NOPE:-fallback}` prints what
     `sh` prints.
  6. `ParseRepoURL` on a credentialed URL, through its test.
- Reread the `Summary` lines, the changelog, preserve, release, and
  status pages, and `docs/architecture.md` against the new behaviour
  before each commit; correct any sentence the change makes false in
  the same commit.

## Later phases, each with its own plan

- Phase 2, the remaining proven defects: the brief omits the push
  policy when branch guarding is off; `pk release` on a detached HEAD;
  `pk pin` reporting any read error as a missing file; guard's two
  tokenisers; protect on a case-insensitive filesystem; the silent
  typed `pk preserve` under the minimum size; the hook exit contract
  and its stale comments. Decisions to take there: guard's subcommand
  list, how far guard chases wrapped commands, the exit code for a
  payload that does not parse, and whether a typed `pk preserve` exits
  non-zero on failure.
- Phase 3, gates and supply chain: `make test` runs the docgen tests;
  exact versions for wrangler, mermaid, and the Claude Code CLI in the
  workflows and layout; the tap token scoped to its step; Dependabot
  watching `tools/docgen`; `--plan` refusing `..`.
- Phase 4, duplication and dead code: the IR types, command-page
  detection, breaking-marker detection, release pre-flight checks,
  config loading, current-branch lookup, hook call sites, pin, docgen's
  semver and URL copies, the unused message catalogue, late policy
  validation, swallowed errors, leftovers. Decisions there: delete or
  adopt the catalogue, and the exit code for an unreachable origin.
- Phase 5, tokens: named mode values, the help style table, narration
  through `msg`, guard's reason prefixes, the stylesheet. Decision
  there: which sizes and radii the stylesheet keeps.
