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
3. Reads the guard mode, push-guard mode (shown only when guard is active), and preserve mode from `.pk.json`, applying defaults for any absent key (guard `block`, push `block`, preserve `manual`). The Modes section shows when plankit hooks are installed.
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

### Modes

The guard, push-guard, and preserve modes are read from `.pk.json` (`guard.mode`, `guard.push`, `preserve.mode`). Status shows the **effective** mode — an absent key resolves to its default (guard `block`, push `block`, preserve `manual`), so a configured project always shows a value. The `push:` line appears only when guard is active (`block` or `ask`); when guard is `off`, push-guard is moot and not shown.

See [.pk.json](pk-json.md#guard) for the keys and how `pk setup` writes them.

### Related commands

- `pk setup` — install plankit
- `pk teardown` — remove plankit (preview by default, `--confirm` to execute)
