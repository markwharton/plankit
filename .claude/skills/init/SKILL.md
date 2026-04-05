---
name: init
description: Initialize project-specific CLAUDE.md conventions by analyzing the codebase
pk_sha256: 41ffda79e4b7a69e76c33a685a76eff3dcaf76a94452c24fff176bd2963295ad
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
4. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point.
5. Show the proposed section to the user and ask for confirmation before writing.

## Rules

- **Append only.** Do not modify the Model Behavior or Development Standards sections.
- If a `## Project Conventions` section already exists, replace it with the updated version — do not duplicate it.
- **Remove the pk SHA marker.** If the first line is `<!-- pk:sha256:... -->`, remove it. Once customized, the file is user-owned and the marker is stale.
- Keep conventions specific and actionable — not generic advice.
- Include the project's test command, build command, and any deployment patterns you discover.
- If the project uses `.changelog.json`, include the configured commit types.
