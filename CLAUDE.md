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
