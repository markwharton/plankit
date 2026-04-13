# Contributing

## Build

```bash
make build                    # Build for current platform -> dist/pk
make build VERSION=1.0.0      # Build with version injected
make build-all                # Cross-compile for all 5 platforms
make install                  # Install to GOPATH/bin (version: dev)
make install VERSION=1.0.0    # Install with version injected
```

The default version is `dev`. To see the installed version:

```bash
pk version    # Shows "pk dev" or "pk 1.0.0" etc.
```

## Test

```bash
make test     # Run tests with race detector
make lint     # Run go vet
make fmt      # Format code
```

## Workflow

All changes go through `develop` — never commit directly to `main`.

In Claude Code, `pk guard` enforces this automatically — it blocks git mutations on `main`. In the terminal, branch protection rules may exist but should not be the only safety net — discipline is on you.

Dependabot PRs target `develop` via `.github/dependabot.yml`. If a hotfix or emergency PR lands directly on `main`, merge main into develop before releasing:

```bash
git switch develop
git merge main
```

This ensures the changelog includes everything in the release and maintains the ancestry that `pk release` needs for fast-forward merges to main.

After merging PRs on GitHub, sync your local branch with rebase to avoid unnecessary merge commits:

```bash
git pull --rebase
```

This replays your unpushed local commits on top of the remote, keeping history linear. Only safe when your local commits haven't been pushed yet — which is exactly when you need it.

## Pull requests

When merging PRs through GitHub, choose the merge method based on the branch:

- **Rebase and merge** for most PRs (e.g., dependabot bumps) — replays commits on top of the target branch. Linear history, and each conventional commit is preserved individually for `pk changelog`.
- **Merge commit** when the PR branch has tags — rebase creates new SHAs which would orphan tags pointing at the originals.
- **Squash is disabled** — it collapses all commits into one, losing the conventional commit messages that `pk changelog` depends on. See [Squash Merge and Release Tags](docs/anti-patterns.md#squash-merge-and-release-tags).

## Release

With `release.branch` configured in `.pk.json`, the full release flow runs from Claude Code or terminal:

```bash
pk changelog --dry-run            # preview changelog and version bump
pk changelog                      # on develop: generate CHANGELOG.md and commit (no tag)
pk release --dry-run              # preview the release flow
pk release                        # read Release-Tag trailer, tag, merge to main, push main + tag, push develop
```

`pk release` merges the current branch into the release branch, validates, pushes, and switches back. See [pk release](docs/pk-release.md) for details.

See [pk changelog](docs/pk-changelog.md) and [pk release](docs/pk-release.md) for details.

Monitor at: https://github.com/markwharton/plankit/actions
