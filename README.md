# plankit

Plan-driven development toolkit for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Designed for small teams and independent developers.

## What it does

- **Preserves approved plans** as timestamped documentation in `docs/plans/`, committed and pushed automatically
- **Protects preserved plans** from accidental edits by Claude Code
- **Installs skills** — `/preserve` for manual plan saving, `/review` for comprehensive code review, `/changelog` for release management
- **Provides CLAUDE.md templates** — battle-tested guidelines for working effectively with Claude Code

## Install

```bash
go install github.com/markwharton/plankit/cmd/pk@latest
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases).

## Setup

```bash
cd your-project
pk setup
```

This configures `.claude/settings.json` with hooks and installs the `/preserve`, `/review`, and `/changelog` skills. Restart Claude Code to apply.

### Modes

```bash
pk setup                       # Auto: preserve plans on ExitPlanMode
pk setup --preserve manual     # Manual: use /preserve when ready
```

Re-run setup anytime to switch.

## Commands

```
pk changelog [options]    Generate changelog, commit, and tag release
pk preserve [--notify]    Preserve approved plan (PostToolUse hook)
pk protect                Block edits to docs/plans/ (PreToolUse hook)
pk setup [options]        Configure project hooks and skills
pk version                Print version and check for updates
```

## Templates

The `templates/` directory contains starter CLAUDE.md files:

| Template | Use for |
|----------|---------|
| `base.md` | Universal principles — start here |
| `go.md` | Go projects |
| `typescript.md` | TypeScript/Node/Bun projects |
| `azure.md` | Azure infrastructure and deployment |

Copy `base.md` into your project as `CLAUDE.md` and extend with relevant technology templates.

## Documentation

- [Getting Started](docs/getting-started.md) — install, setup, first run
- [Changelog](docs/changelog.md) — pk changelog, .changelog.json configuration
- [Methodology](docs/methodology.md) — plan-driven development, guidelines, testing loop
- [Anti-Patterns](docs/anti-patterns.md) — what to watch for

## Cross-platform

`pk` is a single Go binary with zero external dependencies. Builds for macOS, Linux, and Windows. Windows support relies on Git Bash, which is required by Claude Code.

## License

MIT
