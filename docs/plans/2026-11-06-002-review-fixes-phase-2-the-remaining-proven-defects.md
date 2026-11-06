# Review fixes, phase 2: the remaining proven defects

## Context

A code review of the repository at v1.8.3 is being fixed in phases.
Phase 1 fixed the five defects that could cut a wrong release, lose a
plan, or commit a credential. Phase 2 is the rest of the defects that
were reproduced by running a build: places where pk says nothing, says
the wrong thing, or lets through what its pages say it stops.

1. guard reads a command line two ways. The subcommand is found by
   splitting on spaces, so `git -C "/tmp/my dir" push` and `"git" push`
   get no decision under `push: block`. So do `(git push)`,
   `git add . & git push`, `if true; then git push; fi`, and
   `git --git-dir /x push`. On a protected branch, `git cherry-pick`,
   `git revert`, and `git pull` make commits and pass.
2. `pk brief` says nothing about pushing when `guard.mode` is `off` or
   no branch is protected, while guard still denies `git push`.
3. `pk release` on a detached HEAD prints `Error:  does not exist on
   origin` with an empty branch name.
4. `pk pin` reports any read error as a missing file: a directory
   target prints "does not exist; nothing to pin" and exits 0.
5. `pk protect` compares paths as text, so on a filesystem that ignores
   case `DOCS/PLANS/<file>` reaches a preserved plan.
6. `pk preserve` hides its failures: a typed run with a pending plan
   under the minimum size prints nothing and exits 0, and a failed
   commit exits 0 for a typed run and for the automatic hook alike.
7. guard and protect exit 0 on a payload that does not parse, where
   `docs/design.md` says exit 1, and four comments and two lines of
   `.claude/CLAUDE.md` still say every hook always exits 0.

The outcome: eight local commits on `develop`, the suite green before
and after each, and each reproduction rerun against the new build.
Nothing is pushed and no release is cut.

## Fix 1: guard reads a command line once

Every policy needs the same reading of a command line: which simple
commands it holds, what program each runs, and its words. Today
`gitSubcommand`, `bumpOverride`, and `commitMessages` each skip the
prefix words their own way, and only two of them honour quotes.

In `internal/guard/guard.go`:

- `invocations(command) []invocation` is the one reader. An
  `invocation` is a program's base name and the words after it.
  It splits with `splitShellCommands`, reads words with `shellWords`
  (already quote-aware), and drops the words that can precede a
  program: environment assignments, `command`, and the shell words
  `if`, `then`, `elif`, `else`, `while`, `until`, `do`, `!`, and `{`.
- `splitShellCommands` also splits on a single `&` that is not part of
  a redirection (`2>&1`, `&>`), and on `(` and `)` outside quotes, so a
  subshell and a command substitution are read as the commands inside.
- `gitCall(invocation)` returns the git subcommand and its arguments,
  skipping global options. The options that take a separate value are
  `-C`, `-c`, `--git-dir`, `--work-tree`, `--namespace`, and
  `--config-env`.
- `isGitMutation`, `isGitPush`, `hasBreakingCommit`, and `bumpOverride`
  take the invocations `run` read once. `gitSubcommand` and the prefix
  walks inside `bumpOverride` and `commitMessages` are deleted.

Not read, by design: a command run through `sudo`, `sh -c`, or `eval`.
guard is a guardrail against an agent following its defaults, and
these are not what an agent writes by default; the guard page says so
in one sentence under "What guard is".

`skills/guard/SKILL.md`: the sentence listing what the hook recognises
gains `&`, subshells, quoted words, and shell keywords.

Tests in `internal/guard/guard_test.go`: the existing tables move to
the new functions; new rows for each form in item 1 above, for
`git commit -m "x" 2>&1` (one command, not two), and for
`git commit -m "feat(scope): x"` (parentheses inside quotes).

Commit: `fix(guard): one reading of a command line for every policy`

## Fix 2: four more git commands are mutations

`am`, `cherry-pick`, `pull`, and `revert` each put commits on the
branch that is checked out, which is what the branch policy exists to
stop on a protected branch. The list becomes `am`, `cherry-pick`,
`commit`, `merge`, `pull`, `push`, `rebase`, `reset`, `revert`.

- `internal/guard/guard.go`: the list becomes one package-level value
  that `isGitMutation` reads.
