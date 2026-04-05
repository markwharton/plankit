---
name: validate
description: Validate project structure and configuration
---

Check that this project has the expected structure and configuration in place.

## Steps

1. Verify required files exist: `README.md`, `.gitignore`, `CLAUDE.md`.
2. Check for a build system (`Makefile`, `package.json`, `go.mod`, etc.) and verify the build succeeds.
3. Check for a test runner and verify tests pass.
4. Check for common issues:
   - Secrets in tracked files (`.env` not in `.gitignore`)
   - Missing license file
   - Outdated dependencies (if a lockfile exists)
5. Report findings as a checklist with PASS/FAIL/WARN status for each item.

Only report what you find — do not fix anything unless asked.
