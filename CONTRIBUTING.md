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

## Release

```bash
git tag v0.0.0                    # one-time baseline (if no tags exist)
git push origin v0.0.0            # push so comparison links work
pk changelog                      # scan, write CHANGELOG.md, commit, tag
pk release --dry-run              # validate (optional)
pk release                        # push branch + tag
```

`pk changelog` auto-detects the version bump from conventional commits. Override
with `--bump major|minor|patch`. Preview with `--dry-run`.

`pk release` validates pre-flight checks (clean tree, correct branch, not behind
remote), runs the `hooks.preRelease` command from `.changelog.json` if configured,
then pushes the branch and tag to origin.

Git tags are the single version source. If no tags exist, `pk changelog` will
prompt you to create a baseline tag.

Type-to-section mapping, version file updates, and lifecycle hooks are configured
in `.changelog.json`.

Monitor at: https://github.com/markwharton/plankit/actions
