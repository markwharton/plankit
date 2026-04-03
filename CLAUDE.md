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
```

## Architecture

- `cmd/pk/main.go` -- Entry point, subcommand routing
- `internal/hooks/` -- Shared stdin JSON parsing for Claude Code hook payloads
- `internal/protect/` -- `pk protect` (PreToolUse: block edits to docs/plans/)
- `internal/preserve/` -- `pk preserve` (PostToolUse: preserve approved plans)
- `internal/setup/` -- `pk setup` (configure project .claude/settings.json)
- `internal/update/` -- Version checker (GitHub releases, daily cache)
- `internal/version/` -- Build version via ldflags
- `skills/` -- Skill files installed by pk setup
- `templates/` -- Reference CLAUDE.md starters
- `docs/` -- Methodology and getting started documentation

## Conventions

- Go 1.21+, zero external dependencies (stdlib only)
- Subcommand routing via `os.Args` switch + `flag.FlagSet`
- Version injection: `-ldflags "-X .../version.Version=x.y.z"`
- All user messages to stderr, stdout reserved for hook protocol JSON
- Hook commands always exit 0 (errors reported via systemMessage or stderr)
- Tests use dependency injection (Config struct) and `t.TempDir()` for filesystem tests

## Hook Protocol

Claude Code hooks receive JSON on stdin and produce JSON on stdout:

- **PreToolUse**: Output `{"decision":"block","reason":"..."}` + exit 0 to block. Exit 0 with no output to allow. Any non-zero exit (including command-not-found 127) is non-blocking.
- **PostToolUse**: Output `{"systemMessage":"..."}` + exit 0 for feedback. Non-zero exit is a non-blocking error.
