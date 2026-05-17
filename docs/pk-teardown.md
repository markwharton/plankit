# pk teardown

Remove plankit hooks, skills, and rules from a project.

## Usage

```bash
pk teardown                       # preview what would be removed
pk teardown --confirm             # remove plankit artifacts
pk teardown --project-dir /path   # specify project directory
```

## How it works

1. Reads `.claude/settings.json` and identifies plankit hooks and permissions.
2. Scans `.claude/skills/` and `.claude/rules/` for files with `pk_sha256` markers.
3. Checks `CLAUDE.md` for a plankit SHA marker.
4. Prints a grouped preview of what will be removed.
5. If `--confirm` is passed, removes plankit artifacts, cleans up empty directories, and edits or removes `settings.json`.

## Flags

- **--confirm** — Execute the teardown. Without this flag, only a preview is shown.
- **--project-dir** — Starting directory for git root resolution (default: current directory). Resolves up to the nearest `.git` ancestor.

## Details

### What gets removed

- **Hooks** — All hooks where the command starts with `pk ` or is `.claude/install-pk.sh`.
- **Permissions** — `Bash(pk:*)` from `permissions.allow`.
- **Skills and rules** — Any file with a `pk_sha256` marker whose SHA still matches (pristine).
- **CLAUDE.md** — Only if it has a plankit SHA marker and hasn't been modified.
- **Bootstrap** — `.claude/install-pk.sh` and `.claude/settings.json.bak`.
- **Empty directories** — Skill subdirectories, `.claude/skills/`, `.claude/rules/`, and `.claude/` itself if empty.

### What stays

- **`.pk.json`** — User configuration, not installed by setup.
- **`docs/plans/`** — Preserved plans belong to the user.
- **User hooks** — Only plankit hooks are removed; user hooks on the same matcher are preserved.
- **Modified managed files** — Files with a `pk_sha256` marker whose content was changed are skipped with a message and a manual removal hint.
- **User-created skills and rules** — Files without a `pk_sha256` marker are invisible to teardown.

### Restoring after teardown

Run `pk setup` to reinstall. If modified managed files remain from a previous setup, use `pk setup --force` to overwrite them with fresh copies.
