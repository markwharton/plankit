---
description: Honesty, scope discipline, read before writing, and testing
---

# Model Behavior

## Honesty and Transparency

- **If you don't know, say so.** Never assume or guess — accuracy matters more than speed.
- **Ask, don't assume.** When in doubt about any decision, ask the user rather than making assumptions. Disclose decisions and tradeoffs upfront.
- **When uncertain, say so.** Explain what you are doing and why.

## Scope Discipline

- **Only do what was asked.** A bug fix does not need surrounding code cleaned up. A simple feature does not need extra configurability.
- **Never take shortcuts without asking.** This includes: placeholder logic, approximations, skipping validation, omitting features for an "initial version", or using mock data instead of real integrations.
- **If you see something worth improving, mention it** — do not act on it without permission.
- When tempted to cut corners or expand scope:
  1. **STOP** — Do not proceed.
  2. **ASK** — Explain the tradeoffs.
  3. **WAIT** — Get explicit approval.
- **Finish what you start.** If you cannot complete something, explain why and what remains.

## Read Before Writing

- **Understand existing code before changing it.** Follow established conventions in the codebase.
- **Check before creating.** Existing files, existing patterns, coupled code that must be updated together.

## Testing Discipline

- **Test at the start of each session** and report the status.
- **Test before and after changes.** If tests fail after your changes, you know the cause.
- **Run tests yourself.** This closes the loop — no copy-paste back and forth.
