# Stop HTML-escaping in pk-written JSON (`&&` → `\u0026\u0026`)

## Context

`pk setup`'s field-merge rewrite of `.pk.json` re-encoded the user's hand-written `changelog.hooks.preCommit` value, turning `&&` into `\u0026\u0026`. Nothing is broken: `json.Unmarshal` (internal/config/config.go:116) decodes the escape and `sh -c` (internal/hooks/script.go:32) receives a literal `&&`. But `.pk.json` and `.claude/settings.json` are user-owned, hand-edited files, and the escaping hurts clarity for zero benefit.

Root cause: Go's `encoding/json` HTML-escapes `&`, `<`, `>` by default (a web-embedding safety measure irrelevant to config files). Crucially, the escaping happens at the **outer** `json.Marshal`/`json.MarshalIndent` call: even though `OrderedObject` preserves user values as raw bytes, the encoder re-escapes the bytes returned by `OrderedObject.MarshalJSON` (internal/setup/setup.go:136). So raw passthrough does not protect user content; every marshal in the chain must disable HTML escaping.

After the fix, the only escapes left are the JSON-spec-mandated ones (`\"`, `\\`, control chars) — which developers already expect when writing JSON by hand.

## Changes

### 1. Add helpers in `internal/setup/setup.go` (near `OrderedObject`) — one core, two thin wrappers

```go
// encodeNoHTML is the single place encoder options are set: json.NewEncoder
// into a bytes.Buffer, SetEscapeHTML(false), SetIndent("", indent).
func encodeNoHTML(v any, indent string) ([]byte, error)

// MarshalNoHTML: compact, trailing newline trimmed — for embedding as
// json.RawMessage values.
// MarshalIndentNoHTML: two-space indent, KEEPS Encode's trailing newline —
// for writing files.
```

Both exported wrappers are one-liners over `encodeNoHTML`, so the encoder setup exists exactly once. Keeping the newline in the indent variant deletes the three duplicated `output = append(output, '\n')` lines at the file-writing call sites (setup.go:276/:341, teardown.go:183). Exported because `internal/teardown` (which already imports `setup`) needs them too.

### 2. Replace every marshal that feeds the two user-facing files

`.claude/settings.json` and `.pk.json` write paths — all of these, so no layer re-escapes:

- `internal/setup/setup.go:272` — settings.json top-level `MarshalIndent` → `MarshalIndentNoHTML` (and drop the `append '\n'` line)
- `internal/setup/setup.go:337` (`writePkModes`) — .pk.json top-level `MarshalIndent` → `MarshalIndentNoHTML` (and drop the `append '\n'` line)
- `internal/setup/setup.go:143` (`OrderedObject.MarshalJSON`) — key marshal → `MarshalNoHTML`
- `internal/setup/claude.go:54` (`NewHookEntry`), `:294`/`:300` (`addPermission`), `:363`/`:368` (`setNested`), `:402`/`:425` (`mergeHooks`/`mergeCategory`) → `MarshalNoHTML`
- `internal/teardown/teardown.go:179` → `setup.MarshalIndentNoHTML` (and drop the `append '\n'` line); `:554`, `:577`, `:636`, `:643` → `setup.MarshalNoHTML`

### 3. Deliberately unchanged

- `internal/hooks/response.go` — stdout hook-protocol JSON, machine-read by Claude Code; escaping is irrelevant there.
- `internal/update/update.go:83` — internal state entry, not user-edited.
- `internal/changelog/changelog.go:668` — splices a semver string into .pk.json; semver can never contain `&`/`<`/`>`, and using the helper would add a new `changelog → setup` import for a provable no-op.

## Behavior note

Already-escaped values on disk are **not** un-escaped: `OrderedObject` passes untouched values through as raw bytes, and the encoder won't rewrite them. Any file already containing the escaped form keeps it until hand-edited (both forms parse identically; purely cosmetic). Plankit's own `.pk.json` is clean: the escaped value that surfaced this issue was a temporary test, since restored.

## Tests

- Regression in `internal/setup/setup_test.go`: a `.pk.json` containing `"preCommit": "pk pin ... && go run ./evals/footprint"` round-trips through `writePkModes` with `&&` literal (assert output bytes contain `&&`, not `\u0026`).
- A user hook in settings.json containing `&&` survives `mergeHooks` + the top-level write unescaped.
- Helper unit test: `MarshalNoHTML` keeps `&`/`<`/`>` literal, still escapes `"` as `\"`, and has no trailing newline; `MarshalIndentNoHTML` ends with exactly one `\n`.
- `make test`, `make lint`.

## Smoke verification

1. `make build`
2. In a temp git repo, write a `.pk.json` with `"hooks": {"preCommit": "echo a && echo b"}` and a settings.json with a user hook containing `&&`; run `dist/pk setup`.
3. `cat .pk.json .claude/settings.json` — `&&` literal everywhere, no `\u0026`.
4. Negative case: a hook value containing a quoted string keeps `\"` (JSON-mandated) and the file still parses (`dist/pk changelog --dry-run` or `python3 -m json.tool`).

## Docs / commit

No doc changes — encoding is not an enumerated behavior anywhere. One commit, `fix:` type, one-line message (e.g. `fix: stop HTML-escaping & < > in pk-written JSON`).
