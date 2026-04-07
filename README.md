# plankit

Plan-driven development toolkit for [Claude Code](https://code.claude.com) (Anthropic's AI coding agent). Designed for small teams and independent developers.

LLMs are open-ended by nature; development needs deterministic outcomes. plankit bridges that gap — plans commit to an approach before code is written, templates suppress the patterns that cause drift, and tests protect what works.

## What it does

- **Creates CLAUDE.md** with critical rules, and installs detailed guidelines as `.claude/rules/`
- **Guards protected branches** — blocks git mutations on configured branches via hooks
- **Preserves approved plans** as timestamped documentation in `docs/plans/`, committed to git
- **Protects preserved plans** from accidental edits by Claude Code
- **Installs Claude Code skills** — `/init`, `/changelog`, `/release`, `/preserve`

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
pk setup                       # Manual: use /preserve when ready
pk setup --preserve auto       # Auto: preserve plans on ExitPlanMode
```

Re-run setup anytime to switch.

## Commands

| Command | Description |
|---------|-------------|
| `pk setup` | Configure project hooks, skills, and CLAUDE.md — [details](docs/pk-setup.md) |
| `pk changelog` | Generate changelog, commit, and tag — [details](docs/pk-changelog.md) |
| `pk release` | Merge to release branch, validate, and push — [details](docs/pk-release.md) |
| `pk guard` | Block git mutations on protected branches — [details](docs/pk-guard.md) |
| `pk preserve` | Preserve approved plan — [details](docs/pk-preserve.md) |
| `pk protect` | Block edits to docs/plans/ — [details](docs/pk-protect.md) |
| `pk version` | Print version and check for updates — [details](docs/pk-version.md) |

## Documentation

- [Getting Started](docs/getting-started.md) — install, setup, first run
- [Methodology](docs/methodology.md) — plan-driven development, guidelines, testing loop
- [Anti-Patterns](docs/anti-patterns.md) — what to watch for
- [Resources](docs/resources.md) — Claude Code best practices, git references

## Known Limitations

- **Ultraplan (preview)**: plankit hooks require `ExitPlanMode` and a local plan file in `~/.claude/plans/`. Ultraplan runs remotely and delivers plans inline — no local file is written and no `ExitPlanMode` fires, so preservation won't trigger. Use standard `/plan` mode for automatic preservation. ([Provide feedback](https://github.com/anthropics/claude-code/issues))

## Cross-platform

`pk` is a single Go binary with zero external dependencies. Builds for macOS, Linux, and Windows. Windows support relies on Git Bash, which is required by Claude Code.

## License

MIT
