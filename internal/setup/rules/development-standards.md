---
description: Data-first, fail fast, consistency, two-pass, security, and debugging
---

# Development Standards

## Data-First, Model-First

- **Preserve the structure you were given.** When data has hierarchy, parse and maintain it. Let the data model drive the code, not the other way around.
- **Never flatten structured data into flat lists then reconstruct with heuristics** — the context is already lost.

## Fail Fast, No Silent Fallbacks

- **Surface errors clearly.** When something is missing or wrong, fail with a clear message. Never silently fall back to made-up defaults.

## All-or-Nothing Consistency

- **Update every related location together.** No partial renames, no half-updated contracts. If you change it in one place, change it everywhere.
- **Grep before done.** When fixing a pattern, grep the repo for all instances before considering it complete. One fix is not done until every occurrence is fixed.

## Two-Pass Code Generation

- **First pass:** Get it working — focus on correctness and completeness.
- **Second pass:** Review — look for DRY violations, missing abstractions, magic numbers, and unnecessary complexity. Refactor what you find.
- **Resist premature abstraction.** Three similar lines is better than the wrong abstraction. ([The Wrong Abstraction](https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction))

## Verification: Automated and Smoke

Work isn't done until both tiers pass:

- **Automated** — build, tests, and linter all pass. CI should enforce this.
- **Smoke** — a manual end-to-end check of the change, with specific commands and observable outcomes (file contents, stdout strings, exit codes, hook behavior). Include at least one negative case. Required whenever the change alters observable behavior — CLI output, scripts, tool integrations, file writes, or anything a user or downstream consumer can see. Skip for pure internal refactors.

Unit tests verify isolated behavior in mocks; smoke tests verify the wiring. The goal is "I actually ran this and saw the right thing" — not "the tests might catch it."

## Security

- **Never commit secrets** to version control. Use `.env` files in `.gitignore`.
- **Guard against** command injection, XSS, SQL injection, and other OWASP top 10 vulnerabilities.

## Debugging

- **Diagnostic scripts over rebuild cycles.** When debugging, create a minimal script that tests the specific issue. If you are about to do your second full rebuild, stop and write a diagnostic script instead.
