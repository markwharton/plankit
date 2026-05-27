# Specify Bash for pk hooks and skills on Windows

## Context

Claude Code on Windows is progressively rolling out a PowerShell tool (`CLAUDE_CODE_USE_POWERSHELL_TOOL`) that replaces Bash as the primary shell. This broke plan preservation: the PostToolUse hook fires but `pk preserve` receives no stdin payload, preserving nothing. The fix is to explicitly specify bash in hooks, skills, and guard matchers so plankit works correctly regardless of which shell Claude Code defaults to.

## Changes

### 1. Add `Shell` field to Hook struct

**File:** `internal/setup/claude.go`

Add `Shell string` with JSON tag `"shell,omitempty"` to the Hook struct between `Async` and `Timeout`. This matches the Claude Code docs field ordering (type, command, async, shell, timeout, statusMessage).

### 2. Set `Shell: "bash"` on all plankit hooks

**File:** `internal/setup/claude.go`

In `buildHookConfig()` and `preserveHookEntry()`, add `Shell: "bash"` to every Hook literal. This ensures hook commands always run in Git Bash on Windows, regardless of PowerShell tool state.

### 3. Change guard matcher from `"Bash"` to `"Bash|PowerShell"`

**File:** `internal/setup/claude.go`

In `buildHookConfig()`, change the guard hook's matcher from `"Bash"` to `"Bash|PowerShell"`. The pipe-separated syntax is documented: `Edit|Write` matches either tool exactly. This ensures `pk guard` fires for git mutations regardless of which shell tool Claude uses.

No new PreToolUse entry needed — same entry, wider matcher. Entry count stays at 3.

### 4. Update preserve skill to specify Bash tool

**Files:** `internal/setup/skills/preserve/SKILL.md`, `.claude/skills/preserve/SKILL.md`

Add a Bash tool rule. The `allowed-tools: Bash(pk:*)` frontmatter auto-approves Bash but does not prevent Claude from choosing PowerShell. Explicit instruction is needed.

### 5. Update ship skill to specify Bash tool

**Files:** `internal/setup/skills/ship/SKILL.md`, `.claude/skills/ship/SKILL.md`

Add a shell requirement rule in the Rules section. The ship skill has 5+ "Run:" commands; a single rule is cleaner than modifying each line.

### 6. Update tests

**File:** `internal/setup/claude_test.go`

- Add test that `"shell":"bash"` appears in serialized plankit hook JSON
- Add test that user hooks don't get `"shell":"bash"` stamped on them
- Update `TestMergeHooks_existingUserHooks` to expect the first Bash matcher entry to be the user's (since plankit's guard matcher is now `"Bash|PowerShell"`, it won't collide with the user's `"Bash"` matcher)

**File:** `internal/setup/setup_test.go`

- No count changes (still 3 PreToolUse entries)

### 7. Recompute skill SHAs

After modifying embedded skills, run `pk setup` to regenerate local copies with correct `pk_sha256` values. Or manually sync the body changes and recompute.

## Files to modify

- `internal/setup/claude.go` -- Hook struct, buildHookConfig, guard matcher
- `internal/setup/skills/preserve/SKILL.md` -- embedded preserve skill
- `internal/setup/skills/ship/SKILL.md` -- embedded ship skill
- `.claude/skills/preserve/SKILL.md` -- local preserve skill
- `.claude/skills/ship/SKILL.md` -- local ship skill
- `internal/setup/claude_test.go` -- new tests, matcher update

## Not in scope

**Conventions skill:** Doesn't use pk commands. Its git commands are advisory only.

**PowerShell permission:** Not adding `PowerShell(pk:*)` to the auto-approved permissions. Only `Bash(pk:*)` is approved, which acts as a safety net: if Claude ignores the skill instruction and tries PowerShell for pk commands, the user gets prompted.

## Verification

1. `make test` -- all tests pass, including new shell-field tests
2. `make lint` -- no drift
3. Smoke test: run `pk setup` in a test project, verify `settings.json` hooks include `"shell":"bash"` and the `"Bash|PowerShell"` guard matcher, verify skill files include the Bash tool instruction
