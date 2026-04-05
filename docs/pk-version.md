# pk version

Print the current version and check for updates.

## Usage

```bash
pk version              # print version
pk version --verbose    # include build details (Go version, commit, time)
```

## How it works

1. Prints the current version (set via `-ldflags` at build time, or read from `debug.ReadBuildInfo()`).
2. Checks GitHub releases for a newer version (cached daily in `~/.pk/update-check`).
3. If a newer version is available, prints an update notice to stderr.

## Flags

- **--verbose** — Include Go version, VCS revision, and build time from build info.
