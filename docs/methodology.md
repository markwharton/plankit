# Methodology

A methodology for AI-assisted development with Claude Code. Plans define intent, guidelines constrain behavior, tests validate outcomes, review ensures quality.

> Constrain the model with plans and rules, validate with tests, and stay in control through deliberate review.

## 1. Core principle

The developer controls the system. The model executes within it.

The system is composed of plans, guidelines, review, and testing. No single component is sufficient. A more capable model without discipline still drifts. A less capable model inside a stable structure produces work you can extend, audit, and revisit weeks later.

## 2. Plan-driven development

Use plan mode as the foundation for non-trivial work. The plan is a stable reference: it defines *why* the feature exists, *what* problem it solves, and *how* it will be implemented. Once approved, it is preserved and protected from modification.

**Workflow:**

1. Enter plan mode (`/plan`) and describe the need
2. Claude explores the codebase, asks questions, proposes an approach
3. Review and iterate. Adjust scope, push back on assumptions
4. Approve the plan. Claude executes against it
5. Plan is preserved to `docs/plans/` and protected from modification

**When to use:** multi-file changes, architectural decisions, refactoring where approach matters, anything where you'd want to explain "why" to a teammate later.

**When to skip:** quick fixes, typos, single-file changes, exploratory work where you're still figuring out the question.

**Why plans have value beyond the session:** the commit history shows *what* changed; preserved plans show *why*. Plans become timestamped documentation artifacts, called up to stay aligned, not enforced.

Once approved, the plan becomes the reference for all work. Execution should not diverge from it. Refer back to the plan when something feels off. It's alignment, not enforcement.

## 3. Plan review

Validate the plan before execution begins. Catching a wrong approach in the plan costs minutes; catching it after execution costs hours.

- Does the approach match intent?
- Is the scope correct? Not too wide, not too narrow?
- Does the code shape make sense?
- Is anything missing?
- Are there artificial limits that drop data?

Discarding a plan is better than executing a wrong one.

## 4. Guidelines

`pk setup` installs behavioral constraints: critical rules in CLAUDE.md, detailed guidelines as `.claude/rules/` files covering model behavior, development standards, and git discipline.

LLMs are non-deterministic. Without constraints, they reach for familiar patterns: regex for structured data, flattening hierarchies then reconstructing with heuristics, inventing plausible fallbacks instead of surfacing errors. Every convention in the guidelines is a countermeasure:

- **Data-first, model-first:** preserve structure the model was given
- **Fail fast, no silent fallbacks:** surface errors, don't mask them
- **All-or-nothing consistency:** no partial updates across related files
- **Git discipline:** commit with purpose, no autonomous commits or pushes

The same rule can differ by context: "fail fast" for model behavior means don't guess; for code, it means surface errors clearly. The split between CLAUDE.md and `.claude/rules/` reflects this. Critical guardrails in the main file, detailed guidance in rules files.

After `pk setup`, run `/init` to add project-specific conventions. See [Customize your CLAUDE.md](pk-setup.md#customize-your-claudemd).

## 5. Compounding effect

Individual components prevent specific failures: plans prevent drift, guidelines constrain behavior, tests validate correctness. Together, they form a system. The result belongs to the system, not to any single component.

Testing is both validation and collaboration accelerator:

1. **Test at session start:** establishes a baseline
2. **Test before and after changes:** catches regressions immediately
3. **Claude runs tests directly:** closes the feedback loop, no copying output back and forth
4. **Use tools like Playwright for UI:** Claude can see what's happening without having to ask

The developer's role shifts from writing code to directing outcomes: precision in plans, attention to testability, pushing back on assumptions during review. Under-prompting sometimes yields better solutions, but mostly, deterministic outputs come from deliberate constraints.

Two recent papers calibrate this on harder problem classes: Donald Knuth's *Claude's Cycles* (2026) and Keston Aquino-Michaels' *Completing Claude's Cycles* (2026). Different scale, same axis.

## 6. Code review

Two distinct roles: **generator** and **reviewer**. The same LLM plays both, but with different objectives. First pass focuses on correctness. Get it working. Second pass looks for DRY violations, missing abstractions, magic numbers, unnecessary complexity. The developer directs both and decides what ships.

Creation and criticism are different cognitive modes. Mixing them leads to premature abstraction or defensive bloat.

> Code review: DRY violations, anti-patterns, design tokens, security.

The short prompt gives the LLM room to be comprehensive; the guidelines keep it from being wrong. For frequent use, create a `/review` skill. See [Creating skills](creating-skills.md).

## 7. Chaining sessions

Long sessions accumulate context but degrade over time.

1. Ask the old session to write a handoff prompt
2. Start a new session with that prompt and let it build a plan
3. Send the plan back to the old session for review
4. Iterate until the old session is satisfied, then approve

The old session retains deep context; the new session has fresh capacity. Sessions are disposable. Context is not.

## 8. Session dynamics

Exploration should remain separate from documentation. When Claude creates artifacts too early, the conversation shifts from thinking into editing. Iteration becomes local patching instead of clarity.

Exploration ends when the developer says it ends, not when Claude decides an idea is ready to document. When a session is flailing on a draft, the first clear articulation from earlier in the conversation is usually cleaner than anything generated after.

Some failure modes persist through polite iteration. The editing rhythm defends itself; each correction becomes another edit. Direct intervention breaks it:

- **Say "stop"** (or hit Esc). A firm interruption is valid, not rude
- **Ask "what is the value?"** Substance breaks the loop, rephrasing sustains it
- **Paste Claude's earlier clear statement back.** The first unforced articulation is usually the best
- **After three pushbacks that each draw more pushback, stop correcting.** The premise is wrong, not the draft

## 9. Model resilience

Guidelines work when they're read. A real example: a project's CLAUDE.md explicitly specified `heft` as the build tool (not the more common `gulp`). During a session, Claude ignored the instructions and ran `gulp bundle --ship` instead. When asked why:

> "The CLAUDE.md was clear. I just didn't consult it at that moment. The instructions work when I actually read them. The issue is that general training knowledge can override project-specific rules when I'm moving fast and the task feels familiar."

The more common a pattern is in training data, the more likely it is to override project-specific instructions. Less common conventions, the ones that most need documenting, are the ones most at risk.

A different failure mode: Claude had learned "commit and push are separate decisions" via memory. In a project without plankit's guidelines, it started committing and pushing autonomously. Three commits went to the remote that could have been squashed into one. Memory alone wasn't enough. A rule recalled from memory doesn't carry the same weight as a rule in the project's CLAUDE.md that is read every session. Guidelines need to be present, not just remembered.

Keep CLAUDE.md trimmed to essentials so each rule gets read. Detailed guidelines live in `.claude/rules/` where they're loaded automatically but don't compete for attention.

Rules create accountability, not obedience. The model won't follow them every time. But without them, the same failures happen silently and nobody learns.

## 10. When the model shifts

Claude's behavior evolves across versions: how plan mode exits, how long-session context is retained, how auto mode proceeds. pk sits between the model and the action. When the model changes, the gaps show up here first. pk's guarantees live in the CLI layer: git mutation guards, managed-file protection, bounded hook timeouts.

When a release exposes an edge case, the fix belongs in the CLI layer. Reduce the model-dependence that let the bug in, so the next shift can't re-expose the same class of problem. That's the lifecycle, not an aberration. If you hit a surprise, [open an issue](https://github.com/markwharton/plankit/issues).
