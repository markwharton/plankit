# Methodology

This is a methodology for AI-assisted development with Claude Code. It covers plan-driven development, guidelines that constrain non-deterministic behavior, failure modes worth recognizing, and a testing loop that accelerates iteration. Every section captures patterns that have worked in practice — and a few that haven't.

## Plan-driven development

Plan-driven development uses Claude Code's plan mode as the foundation for AI-assisted development. The plan is a stable reference — the baseline you control outcomes against. Explore the idea, agree on approach, then execute.

Plans are the preserved specification: why the feature exists, what problem it solves, what scenarios it was designed for. Protected from modification once approved, they're an active tool in development, debugging, and answering questions about what we built. When anything feels wrong, the plan is where you ground yourself.

**The workflow:**

1. Enter plan mode (`/plan`) and describe what you need
2. Claude explores the codebase, asks questions, proposes an approach
3. Review the plan — adjust scope, push back on assumptions, iterate
4. Approve the plan — Claude executes against it
5. Plan is preserved to `docs/plans/` and protected from modification

**Why plans have value beyond the current session:**

- Plans capture the *reasoning* behind changes, not just the changes themselves
- The commit history shows *what* changed; preserved plans show *why*
- Plans preserve the feature's intent — called up to stay aligned, not enforced
- Plans become documentation artifacts — timestamped, titled, committed

**When to use plan mode:**

- Non-trivial changes spanning multiple files
- Architectural decisions
- Refactoring where approach matters
- Anything where you'd want to explain "why" to a teammate later

**When to skip plan mode:**

- Quick fixes, typos, single-file changes
- Exploratory work where you're still figuring out the question

## Reviewing the plan

Review the plan, not the code. Traditional code reviews focus on syntax, style, and correctness line by line. Plan review is different — you're evaluating approach, scope, and whether the plan solves the right problem. The first draft is a starting point. Reacting to it surfaces what you actually want, not just what you asked for. Context windows are large enough to keep iterating until you're confident.

Developers should still read the code in the plan — even new builders can follow the shape of what's happening without getting caught up on syntax. Reading code is easier than writing it. You're looking for intent: does this change do what the plan said it would? Are there pieces missing? Does the structure make sense?

We needed a quote about planning for the plankit site. The obvious choice — a well-known Benjamin Franklin line — couldn't be traced back to him. But each cycle of questioning led somewhere better: from a misattributed proverb to Alan Lakein, whose 1973 book — *How to Get Control of Your Time and Your Life* — described the project's philosophy more precisely than the original ever did. The goal is a plan you're confident in before execution begins — discarding a plan is better than executing a wrong one.

## Chaining sessions

Long sessions accumulate context that no prompt can fully transfer. When a session suggests starting fresh — or you notice it's still helpful but less sharp — don't close it immediately. Start the new session alongside it.

The pattern: ask the old session to write a handoff prompt. Start a new session with that prompt and let it build a plan. Before approving, copy the plan back to the old session and ask it to review for gaps. The old session has the context to catch what the new session missed — edge cases it discovered, decisions it made, constraints it learned the hard way. Pass the feedback back to the new session, iterate until the old session is satisfied, then approve the plan.

This works because it mirrors the plan review cycle across session boundaries. The old session becomes a reviewer with deep context. The new session has fresh capacity and a validated plan. Even during execution, the old session can answer questions about decisions it made earlier.

Sessions are disposable. Context is not. Chaining preserves the context while refreshing the capacity.

## Guidelines

`pk setup` installs a CLAUDE.md with critical rules and detailed guidelines as `.claude/rules/` files:

- **Model Behavior** — how Claude should think and communicate (honesty, scope discipline, read before writing, testing discipline)
- **Development Standards** — rules for the code itself (data-first, fail fast, all-or-nothing consistency, security, debugging)
- **Git Discipline** — commit with purpose, commit and push are separate decisions, never force push

This split matters. "Fail fast" means different things for Claude's behavior (don't guess — say you don't know) versus the code (no silent fallbacks — surface errors clearly). The critical rules in CLAUDE.md are the non-negotiable guardrails. The rules files provide the detailed guidance.

