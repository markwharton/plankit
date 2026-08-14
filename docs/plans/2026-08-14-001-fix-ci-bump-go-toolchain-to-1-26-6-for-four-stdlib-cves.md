# Fix CI: bump Go toolchain to 1.26.6 for four stdlib CVEs

## Context

The scheduled `main` CI run failed at the `make vuln` step (govulncheck). The scan
flagged four Go standard-library vulnerabilities, all present in `go1.26.5` and all
**fixed in `go1.26.6`**:

- **GO-2026-6218** — quadratic complexity in `resolvePath` (`net/url`)
- **GO-2026-6090** — unbounded post-handshake messages (`crypto/tls`)
- **GO-2026-5972** — recursion depth in `encoding/asn1`
- **GO-2026-5026** — Punycode label rejection (`net/http` via `golang.org/x/net/idna`)

plankit's code reaches these symbols through ordinary stdlib call paths
(`http.Client.Do`, `io.ReadAll`, `fmt.Fprintf`, `exec.Command`), so govulncheck
marks our code affected and exits 3, gating CI.

This is the same class of failure as the 2026-07-10 fix (GO-2026-5856,
`go 1.26.4 → 1.26.5`). The maintainer rule in `.claude/rules/plankit-development.md`
("Security Scanning") anticipates it: *"when `govulncheck` flags a fix in a newer
patch, bump that one line."*

## Change

Single-line edit to `go.mod`:

```
go 1.26.5   →   go 1.26.6
```

That is the only source of truth. CI (`ci.yml`, `release.yml`) and local builds all
resolve the toolchain via `go-version-file: 'go.mod'` / `GOTOOLCHAIN=auto`, so no
other file needs to change. No code, docs, or workflow edits.

## Verification

1. Edit `go.mod` line 3.
2. `make vuln` — must exit 0 (auto-downloads 1.26.6). This is the exact check CI runs.
3. `make build` && `make test` — sanity-check a clean build/test under 1.26.6.

## Commit

On `develop`, commit type `security`:

```
security: bump go toolchain to 1.26.6

Fixes GO-2026-6218, GO-2026-6090, GO-2026-5972, GO-2026-5026;
all stdlib CVEs fixed in go1.26.6.
```

Commit only (no push) — pushing is the developer's call.
