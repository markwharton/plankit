# Getting Started

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
├── CLAUDE.md                 # Universal guidelines (if none existed)
└── .claude/
    ├── settings.json         # Hooks config + Bash(pk:*) permission
    └── skills/
        ├── changelog/
        │   └── SKILL.md      # /changelog skill
        ├── preserve/
        │   └── SKILL.md      # /preserve skill
        └── release/
            └── SKILL.md      # /release skill
```

Restart Claude Code to apply changes. See [pk setup](pk-setup.md) for all options.

## Setup options

```bash
pk setup                          # Default: manual — use /preserve skill when ready
pk setup --preserve auto          # Auto: preserve plans on ExitPlanMode
pk setup --project-dir /path/to   # Specify project directory
```

Re-run setup anytime to switch modes.

## Try it

1. Start Claude Code in your project
2. Enter plan mode (`/plan`)
3. Describe a task, let Claude create a plan
4. Approve the plan and exit plan mode
5. Type `/preserve` to save the plan to `docs/plans/` (or automatic in auto mode)

## What happens

- **Plan preservation**: Approved plans are saved as timestamped files in `docs/plans/`, committed, and pushed to your remote.
- **Plan protection**: Once preserved, plans cannot be accidentally edited or overwritten by Claude Code.
- **Duplicate detection**: If the same plan content has already been preserved today, it's skipped.

## Customize your CLAUDE.md

`pk setup` creates a universal CLAUDE.md if your project doesn't have one — proven guidelines for Model Behavior and Development Standards. To add project-specific conventions, ask Claude to analyze your project and generate a `## Project Conventions` section — it will explore the codebase and propose conventions for your approval. See [pk setup — Customize your CLAUDE.md](pk-setup.md#customize-your-claudemd) for details.
