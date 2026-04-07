# Plan: pk release redesign — merge flow with releaseBranch config

## Context

Guard blocks git mutations on protected branches, which means Claude Code can't run the full release flow — the merge to main and push must be done manually in the terminal. This redesign makes `pk release` the single command that legitimately touches a protected branch, enabling the full release cycle from Claude Code via `/changelog` → `/release`.

Key insight: `pk release` runs git commands via `exec.Command` internally, not through Claude Code's Bash tool. Guard (a PreToolUse hook) only intercepts Bash tool calls — so `pk release` naturally bypasses guard without any bypass mechanism needed.

## New .pk.json config

```json
{
  "changelog": { ... },
  "guard": {
    "protectedBranches": ["main"]
  },
  "release": {
    "branch": "main"
  }
}
```

- `guard.protectedBranches` — branches guard blocks Claude Code from touching (unchanged)
- `release.branch` — where `pk release` merges to and pushes from (new, lives under release — each command owns its config)

## New pk release behavior

When `release.branch` is configured and the current branch is NOT the release branch:

1. Note current branch (implicit source — no hard-coded "dev")
2. Pre-flight: clean working tree, source branch not behind remote
3. Find version tag at HEAD (optional — no tag is OK for CI/CD workflows)
4. Switch to releaseBranch
5. Merge from source branch (`git merge --ff-only`) — fail if not fast-forward
6. Run preRelease hook
7. Push releaseBranch + tag (if tag exists), or just branch (if no tag)
8. Switch back to source branch
9. Push source branch to sync origin

When `release.branch` is NOT configured — legacy flow (unchanged):
- Validate current branch, find tag, validate semver, push

When already ON the release branch — refuse with message: "you're on the release branch, switch to your development branch first." This prevents accidental pushes without a merge.

Dry-run: simulate without switching, merging, or pushing. Check fast-forward is possible via `git merge-base --is-ancestor`.

Error recovery: if any step fails after switching to the release branch, a Go `defer` ensures we switch back to the source branch before returning the error. Guaranteed cleanup.

## Design decisions

