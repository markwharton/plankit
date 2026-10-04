# The shim falls back to pk on PATH, and the brief says when pk is behind the plugin

## Context

The plugin's hooks run `"${CLAUDE_PLUGIN_ROOT}"/bin/pk`, a shim whose
one move is to exec the platform binary beside it and exit 3 if it is
absent. The binaries are release assets, carried only by the archive the
plankit.com marketplace points at. Every other way the plugin files can
arrive, the GitHub repository as a source, a directory listing that tracks
a branch, or a `--plugin-dir` checkout before `make bin-local`, gives a
plugin whose hooks all fail, even on a machine that has pk from Homebrew
or `go install`. The fix is one more branch in the shim: no sibling
binary, use the `pk` on PATH.

That creates one gap the archive never had: the pk that runs can be a
version behind, or ahead of, the plugin's skills. The hooks must not open
a socket to check, and do not need to: Claude Code's marketplace update
already tells the developer when the plugin is behind the world, and the
plugin's own version sits in `.claude-plugin/plugin.json` next to the
shim. So the shim tells pk where the plugin is, and the brief, which
already opens every session with the version, compares the two and says
when they differ. Both are local reads. Neither command touches the
network.

Decisions taken here:

- The shim passes the plugin root only when it falls back. When the
  sibling binary runs, the versions agree by construction, and a
  `make bin-local` development binary beside `--plugin-dir` would
  otherwise report itself as ahead of `plugin.json` in every session.
- The brief is the only surface. A `pk` typed at a terminal is not the
  plugin's dispatch, so there is nothing for `status` or `version` to
  compare. The skew only exists for a pk the shim dispatched, and the
  brief is what runs at every session start through that shim.
- The update hint is `go install github.com/markwharton/plankit/cmd/pk@latest`,
  the one form plankit's hints use; the README lists the others.

## What exists to build on

- `bin/pk` (26 lines, POSIX sh): maps OS and arch, builds the sibling
  path, `exec`s it, exits 3 with "not found; run 'make bin-local' ... or
  reinstall the plugin" otherwise. `bin/pk.cmd` (5 lines) runs
  `%~dp0pk-windows-%ARCH%.exe` with no existence check.
- `internal/version/semver.go`: `Semver{Major, Minor, Patch, PreRelease,
  Build}`, `ParseSemver(s) (Semver, bool)` strict with optional `v`,
  `Bump`, `String`. No compare.
- `internal/version/version.go:23` `Version()`: stamp, else build info,
  else `dev`.
- `internal/brief/brief.go:89` `Text(cfg *config.PkConfig) string`, whose
  first sentence is `plankit %s is configured in this repository.` from
  `version.Version()`. Two callers: the brief hook and `pk init`
  (`internal/repo/repo.go:222`).
- `internal/brief/brief.go:36` `run`: hook shape reads the payload, fails
  open on `ErrNotConfigured`, reports other errors with `msg.Hookf` and
  exit 1. Tests in `brief_test.go` use `scratch(t, configured)` and
  `cli.RunIO` with a payload reader; the hook test for `CLAUDE_PROJECT_DIR`
  shows how an environment variable is set for a run.
- `internal/jsonsplice.Get(content, "version")` reads one key of a JSON
  object without a schema; `plugin.json` is Anthropic's schema, not pk's,
  so it is not decoded strictly.
- `cmd/pk/main_test.go:93` `TestHookWiringMatchesRegisteredCommands` is
  the only test that looks at the shim, by its command text. Nothing runs
  `bin/pk`.
- Nothing in Go reads `CLAUDE_PLUGIN_ROOT`, and nothing reads
  `plugin.json` at runtime.

## Changes

### 1. The shim falls back to pk on PATH, `bin/pk` and `bin/pk.cmd`

In `bin/pk`, when the sibling binary is not executable, walk `$PATH`
entry by entry for an executable `pk`, skipping any entry that resolves
to the shim's own directory: Claude Code puts the plugin's `bin/` on the
Bash tool's PATH, so a naive `command -v pk` can find this very shim and
exec itself forever. The first other `pk` is run with
`PK_PLUGIN_ROOT` exported as the shim's parent directory:

```sh
PK_PLUGIN_ROOT="$root" exec "$found" "$@"
```

No `pk` anywhere: the existing exit-3 message, extended by one clause,
"or install pk (go install github.com/markwharton/plankit/cmd/pk@latest)".
The sibling path is unchanged and still tried first, so the archive
behaves as today.

