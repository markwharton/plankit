# Fix CI: bump Go toolchain to patch GO-2026-5856

## Context

CI's `make vuln` step (govulncheck) is failing. The scan flagged **GO-2026-5856** — an Encrypted Client Hello privacy leak in `crypto/tls`, present in `crypto/tls@go1.26.4` and **fixed in `crypto/tls@go1.26.5`**. plankit's code reaches the vulnerable symbol through ordinary stdlib call paths (`http.Client.Do`, `io.ReadAll`, `fmt.Fprintf`), so govulncheck marks it as affecting our code and exits non-zero, gating CI.

This is exactly the case the `plankit-development.md` "Security Scanning" rule anticipates: the `go` directive pins an exact patch so local auto-download and CI scan the same toolchain; **when govulncheck flags a fix in a newer patch, bump that one line.**

## Change

Single-line edit in `go.mod`:

```
go 1.26.4  →  go 1.26.5
```

That directive is the single source of truth for the toolchain:
- CI (`.github/workflows/ci.yml`, `release.yml`) resolves Go via `go-version-file: 'go.mod'`, so setup-go picks up 1.26.5.
- Local builds with default `GOTOOLCHAIN=auto` auto-download 1.26.5.
- govulncheck then scans against the patched `crypto/tls` and passes.

No `toolchain` line exists in `go.mod`, and no other file references the version — nothing else to update.

Note: the scan also reported "1 vulnerability in packages you import ... but your code doesn't appear to call" it — that one is non-gating and unaffected by this change; leaving it alone is correct.

## Files

- `go.mod` (line 3) — the only edit.

## Verification

1. `make vuln` — should report no gating vulnerabilities (exit 0). This is the check CI runs.
2. `make test` — confirm the toolchain bump doesn't disturb the build/tests.
3. `make build` — sanity-check a clean build under 1.26.5.

## Commit

Conventional Commits, on `develop` (never `main`). Suggested:

```
security: bump go toolchain to 1.26.5 for GO-2026-5856

crypto/tls ECH privacy leak, fixed in go1.26.5.
```
