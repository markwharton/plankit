---
name: smoke-test
description: Run project smoke tests and report results
---

Run a quick smoke test of the project and report results.

## Steps

1. Identify the project's test runner (look for `Makefile`, `package.json`, `go.mod`, etc.).
2. Run the test suite.
3. Report results as a summary table:

```
| Area | Status | Notes |
|------|--------|-------|
| Build | PASS/FAIL | details |
| Tests | PASS/FAIL | details |
| Lint | PASS/FAIL | details |
```

If any step fails, report the failure clearly and stop — do not attempt to fix it unless asked.
