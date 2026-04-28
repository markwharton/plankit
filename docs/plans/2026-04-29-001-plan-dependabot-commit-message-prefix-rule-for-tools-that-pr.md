# Plan: dependabot commit-message prefix + rule for tools that produce commits

## Context

`.github/dependabot.yml` watches the `github-actions` ecosystem (correctly — pinned action SHAs need supply-chain visibility) but produces default Dependabot commit messages like `Bump actions/checkout from X to Y`. These aren't Conventional Commits, so when they merge, `pk changelog` silently skips them. The result: supply-chain bumps that genuinely change the project's security posture don't appear in any release's CHANGELOG entry.

Adding `commit-message: prefix: "chore(deps)"` to the existing block produces messages like `chore(deps): bump actions/checkout from <sha> to <sha>` that flow into the Maintenance section at release time. Lightweight value, real audit trail.

A `gomod` block was considered but dropped: plankit's `go.mod` declares zero third-party dependencies and the "Go 1.21, standard library only" convention is a deliberate principle. A gomod ecosystem watcher would do nothing today; if a Go dependency is ever added, the dependabot block can land in the same PR as that dependency.

While we're here, the same principle — *tools that produce commits should follow the convention* — generalizes beyond Dependabot. Adding a one-line bullet to `git-discipline.md` lets every project that runs `pk setup` pick up the guidance, so their own Dependabot configs (or release-bots, or any other commit-producing automation) are configured correctly from the start.

## Verified before planning

- **`.github/dependabot.yml` exists** with one `github-actions` block. ✓
- **`chore` is a valid type in `.pk.json`** (mapped to "Maintenance"). ✓
- **No commitlint hook** in `.github/workflows/` would reject `chore(deps)`. ✓
- **No Dockerfile, no submodules** — github-actions is the only ecosystem worth watching today. ✓
- **`git-discipline.md` already states** "Follow Conventional Commits to make history scannable" — the new bullet applies that principle to automation. ✓

## Recommended end state

### Change 1: `.github/dependabot.yml`

Add `commit-message:` to the existing block. No new ecosystems.

```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    target-branch: "develop"
    commit-message:
      prefix: "chore(deps)"
```

### Change 2: `internal/setup/rules/git-discipline.md` and `.claude/rules/git-discipline.md`

Add one bullet near the existing "Commit with purpose" line. Proposed wording:

> **Configure automation that produces commits to follow the convention.** Dependabot, release bots, and any tool that opens PRs or pushes commits should set a conventional `commit-message: prefix:` (e.g., `chore(deps)`) so their work flows into `pk changelog` rather than getting silently skipped at release time.

Both copies must be updated together. The local copy carries `pk_sha256` frontmatter — recompute the hash with the existing pattern from CLAUDE.md:

```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256
```

…and replace the `pk_sha256` line in `.claude/rules/git-discipline.md` with the new value.

## Critical files to modify

- **`.github/dependabot.yml`** — add the `commit-message` block.
- **`internal/setup/rules/git-discipline.md`** — add the new bullet (embedded source).
- **`.claude/rules/git-discipline.md`** — add the same bullet + recompute `pk_sha256`.

## Verification

```bash
# 1. YAML validity for dependabot.yml
python3 -c 'import yaml; yaml.safe_load(open(".github/dependabot.yml"))' && echo "yaml ok"

# 2. Build + tests still pass (rule body change shouldn't affect anything but smoke it).
make build
make test

# 3. Sanity-check the recomputed hash against the embedded source.
sed -n '/^---$/,/^---$/!p' internal/setup/rules/git-discipline.md | shasum -a 256
grep '^pk_sha256:' .claude/rules/git-discipline.md
# The two values should match.

# 4. Confirm pk setup, when re-run, doesn't re-write git-discipline.md
#    (proves the local copy's pk_sha256 matches its body).
./dist/pk setup --project-dir /tmp/pk-rule-smoke
# Then re-run on this repo:
./dist/pk setup
# git status should NOT show .claude/rules/git-discipline.md as modified
# (other than the local copy already being updated by hand).
```

Post-merge verification (manual):

1. Repo → Insights → Dependency graph → Dependabot → confirm config is recognized.
2. Wait for the next scheduled Dependabot run; verify any opened PR uses subject `chore(deps): bump …`.
3. Next time `pk changelog` runs at release time, confirm dep bumps appear under Maintenance.

## Commit shape

Two separate commits per project convention (rule changes ship to other repos, ci config doesn't):

1. `ci: add chore(deps) prefix to dependabot config` — touches `.github/dependabot.yml` only.
2. `docs(rules): add bullet for commit-producing automation` — touches both rule copies. (Or use `chore(rules):` — `docs(rules)` matches recent precedent for rule edits.)

Single PR or two PRs? Two commits, single PR-equivalent flow on `develop`. Don't push without explicit go-ahead.
