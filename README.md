# plankit

[![CI](https://github.com/markwharton/plankit/actions/workflows/ci.yml/badge.svg)](https://github.com/markwharton/plankit/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/markwharton/plankit/graph/badge.svg?token=y1SS0kyj3v)](https://codecov.io/gh/markwharton/plankit)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Every new LLM session picks a different approach. Plans narrow a specific task to an approved approach before code is written. Rules reduce drift between sessions. Hooks preserve what was approved and guard what should not change.**

A plan-driven development toolkit for [Claude Code](https://code.claude.com). Plans are shared artifacts: one or two developers review and approve an approach, and that becomes the record. Plan preservation and plan protection keep approved work from being lost or overwritten. Discipline is the multiplier; rules, testing, and branch protection make plans worth keeping. Designed for small teams and independent developers.

Anthropic's [Claude Code Best Practices](https://code.claude.com/docs/en/best-practices) covers the fundamentals that plankit builds on.

## What it does

`pk setup` installs the pieces that make plan-driven development work with Claude Code:

- **Installs rules and guidelines:** CLAUDE.md with critical rules, plus detailed `.claude/rules/` for model behavior, development standards, and git discipline
- **Adds Claude Code skills:** `/init`, `/preserve`, `/ship`
- **Preserves approved plans:** saved as timestamped documentation in `docs/plans/`, committed to git, and protected from accidental edits
- **Guards protected branches:** git mutations blocked via hooks, locally, before the damage happens

After setup, `/ship` is your release workflow. It chains `pk changelog` and `pk release` with preview+confirm at each step.

## Install

Requires [Go](https://go.dev/doc/install) (for `go install`) and [Claude Code](https://code.claude.com).

```bash
go install github.com/markwharton/plankit/cmd/pk@latest
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases) (no Go required). `pk` is a command-line tool: run it from a terminal (PowerShell, Command Prompt, or Git Bash on Windows), not by double-clicking the binary.

After installing, run `pk setup` in your project to configure hooks and skills. See [Setup](#setup) below for details.

## Setup

```bash
cd your-project
pk setup
```

This configures `.claude/settings.json` with hooks and installs skills. Restart Claude Code to apply.

### Modes

```bash
pk setup                       # Default: block guard, manual preserve
pk setup --baseline            # Anchor pk changelog with v0.0.0 tag
pk setup --guard ask           # Prompt instead of blocking on protected branches
pk setup --preserve auto       # Auto: preserve plans on ExitPlanMode
```

Re-run setup to upgrade managed files. Pass `--guard` or `--preserve` explicitly to change modes.

## Commands

| Command | Description |
|---------|-------------|
| `pk setup` | Configure project hooks, skills, and CLAUDE.md. [Details](docs/pk-setup.md) |
| `pk status` | Report plankit configuration state. [Details](docs/pk-status.md) |
| `pk teardown` | Remove plankit hooks, skills, and rules. [Details](docs/pk-teardown.md) |
| `pk changelog` | Generate CHANGELOG.md and commit (tag is created by `pk release`). [Details](docs/pk-changelog.md) |
| `pk release` | Tag, merge to release branch, validate, and push. [Details](docs/pk-release.md) |
| `pk guard` | Block git mutations on protected branches. [Details](docs/pk-guard.md) |
| `pk preserve` | Preserve approved plan. [Details](docs/pk-preserve.md) |
| `pk protect` | Block edits to docs/plans/. [Details](docs/pk-protect.md) |
| `pk pin` | Update pinned version in a file. [Details](docs/pk-pin.md) |
| `pk version` | Print version and check for updates. [Details](docs/pk-version.md) |

## Skills and commands

plankit has three skills that wrap pk commands into workflows Claude Code can run:

| Skill | Wraps | What it does |
|-------|-------|--------------|
| `/init` | — | Analyze the codebase and generate project conventions for CLAUDE.md and `.pk.json` |
| `/preserve` | `pk preserve` | Save the approved plan to `docs/plans/` and commit |
| `/ship` | `pk changelog` + `pk release` | Preview and confirm changelog, then preview and confirm release |

Skills add a preview+confirm cycle and handle the sequencing. The underlying pk commands work standalone from a terminal for power users who want to skip the prompts.

## Documentation

- [Adoption](docs/adoption.md): layered adoption from foundation to release management
- [Anti-Patterns](docs/anti-patterns.md): failure modes identified through real project experience
- [Methodology](docs/methodology.md): plans, guidelines, compounding effect, model resilience
- [Resources](docs/resources.md): Claude Code best practices, git references
- [Versioning](docs/versioning.md): tags as source of truth, version flow into artifacts

### Reference

- [.pk.json](docs/pk-json.md): configuration schema
- [Environment Variables](docs/environment-variables.md): variables pk reads and sets
- [Error Reference](docs/error-reference.md): common errors, causes, and recovery

## Known Limitations

- **Ultraplan (preview)**: plankit hooks require `ExitPlanMode` and a local plan file in `~/.claude/plans/`. Ultraplan runs remotely and delivers plans inline. No local file is written and no `ExitPlanMode` fires, so preservation won't trigger. Use standard `/plan` mode for automatic preservation. ([Provide feedback](https://github.com/anthropics/claude-code/issues))
- **Claude Code on the web**: `pk setup` installs a SessionStart hook that fetches the matching `pk` binary into the cloud sandbox at session start. Protective hooks (`pk guard`, `pk preserve`, `pk protect`) then work normally. Mobile has no shell environment; hooks degrade to no-ops there.

## Cross-platform

`pk` is a single Go binary with zero external dependencies. Builds for macOS, Linux, and Windows. Windows support relies on Git Bash, which is required by Claude Code.

## License

MIT
