# Plan: `pk teardown` command

## Context

plankit has no uninstall path. If a user wants to stop using pk, they have to manually delete hooks from settings.json, remove skill/rule files, and clean up directories. The only guidance is a help text line: "To remove hooks, delete the hooks key from .claude/settings.json." A tool that shows you the exit earns more trust than one that doesn't. `pk teardown` is the clean inverse of `pk setup`.

## Approach

New `internal/teardown/` package with a `Run(cfg Config) error` function. Preview by default (safe defaults), `--confirm` to execute. Imports shared helpers from `internal/setup/` (exported with uppercase names).

## What teardown removes

**From `.claude/settings.json`:**
- PreToolUse hooks: pk guard, pk protect (Edit + Write)
- PostToolUse hook: pk preserve
- SessionStart hook: install-pk.sh
- Permission: `Bash(pk:*)` from `permissions.allow`
- Cleans up empty structures (empty hook categories, empty hooks key, empty allow array, empty permissions key)
- If settings.json becomes `{}`, removes the file

**Scanned files (discovered, not hardcoded):**
- Scans `.claude/skills/*/SKILL.md` and `.claude/rules/*.md` for files with `pk_sha256` in frontmatter
- If `pk_sha256` present AND SHA matches body → remove (pristine pk-managed)
- If `pk_sha256` present AND SHA doesn't match → skip (modified by user)
- If no `pk_sha256` → ignore completely (user-created, not pk-managed)
- Example: `.claude/skills/review-code/SKILL.md` has no `pk_sha256` → ignored

**Explicit targets (checked at known paths):**
- `CLAUDE.md` — checked for `<!-- pk:sha256:HASH -->` marker, only removed if pristine
- `.claude/install-pk.sh` — always removed if present (no SHA check)
- `.claude/settings.json.bak` — always removed if present

**Empty directories cleaned up (leaf-first):**
- Skill subdirs that are now empty (e.g., `.claude/skills/changelog/`)
- `.claude/skills/` if empty (but not if user skills like review-code remain)
- `.claude/rules/` if empty
- `.claude/` if completely empty

**NOT removed:**
- `.pk.json` — user configuration
- `docs/plans/` — user's preserved plans
- User hooks on same matchers — only pk hooks removed
- User-modified managed files — skipped with message
- User-created skills/rules without `pk_sha256` — invisible to teardown

## Config struct

```go
type Config struct {
    Stderr     io.Writer
    ProjectDir string
    Confirm    bool
    ReadFile   func(string) ([]byte, error)
    WriteFile  func(string, []byte, os.FileMode) error
    Remove     func(string) error
    Stat       func(string) (os.FileInfo, error)
    ReadDir    func(string) ([]os.DirEntry, error)
}
```

Follows the changelog/release pattern (injected filesystem functions) rather than setup's direct os calls.

## Algorithm

**Phase 1 — Analyze:**
- Read settings.json, identify pk hooks and permissions
- Scan `.claude/skills/*/SKILL.md` for files with `pk_sha256` frontmatter
- Scan `.claude/rules/*.md` for files with `pk_sha256` frontmatter
- Check `CLAUDE.md` for `<!-- pk:sha256:HASH -->` marker
- Check `.claude/install-pk.sh` and `.claude/settings.json.bak` existence
- For each file with a pk marker, compare stored SHA to body SHA
- Build action list: remove (pristine), skip (modified), ignore (no marker)

**Phase 2 — Preview:** Print grouped summary to stderr (Settings, Skills, Rules, Files, Directories). If no `--confirm`, print "Run with --confirm to apply." and return.

**Phase 3 — Execute:** Remove files, clean empty dirs, edit settings.json, print "Restart Claude Code to apply changes."

## Output format

```
Settings (.claude/settings.json):
  PreToolUse[Bash]: pk guard ... removed
  PostToolUse[ExitPlanMode]: pk preserve --notify ... removed
  permissions.allow: Bash(pk:*) ... removed
Skills:
  changelog/SKILL.md ... removed
  init/SKILL.md ... removed
Rules:
  development-standards.md ... removed
CLAUDE.md ... skipped (modified by user)
Restart Claude Code to apply changes.
```

Preview mode uses "will remove" / "will skip" instead. Footer reads:
```
Run with --confirm to apply these changes.
```

If any files were skipped due to user modification, append:
```
Skipped files were modified after setup. To remove manually:
  rm .claude/rules/development-standards.md
```

## Files to modify

1. **`internal/setup/setup.go`** — Export three helpers:
   - `isPlankitHook` → `IsPlankitHook`
   - `extractSHA` → `ExtractSHA`
   - `contentSHA` → `ContentSHA`

2. **`internal/setup/setup_test.go`** — Update references to match new exported names.

3. **`cmd/pk/main.go`** — Add `"teardown"` case, `runTeardown()` function, import, update `printUsage()` (add teardown to User commands, remove line 259 "To remove hooks..." hint).

## Files to create

4. **`internal/teardown/teardown.go`** — Config, DefaultConfig, Run, removeHooks, removePermission helpers.

5. **`internal/teardown/teardown_test.go`** — Tests covering:
   - Fresh setup then teardown (full cycle)
   - Preview only (nothing changes)
   - Mixed hooks (user + pk on same matcher — only pk removed)
   - User-modified skills/rules (skipped with message)
   - User-created skills without pk_sha256 (ignored completely)
   - Modified vs pristine CLAUDE.md
   - No settings.json (still cleans up other files)
   - Corrupt settings.json (error)
   - Empty project (nothing to do)
   - Permission cleanup (removes pk permission, keeps others)
   - Empty structures cleaned up (hooks key, permissions key)
   - Directory cleanup (empty dirs removed, dirs with user files left)
   - Idempotent (running twice succeeds)

## Documentation

6. **`docs/pk-teardown.md`** — Command doc following `docs/command-doc-template.md`.

7. **`README.md`** — Add teardown to Commands table.

8. **`docs/getting-started.md`** — Mention teardown exists (one line).

## Verification

```bash
# Build
make build

# Tests
make test

# Smoke test: setup then teardown preview
cd /tmp && mkdir pk-test && cd pk-test && git init
pk setup
pk teardown              # preview — nothing changes
ls .claude/              # still there

# Smoke test: confirmed teardown
pk teardown --confirm    # executes
ls .claude/              # gone (or only user files remain)
cat CLAUDE.md            # gone if pristine

# Smoke test: idempotent
pk teardown --confirm    # "No plankit artifacts found."

# Clean up
cd / && rm -rf /tmp/pk-test
```
