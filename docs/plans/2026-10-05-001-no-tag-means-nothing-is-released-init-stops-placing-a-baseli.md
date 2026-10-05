# No tag means nothing is released: init stops placing a baseline, and works from any start

## Context

Starting a repository with plankit fails in the same way every time it
fails: work gets committed before `pk init`, and plankit's own machinery
then treats that work as history rather than as the first release. A
repository started on `develop` with a few conventional commits and the
policy file copied in by hand ends up with a release branch made by hand at
the newest commit, and the no-tag hint, `git tag v0.0.0`, tags HEAD and
marks everything as released. Recovery means rewriting history.

The root is one decision plankit asks at the wrong moment and stores in a
tag: where released history ends. `pk init` answers it by tagging `v0.0.0`
at HEAD, right for adopting a mature repository and wrong for a new one, and
two hints repeat that answer. Every symptom follows: buried work, hints that
are unsafe as written, a release branch holding unreleased commits, and the
rule "init before any work" that exists only to keep the decision from
going wrong.

This plan removes the decision. No version tag means nothing has been
released: the first changelog covers the whole history and its base version
is an implied `v0.0.0`. Init stops tagging, so the order of init and first
commits no longer matters. Init also learns to create the release branch
when `--release` names one that does not exist, at the root commit so it
carries no unreleased work, which makes a `git init -b develop` start work.
Status reports the two states a hand-configured repository leaves behind, a
release branch that does not exist and one that has moved past its last
tag, from git alone. Adopting a repository whose history should stay out of
the first release becomes an explicit act, tagging where the developer
chooses, and the changelog page says so.

Two things stay as they are. pk shells out to git only, so the remote's
default branch is a sentence in the recipe, not a command pk prints. And
the brief stays silent in an unconfigured repository: the architecture's
second promise is that a repository without a policy file gets no action
from any hook.

## What exists to build on

- `internal/changelog/changelog.go:139-153`: tag listing, `latestSemverTag`,
  the shallow-clone guard (`ls-remote --tags origin`, keep), and the no-tag
  error with the `git tag v0.0.0` hint (goes). `:155` the range
  `latestTag+"..HEAD"`; `:242` the compare link; these are the only two
  readers of the previous tag. `baseVersion.Bump(bump)` at `:177`.
- `internal/release/release.go` and `internal/ship` read no previous tag;
  `ReleaseTagState` only asks whether the trailer's tag exists. Release tags
  with its own `git tag` at `release.go:284` rather than `git.CreateTag`.
- `internal/repo/repo.go:124-131` decides `baseline` and `branch`; `:167`
  tags; `:172` `git.CreateBranch` (switches); `:177-189` the push list;
  `:211-223` the notes and the `--no-commit` hint naming `git tag v0.0.0`;
  `:349-363` status readiness notes.
- `internal/git/git.go`: `LatestTag`, `CreateTag` (HEAD only),
  `CreateBranch` (switches), `BranchExists`, `HasOtherLocalBranch`,
  `PushRefs`, `OriginHead`. No helper for a branch at a revision without
  switching, the root commit, or a commit count between revisions.
- `tools/docgen/notes.go:134-147` `previousTag`: a note with no lower tag
  gets plain text, no link.
- Tests: changelog, release, and ship fixtures tag `v0.0.0` after a
  scaffold commit (they now model an adopted repository and stay as they
  are); no test asserts the no-tag error or hint; `repo_test.go` pins the
  baseline at lines 95, 113, 142, 151, 221, 228-240, 251, 275, 285, 311,
  319, 353.

## Changes

### 1. The changelog treats no tag as nothing released, `internal/changelog`

In `changelog.go`, when `latestSemverTag` finds nothing and the
shallow-clone guard does not fire, continue with `latestTag = ""` and
`baseVersion` zero (`v0.0.0`). The range is `HEAD` when `latestTag` is
empty, the whole history, root included. The reference link becomes
`[vX]: <repo>/commits/vX` when there is no previous tag, since a compare
needs two ends; every later release keeps the compare form. The
`to anchor at v0.0.0` hint is deleted; the `git fetch --tags` hint for a
shallow clone stays, because there "no tags" is false.

Why this is the whole change, traced through the release flow:

- `changelog.go` reads `latestTag` in exactly two places, the log range at
  `:155` and the link at `:242`. `baseVersion` feeds only `Bump` at `:177`,
  and `Semver{}.Bump(minor)` is `v0.1.0`. `parseLog`, `detectBump`, the
  section, `insertSection`, the version-file splice, the hooks, and the
  commit with the `Release-Tag` trailer never see a tag.
- `--undo` reads no tag: it needs the trailer, a clean tree, and an
  unpushed commit.
