# pk setup

Configure a project's hooks, skills, and CLAUDE.md for use with plankit.

## Usage

```bash
pk setup                              # default: manual preserve mode
pk setup --preserve auto              # auto-preserve plans on ExitPlanMode
pk setup --force                      # overwrite all managed files
pk setup --project-dir /path/to/dir   # specify project directory
```

## What it does

1. **Configures `.claude/settings.json`** with PreToolUse and PostToolUse hooks, and adds `Bash(pk:*)` permission for skill execution.
2. **Creates `CLAUDE.md`** from the universal template if none exists. If a pk-managed CLAUDE.md exists and hasn't been modified, it is updated. User-modified or unmanaged files are left alone.
3. **Installs skills** to `.claude/skills/`: `/changelog`, `/init`, `/preserve`, `/release`, `/review`. Same update rules as CLAUDE.md — user-modified skills are not overwritten.
4. **Checks PATH** and warns if `pk` is not found.

After setup, restart Claude Code to apply changes.

## CLAUDE.md

The universal CLAUDE.md provides battle-tested guidelines for Model Behavior and Development Standards. On its own, it prevents the most common issues — scope creep, silent fallbacks, shortcuts without permission, untested changes. That's the floor.

Run `/init` to add project-specific conventions — build commands, test runner, commit types, directory patterns. Without `/init`, Claude follows the rules but has to rediscover the project each session. With `/init`, it knows the project from the start.

Run `/init` again anytime to refresh conventions as the project evolves. The Project Conventions section will grow naturally through use — the base doesn't need to be perfect, it needs to be enough that the first session doesn't go off the rails.

## Preserve modes

- **manual** (default) — Use the `/preserve` skill when you're ready to save a plan.
- **auto** — Plans are automatically preserved when you exit plan mode.

Re-run setup anytime to switch modes.

## Managed file protection

Files installed by `pk setup` include a SHA256 marker on the first line:

```
<!-- pk:sha256:abc123... -->
```

On re-run, `pk setup` checks this marker:

- **File is pristine** (SHA matches) — updated to the latest version.
- **File was modified by user** (SHA mismatch) — skipped with a warning.
- **File has no marker** (not managed by pk) — skipped.

Use `--force` to overwrite all managed skills regardless of modifications. CLAUDE.md is never force-overwritten — once customized (via `/init` or manually), it is user-owned.

## Flags

- **--preserve** — Plan preservation mode: `manual` or `auto` (default: `manual`).
- **--force** — Overwrite all managed skills regardless of user modifications. Does not affect CLAUDE.md.
- **--project-dir** — Project directory (default: current directory).
