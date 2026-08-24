# Trunk flow: refuse to release from a non-default branch

## Context

A contributor working in a trunk-flow repository expected `/ship` on a feature branch to block ("needs main/develop"). That holds for merge flow only. In trunk flow (`.pk.json` with no `release.branch` and no `guard.branches`), the only branch gates in `pk changelog` and `pk release` are "the branch exists on origin" and "not behind its own origin counterpart" (verified: `changelog.go:149-158`, `release.go:120-155`). A pushed feature branch passes both: changelog computes a version, release tags HEAD on the feature branch and pushes branch + tag atomically (`release.go:294`, `:326`). The result is a published release tag on unmerged work — in exactly the repos trunk flow is recommended for, where a stray tag is a stray production version. Neither dry-run names the branch it will tag and push; the release dry-run's only "Would" line is `Would create tag vX.Y.Z` (`release.go:286`; `pushBranch` is computed at `:321`, after the dry-run returns at `:288`).

## Decision: gap to close, not a decision to document

- Trunk flow is documented as a single-branch model: `docs/pk-json.md`'s workflow example is headed "Trunk flow (single branch, no guard)", and the draft trunk-flow page (in plankit-www, a private repo testing a new plankit website) opens with "the repository has one branch, main". Releasing from a pushed non-default branch is outside that model; nothing enforces it.
- Plankit's principles: safe defaults with opt-in escalation, and enforce over prose (the guard evals showed documentation alone doesn't stop an agent mid-session).
- The check is derivable from an existing primitive: origin's default branch via `git ls-remote --symref origin HEAD` — a network call in pre-flights that already make `ls-remote` network calls.

**Shape:** in trunk flow only, both commands refuse when the current branch is not the default branch on origin. No new flag, no new config key (the shipped conduct rule "`pk release`'s only flag is `--dry-run`" stays true). Remediation hints show the merge-to-default path; the deliberate escape hatches (change the default branch on the host, or tag manually with git) are documented in `error-reference.md` only — a safety refusal must not print its own bypass into the session log where the agent reads it.

## Implementation (plankit repo, on `develop`)

### 1. New helper `internal/git/defaultbranch.go`

Follow `cleantree.go`'s shape (injected `gitExec` first param, bare returns, semantics in the doc comment):

```go
func DefaultBranch(gitExec func(dir string, args ...string) (string, error), dir string) (string, bool, error)
```

Runs `ls-remote --symref origin HEAD`. On command error: `"", false, err`. Scan output for the line with prefix `ref: refs/heads/` and suffix `\tHEAD`; trim both to get the name (handles slashed names). No `ref:` line (remote advertises no HEAD): `"", false, nil`. Callers treat `false` and error identically: no default established, skip the check.

### 2. `internal/changelog/changelog.go`

- **Enabling change:** add `Release config.ReleaseSection` to `FullConfig` (`changelog.go:381-384`) and populate it in `LoadFullConfig` (`:397`) — `config.Load` already returns it; the struct just drops it today.
- **New check** between the guard check (`:133-144`) and branch-on-origin (`:149-158`). It must precede branch-on-origin: on an *unpushed* feature branch, that check's hint "To push it: git push -u origin feature" would walk the user straight into the new refusal. Guarded by `fullConfig.Release.Branch == ""` (trunk flow only); follows the existing idiom — `branch --show-current` with `err == nil` (git failure skips), `branch != ""` (detached HEAD skips), `DefaultBranch` error or `!ok` skips. Runs in `--dry-run` like the neighbouring checks, so the dry-run surfaces the refusal:

```go
if fullConfig.Release.Branch == "" {
    if branch, err := cfg.GitExec(cfg.Dir, "branch", "--show-current"); err == nil {
        branch = strings.TrimSpace(branch)
        if branch != "" {
            if def, ok, derr := pkgit.DefaultBranch(cfg.GitExec, cfg.Dir); derr == nil && ok && branch != def {
                msg.Errorf(cfg.Stderr, "you're on %q but the default branch on origin is %q; trunk flow releases from the default branch", branch, def)
                msg.Hintf(cfg.Stderr, "To release this work from %s: git switch %s && git merge %s", def, def, branch)
                msg.Hintf(cfg.Stderr, "Then: pk changelog && pk release")
                return 1
            }
        }
    }
}
```

