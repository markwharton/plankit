# Consistent messages + guided adoption (main-only → full plankit)

## Context

The adoption experience for a new developer is unclear, and command output signals ad-hoc handling. Concretely (observed in an existing main-only repo with unpushed local commits, wanting full adoption):

- **Messages are inconsistent across all commands**: four error prefix styles (`Error:`, `pk guard:`, bare sentences, lowercase `warning:`), five next-step hint formats (backticks, parentheticals, indented lines, `or:` lines, none), three section-header styles (`Skills:`, `--- Pre-flight checks ---`, `=== Release vX ===`), mixed casing/punctuation, and `pk setup --help` prints Go's single-dash `-flag` style while all docs use `--kebab-case`. No shared helper exists (only `update.FormatNotice`).
- **Nobody owns the main-only → main/develop transition**: `/conventions` asks the branch-strategy questions and writes `guard.branches`/`release.branch`, but never creates or pushes `develop`. The only `git checkout -b develop && git push -u origin develop` in the repo is the maintainer-local `new-plankit-project` skill (new repos only). `pk status` reports what's *configured*, not what's *ready*: `release.branch: main` with no `develop` on origin shows as fully configured; the failure surfaces weeks later as a raw git error in `pk release` pre-flight (or the dead-end at `release.go:66` "you're on the release branch").

Two workstreams, sequenced so the new adoption output is born consistent: **(A)** a message standard + shared helper package, migrated across every command; **(B)** adoption journey UX built on it.

**Settled decisions** (user-confirmed): shared helper package (not convention-only); `/conventions` executes branch creation after preview + explicit confirmation; readiness lives in a `pk status` section backed by a shared `internal/readiness` package (no new `doctor` command); em-dash messages are restructured in the same pass. Plan recommendations adopted: drop backticks in stderr; keep the `=== Release vX ===` banner (sole sanctioned banner, frames the one multi-stage push-capable operation); hooks keep `pk <cmd>:` attribution.

---

## Workstream A: Message standard + `internal/msg`

### A1. The standard (document in `.claude/rules/plankit-development.md`, new "Runtime Message Style" section — maintainer-side rule, no pk_sha256 dance; also the `internal/msg` package doc comment)

1. **Channel**: all human-readable output to stderr; stdout reserved for hook JSON (existing rule, restated).
2. **Severity prefixes — exactly three**: `Error:` (cannot proceed, exit 1), `Warning:` (proceeded, degraded), `Note:` (informational aside). Kills lowercase `warning:` (changelog.go:457) and bare-sentence errors.
3. **Hook attribution**: hook commands (`guard`, `preserve`, `protect`) prefix stderr diagnostics `pk <cmd>:` (their output interleaves into the session log; attribution matters; always exit 0). User commands never self-attribute: `pk rules: no rules found...` (rules.go), `pk rules --lint:` (lint.go), `pk teardown:` (teardown.go:261) get normalized.
4. **Casing/punctuation**: text after a prefix starts lowercase, reads as continuation; single-clause messages take no trailing period; multi-sentence text uses normal sentences. Indented status lines: sentence case, no trailing period.
5. **Section headers**: `<Title>:` capitalized noun phrase, flush left (`Skills:`, `Settings (.claude/settings.json):`). Replaces release.go's `--- ... ---` dividers. The `=== Release vX ===` open/close banner stays, via a dedicated helper.
6. **Items/progress**: two-space indent under nearest header.
7. **Next-step hints**: two-space indent under the triggering message; short imperative purpose clause + colon + exact command (`  To anchor at v0.0.0: pk setup --baseline`), or bare command when obvious. Git-equivalent escape hatch on the next line: `  or: <git commands>` (enforces the existing Tip Messages rule). **No backticks in stderr** — fixes status.go:109 ``Run `pk setup` to install``.
8. **No em dashes**: restructure the ~18 messages (changelog.go:135,148,166,358; release.go:66,74…; teardown.go:261; baseline.go:34,44; `git.ErrDirtyTree`; etc.) with semicolons/two clauses, e.g. `Error: you're on "main" which is a protected branch; switch to your development branch first`. Ripples into `docs/error-reference.md` (quotes verbatim) and tests, all changing in this pass anyway.
9. **Flag help**: shared `usageFor(fs, "pk setup [flags]")` helper in `cmd/pk/main.go` using `fs.VisitAll` to render `--name` double-dash help; wired as `fs.Usage` on all 11 flagsets. (Go already accepts `--flag` input; only printed help is wrong.)
10. **Dry-run/preview**: same headers/items as the real run, future-tense verbs (existing teardown/release practice; normalize prefixes/indentation only).
11. **Commit suggestions**: when output suggests a commit for the *user* to run, print the full runnable command as a hint line (`  git commit -m "chore: update pk-managed files for v0.23.0"` — setup.go:398 is the model); when previewing a commit *pk itself* will create (dry-run), show it as a labeled field with the bare message (`  Commit: plan: <title> [skip ci]` — preserve.go:211). Today these two forms exist without a rule distinguishing them; the standard makes the distinction deliberate and audits all commit-message output sites (setup.go:397-398, preserve.go:211/245, changelog.go:320, and the message-only suggestion in `internal/setup/rules/git-discipline.md`) for conformance.

