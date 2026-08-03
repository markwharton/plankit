---
name: pk-configure
description: Configure plankit for this repository; detect the git setup, interview for branch and release policy, and write .pk.json
disable-model-invocation: true
---

Configure plankit for this repository: detect the current git setup, ask about branch protection and release policy, and write `.pk.json`.

Run this after `pk setup`, or re-run anytime the branch or release policy changes.

## Steps

1. Detect the current git setup before asking. Run `git branch --list`, `git branch -r`, and `git tag --list 'v*' --sort=-v:refname`; when the default branch has an upstream, also run `git log origin/<branch>..<branch> --oneline`. Also check `.github/workflows/` for actions pinned to mutable tags and whether `.github/dependabot.yml` covers GitHub Actions. Summarize the findings in a sentence (e.g. "main only, 2 unpushed commits, no version tags") so the next step's questions read as a transition from this setup, not a quiz.
2. Ask the user about project configuration:
   - What is the default branch for development? (e.g., `main`, `develop`)
   - Are there branches that should never receive direct commits? (e.g., `main`, `production`)
   - Should releases merge into a separate branch before pushing? Which one? (e.g., `main`)
   - Custom changelog commit types beyond the defaults, or use the defaults?
3. Create or update `.pk.json` based on step 2 answers. If the user specified no protected branches, no release branch, and no custom changelog types, skip this step; do not create an empty `.pk.json`. Otherwise include only the opted-in keys: `{"guard": {"branches": [...]}}`, `{"release": {"branch": "..."}}`, `{"changelog": {"types": [...]}}`. If `.pk.json` already exists, merge the keys; do not overwrite existing config. **Field-merge the `guard` object:** `pk setup` writes `guard.mode` and `guard.push` (and `preserve.mode`) into `.pk.json`, so when you add `guard.branches` merge it into the existing `guard` object and keep those mode fields; never replace the whole object. Sort top-level keys alphabetically.
4. Offer the working branch if the chosen setup implies one that does not exist (e.g. releases merge into `main` but step 1 found only `main`). If the branching point is ambiguous because the default branch is ahead of its origin, show the `git log origin/<branch>..<branch> --oneline` output and ask:
   - Should the new branch start from local `<branch>` or from `origin/<branch>`?
   Then preview the exact commands and run them only after the user confirms:
   ```bash
   git branch develop <start-ref>
   git switch develop
   git push -u origin develop
   ```
5. Offer a baseline nudge if versioned releases are planned. If the user opted into release or changelog customization in step 2 (non-"none" answer to either) and step 1 found no semver tag, tell the user: "No version tags found. To anchor pk changelog: pk setup --baseline --push. Use --at <ref> to fold prior commits into the first changelog entry, or omit it to start fresh from HEAD." This is advisory; do not run the command from the skill. Remote state changes belong in explicit user-invoked commands.
6. Close with the dashboard and the conventions handoff. Suggest the user run `pk status`: its Readiness section confirms the new setup or lists exactly what is still missing (baseline tag, branches on origin). Then, if CLAUDE.md has no `## Project Conventions` section, suggest the built-in `/init` skill for conventions discovery, and pass on the tip it needs: prompt it to read into services and components and extract business rules (actual defaults, calculation rules, workflow states), which codebase analysis does not surface on its own.

## Rules

- **Exit plan mode first.** If you are in plan mode when this skill is invoked, exit plan mode immediately before doing anything else. This skill executes commands; it does not need a plan.
- **Write .pk.json immediately after the config questions.** Do not defer it behind the branch or baseline steps. The release workflow depends on this file; without it, the user's release-branch answer never takes effect and `pk release` stays in trunk flow.
- **Branch creation runs only after explicit confirmation.** Step 4 previews the exact commands and asks first; the push publishes the branch, and that is the user's decision to make per command, never the skill's to assume. If the user declines, leave the previewed commands for them to run later and continue with the remaining steps.
- **The branching point is the user's call when commits are unpushed.** `git branch develop <branch>` carries the unpushed commits onto the new working branch; `git branch develop origin/<branch>` leaves them on the default branch until the user pushes. Never pick one silently; when local and origin match, the local branch is the start ref and no question is needed.
- **Configuration mapping:** Protected branches configures `guard.branches`, release branch configures `release.branch`, custom changelog types configures `changelog.types`. Default commit types: `build`, `chore`, `ci`, `deprecate`, `docs`, `feat`, `fix`, `perf`, `plan` (hidden), `refactor`, `revert`, `security`, `style`, `test`.
- If GitHub Actions use mutable tags (e.g., `@v4`), report this to the user as a security finding; mutable tags are vulnerable to supply chain attacks. If `.github/dependabot.yml` is missing or does not cover GitHub Actions, mention it as a way to keep pinned SHAs current.