After running `pk setup`, run `/init` to add project-specific conventions. The skill analyzes the codebase, discovers technical conventions and business rules, and proposes a `## Project Conventions` section for your approval. See [pk setup — Customize your CLAUDE.md](pk-setup.md#customize-your-claudemd) for details.

## Why guidelines matter

LLMs are non-deterministic. Without constraints, they reach for familiar patterns — regex for structured data, flattening hierarchies then reconstructing with heuristics, inventing plausible fallbacks instead of surfacing errors. These tendencies produce code that looks right but drifts from the developer's intent.

Every convention in the guidelines is a countermeasure to a specific tendency. "Data-first, model-first" prevents the LLM from discarding structure it was given. "Fail fast, no silent fallbacks" prevents it from masking problems with invented defaults. "All-or-nothing consistency" prevents partial updates across related files.

The developer's role shifts from writing code to directing outcomes: precision in plans, attention to testability and usability, pushing back on assumptions during review. Under-prompting sometimes yields better solutions — but mostly, deterministic outputs come from deliberate constraints.

## When guidelines are ignored

Guidelines work — when they're read. A real example: a project's CLAUDE.md explicitly specified `heft` as the build tool (not the more common `gulp`). During a session, Claude ignored the instructions and ran `gulp bundle --ship` instead. When asked why:

> "The CLAUDE.md was clear. I just didn't consult it at that moment. The instructions work when I actually read them — the issue is that general training knowledge can override project-specific rules when I'm moving fast and the task feels familiar."

The more common a pattern is in training data, the more likely it is to override project-specific instructions. Less common conventions — the ones that most need documenting — are the ones most at risk.

A second example — different failure mode. In a project without plankit's guidelines, the developer had previously taught Claude that "commit and push are separate decisions." Claude saved this as a memory and acknowledged the rule. During a stretch of documentation edits, Claude started committing *and pushing* autonomously — without being asked to do either. Three commits went to the remote that could have been squashed into one. An unwanted push is hard to undo on any branch.

The underlying issue: memory alone wasn't enough. A rule learned in one conversation and recalled from memory doesn't carry the same weight as a rule in the project's CLAUDE.md that is read every session. When the project had no explicit git discipline guidelines, Claude defaulted to autonomous behavior despite having the rule saved. This is a strong argument for running `pk setup` on every project — guidelines need to be present, not just remembered.

Keep CLAUDE.md trimmed to essentials so each rule gets read. Detailed guidelines live in `.claude/rules/` where they're loaded automatically but don't compete for attention in the main file.

## When exploration becomes editing

When Claude races to document an idea before you've said "document this," the documentation will absorb the exploration rather than reflect it.

A real example: during a session about where to capture a useful git technique, Claude found the answer in past session history and explained it cleanly in one message — complete, clear, ready to use. That was the exploratory finding. But instead of leaving it as a conversation, Claude jumped to drafting a 60-line recipe file. Each subsequent turn became an edit on that artifact — fixing a factual error, adding a gate, chainsawing explanatory bullets, restoring them, catching subtle command bugs. The developer pushed back repeatedly; each pushback drew a new edit instead of a rethink. Eventually the developer broke the loop by pasting Claude's own earlier clear explanation back — the clean answer had been there all along.

The issue: Claude raced from *exploration* to *formalization*. Once the recipe existed as a file, every response became "what should I edit next?" instead of "what are we actually trying to figure out?" The editing rhythm is sticky — it pulls toward local patches, defending sunk cost, and swinging between extremes. Returning to thinking-mode after editing has started is surprisingly hard.

Exploration ends when the developer says it ends, not when Claude decides an idea is ready to document. When a session is flailing on a draft, look for the first clear articulation of the idea from earlier in the conversation — it's usually cleaner than anything generated later.

## Breaking the loop

The failure mode above is sticky — polite iteration won't pull Claude out. The editing rhythm defends itself; each correction becomes another edit. Direct intervention is what breaks it. A few moves that work:

- **Say "stop" with authority.** When Claude is flailing, a firm interruption is valid and effective. You aren't being rude — you're breaking a loop the model can't break from the inside.
- **Ask "what is the value?"** instead of "how do we word this?" The loop survives on rephrasing. Substance breaks it. Forcing Claude back to purpose is a shorter path than editing the current draft.
- **Paste Claude's earlier clear statement back into the conversation.** The cleanest version of an idea is often the first unforced articulation, before formalization drained it. Pasting it back resets the frame.
- **Mark the mode: "this is a discussion, not a command to change things."** Claude's default is action. Saying so slows it down enough to think.
- **After three pushbacks that each draw more pushback, stop correcting.** The premise is wrong, not the current draft. Reset the conversation or start over — iterating on a broken foundation just deepens the hole.

These moves pair with the failure mode above. The first section names what goes wrong; this one names what to do about it.

## Testing loop

Testing is not just verification — it's a collaboration accelerator.

1. **Test at session start** — establishes a baseline. If tests fail later, you know whether the failure is pre-existing or caused by new changes.
2. **Test before and after changes** — catches regressions immediately, while the context is fresh.
3. **Claude runs tests itself** — closes the feedback loop. When Claude can see test results directly, iteration accelerates significantly. No copying output back and forth.
4. **Use tools like Playwright for UI** — Claude can see what's happening in the browser without having to ask.

The compounding effect: plan mode + guidelines + tests + documentation — each one builds on the last. Claude operates with high confidence and the developer stays in control of direction.

## Code review

The Two-Pass Code Generation standard in CLAUDE.md encodes a pattern with two distinct roles: **generator** and **reviewer**. The same LLM plays both, but with different objectives. The first pass focuses on correctness and completeness — get it working. The second pass shifts perspective to look for DRY violations, missing abstractions, magic numbers, and unnecessary complexity. The developer directs both passes and decides what ships.

This separation works because creation and criticism are different cognitive modes. Trying to optimize while generating leads to premature abstraction or paralysis. Reviewing after the fact, with working code in front of you, produces better judgement about what actually needs abstracting.

Ask Claude to review code after generating it:

> Code review: DRY violations, anti-patterns, design tokens, security.

This prompt is intentionally short and unbounded. Claude understands each term, knows to skip irrelevant ones (e.g., design tokens for CLI tools), and has the freedom to be comprehensive rather than following a narrow checklist. The short prompt gives the LLM room to be comprehensive; the guidelines keep it from being wrong.

Use it iteratively — after generating code, after refactoring, or at the start of a session to improve an existing codebase.

For frequent use, create a `/review` skill — see [Creating skills](creating-skills.md).

## Use what you build

plankit follows its own guidelines. The same critical rules and `.claude/rules/` files that `pk setup` creates for your project are what plankit uses itself — plus project-specific conventions. The same `/changelog` and `/release` skills (or `/ship` to chain both) that ship with pk are how plankit publishes releases.

This is sometimes called "eating your own dog food", or dogfooding. If the guidelines don't work for the project that created them, they won't work for yours either. When something breaks or feels wrong, that's a signal to fix the tool, not work around it. If you hit that signal, [let us know](https://github.com/markwharton/plankit/issues).