- `skills/guard/SKILL.md`: the opening sentence lists the nine. A test
  reads the page and fails when a subcommand in the list is missing
  from it, the way `TestPageNamesThePointerFile` holds the preserve
  page, so the page cannot fall behind the code.
- `git pull --rebase=false` moves from the "not a mutation" rows of
  `TestIsGitMutationAcrossChains` to the mutation rows.

This is new behaviour that the guard page did not promise, so it is a
`feat`: the next release is a minor, and a release note in
`docs/notes/` is written before that release ships.

Commit: `feat(guard): am, cherry-pick, pull, and revert count as mutations`

## Fix 3: the brief states the push policy and the release flow

`guard.push` applies on every branch whatever `guard.mode` is, and the
release flow exists whether or not a branch is guarded. Both sentences
sit inside the branch paragraph today and vanish with it.

`internal/brief/brief.go`, `Text`: the paragraph is built from three
sentences, each from its own dial: the protected branches (`guard.mode`
and `guard.branches`), pushing (`guard.push`), and the release flow
(`release.branch`, always present). The wording of each sentence is
unchanged.

`skills/brief/SKILL.md`: "A dial set to off drops its paragraph"
becomes "its sentence".

Tests, in `TestTextFollowsTheDials`: `guard.mode` off keeps "do not
push" and the flow sentence and drops "protected"; no protected
branches with `guard.push` ask keeps "Pushing prompts for
confirmation."; `guard.push` off drops the push sentence.

Commit: `fix(brief): the push policy and the release flow are stated whatever guard.mode is`

## Fix 4: a release has a branch

A release commit on a detached HEAD cannot be released: there is no
branch to check against origin, to merge, or to push. `pk release`
discovers that with a garbled message, and `pk changelog` skips its
branch checks and makes a release commit that is then stranded.

- `internal/release/release.go`: a detached HEAD is refused first, by
  name, with a hint to switch to the branch that holds the commit.
- `internal/changelog/changelog.go`: the same refusal for a real run.
  `--dry-run` still prints the section, since it writes nothing.

Tests: `TestRefusals` in both packages gains the detached case; the
changelog dry run on a detached HEAD still succeeds.

Commit: `fix(release): a detached HEAD is refused by name, in changelog and release`

## Fix 5: pin tells a missing file from an unreadable one

`internal/release/pin.go`: `pinVersion` and `pinVersionNamed` read the
target through one helper that returns "missing" only for a file that
does not exist. Any other read error is returned, and `runPin` reports
it as a state error. A missing file stays a no-op, which is the
contract release hooks rely on.

`skills/pin/SKILL.md`: one clause, a file that exists and cannot be
read is an error.

Tests in `internal/release/pin_test.go`: a directory target exits 2
and names the path; the missing-file test is unchanged.

Commit: `fix(pin): a file that cannot be read is an error, not a missing file`

## Fix 6: protect recognises the plans directory by identity

Comparing spellings cannot say whether two paths are one directory;
the filesystem can.

`internal/protect/protect.go`, `isUnderPlansDir`: the existing check
stays, because a plans directory that does not exist yet has no
identity and its first file must still be refused. Added to it: when
the plans directory exists, a write is under it if any existing
directory above the target is the same directory on disk
(`os.SameFile`). That covers another spelling on a filesystem that
ignores case, with no test of the operating system.

A plans directory that does not exist yet, reached by another spelling
on such a filesystem, is still not recognised. It holds no records.

`skills/protect/SKILL.md`: the sentence about symlinks and Windows
says the directory is recognised by its identity on disk.

Test in `internal/protect/protect_test.go`: another spelling of an
existing plans directory is denied. The test asks the filesystem
whether it ignores case and skips when it does not, so it runs on a
Mac and is skipped on CI's Linux.

Commit: `fix(protect): the plans directory is recognised by its identity on disk`

## Fix 7: preserve reports its failures

`preserve` in `internal/preserve/preserve.go` reports every failure as
a stderr line and returns nothing, so a typed run exits 0 whatever
happened and the automatic hook's message is never shown: Claude Code
shows a hook's stderr only when it exits non-zero.

- `preserve` returns an error. A refusal (the ignored directory) is a
  state error; a git command that fails is an internal error, as it is
  in `pk changelog`.
