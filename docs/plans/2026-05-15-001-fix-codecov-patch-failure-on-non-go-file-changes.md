# Fix codecov/patch failure on non-Go file changes

## Context

The `codecov/patch` status check fails on commit `933178c` (`chore: release v0.19.7`) with "0.00% of diff hit (target 81.69%)". That commit changed only `.claude/install-pk.sh` (version bump) and `CHANGELOG.md` — neither is Go source, so neither appears in the Go coverage profile. Codecov sees uncovered diff lines and reports 0%. This will recur on every release commit.

There is no `codecov.yml` in the repo. Codecov uses all defaults.

## Plan

Create `codecov.yml` at the repo root with `ignore` patterns for all non-Go files. This tells Codecov to exclude those paths from both project and patch coverage analysis, which is correct since they can never produce coverage data.

### File: `codecov.yml` (new)

```yaml
ignore:
  - ".claude/**"
  - ".github/**"
  - "docs/**"
  - "**/*.md"
  - "**/*.sh"
  - "**/*.json"
  - "LICENSE"
  - "Makefile"
```

Why each entry:
- **`.claude/**`, `.github/**`, `docs/**`** — entire directories with no Go source
- **`**/*.md`** — covers root markdown (`CHANGELOG.md`, `README.md`, etc.) and embedded templates in `internal/setup/rules/*.md`, `internal/setup/skills/*/SKILL.md`, `internal/setup/template/CLAUDE.md`
- **`**/*.sh`** — covers `internal/setup/template/install-pk.sh` (embedded asset inside Go source tree; `.claude/install-pk.sh` is already handled by `.claude/**`)
- **`**/*.json`** — covers `.pk.json` at root, `docs/protect-main.json`
- **`LICENSE`, `Makefile`** — root-level non-Go files

### Why not threshold or informational?

Setting `threshold: 0%` or `informational: true` on the patch check would suppress legitimate coverage failures on Go code changes. The `ignore` approach preserves meaningful coverage checking.

## Verification

1. `make test` and `make lint` pass (no Go changes)
2. Push to develop — CI runs, codecov processes the new config
3. The codecov/patch check should pass since the diff (adding `codecov.yml`) is itself a non-Go file
