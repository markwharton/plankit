# pk setup

Configure a project's hooks, skills, and CLAUDE.md for use with plankit.

## Usage

```bash
pk setup                              # default: manual preserve mode
pk setup --preserve auto              # auto-preserve plans on ExitPlanMode
pk setup --project-dir /path/to/dir   # specify project directory
pk setup --force                      # overwrite all managed skills
```

## How it works

1. **Configures `.claude/settings.json`** with PreToolUse and PostToolUse hooks, and adds `Bash(pk:*)` permission for skill execution. Existing user hooks are preserved — only plankit hooks are added or updated.
2. **Creates `CLAUDE.md`** from the universal template if none exists. If a pk-managed CLAUDE.md exists and hasn't been modified, it is updated. User-modified or unmanaged files are left alone. CLAUDE.md is never force-overwritten — once customized, it is user-owned.
3. **Installs skills** to `.claude/skills/`: `/changelog`, `/preserve`, `/release`. User-modified skills are skipped unless `--force` is used.
4. **Checks PATH** and warns if `pk` is not found.

After setup, restart Claude Code to apply changes.

## CLAUDE.md

The universal CLAUDE.md provides battle-tested guidelines for Model Behavior and Development Standards. On its own, it prevents the most common issues — scope creep, silent fallbacks, shortcuts without permission, untested changes. That's the floor.

Add a `## Project Conventions` section to make Claude productive from the first message of every session. Without project conventions, Claude follows the rules but has to rediscover the project each session. With them, it knows the project from the start.

### Customize your CLAUDE.md

After running `pk setup`, ask Claude to add project-specific conventions. You can paste the prompt below directly, or create it as a reusable `/init` skill (see [Create your own skills](#create-your-own-skills)).

> Analyze this project and generate or refresh the **Project Conventions** section in CLAUDE.md.
>
> Run this after `pk setup` to add project-specific conventions, or re-run anytime as the project evolves.
>
> **Steps:**
>
> 1. Read the existing CLAUDE.md. If it does not exist, stop and tell the user to run `pk setup` first.
> 2. If a `## Project Conventions` section already exists, read it carefully — this is a refresh, not a blank slate. Preserve conventions that are still accurate, update what has changed, and add anything new.
> 3. Explore the project to identify:
>    - Primary language(s) and framework(s)
>    - Build system and test runner
>    - Directory structure and file organization
>    - Existing conventions visible in code (naming, patterns, configuration)
>    - Business and domain rules embedded in application logic, if applicable (default values, calculation rules, workflow states, status transitions, business logic, UI behavior conventions, data safety constraints)
>    - Domain model relationships and creation flows, if applicable (which entities relate to which, what entry points exist, what gets pre-filled)
> 4. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point. Group technical conventions and business/domain rules under separate subheadings.
> 5. Show the proposed section to the user and ask for confirmation before writing.
>
> **Rules:**
>
> - **Append only.** Do not modify the Model Behavior or Development Standards sections.
> - If a `## Project Conventions` section already exists, replace it with the updated version — do not duplicate it.
> - **Remove the pk SHA marker.** If the first line is `<!-- pk:sha256:... -->`, remove it. Once customized, the file is user-owned and the marker is stale.
> - Keep conventions specific and actionable — not generic advice.
> - Include the project's test command, build command, and any deployment patterns you discover.
> - If the project uses `.changelog.json`, include the configured commit types.
> - For business rules, read into services, components, and pages — do not stop at file structure. Extract actual values, defaults, and logic constraints.

### Create your own skills

Skills are markdown files in `.claude/skills/` that Claude Code discovers automatically. You can create skills for any workflow your project needs.

A skill file uses YAML frontmatter for metadata and markdown for instructions:

```markdown
---
name: my-skill
description: What this skill does
---

Instructions for Claude to follow when the user invokes /my-skill.
```

Place the file at `.claude/skills/my-skill/SKILL.md` and restart Claude Code. Users invoke it with `/my-skill`.

Skills work well for repeatable workflows — code review checklists, smoke tests, deployment procedures, project initialization. Keep the prompt focused: Claude understands broad terms like "DRY violations" and "anti-patterns" without needing exhaustive checklists.

## Preserve modes

- **manual** (default) — Use the `/preserve` skill when you're ready to save a plan.
- **auto** — Plans are automatically preserved when you exit plan mode.

Re-run setup anytime to switch modes.

## Managed file protection

Files installed by `pk setup` include a SHA256 integrity marker. The format depends on the file type:

- **CLAUDE.md** — HTML comment on the first line: `<!-- pk:sha256:... -->`
- **Skills** — `pk_sha256` field in the YAML frontmatter

On re-run, `pk setup` checks the marker:

- **File is pristine** (SHA matches) — updated to the latest version.
- **File was modified by user** (SHA mismatch) — skipped with a warning.
- **File has no marker** (not managed by pk) — skipped.

`--force` overrides this for skills only. CLAUDE.md is never force-overwritten.

## Flags

- **--preserve** — Plan preservation mode: `manual` or `auto` (default: `manual`).
- **--force** — Overwrite all managed skills regardless of user modifications. Does not affect CLAUDE.md.
- **--project-dir** — Project directory (default: current directory).