- One function turns that error into the exit for the shape of the
  run. A typed run (`pk preserve`, `--plan`) returns it, so the frame
  prints it and exits 2 or 3. The automatic hook prints it with its
  `pk preserve:` prefix and exits with `hookio.ExitReport`, the code
  the hooks already use for a policy file that fails to load: shown,
  never blocking.
- In `run`, a typed run whose pending plan is under the minimum size,
  or cannot be read, is a state error naming the plan, with a hint
  that `pk preserve --discard` drops the pointer. The size message is
  the one `runNamed` already prints, built in one place. The hook
  still declines a short plan in silence.

`skills/preserve/SKILL.md`: the sentence about a short plan says a
typed run is told; one sentence says a failed preserve exits non-zero.

Tests in `internal/preserve/preserve_test.go`: the failed-commit test
asserts exit 1 for the hook and a non-zero exit for `--plan`; a typed
run with a short pending plan exits 2, names it, and keeps the
pointer; the ignored-directory test asserts exit 1.

Commit: `fix(preserve): a failed preserve is reported, and a typed run exits non-zero`

## Fix 8: one exit contract for every hook

`docs/design.md` states it: a hook exits 0 when it acts or declines,
reports a problem it cannot act on with exit 1, and never exits 2.

- `internal/hookio/hookio.go`, `ReadInput`: an empty payload is no
  payload, `io.EOF`, the same answer as a terminal or a closed stdin.
- guard, protect, brief, and preserve: a payload that does not parse is
  reported and exits with `hookio.ExitReport`; no hook acts on it.
  Today guard and protect exit 0, and brief and preserve treat it as a
  typed run, which for preserve would commit the pending plan. No
  payload behaves as it does now in all four.
- The comments that say "always exit 0" in `internal/hookio/hookio.go`,
  `internal/guard/guard.go`, and `internal/brief/brief.go`, and the two
  lines in `.claude/CLAUDE.md` ("every hook exits 0", "exit 0 always"),
  are rewritten to the contract.

Tests: `TestMalformedPayloadFailsOpen` in guard becomes the exit 1
case; the same case is added for protect, brief, and preserve (which
must leave the pointer alone); `ReadInput` on an empty reader is
`io.EOF`.

Commit: `fix: a payload that does not parse is reported with exit 1 by every hook`

## Where each fact lives after this phase

- The reading of a command line is `invocations` in guard; the four
  policies are given its result. Check: a search for `strings.Fields`
  in `internal/guard` shows no hits.
- The mutation list is one value in guard; the page is held to it by a
  test.
- The brief's three sentences each read one dial.
- "A release needs a branch" is refused in the two commands that make
  and publish a release; sharing their other pre-flight checks is
  phase 4.
- A pin target is read by one helper.
- The plans directory is recognised in `isUnderPlansDir` only.
- A preserve failure becomes an exit code in one function; the
  minimum-size message is built in one place.
- "No payload" is `io.EOF` from `ReadInput` and nothing else; the exit
  for a problem a hook cannot act on is `hookio.ExitReport`.

## Verification

- `make test` and the docgen tests before the first change and after
  each of the eight, with the output shown.
- Each new test is run and seen to fail before its fix, then pass.
- `gofmt -l cmd internal tools` prints nothing; `make docs` leaves no
  diff once each page change is committed.
- Smoke test against a fresh build, in scratch repositories whose bare
  origin is made by cloning, so nothing is pushed:
  1. Each command form in item 1 gets guard's deny under `push: block`,
     and `git cherry-pick`, `git revert`, `git pull` are denied on a
     protected branch.
  2. The case that must fail: `git status`, `echo "git push"`, and
     `sudo git push` still get no decision.
  3. `pk brief` with `guard.mode` off names the push policy.
  4. `pk release` and `pk changelog` on a detached HEAD refuse by name,
     exit 2.
  5. `pk pin --file <directory>` exits 2; a missing file exits 0.
  6. `pk protect` denies another spelling of the plans directory.
  7. `pk preserve --plan` with a failing commit exits non-zero; the
     hook shape exits 1; a typed run with a short pending plan exits 2.
  8. `echo 'not json' | pk guard` exits 1; an empty stdin exits 0.
- Reread the `Summary` lines, the guard, brief, pin, protect, preserve,
  release, and changelog pages, and `docs/architecture.md` against the
  new behaviour before each commit; correct any sentence the change
  makes false in the same commit.

## Later phases, each with its own plan

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
