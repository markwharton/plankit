# pk init carries the bootstrap through: commit, tag, develop, brief

## Context

Configuring a brand-new repository with plankit 1.1.0 deadlocks. `pk init`
writes `.pk.json` with `guard.branches: ["main"]` on the checked-out branch,
then hints "commit .pk.json". The guard hook denies that commit because
`main` is now protected, so the session cannot take the step init just
recommended. The baseline tag never appears either: init only tags when a
commit already exists, and it refuses a second run. On a fresh repository nothing creates a
working branch to move to, so the session is stuck on `main`. The developer
ends up typing
`git add .pk.json && git commit -m "chore: configure plankit" && git tag v0.0.0`
and `git switch -c develop` by hand every time.

Six gaps were recorded from one bootstrap session. All six are real; none is
a bug in an individual piece. They are the pieces combining. The fix follows
plankit's own rule that pk configures and the person does not hand-write:
`pk init` performs the whole bootstrap itself, through git directly (so the
guard hook, which watches the agent's shell tool, never sees it), and ends by
printing the brief the session would otherwise miss.

Two adjacent findings from the same session ride along: every command page
should tell a session about the universal flags (the session reads one page,
and `--project-dir` is how init targets another directory), and the
`.claude/settings.json` allow rule that made `pk ship` seamless has no home
in the developer-facing docs.

The test suite passes on `develop` at the start of this session.

## Decisions taken

- `pk init` commits `.pk.json` and tags `v0.0.0` itself; `--no-commit` opts out. It never pushes by default; `--push` opts in, the same shape as `pk preserve --push`.
- The release model is unchanged: releases come from whatever branch is
  checked out, anything but the protected one. No key or constant names a
  development branch, and the brief, guard, and release wording stays as it
  is. `pk init` creates `develop` as a starting working branch, and switches
  to it, only when the release branch is the only local branch.
- Guard code does not change its decision logic. The deadlock is closed at
  init; a `--no-commit` run gets a hint that names the constraint.
- Homebrew tap automation, release cadence, a cheatsheet, and plugin
  auto-update are follow-ups, listed at the end, not part of this change.

## Changes

### 1. Untouched on purpose

`internal/brief/brief.go:127` ("Work on the development branch"), the guard
deny and ask reasons at `internal/guard/guard.go:114,118`, and the
`git switch -c develop` hints in `internal/release/release.go:71` and
`internal/changelog/changelog.go:105` keep their wording. The development
branch is any unprotected branch; the hints already show `develop` as the
example to start one, and init now runs that same command.

### 2. git helpers, `internal/git/git.go`

Small additions beside `CreateTag`, each a thin exec:

- `CommitPaths(dir, message string, paths ...string) error`: `git add -- <paths>` then `git commit -q --only -m <message> -- <paths>`. Commits only the named paths, so a dirty tree elsewhere is untouched. Works on an unborn branch (creates the root commit).
- `CreateBranch(dir, name string) error`: `git switch -c <name>`.
- `BranchExists(dir, name string) bool`: `git rev-parse --verify -q refs/heads/<name>`.

### 3. `pk init`, `internal/repo/repo.go`

New flags: `--branch <name>` (the working branch init creates when the
release branch is the only one; default `develop`; a flag, not a key, since
`.pk.json` does not name where work happens), `--no-commit` (skip the commit
of `.pk.json`), and `--push` (push what init created to origin, off by
default, the same opt-in `pk preserve --push` has; `pk release` and `pk ship`
always push and carry no flag). `--no-baseline` stays. `--push` with
`--no-commit` is a usage error (exit 1): the push exists to publish the
bootstrap commit. Flags stay alphabetical in the block as `cli.FlagBlock`
renders them.

Sequence, each step through the existing `do(label, fn)` so `--dry-run`
lists every step it would take:

1. Write `.pk.json` (as now).
2. Unless `--no-commit`: commit it as `chore: configure plankit`. Label `commit`.
3. Unless `--no-baseline`, when no `v*` tag exists and HEAD now resolves: tag `v0.0.0`. Label `tag v0.0.0`. (With `--no-commit` on an empty repo, HEAD does not resolve and the tag is skipped, as now.)
4. Create the `--branch` branch (default `develop`) when all hold: the checked-out branch is the release branch, HEAD resolves, and `git.HasOtherLocalBranch(root, release)` is false. `git switch -c <name>`. Label `branch <name>`. A repository that already has another local branch keeps its own model; init says nothing about branches then. `--branch` equal to `--release` is a usage error (exit 1).
5. With `--push`: one `git push --atomic --set-upstream origin <refs>` where refs are the release branch, the tag when step 3 created one, and `develop` when step 4 created it. Label `push origin <refs>`. Before writing anything, `--push` checks that `origin` exists (`git remote get-url origin`) and fails with exit 2 naming the fix ("add an origin remote, or run without --push") so a run never stops half-done. Add `git.PushRefs(dir string, refs ...string) error` and `git.HasRemote(dir, name string) bool` beside the other helpers. pk execs git, so guard's push policy does not see it, exactly as `pk preserve --push` today.

Dry-run must evaluate step 3, 4, and 5 conditions as if the earlier steps
ran, so the preview names everything a real run would do on an empty repo.
Compute `willHaveCommit := git.HasCommits(root) || !noCommit` once and use it
throughout.

Output:

- Text narration moves to stderr. Init now has a commit path, and a command that commits leaves stdout empty (docs/design.md, What every command keeps true). `--format json` stays on stdout.
- Replace the three hand-copied notes (commit convention, breaking markers, "every session is briefed") with the brief itself: after the `plankit configured in ...` line, print `brief.Text(cfg)`. The brief is the source; init becomes another reader of it, and the session that created the repository receives the text it would otherwise never get. Check `internal/brief` does not import `internal/repo` (it imports config, git, hookio, msg, version only).
- Fix the false note at `repo.go:107`: "no baseline tag: the repository has no commits yet" currently also fires for `--no-baseline` on a repository with commits. Print it only when no tag was created and HEAD does not resolve.
- Hints, only when a step was skipped:
  - `--no-commit`: "commit .pk.json yourself: guard blocks a session from committing on <release>; then run pk status". On an empty repo add: "tag v0.0.0 and create develop after that commit: git tag v0.0.0 && git switch -c develop".
  - Develop not created because another local branch exists: no hint; the repository already has a working branch.
- JSON: keep `root`, `release`, `created`, `baseline`, `dryRun`; add `committed` (bool), `branch` (the development branch created, or empty), and `pushed` (the refs pushed, or empty).

Tests, `internal/repo/repo_test.go`, driven through `cli.RunIO` on `scratch()` repos:

- Empty repo, plain `pk init`: `.pk.json` committed as the root commit with subject `chore: configure plankit`, tag `v0.0.0` on it, checked-out branch is `develop`, `main` exists, stdout empty, stderr contains the brief's first line and "plankit configured".
- Repo with a commit and a dirty tree (an unrelated modified file): init commits only `.pk.json`; the other change stays uncommitted.
- Repo with `main` and an existing `feature` branch: no `develop` created; stays on `main`.
- `--branch dev` on an empty repo: checked-out branch is `dev`; `--branch main` with release `main`: exit 1.
- `--no-commit` on an empty repo: `.pk.json` untracked, no tag, no branch, hint names guard and the commands, exit 0.
- `--no-baseline` on a repo with commits: committed, no tag, and the "no commits yet" note is absent.
- `--dry-run` on an empty repo: touches nothing, lists `.pk.json, commit, tag v0.0.0, branch develop`.
- `--format json` shape.
- `--push` against a bare origin in `t.TempDir()` (the pattern the release tests use): origin has `main`, `develop`, and `v0.0.0` afterwards, and `develop` has its upstream set.
- `--push` with no origin: exit 2, nothing written, message names the fix.
- `--push --no-commit`: exit 1.
- Existing tests updated for stderr and for the commit (`TestInitThenStatusRoundTrips` will see a clean tree and a tag).

### 4. `pk status`, `internal/repo/repo.go`

Two readiness notes on stderr when configured, so a repository that reached
a half-bootstrapped state names its own fix (Gap 2 detection):

- HEAD resolves and no `v*` tag: "no baseline tag: run git tag v0.0.0 to baseline the release machinery" (keep the tag at HEAD form the existing baseline logic uses).
- `release.branch` set, no other local branch besides it: "no working branch besides <release>: to start one, git switch -c develop" (the same hint release and changelog give).

JSON gains nothing; the notes are narration. Tests for both notes.

### 5. Init page, `skills/init/SKILL.md`

Rewrite as a procedure the session can run, in the house style (what you
get, not the mechanism):

- Opening: one run configures the repository: writes `.pk.json`, commits it, tags `v0.0.0`, creates and switches to `develop` when the release branch is the only branch, and prints the brief.
- Procedure section for a session: run `pk init`, or `pk init --project-dir <dir>` when the target is not the working directory; relay its output; nothing else to commit. State that init runs git itself, so guard does not apply to it. State that init creates a starting working branch (`develop`, or `--branch <name>`) only when the release branch is the only one, and a repository with its own working branch keeps it.
- Opt-outs and opt-in in one paragraph: `--no-commit` to review the file first (and that guard then blocks a session from committing it on the release branch, so the developer commits), `--no-baseline`, `--dry-run`; `--push` publishes the bootstrap to origin, and without it the developer pushes `main`, the tag, and `develop` when ready.
- Replace the `pk init --release trunk` usage example with `pk init --release main`. `--release` takes a branch name (`repo.go:28`, asserted by `TestInitFlags`), so the current example guards a branch called `trunk`; it reads like a flow selector because "trunk flow" is pk's name for an empty `release.branch`.
- Keep the second-run refusal and the v0.x paragraph.
- Update `description` and `InitCmd.Summary` ("Configure this repository: write .pk.json and baseline a tag") to say purpose, not features: "Configure a repository for plankit". Reread `docs/architecture.md:4,127` for the init mentions; the sentences still hold.

`skills/status/SKILL.md`: mention the readiness notes. The brief page is unchanged.

### 6. Universal flags on every command page, `tools/docgen/regions.go`

Extend the `flags` region body: after the command's own block (or as the
whole body for a command with none), one generated line:

    Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.

