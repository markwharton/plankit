# Fix: `pk setup` skips managed files on Windows (CRLF)

## Context

A developer ran `pk setup` on Windows and settings were updated correctly, but all skills, rules, and CLAUDE.md were reported as "skipped (not managed by pk)" despite having valid `pk_sha256` markers. The root cause: Git on Windows (default `core.autocrlf=true`) checks out files with `\r\n` line endings, but `ExtractSHA` uses hardcoded `\n` for all string matching.

## Approach

Normalize CRLF to LF inside `ExtractSHA` itself. This fixes all callers (current and future) with one change point. The returned body will be LF-only, which matches the SHA that pk originally computed from LF embedded assets.

## Changes

### `internal/setup/managed.go`

1. Add `normalizeLF` helper (after `ContentSHA`, before `ExtractSHA`):
   - Fast path: if no `\r` present, return unchanged (zero allocation on Linux/macOS)
   - Replace `\r\n` with `\n`, then any remaining lone `\r` with `\n`

2. Call `normalizeLF(fileContent)` at the top of `ExtractSHA` (one line added)

No changes to `embedSHA`, `writeManaged`, or `shouldUpdate` — they either operate on embedded content (always LF) or flow through `ExtractSHA` which now normalizes.

### `internal/setup/managed_test.go`

Add tests:
- `TestNormalizeLF` — table-driven: no CR, CRLF, lone CR, mixed, empty
- `TestExtractSHA_htmlComment_CRLF` — HTML comment marker with `\r\n`
- `TestExtractSHA_frontmatter_CRLF` — frontmatter marker with `\r\n`
- `TestShouldUpdate_pristineHTMLComment_CRLF` — full round-trip: pk writes LF, Git converts to CRLF on disk, `shouldUpdate` still recognizes it as managed and reports "updated"
- `TestShouldUpdate_pristineFrontmatter_CRLF` — same for frontmatter format

## What does NOT change

- `embedSHA` — operates on embedded content (always LF)
- `writeManaged` — `content` comes from embedded assets (always LF)
- `internal/teardown/teardown.go` and `internal/status/status.go` — they call `setup.ExtractSHA(string(data))`, so the fix propagates automatically

## Verification

```bash
make test                                           # full suite, no regressions
go test ./internal/setup/ -run "CRLF|NormalizeLF" -v  # new tests pass
```

Smoke test: create a file with CRLF line endings containing a valid `pk_sha256` marker, confirm `ExtractSHA` finds it and the body hash matches.
