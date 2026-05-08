# Architecture

pk's functionality falls into three layers with different levels of environment coupling.

## Three layers

### Git workflow (zero coupling)

Commands that operate on git directly: `pk changelog`, `pk release`, `pk guard`, `pk pin`, `pk version`. These work with any AI coding tool or none at all. They read `.pk.json` for configuration and produce git artifacts (tags, commits, changelogs).

### AI governance (protocol-specific)

Rules and skills that shape AI behavior. Rules are markdown files that AI coding tools load as context. Skills are workflow scripts that tools execute on user request. The content is universal, but the file paths and frontmatter format are environment-specific: Claude Code reads `.claude/rules/` and `.claude/skills/`, other tools use different locations.

### Environment wiring (deep coupling)

Hooks, settings, and bootstrap that integrate pk into a specific AI coding tool. Claude Code's hook protocol (PreToolUse, PostToolUse, SessionStart) enables enforcement: `pk guard` can block a git mutation before it happens, `pk protect` can block an edit to a preserved plan. This layer is inherently Claude Code-specific because no other tool currently offers pre-tool interception.

## File structure

The `internal/setup/` package is organized by concern to make these boundaries visible:

```
internal/setup/
├── baseline.go           Git tag baseline (universal)
├── claude.go             Claude Code provider (hooks, settings, bootstrap)
├── managed.go            SHA-tracked file management (universal)
├── pin.go                Version pinning (universal)
├── setup.go              Config, Run() orchestrator, OrderedObject
├── setup_test.go         Tests (unchanged, same exported API)
├── rules/                Rule content (universal, embedded)
├── skills/               Skill content (Claude Code format, embedded)
└── template/             CLAUDE.md template, install-pk.sh
```

**Universal files** (`baseline.go`, `managed.go`, `pin.go`) contain logic reusable across any provider: version pinning, SHA-tracked file management, git tag operations.

**Provider file** (`claude.go`) contains everything specific to Claude Code: hook types, settings merge, permission management, install script generation.

**Orchestrator** (`setup.go`) holds `Config`, `Run()`, and `OrderedObject`. `Run()` calls into both universal and provider-specific code.

## Adding a new provider

When a second AI coding tool needs support, the path is:

1. Copy `claude.go` to `<provider>.go`.
2. Adapt hook types, settings paths, and file formats to the new tool's conventions.
3. Wire the new provider's steps into `Run()` alongside the existing Claude Code steps, gated on a configuration flag or auto-detection.

The universal files (`managed.go`, `pin.go`, `baseline.go`) and the orchestrator (`setup.go`) do not change.

## AI coding landscape

Not all environments offer the same capabilities. pk adapts its governance model to what each tool provides.

| Capability | Claude Code | Cursor | Windsurf | Cline | Bob IDE |
|------------|-------------|--------|----------|-------|---------|
| Rules (context files) | Yes | Yes | Yes | Yes | Yes |
| Skills (workflows) | Yes | No | No | No | Yes |
| Pre-tool hooks (enforcement) | Yes | No | No | No | No |
| Post-tool hooks (reactions) | Yes | No | No | No | No |
| Plan mode | Yes | No | No | No | No |

### Enforcement vs. advisory

Claude Code is the only environment with pre-tool interception hooks. This means:

- **Claude Code**: full enforcement. `pk guard` blocks git mutations. `pk protect` blocks edits to preserved plans. `pk preserve` reacts to plan approval.
- **Other environments**: advisory governance. Rules carry the behavioral guidance (which covers the majority of value), but there is no backstop for the cases where the AI ignores a rule. Git workflow commands (`pk changelog`, `pk release`) work identically regardless of environment.

Rules carry roughly 90% of the behavioral value. The model follows them in the vast majority of cases. Hooks are a backstop for the remaining cases where enforcement matters: protecting immutable plans, guarding release branches, and preserving approved work.
