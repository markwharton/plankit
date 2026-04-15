---
name: init
description: Initialize project-specific CLAUDE.md conventions by analyzing the codebase
disable-model-invocation: true
---

Analyze this project and generate or refresh the **Project Conventions** section in CLAUDE.md.

Run this after `pk setup` to add project-specific conventions, or re-run anytime as the project evolves.

## Steps

1. Read the existing CLAUDE.md. If it does not exist, stop and tell the user to run `pk setup` first.
2. If a `## Project Conventions` section already exists, read it carefully — this is a refresh, not a blank slate. Preserve conventions that are still accurate, update what has changed, and add anything new.
3. Explore the project to identify:
   - Primary language(s) and framework(s)
   - Build system and test runner
   - Directory structure and file organization
   - Existing conventions visible in code (naming, patterns, configuration)
   - Business and domain rules embedded in application logic, if applicable (default values, calculation rules, workflow states, status transitions, business logic, UI behavior conventions, data safety constraints)
   - Domain model relationships and creation flows, if applicable (which entities relate to which, what entry points exist, what gets pre-filled)
   - CI/CD workflow files (`.github/workflows/`) — whether GitHub Actions are pinned to commit SHAs or use mutable tags, and whether Dependabot is configured for GitHub Actions updates
4. Ask the user three independent opt-in questions about pk features. Each is optional — "none" is a first-class answer. If all three are "none," no `.pk.json` is needed: guard becomes a no-op, release just pushes the current branch, changelog uses its default commit types.
   - **Protected branches (`pk guard`):** Are there branches that should never receive direct commits (e.g., `main`, `production`)? For push-to-main, trunk-based projects, answer "none."
   - **Release branch (`pk release`):** Should `pk release` merge your development branch into a separate release branch before pushing? If there's no separate release branch, answer "none."
   - **Changelog customization (`pk changelog`):** Do you want custom commit types beyond the defaults (`feat`, `fix`, `deprecate`, `revert`, `security`, `refactor`, `perf`, `docs`, `chore`, `test`, `build`, `ci`, `style`)? Most projects answer "no."
5. Also ask the user about the default development branch (e.g., `dev`, `main`, `develop`) so branch conventions can be documented even if no branches are protected.
6. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point. Group technical conventions and business/domain rules under separate subheadings. Only include a "never commit directly to X" convention if the user specified protected branches in step 4.
7. Show the proposed section to the user and ask for confirmation before writing.
8. Create or update `.pk.json` only for the features opted into in step 4. If all three were "none," skip this step — do not create an empty `.pk.json`. Otherwise include only the opted-in keys: `{"guard": {"branches": [...]}}`, `{"release": {"branch": "..."}}`, `{"changelog": {"types": [...]}}`. If `.pk.json` already exists, merge the keys — do not overwrite existing config. Sort top-level keys alphabetically.

## Rules

- **Append only.** Do not modify the Critical Rules section.
- If a `## Project Conventions` section already exists, replace it with the updated version — do not duplicate it.
- **Remove the pk SHA marker.** If the first line is `<!-- pk:sha256:... -->`, remove it. Once customized, the file is user-owned and the marker is stale.
- Keep conventions specific and actionable — not generic advice.
- Include the project's test command, build command, and any deployment patterns you discover.
- If the project uses `.pk.json` with configured commit types, include them in the conventions.
- For business rules, read into services, components, and pages — do not stop at file structure. Extract actual values, defaults, and logic constraints.
- If GitHub Actions use mutable tags (e.g., `@v4`), report this to the user as a security finding — mutable tags are vulnerable to supply chain attacks. If `.github/dependabot.yml` is missing or does not cover GitHub Actions, mention it as a way to keep pinned SHAs current. Include relevant conventions in the draft if the project has workflow files.
