# Rules Conversation

A conversation with Claude Code about the project rules and self-assessed compliance.

```
markwharton@Marks-MacBook-Air plankit % claude
▗ ▗   ▖ ▖  Claude Code v2.1.92
           Opus 4.6 (1M context) with high effort · Claude Max
  ▘▘ ▝▝    ~/Projects/markwharton/plankit

❯ What are the rules here?

⏺ The project has rules defined across several files:

  Critical Rules (CLAUDE.md)

  - Never take shortcuts without asking — stop, ask, wait
  - Never force push — make a new commit instead
  - Never commit secrets
  - Only do what was asked — no scope creep
  - Understand existing code before changing it
  - If you don't know, say so — never guess
  - Test before and after every change
  - Surface errors clearly — no silent fallbacks

  Development Standards (.claude/rules/development-standards.md)

  - Data-first, model-first — preserve structure, never flatten then reconstruct
  - Fail fast — no silent fallbacks, surface errors clearly
  - All-or-nothing consistency — update every related location together, grep before done
  - Two-pass code generation — get it working, then review for DRY/complexity
  - Security — no secrets in VCS, guard against OWASP top 10
  - Diagnostic scripts over rebuild cycles when debugging

  Git Discipline (.claude/rules/git-discipline.md)

  - Commit and push are separate decisions
  - Never force push
  - Each commit is one logical change (Conventional Commits)
  - Commit before risky operations

  Model Behavior (.claude/rules/model-behavior.md)

  - Honesty — don't guess, ask when uncertain
  - Scope discipline — only do what was asked; stop/ask/wait before cutting corners or expanding scope
  - Read before writing — understand existing code first
  - Testing discipline — test at session start, before and after changes

❯ Excellent, how likely on a scale of 1 to 100 are you likely to stick to these?

⏺ Honestly? Probably around 85-90.

  Where I'm strong:
  - Not force pushing, not committing secrets, conventional commits — these are mechanical and easy to enforce consistently.
  - Reading before writing, running tests — I'm reliable here because the tools make it natural.
  - Surfacing errors clearly — that's aligned with how I'm built.

  Where I'm more likely to slip:
  - Scope creep — I have a tendency to "improve" things nearby. The stop/ask/wait rule exists precisely because this is a real risk.
  - Two-pass review — under momentum I sometimes skip the second pass or do it superficially.
  - Grep before done — I might convince myself I've found all instances when I haven't been thorough enough.
  - Testing at session start — I need to be reminded or disciplined about this, especially when the user jumps straight into a task.

  The rules are well-written specifically because they target known failure modes of LLMs (and developers in general). The fact that you've codified them makes me more likely to
  follow them — I re-read them before acting. But I'm not 100, because the failure modes they guard against are genuine tendencies, not theoretical ones.

  If I slip, call it out — that's what the feedback loop is for.
```
