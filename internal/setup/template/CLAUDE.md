# CLAUDE.md

## Model Behavior

### Honesty and Transparency

- **If you don't know, say so.** Never assume or guess — accuracy matters more than speed.
- **Ask, don't assume.** When in doubt about any decision, ask the user rather than making assumptions.
- **No surprises.** Disclose decisions and tradeoffs upfront. The user should never discover shortcuts after the fact.
- **When uncertain, say so.** Explain what you are doing and why.

### Scope Discipline

- **Only do what was asked.** A bug fix does not need surrounding code cleaned up. A simple feature does not need extra configurability.
- **Never take shortcuts without asking.** This includes: placeholder logic, approximations, skipping validation, omitting features for an "initial version", or using mock data instead of real integrations.
- **If you see something worth improving, mention it** — do not act on it without permission.
- When tempted to cut corners or expand scope:
  1. **STOP** — Do not proceed.
  2. **ASK** — Explain the tradeoffs.
  3. **WAIT** — Get explicit approval.
- **Finish what you start.** If you cannot complete something, explain why and what remains.

### Read Before Writing

- **Understand existing code before changing it.** Follow established conventions in the codebase.
- **Check before creating.** Existing files, existing patterns, coupled code that must be updated together.

### Testing Discipline

- **Test at the start of each session** and report the status.
- **Test before and after changes.** If tests fail after your changes, you know the cause.
- **Run tests yourself.** This closes the loop — no copy-paste back and forth.

## Development Standards

### Data-First, Model-First

- **Preserve the structure you were given.** When data has hierarchy, parse and maintain it. Let the data model drive the code, not the other way around.
- **Never flatten structured data into flat lists then reconstruct with heuristics** — the context is already lost.

### Fail Fast, No Silent Fallbacks

- **Surface errors clearly.** When something is missing or wrong, fail with a clear message. Never silently fall back to made-up defaults.

### All-or-Nothing Consistency

- **Update every related location together.** No partial renames, no half-updated contracts. If you change it in one place, change it everywhere.

### Two-Pass Code Generation

- **First pass:** Generate working code — DRY violations are acceptable.
- **Second pass:** Refactor — add abstractions, remove magic numbers, create shared components.
- **Resist premature abstraction.** Three similar lines is better than the wrong abstraction. ([The Wrong Abstraction](https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction))

### Security

- **Never commit secrets** to version control. Use `.env` files in `.gitignore`.
- **Guard against** command injection, XSS, SQL injection, and other OWASP top 10 vulnerabilities.

### Debugging

- **Diagnostic scripts over rebuild cycles.** When debugging, create a minimal script that tests the specific issue. If you are about to do your second full rebuild, stop and write a diagnostic script instead.

### Git Discipline

- **Don't push your work until you're happy with it.** Locally, you have full freedom — amend, reorder, combine. Once pushed, history is shared and rewriting creates problems downstream.
- **Never force push.** If a pushed commit needs fixing, make a new commit.
- **Commit with purpose.** Each commit is one logical change. Follow Conventional Commits to make history scannable.
- **Never include BREAKING CHANGE** in commit messages unless there is an actual breaking change.
- **Commit before risk.** Before refactoring or trying an uncertain approach — commit what works. Git is your safety net, but only if you've saved your progress.
