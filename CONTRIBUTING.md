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

In Claude Code, `pk guard` enforces this automatically by blocking git mutations on `main`. In the terminal, this is a convention — there are no branch rules preventing direct commits to `main`, so discipline is on you.

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
