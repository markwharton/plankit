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
pk changelog                      # scan, write CHANGELOG.md, commit, tag
pk release                        # validate and push branch + tag
```

See [pk changelog](docs/pk-changelog.md) and [pk release](docs/pk-release.md) for details.

Monitor at: https://github.com/markwharton/plankit/actions
