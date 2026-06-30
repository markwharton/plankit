# Prevent and recover from release-branch divergence

## Context

A downstream project hit a painful failure: following bad ad-hoc instructions,
a `pk setup` commit (PK_VERSION bump + a `settings.json.bak`) was made **directly on
`main`** and pushed. Because `pk release` does a **fast-forward-only** merge of the
source branch into the release branch (`git merge --ff-only`, verified at
`internal/release/release.go:225`), `main` must stay a strict fast-forward ancestor of
`develop` and must never carry a commit of its own. The stray commit broke that
invariant, so the next `pk release` failed:

```
Error: merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.
```

Since the commit was already pushed it couldn't be dropped (never force push). The
recovery that worked, and its non-obvious ordering, was: `pk changelog --undo` →
`git merge origin/main` → `pk changelog` → `pk release`. The ordering matters because
`pk release` reads the `Release-Tag` trailer from **HEAD only**
(`internal/changelog/trailer.go`: `git log -1 --format=%(trailers:...) HEAD`). If you
merge `origin/main` *before* undoing the release commit, the merge commit lands on top
and buries the trailer, so the retry fails with "no Release-Tag trailer on HEAD."

Two gaps to close: a **prevention** rule (work stays on the source branch) and a
**recovery** doc (the ordered steps, with the never-force-push constraint).

This also surfaced a documentation drift to fix in the same pass (see below).

## Verified behavior

- Merge is unconditionally `git merge --ff-only <sourceBranch>` — `internal/release/release.go:225`.
- Real divergence error: `merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.` — `internal/release/release.go:226`.
- On failure after switching, `pk release` switches back to the source branch (`docs/pk-release.md:92-94`), so recovery starts on `develop`.
- `pk changelog --undo` requires HEAD to carry a valid `Release-Tag` trailer, a clean tree, and an unpushed HEAD; it runs `git reset --hard HEAD~1` — `internal/changelog/changelog.go:340-378`.
- `git-discipline.md` is a **shipped** managed rule (`pk_sha256` in the local copy, mirrored in `internal/setup/rules/`), so its new bullet must be generic (source/release branch, not develop/main).
- `new-plankit-project` skill has **no** `pk_sha256` and is maintainer-local (markwharton-specific paths) — editing it is a single-file change, no mirror, no hash.

## Changes

### 1. Prevention rule (shipped, both copies + hash)

Add one long single-line bullet to `git-discipline.md`, placed as a peer next to
**"Commit `pk setup` updates on their own"** (the natural neighbor). Draft:

> **All work happens on the source branch; the release branch advances only via `pk release`.** Features, fixes, version bumps, and `pk setup` updates all commit on the source branch (e.g. `develop`). Never `git checkout`/`git switch` to the release branch (e.g. `main`) to make a change: `pk release` fast-forward-merges the source branch into the release branch, so a commit made directly on the release branch breaks that fast-forward and the next release fails with "`<release>` has diverged from `<source>` (not fast-forward)." If that commit was already pushed it can't be dropped (never force push); recovery means reconciling it back into the source branch.

Edit **both**:
- `internal/setup/rules/git-discipline.md` (embedded source, no `pk_sha256` line)
- `.claude/rules/plankit/git-discipline.md` (local copy)

Then recompute the body hash and update the local copy's `pk_sha256`:
```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256
```
Keep `kind: craft` (developer-voiced git discipline — consistent with the existing
procedural bullets in this file).

### 2. Prevention note in the onboarding skill (maintainer-local, single file)

Add a **Design notes** bullet to `.claude/skills/new-plankit-project/SKILL.md` (the
script ends on `develop`, so this is where the invariant should be stated up front):

> **Work stays on `develop` after init.** The script leaves you on `develop`, where all subsequent work belongs — features, fixes, version bumps, and `pk setup` re-runs. `main` is fast-forward-only; `pk release` is the only thing that advances it. Never `git checkout main` to make a change — a manual commit there breaks the invariant and the next `pk release` fails with "main has diverged."

No `pk_sha256` recompute (file is unmanaged).

### 3. Recovery steps — canonical in `docs/error-reference.md`

Flesh out the **Fix** of the `### not fast-forward` entry (`docs/error-reference.md:158`)
from the current one-liner to ordered steps:

> **Fix:** Reconcile the histories, then retry. If you have an unpushed `pk changelog`
> release commit at HEAD, undo it *before* the merge so the `Release-Tag` trailer ends
> up at the released tip rather than buried under the merge commit (`pk release` reads
> the trailer from HEAD only):
>
> 1. `pk changelog --undo` — drop the unpushed release commit (skip if you have not run `pk changelog` yet).
> 2. `git merge origin/main` — merge the release branch into your source branch to reconcile the divergent commit. That commit is already pushed and can't be dropped (never force push), so merging is how you keep it.
> 3. `pk changelog` — regenerate the release commit so the `Release-Tag` trailer sits at HEAD, above the merge commit.
> 4. `pk release` — fast-forwards cleanly now, pushing the release branch, source branch, and tag.

### 4. `docs/pk-release.md` — fix drift + expand Error recovery (cross-linked)

- **Fix the stale error text** at lines 87–90. It currently reads `merge failed — main
  has diverged from dev (not fast-forward).` (em dash, "dev"), which mismatches the real
  message and violates the no-em-dashes rule. Replace with the verbatim message:
  `merge failed; main has diverged from develop (not fast-forward). Resolve on main manually, then try again.`
- **Expand the Error recovery section** (lines 92–94): keep the auto-switch-back note,
  add a sentence on what divergence means and the recovery shape, and **cross-link** to
  the `not fast-forward` entry in `error-reference.md` for the exact ordered steps.

Rationale for cross-linking rather than duplicating the four steps verbatim: keep one
source of truth per concept (error-reference.md is the canonical recovery home);
pk-release.md points to it. This satisfies "both docs" while avoiding two copies that
can drift.

## Out of scope

- No code changes — the error message and FF-only behavior are correct as-is.
- No change to `model-behavior.md` — this is craft (the release model), not agent conduct.

## Verification

1. **Grep for other stale copies** (all-or-nothing): `grep -rn "diverged\|not fast-forward\|has diverged" docs/` and `grep -rn "— .*diverged\|diverged from dev\b" docs/` to confirm no other doc repeats the wrong wording or the recovery concept.
2. **Hash check:** re-run `sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256` and confirm it equals the `pk_sha256` written in `.claude/rules/plankit/git-discipline.md`.
3. **No em dashes / hidden chars:** `grep -n "—" docs/pk-release.md` returns nothing for the fixed block; `make test` passes (its `internal/setup/embed_safety_test.go` scans the edited embedded rule for control/format characters).
4. **Build + lint:** `make build && make test && make lint` all green.
5. **Smoke (read-back):** confirm the new bullet renders correctly in `.claude/rules/plankit/git-discipline.md` and the error-reference Fix shows the four numbered steps in order; confirm `docs/pk-release.md` quotes the real message.
