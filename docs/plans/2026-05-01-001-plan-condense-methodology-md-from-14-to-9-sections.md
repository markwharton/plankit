# Plan: Condense methodology.md from 14 to 9 sections

## Context

methodology.md was previously condensed but still has structural issues: the silent semantic narrowing paragraph in §3 is a detailed aside already covered in anti-patterns.md, §13 "Use what you build" is project meta-commentary rather than methodology, several sections are closely related enough to combine, and em dashes throughout the document signal AI-generated text.

## Changes

All edits are to `docs/methodology.md`.

### 1. Remove §13 "Use what you build"

Delete lines 127-129. Project meta-commentary, not methodology.

### 2. Trim silent semantic narrowing from §3 "Plan review"

Replace the paragraph on line 40 with a brief checklist item. No link, no detailed examples. Keep the closing line.

### 3. Fold §4 "Execution against the plan" into §2 "Plan-driven development"

Append §4's two sentences as a closing paragraph of §2, after "Why plans have value." Delete the §4 heading.

### 4. Combine §6 + §7 into §5 "Compounding effect"

- System-level insight (individual components prevent specific failures, together they compound)
- Testing practices (the numbered list from §7)
- Developer role shift and the Knuth/Aquino-Michaels reference
- Drop the "The compounding effect:" sentence from §7 since the section title carries it

### 5. Combine §10 + §11 into §8 "Session dynamics"

- Exploration should remain separate from documentation (§10)
- When iteration becomes unproductive, intervene directly (§11 with bullet list)

### 6. Combine §12 + §14 into §9 "Model resilience"

- Training data overrides project instructions (heft/gulp example)
- Memory alone isn't enough (commit-and-push example)
- Keep CLAUDE.md trimmed
- Model behavior evolves across versions; pk's guarantees live in the CLI layer
- Drop both "open an issue" links

### 7. Replace em dashes throughout

~30 em dashes in the current file. Replace by category:

**Bold term lists** (`**Term** — definition`): use colon + space (`**Term:** definition`)
- Lines 54-57 (guideline bullets), 75-78 (testing steps), 113-115/117 (breaking-loop bullets)

**Aside connectors** (sentence — aside): restructure as two sentences, or use a colon/comma
- Lines 15, 29, 36, 46, 52, 59, 65, 67, 69, 80, 84, 90, 105, 123, 133, 135

**Workflow steps** (lines 21-22): use period or colon

**Link text** (line 61): `[pk setup — Customize your CLAUDE.md]` rewrite display text (heading in pk-setup.md has no em dash)

Each replacement is a judgment call during editing. The goal: zero em dashes in the final file.

### 8. Renumber sections 1-9

Final structure:
1. Core principle
2. Plan-driven development
3. Plan review
4. Guidelines
5. Compounding effect
6. Code review
7. Chaining sessions
8. Session dynamics
9. Model resilience

## Verification

- Zero em dashes in final file (`grep '—' docs/methodology.md` returns nothing)
- `make test` passes
- Read final file end-to-end for flow and tone consistency
- Grep for internal references to removed/renumbered sections
