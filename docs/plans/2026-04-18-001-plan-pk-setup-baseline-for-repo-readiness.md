# Plan: `pk setup --baseline` for repo readiness

## Context

`pk changelog` requires a version tag as its starting anchor. Today, three scenarios land builders in different places:

1. **New repo, initial commit.** Tag HEAD as `v0.0.0`.
2. **Existing repo, tag current state.** Tag HEAD as `v0.0.0`.
3. **Existing repo, fold prior commits into first changelog.** Tag an earlier ref (e.g., first commit) as `v0.0.0`.

Currently the only guidance is an error-message hint in `docs/pk-changelog.md:133` suggesting raw `git tag` + `git push`. For new builders, this is easy to miss — they read docs, forget the step, run `pk changelog`, hit the error, then context-switch to git. Experienced users do it by hand too. Either way it's friction.

This plan adds `pk setup --baseline [--at <ref>] [--push]` so the whole thing lives inside the pk surface, and adds a discoverability tip to `pk setup` when no version tags are found.

## Design

`--baseline` is a flag on `pk setup`, not a subcommand. Running `pk setup --baseline` performs normal setup AND creates the baseline tag. This composes naturally: setup is already idempotent (via `--force`), and the baseline operation is idempotent (no-op if any version tag exists).

### Contract

| Command | Behavior |
|---|---|
| `pk setup` | Normal setup. If no valid semver tag exists and repo is a git repo, print tip at the end: `No version tags found — run 'pk setup --baseline' to anchor pk changelog`. |
| `pk setup --baseline` | Normal setup, then create `v0.0.0` on HEAD locally. If any tag matching `v*` parses as a valid semver (via `version.ParseSemver`), no-op and print `Found tag <tag> — already anchored`. On creation, print `Tagged v0.0.0 on HEAD (run 'pk setup --baseline --push' to publish, or 'git push origin v0.0.0')`. |
| `pk setup --baseline --at <ref>` | Same, but tag `<ref>` instead of HEAD. Validates via `git rev-parse --verify <ref>`; clean error if ref doesn't resolve. |
| `pk setup --baseline --push` | Same as `--baseline`, then `git push origin v0.0.0`. |
| `pk setup --baseline --at <ref> --push` | All three compose: validate ref, tag it as `v0.0.0` locally, push. |
| `pk setup --push` (without `--baseline`) | Error: `--push requires --baseline`. |
| `pk setup --at <ref>` (without `--baseline`) | Error: `--at requires --baseline`. |

### Notes on design choices

- **HEAD as default (not initial commit).** Scenario 3 is rare; scenarios 1 and 2 both mean "anchor from here." Silently folding 50 prior commits into the first changelog would be surprising.
- **No-op on any valid semver tag (not just `v0.0.0`).** The check must match what `pk changelog` actually looks for. Per `internal/changelog/changelog.go:169-184`, changelog lists `v*` tags and feeds each to `version.ParseSemver`. "Anchored" means "a tag exists that pk changelog will accept as a base version." A lookalike like `v-my-thing` matches the `v*` glob but fails semver parse, so it's not an anchor. Reuse `version.ParseSemver` for both the no-op check and the tip condition — same source of truth as pk changelog.
- **`--push` opt-in, matching `pk preserve`.** Per `docs/pk-preserve.md:28` and the git-discipline rule that commit and push are separate decisions. Creating a tag locally is reversible; publishing is not. Never push without an explicit request. The printed next-step hint (`git push origin v0.0.0`) is not a new pattern — `internal/changelog/changelog.go:177` and `internal/preserve/preserve.go:157` already print git commands as "here's what to run next" for composable operations. Tag pushes are branch-independent (tags are refs, not branch commits), so the hint works regardless of which branch the user is on.
- **Tip is non-invasive.** Only printed when setup runs in a git repo AND no valid semver tag exists. Won't fire on re-runs against already-anchored repos.

## Files to modify

### `cmd/pk/main.go`
- Extend `runSetup` (around line 145) to parse `--baseline`, `--at`, `--push`.
- Validate: `--push` and `--at` require `--baseline` — error if not.
- Pass through to `setup.Config` fields.
- Update the usage line (around line 307) to reflect the new flags.

### `internal/setup/setup.go`
- Add to `Config` struct (line 317):
  - `Baseline bool`
  - `BaselineAt string`
  - `Push bool`
  - `GitExec func(projectDir string, args ...string) (string, error)` — mirror the pattern from `internal/preserve/preserve.go:29`.
- Add to `DefaultConfig()` (line 328): `GitExec: pkgit.Exec`.
- Add a new helper `runBaseline(cfg Config, projectDir string) error` — called from the end of `Run()` when `cfg.Baseline` is true, after existing setup completes.
- Add a helper `hasValidSemverTag(cfg Config, projectDir string) (string, bool)` that lists `v*` tags via `cfg.GitExec` and returns the first one for which `version.ParseSemver` succeeds. Used by both the no-op check (`runBaseline` skips creation) and the tip condition (`Run` skips the tip when a valid tag exists).
- Add an end-of-`Run()` tip: if `cfg.Baseline` is false AND repo is a git repo AND `hasValidSemverTag` returns false, print the tip to `cfg.Stderr`.

