# plankit

Plan-driven development toolkit for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Designed for small teams and independent developers.

LLMs are open-ended by nature; development needs deterministic outcomes. plankit bridges that gap — plans commit to an approach before code is written, templates suppress the patterns that cause drift, and tests protect what works.

## What it does

- **Creates a universal CLAUDE.md** if your project doesn't have one — battle-tested guidelines that work as-is
- **Preserves approved plans** as timestamped documentation in `docs/plans/`, committed and pushed
- **Protects preserved plans** from accidental edits by Claude Code
- **Installs Claude Code skills** — `/init`, `/changelog`, `/release`, `/preserve`, `/review`

## Install

```bash
go install github.com/markwharton/plankit/cmd/pk@latest
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases).

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
| `pk release` | Validate and push release — [details](docs/pk-release.md) |
| `pk preserve` | Preserve approved plan — [details](docs/pk-preserve.md) |
| `pk protect` | Block edits to docs/plans/ — [details](docs/pk-protect.md) |
| `pk version` | Print version and check for updates — [details](docs/pk-version.md) |

## Templates

`pk setup` automatically creates a universal CLAUDE.md if your project doesn't have one. Use `/init` to add project-specific conventions.

The `templates/` directory contains reference material for extending your setup:

| Directory | Contents |
|-----------|----------|
| `templates/` | CLAUDE.md extension examples — `base.md`, `go.md`, `typescript.md`, `azure.md` |
| `templates/skills/` | Example skills to copy and adapt — `smoke-test.md`, `validate.md` |

## Documentation

- [Getting Started](docs/getting-started.md) — install, setup, first run
- [Methodology](docs/methodology.md) — plan-driven development, guidelines, testing loop
- [Anti-Patterns](docs/anti-patterns.md) — what to watch for

## Known Limitations

- **Ultraplan (preview)**: plankit hooks require `ExitPlanMode` and a local plan file in `~/.claude/plans/`. Ultraplan runs remotely and delivers plans inline — no local file is written and no `ExitPlanMode` fires, so preservation won't trigger. Use standard `/plan` mode for automatic preservation. ([Provide feedback](https://github.com/anthropics/claude-code/issues))

## Cross-platform

`pk` is a single Go binary with zero external dependencies. Builds for macOS, Linux, and Windows. Windows support relies on Git Bash, which is required by Claude Code.

## License

MIT
