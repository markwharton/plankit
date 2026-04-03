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

Releases are automated via GitHub Actions. The workflow:

1. Run the release script with a dry run first:

```bash
scripts/release.sh 1.0.0 --dry
```

This validates: clean working tree, on main branch, up to date with origin, tag available, tests pass, cross-compilation works for all 5 platforms.

2. If the dry run passes, run without `--dry`:

```bash
scripts/release.sh 1.0.0
```

This creates and pushes a `v1.0.0` tag. GitHub Actions picks up the tag, builds binaries for all platforms, generates checksums, and creates a GitHub Release.

Monitor at: https://github.com/markwharton/plankit/actions
