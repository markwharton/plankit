# Methodology

## Plan-driven development

Plan-driven development uses Claude Code's plan mode as the foundation for AI-assisted development. Think first, agree on approach, then execute.

**The workflow:**

1. Enter plan mode (`/plan`)
2. Claude explores the codebase, asks questions, proposes an approach
3. Developer reviews, adjusts, approves
4. Claude executes against the approved plan
5. Plan is preserved to `docs/plans/` (auto or manual)
6. Plan is protected from accidental modification

**Why plans have value beyond the current session:**

- Plans capture the *thinking* behind changes, not just the changes themselves
- The commit history shows *what* changed; preserved plans show *why*
- For teams: a searchable record of what developers were working on and the reasoning behind decisions
- Plans become documentation artifacts — timestamped, titled, committed

**When to use plan mode:**

- Non-trivial changes spanning multiple files
- Architectural decisions
- Refactoring where approach matters
- Anything where you'd want to explain "why" to a teammate later

**When to skip plan mode:**

- Quick fixes, typos, single-file changes
- Exploratory work where you're still figuring out the question

## Directing the plan

Plan review is where the developer directs the outcome. Sometimes the plan isn't going in the right direction and that's frustrating. Frustration is a signal that the way you're directing needs to change. You can keep iterating — context windows are large enough now. You can stop and come back later with fresh perspective. Or you can discard the plan entirely.

The goal is a plan you're confident in before execution begins — discarding a plan is better than executing a wrong one.

## Guidelines

`pk setup` installs a universal CLAUDE.md with guidelines split into two sections:

- **Model Behavior** — instructions for how Claude should think and communicate (honesty, scope discipline, read before writing, testing discipline)
- **Development Standards** — rules for the code itself (data-first, fail fast, all-or-nothing consistency, git discipline)

This split matters. "Fail fast" means different things for Claude's behavior (don't guess — say you don't know) versus the code (no silent fallbacks — surface errors clearly). The universal template makes both explicit.

Use `/init` to add project-specific conventions. Technology-specific reference templates (`go.md`, `typescript.md`, `azure.md`) are available in `templates/` for further customization.

## Why guidelines matter

LLMs are non-deterministic. Without constraints, they reach for familiar patterns — regex for structured data, flattening hierarchies then reconstructing with heuristics, inventing plausible fallbacks instead of surfacing errors. These tendencies produce code that looks right but drifts from the developer's intent.

Every convention in the templates is a countermeasure to a specific tendency. "Data-first, model-first" prevents the LLM from discarding structure it was given. "Fail fast, no silent fallbacks" prevents it from masking problems with invented defaults. "All-or-nothing consistency" prevents partial updates across related files.

The developer's role shifts from writing code to directing outcomes: precision in plans, attention to testability and usability, pushing back on assumptions during review. Under-prompting sometimes yields better solutions — but mostly, deterministic outputs come from deliberate constraints.

## Testing loop

Testing is not just verification — it's a collaboration accelerator.

1. **Test at session start** — establishes a baseline. If tests fail later, you know whether the failure is pre-existing or caused by new changes.
2. **Test before and after changes** — catches regressions immediately, while the context is fresh.
3. **Claude runs tests itself** — closes the feedback loop. When Claude can see test results directly, iteration accelerates significantly. No copying output back and forth.
4. **Use tools like Playwright for UI** — Claude can see what's happening in the browser without having to ask.

The compounding effect: plan mode + guidelines + tests + self-testing creates a workflow where Claude operates with high confidence and the developer stays in control of direction.

## Code review

The `/review` skill runs a comprehensive code review:

```
Code review: DRY violations, anti-patterns, design tokens, security.
```

This prompt is intentionally short and unbounded. Claude understands each term, knows to skip irrelevant ones (e.g., design tokens for CLI tools), and has the freedom to be comprehensive rather than following a narrow checklist. The short prompt gives the LLM room to be comprehensive; the templates keep it from being wrong.

Use it iteratively — run `/review` after generating code, after refactoring, or at the start of a session to improve an existing codebase. The two-pass approach (generate first, review second) works because it separates creation from criticism.

## Use what you build

plankit follows its own guidelines. The same CLAUDE.md that `pk setup` creates for your project is what plankit uses itself — universal base plus project conventions added via `/init`. The same `/changelog` and `/release` skills that ship with pk are how plankit publishes releases.

This is sometimes called "eating your own dog food", or dogfooding. If the guidelines don't work for the project that created them, they won't work for yours either. When something breaks or feels wrong, that's a signal to fix the tool, not work around it. If you hit that signal, [let us know](https://github.com/markwharton/plankit/issues).