- `release.go` reads only the trailer's tag and asks `ReleaseTagState`
  whether that tag exists: absent, on HEAD, or elsewhere. With no tags at
  all the answer is absent, which is the normal path: tag, push. Its
  rollback deletes only the tag it made. Merge flow checks the release
  branch exists and fast-forwards it; trunk flow checks the default
  branch. Neither consults a previous tag.
- `ship.go` derives the pending state from the trailer and
  `ReleaseTagState`, nothing else.
- Today's repositories already prove the path from the other side: the
  first entry in this repository's CHANGELOG is `v0.1.0`, computed from
  `v0.0.0` on the root commit. The implied `v0.0.0` gives the same
  arithmetic with the root commit inside the range instead of excluded.

With no tag, the first stderr line of `pk changelog`, dry run or real,
says `no release yet: the first release covers all N commits`, before
anything is written. A repository adopting plankit with a long history
thereby hears what the first entry would be while `--undo` can still take
it back, and the changelog page names the way to start later: tag
`v0.0.0` at the last commit that is history.

The no-tag changelog test is written first and must fail before the
change, so the assurance is a test, not a reading.

`tools/docgen/notes.go`: a note whose `previousTag` is empty links to
`<repo>/commits/<version>` instead of going unlinked, the same choice.

`internal/release/release.go:284` tags through `git.CreateTag` instead of
its own `git tag`, so `CreateTag` keeps one caller after init stops using
it.

### 2. Init places no baseline and creates a missing release branch, `internal/repo/repo.go`

- Remove the `--no-baseline` flag, the `baseline` decision, the `tag`
  step, the `baseline` JSON key, the "no baseline tag: the repository has no
  commits yet" note, and the `git tag v0.0.0 &&` clause of the `--no-commit`
  hint (it keeps `git switch -c <working>`).
- When `--release` names a branch that does not exist locally and is not
  the branch checked out, add a step labelled `branch <release>` that
  creates it at the root commit without switching. The checked-out branch
  is the working branch. With `--release` absent on a branch whose name is
  the `--branch` default, the existing usage refusal gains a hint:
  "name the release branch: pk init --release main". The dry run previews
  the new step like the others.
- The push list is the release branch plus the working branch when one
  exists apart from it: the branch init created, or the branch checked out
  when it is not the release branch. No tag is pushed.
- When the repository had commits before init's own, a note after the
  summary: `N commits precede plankit's and will be in the first release;
  to keep them as history, tag the last of them: git tag v0.0.0 <sha>`,
  with the abbreviated sha of the commit before the configure commit (of
  HEAD under `--no-commit`). The sha, not `HEAD~1`, so the command stays
  exact whenever it is run. This is the adoption path: pk states the
  count and the command and decides nothing. `--quiet` drops it like the
  other notes.
- Three git helpers in `internal/git/git.go`, one caller each at first:
  `RootCommit(dir)` (`rev-list --max-parents=0 HEAD`), `CreateBranchAt(dir,
  name, rev)` (`git branch name rev`, no switch), and `CountCommits(dir,
  from, to)` (`rev-list --count from..to`). `CreateBranch` keeps its
  switching meaning for the working branch.

### 3. Status reports what git shows, `internal/repo/repo.go`

- The `tag` line always prints: the latest tag, or `none yet` when there is
  no release. JSON keeps `latestTag` with `omitempty`.
- Remove the "no baseline tag" note. Add two, under the same `--quiet`
  gate:
  - release branch named in the policy but absent locally:
    `release branch main does not exist: git branch main $(git rev-list
    --max-parents=0 HEAD) creates it at the root, carrying no unreleased
    work`. The command is safe to run as written.
  - merge flow (`release.branch` set), a tag exists, and the release branch
    is ahead of it: `main has 4 commits not in a release`. No command: what
    to do depends on how they got there. Not before the first release,
    because then every fresh repository would be noted.
- The working-branch note stays.

### 4. Pages

- `skills/init/SKILL.md`: drop the tag bullet and `--no-baseline`; the
  JSON keys lose `baseline`; `--release` prose says a named branch that
  does not exist is created at the root commit; a new section, "Starting a
  repository":

  ```
  git init -b main
  pk init
  ```

  then one sentence that `pk init --push` publishes both branches once
  `origin` exists, that work committed before init is in the first release
  like everything after it, and that the remote's default branch is best
  set to the working branch so clones start there. Git and pk only.
- `skills/changelog/SKILL.md`: one paragraph. With no version tag the
  first release covers the whole history from an implied `v0.0.0`, and its
  changelog entry links to the tag's commits; to keep earlier history out
  of it, tag `v0.0.0` at the last commit that is history before running
  `pk changelog`.
- `skills/status/SKILL.md`: the notes paragraph names the three notes.
- `README.md`: the after-init sentence drops "tags the v0.0.0 baseline".
- `docs/design.md`, "How a release is computed": one sentence, no tag
  means nothing released and the first release covers all history.
