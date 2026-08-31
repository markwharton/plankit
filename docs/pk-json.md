# .pk.json

Project-level configuration for pk, at the project root. Each top-level key maps to a pk subcommand. The file is hand-editable: `pk setup` writes the modes (`guard.mode`, `guard.push`, `preserve.mode`) and leaves every other key; `pk init`, `/pk-configure`, or you write the targets (`guard.branches`, `release.branch`, `changelog`); `pk teardown` does not remove it.

No file, or `{}`, means every default. An absent mode key takes its default; `"off"` is an explicit value, distinct from absence.

## Schema

```json
{
  "changelog": { ... },
  "guard": { ... },
  "preserve": { ... },
  "release": { ... }
}
```

## changelog

Read by `pk changelog`. All fields optional.

### changelog.types

Maps conventional commit types to changelog sections. Omitted, the 14 built-in types apply:

| Type | Section | Hidden |
|------|---------|--------|
| `feat` | Added | |
| `fix` | Fixed | |
| `deprecate` | Deprecated | |
| `revert` | Removed | |
| `security` | Security | |
| `refactor` | Changed | |
| `perf` | Changed | |
| `docs` | Documentation | |
| `chore` | Maintenance | |
| `test` | Maintenance | |
| `build` | Maintenance | |
| `ci` | Maintenance | |
| `style` | Maintenance | |
| `plan` | Plans | yes |

A `types` list replaces the defaults; commits of a type not listed are dropped. Each entry: `type`, `section`, and optional `hidden` (`true` excludes the type). Sections appear in the order their first type is listed.

### changelog.versionFiles

Files whose root-level `version` field `pk changelog` rewrites on release. Each entry: `path` (relative to the project root) and `type` (`"json"`, the only value; omitted means `"json"`). The rewrite parses the JSON and keeps formatting, key order and indentation. Non-JSON files are pinned from a hook with [`pk pin`](pk-pin.md).

### changelog.showScope

`true` prefixes entries with the commit scope in bold (`**api:** remove endpoint`). Default `false`.

### changelog.hooks

Shell commands run by `pk changelog`, with `$VERSION` set (see [Hook timeline](#hook-timeline)):

- **postVersion**: after version files are rewritten, before `CHANGELOG.md` is written.
- **preCommit**: after `CHANGELOG.md` is written, before `git add` and the commit. Tracked files the hook modifies are staged (`git add -u`); a new file is not.

A hook that fails aborts the run. Chain steps with `&&`; hooks run through a shell.

## guard

Read by `pk guard`. `pk setup` writes `mode` and `push`; `pk init`, `/pk-configure`, or you write `branches`.

### guard.mode

The branch policy for a git mutation on a protected branch: `block` (deny), `ask` (prompt), `off` (allow). Default `block`. Set with `pk setup --guard <mode>`.

### guard.push

The push policy for any `git push` on any branch: `block` (deny), `ask` (prompt), `off` (allow). Default `block`. It applies to the agent's pushes inside a Claude Code session; your terminal and pk's own publish flows (`pk release`, `pk preserve --push`, `pk setup --baseline --push`, `pk init --push`) are not hooked. Set with `pk setup --push-guard <mode>`.

### guard.branches

Branches on which the branch policy applies. On a listed branch `pk guard` blocks (or, under `ask`, prompts on) `commit`, `push`, `merge`, `rebase`, and `reset`. Omitted or empty, the branch policy does nothing; `guard.push` still applies. `pk init` sets it to the release branch.

```json
{
  "guard": {
    "branches": ["main", "production"],
    "mode": "block",
    "push": "block"
  }
}
```

## preserve

Read by `pk preserve`. `pk setup` writes `mode`.

### preserve.mode

What the `ExitPlanMode` hook does with an approved plan: `auto` commits it to `docs/plans/`; `manual` records a pointer and tells you to run `/preserve`; `off` does nothing. Default `manual`. Set with `pk setup --preserve <mode>`. An explicit `/preserve` commits in every mode.

## release

Read by `pk release`. All fields optional. `pk init` writes `branch`; nothing else in pk does.

### release.branch

The branch `pk release` fast-forwards from the current branch, tags, and pushes.

- **Set:** merge flow. `pk release` switches to the release branch, merges `--ff-only` from the source branch, tags, and pushes both branches.
- **Omitted:** trunk flow. `pk release` tags HEAD on the current branch and pushes; `pk changelog` and `pk release` refuse a branch that is not the default branch on origin.

### release.hooks

Shell commands run by `pk release`, with `$VERSION` and `$TAG` set:

- **preRelease**: after the merge (or on HEAD in trunk flow), before the tag exists. A failure aborts the release; nothing is pushed. Rehearsed by `pk release --dry-run`.
- **prePush**: after the tag is created, before the push. A failure removes the local tag; nothing is pushed. Not run by `--dry-run`.

Neither runs before a commit, so a file a release hook writes is left uncommitted.

```json
{
  "release": {
    "branch": "main",
    "hooks": {
      "preRelease": "go test -race ./...",
      "prePush": "sign-tag $TAG"
    }
  }
}
```

## Hook timeline

| Hook | Key | `$VERSION` | `$TAG` | Tag ref exists | Output is committed | Runs on `--dry-run` |
|------|-----|------------|--------|----------------|---------------------|---------------------|
| `postVersion` | `changelog.hooks` | yes | no | no | yes | no |
| `preCommit` | `changelog.hooks` | yes | no | no | yes | no |
| `preRelease` | `release.hooks` | yes | yes | no | no | yes |
| `prePush` | `release.hooks` | yes | yes | yes | no | no |

`$VERSION` has no leading `v`; `$TAG` has it. pk expands both before the shell runs the command, so one hook line works on macOS, Linux and Windows; `${VAR#pattern}` and other shell-specific forms are not expanded. See [Environment variables](environment-variables.md).

## Examples

Trunk flow, one branch: no `.pk.json`.

Merge flow, as `pk init` and `pk setup` write it:

```json
{
  "guard": {
    "branches": ["main"],
    "mode": "block",
    "push": "block"
  },
  "preserve": {
    "mode": "manual"
  },
  "release": {
    "branch": "main"
  }
}
```

Merge flow with version files and hooks, an npm project:

```json
{
  "changelog": {
    "versionFiles": [
      {"path": "package.json", "type": "json"},
      {"path": "package-lock.json", "type": "json"}
    ],
    "showScope": true,
    "hooks": {
      "postVersion": "npm version $VERSION --workspaces --no-git-tag-version",
      "preCommit": "npm install --package-lock-only && git add packages/*/package.json package-lock.json"
    }
  },
  "guard": {
    "branches": ["main"]
  },
  "release": {
    "branch": "main",
    "hooks": {
      "preRelease": "npm test"
    }
  }
}
```

An artifact that carries its own version: `preCommit` pins the version into a source file, so the commit and the tag cover it; `prePush` builds the artifact named for `$TAG`, which exists by then.

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "python3 scripts/package_skill.py --pin $VERSION"
    }
  },
  "release": {
    "branch": "main",
    "hooks": {
      "prePush": "python3 scripts/package_skill.py --all --version $TAG"
    }
  }
}
```
