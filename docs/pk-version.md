# pk version

Print the version and check for a newer one.

## Usage

```bash
pk version              # the version
pk version --verbose    # and the Go version, build time and commit
```

## How it works

1. Prints the version set by `-ldflags`, or read from `debug.ReadBuildInfo()`.
2. Compares it with the version pinned in `.claude/install-pk.sh`. When they differ, prints a note: the hint is `go install github.com/markwharton/plankit/cmd/pk@latest` when the pin is newer, `pk setup` otherwise. Skipped on dev builds.
3. Checks GitHub Releases, at most once a day (see [Environment variables](environment-variables.md#files)), and prints an update notice on stderr when a newer version exists. The notice names `brew upgrade plankit` for a Homebrew install and `go install` otherwise.

## Flags

- **--verbose**: include Go version, build time and commit from build info. A dev build shows `(dirty)` when the tree was modified at build time.
