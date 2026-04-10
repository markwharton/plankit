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

## Tutorial: Build a /release-pr skill

This walks through building a `/release-pr` skill that wraps `pk release --pr` with a Claude-drafted PR title and description. It pulls together everything in this doc: composing a `pk` command, preview-and-confirm, ask-then-act, the `gh`-with-fallback pattern, and `disable-model-invocation: true`.

### What it does

`pk release --pr` creates a pull request, but uses a hard-coded title (`Release <tag>`) and a `--fill` body (an auto-generated commit dump). For projects that want richer PR descriptions, you can layer a skill on top:

1. Run `pk release --pr --dry-run` to validate, show the user, ask to proceed.
2. Run `pk release --pr` to create the PR.
3. Read the diff between source and release branches.
4. Draft a PR title (≤70 chars) and body (Summary + Test plan).
5. Show the draft, let the user approve or edit.
6. Use `gh pr edit` to update the PR — fall back to printing for manual paste if `gh` is unavailable.

### The skill

Save this as `.claude/skills/release-pr/SKILL.md`:

```markdown
---
name: release-pr
description: Run pk release --pr and replace the PR title and body with a Claude-drafted summary
disable-model-invocation: true
---

This is a high-stakes workflow that pushes code and creates a pull request. The user must trigger this explicitly — never invoke automatically.

## Steps

1. **Preview the release.** Run:

   pk release --pr --dry-run

   Show the output to the user. If checks fail, stop and report. If they pass, ask for confirmation to proceed.

2. **Create the PR.** Run:

   pk release --pr

   The command prints the new PR URL on stderr.

3. **Find the new PR.** Run:

   gh pr view --json number,url,headRefName

   to get the PR number and URL. If `gh` fails, report the URL from step 2 and stop — the rest of this skill needs `gh`.

4. **Read the diff.** Determine the source branch from `git branch --show-current` and the release branch from `.pk.json` (`release.branch`). Then:

   git log <release-branch>..<source-branch> --oneline
   git diff <release-branch>...<source-branch>

5. **Draft a title and body.** Compose:

   - **Title** — under 70 characters, present tense, summarizes the release.
   - **Body** — a `## Summary` section with 1–3 bullets of what changed and why, followed by a `## Test plan` section with checkboxes for what should be verified.

6. **Show the draft and ask.** Present the proposed title and body to the user. Offer three choices: approve as-is, request edits, or cancel.

7. **Apply (or fall back).** If approved, write the body to a file with `mktemp -t pr-body`, pass the path with `--body-file`, and remove the file after `gh pr edit` returns:

   gh pr edit <number> --title "<title>" --body-file "$tmp"

   If `gh` fails, print the title and body so the user can paste manually, then report the PR URL.

## Notes

- Steps 1 and 6 are confirmation gates — never push or edit without explicit user OK.
- Use `--body-file` rather than `--body` to avoid shell escaping issues with multi-line content.
- If any `gh` command fails, fall through to printing instead of erroring out.
```

### Why disable-model-invocation: true

Without this flag, Claude could decide to invoke `/release-pr` on its own — for example, if the user says "I'm ready to ship this." Releases are high-stakes: they push code, create PRs, sometimes trigger CI/CD. The user should always trigger them explicitly. The flag forces `/release-pr` to be typed by the user; Claude can still suggest running it.

### Variations to consider

- **Add a label** — append `--label release` to the `gh pr edit` call if your project uses labels.
- **Pre-fill reviewers** — list reviewers directly in the SKILL.md prompt (e.g., "always add @user1, @user2 as reviewers"). The skill file is its own configuration — to change the list, edit the skill.
- **Use a template** — load `.github/pull_request_template.md` as the body skeleton before drafting.
- **Branch on flow** — if `release.branch` is not set, fall back to a simpler push-only flow.

## References

- [Claude Code Skills](https://code.claude.com/docs/en/skills.md) — full schema, frontmatter fields, advanced features.
- [Claude Code .claude directory](https://code.claude.com/docs/en/claude-directory.md) — file location and discovery rules.
- plankit's installed skills (`.claude/skills/`) — `changelog`, `init`, `preserve`, `release`, `review` as live examples.