- `tools/docgen/demo.go:95` comment drops "tags the baseline".
- `docs/notes/v1.6.0.md`: the first release covers everything; init
  places no baseline and works from a working branch; status names a
  missing or moved release branch; tagging is how history is excluded.

### Tests

- Changelog: a scratch repository with no tag at all, two conventional
  commits from the root; `pk changelog` yields `v0.1.0` for a feat, the
  section lists both commits including the root, the link is the commits
  form, and stderr opens with `no release yet: the first release covers
  all 2 commits` in dry run and real run alike. The shallow-clone guard keeps its behaviour: a test with a
  tagged bare origin and an untagged clone still gets the fetch hint.
- Release: the first release in the no-tag repository tags and pushes;
  `ReleaseTagState` is unchanged.
- Repo, init: the listed assertions lose the tag; the step list is
  `.pk.json, .claude/settings.json, commit, branch develop`; JSON has no
  `baseline`; `--push` pushes `main develop`; a repository started with
  `git init -b develop` and four commits plus `pk init --release main`
  ends with `main` at the root commit, `develop` checked out, the policy
  committed on `develop`, and `--push` sending both; the usage refusal on
  `develop` without `--release` carries the hint; `TestInitNoBaselineOnRepoWithCommits`
  becomes the work-before-init case asserting no tag, the configure
  commit on top, and the note naming the count and the sha of the commit
  below it, absent on an empty repository and under `--quiet`;
  `TestInitFlags` drops `--no-baseline`.
- Repo, status: `none yet` on the tag line; the missing-release-branch
  note with its command, absent once the branch exists; the ahead note in
  merge flow after a tag, absent in trunk flow and absent before a tag.
- Docgen: an unlinked first note becomes the commits link.

## Boundaries, so the change adds no second copy

The implementation introduces no second copy of any fact and none of the
anti-patterns in the repository's record: no text search of history for a
commit subject, no silent fallback, no network call added, no injected
filesystem in tests, no sibling helper where an existing one can be
extended. Before each commit, grep the diff for a repeated block and read
each commit's `--stat` file list.

- **One meaning for "no tag".** Decided once in `changelog.go` where the
  tag is looked up; the range and the link read `latestTag == ""` from
  that one place. Status's `none yet` is display of `LatestTag`, not a
  second decision.
- **One tagger.** `git.CreateTag` with release as its caller; init no
  longer tags.
- **One root-commit read.** `git.RootCommit`, used by init's branch
  creation and by status's note text.
- **One hint text per situation.** The missing-release-branch command
  appears in status only; the init page describes, it does not repeat the
  command.
- **Fixtures stay.** The changelog, release, and ship fixtures keep their
  `v0.0.0`; one new no-tag test each where behaviour differs, no parallel
  fixture family.
- **Existing helpers.** `scratch`, `mustGit`, `run` in `repo_test.go`;
  `repo`, `commit`, `runCL` in `changelog_test.go`.

## Found, not in scope

- The status working-branch note hardcodes `develop` rather than reading
  init's `--branch`; init does not record it, so there is nothing to read.

## Commits

1. `feat(changelog): no tag means nothing has been released`
2. `feat(init): no baseline tag, and the release branch is created at the root when named`
3. `feat(status): note a missing release branch and one ahead of its tag`

## Verification

- `make test` before and after each commit.
- Smoke, started on the working branch: `git init -b develop`, four
  conventional commits, `pk init --release main`; `git branch -v` shows `main` at the
  root and `develop` checked out with the policy on top; `pk status` prints
  `tag: none yet` and no note; `pk changelog --dry-run` generates `v0.1.0`
  with all five commits; `pk release` from develop fast-forwards `main`
  and tags; `pk status` then shows the tag and no note.
- Smoke, fresh: `git init -b main`, `pk init`; `main` and `develop` on the
  configure commit, no tag, step list without `tag`, no note.
- Smoke, adoption: a clone of a repository with real history and no
  tags; `pk init` on its default branch prints the count and the
  `git tag v0.0.0 <sha>` command; `pk changelog --dry-run` before the tag
  opens with the all-commits line; run the command verbatim; the dry run
  now finds nothing after the tag and says so.
- Smoke, hand-configured: `.pk.json` copied in on a branch, no `main`;
  `pk status` prints the missing-release-branch note; running its command
  verbatim creates `main` at the root; the note is gone.
- Smoke, moved release branch: after a release, commit on `main`;
  `pk status` says `main has 1 commits not in a release`.
- Must fail: `git init -b develop`, one commit, plain `pk init`; usage
  error naming `--release main`, nothing written.
- `claude plugin validate . --strict` and `make docs` leave the tree as
  expected.
