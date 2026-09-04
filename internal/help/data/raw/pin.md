---
name: pin
description: Update a version pin in a file, for release hooks
---

# pk pin

Updates a pinned version string in a file. Release and changelog hooks
call it with `$VERSION` so pinned copies (workflow files, install
scripts, documentation) track the release automatically.

## Pin forms

Without `--name`, the first shell-style `SOMETHING_VERSION="vX.Y.Z"`
line is updated. With `--name <identifier>`, the first assignment of a
quoted string to that identifier is updated; `=`, `:=`, and a bare
YAML colon all match, and the `v` prefix is inferred from the existing
value.

## Usage

```bash
pk pin --file .github/workflows/ci.yml --name pk-version $VERSION
```

A missing file is a no-op and a file without a matching pin is a
warning, never a failure, so a renamed target cannot abort a release
from inside a hook.
