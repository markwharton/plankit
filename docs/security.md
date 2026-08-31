# Security

The security model for plankit: what pk ships into other repositories and how that content is scanned, what the hooks do and do not guarantee, what pk touches on the network, and the supply chain behind the binary.

## What pk ships, and the scan on it

`pk setup` installs files that an AI agent reads in every session of every downstream project: `CLAUDE.md`, the rules, the skills, and `.claude/install-pk.sh`. A hidden instruction in one would fan out to every consumer.

- `embed_safety_test.go` walks every embedded managed file under `make test` and fails on control characters, Unicode format characters (zero-width, bidi overrides), bare CR, and invalid UTF-8: the Trojan Source class, CVE-2021-42574.
- The root `.gitattributes` pins the files to LF, so the scan's CR rule cannot false-fail ([design.md](design.md#line-endings)).
- The `pk_sha256` markers detect that a managed file changed, not that a change is safe; the scan is what checks content.
- `pk rules --lint` runs the same character policy over any project's `.claude/rules/`, so a user can scan what they received and what they wrote.

## What the hooks guarantee

- `pk guard` is a guardrail against an agent following its defaults, not a boundary against an adversary. It parses command text, not intent: it splits on shell operators, respects quotes, and skips `VAR=` prefixes and path-qualified git, but command substitution and subshells are out of scope.
- It fails open by design: a missing `pk` exits 127, which Claude Code treats as non-blocking, and the session runs unguarded (`pk setup` warns when `pk` is not on PATH). A crashed hook exits 2, which blocks: guard fails closed on error, open on absence.
- `pk protect` denies writes under `docs/plans/` with no off switch; `pk preserve` refuses nothing.
- Server-side enforcement is GitHub's, not pk's: the ruleset in [branch-protection.md](branch-protection.md), which private personal repositories on the free plan do not enforce.

## Network surface

pk's own network calls are two, both to GitHub over HTTPS:

- `pk version` reads `https://api.github.com/repos/markwharton/plankit/releases/latest`, at most once a day (the cache in [environment-variables.md](environment-variables.md#files)).
- `.claude/install-pk.sh`, in a cloud sandbox, downloads the pinned release binary and `checksums.txt` from `https://github.com/markwharton/plankit/releases/download/<version>/` and verifies the checksum before install.

Everything else is git, run as a child process against the remotes the repository already has. pk sends no telemetry and stores no credential; hooks receive the session's environment and read only `CLAUDE_PROJECT_DIR` from it.

## Supply chain

- `go.mod` has no `require` block: the binary is standard-library Go, `CGO_ENABLED=0`.
- `make vuln` (govulncheck, live database) gates CI; the `go` directive pins an exact toolchain patch so CI and local scans see the same toolchain.
- GitHub Actions are pinned to commit SHAs; Dependabot bumps the pins on `develop`.
- Releases are built by `.github/workflows/release.yml` from the tag, with checksums published beside the binaries ([publishing.md](publishing.md)).
