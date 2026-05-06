---
name: init
description: Initialize project-specific CLAUDE.md conventions by analyzing the codebase
disable-model-invocation: true
---

Analyze this project and generate or refresh the **Project Conventions** section in CLAUDE.md.

Run this after `pk setup` to add project-specific conventions, or re-run anytime as the project evolves.

## Steps

1. Read the existing CLAUDE.md. If it does not exist, create it with the Critical Rules header below before proceeding.
   ```markdown
   # CLAUDE.md

   IMPORTANT: Follow these rules at all times.

   ## Critical Rules

   - NEVER take shortcuts without asking. STOP, ASK, WAIT for approval.
   - NEVER force push. Make a new commit to fix mistakes.
   - NEVER commit secrets to version control.
   - Only do what was asked. No scope creep.
   - Understand existing code before changing it.
   - If you don't know, say so. Never guess.
   - Test before and after every change.
   - Surface errors clearly. No silent fallbacks.
   ```
2. If a `## Project Conventions` section already exists, read it carefully — this is a refresh, not a blank slate. Preserve conventions that are still accurate, update what has changed, and add anything new.
3. Explore the project to identify:
   - Primary language(s) and framework(s)
   - Build system and test runner
   - Directory structure and file organization
   - Existing conventions visible in code (naming, patterns, configuration)
   - Business and domain rules embedded in application logic, if applicable (default values, calculation rules, workflow states, status transitions, business logic, UI behavior conventions, data safety constraints)
   - Domain model relationships and creation flows, if applicable (which entities relate to which, what entry points exist, what gets pre-filled)
   - CI/CD workflow files (`.github/workflows/`) — whether GitHub Actions are pinned to commit SHAs or use mutable tags, and whether Dependabot is configured for GitHub Actions updates
4. Ask the user about project configuration:
   - What is the default branch for development? (e.g., `main`, `develop`)
   - Are there branches that should never receive direct commits? (e.g., `main`, `production`)
   - Should releases merge into a separate branch before pushing? Which one? (e.g., `main`)
   - Custom changelog commit types beyond the defaults, or use the defaults?
5. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point. Group technical conventions and business/domain rules under separate subheadings. Only include a "never commit directly to X" convention if the user specified protected branches in step 4.
6. Show the proposed section to the user and ask for confirmation before writing.
7. Create or update `.pk.json` based on step 4 answers. If the user specified no protected branches, no release branch, and no custom changelog types, skip this step — do not create an empty `.pk.json`. Otherwise include only the opted-in keys: `{"guard": {"branches": [...]}}`, `{"release": {"branch": "..."}}`, `{"changelog": {"types": [...]}}`. If `.pk.json` already exists, merge the keys — do not overwrite existing config. Sort top-level keys alphabetically.
8. Offer a baseline nudge if versioned releases are planned. If the user opted into release or changelog customization in step 4 (non-"none" answer to either), check for a version tag by running `git tag --list 'v*' --sort=-v:refname`. If the output is empty or nothing parses as semver, tell the user: "No version tags found. To anchor `pk changelog`, run `pk setup --baseline --push`. Use `--at <ref>` to fold prior commits into the first changelog entry." This is advisory — do not run the command from the skill. Remote state changes belong in explicit user-invoked commands.

## Rules

- **Append only.** Do not modify the Critical Rules section.
- If a `## Project Conventions` section already exists, replace it with the updated version — do not duplicate it.
- **Remove the pk SHA marker.** If the first line is `<!-- pk:sha256:... -->`, remove it. Once customized, the file is user-owned and the marker is stale.
- Keep conventions specific and actionable — not generic advice.
- Include the project's test command, build command, and any deployment patterns you discover.
- If the project uses `.pk.json` with configured commit types, include them in the conventions.
- For business rules, read into services, components, and pages — do not stop at file structure. Extract actual values, defaults, and logic constraints.
- **Configuration mapping:** Protected branches configures `guard.branches`, release branch configures `release.branch`, custom changelog types configures `changelog.types`. Default commit types: `build`, `chore`, `ci`, `deprecate`, `docs`, `feat`, `fix`, `perf`, `plan` (hidden), `refactor`, `revert`, `security`, `style`, `test`.
- If GitHub Actions use mutable tags (e.g., `@v4`), report this to the user as a security finding — mutable tags are vulnerable to supply chain attacks. If `.github/dependabot.yml` is missing or does not cover GitHub Actions, mention it as a way to keep pinned SHAs current. Include relevant conventions in the draft if the project has workflow files.
