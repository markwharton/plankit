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
your-project/.claude/
├── settings.json         # Hooks config + Bash(pk:*) permission
└── skills/
    ├── preserve/
    │   └── SKILL.md      # /preserve skill
    └── review/
        └── SKILL.md      # /review skill
```

Restart Claude Code to apply changes.

## Setup options

```bash
pk setup                          # Default: auto-preserve plans on ExitPlanMode
pk setup --preserve manual        # Manual: use /preserve skill when ready
pk setup --preserve auto          # Explicit auto mode
pk setup --project-dir /path/to   # Specify project directory
```

Re-run setup anytime to switch modes.

## Try it

1. Start Claude Code in your project
2. Enter plan mode (`/plan`)
3. Describe a task, let Claude create a plan
4. Approve the plan and exit plan mode
5. The plan is preserved in `docs/plans/` (auto mode) or type `/preserve` (manual mode)

## What happens

- **Plan preservation**: Approved plans are saved as timestamped files in `docs/plans/`, committed, and pushed to your remote.
- **Plan protection**: Once preserved, plans cannot be accidentally edited or overwritten by Claude Code.
- **Duplicate detection**: If the same plan content has already been preserved today, it's skipped.

## Copy a CLAUDE.md template

The `templates/` directory contains starter CLAUDE.md files:

- `base.md` — Universal principles (start here)
- `go.md` — Go-specific extensions
- `typescript.md` — TypeScript/Node/Bun extensions
- `azure.md` — Azure infrastructure, secrets, CI/CD

Copy `base.md` into your project as `CLAUDE.md` and extend it with relevant technology templates and your project-specific sections.
