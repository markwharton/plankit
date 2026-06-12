# Preserve hook statusMessage vs preserve mode — investigated, no change

## Context

`pk setup --preserve auto|manual` updates `.pk.json` but never touches the `"statusMessage": "Preserving plan..."` on the preserve hook entry in `.claude/settings.json`. The question was whether this is a miss from the modes refactor and whether the message should change.

## Findings

- The static settings.json is **by design**: the 2026-06-10 modes refactor (docs/plans/2026-06-10-001) made settings.json identical wiring for every project; `pk preserve` resolves auto/manual/off from `.pk.json` at runtime. So setup correctly leaves settings.json untouched when only the mode changes.
- The refactor plan (§F) did propose "generic status text" for the now-shared entry, and the implementation kept the auto-era literal at `internal/setup/claude.go:121` — technically a miss.
- `statusMessage` is cosmetic spinner text only. In manual mode the hook is sub-second; in off mode it's an instant no-op; only auto mode (commit, possibly push) shows it for any visible time — where "Preserving plan..." is accurate.
- No tests pin the literal, no docs quote it. A mode-dependent message is not feasible by design (users can edit `.pk.json` without re-running setup).

## Decision

User chose to keep `"Preserving plan..."` as-is. The slight inaccuracy in manual mode flashes sub-second and doesn't matter.

**No code change.**
