# Getting Started

## Prerequisites

- **[Git](https://git-scm.com/)** — plankit uses git hooks and commits preserved plans
- **[Claude Code](https://code.claude.com)** — Anthropic's AI coding agent (plankit extends Claude Code with hooks, rules, and skills)
- **[Go](https://go.dev/doc/install)** — required for `go install` (skip if using a pre-built binary)

## Install

```bash
go install github.com/markwharton/plankit/cmd/pk@latest
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases).

## Setup your project

Run `pk setup` in your project directory:

```bash
cd your-project
pk setup
```

This creates:

```
your-project/
├── CLAUDE.md                 # Critical rules (if none existed)
└── .claude/
    ├── settings.json         # Hooks config + Bash(pk:*) permission
    ├── rules/
    │   ├── development-standards.md
    │   ├── git-discipline.md
    │   └── model-behavior.md
    └── skills/
        ├── changelog/
        │   └── SKILL.md      # /changelog skill
        ├── init/
        │   └── SKILL.md      # /init skill
        ├── preserve/
        │   └── SKILL.md      # /preserve skill
        └── release/
            └── SKILL.md      # /release skill
```

Restart Claude Code to apply changes. See [pk setup](pk-setup.md) for all options.

## Setup options

```bash
pk setup                          # Default: block guard, manual preserve
pk setup --guard ask              # Prompt instead of blocking on protected branches
pk setup --preserve auto          # Auto: preserve plans on ExitPlanMode
pk setup --project-dir /path/to   # Specify project directory
```

Re-run setup anytime to switch modes. To remove plankit from a project, run `pk teardown` ([details](pk-teardown.md)).

## Try it

1. Start Claude Code in your project
2. Enter plan mode (`/plan`) and describe what you need
3. Review the plan, adjust, iterate, approve
4. Claude executes against the approved plan
5. Type `/preserve` to save the plan to `docs/plans/` (or automatic in auto mode)

## What happens

- **Plan preservation**: Approved plans are saved as timestamped files in `docs/plans/` and committed. Push when you're ready (`--push` to include it automatically).
- **Plan protection**: Once preserved, plans cannot be accidentally edited or overwritten by Claude Code.
- **Duplicate detection**: If the same plan content has already been preserved today, it's skipped.

## Customize your CLAUDE.md

`pk setup` creates a CLAUDE.md with critical rules if your project doesn't have one. Detailed guidelines for model behavior, development standards, and git discipline are installed as `.claude/rules/` files. To add project-specific conventions, run `/init` — it analyzes the codebase, discovers technical conventions and business rules, asks about branch protection, and proposes a `## Project Conventions` section for your approval. See [pk setup — Customize your CLAUDE.md](pk-setup.md#customize-your-claudemd) for details.
