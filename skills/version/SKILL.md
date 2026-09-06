---
name: version
description: Print the pk build version
---

# pk version

Prints the version of the running pk binary. Release builds report the
stamped version. `go install` builds report the module version.
Source builds report `dev` plus the commit.

## Usage

```bash
pk version
pk version --verbose
pk version --format json
```

`--verbose` adds the Go version and platform. `--format json` emits
one object with `version`, `go`, `os`, and `arch` keys.

## Flags

<!-- generated: flags -->
```
  --verbose
        Show build details
  --format <value>
        Output format: text or json (default text)
```
<!-- /generated: flags -->
