# CLAUDE.md

IMPORTANT: Follow these rules at all times.

## Critical Rules

- NEVER take shortcuts without asking — STOP, ASK, WAIT for approval.
- NEVER force push — make a new commit to fix mistakes.
- NEVER commit secrets to version control.
- Only do what was asked — no scope creep.
- Understand existing code before changing it.
- If you don't know, say so — never guess.
- Test before and after every change.
- Surface errors clearly — no silent fallbacks.

## Project Conventions

### Branch & Workflow

- All changes go through `dev` — never commit directly to `main`.
- Release flow: `pk changelog` (on `dev`) → `pk release` (merges to `main`, pushes, switches back).
- Conventional Commits required. Configured types: `feat`, `fix`, `deprecate`, `revert`, `security`, `refactor`, `perf`, `docs`, `chore`, `test`, `build`, `ci`, `style`, `plan` (hidden).

### Quick Commands

```bash
make build          # Build for current platform -> dist/pk
make test           # Run tests with race detector
make build-all      # Cross-compile for 5 platforms
make install        # Install to GOPATH/bin
make lint           # Run go vet
pk changelog        # Generate CHANGELOG.md, commit, and tag
pk release          # Validate pre-flight checks and push to origin
```

- **Always use `make build`, never `go build ./cmd/pk` directly.** Bare `go build` drops a binary in the working directory; the Makefile routes output to `dist/`.

### Language & Build

- **Go 1.21**, standard library only — no third-party dependencies.
- Binary: `pk` — single entrypoint at `cmd/pk/main.go`.
- Build: `make build` (output to `dist/`).
- Test: `make test` (runs `go test -v -race ./...`).
- Cross-compile: `make build-all` (darwin/linux amd64+arm64, windows amd64).
- Version injected via ldflags (`-X .../version.version=$(VERSION)`).
- All user messages to stderr, stdout reserved for hook protocol JSON.
- CLI flags use `--kebab-case` (e.g., `--dry-run`, `--project-dir`).

### Directory Structure

- `cmd/pk/` — CLI entrypoint, flag parsing, subcommand dispatch.
- `internal/` — all packages: `changelog`, `guard`, `hooks`, `preserve`, `protect`, `release`, `setup`, `update`, `version`.
- `docs/` — user-facing documentation. `docs/plans/` — preserved plans (immutable after creation).
- `.claude/skills/` — managed skills (changelog, init, preserve, release).
- `.claude/rules/` — managed rules (development-standards, git-discipline, model-behavior).
- `site/` — landing page.

### Design

- **Safe defaults, opt-in for escalation.** Manual over auto, commit over push — the default should always be the safer, more local action.

### Code Patterns

- **Dependency injection via Config structs.** Every package exports a `Config` struct with injectable deps (`Stdin`, `Stdout`, `Stderr`, `GitExec`, `ReadFile`, etc.) and a `DefaultConfig()` factory wired to real implementations.
- **Tests use Config mocks** — no external test frameworks, no mocking libraries. Tests inject functions that return canned data. Tests use `t.TempDir()` for filesystem tests.
- **Hook commands** read JSON from stdin, write JSON to stdout, and always exit 0. Shared types live in `internal/hooks`.
- **Managed files** embed a SHA marker (HTML comment for CLAUDE.md, YAML frontmatter `pk_sha256` for skills) so `pk setup` can detect user modifications.
- **Embedded assets** via `//go:embed` — templates, skills, and rules are compiled into the binary.

### Configuration

- `.pk.json` is the project-level config file. Top-level keys map to `pk` subcommands (`changelog`, `guard`, `release`).
- `changelog.types` controls commit type → changelog section mapping.
- `changelog.hooks` supports `preCommit`, `postVersion` lifecycle hooks.
- `release.hooks` supports `preRelease` lifecycle hook.
- `guard.protectedBranches` lists branches where git mutations are blocked.
- `release.branch` configures which branch `pk release` merges to and pushes from.

### Documentation

- Convention format: bold principle, then concise context — plain statement when the rule speaks for itself.
- Documentation tight loop: code → tests → command doc (`docs/pk-<command>.md`). New command docs follow `docs/command-doc-template.md`. Higher-level docs (README, getting-started, methodology) link to command docs and only change when concepts change.
- Terminology: "developer" for the role (reviewing, testing, directing), "builder" for the audience (who plankit serves generally).

### Commits and Releases

- GitHub Actions are pinned to commit SHAs, not mutable tags.

### Hook Protocol

Claude Code hooks receive JSON on stdin and produce JSON on stdout:

- **PreToolUse**: Output `{"decision":"block","reason":"..."}` + exit 0 to block. Exit 0 with no output to allow. Any non-zero exit (including command-not-found 127) is non-blocking.
- **PostToolUse**: Output `{"systemMessage":"..."}` + exit 0 for user-visible feedback. Use `{"hookSpecificOutput":{"additionalContext":"..."}}` to inject context into Claude's next turn. Both fields can be combined in one response. Non-zero exit is a non-blocking error.
