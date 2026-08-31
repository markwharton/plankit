# pk teardown

Remove what `pk setup` installed.

## Usage

```bash
pk teardown                       # print what would be removed
pk teardown --confirm             # remove it
pk teardown --project-dir /path   # start the git-root search there
```

## How it works

1. Reads `.claude/settings.json` for pk hooks and the `Bash(pk:*)` permission.
2. Finds managed skills, rules and `CLAUDE.md` by their hash markers.
3. Prints the grouped list.
4. With `--confirm`, removes them and any directory left empty (`.claude/skills/<name>/`, `.claude/skills/`, `.claude/rules/`, `.claude/`), and rewrites or removes `settings.json`.

## Flags

- **--confirm**: remove. Without it, nothing changes.
- **--project-dir `<dir>`**: where the search for the repository root starts.

## Limits

Removed: hooks whose command starts with `pk ` or is `.claude/install-pk.sh`; the `Bash(pk:*)` permission; skills, rules and `CLAUDE.md` whose hash still matches; `.claude/install-pk.sh` and `.claude/settings.json.bak`.

Kept: `.pk.json`; `docs/plans/`; hooks that are not pk's; a managed file whose content was modified (listed with a removal hint); a skill or rule without a marker. `pk setup` reinstalls; `pk setup --force` replaces modified managed files.
