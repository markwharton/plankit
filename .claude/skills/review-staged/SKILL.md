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