In `bin/pk.cmd`, add the existence check it lacks, then the same
fallback: `for %%p in (pk.exe) do set "found=%%~$PATH:p"`. A PATH search
for `pk.exe` can never match `pk.cmd`, so no guard is needed there. Set
`PK_PLUGIN_ROOT` to `%~dp0..` before running it, and give the no-pk case
the same message and `exit /b 3`.

### 2. `Compare` on `Semver`, `internal/version/semver.go`

`func (v Semver) Compare(o Semver) int` by SemVer 2.0 precedence: major,
minor, patch; a pre-release ranks below its release; pre-release
identifiers compare numerically when both are numeric, else as strings,
with the shorter list lower; build metadata is ignored. Table test with
the spec's own ordering examples plus `1.4.0` against
`1.5.0-0.20260930220111-f79562029de1+dirty`, which is what a source build
reports on Go 1.24 and later.

### 3. The brief compares, `internal/brief`

A new file `plugin.go` in the brief package with one reader:

```go
// PluginVersion returns the version of the plugin whose shim dispatched
// this pk, read from <PK_PLUGIN_ROOT>/.claude-plugin/plugin.json, and ""
// when no shim fell back to this binary.
func PluginVersion(getenv func(string) string) (string, error)
```

It uses `jsonsplice.Get` for the one key. An unset variable is "" and no
error. A set variable whose manifest is missing or unreadable, or whose
version is not a string, is an error naming the file: the shim said where
the plugin is, so the plugin not being there is a real problem.

`Text` gains the parameter: `Text(cfg *config.PkConfig, pluginVersion
string) string`. Both callers pass what `PluginVersion(os.Getenv)` returns;
in the hook shape an error goes through `msg.Hookf` and exit 1 like a
broken policy file, in the typed shape it is a state error, and `pk init`
reports it as a state error before printing nothing. The first sentence
becomes, by case:

- `pluginVersion` empty, or equal to `version.Version()` after parsing:
  unchanged, `plankit 1.4.0 is configured in this repository.`
- binary behind the plugin: `plankit 1.5.0 is configured in this
  repository. This session's pk is 1.4.0, from the PATH and behind the
  plugin: update it with go install github.com/markwharton/plankit/cmd/pk@latest.`
- binary ahead of the plugin: `plankit 1.4.0 is configured in this
  repository. This session's pk is 1.5.0, from the PATH and ahead of the
  plugin: /plugin update plankit@plankit brings the skills level.`
- either side fails `ParseSemver` (a `dev` build): the unchanged
  sentence, naming the binary's version as today. The page says dev
  builds are not compared.

The version named first is the plugin's when it is known, because that
is what the skills in the session describe; today's doc comment on `Text`
says the first sentence is the plugin's version, and this keeps it true
in the fallback case too.

### 4. Source builds report `dev` plus the commit again, `internal/version/version.go`

`fromBuildInfo` checks `Main.Version` first and falls through to the VCS
settings only for `(devel)`. Go 1.24 and later fill `Main.Version` with a
pseudo-version in a source checkout, so the VCS branch is dead and
`make build` reports `1.5.0-0.<date>-<sha>+dirty` against the design doc,
the version page, and the Makefile comment, all of which say `dev+<sha>`.
The distinguishing fact is structural, not a string: a build from a
checkout carries `vcs.revision`, a `go install` from the module cache
does not. Reorder: VCS settings first, giving `dev+<sha>` and `.dirty`;
then `Main.Version` when it is set and not `(devel)`, which is the
`go install` case; then `dev`. The stamp stays first of all. No version
string is pattern-matched.

Test: the existing table in `version_test.go` gains a case whose crafted
build info has both a pseudo-version in `Main.Version` and `vcs.revision`,
expecting `dev+<sha>`, and keeps the `go install` case with a module
version and no VCS settings. This is a `fix(version)` commit of its own,
before the feature, so the feature's dev builds report what the page says.

### 5. Pages

- `skills/brief/SKILL.md`, "What it says": the first sentence names the
  plugin's version; when the plugin's shim found no binary beside it and
  ran the pk on PATH, the sentence also names that pk's version and which
  side is behind, with the command that levels them. Dev builds are not
  compared.
- `skills/overview/SKILL.md`, Updating: one sentence that a pk installed
  with `go install` or Homebrew is updated separately, and the brief says
  when it is behind the plugin.
- `README.md`, Install: after the marketplace install, one sentence that
  the plugin runs its bundled pk, and with pk already on PATH from
  `go install` or Homebrew it runs from any source, so the GitHub
  repository works as a marketplace source too.
- `docs/architecture.md:11-13`: "a shim that finds the pk binary for the
  platform, or the pk on PATH".
