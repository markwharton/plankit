# Plan: Redesign CLAUDE.md template and pk setup for modern Claude Code

## Context

Users report Claude ignoring critical rules (e.g., committing to `main` instead of `dev`). The root cause: plankit's 31-rule CLAUDE.md template, combined with project conventions, creates 60+ rules. Claude Code docs confirm: "CLAUDE.md content is a user message, not system prompt -- no guarantee of strict compliance, especially for vague or conflicting instructions." Official guidance: target under 200 lines, prune ruthlessly, convert deterministic rules to hooks.

Claude Code now supports:
- `@path/to/import` syntax in CLAUDE.md (expands into context at launch)
- `.claude/rules/` directory with path-scoped rules via YAML frontmatter
- Plugins marketplace (`/plugin`)
- Custom subagents in `.claude/agents/`

plankit's unique value remains: opinionated template, `pk changelog`, `pk release`, plan preservation to git, one-command setup. But the template needs to follow the platform's own guidance.

## 1. Rename `.changelog.json` to `.pk.json` and restructure

The config file holds more than changelog config. Rename to match the CLI and restructure so each top-level key maps to a `pk` command. Active development, one user -- clean break.

**Rename:** `.changelog.json` to `.pk.json` (project-local only, no global support needed)

**New structure:**
```json
{
  "changelog": {
    "types": [...],
    "versionFiles": [...],
    "hooks": {
      "preRelease": "go test -race ./..."
    }
  },
  "guard": {
    "protectedBranches": ["main"]
  }
}
```

`types`, `versionFiles`, and `hooks` move under `changelog`. Each top-level key owns its command's config. No ambiguity about what `hooks` means.

**Code changes:**
- `internal/changelog/changelog.go` -- `ChangelogConfig` becomes nested under a root `PkConfig`. `LoadConfig` reads `.pk.json`, returns the changelog portion. Update `defaultTypes` fallback.
- `internal/changelog/changelog_test.go` -- update test fixtures for nested structure
- `internal/guard/guard.go` -- reads `PkConfig.Guard` from the same `.pk.json`

**Files to grep for all references:**
- `CLAUDE.md` -- project conventions
- `docs/pk-setup.md` -- config references
- `docs/pk-changelog.md` -- documents the config file
- `docs/getting-started.md` -- may reference config
- `README.md` -- if referenced
- `.changelog.json` renamed to `.pk.json` in plankit's own repo

## 2. Slim the CLAUDE.md template

**File:** `internal/setup/template/CLAUDE.md`

The current 31-rule template becomes a lean critical-rules file. Detailed guidelines move to `.claude/rules/`.

**New structure:**

```markdown
# CLAUDE.md

IMPORTANT: Follow these rules. Read @.claude/rules/ for detailed guidelines.

## Critical Rules
- NEVER take shortcuts without asking. STOP, ASK, WAIT for approval.
- NEVER force push. Make a new commit to fix mistakes.
- NEVER commit secrets to version control.
- Test before and after every change.
- Only do what was asked -- no scope creep.
- Understand existing code before changing it.
- If you don't know, say so. Never guess.
- Surface errors clearly. No silent fallbacks.

## Project Conventions
(added by /init -- build commands, test commands, branch rules, etc.)
```

Target: ~20 lines before project conventions. The 6-8 critical rules are the ones that, if violated, cause the most damage.

## 3. Move detailed guidelines to `.claude/rules/`

**New files installed by `pk setup`:**

| File | Content | Scoped? |
|------|---------|---------|
| `.claude/rules/model-behavior.md` | Honesty, scope discipline, read before writing, testing | No (always loaded) |
| `.claude/rules/development-standards.md` | Data-first, fail fast, consistency, two-pass, security, debugging | No (always loaded) |
| `.claude/rules/git-discipline.md` | Commit with purpose, conventional commits, commit before risk, separate commit/push | No (always loaded) |

These are the current Model Behavior and Development Standards content, reorganized into topic files. Each file should be concise -- the point is modularity, not adding more words.

**Changes to `internal/setup/setup.go`:**
- Add rules file installation alongside skills installation
- Reuse existing `writeManaged` function (frontmatter SHA markers, same as skills -- no new file protection code)
- Store rule templates in `internal/setup/rules/` (embedded via `//go:embed`)

**Changes to `internal/setup/setup_test.go`:**
- Test that rules files are created in `.claude/rules/`
- Verify `writeManaged` behavior for rules (pristine update, modified skip, force overwrite)

## 4. Add `pk guard` for branch protection

**New command:** `pk guard` -- a PreToolUse hook on `Bash` that blocks git operations on protected branches.

**File:** `internal/guard/guard.go`

