# pk changelog

Write the next `CHANGELOG.md` section from the conventional commits since the latest tag, and commit it with a `Release-Tag` trailer. The tag is `pk release`'s.

## Usage

```bash
pk changelog                                # bump from the commits: major, minor or patch
pk changelog --bump minor                   # override the bump
pk changelog --dry-run                      # print the section; write nothing
pk changelog --undo                         # unwind an unpushed release commit
pk changelog --exclude abc1234,def5678      # leave commits out of the section
```

## How it works

1. Refuses on a branch listed in `guard.branches`: `switch to your development branch first`.
2. Refuses on a dirty working tree (not checked under `--dry-run`).
3. In trunk flow (no `release.branch`), refuses a branch that is not origin's default branch (`git ls-remote --symref origin HEAD`); a trunk-flow release tags and pushes the branch it runs on. Skipped when origin advertises no HEAD.
4. Refuses a branch that is not on origin; otherwise `pk release` would fail after the release commit exists.
5. Reads the latest semver tag: the current version.
6. Scans the commits since it. `!` or a `BREAKING CHANGE:` / `BREAKING-CHANGE:` trailer means major; `feat` means minor; anything else means patch. Commits that are not conventional are skipped.
7. Groups the commits by type into sections, in the order of `changelog.types`.
8. Inserts the section into `CHANGELOG.md` above the existing content, and appends a `[vX.Y.Z]: .../compare/vPREV...vX.Y.Z` link built from `git remote get-url origin` (SSH or HTTPS).
9. Rewrites the `version` field of each `changelog.versionFiles` entry.
10. Runs `postVersion` (before the changelog is written) and `preCommit` (after), with `$VERSION` set. A failing hook aborts the run.
11. Commits `CHANGELOG.md`, the version files and every tracked file a hook changed (`git add -u`), with `Release-Tag: vX.Y.Z` as a trailer (`git commit --trailer`).

## Flags

- **--bump `<major|minor|patch>`**: override the bump computed from the commits.
- **--dry-run**: print the section and the version; write nothing.
- **--undo**: `git reset --hard HEAD~1`. Refuses unless HEAD carries a `Release-Tag:` trailer, the tree is clean, and HEAD is unpushed or the branch has no upstream. Version files and pinned files revert with the changelog.
- **--exclude `<sha,...>`**: drop commits from the section. Each value must equal the short hash printed in the section's parentheses. The filter runs before the bump is computed, so excluding every `feat` gives a patch bump. A value that matches nothing warns and does not fail.

To leave a commit out: run `pk changelog`, read the section, copy the short hash, `pk changelog --undo`, then `pk changelog --exclude <sha>`.

## Configuration

`changelog.types`, `changelog.versionFiles`, `changelog.showScope` and `changelog.hooks`: see [.pk.json](pk-json.md#changelog).

Where the version goes after the tag:

| Where the version lives | Mechanism |
|---|---|
| A root `version` field in a JSON file (`package.json`, `package-lock.json`) | `changelog.versionFiles` |
| A quoted constant in source, TOML, or YAML frontmatter | `pk pin --file <f> --name <ident> $VERSION` in `preCommit`; see [pk pin](pk-pin.md) |
| A `SOMETHING_VERSION="v..."` line in a shell script | `pk pin --file <f> $VERSION` in `preCommit` |
| Injected at build time (`-ldflags`), never in source | nothing; the build reads the tag |
| Every package in a monorepo at one version | the ecosystem's bump command in `preCommit`, with a `git add` for the files it writes |
| A format no command writes (SPFx four-part versions, several manifests) | a script in `postVersion` or `preCommit` that receives `$VERSION` and stages what it writes |

`pk changelog --undo` reverts all of them together.

## Limits

- Needs git 2.32 or newer, for `git commit --trailer`.
- With no semver tag: `Error: no version tags found`, with the hint `pk setup --baseline [--at <ref>] --push`. With tags on origin but not locally, the hint is `git fetch --tags`; the bootstrap script runs that at session start in sandboxes.
- Commits from bots are skipped unless their tool is configured with a conventional prefix such as `chore(deps)`.
- `versionFiles` is JSON only.