1. **`--branch` flag** — removed. Replaced by `release.branch` in `.pk.json`. No backward compatibility needed (active development, no users yet).
2. **Non-fast-forward merges** — fail with clear message: "merge failed — [release branch] has diverged from [source branch] (not fast-forward). Resolve on [release branch] manually, then try again." This happens when someone commits directly to the release branch from the terminal (guard doesn't block terminal). User resolves in terminal or asks Claude for help.
3. **Push source branch** — yes, completes the full cycle in one command
4. **Already on release branch** — refuse with clear message ("switch to your development branch first"). Prevents accidental push without merge.
5. **No tag (CI/CD flow)** — works, just pushes branch without tag. Tag is required when using `pk changelog` (the changelog flow creates the tag). Without changelog, no tag is expected.
6. **pk changelog on guarded branch** — warn if run on a protected branch ("you're on a protected branch, switch to your development branch first")
7. **Config ownership** — each command owns its config section in `.pk.json`. `release.branch` lives under `release`, not `guard`. Guard only knows about `protectedBranches`.

## Scope

Guard and `releaseBranch` are for multi-branch workflows (e.g., dev/main). Single-branch developers working directly on `main` don't use guard — they run `pk changelog` and `pk release` on `main` with the legacy flow. No configuration needed.

## Implementation

### Phase 1: Guard config — no changes needed

Guard config is unchanged. `release.branch` lives under the `release` section in `.pk.json`, not under `guard`. Guard only knows about `protectedBranches`.

### Phase 2: Changelog guard-awareness

**`internal/changelog/changelog.go`** — at the start of `Run()`, check if current branch is in `guard.protectedBranches`. If so, print a warning and exit: "you're on a protected branch, switch to your development branch first."

**`internal/changelog/changelog_test.go`** — test that changelog warns on a guarded branch.

### Phase 3: Release redesign

**`internal/release/release.go`** — core changes:

Add local config loader for `release.branch` from `.pk.json` (same pattern as guard/changelog — each package reads only what it needs):
```go
func loadReleaseBranch(readFile func(string) ([]byte, error)) string
```

Remove `--branch` flag from Config struct. The release branch comes from `.pk.json` only.

Rewrite `Run()` with the 9-step flow. Key patterns:
- Go `defer` for guaranteed switch-back on failure after branch switch
- `git merge --ff-only` for safe merges
- Refuse if already on release branch ("switch to your development branch first")
- Dry-run checks `git merge-base --is-ancestor` without actually merging
- Tag is optional when `release.branch` is set (supports CI/CD without changelog)

**`internal/release/release_test.go`** — new tests:
- Merge flow happy path (with tag)
- Merge flow without tag (CI/CD)
- Merge flow dry-run
- Already on release branch (refused with message)
- Merge fails (non-fast-forward) — switches back
- Push fails — switches back
- Dirty tree — fails before merge
- Source behind remote — fails before merge
- Legacy flow unchanged — all existing tests pass

### Phase 4: CLI and config

**`cmd/pk/main.go`** — remove `--branch` flag from release command. Release branch comes from `.pk.json` only.

**`.pk.json`** — add `release` section with `"branch": "main"`.

### Phase 5: Documentation (tight loop)

**`docs/pk-release.md`** — rewrite for new two-mode behavior (merge flow + legacy flow)

**`docs/pk-guard.md`** — add `releaseBranch` to configuration section

**`docs/resources.md`** — update release flow section (two commands: changelog + release)

**`CONTRIBUTING.md`** — update release instructions

**`CLAUDE.md`** — update release flow description in Branch & Workflow

### Phase 6: Skills

**Release skill** (both `.claude/skills/release/SKILL.md` and `internal/setup/skills/release/SKILL.md`):
- Update description to mention merge flow
- Instructions stay similar: dry-run first, confirm, run
- `.claude/skills/release/SKILL.md` — update `pk_sha256` in frontmatter after content change

**Changelog skill** (both copies):
- Add note: run on development branch, not on a guarded branch
- `.claude/skills/changelog/SKILL.md` — update `pk_sha256` in frontmatter after content change

**Init skill** (both `.claude/skills/init/SKILL.md` and `internal/setup/skills/init/SKILL.md`):
- Update to discover/propose `release.branch` config during init (alongside existing `guard.protectedBranches` discovery)
- When user specifies a release branch, add it to `.pk.json` under the `release` section
- `.claude/skills/init/SKILL.md` — update `pk_sha256` in frontmatter after content change

## Implementation order

1. `internal/changelog/changelog.go` — guard-awareness (warn on protected branch)
2. `internal/changelog/changelog_test.go` — test guarded branch warning
3. `internal/release/release.go` — core redesign (remove --branch, add merge flow, config loader)
4. `internal/release/release_test.go` — new tests + verify existing pass
5. `cmd/pk/main.go` — remove --branch flag, update help text
6. `.pk.json` — add `release.branch` section
7. `make test` — all green
8. `docs/pk-release.md` — rewrite
9. `docs/pk-guard.md` — no changes needed (guard config unchanged)
10. `docs/pk-changelog.md` — document guarded branch warning
11. `docs/resources.md` — keep git commands, add note about pk-managed flows
12. `CONTRIBUTING.md` — update
13. `CLAUDE.md` — update workflow
14. Skills (release, changelog, init — both copies each)

## Verification

```bash
make test                                    # all tests pass
make build                                   # binary builds
dist/pk release --dry-run                    # verify merge flow dry-run on dev
dist/pk release                              # verify full merge flow (dev → main → push → back to dev)
```

## Files to modify

| File | Change |
|------|--------|
| `internal/changelog/changelog.go` | Warn if run on a guarded branch |
| `internal/changelog/changelog_test.go` | Test guarded branch warning |
| `internal/release/release.go` | Core redesign: remove --branch, merge flow, config loading, error recovery |
| `internal/release/release_test.go` | 9+ new tests for merge flow |
| `cmd/pk/main.go` | Remove --branch flag, update help text |
| `.pk.json` | Add `release.branch` section |
| `docs/pk-release.md` | Rewrite for new behavior |
| `docs/pk-changelog.md` | Document guarded branch warning |
| `docs/resources.md` | Keep git commands, add note about pk-managed flows |
| `CONTRIBUTING.md` | Update release instructions |
| `CLAUDE.md` | Update workflow description |
| `.claude/skills/release/SKILL.md` | Update for merge flow |
| `internal/setup/skills/release/SKILL.md` | Update embedded copy |
| `.claude/skills/changelog/SKILL.md` | Add guard branch note |
| `internal/setup/skills/changelog/SKILL.md` | Update embedded copy |
| `.claude/skills/init/SKILL.md` | Discover/propose release.branch config |
| `internal/setup/skills/init/SKILL.md` | Update embedded copy |
