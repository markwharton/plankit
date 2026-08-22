# .pk.json

Project-level configuration for pk. Each top-level key maps to a pk subcommand.

## Location

`.pk.json` lives in the project root (the directory where you run `pk setup`). It is user-owned and hand-editable. `pk setup` writes the **behavior modes** (`guard.mode`, `guard.push`, `preserve.mode`) into it but never touches your other keys; `pk teardown` does not remove it. The **targets** (`guard.branches`, `release.branch`, changelog config) are filled in by [`pk init`](pk-init.md) at project creation, by the `/pk-configure` skill, or by hand.

If `.pk.json` does not exist, all commands use their defaults. An empty file (`{}`) is equivalent to no file. Any mode key that is absent falls back to its default; `"off"` is an explicit value, distinct from absence.

## Schema

Four top-level keys, each optional:

```json
{
  "changelog": { ... },
  "guard": { ... },
  "preserve": { ... },
  "release": { ... }
}
```

See [Workflow examples](#workflow-examples) for complete configurations.

## changelog

Configuration for `pk changelog`. All fields are optional.

### changelog.types

Maps conventional commit types to changelog section headings. If omitted, all 14 built-in types are used:

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

When you provide `types`, it **replaces** the defaults entirely. Only the types you list will appear in the changelog; commits with unlisted types are silently dropped. If you only want `feat` and `fix`, that is all you get.

Each entry has:

- **type** — the conventional commit type (e.g., `feat`, `fix`, `docs`).
- **section** — the changelog heading (e.g., "Added", "Fixed").
- **hidden** — if `true`, commits of this type are excluded from the changelog.

Types also control section ordering: sections appear in the order their first type is listed.

### changelog.versionFiles

Files containing a version string to update when a new version is released. Each entry has:

- **path** — path to the file relative to the project root.
- **type** — file format. `"json"` is the only accepted value; an omitted `type` is treated as `"json"`.

For JSON files, `pk changelog` updates the root-level `version` field using proper JSON parsing. Formatting, key order, and indentation are preserved.

`pk changelog` writes versions into these files but never reads versions out of them. Git tags are always the version source of truth.

### changelog.showScope

When `true`, the conventional commit scope is included in changelog entries as a bold prefix (e.g., `**api:** remove endpoint`). Defaults to `false`.

### changelog.hooks

Lifecycle hooks that run as shell commands during the changelog process. The `VERSION` environment variable is set to the new version without the `v` prefix (e.g., `0.8.1`).

- **postVersion** — runs after version files are updated, before CHANGELOG.md is written. Use case: propagate the version to config files not covered by `versionFiles`.
- **preCommit** — runs after CHANGELOG.md is written, before `git add` and commit. Use case: regenerate lockfiles, pin versions in source files via `pk pin`. Chain steps with `&&` (hooks run through a shell); files the hook modifies are staged automatically.

If a hook fails, the changelog process is aborted.

pk pre-expands `$VERSION` before passing the command to the shell, so hooks work on all platforms (macOS, Linux, Windows). Bash-specific parameter expansion like `${VAR#pattern}` is not supported cross-platform.

## guard

Configuration for `pk guard`. All fields are optional. `pk setup` writes `mode` and `push`; `pk init`, `/pk-configure`, or you set `branches`.

### guard.mode

How the branch policy acts on a git mutation on a protected branch: `block` (deny), `ask` (prompt), or `off` (do nothing). Defaults to `block`. Set it with `pk setup --guard <mode>`.

### guard.push

The push policy for any `git push`, on any branch: `block` (deny), `ask` (prompt), or `off` (allow). Defaults to `block`. This blocks the *agent's* direct pushes within a Claude Code session. Your own terminal pushes and pk's publish flows (`pk release`, `pk preserve --push`) are unaffected. Set it with `pk setup --push-guard <mode>`.

### guard.branches

Array of branch names where git mutations are blocked (subject to `guard.mode`). When the current branch matches any entry, `pk guard` blocks git mutations like `commit`, `push`, `merge`, and `rebase`. In ask mode it prompts instead.

Read-only git commands (`status`, `log`, `diff`, `branch`, `fetch`) are always allowed.

If omitted or empty, the branch policy is a no-op (no branches to protect); `guard.push` still applies.

[`pk init`](pk-init.md) sets this to the release branch when it initializes a project.

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

Configuration for `pk preserve`. `pk setup` writes `mode`.

### preserve.mode

How the automatic plan-preservation hook behaves when you exit plan mode. `auto` commits the plan to `docs/plans/`, `manual` notifies you to run `/preserve`, and `off` does nothing. Defaults to `manual`. Set it with `pk setup --preserve <mode>`. An explicit `/preserve` always commits, regardless of this mode.

```json
{
  "preserve": {
    "mode": "manual"
  }
}
```

## release

Configuration for `pk release`. All fields are optional. [`pk init`](pk-init.md) writes `branch`; nothing else in pk does.

### release.branch

The branch that `pk release` merges to and pushes from. The current branch is the implicit source.

- **Set:** merge flow. `pk release` switches to the release branch, merges from the source branch (fast-forward only), creates the tag, and pushes both branches.
- **Omitted:** trunk flow. `pk release` tags HEAD on the current branch and pushes.

```json
{
  "release": {
    "branch": "main"
  }
}
```

### release.hooks

Lifecycle hooks for the release process. Both receive `$VERSION` (the version without the leading `v`, e.g. `0.26.1`) and `$TAG` (with it, e.g. `v0.26.1`) as pre-expanded, cross-platform environment variables. A hook need not re-derive the version from a pinned file. The two differ by *when* they run relative to tag creation:

- **preRelease** — runs after merge (or on HEAD in trunk flow) but before the tag is created. If it fails, the release is aborted and nothing is pushed. Because it runs before tagging, it is rehearsed by `pk release --dry-run`, and a hook that commits produces a commit the tag then covers. **The release tag does not exist yet when preRelease runs.** A hook needing the tag ref wants `prePush` instead: signing, or artifact builds keyed on the tag. Use case: run tests one final time before publishing.
- **prePush** — runs after the tag is created, before the push, so the tag ref exists. If it fails, the release is aborted, the local tag is removed, and nothing is pushed. It does **not** run under `--dry-run` (the dry-run returns before tagging). Use case: sign the tag, or build an artifact named for it.

```json
{
  "release": {
    "hooks": {
      "preRelease": "go test -race ./...",
      "prePush": "sign-tag $TAG"
    }
  }
}
```

## Hook timeline

The four hooks live under two keys, and each command documents only its own pair. Choosing between them means comparing all four, so they are listed here in firing order.

| Hook | Key | `$VERSION` | `$TAG` | Tag ref exists | Output is committed | Runs on `--dry-run` |
|------|-----|------------|--------|----------------|---------------------|---------------------|
| `postVersion` | `changelog.hooks` | yes | no | no | yes | no |
| `preCommit` | `changelog.hooks` | yes | no | no | **yes** | no |
| `preRelease` | `release.hooks` | yes | yes | **no** | no | yes |
| `prePush` | `release.hooks` | yes | yes | **yes** | no | **no** |

The last three columns decide most choices:

- **Committed output.** Only the changelog hooks run before a commit, so only their file edits reach git. A release hook that writes a file leaves the tree dirty afterwards.
- **Tag ref.** Only `prePush` runs after the tag exists. Signing the tag, or naming a build after it, belongs there.
- **Dry-run.** `pk release --dry-run` rehearses `preRelease` but returns before `prePush`. A step that should not run in a rehearsal belongs in `prePush`.

`$TAG` carries the `v` prefix and `$VERSION` does not. Release hooks receive both. Changelog hooks receive `$VERSION` only, because the tag does not exist yet.

## Workflow examples

### Trunk flow (single branch, no guard)

No `.pk.json` needed. `pk changelog` and `pk release` work on the current branch.

### Merge flow (develop + main)

Uses all 14 default types (no `types` key needed). `mode`/`push`/`preserve.mode` are what `pk setup` writes by default:

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

### Merge flow with hooks and version files (npm project)

Uses default types (omitted), adds version file syncing, scoped changelog entries, and lifecycle hooks:

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

### An artifact that carries its own version

Some deliverables leave the repository and lose their filename. A zipped Claude skill, a plugin bundle, or a downloadable archive is unpacked by the recipient, and the version in the filename is gone. Answering "which version is this?" then needs the version *inside* the artifact.

That takes both keys, in order. The changelog hook writes the version into a source file, so the commit and the tag cover it. The release hook builds the artifact afterwards, named for `$TAG`, from a tree that already holds the pinned version:

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

Two properties make this work, and both come from the timeline above. `preCommit` output is committed, so the pinned version is in the tag. `prePush` runs after tagging, so the build is named for a tag that exists.

Pass `$TAG` to the build rather than deriving a version from `git describe`. pk already knows the version it is releasing, and reconstructing it needs a clean tree and a tag that may not exist yet.
