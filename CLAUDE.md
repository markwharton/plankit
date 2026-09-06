# CLAUDE.md

IMPORTANT: Follow these rules at all times.

## Critical Rules

- NEVER take shortcuts without asking. STOP, ASK, WAIT for approval.
- NEVER force push. Make a new commit to fix mistakes.
- NEVER commit secrets to version control.
- Only do what was asked. No scope creep.
- When rules conflict, prefer the safer, more reversible action and ask.

## Project Conventions

This file is development context for working on the plankit repository.
It is not plugin content: the plugin ships context through skills/.

### Branch & Workflow

- All changes go through `develop` — never commit directly to `main`.
- Release flow: `pk changelog` (on `develop`) → `pk release` (merges to `main`, pushes, switches back).
- Conventional Commits required. Configured types: `feat`, `fix`, `deprecate`, `revert`, `security`, `refactor`, `perf`, `docs`, `chore`, `test`, `build`, `ci`, `style`, `plan` (hidden).

### Quick Commands

```bash
make build          # docgen + go build -> ./pk
make test           # go vet + go test ./... (repo and tools/docgen)
make docs           # compile skills/ into internal/help/data (committed)
make fmt            # gofmt the tree
make bin-local      # build bin/pk-<os>-<arch> so the bin/pk shim works
make dist           # cross-compile all shim targets into bin/
go test -run TestName ./internal/<pkg>   # single test
```

### Language & Build

- **Go, standard library only** in the main module — no third-party
  dependencies. Markdown parsing (goldmark) lives in `tools/docgen`,
  a separate module that runs at build time only.
- Binary: `pk`, single entrypoint at `cmd/pk/main.go`; commands are
  registered in `commands()` there.
- Version injected via ldflags; a dev build reports the VCS commit.
- CLI flags use `--kebab-case` (e.g., `--dry-run`, `--project-dir`).

### The plugin is the repository

- `skills/` is the documentation source: one topic per command plus the
  `plankit` overview. docgen compiles them into IR for `pk help`; the
  plugin ships the same files verbatim. Skills mirror commands 1:1 and
  `cmd/pk/main_test.go` enforces it.
- `.claude-plugin/` holds plugin.json and marketplace.json (the repo is
  its own marketplace). `hooks/hooks.json` wires guard, protect, and
  preserve through `"${CLAUDE_PLUGIN_ROOT}"/bin/pk`. `bin/` holds the
  committed shims; the per-platform binaries next to them are release
  assets, gitignored.
- `claude plugin validate . --strict` must pass (marketplace run).

### Design

- **All commands resolve to the git repository root** from any
  subdirectory; explicit `--project-dir` paths are taken as given.
- **Safe defaults, opt-in for escalation.** Manual over auto, commit
  over push.
- **Exit taxonomy**: 0 success, 1 usage, 2 state/precondition,
  3 internal. Errors name the fix.
- **Stream discipline**: artifacts on stdout (dry-run sections, JSON
  state), narration on stderr; commit paths keep stdout empty.
- **Presentation is never configured**: flags (`--format`, `--plain`)
  beat env (`NO_COLOR`, `CLICOLOR_FORCE`) beat the TTY probe. Non-TTY
  help output is the exact authored bytes.
- **Hooks fail open and gate on config**: every hook exits 0, and a
  repository without `.pk.json` gets a fast silent no-op.

### Code Patterns

- Commands are `cli.Command` values with declarative `FlagSpec`s; `Run`
  receives a `cli.Context` (resolved project dir, IO, format). Tests
  drive them through `cli.RunIO` with buffers and real repos in
  `t.TempDir()` (bare origins for push flows). No mocking libraries.
- Package-level `now` variables are stubbed for date-sensitive tests.
- Hook commands read JSON on stdin via `internal/hookio` and never fail
  the tool call on their own errors.

### Configuration

- `.pk.json` is the project config; top-level keys map to commands
  (`guard`, `preserve`, `changelog`, `release`). Strict decode: unknown
  keys are named and refused.
- `changelog.versionFiles` stamps JSON version fields (this repo stamps
  `.claude-plugin/plugin.json`). `changelog.hooks` (`postVersion`,
  `preCommit`) and `release.hooks` (`preRelease`, `prePush`) receive
  `$VERSION`; release hooks also get `$TAG`.
- `guard.branches` lists protected branches; `release.branch` selects
  merge flow, empty selects trunk flow.

### Hook Protocol

- **PreToolUse**: permission decisions go on stdout
  (`hookSpecificOutput` with allow/deny/ask); exit 0 always.
- **PostToolUse**: `systemMessage` for the user,
  `hookSpecificOutput.additionalContext` for Claude's next turn.
- A crashed hook must not block work: pk hooks exit 0 on their own
  failures by design.

### Commits and Releases

- GitHub Actions are pinned to commit SHAs, not mutable tags.

## Changing pk

Run the test suite at the start of a session and report the status,
then before and after each change, so failures are attributable.

### Development Standards

- **Preserve the structure you were given.** Let the data model drive
  the code. Never flatten structured data into flat lists then
  reconstruct with heuristics; the context is already lost.
- **Fail fast, no silent fallbacks.** When something required is
  missing or wrong, fail with a clear message, never a made-up default.
  A documented default for an optional setting is not a fallback. The
  hooks fail the same way, loudly to stderr with their name, but they
  exit 0 and decline to decide rather than block the person's work;
  they never guess a decision.
- **Grep before done.** Update every related location together: when
  fixing a pattern or renaming, grep the repo for all instances. One
  fix is not done until every occurrence is fixed.
- **Work isn't done until automated checks and a smoke test pass.**
  Build, tests, and lint; then a manual end-to-end check with specific
  commands and observable outcomes, including at least one negative
  case, whenever the change alters observable behavior. Skip the smoke
  test for pure internal refactors. A proof whose output you did not
  see is not a proof.
- **Diagnostic scripts over rebuild cycles.** If you are about to do
  your second full rebuild while debugging, stop and write a minimal
  script that tests the specific issue.
- **A failed text search means "not found by this method", never "not
  present".** When absence drives a root cause or code change, confirm
  by parsing the structure (walk the JSON/XML/AST), not by
  string-matching the text.

### Versions and History

- **The git tag is the single source of truth for the version.** pk
  computes the next version from the latest tag and the commits since
  it, and stamps it into files (`versionFiles`, `pk pin`). Never read a
  version out of a file in code you write; a task that needs the
  release version goes through the changelog config.
- **Don't rewrite history between `pk changelog` and `pk release`.**
  Changelog records commit SHAs; release publishes them.
- **Rewrite only unpushed commits.** Confirm the target appears in
  `git log --oneline @{push}..HEAD`; if the command errors or the
  target is absent, it has been pushed: make a new commit instead.

### This Repository's Shape

- Follow the recipes in docs/design.md for a new command, policy,
  document, or composed command.
- Facts that live in code are never hand-copied into docs. Generate
  them, or state the concept once in the document that owns it.
- One-liners (a command's `Summary`, a skill's `description`) state
  purpose, not features.
- When behavior changes, reread the affected `Summary`, `description`,
  and docs/architecture.md before committing.
- Skills ship verbatim into model contexts: plain markdown, no
  placeholders, no diagrams.
- Derived values (digests, pins, compiled output) are never committed
  to a source branch.
