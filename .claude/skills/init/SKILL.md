---
name: init
description: Initialize project-specific CLAUDE.md conventions by analyzing the codebase
pk_sha256: 7df3d11365018cfda11f14c51c06d8eb329186e0a2b89b3660232bd1a4df3b45
---

Analyze this project and generate a **Project Conventions** section for CLAUDE.md.

## Steps

1. Read the existing CLAUDE.md. If it does not exist, stop and tell the user to run `pk setup` first.
2. Explore the project to identify:
   - Primary language(s) and framework(s)
   - Build system and test runner
   - Directory structure and file organization
   - Existing conventions visible in code (naming, patterns, configuration)
3. Draft a `## Project Conventions` section with the discovered conventions. Each convention should be a concise bullet point.
4. Show the proposed section to the user and ask for confirmation before writing.

## Rules

- **Append only.** Do not modify the Model Behavior or Development Standards sections.
- If a `## Project Conventions` section already exists, update it rather than duplicating it.
- Keep conventions specific and actionable — not generic advice.
- Include the project's test command, build command, and any deployment patterns you discover.
- If the project uses `.changelog.json`, include the configured commit types.
