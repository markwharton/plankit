# Plan: Fix hooks merge, help --force, and docs ordering

## Context

`pk setup` silently replaces all existing hooks in `.claude/settings.json` instead of merging plankit hooks with user hooks. This is a data integrity bug — users lose their custom hooks. Additionally, `pk help` is missing the `--force` flag for setup, and the `pk-*.md` docs have inconsistent section ordering.

## 1. Hooks merge (critical)

**File:** `internal/setup/setup.go`

Add a `mergeHooks` function following the existing `addPermission` pattern:

1. Parse existing `settings["hooks"]` into `HooksConfig` (if present)
2. For each category (PreToolUse, PostToolUse): filter out hooks where command starts with `"pk "` from existing entries. If an entry's hooks array becomes empty after filtering, drop the entry entirely.
3. Append plankit's new entries after the preserved user entries
4. Marshal merged result back to `settings["hooks"]`

Replace line 271 (`settings["hooks"] = json.RawMessage(hooksJSON)`) with a call to `mergeHooks`.

Update `pk-setup.md` "What it does" item 1 to mention that existing hooks are preserved (not replaced).

**Tests:** `internal/setup/setup_test.go`

- `TestMergeHooks_freshSettings` — no existing hooks, plankit entries added
- `TestMergeHooks_existingUserHooks` — user hooks preserved, plankit appended
- `TestMergeHooks_existingPlankitHooks` — old plankit hooks replaced with new config
- `TestMergeHooks_mixedHooks` — entry with both user and plankit hooks: plankit removed, user kept, new plankit appended
- `TestRun_existingHooks` — integration: existing user hooks in settings.json survive `pk setup`

## 2. Help missing --force

**File:** `cmd/pk/main.go` line 170

Change:
```
pk setup [--project-dir <dir>] [--preserve auto|manual]
```
To:
```
pk setup [--force] [--project-dir <dir>] [--preserve auto|manual]
```

## 3. Docs ordering

**Standard section order** (4 of 6 files already follow this):
1. Title + one-line description
2. Usage
3. How it works
4. Detail/config sections
5. Flags
6. Hook protocol (hook commands only)
7. Environment (if applicable)

**Specific changes:**

- **pk-changelog.md** — Add `## Flags` section before "Comparison links" with `--bump` and `--dry-run`
- **pk-release.md** — Rename "What it does" to "How it works", fold "Workflow" context into the intro or a note
- **pk-setup.md** — Rename "What it does" to "How it works"; reorder Usage examples to match Flags order (preserve, force, project-dir)

Files already correct: pk-preserve.md, pk-protect.md, pk-version.md

## Verification

```bash
make test        # all tests pass including new merge tests
make build       # binary builds
dist/pk          # help shows --force for setup
dist/pk setup    # verify on a test project with existing hooks
```

## Files to modify

| File | Change |
|------|--------|
| `internal/setup/setup.go` | Add `mergeHooks`, replace line 271 |
| `internal/setup/setup_test.go` | 5 new tests |
| `cmd/pk/main.go` | Add `--force` to help |
| `docs/pk-changelog.md` | Add Flags section |
| `docs/pk-release.md` | Rename "What it does" → "How it works" |
| `docs/pk-setup.md` | Rename "What it does" → "How it works", update hook description, reorder usage |
