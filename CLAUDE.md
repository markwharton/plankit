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
- Release flow: `pk changelog` (on `dev`) → merge to `main` → `pk release`.
- Conventional Commits required. Configured types: `feat`, `fix`, `deprecate`, `revert`, `security`, `refactor`, `perf`, `docs`, `chore`, `test`, `build`, `ci`, `style`, `plan` (hidden).

### Language & Build

- **Go 1.21**, standard library only — no third-party dependencies.
- Binary: `pk` — single entrypoint at `cmd/pk/main.go`.
- Build: `make build` (output to `dist/`).
- Test: `make test` (runs `go test -v -race ./...`).
- Cross-compile: `make build-all` (darwin/linux amd64+arm64, windows amd64).
- Version injected via ldflags (`-X .../version.version=$(VERSION)`).

### Directory Structure

- `cmd/pk/` — CLI entrypoint, flag parsing, subcommand dispatch.
- `internal/` — all packages: `changelog`, `guard`, `hooks`, `preserve`, `protect`, `release`, `setup`, `update`, `version`.
- `docs/` — user-facing documentation. `docs/plans/` — preserved plans (immutable after creation).
- `.claude/skills/` — managed skills (changelog, init, preserve, release, review).
- `.claude/rules/` — managed rules (development-standards, git-discipline, model-behavior).
- `site/` — landing page.

### Code Patterns

- **Dependency injection via Config structs.** Every package exports a `Config` struct with injectable deps (`Stdin`, `Stdout`, `Stderr`, `GitExec`, `ReadFile`, etc.) and a `DefaultConfig()` factory wired to real implementations.
- **Tests use Config mocks** — no external test frameworks, no mocking libraries. Tests inject functions that return canned data.
- **Hook commands** read JSON from stdin, write JSON to stdout, and always exit 0. Shared types live in `internal/hooks`.
- **Managed files** embed a SHA marker (HTML comment for CLAUDE.md, YAML frontmatter `pk_sha256` for skills) so `pk setup` can detect user modifications.
- **Embedded assets** via `//go:embed` — templates, skills, and rules are compiled into the binary.

### Configuration

- `.pk.json` is the project-level config file. Top-level keys map to `pk` subcommands (`changelog`, `guard`).
- `changelog.types` controls commit type → changelog section mapping.
- `changelog.hooks` supports `preRelease`, `preCommit`, `postVersion` lifecycle hooks.
- `guard.protectedBranches` lists branches where git mutations are blocked.
