# CLAUDE.md

IMPORTANT: Follow these rules at all times.

## Critical Rules

- NEVER take shortcuts without asking. STOP, ASK, WAIT for approval.
- NEVER force push. Make a new commit to fix mistakes. (`pk guard` blocks the agent's push; the ruleset blocks a force-push to `main`.)
- NEVER commit secrets to version control.
- Only do what was asked. No scope creep.
- When rules conflict, prefer the safer, more reversible action and ask.

## Project Conventions

### Branch & Workflow

- All changes go through `develop`. `pk guard` blocks a commit, merge, rebase or reset on `main` inside a session (`guard.branches`); the ruleset blocks a push to it.
- Release: `pk changelog` on `develop`, then `pk release`; `pk release` refuses a merge that is not a fast-forward.
- Conventional Commits. A commit of a type outside `changelog.types` is dropped from the changelog by `pk changelog`, without a warning.

### Quick Commands

```bash
make build          # Build for current platform -> dist/pk
make test           # Run tests with race detector
go test -run TestName ./internal/<pkg>   # Run a single test
make build-all      # Cross-compile for 5 platforms
make install        # Install to GOPATH/bin
make lint           # Run go vet + gofmt drift check
make rules-lint     # Lint .claude/rules: hidden chars + house style (--strict)
make vuln           # Scan for known vulnerabilities (govulncheck)
make cover          # Per-function coverage for .go files changed since the latest tag; run before /ship
pk changelog        # Generate CHANGELOG.md and commit (no tag)
pk changelog --undo # Unwind an unpushed release commit
pk release          # Read Release-Tag trailer, merge, create tag, and push
```

- **Build with `make build`**, which writes `dist/pk`; a bare `go build ./cmd/pk` lands a binary at the root. Both paths are git-ignored (`dist/`, `/pk`); the wrong one is invisible to git, not blocked.

### Language & Build

- **Go 1.26, standard library only.** `go.mod` has no `require` block; `make vuln` in CI scans the toolchain.
- Binary: `pk`, entrypoint `cmd/pk/main.go`. Build: `make build`. Test: `make test` (`go test -v -race ./...`). Cross-compile: `make build-all` (darwin and linux on amd64 and arm64, windows on amd64). Version: ldflags `-X .../version.version=$(VERSION)`.
- **User messages go to stderr through `internal/msg`**; stdout carries hook JSON only. `msg_test.go` covers the forms; nothing scans call sites for a stray `fmt.Println`.
- `CGO_ENABLED=0` is exported by the Makefile for builds; `make test` runs with `CGO_ENABLED=1` for the race detector.
- **Flags are `--kebab-case`**; `usageFor` in `cmd/pk/main.go` prints them that way for every flagset.

### Directory Structure

- `cmd/pk/` — CLI entrypoint, flag parsing, subcommand dispatch.
- `internal/` — all packages: `changelog`, `config`, `git`, `guard`, `hooks`, `msg`, `paths`, `preserve`, `protect`, `readiness`, `release`, `rules`, `safety`, `scaffold`, `setup`, `status`, `teardown`, `update`, `version`.
- `internal/setup/` — organized by concern: `claude.go` for Claude Code-specific wiring (hooks, settings, bootstrap), `managed.go`/`pin.go`/`baseline.go` for universal logic, `setup.go` for orchestration.
- `docs/` — user-facing documentation. `docs/plans/` — preserved plans (immutable after creation).
- `.claude/skills/` — managed skills (pk-configure, preserve, ship) plus maintainer-only skills (new-plankit-project, review-code, review-rules, review-staged, workshop-notes) that do not ship via `pk setup`.
- `evals/` — maintainer-only eval harness: rules-ablation and guard-enforcement scripts (`run-evals.sh`, `world.sh`, `guard-eval.sh`, `cases.md`), `footprint` (writes the README footprint line, runs in the changelog preCommit hook), `calibrate` (token-ratio calibration).
- `.claude/rules/plankit/` — managed rules (craft, conduct, docs), installed under a `plankit/` subdirectory so they never collide with a project's own `.claude/rules/` files (Claude Code discovers rules recursively). `development.md` (maintainer-only, not shipped) stays at `.claude/rules/`.

### Design

- **Every command resolves to the repository root** through `git.RepoRoot` (stat-based, no subprocess); only `setup` falls back to the given directory (`--allow-non-git`).

### Code Patterns

- **Dependency injection via Config structs.** Every package exports a `Config` struct with injected deps (`Stdin`, `Stdout`, `Stderr`, `GitExec`, `ReadFile`, …) and a `DefaultConfig()`; a function that does I/O takes `readFile`/`writeFile` parameters. Tests inject them, with `t.TempDir()` for the filesystem; `make cover` lists any changed `.go` file whose error paths are uncovered.
- **Hook commands** read JSON from stdin, write JSON to stdout, and exit 0; shared helpers in `internal/hooks` (`ResolveProjectDir`, `ReadInput`, `WritePostToolUse`, `WritePermissionDecision`). `make test` covers the response writers.
- **Managed files** embed a SHA marker (HTML comment for CLAUDE.md, YAML frontmatter `pk_sha256` for skills and rules) so `pk setup` can detect user modifications.
- **Embedded assets** via `//go:embed`: templates, skills and rules. `embed_safety_test.go` scans them for hidden characters under `make test`.

### Updating pk-managed files

When editing a file that has `pk_sha256` in its frontmatter (skills, rules), update both the embedded source in `internal/setup/` and the local copy in `.claude/`, then recompute the body hash with:

```bash
sed -n '/^---$/,/^---$/!p' <embedded-source> | shasum -a 256
```

Replace the `pk_sha256` line in the local copy with the new value. The sed pattern excludes the frontmatter `---`...`---` block, matching Go's body hash calculation byte-for-byte. `pk status` reports the copy as modified until the hash matches.

### Configuration

- `.pk.json`: [docs/pk-json.md](docs/pk-json.md).

### Documentation

- The local rule `.claude/rules/documentation.md` governs sentences; the house convention [docs/design.md](docs/design.md) governs which files exist; the command-doc sections are in [CONTRIBUTING.md](CONTRIBUTING.md). A new config key, message or env var updates its reference file in the same commit; grep for a sibling flag before declaring an option done.
- Terminology: "developer" for the role (reviewing, testing, directing), "builder" for the audience (who plankit serves generally).

### Commits and Releases

- GitHub Actions are pinned to commit SHAs; Dependabot (`.github/dependabot.yml`) bumps the pins on `develop`.

### Hook Protocol

- The Hook protocol section of each hook's doc: [pk guard](docs/pk-guard.md), [pk protect](docs/pk-protect.md), [pk preserve](docs/pk-preserve.md).
