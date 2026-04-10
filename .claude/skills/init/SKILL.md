---
name: init
description: Initialize project-specific CLAUDE.md conventions by analyzing the codebase
disable-model-invocation: true
pk_sha256: a3cefa2feec9c4a150ee0c25e199caa932a5bcc46f7bcc4e34549fad1f89eb91
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
4. Ask the user about branch conventions:
   - What is the default branch for development? (e.g., `dev`, `main`, `develop`)
   - Are there branches that should never receive direct commits? (e.g., `main`, `production`)
   - Which branch should releases be pushed to? (e.g., `main`)
5. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point. Group technical conventions and business/domain rules under separate subheadings. If the user specified protected branches, include a branch convention (e.g., "All changes go through `dev` — never commit directly to `main`").
6. Show the proposed section to the user and ask for confirmation before writing.
7. If the user specified protected branches or a release branch, create or update `.pk.json`:
   ```json
   {
     "guard": {
       "protectedBranches": ["main"]
     },
     "release": {
       "branch": "main"
     }
   }
   ```
   If `.pk.json` already exists, merge the keys — do not overwrite existing config. Sort top-level keys alphabetically.

## Rules

- **Append only.** Do not modify the Critical Rules section.
- If a `## Project Conventions` section already exists, replace it with the updated version — do not duplicate it.
- **Remove the pk SHA marker.** If the first line is `<!-- pk:sha256:... -->`, remove it. Once customized, the file is user-owned and the marker is stale.
- Keep conventions specific and actionable — not generic advice.
- Include the project's test command, build command, and any deployment patterns you discover.
- If the project uses `.pk.json`, include the configured commit types.
- For business rules, read into services, components, and pages — do not stop at file structure. Extract actual values, defaults, and logic constraints.
