# Anti-Patterns

Patterns to watch for when working with AI-assisted development. Each was identified through real project experience, not theory.

## Structure Degradation

**The pattern:** Data starts as a tree or structured model. Someone parses it with regex into a flat list. Context is lost. Heuristics are added to guess what the structure was. Language starts appearing like "picking the winner" and "X wins because Y." The result is brittle code built on guesswork.

**The signal:** Any time you see heuristics reconstructing hierarchy that the source data already had.

**The fix:** Go back to the source. Parse the structure as a tree. Walk the tree. Let the structure inform the code. Parsers and generators should derive from the data model, not reconstruct it from fragments.

**The principle:** Data-first, model-first. Preserve the structure you were given.

## Symptom Chasing

**The pattern:** A problem appears. You fix it in one place. Another problem appears. You fix that too. Then a third. You're patching symptoms, not addressing the root cause.

**The signal:** If you're applying your third targeted fix in a row, stop. The cascade of fixes is the smell — it means the earlier rules (data-first, read before writing, plan mode) weren't followed, and you're paying for it incrementally.

**The fix:** Stop fixing symptoms. Step back, find the root cause, and address it properly — even if that means refactoring. A proper solution now is better than an infinite series of patches.

**The principle:** Diagnostic scripts help here too. Isolate the actual problem before fixing anything.

## Silent Fallbacks

**The pattern:** Required configuration is missing. Instead of failing with a clear error, the code silently falls back to a default value. Everything appears to work — until it doesn't, and the failure is subtle, intermittent, and far from the root cause.

**Examples:**
```typescript
// WRONG — silent fallback masks configuration errors
const secret = process.env.JWT_SECRET || 'default';
const connString = process.env.AzureWebJobsStorage || 'UseDevelopmentStorage=true';

// RIGHT — fail fast with clear error
const secret = process.env.JWT_SECRET;
if (!secret || secret.length < 32) {
    throw new Error('JWT_SECRET must be at least 32 characters');
}
```

**The signal:** Any `|| 'default'` or `|| fallbackValue` on required configuration.

**The fix:** Fail fast. If required information is missing, throw an error with a message that tells the developer exactly what to configure.

## Shortcut Creep

**The pattern:** "Just this once, let's skip validation." "We'll come back and add tests later." "Let's use a placeholder for now." Each shortcut is small. They accumulate.

**The signal:** Any time Claude (or a developer) says "for now" or "we can improve this later."

**The fix:** The STOP/ASK/WAIT protocol. When tempted to take a shortcut:
1. **STOP** — Do not proceed.
2. **ASK** — Explain the full approach, the shortcut, and the tradeoffs.
3. **WAIT** — Get explicit approval before proceeding.

**The principle:** Shortcuts that are agreed upon are fine. Shortcuts that are hidden are technical debt with compound interest.

## Plaintext Secrets with AI Tools

**The pattern:** AI-assisted development tools read project files as context. Plaintext `.env` and `local.settings.json` files containing secrets are an attack surface — the AI tool reads them, and they may appear in logs, context windows, or transmitted data.

**The fix:**
- Prefer encrypted local config where available (Azure's `local.settings.json` supports encryption).
- Ensure secret files are in `.gitignore` and not in directories the AI tool indexes.
- For production: use secret managers (Azure Key Vault, AWS Secrets Manager, etc.) with runtime resolution.

## Squash Merge and Release Tags

**The pattern:** A developer runs `pk changelog` on a branch, creating a tag at commit C. The PR is squash-merged to main, creating a new commit S. The tag still points at C, which is now orphaned — it's not an ancestor of main.

```
Feature:  A → B → C (tag: v1.2.0)
                    ↓ squash merge
Main:     X → Y → S (new commit — tag is NOT here)
```

**The consequence:** `git log main` won't show the tagged commit. `git describe` on main finds nothing. CI/CD looking for tags on the release branch sees no release. The version history is detached from the branch it was released to.

**The fix:** Use merge commits or rebase merge — not squash — for branches that carry release tags. Both preserve the original commits (and their tags) as ancestors of the target branch. Run `pk changelog` on the branch that receives the final commit, and use `pk release` or `pk release --pr` to move it to the release branch.

**The principle:** Tags must point to commits that are ancestors of the release branch. Any merge strategy that rewrites commits between tagging and landing will orphan the tag.
