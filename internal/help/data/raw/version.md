---
name: version
description: Print the pk build version
---

# pk version

Prints the version of the running pk binary. Release builds report the
stamped semver; development builds report `dev` plus the commit.

## Usage

```bash
pk version
pk version --verbose
pk version --format json
```

`--verbose` adds the Go version and platform. `--format json` emits a
single object with `version`, `go`, `os`, and `arch` keys, for
scripting.
