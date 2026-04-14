# plankit

[![CI](https://github.com/markwharton/plankit/actions/workflows/ci.yml/badge.svg)](https://github.com/markwharton/plankit/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/markwharton/plankit/graph/badge.svg?token=y1SS0kyj3v)](https://codecov.io/gh/markwharton/plankit)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**LLMs are open-ended by nature. Development needs deterministic outcomes. plankit bridges that gap — plans commit to an approach before code is written, templates suppress the patterns that cause drift, and tests protect what works.**

A plan-driven development toolkit for [Claude Code](https://code.claude.com) (Anthropic's AI coding agent). The rules, testing discipline, and branch protection aren't extras — they're what make plans worth keeping. Designed for small teams and independent developers.

## What it does

`pk setup` installs the pieces that make plan-driven development work with Claude Code:

- **Installs rules and guidelines** — CLAUDE.md with critical rules, plus detailed `.claude/rules/` for model behavior, development standards, and git discipline
- **Adds Claude Code skills** — `/init`, `/changelog`, `/release`, `/preserve`
- **Preserves approved plans** — saved as timestamped documentation in `docs/plans/`, committed to git, and protected from accidental edits
- **Guards protected branches** — git mutations blocked via hooks, locally, before the damage happens

After setup, `/changelog` and `/release` become your daily driver — each previews, confirms, then runs.

## Install

Requires [Go](https://go.dev/doc/install) (for `go install`) and [Claude Code](https://code.claude.com).

```bash
go install github.com/markwharton/plankit/cmd/pk@latest
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases) (no Go required).

After installing, run `pk setup` in your project to configure hooks and skills. See [Setup](#setup) below or [Getting Started](docs/getting-started.md) for details.

## Setup

```bash
cd your-project
pk setup
```

This configures `.claude/settings.json` with hooks and installs skills. Restart Claude Code to apply.

### Modes

```bash
pk setup                       # Default: block guard, manual preserve
pk setup --guard ask           # Prompt instead of blocking on protected branches
pk setup --preserve auto       # Auto: preserve plans on ExitPlanMode
```

Re-run setup anytime to switch.

## Commands

| Command | Description |
|---------|-------------|
| `pk setup` | Configure project hooks, skills, and CLAUDE.md — [details](docs/pk-setup.md) |
| `pk teardown` | Remove plankit hooks, skills, and rules — [details](docs/pk-teardown.md) |
| `pk changelog` | Generate CHANGELOG.md and commit (tag is created by `pk release`) — [details](docs/pk-changelog.md) |
| `pk release` | Tag, merge to release branch, validate, and push — [details](docs/pk-release.md) |
| `pk guard` | Block git mutations on protected branches — [details](docs/pk-guard.md) |
| `pk preserve` | Preserve approved plan — [details](docs/pk-preserve.md) |
| `pk protect` | Block edits to docs/plans/ — [details](docs/pk-protect.md) |
| `pk pin` | Update pinned version in a script file — [details](docs/pk-pin.md) |
| `pk version` | Print version and check for updates — [details](docs/pk-version.md) |

## Documentation

- [Getting Started](docs/getting-started.md) — install, setup, first run
- [Methodology](docs/methodology.md) — plan-driven development, guidelines, testing loop
- [Anti-Patterns](docs/anti-patterns.md) — what to watch for
- [Resources](docs/resources.md) — Claude Code best practices, git references

## Known Limitations

- **Ultraplan (preview)**: plankit hooks require `ExitPlanMode` and a local plan file in `~/.claude/plans/`. Ultraplan runs remotely and delivers plans inline — no local file is written and no `ExitPlanMode` fires, so preservation won't trigger. Use standard `/plan` mode for automatic preservation. ([Provide feedback](https://github.com/anthropics/claude-code/issues))
- **Claude Code on the web**: `pk setup` installs a SessionStart hook that fetches the matching `pk` binary into the cloud sandbox at session start. Protective hooks (`pk guard`, `pk preserve`, `pk protect`) then work normally. Mobile has no shell environment — hooks degrade to no-ops there.

## Cross-platform

`pk` is a single Go binary with zero external dependencies. Builds for macOS, Linux, and Windows. Windows support relies on Git Bash, which is required by Claude Code.

## License

MIT
