# Plan: Extend pk pin with --name flag

## Context

`pk pin` currently only matches shell variable patterns (`PK_VERSION="v1.2.3"`). A Go project with `const version = "0.1.0"` in main.go can't use `pk pin` in a preCommit hook. The user hit this gap when setting up a new plankit-tooled Go project (ewmload) that needed version pinning but doesn't use shell scripts.

## Design

Add a `--name` flag to `pk pin`. When present, it switches from the legacy shell-variable matcher to a flexible identifier matcher that finds lines where the named identifier is assigned a quoted string value.

### UX

```bash
# Existing (unchanged):
pk pin --file .claude/install-pk.sh $VERSION

# New:
pk pin --file cmd/myapp/main.go --name version $VERSION
pk pin --file mypackage/__init__.py --name __version__ $VERSION
```

In `.pk.json`:
```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file cmd/ewmload/main.go --name version $VERSION"
    }
  }
}
```

### Matching rules (when --name is present)

Find the first line where:
1. The identifier `name` appears at a word boundary (not part of a larger identifier like `my_version`)
2. Followed by an assignment operator (`=` or `:=`, with optional surrounding whitespace)
3. Followed by a quoted string value (double or single quotes)

Word boundary: character before `name` is not `[a-zA-Z0-9_]` (or name starts at column 0). Character after `name` is not `[a-zA-Z0-9_]`.

### v-prefix handling

Inferred from the existing value in the file:
- If current value is `"v0.1.0"`, replacement gets `v` prefix
- If current value is `"0.1.0"`, replacement is bare
- No `--prefix` flag needed

### Backward compatibility

- No `--name` flag: legacy shell-variable behavior (unchanged)
- With `--name`: new flexible matching
- File-not-found behavior: `(false, nil)` in both modes (safe for hooks)

## Files to modify

1. **`internal/setup/setup.go`** — Add `matchNamedPin()`, `PinVersionNamed()`, `ReadVersionNamed()`
2. **`internal/setup/setup_test.go`** — Tests for new functions
3. **`cmd/pk/main.go`** — Add `--name` flag to `runPin`, dispatch to `PinVersionNamed` when set
4. **`docs/pk-pin.md`** — Document `--name` flag and examples

## Implementation

### New functions in `internal/setup/setup.go`

```go
type namedPinMatch struct {
    linePrefix string // everything up to and including the opening quote
    value      string // version string between quotes
    quote      byte   // quote character (' or ")
}

func matchNamedPin(line, name string) (namedPinMatch, bool)
func PinVersionNamed(filePath, name, ver string) (bool, error)
func ReadVersionNamed(filePath, name string) (string, bool)
```

### `matchNamedPin` logic

1. Find `name` in line with word boundary check on both sides
2. After name, skip whitespace, expect `=` or `:=`
3. Skip whitespace after operator
4. Expect `"` or `'`, scan to matching close quote
5. Return prefix (line up to opening quote inclusive), value, quote char

### `PinVersionNamed` logic

1. Read file, split lines
2. Scan for first `matchNamedPin(line, name)` hit
3. If no match: `return false, fmt.Errorf("%s has no pin for %q", basename, name)`
4. Determine v-prefix from existing value (`strings.HasPrefix(match.value, "v")`)
5. Format new value: if had v, prepend v to ver; otherwise bare (strip v if present)
6. Reconstruct line: `match.linePrefix + newValue + string(match.quote) + rest-of-line-after-closing-quote`
7. Write file

### CLI update in `cmd/pk/main.go`

```go
func runPin(args []string) {
    fs := flag.NewFlagSet("pin", flag.ExitOnError)
    file := fs.String("file", "", "File containing the version pin (required)")
    name := fs.String("name", "", "Identifier to match (e.g., version, __version__)")
    fs.Parse(args)
    // ... validation ...
    if *name != "" {
        updated, err = setup.PinVersionNamed(*file, *name, fs.Arg(0))
    } else {
        updated, err = setup.PinVersion(*file, fs.Arg(0))
    }
}
```

## Tests

- `TestMatchNamedPin_goConst`: `const version = "0.1.0"` matches name `version`
- `TestMatchNamedPin_goVar`: `var version = "0.1.0"` matches
- `TestMatchNamedPin_python`: `__version__ = "0.1.0"` matches name `__version__`
- `TestMatchNamedPin_singleQuote`: `__version__ = '0.1.0'` matches
- `TestMatchNamedPin_toml`: `version = "0.1.0"` matches
- `TestMatchNamedPin_vPrefix`: `version = "v0.1.0"` preserves v
- `TestMatchNamedPin_noMatch`: `comment = "hello"` with name `version` returns false
- `TestMatchNamedPin_partialName`: `my_version = "1.0.0"` does NOT match name `version`
- `TestMatchNamedPin_colonEquals`: `version := "1.0.0"` matches
- `TestPinVersionNamed_updatesGoConst`: full file read/write cycle
- `TestPinVersionNamed_preservesVPrefix`: existing v kept
- `TestPinVersionNamed_stripsVWhenBare`: bare stays bare
- `TestPinVersionNamed_noFile`: returns (false, nil)
- `TestPinVersionNamed_noMatch`: returns error
- `TestReadVersionNamed_found`: reads back the value
- `TestReadVersionNamed_notFound`: returns ("", false)

## Verification

1. `make test` — all existing + new tests pass
2. `make lint` — clean
3. Smoke test with a temp Go file:
   ```bash
   echo 'const version = "0.1.0"' > /tmp/test.go
   pk pin --file /tmp/test.go --name version 0.2.0
   grep version /tmp/test.go  # expect: const version = "0.2.0"
   ```
4. Verify legacy behavior unchanged:
   ```bash
   echo 'PK_VERSION="v0.1.0"' > /tmp/test.sh
   pk pin --file /tmp/test.sh 0.2.0
   grep VERSION /tmp/test.sh  # expect: PK_VERSION="v0.2.0"
   ```