Follows the `pk protect` pattern exactly:
1. Read PreToolUse JSON from stdin (tool_input contains `command` field for Bash)
2. Parse the git command from `tool_input.command`
3. If the command is a git mutation (`commit`, `push`, `merge`) AND the current branch is protected, block
4. Read protected branches from `.pk.json` `guard.protectedBranches`

If no `guard` key exists in `.pk.json` (or no `.pk.json`), `pk guard` exits 0 (no-op). The hook is always installed by `pk setup`, but only activates when configured.

**File:** `internal/hooks/input.go`
- Add `Command string` field to `ToolInput` struct (non-breaking addition)

**File:** `internal/guard/guard_test.go`
- `TestGuard_blocksCommitOnProtectedBranch`
- `TestGuard_allowsCommitOnUnprotectedBranch`
- `TestGuard_blocksForcePush`
- `TestGuard_allowsReadOnlyGitCommands`
- `TestGuard_noGuardConfigIsNoOp`
- `TestGuard_emptyGuardConfigIsNoOp`

**File:** `internal/setup/setup.go`
- Add `Bash` matcher with `pk guard` hook to `buildHookConfig()`

**File:** `cmd/pk/main.go`
- Add `guard` case to subcommand switch
- Add to help text under "Hook commands"

**File:** `docs/pk-guard.md`
- Command documentation following standard format

## 5. Ship /init as a managed skill

**File:** `internal/setup/skills/init/SKILL.md` (new -- managed skill, installed by `pk setup`)

Move the init prompt from `docs/pk-setup.md` inline content into a shipped skill. Preserve all existing improvements (business/domain rule discovery, domain model relationships, service-level exploration). Changes:

- Update rule: "If the project uses `.pk.json`, include the configured commit types" (was `.changelog.json`)
- Add branch conventions discovery step:
  - What is the default development branch?
  - Are there branches that should never receive direct commits?
- If protected branches specified:
  - Add to Critical Rules section: "NEVER commit directly to [branch]"
  - Add guard config to `.pk.json`

**File:** `docs/pk-setup.md`
- Remove the inline init prompt
- Replace with: "Run `/init` to add project-specific conventions"
- Keep the "Create your own skills" section (still educational)

## 6. Update documentation

**File:** `docs/pk-setup.md`
- Document the new rules directory
- Document guard configuration in `.pk.json`
- Update "How it works" to mention rules installation
- Link to official Claude Code docs:
  - https://code.claude.com/docs/en/memory (CLAUDE.md, imports, `.claude/rules/`, auto memory)
  - https://code.claude.com/docs/en/best-practices (effective CLAUDE.md, hooks, context management)

## Verification

```bash
make test                          # all tests pass
make build                         # binary builds
dist/pk setup --project-dir /tmp/test  # verify: lean CLAUDE.md, rules/, hooks include guard
dist/pk guard < test-payload.json  # verify: blocks commit on main
cat /tmp/test/CLAUDE.md            # verify: short, critical rules only
ls /tmp/test/.claude/rules/        # verify: model-behavior.md, development-standards.md, git-discipline.md
```

## Implementation order

1. Rename `.changelog.json` to `.pk.json` + restructure (foundation)
2. Template restructure + rules directory (highest impact, zero risk)
3. pk guard + hook wiring (enforcement layer)
4. /init skill (discovery layer)
5. Documentation updates (tight loop)

## Files to create

| File | Purpose |
|------|---------|
| `internal/setup/rules/model-behavior.md` | Embedded rule template |
| `internal/setup/rules/development-standards.md` | Embedded rule template |
| `internal/setup/rules/git-discipline.md` | Embedded rule template |
| `internal/setup/skills/init/SKILL.md` | Managed init skill |
| `internal/guard/guard.go` | Branch protection hook |
| `internal/guard/guard_test.go` | Guard tests |
| `docs/pk-guard.md` | Command documentation |

## Files to modify

| File | Change |
|------|---------|
| `.changelog.json` to `.pk.json` | Rename and restructure config |
| `internal/changelog/changelog.go` | Nest under `PkConfig`, read `.pk.json` |
| `internal/changelog/changelog_test.go` | Update fixtures for nested structure |
| `internal/setup/template/CLAUDE.md` | Slim to critical rules only |
| `internal/setup/setup.go` | Install rules, add guard hook |
| `internal/setup/setup_test.go` | Rules + guard hook tests |
| `internal/hooks/input.go` | Add Command field to ToolInput |
| `internal/hooks/input_test.go` | Test Command field parsing |
| `cmd/pk/main.go` | Add guard subcommand + help |
| `CLAUDE.md` | Update config references |
| `docs/pk-setup.md` | Rules, guard, imports, remove inline init |
| `docs/pk-changelog.md` | Update config references |
