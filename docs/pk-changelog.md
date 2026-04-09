# pk changelog

Generate CHANGELOG.md from conventional commits, commit, and tag.

## Usage

```bash
pk changelog                      # auto-detect version bump from commits
pk changelog --bump minor         # override: major, minor, or patch
pk changelog --dry-run            # preview without writing, committing, or tagging
pk changelog --push               # push commit and tag to origin after tagging
```

## How it works

1. Checks if the current branch is protected by `guard.protectedBranches` in `.pk.json`. If so, exits with an error: "switch to your development branch first."
2. Reads the latest version tag (git tags are the single version source)
2. Scans commits since that tag for conventional commit messages
3. Groups commits by type into changelog sections
4. Writes or updates CHANGELOG.md with the new version section
5. Updates version files if configured
6. Runs lifecycle hooks if configured
7. Commits CHANGELOG.md and all modified files
8. Tags the new version
9. If `--push` is set, pushes commit and tag to origin

## Version source

Git tags are the single version source. If no tags exist, `pk changelog` exits with a helpful message:

```
Error: no version tags found
  To start from scratch: git tag v0.0.0 && git push origin v0.0.0
  Or tag your current version and push it (e.g., git tag v1.2.3 && git push origin v1.2.3)
```

## Version bump

The bump is auto-detected from conventional commits:

- Any commit with `!` or `BREAKING CHANGE:` trailer → **major**
- Any `feat:` commit → **minor**
- Everything else → **patch**

Override with `--bump major|minor|patch`.

## .pk.json

Changelog configuration lives under the `changelog` key in `.pk.json`. All fields are optional.

```json
{
  "changelog": {
    "types": [
      {"type": "feat", "section": "Added"},
      {"type": "fix", "section": "Fixed"},
      {"type": "docs", "hidden": true}
    ],
    "versionFiles": [
      {"path": "package.json", "type": "json"},
      {"path": "package-lock.json", "type": "json"}
    ],
    "showScope": true,
    "hooks": {
      "postVersion": "node -e \"...propagate version...\"",
      "preCommit": "npm install --package-lock-only"
    }
  }
}
```

### types

Maps conventional commit types to changelog section headings. Each entry has:

- **type** — the conventional commit type (e.g., `feat`, `fix`, `docs`)
- **section** — the changelog heading (e.g., "Added", "Fixed")
- **hidden** — if `true`, commits of this type are excluded from the changelog

Types also control section ordering — sections appear in the order their first type is listed.

If `types` is omitted or empty, defaults are used:

| Type | Section |
|------|---------|
| `feat` | Added |
| `fix` | Fixed |
| `deprecate` | Deprecated |
| `revert` | Removed |
| `security` | Security |
| `refactor` | Changed |
| `perf` | Changed |
| `docs` | Documentation |
| `chore` | Maintenance |
| `test` | Maintenance |
| `build` | Maintenance |
| `ci` | Maintenance |
| `style` | Maintenance |
| `plan` | Plans (hidden) |

### versionFiles

Files containing a version string to update when a new version is released. Each entry has:

- **path** — path to the file relative to the project root
- **type** — file format, currently `"json"` only

For JSON files, `pk changelog` updates the root-level `version` field using proper JSON parsing (no regex). Formatting, key order, and indentation are preserved.

Version files are output-only — `pk changelog` writes to them but never reads from them. The git tag is always the version source.

### showScope

When `true`, the conventional commit scope is included in changelog entries as a bold prefix:

```markdown
- **flow:** resolve Object-in-String-Context pattern (dab3f6d)
- **BREAKING:** **api:** remove endpoint (abc1234)
```

Defaults to `false` — scope is parsed but omitted from the output.

### hooks

Lifecycle hooks that run as shell commands during the release process. The `VERSION` environment variable is set to the new version (without `v` prefix).

- **postVersion** — runs after version files are updated, before CHANGELOG.md is written. Use case: propagate the version to other config files.
- **preCommit** — runs after CHANGELOG.md is written, before `git add` and commit. Use case: regenerate lockfiles, format files.

If a hook fails, the release is aborted.

## Flags

- **--bump** — Override the version bump: `major`, `minor`, or `patch`. If omitted, the bump is auto-detected from conventional commits.
- **--dry-run** — Preview the changelog output without writing, committing, or tagging.
- **--push** — Push the release commit and tag to origin after tagging. Without this flag, the commit and tag remain local. Useful for single-branch workflows where `pk release` is not needed.

## Comparison links

`pk changelog` appends markdown reference-style links at the bottom of CHANGELOG.md, linking each version heading to its GitHub comparison page:

```markdown
## [v0.2.0] - 2026-04-05

### Added

- new feature (abc1234)

[v0.2.0]: https://github.com/owner/repo/compare/v0.1.0...v0.2.0
```

The repository URL is auto-detected from `git remote get-url origin`. Both SSH and HTTPS remote formats are supported.

## Conventional commits

`pk changelog` parses the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format:

```
type(scope)!: description

optional body

BREAKING CHANGE: explanation
```

Breaking changes are detected from both the `!` suffix and `BREAKING CHANGE:` / `BREAKING-CHANGE:` trailers in the commit body.

Non-conventional commits are silently skipped.
