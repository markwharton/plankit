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
8. Evaluates release readiness when plankit hooks are installed in a git repository, and renders a Readiness section (see below).

## Flags

- **--brief** — One-line summary, including `ready`/`not-ready` when readiness is evaluated. Useful for scripting: `if pk status --brief >/dev/null; then ...`
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

### Readiness

A configured project can still be unready for `pk changelog` / `pk release`: no baseline tag, a working branch that only exists locally, a release branch that was never pushed. Status evaluates these facts and reports each gap with the exact command that closes it:

```
Readiness:
  baseline tag     missing
    To anchor at v0.0.0: pk setup --baseline --push
    or: git tag v0.0.0 && git push origin v0.0.0
  working branch   on release branch main
    To start one: git switch -c develop && git push -u origin develop
```

When every check passes, the section collapses to one line: `Readiness: ready for pk changelog / pk release`.

The checks are keyed to what `.pk.json` declares. With `release.branch` set (merge flow): baseline tag, a working branch distinct from the release branch, and both branches on origin. Without it (trunk flow): baseline tag and the current branch on origin. Status never nags about layers the configuration hasn't opted into.

The checks are **offline** — they read local refs (`refs/remotes/origin/*`), never the network — so they reflect state as of the last fetch. `pk release` keeps its own authoritative network pre-flight.

### Related commands

- `pk setup` — install plankit
- `pk teardown` — remove plankit (preview by default, `--confirm` to execute)
