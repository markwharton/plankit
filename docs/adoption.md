# Adoption

Adding plankit to a repository that already has history. A new repository takes every layer in one command: [`pk init --push`](pk-init.md).

## The layers, as commands

```bash
pk setup                                  # rules, skills, hooks, modes; restart Claude Code
/pk-configure                             # in Claude Code: writes guard.branches and release.branch
pk setup --baseline [--at <ref>] --push   # v0.0.0, where pk changelog starts reading
pk status                                 # names the command that closes each remaining gap
```

Each layer stands on its own. `pk setup` changes no branch and no tag. `guard.branches` protects a branch inside Claude Code sessions; the server side is the [ruleset](branch-protection.md). `pk changelog` and `pk release` read git and `.pk.json` only, so a repository shaped with [`pk init --no-setup`](pk-init.md) releases with nothing under `.claude/`. Moving between trunk flow and merge flow: [Changing flow](changing-flow.md).

## Migrating from another release tool

- **`CHANGELOG.md`**: `pk changelog` inserts each new section above the existing content and leaves the rest as it is. Rewriting old entries into Keep a Changelog form is optional and loses what pk omits by design (per-commit SHA links).
- **Baseline**: `pk setup --baseline` tags HEAD, so the first release covers only what follows; `--at $(git rev-list --max-parents=0 HEAD)` tags the first commit, so the first release lists the whole history. An existing semver tag is kept and used. See [pk setup](pk-setup.md#baseline).
- **Commit types**: the defaults cover the Conventional Commits set; `changelog.types` only for custom types or section names. See [.pk.json](pk-json.md#changelogtypes).
- **Version files**: none when only the tag carries the version. JSON manifests and lockfiles go in `changelog.versionFiles`; a version in any other file is pinned with [`pk pin`](pk-pin.md) from `changelog.hooks.preCommit`; files derived from the bump are regenerated there too. Tracked files a hook changes are staged by `git add -u`; a new file needs its own `git add`.
- **Gates**: `release.hooks.preRelease` for tests and builds before the tag; `release.hooks.prePush` for signing or an artifact named for the tag. See [.pk.json](pk-json.md#releasehooks).
- **npm scripts**: `"release": "pk changelog && pk release"`, `"release:dry": "pk changelog --dry-run"`.
- **The old tool**: remove commit-and-tag-version, standard-version or semantic-release from CI and dev dependencies before the first `pk changelog`, so two tools do not write tags or `CHANGELOG.md`.

Paste this into Claude Code inside the repository after `pk setup`:

```text
Migrate this established repo onto plankit (pk). `pk setup` has already run. Work through these steps; confirm with me before any write, and stay advisory on anything that tags or pushes.

1. Baseline tag: run `git tag --list 'v*' --sort=-v:refname`. If there's no semver tag, `pk changelog` has nothing to diff from. Offer `pk setup --baseline --at <ref>` (fold prior history into the first entry) or `pk setup --baseline` (start fresh from HEAD); recommend one and let me run it.

2. CHANGELOG.md: default to leaving it untouched (pk appends new entries above old ones, lossless). Offer a rewrite into plankit's format only if I ask, and warn it drops anything pk omits by design (e.g. per-commit SHA links).

3. Version propagation (needs my knowledge of the build): first decide if it's even needed: if only the git tag carries the version, configure nothing. Otherwise, per version-bearing file: JSON manifests/lockfiles -> `changelog.versionFiles`; non-JSON (pyproject.toml, Python `__version__`, Go `const version`, a shell script) -> `pk pin --file <f> [--name <id>] $VERSION` in `changelog.hooks.preCommit` (versionFiles is JSON-only); files derived from the bump (lockfiles, generated docs, workspace refs) -> regenerate in `preCommit`. pk auto-stages CHANGELOG.md, every versionFiles entry, and already-tracked files a hook changes (`git add -u`); add an explicit `git add` only for newly created files. Optionally add a `release.hooks.preRelease` gate (e.g. `npm ci && npm run lint && npm test && npm run build`). Show me the merged `.pk.json` (preserve existing keys, sort top-level) before writing.

4. Remove the old tool: find commit-and-tag-version / standard-version / semantic-release in devDependencies and CI and tell me where to disable them so they and pk do not both write tags or CHANGELOG.md.
```

## Without pk on the machine

| Feature | Without pk |
|---------|------------|
| `CLAUDE.md`, `.claude/rules/` | loaded by Claude Code as usual |
| `/pk-configure`, `/preserve`, `/ship` | the skill loads; the pk command inside it fails |
| `pk guard`, `pk preserve`, `pk protect` hooks | exit 127, which Claude Code treats as non-blocking: nothing is guarded, preserved or protected |

Cloud sandboxes install pk at session start through `.claude/install-pk.sh`. On a local machine the same hook prints a warning with the install command.
