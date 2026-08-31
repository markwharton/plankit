# pk pin

Rewrite a pinned version in a file. Used from `changelog.hooks.preCommit`, where `$VERSION` is the version being released.

## Usage

```bash
pk pin --file <path> [--name <identifier>] <version>
```

## How it works

1. Reads the file. A missing file exits 0 with no output; a file with no matching line exits 0 with a warning, so a hook proceeds either way.
2. Finds the first matching line:
   - **without `--name`**: `SOMETHING_VERSION="v..."`, any name of uppercase letters and underscores ending in `VERSION`, the value double-quoted and `v`-prefixed;
   - **with `--name`**: the identifier at a word boundary, then `=`, `:=` or a bare colon, then a single- or double-quoted string.
3. Replaces the value. The `v` prefix follows the existing value.

`<version>` must be semver; pre-release and build metadata are accepted (`1.0.0-beta.1`, `1.0.0+build.123`).

## Flags

- **--file `<path>`**: the file. Required.
- **--name `<identifier>`**: the identifier to match; without it, the shell-variable form.

## Configuration

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file cmd/myapp/main.go --name version $VERSION && pk pin --file .claude/install-pk.sh $VERSION"
    }
  }
}
```

| Line in the file | Hook |
|---|---|
| `const version = "0.1.0"` (Go) | `pk pin --file cmd/myapp/main.go --name version $VERSION` |
| `__version__ = "0.1.0"` (Python) | `pk pin --file mypackage/__init__.py --name __version__ $VERSION` |
| `version = "0.1.0"` (Cargo.toml) | `pk pin --file Cargo.toml --name version $VERSION` |
| `version: "0.1.0"` (SKILL.md frontmatter) | `pk pin --file .claude/skills/my-skill/SKILL.md --name version $VERSION` |
| `PK_VERSION="v0.1.0"` (shell) | `pk pin --file .claude/install-pk.sh $VERSION` |

The pinned file is a tracked file the hook changed, so `pk changelog` stages it with `git add -u`, and `pk changelog --undo` reverts it with the changelog.

## Limits

- The first match wins. In `Cargo.toml` that is `[package]`'s `version` only while it precedes `[dependencies]`.
- An unquoted value (`version: 0.1.0`) does not match; the quote is what keeps `version: see below` from matching.
- `pk setup` pins `.claude/install-pk.sh` to the running pk version itself; the hook above is for a project that pins the version `pk changelog` computed.
