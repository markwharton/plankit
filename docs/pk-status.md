# pk status

Report what pk has installed in a project, the effective modes, and whether the repository is ready for `pk changelog` and `pk release`.

## Usage

```bash
pk status                         # the report
pk status --brief                 # one line, ending ready or not-ready
pk status --project-dir /path     # start the git-root search there
```

## How it works

1. Warns when the directory is not a git repository.
2. Reads `.claude/settings.json` for pk hooks and the `Bash(pk:*)` permission.
3. Reads `guard.mode`, `guard.push` and `preserve.mode` from `.pk.json`; an absent key shows its default. The `push:` line appears only while the branch policy is `block` or `ask`; the push policy applies either way.
4. Reports each managed skill, rule and `CLAUDE.md` as pristine or modified by its hash, and whether `.claude/install-pk.sh` exists.
5. Reports the configured fields of `.pk.json`: changelog types, hooks, release branch, guard branches.
6. In a git repository with pk hooks installed, evaluates readiness and names the command that closes each gap:

```
Readiness:
  baseline tag     missing
    To anchor at v0.0.0: pk setup --baseline --push
    or: git tag v0.0.0 && git push origin v0.0.0
  working branch   on release branch main
    To start one: git switch -c develop && git push -u origin develop
```

With `release.branch` set: a baseline tag, a working branch distinct from the release branch, and both on origin; the ready line is `Readiness: ready for pk changelog / pk release (merge flow into main)`. Without it: a baseline tag and the current branch on origin; the ready line is `Readiness: ready for pk changelog / pk release (trunk flow; no release.branch in .pk.json)`.

The checks read `refs/remotes/origin/*` and never the network; they reflect the last fetch. `pk release` runs its own pre-flight against origin.

## Flags

- **--brief**: one line, with `ready` or `not-ready` when readiness was evaluated.
- **--project-dir `<dir>`**: where the search for the repository root starts.

## Exit code

`0` when pk is configured: a pk hook, the `Bash(pk:*)` permission, a managed `CLAUDE.md`, skill or rule, or `.claude/install-pk.sh` is present. `1` otherwise, or on error. `if pk status >/dev/null 2>&1; then ...` works in scripts.
