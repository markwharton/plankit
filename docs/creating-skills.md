# Creating skills

Skills are reusable prompts that Claude Code discovers automatically. You package one as a markdown file with YAML frontmatter, and users invoke it with `/<skill-name>`. They're how you turn a repeatable workflow — a code review checklist, a deploy procedure, a session opener — into a one-command muscle.

Custom slash commands have been merged into skills in Claude Code. Skills are the forward-looking approach.

> Authoritative reference: [Claude Code Skills](https://code.claude.com/docs/en/skills)

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

See the [official Skills reference](https://code.claude.com/docs/en/skills) for the full schema (paths, hooks, model, subagent forking, and more).

## Your first skill

A minimal skill that reviews staged changes:

```markdown
---
name: review-staged
description: Review staged git changes for bugs, security issues, and style violations
---

Review the changes currently staged for commit:

`git diff --cached`

If the diff is empty, report that no changes are staged and stop.

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

## Tutorial: Build a /preview skill

This walks through building a `/preview` skill that pushes the current branch and opens a pull request for preview-environment workflows. It pulls together everything in this doc: composing `git` and `gh`, preview-and-confirm, ask-then-act, the `gh`-with-fallback pattern, and `disable-model-invocation: true`.

### What it does

Preview-environment workflows — Azure Static Web Apps, Netlify, Vercel, GitHub Pages preview actions — fire CI/CD when a pull request opens, build a preview site, and post a URL to the PR. Production deploys fire when the PR merges to the default branch. Nothing about tags or version numbers — this is a different mental model from `pk release`, which is local-merge-and-tag.

The flow that `/preview` automates:

1. Verify the working tree is clean.
2. Find the current source branch and the repo's default target branch.
3. Show the user what will happen and ask to proceed.
4. Push the source branch.
5. Open a PR via `gh pr create --fill`.
6. If `gh` is missing, print a compare URL for manual creation.

plankit doesn't ship a `pk preview` command because the whole thing is about fifteen lines of SKILL.md — a thin composition of `git push` and `gh pr create`. Turning it into a binary would add ceremony without adding capability. A skill is the right level.

### The skill

Save this as `.claude/skills/preview/SKILL.md`:

````markdown
---
name: preview
description: Push the current branch and open a pull request for preview environments
disable-model-invocation: true
allowed-tools: Bash(git:*), Bash(gh:*)
---

This is a high-stakes workflow that pushes code and creates a pull request. The user must trigger this explicitly — never invoke automatically.

## Steps

1. **Check working tree.** Run:

   git status --porcelain

   If the output is non-empty, stop and report: "working tree is not clean — commit or stash changes first."

2. **Find source and target branches.** Source is the current branch:

   git branch --show-current

   Target is the repo's default branch:

   gh repo view --json defaultBranchRef --jq .defaultBranchRef.name

   If `gh` is not available, ask the user for the target branch name.

3. **Preview.** Tell the user: "I'm about to push <source> to origin and open a PR targeting <target>. Proceed?" Wait for an explicit yes before continuing. If the user declines, stop and report.

4. **Push.** Run:

   git push origin <source>

   If the push fails, stop and report the error. Do not force.

5. **Create the PR.** Run:

   gh pr create --base <target> --head <source> --fill

   On success, report the PR URL from the command output.

   If `gh` fails: get the remote URL via `git remote get-url origin`, derive `owner/repo`, and print a compare URL like `https://github.com/<owner>/<repo>/compare/<target>...<source>` so the user can create the PR in a browser.

## Notes

- Step 1 is a hard gate — never push a dirty tree.
- Step 3 is a confirmation gate — never push and open a PR without explicit user OK.
- `gh pr create --fill` uses the commit messages as the PR title and body. For a richer description, layer a second skill on top that reads the diff and drafts a summary.
````

### Why disable-model-invocation: true

Without this flag, Claude could invoke `/preview` on its own — for example, if the user says "I'm ready to share this." Pushing code and opening a PR is something the user should always trigger explicitly; `/preview` touches the remote and creates a visible artifact. The flag forces `/preview` to be typed by the user; Claude can still suggest running it.

### Variations to consider

- **Rich PR description** — layer a second skill on top that runs `git log <target>..<source>` and `git diff <target>...<source>`, drafts a summary, and uses `gh pr edit --body-file` to replace the placeholder body. Keep the two concerns separated: `/preview` creates the PR, the other skill fills it in.
- **Target a non-default branch** — some teams preview against `staging` rather than the repo default. Hard-code the target in the skill file or accept it via `argument-hint: [target-branch]`.
- **Preview a specific commit** — accept an argument and push that ref instead of `HEAD`.

## References

- [Claude Code Skills](https://code.claude.com/docs/en/skills) — full schema, frontmatter fields, advanced features.
- [Claude Code .claude directory](https://code.claude.com/docs/en/claude-directory) — file location and discovery rules.
- plankit's installed skills (`.claude/skills/`) — `changelog`, `init`, `preserve`, `release` as live examples.
