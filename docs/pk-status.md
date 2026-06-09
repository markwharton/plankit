# pk status

Report the plankit configuration state of a project.

## Usage

```bash
pk status                         # report what's configured
pk status --brief                 # one-line summary
pk status --project-dir /path     # specify project directory
```

## How it works

1. Detects whether the directory is a git repository. Warns if not — pk requires git for most commands.
2. Reads `.claude/settings.json` and identifies plankit hooks and the `Bash(pk:*)` permission.
3. Infers guard mode (block, ask, or off), push-guard mode (block, ask, or off — shown only when guard is active), and preserve mode (auto, manual, or off) from hook commands.
4. Scans `.claude/skills/` and `.claude/rules/` for files with `pk_sha256` markers and checks if they match (pristine) or have been modified.
5. Checks `CLAUDE.md` for a plankit SHA marker.
6. Checks for `.claude/install-pk.sh`.
7. Parses `.pk.json` and reports configured fields: changelog types count, configured hooks, release branch, guard branches.

## Flags

- **--brief** — One-line summary. Useful for scripting: `if pk status --brief >/dev/null; then ...`
- **--project-dir** — Starting directory for git root resolution (default: current directory). Resolves up to the nearest `.git` ancestor.

## Exit code

- **0** — plankit is configured in this project.
- **1** — plankit is not configured, or an error occurred.

Useful for scripts: `if pk status >/dev/null 2>&1; then ...`

## Details

### What counts as configured

plankit is reported as configured if any of the following are present:
- A plankit hook in `.claude/settings.json`
- The `Bash(pk:*)` permission
- A `CLAUDE.md` with a plankit SHA marker
- Managed skills in `.claude/skills/`
- Managed rules in `.claude/rules/`
- `.claude/install-pk.sh`

### Mode inference

Guard, push-guard, and preserve modes are not stored explicitly — they're inferred from the hook commands in `settings.json`:

| Command | Mode |
|---------|------|
| `pk guard` | guard: block |
| `pk guard --ask` | guard: ask |
| (absent, other pk hooks present) | guard: off |
| `pk guard --push-guard block` | push: block |
| `pk guard --push-guard ask` | push: ask |
| (guard active, no `--push-guard`) | push: off |
| `pk preserve` | preserve: auto |
| `pk preserve --notify` | preserve: manual |
| (absent, other pk hooks present) | preserve: off |

Push-guard rides on the guard command, so the `push:` line appears only when guard is active (block or ask); when guard is off, push-guard is moot and not shown.

### Related commands

- `pk setup` — install plankit
- `pk teardown` — remove plankit (preview by default, `--confirm` to execute)
