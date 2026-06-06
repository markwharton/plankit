---
name: review-rules
description: Deep-analyze the project's rule set (.claude/rules + CLAUDE.md) for coherence, redundancy, tensions, gaps, and actionable fixes
---

Analyze the project's rule set as a system, not a checklist. The goal is to find what makes the rules work together and what undermines them, then propose concrete fixes. This is the analysis a plain concatenation cannot give you.

Grounding (how Claude actually consumes rules):

- Rules are context, not enforcement. They are followed best-effort, so vague, weak, or buried rules quietly fail.
- When two rules conflict, the model may pick one arbitrarily. Surfacing conflicts and unstated precedence is the highest-value thing this review does.
- Specific, high-signal rules beat volume. Overlap and rarely-relevant rules dilute the set (context rot), so distinguish reinforcement from dilution rather than flagging all repetition.

## Read the set

Read every `.claude/rules/*.md` and the project `CLAUDE.md` (the always-loaded critical rules and conventions). For each rule file note its `kind:` frontmatter: `craft` (developer-voiced standards for the work) or `conduct` (Claude-voiced agent behavior). If there is no `.claude/rules/` directory, say so and stop.

## Analyze

Work each dimension. Cite rule IDs, quote short phrases, and keep findings concrete; do not pad for breadth.

1. **Atomic inventory.** Break the files into atomic rules. Give each a stable ID (e.g. `GIT-02`, `MODEL-05`), a short name, the one-line imperative, its source file, its `kind`, and the trigger (the situation in which it should fire).
2. **Themes.** Group the atomic rules into themes that cut across files, and note which rules co-fire.
3. **Relationships.** For notable pairs, classify: reinforces, depends-on, generalizes/specializes, in-tension, or load-bearing duplicate. Cite IDs.
4. **Redundancy.** Separate load-bearing repetition (always-loaded echoes of fuller rules, or one idea deliberately split across craft and conduct) from genuine dilution worth merging.
5. **Gaps.** Situations a set like this usually covers but this one does not.
6. **Tensions and conflicts.** Pairs that could pull opposite ways, or where the priority between rules is unstated.
7. **Coherence verdict.** Name the organizing spine (the principle the set radiates from), give an internal-consistency read, and name the single biggest structural weakness.

## Render (offer)

Offer these if the developer wants them:

- A compact cross-reference table: rule ID to related IDs with relationship type.
- Flash-cards for the highest-stakes rules (ID, name, kind, trigger, rule, why, links), showing craft and conduct twins side by side so the split is visible.

## Recommend

End with a short, prioritized list of concrete fixes. This is the payoff, not a summary. Lead with the highest-value change (often a missing precedence rule, a merge of diluting overlap, or a filled gap), and for each give the specific edit you would make and why.

Propose only. Do not edit the rule files: recommend, and let the developer decide. Plans in `docs/plans/` are immutable, and scope stays the developer's call.