Derived from `cli.UniversalFlags()` so it cannot drift. Skip the line on the
`help` page, which carries the full `universal-flags` block. Because the body
is now never empty, every command page gains a `## Flags` section; `setRegion`
already appends one when absent. Run `make docs`; the CI drift check covers
the rest. Update the region comment at `regions.go:14-26` to say the flags
region carries the pointer.

### 7. Permissions, `README.md` Install section

Permissions are the developer's to grant, so the guidance goes where the
developer reads and never into a shipped skill, which a session would read as
an instruction about its own permissions. The README's `## Install` section
is composed into the plankit.com front page by `tools/docgen/site.go`
(`frontPage`, `readmeSections`), so one paragraph there reaches both readers.

After the `pk init` step, add a short paragraph that opens by saying this
step is optional: plankit works without it, and Claude Code asks before each
pk command instead. Then the block:

```json
{
  "permissions": {
    "allow": ["Bash(pk:*)"]
  }
}
```

in the repository's `.claude/settings.json`: every pk command then runs
without a prompt, and `pk ship` runs when you ask for a release. One sentence
on the finer form for granularity: `Bash(pk status:*)` allows `pk status` and
its flag variants and nothing else; `Bash(pk status)` without `:*` matches the
exact command only. Syntax verified against the Claude Code permissions docs
(https://code.claude.com/docs/en/permissions). A plugin cannot ship
permission rules, so the repository's settings carry them.

The same README paragraph that says `pk init` configures a repository is
updated to what init now does (commits the policy, tags the baseline, starts
a working branch). Run `make site-preview` and look at the front page.

Remove the ship page's `## Permissions` section (skills/ship/SKILL.md:36-42):
it is a shipped page advising a session about allowlists, and its advice
(leave `pk ship` and `pk release` off) contradicts how the tool is used. The
Releasing section stays and remains the gate. Grep-before-done: no
`allowlist` or `settings.json` permission text left under `skills/` except
the v0.x migration step on the overview, which is about removing hook
entries, not permissions.

### 8. Release notes

This is a `feat` on `develop`, so the next release is a minor and needs
`docs/notes/<version>.md` before `pk changelog`. Writing the note is a
release-time step, separate from this change; `pk changelog --dry-run` gives
the version.

## Verification

1. `make test` before touching anything (passing now), and after each of the steps above.
2. `make build && make docs`; `git diff --stat skills internal/help/data` shows only the intended pages; `claude plugin validate . --strict` passes.
3. Smoke test in a scratch directory with `./pk`:
   - `git init -b main t && cd t && ../pk init`: stdout empty; stderr has "plankit configured", the brief text; `git log --oneline` shows one commit `chore: configure plankit`; `git tag` shows `v0.0.0`; `git branch --show-current` is `develop`.
   - `../pk status` in that repo: no readiness notes.
   - `../pk init` again: exit 2, "already configured".
   - A second scratch: `git init -b main`, `../pk init --no-commit`: exit 0, `.pk.json` untracked, no tag, hint names guard and gives the commands.
   - The failing case: in that second scratch, simulate the session: `echo '{"tool_input":{"command":"git commit -m x"}}' | ../pk guard` on `main` still denies.
   - `../pk init --dry-run --format json` in a third empty scratch lists all four labels and writes nothing.
   - A fourth scratch with a bare origin (`git init --bare o.git; git remote add origin ../o.git`): `../pk init --push` leaves `main`, `develop`, and `v0.0.0` on origin (`git ls-remote origin`). The same command in a scratch without origin exits 2 and writes nothing.
4. `pk help init` and `pk help status` read the new pages; `pk help guard` shows the Flags section now present; `pk help ship` no longer has a Permissions section.
5. `make site-preview`; the front page shows the permissions paragraph under Install.

## Out of scope, recorded as follow-ups

- Homebrew tap: the bump workflow opens a PR that the maintainer merges, then cuts the tap's own release by hand. Full automation is in `markwharton/homebrew-plankit` (auto-merge the bump PR after the four-platform test passes, then run its release), not in this repository.
- Build on Saturday, release on Sunday, and any edit to the v1.1.0 release: process, not code. A published release is not rewritten.
- A cheatsheet: a new surface reading existing sources (the index and each page's Usage block). Design it separately after this lands, so it reads the new init page.
- Plugin auto-update: a Claude Code capability, not pk's. The overview's Updating section already gives the commands.
- Guard allowing the first commit on an unborn branch: not needed once init commits; revisit only if `--no-commit` proves common.
- A `.pk.json` key for the plans directory (`preserve.dir`, default `docs/plans`): a dial with its own readers (preserve, protect, status, brief, `internal/paths`, four pages, tests). Its own plan.
- Trunk flow is unreachable from `pk init`: `config.Default` always writes `release.branch`, and an empty `--release` falls back to the checked-out branch. Selecting it takes a hand edit of `.pk.json` today. Whether init should offer it is a separate decision.
