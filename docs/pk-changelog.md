# pk changelog

Generate CHANGELOG.md from conventional commits and commit the result. Leaves the tag to `pk release`.

## Usage

```bash
pk changelog                                # auto-detect version bump from commits
pk changelog --bump minor                   # override: major, minor, or patch
pk changelog --dry-run                      # preview without writing or committing
pk changelog --undo                         # unwind an unpushed release commit
pk changelog --exclude abc1234,def5678      # drop commits from the section by short SHA
```

## How it works

1. Checks if the current branch is protected by `guard.branches` in `.pk.json`. If so, exits with an error: "switch to your development branch first."
2. Verifies the working tree is clean (skipped in `--dry-run` mode). Exits with an error if there are uncommitted changes.
3. Reads the latest version tag (git tags are the single version source)
4. Scans commits since that tag for conventional commit messages
5. Groups commits by type into changelog sections
6. Writes or updates CHANGELOG.md with the new version section
7. Updates version files if configured
8. Runs lifecycle hooks if configured
9. Commits CHANGELOG.md and all modified files, adding a `Release-Tag: vX.Y.Z` trailer to the commit body via `git commit --trailer`

No git tag is created by `pk changelog`. The tag is the responsibility of `pk release`, which reads the trailer from HEAD and creates the tag just before pushing.

## Flags

- **--bump** — Override the version bump: `major`, `minor`, or `patch`. If omitted, the bump is auto-detected from conventional commits.
- **--dry-run** — Preview the changelog output without writing or committing.
- **--undo** — Unwind the most recent `pk changelog` commit. Refuses unless HEAD carries a `Release-Tag:` trailer, the working tree is clean, and HEAD has not been pushed (or the branch has no upstream). On success, HEAD is reset one commit back via `git reset --hard`, which restores CHANGELOG.md and version files to their prior state.
- **--exclude** — Comma-separated list of commit SHAs to drop from the generated section. Each SHA must match exactly as it appears in `CHANGELOG.md` parentheses (the abbreviated short hash). The filter runs before version-bump detection, so excluding all `feat:` commits falls back to a patch bump, and excluding a breaking change removes its contribution to the bump too. Unmatched exclude values emit a warning but don't fail the release.

## Requirements

- **git 2.32 or newer** for `git commit --trailer` (June 2021).

## Configuration

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

`pk changelog` writes versions into these files but never reads versions out of them — the git tag is always the source of truth.

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

## Details

### Version source

Git tags are the single version source. If no tags exist, `pk changelog` exits with a helpful message:

```
Error: no version tags found
  To anchor at v0.0.0: pk setup --baseline --push
  Or tag a specific version manually (e.g., git tag v1.2.3 && git push origin v1.2.3)
```

If origin has tags but they aren't present locally (common in shallow-clone sandboxes), the error instead points at `git fetch --tags`. The bootstrap script runs this automatically on session start, so the case normally only shows up outside a Claude Code session.

See [pk setup — baseline tag for pk changelog](pk-setup.md#baseline-tag-for-pk-changelog) for the three scenarios.

### Single tag, many files

The tag-as-source rule shines in monorepos with a unified-version policy — every package releases at the same version, governed by one tag. The `preCommit` hook fans the tag-derived version out to every file that needs it. For an npm workspaces monorepo:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "npm version $VERSION --workspaces --no-git-tag-version && git add packages/*/package.json package-lock.json"
    }
  }
}
```

`$VERSION` is set to the computed next version without the `v` prefix (e.g., `0.11.0`), ready for tools like `npm version` that expect a plain semver string. Every package.json gets the same bump, even unchanged ones, the accepted trade-off of unified versioning. pk pre-expands `$VERSION` before passing the command to the shell, so the same hook works on all platforms (macOS, Linux, Windows). Bash-specific parameter expansion like `${VAR#pattern}` is not supported cross-platform.

### Conventional commits

`pk changelog` parses the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format:

```
type(scope)!: description

optional body

BREAKING CHANGE: explanation
```

Breaking changes are detected from both the `!` suffix and `BREAKING CHANGE:` / `BREAKING-CHANGE:` trailers in the commit body.

Non-conventional commits are silently skipped.

### Version bump

The bump is auto-detected from conventional commits:

- Any commit with `!` or `BREAKING CHANGE:` trailer → **major**
- Any `feat:` commit → **minor**
- Everything else → **patch**

Override with `--bump major|minor|patch`.

### Excluding commits from a release

Sometimes a commit that lives in git history shouldn't appear in the release notes — usually because it was added and later removed within the same release window, so the net effect is zero and mentioning it would confuse a reader. `--exclude` is the tool for that.

The intended workflow:

1. Run `pk changelog` normally. It generates the section and commits it.
2. Read `CHANGELOG.md`. If there's an entry you don't want, copy the short SHA from inside its parentheses — for example, `abc1234` from `- add feature (abc1234)`.
3. Run `pk changelog --undo` to unwind the release commit.
4. Run `pk changelog --exclude abc1234` (or a comma-separated list for multiple) to regenerate without that commit.

The matcher is exact string equality against the short hash as it appears in `CHANGELOG.md`. No prefix matching, no full-SHA input, no clever resolution — copy what you see, paste it back. The filter also runs *before* version-bump detection, so excluded commits don't influence the version number.

`CHANGELOG.md` remains the long-lived record of release notes; `--exclude` is just a tool for producing it with surgical omissions.

### Release-Tag trailer

Every `pk changelog` commit carries a git trailer in its body:

```
chore: release v0.6.2

Release-Tag: v0.6.2
```

The trailer is a git-native mechanism for structured commit metadata (see also `Signed-off-by:`, `Co-authored-by:`). It's written via `git commit --trailer` and read via `git log --format=%(trailers:key=Release-Tag,valueonly)`. Two pk commands consume it:

- **`pk release`** reads the trailer to know which version to tag. Without the trailer, `pk release` refuses with "no Release-Tag trailer on HEAD — run 'pk changelog' first."
- **`pk changelog --undo`** checks for the trailer before touching history, so it only unwinds pk-created commits.

The trailer value is validated as strict semver: the value must parse via plankit's semver parser and round-trip back to the same string. Anything else (typos, non-semver strings, extra characters) is rejected.

### Comparison links

`pk changelog` appends markdown reference-style links at the bottom of CHANGELOG.md, linking each version heading to its GitHub comparison page:

```markdown
## [v0.2.0] - 2026-04-05

### Added

- new feature (abc1234)

[v0.2.0]: https://github.com/owner/repo/compare/v0.1.0...v0.2.0
```

The repository URL is auto-detected from `git remote get-url origin`. Both SSH and HTTPS remote formats are supported.
