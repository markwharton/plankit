# pk version

Print the current version and check for updates.

## Usage

```bash
pk version              # print version
pk version --verbose    # include Go version, build time, and commit SHA
```

## How it works

1. Prints the current version (set via `-ldflags` at build time, or read from `debug.ReadBuildInfo()`).
2. Checks `.claude/install-pk.sh` for a pinned version. If the pinned version differs from the running version, prints a note to re-run `pk setup`.
3. Checks GitHub releases for a newer version (cached daily in `~/.pk/update-check`).
4. If a newer version is available, prints an update notice to stderr.

## Flags

- **--verbose** — Include Go version, build time, and commit SHA from build info. Dev builds show `(dirty)` when the working tree was modified at build time.