### `internal/setup/setup_test.go`
- New tests:
  - `TestRun_baseline_createsV000OnHEAD`
  - `TestRun_baseline_noOpWhenTagExists` (seed a tag, assert no creation)
  - `TestRun_baselineAt_usesRef` (seed multiple commits, tag at older ref)
  - `TestRun_baselineAt_invalidRef` (expect error)
  - `TestRun_baselinePush_callsGitPush` (mock GitExec, verify push args)
  - `TestRun_pushWithoutBaseline_errors` (exercised at `main.go` validation level — also add a cmd-level test or handle in validation)
  - `TestRun_noTagsTip_shownWhenNoTags`
  - `TestRun_noTagsTip_hiddenWhenTagsExist`
- Mock `GitExec` via function injection, matching the pattern in `internal/preserve/preserve_test.go`.

### `docs/pk-setup.md`
- Add a "Baseline tag for pk changelog" section covering the three scenarios with exact commands. Link from `docs/getting-started.md` if it doesn't already point to `pk-setup.md`.

### `docs/pk-changelog.md`
- Update the error hint at line 133 to point at `pk setup --baseline` as the preferred path. Keep the raw git fallback for users who want the literal commands.

### `site/pk/start/index.html`
- No change in this PR. The site's "Setup" section calls out `pk setup` flags lightly; we can revisit after the command ships.

## Reuse

- **`internal/git/exec.go:Exec`** — wrapped into `cfg.GitExec` for testability. Same pattern as `internal/preserve/preserve.go:29`.
- **`internal/git/isrepo.go:IsRepo`** — already used at `setup.go:438` for repo detection. Reuse for the tip condition.
- **`internal/version/version.go:ParseSemver`** — use for validating tags the same way `pk changelog` does.
- **Flag-parsing pattern** — copy shape from `runPreserve` at `cmd/pk/main.go:112`. `fs.Bool` for `--baseline` / `--push`, `fs.String` for `--at`.

## Verification

Automated:
- `make test` passes with the new tests.
- `make lint` / `go vet ./...` clean.

Smoke (the project convention requires manual end-to-end checks for observable behavior):

1. **Scenario 1 — new repo, clean baseline.**
   ```bash
   cd $(mktemp -d) && git init && pk setup --baseline
   git tag         # expect: v0.0.0
   git rev-parse v0.0.0 == git rev-parse HEAD
   ```

2. **Scenario 2 — existing repo with commits, tag current state.**
   ```bash
   cd some-repo-with-history
   pk setup --baseline
   git tag         # expect: v0.0.0 added; points at HEAD
   ```

3. **Scenario 3 — existing repo, tag an earlier commit.**
   ```bash
   cd some-repo-with-history
   pk setup --baseline --at $(git rev-list --max-parents=0 HEAD)
   git rev-parse v0.0.0 == git rev-list --max-parents=0 HEAD
   ```

4. **Idempotency.**
   ```bash
   pk setup --baseline      # prints: Found tag v0.0.0 — already anchored
   ```

5. **Push opt-in.**
   ```bash
   pk setup --baseline --push
   git ls-remote --tags origin | grep v0.0.0   # expect: present on remote
   ```

6. **Flag validation.**
   ```bash
   pk setup --push           # expect: exit non-zero, "--push requires --baseline"
   pk setup --at HEAD~1      # expect: exit non-zero, "--at requires --baseline"
   ```

7. **Tip visibility.**
   ```bash
   cd fresh-repo && pk setup              # expect tip
   cd fresh-repo && pk setup --baseline   # no tip, runs baseline
   cd fresh-repo && pk setup              # no tip (tag exists now)
   ```

8. **Non-git repo.**
   ```bash
   cd $(mktemp -d) && pk setup --allow-non-git --baseline
   # expect: clean error about --baseline requiring a git repo
   ```

At least one negative case per the project rule: covered by #6 and #8.

## Out of scope (deliberately)

- **Aligning existing `setup.go` git calls with the `cfg.GitExec` pattern.** Packages were written at different times. `internal/preserve/preserve.go:29` injects `GitExec` via Config. `internal/setup/setup.go:438` uses the package-level `git.IsRepo(os.Stat, ...)` directly. Both work; neither is a regression. This plan adds `cfg.GitExec` to `setup.Config` only for the new `--baseline` code paths. Migrating existing calls would be a nice consistency refactor but is not required and risks scope creep. If done later, it's a standalone PR.
- Updating `site/pk/start/` — deferred to the next site update bundle.
- A `pk baseline` subcommand. Flag on `pk setup` is the agreed shape; a standalone subcommand would double the surface area without adding capability.
