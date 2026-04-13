# pk pin

Update a pinned version in a script file.

## Usage

```bash
pk pin --file <path> <version>
```

## How it works

1. Reads the target file specified by `--file`.
2. Finds the first line matching the pattern `SOMETHING_VERSION="v..."` — any uppercase variable name ending in `VERSION`.
3. Replaces the version value with the given version (normalized with a `v` prefix).
4. If the file does not exist, exits silently. This makes the command safe to use in hooks without requiring the file to be present.

## Flags

- **--file** — Path to the file containing the version pin (required).

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

For custom files, specify the path in the hook:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file scripts/install.sh $VERSION"
    }
  }
}
```

## Environment

- **VERSION** — Set by `pk changelog` when running the `preCommit` hook. Contains the version being released (e.g., `0.8.1`).

## Details

### Version pin format

The version pin must follow this pattern in the target file:

```bash
SOMETHING_VERSION="v<version>"
```

The variable name must be uppercase letters and underscores, ending with `VERSION`. Examples: `PK_VERSION`, `MY_APP_VERSION`, `VERSION`. The value must be `v`-prefixed and double-quoted. The version must be valid [semver](https://semver.org/) — pre-release and build metadata are supported (e.g., `v1.0.0-beta.1`, `v1.0.0+build.123`).

### Interaction with pk changelog --undo

The updated file is staged by `git add -u` during the changelog commit. `pk changelog --undo` performs `git reset --hard HEAD~1`, which reverts the pin along with the changelog and version files.
