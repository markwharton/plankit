# Messaging Audit: Narrow the Solution Space

## Context

The README and methodology docs describe plankit using a "deterministic outcomes" framing that is inaccurate and overly abstract. The README opens with philosophy ("LLMs are open-ended by nature. Development needs deterministic outcomes. plankit bridges that gap...") before telling anyone what plankit actually is or does.

The accurate framing, confirmed through discussion: plankit narrows the solution space. Without plans and rules, every LLM session draws from the full range of possible implementations. Plans narrow a specific task to one or two approved approaches. Rules reduce general drift between sessions. Together they make the LLM's non-determinism manageable, not eliminated.

## Files to Modify

### 1. `README.md` (lines 8-10)

**Line 8 (bold lead):** Replace the entire statement. The new lead should:
- Start with "plankit" so the reader knows what it is immediately
- Use "narrows the solution space" instead of "bridges that gap" or "deterministic outcomes"
- Describe the mechanism concretely (plans, rules, hooks)
- No em dashes

Current:
> **LLMs are open-ended by nature. Development needs deterministic outcomes. plankit bridges that gap: plans commit to an approach before code is written, rules suppress the patterns that cause drift, and tests protect what works.**

**Line 10 (tagline paragraph):** Restructure to:
- Keep "plan-driven development toolkit for Claude Code"
- Add the shared artifact / pair programming concept (plans reviewed by one or more developers)
- Feature "plan preservation" and "plan protection" prominently
- Keep "discipline is the multiplier" but not as the second sentence
- Keep "Designed for small teams and independent developers"
- Remove "(Anthropic's AI coding agent)" parenthetical since Claude Code is already linked

Current:
> A plan-driven development toolkit for [Claude Code](https://code.claude.com) (Anthropic's AI coding agent). Discipline is the multiplier. Rules, testing, and branch protection aren't extras; they're what make plans worth keeping. Designed for small teams and independent developers.

**Line 12:** Keep the Best Practices link but evaluate whether the wording needs adjusting after the tagline changes.

### 2. `docs/methodology.md` (lines 49 and 71)

Two surgical edits only. The surrounding content stays.

**Line 49:** Replace "LLMs are non-deterministic" with language about drawing from a wide solution space. The rest of the paragraph (familiar patterns, countermeasures) is accurate and stays.

Current:
> LLMs are non-deterministic. Without constraints, they reach for familiar patterns...

**Line 71:** Replace "deterministic outputs come from deliberate constraints" with "predictable outcomes" or "narrowing the range of outcomes."

Current:
> Under-prompting sometimes yields better solutions, but mostly, deterministic outputs come from deliberate constraints.

### 3. No other files

- `docs/adoption.md`: concrete, no deterministic framing, no changes
- `docs/plans/*`: immutable, no changes
- `cmd/pk/main.go`: "Plan-driven development toolkit for Claude Code" is accurate, no change
- `site/`: does not exist, out of scope
- `.claude/rules/`: no deterministic framing, no changes

## Verification

1. `make build && make test && make lint` pass
2. Grep the repo for "deterministic" to confirm the only remaining instances are in `docs/plans/` (immutable) and unrelated code (SHA determinism in tests)
3. Read the updated README top-to-bottom and confirm: a reader landing cold learns what plankit is before encountering any philosophy
