# Add "off" mode for guard and preserve

## Context

plankit is useful beyond plan preservation — users want local branch protection (`pk guard`) and release management (`pk changelog`, `pk release`, `/ship`) independently. Currently `--guard` only accepts `block`/`ask` and `--preserve` only accepts `manual`/`auto`. Adding `off` to both lets users pick exactly which features they want. Protect hooks (`pk protect` for `docs/plans/` immutability) stay unconditional because already-committed plans should remain protected regardless of mode.

## Changes

### 1. Accept "off" in flag validation

**File:** `cmd/pk/main.go`

Add `"off"` to both validation switches and update flag help text:
- `--preserve` help: `"Plan preservation mode: manual, auto, or off"`
- `--guard` help: `"Guard mode: block, ask, or off"`
- Validation: `case "auto", "manual", "off":` and `case "block", "ask", "off":`

### 2. Make protect hooks unconditional, guard conditional

**File:** `internal/setup/claude.go`

Restructure `buildHookConfig()` so protect hooks (Edit/Write) are always present and guard hooks are conditional on mode:

```
PreToolUse always:  Edit → pk protect, Write → pk protect
PreToolUse if guard != "off":  Bash|PowerShell → pk guard
```

Guard entry goes first (before protect) when present, matching the current ordering.

### 3. Infer "off" from hook absence

**File:** `internal/setup/claude.go`

Update `InferModesFromCommands()`: when any plankit hook is found (via `IsPlankitHook`) but no guard command is present, return `"off"` for guard. Same for preserve. When no plankit hooks exist at all, return `("", "")` as before (fresh project, use defaults).

This preserves the re-run contract: `pk setup` after `pk setup --guard off` keeps guard off without the user re-passing the flag.

### 4. Update tests

**File:** `internal/setup/claude_test.go`

- Test `buildHookConfig("off", "off")`: 2 PreToolUse entries (protect only), no PostToolUse
- Test `buildHookConfig("manual", "off")`: 2 PreToolUse (no guard), 1 PostToolUse
- Test `buildHookConfig("off", "block")`: 3 PreToolUse (guard + protect), no PostToolUse
- Test `InferModesFromCommands` round-trip for "off" modes
- Test `InferModesFromCommands` with only protect hooks → guard=off, preserve=off

### 5. Update documentation

**Files:**
- `docs/pk-setup.md` — Add "off" to guard/preserve mode lists, add usage examples
- `docs/pk-status.md` — Add "off" to mode inference table
- `docs/error-reference.md` — Update valid mode values in error messages
- `README.md` — Add `--guard off` / `--preserve off` examples
- `docs/adoption.md` — Mention off mode for release-only users

## Files to modify

- `cmd/pk/main.go` — flag help text, validation
- `internal/setup/claude.go` — `buildHookConfig`, `InferModesFromCommands`
- `internal/setup/claude_test.go` — new tests
- `docs/pk-setup.md`, `docs/pk-status.md`, `docs/error-reference.md`, `README.md`, `docs/adoption.md` — doc updates

## Verification

1. `make test` — all tests pass
2. `make lint` — no drift
3. Smoke: `pk setup --guard off --preserve off` in a temp project, verify settings.json has only protect hooks and session start, no guard or preserve hooks
4. Smoke: re-run `pk setup` without flags in same project, verify modes stay "off" (inferred correctly)
5. Smoke: `pk status` shows "guard: off" and "preserve: off"