(Message rules honoured: lowercase after `Error:`, semicolon not em dash, no trailing period, runnable hints; `Then:` precedent at `release.go:80`. Plain `git merge`, not `--ff-only`, so the hint also works when the default branch has advanced.)

### 3. `internal/release/release.go`

- **The check**, between clean-tree (`:114-118`) and exists-on-origin (`:120-128`) — same ordering rationale, and release re-checking what changelog refused matches the existing branch-on-origin pattern. Guarded by `releaseBranch == "" && sourceBranch != ""`. Same message and the first two hints as changelog, but `Then: pk release` (the fast-forward carries the Release-Tag trailer to the default branch's HEAD). Track `defaultVerified := true` on the pass path.
- **Trunk pre-flight line** (`:255`): when `defaultVerified`, print `On %s (default branch on origin)`; otherwise keep `On %s branch`. The differentiated line quietly discloses when the check could not run (symref-less remote), without adding a recurring `Note:`.
- **Dry-run push line:** hoist the `pushBranch` computation (`:321-324`) above the dry-run block (`:283-289`), then after `Would create tag %s` add `msg.Itemf(cfg.Stderr, "Would push %s and %s", pushBranch, tag)` — both flows; mirrors `Pushed %s and %s` (`:331`), per the dry-run-mirrors-real-run rule. Merge flow's best-effort post-switch source push (`:340-344`, warn-only) deliberately gets no "Would" line: it is not part of the release guarantee.

### 4. Tests

- **`internal/git/defaultbranch_test.go`** (new): parses `ref: refs/heads/main\tHEAD\n<sha>\tHEAD\n` → `("main", true, nil)`; slashed name; output without `ref:` and empty output → `("", false, nil)`; gitExec error → `("", false, err)`; asserts exact args and dir forwarding.
- **`internal/release/release_test.go`:** `stubGitExec` dispatches on `args[0]`, so the single `ls-remote` handler in `happyGit` (`:45-47`) receives both the symref and heads queries — branch on `args[1] == "--symref"`, returning `ref: refs/heads/<branch>\tHEAD\n...` (fixture semantics: the fixture branch *is* origin's default; all existing happy/merge/dry-run tests stay green, verified — no test asserts `On main branch`). New tests: `TestRun_trunkFlow_notOnDefaultBranch` (exit 1, names both branches, hints present, no `tag`/`push` calls recorded), `TestRun_trunkFlow_noRemoteHead` (no `ref:` line → proceeds, output has `On main branch` not `(default branch on origin)`), `TestRun_trunkFlow_defaultBranchLine` (happy output has the new line), `TestRun_mergeFlow_noDefaultBranchCheck` (no `--symref` call in merge flow). Extend `TestRun_dryRun` and the merge-flow dry-run test to assert `Would push <branch> and <tag>`.
- **`internal/changelog/changelog_test.go`** (inline GitExec style): `TestRun_trunkFlow_notOnDefaultBranch` (refusal, no `commit` call), `TestRun_trunkFlow_onDefaultBranch` (happy), `TestRun_trunkFlow_noRemoteHead` (skip path), `TestRun_mergeFlow_skipsDefaultBranchCheck` (config sets `release.branch`; no `--symref` call), plus a `LoadFullConfig` assertion that `Release.Branch` populates. Existing tests returning `""` for `branch` or erroring all ls-remote (e.g. `TestRun_branchNotOnOrigin:1382`) skip the new check and stay green.
- `make test` (race), `make lint`, `make cover` (per-function coverage of changed files; cover the skip/error branches through the mocks).

### 5. Docs (same pass, tight loop; reference sentences ≤25 words)

- **`docs/pk-changelog.md`** "How it works": new step between the guard step and branch-on-origin — trunk-flow default-branch verification, noting the skip when origin advertises no HEAD; renumber.
- **`docs/pk-release.md`**: trunk steps `:26-33` (pre-flight list gains the default-branch check); `branch` bullet `:61` (omitted ⇒ both commands refuse a non-default branch); workflows table `:78-81` trunk row → "Tag HEAD on the default branch, push it + tag"; scope statement `:113` ("directly on their working branch" → "directly on the default branch on origin").
- **`docs/error-reference.md`**: new `### not the default branch` entry under both `## pk changelog` and `## pk release` (duplication precedented by "branch not on origin"). Fenced verbatim message + hints; **Cause:** trunk flow publishes from origin's default branch; a tag elsewhere would publish unmerged work; **Fix:** merge into the default branch and rerun; or change the repository's default branch on your host; or tag manually with git (`git tag <tag> && git push origin <branch> <tag>`). Also: if the default branch is listed in `guard.branches` with no `release.branch`, the config is a dead end — configure `release.branch` (merge flow) instead. Note the skip when origin advertises no HEAD.
- **`docs/pk-json.md`**: `release.branch` "Omitted" bullet `:139-144` and the trunk workflow example `:193` — current branch must be the default branch on origin.
- **`README.md`** `:24`: "`pk release` tags the branch you're on and pushes" → tags your default branch and pushes.
- **No `/ship` skill change** ("for trunk-based projects, that's the main branch" already matches; avoids the dual-copy + `pk_sha256` procedure). **No `conduct.md` change** (no new flag).
- Commit as `feat:` so `pk changelog` surfaces it.

### 6. Draft site repo (`plankit-www` — private repo testing a new plankit website; separate change, after the pk release so transcripts come from the released binary)

- **`site/trunk-flow/index.html`**: pre-flight sentence gains the default-branch check; pk changelog bullet gains the refusal; both verbatim transcript blocks updated from real released-pk output (`On main (default branch on origin)`, dry-run gains `Would push main and <tag>`); "When it stops" table gains the *not the default branch* row (nothing written; merge into main and re-run).
- **`site/merge-flow/index.html`**: dry-run transcript gains `Would push main and <tag>`.
- Ground-truth rule: regenerate transcripts verbatim from the released binary, not by hand.

### Out of scope (stated deliberately)

- `pk status`/readiness: offline-only by design; no network call added there.
- Merge flow's own semantics (including fast-forwarding the release branch from a feature branch) — verification asserts merge flow is untouched.
- `pk guard` changes.

## Verification (smoke test, in the scratchpad)

Build with `make build`; use `dist/pk` explicitly (live hooks run the installed pk, not `dist/pk`).

1. **Setup:** `git init --bare origin.git` (HEAD → main); clone to `work/`; commit; `git tag v0.0.0`; `git push origin main v0.0.0`. No `.pk.json`.
2. **Positive refusal (the reported case):** `git switch -c feature && git push -u origin feature`, add a `feat:` commit, run `pk changelog` → refusal naming `"feature"` and `"main"`, exit 1, no CHANGELOG written. Fabricate the release side (`git commit --allow-empty` with a `Release-Tag:` trailer, push), run `pk release --dry-run` → same refusal, no tag created.
3. **Negative (default branch):** on `main`, `pk changelog` then `pk release --dry-run` → pre-flight shows `On main (default branch on origin)`, `Would create tag` + `Would push main and v0.1.0`; real `pk release` → `Pushed main and v0.1.0`, tag on origin.
4. **Merge-flow regression:** add `{"release":{"branch":"main"}}`, work on pushed `develop` → no default-branch refusal, `Would merge` / `Would push main and vX.Y.Z`, real run completes and switches back. And with `guard.branches: ["main"]`, `pk changelog` on main still refuses with "switch to your development branch first".
5. **Skip path:** `git --git-dir=origin.git symbolic-ref HEAD refs/heads/gone`, then trunk release from a pushed non-default branch proceeds and prints `On <branch> branch` (unit-tested regardless).
6. `make test`, `make lint`, `make cover` — empty cover list.
