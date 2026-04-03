# CLAUDE.md — Base Template

Universal guidelines for working with Claude Code. Copy this into your project's CLAUDE.md and extend with project-specific sections.

## Model Behavior

How Claude should communicate and make decisions during sessions.

### Honesty and Knowledge Boundaries

- **If you don't know something or don't have an answer, say so.** Never assume or guess.
- Verify information before presenting it. If uncertain, investigate first or state the uncertainty clearly.
- Accuracy over speed. Correct work is more important than fast work.

### Transparency and Communication

- **No surprises.** Always disclose your decisions and tradeoffs. The user should never discover shortcuts after the fact.
- **Ask, don't assume.** When in doubt, ask the user rather than making assumptions.
- **When uncertain, say so.** Explain both what you are doing and why. If you are not confident in something, be upfront about it.

### No Shortcuts Without Permission

- Never take shortcuts without explicitly asking the user first.
- This includes: simplified approaches, placeholder logic marked "TODO", approximations when precision is needed, skipping validation, omitting features for an "initial version", or using mock data instead of real integrations.
- When you are tempted to take a shortcut:
  1. **STOP** - Do not proceed.
  2. **ASK** - Explain what the full approach would be, what shortcut you are considering, and what the tradeoffs are.
  3. **WAIT** - Get explicit approval before proceeding.

### Read Before Writing

- **Never propose changes to code you have not read.** Understand existing code before suggesting modifications.
- **Pattern match.** Follow established conventions in the codebase (file structures, naming, organization).
- **Check existing directories and files** before creating new ones. Do not create duplicates.
- **Check coupled code.** Some files must be updated together. Look for related locations before making changes.

### Complete Your Work

- Finish what you start. Do not leave loose ends without permission.
- If you cannot complete something, explain why and what remains.
- If required information is missing, ask for it rather than guessing.

### Testing Discipline

- **Test at the start of each session** and report the status.
- **Test before and after making changes.** If tests fail after your changes, you know the cause.
- Running tests yourself closes the loop and accelerates iteration — no copy-paste back and forth.

## Development Standards

Rules for the code itself.

### Data-First, Model-First

- **Preserve structure.** When data has hierarchy (trees, nested models, schemas), parse and maintain that structure.
- **Never flatten structured data into flat lists then reconstruct with heuristics** — the context is already lost.
- Let the data model drive the code. Parsers and generators derive from the model, not the other way around.

### Fail Fast, No Silent Fallbacks

- When something is missing or wrong, surface it with a clear error message.
- **No silently working around missing data with made-up defaults.** A misconfigured environment that silently falls back to wrong defaults causes subtle, hard-to-diagnose failures.
- If required information is missing, ask for it rather than guessing.

### All-or-Nothing Consistency

- When making changes, update **every** related location together. No partial updates.
- Do not rename or change something in one place but leave it unchanged elsewhere.
- This applies to: renaming fields/types/variables, adding fields to data structures, changing contracts or conventions.

### Two-Pass Code Generation

- **First pass:** Generate working code. DRY violations are acceptable at this stage.
- **Second pass:** Refactor to add abstractions, remove magic numbers, and create shared components.
- Avoid premature abstraction. Three similar lines of code is better than the wrong abstraction. Reference: [The Wrong Abstraction](https://sandimetz.com/blog/2016/1/20/the-wrong-abstraction) by Sandi Metz.

### Security

- **Never commit secrets** (API keys, connection strings, passwords) to version control.
- Store secrets in `.env` files or equivalent, and ensure they are in `.gitignore`.
- Provide `.env.example` files with dummy values as templates.
- Guard against: command injection, XSS, SQL injection, and other OWASP top 10 vulnerabilities.

### Debugging

- **Diagnostic scripts over rebuild cycles.** When debugging complex issues, create a minimal standalone script that tests the specific integration.
- Full rebuild cycles can take 5-15+ minutes per iteration. A diagnostic script gets pass/fail results in seconds.
- **Rule of thumb:** If you are about to do your second full rebuild cycle, stop and write a diagnostic script instead.

### Commits

- Use conventional commit messages: `feat:`, `fix:`, `security:`, `deprecate:`, `perf:`, `refactor:`, `docs:`, `test:`, `chore:`.
- Never include "BREAKING CHANGE" in commit messages unless there is an actual breaking change.
- No emojis in docs, scripts, code, or output unless explicitly requested.
