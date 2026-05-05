# pk pin

Update a pinned version in a file.

## Usage

```bash
pk pin --file <path> [--name <identifier>] <version>
```

## How it works

1. Reads the target file specified by `--file`.
2. Finds the first line containing the version pin:
   - **Without `--name`:** matches `SOMETHING_VERSION="v..."` — any uppercase variable name ending in `VERSION`.
   - **With `--name`:** matches the named identifier assigned a quoted string value (e.g., `const version = "0.1.0"`).
3. Replaces the version value. The `v` prefix is inferred from the existing value: if it had `v`, the replacement gets `v`; otherwise bare.
4. If the file does not exist, exits silently. This makes the command safe to use in hooks without requiring the file to be present.

## Flags

- **--file** — Path to the file containing the version pin (required).
- **--name** — Identifier to match (e.g., `version`, `__version__`). When set, uses flexible matching instead of the shell-variable pattern.

## Configuration

plankit uses `pk pin` as a changelog `preCommit` hook to automate version pinning during releases. Add to `.pk.json`:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file .claude/install-pk.sh $VERSION"
    }
  }
}
```

Other projects using `pk setup` do not need this hook — `pk setup` updates the pin automatically using the running `pk` version. The `preCommit` hook is for projects that need to pin a version computed by `pk changelog` rather than the version of `pk` itself.

For Go projects with a version constant:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file cmd/myapp/main.go --name version $VERSION"
    }
  }
}
```

For Python packages:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file mypackage/__init__.py --name __version__ $VERSION"
    }
  }
}
```

Multiple pins via shell chaining:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file .claude/install-pk.sh $VERSION && pk pin --file cmd/pk/main.go --name version $VERSION"
    }
  }
}
```

## Environment

- **VERSION** — Set by `pk changelog` when running the `preCommit` hook. Contains the version being released (e.g., `0.8.1`).

## Details

### Shell-variable format (without --name)

The version pin must follow this pattern in the target file:

```bash
SOMETHING_VERSION="v<version>"
```

The variable name must be uppercase letters and underscores, ending with `VERSION`. Examples: `PK_VERSION`, `MY_APP_VERSION`, `VERSION`. The value must be `v`-prefixed and double-quoted.

### Named identifier format (with --name)

When `--name` is provided, the command finds a line where the named identifier is assigned a quoted string value. The match requires:

- The identifier at a word boundary (not part of a larger name like `my_version`)
- An assignment operator (`=` or `:=`, with optional surrounding whitespace)
- A double-quoted or single-quoted string value

Examples of lines that match `--name version`:

```go
const version = "0.1.0"
var version = "0.1.0"
version := "0.1.0"
version = "0.1.0"
```

Examples of lines that match `--name __version__`:

```python
__version__ = "0.1.0"
__version__ = '0.1.0'
```

The version must be valid [semver](https://semver.org/) — pre-release and build metadata are supported (e.g., `1.0.0-beta.1`, `1.0.0+build.123`).

### Interaction with pk changelog --undo

The updated file is staged by `git add -u` during the changelog commit. `pk changelog --undo` performs `git reset --hard HEAD~1`, which reverts the pin along with the changelog and version files.
