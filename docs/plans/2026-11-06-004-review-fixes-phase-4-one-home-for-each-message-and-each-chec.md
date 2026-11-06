# Review fixes, phase 4: one home for each message and each check

## Context

A code review of the repository at v1.8.3 is being fixed in phases.
Phases 1 to 3 fixed the defects and the gaps around the code. Phase 4
is the duplication inside the main module: checks that two commands
each carry a copy of, and messages written more than once.

The unused message catalogue in `internal/msg` is deleted in this
phase. The condition for deleting it is that the messages left in the
code do not repeat themselves: if a catalogue were ever adopted, moving
the messages into it should be a mechanical switch, one entry per
message, not a reconciliation of the same sentence in different words.
A survey of the 191 message sites in `internal/` found where that
does not hold today:

- "not a git repository: %s" is written at nine sites.
- The hint for an unconfigured repository has three wordings ("run pk
  init to configure this repository", "to configure it", "to write
  .pk.json"), and `pk brief` has its own text for the refusal.
- The trunk-flow refusal, its hint, "%s does not exist on origin" and
  its hint are each copied between `pk changelog` and `pk release`.
  The advice to start a working branch has three variants.
- A failed git command is named twice: commands wrap it as "git
  ls-remote failed: %v" (fifteen wrappers of this kind) around an
  error that already begins "git ls-remote:".
- The two lifecycle-hook shapes name their hooks differently
  ("postVersion hook failed", "pre-release hook failed").
- Smaller repeats: "write error: %v" in three hooks, "failed to read
  plan", "failed to switch back", "read .claude/settings.json".

The scope first listed for this phase also held docgen's copies of
main-module code and a set of leftovers. Those move to a phase of
their own, so that this plan stays one subject. Nothing is dropped:
the later phases are listed at the end.

The outcome: twelve local commits on `develop`, the suite green before
and after each. Nothing is pushed and no release is cut.

## 1. A failed git command names itself once

`internal/git/git.go`: `Error.Error()` renders "git <command> failed:
<detail>". Callers that wrapped it only to say which command failed
return the error as it is: the "git ... failed: %v" wrappers in
`changelog`, `release`, `preserve`, and `CheckCleanTree`, and the
wrappers that add nothing else ("failed to count commits", "failed to
read git log", "failed to list tags"). A wrapper that adds a fact the
command does not carry stays ("failed to switch to %s", "cannot
undo").

Message text changes from "git ls-remote failed: git ls-remote: fatal"
to "git ls-remote failed: fatal".

Commit: `refactor(git): a failed git command names itself once`

## 2. One refusal for "not a git repository" and for "not configured"

`internal/cli` gains two functions, in a new file beside the context:

- `Root(dir) (string, error)`: the repository root, or the one state
  error for a directory that is not in one. The nine sites call it;
  `pk init` adds its own hint to the same error.
- `Policy(root) (*config.PkConfig, error)`: the loaded policy file, or
  the one state error for a repository without one, with the one hint
  ("run pk init to write .pk.json"), or the file's own error.
  `changelog`, `release`, the typed runs of `preserve`, and the typed
  run of `brief` call it. `cli` may import `config`: `config` imports
  nothing of pk's.

The hooks keep calling `config.Load`, because they must tell "not
configured" (stay silent) from "does not load" (report). `pk status`
keeps its own report of an unconfigured repository, and takes the hint
text from the same place.

Commit: `refactor(cli): one refusal for a missing repository and a missing policy file`

## 3. changelog and release share their pre-flight checks

`internal/changelog` (which `release` already imports) holds the
checks both commands run, in one file:

- the trunk-flow check: the branch must be origin's default branch;
- the on-origin check: the branch must exist on origin;
- the advice to start a working branch, one text, used by `changelog`,
  `release`, and the note `pk status` prints.

`release` keeps the checks only it runs (behind origin, the release
branch, divergence).

Commit: `refactor: changelog and release share their pre-flight checks`

## 4. An unreachable origin is a precondition

A git command that had to ask origin and failed (`ls-remote`, `fetch`,
and the push `pk init --push` makes) is a state error, exit 2, with one
message and one hint: origin did not answer, check the network, your
access, and the remote's URL, then run again. One function in
`changelog` builds it; the on-origin check from step 3, `release`'s
fetches, and `init --push` use it.

The reason for exit 2: exit 3 says pk failed, and a remote that cannot
be reached is the repository not being ready to release from. Two
origin failures are already exit 2 (a rejected release push, and the
failed tag fetch in `pk changelog`); this makes the rest agree, and
only an exit 1 or 2 error can carry a hint.

Tests: `changelog` and `release` with origin's URL pointed at nothing
exit 2 and print the hint.

Commit: `fix: an unreachable origin is a precondition, exit 2 with a hint`

## 5. One answer to "which branch is checked out"

`git.CurrentBranch` returns the branch name, and "" for a detached
HEAD, read from git's exit status; an unborn branch is still named.
The three other ways of asking go: `branch --show-current` in
`changelog` (twice) and `release`, and `symbolic-ref -q HEAD` in
`pk changelog --undo`.

Behaviour that changes: on a detached HEAD, `pk status` says so where
it printed "HEAD" as a branch name, and `pk init` refuses where it
took "HEAD" as the branch to guard.

Commit: `refactor(git): one answer to which branch is checked out`

## 6. A breaking marker is read by one parser

`internal/changelog` exports the two questions its own parser answers:
does a subject carry `!`, and does a message carry a `BREAKING CHANGE`
footer. `guard` asks those instead of keeping `breakingSubjectRe` and
`hasBreakingLine`, so guard asks about exactly the commits changelog
will count as breaking.

Commit: `refactor(guard): a breaking marker is read by changelog's parser`

## 7. Lifecycle hooks run through one function

`internal/hookio` gains `RunHook`: it narrates the hook, runs it with
`RunScript`, and returns the one state error, "<name> hook failed",
where the name is the key in `.pk.json` (`postVersion`, `preCommit`,
`preRelease`, `prePush`). The four call sites in `changelog` and
`release` use it.

Message text changes: "pre-release hook failed" becomes "preRelease
hook failed", and the two hooks of `pk changelog` are narrated the way
`pk release` narrates its own.

Commit: `refactor: lifecycle hooks run through one function`

## 8. One way for a hook to report a problem

`internal/hookio` gains `Report(w, hook, err) error`: it writes the
line with the hook's prefix and returns the exit that makes Claude
Code show it. The eight sites in guard, protect, brief, and preserve
that pair `msg.Hookf` with `cli.Silent(hookio.ExitReport)` use it, and
so do the three "write error" sites.

Commit: `refactor(hooks): one way to report a problem a hook cannot act on`

## 9. The message catalogue is removed

`internal/msg/msg.go`: `ID`, `Def`, the catalogue map, `Register`,
`IDs`, and `Lookup` go, with their test. The package comment is
rewritten to what the package is: the severity-prefixed helpers.

Commit: `refactor(msg): the unused message catalogue is removed`

## 10. Policy is checked when the file loads

`internal/config/config.go`, `Validate`: a `changelog.versionFiles`
entry whose `type` is not `json`, and a `release.branch` that begins
with `-`, fail the file at load with the key named, as every other
wrong value does. The checks at the point of use in `changelog` and
`release` go.

Behaviour that changes: these two were reported when a release was
attempted (one of them as a usage error, exit 1); now every command
and hook reports them when the file loads.

Commit: `fix(config): a version file's type and the release branch's name are checked at load`

## 11. Errors that were dropped are reported

- `changelog`: a `CHANGELOG.md` that exists and cannot be read is an
  error; only a missing file starts a new changelog.
- `pk status`: a failed `git status` is an error, where it reported a
  dirty tree.
- `guard`: a failure to read the current branch is reported through
  `hookio.Report`, where it returned in silence.

Commit: `fix: read failures that were dropped are reported`

## 12. pin walks the target once

`internal/release/pin.go`: `pinVersion` and `pinVersionNamed` share
one function that reads the target, rewrites the first matching line,
and writes it back; each supplies only its matcher. `asNoPin` is
replaced by `errors.As`.

Commit: `refactor(pin): one walk over the target's lines`

## The remaining repeats

After the twelve, the survey is run again. The smaller repeats listed
in the context are each folded into one place in the commit that
touches their file: "failed to read plan" (step 2), "failed to switch
back" (step 3), "read .claude/settings.json" (step 2), and the unknown
span error in the roff writer (step 9, the other dead-code commit).
What may remain is format-only text with no words of its own ("%s:
%v").

## Where each fact lives after this phase

- Which git command failed: `git.Error`.
- Not a repository, not configured: `cli.Root`, `cli.Policy`.
- The pre-flight checks two commands share, and origin not answering:
  one file in `changelog`.
- The checked-out branch: `git.CurrentBranch`.
- What a breaking marker is: `changelog`.
- Running a lifecycle hook: `hookio.RunHook`. A hook reporting a
  problem: `hookio.Report`.
- What the policy file may contain: `config.Validate`.

## Verification

- `make test` before the first change and after each of the twelve,
  with the output shown. A step that only moves code needs the suite.
- A step that changes behaviour (4, 5, 10, 11) has a test that is run
  and seen to fail first.
- The survey of message sites is run again at the end and its list of
  text written at more than one site is shown.
- Smoke test against a fresh build, in scratch repositories whose bare
  origin is made by cloning, so nothing is pushed:
  1. Outside a repository, and in one without `.pk.json`, each of
     `pk changelog`, `pk release`, `pk brief`, and `pk preserve` prints
     the same refusal and hint as the others, exit 2.
  2. With origin's URL pointed at nothing, `pk changelog` and
     `pk release` exit 2 with the hint.
  3. A failing `preRelease` hook prints "preRelease hook failed".
  4. A `.pk.json` with a `versionFiles` type of `yaml` fails `pk status`
     with the key named.
  5. The case that must fail: a `.pk.json` with type `json` and a
     normal release branch still loads, and a full `pk ship --dry-run`
     passes.
- `gofmt -l cmd internal tools` prints nothing; `make docs` leaves no
  diff once each page change is committed.
- Reread the `Summary` lines, the affected pages, and
  `docs/architecture.md` before each commit; correct any sentence the
  change makes false in the same commit.

## Later phases, each with its own plan

- Phase 5, docgen reads the main module, and leftovers: the IR types,
  command-page detection and index order, docgen's semver, tag, URL,
  and frontmatter copies; then the dead code and stale comments (the
  discarded sidebar list, orphaned comments, the scaffolding
  vocabulary, the unused Makefile target name, the local `min`, the
  doubled protect block in `hooks.json`).
- Phase 6, tokens: named mode values, the help style table, narration
  through `msg`, guard's reason prefixes, the stylesheet. Decision
  there: which sizes and radii the stylesheet keeps.
