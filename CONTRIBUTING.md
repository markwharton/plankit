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

### Any repo using pk

```bash
git tag v0.0.0                    # one-time baseline (if no tags exist)
pk changelog                      # scan, write CHANGELOG.md, commit, tag
git push --follow-tags            # done
```

### plankit (with build validation)

```bash
pk changelog                      # scan, write CHANGELOG.md, commit, tag
scripts/release.sh --dry          # validate, test, cross-compile (dry run)
scripts/release.sh                # push branch + tag → triggers GitHub Actions
```

`pk changelog` auto-detects the version bump from conventional commits. Override
with `--bump major|minor|patch`. Preview with `--dry-run`.

Git tags are the single version source. If no tags exist, `pk changelog` will
prompt you to create a baseline tag.

Type-to-section mapping, version file updates, and lifecycle hooks are configured
in `.changelog.json`.

The release script finds the tag at HEAD (created by `pk changelog`), validates
the build, then pushes both the branch and tag to trigger CI.

Monitor at: https://github.com/markwharton/plankit/actions
