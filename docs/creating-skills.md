# Creating skills

Skills are reusable prompts that Claude Code discovers automatically. You package one as a markdown file with YAML frontmatter, and users invoke it with `/<skill-name>`. They're how you turn a repeatable workflow — a code review checklist, a deploy procedure, a session opener — into a one-command muscle.

Custom slash commands have been merged into skills in Claude Code. Skills are the forward-looking approach.

> Authoritative reference: [Claude Code Skills](https://code.claude.com/docs/en/skills.md)

## File structure

A skill is a directory containing a `SKILL.md` file. Two locations are checked:

- **Project skills** — `.claude/skills/<skill-name>/SKILL.md` — committed to the project, shared with the team.
- **Personal skills** — `~/.claude/skills/<skill-name>/SKILL.md` — your own toolbox, available everywhere.

Restart Claude Code after adding a skill. Users invoke it with `/<skill-name>`.

## Frontmatter

The frontmatter is YAML. All fields are optional, but `description` matters most — Claude reads it to decide whether to invoke the skill on its own.

```yaml
---
name: my-skill
description: What this skill does and when to use it
---
```

Common fields worth knowing:

- **name** — display name (lowercase, hyphens). Defaults to the directory name.
- **description** — when Claude should use this skill. Front-load the key use case. Limit ~250 chars.
- **disable-model-invocation: true** — block Claude from invoking the skill on its own. Use this for high-stakes workflows that the user must trigger explicitly (releases, deployments, anything destructive).
- **user-invocable: false** — hide the skill from user menus. Use this for background knowledge Claude consults but you don't invoke directly.
- **argument-hint** — hint shown in the `/` autocomplete menu.
- **allowed-tools** — list of tools the skill can use without asking for permission while it's active. For example, `allowed-tools: Bash(pk:*)` lets a `pk`-wrapping skill run without prompting on each `pk` command. Use this to make skills self-contained — they work even in projects where `settings.json` doesn't have a matching permission entry. plankit's installed skills (`/changelog`, `/preserve`, `/release`) use this pattern.

See the [official Skills reference](https://code.claude.com/docs/en/skills.md) for the full schema (paths, hooks, model, subagent forking, and more).

## Your first skill

A minimal skill that reviews staged changes:

```markdown
---
name: review-staged
description: Review staged git changes for bugs, security issues, and style violations
---

Review the changes currently staged for commit:

git diff --cached

Look for:
- Bugs and logic errors
- Security issues (input validation, secrets, injection)
- Style violations against project conventions
- Missing tests

Report findings as a bulleted list grouped by severity.
```

Save this as `.claude/skills/review-staged/SKILL.md`, restart Claude Code, and invoke it with `/review-staged`.

## Interactive patterns

Skills become much more powerful when they involve the user in the loop. The four skills plankit installs (`.claude/skills/changelog/`, `init/`, `preserve/`, `release/`) are working examples worth reading.

### Preview and confirm

Before destructive or hard-to-reverse actions, run the dry-run version, show the user what will happen, and ask for confirmation. The `/changelog`, `/preserve`, and `/release` skills all use this pattern:

```markdown
First, preview with a dry run:

pk changelog --dry-run

Show the preview to the user and ask for confirmation before proceeding.
If confirmed, run:

pk changelog

Report the result to the user.
```

### Ask, then act

When a skill needs information from the user before it can proceed, ask up front rather than guessing. The `/init` skill asks about branch conventions before drafting `## Project Conventions`:

> Ask the user about branch conventions:
> - What is the default branch for development?
> - Are there branches that should never receive direct commits?
> - Which branch should releases be pushed to?

Gather inputs → draft → show → confirm → write. This turns Claude into a structured assistant rather than a guessing oracle.

### Multi-step exploration

For skills that need to discover state before acting, lay out the steps explicitly. The `/init` skill explores the codebase, drafts a section, shows it, asks for confirmation, then writes. Each step is a distinct instruction in the SKILL.md so Claude follows the sequence reliably.

## When skills work well

- **Repeatable workflows** — code review checklists, smoke tests, deployment procedures, project initialization, changelog generation.
- **Multi-step processes** — anything where you'd otherwise re-explain the steps every session.
- **Team knowledge** — onboarding rituals, project-specific patterns, "the way we do X here."
- **High-stakes actions** — pair `disable-model-invocation: true` with preview-and-confirm so users always trigger them deliberately.

Skills work less well for one-off questions or exploratory tasks where the right approach depends on what you find.

## Tips

- **Keep prompts focused.** Claude already understands broad terms like "DRY violations" and "anti-patterns." You don't need exhaustive checklists — name the categories and trust the model.
- **Front-load the description.** Claude scans skill descriptions to decide whether to auto-invoke. Put the key use case at the start.
- **Trust supporting files.** SKILL.md doesn't have to contain everything. Reference larger templates or examples in sibling files; Claude will read them when relevant.
- **Iterate with real use.** Write a draft, use the skill in a session, notice where Claude gets stuck or asks the wrong questions, refine the prompt.

## References

- [Claude Code Skills](https://code.claude.com/docs/en/skills.md) — full schema, frontmatter fields, advanced features.
- [Claude Code .claude directory](https://code.claude.com/docs/en/claude-directory.md) — file location and discovery rules.
- plankit's installed skills (`.claude/skills/`) — `changelog`, `init`, `preserve`, `release`, `review` as live examples.
