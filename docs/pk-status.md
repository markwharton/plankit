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
3. Infers guard mode (block or ask) and preserve mode (auto or manual) from hook commands.
4. Scans `.claude/skills/` and `.claude/rules/` for files with `pk_sha256` markers and checks if they match (pristine) or have been modified.
5. Checks `CLAUDE.md` for a plankit SHA marker.
6. Checks for `.claude/install-pk.sh`.
7. Parses `.pk.json` and reports configured fields: changelog types count, configured hooks, release branch, guard branches.

## Flags

- **--brief** — One-line summary. Useful for scripting: `if pk status --brief >/dev/null; then ...`
- **--project-dir** — Project directory (default: current directory).

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

Guard and preserve modes are not stored explicitly — they're inferred from the hook commands in `settings.json`:

| Command | Mode |
|---------|------|
| `pk guard` | guard: block |
| `pk guard --ask` | guard: ask |
| `pk preserve` | preserve: auto |
| `pk preserve --notify` | preserve: manual |

### Related commands

- `pk setup` — install plankit
- `pk teardown` — remove plankit (preview by default, `--confirm` to execute)
