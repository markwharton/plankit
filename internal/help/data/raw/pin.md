---
name: pin
description: Update a version pin in a file, for release hooks
argument-hint: --file <path> [--name <identifier>] <version>
---

# pk pin

Updates a pinned version string in a file. The `release` and
`changelog` hooks call it with `$VERSION`.

## Pin forms

Without `--name`, the first shell-style `SOMETHING_VERSION="vX.Y.Z"`
line is updated. With `--name <identifier>`, the first assignment of
a quoted string to that identifier is updated. `=`, `:=`, and a bare
YAML colon match. The `v` prefix is inferred from the existing value.

## Usage

```bash
pk pin --file .github/workflows/ci.yml --name pk-version $VERSION
```

A missing file is a no-op and a file without a matching pin is a
warning, never a failure, so a renamed target cannot abort a release
from inside a hook.

## Flags

<!-- generated: flags -->
```
  --file <value>
        File containing the pin (relative to the project directory)
  --name <value>
        Identifier of a named pin; default is the SOMETHING_VERSION="v..." shell form
```
<!-- /generated: flags -->