### A2. Helper package: `internal/msg/msg.go` (+ `msg_test.go`)

Stdlib only, `io.Writer` first (fits Config DI), void returns (callers ignore write errors; hooks never fail on them):

```go
func Errorf(w io.Writer, format string, args ...any)   // "Error: " + msg + "\n"
func Warnf(w io.Writer, format string, args ...any)    // "Warning: "
func Notef(w io.Writer, format string, args ...any)    // "Note: "
func Hookf(w io.Writer, cmd, format string, args ...any) // "pk <cmd>: "
func Section(w io.Writer, title string)                // title + ":\n"
func Itemf(w io.Writer, format string, args ...any)    // "  " + msg + "\n"
func Hintf(w io.Writer, format string, args ...any)    // "  " (semantic alias of Itemf; call sites read as intent)
func Or(w io.Writer, gitCmd string)                    // "  or: " + cmd + "\n"
func Banner(w io.Writer, s string)                     // "=== " + s + " ===\n"
```

Deliberately not in msg: check/step structs, color, verbosity, a Logger. Plain `Fprintf` stays legitimate for free-form body text (changelog dry-run dump ~changelog.go:235, rules footprint document, preserve's aligned `Plan:/File:/Commit:` fields). `update.FormatNotice` stays string-returning (preserve embeds it in hook JSON `systemMessage`); restyle its text only if it violates the standard.

### A3. Migration inventory (~256 non-test Fprint sites; per-file counts and test impact)

| File | ~Sites | Notes | Tests with string assertions |
|---|---|---|---|
| `cmd/pk/main.go` | 47 | collapse 28-line printUsage to one raw string; `usageFor` + wire 11 flagsets; `Error:` exits | none |
| `internal/release/release.go` | 49 | banner, 4 dashed headers → `Section`, checks → `Itemf`, errors, warnings | `release_test.go` (~50) |
| `internal/changelog/changelog.go` | 40 | errors, hints + `or:` lines, progress, lowercase `warning:` fix | `changelog_test.go` (~121, biggest churn) |
| `internal/status/status.go` | 41 | headers, items, not-configured block (drop backticks); leave `--brief` machine line | `status_test.go` (~39) |
| `internal/teardown/teardown.go` | 18 | headers, preview items, `pk teardown:` → `Warnf` | `teardown_test.go` (~14) |
| `internal/setup/setup.go` | 16 | headers, warnings, baseline/conventions nudges, commit tip | `setup_test.go` (~40) |
| `internal/preserve/preserve.go` | 15 | `pk preserve:` → `Hookf`; dry-run fields stay plain | `preserve_test.go` (~43) |
| `internal/setup/baseline.go` | 8 | progress + hint + `or:` line; em dashes at :34,:44 | `baseline_test.go` (~24) |
| `internal/setup/managed.go` | 6 | per-file results → `Itemf` | `managed_test.go` (~26) |
| `internal/rules/{rules,lint}.go` | 5 | drop `pk rules:` self-attribution | `rules_test.go`, `lint_test.go` |
| `internal/guard/guard.go`, `internal/protect/protect.go` | 5 | → `Hookf` | `guard_test.go` (~18), `protect_test.go` |
| `internal/setup/claude.go` | 2 | → `Itemf` | `claude_test.go` |
| `internal/git/cleantree.go` | — | `ErrDirtyTree` text em dash | `cleantree_test.go` |

Docs quoting output verbatim: `docs/error-reference.md` (also fix its existing drift vs changelog.go:170-171 hint text), plus fenced sample-output blocks in `docs/pk-setup.md`, `docs/pk-status.md`, `docs/pk-teardown.md`, `docs/pk-release.md`, `docs/pk-changelog.md` (grep each).

---

## Workstream B: Adoption journey

**Model**: adoption is a transition in a state space — repo state (fresh / main-only / main-only with unpushed commits / main+develop / migrating from another tool — all *detected*, never asked when detectable) × target setup (L1 foundation / +protected branch / +release management, trunk or merge flow). Each target has a **readiness contract** (baseline semver tag; working branch on origin; `release.branch` on origin; `.pk.json` keys). Compute the contract once, surface it in three layers: dashboard (`pk status`), guided transition (`/conventions`), last-resort failure hints (changelog/release).

### B1. `internal/readiness` package (the shared mechanism)

```go
type Check struct {
    Label string // "baseline tag", "develop on origin"
    OK    bool
    Value string // "v0.0.0" / "missing"
    Hint  string // exact next-step command, "" when OK
    Or    string // git equivalent, "" when none
}
func Evaluate(gitExec func(dir string, args ...string) (string, error), dir string, conf config.PkConfig) []Check
```

- **Offline only**: reads local refs (`git tag --list 'v*'`, `refs/remotes/origin/<branch>` via `rev-parse --verify`), never `ls-remote` — status stays fast; release pre-flight keeps its authoritative network checks.
- Checks keyed to what `.pk.json` declares: merge flow (release.branch set) → baseline tag, working branch on origin, release branch on origin, unpushed-commits informational line; trunk flow → baseline tag + current branch on origin. Never nags about layers not opted into.
- Move `hasValidSemverTag` from `internal/setup/baseline.go:12` here (exported, function-param signature instead of `setup.Config`) so setup, status, and readiness share it.

### B2. `pk status` readiness dashboard

`internal/status/status.go`: add `GitExec` to `status.Config` (first subprocess git in status; wired in `DefaultConfig`/`cmd/pk/main.go`). After the `Config (.pk.json):` section, render a `Readiness:` section via msg helpers, only when the repo is git and hooks are installed:

```
Readiness:
  baseline tag        v0.0.0
  develop on origin   missing
    To create and publish it: git switch -c develop && git push -u origin develop
  main on origin      ok
```

All-pass collapses to `Readiness: ready for pk changelog / pk release`. `--brief` appends `ready`/`not-ready`. This directly fixes "configured but not ready" being invisible.

### B3. Failure-point hints (using msg helpers)

- `release.go:66` (on the release branch, the main-only dead-end): append `  To start a working branch: git switch -c develop && git push -u origin develop` + `  Then: pk changelog`.
- Release pre-flight: verify `release.branch` resolves on origin before `git switch`, erroring with a hint instead of the raw git error (~release.go:184).
- `changelog.go:135` (protected branch): add the create-develop hint when no other local branch exists.

### B4. `/conventions` becomes the guided transition

Edit **both** `internal/setup/skills/conventions/SKILL.md` (embedded) and `.claude/skills/conventions/SKILL.md` (local), recompute `pk_sha256` per CLAUDE.md's sed/shasum recipe:

1. **Detect before asking** (new step before the branch questions): `git branch --list`, `git branch -r`, `git tag --list 'v*'`, `git log origin/<default>..<default> --oneline`; summarize ("main only, 2 unpushed commits, no version tags") and frame the questions as a transition from the current setup.
2. **Branch creation offer** (after the `.pk.json` write): when the chosen topology implies a missing working branch, preview the exact commands and **execute after explicit confirmation** (consistent with /ship's preview-confirm-execute). **Unpushed-commits wrinkle is a mandatory question**: when local main is ahead of origin/main, show `git log origin/main..main --oneline` and ask whether develop branches from local `main` (carries the commits) or `origin/main` (they stay on main); then `git branch develop <ref> && git switch develop && git push -u origin develop`.
3. Baseline nudge stays advisory, reworded to the hint standard, gains the `--at` fold-history-vs-start-fresh framing.
4. Closing step: "run `pk status`; the Readiness section confirms the setup."

Per the skill-authoring rule: questions as plain bullets; interpretation context in the Rules section.

### B5. `pk setup`

No behavior change beyond message migration; restyle its two nudges (baseline setup.go:384-387, conventions :404-407); conventions nudge also mentions `pk status` readiness. Setup stays one-command/sensible-defaults; branch creation is interactive judgment → /conventions.

### Docs (B)

- `docs/adoption.md`: new **"Moving between setups"** section between Layers 3 and 4 — main-only → main/develop full recipe (incl. the unpushed-commits judgment call), trunk → merge flow, "pk status is your dashboard"; update the mermaid graph.
- `docs/pk-status.md`: Readiness section + `--brief` change. `docs/pk-release.md`/`pk-changelog.md`/`error-reference.md`: new/changed hints. `docs/pk-json.md`: no new keys (confirm by grep).
- README greps per the enumeration rule: `pk status` description and adoption-doc description likely need "release readiness".

### Out of scope (explicit)

No `pk doctor`/`pk adopt` command; pk binary never creates or pushes branches (only the skill, after confirmation, via plain git); `pk release`'s silent trunk-flow fallback stays (documented design — status/setup now make the gap visible); `new-plankit-project` skill untouched; no GitHub Ruleset automation.

---

## Sequencing (one commit per step on `develop`, `make test` green at each)

1. **A1** standard text into plankit-development.md + `internal/msg` + tests (`feat:`/`docs:` as fits)
2. **A2** `usageFor` flag-help + migrate `cmd/pk/main.go` (`refactor:`)
3. **A3** migrate hooks: guard, protect, preserve, pin (`refactor:`)
4. **A4** migrate setup family, teardown, status, rules (`refactor:`)
5. **A5** migrate changelog + release + `git.ErrDirtyTree` (largest test churn; em-dash rewording lands here) (`refactor:`)
6. **A6** docs pass A: error-reference.md (incl. existing drift), sample outputs (`docs:`)
7. **B1** `internal/readiness` + move `hasValidSemverTag` (`feat:`)
8. **B2** `pk status` Readiness section + `--brief` (`feat:`)
9. **B3** release/changelog failure-point hints (`feat:`)
10. **B4** /conventions skill update, both copies + pk_sha256 (`feat:`)
11. **B5** docs pass B: adoption.md, pk-status.md, README greps (`docs:`)

## Verification

**Automated**: `make test` + `make lint` at every step; `msg_test.go` exact-output per helper; readiness tests with canned GitExec (no tags / tag present / origin ref missing / unpushed commits); status tests for the Readiness section (configured-but-not-ready must print the develop hint); release/changelog tests for new hints; embed safety test auto-covers the skill edit.

**Smoke** (specific commands, expected stderr, incl. negative cases):
- `dist/pk setup --help` → `--kebab-case` flags, not `-flag`
- `dist/pk status` in plankit → new-style headers + `Readiness: ready for pk changelog / pk release`
- Scratch repo reproducing the observed starting state (main-only, unpushed local commits, `.pk.json` with `release.branch: main`, baseline tag absent): `pk status` → `develop on origin   missing` + exact hint; `pk changelog` on main → protected-branch error with create-develop hint (negative case); after following the hints, `pk changelog && pk release --dry-run` runs clean
- `echo bad | dist/pk guard` → `pk guard: failed to read input: ...`, exit 0
- `dist/pk teardown` (no `--confirm`) → preview in new style
- `/conventions` walkthrough in the scratch repo: detects state, asks the local-vs-origin/main branching question, previews commands, executes only after confirm
