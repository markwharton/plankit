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
- **allowed-tools** — list of tools the skill can use without asking for permission while it's active. For example, `allowed-tools: Bash(pk:*)` lets a `pk`-wrapping skill run without prompting on each `pk` command. Use this to make skills self-contained — they work even in projects where `settings.json` doesn't have a matching permission entry. plankit's installed skills (`/changelog`, `/preserve`, `/release`, `/ship`) use this pattern.

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

Skills become much more powerful when they involve the user in the loop. The five skills plankit installs (`.claude/skills/changelog/`, `init/`, `preserve/`, `release/`, `ship/`) are working examples worth reading.

### Preview and confirm

Before destructive or hard-to-reverse actions, run the dry-run version, show the user what will happen, and ask for confirmation. The `/changelog` and `/release` skills use this pattern:

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

Preview-environment workflows — [Azure Static Web Apps](https://learn.microsoft.com/en-us/azure/static-web-apps/preview-environments), [Netlify](https://docs.netlify.com/site-deploys/deploy-previews/), [Vercel](https://vercel.com/docs/deployments/preview-deployments), GitHub Pages preview actions — fire CI/CD when a pull request opens, build a preview site, and post a URL to the PR. Production deploys fire when the PR merges to the default branch. Nothing about tags or version numbers — this is a different mental model from `pk release`, which is local-merge-and-tag.

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

   If source and target are the same branch, stop and report: "you're on the default branch — create a feature branch first."

3. **Check for existing PR.** Run:

   gh pr view --json url 2>/dev/null

   If a PR already exists for this branch, report the URL and stop — no need to create another.

4. **Preview.** Tell the user: "I'm about to push <source> to origin and open a PR targeting <target>. Proceed?" Wait for an explicit yes before continuing. If the user declines, stop and report.

5. **Push.** Run:

   git push -u origin <source>

   If the push fails, stop and report the error. Do not force.

6. **Create the PR.** Run:

   gh pr create --base <target> --head <source> --fill

   On success, report the PR URL from the command output.

   If `gh` fails: get the remote URL via `git remote get-url origin`, derive `owner/repo`, and print a compare URL like `https://github.com/<owner>/<repo>/compare/<target>...<source>` so the user can create the PR in a browser.

## Notes

- Never push a dirty tree.
- Running `/preview` on the default branch is caught early — create a feature branch first.
- Running `/preview` twice doesn't create duplicate PRs.
- Never push and open a PR without explicit user confirmation.
- `gh pr create --fill` uses the commit messages as the PR title and body. For a richer description, layer a second skill on top that reads the diff and drafts a summary.
````

### Why disable-model-invocation: true

Without this flag, Claude could invoke `/preview` on its own — for example, if the user says "I'm ready to share this." Pushing code and opening a PR is something the user should always trigger explicitly; `/preview` touches the remote and creates a visible artifact. The flag forces `/preview` to be typed by the user; Claude can still suggest running it.

### Variations to consider

- **Rich PR description** — layer a second skill on top that runs `git log <target>..<source>` and `git diff <target>...<source>`, drafts a summary, and uses `gh pr edit --body-file` to replace the placeholder body. Keep the two concerns separated: `/preview` creates the PR, the other skill fills it in.
- **Target a non-default branch** — some teams preview against `staging` rather than the repo default. Hard-code the target in the skill file or accept it via `argument-hint: [target-branch]`.
- **Preview a specific commit** — accept an argument and push that ref instead of `HEAD`.

## Tutorial: Build a /rollback skill

This walks through building a `/rollback` skill that reverts a bad commit via a pull request. It complements `/preview` for trunk-based workflows where the recovery path matters as much as the deploy path.

### What it does

Trunk-based teams push directly to the default branch and rely on rollback when something breaks. The flow that `/rollback` automates:

1. Show recent commits so the user can identify the bad one.
2. Preview the revert without committing — abort on conflicts.
3. Show the user what will change and ask to proceed.
4. Create a rollback branch, commit the revert, push, and open a PR.

The PR-based approach is the safe default: CI validates the revert before it hits production, and the change is visible to the team. Like `/preview`, this is a thin composition of git commands — a skill is the right level.

### The skill

Save this as `.claude/skills/rollback/SKILL.md`:

````markdown
---
name: rollback
description: Revert a bad commit via a branch and pull request for trunk-based recovery
disable-model-invocation: true
allowed-tools: Bash(git:*), Bash(gh:*)
---

This is a high-stakes workflow that reverts a commit and opens a pull request. The user must trigger this explicitly — never invoke automatically.

## Steps

1. **Check working tree.** Run:

   git status --porcelain

   If the output is non-empty, stop and report: "working tree is not clean — commit or stash changes first."

2. **Show recent commits.** Run:

   git log --oneline -10

   Ask the user which commit to revert. Default to HEAD if they confirm the most recent commit is the problem.

3. **Check for existing rollback.** Check if a branch named `rollback/<short-sha>` already exists:

   git branch --list rollback/<short-sha>

   If it exists, check for an open PR:

   gh pr view rollback/<short-sha> --json url 2>/dev/null

   If a PR exists, report the URL and stop. If the branch exists without a PR, report it and ask the user how to proceed.

4. **Check for merge commit.** Run:

   git show --no-patch --pretty=%P <commit>

   If multiple parents are returned, inform the user this is a merge commit and ask: "Revert relative to parent 1 (mainline)?" If confirmed, use `-m 1` in subsequent revert commands. If not confirmed, stop.

5. **Preview the revert.** Run:

   git revert --no-commit <commit>

   If this produces conflicts:
   - Run `git status` and report the conflicts.
   - Run `git revert --abort` to clean up.
   - Stop — do not continue to confirmation.

   Show the impact:

   git diff --stat

   Then clean up the preview without `reset --hard`:

   git restore --staged .
   git restore .

6. **Confirm.** Tell the user: "This will create a rollback branch, revert <commit> (<subject>), push it, and open a PR. Proceed?" Wait for an explicit yes. If the user declines, stop.

7. **Create rollback branch and revert.** Run:

   git checkout -b rollback/<short-sha>
   git revert <commit>

   If conflicts occur during the real revert, report them and stop — do not auto-resolve.

8. **Push and open PR.** Run:

   git push -u origin rollback/<short-sha>
   gh pr create --base <source-branch> --head rollback/<short-sha> --fill

   On success, report the PR URL.

   If `gh` fails: derive `owner/repo` from `git remote get-url origin` and print a compare URL so the user can create the PR in a browser.

   Switch back to the source branch:

   git checkout <source-branch>

## Notes

- Never revert on a dirty tree.
- Running `/rollback` twice for the same commit doesn't create duplicate branches or PRs.
- The preview shows impact without committing. Conflicts abort cleanly.
- Never push a revert without explicit user confirmation.
- Never auto-resolve conflicts — stop and report.
- Reverts add new commits — they do not rewrite history.
````

### Why this complements /preview

`/preview` and `/rollback` are two sides of the same workflow: preview gets code into production via PR, rollback gets it out via PR. Together they cover the trunk-based cycle — deploy forward, recover backward. Both go through CI before reaching production. Neither requires branch protection, changelogs, or release tags.

### Variations to consider

- **Direct push for emergencies** — when production is completely down and CI latency is unacceptable, skip the branch and PR: `git revert <commit> && git push`. This trades visibility for speed. Use only when the outage cost exceeds the risk of an unvalidated revert.
- **Revert a range** — for cascading failures, accept a range like `<good>..<bad>` and revert each commit in reverse order.
- **CI status check** — before reverting, run `gh run list --limit 5` to show recent CI status. This helps confirm which commit broke the build.

## References

- [Claude Code Skills](https://code.claude.com/docs/en/skills) — full schema, frontmatter fields, advanced features.
- [Claude Code .claude directory](https://code.claude.com/docs/en/claude-directory) — file location and discovery rules.
- plankit's installed skills (`.claude/skills/`) — `changelog`, `init`, `preserve`, `release`, `ship` as live examples.