- `docs/design.md` "Where the code is", `bin/`: the fallback in one
  clause. "What every command keeps true" already says no network.
- `CLAUDE.md`, "The plugin is the repository": `hooks/hooks.json` wires
  brief, guard, protect, and preserve; brief is missing from the list
  today.
- `docs/notes/v1.5.0.md`: this is a `feat`, so the release is a minor.
  Two paragraphs: the plugin works from any source once pk is on PATH,
  and the brief says when pk and the plugin disagree.

### Tests

- `cmd/pk/shim_test.go` (skipped on Windows): copy `bin/pk` into a temp
  `plugin/bin/` with no binaries beside it; write a fake `pk` script into
  a temp `path/` that prints `PK_PLUGIN_ROOT` and its arguments; run the
  shim with `PATH` set to `plugin/bin:path` so the shim's own directory
  comes first; assert the fake ran once, saw the plugin root, got the
  arguments, and the exit code passed through. A second case with no
  fake on PATH asserts exit 3 and the message naming `go install`. A
  third copies a fake sibling `pk-<os>-<arch>` and asserts it wins and
  `PK_PLUGIN_ROOT` is unset.
- `internal/version/semver_test.go`: the `Compare` table.
- `internal/brief/brief_test.go`: `Text` for the four sentence cases;
  the hook shape with `PK_PLUGIN_ROOT` pointing at a temp plugin whose
  `plugin.json` is behind, ahead, equal, and absent (exit 1, stderr names
  the file); `PluginVersion` with the variable unset.
- `internal/repo/repo_test.go`: `pk init` with `PK_PLUGIN_ROOT` set to a
  temp plugin prints the comparison sentence in its brief.

## Boundaries, so the change adds no second copy

The implementation introduces no second copy of any fact and none of the
anti-patterns in the repository's record: no developer file decoded into
a struct and re-encoded, no silent fallback where a required thing is
missing, no network call, no injected filesystem in tests, no sibling
helper where an existing one can be extended. Before each commit, grep
the diff for a repeated block and read each commit's `--stat` file list.

- **One compare.** `Semver.Compare` in `internal/version`; nothing else
  orders versions. git keeps sorting tags.
- **One reader of `plugin.json`.** `brief.PluginVersion`, through
  `jsonsplice.Get`. Neither `status` nor `version` reads it.
- **One sentence.** Built in `Text`, which both callers already use;
  `pk init` gains no prose of its own.
- **One name for the variable.** `PK_PLUGIN_ROOT` is set in the two shims
  and read in one Go constant in `internal/brief`; the shim test ties the
  shell side to the Go side.
- **The shim's dispatch is unchanged.** The fallback is a branch after
  the existing check, not a rewrite; the platform mapping, the sibling
  path, and the exec are the lines they are today.
- **Existing helpers.** `scratch` and the payload-reader pattern from
  `brief_test.go`; `scratch`, `mustGit`, `run` from `repo_test.go`;
  `ParseSemver` for both sides of the comparison.

## Commits

1. `fix(version): source builds report dev plus the commit on Go 1.24`
2. `feat(brief): the shim falls back to pk on PATH and the brief says when it is behind`

## Verification

- `make test` before and after each change; the shim test runs in it.
- Smoke, version: `make build` then `./pk version` prints `pk dev+<sha>`
  with `.dirty` while the tree is dirty; `go version -m ./pk` shows the
  pseudo-version the toolchain recorded, which the output no longer
  echoes.
- Smoke, fallback: `make build`, copy the tree's `bin/pk` into a scratch
  `plugin/bin/` with no binary, put `./pk` on PATH as `pk`, run
  `plugin/bin/pk version` and see the PATH pk's version; run
  `plugin/bin/pk brief --project-dir <configured scratch repo>` with
  `.claude-plugin/plugin.json` in the scratch plugin saying `9.9.9` and
  read the "behind the plugin" sentence; set it to `0.0.1` and read
  "ahead".
- Smoke, no change for the archive: `make bin-local`, run `bin/pk
  version` from the checkout and see the sibling binary answer with no
  `PK_PLUGIN_ROOT` in play.
- Smoke, recursion guard: with `PATH=<checkout>/bin` only and no sibling
  binary, `bin/pk version` exits 3 with the install hint instead of
  hanging.
- Must fail: `PK_PLUGIN_ROOT` pointing at a directory with no
  `.claude-plugin/plugin.json`; `pk brief` exits 2 naming the file, and
  as a hook exits 1 with the same message on stderr.
- `claude plugin validate . --strict` and `make docs` leave the tree as
  expected.
