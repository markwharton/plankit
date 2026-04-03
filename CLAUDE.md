# CLAUDE.md

## Project Overview

plankit is a plan-driven development toolkit for Claude Code, designed for small teams and independent developers. It provides a cross-platform Go binary (`pk`) for plan preservation and protection, along with skills, CLAUDE.md templates, and methodology documentation.

## Quick Commands

```bash
make build          # Build for current platform -> dist/pk
make test           # Run tests with race detector
make build-all      # Cross-compile for 5 platforms
make install        # Install to GOPATH/bin
make lint           # Run go vet
pk changelog        # Generate CHANGELOG.md, commit, and tag release
pk release          # Validate and push release to origin
```

## Architecture

- `cmd/pk/main.go` -- Entry point, subcommand routing
- `docs/` -- Methodology and getting started documentation
- `internal/changelog/` -- `pk changelog` (generate changelog from conventional commits, commit, tag)
- `internal/hooks/` -- Shared stdin JSON parsing for Claude Code hook payloads
- `internal/release/` -- `pk release` (validate pre-flight checks, push tag to origin)
- `internal/preserve/` -- `pk preserve` (PostToolUse: preserve approved plans)
- `internal/protect/` -- `pk protect` (PreToolUse: block edits to docs/plans/)
- `internal/setup/` -- `pk setup` (configure project .claude/settings.json)
- `internal/setup/skills/` -- Embedded skill files compiled into pk binary
- `internal/update/` -- Version checker (GitHub releases, daily cache)
- `internal/version/` -- Build version via ldflags
- `templates/` -- Reference CLAUDE.md starters

## Conventions

- Go 1.21+, zero external dependencies (stdlib only)
- Subcommand routing via `os.Args` switch + `flag.FlagSet`
- Version injection: `-ldflags "-X .../version.version=x.y.z"` (overrides `debug.ReadBuildInfo()`)
- All user messages to stderr, stdout reserved for hook protocol JSON
- CLI flags use `--kebab-case` (e.g., `--dry-run`, `--project-dir`)
- Hook commands always exit 0 (errors reported via systemMessage or stderr)
- Tests use dependency injection (Config struct) and `t.TempDir()` for filesystem tests

## Hook Protocol

Claude Code hooks receive JSON on stdin and produce JSON on stdout:

- **PreToolUse**: Output `{"decision":"block","reason":"..."}` + exit 0 to block. Exit 0 with no output to allow. Any non-zero exit (including command-not-found 127) is non-blocking.
- **PostToolUse**: Output `{"systemMessage":"..."}` + exit 0 for feedback. Non-zero exit is a non-blocking error.
